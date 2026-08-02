package retention

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRunRedactsSensitiveEvidenceAndDeletesExpiredAudit(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)

	smsCollection, _ := app.FindCollectionByNameOrId("sms_events")
	smsRecord := core.NewRecord(smsCollection)
	smsRecord.Set("source", "gmessages")
	smsRecord.Set("body", "Received Rs.100.01 from private@upi UPI Ref 123456789012")
	smsRecord.Set("sender", "BANK")
	smsRecord.Set("upi_id", "private@upi")
	smsRecord.Set("payer_name", "Private Payer")
	smsRecord.Set("rrn", "123456789012")
	smsRecord.Set("amount", 10001)
	smsRecord.Set("processing_status", "matched")
	smsRecord.Set("raw_payload", map[string]any{"private": true})
	smsRecord.Set("message_time", now.Add(-91*24*time.Hour))
	if err := app.SaveNoValidate(smsRecord); err != nil {
		t.Fatal(err)
	}

	runCollection, _ := app.FindCollectionByNameOrId("reconciliation_runs")
	run := core.NewRecord(runCollection)
	run.Set("filename", "statement.csv")
	run.Set("sha256", "abc")
	run.Set("status", "completed")
	run.Set("started_at", now.Add(-400*24*time.Hour))
	if err := app.Save(run); err != nil {
		t.Fatal(err)
	}
	entryCollection, _ := app.FindCollectionByNameOrId("reconciliation_entries")
	entry := core.NewRecord(entryCollection)
	entry.Set("run", run.Id)
	entry.Set("row_number", 1)
	entry.Set("amount", 10001)
	entry.Set("status", "unmatched")
	entry.Set("description", "private narration")
	entry.Set("raw_row", map[string]any{"account": "private"})
	entry.Set("transaction_time", now.Add(-366*24*time.Hour))
	if err := app.SaveNoValidate(entry); err != nil {
		t.Fatal(err)
	}

	auditCollection, _ := app.FindCollectionByNameOrId("audit_events")
	auditRecord := core.NewRecord(auditCollection)
	auditRecord.Set("action", "old.action")
	auditRecord.Set("entity_type", "payment")
	auditRecord.Set("entity_id", "payment-1")
	auditRecord.Set("summary", "old audit")
	auditRecord.Set("occurred_at", now.Add(-731*24*time.Hour))
	if err := app.Save(auditRecord); err != nil {
		t.Fatal(err)
	}

	service := NewService(app, config.Config{
		RetentionEnabled:           true,
		SMSRawRetention:            90 * 24 * time.Hour,
		ReconciliationRawRetention: 365 * 24 * time.Hour,
		AuditRetention:             730 * 24 * time.Hour,
	})
	service.Now = func() time.Time { return now }
	result, err := service.Run()
	if err != nil {
		t.Fatal(err)
	}
	if result.SMSEventsRedacted != 1 || result.ReconciliationEntriesRedacted != 1 || result.AuditEventsDeleted != 1 {
		t.Fatalf("result=%+v", result)
	}
	storedSMS, _ := app.FindRecordById("sms_events", smsRecord.Id)
	if storedSMS.GetString("body") != redactedMarker || storedSMS.GetString("sender") != "" || storedSMS.GetString("rrn") != "123456789012" {
		t.Fatalf("stored sms body=%q sender=%q rrn=%q", storedSMS.GetString("body"), storedSMS.GetString("sender"), storedSMS.GetString("rrn"))
	}
	storedEntry, _ := app.FindRecordById("reconciliation_entries", entry.Id)
	if storedEntry.GetString("description") != redactedMarker || storedEntry.GetInt("amount") != 10001 {
		t.Fatalf("stored entry description=%q amount=%d", storedEntry.GetString("description"), storedEntry.GetInt("amount"))
	}
	if count, _ := app.CountRecords("audit_events"); count != 0 {
		t.Fatalf("audit count=%d", count)
	}
}

func TestRunProcessesMoreThanOneRetentionBatch(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	collection, err := app.FindCollectionByNameOrId("sms_events")
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < retentionBatchSize+1; index++ {
		record := core.NewRecord(collection)
		record.Set("source", "manual")
		record.Set("body", "Received Rs.1.01 UPI Ref 12345678")
		record.Set("processing_status", "matched")
		record.Set("message_time", now.Add(-91*24*time.Hour))
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(app, config.Config{
		RetentionEnabled: true, SMSRawRetention: 90 * 24 * time.Hour,
		ReconciliationRawRetention: 365 * 24 * time.Hour, AuditRetention: 730 * 24 * time.Hour,
	})
	service.Now = func() time.Time { return now }
	result, err := service.Run()
	if err != nil {
		t.Fatal(err)
	}
	if result.SMSEventsRedacted != retentionBatchSize+1 {
		t.Fatalf("redacted=%d", result.SMSEventsRedacted)
	}
	remaining, err := app.CountRecords("sms_events", dbx.NewExp("body != {:redacted}", dbx.Params{"redacted": redactedMarker}))
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("unredacted=%d", remaining)
	}
}
