package alerts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOpenDeduplicatesAndResolveCloses(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	service := NewService(app)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	id, created, err := service.Open(Input{Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:daily", Message: "backup failed"})
	if err != nil || !created {
		t.Fatalf("id=%s created=%v err=%v", id, created, err)
	}
	now = now.Add(time.Minute)
	second, created, err := service.Open(Input{Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:daily", Message: "backup failed again"})
	if err != nil || created || second != id {
		t.Fatalf("second=%s created=%v err=%v", second, created, err)
	}
	record, _ := app.FindRecordById("alerts", id)
	if record.GetInt("occurrence_count") != 2 {
		t.Fatalf("occurrences=%d", record.GetInt("occurrence_count"))
	}
	if err := service.Resolve("backup:daily"); err != nil {
		t.Fatal(err)
	}
	record, _ = app.FindRecordById("alerts", id)
	if record.GetString("status") != "resolved" {
		t.Fatalf("status=%s", record.GetString("status"))
	}
}

func TestConnectorGracePeriodsAndCapacityAlerts(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	service := NewService(app)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	status := gmessages.Status{Enabled: true, Paired: true, Connected: false, PhoneResponsive: false, State: "disconnected"}
	if err := service.CheckConnector(status); err != nil {
		t.Fatal(err)
	}
	if count, _ := service.OpenCount(); count != 0 {
		t.Fatalf("alerts before grace=%d", count)
	}
	now = now.Add(16 * time.Minute)
	if err := service.CheckConnector(status); err != nil {
		t.Fatal(err)
	}
	if count, _ := service.OpenCount(); count != 2 {
		t.Fatalf("alerts after grace=%d", count)
	}
	if err := service.CheckCapacity(payments.CapacitySnapshot{Pools: []payments.CapacityPool{{RequestedAmountPaise: 10000, RequestedAmount: "100.00", Blocked: 95, UtilizationPercent: 95.96, Level: "critical"}}}); err != nil {
		t.Fatal(err)
	}
	if count, _ := service.OpenCount(); count != 3 {
		t.Fatalf("alerts after capacity=%d", count)
	}
}

func TestSignedOperatorAlertWebhookDeliversOpenAndResolvedWithoutSpam(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("X-PayGate-Timestamp")
		want := "v1=" + webhooks.Sign("operator-alert-secret-1234567890", timestamp, body)
		if got := r.Header.Get("X-PayGate-Signature"); got != want {
			t.Errorf("signature=%q want=%q", got, want)
		}
		if r.Header.Get("X-PayGate-Event-Id") == "" {
			t.Error("missing event id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	service := NewService(app)
	service.ConfigureWebhook(server.URL, "operator-alert-secret-1234567890")
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	id, _, err := service.Open(Input{Kind: "connector_reauth", Severity: "critical", DedupeKey: "connector:reauth", Message: "reauth required"})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	record, _ := app.FindRecordById("alerts", id)
	if record.GetString("notification_status") != "delivered" {
		t.Fatalf("status=%s", record.GetString("notification_status"))
	}
	if _, _, err := service.Open(Input{Kind: "connector_reauth", Severity: "critical", DedupeKey: "connector:reauth", Message: "still reauth required"}); err != nil {
		t.Fatal(err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 0 {
		t.Fatalf("repeat processed=%d err=%v", processed, err)
	}
	if err := service.Resolve("connector:reauth"); err != nil {
		t.Fatal(err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("resolve processed=%d err=%v", processed, err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d", requests)
	}
}

func TestOperatorAlertWebhookFailureSchedulesRetry(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }))
	defer server.Close()
	service := NewService(app)
	service.ConfigureWebhook(server.URL, "operator-alert-secret-1234567890")
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	id, _, err := service.Open(Input{Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:create", Message: "backup failed"})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	record, _ := app.FindRecordById("alerts", id)
	if record.GetString("notification_status") != "failed" || record.GetInt("notification_attempts") != 1 {
		t.Fatalf("status=%s attempts=%d", record.GetString("notification_status"), record.GetInt("notification_attempts"))
	}
	if next := record.GetDateTime("notification_next_attempt_at").Time(); !next.Equal(now.Add(time.Minute)) {
		t.Fatalf("next=%s", next)
	}
}

func TestSeverityEscalationQueuesNewNotification(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	service := NewService(app)
	service.ConfigureWebhook(server.URL, "operator-alert-secret-1234567890")
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	if _, _, err := service.Open(Input{Kind: "capacity_high", Severity: "warning", DedupeKey: "capacity:10000", Message: "warning"}); err != nil {
		t.Fatal(err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("warning processed=%d err=%v", processed, err)
	}
	now = now.Add(time.Minute)
	if _, created, err := service.Open(Input{Kind: "capacity_high", Severity: "critical", DedupeKey: "capacity:10000", Message: "critical"}); err != nil || created {
		t.Fatalf("critical created=%v err=%v", created, err)
	}
	if processed, err := service.SendPending(context.Background()); err != nil || processed != 1 {
		t.Fatalf("critical processed=%d err=%v", processed, err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d", requests)
	}
}

func TestOperatorAlertClientDoesNotFollowRedirects(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	var redirected int
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer destination.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()
	service := NewService(app)
	service.ConfigureWebhook(redirector.URL, "operator-alert-secret-1234567890")
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	id, _, err := service.Open(Input{Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:redirect", Message: "redirect test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SendPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if redirected != 0 {
		t.Fatalf("redirect destination received %d requests", redirected)
	}
	record, _ := app.FindRecordById("alerts", id)
	if record.GetString("notification_status") != "failed" {
		t.Fatalf("notification status=%s", record.GetString("notification_status"))
	}
}
