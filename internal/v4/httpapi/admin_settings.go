package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/v4/operator"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

type webhookSettingsRequest struct {
	Endpoint     string `json:"endpoint"`
	RotateSecret bool   `json:"rotate_secret,omitempty"`
}

type profileDestinationRequest struct {
	UPIID     string `json:"upi_id"`
	PayeeName string `json:"payee_name,omitempty"`
}

type profileRequest struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	UPIID     string `json:"upi_id"`
	PayeeName string `json:"payee_name,omitempty"`
	Parser    string `json:"parser"`
	Enabled   bool   `json:"enabled"`
}

func (h *AdminHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	if h.Settings == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Settings are unavailable")
		return
	}
	setting, err := h.Settings.Webhook(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load settings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhook": webhookSettingsResponse(setting)})
}

func (h *AdminHandler) updateWebhookSettings(w http.ResponseWriter, r *http.Request) {
	if h.Settings == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Settings are unavailable")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input webhookSettingsRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	setting, secret, err := h.Settings.ConfigureWebhook(r.Context(), input.Endpoint, input.RotateSecret)
	if err != nil {
		if errors.Is(err, webhooks.ErrInvalidConfig) {
			writeError(w, http.StatusBadRequest, "invalid_webhook", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not update webhook settings")
		return
	}
	response := map[string]any{"webhook": webhookSettingsResponse(setting)}
	if secret != "" {
		response["signing_secret"] = secret
	}
	writeJSON(w, http.StatusOK, response)
}

func webhookSettingsResponse(value operator.WebhookSettings) map[string]any {
	return map[string]any{
		"enabled": value.Enabled, "endpoint": value.Endpoint,
		"secret_configured": value.SecretConfigured,
	}
}

func (h *AdminHandler) listProfiles(w http.ResponseWriter, r *http.Request) {
	if h.Profiles == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Collection profiles are unavailable")
		return
	}
	items, err := h.Profiles.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load collection profiles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *AdminHandler) upsertProfile(w http.ResponseWriter, r *http.Request) {
	if h.Profiles == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Collection profiles are unavailable")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input profileRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	profile, err := h.Profiles.Upsert(r.Context(), profiles.UpsertInput{
		ID: input.ID, Label: input.Label, UPIID: input.UPIID,
		PayeeName: input.PayeeName, Parser: input.Parser, Enabled: input.Enabled,
	})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *AdminHandler) updateProfileDestination(w http.ResponseWriter, r *http.Request) {
	if h.Profiles == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Collection profiles are unavailable")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input profileDestinationRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "active" {
		active, err := h.Profiles.Active(r.Context())
		if err != nil {
			writeProfileError(w, err)
			return
		}
		id = active.ID
	}
	profile, err := h.Profiles.UpdateDestination(r.Context(), id, profiles.DestinationInput{
		UPIID: input.UPIID, PayeeName: input.PayeeName,
	})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *AdminHandler) activateProfile(w http.ResponseWriter, r *http.Request) {
	if h.Profiles == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Collection profiles are unavailable")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	profile, err := h.Profiles.Activate(r.Context(), id)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile})
}
func writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profiles.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "Collection profile not found")
	case errors.Is(err, profiles.ErrProfileDisabled):
		writeError(w, http.StatusConflict, "profile_disabled", "Disabled collection profile cannot be activated")
	case errors.Is(err, profiles.ErrCannotDisableActiveProfile):
		writeError(w, http.StatusConflict, "active_profile", "Activate another profile before disabling this one")
	case errors.Is(err, profiles.ErrInvalidProfile):
		writeError(w, http.StatusBadRequest, "invalid_profile", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not update collection profile")
	}
}
