package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
)

const (
	merchantIdempotencyScope = "merchant:v1"
	maxJSONBodyBytes         = 32 << 10
)

type MerchantHandler struct {
	Auth     *auth.Service
	Payments *payments.Service
	mux      *http.ServeMux
}

func NewMerchantHandler(authService *auth.Service, paymentService *payments.Service) *MerchantHandler {
	h := &MerchantHandler{Auth: authService, Payments: paymentService, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /v1/payments", h.createPayment)
	h.mux.HandleFunc("GET /v1/payments/{id}", h.getPayment)
	h.mux.HandleFunc("POST /v1/payments/{id}/cancel", h.cancelPayment)
	return h
}
func (h *MerchantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || h.Auth == nil || h.Payments == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "PayGate is not ready")
		return
	}
	if _, err := h.authenticate(r); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
		return
	}
	h.mux.ServeHTTP(w, r)
}

func (h *MerchantHandler) authenticate(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", auth.ErrInvalidAPIKey
	}
	return h.Auth.AuthenticateAPIKey(r.Context(), parts[1])
}

type createPaymentRequest struct {
	Amount     int64           `json:"amount"`
	Name       string          `json:"name"`
	ExternalID string          `json:"external_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

func (h *MerchantHandler) createPayment(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input createPaymentRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key is required")
		return
	}
	if input.Amount <= 0 || input.Amount > math.MaxInt64/100 {
		writeError(w, http.StatusBadRequest, "invalid_amount", "amount must be a positive whole INR value")
		return
	}
	result, err := h.Payments.Create(r.Context(), payments.CreateInput{
		RequestedAmountPaise: input.Amount * 100,
		Name:                 input.Name,
		ExternalID:           input.ExternalID,
		Metadata:             input.Metadata,
		IdempotencyScope:     merchantIdempotencyScope,
		IdempotencyKey:       idempotencyKey,
	})
	if err != nil {
		writePaymentError(w, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, paymentResponse(result.Payment, result.UPIURI))
}

func (h *MerchantHandler) getPayment(w http.ResponseWriter, r *http.Request) {
	result, err := h.Payments.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, paymentResponse(result.Payment, result.UPIURI))
}

func (h *MerchantHandler) cancelPayment(w http.ResponseWriter, r *http.Request) {
	payment, err := h.Payments.Cancel(r.Context(), r.PathValue("id"))
	if err != nil {
		writePaymentError(w, err)
		return
	}
	result, err := h.Payments.Get(r.Context(), payment.ID)
	if err != nil {
		writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, paymentResponse(result.Payment, result.UPIURI))
}

type merchantPaymentResponse struct {
	ID              string          `json:"id"`
	Object          string          `json:"object"`
	Name            string          `json:"name"`
	ExternalID      string          `json:"external_id,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
	Status          string          `json:"status"`
	Currency        string          `json:"currency"`
	RequestedAmount string          `json:"requested_amount"`
	PayableAmount   string          `json:"payable_amount"`
	Adjustment      string          `json:"adjustment"`
	UPIURI          string          `json:"upi_uri"`
	TransactionNote string          `json:"transaction_note"`
	CreatedAt       time.Time       `json:"created_at"`
	ExpiresAt       time.Time       `json:"expires_at"`
	GraceUntil      time.Time       `json:"grace_until"`
	PaidAt          *time.Time      `json:"paid_at"`
	Payer           *payerResponse  `json:"payer,omitempty"`
}

type payerResponse struct {
	Name  *string `json:"name"`
	UPIID *string `json:"upi_id"`
}

func paymentResponse(p payments.Payment, upiURI string) merchantPaymentResponse {
	response := merchantPaymentResponse{
		ID: p.ID, Object: "payment", Name: p.Name, ExternalID: p.ExternalID, Metadata: p.Metadata,
		Status: p.Status, Currency: "INR", RequestedAmount: money(p.RequestedAmountPaise),
		PayableAmount: money(p.PayableAmountPaise), Adjustment: money(p.AdjustmentPaise), UPIURI: upiURI,
		TransactionNote: payments.TransactionNote(p.ID),
		CreatedAt:       p.CreatedAt, ExpiresAt: p.ExpiresAt, GraceUntil: p.GraceUntil, PaidAt: p.PaidAt,
	}
	if p.Status == "paid" || p.PayerName != "" || p.PayerUPIID != "" {
		payer := &payerResponse{}
		if p.PayerName != "" {
			value := p.PayerName
			payer.Name = &value
		}
		if p.PayerUPIID != "" {
			value := p.PayerUPIID
			payer.UPIID = &value
		}
		response.Payer = payer
	}
	return response
}

func money(paise int64) string {
	return fmt.Sprintf("%d.%02d", paise/100, paise%100)
}

func isJSON(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "application/json" || strings.HasPrefix(value, "application/json;")
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}
func writePaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payments.ErrInvalidPaymentInput):
		writeError(w, http.StatusBadRequest, "invalid_payment", "Invalid payment request")
	case errors.Is(err, payments.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request")
	case errors.Is(err, payments.ErrNoActiveProfile):
		writeError(w, http.StatusServiceUnavailable, "collection_unavailable", "No collection destination is currently available")
	case errors.Is(err, payments.ErrPaymentNotFound):
		writeError(w, http.StatusNotFound, "payment_not_found", "Payment not found")
	case errors.Is(err, payments.ErrPaymentTerminal):
		writeError(w, http.StatusConflict, "payment_terminal", "Payment can no longer be cancelled")
	case errors.Is(err, payments.ErrPaymentCapacity):
		writeError(w, http.StatusServiceUnavailable, "payment_capacity_unavailable", "Payment capacity is temporarily unavailable")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "PayGate could not complete the request")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
