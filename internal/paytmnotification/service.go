package paytmnotification

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

const PaytmBusinessPackage = "com.paytm.business"

type Input struct {
	SourceEventID    string
	AppPackage       string
	AppName          string
	Title            string
	Body             string
	BigText          string
	Channel          string
	NotificationTime time.Time
	RawPayload       any
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
	input.SourceEventID = strings.TrimSpace(input.SourceEventID)
	input.AppPackage = strings.TrimSpace(input.AppPackage)
	input.AppName = strings.TrimSpace(input.AppName)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.BigText = strings.TrimSpace(input.BigText)
	input.Channel = strings.TrimSpace(input.Channel)
	if input.SourceEventID == "" || utf8.RuneCountInString(input.SourceEventID) > 255 {
		return Result{}, domain.New("INVALID_NOTIFICATION_SOURCE_ID", "sourceId is required and must be at most 255 characters", 400)
	}
	if input.AppPackage != PaytmBusinessPackage {
		return Result{}, domain.New("INVALID_NOTIFICATION_APP", "notification must come from the Paytm for Business app", 400)
	}
	if utf8.RuneCountInString(input.Title)+utf8.RuneCountInString(input.Body)+utf8.RuneCountInString(input.BigText) > 128*1024 {
		return Result{}, domain.New("NOTIFICATION_TOO_LARGE", "notification content is too large", 400)
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	notificationTime := input.NotificationTime.UTC()
	if notificationTime.IsZero() || notificationTime.After(now) {
		notificationTime = now
	}

	var result Result
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		existing, err := tx.FindFirstRecordByFilter("notification_events", "source = macrodroid && source_event_id = {:id}", dbx.Params{"id": input.SourceEventID})
		if err == nil {
			result = resultFromEvent(existing)
			result.Action = "duplicate_event"
			result.Duplicate = true
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		collection, err := tx.FindCollectionByNameOrId("notification_events")
		if err != nil {
			return err
		}
		event := core.NewRecord(collection)
		event.Set("source", "macrodroid")
		event.Set("source_event_id", input.SourceEventID)
		event.Set("app_package", input.AppPackage)
		event.Set("app_name", input.AppName)
		event.Set("title", input.Title)
		event.Set("body", input.Body)
		event.Set("big_text", input.BigText)
		event.Set("channel", input.Channel)
		event.Set("notification_time", notificationTime)
		event.Set("payment_account", string(domain.PaymentAccountPaytm))
		event.Set("processing_status", "received")
		if input.RawPayload != nil {
			event.Set("raw_payload", input.RawPayload)
		}
		if err := tx.Save(event); err != nil {
			return err
		}
		result.EventID = event.Id

		combined := strings.TrimSpace(strings.Join([]string{input.Title, input.Body, input.BigText}, "\n"))
		parsed, parseErr := Parse(combined)
		if errors.Is(parseErr, ErrUnrecognized) {
			event.Set("processing_status", "ignored")
			event.Set("error", "not a recognized Paytm customer-payment notification")
			if err := tx.Save(event); err != nil {
				return err
			}
			result.Status = "ignored"
			result.Action = "ignored_non_payment_notification"
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
			return nil
		}
		event.Set("amount", parsed.AmountPaise)
		event.Set("payer_name", parsed.PayerName)
		event.Set("processing_status", "parsed")

		payment, action, matchQueued, matchErr := s.Payments.MatchNotificationInApp(tx, payments.NotificationEvidence{
			Account: domain.PaymentAccountPaytm, AmountPaise: parsed.AmountPaise, PayerName: parsed.PayerName,
			OccurredAt: notificationTime, Reference: "paytm-notification:" + input.SourceEventID,
		}, now)
		queued = queued || matchQueued
		if matchErr != nil {
			event.Set("processing_status", "error")
			event.Set("error", matchErr.Error())
			if err := tx.Save(event); err != nil {
				return err
			}
			result.Status = "error"
			result.Action = "match_error"
			return nil
		}
		switch action {
		case "marked_paid", "marked_late":
			event.Set("processing_status", "matched")
			event.Set("matched_payment", payment.Id)
			result.Status = "matched"
			result.PaymentID = payment.Id
		case "duplicate_evidence":
			event.Set("processing_status", "duplicate")
			event.Set("matched_payment", payment.Id)
			result.Status = "duplicate"
			result.PaymentID = payment.Id
			result.Duplicate = true
		case "unmatched":
			event.Set("processing_status", "unmatched")
			event.Set("error", "no eligible Paytm payment has this exact amount")
			result.Status = "unmatched"
		default:
			event.Set("processing_status", "error")
			event.Set("error", "unexpected matching action: "+action)
			result.Status = "error"
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

func resultFromEvent(event *core.Record) Result {
	return Result{EventID: event.Id, Status: event.GetString("processing_status"), PaymentID: event.GetString("matched_payment")}
}
