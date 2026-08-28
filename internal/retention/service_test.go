package retention

import (
	"fmt"
	"strings"
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
	smsRecord.Set("payment_account", "kotak")
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

	emailCollection, _ := app.FindCollectionByNameOrId("email_events")
	emailRecord := core.NewRecord(emailCollection)
	emailRecord.Set("source", "cloudflare_email")
	emailRecord.Set("payment_account", "slice")
	emailRecord.Set("source_event_id", "private-message-id")
	emailRecord.Set("envelope_sender", "noreply@slice.bank.in")
	emailRecord.Set("recipient", "payments@example.org")
	emailRecord.Set("sender", "noreply@slice.bank.in")
	emailRecord.Set("subject", "Received ₹100.01 via UPI")
	emailRecord.Set("body", "Received ₹100.01 from private@upi UPI Ref 123456789012")
	emailRecord.Set("auth_result", "mx.cloudflare.net; dkim=pass header.d=slice.bank.in")
	emailRecord.Set("upi_id", "private@upi")
	emailRecord.Set("payer_name", "Private Payer")
	emailRecord.Set("rrn", "123456789012")
	emailRecord.Set("amount", 10001)
	emailRecord.Set("processing_status", "matched")
	emailRecord.Set("raw_payload", map[string]any{"private": true})
	emailRecord.Set("message_time", now.Add(-91*24*time.Hour))
	emailRecord.Set("received_at", now.Add(-91*24*time.Hour))
	if err := app.SaveNoValidate(emailRecord); err != nil {
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
		EmailRawRetention:          90 * 24 * time.Hour,
		ReconciliationRawRetention: 365 * 24 * time.Hour,
		AuditRetention:             730 * 24 * time.Hour,
	})
	service.Now = func() time.Time { return now }
	result, err := service.Run()
	if err != nil {
		t.Fatal(err)
	}
	if result.SMSEventsRedacted != 1 || result.EmailEventsRedacted != 1 || result.ReconciliationEntriesRedacted != 1 || result.AuditEventsDeleted != 1 {
		t.Fatalf("result=%+v", result)
	}
	storedEmail, _ := app.FindRecordById("email_events", emailRecord.Id)
	if storedEmail.GetString("body") != redactedMarker || storedEmail.GetString("subject") != redactedMarker || storedEmail.GetString("sender") != "" || storedEmail.GetString("rrn") != "123456789012" {
		t.Fatalf("stored email body=%q subject=%q sender=%q rrn=%q", storedEmail.GetString("body"), storedEmail.GetString("subject"), storedEmail.GetString("sender"), storedEmail.GetString("rrn"))
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
		record.Set("payment_account", "kotak")
		record.Set("body", "Received Rs.1.01 UPI Ref 12345678")
		record.Set("processing_status", "matched")
		record.Set("message_time", now.Add(-91*24*time.Hour))
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(app, config.Config{
		RetentionEnabled: true, SMSRawRetention: 90 * 24 * time.Hour,
		EmailRawRetention:          90 * 24 * time.Hour,
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

func TestRetentionRedactsPaytmAndRelayRawEvidence(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	devices, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(devices)
	device.Set("device_id", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	device.Set("name", "Phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}

	notifications, _ := app.FindCollectionByNameOrId("notification_events")
	n := core.NewRecord(notifications)
	n.Set("source", "android_relay")
	n.Set("source_event_id", "retention-paytm")
	n.Set("app_package", "com.paytm.business")
	n.Set("title", "Payment Received")
	n.Set("body", "₹1.23 Received from Test User")
	n.Set("big_text", "private raw detail")
	n.Set("notification_time", time.Now().UTC())
	n.Set("payment_account", "paytm")
	n.Set("amount", 123)
	n.Set("payer_name", "Test User")
	n.Set("processing_status", "unmatched")
	n.Set("raw_payload", map[string]any{"private": "value"})
	if err := app.Save(n); err != nil {
		t.Fatal(err)
	}

	relays, _ := app.FindCollectionByNameOrId("relay_events")
	r := core.NewRecord(relays)
	r.Set("device", device.Id)
	r.Set("event_id", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	r.Set("kind", "notification")
	r.Set("app_package", "com.paytm.business")
	r.Set("title", "Payment Received")
	r.Set("body", "private body")
	r.Set("custom_texts", []string{"₹1.23 Received from Test User"})
	r.Set("processing_status", "forwarded")
	r.Set("raw_payload", map[string]any{"private": "value"})
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		RetentionEnabled:              true,
		SMSRawRetention:               365 * 24 * time.Hour,
		EmailRawRetention:             365 * 24 * time.Hour,
		ReconciliationRawRetention:    365 * 24 * time.Hour,
		AuditRetention:                365 * 24 * time.Hour,
		PaytmNotificationRawRetention: 24 * time.Hour,
		RelayRawRetention:             24 * time.Hour,
	}
	service := NewService(app, cfg)
	service.Now = func() time.Time { return time.Now().UTC().Add(48 * time.Hour) }
	result, err := service.Run()
	if err != nil {
		t.Fatal(err)
	}
	if result.PaytmNotificationsRedacted != 1 || result.RelayEventsRedacted != 1 {
		t.Fatalf("retention result = %+v", result)
	}

	storedN, _ := app.FindRecordById("notification_events", n.Id)
	if storedN.GetString("body") != redactedMarker || storedN.GetString("payer_name") != "" || strings.Contains(fmt.Sprint(storedN.Get("raw_payload")), "private") || storedN.GetDateTime("raw_redacted_at").Time().IsZero() {
		t.Fatalf("notification raw evidence not redacted: %+v", storedN)
	}
	if storedN.GetInt("amount") != 123 || storedN.GetString("processing_status") != "unmatched" {
		t.Fatalf("notification matching metadata must remain: amount=%d status=%s", storedN.GetInt("amount"), storedN.GetString("processing_status"))
	}

	storedR, _ := app.FindRecordById("relay_events", r.Id)
	if storedR.GetString("body") != redactedMarker || strings.Contains(fmt.Sprint(storedR.Get("raw_payload")), "private") || storedR.GetDateTime("raw_redacted_at").Time().IsZero() {
		t.Fatalf("relay raw evidence not redacted: %+v", storedR)
	}
	if storedR.GetString("processing_status") != "forwarded" || storedR.GetString("event_id") == "" {
		t.Fatalf("relay audit metadata must remain")
	}
}
