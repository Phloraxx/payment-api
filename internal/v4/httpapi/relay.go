package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/v4/relay"
)

const relayHTTPBodyLimit = 64 << 10

const (
	relayDeviceHeader    = "X-PayGate-Relay-Device"
	relayTimeHeader      = "X-PayGate-Relay-Time"
	relaySignatureHeader = "X-PayGate-Relay-Signature"
)

type RelayHandler struct {
	Relay *relay.Service
	mux   *http.ServeMux
}

func NewRelayHandler(service *relay.Service) *RelayHandler {
	h := &RelayHandler{Relay: service, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v4/relay/pair", h.pair)
	h.mux.HandleFunc("POST "+relay.EventPath, h.event)
	h.mux.HandleFunc("POST "+relay.HeartbeatPath, h.heartbeat)
	return h
}
func (h *RelayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || h.Relay == nil || h.mux == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate relay is unavailable")
		return
	}
	if r.URL.RawQuery != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Relay endpoints do not accept query parameters")
		return
	}
	h.mux.ServeHTTP(w, r)
}

type relayPairRequest struct {
	Token          string `json:"token"`
	Name           string `json:"name"`
	PublicKeyPEM   string `json:"public_key_pem"`
	AppVersion     string `json:"app_version,omitempty"`
	DeviceModel    string `json:"device_model,omitempty"`
	AndroidVersion string `json:"android_version,omitempty"`
}

func (h *RelayHandler) pair(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input relayPairRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.Relay.PairDevice(r.Context(), relay.PairDeviceInput{
		Token: input.Token, Name: input.Name, PublicKeyPEM: input.PublicKeyPEM,
		AppVersion: input.AppVersion, DeviceModel: input.DeviceModel, AndroidVersion: input.AndroidVersion,
	})
	if err != nil {
		writeRelayPairError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": result.DeviceID, "enabled": result.Enabled,
		"replaced_device_id": emptyToNil(result.ReplacedDeviceID),
	})
}

func (h *RelayHandler) event(w http.ResponseWriter, r *http.Request) {
	raw, ok := readRelayBody(w, r)
	if !ok {
		return
	}
	result, err := h.Relay.IngestSigned(r.Context(), relayAuth(r, relay.EventPath), raw)
	if err != nil {
		writeRelayError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *RelayHandler) heartbeat(w http.ResponseWriter, r *http.Request) {
	raw, ok := readRelayBody(w, r)
	if !ok {
		return
	}
	result, err := h.Relay.HeartbeatSigned(r.Context(), relayAuth(r, relay.HeartbeatPath), raw)
	if err != nil {
		writeRelayError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func relayAuth(r *http.Request, path string) relay.RequestAuth {
	return relay.RequestAuth{
		DeviceID: r.Header.Get(relayDeviceHeader), Timestamp: r.Header.Get(relayTimeHeader),
		Signature: r.Header.Get(relaySignatureHeader), Method: r.Method, Path: path,
	}
}

func readRelayBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, relayHTTPBodyLimit)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Relay body is too large or unreadable")
		return nil, false
	}
	return raw, true
}
func writeRelayPairError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, relay.ErrPairingTokenInvalid), errors.Is(err, relay.ErrPairingTokenExpired), errors.Is(err, relay.ErrPairingTokenUsed):
		writeError(w, http.StatusUnauthorized, "invalid_pairing", "Pairing link is invalid or expired")
	case errors.Is(err, relay.ErrRelayAlreadyActive):
		writeError(w, http.StatusConflict, "device_already_connected", "A PayGate phone is already connected")
	case errors.Is(err, relay.ErrInvalidDevice):
		writeError(w, http.StatusBadRequest, "invalid_device", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not connect PayGate phone")
	}
}

func writeRelayError(w http.ResponseWriter, err error) {
	var relayErr *relay.Error
	if errors.As(err, &relayErr) {
		writeError(w, relayErr.HTTPStatus, strings.ToLower(relayErr.Code), relayErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "PayGate could not process the relay request")
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
