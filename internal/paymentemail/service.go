package paymentemail

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
	Source         string
	SourceEventID  string
	EnvelopeSender string
	Recipient      string
	Message        Message
	ReceivedAt     time.Time
	RawPayload     any
}

type ReviewInput struct {
	Kind                string
	Severity            string
	EmailEventID        string
	PaymentID           string
	CandidatePaymentIDs []string
	Reason              string
	OpenedAt            time.Time
}

type ReviewWriter interface {
	OpenEmailReview(uow store.UnitOfWork, input ReviewInput) (string, error)
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
	Store         store.Database
	Payments      *payments.Service
	Reviews       ReviewWriter
	AllowedSender string
	AuthServID    string
	Now           func() time.Time
}

func NewService(app core.App, paymentService *payments.Service, allowedSender, authServID string) *Service {
	return &Service{Store: store.NewPocketBase(app), Payments: paymentService, AllowedSender: strings.ToLower(strings.TrimSpace(allowedSender)), AuthServID: strings.TrimSpace(authServID), Now: time.Now}
}

func (s *Service) Ingest(input Input) (Result, error) {
	input.Source = strings.TrimSpace(input.Source)
	if input.Source == "" {
		input.Source = "cloudflare_email"
	}
	if input.Source != "cloudflare_email" && input.Source != "manual" {
		return Result{}, domain.New("INVALID_EMAIL_SOURCE", "source must be cloudflare_email or manual", 400)
	}
	input.SourceEventID = strings.TrimSpace(input.SourceEventID)
	if input.SourceEventID == "" {
		input.SourceEventID = strings.TrimSpace(input.Message.MessageID)
	}
	if utf8.RuneCountInString(input.SourceEventID) > 255 {
		return Result{}, domain.New("INVALID_PAYMENT_EMAIL", "sourceId must be at most 255 characters", 400)
	}
	now := s.now()
	receivedAt := input.ReceivedAt.UTC()
	if receivedAt.IsZero() || receivedAt.After(now) {
		receivedAt = now
	}
	messageTime := input.Message.Date.UTC()
	if messageTime.IsZero() || messageTime.After(now) {
		messageTime = receivedAt
	}

	var result Result
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		events := uow.EmailEvents()
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
		event := &domain.EmailEvent{Source: input.Source, SourceEventID: input.SourceEventID, EnvelopeSender: truncateRunes(strings.TrimSpace(input.EnvelopeSender), 255), Recipient: truncateRunes(strings.TrimSpace(input.Recipient), 255), Sender: truncateRunes(input.Message.From, 255), Subject: truncateRunes(input.Message.Subject, 1024), Body: truncateRunes(input.Message.Body, 64*1024), Account: domain.PaymentAccountSlice, MessageTime: messageTime, ReceivedAt: receivedAt, AuthResult: truncateRunes(strings.Join(input.Message.AuthenticationResults, "\n"), 8192), ProcessingStatus: "received", RawPayload: input.RawPayload}
		if err := events.Create(event); err != nil {
			return err
		}
		result.EventID = event.ID
		if !strings.EqualFold(input.Message.From, s.AllowedSender) {
			event.ProcessingStatus, event.Error = "ignored", "sender is not the configured bank notification address"
			if err := events.Save(event); err != nil {
				return err
			}
			result.Status, result.Action = "ignored", "ignored_sender"
			return nil
		}
		parsed, parseErr := Parse(input.Message)
		parsed.Account = domain.PaymentAccountSlice
		parsed.OccurredAt = messageTime
		if errors.Is(parseErr, ErrUnrecognized) {
			event.ProcessingStatus, event.Error = "ignored", ErrUnrecognized.Error()
			if err := events.Save(event); err != nil {
				return err
			}
			result.Status, result.Action = "ignored", "ignored_non_payment_email"
			return nil
		}
		domainPart := input.Message.From
		if at := strings.LastIndex(domainPart, "@"); at >= 0 {
			domainPart = domainPart[at+1:]
		}
		if input.Source != "manual" && !AuthenticatedSender(input.Message.AuthenticationResults, s.AuthServID, domainPart) {
			event.ProcessingStatus, event.Error = "error", "bank sender authentication did not pass through the trusted mail receiver"
			if err := events.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "email_auth_failed", Severity: "critical", EmailEventID: event.ID, Reason: "Payment-looking email failed trusted DKIM/DMARC authentication", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "email_auth_failed", caseID
			return nil
		}
		if parseErr != nil {
			event.ProcessingStatus, event.Error = "error", parseErr.Error()
			if err := events.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "parse_error", Severity: "warning", EmailEventID: event.ID, Reason: "Bank-credit-like email could not be parsed: " + parseErr.Error(), OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "parse_error", caseID
			return nil
		}
		event.AmountPaise, event.RRN, event.UPIID, event.PayerName, event.ProcessingStatus = parsed.AmountPaise, parsed.RRN, parsed.UPIId, parsed.PayerName, "parsed"
		if strings.TrimSpace(parsed.RRN) == "" {
			event.ProcessingStatus, event.Error = "error", "bank credit email has no usable UPI reference/RRN"
			if err := events.Save(event); err != nil {
				return err
			}
			candidates, err := candidatePaymentIDs(uow, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReview(uow, ReviewInput{Kind: "missing_rrn", Severity: "warning", EmailEventID: event.ID, CandidatePaymentIDs: candidates, Reason: "Bank credit email has an amount but no usable UPI reference/RRN", OpenedAt: now})
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
				caseID, err := s.openReview(uow, ReviewInput{Kind: kind, Severity: "critical", EmailEventID: event.ID, CandidatePaymentIDs: candidates, Reason: dErr.Message, OpenedAt: now})
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
			caseID, err := s.openReview(uow, ReviewInput{Kind: "unmatched", Severity: "warning", EmailEventID: event.ID, CandidatePaymentIDs: candidates, Reason: "No eligible payment has this exact amount", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.ReviewCaseID = "review_required", caseID
		default:
			return domain.New("INTERNAL_EMAIL_MATCH_STATE", "unexpected email matching result", 500)
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
	return result, nil
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *Service) openReview(uow store.UnitOfWork, input ReviewInput) (string, error) {
	if s.Reviews == nil {
		return "", nil
	}
	return s.Reviews.OpenEmailReview(uow, input)
}

func resultFromEvent(event *domain.EmailEvent) Result {
	return Result{EventID: event.ID, Status: event.ProcessingStatus, PaymentID: event.MatchedPaymentID}
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

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
