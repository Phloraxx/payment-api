package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type pocketBaseDatabase struct{ app core.App }
type pocketBaseUnit struct{ app core.App }
type pocketBasePayments struct{ app core.App }
type pocketBaseRelay struct{ app core.App }
type pocketBaseOutbox struct{ app core.App }

func NewPocketBase(app core.App) Database { return &pocketBaseDatabase{app: app} }

// NewPocketBaseUnit adapts an already-open PocketBase transaction while
// legacy callers are migrated to Database.Write.
func NewPocketBaseUnit(app core.App) UnitOfWork { return &pocketBaseUnit{app: app} }

func (db *pocketBaseDatabase) View(_ context.Context, fn func(UnitOfWork) error) error {
	return fn(&pocketBaseUnit{app: db.app})
}

func (db *pocketBaseDatabase) Write(_ context.Context, fn func(UnitOfWork) error) error {
	return db.app.RunInTransaction(func(tx core.App) error { return fn(&pocketBaseUnit{app: tx}) })
}

func (u *pocketBaseUnit) Payments() PaymentRepository { return &pocketBasePayments{app: u.app} }
func (u *pocketBaseUnit) Relay() RelayRepository      { return &pocketBaseRelay{app: u.app} }
func (u *pocketBaseUnit) Outbox() OutboxRepository    { return &pocketBaseOutbox{app: u.app} }

func (r *pocketBasePayments) Get(id string) (*domain.Payment, error) {
	record, err := r.app.FindRecordById("payments", id)
	if err != nil {
		return nil, err
	}
	return paymentFromRecord(record), nil
}

func (r *pocketBasePayments) FindByIdempotencyKey(key string) (*domain.Payment, error) {
	record, err := r.app.FindFirstRecordByData("payments", "idempotency_key", key)
	if err != nil {
		return nil, err
	}
	return paymentFromRecord(record), nil
}

func (r *pocketBasePayments) IsFingerprintBlocked(payableAmount int64, now time.Time) (bool, error) {
	record, err := r.app.FindFirstRecordByFilter("payments", "payable_amount = {:amount} && reuse_after > {:now}", dbx.Params{"amount": payableAmount, "now": storeDate(now)})
	if err == nil && record != nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *pocketBasePayments) Create(payment NewPayment) (*domain.Payment, error) {
	collection, err := r.app.FindCollectionByNameOrId("payments")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("created_at", payment.CreatedAt)
	record.Set("payment_account", string(payment.Account))
	record.Set("requested_amount", payment.RequestedPaise)
	record.Set("payable_amount", payment.PayablePaise)
	record.Set("status", string(domain.StatusPending))
	record.Set("expires_at", payment.ExpiresAt)
	record.Set("reuse_after", payment.ReuseAfter)
	record.Set("external_id", payment.ExternalID)
	record.Set("idempotency_key", payment.IdempotencyKey)
	if payment.Metadata != nil {
		record.Set("metadata", payment.Metadata)
	}
	if err := r.app.Save(record); err != nil {
		return nil, err
	}
	return paymentFromRecord(record), nil
}

func (r *pocketBasePayments) FindByEvidenceReference(kind domain.EvidenceReferenceKind, reference string) (*domain.Payment, error) {
	field := ""
	switch kind {
	case domain.EvidenceReferenceRRN:
		field = "rrn"
	case domain.EvidenceReferenceRelay:
		field = "evidence_reference"
	default:
		return nil, fmt.Errorf("unsupported evidence reference kind %q", kind)
	}
	record, err := r.app.FindFirstRecordByData("payments", field, reference)
	if err != nil {
		return nil, err
	}
	return paymentFromRecord(record), nil
}

func (r *pocketBasePayments) FindOnTimeCandidates(account domain.PaymentAccount, amount int64, evidenceAt, createdBefore, now time.Time) ([]*domain.Payment, error) {
	records, err := r.app.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && created_at <= {:createdBefore} && ((status = 'pending' && expires_at >= {:evidenceAt} && reuse_after > {:now}) || (status = 'expired' && expires_at >= {:evidenceAt} && reuse_after > {:now}) || (status = 'cancelled' && resolved_at != '' && resolved_at >= {:evidenceAt} && reuse_after > {:now}))",
		"created", 2, 0,
		dbx.Params{"account": string(account), "amount": amount, "now": storeDate(now), "evidenceAt": storeDate(evidenceAt), "createdBefore": storeDate(createdBefore)},
	)
	if err != nil {
		return nil, err
	}
	return paymentsFromRecords(records), nil
}

func (r *pocketBasePayments) FindLateCandidates(account domain.PaymentAccount, amount int64, evidenceAt, createdBefore, now time.Time) ([]*domain.Payment, error) {
	records, err := r.app.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && (status = 'expired' || status = 'cancelled' || (status = 'pending' && expires_at < {:evidenceAt})) && reuse_after > {:now} && created_at <= {:createdBefore}",
		"-created", 2, 0,
		dbx.Params{"account": string(account), "amount": amount, "now": storeDate(now), "evidenceAt": storeDate(evidenceAt), "createdBefore": storeDate(createdBefore)},
	)
	if err != nil {
		return nil, err
	}
	return paymentsFromRecords(records), nil
}

func (r *pocketBasePayments) Save(payment *domain.Payment) error {
	if payment == nil {
		return nil
	}
	record, err := r.app.FindRecordById("payments", payment.ID)
	if err != nil {
		return err
	}
	record.Set("payment_account", string(payment.Account))
	record.Set("requested_amount", payment.RequestedPaise)
	record.Set("payable_amount", payment.PayablePaise)
	record.Set("status", string(payment.Status))
	record.Set("expires_at", payment.ExpiresAt)
	record.Set("reuse_after", payment.ReuseAfter)
	record.Set("rrn", payment.RRN)
	record.Set("upi_id", payment.UPIId)
	record.Set("payer_name", payment.PayerName)
	record.Set("evidence_source", payment.EvidenceSource)
	record.Set("evidence_reference", payment.EvidenceReference)
	record.Set("paid_at", payment.PaidAt)
	record.Set("resolved_at", payment.ResolvedAt)
	record.Set("external_id", payment.ExternalID)
	record.Set("idempotency_key", payment.IdempotencyKey)
	return r.app.Save(record)
}

func (r *pocketBasePayments) ListDue(now time.Time, limit int) ([]*domain.Payment, error) {
	if limit <= 0 {
		return nil, nil
	}
	records, err := r.app.FindRecordsByFilter(
		"payments", "status = 'pending' && expires_at <= {:now}", "expires_at", limit, 0,
		dbx.Params{"now": storeDate(now)},
	)
	if err != nil {
		return nil, err
	}
	return paymentsFromRecords(records), nil
}

func (r *pocketBasePayments) ListAll() ([]*domain.Payment, error) {
	records, err := r.app.FindAllRecords("payments")
	if err != nil {
		return nil, err
	}
	return paymentsFromRecords(records), nil
}

func (r *pocketBasePayments) ListBlocked(now time.Time) ([]*domain.Payment, error) {
	records, err := r.app.FindRecordsByFilter("payments", "reuse_after > {:now}", "requested_amount,payable_amount", 0, 0, dbx.Params{"now": storeDate(now)})
	if err != nil {
		return nil, err
	}
	return paymentsFromRecords(records), nil
}

func paymentsFromRecords(records []*core.Record) []*domain.Payment {
	result := make([]*domain.Payment, 0, len(records))
	for _, record := range records {
		result = append(result, paymentFromRecord(record))
	}
	return result
}

func paymentFromRecord(record *core.Record) *domain.Payment {
	if record == nil {
		return nil
	}
	return &domain.Payment{
		ID: record.Id, Account: domain.PaymentAccount(record.GetString("payment_account")),
		RequestedPaise: int64(record.GetInt("requested_amount")), PayablePaise: int64(record.GetInt("payable_amount")),
		Status: domain.PaymentStatus(record.GetString("status")), CreatedAt: record.GetDateTime("created_at").Time(),
		ExpiresAt: record.GetDateTime("expires_at").Time(), ReuseAfter: record.GetDateTime("reuse_after").Time(),
		RRN: record.GetString("rrn"), UPIId: record.GetString("upi_id"), PayerName: record.GetString("payer_name"),
		EvidenceSource: record.GetString("evidence_source"), EvidenceReference: record.GetString("evidence_reference"),
		PaidAt: record.GetDateTime("paid_at").Time(), ResolvedAt: record.GetDateTime("resolved_at").Time(),
		ExternalID: record.GetString("external_id"), IdempotencyKey: record.GetString("idempotency_key"), Metadata: record.Get("metadata"),
	}
}

func (r *pocketBaseRelay) EnabledDevices(limit int) ([]domain.RelayDeviceHealth, error) {
	if limit <= 0 {
		limit = 100
	}
	records, err := r.app.FindRecordsByFilter("relay_devices", "enabled = true", "-last_seen_at", limit, 0)
	if err != nil {
		return nil, err
	}
	result := make([]domain.RelayDeviceHealth, 0, len(records))
	for _, record := range records {
		result = append(result, domain.RelayDeviceHealth{
			Enabled: record.GetBool("enabled"), AppVersion: record.GetString("app_version"),
			LastSeenAt: record.GetDateTime("last_seen_at").Time(), LastHeartbeatAt: record.GetDateTime("last_heartbeat_at").Time(),
			HeartbeatGraceUntil: record.GetDateTime("heartbeat_grace_until").Time(),
			NotificationAccess:  record.GetBool("notification_access"), ListenerConnected: record.GetBool("listener_connected"),
			PowerHealthReported: record.GetBool("power_health_reported"), BatteryOptimizationExempt: record.GetBool("battery_optimization_exempt"),
			BackgroundRestricted: record.GetBool("background_restricted"), ForegroundServiceActive: record.GetBool("foreground_service_active"),
		})
	}
	return result, nil
}

func (r *pocketBaseOutbox) Enqueue(delivery OutboxDelivery) error {
	collection, err := r.app.FindCollectionByNameOrId("webhook_deliveries")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("event_id", delivery.EventID)
	record.Set("event", delivery.Event)
	record.Set("payment", delivery.PaymentID)
	if delivery.RefundID != "" {
		record.Set("refund", delivery.RefundID)
	}
	record.Set("url", delivery.URL)
	record.Set("body", delivery.Body)
	record.Set("status", "pending")
	record.Set("next_attempt_at", delivery.CreatedAt.UTC())
	return r.app.Save(record)
}

func storeDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}
