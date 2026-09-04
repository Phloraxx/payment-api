package relay

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrPairingTokenInvalid = errors.New("pairing token is invalid")
	ErrPairingTokenExpired = errors.New("pairing token has expired")
	ErrPairingTokenUsed    = errors.New("pairing token has already been used")
	ErrRelayAlreadyActive  = errors.New("an active relay device already exists")
	ErrInvalidDevice       = errors.New("invalid relay device")
)

const defaultPairingTTL = 2 * time.Minute

type PairingSession struct {
	Token           string
	ExpiresAt       time.Time
	ReplaceExisting bool
}

type PairDeviceInput struct {
	Token          string
	Name           string
	PublicKeyPEM   string
	AppVersion     string
	DeviceModel    string
	AndroidVersion string
}

type PairDeviceResult struct {
	DeviceID         string
	ReplacedDeviceID string
	Enabled          bool
}

type DeviceInfo struct {
	ID                        string     `json:"id"`
	Name                      string     `json:"name"`
	Enabled                   bool       `json:"enabled"`
	EnrolledAt                time.Time  `json:"enrolled_at"`
	LastSeenAt                *time.Time `json:"last_seen_at,omitempty"`
	LastHeartbeatAt           *time.Time `json:"last_heartbeat_at,omitempty"`
	AppVersion                string     `json:"app_version,omitempty"`
	DeviceModel               string     `json:"device_model,omitempty"`
	AndroidVersion            string     `json:"android_version,omitempty"`
	NotificationAccess        *bool      `json:"notification_access,omitempty"`
	ListenerConnected         *bool      `json:"listener_connected,omitempty"`
	BatteryOptimizationExempt *bool      `json:"battery_optimization_exempt,omitempty"`
	PowerSaveMode             *bool      `json:"power_save_mode,omitempty"`
	BackgroundRestricted      *bool      `json:"background_restricted,omitempty"`
	ForegroundService         *bool      `json:"foreground_service,omitempty"`
	PendingCount              *int       `json:"pending_count,omitempty"`
	FailedCount               *int       `json:"failed_count,omitempty"`
	LastSuccessfulDeliveryAt  *time.Time `json:"last_successful_delivery_at,omitempty"`
	LastClientError           string     `json:"last_client_error,omitempty"`
}

func (s *Service) CreatePairing(ctx context.Context, replaceExisting bool) (PairingSession, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return PairingSession{}, errors.New("relay storage is required")
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	tokenFn := s.NewPairingToken
	if tokenFn == nil {
		tokenFn = randomPairingToken
	}
	idFn := s.NewID
	if idFn == nil {
		idFn = randomID
	}
	ttl := s.PairingTTL
	if ttl <= 0 {
		ttl = defaultPairingTTL
	}
	now := nowFn().UTC()
	expiresAt := now.Add(ttl)
	token, err := tokenFn()
	if err != nil {
		return PairingSession{}, fmt.Errorf("generate pairing token: %w", err)
	}
	if len(strings.TrimSpace(token)) < 22 {
		return PairingSession{}, errors.New("generated pairing token is too short")
	}
	sessionID, err := idFn("pair")
	if err != nil {
		return PairingSession{}, fmt.Errorf("generate pairing session id: %w", err)
	}
	tokenHash := sha256.Sum256([]byte(token))
	// replaceExisting is retained only for wire compatibility with older clients.
	// Pairing is additive in v5: every valid QR enrolls one independently revocable device.
	err = s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO pairing_sessions(id,token_hash,replace_existing,created_at,expires_at)
			VALUES(?,?,?,?,?)`, sessionID, tokenHash[:], 0, now.UnixMilli(), expiresAt.UnixMilli())
		if err != nil {
			return fmt.Errorf("create pairing session: %w", err)
		}
		return nil
	})
	if err != nil {
		return PairingSession{}, err
	}
	return PairingSession{Token: token, ExpiresAt: expiresAt, ReplaceExisting: false}, nil
}

func (s *Service) PairDevice(ctx context.Context, input PairDeviceInput) (PairDeviceResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return PairDeviceResult{}, errors.New("relay storage is required")
	}
	normalized, deviceID, err := normalizePairDeviceInput(input)
	if err != nil {
		return PairDeviceResult{}, err
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	tokenHash := sha256.Sum256([]byte(normalized.Token))
	result := PairDeviceResult{DeviceID: deviceID, Enabled: true}
	err = s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var expiresAt int64
		var consumedAt sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT expires_at,consumed_at FROM pairing_sessions WHERE token_hash=?`, tokenHash[:]).Scan(&expiresAt, &consumedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPairingTokenInvalid
		}
		if err != nil {
			return fmt.Errorf("read pairing session: %w", err)
		}
		if consumedAt.Valid {
			return ErrPairingTokenUsed
		}
		if now.UnixMilli() >= expiresAt {
			return ErrPairingTokenExpired
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO relay_devices(
			id,name,public_key_pem,enabled,enrolled_at,app_version,device_model,android_version)
			VALUES(?,?,?,1,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET name=excluded.name,public_key_pem=excluded.public_key_pem,
				enabled=1,app_version=excluded.app_version,device_model=excluded.device_model,android_version=excluded.android_version`,
			deviceID, normalized.Name, normalized.PublicKeyPEM, now.UnixMilli(), nullableText(normalized.AppVersion),
			nullableText(normalized.DeviceModel), nullableText(normalized.AndroidVersion))
		if err != nil {
			return fmt.Errorf("enroll relay device: %w", err)
		}
		updated, err := tx.ExecContext(ctx, `UPDATE pairing_sessions SET consumed_at=? WHERE token_hash=? AND consumed_at IS NULL`, now.UnixMilli(), tokenHash[:])
		if err != nil {
			return fmt.Errorf("consume pairing token: %w", err)
		}
		if rows, _ := updated.RowsAffected(); rows != 1 {
			return ErrPairingTokenUsed
		}
		return nil
	})
	if err != nil {
		return PairDeviceResult{}, err
	}
	return result, nil
}

func (s *Service) RevokeDevice(ctx context.Context, deviceID string) error {
	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	if s == nil || s.DB == nil || s.DB.SQL == nil || deviceID == "" {
		return ErrInvalidDevice
	}
	result, err := s.DB.SQL.ExecContext(ctx, `UPDATE relay_devices SET enabled=0 WHERE id=? AND enabled=1`, deviceID)
	if err != nil {
		return fmt.Errorf("revoke relay device: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return ErrInvalidDevice
	}
	return nil
}
func normalizePairDeviceInput(input PairDeviceInput) (PairDeviceInput, string, error) {
	input.Token = strings.TrimSpace(input.Token)
	input.Name = strings.TrimSpace(input.Name)
	input.PublicKeyPEM = strings.TrimSpace(input.PublicKeyPEM)
	input.AppVersion = trimRunes(input.AppVersion, 64)
	input.DeviceModel = trimRunes(input.DeviceModel, 255)
	input.AndroidVersion = trimRunes(input.AndroidVersion, 64)
	if len(input.Token) < 22 {
		return PairDeviceInput{}, "", ErrPairingTokenInvalid
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 120 {
		return PairDeviceInput{}, "", fmt.Errorf("%w: device name must contain 1-120 characters", ErrInvalidDevice)
	}
	_, der, err := parsePublicKey(input.PublicKeyPEM)
	if err != nil {
		return PairDeviceInput{}, "", fmt.Errorf("%w: public key must be P-256 ECDSA SubjectPublicKeyInfo PEM", ErrInvalidDevice)
	}
	sum := sha256.Sum256(der)
	return input, hex.EncodeToString(sum[:]), nil
}

func randomPairingToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func (s *Service) Devices(ctx context.Context) ([]DeviceInfo, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return nil, errors.New("relay storage is required")
	}
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT id,COALESCE(name,''),enabled,enrolled_at,last_seen_at,last_heartbeat_at,
		app_version,device_model,android_version,notification_access,listener_connected,battery_optimization_exempt,
		power_save_mode,background_restricted,foreground_service,pending_count,failed_count,last_successful_delivery_at,last_client_error
		FROM relay_devices WHERE enabled=1 ORDER BY COALESCE(last_heartbeat_at,last_seen_at,enrolled_at) DESC,id`)
	if err != nil {
		return nil, fmt.Errorf("read relay devices: %w", err)
	}
	defer rows.Close()
	items := make([]DeviceInfo, 0)
	for rows.Next() {
		var info DeviceInfo
		var lastSeen, lastHeartbeat, lastDelivered sql.NullInt64
		var appVersion, model, androidVersion, lastError sql.NullString
		var notificationAccess, listenerConnected, batteryExempt, powerSave, backgroundRestricted, foregroundService sql.NullInt64
		var pendingCount, failedCount sql.NullInt64
		var enrolledAt int64
		var enabled int
		if err := rows.Scan(&info.ID, &info.Name, &enabled, &enrolledAt, &lastSeen, &lastHeartbeat, &appVersion, &model, &androidVersion,
			&notificationAccess, &listenerConnected, &batteryExempt, &powerSave, &backgroundRestricted, &foregroundService,
			&pendingCount, &failedCount, &lastDelivered, &lastError); err != nil {
			return nil, fmt.Errorf("scan relay device: %w", err)
		}
		info.Enabled = enabled == 1
		info.EnrolledAt = time.UnixMilli(enrolledAt).UTC()
		info.AppVersion, info.DeviceModel, info.AndroidVersion, info.LastClientError = appVersion.String, model.String, androidVersion.String, lastError.String
		info.LastSeenAt = nullableTimePointer(lastSeen)
		info.LastHeartbeatAt = nullableTimePointer(lastHeartbeat)
		info.LastSuccessfulDeliveryAt = nullableTimePointer(lastDelivered)
		info.NotificationAccess = nullableBoolPointer(notificationAccess)
		info.ListenerConnected = nullableBoolPointer(listenerConnected)
		info.BatteryOptimizationExempt = nullableBoolPointer(batteryExempt)
		info.PowerSaveMode = nullableBoolPointer(powerSave)
		info.BackgroundRestricted = nullableBoolPointer(backgroundRestricted)
		info.ForegroundService = nullableBoolPointer(foregroundService)
		info.PendingCount = nullableIntPointer(pendingCount)
		info.FailedCount = nullableIntPointer(failedCount)
		items = append(items, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relay devices: %w", err)
	}
	return items, nil
}

func (s *Service) ActiveDevice(ctx context.Context) (*DeviceInfo, error) {
	items, err := s.Devices(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (s *Service) Device(ctx context.Context, id string) (*DeviceInfo, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return nil, ErrInvalidDevice
	}
	items, err := s.Devices(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, ErrInvalidDevice
}

func nullableTimePointer(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := time.UnixMilli(value.Int64).UTC()
	return &t
}
func nullableBoolPointer(value sql.NullInt64) *bool {
	if !value.Valid {
		return nil
	}
	v := value.Int64 == 1
	return &v
}
func nullableIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}
