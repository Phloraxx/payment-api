package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
)

const testAdminPassword = "correct horse battery staple"

func newTestApp(t *testing.T, mutate func(*Config)) *App {
	t.Helper()
	cfg := Config{
		DataDir: t.TempDir(), BootstrapAdminPassword: testAdminPassword,
		BackupHourUTC: 3, BackupRetention: 30,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	return app
}
func TestNewRequiresBootstrapPasswordOnFreshDatabase(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(context.Background(), Config{DataDir: dir}); err == nil {
		t.Fatal("expected fresh database bootstrap refusal")
	}
	app, err := New(context.Background(), Config{DataDir: dir, BootstrapAdminPassword: testAdminPassword})
	if err != nil {
		t.Fatalf("second startup after refusal: %v", err)
	}
	_ = app.Close()
}

func TestNewHoldsExclusiveProcessLock(t *testing.T) {
	dir := t.TempDir()
	first, err := New(context.Background(), Config{DataDir: dir, BootstrapAdminPassword: testAdminPassword})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := New(context.Background(), Config{DataDir: dir}); err == nil {
		t.Fatal("expected second runtime to fail process lock")
	}
}
func TestRuntimeHealthCORSAndAndroidAdminLogin(t *testing.T) {
	app := newTestApp(t, func(cfg *Config) {
		cfg.AllowedOrigins = []string{"https://payment.mulearnscet.in"}
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"database":"ok"`) {
		t.Fatalf("health=%d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodOptions, "/v1/payments", nil)
	req.Header.Set("Origin", "https://payment.mulearnscet.in")
	res = httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || res.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("preflight=%d headers=%v", res.Code, res.Header())
	}

	body := bytes.NewBufferString(`{"password":"` + testAdminPassword + `","client":"android"}`)
	req = httptest.NewRequest(http.MethodPost, "/admin/session", body)
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("login=%d %s", res.Code, res.Body.String())
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &login); err != nil || !strings.HasPrefix(login.Token, "pg_admin_") {
		t.Fatalf("login response=%s err=%v", res.Body.String(), err)
	}
}
func TestRuntimeMerchantCreateKeepsRoutingServerOwned(t *testing.T) {
	app := newTestApp(t, nil)
	ctx := context.Background()
	if _, err := app.Profiles.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "merchant@paytm",
		PayeeName: "PayGate", Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Profiles.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	key, err := app.Auth.CreateAPIKey(ctx, "runtime test")
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_123"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key.Secret)
	req.Header.Set("Idempotency-Key", "registration_284")
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", res.Code, res.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["name"] != "Sourav P Bijoy" || payload["external_id"] != "evt_123" {
		t.Fatalf("identity response=%v", payload)
	}
	upi, _ := payload["upi_uri"].(string)
	if !strings.Contains(upi, "pa=merchant%40paytm") || !strings.Contains(upi, "am=100.") {
		t.Fatalf("unexpected UPI URI %q", upi)
	}
	if _, exists := payload["collection_profile_id"]; exists {
		t.Fatalf("merchant response leaked collection profile: %v", payload)
	}
	if _, exists := payload["payment_account"]; exists {
		t.Fatalf("merchant response leaked payment account: %v", payload)
	}
}

func TestBackupNowRetainsNewestSnapshots(t *testing.T) {
	app := newTestApp(t, func(cfg *Config) { cfg.BackupRetention = 2 })
	base := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := app.BackupNow(context.Background(), base.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(app.Config.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".db") {
			names = append(names, entry.Name())
		}
	}
	if len(names) != 2 {
		t.Fatalf("backup count=%d names=%v", len(names), names)
	}
	if _, err := os.Stat(filepath.Join(app.Config.BackupDir, "paygate-20260901T030000Z.db")); !os.IsNotExist(err) {
		t.Fatalf("oldest backup still exists, err=%v", err)
	}
}
func TestExpiryWorkerExpiresAfterGrace(t *testing.T) {
	app := newTestApp(t, func(cfg *Config) { cfg.ExpiryInterval = 10 * time.Millisecond })
	ctx := context.Background()
	if _, err := app.Profiles.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "merchant@paytm", Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Profiles.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	app.Payments.Now = func() time.Time { return createdAt }
	created, err := app.Payments.Create(ctx, payments.CreateInput{
		RequestedAmountPaise: 10000, Name: "Sourav P Bijoy", ExternalID: "evt_123",
		IdempotencyScope: "test", IdempotencyKey: "one",
	})
	if err != nil {
		t.Fatal(err)
	}
	app.Payments.Now = func() time.Time { return createdAt.Add(11 * time.Minute) }
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.RunWorkers(workerCtx)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := app.Payments.Get(ctx, created.Payment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Payment.Status == "expired" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("payment did not expire through runtime worker")
}
func TestNextDailyBackupUsesUTCAndMovesToTomorrowAtBoundary(t *testing.T) {
	before := time.Date(2026, 9, 1, 2, 59, 0, 0, time.UTC)
	if got := nextDailyBackup(before, 3); !got.Equal(time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("before boundary got %s", got)
	}
	atBoundary := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	if got := nextDailyBackup(atBoundary, 3); !got.Equal(time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("at boundary got %s", got)
	}
}
func TestRuntimePairingSessionUsesPublicAppLinkNotRelayEndpoint(t *testing.T) {
	app := newTestApp(t, func(cfg *Config) { cfg.PublicURL = "https://pay.example" })
	loginBody := bytes.NewBufferString(`{"password":"` + testAdminPassword + `","client":"android"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/admin/session", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	app.Handler().ServeHTTP(loginRes, loginReq)
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRes.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/device/pairing-session", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+login.Token)
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("pairing session=%d %s", res.Code, res.Body.String())
	}
	var payload struct {
		PairingURL string `json:"pairing_url"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(payload.PairingURL, "https://pay.example/device/pair/") {
		t.Fatalf("pairing URL=%q", payload.PairingURL)
	}
}

func TestRuntimeServesAndroidAssetLinksForProductionSigner(t *testing.T) {
	app := newTestApp(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/assetlinks.json", nil)
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("assetlinks status=%d body=%s", res.Code, res.Body.String())
	}
	for _, want := range []string{"io.github.phloraxx.paygaterelay", "41:2B:8F:66:C0:6A:CD:93", "delegate_permission/common.handle_all_urls"} {
		if !strings.Contains(res.Body.String(), want) {
			t.Fatalf("assetlinks missing %q: %s", want, res.Body.String())
		}
	}
}

func TestRuntimeServesV4DashboardWithoutShadowingAdminAPI(t *testing.T) {
	app := newTestApp(t, nil)
	for _, path := range []string{"/", "/device/pair/example-token"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		app.Handler().ServeHTTP(res, req)
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `id="root"`) {
			t.Fatalf("dashboard %s = %d %s", path, res.Code, res.Body.String())
		}
		if !strings.Contains(res.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatalf("dashboard %s missing CSP: %v", path, res.Header())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/overview", nil)
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized || !strings.Contains(res.Body.String(), `"unauthorized"`) {
		t.Fatalf("admin route shadowed by SPA: %d %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("admin route returned SPA content type: %s", res.Header().Get("Content-Type"))
	}
}

func TestRuntimeCutoverBootstrapPreservesLegacyMerchantAndWebhook(t *testing.T) {
	dir := t.TempDir()
	legacyKey := "legacy-v3-merchant-key-0123456789abcdef0123456789"
	legacyWebhookSecret := "legacy-webhook-secret-0123456789abcdef"
	cfg := Config{
		DataDir: dir, BootstrapAdminPassword: testAdminPassword,
		BootstrapMerchantAPIKey:  legacyKey,
		BootstrapWebhookEndpoint: "https://merchant.example/paygate",
		BootstrapWebhookSecret:   legacyWebhookSecret,
		BackupHourUTC:            3, BackupRetention: 30,
	}
	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := app.Auth.AuthenticateAPIKey(context.Background(), legacyKey); err != nil || id != "key_legacy_v3" {
		t.Fatalf("legacy merchant key id=%q err=%v", id, err)
	}
	webhook, err := app.Settings.Webhook(context.Background())
	if err != nil || !webhook.Enabled || webhook.Endpoint != cfg.BootstrapWebhookEndpoint {
		t.Fatalf("webhook=%+v err=%v", webhook, err)
	}
	var plaintext int
	if err := app.DB.SQL.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE CAST(secret_hash AS TEXT)=?`, legacyKey).Scan(&plaintext); err != nil {
		t.Fatal(err)
	}
	if plaintext != 0 {
		t.Fatal("legacy merchant API key stored in plaintext")
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := New(context.Background(), Config{
		DataDir:                  dir,
		BootstrapMerchantAPIKey:  "replacement-v3-merchant-key-0123456789abcdef0123456789",
		BootstrapWebhookEndpoint: "https://replacement.example/paygate",
		BootstrapWebhookSecret:   "replacement-webhook-secret-0123456789abcdef",
		BackupHourUTC:            3, BackupRetention: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := second.Auth.AuthenticateAPIKey(context.Background(), legacyKey); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Auth.AuthenticateAPIKey(context.Background(), "replacement-v3-merchant-key-0123456789abcdef0123456789"); !errors.Is(err, auth.ErrInvalidAPIKey) {
		t.Fatalf("replacement key unexpectedly accepted: %v", err)
	}
	webhook, err = second.Settings.Webhook(context.Background())
	if err != nil || webhook.Endpoint != cfg.BootstrapWebhookEndpoint {
		t.Fatalf("persisted webhook overwritten: %+v err=%v", webhook, err)
	}
}
