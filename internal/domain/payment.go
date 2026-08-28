package domain

import "time"

type PaymentStatus string

type PaymentAccount string

const (
	StatusPending   PaymentStatus = "pending"
	StatusPaid      PaymentStatus = "paid"
	StatusExpired   PaymentStatus = "expired"
	StatusCancelled PaymentStatus = "cancelled"
	StatusLate      PaymentStatus = "late"
)

const (
	PaymentAccountKotak PaymentAccount = "kotak"
	PaymentAccountSlice PaymentAccount = "slice"
	PaymentAccountPaytm PaymentAccount = "paytm"
)

type ParsedSMS struct {
	AmountPaise int64
	RRN         string
	UPIId       string
	PayerName   string
	OccurredAt  time.Time
	Account     PaymentAccount
}

type Payment struct {
	ID                string
	Account           PaymentAccount
	RequestedPaise    int64
	PayablePaise      int64
	Status            PaymentStatus
	CreatedAt         time.Time
	ExpiresAt         time.Time
	ReuseAfter        time.Time
	RRN               string
	UPIId             string
	PayerName         string
	EvidenceSource    string
	EvidenceReference string
	PaidAt            time.Time
	ResolvedAt        time.Time
	ExternalID        string
	IdempotencyKey    string
	Metadata          any
}
