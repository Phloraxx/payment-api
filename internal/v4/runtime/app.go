package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Phloraxx/payment-api/internal/processlock"
	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/httpapi"
	"github.com/Phloraxx/payment-api/internal/v4/operator"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/relay"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
	"github.com/Phloraxx/payment-api/internal/v4web"
)

type App struct {
	Config   Config
	DB       *storage.DB
	Payments *payments.Service
	Auth     *auth.Service
	Profiles *profiles.Service
	Relay    *relay.Service
	Webhooks *webhooks.Service
	Operator *operator.Service
	Settings *operator.SettingsService

	lock    *processlock.Lock
	handler http.Handler
	mu      sync.Mutex
	closed  bool
}

func New(ctx context.Context, cfg Config) (_ *App, err error) {
	cfg, err = cfg.normalized()
	if err != nil {
		return nil, err
	}
	lock, err := processlock.Acquire(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = lock.Release()
		}
	}()

	db, err := storage.Open(ctx, filepath.Join(cfg.DataDir, "paygate.db"))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()

	paymentService := payments.NewService(db)
	authService := auth.NewService(db)
	profileService := profiles.NewService(db)
	relayService := relay.NewService(db, paymentService)
	webhookService := webhooks.NewService(db, webhooks.Config{})
	operatorService := operator.NewService(db)
	settingsService := operator.NewSettingsService(db, webhookService)
	if err := bootstrapAdmin(ctx, db, authService, cfg.BootstrapAdminPassword); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.BootstrapMerchantAPIKey) != "" {
		if err := authService.BootstrapAPIKey(ctx, "Migrated v3 merchant key", cfg.BootstrapMerchantAPIKey); err != nil {
			return nil, fmt.Errorf("bootstrap merchant API key: %w", err)
		}
	}
	if err := settingsService.BootstrapWebhook(ctx, cfg.BootstrapWebhookEndpoint, cfg.BootstrapWebhookSecret); err != nil {
		return nil, fmt.Errorf("bootstrap webhook settings: %w", err)
	}
	if err := settingsService.ApplyPersistedWebhook(ctx); err != nil {
		return nil, fmt.Errorf("load persisted webhook settings: %w", err)
	}
	adminPaymentService := adminpayments.NewService(db)
	merchantHandler := httpapi.NewMerchantHandler(authService, paymentService)
	adminHandler := httpapi.NewAdminHandler(authService, adminPaymentService, operatorService, settingsService,
		profileService, relayService, webhookService)
	if cfg.PublicURL != "" {
		adminHandler.PairingBaseURL = cfg.PublicURL
	}
	relayHandler := httpapi.NewRelayHandler(relayService)

	app := &App{
		Config: cfg, DB: db, Payments: paymentService, Auth: authService,
		Profiles: profileService, Relay: relayService, Webhooks: webhookService,
		Operator: operatorService, Settings: settingsService, lock: lock,
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/", merchantHandler)
	mux.Handle("/admin/", adminHandler)
	mux.Handle("/api/v4/relay/", relayHandler)
	mux.Handle(relay.DevicePath, relayHandler)
	mux.HandleFunc("GET /.well-known/assetlinks.json", app.assetLinks)
	mux.HandleFunc("GET /health", app.health)
	mux.Handle("/", v4web.Handler())
	app.handler = cors(cfg.AllowedOrigins, mux)
	return app, nil
}

func bootstrapAdmin(ctx context.Context, db *storage.DB, service *auth.Service, password string) error {
	var count int
	if err := db.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_credentials`).Scan(&count); err != nil {
		return fmt.Errorf("read admin bootstrap state: %w", err)
	}
	if count > 0 {
		return nil
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("fresh PayGate v4 database requires a bootstrap admin password")
	}
	if err := service.BootstrapPassword(ctx, password); err != nil {
		return fmt.Errorf("bootstrap admin password: %w", err)
	}
	return nil
}

func (a *App) Handler() http.Handler {
	if a == nil || a.handler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "PayGate unavailable", http.StatusServiceUnavailable)
		})
	}
	return a.handler
}

const androidAssetLinks = `[
  {
    "relation": ["delegate_permission/common.handle_all_urls"],
    "target": {
      "namespace": "android_app",
      "package_name": "io.github.phloraxx.paygaterelay",
      "sha256_cert_fingerprints": ["41:2B:8F:66:C0:6A:CD:93:95:8C:4D:D1:1C:AA:32:14:BA:95:00:59:C0:0E:57:15:9D:56:73:F9:47:00:D4:4A"]
    }
  }
]`

func (a *App) assetLinks(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(androidAssetLinks))
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.DB == nil || a.DB.SQL == nil {
		http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable)
		return
	}
	var one int
	if err := a.DB.SQL.QueryRowContext(r.Context(), `SELECT 1`).Scan(&one); err != nil || one != 1 {
		http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(`{"status":"healthy","database":"ok","version":"v4"}`))
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	var errs []error
	if a.DB != nil {
		errs = append(errs, a.DB.Close())
	}
	if a.lock != nil {
		errs = append(errs, a.lock.Release())
	}
	return errors.Join(errs...)
}
func cors(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
