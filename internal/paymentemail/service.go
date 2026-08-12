package paymentemail

import (
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/pocketbase/dbx"
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
	OpenEmailReviewInApp(app core.App, input ReviewInput) (string, error)
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
	App           core.App
	Payments      *payments.Service
	Reviews       ReviewWriter
	AllowedSender string
	AuthServID    string
	Now           func() time.Time
}

func NewService(app core.App, paymentService *payments.Service, allowedSender, authServID string) *Service {
	return &Service{App: app, Payments: paymentService, AllowedSender: strings.ToLower(strings.TrimSpace(allowedSender)), AuthServID: strings.TrimSpace(authServID), Now: time.Now}
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
	err := s.App.RunInTransaction(func(tx core.App) error {
		if input.SourceEventID != "" {
			existing, err := tx.FindFirstRecordByFilter("email_events", "source = {:source} && source_event_id = {:id}", dbx.Params{"source": input.Source, "id": input.SourceEventID})
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

		collection, err := tx.FindCollectionByNameOrId("email_events")
		if err != nil {
			return err
		}
		event := core.NewRecord(collection)
		event.Set("source", input.Source)
		event.Set("source_event_id", input.SourceEventID)
		event.Set("envelope_sender", truncateRunes(strings.TrimSpace(input.EnvelopeSender), 255))
		event.Set("recipient", truncateRunes(strings.TrimSpace(input.Recipient), 255))
		event.Set("sender", truncateRunes(input.Message.From, 255))
		event.Set("subject", truncateRunes(input.Message.Subject, 1024))
		event.Set("body", truncateRunes(input.Message.Body, 64*1024))
		event.Set("payment_account", string(domain.PaymentAccountSlice))
		event.Set("message_time", messageTime)
		event.Set("received_at", receivedAt)
		event.Set("auth_result", truncateRunes(strings.Join(input.Message.AuthenticationResults, "\n"), 8192))
		event.Set("processing_status", "received")
		if input.RawPayload != nil {
			event.Set("raw_payload", input.RawPayload)
		}
		if err := tx.Save(event); err != nil {
			return err
		}
		result.EventID = event.Id

		if !strings.EqualFold(input.Message.From, s.AllowedSender) {
			event.Set("processing_status", "ignored")
			event.Set("error", "sender is not the configured bank notification address")
			if err := tx.Save(event); err != nil {
				return err
			}
			result.Status, result.Action = "ignored", "ignored_sender"
			return nil
		}
		parsed, parseErr := Parse(input.Message)
		parsed.Account = domain.PaymentAccountSlice
		parsed.OccurredAt = messageTime
		if errors.Is(parseErr, ErrUnrecognized) {
			event.Set("processing_status", "ignored")
			event.Set("error", ErrUnrecognized.Error())
			if err := tx.Save(event); err != nil {
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
			event.Set("processing_status", "error")
			event.Set("error", "bank sender authentication did not pass through the trusted mail receiver")
			if err := tx.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReview(tx, ReviewInput{Kind: "email_auth_failed", Severity: "critical", EmailEventID: event.Id, Reason: "Payment-looking email failed trusted DKIM/DMARC authentication", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "email_auth_failed", caseID
			return nil
		}

		if parseErr != nil {
			event.Set("processing_status", "error")
			event.Set("error", parseErr.Error())
			if err := tx.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReview(tx, ReviewInput{Kind: "parse_error", Severity: "warning", EmailEventID: event.Id, Reason: "Bank-credit-like email could not be parsed: " + parseErr.Error(), OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "parse_error", caseID
			return nil
		}
		event.Set("amount", parsed.AmountPaise)
		event.Set("rrn", parsed.RRN)
		event.Set("upi_id", parsed.UPIId)
		event.Set("payer_name", parsed.PayerName)
		event.Set("processing_status", "parsed")
		if strings.TrimSpace(parsed.RRN) == "" {
			event.Set("processing_status", "error")
			event.Set("error", "bank credit email has no usable UPI reference/RRN")
			if err := tx.Save(event); err != nil {
				return err
			}
			candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReview(tx, ReviewInput{Kind: "missing_rrn", Severity: "warning", EmailEventID: event.Id, CandidatePaymentIDs: candidates, Reason: "Bank credit email has an amount but no usable UPI reference/RRN", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.Action, result.ReviewCaseID = "review_required", "missing_rrn", caseID
			return nil
		}

		payment, action, matchQueued, matchErr := s.Payments.MatchInApp(tx, parsed, now)
		queued = queued || matchQueued
		if matchErr != nil {
			var dErr *domain.Error
			if errors.As(matchErr, &dErr) {
				event.Set("processing_status", "error")
				event.Set("error", dErr.Message)
				if err := tx.Save(event); err != nil {
					return err
				}
				kind := "ambiguous"
				if dErr.Code == "RRN_AMOUNT_MISMATCH" || dErr.Code == "RRN_ACCOUNT_MISMATCH" {
					kind = "rrn_conflict"
				}
				candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
				if err != nil {
					return err
				}
				caseID, err := s.openReview(tx, ReviewInput{Kind: kind, Severity: "critical", EmailEventID: event.Id, CandidatePaymentIDs: candidates, Reason: dErr.Message, OpenedAt: now})
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
			event.Set("processing_status", "matched")
			event.Set("matched_payment", payment.Id)
			result.Status, result.PaymentID = "matched", payment.Id
		case "duplicate_rrn":
			event.Set("processing_status", "duplicate")
			event.Set("matched_payment", payment.Id)
			result.Status, result.PaymentID, result.Duplicate = "duplicate", payment.Id, true
		case "unmatched":
			event.Set("processing_status", "unmatched")
			event.Set("error", "no eligible payment has this exact amount")
			candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReview(tx, ReviewInput{Kind: "unmatched", Severity: "warning", EmailEventID: event.Id, CandidatePaymentIDs: candidates, Reason: "No eligible payment has this exact amount", OpenedAt: now})
			if err != nil {
				return err
			}
			result.Status, result.ReviewCaseID = "review_required", caseID
		default:
			return domain.New("INTERNAL_EMAIL_MATCH_STATE", "unexpected email matching result", 500)
		}
		result.Action = action
		return tx.Save(event)
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

func (s *Service) openReview(app core.App, input ReviewInput) (string, error) {
	if s.Reviews == nil {
		return "", nil
	}
	return s.Reviews.OpenEmailReviewInApp(app, input)
}

func resultFromEvent(event *core.Record) Result {
	return Result{EventID: event.Id, Status: event.GetString("processing_status"), PaymentID: event.GetString("matched_payment")}
}

func candidatePaymentIDs(app core.App, account domain.PaymentAccount, amountPaise int64, now time.Time) ([]string, error) {
	if amountPaise <= 0 {
		return nil, nil
	}
	records, err := app.FindRecordsByFilter("payments", "payment_account = {:account} && payable_amount = {:amount} && reuse_after > {:now}", "-created_at", 10, 0, dbx.Params{"account": string(account), "amount": amountPaise, "now": now.UTC().Format("2006-01-02 15:04:05.000Z")})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Id)
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
