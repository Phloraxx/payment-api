package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrPaymentTerminal = errors.New("payment is already terminal")
)

type GetResult struct {
	Payment Payment
	UPIURI  string
}

func (s *Service) Get(ctx context.Context, id string) (GetResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return GetResult{}, errors.New("payment storage is required")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return GetResult{}, ErrPaymentNotFound
	}
	payment, err := loadPayment(ctx, s.DB.SQL, id)
	if errors.Is(err, sql.ErrNoRows) {
		return GetResult{}, ErrPaymentNotFound
	}
	if err != nil {
		return GetResult{}, fmt.Errorf("get payment: %w", err)
	}
	return GetResult{Payment: payment, UPIURI: buildUPIURI(payment.UPIIDSnapshot, payment.PayeeNameSnapshot, payment.PayableAmountPaise)}, nil
}
func (s *Service) Cancel(ctx context.Context, id string) (Payment, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return Payment{}, errors.New("payment storage is required")
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
		payment, err := loadPayment(ctx, tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPaymentNotFound
		}
		if err != nil {
			return fmt.Errorf("load payment for cancel: %w", err)
		}
		switch payment.Status {
		case "cancelled":
			out = payment
			return nil
		case "paid", "expired":
			return ErrPaymentTerminal
		}
		if _, err := tx.ExecContext(ctx, `UPDATE payments SET status='cancelled' WHERE id=? AND status='pending'`, id); err != nil {
			return fmt.Errorf("cancel payment: %w", err)
		}
		payment.Status = "cancelled"
		if err := appendTransition(ctx, tx, idFn, "payment.cancelled", payment, now, `{"status":{"from":"pending","to":"cancelled"}}`); err != nil {
			return err
		}
		out = payment
		return nil
	})
	return out, err
}
func (s *Service) ExpireDue(ctx context.Context, limit int) (int, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return 0, errors.New("payment storage is required")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
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

	expired := 0
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM payments WHERE status='pending' AND grace_until <= ? ORDER BY grace_until,id LIMIT ?`, now.UnixMilli(), limit)
		if err != nil {
			return fmt.Errorf("list due payments: %w", err)
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan due payment: %w", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate due payments: %w", err)
		}

		for _, id := range ids {
			result, err := tx.ExecContext(ctx, `UPDATE payments SET status='expired' WHERE id=? AND status='pending' AND grace_until <= ?`, id, now.UnixMilli())
			if err != nil {
				return fmt.Errorf("expire payment %s: %w", id, err)
			}
			changed, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if changed == 0 {
				continue
			}
			payment, err := loadPayment(ctx, tx, id)
			if err != nil {
				return fmt.Errorf("reload expired payment %s: %w", id, err)
			}
			if err := appendTransition(ctx, tx, idFn, "payment.expired", payment, now, `{"status":{"from":"pending","to":"expired"}}`); err != nil {
				return err
			}
			expired++
		}
		return nil
	})
	return expired, err
}
func appendTransition(ctx context.Context, tx *storage.ImmediateTx, idFn func(string) (string, error), eventType string, payment Payment, now time.Time, changesJSON string) error {
	historyID, err := idFn("hist")
	if err != nil {
		return fmt.Errorf("generate history id: %w", err)
	}
	eventID, err := idFn("evt")
	if err != nil {
		return fmt.Errorf("generate webhook id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO payment_history(id,payment_id,type,actor,summary,changes_json,created_at)
		VALUES(?,?,?,?,?,?,?)`, historyID, payment.ID, eventType, "system", transitionSummary(eventType), changesJSON, now.UnixMilli()); err != nil {
		return fmt.Errorf("insert payment history: %w", err)
	}
	payload, err := paymentEventPayloadAt(eventID, eventType, payment, now)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,event_type,payment_id,payload_json,status,attempts,created_at)
		VALUES(?,?,?,?,'pending',0,?)`, eventID, eventType, payment.ID, string(payload), now.UnixMilli()); err != nil {
		return fmt.Errorf("insert webhook outbox: %w", err)
	}
	return nil
}

func transitionSummary(eventType string) string {
	switch eventType {
	case "payment.cancelled":
		return "Payment cancelled"
	case "payment.expired":
		return "Payment expired"
	case "payment.paid":
		return "Payment paid"
	default:
		return eventType
	}
}
