package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

const (
	signatureTolerance = 5 * time.Minute
	DevicePath         = "/api/v4/device"
)

type RequestAuth struct {
	DeviceID  string
	Timestamp string
	Signature string
	Method    string
	Path      string
}
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

func relayError(code, message string, status int) error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

type verifiedDevice struct {
	ID         string
	EnrolledAt time.Time
}

func CanonicalRequest(method, path, timestamp string, body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%s\n%s\n%s\n%s", strings.ToUpper(method), path,
		strings.TrimSpace(timestamp), hex.EncodeToString(sum[:]))
}
func verifyRequest(ctx context.Context, db *storage.DB, auth RequestAuth, body []byte, now time.Time) (verifiedDevice, error) {
	deviceID := strings.ToLower(strings.TrimSpace(auth.DeviceID))
	if len(deviceID) != 64 {
		return verifiedDevice{}, relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	if _, err := hex.DecodeString(deviceID); err != nil {
		return verifiedDevice{}, relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	tsMS, err := strconv.ParseInt(strings.TrimSpace(auth.Timestamp), 10, 64)
	if err != nil || tsMS <= 0 {
		return verifiedDevice{}, relayError("INVALID_RELAY_TIMESTAMP", "invalid relay timestamp", 401)
	}
	requestTime := time.UnixMilli(tsMS).UTC()
	delta := now.UTC().Sub(requestTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > signatureTolerance {
		return verifiedDevice{}, relayError("STALE_RELAY_REQUEST", "relay request timestamp is outside the allowed window", 401)
	}
	var publicKeyPEM string
	var enabled int
	var enrolledAt int64
	err = db.SQL.QueryRowContext(ctx, `SELECT public_key_pem,enabled,enrolled_at FROM relay_devices WHERE id=?`, deviceID).
		Scan(&publicKeyPEM, &enabled, &enrolledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return verifiedDevice{}, relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	if err != nil {
		return verifiedDevice{}, fmt.Errorf("read relay device: %w", err)
	}
	if enabled != 1 {
		return verifiedDevice{}, relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	pub, der, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return verifiedDevice{}, relayError("INVALID_RELAY_DEVICE_KEY", "stored relay device key is invalid", 500)
	}
	fingerprint := sha256.Sum256(der)
	if hex.EncodeToString(fingerprint[:]) != deviceID {
		return verifiedDevice{}, relayError("INVALID_RELAY_DEVICE_KEY", "stored relay device fingerprint does not match its key", 500)
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(auth.Signature))
	if err != nil {
		return verifiedDevice{}, relayError("INVALID_RELAY_SIGNATURE", "invalid relay signature", 401)
	}
	canonical := CanonicalRequest(auth.Method, auth.Path, auth.Timestamp, body)
	digest := sha256.Sum256([]byte(canonical))
	if !ecdsa.VerifyASN1(pub, digest[:], signature) {
		return verifiedDevice{}, relayError("INVALID_RELAY_SIGNATURE", "invalid relay signature", 401)
	}
	return verifiedDevice{ID: deviceID, EnrolledAt: time.UnixMilli(enrolledAt).UTC()}, nil
}

// AuthenticateDevice verifies a signed request from an enabled QR-enrolled device.
// It is shared by relay ingestion and the Android operational dashboard; callers
// decide which application routes a device identity is authorized to use.
func (s *Service) AuthenticateDevice(ctx context.Context, auth RequestAuth, body []byte) (string, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return "", errors.New("relay storage is required")
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	device, err := verifyRequest(ctx, s.DB, auth, body, nowFn().UTC())
	if err != nil {
		return "", err
	}
	return device.ID, nil
}

func parsePublicKey(value string) (*ecdsa.PublicKey, []byte, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil {
		return nil, nil, errors.New("invalid PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() {
		return nil, nil, errors.New("public key is not P-256 ECDSA")
	}
	return pub, block.Bytes, nil
}
