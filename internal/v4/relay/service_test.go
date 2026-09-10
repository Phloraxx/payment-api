package relay

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
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

func signedAuthWithEpoch(t *testing.T, priv *ecdsa.PrivateKey, deviceID string, now time.Time, body []byte, enrollmentEpoch int64) RequestAuth {
	t.Helper()
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	canonical := CanonicalRequestWithEpoch(http.MethodPost, EventPath, timestamp, enrollmentEpoch, body)
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return RequestAuth{
		DeviceID: deviceID, Timestamp: timestamp,
		Signature:       base64.StdEncoding.EncodeToString(sig),
		EnrollmentEpoch: strconv.FormatInt(enrollmentEpoch, 10),
		Method:          http.MethodPost, Path: EventPath,
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
func TestEpochBoundAuthenticationRejectsMalformedAndWrongEpoch(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 3, 30, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Minute))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	body := []byte(`{"schema_version":1}`)
	epoch := now.Add(-time.Minute).UnixMilli()

	if got, _, err := service.AuthenticateDeviceWithEpoch(ctx, signedAuthWithEpoch(t, priv, deviceID, now, body, epoch), body); err != nil || got != deviceID {
		t.Fatalf("current epoch authentication device=%q err=%v", got, err)
	}

	malformed := signedAuthWithEpoch(t, priv, deviceID, now, body, epoch)
	malformed.EnrollmentEpoch = "not-an-integer"
	if _, err := service.AuthenticateDevice(ctx, malformed, body); err == nil {
		t.Fatal("malformed epoch was accepted")
	}

	wrong := signedAuthWithEpoch(t, priv, deviceID, now, body, epoch+1)
	if _, err := service.AuthenticateDevice(ctx, wrong, body); err == nil {
		t.Fatal("wrong epoch was accepted")
	}

	if _, err := service.AuthenticateDevice(ctx, signedAuth(t, priv, deviceID, now, body), body); err != nil {
		t.Fatalf("legacy authentication rejected: %v", err)
	}
}

func TestLegacySignatureStopsWorkingAfterSameKeyRepair(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	requestTime := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, requestTime.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return requestTime }
	body := []byte(`{"schema_version":1}`)
	legacyAuth := signedAuth(t, priv, deviceID, requestTime, body)
	if got, err := service.AuthenticateDevice(ctx, legacyAuth, body); err != nil || got != deviceID {
		t.Fatalf("legacy device authentication device=%q err=%v", got, err)
	}

	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	pairing, err := service.CreatePairing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repaired, err := service.PairDevice(ctx, PairDeviceInput{
		Token: pairing.Token, Name: "Test Phone", PublicKeyPEM: publicKeyPEM, AppVersion: "test-v2",
	})
	if err != nil {
		t.Fatal(err)
	}
	var epochRequired int
	if err := db.SQL.QueryRow(`SELECT epoch_required FROM relay_devices WHERE id=?`, deviceID).Scan(&epochRequired); err != nil {
		t.Fatal(err)
	}
	if repaired.DeviceID != deviceID || repaired.EnrolledAtMS <= requestTime.Add(-time.Hour).UnixMilli() || epochRequired != 1 {
		t.Fatalf("re-pair device=%q epoch=%d epoch_required=%d", repaired.DeviceID, repaired.EnrolledAtMS, epochRequired)
	}

	if _, err := service.AuthenticateDevice(ctx, legacyAuth, body); err == nil {
		t.Fatal("legacy signature captured before re-pair was accepted after re-pair")
	}
	fresh := signedAuthWithEpoch(t, priv, deviceID, requestTime, body, repaired.EnrolledAtMS)
	if got, err := service.AuthenticateDevice(ctx, fresh, body); err != nil || got != deviceID {
		t.Fatalf("fresh epoch authentication device=%q err=%v", got, err)
	}
}

func TestEpochSignatureIsRejectedAfterSameMillisecondRepair(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 8, 7, 0, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	pair := func() PairDeviceResult {
		session, err := service.CreatePairing(ctx)
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.PairDevice(ctx, PairDeviceInput{Token: session.Token, Name: "Test Phone", PublicKeyPEM: publicKeyPEM})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}

	first := pair()
	body := []byte(`{"schema_version":1}`)
	captured := signedAuthWithEpoch(t, priv, deviceID, now, body, first.EnrolledAtMS)
	second := pair() // service.Now is unchanged: same wall-clock millisecond.
	if second.EnrolledAtMS != first.EnrolledAtMS+1 {
		t.Fatalf("same-millisecond re-pair epoch first=%d second=%d", first.EnrolledAtMS, second.EnrolledAtMS)
	}
	if _, err := service.AuthenticateDevice(ctx, captured, body); err == nil {
		t.Fatal("captured epoch-bound signature survived same-millisecond re-pair")
	}
	fresh := signedAuthWithEpoch(t, priv, deviceID, now, body, second.EnrolledAtMS)
	if got, err := service.AuthenticateDevice(ctx, fresh, body); err != nil || got != deviceID {
		t.Fatalf("fresh epoch authentication device=%q err=%v", got, err)
	}
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
func TestDuplicateRelayEventIDRejectsChangedPayload(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 1, 3, 30, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "duplicate-payload")
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	relayService := NewService(db, paymentService)
	receivedAt := createdAt.Add(2 * time.Minute)
	relayService.Now = func() time.Time { return receivedAt }
	eventID := strings.Repeat("d", 64)
	first := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: eventID, PackageName: observations.PaytmBusinessPackage,
		PostedAtMS: receivedAt.UnixMilli(), Title: "Payment Received on Paytm",
		Text: "₹100.37 Received from Rahul", AmountHintPaise: 10037,
	})
	if _, err := relayService.IngestSigned(ctx, signedAuth(t, priv, deviceID, receivedAt, first), first); err != nil {
		t.Fatal(err)
	}
	changed := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: eventID, PackageName: observations.PaytmBusinessPackage,
		PostedAtMS: receivedAt.UnixMilli(), Title: "Payment Received on Paytm",
		Text: "₹100.37 Received from Mallory", AmountHintPaise: 10037,
	})
	_, err := relayService.IngestSigned(ctx, signedAuth(t, priv, deviceID, receivedAt, changed), changed)
	var relayErr *Error
	if !errors.As(err, &relayErr) || relayErr.Code != "RELAY_EVENT_ID_CONFLICT" || relayErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("changed duplicate error=%v, want relay event conflict", err)
	}
	if countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 1 {
		t.Fatal("changed duplicate mutated stored event state")
	}
	if created.Payment.ID == "" {
		t.Fatal("test payment was not created")
	}
}

func TestAmbiguousRelayNotificationIsVisibleInActivity(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Minute))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("b", 64),
		PackageName: observations.GooglePayPackage, PostedAtMS: now.UnixMilli(),
		Title: "Payment received", Text: "₹100.37 received; balance ₹200.00",
		AmountHintPaise: 10037,
	})

	result, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, now, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ambiguous" || result.Transitioned || result.PaymentID != "" {
		t.Fatalf("ambiguous relay result=%+v", result)
	}
	var status, reason string
	if err := db.SQL.QueryRow(`SELECT status,error FROM relay_events WHERE id=?`, result.RelayEventID).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "ambiguous" || !strings.Contains(reason, "multiple monetary amounts") {
		t.Fatalf("relay event status=%s reason=%q", status, reason)
	}
	if countRows(t, db, "payment_observations") != 0 {
		t.Fatal("ambiguous notification must not create a payment observation")
	}
}

func TestAmbiguousFinalizerReportsRetentionWinner(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 30, 0, 0, time.UTC)
	if _, err := db.SQL.Exec(`INSERT INTO relay_devices(id,public_key_pem,enabled,enrolled_at) VALUES('race-device','pem',1,?)`, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status)
		VALUES('race-event','race-device','race-source','com.example.wallet',?,?,?)`,
		now.UnixMilli(), now.UnixMilli(), "ignored"); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, payments.NewService(db))
	status, err := service.finishAmbiguous(ctx, "race-event", errors.New("ambiguous parser result"))
	if err != nil || status != "ignored" {
		t.Fatalf("finalizer status=%q err=%v", status, err)
	}
	var stored string
	if err := db.SQL.QueryRow(`SELECT status FROM relay_events WHERE id='race-event'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "ignored" {
		t.Fatalf("retention winner was overwritten: %q", stored)
	}
}

func TestRedactRawEventsPreservesIdentityAndMarksStuckReceived(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 10, 3, 0, 0, 0, time.UTC)
	_, err := db.SQL.Exec(`INSERT INTO relay_devices(id,public_key_pem,enabled,enrolled_at) VALUES('retention-device','pem',1,?)`, now.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	old := now.Add(-15 * 24 * time.Hour).UnixMilli()
	recent := now.Add(-24 * time.Hour).UnixMilli()
	insert := `INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,title,text,big_text,payload_hash,status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`
	if _, err := db.SQL.Exec(insert, "old-event", "retention-device", "old-source", "com.example.paytm", old, old,
		"old title", "old body", "old details", bytes.Repeat([]byte{1}, sha256.Size), "matched"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(insert, "pending-event", "retention-device", "pending-source", "com.example.paytm", old, old,
		"pending title", "pending body", "pending details", bytes.Repeat([]byte{2}, sha256.Size), "received"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(insert, "legacy-event", "retention-device", "legacy-source", "com.example.paytm", old, old,
		"legacy title", "legacy body", "legacy details", nil, "matched"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(insert, "recent-event", "retention-device", "recent-source", "com.example.paytm", recent, recent,
		"recent title", "recent body", "recent details", bytes.Repeat([]byte{3}, sha256.Size), "matched"); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, payments.NewService(db))
	redacted, err := service.RedactRawEvents(ctx, now.Add(-14*24*time.Hour), 10)
	if err != nil || redacted != 3 {
		t.Fatalf("redacted=%d err=%v", redacted, err)
	}
	var oldTitle, oldText, pendingTitle, recentTitle sql.NullString
	var oldStatus, pendingStatus, pendingError string
	var payloadHash []byte
	if err := db.SQL.QueryRow(`SELECT title,text,payload_hash,status FROM relay_events WHERE id='old-event'`).Scan(&oldTitle, &oldText, &payloadHash, &oldStatus); err != nil {
		t.Fatal(err)
	}
	if oldTitle.Valid || oldText.Valid || len(payloadHash) != sha256.Size || oldStatus != "matched" {
		t.Fatalf("redacted event title=%v text=%v payload_hash=%d status=%s", oldTitle, oldText, len(payloadHash), oldStatus)
	}
	var legacyHash []byte
	if err := db.SQL.QueryRow(`SELECT title,text,payload_hash FROM relay_events WHERE id='legacy-event'`).Scan(&oldTitle, &oldText, &legacyHash); err != nil {
		t.Fatal(err)
	}
	if oldTitle.Valid || oldText.Valid || !bytes.HasPrefix(legacyHash, []byte("legacy:")) {
		t.Fatalf("legacy redaction title=%v text=%v payload_hash=%q", oldTitle, oldText, legacyHash)
	}
	legacyInput := EventInput{
		PackageName: "com.example.paytm", PostedAtMS: old,
		Title: "legacy title", Text: "legacy body", BigText: "legacy details",
	}
	var found, same bool
	if err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		_, found, same, err = existingRelayEvent(ctx, tx, "retention-device", "legacy-source",
			legacyInput, time.UnixMilli(old), true, bytes.Repeat([]byte{9}, sha256.Size))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !found || !same {
		t.Fatalf("redacted legacy event did not remain idempotent: found=%v same=%v", found, same)
	}
	legacyInput.PostedAtMS = old + 1000
	if err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		_, found, same, err = existingRelayEvent(ctx, tx, "retention-device", "legacy-source",
			legacyInput, time.UnixMilli(old+1000), true, bytes.Repeat([]byte{9}, sha256.Size))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !found || same {
		t.Fatalf("redacted legacy event accepted changed posted_at: found=%v same=%v", found, same)
	}

	legacyInput.PostedAtMS = 0
	if err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		_, found, same, err = existingRelayEvent(ctx, tx, "retention-device", "legacy-source",
			legacyInput, now, false, bytes.Repeat([]byte{9}, sha256.Size))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !found || !same {
		t.Fatalf("redacted legacy event with unreliable posted_at lost idempotency: found=%v same=%v", found, same)
	}

	if err := db.SQL.QueryRow(`SELECT title,status,error FROM relay_events WHERE id='pending-event'`).Scan(&pendingTitle, &pendingStatus, &pendingError); err != nil {
		t.Fatal(err)
	}
	if pendingTitle.Valid || pendingStatus != "ignored" || pendingError == "" {
		t.Fatalf("stuck received event title=%v status=%s error=%q", pendingTitle, pendingStatus, pendingError)
	}
	if err := db.SQL.QueryRow(`SELECT title FROM relay_events WHERE id='recent-event'`).Scan(&recentTitle); err != nil {
		t.Fatal(err)
	}
	if !recentTitle.Valid {
		t.Fatal("recent event body was redacted")
	}
}

func TestPaytmMinuteTimestampDoesNotPreDateMidMinutePayment(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 2, 18, 57, 38, 732_000_000, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "paytm-minute-regression")
	postedAt := time.Date(2026, 9, 2, 18, 57, 55, 267_000_000, time.UTC)
	receivedAt := time.Date(2026, 9, 2, 18, 57, 56, 88_000_000, time.UTC)
	paymentService.Now = func() time.Time { return receivedAt }
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	service := NewService(db, paymentService)
	service.Now = func() time.Time { return receivedAt }

	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("e", 64),
		PackageName: observations.PaytmBusinessPackage, PostedAtMS: postedAt.UnixMilli(),
		Title:           "Payment Received on Paytm",
		BigText:         "₹100.37 Received from Test User\nReceived on 3 Sep 2026 12:27 AM",
		AmountHintPaise: 10037,
	})
	result, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, receivedAt, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.PaymentID != created.Payment.ID || !result.Transitioned {
		t.Fatalf("ingest result = %+v", result)
	}
	var occurredAt int64
	var occurredSource string
	if err := db.SQL.QueryRow(`SELECT occurred_at,occurred_at_source FROM payment_observations WHERE matched_payment_id=?`, created.Payment.ID).Scan(&occurredAt, &occurredSource); err != nil {
		t.Fatal(err)
	}
	if occurredAt != postedAt.UnixMilli() || occurredSource != "notification_posted_at" {
		t.Fatalf("occurred_at=%d source=%s", occurredAt, occurredSource)
	}
}

func TestIndependentNotificationCorroboratesAndReplaysWithoutSecondTransition(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	createdAt := time.Date(2026, 9, 1, 3, 30, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, createdAt)
	paymentService, created := createPayment(t, db, createdAt, "corroborate-1")
	firstAt := createdAt.Add(2 * time.Minute)
	priv, deviceID := enrollTestDevice(t, db, createdAt.Add(-time.Minute))
	service := NewService(db, paymentService)

	firstBody := marshalEvent(t, EventInput{SchemaVersion: 1, EventID: strings.Repeat("c", 64), PackageName: observations.PaytmBusinessPackage,
		PostedAtMS: firstAt.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹100.37 Received from Rahul", AmountHintPaise: 10037})
	service.Now = func() time.Time { return firstAt.Add(time.Second) }
	if first, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, service.Now(), firstBody), firstBody); err != nil || first.Status != "matched" || !first.Transitioned {
		t.Fatalf("first=%+v err=%v", first, err)
	}

	secondAt := firstAt.Add(10 * time.Second)
	secondBody := marshalEvent(t, EventInput{SchemaVersion: 1, EventID: strings.Repeat("d", 64), PackageName: observations.PaytmBusinessPackage,
		PostedAtMS: secondAt.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹100.37 Received from Rahul", AmountHintPaise: 10037})
	service.Now = func() time.Time { return secondAt.Add(time.Second) }
	secondAuth := signedAuth(t, priv, deviceID, service.Now(), secondBody)
	second, err := service.IngestSigned(ctx, secondAuth, secondBody)
	if err != nil || second.Status != "corroborated" || second.PaymentID != created.Payment.ID || second.Transitioned || second.Duplicate {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	replay, err := service.IngestSigned(ctx, secondAuth, secondBody)
	if err != nil || !replay.Duplicate || replay.Status != "corroborated" || replay.PaymentID != created.Payment.ID || replay.Transitioned {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	if countRows(t, db, "relay_events") != 2 || countRows(t, db, "payment_observations") != 2 || countRows(t, db, "webhook_deliveries") != 2 {
		t.Fatal("corroboration created duplicate payment side effects")
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
	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at) VALUES('generic','Generic','merchant@upi','paytm_notification',1,1,?,?)`, now.Add(-time.Hour).UnixMilli(), now.Add(-time.Hour).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	gpay := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("c", 64),
		PackageName: "com.google.android.apps.nbu.paisa.user",
		PostedAtMS:  now.UnixMilli(), Text: "₹100.37 received",
	})
	gpayResult, err := service.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now, gpay), gpay)
	if err != nil {
		t.Fatal(err)
	}
	if gpayResult.Status != "unmatched" || gpayResult.PaymentID != "" {
		t.Fatalf("generic GPay result = %+v", gpayResult)
	}
	if countRows(t, db, "relay_events") != 1 || countRows(t, db, "payment_observations") != 1 {
		t.Fatal("generic incoming notification was not normalized as evidence")
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
	if countRows(t, db, "relay_events") != 2 || countRows(t, db, "payment_observations") != 1 {
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
		PostedAtMS:  now.Add(-90 * time.Second).UnixMilli(),
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
func TestMissingNotificationTimestampIsStoredButIgnored(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 4, 45, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Minute))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("d", 64),
		PackageName: observations.PaytmBusinessPackage,
		Title:       "Payment Received on Paytm", Text: "₹100.37 Received from Rahul",
	})
	result, err := service.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || countRows(t, db, "payment_observations") != 0 {
		t.Fatalf("missing timestamp result = %+v", result)
	}
	var status, errorText string
	if err := db.SQL.QueryRow(`SELECT status,error FROM relay_events WHERE id=?`, result.RelayEventID).Scan(&status, &errorText); err != nil {
		t.Fatal(err)
	}
	if status != "ignored" || errorText != "notification posting time is missing or invalid" {
		t.Fatalf("stored missing timestamp state=%q error=%q", status, errorText)
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
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: sourceID,
		PackageName: observations.PaytmBusinessPackage,
		PostedAtMS:  occurredAt.UnixMilli(), Title: "Payment Received on Paytm", Text: "₹100.37 Received from Rahul",
		AmountHintPaise: 10037,
	})
	_, err := db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,amount_hint_paise,title,text,status)
		VALUES('relay_existing',?,?,?,?,?,10037,?,?,'received')`, deviceID, sourceID, observations.PaytmBusinessPackage, occurredAt.UnixMilli(), receivedAt.UnixMilli(), "Payment Received on Paytm", "₹100.37 Received from Rahul")
	if err != nil {
		t.Fatal(err)
	}
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
func TestSignedBlockedMessageAndEmailPackagesAreRejectedBeforeStorage(t *testing.T) {
	ctx := context.Background()
	db := openRelayDB(t)
	now := time.Date(2026, 9, 1, 5, 30, 0, 0, time.UTC)
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Minute))
	service := NewService(db, payments.NewService(db))
	service.Now = func() time.Time { return now }
	for index, packageName := range []string{observations.GoogleMessagesPackage, observations.GmailPackage} {
		body := marshalEvent(t, EventInput{
			SchemaVersion: 1, EventID: strings.Repeat(string(rune('1'+index)), 64),
			PackageName: packageName, PostedAtMS: now.UnixMilli(),
			Title: "Payment received", Text: "Received Rs. 100.37 from maya@okaxis",
		})
		if _, err := service.IngestSigned(ctx, signedAuth(t, priv, deviceID, now, body), body); err == nil || !strings.Contains(err.Error(), "no longer accepted") {
			t.Fatalf("package %s error=%v", packageName, err)
		}
	}
	if countRows(t, db, "relay_events") != 0 {
		t.Fatal("blocked notification packages must not be stored")
	}
}

func TestUntrustedGenericNotificationCannotAutoConfirm(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 4, 7, 45, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, now.Add(-time.Hour))
	paymentService, created := createPayment(t, db, now, "untrusted-generic")
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	relayService := NewService(db, paymentService)
	relayService.Now = func() time.Time { return now.Add(time.Minute) }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("e", 64),
		PackageName: "com.example.wallet", PostedAtMS: now.Add(time.Minute).UnixMilli(),
		Text: "Payment received ₹100.37 from Rahul",
	})
	result, err := relayService.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now.Add(time.Minute), body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || result.PaymentID != "" || countRows(t, db, "payment_observations") != 0 {
		t.Fatalf("untrusted generic result=%+v observations=%d", result, countRows(t, db, "payment_observations"))
	}
	got, err := paymentService.Get(context.Background(), created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "pending" {
		t.Fatalf("untrusted generic notification changed payment to %q", got.Payment.Status)
	}
}

func TestGenericWalletNotificationMatchesActiveProfilePayment(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 4, 8, 15, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, now.Add(-time.Hour))
	paymentService, created := createPayment(t, db, now, "generic-wallet-match")
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	relayService := NewService(db, paymentService)
	relayService.Now = func() time.Time { return now.Add(time.Minute) }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("a", 64),
		PackageName: observations.GooglePayPackage, PostedAtMS: now.Add(time.Minute).UnixMilli(),
		Text: "Payment received ₹100.37 from Rahul",
	})
	result, err := relayService.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, now.Add(time.Minute), body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.PaymentID != created.Payment.ID || !result.Transitioned {
		t.Fatalf("generic wallet result=%+v", result)
	}
	got, err := paymentService.Get(context.Background(), created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "paid" || got.Payment.PayerName != "Rahul" {
		t.Fatalf("payment=%+v", got.Payment)
	}
}

func TestGenericWalletNotificationUsesReservationProfileAfterActiveSwitch(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 5, 8, 15, 0, 0, time.UTC)
	insertProfile(t, db, "old-profile", "paytm_notification", "old@upi", true, now.Add(-time.Hour))
	paymentService, _ := createPayment(t, db, now, "generic-profile-switch")
	if _, err := db.SQL.Exec(`UPDATE collection_profiles SET active=0,updated_at=? WHERE id='old-profile'`, now.Add(20*time.Second).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	insertProfile(t, db, "new-profile", "paytm_notification", "new@upi", true, now.Add(20*time.Second))

	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	occurred := now.Add(time.Minute)
	received := occurred.Add(30 * time.Second)
	relayService := NewService(db, paymentService)
	relayService.Now = func() time.Time { return received }
	body := marshalEvent(t, EventInput{
		SchemaVersion: 1, EventID: strings.Repeat("d", 64),
		PackageName: observations.GooglePayPackage, PostedAtMS: occurred.UnixMilli(),
		Text: "Payment received ₹100.37 from Rahul",
	})
	result, err := relayService.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, received, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.PaymentID == "" || !result.Transitioned {
		t.Fatalf("generic delayed result=%+v", result)
	}
	var profileID, matchResult string
	if err := db.SQL.QueryRow(`SELECT collection_profile_id,match_result FROM payment_observations WHERE relay_event_id=?`, result.RelayEventID).Scan(&profileID, &matchResult); err != nil {
		t.Fatal(err)
	}
	if profileID != "old-profile" || matchResult != "matched" {
		t.Fatalf("observation profile=%q result=%q", profileID, matchResult)
	}
}

func TestOneEnabledPhoneCanRelaySameIncomingPaymentSafely(t *testing.T) {
	db := openRelayDB(t)
	now := time.Date(2026, 9, 4, 8, 30, 0, 0, time.UTC)
	insertProfile(t, db, "paytm", "paytm_notification", "merchant@paytm", true, now.Add(-time.Hour))
	paymentService, created := createPayment(t, db, now, "one-phone-match")
	priv, deviceID := enrollTestDevice(t, db, now.Add(-time.Hour))
	relayService := NewService(db, paymentService)
	occurred := now.Add(time.Minute)
	relayService.Now = func() time.Time { return occurred.Add(time.Second) }
	first := marshalEvent(t, EventInput{SchemaVersion: 1, EventID: strings.Repeat("b", 64), PackageName: observations.PaytmBusinessPackage, PostedAtMS: occurred.UnixMilli(), Text: "₹100.37 received from Rahul"})
	one, err := relayService.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, occurred.Add(time.Second), first), first)
	if err != nil {
		t.Fatal(err)
	}
	second := marshalEvent(t, EventInput{SchemaVersion: 1, EventID: strings.Repeat("c", 64), PackageName: observations.PaytmBusinessPackage, PostedAtMS: occurred.Add(500 * time.Millisecond).UnixMilli(), Text: "₹100.37 received from Rahul"})
	two, err := relayService.IngestSigned(context.Background(), signedAuth(t, priv, deviceID, occurred.Add(2*time.Second), second), second)
	if err != nil {
		t.Fatal(err)
	}
	if one.Status != "matched" || one.PaymentID != created.Payment.ID || !one.Transitioned {
		t.Fatalf("first=%+v", one)
	}
	if two.Status != "corroborated" || two.PaymentID != created.Payment.ID || two.Transitioned {
		t.Fatalf("second=%+v", two)
	}
	if countRows(t, db, "relay_events") != 2 || countRows(t, db, "payment_observations") != 2 {
		t.Fatal("audit evidence was not retained")
	}
}
