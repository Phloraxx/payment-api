package adminpayments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrInvalidFilter   = errors.New("invalid payment filter")
	ErrInvalidEdit     = errors.New("invalid payment edit")
	ErrUnsafeReopen    = errors.New("payment amount reservation can no longer be safely reopened")
)

type Service struct {
	DB    *storage.DB
	Now   func() time.Time
	NewID func(prefix string) (string, error)
}

func NewService(db *storage.DB) *Service {
	return &Service{DB: db, Now: time.Now, NewID: randomID}
}

type Payment struct {
	ID                   string
	Name                 string
	ExternalID           string
	Metadata             json.RawMessage
	RequestedAmountPaise int64
	PayableAmountPaise   int64
	AdjustmentPaise      int64
	CollectionProfileID  string
	UPIIDSnapshot        string
	PayeeNameSnapshot    string
	Status               string
	CreatedAt            time.Time
	ExpiresAt            time.Time
	GraceUntil           time.Time
	ReuseAfter           time.Time
	PaidAt               *time.Time
	PayerName            string
	PayerUPIID           string
	InternalNote         string
}

type ListInput struct {
	Query       string
	Status      string
	ExternalID  string
	ProfileID   string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type ListResult struct {
	Items  []Payment
	Total  int
	Limit  int
	Offset int
}
type HistoryEntry struct {
	ID        string
	Type      string
	Actor     string
	Summary   string
	Changes   json.RawMessage
	CreatedAt time.Time
}

type WebhookEntry struct {
	ID             string
	EventType      string
	Status         string
	Attempts       int
	NextAttemptAt  *time.Time
	LastHTTPStatus *int
	LastError      string
	CreatedAt      time.Time
	DeliveredAt    *time.Time
}

type Detail struct {
	Payment  Payment
	History  []HistoryEntry
	Webhooks []WebhookEntry
}

type rowScanner interface {
	Scan(...any) error
}

const paymentColumns = `id,name,external_id,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,
	collection_profile_id,upi_id_snapshot,payee_name_snapshot,status,created_at,expires_at,grace_until,reuse_after,
	paid_at,payer_name,payer_upi_id,internal_note`

func (s *Service) ready() error {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return errors.New("admin payment storage is required")
	}
	return nil
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, error) {
	if err := s.ready(); err != nil {
		return ListResult{}, err
	}
	where, args, err := buildWhere(input)
	if err != nil {
		return ListResult{}, err
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := input.Offset
	if offset < 0 {
		return ListResult{}, fmt.Errorf("%w: offset cannot be negative", ErrInvalidFilter)
	}
	var total int
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments`+where, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count payments: %w", err)
	}
	query := `SELECT ` + paymentColumns + ` FROM payments` + where + ` ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?`
	rows, err := s.DB.SQL.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()
	items := make([]Payment, 0, limit)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("scan payment: %w", err)
		}
		items = append(items, payment)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate payments: %w", err)
	}
	return ListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func buildWhere(input ListInput) (string, []any, error) {
	var clauses []string
	var args []any
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != "" {
		switch status {
		case "pending", "paid", "expired", "cancelled":
		default:
			return "", nil, fmt.Errorf("%w: unknown status %q", ErrInvalidFilter, status)
		}
		clauses = append(clauses, "status=?")
		args = append(args, status)
	}
	if value := strings.TrimSpace(input.ExternalID); value != "" {
		clauses = append(clauses, "external_id=?")
		args = append(args, value)
	}
	if value := strings.ToLower(strings.TrimSpace(input.ProfileID)); value != "" {
		clauses = append(clauses, "collection_profile_id=?")
		args = append(args, value)
	}
	if input.CreatedFrom != nil {
		clauses = append(clauses, "created_at>=?")
		args = append(args, input.CreatedFrom.UTC().UnixMilli())
	}
	if input.CreatedTo != nil {
		clauses = append(clauses, "created_at<=?")
		args = append(args, input.CreatedTo.UTC().UnixMilli())
	}
	query := strings.TrimSpace(input.Query)
	if query != "" {
		like := "%" + escapeLike(strings.ToLower(query)) + "%"
		search := `(lower(id) LIKE ? ESCAPE '\' OR lower(name) LIKE ? ESCAPE '\' OR lower(COALESCE(external_id,'')) LIKE ? ESCAPE '\' OR
			lower(COALESCE(payer_name,'')) LIKE ? ESCAPE '\' OR lower(COALESCE(payer_upi_id,'')) LIKE ? ESCAPE '\' OR lower(metadata_json) LIKE ? ESCAPE '\')`
		searchArgs := []any{like, like, like, like, like, like}
		if amount, ok := parseMoney(query); ok {
			search = "(" + search + " OR requested_amount_paise=? OR payable_amount_paise=?)"
			searchArgs = append(searchArgs, amount, amount)
		}
		clauses = append(clauses, search)
		args = append(args, searchArgs...)
	}
	if len(clauses) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
func parseMoney(value string) (int64, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "₹"))
	if value == "" {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, false
	}
	rupees, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || rupees < 0 {
		return 0, false
	}
	paise := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) == 1 {
			parts[1] += "0"
		}
		if len(parts[1]) != 2 {
			return 0, false
		}
		paise, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || paise < 0 || paise > 99 {
			return 0, false
		}
	}
	if rupees > (1<<63-1-paise)/100 {
		return 0, false
	}
	return rupees*100 + paise, true
}

func (s *Service) Get(ctx context.Context, id string) (Detail, error) {
	if err := s.ready(); err != nil {
		return Detail{}, err
	}
	id = strings.TrimSpace(id)
	payment, err := getPayment(ctx, s.DB.SQL, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, ErrPaymentNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("get payment: %w", err)
	}
	history, err := loadHistory(ctx, s.DB.SQL, id)
	if err != nil {
		return Detail{}, err
	}
	webhooks, err := loadWebhooks(ctx, s.DB.SQL, id)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Payment: payment, History: history, Webhooks: webhooks}, nil
}

func getPayment(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (Payment, error) {
	return scanPayment(q.QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM payments WHERE id=?`, id))
}

func scanPayment(row rowScanner) (Payment, error) {
	var p Payment
	var externalID, payeeName, payerName, payerUPI, note sql.NullString
	var metadata string
	var created, expires, grace, reuse int64
	var paidAt sql.NullInt64
	if err := row.Scan(&p.ID, &p.Name, &externalID, &metadata, &p.RequestedAmountPaise, &p.PayableAmountPaise,
		&p.AdjustmentPaise, &p.CollectionProfileID, &p.UPIIDSnapshot, &payeeName, &p.Status,
		&created, &expires, &grace, &reuse, &paidAt, &payerName, &payerUPI, &note); err != nil {
		return Payment{}, err
	}
	p.ExternalID, p.PayeeNameSnapshot, p.PayerName, p.PayerUPIID, p.InternalNote = externalID.String, payeeName.String, payerName.String, payerUPI.String, note.String
	p.Metadata = json.RawMessage(metadata)
	p.CreatedAt, p.ExpiresAt = time.UnixMilli(created).UTC(), time.UnixMilli(expires).UTC()
	p.GraceUntil, p.ReuseAfter = time.UnixMilli(grace).UTC(), time.UnixMilli(reuse).UTC()
	if paidAt.Valid {
		value := time.UnixMilli(paidAt.Int64).UTC()
		p.PaidAt = &value
	}
	return p, nil
}
func loadHistory(ctx context.Context, db *sql.DB, paymentID string) ([]HistoryEntry, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,type,actor,summary,changes_json,created_at FROM payment_history WHERE payment_id=? ORDER BY created_at,rowid`, paymentID)
	if err != nil {
		return nil, fmt.Errorf("load payment history: %w", err)
	}
	defer rows.Close()
	var out []HistoryEntry
	for rows.Next() {
		var entry HistoryEntry
		var changes string
		var created int64
		if err := rows.Scan(&entry.ID, &entry.Type, &entry.Actor, &entry.Summary, &changes, &created); err != nil {
			return nil, fmt.Errorf("scan payment history: %w", err)
		}
		entry.Changes = json.RawMessage(changes)
		entry.CreatedAt = time.UnixMilli(created).UTC()
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment history: %w", err)
	}
	return out, nil
}

func loadWebhooks(ctx context.Context, db *sql.DB, paymentID string) ([]WebhookEntry, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,event_type,status,attempts,next_attempt_at,last_http_status,last_error,created_at,delivered_at
		FROM webhook_deliveries WHERE payment_id=? ORDER BY created_at,rowid`, paymentID)
	if err != nil {
		return nil, fmt.Errorf("load payment webhooks: %w", err)
	}
	defer rows.Close()
	var out []WebhookEntry
	for rows.Next() {
		var entry WebhookEntry
		var next, delivered sql.NullInt64
		var statusCode sql.NullInt64
		var lastError sql.NullString
		var created int64
		if err := rows.Scan(&entry.ID, &entry.EventType, &entry.Status, &entry.Attempts, &next, &statusCode,
			&lastError, &created, &delivered); err != nil {
			return nil, fmt.Errorf("scan payment webhook: %w", err)
		}
		entry.LastError = lastError.String
		entry.CreatedAt = time.UnixMilli(created).UTC()
		if next.Valid {
			value := time.UnixMilli(next.Int64).UTC()
			entry.NextAttemptAt = &value
		}
		if statusCode.Valid {
			value := int(statusCode.Int64)
			entry.LastHTTPStatus = &value
		}
		if delivered.Valid {
			value := time.UnixMilli(delivered.Int64).UTC()
			entry.DeliveredAt = &value
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment webhooks: %w", err)
	}
	return out, nil
}
