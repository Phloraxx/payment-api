package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/razorpaylive"
	"github.com/Phloraxx/payment-api/internal/sms"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type apiRazorpayLiveProvider struct {
	order   razorpaylive.ProviderOrder
	payment razorpaylive.ProviderPayment
}

func (p *apiRazorpayLiveProvider) CreateOrder(_ context.Context, amount int64, receipt string) (razorpaylive.ProviderOrder, error) {
	order := p.order
	if order.ID == "" {
		order = razorpaylive.ProviderOrder{ID: "order_api_live", Amount: amount, Currency: "INR", Receipt: receipt, Status: "created"}
	}
	return order, nil
}

func (p *apiRazorpayLiveProvider) FetchPayment(_ context.Context, paymentID string) (razorpaylive.ProviderPayment, error) {
	payment := p.payment
	payment.ID = paymentID
	return payment, nil
}

type razorpayLiveAPIFixture struct {
	app      *tests.TestApp
	server   *httptest.Server
	token    string
	apiKey   string
	service  *razorpaylive.Service
	provider *apiRazorpayLiveProvider
}

func newRazorpayLiveAPIFixture(t *testing.T, enabled bool) *razorpayLiveAPIFixture {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	users, _ := app.FindCollectionByNameOrId("users")
	operator := core.NewRecord(users)
	operator.SetEmail("razorpay@example.com")
	operator.SetPassword("test-password-123")
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	token, _ := operator.NewAuthToken()
	cfg := config.Config{
		TestMode: true, APIKey: "razorpay-live-api-key-1234567890123456", PaymentTTL: 5, AmountQuarantine: 0,
		RazorpayLiveEnabled: enabled, RazorpayLiveKeyID: "rzp_live_api",
		RazorpayLiveKeySecret: "checkout-secret-123456", RazorpayLiveWebhookSecret: "webhook-secret-123456789012",
		RazorpayLiveDisplayName: "PayGate Live",
		CheckoutAllowedOrigins:  []string{checkoutOrigin},
	}
	paymentService := payments.NewService(app, cfg, nil)
	smsService := sms.NewService(app, paymentService)
	provider := &apiRazorpayLiveProvider{}
	service := razorpaylive.NewService(app, provider, cfg.RazorpayLiveKeyID, cfg.RazorpayLiveKeySecret, cfg.RazorpayLiveWebhookSecret, cfg.RazorpayLiveDisplayName)
	apiService := New(cfg, paymentService, smsService, nil)
	if enabled {
		apiService.RazorpayLive = service
	}
	apiService.Register(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, _ := serveEvent.Router.BuildMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &razorpayLiveAPIFixture{app: app, server: server, token: token, apiKey: cfg.APIKey, service: service, provider: provider}
}

func (f *razorpayLiveAPIFixture) request(t *testing.T, method, path, body string, authenticated bool, headers map[string]string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		req.Header.Set("Authorization", f.token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := f.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res, string(raw)
}

func (f *razorpayLiveAPIFixture) apiKeyRequest(t *testing.T, method, path, body string, headers map[string]string) (*http.Response, string) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Authorization"] = "Bearer " + f.apiKey
	return f.request(t, method, path, body, false, headers)
}

func TestRazorpayLiveRoutesAreDisabledByDefault(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, false)
	res, body := fixture.request(t, http.MethodGet, "/api/razorpay/live/config", "", true, nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("config status=%d body=%s", res.StatusCode, body)
	}
	res, _ = fixture.request(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":100}`, true, map[string]string{"Idempotency-Key": "disabled"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled create status=%d", res.StatusCode)
	}
}

func TestRazorpayLiveHTTPCreateVerifyAndAuth(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	res, _ := fixture.request(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":100}`, false, map[string]string{"Idempotency-Key": "api-test"})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", res.StatusCode)
	}
	res, body := fixture.request(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":100,"externalId":"test-order"}`, true, map[string]string{"Idempotency-Key": "api-test"})
	if res.StatusCode != http.StatusCreated || !strings.Contains(body, `"razorpayOrderId":"order_api_live"`) {
		t.Fatalf("create status=%d body=%s", res.StatusCode, body)
	}
	localID := jsonLiveStringField(t, body, "id")
	fixture.provider.payment = razorpaylive.ProviderPayment{OrderID: "order_api_live", Amount: 100, Currency: "INR", Status: "captured", Method: "upi", Captured: true}
	verifyBody := `{"razorpay_order_id":"order_api_live","razorpay_payment_id":"pay_api_test","razorpay_signature":"bad"}`
	res, _ = fixture.request(t, http.MethodPost, "/api/razorpay/live/orders/"+localID+"/verify", verifyBody, true, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("tampered verify status=%d", res.StatusCode)
	}
	signature := apiLiveCheckoutSignature("checkout-secret-123456", "order_api_live", "pay_api_test")
	verifyBody = `{"razorpay_order_id":"order_api_live","razorpay_payment_id":"pay_api_test","razorpay_signature":"` + signature + `"}`
	res, body = fixture.request(t, http.MethodPost, "/api/razorpay/live/orders/"+localID+"/verify", verifyBody, true, nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"status":"captured"`) {
		t.Fatalf("verify status=%d body=%s", res.StatusCode, body)
	}
}

func TestRazorpayLiveHTTPAcceptsEventAmount(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	res, body := fixture.request(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":25000,"externalId":"ieee-event-registration"}`, true, map[string]string{"Idempotency-Key": "ieee-event-registration"})
	if res.StatusCode != http.StatusCreated || !strings.Contains(body, `"amountPaise":25000`) {
		t.Fatalf("create status=%d body=%s", res.StatusCode, body)
	}
}

func TestRazorpayLiveWebhookRequiresSignatureAndDeduplicates(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	res, createBody := fixture.request(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":100}`, true, map[string]string{"Idempotency-Key": "webhook-api"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create body=%s", createBody)
	}
	body := `{"event":"payment.captured","created_at":1785672000,"payload":{"payment":{"entity":{"id":"pay_hook_api","order_id":"order_api_live","amount":100,"currency":"INR","status":"captured","method":"upi","captured":true}}}}`
	res, _ = fixture.request(t, http.MethodPost, "/api/razorpay/live/webhook", body, false, map[string]string{"X-Razorpay-Event-Id": "evt_api", "X-Razorpay-Signature": "bad"})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid signature status=%d", res.StatusCode)
	}
	signature := razorpaylive.Sign("webhook-secret-123456789012", []byte(body))
	headers := map[string]string{"X-Razorpay-Event-Id": "evt_api", "X-Razorpay-Signature": signature}
	res, response := fixture.request(t, http.MethodPost, "/api/razorpay/live/webhook", body, false, headers)
	if res.StatusCode != http.StatusOK || !strings.Contains(response, `"processed":true`) {
		t.Fatalf("webhook status=%d body=%s", res.StatusCode, response)
	}
	res, response = fixture.request(t, http.MethodPost, "/api/razorpay/live/webhook", body, false, headers)
	if res.StatusCode != http.StatusOK || !strings.Contains(response, `"duplicate":true`) {
		t.Fatalf("duplicate status=%d body=%s", res.StatusCode, response)
	}
}

func apiLiveCheckoutSignature(secret, orderID, paymentID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}

func jsonLiveStringField(t *testing.T, raw, key string) string {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	result, _ := value[key].(string)
	if result == "" {
		t.Fatalf("missing %s in %s", key, raw)
	}
	return result
}

func TestRazorpayLiveEnabledCSPAllowsOnlyRazorpayCheckoutOrigins(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	res, _ := fixture.request(t, http.MethodGet, "/", "", false, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("root status=%d", res.StatusCode)
	}
	csp := res.Header.Get("Content-Security-Policy")
	for _, required := range []string{"https://checkout.razorpay.com", "frame-src https://api.razorpay.com", "https://*.razorpay.com"} {
		if !strings.Contains(csp, required) {
			t.Fatalf("CSP missing %q: %s", required, csp)
		}
	}
	if strings.Contains(csp, "'unsafe-inline'") || strings.Contains(csp, "'unsafe-eval'") {
		t.Fatalf("CSP was weakened: %s", csp)
	}
}

func TestRazorpayLiveRoutesAcceptServerAPIKey(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	res, body := fixture.apiKeyRequest(t, http.MethodGet, "/api/razorpay/live/config", "", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"enabled":true`) {
		t.Fatalf("config status=%d body=%s", res.StatusCode, body)
	}
	res, body = fixture.apiKeyRequest(t, http.MethodPost, "/api/razorpay/live/orders", `{"amountPaise":100,"externalId":"portal-live"}`, map[string]string{"Idempotency-Key": "portal-live"})
	if res.StatusCode != http.StatusCreated || !strings.Contains(body, `"razorpayOrderId":"order_api_live"`) {
		t.Fatalf("create status=%d body=%s", res.StatusCode, body)
	}
	localID := jsonLiveStringField(t, body, "id")
	record, err := fixture.app.FindRecordById("razorpay_live_orders", localID)
	if err != nil {
		t.Fatal(err)
	}
	if record.GetString("created_by") != "" {
		t.Fatalf("API-key order unexpectedly has created_by=%q", record.GetString("created_by"))
	}
	res, body = fixture.apiKeyRequest(t, http.MethodGet, "/api/razorpay/live/orders/"+localID, "", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"amountPaise":100`) {
		t.Fatalf("get status=%d body=%s", res.StatusCode, body)
	}
}

func TestCheckoutRazorpayLiveKeepsOneRupeePilot(t *testing.T) {
	fixture := newRazorpayLiveAPIFixture(t, true)
	headers := map[string]string{
		"Origin":          checkoutOrigin,
		"Idempotency-Key": "40000000-0000-4000-8000-000000000001",
	}
	res, body := fixture.request(t, http.MethodPost, "/api/checkout/v2/razorpay/live/orders", `{"amount":2}`, false, headers)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, "exactly ₹1") {
		t.Fatalf("live non-pilot amount status=%d body=%s", res.StatusCode, body)
	}
	headers["Idempotency-Key"] = "40000000-0000-4000-8000-000000000002"
	res, body = fixture.request(t, http.MethodPost, "/api/checkout/v2/razorpay/live/orders", `{"amount":1}`, false, headers)
	if res.StatusCode != http.StatusCreated || !strings.Contains(body, `"amountPaise":100`) || !strings.Contains(body, `"keyId":"rzp_live_api"`) {
		t.Fatalf("live pilot create status=%d body=%s", res.StatusCode, body)
	}
}
