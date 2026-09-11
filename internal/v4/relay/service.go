package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
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
	EventBatchPath           = "/api/v4/relay/events/batch"
	maxRawBodyBytes          = 64 << 10
	maxBatchRawBodyBytes     = 1 << 20
	maxBatchEvents           = 50
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
	EventID      string `json:"event_id,omitempty"`
	RelayEventID string `json:"relay_event_id"`
	Status       string `json:"status"`
	PaymentID    string `json:"payment_id,omitempty"`
	Duplicate    bool   `json:"duplicate,omitempty"`
	Transitioned bool   `json:"transitioned,omitempty"`
}

type BatchInput struct {
	SchemaVersion int               `json:"schema_version"`
	Events        []json.RawMessage `json:"events"`
}

type BatchIngestResult struct {
	Results []IngestResult `json:"results"`
}

func NewService(db *storage.DB, paymentService *payments.Service) *Service {
	return &Service{
		DB: db, Payments: paymentService, Now: time.Now, NewID: randomID,
		NewPairingToken: randomPairingToken, PairingTTL: 2 * time.Minute,
	}
}

// RedactRawEvents removes retained notification bodies while keeping the
// immutable event identity and normalized observation available for audit.
func (s *Service) RedactRawEvents(ctx context.Context, before time.Time, limit int) (int64, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return 0, errors.New("relay storage is required")
	}
	if before.IsZero() {
		return 0, errors.New("raw event retention boundary is required")
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}

	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT id,package_name,posted_at,amount_hint_paise,
		title,text,big_text,payload_hash
		FROM relay_events
		WHERE received_at < ?
			AND (title IS NOT NULL OR text IS NOT NULL OR big_text IS NOT NULL)
		ORDER BY received_at,id LIMIT ?`, before.UTC().UnixMilli(), limit)
	if err != nil {
		return 0, fmt.Errorf("select relay event bodies for redaction: %w", err)
	}
	type retainedEvent struct {
		id, packageName      string
		postedAt             int64
		amountHint           sql.NullInt64
		title, text, bigText sql.NullString
		payloadHash          []byte
	}
	events := make([]retainedEvent, 0, limit)
	for rows.Next() {
		var event retainedEvent
		if err := rows.Scan(&event.id, &event.packageName, &event.postedAt, &event.amountHint,
			&event.title, &event.text, &event.bigText, &event.payloadHash); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan relay event body for redaction: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate relay event bodies for redaction: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("close relay event bodies for redaction: %w", err)
	}

	var redacted int64
	err = s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		for _, event := range events {
			payloadHash := event.payloadHash
			if len(payloadHash) == 0 {
				// Legacy rows predate payload hashing. Preserve a deterministic
				// retained-field fingerprint before clearing their raw bodies.
				payloadHash = legacyPayloadFingerprint(event.packageName, event.postedAt,
					event.amountHint, event.title, event.text, event.bigText)
			}
			result, err := tx.ExecContext(ctx, `UPDATE relay_events
				SET title=NULL,text=NULL,big_text=NULL,payload_hash=?,
					status=CASE WHEN status='received' THEN 'ignored' ELSE status END,
					error=CASE WHEN status='received' THEN 'raw notification expired before processing' ELSE error END
				WHERE id=? AND received_at < ?
					AND (title IS NOT NULL OR text IS NOT NULL OR big_text IS NOT NULL)`,
				payloadHash, event.id, before.UTC().UnixMilli())
			if err != nil {
				return fmt.Errorf("redact relay event %s: %w", event.id, err)
			}
			count, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("count redacted relay event %s: %w", event.id, err)
			}
			redacted += count
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return redacted, nil
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
	return s.ingestVerifiedEvent(ctx, device, input, rawBody, now)
}

func (s *Service) IngestBatchSigned(ctx context.Context, auth RequestAuth, rawBody []byte) (BatchIngestResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil || s.Payments == nil {
		return BatchIngestResult{}, errors.New("relay storage and payment service are required")
	}
	if len(rawBody) == 0 || len(rawBody) > maxBatchRawBodyBytes {
		return BatchIngestResult{}, relayError("RELAY_BATCH_TOO_LARGE", "relay batch body is empty or too large", 400)
	}
	if strings.ToUpper(strings.TrimSpace(auth.Method)) != "POST" || auth.Path != EventBatchPath {
		return BatchIngestResult{}, relayError("INVALID_RELAY_ENDPOINT", "relay signature is not for the v4 batch endpoint", 401)
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	device, err := verifyRequest(ctx, s.DB, auth, rawBody, now)
	if err != nil {
		return BatchIngestResult{}, err
	}
	var batch BatchInput
	if err := json.Unmarshal(rawBody, &batch); err != nil {
		return BatchIngestResult{}, relayError("INVALID_RELAY_BATCH", "relay batch is not valid JSON", 400)
	}
	if batch.SchemaVersion != SchemaVersion {
		return BatchIngestResult{}, relayError("UNSUPPORTED_RELAY_SCHEMA", "schema_version must be 1", 400)
	}
	if len(batch.Events) == 0 || len(batch.Events) > maxBatchEvents {
		return BatchIngestResult{}, relayError("INVALID_RELAY_BATCH", "relay batch must contain between 1 and 50 events", 400)
	}

	inputs := make([]EventInput, len(batch.Events))
	for i, rawEvent := range batch.Events {
		if len(rawEvent) == 0 || len(rawEvent) > maxRawBodyBytes {
			return BatchIngestResult{}, relayError("RELAY_EVENT_TOO_LARGE", "relay batch contains an empty or oversized event", 400)
		}
		if err := json.Unmarshal(rawEvent, &inputs[i]); err != nil {
			return BatchIngestResult{}, relayError("INVALID_RELAY_EVENT", "relay batch contains invalid event JSON", 400)
		}
		if err := validateEventInput(&inputs[i]); err != nil {
			return BatchIngestResult{}, err
		}
	}

	out := BatchIngestResult{Results: make([]IngestResult, 0, len(inputs))}
	for i, input := range inputs {
		result, err := s.ingestVerifiedEvent(ctx, device, input, batch.Events[i], now)
		if err != nil {
			// Earlier items may already be durable. Retrying the entire signed batch
			// is safe because source event IDs are idempotent per relay device.
			return BatchIngestResult{}, err
		}
		out.Results = append(out.Results, result)
	}
	return out, nil
}

func (s *Service) ingestVerifiedEvent(ctx context.Context, device verifiedDevice, input EventInput, rawEvent []byte, now time.Time) (IngestResult, error) {
	payloadHash := sha256.Sum256(rawEvent)
	postedAt, postedReliable := sanitizePostedAt(input.PostedAtMS, now)
	result, inserted, err := s.acceptEvent(ctx, device, input, postedAt, postedReliable, now, payloadHash[:])
	result.EventID = input.EventID
	if err != nil || !inserted {
		return result, err
	}
	if !postedReliable {
		if err := s.finishIgnored(ctx, result.RelayEventID, errors.New("notification posting time is missing or invalid")); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	obs, parseErr := observations.Parse(observations.Snapshot{
		PackageName: input.PackageName,
		PostedAt:    postedAt,
		Title:       input.Title,
		Text:        input.Text,
		BigText:     input.BigText,
	})
	if parseErr != nil {
		if errors.Is(parseErr, observations.ErrAmbiguousAmount) {
			status, err := s.finishAmbiguous(ctx, result.RelayEventID, parseErr)
			if err != nil {
				return IngestResult{}, err
			}
			result.Status = status
			return result, nil
		}
		if err := s.finishIgnored(ctx, result.RelayEventID, parseErr); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	if obs.OccurredAt.After(now.Add(2 * time.Minute)) {
		if err := s.finishIgnored(ctx, result.RelayEventID, errors.New("payment occurrence time is implausibly in the future")); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	if !device.EnrolledAt.IsZero() && obs.OccurredAt.Before(device.EnrolledAt) {
		if err := s.finishIgnored(ctx, result.RelayEventID, errors.New("notification predates relay enrollment")); err != nil {
			return IngestResult{}, err
		}
		result.Status = "ignored"
		return result, nil
	}
	matched, err := s.Payments.ApplyObservationForRelay(ctx, result.RelayEventID, obs, now, device.ID, device.EnrolledAt)
	if errors.Is(err, payments.ErrRelayEventNotFound) {
		if finishErr := s.finishIgnored(ctx, result.RelayEventID, errors.New("relay event was no longer authorized for processing")); finishErr != nil {
			return IngestResult{}, finishErr
		}
		result.Status = "ignored"
		return result, nil
	}
	if errors.Is(err, payments.ErrObservationAmbiguous) {
		status, finishErr := s.finishAmbiguous(ctx, result.RelayEventID, err)
		if finishErr != nil {
			return IngestResult{}, finishErr
		}
		result.Status = status
		return result, nil
	}
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
	if in.PackageName == "" || len(in.PackageName) > 255 {
		return relayError("INVALID_RELAY_APP", "notification package name is required and must be at most 255 characters", 400)
	}
	if blockedRelayPackage(in.PackageName) {
		return relayError("UNSUPPORTED_RELAY_APP", "this notification source is no longer accepted", 400)
	}
	if len(in.Title)+len(in.Text)+len(in.BigText) == 0 || len(in.Title)+len(in.Text)+len(in.BigText) > maxNotificationTextBytes {
		return relayError("RELAY_EVENT_TOO_LARGE", "notification text is empty or too large", 400)
	}
	if in.AmountHintPaise != 0 && (in.AmountHintPaise <= 0 || in.AmountHintPaise%100 == 0) {
		return relayError("INVALID_RELAY_AMOUNT_HINT", "amount hint must be a non-.00 positive amount", 400)
	}
	return nil
}

func blockedRelayPackage(packageName string) bool {
	switch strings.ToLower(strings.TrimSpace(packageName)) {
	case observations.GoogleMessagesPackage, observations.GmailPackage:
		return true
	default:
		return false
	}
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
func (s *Service) acceptEvent(ctx context.Context, device verifiedDevice, in EventInput, postedAt time.Time, postedReliable bool, now time.Time, payloadHash []byte) (IngestResult, bool, error) {
	idFn := s.NewID
	if idFn == nil {
		idFn = randomID
	}
	var result IngestResult
	needsProcessing := false
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		updated, err := tx.ExecContext(ctx, `UPDATE relay_devices SET last_seen_at=? WHERE id=? AND enabled=1 AND enrolled_at=?`, now.UnixMilli(), device.ID, device.EnrolledAt.UnixMilli())
		if err != nil {
			return fmt.Errorf("refresh relay device: %w", err)
		}
		if rows, _ := updated.RowsAffected(); rows != 1 {
			return relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
		}
		existing, found, same, err := existingRelayEvent(ctx, tx, device.ID, in.EventID, in, postedAt, postedReliable, payloadHash)
		if err != nil {
			return err
		}
		if found {
			if !same {
				return relayError("RELAY_EVENT_ID_CONFLICT", "source event ID was already used with different notification data", 409)
			}
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
		if postedReliable && !device.EnrolledAt.IsZero() && postedAt.Before(device.EnrolledAt) {
			status = "ignored"
			errorText = "notification predates relay enrollment"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO relay_events(
			id,device_id,source_event_id,package_name,posted_at,received_at,amount_hint_paise,title,text,big_text,payload_hash,status,error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, relayID, device.ID, in.EventID, in.PackageName,
			postedAt.UnixMilli(), now.UnixMilli(), nullableAmount(in.AmountHintPaise), nullableText(in.Title),
			nullableText(in.Text), nullableText(in.BigText), payloadHash, status, errorText)
		if err != nil {
			return fmt.Errorf("insert relay event: %w", err)
		}
		result = IngestResult{RelayEventID: relayID, Status: status}
		needsProcessing = status == "received"
		return nil
	})
	return result, needsProcessing, err
}
func existingRelayEvent(ctx context.Context, tx *storage.ImmediateTx, deviceID, sourceEventID string, in EventInput, postedAt time.Time, postedReliable bool, payloadHash []byte) (IngestResult, bool, bool, error) {
	var result IngestResult
	var paymentID sql.NullString
	var packageName string
	var storedPostedAt int64
	var amountHint sql.NullInt64
	var title, text, bigText sql.NullString
	var storedHash []byte
	err := tx.QueryRowContext(ctx, `SELECT r.id,COALESCE(o.match_result,r.status),o.matched_payment_id,
		r.package_name,r.posted_at,r.amount_hint_paise,r.title,r.text,r.big_text,r.payload_hash
		FROM relay_events r LEFT JOIN payment_observations o ON o.relay_event_id=r.id
		WHERE r.device_id=? AND r.source_event_id=?`, deviceID, sourceEventID).
		Scan(&result.RelayEventID, &result.Status, &paymentID, &packageName, &storedPostedAt, &amountHint, &title, &text, &bigText, &storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		return IngestResult{}, false, false, nil
	}
	if err != nil {
		return IngestResult{}, false, false, fmt.Errorf("read relay event: %w", err)
	}
	result.PaymentID = paymentID.String
	same := false
	if len(storedHash) == sha256.Size {
		same = bytes.Equal(storedHash, payloadHash) || relayEventFieldsEqual(packageName, storedPostedAt, amountHint, title, text, bigText, in, postedAt, postedReliable)
	} else if bytes.HasPrefix(storedHash, []byte("legacy:")) {
		comparisonPostedAt := postedAt
		if !postedReliable {
			comparisonPostedAt = time.UnixMilli(storedPostedAt).UTC()
		}
		same = bytes.Equal(storedHash, inputPayloadFingerprint(in, comparisonPostedAt))
	} else if len(storedHash) == 0 {
		// Legacy rows without a hash retain their fields until this service
		// redacts them, so retries remain idempotent during migration.
		same = relayEventFieldsEqual(packageName, storedPostedAt, amountHint, title, text, bigText, in, postedAt, postedReliable)
	}
	return result, true, same, nil
}
func relayEventFieldsEqual(packageName string, storedPostedAt int64, amountHint sql.NullInt64, title, text, bigText sql.NullString, in EventInput, postedAt time.Time, postedReliable bool) bool {
	return packageName == in.PackageName &&
		(!postedReliable || storedPostedAt == postedAt.UnixMilli()) &&
		nullableAmountEqual(amountHint, in.AmountHintPaise) &&
		nullableTextEqual(title, in.Title) &&
		nullableTextEqual(text, in.Text) &&
		nullableTextEqual(bigText, in.BigText)
}

func legacyPayloadFingerprint(packageName string, postedAt int64, amountHint sql.NullInt64,
	title, text, bigText sql.NullString) []byte {
	data := make([]byte, 0, 128)
	data = appendFingerprintField(data, packageName, true)
	data = appendFingerprintNumber(data, postedAt, true)
	data = appendFingerprintNumber(data, amountHint.Int64, amountHint.Valid)
	data = appendFingerprintField(data, title.String, title.Valid)
	data = appendFingerprintField(data, text.String, text.Valid)
	data = appendFingerprintField(data, bigText.String, bigText.Valid)
	digest := sha256.Sum256(data)
	fingerprint := make([]byte, len("legacy:")+len(digest))
	copy(fingerprint, "legacy:")
	copy(fingerprint[len("legacy:"):], digest[:])
	return fingerprint
}

func inputPayloadFingerprint(in EventInput, postedAt time.Time) []byte {
	return legacyPayloadFingerprint(in.PackageName, postedAt.UnixMilli(),
		sql.NullInt64{Int64: in.AmountHintPaise, Valid: in.AmountHintPaise != 0},
		fingerprintText(in.Title), fingerprintText(in.Text), fingerprintText(in.BigText))
}

func fingerprintText(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func appendFingerprintField(dst []byte, value string, valid bool) []byte {
	if !valid {
		return append(dst, 0)
	}
	dst = append(dst, 1)
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	dst = append(dst, length[:]...)
	return append(dst, value...)
}

func appendFingerprintNumber(dst []byte, value int64, valid bool) []byte {
	if !valid {
		return append(dst, 0)
	}
	dst = append(dst, 1)
	var number [8]byte
	binary.BigEndian.PutUint64(number[:], uint64(value))
	return append(dst, number[:]...)
}

func nullableAmountEqual(stored sql.NullInt64, wanted int64) bool {
	return stored.Valid == (wanted != 0) && (!stored.Valid || stored.Int64 == wanted)
}

func nullableTextEqual(stored sql.NullString, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	return stored.Valid == (wanted != "") && (!stored.Valid || stored.String == wanted)
}

func (s *Service) finishAmbiguous(ctx context.Context, relayEventID string, reason error) (string, error) {
	message := "ambiguous generic notification"
	if reason != nil {
		message = trimError(reason.Error(), 512)
	}
	var status string
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE relay_events SET status='ambiguous',error=? WHERE id=? AND status='received'`, message, relayEventID); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, `SELECT status FROM relay_events WHERE id=?`, relayEventID).Scan(&status)
	})
	if err != nil {
		return "", err
	}
	return status, nil
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
