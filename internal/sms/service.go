package sms

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

type Input struct {
	Source        string
	SourceEventID string
	Sender        string
	Body          string
	MessageTime   time.Time
	RawPayload    any
}

type ReviewInput struct {
	Kind                string
	Severity            string
	SMSEventID          string
	PaymentID           string
	CandidatePaymentIDs []string
	Reason              string
	OpenedAt            time.Time
}

type ReviewWriter interface {
	OpenSMSReview(uow store.UnitOfWork, input ReviewInput) (string, error)
}

type Result struct {
	EventID      string `json:"eventId"`
	Status       string `json:"status"`
	Action       string `json:"action"`
	PaymentID    string `json:"paymentId,omitempty"`
	ReviewCaseID string `json:"reviewCaseId,omitempty"`
	Duplicate    bool   `json:"duplicate,omitempty"`
}

type Service struct {
	Store    store.Database
	Payments *payments.Service
	Reviews  ReviewWriter
	Now      func() time.Time
}

func NewService(app core.App, paymentService *payments.Service) *Service {
	return &Service{Store: store.NewPocketBase(app), Payments: paymentService, Now: time.Now}
}

func (s *Service) Ingest(input Input) (Result, error) {
	input.Source = strings.TrimSpace(input.Source)
	if input.Source == "" {
		input.Source = "manual"
	}
	if !validSource(input.Source) {
		return Result{}, domain.New("INVALID_SMS_SOURCE", "source must be android_webhook, gmessages, or manual", 400)
	}
	input.SourceEventID = strings.TrimSpace(input.SourceEventID)
	input.Sender = strings.TrimSpace(input.Sender)
	input.Body = strings.TrimSpace(input.Body)
	if utf8.RuneCountInString(input.SourceEventID) > 255 {
		return Result{}, domain.InvalidSMS("sourceId must be at most 255 characters")
	}
	if utf8.RuneCountInString(input.Sender) > 255 {
		return Result{}, domain.InvalidSMS("sender must be at most 255 characters")
	}
	if utf8.RuneCountInString(input.Body) > 64*1024 {
		return Result{}, domain.InvalidSMS("sms body must be at most 65536 characters")
	}
	if input.Body == "" {
		return Result{}, domain.InvalidSMS("sms body is required")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	messageTime := input.MessageTime.UTC()
	if messageTime.IsZero() || messageTime.After(now) {
		messageTime = now
	}

	var result Result
	var domainErr error
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		events := uow.SMSEvents()
		if input.SourceEventID != "" {
			existing, err := events.FindBySourceEvent(input.Source, input.SourceEventID)
			if err == nil {
				result = resultFromEvent(existing)
				result.Action = "duplicate_event"
				result.Duplicate = true
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		event := &domain.SMSEvent{Source: input.Source, SourceEventID: input.SourceEventID, Sender: input.Sender, Body: input.Body, Account: domain.PaymentAccountKotak, MessageTime: messageTime, ProcessingStatus: "received", RawPayload: input.RawPayload}
		if err := events.Create(event); err != nil {
			return err
		}
		result.EventID = event.ID

		parsed, parseErr := Parse(input.Body)
		parsed.Account = domain.PaymentAccountKotak
		parsed.OccurredAt = messageTime
		if errors.Is(parseErr, ErrUnrecognized) {
			event.ProcessingStatus, event.Error = "ignored", "not a recognized bank credit message"
			if err := events.Save(event); err != nil {
				return err
			}
			result.Status, result.Action = "ignored", "ignored_non_bank_sms"
			return nil
		}
		if parseErr != nil {
			event.ProcessingStatus, event.Error = "error", parseErr.Error()
			if err := events.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "parse_error", Severity: "warning", SMSEventID: event.ID, Reason: "Bank-credit-like message could not be parsed: " + parseErr.Error(), OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "parse_error", caseID
			return nil
		}

		event.AmountPaise, event.RRN, event.UPIID, event.PayerName, event.ProcessingStatus = parsed.AmountPaise, parsed.RRN, parsed.UPIId, parsed.PayerName, "parsed"
		if strings.TrimSpace(parsed.RRN) == "" {
			event.ProcessingStatus, event.Error = "error", "bank credit has no usable UPI reference/RRN"
			if err := events.Save(event); err != nil {
				return err
			}
			candidates, err := candidatePaymentIDs(uow, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "missing_rrn", Severity: "warning", SMSEventID: event.ID, CandidatePaymentIDs: candidates, Reason: "Bank credit has an amount but no usable UPI reference/RRN", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "missing_rrn", caseID
			return nil
		}

		payment, outcome, matchQueued, matchErr := s.Payments.MatchBankEvidence(uow, parsed, now)
		action := string(outcome)
		queued = queued || matchQueued
		if matchErr != nil {
			var dErr *domain.Error
			if errors.As(matchErr, &dErr) {
				event.ProcessingStatus, event.Error = "error", dErr.Message
				if err := events.Save(event); err != nil {
					return err
				}
				kind := "ambiguous"
				if dErr.Code == "RRN_AMOUNT_MISMATCH" || dErr.Code == "RRN_ACCOUNT_MISMATCH" {
					kind = "rrn_conflict"
				}
				candidates, err := candidatePaymentIDs(uow, parsed.Account, parsed.AmountPaise, now)
				if err != nil {
					return err
				}
				caseID, err := s.openReview(uow, ReviewInput{Kind: kind, Severity: "critical", SMSEventID: event.ID, CandidatePaymentIDs: candidates, Reason: dErr.Message, OpenedAt: now})
				if err != nil {
					return err
				}
				result.Status, result.Action, result.ReviewCaseID = "review_required", "match_error", caseID
				return nil
			}
			return matchErr
		}

		switch action {
		case "marked_paid", "marked_late":
			event.ProcessingStatus, event.MatchedPaymentID, result.Status, result.PaymentID = "matched", payment.ID, "matched", payment.ID
		case "duplicate_rrn":
			event.ProcessingStatus, event.MatchedPaymentID, result.Status, result.PaymentID, result.Duplicate = "duplicate", payment.ID, "duplicate", payment.ID, true
		case "unmatched":
			event.ProcessingStatus, event.Error = "unmatched", "no eligible payment has this exact amount"
			candidates, err := candidatePaymentIDs(uow, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "unmatched", Severity: "warning", SMSEventID: event.ID, CandidatePaymentIDs: candidates, Reason: "No eligible payment has this exact amount", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.ReviewCaseID = "review_required", caseID
		default:
			event.ProcessingStatus, event.Error, result.Status, domainErr = "error", "unexpected matching action: "+action, "error", domain.New("INTERNAL_MATCH_STATE", "unexpected matching result", 500)
		}
		result.Action = action
		return events.Save(event)
	})
	if err != nil {
		return Result{}, err
	}
	if queued {
		s.Payments.WakeWebhooks()
	}
	if domainErr != nil {
		return result, domainErr
	}
	return result, nil
}

func resultFromEvent(event *domain.SMSEvent) Result {
	return Result{EventID: event.ID, Status: event.ProcessingStatus, PaymentID: event.MatchedPaymentID}
}

func validSource(source string) bool {
	switch source {
	case "android_webhook", "gmessages", "manual":
		return true
	default:
		return false
	}
}

func (s *Service) openReview(uow store.UnitOfWork, input ReviewInput) (string, error) {
	if s.Reviews == nil {
		return "", nil
	}
	return s.Reviews.OpenSMSReview(uow, input)
}

func candidatePaymentIDs(uow store.UnitOfWork, account domain.PaymentAccount, amountPaise int64, now time.Time) ([]string, error) {
	if amountPaise <= 0 {
		return nil, nil
	}
	payments, err := uow.Payments().ListFingerprintCandidates(account, amountPaise, now, 10)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(payments))
	for _, payment := range payments {
		ids = append(ids, payment.ID)
	}
	return ids, nil
}
