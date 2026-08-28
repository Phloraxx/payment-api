package operatorview

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type Service struct{ App core.App }

func New(app core.App) *Service { return &Service{App: app} }

type PaymentSummary struct {
	ID                   string `json:"id"`
	PaymentAccount       string `json:"paymentAccount"`
	RequestedAmountPaise int64  `json:"requestedAmountPaise"`
	PayableAmountPaise   int64  `json:"payableAmountPaise"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
	ExpiresAt            string `json:"expiresAt"`
	PaidAt               string `json:"paidAt,omitempty"`
}
type PaymentDetail struct {
	PaymentSummary
	ExternalID        string `json:"externalId,omitempty"`
	PayerName         string `json:"payerName,omitempty"`
	UPIID             string `json:"upiId,omitempty"`
	RRN               string `json:"rrn,omitempty"`
	EvidenceSource    string `json:"evidenceSource,omitempty"`
	EvidenceReference string `json:"evidenceReference,omitempty"`
	ResolvedAt        string `json:"resolvedAt,omitempty"`
}

type ReviewSummary struct {
	ID                  string   `json:"id"`
	Kind                string   `json:"kind"`
	Status              string   `json:"status"`
	Severity            string   `json:"severity"`
	PaymentID           string   `json:"paymentId,omitempty"`
	CandidatePaymentIDs []string `json:"candidatePaymentIds,omitempty"`
	Reason              string   `json:"reason"`
	OpenedAt            string   `json:"openedAt"`
}

type EvidenceDetail struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Source      string `json:"source,omitempty"`
	Sender      string `json:"sender,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Amount      int64  `json:"amountPaise,omitempty"`
	Reference   string `json:"reference,omitempty"`
	UPIID       string `json:"upiId,omitempty"`
	PayerName   string `json:"payerName,omitempty"`
	OccurredAt  string `json:"occurredAt,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type ReviewDetail struct {
	ReviewSummary
	Resolution     string          `json:"resolution,omitempty"`
	ResolutionNote string          `json:"resolutionNote,omitempty"`
	ResolvedAt     string          `json:"resolvedAt,omitempty"`
	Evidence       *EvidenceDetail `json:"evidence,omitempty"`
}

type AlertSummary struct {
	ID                      string `json:"id"`
	Kind                    string `json:"kind"`
	Status                  string `json:"status"`
	Severity                string `json:"severity"`
	Message                 string `json:"message"`
	OccurrenceCount         int    `json:"occurrenceCount"`
	FirstSeenAt             string `json:"firstSeenAt"`
	LastSeenAt              string `json:"lastSeenAt"`
	NotificationStatus      string `json:"notificationStatus,omitempty"`
	NotificationAttempts    int    `json:"notificationAttempts,omitempty"`
	NotificationLastError   string `json:"notificationLastError,omitempty"`
	NotificationDeliveredAt string `json:"notificationDeliveredAt,omitempty"`
}

type Overview struct {
	PaymentCounts map[string]int64 `json:"paymentCounts"`
	OpenReviews   int64            `json:"openReviews"`
	OpenAlerts    int64            `json:"openAlerts"`
	Recent        []PaymentSummary `json:"recentPayments"`
}

func (s *Service) Overview(limit int) (Overview, error) {
	limit = clampLimit(limit, 8, 20)
	counts := map[string]int64{"total": 0, "pending": 0, "paid": 0, "late": 0, "expired": 0, "cancelled": 0}
	var err error
	counts["total"], err = s.App.CountRecords("payments")
	if err != nil {
		return Overview{}, err
	}
	for _, status := range []string{"pending", "paid", "late", "expired", "cancelled"} {
		counts[status], err = s.App.CountRecords("payments", dbx.NewExp("status = {:status}", dbx.Params{"status": status}))
		if err != nil {
			return Overview{}, err
		}
	}
	openReviews, err := s.App.CountRecords("review_cases", dbx.NewExp("status = 'open'"))
	if err != nil {
		return Overview{}, err
	}
	openAlerts, err := s.App.CountRecords("alerts", dbx.NewExp("status = 'open'"))
	if err != nil {
		return Overview{}, err
	}
	recent, err := s.ListPayments("", limit)
	if err != nil {
		return Overview{}, err
	}
	return Overview{
		PaymentCounts: counts,
		OpenReviews:   openReviews,
		OpenAlerts:    openAlerts,
		Recent:        recent,
	}, nil
}

func (s *Service) ListPayments(status string, limit int) ([]PaymentSummary, error) {
	limit = clampLimit(limit, 50, 100)
	status = strings.TrimSpace(strings.ToLower(status))
	filter := "id != ''"
	params := dbx.Params{}
	if status != "" {
		if !validPaymentStatus(status) {
			return nil, fmt.Errorf("invalid payment status")
		}
		filter += " && status = {:status}"
		params["status"] = status
	}
	records, err := s.App.FindRecordsByFilter("payments", filter, "-created_at", limit, 0, params)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentSummary, 0, len(records))
	for _, record := range records {
		out = append(out, paymentSummary(record))
	}
	return out, nil
}

func (s *Service) GetPayment(id string) (PaymentDetail, error) {
	record, err := s.App.FindRecordById("payments", strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentDetail{}, sql.ErrNoRows
		}
		return PaymentDetail{}, err
	}
	return PaymentDetail{
		PaymentSummary:    paymentSummary(record),
		ExternalID:        record.GetString("external_id"),
		PayerName:         record.GetString("payer_name"),
		UPIID:             record.GetString("upi_id"),
		RRN:               record.GetString("rrn"),
		EvidenceSource:    record.GetString("evidence_source"),
		EvidenceReference: record.GetString("evidence_reference"),
		ResolvedAt:        dateString(record, "resolved_at"),
	}, nil
}
func (s *Service) ListReviews(status string, limit int) ([]ReviewSummary, error) {
	limit = clampLimit(limit, 50, 100)
	status = strings.TrimSpace(strings.ToLower(status))
	filter := "id != ''"
	params := dbx.Params{}
	if status != "" {
		if status != "open" && status != "resolved" && status != "dismissed" {
			return nil, fmt.Errorf("invalid review status")
		}
		filter += " && status = {:status}"
		params["status"] = status
	}
	records, err := s.App.FindRecordsByFilter("review_cases", filter, "-opened_at", limit, 0, params)
	if err != nil {
		return nil, err
	}
	out := make([]ReviewSummary, 0, len(records))
	for _, record := range records {
		out = append(out, ReviewSummary{
			ID: record.Id, Kind: record.GetString("kind"), Status: record.GetString("status"), Severity: record.GetString("severity"),
			PaymentID: record.GetString("payment"), CandidatePaymentIDs: stringSlice(record.Get("candidate_payment_ids")),
			Reason: record.GetString("reason"), OpenedAt: dateString(record, "opened_at"),
		})
	}
	return out, nil
}
func (s *Service) GetReview(id string) (ReviewDetail, error) {
	record, err := s.App.FindRecordById("review_cases", strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReviewDetail{}, sql.ErrNoRows
		}
		return ReviewDetail{}, err
	}
	summary := ReviewSummary{
		ID: record.Id, Kind: record.GetString("kind"), Status: record.GetString("status"), Severity: record.GetString("severity"),
		PaymentID: record.GetString("payment"), CandidatePaymentIDs: stringSlice(record.Get("candidate_payment_ids")),
		Reason: record.GetString("reason"), OpenedAt: dateString(record, "opened_at"),
	}
	detail := ReviewDetail{ReviewSummary: summary, Resolution: record.GetString("resolution"), ResolutionNote: record.GetString("resolution_note"), ResolvedAt: dateString(record, "resolved_at")}
	for _, relation := range []struct{ field, collection, kind string }{
		{"sms_event", "sms_events", "sms"}, {"email_event", "email_events", "email"}, {"reconciliation_entry", "reconciliation_entries", "reconciliation"},
	} {
		id := record.GetString(relation.field)
		if id == "" {
			continue
		}
		evidence, findErr := s.App.FindRecordById(relation.collection, id)
		if findErr != nil {
			return ReviewDetail{}, findErr
		}
		detail.Evidence = evidenceDetail(evidence, relation.kind)
		break
	}
	return detail, nil
}

func evidenceDetail(record *core.Record, kind string) *EvidenceDetail {
	occurred := dateString(record, "message_time")
	if kind == "reconciliation" {
		occurred = dateString(record, "transaction_time")
	}
	status := record.GetString("processing_status")
	if kind == "reconciliation" {
		status = record.GetString("status")
	}
	return &EvidenceDetail{
		Kind: kind, ID: record.Id, Source: record.GetString("source"), Sender: record.GetString("sender"), Subject: record.GetString("subject"),
		Amount: int64(record.GetInt("amount")), Reference: record.GetString("rrn"), UPIID: record.GetString("upi_id"), PayerName: record.GetString("payer_name"),
		OccurredAt: occurred, Description: record.GetString("description"), Status: status, Notes: record.GetString("notes"),
	}
}

func (s *Service) ListAlerts(status string, limit int) ([]AlertSummary, error) {
	limit = clampLimit(limit, 50, 100)
	status = strings.TrimSpace(strings.ToLower(status))
	filter := "id != ''"
	params := dbx.Params{}
	if status != "" {
		if status != "open" && status != "resolved" {
			return nil, fmt.Errorf("invalid alert status")
		}
		filter += " && status = {:status}"
		params["status"] = status
	}
	records, err := s.App.FindRecordsByFilter("alerts", filter, "-last_seen_at", limit, 0, params)
	if err != nil {
		return nil, err
	}
	out := make([]AlertSummary, 0, len(records))
	for _, record := range records {
		out = append(out, AlertSummary{
			ID: record.Id, Kind: record.GetString("kind"), Status: record.GetString("status"), Severity: record.GetString("severity"),
			Message: record.GetString("message"), OccurrenceCount: record.GetInt("occurrence_count"),
			FirstSeenAt: dateString(record, "first_seen_at"), LastSeenAt: dateString(record, "last_seen_at"),
			NotificationStatus: record.GetString("notification_status"), NotificationAttempts: record.GetInt("notification_attempts"),
			NotificationLastError: record.GetString("notification_last_error"), NotificationDeliveredAt: dateString(record, "notification_delivered_at"),
		})
	}
	return out, nil
}
func paymentSummary(record *core.Record) PaymentSummary {
	return PaymentSummary{
		ID:                   record.Id,
		PaymentAccount:       record.GetString("payment_account"),
		RequestedAmountPaise: int64(record.GetInt("requested_amount")),
		PayableAmountPaise:   int64(record.GetInt("payable_amount")),
		Status:               record.GetString("status"),
		CreatedAt:            dateString(record, "created_at"),
		ExpiresAt:            dateString(record, "expires_at"),
		PaidAt:               dateString(record, "paid_at"),
	}
}

func dateString(record *core.Record, field string) string {
	value := record.GetDateTime(field).Time()
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func validPaymentStatus(status string) bool {
	switch status {
	case "pending", "paid", "late", "expired", "cancelled":
		return true
	default:
		return false
	}
}
func stringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func clampLimit(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

type ReconciliationRunSummary struct {
	ID            string `json:"id"`
	Filename      string `json:"filename"`
	Status        string `json:"status"`
	TotalRows     int    `json:"totalRows"`
	MatchedRows   int    `json:"matchedRows"`
	UnmatchedRows int    `json:"unmatchedRows"`
	DuplicateRows int    `json:"duplicateRows"`
	ConflictRows  int    `json:"conflictRows"`
	InvalidRows   int    `json:"invalidRows"`
	Error         string `json:"error,omitempty"`
	StartedAt     string `json:"startedAt"`
	CompletedAt   string `json:"completedAt,omitempty"`
}
type ReconciliationEntrySummary struct {
	ID              string `json:"id"`
	RowNumber       int    `json:"rowNumber"`
	TransactionTime string `json:"transactionTime,omitempty"`
	AmountPaise     int64  `json:"amountPaise,omitempty"`
	Reference       string `json:"reference,omitempty"`
	Description     string `json:"description,omitempty"`
	Status          string `json:"status"`
	PaymentID       string `json:"paymentId,omitempty"`
	Notes           string `json:"notes,omitempty"`
}
type RefundSummary struct {
	ID          string `json:"id"`
	PaymentID   string `json:"paymentId"`
	AmountPaise int64  `json:"amountPaise"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	Reference   string `json:"reference,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
	RequestedAt string `json:"requestedAt"`
	CompletedAt string `json:"completedAt,omitempty"`
}

func (s *Service) ListReconciliationRuns(limit int) ([]ReconciliationRunSummary, error) {
	records, err := s.App.FindRecordsByFilter("reconciliation_runs", "id != ''", "-started_at", clampLimit(limit, 50, 100), 0)
	if err != nil {
		return nil, err
	}
	out := make([]ReconciliationRunSummary, 0, len(records))
	for _, r := range records {
		out = append(out, ReconciliationRunSummary{ID: r.Id, Filename: r.GetString("filename"), Status: r.GetString("status"), TotalRows: r.GetInt("total_rows"), MatchedRows: r.GetInt("matched_rows"), UnmatchedRows: r.GetInt("unmatched_rows"), DuplicateRows: r.GetInt("duplicate_rows"), ConflictRows: r.GetInt("conflict_rows"), InvalidRows: r.GetInt("invalid_rows"), Error: r.GetString("error"), StartedAt: dateString(r, "started_at"), CompletedAt: dateString(r, "completed_at")})
	}
	return out, nil
}
func (s *Service) ListReconciliationEntries(runID string, limit int) ([]ReconciliationEntrySummary, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	records, err := s.App.FindRecordsByFilter("reconciliation_entries", "run = {:run}", "row_number", clampLimit(limit, 250, 500), 0, dbx.Params{"run": runID})
	if err != nil {
		return nil, err
	}
	out := make([]ReconciliationEntrySummary, 0, len(records))
	for _, r := range records {
		out = append(out, ReconciliationEntrySummary{ID: r.Id, RowNumber: r.GetInt("row_number"), TransactionTime: dateString(r, "transaction_time"), AmountPaise: int64(r.GetInt("amount")), Reference: r.GetString("rrn"), Description: r.GetString("description"), Status: r.GetString("status"), PaymentID: r.GetString("payment"), Notes: r.GetString("notes")})
	}
	return out, nil
}
func (s *Service) ListRefunds(limit int) ([]RefundSummary, error) {
	records, err := s.App.FindRecordsByFilter("refunds", "id != ''", "-requested_at", clampLimit(limit, 50, 100), 0)
	if err != nil {
		return nil, err
	}
	out := make([]RefundSummary, 0, len(records))
	for _, r := range records {
		out = append(out, RefundSummary{ID: r.Id, PaymentID: r.GetString("payment"), AmountPaise: int64(r.GetInt("amount")), Status: r.GetString("status"), Reason: r.GetString("reason"), Reference: r.GetString("reference"), ExternalID: r.GetString("external_id"), RequestedAt: dateString(r, "requested_at"), CompletedAt: dateString(r, "completed_at")})
	}
	return out, nil
}

type OperationalRecord struct {
	ID        string         `json:"id"`
	CreatedAt string         `json:"createdAt,omitempty"`
	Fields    map[string]any `json:"fields"`
}
type operationalSpec struct {
	collection, sort string
	fields, dates    []string
}

func operationalRecordSpec(kind string) (operationalSpec, bool) {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "sms":
		return operationalSpec{"sms_events", "-created", []string{"payment_account", "source", "source_event_id", "sender", "body", "amount", "rrn", "upi_id", "payer_name", "processing_status", "matched_payment", "error"}, []string{"message_time"}}, true
	case "email":
		return operationalSpec{"email_events", "-created", []string{"payment_account", "source", "source_event_id", "sender", "recipient", "subject", "body", "amount", "rrn", "upi_id", "payer_name", "processing_status", "matched_payment", "error"}, []string{"message_time", "received_at"}}, true
	case "audit":
		return operationalSpec{"audit_events", "-occurred_at", []string{"action", "actor_email", "entity_type", "entity_id", "summary", "details"}, []string{"occurred_at"}}, true
	case "webhooks":
		return operationalSpec{"webhook_deliveries", "-created", []string{"event_id", "event", "payment", "status", "attempts", "response_code", "last_error"}, []string{"next_attempt_at", "last_attempt_at", "delivered_at"}}, true
	default:
		return operationalSpec{}, false
	}
}
func (s *Service) ListOperationalRecords(kind string, limit int) ([]OperationalRecord, error) {
	spec, ok := operationalRecordSpec(kind)
	if !ok {
		return nil, fmt.Errorf("invalid operational record kind")
	}
	records, err := s.App.FindRecordsByFilter(spec.collection, "id != ''", spec.sort, clampLimit(limit, 50, 100), 0)
	if err != nil {
		return nil, err
	}
	out := make([]OperationalRecord, 0, len(records))
	for _, record := range records {
		fields := make(map[string]any, len(spec.fields)+len(spec.dates))
		for _, field := range spec.fields {
			if value := record.Get(field); value != nil && value != "" {
				fields[field] = value
			}
		}
		for _, field := range spec.dates {
			if value := dateString(record, field); value != "" {
				fields[field] = value
			}
		}
		out = append(out, OperationalRecord{ID: record.Id, CreatedAt: dateString(record, "created"), Fields: fields})
	}
	return out, nil
}

type RazorpayOrderSummary struct {
	ID                string `json:"id"`
	AmountPaise       int64  `json:"amountPaise"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	ExternalID        string `json:"externalId,omitempty"`
	RazorpayOrderID   string `json:"razorpayOrderId,omitempty"`
	RazorpayPaymentID string `json:"razorpayPaymentId,omitempty"`
	ProviderStatus    string `json:"providerStatus,omitempty"`
	PaymentMethod     string `json:"paymentMethod,omitempty"`
	AmountRefunded    int64  `json:"amountRefunded,omitempty"`
	Error             string `json:"error,omitempty"`
	CreatedAt         string `json:"createdAt"`
	CapturedAt        string `json:"capturedAt,omitempty"`
}

func (s *Service) ListRazorpayOrders(mode string, limit int) ([]RazorpayOrderSummary, error) {
	collection := "razorpay_test_orders"
	if strings.TrimSpace(strings.ToLower(mode)) != "test" {
		return nil, fmt.Errorf("invalid razorpay mode")
	}
	records, err := s.App.FindRecordsByFilter(collection, "id != ''", "-created_at", clampLimit(limit, 50, 100), 0)
	if err != nil {
		return nil, err
	}
	out := make([]RazorpayOrderSummary, 0, len(records))
	for _, r := range records {
		out = append(out, RazorpayOrderSummary{ID: r.Id, AmountPaise: int64(r.GetInt("amount")), Currency: r.GetString("currency"), Status: r.GetString("status"), ExternalID: r.GetString("external_id"), RazorpayOrderID: r.GetString("razorpay_order_id"), RazorpayPaymentID: r.GetString("razorpay_payment_id"), ProviderStatus: r.GetString("provider_status"), PaymentMethod: r.GetString("payment_method"), AmountRefunded: int64(r.GetInt("amount_refunded")), Error: r.GetString("error"), CreatedAt: dateString(r, "created_at"), CapturedAt: dateString(r, "captured_at")})
	}
	return out, nil
}
