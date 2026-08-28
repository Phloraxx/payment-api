package store

import (
	"context"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
)

// Database exposes typed repository access without leaking the persistence
// framework into application logic. Write is transaction-scoped; View uses the
// current committed database state.
type Database interface {
	View(context.Context, func(UnitOfWork) error) error
	Write(context.Context, func(UnitOfWork) error) error
}

type UnitOfWork interface {
	Payments() PaymentRepository
	Relay() RelayRepository
	Outbox() OutboxRepository
}

type PaymentRepository interface {
	Get(id string) (*domain.Payment, error)
	FindByIdempotencyKey(key string) (*domain.Payment, error)
	IsFingerprintBlocked(payableAmount int64, now time.Time) (bool, error)
	Create(payment NewPayment) (*domain.Payment, error)
	Save(payment *domain.Payment) error
	FindByEvidenceReference(kind domain.EvidenceReferenceKind, reference string) (*domain.Payment, error)
	FindOnTimeCandidates(account domain.PaymentAccount, amount int64, evidenceAt, createdBefore, now time.Time) ([]*domain.Payment, error)
	FindLateCandidates(account domain.PaymentAccount, amount int64, evidenceAt, createdBefore, now time.Time) ([]*domain.Payment, error)
	ListDue(now time.Time, limit int) ([]*domain.Payment, error)
	ListAll() ([]*domain.Payment, error)
	ListBlocked(now time.Time) ([]*domain.Payment, error)
}

type NewPayment struct {
	Account        domain.PaymentAccount
	RequestedPaise int64
	PayablePaise   int64
	CreatedAt      time.Time
	ExpiresAt      time.Time
	ReuseAfter     time.Time
	ExternalID     string
	IdempotencyKey string
	Metadata       any
}

type RelayRepository interface {
	EnabledDevices(limit int) ([]domain.RelayDeviceHealth, error)
}

type OutboxDelivery struct {
	EventID   string
	Event     string
	PaymentID string
	RefundID  string
	URL       string
	Body      string
	CreatedAt time.Time
}

type OutboxRepository interface {
	Enqueue(OutboxDelivery) error
}
