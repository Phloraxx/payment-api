package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/sms"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func apiTestFactory(t testing.TB, before func(*tests.TestApp, *payments.Service)) *tests.TestApp {
	return apiTestFactoryWithConfig(t, nil, before)
}

func apiTestFactoryWithConfig(t testing.TB, configure func(*config.Config), before func(*tests.TestApp, *payments.Service)) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create PocketBase test app: %v", err)
	}
	cfg := config.Config{
		UPIID: "operator@bank", UPIPayeeName: "PayGate",
		APIKey: "api-secret", SMSWebhookSecret: "sms-secret",
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
	}
	if configure != nil {
		configure(&cfg)
	}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	smsService := sms.NewService(app, paymentService)
	if before != nil {
		before(app, paymentService)
	}
	New(cfg, paymentService, smsService, nil).Register(app)
	return app
}

func TestPaymentAPIAuthenticationAndAmountValidation(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name: "create requires auth", Method: http.MethodPost, URL: "/api/payments",
			Body:            strings.NewReader(`{"amount":100}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{"API key or dashboard authentication is required"},
		},
		{
			Name: "fractional amount rejected", Method: http.MethodPost, URL: "/api/payments",
			Headers:         map[string]string{"Authorization": "Bearer api-secret", "Content-Type": "application/json"},
			Body:            strings.NewReader(`{"amount":100.01}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"INVALID_AMOUNT"},
		},
		{
			Name: "unknown JSON field rejected", Method: http.MethodPost, URL: "/api/payments",
			Headers:         map[string]string{"Authorization": "Bearer api-secret", "Content-Type": "application/json"},
			Body:            strings.NewReader(`{"amount":100,"surprise":true}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Invalid JSON body."},
		},
		{
			Name: "oversized payment body rejected", Method: http.MethodPost, URL: "/api/payments",
			Headers:         map[string]string{"Authorization": "Bearer api-secret", "Content-Type": "application/json"},
			Body:            strings.NewReader(`{"amount":100,"metadata":"` + strings.Repeat("a", int(maxPaymentRequestBytes)) + `"}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
			ExpectedStatus:  http.StatusRequestEntityTooLarge,
			ExpectedContent: []string{"Request entity too large"},
		},
		{
			Name: "valid whole rupee payment", Method: http.MethodPost, URL: "/api/payments",
			Headers:         map[string]string{"Authorization": "Bearer api-secret", "Content-Type": "application/json", "Idempotency-Key": "http-test-idem"},
			Body:            strings.NewReader(`{"amount":100,"externalId":"order-http"}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
			ExpectedStatus:  http.StatusCreated,
			ExpectedContent: []string{`"requestedAmount":100`, `"payableAmount":"100.01"`, `"externalId":"order-http"`, `upi://pay?`},
		},
	}
	for i := range scenarios {
		scenarios[i].Test(t)
	}
}

func TestPaymentCreateThroughRealHTTPServer(t *testing.T) {
	app := apiTestFactory(t, nil)
	defer app.Cleanup()

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
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/payments", strings.NewReader(`{"amount":100,"externalId":"network-http"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer api-secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "network-http-idem")

	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", res.StatusCode, body)
	}
	content := string(body)
	for _, want := range []string{`"requestedAmount":100`, `"externalId":"network-http"`, `upi://pay?`} {
		if !strings.Contains(content, want) {
			t.Fatalf("response missing %q: %s", want, content)
		}
	}
}

func TestPublicPaymentStatusRedactsSensitiveEvidence(t *testing.T) {
	const paymentID = "paytest00000001"
	scenario := tests.ApiScenario{
		Name: "public status redacts payer evidence", Method: http.MethodGet, URL: "/api/payments/" + paymentID,
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return apiTestFactory(t, func(app *tests.TestApp, _ *payments.Service) {
				collection, err := app.FindCollectionByNameOrId("payments")
				if err != nil {
					t.Fatal(err)
				}
				record := core.NewRecord(collection)
				record.Id = paymentID
				record.Set("created_at", time.Now().Add(-time.Minute))
				record.Set("requested_amount", 10000)
				record.Set("payable_amount", 10001)
				record.Set("status", "paid")
				record.Set("expires_at", time.Now().Add(time.Minute))
				record.Set("reuse_after", time.Now().Add(time.Hour))
				record.Set("rrn", "123456789012")
				record.Set("upi_id", "private@upi")
				record.Set("payer_name", "Private Person")
				record.Set("paid_at", time.Now())
				record.Set("external_id", "private-order-123")
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
			})
		},
		ExpectedStatus:     http.StatusOK,
		ExpectedContent:    []string{`"id":"` + paymentID + `"`, `"status":"paid"`, `"payableAmount":"100.01"`},
		NotExpectedContent: []string{"123456789012", "private@upi", "Private Person", "private-order-123", `"rrn"`, `"payerName"`, `"externalId"`},
	}
	scenario.Test(t)
}

func TestLegacySMSWebhookMatchesPayment(t *testing.T) {
	const paymentID = "smstest00000001"
	scenario := tests.ApiScenario{
		Name: "legacy sms webhook", Method: http.MethodPost, URL: "/api/events/sms",
		Headers: map[string]string{"X-Webhook-Secret": "sms-secret", "Content-Type": "application/json"},
		Body:    strings.NewReader(`{"sms":"Confirmed payment for Received Rs.100.01 in your Kotak Bank AC X4959 from user@oksbi.UPI Ref:606703736479."}`),
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return apiTestFactory(t, func(app *tests.TestApp, _ *payments.Service) {
				collection, err := app.FindCollectionByNameOrId("payments")
				if err != nil {
					t.Fatal(err)
				}
				record := core.NewRecord(collection)
				record.Id = paymentID
				record.Set("created_at", time.Now().Add(-time.Minute))
				record.Set("requested_amount", 10000)
				record.Set("payable_amount", 10001)
				record.Set("status", "pending")
				record.Set("expires_at", time.Now().Add(5*time.Minute))
				record.Set("reuse_after", time.Now().Add(24*time.Hour))
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
			})
		},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"status":"matched"`, `"action":"marked_paid"`, `"paymentId":"` + paymentID + `"`},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			record, err := app.FindRecordById("payments", paymentID)
			if err != nil {
				t.Fatal(err)
			}
			if record.GetString("status") != "paid" || record.GetString("rrn") != "606703736479" {
				t.Fatalf("payment after webhook = status=%s rrn=%s", record.GetString("status"), record.GetString("rrn"))
			}
		},
	}
	scenario.Test(t)
}

func TestSMSWebhookSecretAndHealthRoutes(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{Name: "bad sms secret", Method: http.MethodPost, URL: "/api/events/sms", Headers: map[string]string{"X-Webhook-Secret": "wrong", "Content-Type": "application/json"}, Body: strings.NewReader(`{"sms":"Received Rs.100.01 UPI Ref:123456789012"}`), TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) }, ExpectedStatus: http.StatusUnauthorized, ExpectedContent: []string{"Invalid webhook secret."}},
		{Name: "rich health", Method: http.MethodGet, URL: "/api/paygate/health", TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) }, ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"status":"healthy"`, `"db":"ok"`}},
		{Name: "PocketBase health remains available", Method: http.MethodGet, URL: "/api/health", TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) }, ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"code":200`}},
		{Name: "invalid sms source", Method: http.MethodPost, URL: "/api/events/sms", Headers: map[string]string{"X-Webhook-Secret": "sms-secret", "Content-Type": "application/json"}, Body: strings.NewReader(`{"sms":"Received Rs.100.01 UPI Ref:123456789012","source":"bogus"}`), TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) }, ExpectedStatus: http.StatusBadRequest, ExpectedContent: []string{"INVALID_SMS_SOURCE"}},
		{Name: "unknown API path stays JSON 404", Method: http.MethodGet, URL: "/api/not-a-paygate-route", TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) }, ExpectedStatus: http.StatusNotFound, NotExpectedContent: []string{"<title>PayGate</title>"}},
	}
	for i := range scenarios {
		scenarios[i].Test(t)
	}
}

func TestLegacyWebhookAliasIsExplicitlyGated(t *testing.T) {
	disabled := tests.ApiScenario{
		Name: "legacy webhook disabled", Method: http.MethodPost, URL: "/api/webhook",
		Headers:         map[string]string{"X-Webhook-Secret": "legacy-secret", "Content-Type": "application/json"},
		Body:            strings.NewReader(`{"sms":"Received Rs.777.77 from user@oksbi UPI Ref:777788889999"}`),
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactory(t, nil) },
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{"Route not found."},
	}
	disabled.Test(t)

	enabled := tests.ApiScenario{
		Name: "legacy webhook enabled", Method: http.MethodPost, URL: "/api/webhook",
		Headers: map[string]string{"X-Webhook-Secret": "legacy-secret", "Content-Type": "application/json"},
		Body:    strings.NewReader(`{"sms":"Received Rs.777.77 from user@oksbi UPI Ref:777788889999","source":"gmessages"}`),
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return apiTestFactoryWithConfig(t, func(cfg *config.Config) {
				cfg.LegacySMSWebhookEnabled = true
				cfg.LegacySMSWebhookSecret = "legacy-secret"
			}, nil)
		},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"status":"unmatched"`},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			records, err := app.FindRecordsByFilter("sms_events", "", "-created", 1, 0)
			if err != nil || len(records) != 1 {
				t.Fatalf("legacy event records = %d, %v", len(records), err)
			}
			if got := records[0].GetString("source"); got != "android_webhook" {
				t.Fatalf("legacy route source = %q; want android_webhook", got)
			}
		},
	}
	enabled.Test(t)
}
