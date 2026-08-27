package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/androidrelay"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

const maxAndroidRelayRequestBytes int64 = 256 << 10

func (a *API) androidRelayEnroll(e *core.RequestEvent) error {
	if a.AndroidRelay == nil || strings.TrimSpace(a.Config.AndroidRelayPairingSecret) == "" {
		return e.NotFoundError("route not found", nil)
	}
	if !constantTimeEqual(a.Config.AndroidRelayPairingSecret, e.Request.Header.Get("X-Pairing-Secret")) {
		return e.UnauthorizedError("invalid pairing secret", nil)
	}
	var body androidrelay.EnrollmentInput
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	result, err := a.AndroidRelay.Enroll(body)
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusCreated, result)
}

func (a *API) androidRelayEvent(e *core.RequestEvent) error {
	if a.AndroidRelay == nil {
		return e.NotFoundError("route not found", nil)
	}
	rawBody, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return e.BadRequestError("could not read request body", err)
	}
	device, err := a.AndroidRelay.Verify(
		e.Request.Header.Get("X-PayGate-Relay-Device"),
		e.Request.Header.Get("X-PayGate-Relay-Time"),
		e.Request.Header.Get("X-PayGate-Relay-Signature"),
		e.Request.Method, e.Request.URL.Path, rawBody,
	)
	if err != nil {
		return writeDomainError(e, err)
	}
	var body androidrelay.EventInput
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	var rawPayload any
	_ = json.Unmarshal(rawBody, &rawPayload)
	result, err := a.AndroidRelay.Ingest(device, body, rawPayload)
	if err != nil {
		if _, ok := err.(*domain.Error); ok {
			return writeDomainErrorWithData(e, err, map[string]any{"event": result})
		}
		return writeDomainError(e, err)
	}
	status := http.StatusAccepted
	if result.Duplicate {
		status = http.StatusOK
	}
	return e.JSON(status, result)
}
