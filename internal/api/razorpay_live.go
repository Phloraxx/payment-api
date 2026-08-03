package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/razorpaylive"
	"github.com/pocketbase/pocketbase/core"
)

type razorpayLiveCreateBody struct {
	AmountPaise int64  `json:"amountPaise"`
	ExternalID  string `json:"externalId"`
}

type razorpayLiveVerifyBody struct {
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	RazorpaySignature string `json:"razorpay_signature"`
}

func (a *API) razorpayLiveConfig(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	enabled := a.Config.RazorpayLiveEnabled && a.RazorpayLive != nil
	keyID := ""
	if enabled {
		keyID = a.Config.RazorpayLiveKeyID
	}
	return e.JSON(http.StatusOK, map[string]any{
		"enabled":     enabled,
		"keyId":       keyID,
		"displayName": a.Config.RazorpayLiveDisplayName,
		"mode":        "live",
	})
}

func (a *API) razorpayLiveCreateOrder(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if !a.razorpayLiveAvailable() {
		return e.NotFoundError("Razorpay live rail is disabled", nil)
	}
	var body razorpayLiveCreateBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	record, replayed, err := a.RazorpayLive.Create(e.Request.Context(), razorpaylive.CreateInput{
		AmountPaise: body.AmountPaise, ExternalID: body.ExternalID,
		IdempotencyKey: strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")), ActorID: a.razorpayActorID(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, razorpaylive.OrderResponse(record, a.Config.RazorpayLiveKeyID, a.Config.RazorpayLiveDisplayName))
}

func (a *API) razorpayLiveGetOrder(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if !a.razorpayLiveAvailable() {
		return e.NotFoundError("Razorpay live rail is disabled", nil)
	}
	record, err := a.RazorpayLive.Get(e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaylive.OrderResponse(record, a.Config.RazorpayLiveKeyID, a.Config.RazorpayLiveDisplayName))
}

func (a *API) razorpayLiveVerify(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if !a.razorpayLiveAvailable() {
		return e.NotFoundError("Razorpay live rail is disabled", nil)
	}
	var body razorpayLiveVerifyBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	record, err := a.RazorpayLive.Verify(e.Request.Context(), razorpaylive.VerifyInput{
		LocalOrderID: e.Request.PathValue("id"), RazorpayOrderID: body.RazorpayOrderID,
		RazorpayPaymentID: body.RazorpayPaymentID, RazorpaySignature: body.RazorpaySignature,
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaylive.OrderResponse(record, a.Config.RazorpayLiveKeyID, a.Config.RazorpayLiveDisplayName))
}

func (a *API) razorpayLiveRefresh(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if !a.razorpayLiveAvailable() {
		return e.NotFoundError("Razorpay live rail is disabled", nil)
	}
	record, err := a.RazorpayLive.Refresh(e.Request.Context(), e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaylive.OrderResponse(record, a.Config.RazorpayLiveKeyID, a.Config.RazorpayLiveDisplayName))
}

func (a *API) razorpayLiveWebhook(e *core.RequestEvent) error {
	if !a.razorpayLiveAvailable() {
		return e.NotFoundError("Razorpay live rail is disabled", nil)
	}
	raw, err := io.ReadAll(io.LimitReader(e.Request.Body, maxRazorpayLiveRequestBytes+1))
	if err != nil {
		return e.BadRequestError("failed to read Razorpay webhook", err)
	}
	if len(raw) > int(maxRazorpayLiveRequestBytes) {
		return e.JSON(http.StatusRequestEntityTooLarge, map[string]any{"error": map[string]any{"code": "RAZORPAY_LIVE_WEBHOOK_TOO_LARGE", "message": "webhook exceeds 1 MiB"}})
	}
	result, err := a.RazorpayLive.IngestWebhook(
		e.Request.Header.Get("X-Razorpay-Event-Id"),
		e.Request.Header.Get("X-Razorpay-Signature"), raw,
	)
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) razorpayLiveAvailable() bool {
	return a.Config.RazorpayLiveEnabled && a.RazorpayLive != nil
}
