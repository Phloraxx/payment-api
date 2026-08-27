package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/androidrelay"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/paymentemail"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/paytmnotification"
	"github.com/Phloraxx/payment-api/internal/sms"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/rs/zerolog"
)

const testEmailWebhookSecret = "email-webhook-secret-long-enough"

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
	apiService := New(cfg, paymentService, smsService, nil)
	apiService.PaytmNotifications = paytmnotification.NewService(app, paymentService)
	apiService.AndroidRelay = androidrelay.NewService(app, apiService.PaytmNotifications)
	apiService.Email = paymentemail.NewService(app, paymentService, cfg.EmailAllowedSender, cfg.EmailAuthServID)
	apiService.Register(app)
	return app
}

func apiTestFactoryWithGMessages(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create PocketBase test app: %v", err)
	}
	cfg := config.Config{
		UPIID: "operator@bank", UPIPayeeName: "PayGate",
		APIKey: "api-secret", SMSWebhookSecret: "sms-secret",
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
		GMessagesEnabled: true, GMessagesSessionPath: filepath.Join(t.TempDir(), "session.json"),
	}
	paymentService := payments.NewService(app, cfg, nil)
	smsService := sms.NewService(app, paymentService)
	manager := gmessages.NewManager(cfg, zerolog.Nop(), nil)
	New(cfg, paymentService, smsService, manager).Register(app)
	return app
}

func TestPaymentEmailWebhookSignatureAndDeduplication(t *testing.T) {
	var paymentID string
	app := apiTestFactoryWithConfig(t, func(cfg *config.Config) {
		cfg.EmailEvidenceEnabled = true
		cfg.DefaultPaymentAccount = "kotak"
		cfg.KotakUPIID = "operator@kotak"
		cfg.SliceUPIID = "operator@slice"
		cfg.EmailWebhookSecret = testEmailWebhookSecret
		cfg.EmailAllowedSender = "noreply@slice.bank.in"
		cfg.EmailAuthServID = "mx.cloudflare.net"
		cfg.EmailSignatureTolerance = 5 * time.Minute
	}, func(_ *tests.TestApp, service *payments.Service) {
		payment, _, err := service.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
		if err != nil {
			t.Fatal(err)
		}
		paymentID = payment.ID
	})
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

	now := time.Now().UTC()
	raw := "From: slice <noreply@slice.bank.in>\r\n" +
		"Message-ID: <api-email-1@example>\r\n" +
		"Date: " + now.Format(time.RFC1123Z) + "\r\n" +
		"Subject: Received INR 100.01 in your slice account\r\n" +
		"Authentication-Results: mx.cloudflare.net; dkim=pass header.d=slice.bank.in\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Received INR 100.01 via UPI from alice@oksbi. UPI Ref:606703736479"
	payload, err := json.Marshal(emailBody{
		SourceID: "api-email-1@example", EnvelopeFrom: "noreply@slice.bank.in", EnvelopeTo: "payments@example.org",
		ReceivedAt: now.Format(time.RFC3339), RawEmailBase64: base64.StdEncoding.EncodeToString([]byte(raw)),
	})
	if err != nil {
		t.Fatal(err)
	}

	send := func(signatureSecret string, timestamp time.Time) (int, string) {
		t.Helper()
		timestampText := timestamp.Unix()
		mac := hmac.New(sha256.New, []byte(signatureSecret))
		_, _ = mac.Write([]byte(strconv.FormatInt(timestampText, 10) + "."))
		_, _ = mac.Write(payload)
		req, requestErr := http.NewRequest(http.MethodPost, server.URL+"/api/events/email", strings.NewReader(string(payload)))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-PayGate-Timestamp", strconv.FormatInt(timestampText, 10))
		req.Header.Set("X-PayGate-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		response, requestErr := server.Client().Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		responseBody, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(responseBody)
	}

	if status, body := send("wrong-secret", now); status != http.StatusUnauthorized {
		t.Fatalf("bad signature status=%d body=%s", status, body)
	}
	if count, _ := app.CountRecords("email_events"); count != 0 {
		t.Fatalf("unauthenticated request persisted %d events", count)
	}
	if status, body := send(testEmailWebhookSecret, now.Add(-10*time.Minute)); status != http.StatusUnauthorized {
		t.Fatalf("stale signature status=%d body=%s", status, body)
	}
	if status, body := send(testEmailWebhookSecret, now); status != http.StatusAccepted || !strings.Contains(body, paymentID) || !strings.Contains(body, "marked_paid") {
		t.Fatalf("valid email status=%d body=%s", status, body)
	}
	if status, body := send(testEmailWebhookSecret, now); status != http.StatusOK || !strings.Contains(body, "duplicate_event") {
		t.Fatalf("duplicate email status=%d body=%s", status, body)
	}
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
				record.Set("payment_account", "kotak")
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

func TestGoogleMessagesAccountPairEndpointValidation(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name: "Google pair requires dashboard auth", Method: http.MethodPost, URL: "/api/connector/gmessages/pair/google",
			Body:           strings.NewReader(`{"cookieData":"SID=missing-rest"}`),
			TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactoryWithGMessages(t) },
			ExpectedStatus: http.StatusUnauthorized, ExpectedContent: []string{"Dashboard authentication is required."},
		},
		{
			Name: "payment API key cannot control Google pairing", Method: http.MethodPost, URL: "/api/connector/gmessages/pair/google",
			Headers:        map[string]string{"Authorization": "Bearer api-secret"},
			Body:           strings.NewReader(`{"cookieData":"SID=missing-rest"}`),
			TestAppFactory: func(t testing.TB) *tests.TestApp { return apiTestFactoryWithGMessages(t) },
			ExpectedStatus: http.StatusUnauthorized, ExpectedContent: []string{"Dashboard authentication is required."},
		},
	}
	// API keys intentionally do not grant dashboard-only connector access.
	for i := range scenarios {
		scenarios[i].Test(t)
	}
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
				record.Set("payment_account", "kotak")
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
		ExpectedContent: []string{`"status":"review_required"`, `"action":"unmatched"`},
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

func TestPaytmNotificationWebhookAuthenticatesAndMatches(t *testing.T) {
	var paymentID string
	app := apiTestFactoryWithConfig(t, func(cfg *config.Config) {
		cfg.PaytmQRPayload = "paytm-issued-static-qr"
		cfg.PaytmNotificationWebhookSecret = "paytm-notification-secret-long-enough"
	}, func(_ *tests.TestApp, service *payments.Service) {
		payment, _, err := service.Create(payments.CreateInput{AmountRupees: 1, PaymentAccount: "paytm"})
		if err != nil {
			t.Fatal(err)
		}
		paymentID = payment.ID
	})
	defer app.Cleanup()

	bad := tests.ApiScenario{
		Name: "bad Paytm webhook secret", Method: http.MethodPost, URL: "/api/events/paytm-notification",
		Headers: map[string]string{"X-Webhook-Secret": "wrong", "Content-Type": "application/json"},
		Body:    strings.NewReader(`{"sourceId":"evt-bad","appPackage":"com.paytm.business","body":"₹1.01 paid by Test","notificationTimestampMs":"1787763600000"}`),
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return apiTestFactoryWithConfig(t, func(cfg *config.Config) {
				cfg.PaytmQRPayload = "qr"
				cfg.PaytmNotificationWebhookSecret = "paytm-notification-secret-long-enough"
			}, nil)
		},
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{"Invalid webhook secret."},
	}
	bad.Test(t)

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
	payload := `{"sourceId":"evt-1","appPackage":"com.paytm.business","appName":"Paytm for Business","title":"Payment received","body":"₹1.01 paid by Test User","notificationTimestampMs":"` + strconv.FormatInt(time.Now().UnixMilli(), 10) + `"}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/events/paytm-notification", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", "paytm-notification-secret-long-enough")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted || !strings.Contains(string(body), `"status":"matched"`) {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	record, err := app.FindRecordById("payments", paymentID)
	if err != nil || record.GetString("status") != "paid" || record.GetString("evidence_source") != "paytm_notification" {
		t.Fatalf("payment status=%v err=%v", record, err)
	}
}
