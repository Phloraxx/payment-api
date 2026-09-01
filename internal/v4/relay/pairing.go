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
	ID             string
	Name           string
	Enabled        bool
	EnrolledAt     time.Time
	LastSeenAt     *time.Time
	AppVersion     string
	DeviceModel    string
	AndroidVersion string
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
	err = s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if !replaceExisting {
			var activeID string
			err := tx.QueryRowContext(ctx, `SELECT id FROM relay_devices WHERE enabled=1 LIMIT 1`).Scan(&activeID)
			if err == nil {
				return ErrRelayAlreadyActive
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("read active relay device: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM pairing_sessions WHERE consumed_at IS NULL`); err != nil {
			return fmt.Errorf("invalidate older pairing sessions: %w", err)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO pairing_sessions(id,token_hash,replace_existing,created_at,expires_at)
			VALUES(?,?,?,?,?)`, sessionID, tokenHash[:], boolInt(replaceExisting), now.UnixMilli(), expiresAt.UnixMilli())
		if err != nil {
			return fmt.Errorf("create pairing session: %w", err)
		}
		return nil
	})
	if err != nil {
		return PairingSession{}, err
	}
	return PairingSession{Token: token, ExpiresAt: expiresAt, ReplaceExisting: replaceExisting}, nil
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
		var replaceExisting int
		var expiresAt int64
		var consumedAt sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT replace_existing,expires_at,consumed_at FROM pairing_sessions WHERE token_hash=?`, tokenHash[:]).
			Scan(&replaceExisting, &expiresAt, &consumedAt)
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
		var activeID string
		activeErr := tx.QueryRowContext(ctx, `SELECT id FROM relay_devices WHERE enabled=1 LIMIT 1`).Scan(&activeID)
		if activeErr != nil && !errors.Is(activeErr, sql.ErrNoRows) {
			return fmt.Errorf("read active relay device: %w", activeErr)
		}
		if activeErr == nil && activeID != deviceID {
			if replaceExisting != 1 {
				return ErrRelayAlreadyActive
			}
			if _, err := tx.ExecContext(ctx, `UPDATE relay_devices SET enabled=0 WHERE id=? AND enabled=1`, activeID); err != nil {
				return fmt.Errorf("disable replaced relay device: %w", err)
			}
			result.ReplacedDeviceID = activeID
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

func (s *Service) ActiveDevice(ctx context.Context) (*DeviceInfo, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return nil, errors.New("relay storage is required")
	}
	var info DeviceInfo
	var lastSeen sql.NullInt64
	var appVersion, model, androidVersion sql.NullString
	var enrolledAt int64
	var enabled int
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT id,COALESCE(name,''),enabled,enrolled_at,last_seen_at,app_version,device_model,android_version
		FROM relay_devices WHERE enabled=1 LIMIT 1`).Scan(&info.ID, &info.Name, &enabled, &enrolledAt, &lastSeen, &appVersion, &model, &androidVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read active relay device: %w", err)
	}
	info.Enabled = enabled == 1
	info.EnrolledAt = time.UnixMilli(enrolledAt).UTC()
	info.AppVersion = appVersion.String
	info.DeviceModel = model.String
	info.AndroidVersion = androidVersion.String
	if lastSeen.Valid {
		value := time.UnixMilli(lastSeen.Int64).UTC()
		info.LastSeenAt = &value
	}
	return &info, nil
}
