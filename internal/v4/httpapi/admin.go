package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/operator"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/relay"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

const (
	adminCookieName           = "paygate_admin"
	adminLoginConcurrency     = 2
	adminLoginFailureWindow   = 15 * time.Minute
	adminLoginFailureLimit    = 5
	adminLoginBlockDuration   = time.Minute
	adminLoginThrottleEntries = 1024
)

type adminLoginFailure struct {
	failures     int
	firstFailure time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

type adminDeviceAuthorizationContextKey struct{}

type adminDeviceAuthorization struct {
	ID         string
	EnrolledAt time.Time
}

func adminDeviceAuthorizationFromContext(ctx context.Context) (adminDeviceAuthorization, bool) {
	if ctx == nil {
		return adminDeviceAuthorization{}, false
	}
	value, ok := ctx.Value(adminDeviceAuthorizationContextKey{}).(adminDeviceAuthorization)
	return value, ok && value.ID != "" && !value.EnrolledAt.IsZero()
}

type AdminHandler struct {
	Auth           *auth.Service
	Payments       *adminpayments.Service
	Operator       *operator.Service
	Settings       *operator.SettingsService
	Profiles       *profiles.Service
	Relay          *relay.Service
	Webhooks       *webhooks.Service
	PairingBaseURL string
	SecureCookies  bool
	loginSlots     chan struct{}
	loginMu        sync.Mutex
	loginFailures  map[string]adminLoginFailure
	mux            *http.ServeMux
}

func NewAdminHandler(authService *auth.Service, paymentService *adminpayments.Service, operatorService *operator.Service,
	settingsService *operator.SettingsService, profileService *profiles.Service, relayService *relay.Service, webhookService *webhooks.Service) *AdminHandler {
	h := &AdminHandler{Auth: authService, Payments: paymentService, Operator: operatorService, Settings: settingsService,
		Profiles: profileService, Relay: relayService, Webhooks: webhookService, SecureCookies: true,
		loginSlots: make(chan struct{}, adminLoginConcurrency), loginFailures: make(map[string]adminLoginFailure), mux: http.NewServeMux()}
	h.registerRoutes()
	return h
}
func (h *AdminHandler) registerRoutes() {
	h.mux.HandleFunc("POST /admin/session", h.login)
	h.mux.HandleFunc("DELETE /admin/session", h.logout)
	h.mux.HandleFunc("GET /admin/overview", h.overview)
	h.mux.HandleFunc("GET /admin/activity", h.activity)
	h.mux.HandleFunc("GET /admin/payments", h.listPayments)
	h.mux.HandleFunc("GET /admin/payments/{id}", h.getAdminPayment)
	h.mux.HandleFunc("PATCH /admin/payments/{id}", h.editAdminPayment)
	h.mux.HandleFunc("GET /admin/settings", h.getSettings)
	h.mux.HandleFunc("PATCH /admin/settings/webhook", h.updateWebhookSettings)
	h.mux.HandleFunc("PATCH /admin/settings/password", h.changePassword)
	h.mux.HandleFunc("POST /admin/webhooks/{id}/retry", h.retryWebhook)
	h.mux.HandleFunc("GET /admin/profiles", h.listProfiles)
	h.mux.HandleFunc("POST /admin/profiles", h.upsertProfile)
	h.mux.HandleFunc("PATCH /admin/profiles/{id}/destination", h.updateProfileDestination)
	h.mux.HandleFunc("POST /admin/profiles/{id}/activate", h.activateProfile)
	h.mux.HandleFunc("GET /admin/api-keys", h.listAPIKeys)
	h.mux.HandleFunc("POST /admin/api-keys", h.createAPIKey)
	h.mux.HandleFunc("DELETE /admin/api-keys/{id}", h.revokeAPIKey)
	h.mux.HandleFunc("GET /admin/device", h.getDevice)
	h.mux.HandleFunc("POST /admin/device/pairing-session", h.createPairingSession)
	h.mux.HandleFunc("DELETE /admin/device/{id}", h.revokeDevice)
}

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	if h == nil || h.Auth == nil || h.mux == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate is not ready")
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/admin/session" {
		h.mux.ServeHTTP(w, r)
		return
	}
	if _, err := h.adminToken(r); err == nil {
		h.mux.ServeHTTP(w, r)
		return
	}
	deviceID, enrolledAt, ok := h.deviceAuthorization(w, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Admin or connected-device authentication is required")
		return
	}
	if !deviceOperationalRoute(r.Method, r.URL.Path) {
		writeError(w, http.StatusForbidden, "admin_required", "This setting requires the web admin session")
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), adminDeviceAuthorizationContextKey{}, adminDeviceAuthorization{
		ID: deviceID, EnrolledAt: enrolledAt,
	}))
	h.mux.ServeHTTP(w, r)
}

const adminDeviceBodyLimit = 256 << 10

func (h *AdminHandler) deviceAuthorization(w http.ResponseWriter, r *http.Request) (string, time.Time, bool) {
	if h.Relay == nil {
		return "", time.Time{}, false
	}
	deviceID := strings.TrimSpace(r.Header.Get("X-PayGate-Relay-Device"))
	timestamp := strings.TrimSpace(r.Header.Get("X-PayGate-Relay-Time"))
	signature := strings.TrimSpace(r.Header.Get("X-PayGate-Relay-Signature"))
	if deviceID == "" || timestamp == "" || signature == "" {
		return "", time.Time{}, false
	}
	var body []byte
	if r.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(r.Body, adminDeviceBodyLimit+1))
		if err != nil || len(raw) > adminDeviceBodyLimit {
			return "", time.Time{}, false
		}
		body = raw
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	target := r.URL.RequestURI()
	id, enrolledAt, err := h.Relay.AuthenticateDeviceWithEpoch(r.Context(), relay.RequestAuth{
		DeviceID: deviceID, Timestamp: timestamp, Signature: signature, Method: r.Method, Path: target,
	}, body)
	return id, enrolledAt, err == nil
}

func deviceOperationalRoute(method, path string) bool {
	if method == http.MethodGet {
		return path == "/admin/overview" || path == "/admin/activity" || path == "/admin/payments" ||
			strings.HasPrefix(path, "/admin/payments/") || path == "/admin/settings" ||
			path == "/admin/profiles" || path == "/admin/device"
	}
	// A paired phone is evidence transport, not a payment authority. The only
	// device-authenticated mutation is the operator-requested active UPI destination
	// change. Payment edits, webhook retries, key/profile administration and pairing
	// remain web-admin-only even if the Android UI accidentally exposes them.
	return method == http.MethodPatch && path == "/admin/profiles/active/destination"
}

type adminLoginRequest struct {
	Password string `json:"password"`
	Client   string `json:"client,omitempty"`
}

type adminLoginResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token,omitempty"`
}

func (h *AdminHandler) login(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input adminLoginRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	key := loginRemoteKey(r)
	now := time.Now().UTC()
	if allowed, retry := h.loginAttemptAllowed(key, now); !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		writeError(w, http.StatusTooManyRequests, "login_rate_limited", "Too many failed login attempts; try again later")
		return
	}
	if !h.acquireLoginSlot() {
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusTooManyRequests, "login_busy", "Too many login attempts are already being verified")
		return
	}
	defer h.releaseLoginSlot()
	session, err := h.Auth.CreateAdminSession(r.Context(), input.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrNotInitialized) {
			h.recordLoginFailure(key, now)
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Password is incorrect")
			return
		}
		if writeAdminLoginServiceError(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "PayGate could not create the admin session")
		return
	}
	h.clearLoginFailures(key)
	h.setAdminCookie(w, session.Token, session.ExpiresAt)
	response := adminLoginResponse{ExpiresAt: session.ExpiresAt}
	if strings.EqualFold(strings.TrimSpace(input.Client), "android") {
		response.Token = session.Token
	}
	writeJSON(w, http.StatusOK, response)
}

func writeAdminLoginServiceError(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, storage.ErrBusy) {
		return false
	}
	w.Header().Set("Retry-After", "1")
	writeError(w, http.StatusServiceUnavailable, "login_retryable", "PayGate is temporarily busy; try again shortly")
	return true
}

func (h *AdminHandler) acquireLoginSlot() bool {
	if h == nil || h.loginSlots == nil {
		return true
	}
	select {
	case h.loginSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (h *AdminHandler) releaseLoginSlot() {
	if h == nil || h.loginSlots == nil {
		return
	}
	<-h.loginSlots
}
func loginRemoteKey(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	remote := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remote); err == nil && host != "" {
		return host
	}
	if remote != "" {
		return remote
	}
	return "unknown"
}

func retryAfterSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 1
	}
	return int((duration + time.Second - 1) / time.Second)
}

func (h *AdminHandler) loginAttemptAllowed(key string, now time.Time) (bool, time.Duration) {
	if h == nil || h.loginFailures == nil {
		return true, 0
	}
	h.loginMu.Lock()
	defer h.loginMu.Unlock()
	h.pruneLoginFailuresLocked(now)
	state, ok := h.loginFailures[key]
	if !ok {
		return true, 0
	}
	if now.Before(state.blockedUntil) {
		state.lastSeen = now
		h.loginFailures[key] = state
		return false, state.blockedUntil.Sub(now)
	}
	if state.firstFailure.IsZero() || now.Sub(state.firstFailure) >= adminLoginFailureWindow {
		delete(h.loginFailures, key)
	}
	return true, 0
}

func (h *AdminHandler) recordLoginFailure(key string, now time.Time) {
	if h == nil || h.loginFailures == nil {
		return
	}
	h.loginMu.Lock()
	defer h.loginMu.Unlock()
	h.pruneLoginFailuresLocked(now)
	state := h.loginFailures[key]
	if state.firstFailure.IsZero() || now.Sub(state.firstFailure) >= adminLoginFailureWindow {
		state = adminLoginFailure{firstFailure: now}
	}
	state.failures++
	state.lastSeen = now
	if state.failures >= adminLoginFailureLimit {
		state.blockedUntil = now.Add(adminLoginBlockDuration)
	}
	h.loginFailures[key] = state
}

func (h *AdminHandler) clearLoginFailures(key string) {
	if h == nil || h.loginFailures == nil {
		return
	}
	h.loginMu.Lock()
	delete(h.loginFailures, key)
	h.loginMu.Unlock()
}

func (h *AdminHandler) pruneLoginFailuresLocked(now time.Time) {
	for key, state := range h.loginFailures {
		if now.After(state.blockedUntil) && now.Sub(state.lastSeen) >= adminLoginFailureWindow {
			delete(h.loginFailures, key)
		}
	}
	for len(h.loginFailures) >= adminLoginThrottleEntries {
		var oldestKey string
		var oldest time.Time
		for key, state := range h.loginFailures {
			if oldestKey == "" || state.lastSeen.Before(oldest) {
				oldestKey, oldest = key, state.lastSeen
			}
		}
		if oldestKey == "" {
			return
		}
		delete(h.loginFailures, oldestKey)
	}
}

func (h *AdminHandler) logout(w http.ResponseWriter, r *http.Request) {
	token, err := h.adminToken(r)
	if err == nil {
		err = h.Auth.RevokeAdminSession(r.Context(), token)
		if err != nil && !errors.Is(err, auth.ErrInvalidSession) {
			if writeStorageBusyError(w, err) {
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not revoke admin session")
			return
		}
	}
	h.clearAdminCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
func (h *AdminHandler) adminToken(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		if err := h.Auth.AuthenticateAdminSession(r.Context(), parts[1]); err != nil {
			return "", err
		}
		return parts[1], nil
	}
	cookie, err := r.Cookie(adminCookieName)
	if err != nil {
		return "", auth.ErrInvalidSession
	}
	if err := h.Auth.AuthenticateAdminSession(r.Context(), cookie.Value); err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (h *AdminHandler) setAdminCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: adminCookieName, Value: token, Path: "/admin", Expires: expires,
		MaxAge: int(time.Until(expires).Seconds()), Secure: h.SecureCookies,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

func (h *AdminHandler) clearAdminCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: adminCookieName, Value: "", Path: "/admin", MaxAge: -1,
		Expires: time.Unix(1, 0), Secure: h.SecureCookies,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

type adminPasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AdminHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input adminPasswordChangeRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.Auth.ChangePassword(r.Context(), input.CurrentPassword, input.NewPassword); err != nil {
		if writeStorageBusyError(w, err) {
			return
		}
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Current password is incorrect")
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not change admin password")
		}
		return
	}
	h.clearAdminCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) overview(w http.ResponseWriter, r *http.Request) {
	if h.Operator == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Overview is unavailable")
		return
	}
	result, err := h.Operator.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load overview")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *AdminHandler) activity(w http.ResponseWriter, r *http.Request) {
	if h.Operator == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Activity is unavailable")
		return
	}
	limit := 100
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 || parsed > 500 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = parsed
	}
	entries, err := h.Operator.Activity(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load activity")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": entries})
}
