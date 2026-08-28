package payments

import (
	"context"
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
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

const EvidenceTimestampTolerance = 2 * time.Second

type WebhookScheduler interface {
	Schedule(app core.App, event string, payment *core.Record, at time.Time) error // transitional adapter
	SchedulePayment(uow store.UnitOfWork, event string, payment *domain.Payment, at time.Time) error
	Wake()
}

type Service struct {
	App      core.App // transitional write adapter; remove after write repositories migrate
	Store    store.Database
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
		Store:       store.NewPocketBase(app),
		Config:      cfg,
		Webhooks:    webhooks,
		Now:         time.Now,
		SuffixStart: randomSuffixStart,
	}
}

type CreateGate func(store.UnitOfWork) error

func (s *Service) Create(input CreateInput) (*domain.Payment, bool, error) {
	return s.CreateGuarded(input, nil)
}

// CreateGuarded preserves idempotency semantics while allowing callers to
// fail closed before allocating a new payment. The gate runs inside the same
// database transaction only after an idempotency replay has been ruled out.
func (s *Service) CreateGuarded(input CreateInput, gate CreateGate) (*domain.Payment, bool, error) {
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
	var result *domain.Payment
	reused := false

	err = s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		repo := uow.Payments()
		if input.IdempotencyKey != "" {
			existing, findErr := repo.FindByIdempotencyKey(input.IdempotencyKey)
			if findErr == nil {
				if existing.RequestedPaise != requested || existing.Account != domain.PaymentAccount(account) || existing.ExternalID != input.ExternalID || !metadataEqual(existing.Metadata, metadata) {
					return domain.IdempotencyConflict()
				}
				result = existing
				reused = true
				return nil
			}
			if !errors.Is(findErr, sql.ErrNoRows) {
				return findErr
			}
		}

		if gate != nil {
			if err := gate(uow); err != nil {
				return err
			}
		}

		start, err := s.SuffixStart()
		if err != nil {
			return fmt.Errorf("choose amount fingerprint: %w", err)
		}
		if start < 1 || start > 99 {
			return fmt.Errorf("invalid amount fingerprint start %d", start)
		}
		expiresAt := now.Add(s.Config.PaymentTTL)
		reuseAfter := expiresAt.Add(s.Config.AmountQuarantine)
		for i := int64(0); i < 99; i++ {
			suffix := ((start - 1 + i) % 99) + 1
			candidate := requested + suffix
			blocked, err := repo.IsFingerprintBlocked(candidate, now)
			if err != nil {
				return err
			}
			if blocked {
				continue
			}
			created, err := repo.Create(store.NewPayment{
				Account: domain.PaymentAccount(account), RequestedPaise: requested, PayablePaise: candidate,
				CreatedAt: now, ExpiresAt: expiresAt, ReuseAfter: reuseAfter,
				ExternalID: input.ExternalID, IdempotencyKey: input.IdempotencyKey, Metadata: metadata,
			})
			if err != nil {
				if input.IdempotencyKey != "" {
					if existing, findErr := repo.FindByIdempotencyKey(input.IdempotencyKey); findErr == nil {
						result, reused = existing, true
						return nil
					}
				}
				return err
			}
			result = created
			return nil
		}
		return domain.CapacityExhausted()
	})
	if err != nil {
		return nil, false, err
	}
	return result, reused, nil
}

func (s *Service) Get(id string) (*domain.Payment, error) {
	var payment *domain.Payment
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		payment, err = uow.Payments().Get(strings.TrimSpace(id))
		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.PaymentNotFound()
		}
		return nil, err
	}
	if payment.Status == domain.StatusPending && !payment.ExpiresAt.IsZero() && !payment.ExpiresAt.After(s.now()) {
		payment.Status = domain.StatusExpired
	}
	return payment, nil
}

func (s *Service) Cancel(id string) (*domain.Payment, error) {
	now := s.now()
	var result *domain.Payment
	queued := false
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		payment, err := uow.Payments().Get(strings.TrimSpace(id))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.PaymentNotFound()
			}
			return err
		}
		if payment.Status == domain.StatusCancelled {
			result = payment
			return nil
		}
		if payment.Status != domain.StatusPending {
			return domain.PaymentResolved(string(payment.Status))
		}
		if !payment.ExpiresAt.IsZero() && !payment.ExpiresAt.After(now) {
			return domain.PaymentResolved(string(domain.StatusExpired))
		}
		payment.Status = domain.StatusCancelled
		payment.ResolvedAt = now
		candidate := now.Add(s.Config.AmountQuarantine)
		if payment.ReuseAfter.IsZero() || candidate.After(payment.ReuseAfter) {
			payment.ReuseAfter = candidate
		}
		if err := uow.Payments().Save(payment); err != nil {
			return err
		}
		if err := s.scheduleTyped(uow, "payment.cancelled", payment, now); err != nil {
			return err
		}
		queued = true
		result = payment
		return nil
	})
	if err != nil {
		return nil, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return result, nil
}

func (s *Service) Match(parsed domain.ParsedSMS) (*MatchResult, error) {
	if parsed.AmountPaise <= 0 || strings.TrimSpace(parsed.RRN) == "" {
		return nil, domain.New("SMS_NOT_MATCHABLE", "bank SMS requires an exact amount and RRN", http.StatusUnprocessableEntity)
	}
	now := s.now()
	var payment *domain.Payment
	var outcome domain.MatchOutcome
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		payment, outcome, queued, err = s.MatchBankEvidence(uow, parsed, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	if queued {
		s.WakeWebhooks()
	}
	return &MatchResult{Payment: payment, Action: string(outcome)}, nil
}

// MatchBankEvidence normalizes trusted bank evidence into the common matcher.
func (s *Service) MatchBankEvidence(uow store.UnitOfWork, parsed domain.ParsedSMS, now time.Time) (*domain.Payment, domain.MatchOutcome, bool, error) {
	if parsed.AmountPaise <= 0 || strings.TrimSpace(parsed.RRN) == "" {
		return nil, domain.MatchNotMatchable, false, domain.New("SMS_NOT_MATCHABLE", "bank SMS requires an exact amount and RRN", http.StatusUnprocessableEntity)
	}
	source := domain.EvidenceSourceBankSMS
	if parsed.Account == domain.PaymentAccountSlice {
		source = domain.EvidenceSourceBankEmail
	}
	return s.MatchEvidence(uow, domain.Evidence{
		Account: parsed.Account, AmountPaise: parsed.AmountPaise,
		OccurredFrom: parsed.OccurredAt, OccurredUntil: parsed.OccurredAt,
		Reference: strings.TrimSpace(parsed.RRN), ReferenceKind: domain.EvidenceReferenceRRN,
		Source: source, PayerName: parsed.PayerName, UPIID: parsed.UPIId,
	}, now)
}

type NotificationEvidence struct {
	Account       domain.PaymentAccount
	AmountPaise   int64
	PayerName     string
	OccurredAt    time.Time
	OccurredUntil time.Time
	Reference     string
}

// MatchNotification applies normalized Paytm relay evidence through the same matcher.
func (s *Service) MatchNotification(uow store.UnitOfWork, evidence NotificationEvidence, now time.Time) (*domain.Payment, string, bool, error) {
	if evidence.Account != domain.PaymentAccountPaytm {
		return nil, string(domain.MatchNotMatchable), false, domain.New("NOTIFICATION_ACCOUNT_INVALID", "notification evidence is only valid for the Paytm account", http.StatusBadRequest)
	}
	if evidence.AmountPaise <= 0 || strings.TrimSpace(evidence.Reference) == "" {
		return nil, string(domain.MatchNotMatchable), false, domain.New("NOTIFICATION_NOT_MATCHABLE", "Paytm notification requires an exact amount and evidence reference", http.StatusUnprocessableEntity)
	}
	payment, outcome, queued, err := s.MatchEvidence(uow, domain.Evidence{
		Account: evidence.Account, AmountPaise: evidence.AmountPaise,
		OccurredFrom: evidence.OccurredAt, OccurredUntil: evidence.OccurredUntil,
		Reference: strings.TrimSpace(evidence.Reference), ReferenceKind: domain.EvidenceReferenceRelay,
		Source: domain.EvidenceSourcePaytmNotification, PayerName: evidence.PayerName,
	}, now)
	return payment, string(outcome), queued, err
}

// ManualMatch applies reviewed bank evidence to one explicitly selected payment.
// Operator choice does not bypass account, exact amount, RRN uniqueness,
// creation-time or quarantine invariants.
func (s *Service) ManualMatch(uow store.UnitOfWork, paymentID string, parsed domain.ParsedSMS, now time.Time) (*domain.Payment, string, bool, error) {
	now = now.UTC()
	paymentID = strings.TrimSpace(paymentID)
	rrn := strings.TrimSpace(parsed.RRN)
	if paymentID == "" || parsed.AmountPaise <= 0 || rrn == "" {
		return nil, "not_matchable", false, domain.New("MANUAL_MATCH_INVALID", "payment, exact amount and bank reference are required", http.StatusBadRequest)
	}
	payment, err := uow.Payments().Get(paymentID)
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
	if payment.Account != domain.PaymentAccount(account) {
		return nil, "account_mismatch", false, domain.New("PAYMENT_ACCOUNT_MISMATCH", "bank evidence belongs to a different payment account", http.StatusConflict)
	}
	if payment.PayablePaise != parsed.AmountPaise {
		return nil, "amount_mismatch", false, domain.New("MANUAL_AMOUNT_MISMATCH", "bank evidence amount does not equal the payment payable amount", http.StatusConflict)
	}

	existing, findErr := uow.Payments().FindByEvidenceReference(domain.EvidenceReferenceRRN, rrn)
	if findErr == nil && existing.ID != payment.ID {
		return nil, "rrn_conflict", false, domain.New("RRN_ALREADY_ASSIGNED", "the bank reference is already assigned to another payment", http.StatusConflict)
	}
	if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
		return nil, "error", false, findErr
	}
	if payment.Status == domain.StatusPaid || payment.Status == domain.StatusLate {
		if payment.RRN == rrn {
			return payment, "already_matched", false, nil
		}
		return nil, "resolved", false, domain.PaymentResolved(string(payment.Status))
	}

	evidenceAt := parsed.OccurredAt.UTC()
	if evidenceAt.IsZero() || evidenceAt.After(now) {
		evidenceAt = now
	}
	if !payment.CreatedAt.IsZero() && evidenceAt.Add(EvidenceTimestampTolerance).Before(payment.CreatedAt) {
		return nil, "stale", false, domain.New("STALE_BANK_EVIDENCE", "bank evidence predates this payment", http.StatusConflict)
	}
	if !payment.ReuseAfter.IsZero() && evidenceAt.After(payment.ReuseAfter) {
		return nil, "stale", false, domain.New("PAYMENT_QUARANTINE_ELAPSED", "bank transaction occurred after this amount fingerprint became reusable", http.StatusConflict)
	}

	target := domain.StatusLate
	if payment.Status == domain.StatusPending && (payment.ExpiresAt.IsZero() || !evidenceAt.After(payment.ExpiresAt)) {
		target = domain.StatusPaid
	} else if payment.Status == domain.StatusExpired && !payment.ExpiresAt.IsZero() && !evidenceAt.After(payment.ExpiresAt) {
		target = domain.StatusPaid
	} else if payment.Status == domain.StatusCancelled && !payment.ResolvedAt.IsZero() && !evidenceAt.After(payment.ResolvedAt) {
		target = domain.StatusPaid
	}

	applyBankEvidence(payment, parsed, target, now, s.Config.AmountQuarantine)
	if err := uow.Payments().Save(payment); err != nil {
		return nil, "error", false, err
	}
	event, action := "payment.late", "marked_late"
	if target == domain.StatusPaid {
		event, action = "payment.paid", "marked_paid"
	}
	if err := s.scheduleTyped(uow, event, payment, now); err != nil {
		return nil, "error", false, err
	}
	return payment, action, true, nil
}

func applyBankEvidence(payment *domain.Payment, parsed domain.ParsedSMS, status domain.PaymentStatus, now time.Time, quarantine time.Duration) {
	paidAt := parsed.OccurredAt.UTC()
	if paidAt.IsZero() || paidAt.After(now) {
		paidAt = now.UTC()
	}
	payment.Status = status
	payment.RRN = strings.TrimSpace(parsed.RRN)
	payment.UPIId = strings.TrimSpace(parsed.UPIId)
	payment.PayerName = strings.TrimSpace(parsed.PayerName)
	payment.PaidAt = paidAt
	payment.ResolvedAt = now.UTC()
	candidate := now.UTC().Add(quarantine)
	if payment.ReuseAfter.IsZero() || candidate.After(payment.ReuseAfter) {
		payment.ReuseAfter = candidate
	}
}

const (
	expireBatchSize  = 100
	expireMaxBatches = 10
)

func (s *Service) ExpireDue() (int, error) {
	now := s.now()
	total := 0
	for batch := 0; batch < expireMaxBatches; batch++ {
		count := 0
		err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
			var err error
			count, err = s.ExpireDueUoW(uow, now)
			return err
		})
		if err != nil {
			return total, err
		}
		total += count
		if count < expireBatchSize {
			break
		}
	}
	if total > 0 {
		s.WakeWebhooks()
	}
	return total, nil
}

// ExpireDueUoW processes one bounded batch using typed repositories. Payment
// state and any outgoing event are committed atomically by the caller's UoW.
func (s *Service) ExpireDueUoW(uow store.UnitOfWork, now time.Time) (int, error) {
	now = now.UTC()
	payments, err := uow.Payments().ListDue(now, expireBatchSize)
	if err != nil {
		return 0, err
	}
	for _, payment := range payments {
		payment.Status = domain.StatusExpired
		payment.ResolvedAt = now
		if payment.ReuseAfter.IsZero() {
			payment.ReuseAfter = now.Add(s.Config.AmountQuarantine)
		}
		if err := uow.Payments().Save(payment); err != nil {
			return 0, err
		}
		if err := s.scheduleTyped(uow, "payment.expired", payment, now); err != nil {
			return 0, err
		}
	}
	return len(payments), nil
}

func (s *Service) Stats() (map[string]int64, error) {
	now := s.now()
	result := map[string]int64{
		"total": 0, "pending": 0, "paid": 0, "expired": 0, "cancelled": 0, "late": 0,
	}
	var payments []*domain.Payment
	if err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		payments, err = uow.Payments().ListAll()
		return err
	}); err != nil {
		return nil, err
	}
	for _, payment := range payments {
		result["total"]++
		status := payment.Status
		if status == domain.StatusPending && !payment.ExpiresAt.IsZero() && !payment.ExpiresAt.After(now) {
			status = domain.StatusExpired
		}
		if _, ok := result[string(status)]; ok {
			result[string(status)]++
		}
	}
	return result, nil
}

func FromRecord(record *core.Record) *domain.Payment {
	if record == nil {
		return nil
	}
	return &domain.Payment{
		ID:                record.Id,
		Account:           domain.PaymentAccount(record.GetString("payment_account")),
		RequestedPaise:    int64(record.GetInt("requested_amount")),
		PayablePaise:      int64(record.GetInt("payable_amount")),
		Status:            domain.PaymentStatus(record.GetString("status")),
		CreatedAt:         record.GetDateTime("created_at").Time(),
		ExpiresAt:         record.GetDateTime("expires_at").Time(),
		ReuseAfter:        record.GetDateTime("reuse_after").Time(),
		RRN:               record.GetString("rrn"),
		UPIId:             record.GetString("upi_id"),
		PayerName:         record.GetString("payer_name"),
		EvidenceSource:    record.GetString("evidence_source"),
		EvidenceReference: record.GetString("evidence_reference"),
		PaidAt:            record.GetDateTime("paid_at").Time(),
		ResolvedAt:        record.GetDateTime("resolved_at").Time(),
		ExternalID:        record.GetString("external_id"),
		IdempotencyKey:    record.GetString("idempotency_key"),
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
	response["paymentFlow"] = account.Flow
	if account.Flow == "merchant_qr" {
		response["qrPayload"] = account.QRPayload
	} else {
		query := url.Values{}
		query.Set("pa", account.UPIID)
		if account.Flow != "qr_only" {
			query.Set("pn", account.PayeeName)
		}
		query.Set("am", money.FormatPaise(payment.PayablePaise))
		query.Set("cu", "INR")
		response["upiUri"] = "upi://pay?" + query.Encode()
	}
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

func (s *Service) scheduleTyped(uow store.UnitOfWork, event string, payment *domain.Payment, at time.Time) error {
	if s.Webhooks == nil {
		return nil
	}
	return s.Webhooks.SchedulePayment(uow, event, payment, at)
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
	var payments []*domain.Payment
	if err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		payments, err = uow.Payments().ListBlocked(now)
		return err
	}); err != nil {
		return CapacitySnapshot{}, err
	}
	type counts struct {
		pending     int64
		quarantined int64
		amounts     map[int64]struct{}
	}
	byRequested := map[int64]*counts{}
	for _, payment := range payments {
		entry := byRequested[payment.RequestedPaise]
		if entry == nil {
			entry = &counts{amounts: map[int64]struct{}{}}
			byRequested[payment.RequestedPaise] = entry
		}
		entry.amounts[payment.PayablePaise] = struct{}{}
		if payment.Status == domain.StatusPending && (payment.ExpiresAt.IsZero() || payment.ExpiresAt.After(now)) {
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
			RequestedAmountPaise: requested, RequestedAmount: money.FormatPaise(requested),
			Pending: entry.pending, Quarantined: entry.quarantined, Blocked: blocked,
			Available: available, UtilizationPercent: utilization, Level: level,
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
