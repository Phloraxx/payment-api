package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/v4/relay"
)

type pairingSessionRequest struct {
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

func (h *AdminHandler) getDevice(w http.ResponseWriter, r *http.Request) {
	if h.Relay == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate device is unavailable")
		return
	}
	devices, err := h.Relay.Devices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load PayGate devices")
		return
	}
	var primary any
	if len(devices) > 0 {
		primary = devices[0]
	}
	writeJSON(w, http.StatusOK, map[string]any{"device": primary, "devices": devices})
}

func (h *AdminHandler) createPairingSession(w http.ResponseWriter, r *http.Request) {
	if h.Relay == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate device is unavailable")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input pairingSessionRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	session, err := h.Relay.CreatePairing(r.Context(), input.ReplaceExisting)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not create pairing session")
		return
	}
	response := map[string]any{
		"token": session.Token, "expires_at": session.ExpiresAt,
		"replace_existing": session.ReplaceExisting,
	}
	if base := strings.TrimRight(strings.TrimSpace(h.PairingBaseURL), "/"); base != "" {
		response["pairing_url"] = base + "/device/pair/" + session.Token
	}
	writeJSON(w, http.StatusCreated, response)
}
func (h *AdminHandler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	if h.Relay == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate device is unavailable")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.Relay.RevokeDevice(r.Context(), id); err != nil {
		if errors.Is(err, relay.ErrInvalidDevice) {
			writeError(w, http.StatusNotFound, "device_not_found", "PayGate device not found or already revoked")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not revoke PayGate device")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) retryWebhook(w http.ResponseWriter, r *http.Request) {
	if h.Webhooks == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Webhook delivery is unavailable")
		return
	}
	if err := h.Webhooks.RetryOne(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusConflict, "webhook_not_retryable", "Webhook is not retryable")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "pending"})
}
