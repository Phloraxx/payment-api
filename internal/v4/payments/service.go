package payments

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrInvalidPaymentInput = errors.New("invalid payment input")
	ErrNoActiveProfile     = errors.New("no active collection profile")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
)

const (
	defaultActiveWindow     = 5 * time.Minute
	defaultGraceWindow      = 5 * time.Minute
	defaultQuarantineWindow = 5 * time.Minute
	defaultIdempotencyTTL   = 24 * time.Hour
	maxMetadataBytes        = 16 << 10
)

type Service struct {
	DB         *storage.DB
	Allocator  Allocator
	Now        func() time.Time
	NewID      func(prefix string) (string, error)
	ActiveFor  time.Duration
	GraceFor   time.Duration
	Quarantine time.Duration
}

type CreateInput struct {
	RequestedAmountPaise int64
	Name                 string
	ExternalID           string
	Metadata             json.RawMessage
	IdempotencyScope     string
	IdempotencyKey       string
}

type Payment struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	ExternalID           string          `json:"external_id,omitempty"`
	Metadata             json.RawMessage `json:"metadata"`
	RequestedAmountPaise int64           `json:"-"`
	PayableAmountPaise   int64           `json:"-"`
	AdjustmentPaise      int64           `json:"-"`
	Status               string          `json:"status"`
	CollectionProfileID  string          `json:"-"`
	UPIIDSnapshot        string          `json:"-"`
	PayeeNameSnapshot    string          `json:"-"`
	CreatedAt            time.Time       `json:"created_at"`
	ExpiresAt            time.Time       `json:"expires_at"`
	GraceUntil           time.Time       `json:"grace_until"`
	ReuseAfter           time.Time       `json:"-"`
	PaidAt               *time.Time      `json:"paid_at,omitempty"`
	PayerName            string          `json:"payer_name,omitempty"`
	PayerUPIID           string          `json:"payer_upi_id,omitempty"`
}

type CreateResult struct {
	Payment  Payment
	UPIURI   string
	Replayed bool
}

type profileSnapshot struct {
	ID        string
	UPIID     string
	PayeeName string
}

func NewService(db *storage.DB) *Service {
	return &Service{
		DB:         db,
		Allocator:  NewAllocator(),
		Now:        time.Now,
		NewID:      randomID,
		ActiveFor:  defaultActiveWindow,
		GraceFor:   defaultGraceWindow,
		Quarantine: defaultQuarantineWindow,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	normalized, requestHash, keyHash, err := normalizeCreateInput(input)
	if err != nil {
		return CreateResult{}, err
	}
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return CreateResult{}, errors.New("payment storage is required")
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	idFn := s.NewID
	if idFn == nil {
		idFn = randomID
	}
	activeFor := positiveOr(s.ActiveFor, defaultActiveWindow)
	graceFor := positiveOr(s.GraceFor, defaultGraceWindow)
	quarantineFor := positiveOr(s.Quarantine, defaultQuarantineWindow)
	allocator := s.Allocator
	if allocator.Buckets == 0 && allocator.Random == nil && allocator.SoftHorizon == 0 {
		allocator = NewAllocator()
	}

	now := nowFn().UTC()
	var result CreateResult
	err = s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE expires_at <= ?`, now.UnixMilli()); err != nil {
			return fmt.Errorf("purge expired idempotency keys: %w", err)
		}
		replayed, err := loadIdempotentPayment(ctx, tx, normalized.IdempotencyScope, keyHash, requestHash)
		if err != nil {
			return err
		}
		if replayed != nil {
			result.Payment = *replayed
			result.UPIURI = buildUPIURI(replayed.UPIIDSnapshot, replayed.PayeeNameSnapshot, replayed.PayableAmountPaise)
			result.Replayed = true
			return nil
		}

		profile, err := activeProfile(ctx, tx)
		if err != nil {
			return err
		}
		payable, err := allocator.Select(ctx, tx, profile.ID, normalized.RequestedAmountPaise, now)
		if err != nil {
			return err
		}

		paymentID, err := idFn("pay")
		if err != nil {
			return fmt.Errorf("generate payment id: %w", err)
		}
		historyID, err := idFn("hist")
		if err != nil {
			return fmt.Errorf("generate history id: %w", err)
		}
		webhookID, err := idFn("evt")
		if err != nil {
			return fmt.Errorf("generate webhook id: %w", err)
		}
		reservationID, err := idFn("res")
		if err != nil {
			return fmt.Errorf("generate reservation id: %w", err)
		}

		expires := now.Add(activeFor)
		graceUntil := expires.Add(graceFor)
		reuseAfter := graceUntil.Add(quarantineFor)
		payment := Payment{
			ID: paymentID, Name: normalized.Name, ExternalID: normalized.ExternalID, Metadata: normalized.Metadata,
			RequestedAmountPaise: normalized.RequestedAmountPaise,
			PayableAmountPaise:   payable,
			AdjustmentPaise:      payable - normalized.RequestedAmountPaise,
			Status:               "pending",
			CollectionProfileID:  profile.ID,
			UPIIDSnapshot:        profile.UPIID,
			PayeeNameSnapshot:    profile.PayeeName,
			CreatedAt:            now, ExpiresAt: expires, GraceUntil: graceUntil, ReuseAfter: reuseAfter,
		}
		if err := insertPayment(ctx, tx, payment); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
            VALUES(?,?,?,?,?,?,?)`, reservationID, profile.ID, payable, paymentID, now.UnixMilli(), reuseAfter.UnixMilli(), now.UnixMilli()); err != nil {
			return fmt.Errorf("insert amount reservation: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO payment_history(id,payment_id,type,actor,summary,changes_json,created_at)
            VALUES(?,?,?,?,?,'{}',?)`, historyID, paymentID, "payment.created", "system", "Payment created", now.UnixMilli()); err != nil {
			return fmt.Errorf("insert payment history: %w", err)
		}

		payload, err := paymentEventPayload(webhookID, "payment.created", payment)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,event_type,payment_id,payload_json,status,attempts,created_at)
            VALUES(?,?,?,?,'pending',0,?)`, webhookID, "payment.created", paymentID, string(payload), now.UnixMilli()); err != nil {
			return fmt.Errorf("insert webhook outbox: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO idempotency_keys(scope,key_hash,request_hash,payment_id,created_at,expires_at)
            VALUES(?,?,?,?,?,?)`, normalized.IdempotencyScope, keyHash[:], requestHash[:], paymentID, now.UnixMilli(), now.Add(defaultIdempotencyTTL).UnixMilli()); err != nil {
			return fmt.Errorf("insert idempotency result: %w", err)
		}

		result.Payment = payment
		result.UPIURI = buildUPIURI(profile.UPIID, profile.PayeeName, payable)
		return nil
	})
	if err != nil {
		return CreateResult{}, err
	}
	return result, nil
}

func normalizeCreateInput(input CreateInput) (CreateInput, [32]byte, [32]byte, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.IdempotencyScope = strings.TrimSpace(input.IdempotencyScope)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)

	if input.RequestedAmountPaise <= 0 || input.RequestedAmountPaise%100 != 0 {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("%w: amount must be positive whole INR", ErrInvalidPaymentInput)
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 120 {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("%w: name must contain 1-120 characters", ErrInvalidPaymentInput)
	}
	if len(input.ExternalID) > 255 {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("%w: external_id is too long", ErrInvalidPaymentInput)
	}
	if input.IdempotencyScope == "" || len(input.IdempotencyScope) > 120 {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("%w: idempotency scope is required", ErrInvalidPaymentInput)
	}
	if input.IdempotencyKey == "" || len(input.IdempotencyKey) > 255 {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("%w: idempotency key is required", ErrInvalidPaymentInput)
	}
	metadata, err := canonicalMetadata(input.Metadata)
	if err != nil {
		return CreateInput{}, [32]byte{}, [32]byte{}, err
	}
	input.Metadata = metadata

	fingerprint, err := json.Marshal(struct {
		Amount     int64           `json:"amount"`
		Name       string          `json:"name"`
		ExternalID string          `json:"external_id"`
		Metadata   json.RawMessage `json:"metadata"`
	}{input.RequestedAmountPaise, input.Name, input.ExternalID, input.Metadata})
	if err != nil {
		return CreateInput{}, [32]byte{}, [32]byte{}, fmt.Errorf("fingerprint payment request: %w", err)
	}
	return input, sha256.Sum256(fingerprint), sha256.Sum256([]byte(input.IdempotencyKey)), nil
}

func canonicalMetadata(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if len(raw) > maxMetadataBytes {
		return nil, fmt.Errorf("%w: metadata exceeds %d bytes", ErrInvalidPaymentInput, maxMetadataBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: metadata is invalid JSON", ErrInvalidPaymentInput)
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("%w: metadata must be a JSON object", ErrInvalidPaymentInput)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: metadata has trailing JSON", ErrInvalidPaymentInput)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("canonicalize metadata: %w", err)
	}
	return canonical, nil
}

func loadIdempotentPayment(ctx context.Context, tx *storage.ImmediateTx, scope string, keyHash, requestHash [32]byte) (*Payment, error) {
	var storedRequest []byte
	var paymentID string
	err := tx.QueryRowContext(ctx, `SELECT request_hash,payment_id FROM idempotency_keys WHERE scope=? AND key_hash=?`, scope, keyHash[:]).Scan(&storedRequest, &paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read idempotency result: %w", err)
	}
	if !bytes.Equal(storedRequest, requestHash[:]) {
		return nil, ErrIdempotencyConflict
	}
	payment, err := loadPayment(ctx, tx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("load idempotent payment: %w", err)
	}
	return &payment, nil
}

func activeProfile(ctx context.Context, tx *storage.ImmediateTx) (profileSnapshot, error) {
	var p profileSnapshot
	err := tx.QueryRowContext(ctx, `SELECT id,upi_id,COALESCE(payee_name,'') FROM collection_profiles WHERE active=1 AND enabled=1`).Scan(&p.ID, &p.UPIID, &p.PayeeName)
	if errors.Is(err, sql.ErrNoRows) {
		return profileSnapshot{}, ErrNoActiveProfile
	}
	if err != nil {
		return profileSnapshot{}, fmt.Errorf("read active collection profile: %w", err)
	}
	return p, nil
}

func insertPayment(ctx context.Context, tx *storage.ImmediateTx, p Payment) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO payments(
        id,name,external_id,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,currency,
        collection_profile_id,upi_id_snapshot,payee_name_snapshot,status,created_at,expires_at,grace_until,reuse_after)
        VALUES(?,?,?,?,?,?,?,'INR',?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, nullableString(p.ExternalID), string(p.Metadata), p.RequestedAmountPaise, p.PayableAmountPaise, p.AdjustmentPaise,
		p.CollectionProfileID, p.UPIIDSnapshot, nullableString(p.PayeeNameSnapshot), p.Status,
		p.CreatedAt.UnixMilli(), p.ExpiresAt.UnixMilli(), p.GraceUntil.UnixMilli(), p.ReuseAfter.UnixMilli())
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

type paymentQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadPayment(ctx context.Context, tx paymentQueryRower, id string) (Payment, error) {
	var p Payment
	var metadata string
	var externalID, payeeName, payerName, payerUPI sql.NullString
	var created, expires, grace, reuse int64
	var paidAt sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT id,name,external_id,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,
        status,collection_profile_id,upi_id_snapshot,payee_name_snapshot,created_at,expires_at,grace_until,reuse_after,paid_at,payer_name,payer_upi_id
        FROM payments WHERE id=?`, id).Scan(
		&p.ID, &p.Name, &externalID, &metadata, &p.RequestedAmountPaise, &p.PayableAmountPaise, &p.AdjustmentPaise,
		&p.Status, &p.CollectionProfileID, &p.UPIIDSnapshot, &payeeName, &created, &expires, &grace, &reuse, &paidAt, &payerName, &payerUPI)
	if err != nil {
		return Payment{}, err
	}
	p.ExternalID = externalID.String
	p.Metadata = json.RawMessage(metadata)
	p.PayeeNameSnapshot = payeeName.String
	p.PayerName = payerName.String
	p.PayerUPIID = payerUPI.String
	p.CreatedAt = time.UnixMilli(created).UTC()
	p.ExpiresAt = time.UnixMilli(expires).UTC()
	p.GraceUntil = time.UnixMilli(grace).UTC()
	p.ReuseAfter = time.UnixMilli(reuse).UTC()
	if paidAt.Valid {
		t := time.UnixMilli(paidAt.Int64).UTC()
		p.PaidAt = &t
	}
	return p, nil
}

func buildUPIURI(upiID, payeeName string, amountPaise int64) string {
	q := url.Values{}
	q.Set("pa", upiID)
	if strings.TrimSpace(payeeName) != "" {
		q.Set("pn", payeeName)
	}
	q.Set("am", moneyString(amountPaise))
	q.Set("cu", "INR")
	return "upi://pay?" + q.Encode()
}

func paymentEventPayload(eventID, eventType string, p Payment) ([]byte, error) {
	return paymentEventPayloadAt(eventID, eventType, p, p.CreatedAt)
}

func paymentEventPayloadAt(eventID, eventType string, p Payment, eventAt time.Time) ([]byte, error) {
	paymentData := map[string]any{
		"id": p.ID, "name": p.Name, "external_id": p.ExternalID, "status": p.Status,
		"requested_amount": moneyString(p.RequestedAmountPaise), "payable_amount": moneyString(p.PayableAmountPaise),
		"metadata": json.RawMessage(p.Metadata),
	}
	if p.PaidAt != nil {
		paymentData["paid_at"] = p.PaidAt.UTC().Format(time.RFC3339Nano)
		paymentData["payer"] = map[string]any{"name": nullableJSON(p.PayerName), "upi_id": nullableJSON(p.PayerUPIID)}
	}
	payload := map[string]any{
		"id": eventID, "type": eventType, "created_at": eventAt.UTC().Format(time.RFC3339Nano),
		"data": map[string]any{"payment": paymentData},
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode webhook payload: %w", err)
	}
	return out, nil
}

func randomID(prefix string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return prefix + "_" + strings.ToLower(encoded), nil
}

func moneyString(paise int64) string {
	return strconv.FormatInt(paise/100, 10) + "." + fmt.Sprintf("%02d", paise%100)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableJSON(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func positiveOr(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}
