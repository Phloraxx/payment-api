package domain

import "net/http"

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func New(code, message string, status int) *Error {
	return &Error{Code: code, Message: message, Status: status}
}

func InvalidAmount() *Error {
	return New("INVALID_AMOUNT", "amount must be a positive whole number of INR rupees", http.StatusBadRequest)
}

func InvalidExternalID() *Error {
	return New("INVALID_EXTERNAL_ID", "externalId must be at most 255 characters", http.StatusBadRequest)
}

func InvalidIdempotencyKey() *Error {
	return New("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be at most 255 characters", http.StatusBadRequest)
}

func InvalidMetadata() *Error {
	return New("INVALID_METADATA", "metadata must be valid JSON no larger than 1 MiB", http.StatusBadRequest)
}

func InvalidSMS(message string) *Error {
	return New("INVALID_SMS", message, http.StatusBadRequest)
}

func PaymentNotFound() *Error {
	return New("PAYMENT_NOT_FOUND", "payment not found", http.StatusNotFound)
}

func CapacityExhausted() *Error {
	return New("AMOUNT_CAPACITY_EXHAUSTED", "all 99 paise fingerprints for this amount are temporarily unavailable", http.StatusConflict)
}

func IdempotencyConflict() *Error {
	return New("IDEMPOTENCY_CONFLICT", "the idempotency key was already used with different payment parameters", http.StatusConflict)
}

func PaymentResolved(status string) *Error {
	e := New("PAYMENT_ALREADY_RESOLVED", "payment is already resolved", http.StatusConflict)
	e.Details = map[string]any{"status": status}
	return e
}

func AmbiguousMatch() *Error {
	return New("AMBIGUOUS_PAYMENT_MATCH", "multiple active payments have the same payable amount; automatic confirmation was refused", http.StatusConflict)
}
