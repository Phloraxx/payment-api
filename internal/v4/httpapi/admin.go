package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/operator"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/relay"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

const adminCookieName = "paygate_admin"

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
	mux            *http.ServeMux
}

func NewAdminHandler(authService *auth.Service, paymentService *adminpayments.Service, operatorService *operator.Service,
	settingsService *operator.SettingsService, profileService *profiles.Service, relayService *relay.Service, webhookService *webhooks.Service) *AdminHandler {
	h := &AdminHandler{Auth: authService, Payments: paymentService, Operator: operatorService, Settings: settingsService,
		Profiles: profileService, Relay: relayService, Webhooks: webhookService, SecureCookies: true, mux: http.NewServeMux()}
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
	if _, err := h.adminToken(r); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Admin session is invalid")
		return
	}
	h.mux.ServeHTTP(w, r)
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
	session, err := h.Auth.CreateAdminSession(r.Context(), input.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrNotInitialized) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Password is incorrect")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "PayGate could not create the admin session")
		return
	}
	h.setAdminCookie(w, session.Token, session.ExpiresAt)
	response := adminLoginResponse{ExpiresAt: session.ExpiresAt}
	if strings.EqualFold(strings.TrimSpace(input.Client), "android") {
		response.Token = session.Token
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) logout(w http.ResponseWriter, r *http.Request) {
	token, err := h.adminToken(r)
	if err == nil {
		_ = h.Auth.RevokeAdminSession(r.Context(), token)
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
