package domain

import "time"

type AuditEvent struct {
	Action     string
	ActorID    string
	ActorEmail string
	EntityType string
	EntityID   string
	Summary    string
	Details    any
	OccurredAt time.Time
}

type ReconciliationEntry struct {
	ID              string
	RunID           string
	RowNumber       int
	TransactionTime time.Time
	AmountPaise     int64
	RRN             string
	Description     string
	Status          string
	PaymentID       string
	Notes           string
	RawRow          any
}

type Refund struct {
	ID             string
	PaymentID      string
	AmountPaise    int64
	Status         string
	Reason         string
	Reference      string
	ExternalID     string
	IdempotencyKey string
	Metadata       any
	RequestedBy    string
	RequestedAt    time.Time
	CompletedAt    time.Time
}

type ReconciliationRun struct {
	ID            string
	Filename      string
	SHA256        string
	Status        string
	CreatedBy     string
	StartedAt     time.Time
	CompletedAt   time.Time
	TotalRows     int
	MatchedRows   int
	UnmatchedRows int
	DuplicateRows int
	ConflictRows  int
	InvalidRows   int
	Error         string
	Summary       any
}
