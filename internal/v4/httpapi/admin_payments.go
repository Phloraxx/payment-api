package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
)

type adminPaymentResponse struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	ExternalID           string          `json:"external_id,omitempty"`
	Metadata             json.RawMessage `json:"metadata"`
	RequestedAmountPaise int64           `json:"requested_amount_paise"`
	PayableAmountPaise   int64           `json:"payable_amount_paise"`
	AdjustmentPaise      int64           `json:"adjustment_paise"`
	CollectionProfileID  string          `json:"collection_profile_id"`
	UPIIDSnapshot        string          `json:"upi_id_snapshot"`
	PayeeNameSnapshot    string          `json:"payee_name_snapshot,omitempty"`
	TransactionNote      string          `json:"transaction_note"`
	Status               string          `json:"status"`
	CreatedAt            time.Time       `json:"created_at"`
	ExpiresAt            time.Time       `json:"expires_at"`
	GraceUntil           time.Time       `json:"grace_until"`
	ReuseAfter           time.Time       `json:"reuse_after"`
	PaidAt               *time.Time      `json:"paid_at"`
	PayerName            string          `json:"payer_name,omitempty"`
	PayerUPIID           string          `json:"payer_upi_id,omitempty"`
	InternalNote         string          `json:"internal_note,omitempty"`
}

func adminPayment(p adminpayments.Payment) adminPaymentResponse {
	return adminPaymentResponse{
		ID: p.ID, Name: p.Name, ExternalID: p.ExternalID, Metadata: p.Metadata,
		RequestedAmountPaise: p.RequestedAmountPaise, PayableAmountPaise: p.PayableAmountPaise,
		AdjustmentPaise: p.AdjustmentPaise, CollectionProfileID: p.CollectionProfileID,
		UPIIDSnapshot: p.UPIIDSnapshot, PayeeNameSnapshot: p.PayeeNameSnapshot,
		TransactionNote: payments.TransactionNote(p.ID), Status: p.Status,
		CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, GraceUntil: p.GraceUntil, ReuseAfter: p.ReuseAfter,
		PaidAt: p.PaidAt, PayerName: p.PayerName, PayerUPIID: p.PayerUPIID, InternalNote: p.InternalNote,
	}
}

func (h *AdminHandler) listPayments(w http.ResponseWriter, r *http.Request) {
	if h.Payments == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Payments are unavailable")
		return
	}
	input := adminpayments.ListInput{
		Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
		ExternalID: r.URL.Query().Get("external_id"), ProfileID: r.URL.Query().Get("profile"),
	}
	var err error
	input.Limit, err = queryInt(r, "limit", 50, 1, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}
	input.Offset, err = queryInt(r, "offset", 0, 0, 1_000_000)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_offset", err.Error())
		return
	}
	if input.CreatedFrom, err = queryTime(r, "created_from"); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_created_from", err.Error())
		return
	}
	if input.CreatedTo, err = queryTime(r, "created_to"); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_created_to", err.Error())
		return
	}
	result, err := h.Payments.List(r.Context(), input)
	if err != nil {
		if errors.Is(err, adminpayments.ErrInvalidFilter) {
			writeError(w, http.StatusBadRequest, "invalid_filter", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load payments")
		return
	}
	items := make([]adminPaymentResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, adminPayment(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items, "total": result.Total, "limit": result.Limit, "offset": result.Offset,
	})
}

func (h *AdminHandler) getAdminPayment(w http.ResponseWriter, r *http.Request) {
	detail, err := h.Payments.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAdminPaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"payment": adminPayment(detail.Payment), "history": detail.History, "webhooks": detail.Webhooks,
	})
}

type adminEditPaymentRequest struct {
	Name         *string          `json:"name,omitempty"`
	ExternalID   *string          `json:"external_id,omitempty"`
	Metadata     *json.RawMessage `json:"metadata,omitempty"`
	Status       *string          `json:"status,omitempty"`
	PayerName    *string          `json:"payer_name,omitempty"`
	PayerUPIID   *string          `json:"payer_upi_id,omitempty"`
	PaidAt       *time.Time       `json:"paid_at,omitempty"`
	InternalNote *string          `json:"internal_note,omitempty"`
}

func (h *AdminHandler) editAdminPayment(w http.ResponseWriter, r *http.Request) {
	if h.Payments == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Payments are unavailable")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}
	var input adminEditPaymentRequest
	if err := decodeStrictJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	updated, err := h.Payments.Edit(r.Context(), r.PathValue("id"), adminpayments.EditInput{
		Name: input.Name, ExternalID: input.ExternalID, Metadata: input.Metadata,
		Status: input.Status, PayerName: input.PayerName, PayerUPIID: input.PayerUPIID,
		PaidAt: input.PaidAt, InternalNote: input.InternalNote,
	})
	if err != nil {
		writeAdminPaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": adminPayment(updated)})
}

func writeAdminPaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, adminpayments.ErrPaymentNotFound):
		writeError(w, http.StatusNotFound, "payment_not_found", "Payment not found")
	case errors.Is(err, adminpayments.ErrInvalidEdit):
		writeError(w, http.StatusBadRequest, "invalid_edit", err.Error())
	case errors.Is(err, adminpayments.ErrUnsafeReopen):
		writeError(w, http.StatusConflict, "unsafe_reopen", "Payment can no longer be safely reopened")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not update payment")
	}
}
func queryInt(r *http.Request, key string, fallback, min, max int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return 0, errors.New(key + " is outside the allowed range")
	}
	return parsed, nil
}

func queryTime(r *http.Request, key string) (*time.Time, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, errors.New(key + " must be RFC3339")
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
