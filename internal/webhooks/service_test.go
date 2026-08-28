package webhooks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/store"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func webhookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create PocketBase test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func createWebhookTestPayment(t *testing.T, app *tests.TestApp, cfg config.Config, now time.Time) string {
	t.Helper()
	service := payments.NewService(app, cfg, nil)
	service.Now = func() time.Time { return now }
	service.SuffixStart = func() (int64, error) { return 1, nil }
	payment, _, err := service.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	return payment.ID
}

func scheduleWebhookTestPayment(t *testing.T, service *Service, app *tests.TestApp, event, paymentID string, at time.Time) {
	t.Helper()
	db := store.NewPocketBase(app)
	if err := db.Write(context.Background(), func(uow store.UnitOfWork) error {
		payment, err := uow.Payments().Get(paymentID)
		if err != nil {
			return err
		}
		return service.SchedulePayment(uow, event, payment, at)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSignIsStableHMAC(t *testing.T) {
	got := Sign("secret", "123", []byte(`{"ok":true}`))
	want := "12f14ade5e7e737164d9ae20ea4e070056a3045b2c8f42f5f216008eae4684dd"
	if got != want {
		t.Fatalf("Sign() = %s; want %s", got, want)
	}
}

func TestWebhookDeliveryPersistsSuccessAndSignature(t *testing.T) {
	app := webhookTestApp(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	var seenSignature, seenTimestamp, seenEventID, seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenSignature = r.Header.Get("X-PayGate-Signature")
		seenTimestamp = r.Header.Get("X-PayGate-Timestamp")
		seenEventID = r.Header.Get("X-PayGate-Event-Id")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	cfg := config.Config{PaymentTTL: time.Minute, AmountQuarantine: time.Hour, OutgoingWebhookURL: server.URL, OutgoingWebhookSecret: "secret"}
	paymentID := createWebhookTestPayment(t, app, cfg, now)
	service := NewService(app, cfg)
	service.Now = func() time.Time { return now }
	scheduleWebhookTestPayment(t, service, app, "payment.paid", paymentID, now)
	processed, err := service.SendPending(context.Background())
	if err != nil || processed != 1 {
		t.Fatalf("SendPending() = %d, %v", processed, err)
	}
	if seenEventID == "" || seenTimestamp == "" || seenBody == "" {
		t.Fatalf("missing webhook request fields")
	}
	if seenSignature != "v1="+Sign("secret", seenTimestamp, []byte(seenBody)) {
		t.Fatalf("invalid signature %q", seenSignature)
	}
	records, _ := app.FindAllRecords("webhook_deliveries")
	if len(records) != 1 || records[0].GetString("status") != "delivered" || records[0].GetInt("attempts") != 1 {
		t.Fatalf("delivery record = %+v", records)
	}
}

func TestTypedPaymentScheduleUsesUnitOfWorkOutbox(t *testing.T) {
	app := webhookTestApp(t)
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: time.Minute, AmountQuarantine: time.Hour, OutgoingWebhookURL: "https://example.invalid/paygate", OutgoingWebhookSecret: "typed-secret"}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100, ExternalID: "typed-uow"})
	if err != nil {
		t.Fatal(err)
	}
	payment.Status = "cancelled"
	payment.ResolvedAt = now
	service := NewService(app, cfg)
	db := store.NewPocketBase(app)
	if err := db.Write(context.Background(), func(uow store.UnitOfWork) error {
		if err := uow.Payments().Save(payment); err != nil {
			return err
		}
		return service.SchedulePayment(uow, "payment.cancelled", payment, now)
	}); err != nil {
		t.Fatal(err)
	}
	records, err := app.FindAllRecords("webhook_deliveries")
	if err != nil || len(records) != 1 {
		t.Fatalf("deliveries=%d err=%v", len(records), err)
	}
	record := records[0]
	if record.GetString("event") != "payment.cancelled" || record.GetString("payment") != payment.ID || record.GetString("status") != "pending" {
		t.Fatalf("delivery event=%s payment=%s status=%s", record.GetString("event"), record.GetString("payment"), record.GetString("status"))
	}
	body := record.GetString("body")
	for _, want := range []string{`"externalId":"typed-uow"`, `"status":"cancelled"`, `"payableAmountPaise":10001`} {
		if !strings.Contains(body, want) {
			t.Fatalf("typed webhook body missing %s: %s", want, body)
		}
	}
}

func TestWebhookRetryIsDurableAndEventuallySucceeds(t *testing.T) {
	app := webhookTestApp(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	var fail atomic.Bool
	fail.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, "try later", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := config.Config{PaymentTTL: time.Minute, AmountQuarantine: time.Hour, OutgoingWebhookURL: server.URL, OutgoingWebhookSecret: "secret"}
	paymentID := createWebhookTestPayment(t, app, cfg, now)
	service := NewService(app, cfg)
	service.Now = func() time.Time { return now }
	scheduleWebhookTestPayment(t, service, app, "payment.paid", paymentID, now)
	if _, err := service.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	records, _ := app.FindAllRecords("webhook_deliveries")
	if records[0].GetString("status") != "failed" || records[0].GetInt("attempts") != 1 {
		t.Fatalf("first attempt = status=%s attempts=%d", records[0].GetString("status"), records[0].GetInt("attempts"))
	}

	fail.Store(false)
	now = now.Add(2 * time.Minute)
	if _, err := service.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	records, _ = app.FindAllRecords("webhook_deliveries")
	if records[0].GetString("status") != "delivered" || records[0].GetInt("attempts") != 2 {
		t.Fatalf("retry = status=%s attempts=%d", records[0].GetString("status"), records[0].GetInt("attempts"))
	}
}

func TestConcurrentWebhookPassesClaimDeliveryOnce(t *testing.T) {
	app := webhookTestApp(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := config.Config{PaymentTTL: time.Minute, AmountQuarantine: time.Hour, OutgoingWebhookURL: server.URL, OutgoingWebhookSecret: "secret"}
	paymentID := createWebhookTestPayment(t, app, cfg, now)
	service := NewService(app, cfg)
	service.Now = func() time.Time { return now }
	scheduleWebhookTestPayment(t, service, app, "payment.paid", paymentID, now)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = service.SendPending(context.Background()) }()
	}
	wg.Wait()
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP requests = %d; want exactly 1", got)
	}
}

func TestWebhookClientDoesNotFollowRedirects(t *testing.T) {
	app := webhookTestApp(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	var redirected atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer destination.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()
	cfg := config.Config{PaymentTTL: time.Minute, AmountQuarantine: time.Hour, OutgoingWebhookURL: redirector.URL, OutgoingWebhookSecret: "redirect-secret"}
	paymentID := createWebhookTestPayment(t, app, cfg, now)
	service := NewService(app, cfg)
	service.Now = func() time.Time { return now }
	scheduleWebhookTestPayment(t, service, app, "payment.paid", paymentID, now)
	if _, err := service.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if redirected.Load() != 0 {
		t.Fatalf("redirect destination received %d requests", redirected.Load())
	}
	records, _ := app.FindAllRecords("webhook_deliveries")
	if len(records) != 1 || records[0].GetString("status") != "failed" || records[0].GetInt("response_code") != http.StatusTemporaryRedirect {
		t.Fatalf("delivery=%v", records)
	}
}
