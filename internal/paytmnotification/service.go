package paytmnotification

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

const PaytmBusinessPackage = "com.paytm.business"

type Input struct {
	Source           string
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
	Store    store.Database
	Payments *payments.Service
	Now      func() time.Time
}

func NewService(app core.App, paymentService *payments.Service) *Service {
	return &Service{Store: store.NewPocketBase(app), Payments: paymentService, Now: time.Now}
}

func (s *Service) Ingest(input Input) (Result, error) {
	var result Result
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		result, queued, err = s.IngestUoW(uow, input)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	if queued {
		s.Payments.WakeWebhooks()
	}
	return result, nil
}

// IngestUoW persists Paytm evidence and applies matching in the caller's
// transaction. Android relay uses this to keep the relay event and payment
// mutation atomic with the downstream notification record.
func (s *Service) IngestUoW(uow store.UnitOfWork, input Input) (Result, bool, error) {
	input.Source = strings.TrimSpace(input.Source)
	if input.Source == "" {
		input.Source = "macrodroid"
	}
	input.SourceEventID = strings.TrimSpace(input.SourceEventID)
	input.AppPackage = strings.TrimSpace(input.AppPackage)
	input.AppName = strings.TrimSpace(input.AppName)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.BigText = strings.TrimSpace(input.BigText)
	input.Channel = strings.TrimSpace(input.Channel)
	if input.Source != "macrodroid" && input.Source != "android_relay" {
		return Result{}, false, domain.New("INVALID_NOTIFICATION_SOURCE", "unsupported Paytm notification source", 400)
	}
	if input.SourceEventID == "" || utf8.RuneCountInString(input.SourceEventID) > 255 {
		return Result{}, false, domain.New("INVALID_NOTIFICATION_SOURCE_ID", "sourceId is required and must be at most 255 characters", 400)
	}
	if input.AppPackage != PaytmBusinessPackage {
		return Result{}, false, domain.New("INVALID_NOTIFICATION_APP", "notification must come from the Paytm for Business app", 400)
	}
	if utf8.RuneCountInString(input.Title)+utf8.RuneCountInString(input.Body)+utf8.RuneCountInString(input.BigText) > 128*1024 {
		return Result{}, false, domain.New("NOTIFICATION_TOO_LARGE", "notification content is too large", 400)
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	notificationTime := input.NotificationTime.UTC()
	if notificationTime.IsZero() || notificationTime.After(now.Add(5*time.Minute)) {
		notificationTime = now
	}

	events := uow.NotificationEvents()
	event, findErr := events.FindBySourceEvent(input.Source, input.SourceEventID)
	if findErr == nil {
		if event.ProcessingStatus != "unmatched" {
			result := resultFromEvent(event)
			result.Action = "duplicate_event"
			result.Duplicate = true
			return result, false, nil
		}
		event.ProcessingStatus, event.Error = "received", ""
		if err := events.Save(event); err != nil {
			return Result{}, false, err
		}
	} else {
		if !errors.Is(findErr, sql.ErrNoRows) {
			return Result{}, false, findErr
		}
		event = &domain.NotificationEvent{Source: input.Source, SourceEventID: input.SourceEventID, AppPackage: input.AppPackage, AppName: input.AppName, Title: input.Title, Body: input.Body, BigText: input.BigText, Channel: input.Channel, NotificationTime: notificationTime, Account: domain.PaymentAccountPaytm, ProcessingStatus: "received", RawPayload: input.RawPayload}
		if err := events.Create(event); err != nil {
			return Result{}, false, err
		}
	}
	result := Result{EventID: event.ID}
	combined := strings.TrimSpace(strings.Join([]string{input.Title, input.Body, input.BigText}, "\n"))
	parsed, parseErr := Parse(combined)
	if errors.Is(parseErr, ErrUnrecognized) {
		event.ProcessingStatus, event.Error = "ignored", "not a recognized Paytm customer-payment notification"
		if err := events.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status, result.Action = "ignored", "ignored_non_payment_notification"
		return result, false, nil
	}
	if parseErr != nil {
		event.ProcessingStatus, event.Error = "error", parseErr.Error()
		if err := events.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status, result.Action = "error", "parse_error"
		return result, false, nil
	}
	event.AmountPaise, event.PayerName, event.ProcessingStatus = parsed.AmountPaise, parsed.PayerName, "parsed"
	occurredAt, occurredUntil := notificationTime, notificationTime
	if !parsed.OccurredAt.IsZero() && !parsed.OccurredAt.After(now.Add(5*time.Minute)) {
		minuteStart := parsed.OccurredAt.UTC()
		minuteEnd := minuteStart.Add(time.Minute).Add(-time.Nanosecond)
		if notificationTime.Before(minuteStart) || notificationTime.After(minuteEnd) {
			occurredAt, occurredUntil = minuteStart, minuteEnd
		}
	}
	payment, action, queued, matchErr := s.Payments.MatchNotification(uow, payments.NotificationEvidence{Account: domain.PaymentAccountPaytm, AmountPaise: parsed.AmountPaise, PayerName: parsed.PayerName, OccurredAt: occurredAt, OccurredUntil: occurredUntil, Reference: "paytm-notification:" + input.SourceEventID}, now)
	if matchErr != nil {
		event.ProcessingStatus, event.Error = "error", matchErr.Error()
		if err := events.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status, result.Action = "error", "match_error"
		return result, false, nil
	}
	switch action {
	case "marked_paid", "marked_late":
		event.ProcessingStatus, event.MatchedPaymentID, result.Status, result.PaymentID = "matched", payment.ID, "matched", payment.ID
	case "duplicate_evidence":
		event.ProcessingStatus, event.MatchedPaymentID, result.Status, result.PaymentID, result.Duplicate = "duplicate", payment.ID, "duplicate", payment.ID, true
	case "unmatched":
		event.ProcessingStatus, event.Error, result.Status = "unmatched", "no eligible Paytm payment has this exact amount", "unmatched"
	default:
		event.ProcessingStatus, event.Error, result.Status = "error", "unexpected matching action: "+action, "error"
	}
	result.Action = action
	if err := events.Save(event); err != nil {
		return Result{}, false, err
	}
	return result, queued, nil
}

func (s *Service) RetryEvent(eventID string) (Result, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return Result{}, domain.New("INVALID_NOTIFICATION_EVENT_ID", "notification event id is required", 400)
	}
	var input Input
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		event, err := uow.NotificationEvents().Get(eventID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.New("NOTIFICATION_EVENT_NOT_FOUND", "notification event was not found", 404)
			}
			return err
		}
		if event.ProcessingStatus != "unmatched" && event.ProcessingStatus != "error" {
			return domain.New("NOTIFICATION_EVENT_NOT_RETRYABLE", "only unmatched or failed notification events can be retried", 409)
		}
		if event.MatchedPaymentID != "" {
			return domain.New("NOTIFICATION_EVENT_NOT_RETRYABLE", "matched notification events cannot be retried", 409)
		}
		if event.ProcessingStatus == "error" {
			event.ProcessingStatus, event.Error = "unmatched", ""
			if err := uow.NotificationEvents().Save(event); err != nil {
				return err
			}
		}
		input = Input{Source: event.Source, SourceEventID: event.SourceEventID, AppPackage: event.AppPackage, AppName: event.AppName, Title: event.Title, Body: event.Body, BigText: event.BigText, Channel: event.Channel, NotificationTime: event.NotificationTime, RawPayload: event.RawPayload}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return s.Ingest(input)
}

func resultFromEvent(event *domain.NotificationEvent) Result {
	return Result{EventID: event.ID, Status: event.ProcessingStatus, PaymentID: event.MatchedPaymentID}
}
