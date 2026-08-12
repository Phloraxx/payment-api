package sms

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
	OpenSMSReviewInApp(app core.App, input ReviewInput) (string, error)
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
	App      core.App
	Payments *payments.Service
	Reviews  ReviewWriter
	Now      func() time.Time
}

func NewService(app core.App, paymentService *payments.Service) *Service {
	return &Service{App: app, Payments: paymentService, Now: time.Now}
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
	err := s.App.RunInTransaction(func(tx core.App) error {
		if input.SourceEventID != "" {
			existing, err := tx.FindFirstRecordByFilter(
				"sms_events",
				"source = {:source} && source_event_id = {:id}",
				dbx.Params{"source": input.Source, "id": input.SourceEventID},
			)
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

		collection, err := tx.FindCollectionByNameOrId("sms_events")
		if err != nil {
			return err
		}
		event := core.NewRecord(collection)
		event.Set("source", input.Source)
		event.Set("source_event_id", input.SourceEventID)
		event.Set("sender", input.Sender)
		event.Set("body", input.Body)
		event.Set("payment_account", string(domain.PaymentAccountKotak))
		event.Set("processing_status", "received")
		event.Set("message_time", messageTime)
		if input.RawPayload != nil {
			event.Set("raw_payload", input.RawPayload)
		}
		if err := tx.Save(event); err != nil {
			return err
		}
		result.EventID = event.Id

		parsed, parseErr := Parse(input.Body)
		parsed.Account = domain.PaymentAccountKotak
		parsed.OccurredAt = messageTime
		if errors.Is(parseErr, ErrUnrecognized) {
			event.Set("processing_status", "ignored")
			event.Set("error", "not a recognized bank credit message")
			if err := tx.Save(event); err != nil {
				return err
			}
			result.Status = "ignored"
			result.Action = "ignored_non_bank_sms"
			return nil
		}
		if parseErr != nil {
			event.Set("processing_status", "error")
			event.Set("error", parseErr.Error())
			if err := tx.Save(event); err != nil {
				return err
			}
			caseID, err := s.openReviewInApp(tx, ReviewInput{
				Kind: "parse_error", Severity: "warning", SMSEventID: event.Id,
				Reason: "Bank-credit-like message could not be parsed: " + parseErr.Error(), OpenedAt: now,
			})
			if err != nil {
				return err
			}
			result.Status = "review_required"
			result.Action = "parse_error"
			result.ReviewCaseID = caseID
			return nil
		}

		event.Set("amount", parsed.AmountPaise)
		event.Set("rrn", parsed.RRN)
		event.Set("upi_id", parsed.UPIId)
		event.Set("payer_name", parsed.PayerName)
		event.Set("processing_status", "parsed")
		if strings.TrimSpace(parsed.RRN) == "" {
			event.Set("processing_status", "error")
			event.Set("error", "bank credit has no usable UPI reference/RRN")
			if err := tx.Save(event); err != nil {
				return err
			}
			candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReviewInApp(tx, ReviewInput{
				Kind: "missing_rrn", Severity: "warning", SMSEventID: event.Id,
				CandidatePaymentIDs: candidates, Reason: "Bank credit has an amount but no usable UPI reference/RRN", OpenedAt: now,
			})
			if err != nil {
				return err
			}
			result.Status = "review_required"
			result.Action = "missing_rrn"
			result.ReviewCaseID = caseID
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
				severity := "critical"
				if dErr.Code == "RRN_AMOUNT_MISMATCH" || dErr.Code == "RRN_ACCOUNT_MISMATCH" {
					kind = "rrn_conflict"
				}
				candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
				if err != nil {
					return err
				}
				caseID, err := s.openReviewInApp(tx, ReviewInput{
					Kind: kind, Severity: severity, SMSEventID: event.Id,
					CandidatePaymentIDs: candidates, Reason: dErr.Message, OpenedAt: now,
				})
				if err != nil {
					return err
				}
				result.Status = "review_required"
				result.Action = "match_error"
				result.ReviewCaseID = caseID
				return nil
			}
			return matchErr
		}

		switch action {
		case "marked_paid", "marked_late":
			event.Set("processing_status", "matched")
			event.Set("matched_payment", payment.Id)
			result.Status = "matched"
			result.PaymentID = payment.Id
		case "duplicate_rrn":
			event.Set("processing_status", "duplicate")
			event.Set("matched_payment", payment.Id)
			result.Status = "duplicate"
			result.PaymentID = payment.Id
			result.Duplicate = true
		case "unmatched":
			event.Set("processing_status", "unmatched")
			event.Set("error", "no eligible payment has this exact amount")
			candidates, err := candidatePaymentIDs(tx, parsed.Account, parsed.AmountPaise, now)
			if err != nil {
				return err
			}
			caseID, err := s.openReviewInApp(tx, ReviewInput{
				Kind: "unmatched", Severity: "warning", SMSEventID: event.Id,
				CandidatePaymentIDs: candidates, Reason: "No eligible payment has this exact amount", OpenedAt: now,
			})
			if err != nil {
				return err
			}
			result.Status = "review_required"
			result.ReviewCaseID = caseID
		default:
			event.Set("processing_status", "error")
			event.Set("error", "unexpected matching action: "+action)
			result.Status = "error"
			domainErr = domain.New("INTERNAL_MATCH_STATE", "unexpected matching result", 500)
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
	if domainErr != nil {
		return result, domainErr
	}
	return result, nil
}

func resultFromEvent(event *core.Record) Result {
	return Result{
		EventID:   event.Id,
		Status:    event.GetString("processing_status"),
		PaymentID: event.GetString("matched_payment"),
	}
}

func validSource(source string) bool {
	switch source {
	case "android_webhook", "gmessages", "manual":
		return true
	default:
		return false
	}
}

func (s *Service) openReviewInApp(app core.App, input ReviewInput) (string, error) {
	if s.Reviews == nil {
		return "", nil
	}
	return s.Reviews.OpenSMSReviewInApp(app, input)
}

func candidatePaymentIDs(app core.App, account domain.PaymentAccount, amountPaise int64, now time.Time) ([]string, error) {
	if amountPaise <= 0 {
		return nil, nil
	}
	records, err := app.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && reuse_after > {:now}",
		"-created_at",
		10,
		0,
		dbx.Params{"account": string(account), "amount": amountPaise, "now": now.UTC().Format("2006-01-02 15:04:05.000Z")},
	)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Id)
	}
	return ids, nil
}
