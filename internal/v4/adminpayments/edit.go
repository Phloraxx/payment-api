package adminpayments

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

type EditInput struct {
	Name         *string
	ExternalID   *string
	Metadata     *json.RawMessage
	Status       *string
	PayerName    *string
	PayerUPIID   *string
	PaidAt       *time.Time
	InternalNote *string
}

func (s *Service) Edit(ctx context.Context, id string, input EditInput) (Payment, error) {
	if err := s.ready(); err != nil {
		return Payment{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Payment{}, ErrPaymentNotFound
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	idFn := s.NewID
	if idFn == nil {
		idFn = randomID
	}
	now := nowFn().UTC()
	var out Payment
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		current, err := getPaymentTx(ctx, tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPaymentNotFound
		}
		if err != nil {
			return fmt.Errorf("load payment for edit: %w", err)
		}
		next, changes, merchantVisible, err := applyEdit(current, input, now)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			out = current
			return nil
		}
		if current.Status != "pending" && next.Status == "pending" {
			if err := ensureReservationReopenable(ctx, tx, current, now); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE payments SET name=?,external_id=?,metadata_json=?,status=?,paid_at=?,payer_name=?,payer_upi_id=?,internal_note=? WHERE id=?`,
			next.Name, nullable(next.ExternalID), string(next.Metadata), next.Status, nullableTime(next.PaidAt), nullable(next.PayerName),
			nullable(next.PayerUPIID), nullable(next.InternalNote), next.ID); err != nil {
			return fmt.Errorf("update payment: %w", err)
		}
		historyType := operatorHistoryType(current.Status, next.Status)
		historyID, err := idFn("hist")
		if err != nil {
			return fmt.Errorf("generate history id: %w", err)
		}
		changesJSON, err := json.Marshal(changes)
		if err != nil {
			return fmt.Errorf("encode payment changes: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO payment_history(id,payment_id,type,actor,summary,changes_json,created_at) VALUES(?,?,?,?,?,?,?)`,
			historyID, next.ID, historyType, "admin", "Payment updated by operator", string(changesJSON), now.UnixMilli()); err != nil {
			return fmt.Errorf("insert operator payment history: %w", err)
		}
		if merchantVisible {
			eventType := operatorWebhookType(current.Status, next.Status)
			eventID, err := idFn("evt")
			if err != nil {
				return fmt.Errorf("generate webhook id: %w", err)
			}
			payload, err := payments.BuildEventPayloadAt(eventID, eventType, toDomainPayment(next), now)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,event_type,payment_id,payload_json,status,attempts,created_at) VALUES(?,?,?,?,'pending',0,?)`,
				eventID, eventType, next.ID, string(payload), now.UnixMilli()); err != nil {
				return fmt.Errorf("insert operator webhook: %w", err)
			}
		}
		out = next
		return nil
	})
	return out, err
}

func getPaymentTx(ctx context.Context, tx *storage.ImmediateTx, id string) (Payment, error) {
	return scanPayment(tx.QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM payments WHERE id=?`, id))
}

func ensureReservationReopenable(ctx context.Context, tx *storage.ImmediateTx, p Payment, now time.Time) error {
	if !now.Before(p.ReuseAfter) {
		return ErrUnsafeReopen
	}
	var released sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT released_at FROM amount_reservations WHERE payment_id=?`, p.ID).Scan(&released); err != nil {
		return fmt.Errorf("read amount reservation: %w", err)
	}
	if released.Valid {
		return ErrUnsafeReopen
	}
	return nil
}
func applyEdit(current Payment, input EditInput, now time.Time) (Payment, map[string]any, bool, error) {
	next := current
	changes := map[string]any{}
	merchantVisible := false
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if value == "" || utf8.RuneCountInString(value) > 120 {
			return Payment{}, nil, false, fmt.Errorf("%w: name must contain 1-120 characters", ErrInvalidEdit)
		}
		if value != next.Name {
			changes["name"] = change(next.Name, value)
			next.Name, merchantVisible = value, true
		}
	}
	if input.ExternalID != nil {
		value := strings.TrimSpace(*input.ExternalID)
		if len(value) > 255 {
			return Payment{}, nil, false, fmt.Errorf("%w: external_id is too long", ErrInvalidEdit)
		}
		if value != next.ExternalID {
			changes["external_id"] = change(next.ExternalID, value)
			next.ExternalID, merchantVisible = value, true
		}
	}
	if input.Metadata != nil {
		value, err := canonicalMetadata(*input.Metadata)
		if err != nil {
			return Payment{}, nil, false, err
		}
		if !bytes.Equal(value, next.Metadata) {
			changes["metadata"] = map[string]any{"changed": true}
			next.Metadata, merchantVisible = value, true
		}
	}
	if input.Status != nil {
		value := strings.ToLower(strings.TrimSpace(*input.Status))
		switch value {
		case "pending", "paid", "expired", "cancelled":
		default:
			return Payment{}, nil, false, fmt.Errorf("%w: invalid status %q", ErrInvalidEdit, value)
		}
		if value != next.Status {
			changes["status"] = change(next.Status, value)
			next.Status, merchantVisible = value, true
		}
	}
	if input.PayerName != nil {
		value := strings.TrimSpace(*input.PayerName)
		if utf8.RuneCountInString(value) > 255 {
			return Payment{}, nil, false, fmt.Errorf("%w: payer_name is too long", ErrInvalidEdit)
		}
		if value != next.PayerName {
			changes["payer_name"] = change(next.PayerName, value)
			next.PayerName, merchantVisible = value, true
		}
	}
	if input.PayerUPIID != nil {
		value := strings.TrimSpace(*input.PayerUPIID)
		if utf8.RuneCountInString(value) > 255 {
			return Payment{}, nil, false, fmt.Errorf("%w: payer_upi_id is too long", ErrInvalidEdit)
		}
		if value != next.PayerUPIID {
			changes["payer_upi_id"] = change(next.PayerUPIID, value)
			next.PayerUPIID, merchantVisible = value, true
		}
	}
	if input.InternalNote != nil {
		value := strings.TrimSpace(*input.InternalNote)
		if utf8.RuneCountInString(value) > 4000 {
			return Payment{}, nil, false, fmt.Errorf("%w: internal_note is too long", ErrInvalidEdit)
		}
		if value != next.InternalNote {
			changes["internal_note"] = map[string]any{"changed": true}
			next.InternalNote = value
		}
	}
	if next.Status == "paid" {
		if input.PaidAt != nil {
			value := input.PaidAt.UTC()
			if next.PaidAt == nil || !next.PaidAt.Equal(value) {
				changes["paid_at"] = change(timeJSON(next.PaidAt), value.Format(time.RFC3339Nano))
				next.PaidAt, merchantVisible = &value, true
			}
		} else if next.PaidAt == nil {
			value := now.UTC()
			changes["paid_at"] = change(nil, value.Format(time.RFC3339Nano))
			next.PaidAt, merchantVisible = &value, true
		}
	} else {
		if input.PaidAt != nil {
			return Payment{}, nil, false, fmt.Errorf("%w: paid_at requires status paid", ErrInvalidEdit)
		}
		if next.PaidAt != nil {
			changes["paid_at"] = change(next.PaidAt.UTC().Format(time.RFC3339Nano), nil)
			next.PaidAt, merchantVisible = nil, true
		}
	}
	return next, changes, merchantVisible, nil
}
func operatorHistoryType(from, to string) string {
	if from == to {
		return "payment.updated"
	}
	return operatorWebhookType(from, to)
}

func operatorWebhookType(from, to string) string {
	if from != to {
		switch to {
		case "paid":
			return "payment.paid"
		case "cancelled":
			if from == "pending" {
				return "payment.cancelled"
			}
		case "expired":
			if from == "pending" {
				return "payment.expired"
			}
		}
	}
	return "payment.updated"
}

func toDomainPayment(p Payment) payments.Payment {
	return payments.Payment{
		ID: p.ID, Name: p.Name, ExternalID: p.ExternalID, Metadata: p.Metadata,
		RequestedAmountPaise: p.RequestedAmountPaise, PayableAmountPaise: p.PayableAmountPaise,
		AdjustmentPaise: p.AdjustmentPaise, Status: p.Status, CollectionProfileID: p.CollectionProfileID,
		UPIIDSnapshot: p.UPIIDSnapshot, PayeeNameSnapshot: p.PayeeNameSnapshot,
		CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, GraceUntil: p.GraceUntil, ReuseAfter: p.ReuseAfter,
		PaidAt: p.PaidAt, PayerName: p.PayerName, PayerUPIID: p.PayerUPIID,
	}
}
func canonicalMetadata(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if len(raw) > 16<<10 {
		return nil, fmt.Errorf("%w: metadata is too large", ErrInvalidEdit)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: metadata is invalid JSON", ErrInvalidEdit)
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("%w: metadata must be a JSON object", ErrInvalidEdit)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: metadata has trailing JSON", ErrInvalidEdit)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("canonicalize metadata: %w", err)
	}
	return canonical, nil
}

func randomID(prefix string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return prefix + "_" + strings.ToLower(encoded), nil
}
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().UnixMilli()
}

func change(from, to any) map[string]any {
	return map[string]any{"from": from, "to": to}
}

func timeJSON(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
