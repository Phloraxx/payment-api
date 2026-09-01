package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/operator"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/relay"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

type adminHTTPFixture struct {
	db       *storage.DB
	handler  *AdminHandler
	auth     *auth.Service
	payments *payments.Service
	cookie   *http.Cookie
	token    string
}

func newAdminHTTPFixture(t *testing.T) adminHTTPFixture {
	t.Helper()
	ctx := context.Background()
	db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	authService := auth.NewService(db)
	if err := authService.BootstrapPassword(ctx, "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	profileService := profiles.NewService(db)
	for _, input := range []profiles.UpsertInput{
		{ID: "paytm", Label: "Paytm", UPIID: "paygate@paytm", PayeeName: "PayGate", Parser: "paytm_notification", Enabled: true},
		{ID: "kotak", Label: "Kotak", UPIID: "paygate@kotak", PayeeName: "PayGate", Parser: "kotak_sms", Enabled: true},
	} {
		if _, err := profileService.Upsert(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := profileService.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	paymentService := payments.NewService(db)
	webhookService := webhooks.NewService(db, webhooks.Config{})
	settingsService := operator.NewSettingsService(db, webhookService)
	relayService := relay.NewService(db, paymentService)
	handler := NewAdminHandler(
		authService,
		adminpayments.NewService(db),
		operator.NewService(db),
		settingsService,
		profileService,
		relayService,
		webhookService,
	)
	handler.SecureCookies = false
	handler.PairingBaseURL = "https://pay.example"

	cookie := loginAdmin(t, handler, false)
	token := loginAdminToken(t, handler)
	return adminHTTPFixture{db: db, handler: handler, auth: authService, payments: paymentService, cookie: cookie, token: token}
}

func loginAdmin(t *testing.T, handler http.Handler, android bool) *http.Cookie {
	t.Helper()
	client := "web"
	if android {
		client = "android"
	}
	body := `{"password":"correct horse battery staple","client":"` + client + `"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rr.Code, rr.Body.String())
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == adminCookieName {
			return cookie
		}
	}
	t.Fatal("admin cookie missing")
	return nil
}
func loginAdminToken(t *testing.T, handler http.Handler) string {
	t.Helper()
	body := `{"password":"correct horse battery staple","client":"android"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("android login status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil || response.Token == "" {
		t.Fatalf("android login response=%s err=%v", rr.Body.String(), err)
	}
	return response.Token
}

func adminRequest(t *testing.T, f adminHTTPFixture, method, path string, body []byte, bearer bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer {
		req.Header.Set("Authorization", "Bearer "+f.token)
	} else if f.cookie != nil {
		req.AddCookie(f.cookie)
	}
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	return rr
}
func createAdminTestPayment(t *testing.T, f adminHTTPFixture, name, eventID, idem string) payments.Payment {
	t.Helper()
	result, err := f.payments.Create(context.Background(), payments.CreateInput{
		RequestedAmountPaise: 10000,
		Name:                 name,
		ExternalID:           eventID,
		IdempotencyScope:     "admin-http-test",
		IdempotencyKey:       idem,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Payment
}

func TestAdminSessionCookieAndAndroidBearer(t *testing.T) {
	f := newAdminHTTPFixture(t)
	for _, bearer := range []bool{false, true} {
		rr := adminRequest(t, f, http.MethodGet, "/admin/overview", nil, bearer)
		if rr.Code != http.StatusOK {
			t.Fatalf("overview bearer=%v status=%d body=%s", bearer, rr.Code, rr.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/overview", nil)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated overview status=%d", rr.Code)
	}
}
func TestAdminPaymentsFilterDetailAndEdit(t *testing.T) {
	f := newAdminHTTPFixture(t)
	one := createAdminTestPayment(t, f, "Sourav P Bijoy", "evt_shared", "p1")
	_ = createAdminTestPayment(t, f, "Another Person", "evt_shared", "p2")

	rr := adminRequest(t, f, http.MethodGet, "/admin/payments?external_id=evt_shared", nil, false)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"total":2`) {
		t.Fatalf("filter status=%d body=%s", rr.Code, rr.Body.String())
	}

	patch := []byte(`{"name":"Sourav P. Bijoy","status":"paid","payer_name":"Bijoy P","payer_upi_id":"bijoy@okaxis"}`)
	rr = adminRequest(t, f, http.MethodPatch, "/admin/payments/"+one.ID, patch, true)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"paid"`) {
		t.Fatalf("edit status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = adminRequest(t, f, http.MethodGet, "/admin/payments/"+one.ID, nil, false)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"payer_name":"Bijoy P"`) || !strings.Contains(rr.Body.String(), `"payment.paid"`) {
		t.Fatalf("detail status=%d body=%s", rr.Code, rr.Body.String())
	}
	for _, key := range []string{`"created_at"`, `"event_type"`, `"changes"`} {
		if !strings.Contains(rr.Body.String(), key) {
			t.Fatalf("detail must use snake_case key %s: %s", key, rr.Body.String())
		}
	}
	if strings.Contains(rr.Body.String(), `"CreatedAt"`) || strings.Contains(rr.Body.String(), `"EventType"`) {
		t.Fatalf("detail leaked Go field names: %s", rr.Body.String())
	}
}
func TestAdminSettingsProfilesKeysAndPairing(t *testing.T) {
	f := newAdminHTTPFixture(t)

	rr := adminRequest(t, f, http.MethodPatch, "/admin/settings/webhook", []byte(`{"endpoint":"https://merchant.example/webhook"}`), false)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"signing_secret":"whsec_`) {
		t.Fatalf("webhook configure status=%d body=%s", rr.Code, rr.Body.String())
	}
	rr = adminRequest(t, f, http.MethodGet, "/admin/settings", nil, false)
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), "whsec_") || !strings.Contains(rr.Body.String(), `"secret_configured":true`) {
		t.Fatalf("settings status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = adminRequest(t, f, http.MethodPost, "/admin/profiles/kotak/activate", nil, false)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"active":true`) {
		t.Fatalf("activate profile status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = adminRequest(t, f, http.MethodPost, "/admin/api-keys", []byte(`{"label":"Test frontend"}`), false)
	if rr.Code != http.StatusCreated || !strings.Contains(rr.Body.String(), `"secret":"pg_live_`) {
		t.Fatalf("create api key status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = adminRequest(t, f, http.MethodPost, "/admin/device/pairing-session", []byte(`{}`), false)
	if rr.Code != http.StatusCreated || !strings.Contains(rr.Body.String(), `"pairing_url":"https://pay.example/device/pair/`) {
		t.Fatalf("pairing status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminPasswordChangeRequiresCurrentAndRevokesAllSessions(t *testing.T) {
	f := newAdminHTTPFixture(t)
	wrong := adminRequest(t, f, http.MethodPatch, "/admin/settings/password", []byte(`{"current_password":"wrong password","new_password":"replacement password"}`), false)
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current status=%d body=%s", wrong.Code, wrong.Body.String())
	}
	if rr := adminRequest(t, f, http.MethodGet, "/admin/overview", nil, true); rr.Code != http.StatusOK {
		t.Fatalf("wrong change invalidated bearer status=%d", rr.Code)
	}

	changed := adminRequest(t, f, http.MethodPatch, "/admin/settings/password", []byte(`{"current_password":"correct horse battery staple","new_password":"replacement password"}`), false)
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change status=%d body=%s", changed.Code, changed.Body.String())
	}
	if rr := adminRequest(t, f, http.MethodGet, "/admin/overview", nil, true); rr.Code != http.StatusUnauthorized {
		t.Fatalf("old bearer status=%d", rr.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/session", strings.NewReader(`{"password":"replacement password"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("new password login status=%d body=%s", rr.Code, rr.Body.String())
	}
}
