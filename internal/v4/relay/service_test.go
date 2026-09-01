package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/observations"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func openRelayDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertProfile(t *testing.T, db *storage.DB, id, parser, upi string, active bool, now time.Time) {
	t.Helper()
	activeInt := 0
	if active {
		activeInt = 1
	}
	_, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
		VALUES(?,?,?,?,1,?,?,?)`, id, strings.ToUpper(id), upi, parser, activeInt, now.UnixMilli(), now.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
}
func enrollTestDevice(t *testing.T, db *storage.DB, enrolledAt time.Time) (*ecdsa.PrivateKey, string) {
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
	deviceID := hex.EncodeToString(sum[:])
	_, err = db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at) VALUES(?,?,?,?,?)`,
		deviceID, "Test Phone", pemValue, 1, enrolledAt.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	return priv, deviceID
}
func signedAuth(t *testing.T, priv *ecdsa.PrivateKey, deviceID string, now time.Time, body []byte) RequestAuth {
	t.Helper()
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	canonical := CanonicalRequest(http.MethodPost, EventPath, timestamp, body)
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return RequestAuth{
		DeviceID: deviceID, Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(sig),
		Method:    http.MethodPost, Path: EventPath,
	}
}

func marshalEvent(t *testing.T, input EventInput) []byte {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func createPayment(t *testing.T, db *storage.DB, now time.Time, key string) (*payments.Service, payments.CreateResult) {
	t.Helper()
	service := payments.NewService(db)
	service.Now = func() time.Time { return now }
	service.Allocator.Random = func(max int) (int, error) { return 36, nil }
	result, err := service.Create(context.Background(), payments.CreateInput{
		RequestedAmountPaise: 10000,
		Name:                 "Sourav P Bijoy",
		ExternalID:           "evt_hardware_security_2026",
		Metadata:             json.RawMessage(`{"registration_id":"reg_284"}`),
		IdempotencyScope:     "merchant_1",
		IdempotencyKey:       key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, result
}

func countRows(t *testing.T, db *storage.DB, table string) int {
	t.Helper()
	var count int
	if err := db.SQL.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
func TestSignedPaytmEventMatchesPaymentAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 1, 3, 30, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "paytm-1")
	occurredAt := createdAt.Add(2 * time.Minute)
	receivedAt := occurredAt.Add(time.Second)
	paymentService.Now = func() time.Time { return receivedAt }
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return receivedAt }

	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("a", 64),
		PackageName:     observations.PaytmBusinessPackage,
		PostedAtMS:      occurredAt.UnixMilli(),
		Title:           "Payment Received on Paytm",
		Text:            "₹100.37 Received from Rahul",
		AmountHintPaise: 10037,
	})
	auth := signedAuth(t, priv, deviceID, receivedAt, body)
	result, err := service.IngestSigned(ctx, auth, body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.PaymentID != created.Payment.ID || !result.Transitioned || result.Duplicate {
		t.Fatalf("ingest result = %+v", result)
	}
	got, err := paymentService.Get(ctx, created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "paid" || got.Payment.PayerName != "Rahul" {
		t.Fatalf("payment = %+v", got.Payment)
	}
	if countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 1 {
		t.Fatal("expected exactly one relay event and one observation")
	}
	var lastSeen int64
	if err := db.SQL.QueryRow(`SELECT last_seen_at FROM relay_devices WHERE id=?`, deviceID).Scan(&lastSeen); err != nil || lastSeen != receivedAt.UnixMilli() {
		t.Fatalf("last_seen_at=%d err=%v", lastSeen, err)
	}
	replay, err := service.IngestSigned(ctx, auth, body)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Duplicate || replay.Status != "matched" || replay.PaymentID != created.Payment.ID || replay.Transitioned {
		t.Fatalf("replay = %+v", replay)
	}
	if countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 1 || countRows(t, db, "webhook_deliveries") != 2 {
		t.Fatal("duplicate relay event created duplicate state")
	}
}
func TestRelaySignatureFailuresDoNotCreateEvents(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)
	paymentService := payments.NewService(db)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return now }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("b", 64),
		PackageName: observations.PaytmBusinessPackage,
		PostedAtMS:  now.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹1.01 Received from Test",
	})

	t.Run("invalid signature", func(t *testing.T) {
		auth := signedAuth(t, priv, deviceID, now, body)
		auth.Signature = base64.StdEncoding.EncodeToString([]byte("invalid"))
		if _, err := service.IngestSigned(context.Background(), auth, body); err == nil {
			t.Fatal("expected invalid signature")
		}
	})
	t.Run("stale timestamp", func(t *testing.T) {
		auth := signedAuth(t, priv, deviceID, now, body)
		auth.Timestamp = strconv.FormatInt(now.Add(-6*time.Minute).UnixMilli(), 10)
		if _, err := service.IngestSigned(context.Background(), auth, body); err == nil {
			t.Fatal("expected stale timestamp")
		}
	})

	t.Run("tampered body", func(t *testing.T) {
		auth := signedAuth(t, priv, deviceID, now, body)
		tampered := append(append([]byte{}, body...), byte(' '))
		if _, err := service.IngestSigned(context.Background(), auth, tampered); err == nil {
			t.Fatal("expected tampered body signature failure")
		}
	})

	if countRows(t, db, "relay_events") != 0 {
		t.Fatal("signature failure persisted a relay event")
	}
}
func TestSignedButUnsupportedOrNonPayGateNotificationsCannotMatch(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 15, 0, 0, time.UTC)
	paymentService := payments.NewService(db)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return now }

	gpay := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("c", 64),
		PackageName: "com.google.android.apps.nbu.paisa.user",
		PostedAtMS:  now.UnixMilli(), Text: "₹100.37 received",
	})
	if _, err := service.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now, gpay), gpay); err == nil {
		t.Fatal("expected GPay package to be rejected in v4.0")
	}
	if countRows(t, db, "relay_events") != 0 {
		t.Fatal("unsupported package persisted a relay event")
	}
	nonPayGate := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("d", 64),
		PackageName: observations.PaytmBusinessPackage,
		PostedAtMS:  now.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹100.00 Received from Rahul",
	})
	result, err := service.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now, nonPayGate), nonPayGate)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || result.PaymentID != "" {
		t.Fatalf("non-PayGate result = %+v", result)
	}
	if countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 0 {
		t.Fatal(".00 event was not safely ignored")
	}
}
func TestPreEnrollmentNotificationIsStoredButIgnored(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 30, 0, 0, time.UTC)
	paymentService := payments.NewService(db)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Minute))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return now }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("e", 64),
		PackageName: observations.PaytmBusinessPackage,
		PostedAtMS:  now.Add(-10 * time.Minute).UnixMilli(),
		Title:       "Payment Received on Paytm", Text: "₹100.37 Received from Rahul",
	})
	result, err := service.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || countRows(t, db, "payment_observations") != 0 {
		t.Fatalf("pre-enrollment result = %+v", result)
	}
}
func TestRetryResumesPreviouslyReceivedRelayEvent(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "retry-1")
	occurredAt := createdAt.Add(2 * time.Minute)
	receivedAt := occurredAt.Add(time.Second)
	paymentService.Now = func() time.Time { return receivedAt }
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return receivedAt }
	sourceID := strings.Repeat("f", 64)
	_, err := db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status)
		VALUES('relay_existing',?,?,?,?,?,'received')`, deviceID, sourceID, observations.PaytmBusinessPackage, occurredAt.UnixMilli(), receivedAt.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: sourceID,
		PackageName: observations.PaytmBusinessPackage,
		PostedAtMS:  occurredAt.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹100.37 Received from Rahul",
	})
	result, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, receivedAt, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Duplicate || !result.Transitioned || result.Status != "matched" || result.PaymentID != created.Payment.ID {
		t.Fatalf("retry result = %+v", result)
	}
	if result.RelayEventID != "relay_existing" || countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 1 {
		t.Fatal("retry did not resume the existing relay event")
	}
}
func TestSignedKotakGoogleMessagesEventMatchesKotakPayment(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 1, 5, 30, 0, 0, time.UTC)
	insertProfile(t, db, "kotak", "kotak_sms", "merchant@kotak", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "kotak-1")
	occurredAt := createdAt.Add(2 * time.Minute)
	receivedAt := occurredAt.Add(time.Second)
	paymentService.Now = func() time.Time { return receivedAt }
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return receivedAt }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("1", 64),
		PackageName: observations.GoogleMessagesPackage,
		PostedAtMS:  occurredAt.UnixMilli(),
		Title:       "Kotak Mahindra Bank",
		Text:        "Kotak: Received Rs. 100.37 from maya@okaxis",
	})
	result, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, receivedAt, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.PaymentID != created.Payment.ID || !result.Transitioned {
		t.Fatalf("Kotak result = %+v", result)
	}
	got, err := paymentService.Get(ctx, created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "paid" || got.Payment.PayerUPIID != "maya@okaxis" {
		t.Fatalf("Kotak paid payment = %+v", got.Payment)
	}
}
