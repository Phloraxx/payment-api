package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const checkoutOrigin = "https://payment.example.com"

func checkoutHTTPServer(t *testing.T, configure func(*config.Config), before func(*tests.TestApp, *payments.Service)) (*tests.TestApp, *httptest.Server) {
	t.Helper()
	app := apiTestFactoryWithConfig(t, configure, before)
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
	t.Cleanup(app.Cleanup)
	return app, server
}

func checkoutRequest(t *testing.T, server *httptest.Server, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := server.Client().Do(req)
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

func enabledCheckoutConfig(cfg *config.Config) {
	cfg.CheckoutAllowedOrigins = []string{checkoutOrigin}
	cfg.DefaultPaymentAccount = "kotak"
	cfg.KotakUPIID = "operator@kotak"
}

func TestCheckoutSurfaceDisabledByDefault(t *testing.T) {
	_, server := checkoutHTTPServer(t, nil, nil)
	res, _ := checkoutRequest(t, server, http.MethodGet, "/api/checkout/v2/payment-accounts", "", map[string]string{"Origin": checkoutOrigin})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled checkout status=%d; want 404", res.StatusCode)
	}
}

func TestCheckoutCORSAllowlistAndPreflight(t *testing.T) {
	_, server := checkoutHTTPServer(t, enabledCheckoutConfig, nil)
	res, _ := checkoutRequest(t, server, http.MethodOptions, "/api/checkout/v2/payments", "", map[string]string{"Origin": checkoutOrigin})
	if res.StatusCode != http.StatusNoContent || res.Header.Get("Access-Control-Allow-Origin") != checkoutOrigin {
		t.Fatalf("allowed preflight status=%d origin=%q", res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"))
	}
	res, _ = checkoutRequest(t, server, http.MethodGet, "/api/checkout/v2/payment-accounts", "", map[string]string{"Origin": "https://evil.example"})
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("foreign origin status=%d cors=%q", res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"))
	}
}
func TestCheckoutCreateReplaysIdempotentlyAndRedactsStatus(t *testing.T) {
	_, server := checkoutHTTPServer(t, enabledCheckoutConfig, nil)
	headers := map[string]string{
		"Origin":          checkoutOrigin,
		"Content-Type":    "application/json",
		"Idempotency-Key": "2f54d1c8-4ef4-4c21-9ff8-b9f4fc8e79a1",
	}
	body := `{"amount":100,"paymentAccount":"kotak"}`
	first, firstBody := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", body, headers)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.StatusCode, firstBody)
	}
	var payment map[string]any
	if err := json.Unmarshal(firstBody, &payment); err != nil {
		t.Fatal(err)
	}
	id, _ := payment["id"].(string)
	if id == "" {
		t.Fatalf("missing payment id: %s", firstBody)
	}
	replay, replayBody := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", body, headers)
	if replay.StatusCode != http.StatusOK || replay.Header.Get("X-Idempotent-Replayed") != "true" {
		t.Fatalf("replay status=%d replay=%q body=%s", replay.StatusCode, replay.Header.Get("X-Idempotent-Replayed"), replayBody)
	}
	status, statusBody := checkoutRequest(t, server, http.MethodGet, "/api/checkout/v2/payments/"+id, "", map[string]string{"Origin": checkoutOrigin})
	if status.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", status.StatusCode, statusBody)
	}
	for _, forbidden := range []string{`"rrn"`, `"payerName"`, `"upiId"`, `"rawSms"`} {
		if strings.Contains(string(statusBody), forbidden) {
			t.Fatalf("public checkout response exposed %s: %s", forbidden, statusBody)
		}
	}
}
func TestCheckoutRejectsInvalidRequestsBeforeQuota(t *testing.T) {
	_, server := checkoutHTTPServer(t, enabledCheckoutConfig, nil)
	headers := map[string]string{
		"Origin":          checkoutOrigin,
		"Content-Type":    "application/json",
		"Idempotency-Key": "2f54d1c8-4ef4-4c21-9ff8-b9f4fc8e79a2",
	}
	cases := []string{
		`{"amount":0,"paymentAccount":"kotak"}`,
		`{"amount":100,"paymentAccount":"unknown"}`,
		`{"amount":100,"paymentAccount":"kotak","extra":true}`,
	}
	for _, body := range cases {
		res, _ := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", body, headers)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid body %s status=%d", body, res.StatusCode)
		}
	}
	for i := 0; i < 5; i++ {
		headers["Idempotency-Key"] = "00000000-0000-4000-8000-00000000000" + string(rune('1'+i))
		res, body := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", `{"amount":100,"paymentAccount":"kotak"}`, headers)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("valid create #%d status=%d body=%s", i+1, res.StatusCode, body)
		}
	}
}
func TestCheckoutCreationRateLimitAndRetryAfter(t *testing.T) {
	_, server := checkoutHTTPServer(t, enabledCheckoutConfig, nil)
	ids := []string{
		"10000000-0000-4000-8000-000000000001",
		"10000000-0000-4000-8000-000000000002",
		"10000000-0000-4000-8000-000000000003",
		"10000000-0000-4000-8000-000000000004",
		"10000000-0000-4000-8000-000000000005",
		"10000000-0000-4000-8000-000000000006",
	}
	for i, id := range ids {
		res, body := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", `{"amount":101,"paymentAccount":"kotak"}`, map[string]string{
			"Origin": checkoutOrigin, "Content-Type": "application/json", "Idempotency-Key": id,
		})
		if i < 5 && res.StatusCode != http.StatusCreated {
			t.Fatalf("create #%d status=%d body=%s", i+1, res.StatusCode, body)
		}
		if i == 5 {
			if res.StatusCode != http.StatusTooManyRequests || res.Header.Get("Retry-After") == "" || !strings.Contains(string(body), `"code":"RATE_LIMITED"`) {
				t.Fatalf("limited status=%d retry=%q body=%s", res.StatusCode, res.Header.Get("Retry-After"), body)
			}
		}
	}
}

func TestCheckoutGlobalQuotaDoesNotConsumePerIPOnRejection(t *testing.T) {
	now := time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC)
	quota := checkoutQuota{perIP: map[string]checkoutBucket{}, perIPLimit: 5, globalLimit: 1, perIPWindow: 5 * time.Minute, globalWindow: time.Minute}
	if allowed, _ := quota.allow("198.51.100.1", now); !allowed {
		t.Fatal("first global slot should be allowed")
	}
	if allowed, _ := quota.allow("198.51.100.2", now); allowed {
		t.Fatal("second client should be rejected by global quota")
	}
	if got := quota.perIP["198.51.100.2"].count; got != 0 {
		t.Fatalf("global rejection consumed per-IP quota: count=%d", got)
	}
}
func TestCheckoutUnavailablePaytmFailsClosed(t *testing.T) {
	_, server := checkoutHTTPServer(t, func(cfg *config.Config) {
		enabledCheckoutConfig(cfg)
		cfg.PaytmUPIID = "merchant@paytm"
	}, nil)
	res, body := checkoutRequest(t, server, http.MethodPost, "/api/checkout/v2/payments", `{"amount":100,"paymentAccount":"paytm"}`, map[string]string{
		"Origin":          checkoutOrigin,
		"Content-Type":    "application/json",
		"Idempotency-Key": "20000000-0000-4000-8000-000000000001",
	})
	if res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "PAYMENT_ACCOUNT_UNAVAILABLE") {
		t.Fatalf("Paytm unavailable status=%d body=%s", res.StatusCode, body)
	}
}

func TestCheckoutInvalidPaymentIDDoesNotConsumeStatusQuota(t *testing.T) {
	_, server := checkoutHTTPServer(t, enabledCheckoutConfig, nil)
	for i := 0; i < 5; i++ {
		res, _ := checkoutRequest(t, server, http.MethodGet, "/api/checkout/v2/payments/!", "", map[string]string{"Origin": checkoutOrigin})
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid id request #%d status=%d", i+1, res.StatusCode)
		}
	}
	res, body := checkoutRequest(t, server, http.MethodGet, "/api/checkout/v2/payment-accounts", "", map[string]string{"Origin": checkoutOrigin})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("valid status request was throttled after invalid IDs: status=%d body=%s", res.StatusCode, body)
	}
}
