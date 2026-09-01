package webhooks

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

const testSecret = "0123456789abcdef0123456789abcdef"

type webhookFixture struct {
	db      *storage.DB
	payment payments.Payment
	now     *time.Time
}

func newWebhookFixture(t *testing.T) webhookFixture {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	profileService := profiles.NewService(db)
	ctx := context.Background()
	if _, err := profileService.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "paygate@paytm", PayeeName: "PayGate",
		Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	paymentService := payments.NewService(db)
	paymentService.Now = func() time.Time { return now }
	created, err := paymentService.Create(ctx, payments.CreateInput{
		RequestedAmountPaise: 10000, Name: "Sourav P Bijoy", ExternalID: "evt_test",
		IdempotencyScope: "test", IdempotencyKey: "webhook-fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	return webhookFixture{db: db, payment: created.Payment, now: &now}
}

func newTestService(f webhookFixture, endpoint string) *Service {
	s := NewService(f.db, Config{Endpoint: endpoint, Secret: testSecret, AllowInsecureHTTP: true})
	s.Now = func() time.Time { return *f.now }
	return s
}
func TestSendPendingSignsAndDelivers(t *testing.T) {
	f := newWebhookFixture(t)
	var seen int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&seen, 1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		eventID := r.Header.Get("PayGate-Event-Id")
		timestamp := r.Header.Get("PayGate-Timestamp")
		signature := r.Header.Get("PayGate-Signature")
		if eventID == "" || timestamp == "" || signature == "" {
			t.Fatalf("missing PayGate headers: %v", r.Header)
		}
		want := "v1=" + Sign(testSecret, timestamp, body)
		if !hmac.Equal([]byte(signature), []byte(want)) {
			t.Fatalf("signature=%q want=%q", signature, want)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["id"] != eventID || payload["type"] != "payment.created" {
			t.Fatalf("payload=%v event=%s", payload, eventID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	s := newTestService(f, server.URL)
	processed, err := s.SendPending(context.Background())
	if err != nil || processed != 1 || atomic.LoadInt32(&seen) != 1 {
		t.Fatalf("processed=%d seen=%d err=%v", processed, seen, err)
	}
	var status string
	var attempts int
	var httpStatus int
	var delivered int64
	if err := f.db.SQL.QueryRow(`SELECT status,attempts,last_http_status,delivered_at FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).
		Scan(&status, &attempts, &httpStatus, &delivered); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || attempts != 1 || httpStatus != http.StatusNoContent || delivered != f.now.UnixMilli() {
		t.Fatalf("delivery state status=%s attempts=%d http=%d delivered=%d", status, attempts, httpStatus, delivered)
	}
}

func TestRetryable500BacksOffThenSucceeds(t *testing.T) {
	f := newWebhookFixture(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTestService(f, server.URL)
	if processed, err := s.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("first pass processed=%d err=%v", processed, err)
	}
	var status string
	var attempts int
	var next int64
	if err := f.db.SQL.QueryRow(`SELECT status,attempts,next_attempt_at FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).
		Scan(&status, &attempts, &next); err != nil {
		t.Fatal(err)
	}
	if status != "retry" || attempts != 1 || next != f.now.Add(time.Minute).UnixMilli() {
		t.Fatalf("retry state status=%s attempts=%d next=%d", status, attempts, next)
	}
	*f.now = f.now.Add(time.Minute)
	if processed, err := s.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("second pass processed=%d err=%v", processed, err)
	}
	if err := f.db.SQL.QueryRow(`SELECT status,attempts FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).
		Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || attempts != 2 || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("final retry state status=%s attempts=%d calls=%d", status, attempts, calls)
	}
}

func TestPermanent404ExhaustsImmediately(t *testing.T) {
	f := newWebhookFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer server.Close()

	s := newTestService(f, server.URL)
	if processed, err := s.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	var status string
	var attempts int
	var next *int64
	var httpStatus int
	if err := f.db.SQL.QueryRow(`SELECT status,attempts,next_attempt_at,last_http_status FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).
		Scan(&status, &attempts, &next, &httpStatus); err != nil {
		t.Fatal(err)
	}
	if status != "exhausted" || attempts != 1 || next != nil || httpStatus != http.StatusNotFound {
		t.Fatalf("404 state status=%s attempts=%d next=%v http=%d", status, attempts, next, httpStatus)
	}
}
func TestRetryOneResetsOnlyExplicitFailedEvent(t *testing.T) {
	f := newWebhookFixture(t)
	var statusCode int32 = http.StatusNotFound
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(atomic.LoadInt32(&statusCode)))
	}))
	defer server.Close()
	s := newTestService(f, server.URL)
	if _, err := s.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	var eventID string
	if err := f.db.SQL.QueryRow(`SELECT id FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	atomic.StoreInt32(&statusCode, http.StatusOK)
	if err := s.RetryOne(context.Background(), eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	if err := f.db.SQL.QueryRow(`SELECT status,attempts FROM webhook_deliveries WHERE id=?`, eventID).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || attempts != 1 {
		t.Fatalf("manual retry status=%s attempts=%d", status, attempts)
	}
	if err := s.RetryOne(context.Background(), eventID); err == nil {
		t.Fatal("delivered webhook should not be manually requeued")
	}
}

func TestClaimLeaseRecoversAfterInterruptedDelivery(t *testing.T) {
	f := newWebhookFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	s := newTestService(f, server.URL)
	var eventID string
	if err := f.db.SQL.QueryRow(`SELECT id FROM webhook_deliveries WHERE payment_id=?`, f.payment.ID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	claimed, err := s.claim(context.Background(), eventID, *f.now)
	if err != nil || claimed == nil || claimed.Attempts != 1 {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	if processed, err := s.SendPending(context.Background()); err != nil || processed != 0 {
		t.Fatalf("leased event processed=%d err=%v", processed, err)
	}
	*f.now = f.now.Add(defaultLease + time.Second)
	if processed, err := s.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("recovered event processed=%d err=%v", processed, err)
	}
	var status string
	var attempts int
	if err := f.db.SQL.QueryRow(`SELECT status,attempts FROM webhook_deliveries WHERE id=?`, eventID).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || attempts != 2 {
		t.Fatalf("recovered state status=%s attempts=%d", status, attempts)
	}
}

func TestConfigurationValidation(t *testing.T) {
	f := newWebhookFixture(t)
	cases := []Config{
		{Endpoint: "http://example.com/hook", Secret: testSecret},
		{Endpoint: "https://user:pass@example.com/hook", Secret: testSecret},
		{Endpoint: "https://example.com/hook?token=x", Secret: testSecret},
		{Endpoint: "https://example.com/hook", Secret: "short"},
	}
	for _, cfg := range cases {
		s := NewService(f.db, cfg)
		if _, err := s.SendPending(context.Background()); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("config %+v err=%v", cfg, err)
		}
	}
}
