package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/razorpaytest"
	"github.com/pocketbase/pocketbase/core"
)

type razorpayTestCreateBody struct {
	AmountPaise int64  `json:"amountPaise"`
	ExternalID  string `json:"externalId"`
}

type razorpayTestVerifyBody struct {
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	RazorpaySignature string `json:"razorpay_signature"`
}

func (a *API) razorpayTestConfig(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	enabled := a.Config.RazorpayTestEnabled && a.RazorpayTest != nil
	keyID := ""
	if enabled {
		keyID = a.Config.RazorpayTestKeyID
	}
	return e.JSON(http.StatusOK, map[string]any{
		"enabled":     enabled,
		"keyId":       keyID,
		"displayName": a.Config.RazorpayTestDisplayName,
		"mode":        "test",
	})
}

func (a *API) razorpayTestCreateOrder(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if !a.razorpayTestAvailable() {
		return e.NotFoundError("Razorpay test rail is disabled", nil)
	}
	var body razorpayTestCreateBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	record, replayed, err := a.RazorpayTest.Create(e.Request.Context(), razorpaytest.CreateInput{
		AmountPaise: body.AmountPaise, ExternalID: body.ExternalID,
		IdempotencyKey: strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")), ActorID: e.Auth.Id,
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, razorpaytest.OrderResponse(record, a.Config.RazorpayTestKeyID, a.Config.RazorpayTestDisplayName))
}

func (a *API) razorpayTestGetOrder(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if !a.razorpayTestAvailable() {
		return e.NotFoundError("Razorpay test rail is disabled", nil)
	}
	record, err := a.RazorpayTest.Get(e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaytest.OrderResponse(record, a.Config.RazorpayTestKeyID, a.Config.RazorpayTestDisplayName))
}

func (a *API) razorpayTestVerify(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if !a.razorpayTestAvailable() {
		return e.NotFoundError("Razorpay test rail is disabled", nil)
	}
	var body razorpayTestVerifyBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	record, err := a.RazorpayTest.Verify(e.Request.Context(), razorpaytest.VerifyInput{
		LocalOrderID: e.Request.PathValue("id"), RazorpayOrderID: body.RazorpayOrderID,
		RazorpayPaymentID: body.RazorpayPaymentID, RazorpaySignature: body.RazorpaySignature,
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaytest.OrderResponse(record, a.Config.RazorpayTestKeyID, a.Config.RazorpayTestDisplayName))
}

func (a *API) razorpayTestRefresh(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if !a.razorpayTestAvailable() {
		return e.NotFoundError("Razorpay test rail is disabled", nil)
	}
	record, err := a.RazorpayTest.Refresh(e.Request.Context(), e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, razorpaytest.OrderResponse(record, a.Config.RazorpayTestKeyID, a.Config.RazorpayTestDisplayName))
}

func (a *API) razorpayTestWebhook(e *core.RequestEvent) error {
	if !a.razorpayTestAvailable() {
		return e.NotFoundError("Razorpay test rail is disabled", nil)
	}
	raw, err := io.ReadAll(io.LimitReader(e.Request.Body, maxRazorpayTestRequestBytes+1))
	if err != nil {
		return e.BadRequestError("failed to read Razorpay webhook", err)
	}
	if len(raw) > int(maxRazorpayTestRequestBytes) {
		return e.JSON(http.StatusRequestEntityTooLarge, map[string]any{"error": map[string]any{"code": "RAZORPAY_TEST_WEBHOOK_TOO_LARGE", "message": "webhook exceeds 1 MiB"}})
	}
	result, err := a.RazorpayTest.IngestWebhook(
		e.Request.Header.Get("X-Razorpay-Event-Id"),
		e.Request.Header.Get("X-Razorpay-Signature"), raw,
	)
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) razorpayTestAvailable() bool {
	return a.Config.RazorpayTestEnabled && a.RazorpayTest != nil
}

func (a *API) setOperatorSecurityHeaders(e *core.RequestEvent) {
	headers := e.Response.Header()
	csp := "default-src 'self'; base-uri 'none'; connect-src 'self'; font-src 'self'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data: blob:; object-src 'none'; script-src 'self'; style-src 'self'"
	if a.Config.RazorpayTestEnabled {
		csp = "default-src 'self'; base-uri 'none'; connect-src 'self' https://api.razorpay.com https://*.razorpay.com; font-src 'self' https://*.razorpay.com; form-action 'self' https://api.razorpay.com; frame-ancestors 'none'; frame-src https://api.razorpay.com https://*.razorpay.com; img-src 'self' data: blob: https://*.razorpay.com; object-src 'none'; script-src 'self' https://checkout.razorpay.com; style-src 'self'"
	}
	headers.Set("Content-Security-Policy", csp)
	headers.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=()")
	headers.Set("Referrer-Policy", "no-referrer")
	headers.Set("Strict-Transport-Security", "max-age=31536000")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}
