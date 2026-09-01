package relay

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/observations"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

const (
	SchemaVersion            = 1
	EventPath                = "/api/v4/relay/events"
	maxRawBodyBytes          = 64 << 10
	maxNotificationTextBytes = 32 << 10
)

type Service struct {
	DB              *storage.DB
	Payments        *payments.Service
	Now             func() time.Time
	NewID           func(prefix string) (string, error)
	NewPairingToken func() (string, error)
	PairingTTL      time.Duration
}
type EventInput struct {
	SchemaVersion   int    `json:"schema_version"`
	EventID         string `json:"event_id"`
	PackageName     string `json:"package_name"`
	PostedAtMS      int64  `json:"posted_at_ms"`
	Title           string `json:"title,omitempty"`
	Text            string `json:"text,omitempty"`
	BigText         string `json:"big_text,omitempty"`
	AmountHintPaise int64  `json:"amount_hint_paise,omitempty"`
}

type IngestResult struct {
	RelayEventID string `json:"relay_event_id"`
	Status       string `json:"status"`
	PaymentID    string `json:"payment_id,omitempty"`
	Duplicate    bool   `json:"duplicate,omitempty"`
	Transitioned bool   `json:"transitioned,omitempty"`
}

func NewService(db *storage.DB, paymentService *payments.Service) *Service {
	return &Service{
		DB: db, Payments: paymentService, Now: time.Now, NewID: randomID,
		NewPairingToken: randomPairingToken, PairingTTL: 2 * time.Minute,
	}
}
func (s *Service) IngestSigned(ctx context.Context, auth RequestAuth, rawBody []byte) (IngestResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil || s.Payments == nil {
		return IngestResult{}, errors.New("relay storage and payment service are required")
	}
	if len(rawBody) == 0 || len(rawBody) > maxRawBodyBytes {
		return IngestResult{}, relayError("RELAY_EVENT_TOO_LARGE", "relay event body is empty or too large", 400)
	}
	if strings.ToUpper(strings.TrimSpace(auth.Method)) != "POST" || auth.Path != EventPath {
		return IngestResult{}, relayError("INVALID_RELAY_ENDPOINT", "relay signature is not for the v4 event endpoint", 401)
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	device, err := verifyRequest(ctx, s.DB, auth, rawBody, now)
	if err != nil {
		return IngestResult{}, err
	}
	var input EventInput
	if err := json.Unmarshal(rawBody, &input); err != nil {
		return IngestResult{}, relayError("INVALID_RELAY_EVENT", "relay event is not valid JSON", 400)
	}
	if err := validateEventInput(&input); err != nil {
		return IngestResult{}, err
	}
	postedAt, postedReliable := sanitizePostedAt(input.PostedAtMS, now)
	result, inserted, err := s.acceptEvent(ctx, device, input, postedAt, postedReliable, now)
	if err != nil || !inserted {
		return result, err
	}
	if !postedReliable {
		postedAt = now
	}
	obs, parseErr := observations.Parse(observations.Snapshot{
		PackageName: input.PackageName,
		PostedAt:    postedAt,
		Title:       input.Title,
		Text:        input.Text,
		BigText:     input.BigText,
	})
	if parseErr != nil {
		if err := s.finishIgnored(ctx, result.RelayEventID, parseErr); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	if !postedReliable && obs.OccurredAtSource == "notification_posted_at" {
		obs.OccurredAt = now
		obs.OccurredAtSource = "server_received_at"
	}
	if obs.OccurredAt.After(now.Add(2 * time.Minute)) {
		if err := s.finishIgnored(ctx, result.RelayEventID, errors.New("payment occurrence time is implausibly in the future")); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	if !device.EnrolledAt.IsZero() && obs.OccurredAt.Before(device.EnrolledAt.Add(-2*time.Minute)) {
		if err := s.finishIgnored(ctx, result.RelayEventID, errors.New("notification predates relay enrollment")); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	matched, err := s.Payments.ApplyObservation(ctx, result.RelayEventID, obs, now)
	if err != nil {
		return IngestResult{}, err
	}
	result.Status = matched.Result
	result.PaymentID = matched.PaymentID
	result.Transitioned = matched.Transitioned
	return result, nil
}
func validateEventInput(in *EventInput) error {
	if in.SchemaVersion != SchemaVersion {
		return relayError("UNSUPPORTED_RELAY_SCHEMA", "schema_version must be 1", 400)
	}
	in.EventID = strings.ToLower(strings.TrimSpace(in.EventID))
	if len(in.EventID) != 64 {
		return relayError("INVALID_RELAY_EVENT_ID", "event_id must be a SHA-256 hex string", 400)
	}
	if _, err := hex.DecodeString(in.EventID); err != nil {
		return relayError("INVALID_RELAY_EVENT_ID", "event_id must be a SHA-256 hex string", 400)
	}
	in.PackageName = strings.TrimSpace(in.PackageName)
	if in.PackageName != observations.PaytmBusinessPackage && in.PackageName != observations.GoogleMessagesPackage {
		return relayError("UNSUPPORTED_RELAY_APP", "notification app is not allowlisted", 400)
	}
	if len(in.Title)+len(in.Text)+len(in.BigText) == 0 || len(in.Title)+len(in.Text)+len(in.BigText) > maxNotificationTextBytes {
		return relayError("RELAY_EVENT_TOO_LARGE", "notification text is empty or too large", 400)
	}
	if in.AmountHintPaise != 0 && (in.AmountHintPaise <= 0 || in.AmountHintPaise%100 == 0) {
		return relayError("INVALID_RELAY_AMOUNT_HINT", "amount hint must be a non-.00 positive amount", 400)
	}
	return nil
}

func sanitizePostedAt(ms int64, now time.Time) (time.Time, bool) {
	if ms <= 0 {
		return now.UTC(), false
	}
	posted := time.UnixMilli(ms).UTC()
	if posted.Year() < 2020 || posted.After(now.Add(5*time.Minute)) {
		return now.UTC(), false
	}
	return posted, true
}
func (s *Service) acceptEvent(ctx context.Context, device verifiedDevice, in EventInput, postedAt time.Time, postedReliable bool, now time.Time) (IngestResult, bool, error) {
	idFn := s.NewID
	if idFn == nil {
		idFn = randomID
	}
	var result IngestResult
	needsProcessing := false
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		updated, err := tx.ExecContext(ctx, `UPDATE relay_devices SET last_seen_at=? WHERE id=? AND enabled=1`, now.UnixMilli(), device.ID)
		if err != nil {
			return fmt.Errorf("refresh relay device: %w", err)
		}
		if rows, _ := updated.RowsAffected(); rows != 1 {
			return relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
		}
		existing, found, err := existingRelayEvent(ctx, tx, device.ID, in.EventID)
		if err != nil {
			return err
		}
		if found {
			result = existing
			result.Duplicate = true
			needsProcessing = existing.Status == "received"
			return nil
		}
		relayID, err := idFn("relay")
		if err != nil {
			return fmt.Errorf("generate relay event id: %w", err)
		}
		status := "received"
		errorText := any(nil)
		if postedReliable && !device.EnrolledAt.IsZero() && postedAt.Before(device.EnrolledAt.Add(-2*time.Minute)) {
			status = "ignored"
			errorText = "notification predates relay enrollment"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO relay_events(
			id,device_id,source_event_id,package_name,posted_at,received_at,amount_hint_paise,title,text,big_text,status,error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, relayID, device.ID, in.EventID, in.PackageName,
			postedAt.UnixMilli(), now.UnixMilli(), nullableAmount(in.AmountHintPaise), nullableText(in.Title),
			nullableText(in.Text), nullableText(in.BigText), status, errorText)
		if err != nil {
			return fmt.Errorf("insert relay event: %w", err)
		}
		result = IngestResult{RelayEventID: relayID, Status: status}
		needsProcessing = status == "received"
		return nil
	})
	return result, needsProcessing, err
}
func existingRelayEvent(ctx context.Context, tx *storage.ImmediateTx, deviceID, sourceEventID string) (IngestResult, bool, error) {
	var result IngestResult
	var paymentID sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT r.id,COALESCE(o.match_result,r.status),o.matched_payment_id
		FROM relay_events r LEFT JOIN payment_observations o ON o.relay_event_id=r.id
		WHERE r.device_id=? AND r.source_event_id=?`, deviceID, sourceEventID).
		Scan(&result.RelayEventID, &result.Status, &paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return IngestResult{}, false, nil
	}
	if err != nil {
		return IngestResult{}, false, fmt.Errorf("read relay event: %w", err)
	}
	result.PaymentID = paymentID.String
	return result, true, nil
}

func (s *Service) finishIgnored(ctx context.Context, relayEventID string, reason error) error {
	message := "unrecognized notification"
	if reason != nil {
		message = trimError(reason.Error(), 512)
	}
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		_, err := tx.ExecContext(ctx, `UPDATE relay_events SET status='ignored',error=? WHERE id=? AND status='received'`, message, relayEventID)
		return err
	})
}
func nullableAmount(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func trimError(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func randomID(prefix string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return prefix + "_" + strings.ToLower(encoded), nil
}
