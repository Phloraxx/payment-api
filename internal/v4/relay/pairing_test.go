package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
)

func newPairingPublicKey(t *testing.T) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pemValue := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	sum := sha256.Sum256(der)
	return pemValue, hex.EncodeToString(sum[:])
}

func TestCreatePairingStoresOnlyTokenHash(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	token := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	service.NewPairingToken = func() (string, error) { return token, nil }

	session, err := service.CreatePairing(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if session.Token != token || !session.ExpiresAt.Equal(now.Add(2*time.Minute)) || session.ReplaceExisting {
		t.Fatalf("pairing session = %+v", session)
	}
	var stored []byte
	var replace int
	var expires int64
	if err := db.SQL.QueryRow(`SELECT token_hash,replace_existing,expires_at FROM pairing_sessions`).Scan(&stored, &replace, &expires); err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256([]byte(token))
	if hex.EncodeToString(stored) != hex.EncodeToString(wantHash[:]) || replace != 0 || expires != session.ExpiresAt.UnixMilli() {
		t.Fatal("stored pairing session does not match hashed-token contract")
	}
	var rawCount int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM pairing_sessions WHERE CAST(token_hash AS TEXT)=?`, token).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if rawCount != 0 {
		t.Fatal("raw pairing token was stored")
	}
}
func TestPairDeviceConsumesTokenAndEnablesFingerprintDevice(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 6, 15, 0, 0, time.UTC)
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	session, err := service.CreatePairing(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, deviceID := newPairingPublicKey(t)
	result, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: session.Token, Name: "Motorola Edge 60 Stylus", PublicKeyPEM: publicKey,
		AppVersion: "0.5.0", DeviceModel: "motorola edge 60 stylus", AndroidVersion: "16",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeviceID != deviceID || !result.Enabled || result.ReplacedDeviceID != "" {
		t.Fatalf("pair result = %+v", result)
	}
	active, err := service.ActiveDevice(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != deviceID || active.Name != "Motorola Edge 60 Stylus" || !active.Enabled {
		t.Fatalf("active device = %+v", active)
	}
	var consumed sql.NullInt64
	if err := db.SQL.QueryRow(`SELECT consumed_at FROM pairing_sessions`).Scan(&consumed); err != nil || !consumed.Valid {
		t.Fatalf("consumed_at=%v err=%v", consumed, err)
	}
	if _, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: session.Token, Name: "Phone", PublicKeyPEM: publicKey,
	}); !errors.Is(err, ErrPairingTokenUsed) {
		t.Fatalf("replay error = %v", err)
	}
}
func TestConnectRequiresExplicitReplaceWhenDeviceAlreadyActive(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 6, 30, 0, 0, time.UTC)
	enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	if _, err := service.CreatePairing(context.Background(), false); !errors.Is(err, ErrRelayAlreadyActive) {
		t.Fatalf("connect pairing error = %v", err)
	}
	if countRows(t, db, "pairing_sessions") != 0 {
		t.Fatal("blocked Connect created a pairing session")
	}
	replace, err := service.CreatePairing(context.Background(), true)
	if err != nil || !replace.ReplaceExisting {
		t.Fatalf("replace pairing = %+v err=%v", replace, err)
	}
}
func TestReplaceDeviceAtomicallyMovesActiveRelay(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 6, 45, 0, 0, time.UTC)
	_, oldID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	session, err := service.CreatePairing(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, newID := newPairingPublicKey(t)
	result, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: session.Token, Name: "Replacement Phone", PublicKeyPEM: publicKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeviceID != newID || result.ReplacedDeviceID != oldID {
		t.Fatalf("replace result = %+v", result)
	}
	var oldEnabled, newEnabled int
	if err := db.SQL.QueryRow(`SELECT enabled FROM relay_devices WHERE id=?`, oldID).Scan(&oldEnabled); err != nil {
		t.Fatal(err)
	}
	if err := db.SQL.QueryRow(`SELECT enabled FROM relay_devices WHERE id=?`, newID).Scan(&newEnabled); err != nil {
		t.Fatal(err)
	}
	if oldEnabled != 0 || newEnabled != 1 {
		t.Fatalf("enabled states old=%d new=%d", oldEnabled, newEnabled)
	}
	active, err := service.ActiveDevice(context.Background())
	if err != nil || active == nil || active.ID != newID {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if countRows(t, db, "relay_devices") != 2 {
		t.Fatal("replacement deleted historical device record")
	}
}
func TestExpiredReplacementLeavesOldDeviceEnabled(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC)
	_, oldID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	session, err := service.CreatePairing(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, _ := newPairingPublicKey(t)
	service.Now = func() time.Time { return session.ExpiresAt.Add(time.Millisecond) }
	_, err = service.PairDevice(context.Background(), PairDeviceInput{
		Token: session.Token, Name: "Too Late", PublicKeyPEM: publicKey,
	})
	if !errors.Is(err, ErrPairingTokenExpired) {
		t.Fatalf("expired pairing error = %v", err)
	}
	var enabled int
	if err := db.SQL.QueryRow(`SELECT enabled FROM relay_devices WHERE id=?`, oldID).Scan(&enabled); err != nil || enabled != 1 {
		t.Fatalf("old device enabled=%d err=%v", enabled, err)
	}
}
func TestNewPairingInvalidatesOlderUnusedQR(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 7, 15, 0, 0, time.UTC)
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	tokens := []string{
		"first-pairing-token-abcdefghijklmnopqrstuvwxyz",
		"second-pairing-token-abcdefghijklmnopqrstuvwxyz",
	}
	index := 0
	service.NewPairingToken = func() (string, error) {
		value := tokens[index]
		index++
		return value, nil
	}
	first, err := service.CreatePairing(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreatePairing(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Token == second.Token || countRows(t, db, "pairing_sessions") != 1 {
		t.Fatal("new QR did not replace the older pending session")
	}
	publicKey, _ := newPairingPublicKey(t)
	if _, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: first.Token, Name: "Old QR", PublicKeyPEM: publicKey,
	}); !errors.Is(err, ErrPairingTokenInvalid) {
		t.Fatalf("old QR error = %v", err)
	}
	if _, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: second.Token, Name: "Current QR", PublicKeyPEM: publicKey,
	}); err != nil {
		t.Fatalf("current QR failed: %v", err)
	}
}
func TestRevokeDeviceDisablesWithoutDeletingHistory(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 7, 30, 0, 0, time.UTC)
	_, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	if err := service.RevokeDevice(context.Background(), deviceID); err != nil {
		t.Fatal(err)
	}
	active, err := service.ActiveDevice(context.Background())
	if err != nil || active != nil {
		t.Fatalf("active after revoke = %+v err=%v", active, err)
	}
	if countRows(t, db, "relay_devices") != 1 {
		t.Fatal("revocation deleted relay history")
	}
}
func TestReplacementRollbackKeepsOldDeviceAndTokenUnused(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 7, 45, 0, 0, time.UTC)
	_, oldID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	session, err := service.CreatePairing(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, _ := newPairingPublicKey(t)
	if _, err := db.SQL.Exec(`CREATE TRIGGER reject_new_relay BEFORE INSERT ON relay_devices BEGIN SELECT RAISE(ABORT,'forced enrollment failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PairDevice(context.Background(), PairDeviceInput{
		Token: session.Token, Name: "Replacement", PublicKeyPEM: publicKey,
	}); err == nil {
		t.Fatal("expected forced enrollment failure")
	}
	var enabled int
	if err := db.SQL.QueryRow(`SELECT enabled FROM relay_devices WHERE id=?`, oldID).Scan(&enabled); err != nil || enabled != 1 {
		t.Fatalf("old device enabled=%d err=%v", enabled, err)
	}
	var consumed sql.NullInt64
	if err := db.SQL.QueryRow(`SELECT consumed_at FROM pairing_sessions`).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed.Valid {
		t.Fatal("failed replacement consumed the pairing token")
	}
	if countRows(t, db, "relay_devices") != 1 {
		t.Fatal("failed replacement persisted a partial new device")
	}
}
