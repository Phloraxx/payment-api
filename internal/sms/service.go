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

type Result struct {
	EventID   string `json:"eventId"`
	Status    string `json:"status"`
	Action    string `json:"action"`
	PaymentID string `json:"paymentId,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type Service struct {
	App      core.App
	Payments *payments.Service
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
			result.Status = "error"
			result.Action = "parse_error"
			domainErr = domain.New("SMS_PARSE_ERROR", parseErr.Error(), 422)
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
			result.Status = "error"
			result.Action = "missing_rrn"
			domainErr = domain.New("SMS_MISSING_RRN", "bank credit has no usable UPI reference/RRN", 422)
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
				result.Status = "error"
				result.Action = "match_error"
				domainErr = dErr
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
			result.Status = "unmatched"
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
