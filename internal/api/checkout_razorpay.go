package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/Phloraxx/payment-api/internal/razorpaycore"
	"github.com/pocketbase/pocketbase/core"
)

var (
	checkoutRazorpayOrderID   = regexp.MustCompile(`^order_[A-Za-z0-9]{6,64}$`)
	checkoutRazorpayPaymentID = regexp.MustCompile(`^pay_[A-Za-z0-9_]{6,64}$`)
	checkoutRazorpaySignature = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type checkoutRazorpayMode struct {
	name, disabledCode, keyID, displayName string
	enabled                                bool
	service                                *razorpaycore.Service
	livePilot                              bool
}

func (a *API) checkoutRazorpayModeFor(name string) (checkoutRazorpayMode, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "test":
		return checkoutRazorpayMode{
			name: "test", disabledCode: "RAZORPAY_TEST_DISABLED",
			keyID: a.Config.RazorpayTestKeyID, displayName: a.Config.RazorpayTestDisplayName,
			enabled: a.Config.RazorpayTestEnabled && a.RazorpayTest != nil, service: a.RazorpayTest,
		}, true
	case "live":
		return checkoutRazorpayMode{
			name: "live", disabledCode: "RAZORPAY_LIVE_DISABLED",
			keyID: a.Config.RazorpayLiveKeyID, displayName: a.Config.RazorpayLiveDisplayName,
			enabled: a.Config.RazorpayLiveEnabled && a.RazorpayLive != nil, service: a.RazorpayLive, livePilot: true,
		}, true
	default:
		return checkoutRazorpayMode{}, false
	}
}

func (m checkoutRazorpayMode) disabled(e *core.RequestEvent) error {
	return checkoutError(e, http.StatusNotFound, m.disabledCode, "Razorpay "+m.name+" mode is disabled.")
}

func (m checkoutRazorpayMode) orderResponse(record *core.Record) map[string]any {
	return razorpaycore.OrderResponse(record, m.keyID, m.displayName)
}
func (a *API) checkoutRazorpayConfig(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	if err := a.checkoutRateLimit(e, false); err != nil {
		return err
	}
	mode, ok := a.checkoutRazorpayModeFor(e.Request.PathValue("mode"))
	if !ok {
		return e.NotFoundError("route not found", nil)
	}
	keyID := ""
	if mode.enabled {
		keyID = mode.keyID
	}
	return e.JSON(http.StatusOK, map[string]any{
		"enabled": mode.enabled, "keyId": keyID,
		"displayName": mode.displayName, "mode": mode.name,
	})
}

type checkoutRazorpayCreateBody struct {
	Amount json.RawMessage `json:"amount"`
}

func (a *API) checkoutRazorpayCreateOrder(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	mode, ok := a.checkoutRazorpayModeFor(e.Request.PathValue("mode"))
	if !ok {
		return e.NotFoundError("route not found", nil)
	}
	if !mode.enabled {
		return mode.disabled(e)
	}
	if ct := strings.ToLower(strings.TrimSpace(strings.Split(e.Request.Header.Get("Content-Type"), ";")[0])); ct != "application/json" {
		return checkoutError(e, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Content-Type must be application/json.")
	}
	requestID := strings.ToLower(strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")))
	if !checkoutRequestID.MatchString(requestID) {
		return checkoutError(e, http.StatusBadRequest, "INVALID_REQUEST", "requestId must be a UUID.")
	}
	var body checkoutRazorpayCreateBody
	if err := decodeJSON(e, &body); err != nil {
		return checkoutError(e, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
	}
	rupees, err := money.ParseWholeRupees(body.Amount)
	if err != nil {
		return checkoutError(e, http.StatusBadRequest, "INVALID_REQUEST", "Amount must be a positive whole number of rupees.")
	}
	amountPaise, err := money.RupeesToPaise(rupees)
	if err != nil || (mode.livePilot && amountPaise != 100) {
		message := "Razorpay amount must be between ₹1 and ₹1,00,000."
		if mode.livePilot {
			message = "Razorpay Live pilot amount must be exactly ₹1."
		}
		return checkoutError(e, http.StatusBadRequest, "INVALID_REQUEST", message)
	}
	if err := a.checkoutRateLimit(e, true); err != nil {
		return err
	}
	record, replayed, err := mode.service.Create(e.Request.Context(), razorpaycore.CreateInput{
		AmountPaise: amountPaise, ExternalID: "portal:" + requestID,
		IdempotencyKey: requestID,
	})
	if err != nil {
		return a.checkoutDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, mode.orderResponse(record))
}

func (a *API) checkoutRazorpayGetOrder(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	mode, ok := a.checkoutRazorpayModeFor(e.Request.PathValue("mode"))
	if !ok {
		return e.NotFoundError("route not found", nil)
	}
	if !mode.enabled {
		return mode.disabled(e)
	}
	id := strings.TrimSpace(e.Request.PathValue("id"))
	if !checkoutPaymentID.MatchString(id) {
		return checkoutError(e, http.StatusBadRequest, "INVALID_ORDER_ID", "Invalid Razorpay order ID.")
	}
	if err := a.checkoutRateLimit(e, false); err != nil {
		return err
	}
	record, err := mode.service.Get(id)
	if err != nil {
		return a.checkoutDomainError(e, err)
	}
	return e.JSON(http.StatusOK, mode.orderResponse(record))
}

type checkoutRazorpayVerifyBody struct {
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	RazorpaySignature string `json:"razorpay_signature"`
}

func (a *API) checkoutRazorpayVerify(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	mode, ok := a.checkoutRazorpayModeFor(e.Request.PathValue("mode"))
	if !ok {
		return e.NotFoundError("route not found", nil)
	}
	if !mode.enabled {
		return mode.disabled(e)
	}
	id := strings.TrimSpace(e.Request.PathValue("id"))
	if !checkoutPaymentID.MatchString(id) {
		return checkoutError(e, http.StatusBadRequest, "INVALID_ORDER_ID", "Invalid Razorpay order ID.")
	}
	if ct := strings.ToLower(strings.TrimSpace(strings.Split(e.Request.Header.Get("Content-Type"), ";")[0])); ct != "application/json" {
		return checkoutError(e, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Content-Type must be application/json.")
	}
	var body checkoutRazorpayVerifyBody
	if err := decodeJSON(e, &body); err != nil {
		return checkoutError(e, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
	}
	if !checkoutRazorpayOrderID.MatchString(body.RazorpayOrderID) ||
		!checkoutRazorpayPaymentID.MatchString(body.RazorpayPaymentID) ||
		!checkoutRazorpaySignature.MatchString(body.RazorpaySignature) {
		return checkoutError(e, http.StatusBadRequest, "INVALID_REQUEST", "Invalid Razorpay verification response.")
	}
	if err := a.checkoutRateLimit(e, false); err != nil {
		return err
	}
	record, err := mode.service.Verify(e.Request.Context(), razorpaycore.VerifyInput{
		LocalOrderID: id, RazorpayOrderID: body.RazorpayOrderID,
		RazorpayPaymentID: body.RazorpayPaymentID, RazorpaySignature: body.RazorpaySignature,
	})
	if err != nil {
		return a.checkoutDomainError(e, err)
	}
	return e.JSON(http.StatusOK, mode.orderResponse(record))
}
