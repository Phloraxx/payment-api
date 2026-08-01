package reconciliation

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/xuri/excelize/v2"
)

const (
	MaxFileBytes = 10 << 20
	MaxRows      = 10_000
	MaxColumns   = 64
)

type Service struct {
	App               core.App
	Reviews           *reviews.Service
	Alerts            *alerts.Service
	Audit             *audit.Service
	Now               func() time.Time
	StatementLocation *time.Location
}

type ImportInput struct {
	Filename string
	Data     []byte
	Actor    audit.Actor
}

type Result struct {
	RunID         string `json:"runId"`
	Status        string `json:"status"`
	TotalRows     int    `json:"totalRows"`
	MatchedRows   int    `json:"matchedRows"`
	UnmatchedRows int    `json:"unmatchedRows"`
	DuplicateRows int    `json:"duplicateRows"`
	ConflictRows  int    `json:"conflictRows"`
	InvalidRows   int    `json:"invalidRows"`
	ReviewCases   int    `json:"reviewCases"`
}

type statementRow struct {
	RowNumber       int
	TransactionTime time.Time
	AmountPaise     int64
	RRN             string
	Description     string
	Raw             map[string]string
	InvalidReason   string
}

type columns struct {
	date, credit, debit, amount, kind, description, rrn int
}

func NewService(app core.App, reviewService *reviews.Service, alertService *alerts.Service, auditService *audit.Service) *Service {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		location = time.FixedZone("IST", 5*60*60+30*60)
	}
	return &Service{App: app, Reviews: reviewService, Alerts: alertService, Audit: auditService, Now: time.Now, StatementLocation: location}
}

func (s *Service) Import(input ImportInput) (Result, error) {
	input.Filename = filepath.Base(strings.TrimSpace(input.Filename))
	if input.Filename == "." || input.Filename == "" {
		return Result{}, domain.New("INVALID_STATEMENT_FILE", "statement filename is required", 400)
	}
	if len(input.Data) == 0 || len(input.Data) > MaxFileBytes {
		return Result{}, domain.New("INVALID_STATEMENT_FILE", "statement must be between 1 byte and 10 MiB", 400)
	}
	hashBytes := sha256.Sum256(input.Data)
	hash := hex.EncodeToString(hashBytes[:])
	if existing, err := s.App.FindFirstRecordByFilter("reconciliation_runs", "sha256 = {:hash} && status = 'completed'", dbx.Params{"hash": hash}); err == nil {
		domainErr := domain.New("STATEMENT_ALREADY_IMPORTED", "this exact statement file was already imported", 409)
		domainErr.Details = map[string]any{"runId": existing.Id}
		return Result{}, domainErr
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Result{}, err
	}

	now := s.now()
	run, err := s.createRun(input, hash, now)
	if err != nil {
		return Result{}, err
	}
	rows, parseErr := parseStatement(input.Filename, input.Data, s.statementLocation())
	if parseErr != nil {
		s.failRun(run, parseErr, now)
		return Result{}, domain.New("STATEMENT_PARSE_FAILED", parseErr.Error(), 422)
	}

	result := Result{RunID: run.Id, Status: "completed"}
	seenRRN := map[string]int{}
	err = s.App.RunInTransaction(func(tx core.App) error {
		entryCollection, err := tx.FindCollectionByNameOrId("reconciliation_entries")
		if err != nil {
			return err
		}
		for _, row := range rows {
			result.TotalRows++
			entry := core.NewRecord(entryCollection)
			entry.Set("run", run.Id)
			entry.Set("row_number", row.RowNumber)
			entry.Set("transaction_time", row.TransactionTime)
			entry.Set("amount", row.AmountPaise)
			entry.Set("rrn", row.RRN)
			entry.Set("description", truncate(row.Description, 4096))
			entry.Set("raw_row", row.Raw)

			status, paymentID, note, candidateIDs, classifyErr := s.classify(tx, row, seenRRN, now)
			if classifyErr != nil {
				return classifyErr
			}
			entry.Set("status", status)
			entry.Set("payment", paymentID)
			entry.Set("notes", note)
			if err := tx.Save(entry); err != nil {
				return err
			}

			switch status {
			case "matched":
				result.MatchedRows++
			case "unmatched":
				result.UnmatchedRows++
			case "duplicate":
				result.DuplicateRows++
			case "conflict":
				result.ConflictRows++
			case "invalid":
				result.InvalidRows++
			}

			needsReview := status == "conflict" || (status == "unmatched" && (row.RRN != "" || strings.Contains(strings.ToLower(row.Description), "upi")))
			if needsReview && s.Reviews != nil {
				caseID, err := s.Reviews.OpenInApp(tx, reviews.OpenInput{
					Kind: "reconciliation_conflict", Severity: severityForStatus(status),
					ReconciliationEntryID: entry.Id, PaymentID: paymentID,
					CandidatePaymentIDs: candidateIDs, Reason: note, OpenedAt: now,
				})
				if err != nil {
					return err
				}
				if caseID != "" {
					result.ReviewCases++
				}
			}
		}

		runRecord, err := tx.FindRecordById("reconciliation_runs", run.Id)
		if err != nil {
			return err
		}
		runRecord.Set("status", "completed")
		runRecord.Set("total_rows", result.TotalRows)
		runRecord.Set("matched_rows", result.MatchedRows)
		runRecord.Set("unmatched_rows", result.UnmatchedRows)
		runRecord.Set("duplicate_rows", result.DuplicateRows)
		runRecord.Set("conflict_rows", result.ConflictRows)
		runRecord.Set("invalid_rows", result.InvalidRows)
		runRecord.Set("completed_at", now)
		runRecord.Set("summary", result)
		if err := tx.Save(runRecord); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordInApp(tx, audit.Entry{
				Action: "reconciliation.import", Actor: input.Actor,
				EntityType: "reconciliation_run", EntityID: run.Id,
				Summary: "Imported bank statement for reconciliation", Details: result, OccurredAt: now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.failRun(run, err, now)
		return Result{}, err
	}
	if result.ConflictRows > 0 && s.Alerts != nil {
		_, _, _ = s.Alerts.Open(alerts.Input{
			Kind: "reconciliation_conflict", Severity: "warning", DedupeKey: "reconciliation:" + run.Id,
			Message: fmt.Sprintf("Statement reconciliation found %d conflicting rows", result.ConflictRows), Details: result,
		})
	}
	return result, nil
}

func (s *Service) createRun(input ImportInput, hash string, now time.Time) (*core.Record, error) {
	collection, err := s.App.FindCollectionByNameOrId("reconciliation_runs")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("filename", truncate(input.Filename, 255))
	record.Set("sha256", hash)
	record.Set("status", "processing")
	record.Set("created_by", input.Actor.ID)
	record.Set("started_at", now)
	if err := s.App.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) failRun(run *core.Record, cause error, now time.Time) {
	if run == nil {
		return
	}
	record, err := s.App.FindRecordById("reconciliation_runs", run.Id)
	if err != nil {
		return
	}
	record.Set("status", "failed")
	record.Set("error", truncate(cause.Error(), 4096))
	record.Set("completed_at", now)
	_ = s.App.Save(record)
}

func (s *Service) classify(app core.App, row statementRow, seenRRN map[string]int, now time.Time) (status, paymentID, note string, candidateIDs []string, err error) {
	if row.InvalidReason != "" || row.AmountPaise <= 0 {
		return "invalid", "", row.InvalidReason, nil, nil
	}
	if row.RRN != "" {
		if firstRow, ok := seenRRN[row.RRN]; ok {
			return "duplicate", "", fmt.Sprintf("Duplicate bank reference already appeared on row %d", firstRow), nil, nil
		}
		seenRRN[row.RRN] = row.RowNumber
		existing, findErr := app.FindFirstRecordByData("payments", "rrn", row.RRN)
		if findErr == nil {
			if int64(existing.GetInt("payable_amount")) == row.AmountPaise {
				return "matched", existing.Id, "Exact amount and bank reference match", []string{existing.Id}, nil
			}
			return "conflict", existing.Id, "Bank reference exists in PayGate with a different amount", []string{existing.Id}, nil
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return "", "", "", nil, findErr
		}
	}

	candidates, findErr := reconciliationCandidates(app, row.AmountPaise, row.TransactionTime, now)
	if findErr != nil {
		return "", "", "", nil, findErr
	}
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.Id)
	}
	if len(candidates) == 1 {
		return "conflict", candidates[0].Id, "Exact amount matches one PayGate payment, but the bank reference was not linked", candidateIDs, nil
	}
	if len(candidates) > 1 {
		return "conflict", "", "Multiple historical PayGate payments are plausible for this statement row", candidateIDs, nil
	}
	return "unmatched", "", "No PayGate payment matches this statement credit", nil, nil
}

func reconciliationCandidates(app core.App, amount int64, transactionTime, now time.Time) ([]*core.Record, error) {
	if amount <= 0 {
		return nil, nil
	}
	if !transactionTime.IsZero() {
		return app.FindRecordsByFilter(
			"payments",
			"payable_amount = {:amount} && created_at <= {:createdBefore} && reuse_after >= {:at}",
			"-created_at", 10, 0,
			dbx.Params{
				"amount": amount, "at": formatDate(transactionTime),
				"createdBefore": formatDate(transactionTime.Add(payments.EvidenceTimestampTolerance)),
			},
		)
	}
	return app.FindRecordsByFilter(
		"payments", "payable_amount = {:amount} && reuse_after > {:now}",
		"-created_at", 10, 0, dbx.Params{"amount": amount, "now": formatDate(now)},
	)
}

func parseStatement(filename string, data []byte, location *time.Location) ([]statementRow, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	var table [][]string
	var err error
	switch ext {
	case ".csv", ".txt", ".tsv":
		table, err = parseDelimited(data)
	case ".xlsx":
		table, err = parseXLSX(data)
	default:
		return nil, fmt.Errorf("supported statement formats are CSV, TSV and XLSX")
	}
	if err != nil {
		return nil, err
	}
	return normalizeTable(table, location)
}

func parseDelimited(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	firstLine := string(data)
	if index := strings.IndexByte(firstLine, '\n'); index >= 0 {
		firstLine = firstLine[:index]
	}
	delimiter := ','
	if strings.Count(firstLine, "\t") > strings.Count(firstLine, ",") {
		delimiter = '\t'
	} else if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		delimiter = ';'
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	var rows [][]string
	for len(rows) <= MaxRows+20 {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV: %w", err)
		}
		if len(row) > MaxColumns {
			return nil, fmt.Errorf("statement has more than %d columns", MaxColumns)
		}
		rows = append(rows, row)
	}
	if len(rows) > MaxRows+20 {
		return nil, fmt.Errorf("statement has more than %d rows", MaxRows)
	}
	return rows, nil
}

func parseXLSX(data []byte) ([][]string, error) {
	if err := validateXLSXArchive(data); err != nil {
		return nil, err
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open XLSX: %w", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("XLSX contains no worksheets")
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read XLSX: %w", err)
	}
	if len(rows) > MaxRows+20 {
		return nil, fmt.Errorf("statement has more than %d rows", MaxRows)
	}
	for _, row := range rows {
		if len(row) > MaxColumns {
			return nil, fmt.Errorf("statement has more than %d columns", MaxColumns)
		}
	}
	return rows, nil
}

func validateXLSXArchive(data []byte) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid XLSX ZIP archive: %w", err)
	}
	if len(archive.File) > 5_000 {
		return errors.New("XLSX contains too many archive entries")
	}
	const maxEntryBytes = uint64(32 << 20)
	const maxExpandedBytes = uint64(64 << 20)
	var expanded uint64
	for _, file := range archive.File {
		clean := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return fmt.Errorf("XLSX contains unsafe archive path %q", file.Name)
		}
		if file.UncompressedSize64 > maxEntryBytes {
			return fmt.Errorf("XLSX entry %q expands beyond 32 MiB", file.Name)
		}
		if expanded > maxExpandedBytes-file.UncompressedSize64 {
			return errors.New("XLSX expands beyond 64 MiB")
		}
		expanded += file.UncompressedSize64
	}
	return nil
}

func normalizeTable(table [][]string, location *time.Location) ([]statementRow, error) {
	headerIndex, cols, headers, err := detectHeader(table)
	if err != nil {
		return nil, err
	}
	var result []statementRow
	for index := headerIndex + 1; index < len(table); index++ {
		row := table[index]
		if rowEmpty(row) {
			continue
		}
		statement, include := parseRow(index+1, row, cols, headers, location)
		if include {
			result = append(result, statement)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("statement contains no recognizable credit rows")
	}
	if len(result) > MaxRows {
		return nil, fmt.Errorf("statement has more than %d credit rows", MaxRows)
	}
	return result, nil
}

func detectHeader(table [][]string) (int, columns, []string, error) {
	limit := len(table)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(table[i]))
		for j, value := range table[i] {
			headers[j] = normalizeHeader(value)
		}
		cols := columns{
			date:        findColumn(headers, "date", "transaction date", "txn date", "value date", "timestamp", "transaction time"),
			credit:      findColumn(headers, "credit", "credit amount", "deposit", "deposit amount", "deposit amt", "cr amount"),
			debit:       findColumn(headers, "debit", "debit amount", "withdrawal", "withdrawal amount", "withdrawal amt", "dr amount"),
			amount:      findColumn(headers, "amount", "transaction amount", "txn amount"),
			kind:        findColumn(headers, "type", "transaction type", "txn type", "dr cr", "debit credit"),
			description: findColumn(headers, "description", "narration", "remarks", "particulars", "transaction details", "details"),
			rrn:         findColumn(headers, "rrn", "upi ref", "upi reference", "reference", "reference number", "utr", "transaction id", "txn id"),
		}
		if (cols.credit >= 0 || cols.amount >= 0) && (cols.description >= 0 || cols.rrn >= 0) {
			return i, cols, headers, nil
		}
	}
	return -1, columns{}, nil, errors.New("could not identify statement headers; include an amount/credit column and narration/reference column")
}

func parseRow(rowNumber int, row []string, cols columns, headers []string, location *time.Location) (statementRow, bool) {
	get := func(index int) string {
		if index < 0 || index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}
	creditText := get(cols.credit)
	amountText := creditText
	if cols.credit >= 0 {
		if creditText == "" || isZeroAmount(creditText) {
			return statementRow{}, false
		}
	} else {
		amountText = get(cols.amount)
		kind := strings.ToLower(get(cols.kind))
		if strings.Contains(kind, "debit") || kind == "dr" || strings.HasSuffix(kind, " dr") {
			return statementRow{}, false
		}
		if cols.debit >= 0 && get(cols.debit) != "" && !isZeroAmount(get(cols.debit)) {
			return statementRow{}, false
		}
	}
	parsed := statementRow{RowNumber: rowNumber, Description: get(cols.description), Raw: map[string]string{}}
	for i, value := range row {
		key := fmt.Sprintf("column_%d", i+1)
		if i < len(headers) && headers[i] != "" {
			key = headers[i]
		}
		parsed.Raw[key] = truncate(strings.TrimSpace(value), 4096)
	}
	amount, err := parseStatementAmount(amountText)
	if err != nil || amount <= 0 {
		parsed.InvalidReason = "Could not parse positive credit amount"
		return parsed, true
	}
	parsed.AmountPaise = amount
	parsed.TransactionTime = parseStatementDate(get(cols.date), location)
	parsed.RRN = normalizeReference(get(cols.rrn))
	if parsed.RRN == "" {
		parsed.RRN = sms.ExtractReference(parsed.Description)
	}
	return parsed, true
}

func parseStatementAmount(value string) (int64, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	for _, token := range []string{"INR", "RS.", "RS", "₹", " CR", "CR"} {
		value = strings.ReplaceAll(value, token, "")
	}
	value = strings.TrimSpace(value)
	return money.ParseAmount(value)
}

func parseStatementDate(value string, location *time.Location) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	formats := []string{
		time.RFC3339, "2006-01-02 15:04:05", "2006-01-02", "02/01/2006 15:04:05",
		"02/01/2006", "02-01-2006 15:04:05", "02-01-2006", "02 Jan 2006", "02-Jan-2006",
		"01/02/2006", "01/02/2006 15:04:05",
	}
	if location == nil {
		location = time.UTC
	}
	for _, format := range formats {
		if parsed, err := time.ParseInLocation(format, value, location); err == nil {
			return parsed.UTC()
		}
	}
	if serial, err := strconv.ParseFloat(value, 64); err == nil && serial > 1 {
		if parsed, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), location).UTC()
		}
	}
	return time.Time{}
}

func normalizeReference(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.Trim(value, "#:-.")
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ", ".", "", "/", " ", "(", " ", ")", " ")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func findColumn(headers []string, aliases ...string) int {
	for index, header := range headers {
		for _, alias := range aliases {
			if header == alias {
				return index
			}
		}
	}
	return -1
}

func rowEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func isZeroAmount(value string) bool {
	amount, err := parseStatementAmount(value)
	return err == nil && amount == 0
}

func severityForStatus(status string) string {
	if status == "conflict" {
		return "critical"
	}
	return "warning"
}

func formatDate(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000Z")
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func (s *Service) statementLocation() *time.Location {
	if s.StatementLocation == nil {
		return time.UTC
	}
	return s.StatementLocation
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
