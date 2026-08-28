package refunds

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type WebhookScheduler interface {
	ScheduleRefund(app core.App, event string, payment, refund *core.Record, at time.Time) error
	Wake()
}

type Service struct {
	App      core.App
	Audit    *audit.Service
	Webhooks WebhookScheduler
	Now      func() time.Time
}

type RequestInput struct {
	PaymentID      string
	AmountPaise    int64
	Reason         string
	ExternalID     string
	IdempotencyKey string
	Metadata       any
	Actor          audit.Actor
}

type UpdateInput struct {
	RefundID  string
	Status    string
	Reference string
	Note      string
	Actor     audit.Actor
}

func NewService(app core.App, auditService *audit.Service, webhooks WebhookScheduler) *Service {
	return &Service{App: app, Audit: auditService, Webhooks: webhooks, Now: time.Now}
}

func (s *Service) Request(input RequestInput) (*core.Record, bool, error) {
	input.PaymentID = strings.TrimSpace(input.PaymentID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.PaymentID == "" || input.AmountPaise <= 0 {
		return nil, false, domain.New("INVALID_REFUND", "paymentId and a positive refund amount are required", 400)
	}
	if len(input.Reason) > 4096 || len(input.ExternalID) > 255 || len(input.IdempotencyKey) > 255 {
		return nil, false, domain.New("INVALID_REFUND", "refund fields exceed storage limits", 400)
	}
	metadata, err := normalizeMetadata(input.Metadata)
	if err != nil {
		return nil, false, domain.New("INVALID_REFUND_METADATA", "refund metadata must be valid JSON no larger than 1 MiB", 400)
	}
	now := s.now()
	var result *core.Record
	var replayed bool
	var queued bool
	err = s.App.RunInTransaction(func(tx core.App) error {
		if input.IdempotencyKey != "" {
			existing, err := tx.FindFirstRecordByData("refunds", "idempotency_key", input.IdempotencyKey)
			if err == nil {
				existingMetadata, metadataErr := normalizeMetadata(existing.Get("metadata"))
				if metadataErr != nil {
					return metadataErr
				}
				if existing.GetString("payment") != input.PaymentID || int64(existing.GetInt("amount")) != input.AmountPaise || existing.GetString("reason") != input.Reason || existing.GetString("external_id") != input.ExternalID || !reflect.DeepEqual(existingMetadata, metadata) {
					return domain.New("REFUND_IDEMPOTENCY_CONFLICT", "the refund idempotency key was already used with different parameters", 409)
				}
				result = existing.Clone()
				replayed = true
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		payment, err := tx.FindRecordById("payments", input.PaymentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.PaymentNotFound()
			}
			return err
		}
		status := payment.GetString("status")
		if status != "paid" && status != "late" {
			return domain.New("PAYMENT_NOT_REFUNDABLE", "only paid or late payments can have refund records", 409)
		}
		paidAmount := int64(payment.GetInt("payable_amount"))
		reserved, err := reservedRefundAmount(tx, payment.Id)
		if err != nil {
			return err
		}
		if input.AmountPaise > paidAmount-reserved {
			return domain.New("REFUND_AMOUNT_EXCEEDS_AVAILABLE", "refund amount exceeds the remaining refundable payment amount", 409)
		}
		collection, err := tx.FindCollectionByNameOrId("refunds")
		if err != nil {
			return err
		}
		record := core.NewRecord(collection)
		record.Set("payment", payment.Id)
		record.Set("amount", input.AmountPaise)
		record.Set("status", "requested")
		record.Set("reason", input.Reason)
		record.Set("external_id", input.ExternalID)
		record.Set("idempotency_key", input.IdempotencyKey)
		record.Set("metadata", metadata)
		record.Set("requested_by", input.Actor.ID)
		record.Set("requested_at", now)
		if err := tx.Save(record); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(store.NewPocketBaseUnit(tx), audit.Entry{
				Action: "refund.requested", Actor: input.Actor, EntityType: "refund", EntityID: record.Id,
				Summary: "Operator recorded a refund request", Details: map[string]any{"paymentId": payment.Id, "amountPaise": input.AmountPaise, "reason": input.Reason}, OccurredAt: now,
			}); err != nil {
				return err
			}
		}
		if s.Webhooks != nil {
			if err := s.Webhooks.ScheduleRefund(tx, "refund.requested", payment, record, now); err != nil {
				return err
			}
			queued = true
		}
		result = record.Clone()
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if queued && s.Webhooks != nil {
		s.Webhooks.Wake()
	}
	return result, replayed, nil
}

func (s *Service) Update(input UpdateInput) (*core.Record, error) {
	input.RefundID = strings.TrimSpace(input.RefundID)
	input.Status = strings.TrimSpace(input.Status)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Note = strings.TrimSpace(input.Note)
	if input.RefundID == "" || input.Status == "" || len(input.Reference) > 255 || len(input.Note) > 4096 {
		return nil, domain.New("INVALID_REFUND_UPDATE", "invalid refund update", 400)
	}
	if input.Status == "completed" && input.Reference == "" {
		return nil, domain.New("REFUND_REFERENCE_REQUIRED", "a bank refund reference is required before marking a refund completed", 400)
	}
	now := s.now()
	var result *core.Record
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		refund, err := tx.FindRecordById("refunds", input.RefundID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.New("REFUND_NOT_FOUND", "refund not found", 404)
			}
			return err
		}
		current := refund.GetString("status")
		if current == input.Status {
			if input.Reference != "" && input.Reference != refund.GetString("reference") {
				return domain.New("REFUND_REFERENCE_CONFLICT", "same-status retry supplied a different refund reference", 409)
			}
			result = refund.Clone()
			return nil
		}
		if !validTransition(current, input.Status) {
			return domain.New("INVALID_REFUND_TRANSITION", fmt.Sprintf("refund cannot transition from %s to %s", current, input.Status), 409)
		}
		if !statusReservesFunds(current) && statusReservesFunds(input.Status) {
			paymentID := refund.GetString("payment")
			payment, err := tx.FindRecordById("payments", paymentID)
			if err != nil {
				return err
			}
			reserved, err := reservedRefundAmount(tx, paymentID)
			if err != nil {
				return err
			}
			available := int64(payment.GetInt("payable_amount")) - reserved
			if int64(refund.GetInt("amount")) > available {
				return domain.New("REFUND_AMOUNT_EXCEEDS_AVAILABLE", "refund amount exceeds the remaining refundable payment amount", 409)
			}
		}
		if input.Reference != "" {
			existing, findErr := tx.FindFirstRecordByData("refunds", "reference", input.Reference)
			if findErr == nil && existing.Id != refund.Id {
				return domain.New("REFUND_REFERENCE_CONFLICT", "refund reference is already assigned", 409)
			}
			if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
				return findErr
			}
		}
		refund.Set("status", input.Status)
		if input.Reference != "" {
			refund.Set("reference", input.Reference)
		}
		if input.Status == "completed" {
			refund.Set("completed_at", now)
		}
		if err := tx.Save(refund); err != nil {
			return err
		}
		payment, err := tx.FindRecordById("payments", refund.GetString("payment"))
		if err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(store.NewPocketBaseUnit(tx), audit.Entry{
				Action: "refund." + input.Status, Actor: input.Actor, EntityType: "refund", EntityID: refund.Id,
				Summary: "Operator updated a refund lifecycle record", Details: map[string]any{"from": current, "to": input.Status, "reference": input.Reference, "note": input.Note}, OccurredAt: now,
			}); err != nil {
				return err
			}
		}
		if s.Webhooks != nil {
			event := "refund." + input.Status
			if err := s.Webhooks.ScheduleRefund(tx, event, payment, refund, now); err != nil {
				return err
			}
			queued = true
		}
		result = refund.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if queued && s.Webhooks != nil {
		s.Webhooks.Wake()
	}
	return result, nil
}

func reservedRefundAmount(app core.App, paymentID string) (int64, error) {
	records, err := app.FindRecordsByFilter("refunds", "payment = {:payment} && status != 'cancelled' && status != 'failed'", "created", 0, 0, dbx.Params{"payment": paymentID})
	if err != nil {
		return 0, err
	}
	var total int64
	for _, record := range records {
		total += int64(record.GetInt("amount"))
	}
	return total, nil
}

func statusReservesFunds(status string) bool {
	switch status {
	case "requested", "processing", "completed":
		return true
	default:
		return false
	}
}

func validTransition(from, to string) bool {
	switch from {
	case "requested":
		return to == "processing" || to == "completed" || to == "failed" || to == "cancelled"
	case "processing":
		return to == "completed" || to == "failed" || to == "cancelled"
	case "failed":
		return to == "processing" || to == "cancelled"
	default:
		return false
	}
}

func normalizeMetadata(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 1<<20 {
		return nil, errors.New("invalid metadata")
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
