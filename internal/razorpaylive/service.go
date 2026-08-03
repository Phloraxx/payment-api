package razorpaylive

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

const maxWebhookBytes = 1 << 20

type ProviderClient interface {
	CreateOrder(ctx context.Context, amountPaise int64, receipt string) (ProviderOrder, error)
	FetchPayment(ctx context.Context, paymentID string) (ProviderPayment, error)
}

type Service struct {
	App           core.App
	Client        ProviderClient
	KeyID         string
	KeySecret     string
	WebhookSecret string
	DisplayName   string
	Now           func() time.Time
}

type CreateInput struct {
	AmountPaise    int64
	ExternalID     string
	IdempotencyKey string
	ActorID        string
}

type VerifyInput struct {
	LocalOrderID      string
	RazorpayOrderID   string
	RazorpayPaymentID string
	RazorpaySignature string
}

type WebhookResult struct {
	Duplicate bool   `json:"duplicate"`
	Processed bool   `json:"processed"`
	Ignored   bool   `json:"ignored"`
	EventID   string `json:"eventId"`
	OrderID   string `json:"orderId,omitempty"`
	Status    string `json:"status,omitempty"`
}

type webhookEnvelope struct {
	Event     string `json:"event"`
	CreatedAt int64  `json:"created_at"`
	Payload   struct {
		Payment struct {
			Entity ProviderPayment `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

func NewService(app core.App, client ProviderClient, keyID, keySecret, webhookSecret, displayName string) *Service {
	return &Service{
		App: app, Client: client, KeyID: keyID, KeySecret: keySecret,
		WebhookSecret: webhookSecret, DisplayName: displayName, Now: time.Now,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*core.Record, bool, error) {
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.AmountPaise != 100 {
		return nil, false, domain.New("RAZORPAY_LIVE_INVALID_AMOUNT", "live pilot amount must be exactly ₹1", 400)
	}
	if input.IdempotencyKey == "" || len(input.IdempotencyKey) > 255 {
		return nil, false, domain.New("RAZORPAY_LIVE_IDEMPOTENCY_REQUIRED", "a valid Idempotency-Key is required", 400)
	}
	if len(input.ExternalID) > 255 {
		return nil, false, domain.InvalidExternalID()
	}

	if existing, err := s.App.FindFirstRecordByData("razorpay_live_orders", "idempotency_key", input.IdempotencyKey); err == nil {
		if int64(existing.GetInt("amount")) != input.AmountPaise || existing.GetString("external_id") != input.ExternalID {
			return nil, false, domain.New("RAZORPAY_LIVE_IDEMPOTENCY_CONFLICT", "the idempotency key was already used with different parameters", 409)
		}
		if status := existing.GetString("status"); status == "creating" || status == "create_failed" {
			domainErr := domain.New("RAZORPAY_LIVE_CREATE_STATE_UNKNOWN", "the previous provider-order attempt did not complete cleanly; inspect the Razorpay Live Dashboard using the local receipt before starting a new attempt", 409)
			domainErr.Details = map[string]any{"localOrderId": existing.Id, "receipt": "pgl_" + existing.Id, "status": status}
			return nil, false, domainErr
		}
		return existing, true, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	collection, err := s.App.FindCollectionByNameOrId("razorpay_live_orders")
	if err != nil {
		return nil, false, err
	}
	now := s.now()
	record := core.NewRecord(collection)
	record.Set("amount", input.AmountPaise)
	record.Set("currency", "INR")
	record.Set("status", "creating")
	record.Set("external_id", input.ExternalID)
	record.Set("idempotency_key", input.IdempotencyKey)
	record.Set("created_by", input.ActorID)
	record.Set("created_at", now)
	if err := s.App.Save(record); err != nil {
		if existing, findErr := s.App.FindFirstRecordByData("razorpay_live_orders", "idempotency_key", input.IdempotencyKey); findErr == nil {
			return existing, true, nil
		}
		return nil, false, err
	}

	providerOrder, err := s.Client.CreateOrder(ctx, input.AmountPaise, "pgl_"+record.Id)
	if err != nil {
		record.Set("status", "create_failed")
		record.Set("error", truncate(err.Error(), 4096))
		record.Set("last_synced_at", now)
		_ = s.App.Save(record)
		domainErr := domain.New("RAZORPAY_LIVE_CREATE_FAILED", "Razorpay live order creation failed", 502)
		domainErr.Details = map[string]any{"localOrderId": record.Id}
		return record, false, domainErr
	}
	record.Set("razorpay_order_id", providerOrder.ID)
	record.Set("provider_status", providerOrder.Status)
	record.Set("status", "created")
	record.Set("error", "")
	record.Set("last_synced_at", now)
	if err := s.App.Save(record); err != nil {
		return nil, false, err
	}
	return record, false, nil
}

func (s *Service) Get(localOrderID string) (*core.Record, error) {
	record, err := s.App.FindRecordById("razorpay_live_orders", strings.TrimSpace(localOrderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.New("RAZORPAY_LIVE_ORDER_NOT_FOUND", "Razorpay live order not found", 404)
	}
	return record, err
}

func (s *Service) Verify(ctx context.Context, input VerifyInput) (*core.Record, error) {
	record, err := s.Get(input.LocalOrderID)
	if err != nil {
		return nil, err
	}
	providerOrderID := record.GetString("razorpay_order_id")
	if providerOrderID == "" || input.RazorpayOrderID != providerOrderID {
		return nil, domain.New("RAZORPAY_LIVE_ORDER_MISMATCH", "checkout order id does not match the server-created order", 400)
	}
	if !strings.HasPrefix(input.RazorpayPaymentID, "pay_") {
		return nil, domain.New("RAZORPAY_LIVE_INVALID_PAYMENT", "invalid Razorpay payment id", 400)
	}
	if !verifyHexHMAC(s.KeySecret, providerOrderID+"|"+input.RazorpayPaymentID, input.RazorpaySignature) {
		return nil, domain.New("RAZORPAY_LIVE_SIGNATURE_INVALID", "Razorpay checkout signature verification failed", 400)
	}

	err = s.App.RunInTransaction(func(tx core.App) error {
		current, err := tx.FindRecordById("razorpay_live_orders", record.Id)
		if err != nil {
			return err
		}
		if existing := current.GetString("razorpay_payment_id"); existing != "" && existing != input.RazorpayPaymentID {
			return domain.New("RAZORPAY_LIVE_PAYMENT_CONFLICT", "the order is already linked to another Razorpay payment", 409)
		}
		if other, findErr := tx.FindFirstRecordByData("razorpay_live_orders", "razorpay_payment_id", input.RazorpayPaymentID); findErr == nil && other.Id != current.Id {
			return domain.New("RAZORPAY_LIVE_PAYMENT_CONFLICT", "the Razorpay payment is already linked to another live order", 409)
		} else if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		current.Set("razorpay_payment_id", input.RazorpayPaymentID)
		current.Set("signature_verified_at", s.now())
		if current.GetString("status") != "captured" && current.GetString("status") != "refunded" && current.GetString("status") != "partially_refunded" {
			current.Set("status", "verification_pending")
		}
		return tx.Save(current)
	})
	if err != nil {
		return nil, err
	}

	// A signed browser callback proves authenticity, not capture. Fetch the
	// provider state immediately for responsive test UX; webhooks remain the
	// authoritative asynchronous path if this fetch fails.
	if _, refreshErr := s.Refresh(ctx, record.Id); refreshErr != nil {
		var domainErr *domain.Error
		if errors.As(refreshErr, &domainErr) && domainErr.Code != "RAZORPAY_LIVE_REFRESH_FAILED" {
			return nil, refreshErr
		}
		return s.Get(record.Id)
	}
	return s.Get(record.Id)
}

func (s *Service) Refresh(ctx context.Context, localOrderID string) (*core.Record, error) {
	record, err := s.Get(localOrderID)
	if err != nil {
		return nil, err
	}
	paymentID := record.GetString("razorpay_payment_id")
	if paymentID == "" {
		return nil, domain.New("RAZORPAY_LIVE_PAYMENT_UNKNOWN", "no Razorpay payment id is linked to this order yet", 409)
	}
	payment, err := s.Client.FetchPayment(ctx, paymentID)
	if err != nil {
		return nil, domain.New("RAZORPAY_LIVE_REFRESH_FAILED", "could not fetch the Razorpay payment", 502)
	}
	if err := s.applyPayment(record.Id, payment, s.now()); err != nil {
		return nil, err
	}
	return s.Get(record.Id)
}

func (s *Service) IngestWebhook(eventID, signature string, raw []byte) (WebhookResult, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" || len(eventID) > 128 {
		return WebhookResult{}, domain.New("RAZORPAY_LIVE_EVENT_ID_REQUIRED", "X-Razorpay-Event-Id is required", 400)
	}
	if len(raw) == 0 || len(raw) > maxWebhookBytes {
		return WebhookResult{}, domain.New("RAZORPAY_LIVE_WEBHOOK_INVALID", "webhook body must be between 1 byte and 1 MiB", 400)
	}
	if !verifyHexHMAC(s.WebhookSecret, string(raw), signature) {
		return WebhookResult{}, domain.New("RAZORPAY_LIVE_WEBHOOK_SIGNATURE_INVALID", "invalid Razorpay webhook signature", 401)
	}
	hashBytes := sha256.Sum256(raw)
	payloadHash := hex.EncodeToString(hashBytes[:])
	if existing, err := s.App.FindFirstRecordByData("razorpay_live_events", "event_id", eventID); err == nil {
		if existing.GetString("payload_hash") != payloadHash {
			return WebhookResult{}, domain.New("RAZORPAY_LIVE_EVENT_ID_CONFLICT", "the Razorpay event id was already used with a different payload", 409)
		}
		return WebhookResult{Duplicate: true, EventID: eventID, OrderID: existing.GetString("live_order"), Status: existing.GetString("status")}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return WebhookResult{}, err
	}

	var envelope webhookEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return WebhookResult{}, domain.New("RAZORPAY_LIVE_WEBHOOK_INVALID", "invalid Razorpay webhook JSON", 400)
	}
	payment := envelope.Payload.Payment.Entity
	result := WebhookResult{EventID: eventID}
	now := s.now()
	err := s.App.RunInTransaction(func(tx core.App) error {
		if existing, err := tx.FindFirstRecordByData("razorpay_live_events", "event_id", eventID); err == nil {
			if existing.GetString("payload_hash") != payloadHash {
				return domain.New("RAZORPAY_LIVE_EVENT_ID_CONFLICT", "the Razorpay event id was already used with a different payload", 409)
			}
			result.Duplicate = true
			result.OrderID = existing.GetString("live_order")
			result.Status = existing.GetString("status")
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		collection, err := tx.FindCollectionByNameOrId("razorpay_live_events")
		if err != nil {
			return err
		}
		event := core.NewRecord(collection)
		event.Set("event_id", eventID)
		event.Set("event_type", truncate(envelope.Event, 128))
		event.Set("razorpay_order_id", payment.OrderID)
		event.Set("razorpay_payment_id", payment.ID)
		event.Set("payload_hash", payloadHash)
		event.Set("received_at", now)
		if envelope.CreatedAt > 0 {
			event.Set("provider_created_at", time.Unix(envelope.CreatedAt, 0).UTC())
		}

		order, findErr := tx.FindFirstRecordByData("razorpay_live_orders", "razorpay_order_id", payment.OrderID)
		if errors.Is(findErr, sql.ErrNoRows) {
			event.Set("status", "ignored")
			event.Set("error", "No local Razorpay live order matches this event")
			result.Ignored = true
			result.Status = "ignored"
			return tx.Save(event)
		}
		if findErr != nil {
			return findErr
		}
		event.Set("live_order", order.Id)
		result.OrderID = order.Id
		if envelope.Event != "payment.captured" && envelope.Event != "payment.failed" {
			event.Set("status", "ignored")
			result.Ignored = true
			result.Status = "ignored"
			return tx.Save(event)
		}
		if err := validateProviderPayment(order, payment); err != nil {
			event.Set("status", "failed")
			event.Set("error", truncate(err.Error(), 4096))
			result.Status = "failed"
			return tx.Save(event)
		}
		if err := applyProviderPayment(order, payment, now); err != nil {
			return err
		}
		if err := tx.Save(order); err != nil {
			return err
		}
		event.Set("status", "processed")
		result.Processed = true
		result.Status = order.GetString("status")
		return tx.Save(event)
	})
	return result, err
}

func (s *Service) applyPayment(localOrderID string, payment ProviderPayment, at time.Time) error {
	return s.App.RunInTransaction(func(tx core.App) error {
		order, err := tx.FindRecordById("razorpay_live_orders", localOrderID)
		if err != nil {
			return err
		}
		if err := validateProviderPayment(order, payment); err != nil {
			return err
		}
		if err := applyProviderPayment(order, payment, at); err != nil {
			return err
		}
		return tx.Save(order)
	})
}

func validateProviderPayment(order *core.Record, payment ProviderPayment) error {
	if payment.ID == "" || payment.OrderID != order.GetString("razorpay_order_id") {
		return domain.New("RAZORPAY_LIVE_PROVIDER_MISMATCH", "Razorpay payment does not belong to the local order", 409)
	}
	if payment.Amount != int64(order.GetInt("amount")) || !strings.EqualFold(payment.Currency, order.GetString("currency")) {
		return domain.New("RAZORPAY_LIVE_PROVIDER_MISMATCH", "Razorpay payment amount or currency does not match the local order", 409)
	}
	return nil
}

func applyProviderPayment(order *core.Record, payment ProviderPayment, at time.Time) error {
	if existing := order.GetString("razorpay_payment_id"); existing != "" && existing != payment.ID {
		return domain.New("RAZORPAY_LIVE_PAYMENT_CONFLICT", "the local order is linked to another Razorpay payment", 409)
	}
	current := order.GetString("status")
	next := localStatus(payment)
	if !shouldApplyStatus(current, next) {
		return nil
	}
	order.Set("razorpay_payment_id", payment.ID)
	order.Set("payment_method", truncate(payment.Method, 64))
	order.Set("provider_status", truncate(payment.Status, 64))
	order.Set("amount_refunded", payment.AmountRefunded)
	order.Set("last_synced_at", at)
	order.Set("error", truncate(firstNonEmpty(payment.ErrorDescription, payment.ErrorCode), 4096))
	order.Set("status", next)
	switch next {
	case "captured":
		order.Set("captured_at", at)
	case "failed":
		order.Set("failed_at", at)
	}
	return nil
}

func localStatus(payment ProviderPayment) string {
	if payment.AmountRefunded >= payment.Amount && payment.Amount > 0 {
		return "refunded"
	}
	if payment.AmountRefunded > 0 {
		return "partially_refunded"
	}
	switch payment.Status {
	case "captured":
		return "captured"
	case "authorized":
		return "authorized"
	case "failed":
		return "failed"
	case "refunded":
		return "refunded"
	default:
		return "verification_pending"
	}
}

func shouldApplyStatus(current, next string) bool {
	if current == next {
		return true
	}
	if current == "refunded" {
		return false
	}
	if current == "partially_refunded" {
		return next == "refunded"
	}
	if current == "captured" {
		return next == "partially_refunded" || next == "refunded"
	}
	if next == "captured" || next == "partially_refunded" || next == "refunded" {
		return true
	}
	if current == "failed" {
		return false
	}
	return true
}

func verifyHexHMAC(secret, message, provided string) bool {
	providedBytes, err := hex.DecodeString(strings.TrimSpace(provided))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	expected := mac.Sum(nil)
	return len(providedBytes) == len(expected) && subtle.ConstantTimeCompare(providedBytes, expected) == 1
}

func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func OrderResponse(record *core.Record, keyID, displayName string) map[string]any {
	if record == nil {
		return nil
	}
	return map[string]any{
		"id": record.Id, "amountPaise": record.GetInt("amount"), "currency": record.GetString("currency"),
		"status": record.GetString("status"), "externalId": record.GetString("external_id"),
		"razorpayOrderId": record.GetString("razorpay_order_id"), "razorpayPaymentId": record.GetString("razorpay_payment_id"),
		"providerStatus": record.GetString("provider_status"), "paymentMethod": record.GetString("payment_method"),
		"amountRefunded": record.GetInt("amount_refunded"), "error": record.GetString("error"),
		"createdAt": record.GetDateTime("created_at").String(), "capturedAt": record.GetDateTime("captured_at").String(),
		"keyId": keyID, "displayName": displayName,
	}
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
