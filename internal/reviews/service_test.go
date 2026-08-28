package reviews

import (
	"errors"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/paymentemail"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/Phloraxx/payment-api/internal/store"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func reviewTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time) {
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
	operator.Id = "operator0000001"
	operator.SetEmail("operator@example.com")
	operator.SetPassword("test-password-123")
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	cfg := config.Config{
		DefaultPaymentAccount: "kotak", KotakUPIID: "operator@kotak", SliceUPIID: "operator@slice",
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
	}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	auditService := audit.NewService(app)
	auditService.Now = func() time.Time { return now }
	service := NewService(app, paymentService, auditService)
	service.Now = func() time.Time { return now }
	return service, paymentService, app, &now
}

func createSMSEvent(t *testing.T, app core.App, amount int64, rrn string, at time.Time) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("sms_events")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("source", "gmessages")
	record.Set("body", "sanitized bank credit evidence")
	record.Set("payment_account", "kotak")
	record.Set("message_time", at)
	record.Set("amount", amount)
	record.Set("rrn", rrn)
	record.Set("processing_status", "unmatched")
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record.Id
}

func createEmailEvent(t *testing.T, app core.App, amount int64, rrn string, at time.Time) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("email_events")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("source", "cloudflare_email")
	record.Set("body", "sanitized bank credit email evidence")
	record.Set("payment_account", "slice")
	record.Set("received_at", at)
	record.Set("message_time", at)
	record.Set("amount", amount)
	record.Set("rrn", rrn)
	record.Set("processing_status", "unmatched")
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record.Id
}

func TestResolveManualMatchPersistsPaymentEventCaseAndAudit(t *testing.T) {
	service, paymentService, app, now := reviewTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	eventID := createSMSEvent(t, app, payment.PayablePaise, "", now.Add(time.Minute))
	caseID, err := service.OpenSMSReview(store.NewPocketBaseUnit(app), sms.ReviewInput{
		Kind: "missing_rrn", Severity: "warning", SMSEventID: eventID,
		CandidatePaymentIDs: []string{payment.ID}, Reason: "missing reference", OpenedAt: *now,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Resolve(ResolveInput{
		CaseID: caseID, Action: "manual_match", PaymentID: payment.ID,
		BankReference: "123456789012", Note: "Verified against the bank statement.",
		Actor: audit.Actor{ID: "operator0000001", Email: "operator@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "resolved" || result.PaymentID != payment.ID || result.Action != "marked_paid" {
		t.Fatalf("result=%+v", result)
	}
	stored, err := paymentService.Get(payment.ID)
	if err != nil || stored.Status != domain.StatusPaid || stored.RRN != "123456789012" {
		t.Fatalf("payment=%+v err=%v", stored, err)
	}
	event, _ := app.FindRecordById("sms_events", eventID)
	if event.GetString("processing_status") != "matched" || event.GetString("matched_payment") != payment.ID {
		t.Fatalf("event status=%s payment=%s", event.GetString("processing_status"), event.GetString("matched_payment"))
	}
	caseRecord, _ := app.FindRecordById("review_cases", caseID)
	if caseRecord.GetString("status") != "resolved" || caseRecord.GetString("resolved_by") != "operator0000001" {
		t.Fatalf("case status=%s actor=%s", caseRecord.GetString("status"), caseRecord.GetString("resolved_by"))
	}
	if count, _ := app.CountRecords("audit_events"); count != 1 {
		t.Fatalf("audit count=%d", count)
	}
}

func TestManualMatchRejectsDifferentAmount(t *testing.T) {
	service, paymentService, app, now := reviewTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 200})
	if err != nil {
		t.Fatal(err)
	}
	eventID := createSMSEvent(t, app, payment.PayablePaise+1, "999988887777", *now)
	caseID, err := service.OpenSMSReview(store.NewPocketBaseUnit(app), sms.ReviewInput{Kind: "unmatched", Severity: "warning", SMSEventID: eventID, Reason: "amount mismatch"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Resolve(ResolveInput{CaseID: caseID, Action: "manual_match", PaymentID: payment.ID, Note: "Attempted reviewed match.", Actor: audit.Actor{ID: "operator0000001", Email: "operator@example.com"}})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "MANUAL_AMOUNT_MISMATCH" {
		t.Fatalf("error=%v", err)
	}
	caseRecord, _ := app.FindRecordById("review_cases", caseID)
	if caseRecord.GetString("status") != "open" {
		t.Fatalf("case status=%s", caseRecord.GetString("status"))
	}
}

func TestResolveManualEmailMatchUpdatesEmailEvidence(t *testing.T) {
	service, paymentService, app, now := reviewTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 300, PaymentAccount: "slice"})
	if err != nil {
		t.Fatal(err)
	}
	eventID := createEmailEvent(t, app, payment.PayablePaise, "", now.Add(time.Minute))
	caseID, err := service.OpenEmailReview(store.NewPocketBaseUnit(app), paymentemail.ReviewInput{
		Kind: "missing_rrn", Severity: "warning", EmailEventID: eventID,
		CandidatePaymentIDs: []string{payment.ID}, Reason: "missing reference", OpenedAt: *now,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Resolve(ResolveInput{
		CaseID: caseID, Action: "manual_match", PaymentID: payment.ID,
		BankReference: "777766665555", Note: "Verified against the bank statement.",
		Actor: audit.Actor{ID: "operator0000001", Email: "operator@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "resolved" || result.PaymentID != payment.ID {
		t.Fatalf("result=%+v", result)
	}
	event, _ := app.FindRecordById("email_events", eventID)
	if event.GetString("processing_status") != "matched" || event.GetString("matched_payment") != payment.ID || event.GetString("rrn") != "777766665555" {
		t.Fatalf("email status=%s payment=%s rrn=%s", event.GetString("processing_status"), event.GetString("matched_payment"), event.GetString("rrn"))
	}
}

func TestOpenSMSReviewIsIdempotentPerEvidenceEvent(t *testing.T) {
	service, _, app, now := reviewTestService(t)
	eventID := createSMSEvent(t, app, 10001, "", *now)
	first, err := service.OpenSMSReview(store.NewPocketBaseUnit(app), sms.ReviewInput{Kind: "missing_rrn", Severity: "warning", SMSEventID: eventID, Reason: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.OpenSMSReview(store.NewPocketBaseUnit(app), sms.ReviewInput{Kind: "missing_rrn", Severity: "warning", SMSEventID: eventID, Reason: "missing again"})
	if err != nil || first != second {
		t.Fatalf("first=%s second=%s err=%v", first, second, err)
	}
}
