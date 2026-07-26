package sms

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func smsTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create PocketBase test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	service := NewService(app, paymentService)
	service.Now = func() time.Time { return now }
	return service, paymentService, app, &now
}

func TestIngestMatchesExactBankSMSAndPersistsEvidence(t *testing.T) {
	service, paymentService, app, _ := smsTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.Ingest(Input{
		Source: "android_webhook", SourceEventID: "sms-1", Sender: "VK-KOTAKB",
		Body:        "Confirmed payment for Received Rs.100.01 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736479.",
		MessageTime: time.Date(2026, 7, 25, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Status != "matched" || result.Action != "marked_paid" || result.PaymentID != payment.ID {
		t.Fatalf("Ingest() = %+v", result)
	}
	stored, err := paymentService.Get(payment.ID)
	if err != nil || stored.Status != domain.StatusPaid || stored.RRN != "606703736479" {
		t.Fatalf("stored payment = %+v, %v", stored, err)
	}
	event, err := app.FindRecordById("sms_events", result.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if event.GetString("processing_status") != "matched" || event.GetString("matched_payment") != payment.ID {
		t.Fatalf("event state = status=%s payment=%s", event.GetString("processing_status"), event.GetString("matched_payment"))
	}
}

func TestIngestDedupesBySourceAndSourceEventID(t *testing.T) {
	service, paymentService, app, _ := smsTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	body := "Received Rs.100.01 from user@oksbi. UPI Ref:123456789012"
	first, err := service.Ingest(Input{Source: "android_webhook", SourceEventID: "same", Body: body})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Ingest(Input{Source: "android_webhook", SourceEventID: "same", Body: body})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate || second.Action != "duplicate_event" || second.EventID != first.EventID {
		t.Fatalf("duplicate result = %+v; first=%+v", second, first)
	}
	count, err := app.CountRecords("sms_events")
	if err != nil || count != 1 {
		t.Fatalf("sms event count = %d, %v; want 1", count, err)
	}
	stored, _ := paymentService.Get(payment.ID)
	if stored.Status != domain.StatusPaid {
		t.Fatalf("payment status = %s", stored.Status)
	}

	// The same provider ID from another connector is a different source event,
	// but the RRN makes the payment evidence itself idempotent.
	third, err := service.Ingest(Input{Source: "gmessages", SourceEventID: "same", Body: body})
	if err != nil {
		t.Fatal(err)
	}
	if !third.Duplicate || third.Action != "duplicate_rrn" {
		t.Fatalf("cross-source duplicate = %+v", third)
	}
	count, _ = app.CountRecords("sms_events")
	if count != 2 {
		t.Fatalf("sms event count = %d; want 2", count)
	}
}

func TestIngestIgnoresUnrelatedMessagesButKeepsAuditRecord(t *testing.T) {
	service, _, app, _ := smsTestService(t)
	result, err := service.Ingest(Input{Source: "android_webhook", SourceEventID: "otp-1", Body: "Your OTP is 123456 for Rs.100"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || result.Action != "ignored_non_bank_sms" {
		t.Fatalf("result = %+v", result)
	}
	event, err := app.FindRecordById("sms_events", result.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if event.GetString("processing_status") != "ignored" {
		t.Fatalf("event status = %s", event.GetString("processing_status"))
	}
}

func TestIngestPersistsMissingRRNFailure(t *testing.T) {
	service, _, app, _ := smsTestService(t)
	result, err := service.Ingest(Input{Source: "android_webhook", SourceEventID: "missing-rrn", Body: "Received Rs.100.01 from user@oksbi"})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "SMS_MISSING_RRN" {
		t.Fatalf("error = %v", err)
	}
	event, findErr := app.FindRecordById("sms_events", result.EventID)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if event.GetString("processing_status") != "error" {
		t.Fatalf("event status = %s", event.GetString("processing_status"))
	}
}

func TestIngestUnmatchedBankCreditDoesNotMutatePayments(t *testing.T) {
	service, _, app, _ := smsTestService(t)
	result, err := service.Ingest(Input{Source: "android_webhook", SourceEventID: "unmatched", Body: "Received Rs.777.77 from user@oksbi UPI Ref:777788889999"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmatched" || result.PaymentID != "" {
		t.Fatalf("result = %+v", result)
	}
	count, _ := app.CountRecords("payments")
	if count != 0 {
		t.Fatalf("payments count = %d; want 0", count)
	}
}

func TestDelayedOldSMSCannotConfirmReusedAmount(t *testing.T) {
	service, paymentService, _, now := smsTestService(t)
	first, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 600})
	if err != nil {
		t.Fatal(err)
	}
	oldMessageTime := now.Add(2 * time.Minute)

	// Move beyond expiry + quarantine so the deterministic .01 slot can be reused.
	*now = now.Add(24*time.Hour + 6*time.Minute + time.Second)
	second, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 600})
	if err != nil {
		t.Fatal(err)
	}
	if second.PayablePaise != first.PayablePaise {
		t.Fatalf("test requires amount reuse: first=%d second=%d", first.PayablePaise, second.PayablePaise)
	}

	result, err := service.Ingest(Input{
		Source:        "gmessages",
		SourceEventID: "old-catchup-message",
		Body:          "Received Rs.600.01 from oldpayer@oksbi UPI Ref:121212121212",
		MessageTime:   oldMessageTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmatched" || result.PaymentID != "" {
		t.Fatalf("old catch-up SMS result = %+v; want unmatched", result)
	}
	storedSecond, err := paymentService.Get(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedSecond.Status != domain.StatusPending {
		t.Fatalf("reused payment status = %s; old SMS must not confirm it", storedSecond.Status)
	}
}

func TestIngestRejectsUnknownSource(t *testing.T) {
	service, _, app, _ := smsTestService(t)
	_, err := service.Ingest(Input{Source: "unknown_connector", Body: "Received Rs.10.01 UPI Ref:123456789012"})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "INVALID_SMS_SOURCE" {
		t.Fatalf("error = %v; want INVALID_SMS_SOURCE", err)
	}
	count, countErr := app.CountRecords("sms_events")
	if countErr != nil || count != 0 {
		t.Fatalf("sms event count = %d, %v; invalid source must not persist", count, countErr)
	}
}

func TestIngestRejectsValuesThatCannotFitStorage(t *testing.T) {
	service, _, app, _ := smsTestService(t)
	cases := []struct {
		name  string
		input Input
	}{
		{name: "source event id", input: Input{Source: "android_webhook", SourceEventID: strings.Repeat("e", 256), Body: "Received Rs.10.01 UPI Ref:123456789012"}},
		{name: "sender", input: Input{Source: "android_webhook", Sender: strings.Repeat("s", 256), Body: "Received Rs.10.01 UPI Ref:123456789012"}},
		{name: "body", input: Input{Source: "android_webhook", Body: strings.Repeat("x", 64*1024+1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Ingest(tc.input)
			var domainErr *domain.Error
			if !errors.As(err, &domainErr) || domainErr.Code != "INVALID_SMS" {
				t.Fatalf("error = %v; want INVALID_SMS", err)
			}
		})
	}
	count, err := app.CountRecords("sms_events")
	if err != nil || count != 0 {
		t.Fatalf("sms event count = %d, %v; invalid records must not persist", count, err)
	}
}
