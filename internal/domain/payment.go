package domain

import "time"

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusPaid      PaymentStatus = "paid"
	StatusExpired   PaymentStatus = "expired"
	StatusCancelled PaymentStatus = "cancelled"
	StatusLate      PaymentStatus = "late"
)

type ParsedSMS struct {
	AmountPaise int64
	RRN         string
	UPIId       string
	PayerName   string
	OccurredAt  time.Time
}

type Payment struct {
	ID             string
	RequestedPaise int64
	PayablePaise   int64
	Status         PaymentStatus
	ExpiresAt      time.Time
	ReuseAfter     time.Time
	RRN            string
	UPIId          string
	PayerName      string
	PaidAt         time.Time
	ResolvedAt     time.Time
	ExternalID     string
	IdempotencyKey string
}
