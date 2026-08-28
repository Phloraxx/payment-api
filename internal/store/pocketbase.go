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
type pocketBaseSMSEvents struct{ app core.App }
type pocketBaseEmailEvents struct{ app core.App }
type pocketBaseReconciliationRuns struct{ app core.App }
type pocketBaseReconciliationEntries struct{ app core.App }
type pocketBaseAudit struct{ app core.App }
type pocketBaseRefunds struct{ app core.App }
type pocketBaseNotificationEvents struct{ app core.App }
type pocketBaseReviews struct{ app core.App }
type pocketBaseRelay struct{ app core.App }
type pocketBaseRelayEvents struct{ app core.App }
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

func (u *pocketBaseUnit) Payments() PaymentRepository   { return &pocketBasePayments{app: u.app} }
func (u *pocketBaseUnit) SMSEvents() SMSEventRepository { return &pocketBaseSMSEvents{app: u.app} }
func (u *pocketBaseUnit) EmailEvents() EmailEventRepository {
	return &pocketBaseEmailEvents{app: u.app}
}
func (u *pocketBaseUnit) ReconciliationRuns() ReconciliationRunRepository {
	return &pocketBaseReconciliationRuns{app: u.app}
}
func (u *pocketBaseUnit) ReconciliationEntries() ReconciliationEntryRepository {
	return &pocketBaseReconciliationEntries{app: u.app}
}
func (u *pocketBaseUnit) Audit() AuditRepository    { return &pocketBaseAudit{app: u.app} }
func (u *pocketBaseUnit) Refunds() RefundRepository { return &pocketBaseRefunds{app: u.app} }
func (u *pocketBaseUnit) NotificationEvents() NotificationEventRepository {
	return &pocketBaseNotificationEvents{app: u.app}
}
func (u *pocketBaseUnit) Reviews() ReviewRepository { return &pocketBaseReviews{app: u.app} }
func (u *pocketBaseUnit) Relay() RelayRepository    { return &pocketBaseRelay{app: u.app} }
func (u *pocketBaseUnit) RelayEvents() RelayEventRepository {
	return &pocketBaseRelayEvents{app: u.app}
}
func (u *pocketBaseUnit) Outbox() OutboxRepository { return &pocketBaseOutbox{app: u.app} }

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

func (r *pocketBasePayments) FindReconciliationCandidates(account domain.PaymentAccount, amount int64, transactionTime, now time.Time, limit int) ([]*domain.Payment, error) {
	if amount <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var records []*core.Record
	var err error
	if !transactionTime.IsZero() {
		records, err = r.app.FindRecordsByFilter("payments", "payment_account = {:account} && payable_amount = {:amount} && created_at <= {:createdBefore} && reuse_after >= {:at}", "-created_at", limit, 0, dbx.Params{"account": string(account), "amount": amount, "at": storeDate(transactionTime), "createdBefore": storeDate(transactionTime.Add(2 * time.Second))})
	} else {
		records, err = r.app.FindRecordsByFilter("payments", "payment_account = {:account} && payable_amount = {:amount} && reuse_after > {:now}", "-created_at", limit, 0, dbx.Params{"account": string(account), "amount": amount, "now": storeDate(now)})
	}
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

func (r *pocketBasePayments) ListFingerprintCandidates(account domain.PaymentAccount, amount int64, now time.Time, limit int) ([]*domain.Payment, error) {
	if amount <= 0 || limit <= 0 {
		return nil, nil
	}
	records, err := r.app.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && reuse_after > {:now}",
		"-created_at", limit, 0,
		dbx.Params{"account": string(account), "amount": amount, "now": storeDate(now)},
	)
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

func (r *pocketBaseSMSEvents) Get(id string) (*domain.SMSEvent, error) {
	record, err := r.app.FindRecordById("sms_events", id)
	if err != nil {
		return nil, err
	}
	return smsEventFromRecord(record), nil
}

func (r *pocketBaseSMSEvents) FindBySourceEvent(source, sourceEventID string) (*domain.SMSEvent, error) {
	record, err := r.app.FindFirstRecordByFilter("sms_events", "source = {:source} && source_event_id = {:id}", dbx.Params{"source": source, "id": sourceEventID})
	if err != nil {
		return nil, err
	}
	return smsEventFromRecord(record), nil
}

func (r *pocketBaseSMSEvents) Create(event *domain.SMSEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("sms_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeSMSEvent(record, event)
	if err := r.app.Save(record); err != nil {
		return err
	}
	event.ID = record.Id
	return nil
}

func (r *pocketBaseSMSEvents) Save(event *domain.SMSEvent) error {
	record, err := r.app.FindRecordById("sms_events", event.ID)
	if err != nil {
		return err
	}
	writeSMSEvent(record, event)
	return r.app.Save(record)
}

func (r *pocketBaseSMSEvents) ListBySourceSince(source string, since time.Time, limit int) ([]*domain.SMSEvent, error) {
	if limit <= 0 {
		limit = 5000
	}
	records, err := r.app.FindRecordsByFilter("sms_events", "source = {:source} && message_time >= {:since}", "message_time", limit, 0, dbx.Params{"source": source, "since": storeDate(since)})
	if err != nil {
		return nil, err
	}
	items := make([]*domain.SMSEvent, 0, len(records))
	for _, record := range records {
		items = append(items, smsEventFromRecord(record))
	}
	return items, nil
}

func writeSMSEvent(record *core.Record, event *domain.SMSEvent) {
	record.Set("source", event.Source)
	record.Set("source_event_id", event.SourceEventID)
	record.Set("sender", event.Sender)
	record.Set("body", event.Body)
	record.Set("payment_account", string(event.Account))
	record.Set("message_time", event.MessageTime)
	record.Set("amount", event.AmountPaise)
	record.Set("rrn", event.RRN)
	record.Set("upi_id", event.UPIID)
	record.Set("payer_name", event.PayerName)
	record.Set("processing_status", event.ProcessingStatus)
	record.Set("matched_payment", event.MatchedPaymentID)
	record.Set("error", event.Error)
	if event.RawPayload != nil {
		record.Set("raw_payload", event.RawPayload)
	}
}

func smsEventFromRecord(record *core.Record) *domain.SMSEvent {
	if record == nil {
		return nil
	}
	return &domain.SMSEvent{ID: record.Id, Source: record.GetString("source"), SourceEventID: record.GetString("source_event_id"), Sender: record.GetString("sender"), Body: record.GetString("body"), Account: domain.PaymentAccount(record.GetString("payment_account")), MessageTime: record.GetDateTime("message_time").Time(), AmountPaise: int64(record.GetInt("amount")), RRN: record.GetString("rrn"), UPIID: record.GetString("upi_id"), PayerName: record.GetString("payer_name"), ProcessingStatus: record.GetString("processing_status"), MatchedPaymentID: record.GetString("matched_payment"), Error: record.GetString("error"), RawPayload: record.Get("raw_payload")}
}

func (r *pocketBaseEmailEvents) Get(id string) (*domain.EmailEvent, error) {
	record, err := r.app.FindRecordById("email_events", id)
	if err != nil {
		return nil, err
	}
	return emailEventFromRecord(record), nil
}

func (r *pocketBaseEmailEvents) FindBySourceEvent(source, sourceEventID string) (*domain.EmailEvent, error) {
	record, err := r.app.FindFirstRecordByFilter("email_events", "source = {:source} && source_event_id = {:id}", dbx.Params{"source": source, "id": sourceEventID})
	if err != nil {
		return nil, err
	}
	return emailEventFromRecord(record), nil
}

func (r *pocketBaseEmailEvents) Create(event *domain.EmailEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("email_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeEmailEvent(record, event)
	if err := r.app.Save(record); err != nil {
		return err
	}
	event.ID = record.Id
	return nil
}

func (r *pocketBaseEmailEvents) Save(event *domain.EmailEvent) error {
	record, err := r.app.FindRecordById("email_events", event.ID)
	if err != nil {
		return err
	}
	writeEmailEvent(record, event)
	return r.app.Save(record)
}

func writeEmailEvent(record *core.Record, event *domain.EmailEvent) {
	record.Set("source", event.Source)
	record.Set("source_event_id", event.SourceEventID)
	record.Set("envelope_sender", event.EnvelopeSender)
	record.Set("recipient", event.Recipient)
	record.Set("sender", event.Sender)
	record.Set("subject", event.Subject)
	record.Set("body", event.Body)
	record.Set("payment_account", string(event.Account))
	record.Set("message_time", event.MessageTime)
	record.Set("received_at", event.ReceivedAt)
	record.Set("auth_result", event.AuthResult)
	record.Set("amount", event.AmountPaise)
	record.Set("rrn", event.RRN)
	record.Set("upi_id", event.UPIID)
	record.Set("payer_name", event.PayerName)
	record.Set("processing_status", event.ProcessingStatus)
	record.Set("matched_payment", event.MatchedPaymentID)
	record.Set("error", event.Error)
	if event.RawPayload != nil {
		record.Set("raw_payload", event.RawPayload)
	}
}

func emailEventFromRecord(record *core.Record) *domain.EmailEvent {
	if record == nil {
		return nil
	}
	return &domain.EmailEvent{ID: record.Id, Source: record.GetString("source"), SourceEventID: record.GetString("source_event_id"), EnvelopeSender: record.GetString("envelope_sender"), Recipient: record.GetString("recipient"), Sender: record.GetString("sender"), Subject: record.GetString("subject"), Body: record.GetString("body"), Account: domain.PaymentAccount(record.GetString("payment_account")), MessageTime: record.GetDateTime("message_time").Time(), ReceivedAt: record.GetDateTime("received_at").Time(), AuthResult: record.GetString("auth_result"), AmountPaise: int64(record.GetInt("amount")), RRN: record.GetString("rrn"), UPIID: record.GetString("upi_id"), PayerName: record.GetString("payer_name"), ProcessingStatus: record.GetString("processing_status"), MatchedPaymentID: record.GetString("matched_payment"), Error: record.GetString("error"), RawPayload: record.Get("raw_payload")}
}

func (r *pocketBaseReconciliationRuns) FindCompletedByHash(hash string) (*domain.ReconciliationRun, error) {
	record, err := r.app.FindFirstRecordByFilter("reconciliation_runs", "sha256 = {:hash} && status = 'completed'", dbx.Params{"hash": hash})
	if err != nil {
		return nil, err
	}
	return reconciliationRunFromRecord(record), nil
}

func (r *pocketBaseReconciliationRuns) Get(id string) (*domain.ReconciliationRun, error) {
	record, err := r.app.FindRecordById("reconciliation_runs", id)
	if err != nil {
		return nil, err
	}
	return reconciliationRunFromRecord(record), nil
}

func (r *pocketBaseReconciliationRuns) Create(run *domain.ReconciliationRun) error {
	collection, err := r.app.FindCollectionByNameOrId("reconciliation_runs")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	applyReconciliationRunRecord(record, run)
	if err := r.app.Save(record); err != nil {
		return err
	}
	run.ID = record.Id
	return nil
}

func (r *pocketBaseReconciliationRuns) Save(run *domain.ReconciliationRun) error {
	record, err := r.app.FindRecordById("reconciliation_runs", run.ID)
	if err != nil {
		return err
	}
	applyReconciliationRunRecord(record, run)
	return r.app.Save(record)
}

func applyReconciliationRunRecord(record *core.Record, run *domain.ReconciliationRun) {
	record.Set("filename", run.Filename)
	record.Set("sha256", run.SHA256)
	record.Set("status", run.Status)
	record.Set("created_by", run.CreatedBy)
	record.Set("started_at", run.StartedAt)
	record.Set("completed_at", run.CompletedAt)
	record.Set("total_rows", run.TotalRows)
	record.Set("matched_rows", run.MatchedRows)
	record.Set("unmatched_rows", run.UnmatchedRows)
	record.Set("duplicate_rows", run.DuplicateRows)
	record.Set("conflict_rows", run.ConflictRows)
	record.Set("invalid_rows", run.InvalidRows)
	record.Set("error", run.Error)
	if run.Summary != nil {
		record.Set("summary", run.Summary)
	}
}

func reconciliationRunFromRecord(record *core.Record) *domain.ReconciliationRun {
	if record == nil {
		return nil
	}
	return &domain.ReconciliationRun{ID: record.Id, Filename: record.GetString("filename"), SHA256: record.GetString("sha256"), Status: record.GetString("status"), CreatedBy: record.GetString("created_by"), StartedAt: record.GetDateTime("started_at").Time(), CompletedAt: record.GetDateTime("completed_at").Time(), TotalRows: record.GetInt("total_rows"), MatchedRows: record.GetInt("matched_rows"), UnmatchedRows: record.GetInt("unmatched_rows"), DuplicateRows: record.GetInt("duplicate_rows"), ConflictRows: record.GetInt("conflict_rows"), InvalidRows: record.GetInt("invalid_rows"), Error: record.GetString("error"), Summary: record.Get("summary")}
}

func (r *pocketBaseReconciliationEntries) Create(entry *domain.ReconciliationEntry) error {
	collection, err := r.app.FindCollectionByNameOrId("reconciliation_entries")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("run", entry.RunID)
	record.Set("row_number", entry.RowNumber)
	record.Set("transaction_time", entry.TransactionTime)
	record.Set("amount", entry.AmountPaise)
	record.Set("rrn", entry.RRN)
	record.Set("description", entry.Description)
	record.Set("status", entry.Status)
	record.Set("payment", entry.PaymentID)
	record.Set("notes", entry.Notes)
	if entry.RawRow != nil {
		record.Set("raw_row", entry.RawRow)
	}
	if err := r.app.Save(record); err != nil {
		return err
	}
	entry.ID = record.Id
	return nil
}

func (r *pocketBaseReconciliationEntries) Get(id string) (*domain.ReconciliationEntry, error) {
	record, err := r.app.FindRecordById("reconciliation_entries", id)
	if err != nil {
		return nil, err
	}
	return reconciliationEntryFromRecord(record), nil
}

func (r *pocketBaseReconciliationEntries) Save(entry *domain.ReconciliationEntry) error {
	record, err := r.app.FindRecordById("reconciliation_entries", entry.ID)
	if err != nil {
		return err
	}
	record.Set("rrn", entry.RRN)
	record.Set("status", entry.Status)
	record.Set("payment", entry.PaymentID)
	record.Set("notes", entry.Notes)
	return r.app.Save(record)
}

func reconciliationEntryFromRecord(record *core.Record) *domain.ReconciliationEntry {
	if record == nil {
		return nil
	}
	return &domain.ReconciliationEntry{ID: record.Id, RunID: record.GetString("run"), RowNumber: record.GetInt("row_number"), TransactionTime: record.GetDateTime("transaction_time").Time(), AmountPaise: int64(record.GetInt("amount")), RRN: record.GetString("rrn"), Description: record.GetString("description"), Status: record.GetString("status"), PaymentID: record.GetString("payment"), Notes: record.GetString("notes"), RawRow: record.Get("raw_row")}
}

func (r *pocketBaseRefunds) Get(id string) (*domain.Refund, error) {
	record, err := r.app.FindRecordById("refunds", id)
	if err != nil {
		return nil, err
	}
	return refundFromRecord(record), nil
}

func (r *pocketBaseRefunds) FindByIdempotencyKey(key string) (*domain.Refund, error) {
	record, err := r.app.FindFirstRecordByData("refunds", "idempotency_key", key)
	if err != nil {
		return nil, err
	}
	return refundFromRecord(record), nil
}

func (r *pocketBaseRefunds) FindByReference(reference string) (*domain.Refund, error) {
	record, err := r.app.FindFirstRecordByData("refunds", "reference", reference)
	if err != nil {
		return nil, err
	}
	return refundFromRecord(record), nil
}

func (r *pocketBaseRefunds) ReservedAmount(paymentID string) (int64, error) {
	records, err := r.app.FindRecordsByFilter("refunds", "payment = {:payment} && status != 'cancelled' && status != 'failed'", "created", 0, 0, dbx.Params{"payment": paymentID})
	if err != nil {
		return 0, err
	}
	var total int64
	for _, record := range records {
		total += int64(record.GetInt("amount"))
	}
	return total, nil
}

func (r *pocketBaseRefunds) Create(refund *domain.Refund) error {
	collection, err := r.app.FindCollectionByNameOrId("refunds")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	applyRefundRecord(record, refund)
	if err := r.app.Save(record); err != nil {
		return err
	}
	refund.ID = record.Id
	return nil
}

func (r *pocketBaseRefunds) Save(refund *domain.Refund) error {
	record, err := r.app.FindRecordById("refunds", refund.ID)
	if err != nil {
		return err
	}
	applyRefundRecord(record, refund)
	return r.app.Save(record)
}

func applyRefundRecord(record *core.Record, refund *domain.Refund) {
	record.Set("payment", refund.PaymentID)
	record.Set("amount", refund.AmountPaise)
	record.Set("status", refund.Status)
	record.Set("reason", refund.Reason)
	record.Set("reference", refund.Reference)
	record.Set("external_id", refund.ExternalID)
	record.Set("idempotency_key", refund.IdempotencyKey)
	if refund.Metadata != nil {
		record.Set("metadata", refund.Metadata)
	}
	record.Set("requested_by", refund.RequestedBy)
	record.Set("requested_at", refund.RequestedAt)
	record.Set("completed_at", refund.CompletedAt)
}

func refundFromRecord(record *core.Record) *domain.Refund {
	if record == nil {
		return nil
	}
	return &domain.Refund{ID: record.Id, PaymentID: record.GetString("payment"), AmountPaise: int64(record.GetInt("amount")), Status: record.GetString("status"), Reason: record.GetString("reason"), Reference: record.GetString("reference"), ExternalID: record.GetString("external_id"), IdempotencyKey: record.GetString("idempotency_key"), Metadata: record.Get("metadata"), RequestedBy: record.GetString("requested_by"), RequestedAt: record.GetDateTime("requested_at").Time(), CompletedAt: record.GetDateTime("completed_at").Time()}
}

func (r *pocketBaseAudit) Record(event domain.AuditEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("audit_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("action", event.Action)
	record.Set("actor_id", event.ActorID)
	record.Set("actor_email", event.ActorEmail)
	record.Set("entity_type", event.EntityType)
	record.Set("entity_id", event.EntityID)
	record.Set("summary", event.Summary)
	record.Set("occurred_at", event.OccurredAt)
	if event.Details != nil {
		record.Set("details", event.Details)
	}
	return r.app.Save(record)
}

func (r *pocketBaseNotificationEvents) FindBySourceEvent(source, sourceEventID string) (*domain.NotificationEvent, error) {
	record, err := r.app.FindFirstRecordByFilter("notification_events", "source = {:source} && source_event_id = {:id}", dbx.Params{"source": source, "id": sourceEventID})
	if err != nil {
		return nil, err
	}
	return notificationEventFromRecord(record), nil
}

func (r *pocketBaseNotificationEvents) Get(id string) (*domain.NotificationEvent, error) {
	record, err := r.app.FindRecordById("notification_events", id)
	if err != nil {
		return nil, err
	}
	return notificationEventFromRecord(record), nil
}

func (r *pocketBaseNotificationEvents) Create(event *domain.NotificationEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("notification_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeNotificationEvent(record, event)
	if err := r.app.Save(record); err != nil {
		return err
	}
	event.ID = record.Id
	return nil
}

func (r *pocketBaseNotificationEvents) Save(event *domain.NotificationEvent) error {
	record, err := r.app.FindRecordById("notification_events", event.ID)
	if err != nil {
		return err
	}
	writeNotificationEvent(record, event)
	return r.app.Save(record)
}

func writeNotificationEvent(record *core.Record, event *domain.NotificationEvent) {
	record.Set("source", event.Source)
	record.Set("source_event_id", event.SourceEventID)
	record.Set("app_package", event.AppPackage)
	record.Set("app_name", event.AppName)
	record.Set("title", event.Title)
	record.Set("body", event.Body)
	record.Set("big_text", event.BigText)
	record.Set("channel", event.Channel)
	record.Set("notification_time", event.NotificationTime)
	record.Set("payment_account", string(event.Account))
	record.Set("amount", event.AmountPaise)
	record.Set("payer_name", event.PayerName)
	record.Set("processing_status", event.ProcessingStatus)
	record.Set("matched_payment", event.MatchedPaymentID)
	record.Set("error", event.Error)
	if event.RawPayload != nil {
		record.Set("raw_payload", event.RawPayload)
	}
}

func notificationEventFromRecord(record *core.Record) *domain.NotificationEvent {
	if record == nil {
		return nil
	}
	return &domain.NotificationEvent{ID: record.Id, Source: record.GetString("source"), SourceEventID: record.GetString("source_event_id"), AppPackage: record.GetString("app_package"), AppName: record.GetString("app_name"), Title: record.GetString("title"), Body: record.GetString("body"), BigText: record.GetString("big_text"), Channel: record.GetString("channel"), NotificationTime: record.GetDateTime("notification_time").Time(), Account: domain.PaymentAccount(record.GetString("payment_account")), AmountPaise: int64(record.GetInt("amount")), PayerName: record.GetString("payer_name"), ProcessingStatus: record.GetString("processing_status"), MatchedPaymentID: record.GetString("matched_payment"), Error: record.GetString("error"), RawPayload: record.Get("raw_payload")}
}

func (r *pocketBaseReviews) FindByEvidence(smsEventID, emailEventID, reconciliationEntryID string) (*domain.ReviewCase, error) {
	field, value := "", ""
	switch {
	case emailEventID != "":
		field, value = "email_event", emailEventID
	case smsEventID != "":
		field, value = "sms_event", smsEventID
	case reconciliationEntryID != "":
		field, value = "reconciliation_entry", reconciliationEntryID
	default:
		return nil, sql.ErrNoRows
	}
	record, err := r.app.FindFirstRecordByData("review_cases", field, value)
	if err != nil {
		return nil, err
	}
	return reviewFromRecord(record), nil
}

func (r *pocketBaseReviews) Create(review *domain.ReviewCase) error {
	collection, err := r.app.FindCollectionByNameOrId("review_cases")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeReview(record, review)
	if err := r.app.Save(record); err != nil {
		return err
	}
	review.ID = record.Id
	return nil
}

func (r *pocketBaseReviews) Get(id string) (*domain.ReviewCase, error) {
	record, err := r.app.FindRecordById("review_cases", id)
	if err != nil {
		return nil, err
	}
	return reviewFromRecord(record), nil
}

func (r *pocketBaseReviews) Save(review *domain.ReviewCase) error {
	record, err := r.app.FindRecordById("review_cases", review.ID)
	if err != nil {
		return err
	}
	writeReview(record, review)
	return r.app.Save(record)
}

func (r *pocketBaseReviews) OpenCount() (int64, error) {
	return r.app.CountRecords("review_cases", dbx.NewExp("status = 'open'"))
}

func writeReview(record *core.Record, review *domain.ReviewCase) {
	record.Set("kind", review.Kind)
	record.Set("status", review.Status)
	record.Set("severity", review.Severity)
	record.Set("sms_event", review.SMSEventID)
	record.Set("email_event", review.EmailEventID)
	record.Set("reconciliation_entry", review.ReconciliationEntryID)
	record.Set("payment", review.PaymentID)
	record.Set("candidate_payment_ids", review.CandidatePaymentIDs)
	record.Set("reason", review.Reason)
	record.Set("resolution", review.Resolution)
	record.Set("resolution_note", review.ResolutionNote)
	record.Set("resolved_by", review.ResolvedBy)
	record.Set("opened_at", review.OpenedAt)
	record.Set("resolved_at", review.ResolvedAt)
}

func reviewFromRecord(record *core.Record) *domain.ReviewCase {
	if record == nil {
		return nil
	}
	return &domain.ReviewCase{ID: record.Id, Kind: record.GetString("kind"), Status: record.GetString("status"), Severity: record.GetString("severity"), SMSEventID: record.GetString("sms_event"), EmailEventID: record.GetString("email_event"), ReconciliationEntryID: record.GetString("reconciliation_entry"), PaymentID: record.GetString("payment"), CandidatePaymentIDs: stringSlice(record.Get("candidate_payment_ids")), Reason: record.GetString("reason"), Resolution: record.GetString("resolution"), ResolutionNote: record.GetString("resolution_note"), ResolvedBy: record.GetString("resolved_by"), OpenedAt: record.GetDateTime("opened_at").Time(), ResolvedAt: record.GetDateTime("resolved_at").Time()}
}

func stringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		result := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
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

func (r *pocketBaseRelay) Get(id string) (*domain.RelayDevice, error) {
	record, err := r.app.FindRecordById("relay_devices", id)
	if err != nil {
		return nil, err
	}
	return relayDeviceFromRecord(record), nil
}

func (r *pocketBaseRelay) FindByDeviceID(deviceID string) (*domain.RelayDevice, error) {
	record, err := r.app.FindFirstRecordByFilter("relay_devices", "device_id = {:id}", dbx.Params{"id": deviceID})
	if err != nil {
		return nil, err
	}
	return relayDeviceFromRecord(record), nil
}

func (r *pocketBaseRelay) Create(device *domain.RelayDevice) error {
	collection, err := r.app.FindCollectionByNameOrId("relay_devices")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeRelayDevice(record, device)
	if err := r.app.Save(record); err != nil {
		return err
	}
	device.ID = record.Id
	device.CreatedAt = record.GetDateTime("created").Time()
	return nil
}

func (r *pocketBaseRelay) Save(device *domain.RelayDevice) error {
	record, err := r.app.FindRecordById("relay_devices", device.ID)
	if err != nil {
		return err
	}
	writeRelayDevice(record, device)
	return r.app.Save(record)
}

func (r *pocketBaseRelay) All(limit int) ([]*domain.RelayDevice, error) {
	if limit <= 0 {
		limit = 100
	}
	records, err := r.app.FindRecordsByFilter("relay_devices", "", "-last_seen_at,-created", limit, 0)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.RelayDevice, 0, len(records))
	for _, record := range records {
		result = append(result, relayDeviceFromRecord(record))
	}
	return result, nil
}

func writeRelayDevice(record *core.Record, device *domain.RelayDevice) {
	record.Set("device_id", device.DeviceID)
	record.Set("name", device.Name)
	record.Set("public_key_pem", device.PublicKeyPEM)
	record.Set("enabled", device.Enabled)
	record.Set("app_version", device.AppVersion)
	record.Set("android_version", device.AndroidVersion)
	record.Set("device_model", device.DeviceModel)
	record.Set("enrolled_at", device.EnrolledAt)
	record.Set("last_seen_at", device.LastSeenAt)
	record.Set("last_heartbeat_at", device.LastHeartbeatAt)
	record.Set("heartbeat_grace_until", device.HeartbeatGraceUntil)
	record.Set("notification_access", device.NotificationAccess)
	record.Set("listener_connected", device.ListenerConnected)
	record.Set("power_health_reported", device.PowerHealthReported)
	record.Set("battery_optimization_exempt", device.BatteryOptimizationExempt)
	record.Set("power_save_mode", device.PowerSaveMode)
	record.Set("background_restricted", device.BackgroundRestricted)
	record.Set("foreground_service_active", device.ForegroundServiceActive)
	record.Set("pending_count", device.PendingCount)
	record.Set("failed_count", device.FailedCount)
	record.Set("last_client_error", device.LastClientError)
	record.Set("last_client_delivery_at", device.LastClientDeliveryAt)
}

func relayDeviceFromRecord(record *core.Record) *domain.RelayDevice {
	if record == nil {
		return nil
	}
	return &domain.RelayDevice{
		ID: record.Id, DeviceID: record.GetString("device_id"), Name: record.GetString("name"), PublicKeyPEM: record.GetString("public_key_pem"), Enabled: record.GetBool("enabled"),
		AppVersion: record.GetString("app_version"), AndroidVersion: record.GetString("android_version"), DeviceModel: record.GetString("device_model"),
		EnrolledAt: record.GetDateTime("enrolled_at").Time(), LastSeenAt: record.GetDateTime("last_seen_at").Time(), LastHeartbeatAt: record.GetDateTime("last_heartbeat_at").Time(), HeartbeatGraceUntil: record.GetDateTime("heartbeat_grace_until").Time(),
		NotificationAccess: record.GetBool("notification_access"), ListenerConnected: record.GetBool("listener_connected"), PowerHealthReported: record.GetBool("power_health_reported"), BatteryOptimizationExempt: record.GetBool("battery_optimization_exempt"), PowerSaveMode: record.GetBool("power_save_mode"), BackgroundRestricted: record.GetBool("background_restricted"), ForegroundServiceActive: record.GetBool("foreground_service_active"),
		PendingCount: record.GetInt("pending_count"), FailedCount: record.GetInt("failed_count"), LastClientError: record.GetString("last_client_error"), LastClientDeliveryAt: record.GetDateTime("last_client_delivery_at").Time(), CreatedAt: record.GetDateTime("created").Time(),
	}
}

func (r *pocketBaseRelayEvents) FindByDeviceEvent(deviceRecordID, eventID string) (*domain.RelayEvent, error) {
	record, err := r.app.FindFirstRecordByFilter("relay_events", "device = {:device} && event_id = {:event}", dbx.Params{"device": deviceRecordID, "event": eventID})
	if err != nil {
		return nil, err
	}
	return relayEventFromRecord(record), nil
}

func (r *pocketBaseRelayEvents) Create(event *domain.RelayEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("relay_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	writeRelayEvent(record, event)
	if err := r.app.Save(record); err != nil {
		return err
	}
	event.ID = record.Id
	event.CreatedAt = record.GetDateTime("created").Time()
	return nil
}

func (r *pocketBaseRelayEvents) Save(event *domain.RelayEvent) error {
	record, err := r.app.FindRecordById("relay_events", event.ID)
	if err != nil {
		return err
	}
	writeRelayEvent(record, event)
	return r.app.Save(record)
}

func (r *pocketBaseRelayEvents) Latest(deviceRecordID string) (*domain.RelayEvent, error) {
	filter, params := "", dbx.Params{}
	if deviceRecordID != "" {
		filter, params = "device = {:device}", dbx.Params{"device": deviceRecordID}
	}
	records, err := r.app.FindRecordsByFilter("relay_events", filter, "-created", 1, 0, params)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return relayEventFromRecord(records[0]), nil
}

func (r *pocketBaseRelayEvents) LatestMatched(deviceRecordID string) (*domain.RelayEvent, error) {
	filter := "matched_payment != ''"
	params := dbx.Params{}
	if deviceRecordID != "" {
		filter = "device = {:device} && matched_payment != ''"
		params = dbx.Params{"device": deviceRecordID}
	}
	records, err := r.app.FindRecordsByFilter("relay_events", filter, "-created", 1, 0, params)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return relayEventFromRecord(records[0]), nil
}

func (r *pocketBaseRelayEvents) CountErrorsSince(deviceRecordID string, since time.Time) (int64, error) {
	if deviceRecordID == "" {
		return r.app.CountRecords("relay_events", dbx.NewExp("processing_status = 'error' AND created >= {:cutoff}", dbx.Params{"cutoff": storeDate(since)}))
	}
	return r.app.CountRecords("relay_events", dbx.NewExp("device = {:device} AND processing_status = 'error' AND created >= {:cutoff}", dbx.Params{"device": deviceRecordID, "cutoff": storeDate(since)}))
}

func (r *pocketBaseRelayEvents) ListByPackageSince(appPackage string, since time.Time, limit int) ([]*domain.RelayEvent, error) {
	if limit <= 0 {
		limit = 5000
	}
	records, err := r.app.FindRecordsByFilter("relay_events", "app_package = {:package} && created >= {:since}", "created", limit, 0, dbx.Params{"package": appPackage, "since": storeDate(since)})
	if err != nil {
		return nil, err
	}
	items := make([]*domain.RelayEvent, 0, len(records))
	for _, record := range records {
		items = append(items, relayEventFromRecord(record))
	}
	return items, nil
}

func writeRelayEvent(record *core.Record, event *domain.RelayEvent) {
	record.Set("device", event.DeviceRecordID)
	record.Set("event_id", event.EventID)
	record.Set("kind", event.Kind)
	record.Set("app_package", event.AppPackage)
	record.Set("app_name", event.AppName)
	record.Set("notification_key", event.NotificationKey)
	record.Set("notification_id", event.NotificationID)
	record.Set("notification_tag", event.NotificationTag)
	record.Set("group_key", event.GroupKey)
	record.Set("is_group_summary", event.IsGroupSummary)
	record.Set("post_time", event.PostTime)
	record.Set("notification_when", event.NotificationWhen)
	record.Set("captured_at", event.CapturedAt)
	record.Set("channel_id", event.ChannelID)
	record.Set("category", event.Category)
	record.Set("title", event.Title)
	record.Set("body", event.Body)
	record.Set("big_text", event.BigText)
	record.Set("sub_text", event.SubText)
	record.Set("summary_text", event.SummaryText)
	record.Set("text_lines", event.TextLines)
	record.Set("custom_texts", event.CustomTexts)
	record.Set("processing_status", event.ProcessingStatus)
	record.Set("downstream_event_id", event.DownstreamEventID)
	record.Set("matched_payment", event.MatchedPaymentID)
	record.Set("provider_result", event.ProviderResult)
	record.Set("error", event.Error)
	if event.RawPayload != nil {
		record.Set("raw_payload", event.RawPayload)
	}
}

func relayEventFromRecord(record *core.Record) *domain.RelayEvent {
	if record == nil {
		return nil
	}
	return &domain.RelayEvent{ID: record.Id, DeviceRecordID: record.GetString("device"), EventID: record.GetString("event_id"), Kind: record.GetString("kind"), AppPackage: record.GetString("app_package"), AppName: record.GetString("app_name"), NotificationKey: record.GetString("notification_key"), NotificationID: record.GetInt("notification_id"), NotificationTag: record.GetString("notification_tag"), GroupKey: record.GetString("group_key"), IsGroupSummary: record.GetBool("is_group_summary"), PostTime: record.GetDateTime("post_time").Time(), NotificationWhen: record.GetDateTime("notification_when").Time(), CapturedAt: record.GetDateTime("captured_at").Time(), ChannelID: record.GetString("channel_id"), Category: record.GetString("category"), Title: record.GetString("title"), Body: record.GetString("body"), BigText: record.GetString("big_text"), SubText: record.GetString("sub_text"), SummaryText: record.GetString("summary_text"), TextLines: stringSlice(record.Get("text_lines")), CustomTexts: stringSlice(record.Get("custom_texts")), ProcessingStatus: record.GetString("processing_status"), DownstreamEventID: record.GetString("downstream_event_id"), MatchedPaymentID: record.GetString("matched_payment"), ProviderResult: record.Get("provider_result"), Error: record.GetString("error"), RawPayload: record.Get("raw_payload"), CreatedAt: record.GetDateTime("created").Time()}
}
