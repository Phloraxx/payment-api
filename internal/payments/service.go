package payments

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const EvidenceTimestampTolerance = 2 * time.Second

type WebhookScheduler interface {
	Schedule(app core.App, event string, payment *core.Record, at time.Time) error
	Wake()
}

type Service struct {
	App      core.App
	Config   config.Config
	Webhooks WebhookScheduler
	Now      func() time.Time

	// SuffixStart is injectable for deterministic tests. Production uses crypto/rand.
	SuffixStart func() (int64, error)
}

type CreateInput struct {
	AmountRupees   int64
	PaymentAccount string
	ExternalID     string
	Metadata       any
	IdempotencyKey string
}

type MatchResult struct {
	Payment *domain.Payment
	Action  string
}

func NewService(app core.App, cfg config.Config, webhooks WebhookScheduler) *Service {
	return &Service{
		App:         app,
		Config:      cfg,
		Webhooks:    webhooks,
		Now:         time.Now,
		SuffixStart: randomSuffixStart,
	}
}

func (s *Service) Create(input CreateInput) (*domain.Payment, bool, error) {
	requested, err := money.RupeesToPaise(input.AmountRupees)
	if err != nil {
		return nil, false, domain.InvalidAmount()
	}
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	account, _, err := s.paymentAccount(input.PaymentAccount)
	if err != nil {
		return nil, false, err
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.ExternalID) > 255 {
		return nil, false, domain.InvalidExternalID()
	}
	if len(input.IdempotencyKey) > 255 {
		return nil, false, domain.InvalidIdempotencyKey()
	}
	metadata, err := validateAndNormalizeMetadata(input.Metadata)
	if err != nil {
		return nil, false, err
	}
	now := s.now()
	var result *core.Record
	var reused bool
	var queued bool

	err = s.App.RunInTransaction(func(tx core.App) error {
		expired, err := s.ExpireDueInApp(tx, now)
		if err != nil {
			return err
		}
		queued = expired > 0

		if input.IdempotencyKey != "" {
			existing, findErr := tx.FindFirstRecordByData("payments", "idempotency_key", input.IdempotencyKey)
			if findErr == nil {
				if int64(existing.GetInt("requested_amount")) != requested ||
					existing.GetString("payment_account") != account ||
					existing.GetString("external_id") != input.ExternalID ||
					!metadataEqual(existing.Get("metadata"), metadata) {
					return domain.IdempotencyConflict()
				}
				result = existing.Clone()
				reused = true
				return nil
			}
			if !errors.Is(findErr, sql.ErrNoRows) {
				return findErr
			}
		}

		start, err := s.SuffixStart()
		if err != nil {
			return fmt.Errorf("choose amount fingerprint: %w", err)
		}
		if start < 1 || start > 99 {
			return fmt.Errorf("invalid amount fingerprint start %d", start)
		}
		collection, err := tx.FindCollectionByNameOrId("payments")
		if err != nil {
			return err
		}
		expiresAt := now.Add(s.Config.PaymentTTL)
		reuseAfter := expiresAt.Add(s.Config.AmountQuarantine)

		for i := int64(0); i < 99; i++ {
			suffix := ((start - 1 + i) % 99) + 1
			candidate := requested + suffix
			blocked, err := tx.FindFirstRecordByFilter(
				"payments",
				"payable_amount = {:amount} && reuse_after > {:now}",
				dbx.Params{"amount": candidate, "now": filterDate(now)},
			)
			if err == nil && blocked != nil {
				continue
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			record := core.NewRecord(collection)
			record.Set("created_at", now)
			record.Set("payment_account", account)
			record.Set("requested_amount", requested)
			record.Set("payable_amount", candidate)
			record.Set("status", string(domain.StatusPending))
			record.Set("expires_at", expiresAt)
			record.Set("reuse_after", reuseAfter)
			record.Set("external_id", input.ExternalID)
			record.Set("idempotency_key", input.IdempotencyKey)
			if metadata != nil {
				record.Set("metadata", metadata)
			}
			if err := tx.Save(record); err != nil {
				return err
			}
			result = record.Clone()
			return nil
		}
		return domain.CapacityExhausted()
	})
	if err != nil {
		return nil, false, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return FromRecord(result), reused, nil
}

func (s *Service) Get(id string) (*domain.Payment, error) {
	now := s.now()
	var result *core.Record
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		expired, err := s.ExpireDueInApp(tx, now)
		if err != nil {
			return err
		}
		queued = expired > 0
		record, err := tx.FindRecordById("payments", id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.PaymentNotFound()
			}
			return err
		}
		result = record.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return FromRecord(result), nil
}

func (s *Service) Cancel(id string) (*domain.Payment, error) {
	now := s.now()
	var result *core.Record
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		expired, err := s.ExpireDueInApp(tx, now)
		if err != nil {
			return err
		}
		queued = expired > 0
		record, err := tx.FindRecordById("payments", id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.PaymentNotFound()
			}
			return err
		}
		status := record.GetString("status")
		if status == string(domain.StatusCancelled) {
			result = record.Clone()
			return nil
		}
		if status != string(domain.StatusPending) {
			return domain.PaymentResolved(status)
		}
		record.Set("status", string(domain.StatusCancelled))
		record.Set("resolved_at", now)
		extendReuseAfter(record, now.Add(s.Config.AmountQuarantine))
		if err := tx.Save(record); err != nil {
			return err
		}
		if err := s.schedule(tx, "payment.cancelled", record, now); err != nil {
			return err
		}
		queued = true
		result = record.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return FromRecord(result), nil
}

func (s *Service) Match(parsed domain.ParsedSMS) (*MatchResult, error) {
	if parsed.AmountPaise <= 0 || strings.TrimSpace(parsed.RRN) == "" {
		return nil, domain.New("SMS_NOT_MATCHABLE", "bank SMS requires an exact amount and RRN", http.StatusUnprocessableEntity)
	}
	now := s.now()
	var record *core.Record
	var action string
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		var err error
		record, action, queued, err = s.MatchInApp(tx, parsed, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return &MatchResult{Payment: FromRecord(record), Action: action}, nil
}

// MatchInApp applies exact-amount matching inside the caller's transaction.
// It returns whether outgoing webhook work was queued so the caller can wake the
// delivery loop only after the transaction commits.
func (s *Service) MatchInApp(tx core.App, parsed domain.ParsedSMS, now time.Time) (*core.Record, string, bool, error) {
	now = now.UTC()
	account, _, err := s.paymentAccount(string(parsed.Account))
	if err != nil {
		return nil, "not_matchable", false, err
	}
	rrn := strings.TrimSpace(parsed.RRN)
	evidenceAt := parsed.OccurredAt.UTC()
	if evidenceAt.IsZero() || evidenceAt.After(now) {
		evidenceAt = now
	}
	if rrn == "" || parsed.AmountPaise <= 0 {
		return nil, "not_matchable", false, domain.New("SMS_NOT_MATCHABLE", "bank SMS requires an exact amount and RRN", http.StatusUnprocessableEntity)
	}

	existing, err := tx.FindFirstRecordByData("payments", "rrn", rrn)
	if err == nil {
		if existing.GetString("payment_account") != account {
			return nil, "rrn_account_mismatch", false, domain.New("RRN_ACCOUNT_MISMATCH", "the UPI reference was already recorded for a different payment account", http.StatusConflict)
		}
		if int64(existing.GetInt("payable_amount")) != parsed.AmountPaise {
			return nil, "rrn_amount_mismatch", false, domain.New("RRN_AMOUNT_MISMATCH", "the UPI reference was already recorded with a different amount", http.StatusConflict)
		}
		return existing, "duplicate_rrn", false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, "error", false, err
	}

	createdBefore := evidenceAt.Add(EvidenceTimestampTolerance)

	// Match by when the bank says the credit occurred, not merely when the SMS
	// happened to reach PayGate. This prevents an on-time payment from becoming
	// "late" solely because the bank/phone delivered the SMS after expiry.
	onTime, err := tx.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && created_at <= {:createdBefore} && ((status = 'pending' && expires_at >= {:evidenceAt}) || (status = 'expired' && expires_at >= {:evidenceAt} && reuse_after > {:now}) || (status = 'cancelled' && resolved_at != '' && resolved_at >= {:evidenceAt} && reuse_after > {:now}))",
		"created",
		2,
		0,
		dbx.Params{"account": account, "amount": parsed.AmountPaise, "now": filterDate(now), "evidenceAt": filterDate(evidenceAt), "createdBefore": filterDate(createdBefore)},
	)
	if err != nil {
		return nil, "error", false, err
	}
	if len(onTime) > 1 {
		return nil, "ambiguous", false, domain.AmbiguousMatch()
	}
	if len(onTime) == 1 {
		record := onTime[0]
		applyEvidence(record, parsed, domain.StatusPaid, now, s.Config.AmountQuarantine)
		if err := tx.Save(record); err != nil {
			return nil, "error", false, err
		}
		if err := s.schedule(tx, "payment.paid", record, now); err != nil {
			return nil, "error", false, err
		}
		return record, "marked_paid", true, nil
	}

	expired, err := s.ExpireDueInApp(tx, now)
	if err != nil {
		return nil, "error", false, err
	}
	queued := expired > 0

	late, err := tx.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && (status = 'expired' || status = 'cancelled') && reuse_after > {:now} && created_at <= {:createdBefore}",
		"-created",
		2,
		0,
		dbx.Params{"account": account, "amount": parsed.AmountPaise, "now": filterDate(now), "evidenceAt": filterDate(evidenceAt), "createdBefore": filterDate(createdBefore)},
	)
	if err != nil {
		return nil, "error", queued, err
	}
	if len(late) > 1 {
		return nil, "ambiguous", queued, domain.AmbiguousMatch()
	}
	if len(late) == 1 {
		record := late[0]
		applyEvidence(record, parsed, domain.StatusLate, now, s.Config.AmountQuarantine)
		if err := tx.Save(record); err != nil {
			return nil, "error", queued, err
		}
		if err := s.schedule(tx, "payment.late", record, now); err != nil {
			return nil, "error", queued, err
		}
		return record, "marked_late", true, nil
	}

	return nil, "unmatched", queued, nil
}

// ManualMatchInApp explicitly links reviewed bank evidence to one payment. It
// still enforces exact amount equality and global RRN uniqueness; the operator
// chooses the payment, but cannot override those monetary invariants.
func (s *Service) ManualMatchInApp(tx core.App, paymentID string, parsed domain.ParsedSMS, now time.Time) (*core.Record, string, bool, error) {
	now = now.UTC()
	paymentID = strings.TrimSpace(paymentID)
	rrn := strings.TrimSpace(parsed.RRN)
	if paymentID == "" || parsed.AmountPaise <= 0 || rrn == "" {
		return nil, "not_matchable", false, domain.New("MANUAL_MATCH_INVALID", "payment, exact amount and bank reference are required", http.StatusBadRequest)
	}
	record, err := tx.FindRecordById("payments", paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "not_found", false, domain.PaymentNotFound()
		}
		return nil, "error", false, err
	}
	account, _, accountErr := s.paymentAccount(string(parsed.Account))
	if accountErr != nil {
		return nil, "not_matchable", false, accountErr
	}
	if record.GetString("payment_account") != account {
		return nil, "account_mismatch", false, domain.New("PAYMENT_ACCOUNT_MISMATCH", "bank evidence belongs to a different payment account", http.StatusConflict)
	}
	if int64(record.GetInt("payable_amount")) != parsed.AmountPaise {
		return nil, "amount_mismatch", false, domain.New("MANUAL_AMOUNT_MISMATCH", "bank evidence amount does not equal the payment payable amount", http.StatusConflict)
	}

	existing, findErr := tx.FindFirstRecordByData("payments", "rrn", rrn)
	if findErr == nil && existing.Id != record.Id {
		return nil, "rrn_conflict", false, domain.New("RRN_ALREADY_ASSIGNED", "the bank reference is already assigned to another payment", http.StatusConflict)
	}
	if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
		return nil, "error", false, findErr
	}
	status := domain.PaymentStatus(record.GetString("status"))
	if status == domain.StatusPaid || status == domain.StatusLate {
		if record.GetString("rrn") == rrn {
			return record, "already_matched", false, nil
		}
		return nil, "resolved", false, domain.PaymentResolved(string(status))
	}

	evidenceAt := parsed.OccurredAt.UTC()
	if evidenceAt.IsZero() || evidenceAt.After(now) {
		evidenceAt = now
	}
	createdAt := record.GetDateTime("created_at").Time()
	if !createdAt.IsZero() && evidenceAt.Add(EvidenceTimestampTolerance).Before(createdAt) {
		return nil, "stale", false, domain.New("STALE_BANK_EVIDENCE", "bank evidence predates this payment", http.StatusConflict)
	}
	reuseAfter := record.GetDateTime("reuse_after").Time()
	if !reuseAfter.IsZero() && evidenceAt.After(reuseAfter) {
		return nil, "stale", false, domain.New("PAYMENT_QUARANTINE_ELAPSED", "bank transaction occurred after this amount fingerprint became reusable", http.StatusConflict)
	}

	target := domain.StatusLate
	expiresAt := record.GetDateTime("expires_at").Time()
	resolvedAt := record.GetDateTime("resolved_at").Time()
	if status == domain.StatusPending && (expiresAt.IsZero() || !evidenceAt.After(expiresAt)) {
		target = domain.StatusPaid
	} else if status == domain.StatusExpired && !expiresAt.IsZero() && !evidenceAt.After(expiresAt) {
		target = domain.StatusPaid
	} else if status == domain.StatusCancelled && !resolvedAt.IsZero() && !evidenceAt.After(resolvedAt) {
		target = domain.StatusPaid
	}

	applyEvidence(record, parsed, target, now, s.Config.AmountQuarantine)
	if err := tx.Save(record); err != nil {
		return nil, "error", false, err
	}
	event := "payment.late"
	action := "marked_late"
	if target == domain.StatusPaid {
		event = "payment.paid"
		action = "marked_paid"
	}
	if err := s.schedule(tx, event, record, now); err != nil {
		return nil, "error", false, err
	}
	return record, action, true, nil
}

func applyEvidence(record *core.Record, parsed domain.ParsedSMS, status domain.PaymentStatus, now time.Time, quarantine time.Duration) {
	paidAt := parsed.OccurredAt.UTC()
	if paidAt.IsZero() || paidAt.After(now) {
		paidAt = now.UTC()
	}
	record.Set("status", string(status))
	record.Set("rrn", strings.TrimSpace(parsed.RRN))
	record.Set("upi_id", strings.TrimSpace(parsed.UPIId))
	record.Set("payer_name", strings.TrimSpace(parsed.PayerName))
	record.Set("paid_at", paidAt)
	record.Set("resolved_at", now.UTC())
	extendReuseAfter(record, now.UTC().Add(quarantine))
}

func extendReuseAfter(record *core.Record, candidate time.Time) {
	existing := record.GetDateTime("reuse_after").Time()
	if existing.IsZero() || candidate.After(existing) {
		record.Set("reuse_after", candidate.UTC())
	}
}

func (s *Service) ExpireDue() (int, error) {
	now := s.now()
	count := 0
	err := s.App.RunInTransaction(func(tx core.App) error {
		var err error
		count, err = s.ExpireDueInApp(tx, now)
		return err
	})
	if err == nil && count > 0 {
		s.WakeWebhooks()
	}
	return count, err
}

// ExpireDueInApp changes only currently pending records whose persisted expiry
// timestamp is due. Existing reuse_after is retained because it was fixed at
// creation as expires_at + quarantine.
func (s *Service) ExpireDueInApp(tx core.App, now time.Time) (int, error) {
	now = now.UTC()
	records, err := tx.FindRecordsByFilter(
		"payments",
		"status = 'pending' && expires_at <= {:now}",
		"expires_at",
		0,
		0,
		dbx.Params{"now": filterDate(now)},
	)
	if err != nil {
		return 0, err
	}
	for _, record := range records {
		record.Set("status", string(domain.StatusExpired))
		record.Set("resolved_at", now)
		if record.GetDateTime("reuse_after").IsZero() {
			record.Set("reuse_after", now.Add(s.Config.AmountQuarantine))
		}
		if err := tx.Save(record); err != nil {
			return 0, err
		}
		if err := s.schedule(tx, "payment.expired", record, now); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func (s *Service) Stats() (map[string]int64, error) {
	if _, err := s.ExpireDue(); err != nil {
		return nil, err
	}
	result := map[string]int64{
		"total": 0, "pending": 0, "paid": 0, "expired": 0, "cancelled": 0, "late": 0,
	}
	records, err := s.App.FindAllRecords("payments")
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		result["total"]++
		status := record.GetString("status")
		if _, ok := result[status]; ok {
			result[status]++
		}
	}
	return result, nil
}

func FromRecord(record *core.Record) *domain.Payment {
	if record == nil {
		return nil
	}
	return &domain.Payment{
		ID:             record.Id,
		Account:        domain.PaymentAccount(record.GetString("payment_account")),
		RequestedPaise: int64(record.GetInt("requested_amount")),
		PayablePaise:   int64(record.GetInt("payable_amount")),
		Status:         domain.PaymentStatus(record.GetString("status")),
		ExpiresAt:      record.GetDateTime("expires_at").Time(),
		ReuseAfter:     record.GetDateTime("reuse_after").Time(),
		RRN:            record.GetString("rrn"),
		UPIId:          record.GetString("upi_id"),
		PayerName:      record.GetString("payer_name"),
		PaidAt:         record.GetDateTime("paid_at").Time(),
		ResolvedAt:     record.GetDateTime("resolved_at").Time(),
		ExternalID:     record.GetString("external_id"),
		IdempotencyKey: record.GetString("idempotency_key"),
	}
}

func PublicPayment(payment *domain.Payment) map[string]any {
	if payment == nil {
		return nil
	}
	return map[string]any{
		"id":                   payment.ID,
		"paymentAccount":       payment.Account,
		"requestedAmount":      payment.RequestedPaise / 100,
		"requestedAmountPaise": payment.RequestedPaise,
		"payableAmount":        money.FormatPaise(payment.PayablePaise),
		"payableAmountPaise":   payment.PayablePaise,
		"status":               payment.Status,
		"expiresAt":            formatTime(payment.ExpiresAt),
		"paidAt":               formatOptionalTime(payment.PaidAt),
	}
}

func CreateResponse(payment *domain.Payment, cfg config.Config) map[string]any {
	response := PublicPayment(payment)
	response["externalId"] = payment.ExternalID
	account, ok := cfg.PaymentAccount(string(payment.Account))
	if !ok {
		account, _ = cfg.PaymentAccount(cfg.DefaultPaymentAccount)
	}
	response["paymentAccountLabel"] = account.Label
	response["verificationMethod"] = account.Verification
	query := url.Values{}
	query.Set("pa", account.UPIID)
	query.Set("pn", account.PayeeName)
	query.Set("am", money.FormatPaise(payment.PayablePaise))
	query.Set("cu", "INR")
	query.Set("tr", payment.ID)
	query.Set("tn", payment.ID)
	response["upiUri"] = "upi://pay?" + query.Encode()
	return response
}

func (s *Service) paymentAccount(value string) (string, config.PaymentAccount, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(s.Config.DefaultPaymentAccount))
	}
	if value == "" {
		value = "kotak"
	}
	account, ok := s.Config.PaymentAccount(value)
	// Serving validates that Kotak has a real UPI ID. Keeping the domain service
	// usable without delivery configuration makes migrations and isolated tests
	// independent from production secrets.
	if !ok && value == string(domain.PaymentAccountKotak) {
		account = config.PaymentAccount{ID: "kotak", Label: "Kotak", Verification: "sms"}
		ok = true
	}
	if !ok {
		return "", config.PaymentAccount{}, domain.New("INVALID_PAYMENT_ACCOUNT", "paymentAccount must be an enabled account", http.StatusBadRequest)
	}
	return account.ID, account, nil
}

func (s *Service) WakeWebhooks() {
	if s.Webhooks != nil {
		s.Webhooks.Wake()
	}
}

func (s *Service) schedule(tx core.App, event string, payment *core.Record, at time.Time) error {
	if s.Webhooks == nil {
		return nil
	}
	return s.Webhooks.Schedule(tx, event, payment, at)
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func randomSuffixStart() (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(99))
	if err != nil {
		return 0, err
	}
	return n.Int64() + 1, nil
}

func validateAndNormalizeMetadata(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 1<<20 {
		return nil, domain.InvalidMetadata()
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, domain.InvalidMetadata()
	}
	return normalized, nil
}

func normalizeMetadata(value any) any {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var normalized any
	if json.Unmarshal(raw, &normalized) != nil {
		return value
	}
	return normalized
}

func metadataEqual(a, b any) bool {
	return reflect.DeepEqual(normalizeMetadata(a), normalizeMetadata(b))
}

func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatOptionalTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatTime(t)
}

type CapacityPool struct {
	RequestedAmountPaise int64   `json:"requestedAmountPaise"`
	RequestedAmount      string  `json:"requestedAmount"`
	Pending              int64   `json:"pending"`
	Quarantined          int64   `json:"quarantined"`
	Blocked              int64   `json:"blocked"`
	Available            int64   `json:"available"`
	UtilizationPercent   float64 `json:"utilizationPercent"`
	Level                string  `json:"level"`
}

type CapacitySnapshot struct {
	Pools         []CapacityPool `json:"pools"`
	WarningPools  int            `json:"warningPools"`
	CriticalPools int            `json:"criticalPools"`
}

func (s *Service) Capacity() (CapacitySnapshot, error) {
	now := s.now()
	if _, err := s.ExpireDue(); err != nil {
		return CapacitySnapshot{}, err
	}
	records, err := s.App.FindRecordsByFilter(
		"payments",
		"reuse_after > {:now}",
		"requested_amount,payable_amount",
		0,
		0,
		dbx.Params{"now": filterDate(now)},
	)
	if err != nil {
		return CapacitySnapshot{}, err
	}
	type counts struct {
		pending     int64
		quarantined int64
		amounts     map[int64]struct{}
	}
	byRequested := map[int64]*counts{}
	for _, record := range records {
		requested := int64(record.GetInt("requested_amount"))
		entry := byRequested[requested]
		if entry == nil {
			entry = &counts{amounts: map[int64]struct{}{}}
			byRequested[requested] = entry
		}
		entry.amounts[int64(record.GetInt("payable_amount"))] = struct{}{}
		if record.GetString("status") == string(domain.StatusPending) {
			entry.pending++
		} else {
			entry.quarantined++
		}
	}
	result := CapacitySnapshot{}
	for requested, entry := range byRequested {
		blocked := int64(len(entry.amounts))
		available := int64(99) - blocked
		if available < 0 {
			available = 0
		}
		utilization := float64(blocked) / 99 * 100
		level := "normal"
		if utilization >= 95 {
			level = "critical"
			result.CriticalPools++
		} else if utilization >= 70 {
			level = "warning"
			result.WarningPools++
		}
		result.Pools = append(result.Pools, CapacityPool{
			RequestedAmountPaise: requested,
			RequestedAmount:      money.FormatPaise(requested),
			Pending:              entry.pending,
			Quarantined:          entry.quarantined,
			Blocked:              blocked,
			Available:            available,
			UtilizationPercent:   utilization,
			Level:                level,
		})
	}
	sort.Slice(result.Pools, func(i, j int) bool {
		if result.Pools[i].Blocked == result.Pools[j].Blocked {
			return result.Pools[i].RequestedAmountPaise < result.Pools[j].RequestedAmountPaise
		}
		return result.Pools[i].Blocked > result.Pools[j].Blocked
	})
	return result, nil
}
