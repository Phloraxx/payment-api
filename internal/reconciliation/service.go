package reconciliation

import (
	"archive/zip"
	"bytes"
	"context"
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
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
	"github.com/xuri/excelize/v2"
)

const (
	MaxFileBytes            = 10 << 20
	MaxRows                 = 10_000
	MaxColumns              = 64
	ReconciliationBatchSize = 250
)

type Service struct {
	Store             store.Database
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
	return &Service{Store: store.NewPocketBase(app), Reviews: reviewService, Alerts: alertService, Audit: auditService, Now: time.Now, StatementLocation: location}
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
	var existing *domain.ReconciliationRun
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var findErr error
		existing, findErr = uow.ReconciliationRuns().FindCompletedByHash(hash)
		return findErr
	})
	if err == nil {
		domainErr := domain.New("STATEMENT_ALREADY_IMPORTED", "this exact statement file was already imported", 409)
		domainErr.Details = map[string]any{"runId": existing.ID}
		return Result{}, domainErr
	}
	if !errors.Is(err, sql.ErrNoRows) {
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

	result := Result{RunID: run.ID, Status: "completed"}
	seenRRN := map[string]int{}
	for start := 0; start < len(rows); start += ReconciliationBatchSize {
		end := start + ReconciliationBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := s.persistRowsBatch(run.ID, rows[start:end], seenRRN, now, &result); err != nil {
			s.failRun(run, err, now)
			return Result{}, err
		}
	}
	if err := s.completeRun(run.ID, input.Actor, now, result); err != nil {
		s.failRun(run, err, now)
		return Result{}, err
	}
	if result.ConflictRows > 0 && s.Alerts != nil {
		_, _, _ = s.Alerts.Open(alerts.Input{
			Kind: "reconciliation_conflict", Severity: "warning", DedupeKey: "reconciliation:" + run.ID,
			Message: fmt.Sprintf("Statement reconciliation found %d conflicting rows", result.ConflictRows), Details: result,
		})
	}
	return result, nil
}

func (s *Service) persistRowsBatch(runID string, rows []statementRow, seenRRN map[string]int, now time.Time, result *Result) error {
	if len(rows) == 0 {
		return nil
	}
	if len(rows) > ReconciliationBatchSize {
		return fmt.Errorf("reconciliation batch exceeds %d rows", ReconciliationBatchSize)
	}
	delta := Result{}
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		for _, row := range rows {
			delta.TotalRows++
			status, paymentID, note, candidateIDs, err := s.classify(uow, row, seenRRN, now)
			if err != nil {
				return err
			}
			entry := &domain.ReconciliationEntry{
				RunID: runID, RowNumber: row.RowNumber, TransactionTime: row.TransactionTime,
				AmountPaise: row.AmountPaise, RRN: row.RRN, Description: truncate(row.Description, 4096),
				Status: status, PaymentID: paymentID, Notes: note, RawRow: row.Raw,
			}
			if err := uow.ReconciliationEntries().Create(entry); err != nil {
				return err
			}
			switch status {
			case "matched":
				delta.MatchedRows++
			case "unmatched":
				delta.UnmatchedRows++
			case "duplicate":
				delta.DuplicateRows++
			case "conflict":
				delta.ConflictRows++
			case "invalid":
				delta.InvalidRows++
			}
			needsReview := status == "conflict" || (status == "unmatched" && (row.RRN != "" || strings.Contains(strings.ToLower(row.Description), "upi")))
			if needsReview && s.Reviews != nil {
				caseID, err := s.Reviews.Open(uow, reviews.OpenInput{
					Kind: "reconciliation_conflict", Severity: severityForStatus(status),
					ReconciliationEntryID: entry.ID, PaymentID: paymentID,
					CandidatePaymentIDs: candidateIDs, Reason: note, OpenedAt: now,
				})
				if err != nil {
					return err
				}
				if caseID != "" {
					delta.ReviewCases++
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	addResult(result, delta)
	return nil
}

func (s *Service) completeRun(runID string, actor audit.Actor, now time.Time, result Result) error {
	return s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		run, err := uow.ReconciliationRuns().Get(runID)
		if err != nil {
			return err
		}
		run.Status = "completed"
		run.TotalRows, run.MatchedRows, run.UnmatchedRows = result.TotalRows, result.MatchedRows, result.UnmatchedRows
		run.DuplicateRows, run.ConflictRows, run.InvalidRows = result.DuplicateRows, result.ConflictRows, result.InvalidRows
		run.CompletedAt, run.Summary = now, result
		if err := uow.ReconciliationRuns().Save(run); err != nil {
			return err
		}
		if s.Audit != nil {
			if err := s.Audit.RecordUoW(uow, audit.Entry{Action: "reconciliation.import", Actor: actor, EntityType: "reconciliation_run", EntityID: runID, Summary: "Imported bank statement for reconciliation", Details: result, OccurredAt: now}); err != nil {
				return err
			}
		}
		return nil
	})
}

func addResult(target *Result, delta Result) {
	if target == nil {
		return
	}
	target.TotalRows += delta.TotalRows
	target.MatchedRows += delta.MatchedRows
	target.UnmatchedRows += delta.UnmatchedRows
	target.DuplicateRows += delta.DuplicateRows
	target.ConflictRows += delta.ConflictRows
	target.InvalidRows += delta.InvalidRows
	target.ReviewCases += delta.ReviewCases
}

func (s *Service) createRun(input ImportInput, hash string, now time.Time) (*domain.ReconciliationRun, error) {
	run := &domain.ReconciliationRun{Filename: truncate(input.Filename, 255), SHA256: hash, Status: "processing", CreatedBy: input.Actor.ID, StartedAt: now}
	if err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error { return uow.ReconciliationRuns().Create(run) }); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) failRun(run *domain.ReconciliationRun, cause error, now time.Time) {
	if run == nil {
		return
	}
	_ = s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		stored, err := uow.ReconciliationRuns().Get(run.ID)
		if err != nil {
			return err
		}
		stored.Status, stored.Error, stored.CompletedAt = "failed", truncate(cause.Error(), 4096), now
		return uow.ReconciliationRuns().Save(stored)
	})
}

func (s *Service) classify(uow store.UnitOfWork, row statementRow, seenRRN map[string]int, now time.Time) (status, paymentID, note string, candidateIDs []string, err error) {
	if row.InvalidReason != "" || row.AmountPaise <= 0 {
		return "invalid", "", row.InvalidReason, nil, nil
	}
	paymentsRepo := uow.Payments()
	if row.RRN != "" {
		if firstRow, ok := seenRRN[row.RRN]; ok {
			return "duplicate", "", fmt.Sprintf("Duplicate bank reference already appeared on row %d", firstRow), nil, nil
		}
		seenRRN[row.RRN] = row.RowNumber
		existing, findErr := paymentsRepo.FindByEvidenceReference(domain.EvidenceReferenceRRN, row.RRN)
		if findErr == nil {
			if existing.Account != domain.PaymentAccountKotak {
				return "conflict", existing.ID, "Bank reference belongs to a non-Kotak payment", []string{existing.ID}, nil
			}
			if existing.PayablePaise == row.AmountPaise {
				return "matched", existing.ID, "Exact amount and bank reference match", []string{existing.ID}, nil
			}
			return "conflict", existing.ID, "Bank reference exists in PayGate with a different amount", []string{existing.ID}, nil
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return "", "", "", nil, findErr
		}
	}
	candidates, findErr := paymentsRepo.FindReconciliationCandidates(domain.PaymentAccountKotak, row.AmountPaise, row.TransactionTime, now, 10)
	if findErr != nil {
		return "", "", "", nil, findErr
	}
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.ID)
	}
	if len(candidates) == 1 {
		return "conflict", candidates[0].ID, "Exact amount matches one PayGate payment, but the bank reference was not linked", candidateIDs, nil
	}
	if len(candidates) > 1 {
		return "conflict", "", "Multiple historical PayGate payments are plausible for this statement row", candidateIDs, nil
	}
	return "unmatched", "", "No PayGate payment matches this statement credit", nil, nil
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
