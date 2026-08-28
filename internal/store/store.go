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
	SMSEvents() SMSEventRepository
	EmailEvents() EmailEventRepository
	ReconciliationRuns() ReconciliationRunRepository
	ReconciliationEntries() ReconciliationEntryRepository
	Audit() AuditRepository
	Refunds() RefundRepository
	NotificationEvents() NotificationEventRepository
	Reviews() ReviewRepository
	Relay() RelayRepository
	RelayEvents() RelayEventRepository
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
	ListFingerprintCandidates(account domain.PaymentAccount, amount int64, now time.Time, limit int) ([]*domain.Payment, error)
	FindReconciliationCandidates(account domain.PaymentAccount, amount int64, transactionTime, now time.Time, limit int) ([]*domain.Payment, error)
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

type SMSEventRepository interface {
	Get(id string) (*domain.SMSEvent, error)
	FindBySourceEvent(source, sourceEventID string) (*domain.SMSEvent, error)
	Create(event *domain.SMSEvent) error
	Save(event *domain.SMSEvent) error
	ListBySourceSince(source string, since time.Time, limit int) ([]*domain.SMSEvent, error)
}

type EmailEventRepository interface {
	Get(id string) (*domain.EmailEvent, error)
	FindBySourceEvent(source, sourceEventID string) (*domain.EmailEvent, error)
	Create(event *domain.EmailEvent) error
	Save(event *domain.EmailEvent) error
}

type ReconciliationRunRepository interface {
	FindCompletedByHash(hash string) (*domain.ReconciliationRun, error)
	Get(id string) (*domain.ReconciliationRun, error)
	Create(run *domain.ReconciliationRun) error
	Save(run *domain.ReconciliationRun) error
}

type ReconciliationEntryRepository interface {
	Get(id string) (*domain.ReconciliationEntry, error)
	Create(entry *domain.ReconciliationEntry) error
	Save(entry *domain.ReconciliationEntry) error
}

type AuditRepository interface {
	Record(event domain.AuditEvent) error
}

type RefundRepository interface {
	Get(id string) (*domain.Refund, error)
	FindByIdempotencyKey(key string) (*domain.Refund, error)
	FindByReference(reference string) (*domain.Refund, error)
	ReservedAmount(paymentID string) (int64, error)
	Create(refund *domain.Refund) error
	Save(refund *domain.Refund) error
}

type NotificationEventRepository interface {
	FindBySourceEvent(source, sourceEventID string) (*domain.NotificationEvent, error)
	Get(id string) (*domain.NotificationEvent, error)
	Create(event *domain.NotificationEvent) error
	Save(event *domain.NotificationEvent) error
}

type ReviewRepository interface {
	FindByEvidence(smsEventID, emailEventID, reconciliationEntryID string) (*domain.ReviewCase, error)
	Create(review *domain.ReviewCase) error
	Get(id string) (*domain.ReviewCase, error)
	Save(review *domain.ReviewCase) error
	OpenCount() (int64, error)
}

type RelayRepository interface {
	Get(id string) (*domain.RelayDevice, error)
	FindByDeviceID(deviceID string) (*domain.RelayDevice, error)
	Create(device *domain.RelayDevice) error
	Save(device *domain.RelayDevice) error
	All(limit int) ([]*domain.RelayDevice, error)
	EnabledDevices(limit int) ([]domain.RelayDeviceHealth, error)
}

type RelayEventRepository interface {
	FindByDeviceEvent(deviceRecordID, eventID string) (*domain.RelayEvent, error)
	Create(event *domain.RelayEvent) error
	Save(event *domain.RelayEvent) error
	Latest(deviceRecordID string) (*domain.RelayEvent, error)
	LatestMatched(deviceRecordID string) (*domain.RelayEvent, error)
	CountErrorsSince(deviceRecordID string, since time.Time) (int64, error)
	ListByPackageSince(appPackage string, since time.Time, limit int) ([]*domain.RelayEvent, error)
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
