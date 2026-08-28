package reviews

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/paymentemail"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

type Service struct {
	App      core.App // transitional adapter for resolve/evidence reads
	Store    store.Database
	Payments *payments.Service
	Audit    *audit.Service
	Now      func() time.Time
}

type OpenInput struct {
	Kind                  string
	Severity              string
	SMSEventID            string
	EmailEventID          string
	ReconciliationEntryID string
	PaymentID             string
	CandidatePaymentIDs   []string
	Reason                string
	OpenedAt              time.Time
}

type ResolveInput struct {
	CaseID        string
	Action        string
	PaymentID     string
	BankReference string
	Note          string
	Actor         audit.Actor
}

type ResolveResult struct {
	CaseID    string `json:"caseId"`
	Status    string `json:"status"`
	PaymentID string `json:"paymentId,omitempty"`
	Action    string `json:"action"`
}

func NewService(app core.App, paymentService *payments.Service, auditService *audit.Service) *Service {
	return &Service{App: app, Store: store.NewPocketBase(app), Payments: paymentService, Audit: auditService, Now: time.Now}
}

func (s *Service) OpenSMSReview(uow store.UnitOfWork, input sms.ReviewInput) (string, error) {
	return s.Open(uow, OpenInput{
		Kind: input.Kind, Severity: input.Severity, SMSEventID: input.SMSEventID,
		PaymentID: input.PaymentID, CandidatePaymentIDs: input.CandidatePaymentIDs,
		Reason: input.Reason, OpenedAt: input.OpenedAt,
	})
}

func (s *Service) OpenSMSReviewInApp(app core.App, input sms.ReviewInput) (string, error) {
	return s.OpenSMSReview(store.NewPocketBaseUnit(app), input)
}

func (s *Service) OpenEmailReview(uow store.UnitOfWork, input paymentemail.ReviewInput) (string, error) {
	return s.Open(uow, OpenInput{
		Kind: input.Kind, Severity: input.Severity, EmailEventID: input.EmailEventID,
		PaymentID: input.PaymentID, CandidatePaymentIDs: input.CandidatePaymentIDs,
		Reason: input.Reason, OpenedAt: input.OpenedAt,
	})
}

func (s *Service) OpenEmailReviewInApp(app core.App, input paymentemail.ReviewInput) (string, error) {
	return s.OpenEmailReview(store.NewPocketBaseUnit(app), input)
}

func (s *Service) Open(uow store.UnitOfWork, input OpenInput) (string, error) {
	if input.SMSEventID == "" && input.EmailEventID == "" && input.ReconciliationEntryID == "" {
		return "", errors.New("evidence record is required for review")
	}
	repo := uow.Reviews()
	existing, err := repo.FindByEvidence(input.SMSEventID, input.EmailEventID, input.ReconciliationEntryID)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	openedAt := input.OpenedAt.UTC()
	if openedAt.IsZero() {
		openedAt = s.now()
	}
	review := &domain.ReviewCase{
		Kind: input.Kind, Status: "open", Severity: input.Severity,
		SMSEventID: input.SMSEventID, EmailEventID: input.EmailEventID,
		ReconciliationEntryID: input.ReconciliationEntryID, PaymentID: input.PaymentID,
		CandidatePaymentIDs: append([]string(nil), input.CandidatePaymentIDs...),
		Reason:              truncate(input.Reason, 4096), OpenedAt: openedAt,
	}
	if err := repo.Create(review); err != nil {
		return "", err
	}
	return review.ID, nil
}

// OpenInApp is retained while legacy evidence/reconciliation transactions are
// migrated to Store.Write. New callers should pass the existing UnitOfWork.
func (s *Service) OpenInApp(app core.App, input OpenInput) (string, error) {
	return s.Open(store.NewPocketBaseUnit(app), input)
}

func (s *Service) Resolve(input ResolveInput) (ResolveResult, error) {
	input.CaseID = strings.TrimSpace(input.CaseID)
	input.Action = strings.TrimSpace(input.Action)
	input.PaymentID = strings.TrimSpace(input.PaymentID)
	input.BankReference = normalizeReference(input.BankReference)
	input.Note = strings.TrimSpace(input.Note)
	if input.CaseID == "" || input.Action == "" {
		return ResolveResult{}, domain.New("INVALID_REVIEW_RESOLUTION", "case and action are required", 400)
	}
	if len(input.Note) < 3 || len(input.Note) > 4096 {
		return ResolveResult{}, domain.New("INVALID_REVIEW_NOTE", "resolution note must be between 3 and 4096 characters", 400)
	}
	now := s.now()
	var result ResolveResult
	var wake bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		caseRecord, err := tx.FindRecordById("review_cases", input.CaseID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.New("REVIEW_CASE_NOT_FOUND", "review case not found", 404)
			}
			return err
		}
		if caseRecord.GetString("status") != "open" {
			return domain.New("REVIEW_CASE_RESOLVED", "review case is already resolved", 409)
		}

		resolution := input.Action
		switch input.Action {
		case "manual_match":
			if input.PaymentID == "" {
				return domain.New("INVALID_REVIEW_RESOLUTION", "paymentId is required for a manual match", 400)
			}
			eventID := caseRecord.GetString("sms_event")
			emailEventID := caseRecord.GetString("email_event")
			entryID := caseRecord.GetString("reconciliation_entry")
			var parsed domain.ParsedSMS
			var event *core.Record
			var entry *core.Record
			var emailEvent *core.Record
			if eventID != "" {
				event, err = tx.FindRecordById("sms_events", eventID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{
					Account:     domain.PaymentAccount(event.GetString("payment_account")),
					AmountPaise: int64(event.GetInt("amount")), RRN: event.GetString("rrn"),
					UPIId: event.GetString("upi_id"), PayerName: event.GetString("payer_name"),
					OccurredAt: event.GetDateTime("message_time").Time(),
				}
			} else if emailEventID != "" {
				emailEvent, err = tx.FindRecordById("email_events", emailEventID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{
					Account:     domain.PaymentAccount(emailEvent.GetString("payment_account")),
					AmountPaise: int64(emailEvent.GetInt("amount")), RRN: emailEvent.GetString("rrn"),
					UPIId: emailEvent.GetString("upi_id"), PayerName: emailEvent.GetString("payer_name"),
					OccurredAt: emailEvent.GetDateTime("message_time").Time(),
				}
			} else if entryID != "" {
				entry, err = tx.FindRecordById("reconciliation_entries", entryID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{
					Account:     domain.PaymentAccountKotak,
					AmountPaise: int64(entry.GetInt("amount")), RRN: entry.GetString("rrn"),
					OccurredAt: entry.GetDateTime("transaction_time").Time(),
				}
			} else {
				return domain.New("REVIEW_HAS_NO_EVIDENCE", "review case has no bank evidence", 409)
			}
			reference := normalizeReference(parsed.RRN)
			if reference == "" {
				reference = input.BankReference
			}
			if reference == "" {
				return domain.New("BANK_REFERENCE_REQUIRED", "enter the bank RRN/UTR before manually matching this evidence", 400)
			}
			parsed.RRN = reference
			payment, action, queued, err := s.Payments.ManualMatchInApp(tx, input.PaymentID, parsed, now)
			if err != nil {
				return err
			}
			wake = wake || queued
			if event != nil {
				event.Set("rrn", reference)
				event.Set("processing_status", "matched")
				event.Set("matched_payment", payment.Id)
				event.Set("error", "")
				if err := tx.Save(event); err != nil {
					return err
				}
			}
			if emailEvent != nil {
				emailEvent.Set("rrn", reference)
				emailEvent.Set("processing_status", "matched")
				emailEvent.Set("matched_payment", payment.Id)
				emailEvent.Set("error", "")
				if err := tx.Save(emailEvent); err != nil {
					return err
				}
			}
			if entry != nil {
				entry.Set("rrn", reference)
				entry.Set("status", "matched")
				entry.Set("payment", payment.Id)
				entry.Set("notes", "Manually reconciled by operator")
				if err := tx.Save(entry); err != nil {
					return err
				}
			}
			caseRecord.Set("payment", payment.Id)
			result.PaymentID = payment.Id
			result.Action = action
		case "dismissed", "duplicate", "not_payment", "corrected":
			if input.Action == "dismissed" {
				caseRecord.Set("status", "dismissed")
			}
			result.Action = input.Action
		default:
			return domain.New("INVALID_REVIEW_RESOLUTION", "unsupported review action", 400)
		}

		if caseRecord.GetString("status") != "dismissed" {
			caseRecord.Set("status", "resolved")
		}
		caseRecord.Set("resolution", resolution)
		caseRecord.Set("resolution_note", input.Note)
		caseRecord.Set("resolved_by", input.Actor.ID)
		caseRecord.Set("resolved_at", now)
		if err := tx.Save(caseRecord); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordInApp(tx, audit.Entry{
				Action: "review." + resolution, Actor: input.Actor,
				EntityType: "review_case", EntityID: caseRecord.Id,
				Summary: "Operator resolved payment evidence review",
				Details: map[string]any{
					"paymentId":             result.PaymentID,
					"smsEventId":            caseRecord.GetString("sms_event"),
					"emailEventId":          caseRecord.GetString("email_event"),
					"reconciliationEntryId": caseRecord.GetString("reconciliation_entry"),
					"resolution":            resolution,
					"note":                  input.Note,
				},
				OccurredAt: now,
			}); err != nil {
				return err
			}
		}
		result.CaseID = caseRecord.Id
		result.Status = caseRecord.GetString("status")
		return nil
	})
	if err != nil {
		return ResolveResult{}, err
	}
	if wake {
		s.Payments.WakeWebhooks()
	}
	return result, nil
}

func (s *Service) OpenCount() (int64, error) {
	var count int64
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		count, err = uow.Reviews().OpenCount()
		return err
	})
	return count, err
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func normalizeReference(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}
