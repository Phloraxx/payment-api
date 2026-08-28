package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/androidrelay"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

const maxAndroidRelayRequestBytes int64 = 256 << 10

func (a *API) androidRelayEnroll(e *core.RequestEvent) error {
	if a.AndroidRelay == nil || !a.Config.AndroidRelayEnrollmentEnabled || strings.TrimSpace(a.Config.AndroidRelayPairingSecret) == "" {
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
	device, rawBody, err := a.verifyAndroidRelayRequest(e)
	if err != nil {
		return err
	}
	var body androidrelay.EventInput
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	var rawPayload any
	_ = json.Unmarshal(rawBody, &rawPayload)
	result, ingestErr := a.AndroidRelay.Ingest(device, body, rawPayload)
	if ingestErr != nil {
		if _, ok := ingestErr.(*domain.Error); ok {
			return writeDomainErrorWithData(e, ingestErr, map[string]any{"event": result})
		}
		return writeDomainError(e, ingestErr)
	}
	status := http.StatusAccepted
	if result.Duplicate {
		status = http.StatusOK
	}
	return e.JSON(status, result)
}

func (a *API) androidRelayHeartbeat(e *core.RequestEvent) error {
	device, rawBody, err := a.verifyAndroidRelayRequest(e)
	if err != nil {
		return err
	}
	var body androidrelay.HeartbeatInput
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	result, err := a.AndroidRelay.Heartbeat(device, body)
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) relayStatus(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.AndroidRelay == nil {
		return e.NotFoundError("relay service is unavailable", nil)
	}
	status, err := a.AndroidRelay.Status(a.Config.AndroidRelayStaleAfter)
	if err != nil {
		return e.InternalServerError("failed to load relay status", err)
	}
	return e.JSON(http.StatusOK, status)
}

func (a *API) relayDevices(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.AndroidRelay == nil {
		return e.NotFoundError("relay service is unavailable", nil)
	}
	devices, err := a.AndroidRelay.Devices(a.Config.AndroidRelayStaleAfter)
	if err != nil {
		return e.InternalServerError("failed to load relay devices", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"devices": devices})
}

type relayDeviceEnabledBody struct {
	Enabled *bool `json:"enabled"`
}

func (a *API) setRelayDeviceEnabled(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.AndroidRelay == nil {
		return e.NotFoundError("relay service is unavailable", nil)
	}
	var body relayDeviceEnabledBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	if body.Enabled == nil {
		return e.BadRequestError("enabled must be true or false", nil)
	}
	var record *core.Record
	err := e.App.RunInTransaction(func(tx core.App) error {
		updated, updateErr := a.AndroidRelay.SetEnabledInApp(tx, e.Request.PathValue("id"), *body.Enabled)
		if updateErr != nil {
			return updateErr
		}
		auditService := a.Audit
		if auditService == nil {
			auditService = audit.NewService(e.App)
		}
		if auditErr := auditService.RecordUoW(store.NewPocketBaseUnit(tx), audit.Entry{
			Action: "relay_device.enabled_changed", Actor: a.actor(e), EntityType: "relay_device", EntityID: updated.Id,
			Summary: "Android relay device enabled state changed", Details: map[string]any{"enabled": *body.Enabled, "deviceName": updated.GetString("name")},
		}); auditErr != nil {
			return auditErr
		}
		record = updated
		return nil
	})
	if err != nil {
		if _, ok := err.(*domain.Error); ok {
			return writeDomainError(e, err)
		}
		return e.InternalServerError("failed to update relay device", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"id": record.Id, "enabled": record.GetBool("enabled")})
}

func (a *API) verifyAndroidRelayRequest(e *core.RequestEvent) (*core.Record, []byte, error) {
	if a.AndroidRelay == nil {
		return nil, nil, e.NotFoundError("route not found", nil)
	}
	rawBody, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return nil, nil, e.BadRequestError("could not read request body", err)
	}
	device, err := a.AndroidRelay.Verify(
		e.Request.Header.Get("X-PayGate-Relay-Device"),
		e.Request.Header.Get("X-PayGate-Relay-Time"),
		e.Request.Header.Get("X-PayGate-Relay-Signature"),
		e.Request.Method, e.Request.URL.Path, rawBody,
	)
	if err != nil {
		return nil, nil, writeDomainError(e, err)
	}
	return device, rawBody, nil
}
