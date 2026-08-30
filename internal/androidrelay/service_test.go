package androidrelay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/paytmnotification"
	"github.com/Phloraxx/payment-api/internal/store"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func testKey(t *testing.T) (*ecdsa.PrivateKey, string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	sum := sha256.Sum256(der)
	return priv, hex.EncodeToString(sum[:]), string(block)
}

func typedRelayDevice(t *testing.T, service *Service, recordID string) *domain.RelayDevice {
	t.Helper()
	var device *domain.RelayDevice
	if err := service.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		device, err = uow.Relay().Get(recordID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return device
}

func TestEnrollVerifyAndIngestPaytmRemoteViewEvidence(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 27, 14, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: 20 * time.Minute, AmountQuarantine: 24 * time.Hour, TestMode: true, PaytmQRPayload: "merchant", PaytmNotificationWebhookSecret: "paytm-notification-secret-long-enough"}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	paytm := paytmnotification.NewService(app, paymentService)
	paytm.Now = func() time.Time { return now }
	service := NewService(app, paytm)
	service.Now = func() time.Time { return now }

	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 1, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 8, 27, 14, 10, 0, 0, time.UTC)
	priv, deviceID, publicPEM := testKey(t)
	enrolled, err := service.Enroll(EnrollmentInput{DeviceID: deviceID, Name: "Test Phone", PublicKeyPEM: publicPEM, AppVersion: "0.2.0", AndroidVersion: "16", DeviceModel: "Test"})
	if err != nil || !enrolled.Enabled {
		t.Fatalf("enroll=%+v err=%v", enrolled, err)
	}

	body := []byte(`{"schemaVersion":1,"eventId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"notification"}`)
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	canonical := CanonicalRequest(http.MethodPost, "/api/relay/v1/events", ts, body)
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	device, err := service.Verify(deviceID, ts, base64.StdEncoding.EncodeToString(sig), http.MethodPost, "/api/relay/v1/events", body)
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.Ingest(device, EventInput{
		SchemaVersion: 1,
		EventID:       "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Kind:          "notification",
		CapturedAtMs:  now.Add(-time.Minute).UnixMilli(),
		Notification: Notification{
			PackageName: PaytmBusinessPackage,
			AppName:     "Paytm for Business",
			Key:         "0|com.paytm.business|1133295218|null|10523",
			ID:          1133295218,
			GroupKey:    "0|com.paytm.business|g:p4b_payment",
			PostTimeMs:  now.Add(-time.Minute).UnixMilli(),
			WhenMs:      now.Add(-time.Minute).UnixMilli(),
			ChannelID:   "payment_notification",
			Title:       "Payment Received on Paytm for Business",
			CustomTexts: []string{"₹1.01 Received from Test User", "Received on 27 Aug 2026 07:38 PM"},
		},
	}, map[string]any{"fixture": true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.Action != "marked_paid" || result.PaymentID != payment.ID {
		t.Fatalf("result=%+v", result)
	}
	stored, err := paymentService.Get(payment.ID)
	if err != nil || stored.Status != domain.StatusPaid || stored.PayerName != "Test User" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	if stored.EvidenceReference != "paytm-notification:android:"+deviceID+":bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("evidence reference=%q", stored.EvidenceReference)
	}
	if count, _ := app.CountRecords("relay_events"); count != 1 {
		t.Fatalf("relay event count=%d", count)
	}
}

func TestRelayObservesGPayWithoutMatching(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	service := NewService(app, nil)
	priv, deviceID, publicPEM := testKey(t)
	_ = priv
	if _, err := service.Enroll(EnrollmentInput{DeviceID: deviceID, Name: "Phone", PublicKeyPEM: publicPEM}); err != nil {
		t.Fatal(err)
	}
	device, err := app.FindFirstRecordByFilter("relay_devices", "device_id={:id}", map[string]any{"id": deviceID})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Ingest(typedRelayDevice(t, service, device.Id), EventInput{SchemaVersion: 1, EventID: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Kind: "notification", Notification: Notification{PackageName: GPayPersonalPackage, Title: "You received money"}}, nil)
	if err != nil || result.Status != "observed" || result.Action != "observed_only" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRelayIgnoresNotificationThatPredatesEnrollment(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	_, deviceID, publicPEM := testKey(t)
	if _, err := service.Enroll(EnrollmentInput{DeviceID: deviceID, Name: "Phone", PublicKeyPEM: publicPEM}); err != nil {
		t.Fatal(err)
	}
	device, err := app.FindFirstRecordByFilter("relay_devices", "device_id={:id}", map[string]any{"id": deviceID})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Ingest(typedRelayDevice(t, service, device.Id), EventInput{
		SchemaVersion: 1,
		EventID:       "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Kind:          "notification",
		CapturedAtMs:  now.UnixMilli(),
		Notification: Notification{
			PackageName: GPayPersonalPackage,
			PostTimeMs:  now.Add(-10 * time.Minute).UnixMilli(),
			WhenMs:      now.Add(-10 * time.Minute).UnixMilli(),
			Title:       "Old active notification",
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || result.Action != "ignored_pre_enrollment" {
		t.Fatalf("old notification result=%+v", result)
	}
}

func TestVerifyAloneDoesNotRefreshRelayReadiness(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	priv, deviceID, publicPEM := testKey(t)
	if _, err := service.Enroll(EnrollmentInput{DeviceID: deviceID, Name: "Phone", PublicKeyPEM: publicPEM}); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"schemaVersion":999}`)
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	canonical := CanonicalRequest(http.MethodPost, "/api/relay/v1/events", ts, body)
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Verify(deviceID, ts, base64.StdEncoding.EncodeToString(sig), http.MethodPost, "/api/relay/v1/events", body); err != nil {
		t.Fatal(err)
	}
	device, err := app.FindFirstRecordByFilter("relay_devices", "device_id={:id}", map[string]any{"id": deviceID})
	if err != nil {
		t.Fatal(err)
	}
	if !device.GetDateTime("last_seen_at").Time().IsZero() {
		t.Fatalf("signature verification alone refreshed last_seen_at: %s", device.GetDateTime("last_seen_at"))
	}
}

func TestRelayGoogleMessagesIsShadowOnlyAndCannotMatchPayment(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: 20 * time.Minute, AmountQuarantine: 24 * time.Hour, TestMode: true}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "kotak"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	_, deviceID, publicPEM := testKey(t)
	if _, err := service.Enroll(EnrollmentInput{DeviceID: deviceID, Name: "Phone", PublicKeyPEM: publicPEM}); err != nil {
		t.Fatal(err)
	}
	record, err := app.FindFirstRecordByFilter("relay_devices", "device_id={:id}", map[string]any{"id": deviceID})
	if err != nil {
		t.Fatal(err)
	}
	device := typedRelayDevice(t, service, record.Id)
	result, err := service.Ingest(device, EventInput{SchemaVersion: 1, EventID: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", Kind: "notification", CapturedAtMs: now.UnixMilli(), Notification: Notification{PackageName: GoogleMessagesPackage, AppName: "Messages", PostTimeMs: now.UnixMilli(), WhenMs: now.UnixMilli(), Title: "Bank", Text: "Received Rs.100.01 from Person UPI Ref:123456789012"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "observed" || result.Action != "shadow_complete" || result.PaymentID != "" {
		t.Fatalf("shadow result=%+v", result)
	}
	stored, err := paymentService.Get(payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.StatusPending || stored.RRN != "" || stored.EvidenceSource != "" {
		t.Fatalf("shadow evidence mutated payment: %+v", stored)
	}
	var relayEvent *domain.RelayEvent
	if err := service.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var loadErr error
		relayEvent, loadErr = uow.RelayEvents().FindByDeviceEvent(device.ID, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
		return loadErr
	}); err != nil {
		t.Fatal(err)
	}
	if relayEvent.ProcessingStatus != "shadow_observed" || relayEvent.ProviderResult == nil {
		t.Fatalf("relay event=%+v", relayEvent)
	}
}
