package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/backups"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/reconciliation"
	"github.com/Phloraxx/payment-api/internal/refunds"
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type operationsFixture struct {
	app            *tests.TestApp
	server         *httptest.Server
	token          string
	cfg            config.Config
	now            time.Time
	payments       *payments.Service
	sms            *sms.Service
	reviews        *reviews.Service
	reconciliation *reconciliation.Service
	refunds        *refunds.Service
	backups        *backups.Service
}

func newOperationsFixture(t *testing.T) *operationsFixture {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	operator := core.NewRecord(users)
	operator.SetEmail("operator@example.com")
	operator.SetPassword("test-password-123")
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	token, err := operator.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	cfg := config.Config{
		UPIID: "operator@bank", UPIPayeeName: "PayGate",
		APIKey: "api-secret", SMSWebhookSecret: "sms-secret",
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
		BackupMaxKeep: 3,
	}
	webhookService := webhooks.NewService(app, cfg)
	paymentService := payments.NewService(app, cfg, webhookService)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	auditService := audit.NewService(app)
	auditService.Now = func() time.Time { return now }
	alertService := alerts.NewService(app)
	alertService.Now = func() time.Time { return now }
	reviewService := reviews.NewService(app, paymentService, auditService)
	reviewService.Now = func() time.Time { return now }
	smsService := sms.NewService(app, paymentService)
	smsService.Now = func() time.Time { return now }
	smsService.Reviews = reviewService
	reconciliationService := reconciliation.NewService(app, reviewService, alertService, auditService)
	reconciliationService.Now = func() time.Time { return now }
	refundService := refunds.NewService(app, auditService, webhookService)
	refundService.Now = func() time.Time { return now }
	backupService := backups.NewService(app, cfg, alertService)
	backupService.Now = func() time.Time { return now }

	apiService := New(cfg, paymentService, smsService, nil)
	apiService.Reviews = reviewService
	apiService.Reconciliation = reconciliationService
	apiService.Alerts = alertService
	apiService.Refunds = refundService
	apiService.Backups = backupService
	apiService.Register(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &operationsFixture{
		app: app, server: server, token: token, cfg: cfg, now: now,
		payments: paymentService, sms: smsService, reviews: reviewService,
		reconciliation: reconciliationService, refunds: refundService, backups: backupService,
	}
}

func (f *operationsFixture) request(t *testing.T, method, path string, body io.Reader, contentType string, authenticated bool) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, f.server.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authenticated {
		req.Header.Set("Authorization", f.token)
	}
	res, err := f.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res, data
}

func TestReviewResolutionHTTPFlow(t *testing.T) {
	fixture := newOperationsFixture(t)
	payment, _, err := fixture.payments.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	ingested, err := fixture.sms.Ingest(sms.Input{Source: "gmessages", SourceEventID: "review-http", Body: "Received Rs.100.01 from payer@upi", MessageTime: fixture.now.Add(time.Minute)})
	if err != nil || ingested.ReviewCaseID == "" {
		t.Fatalf("ingested=%+v err=%v", ingested, err)
	}
	body := strings.NewReader(`{"action":"manual_match","paymentId":"` + payment.ID + `","bankReference":"123456789012","note":"Confirmed against downloaded bank statement."}`)
	res, data := fixture.request(t, http.MethodPost, "/api/review-cases/"+ingested.ReviewCaseID+"/resolve", body, "application/json", true)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"status":"resolved"`)) {
		t.Fatalf("status=%d body=%s", res.StatusCode, data)
	}
	stored, _ := fixture.payments.Get(payment.ID)
	if stored.Status != domain.StatusPaid || stored.RRN != "123456789012" {
		t.Fatalf("payment=%+v", stored)
	}
}

func TestReconciliationUploadHTTPFlowAndAuth(t *testing.T) {
	fixture := newOperationsFixture(t)
	payment, _, err := fixture.payments.Create(payments.CreateInput{AmountRupees: 200})
	if err != nil {
		t.Fatal(err)
	}
	csv := "Transaction Date,Credit,Narration,UPI Reference\n" + fixture.now.Add(time.Minute).Format(time.RFC3339) + ",200.01,UPI PAYMENT RECEIVED,999988887777\n"
	makeBody := func() (*bytes.Buffer, string) {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		part, err := writer.CreateFormFile("statement", "statement.csv")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte(csv))
		_ = writer.Close()
		return &buffer, writer.FormDataContentType()
	}
	unauthBody, unauthType := makeBody()
	res, _ := fixture.request(t, http.MethodPost, "/api/reconciliation/import", unauthBody, unauthType, false)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", res.StatusCode)
	}
	authBody, authType := makeBody()
	res, data := fixture.request(t, http.MethodPost, "/api/reconciliation/import", authBody, authType, true)
	if res.StatusCode != http.StatusCreated || !bytes.Contains(data, []byte(`"conflictRows":1`)) || !bytes.Contains(data, []byte(`"reviewCases":1`)) {
		t.Fatalf("status=%d body=%s payment=%s", res.StatusCode, data, payment.ID)
	}
}

func TestRefundHTTPFlowAndIdempotency(t *testing.T) {
	fixture := newOperationsFixture(t)
	payment, _, err := fixture.payments.Create(payments.CreateInput{AmountRupees: 300})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.payments.Match(domain.ParsedSMS{AmountPaise: payment.PayablePaise, RRN: "300300300300", OccurredAt: fixture.now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	payload := `{"paymentId":"` + payment.ID + `","amountPaise":10000,"reason":"Customer return"}`
	req, err := http.NewRequest(http.MethodPost, fixture.server.URL+"/api/refunds", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer api-secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "refund-http-idem")
	res, err := fixture.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", res.StatusCode, data)
	}
	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatal(err)
	}
	refundID, _ := created["id"].(string)
	res, data = fixture.request(t, http.MethodPost, "/api/refunds/"+refundID+"/status", strings.NewReader(`{"status":"completed","reference":"REFUND123456","note":"Verified bank transfer."}`), "application/json", true)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"status":"completed"`)) {
		t.Fatalf("update status=%d body=%s", res.StatusCode, data)
	}
}

func TestDashboardCapacityAndBackupOperatorRoutes(t *testing.T) {
	fixture := newOperationsFixture(t)
	for i := 0; i < 70; i++ {
		if _, _, err := fixture.payments.Create(payments.CreateInput{AmountRupees: 400}); err != nil {
			t.Fatal(err)
		}
	}
	res, data := fixture.request(t, http.MethodGet, "/api/dashboard", nil, "", true)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"level":"warning"`)) || !bytes.Contains(data, []byte(`"openReviewCount":0`)) {
		t.Fatalf("dashboard status=%d body=%s", res.StatusCode, data)
	}
	res, data = fixture.request(t, http.MethodPost, "/api/paygate/backups", nil, "", true)
	if res.StatusCode != http.StatusCreated || !bytes.Contains(data, []byte("paygate_manual_20260801_080000.zip")) {
		t.Fatalf("backup status=%d body=%s", res.StatusCode, data)
	}
	res, data = fixture.request(t, http.MethodPost, "/api/paygate/backups/verify", nil, "", true)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"latestVerified":true`)) {
		t.Fatalf("verify status=%d body=%s", res.StatusCode, data)
	}
	res, data = fixture.request(t, http.MethodPost, "/api/paygate/backups/restore-drill", nil, "", true)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"integrityChecked":`)) {
		t.Fatalf("drill status=%d body=%s", res.StatusCode, data)
	}
}

func TestConfigReportsOperationalFeatureFlags(t *testing.T) {
	fixture := newOperationsFixture(t)
	res, data := fixture.request(t, http.MethodGet, "/api/config", nil, "", true)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.StatusCode, data)
	}
	if !bytes.Contains(data, []byte(`"operatorAlertWebhookConfigured":false`)) {
		t.Fatalf("missing operator alert flag: %s", data)
	}
}

func TestOperatorSPAUsesSecurityHeadersAndRejectsUnknownBrowserRoutes(t *testing.T) {
	fixture := newOperationsFixture(t)
	res, data := fixture.request(t, http.MethodGet, "/", nil, "", false)
	if res.StatusCode != http.StatusOK || !bytes.Contains(data, []byte("PayGate")) {
		t.Fatalf("root status=%d body=%s", res.StatusCode, data)
	}
	for _, name := range []string{"Content-Security-Policy", "Permissions-Policy", "Referrer-Policy", "Strict-Transport-Security", "X-Robots-Tag"} {
		if res.Header.Get(name) == "" {
			t.Fatalf("missing %s", name)
		}
	}
	res, robots := fixture.request(t, http.MethodGet, "/robots.txt", nil, "", false)
	if res.StatusCode != http.StatusOK || !bytes.Contains(robots, []byte("User-agent: *")) || res.Header.Get("X-Robots-Tag") == "" {
		t.Fatalf("robots status=%d header=%q body=%s", res.StatusCode, res.Header.Get("X-Robots-Tag"), robots)
	}
	for _, path := range []string{"/contact/", "/sitemap.xml"} {
		res, _ := fixture.request(t, http.MethodGet, path, nil, "", false)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("%s status=%d", path, res.StatusCode)
		}
	}
	res, _ = fixture.request(t, http.MethodGet, "/api/config", nil, "", true)
	if res.Header.Get("X-Robots-Tag") == "" {
		t.Fatal("missing X-Robots-Tag on API response")
	}
}
