package payments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/observations"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrRelayEventNotFound = errors.New("relay event not found")
	ErrInvalidObservation = errors.New("invalid payment observation")
)

type MatchResult struct {
	Result       string
	PaymentID    string
	Transitioned bool
	Replayed     bool
}

type matchCandidate struct {
	PaymentID     string
	Status        string
	ReservedAt    int64
	ReservedUntil int64
}

func (s *Service) ApplyObservation(ctx context.Context, relayEventID string, obs observations.Observation, receivedAt time.Time) (MatchResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return MatchResult{}, errors.New("payment storage is required")
	}
	relayEventID = strings.TrimSpace(relayEventID)
	if relayEventID == "" {
		return MatchResult{}, ErrRelayEventNotFound
	}
	if err := validateObservation(obs); err != nil {
		return MatchResult{}, err
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
	if receivedAt.IsZero() {
		receivedAt = now
	} else {
		receivedAt = receivedAt.UTC()
	}

	var result MatchResult
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		replayed, found, err := existingObservationResult(ctx, tx, relayEventID)
		if err != nil {
			return err
		}
		if found {
			result = replayed
			result.Replayed = true
			return nil
		}
		packageName, err := relayPackage(ctx, tx, relayEventID)
		if err != nil {
			return err
		}
		if expected := expectedPackage(obs.Source); expected != "" && packageName != expected {
			return fmt.Errorf("%w: source %s does not match relay package %s", ErrInvalidObservation, obs.Source, packageName)
		}

		candidates, err := matchingCandidates(ctx, tx, obs)
		if err != nil {
			return err
		}
		matchResult := "unmatched"
		matchedID := ""
		if len(candidates) > 1 {
			matchResult = "ambiguous"
		} else if len(candidates) == 1 {
			candidate := candidates[0]
			unsafe, err := reusedLowConfidenceLatest(ctx, tx, obs, candidate)
			if err != nil {
				return err
			}
			if unsafe {
				matchResult = "ambiguous"
			} else {
				matchResult = "matched"
				if candidate.Status == "paid" {
					prior, err := hasConfirmedObservation(ctx, tx, candidate.PaymentID)
					if err != nil {
						return err
					}
					if prior {
						matchResult = "corroborated"
					}
				}
				matchedID = candidate.PaymentID
				transitioned, err := applyMatchedPayment(ctx, tx, idFn, candidate, obs, now)
				if err != nil {
					return err
				}
				result.Transitioned = transitioned
			}
		}
		observationID, err := idFn("obs")
		if err != nil {
			return fmt.Errorf("generate observation id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO payment_observations(
			id,relay_event_id,source,collection_profile_id,amount_paise,payer_name,payer_upi_id,
			occurred_at,occurred_at_source,received_at,matched_payment_id,match_result)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			observationID, relayEventID, obs.Source, obs.CollectionProfileID, obs.AmountPaise,
			nullableString(obs.PayerName), nullableString(obs.PayerUPIID), obs.OccurredAt.UnixMilli(),
			obs.OccurredAtSource, receivedAt.UnixMilli(), nullableString(matchedID), matchResult); err != nil {
			return fmt.Errorf("insert payment observation: %w", err)
		}
		relayStatus := matchResult
		if relayStatus == "corroborated" {
			relayStatus = "matched"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE relay_events SET status=?,error=NULL WHERE id=?`, relayStatus, relayEventID); err != nil {
			return fmt.Errorf("update relay event status: %w", err)
		}
		result.Result = matchResult
		result.PaymentID = matchedID
		return nil
	})
	if err != nil {
		return MatchResult{}, err
	}
	return result, nil
}

func validateObservation(obs observations.Observation) error {
	if obs.AmountPaise <= 0 || obs.AmountPaise%100 == 0 || obs.OccurredAt.IsZero() {
		return fmt.Errorf("%w: amount/time", ErrInvalidObservation)
	}
	switch obs.Source {
	case "paytm_notification":
		if obs.CollectionProfileID != "paytm" {
			return fmt.Errorf("%w: Paytm source/profile mismatch", ErrInvalidObservation)
		}
	case "kotak_sms":
		if obs.CollectionProfileID != "kotak" {
			return fmt.Errorf("%w: Kotak source/profile mismatch", ErrInvalidObservation)
		}
	case observations.GenericNotificationSource, observations.GenericMessageSource:
		if strings.TrimSpace(obs.CollectionProfileID) == "" {
			return fmt.Errorf("%w: generic source requires resolved collection profile", ErrInvalidObservation)
		}
	default:
		return fmt.Errorf("%w: unsupported source %q", ErrInvalidObservation, obs.Source)
	}
	switch obs.OccurredAtSource {
	case "notification_text", "notification_posted_at", "server_received_at":
	default:
		return fmt.Errorf("%w: unsupported occurred_at_source %q", ErrInvalidObservation, obs.OccurredAtSource)
	}
	return nil
}

func expectedPackage(source string) string {
	if source == "paytm_notification" {
		return observations.PaytmBusinessPackage
	}
	if source == "kotak_sms" {
		return observations.GoogleMessagesPackage
	}
	return ""
}
func existingObservationResult(ctx context.Context, tx *storage.ImmediateTx, relayEventID string) (MatchResult, bool, error) {
	var matchResult string
	var paymentID sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT match_result,matched_payment_id FROM payment_observations WHERE relay_event_id=?`, relayEventID).Scan(&matchResult, &paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return MatchResult{}, false, nil
	}
	if err != nil {
		return MatchResult{}, false, fmt.Errorf("read existing observation: %w", err)
	}
	return MatchResult{Result: matchResult, PaymentID: paymentID.String}, true, nil
}

func relayPackage(ctx context.Context, tx *storage.ImmediateTx, relayEventID string) (string, error) {
	var packageName string
	err := tx.QueryRowContext(ctx, `SELECT package_name FROM relay_events WHERE id=?`, relayEventID).Scan(&packageName)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRelayEventNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read relay event: %w", err)
	}
	return packageName, nil
}

func matchingCandidates(ctx context.Context, tx *storage.ImmediateTx, obs observations.Observation) ([]matchCandidate, error) {
	occurred := obs.OccurredAt.UnixMilli()
	rows, err := tx.QueryContext(ctx, `SELECT p.id,p.status,r.reserved_at,r.reserved_until
		FROM amount_reservations r JOIN payments p ON p.id=r.payment_id
		WHERE r.collection_profile_id=? AND r.payable_amount_paise=? AND p.created_at<=? AND r.reserved_until>=?
		ORDER BY r.reserved_at`, obs.CollectionProfileID, obs.AmountPaise, occurred, occurred)
	if err != nil {
		return nil, fmt.Errorf("find matching reservations: %w", err)
	}
	var raw []matchCandidate
	for rows.Next() {
		var c matchCandidate
		if err := rows.Scan(&c.PaymentID, &c.Status, &c.ReservedAt, &c.ReservedUntil); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, c)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate matching reservations: %w", err)
	}

	candidates := make([]matchCandidate, 0, len(raw))
	for _, c := range raw {
		if c.Status == "cancelled" {
			allowed, err := occurredBeforeCancellation(ctx, tx, c.PaymentID, occurred)
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

func occurredBeforeCancellation(ctx context.Context, tx *storage.ImmediateTx, paymentID string, occurredAt int64) (bool, error) {
	var cancelledAt int64
	err := tx.QueryRowContext(ctx, `SELECT created_at FROM payment_history WHERE payment_id=? AND type='payment.cancelled' ORDER BY created_at DESC LIMIT 1`, paymentID).Scan(&cancelledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read cancellation time: %w", err)
	}
	return occurredAt <= cancelledAt, nil
}

func hasConfirmedObservation(ctx context.Context, tx *storage.ImmediateTx, paymentID string) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM payment_observations WHERE matched_payment_id=? AND match_result IN ('matched','corroborated'))`, paymentID).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("read prior payment observations: %w", err)
	}
	return found == 1, nil
}

func reusedLowConfidenceLatest(ctx context.Context, tx *storage.ImmediateTx, obs observations.Observation, candidate matchCandidate) (bool, error) {
	lowConfidence := obs.OccurredAtSource == "server_received_at" || obs.OccurredAtSource == "notification_posted_at"
	if !lowConfidence {
		return false, nil
	}
	var count int
	var latest sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*),MAX(reserved_at) FROM amount_reservations WHERE collection_profile_id=? AND payable_amount_paise=?`, obs.CollectionProfileID, obs.AmountPaise).Scan(&count, &latest); err != nil {
		return false, fmt.Errorf("read reservation reuse history: %w", err)
	}
	return count > 1 && latest.Valid && candidate.ReservedAt == latest.Int64, nil
}

func applyMatchedPayment(ctx context.Context, tx *storage.ImmediateTx, idFn func(string) (string, error), candidate matchCandidate, obs observations.Observation, transitionAt time.Time) (bool, error) {
	if candidate.Status == "paid" {
		if _, err := tx.ExecContext(ctx, `UPDATE payments SET
			payer_name=CASE WHEN COALESCE(payer_name,'')='' AND ?<>'' THEN ? ELSE payer_name END,
			payer_upi_id=CASE WHEN COALESCE(payer_upi_id,'')='' AND ?<>'' THEN ? ELSE payer_upi_id END
			WHERE id=?`, obs.PayerName, obs.PayerName, obs.PayerUPIID, obs.PayerUPIID, candidate.PaymentID); err != nil {
			return false, fmt.Errorf("enrich already-paid payment: %w", err)
		}
		return false, nil
	}

	if _, err := tx.ExecContext(ctx, `UPDATE payments SET status='paid',paid_at=?,
		payer_name=CASE WHEN ?<>'' THEN ? ELSE payer_name END,
		payer_upi_id=CASE WHEN ?<>'' THEN ? ELSE payer_upi_id END
		WHERE id=?`, obs.OccurredAt.UnixMilli(), obs.PayerName, obs.PayerName, obs.PayerUPIID, obs.PayerUPIID, candidate.PaymentID); err != nil {
		return false, fmt.Errorf("mark payment paid: %w", err)
	}
	payment, err := loadPayment(ctx, tx, candidate.PaymentID)
	if err != nil {
		return false, fmt.Errorf("reload paid payment: %w", err)
	}
	changes, err := json.Marshal(map[string]any{
		"status": map[string]string{"from": candidate.Status, "to": "paid"},
		"source": obs.Source,
	})
	if err != nil {
		return false, err
	}
	if err := appendTransition(ctx, tx, idFn, "payment.paid", payment, transitionAt, string(changes)); err != nil {
		return false, err
	}
	return true, nil
}
