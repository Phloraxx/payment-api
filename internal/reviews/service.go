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
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		review, err := uow.Reviews().Get(input.CaseID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.New("REVIEW_CASE_NOT_FOUND", "review case not found", 404)
			}
			return err
		}
		if review.Status != "open" {
			return domain.New("REVIEW_CASE_RESOLVED", "review case is already resolved", 409)
		}

		resolution := input.Action
		switch input.Action {
		case "manual_match":
			if input.PaymentID == "" {
				return domain.New("INVALID_REVIEW_RESOLUTION", "paymentId is required for a manual match", 400)
			}
			var parsed domain.ParsedSMS
			var smsEvent *domain.SMSEvent
			var emailEvent *domain.EmailEvent
			var entry *domain.ReconciliationEntry
			switch {
			case review.SMSEventID != "":
				smsEvent, err = uow.SMSEvents().Get(review.SMSEventID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{Account: smsEvent.Account, AmountPaise: smsEvent.AmountPaise, RRN: smsEvent.RRN, UPIId: smsEvent.UPIID, PayerName: smsEvent.PayerName, OccurredAt: smsEvent.MessageTime}
			case review.EmailEventID != "":
				emailEvent, err = uow.EmailEvents().Get(review.EmailEventID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{Account: emailEvent.Account, AmountPaise: emailEvent.AmountPaise, RRN: emailEvent.RRN, UPIId: emailEvent.UPIID, PayerName: emailEvent.PayerName, OccurredAt: emailEvent.MessageTime}
			case review.ReconciliationEntryID != "":
				entry, err = uow.ReconciliationEntries().Get(review.ReconciliationEntryID)
				if err != nil {
					return err
				}
				parsed = domain.ParsedSMS{Account: domain.PaymentAccountKotak, AmountPaise: entry.AmountPaise, RRN: entry.RRN, OccurredAt: entry.TransactionTime}
			default:
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
			payment, action, queued, err := s.Payments.ManualMatch(uow, input.PaymentID, parsed, now)
			if err != nil {
				return err
			}
			wake = wake || queued
			if smsEvent != nil {
				smsEvent.RRN, smsEvent.ProcessingStatus, smsEvent.MatchedPaymentID, smsEvent.Error = reference, "matched", payment.ID, ""
				if err := uow.SMSEvents().Save(smsEvent); err != nil {
					return err
				}
			}
			if emailEvent != nil {
				emailEvent.RRN, emailEvent.ProcessingStatus, emailEvent.MatchedPaymentID, emailEvent.Error = reference, "matched", payment.ID, ""
				if err := uow.EmailEvents().Save(emailEvent); err != nil {
					return err
				}
			}
			if entry != nil {
				entry.RRN, entry.Status, entry.PaymentID, entry.Notes = reference, "matched", payment.ID, "Manually reconciled by operator"
				if err := uow.ReconciliationEntries().Save(entry); err != nil {
					return err
				}
			}
			review.PaymentID = payment.ID
			result.PaymentID, result.Action = payment.ID, action
		case "dismissed", "duplicate", "not_payment", "corrected":
			if input.Action == "dismissed" {
				review.Status = "dismissed"
			}
			result.Action = input.Action
		default:
			return domain.New("INVALID_REVIEW_RESOLUTION", "unsupported review action", 400)
		}

		if review.Status != "dismissed" {
			review.Status = "resolved"
		}
		review.Resolution = resolution
		review.ResolutionNote = input.Note
		review.ResolvedBy = input.Actor.ID
		review.ResolvedAt = now
		if err := uow.Reviews().Save(review); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(uow, audit.Entry{
				Action: "review." + resolution, Actor: input.Actor,
				EntityType: "review_case", EntityID: review.ID,
				Summary: "Operator resolved payment evidence review",
				Details: map[string]any{
					"paymentId": result.PaymentID, "smsEventId": review.SMSEventID,
					"emailEventId": review.EmailEventID, "reconciliationEntryId": review.ReconciliationEntryID,
					"resolution": resolution, "note": input.Note,
				}, OccurredAt: now,
			}); err != nil {
				return err
			}
		}
		result.CaseID, result.Status = review.ID, review.Status
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
