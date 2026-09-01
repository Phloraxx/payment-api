package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/v4/auth"
)

type apiKeyCreateRequest struct {
	Label string `json:"label"`
}

func (h *AdminHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	items, err := h.Auth.ListAPIKeys(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load API keys")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *AdminHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input apiKeyCreateRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	created, err := h.Auth.CreateAPIKey(r.Context(), input.Label)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_api_key", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not create API key")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": created.ID, "label": created.Label, "secret": created.Secret,
		"created_at": created.CreatedAt,
	})
}

func (h *AdminHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.Auth.RevokeAPIKey(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrInvalidAPIKey) {
			writeError(w, http.StatusNotFound, "api_key_not_found", "API key not found or already revoked")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not revoke API key")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
