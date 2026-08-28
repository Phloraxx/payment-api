package reconciliation

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/reviews"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/xuri/excelize/v2"
)

func reconciliationTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time, audit.Actor) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	operator := core.NewRecord(users)
	operator.Id = "reconop00000001"
	operator.SetEmail("recon@example.com")
	operator.SetPassword("test-password-123")
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	auditService := audit.NewService(app)
	auditService.Now = func() time.Time { return now }
	alertService := alerts.NewService(app)
	alertService.Now = func() time.Time { return now }
	reviewService := reviews.NewService(app, paymentService, auditService)
	reviewService.Now = func() time.Time { return now }
	service := NewService(app, reviewService, alertService, auditService)
	service.Now = func() time.Time { return now }
	return service, paymentService, app, &now, audit.Actor{ID: operator.Id, Email: operator.Email()}
}

func TestImportCSVMatchesExistingPaymentByReferenceAndAmount(t *testing.T) {
	service, paymentService, app, now, actor := reconciliationTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	_, err = paymentService.Match(domain.ParsedSMS{
		AmountPaise: payment.PayablePaise,
		RRN:         "606703736479",
		OccurredAt:  now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	csv := "Date,Credit Amount,Narration,RRN\n01/08/2026,100.01,UPI credit from payer,606703736479\n"
	result, err := service.Import(ImportInput{Filename: "statement.csv", Data: []byte(csv), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchedRows != 1 || result.TotalRows != 1 || result.ReviewCases != 0 {
		t.Fatalf("result=%+v", result)
	}
	entries, err := app.FindAllRecords("reconciliation_entries")
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	if entries[0].GetString("status") != "matched" || entries[0].GetString("payment") != payment.ID {
		t.Fatalf("entry status=%s payment=%s", entries[0].GetString("status"), entries[0].GetString("payment"))
	}
}

func TestImportCreatesReviewAndAlertForAmountOnlyConflict(t *testing.T) {
	service, paymentService, app, now, actor := reconciliationTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 200})
	if err != nil {
		t.Fatal(err)
	}
	csv := fmt.Sprintf("Transaction Date,Credit,Narration,UPI Reference\n%s,200.01,UPI PAYMENT RECEIVED,999988887777\n", now.Add(time.Minute).Format(time.RFC3339))
	result, err := service.Import(ImportInput{Filename: "hdfc.csv", Data: []byte(csv), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if result.ConflictRows != 1 || result.ReviewCases != 1 {
		t.Fatalf("result=%+v", result)
	}
	cases, _ := app.FindAllRecords("review_cases")
	if len(cases) != 1 || cases[0].GetString("payment") != payment.ID || cases[0].GetString("status") != "open" {
		t.Fatalf("cases=%v", cases)
	}
	alerts, _ := app.FindAllRecords("alerts")
	if len(alerts) != 1 || alerts[0].GetString("kind") != "reconciliation_conflict" {
		t.Fatalf("alerts=%v", alerts)
	}
}

func TestImportIgnoresDebitRowsAndDetectsDuplicateReference(t *testing.T) {
	service, _, app, _, actor := reconciliationTestService(t)
	csv := "Date,Debit,Credit,Narration,RRN\n01/08/2026,50.00,,UPI debit,111122223333\n01/08/2026,,75.50,UPI credit,444455556666\n01/08/2026,,75.50,UPI duplicate,444455556666\n"
	result, err := service.Import(ImportInput{Filename: "mixed.csv", Data: []byte(csv), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 2 || result.UnmatchedRows != 1 || result.DuplicateRows != 1 {
		t.Fatalf("result=%+v", result)
	}
	entries, _ := app.FindAllRecords("reconciliation_entries")
	if len(entries) != 2 {
		t.Fatalf("entries=%d", len(entries))
	}
}

func TestImportPersistsLargeStatementInBatchesAndTracksDuplicateAcrossBatches(t *testing.T) {
	service, _, app, _, actor := reconciliationTestService(t)
	var csv strings.Builder
	csv.WriteString("Date,Credit,Narration,RRN\n")
	for i := 1; i <= ReconciliationBatchSize*2+1; i++ {
		rrn := fmt.Sprintf("%012d", i)
		if i == ReconciliationBatchSize+1 {
			rrn = "000000000001" // duplicate row 1, but in the next transaction batch
		}
		fmt.Fprintf(&csv, "01/08/2026,10.01,UPI credit,%s\n", rrn)
	}
	result, err := service.Import(ImportInput{Filename: "large.csv", Data: []byte(csv.String()), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	wantRows := ReconciliationBatchSize*2 + 1
	if result.TotalRows != wantRows || result.DuplicateRows != 1 || result.UnmatchedRows != wantRows-1 {
		t.Fatalf("result=%+v", result)
	}
	entries, err := app.FindAllRecords("reconciliation_entries")
	if err != nil || len(entries) != wantRows {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	runs, err := app.FindAllRecords("reconciliation_runs")
	if err != nil || len(runs) != 1 || runs[0].GetString("status") != "completed" || runs[0].GetInt("total_rows") != wantRows {
		t.Fatalf("runs=%d status=%v total=%v err=%v", len(runs), func() string {
			if len(runs) == 0 {
				return ""
			}
			return runs[0].GetString("status")
		}(), func() int {
			if len(runs) == 0 {
				return 0
			}
			return runs[0].GetInt("total_rows")
		}(), err)
	}
}

func TestImportRejectsSameFileHashAfterCompletedRun(t *testing.T) {
	service, _, _, _, actor := reconciliationTestService(t)
	data := []byte("Date,Credit,Narration,RRN\n01/08/2026,10.01,UPI credit,123456789012\n")
	if _, err := service.Import(ImportInput{Filename: "same.csv", Data: data, Actor: actor}); err != nil {
		t.Fatal(err)
	}
	_, err := service.Import(ImportInput{Filename: "renamed.csv", Data: data, Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "STATEMENT_ALREADY_IMPORTED" {
		t.Fatalf("error=%v", err)
	}
}

func TestImportXLSXReadsFirstWorksheet(t *testing.T) {
	service, _, _, _, actor := reconciliationTestService(t)
	book := excelize.NewFile()
	sheet := book.GetSheetName(0)
	rows := [][]any{
		{"Value Date", "Deposit Amount", "Transaction Details", "UTR"},
		{"01-Aug-2026", 25.25, "UPI received", "ABCDEF123456"},
	}
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := book.SetCellValue(sheet, cell, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	var buffer bytes.Buffer
	if err := book.Write(&buffer); err != nil {
		t.Fatal(err)
	}
	result, err := service.Import(ImportInput{Filename: "statement.xlsx", Data: buffer.Bytes(), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 1 || result.UnmatchedRows != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestImportRejectsUnknownHeadersAndUnsupportedExtension(t *testing.T) {
	service, _, app, _, actor := reconciliationTestService(t)
	_, err := service.Import(ImportInput{Filename: "bad.csv", Data: []byte("foo,bar\na,b\n"), Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "STATEMENT_PARSE_FAILED" {
		t.Fatalf("header error=%v", err)
	}
	runs, _ := app.FindAllRecords("reconciliation_runs")
	if len(runs) != 1 || runs[0].GetString("status") != "failed" {
		t.Fatalf("runs=%v", runs)
	}
	_, err = service.Import(ImportInput{Filename: "statement.pdf", Data: []byte("not a statement"), Actor: actor})
	if !errors.As(err, &domainErr) || domainErr.Code != "STATEMENT_PARSE_FAILED" {
		t.Fatalf("extension error=%v", err)
	}
}

func TestXLSXArchiveValidationRejectsUnsafePaths(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("../evil.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("evil"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validateXLSXArchive(buffer.Bytes()); err == nil || !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("error=%v", err)
	}
}

func TestXLSXArchiveValidationRejectsNonZipData(t *testing.T) {
	if err := validateXLSXArchive([]byte("not a zip")); err == nil {
		t.Fatal("expected invalid ZIP error")
	}
}

func TestParseStatementDateUsesIndianBankTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	got := parseStatementDate("01/08/2026 14:00:00", location)
	want := time.Date(2026, 8, 1, 8, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parsed=%s want=%s", got, want)
	}
}
