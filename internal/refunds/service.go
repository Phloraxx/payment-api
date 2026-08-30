package refunds

import (
	"context"
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
	"github.com/pocketbase/pocketbase/core"
)

type WebhookScheduler interface {
	ScheduleRefundPayment(uow store.UnitOfWork, event string, payment *domain.Payment, refund *domain.Refund, at time.Time) error
	Wake()
}

type Service struct {
	Store    store.Database
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
	return &Service{Store: store.NewPocketBase(app), Audit: auditService, Webhooks: webhooks, Now: time.Now}
}

func (s *Service) Request(input RequestInput) (*domain.Refund, bool, error) {
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
	var result *domain.Refund
	var replayed, queued bool
	err = s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		repo := uow.Refunds()
		if input.IdempotencyKey != "" {
			existing, findErr := repo.FindByIdempotencyKey(input.IdempotencyKey)
			if findErr == nil {
				existingMetadata, metadataErr := normalizeMetadata(existing.Metadata)
				if metadataErr != nil {
					return metadataErr
				}
				if existing.PaymentID != input.PaymentID || existing.AmountPaise != input.AmountPaise || existing.Reason != input.Reason || existing.ExternalID != input.ExternalID || !reflect.DeepEqual(existingMetadata, metadata) {
					return domain.New("REFUND_IDEMPOTENCY_CONFLICT", "the refund idempotency key was already used with different parameters", 409)
				}
				result, replayed = existing, true
				return nil
			}
			if !errors.Is(findErr, sql.ErrNoRows) {
				return findErr
			}
		}
		payment, err := uow.Payments().Get(input.PaymentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.PaymentNotFound()
			}
			return err
		}
		if payment.Status != domain.StatusPaid && payment.Status != domain.StatusLate {
			return domain.New("PAYMENT_NOT_REFUNDABLE", "only paid or late payments can have refund records", 409)
		}
		reserved, err := repo.ReservedAmount(payment.ID)
		if err != nil {
			return err
		}
		if input.AmountPaise > payment.PayablePaise-reserved {
			return domain.New("REFUND_AMOUNT_EXCEEDS_AVAILABLE", "refund amount exceeds the remaining refundable payment amount", 409)
		}
		refund := &domain.Refund{PaymentID: payment.ID, AmountPaise: input.AmountPaise, Status: "requested", Reason: input.Reason, ExternalID: input.ExternalID, IdempotencyKey: input.IdempotencyKey, Metadata: metadata, RequestedBy: input.Actor.ID, RequestedAt: now}
		if err := repo.Create(refund); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(uow, audit.Entry{Action: "refund.requested", Actor: input.Actor, EntityType: "refund", EntityID: refund.ID, Summary: "Operator recorded a refund request", Details: map[string]any{"paymentId": payment.ID, "amountPaise": input.AmountPaise, "reason": input.Reason}, OccurredAt: now}); err != nil {
				return err
			}
		}
		if s.Webhooks != nil {
			if err := s.Webhooks.ScheduleRefundPayment(uow, "refund.requested", payment, refund, now); err != nil {
				return err
			}
			queued = true
		}
		result = refund
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

func (s *Service) Update(input UpdateInput) (*domain.Refund, error) {
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
	var result *domain.Refund
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		repo := uow.Refunds()
		refund, err := repo.Get(input.RefundID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.New("REFUND_NOT_FOUND", "refund not found", 404)
			}
			return err
		}
		current := refund.Status
		if current == input.Status {
			if input.Reference != "" && input.Reference != refund.Reference {
				return domain.New("REFUND_REFERENCE_CONFLICT", "same-status retry supplied a different refund reference", 409)
			}
			result = refund
			return nil
		}
		if !validTransition(current, input.Status) {
			return domain.New("INVALID_REFUND_TRANSITION", fmt.Sprintf("refund cannot transition from %s to %s", current, input.Status), 409)
		}
		payment, err := uow.Payments().Get(refund.PaymentID)
		if err != nil {
			return err
		}
		if !statusReservesFunds(current) && statusReservesFunds(input.Status) {
			reserved, err := repo.ReservedAmount(refund.PaymentID)
			if err != nil {
				return err
			}
			if refund.AmountPaise > payment.PayablePaise-reserved {
				return domain.New("REFUND_AMOUNT_EXCEEDS_AVAILABLE", "refund amount exceeds the remaining refundable payment amount", 409)
			}
		}
		if input.Reference != "" {
			existing, findErr := repo.FindByReference(input.Reference)
			if findErr == nil && existing.ID != refund.ID {
				return domain.New("REFUND_REFERENCE_CONFLICT", "refund reference is already assigned", 409)
			}
			if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
				return findErr
			}
		}
		refund.Status = input.Status
		if input.Reference != "" {
			refund.Reference = input.Reference
		}
		if input.Status == "completed" {
			refund.CompletedAt = now
		}
		if err := repo.Save(refund); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(uow, audit.Entry{Action: "refund." + input.Status, Actor: input.Actor, EntityType: "refund", EntityID: refund.ID, Summary: "Operator updated a refund lifecycle record", Details: map[string]any{"from": current, "to": input.Status, "reference": input.Reference, "note": input.Note}, OccurredAt: now}); err != nil {
				return err
			}
		}
		if s.Webhooks != nil {
			if err := s.Webhooks.ScheduleRefundPayment(uow, "refund."+input.Status, payment, refund, now); err != nil {
				return err
			}
			queued = true
		}
		result = refund
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
