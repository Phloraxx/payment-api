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
	App      core.App
	Payments *payments.Service
	Now      func() time.Time
}

func NewService(app core.App, paymentService *payments.Service) *Service {
	return &Service{App: app, Payments: paymentService, Now: time.Now}
}

func (s *Service) Ingest(input Input) (Result, error) {
	var result Result
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		var err error
		result, queued, err = s.IngestInApp(tx, input)
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

// IngestInApp performs the Paytm evidence write and payment match in the caller's transaction.
// The returned queued flag tells the caller to wake outgoing webhooks only after its outer
// transaction has committed.
func (s *Service) IngestInApp(app core.App, input Input) (Result, bool, error) {
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

	existing, err := app.FindFirstRecordByFilter("notification_events", "source = {:source} && source_event_id = {:id}", dbx.Params{"source": input.Source, "id": input.SourceEventID})
	var event *core.Record
	if err == nil {
		if existing.GetString("processing_status") != "unmatched" {
			result := resultFromEvent(existing)
			result.Action = "duplicate_event"
			result.Duplicate = true
			return result, false, nil
		}
		event = existing
		event.Set("processing_status", "received")
		event.Set("error", "")
	} else {
		if !errors.Is(err, sql.ErrNoRows) {
			return Result{}, false, err
		}
		collection, err := app.FindCollectionByNameOrId("notification_events")
		if err != nil {
			return Result{}, false, err
		}
		event = core.NewRecord(collection)
		event.Set("source", input.Source)
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
	}
	if err := app.Save(event); err != nil {
		return Result{}, false, err
	}
	result := Result{EventID: event.Id}

	combined := strings.TrimSpace(strings.Join([]string{input.Title, input.Body, input.BigText}, "\n"))
	parsed, parseErr := Parse(combined)
	if errors.Is(parseErr, ErrUnrecognized) {
		event.Set("processing_status", "ignored")
		event.Set("error", "not a recognized Paytm customer-payment notification")
		if err := app.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status = "ignored"
		result.Action = "ignored_non_payment_notification"
		return result, false, nil
	}
	if parseErr != nil {
		event.Set("processing_status", "error")
		event.Set("error", parseErr.Error())
		if err := app.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status = "error"
		result.Action = "parse_error"
		return result, false, nil
	}
	event.Set("amount", parsed.AmountPaise)
	event.Set("payer_name", parsed.PayerName)
	event.Set("processing_status", "parsed")
	occurredAt := notificationTime
	occurredUntil := notificationTime
	if !parsed.OccurredAt.IsZero() && !parsed.OccurredAt.After(now.Add(5*time.Minute)) {
		minuteStart := parsed.OccurredAt.UTC()
		minuteEnd := minuteStart.Add(time.Minute).Add(-time.Nanosecond)
		if !notificationTime.Before(minuteStart) && !notificationTime.After(minuteEnd) {
			// Paytm only prints minutes. Android's notification timestamp supplies
			// precise seconds when it agrees with that displayed minute.
			occurredAt = notificationTime
			occurredUntil = notificationTime
		} else {
			// For delayed/offline delivery, preserve the full minute interval so a
			// checkout created later in the same displayed minute remains eligible.
			occurredAt = minuteStart
			occurredUntil = minuteEnd
		}
	}

	payment, action, queued, matchErr := s.Payments.MatchNotificationInApp(app, payments.NotificationEvidence{
		Account: domain.PaymentAccountPaytm, AmountPaise: parsed.AmountPaise, PayerName: parsed.PayerName,
		OccurredAt: occurredAt, OccurredUntil: occurredUntil, Reference: "paytm-notification:" + input.SourceEventID,
	}, now)
	if matchErr != nil {
		event.Set("processing_status", "error")
		event.Set("error", matchErr.Error())
		if err := app.Save(event); err != nil {
			return Result{}, false, err
		}
		result.Status = "error"
		result.Action = "match_error"
		return result, false, nil
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
	if err := app.Save(event); err != nil {
		return Result{}, false, err
	}
	return result, queued, nil
}

func (s *Service) RetryEvent(eventID string) (Result, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return Result{}, domain.New("INVALID_NOTIFICATION_EVENT_ID", "notification event id is required", 400)
	}
	event, err := s.App.FindRecordById("notification_events", eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{}, domain.New("NOTIFICATION_EVENT_NOT_FOUND", "notification event was not found", 404)
		}
		return Result{}, err
	}
	if event.GetString("processing_status") != "unmatched" {
		return Result{}, domain.New("NOTIFICATION_EVENT_NOT_RETRYABLE", "only unmatched notification events can be retried", 409)
	}
	return s.Ingest(Input{
		Source: event.GetString("source"), SourceEventID: event.GetString("source_event_id"),
		AppPackage: event.GetString("app_package"), AppName: event.GetString("app_name"),
		Title: event.GetString("title"), Body: event.GetString("body"), BigText: event.GetString("big_text"),
		Channel: event.GetString("channel"), NotificationTime: event.GetDateTime("notification_time").Time(),
		RawPayload: event.Get("raw_payload"),
	})
}

func resultFromEvent(event *core.Record) Result {
	return Result{EventID: event.Id, Status: event.GetString("processing_status"), PaymentID: event.GetString("matched_payment")}
}
