package paymentemail

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type reviewRecorder struct {
	inputs []ReviewInput
}

func (r *reviewRecorder) OpenEmailReviewInApp(_ core.App, input ReviewInput) (string, error) {
	r.inputs = append(r.inputs, input)
	return "review-case", nil
}

func emailTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time, *reviewRecorder) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{
		DefaultPaymentAccount: "kotak", KotakUPIID: "operator@kotak", SliceUPIID: "operator@slice",
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
	}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	reviews := &reviewRecorder{}
	service := NewService(app, paymentService, "noreply@slice.bank.in", "mx.cloudflare.net")
	service.Now = func() time.Time { return now }
	service.Reviews = reviews
	return service, paymentService, app, &now, reviews
}

func creditMessage(at time.Time, rrn string) Message {
	return Message{
		MessageID: "email-" + rrn, From: "noreply@slice.bank.in",
		Subject:               "Received ₹100.01 in your slice account",
		Body:                  "You have received ₹100.01 via UPI from Alice alice@oksbi. UPI Ref: " + rrn,
		Date:                  at,
		AuthenticationResults: []string{"mx.cloudflare.net; dkim=pass header.d=slice.bank.in"},
	}
}

func TestIngestMatchesAuthenticatedEmailAndDeduplicatesMessageID(t *testing.T) {
	service, paymentService, app, now, _ := emailTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Source: "cloudflare_email", Message: creditMessage(now.Add(time.Minute), "606703736479"), ReceivedAt: now.Add(time.Minute)}
	first, err := service.Ingest(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "matched" || first.Action != "marked_paid" || first.PaymentID != payment.ID {
		t.Fatalf("first result = %+v", first)
	}
	second, err := service.Ingest(input)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate || second.Action != "duplicate_event" || second.EventID != first.EventID {
		t.Fatalf("duplicate result = %+v", second)
	}
	stored, _ := paymentService.Get(payment.ID)
	if stored.Status != domain.StatusPaid || stored.RRN != "606703736479" {
		t.Fatalf("stored payment = %+v", stored)
	}
	if count, _ := app.CountRecords("email_events"); count != 1 {
		t.Fatalf("email event count = %d", count)
	}
}

func TestEmailCannotSettleKotakPayment(t *testing.T) {
	emailService, paymentService, _, now, _ := emailTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100})
	if err != nil {
		t.Fatal(err)
	}
	emailResult, err := emailService.Ingest(Input{Source: "cloudflare_email", Message: creditMessage(now.Add(time.Minute), "606703736479"), ReceivedAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if emailResult.Status != "review_required" || emailResult.Action != "unmatched" || emailResult.PaymentID != "" {
		t.Fatalf("email result = %+v", emailResult)
	}
	stored, _ := paymentService.Get(payment.ID)
	if stored.Status != domain.StatusPending {
		t.Fatalf("Slice email changed Kotak payment to %s", stored.Status)
	}
}

func TestUntrustedEmailCannotMatchAndOpensCriticalReview(t *testing.T) {
	service, paymentService, app, now, reviews := emailTestService(t)
	payment, _, _ := paymentService.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
	message := creditMessage(now.Add(time.Minute), "123456789012")
	message.AuthenticationResults = []string{"mx.cloudflare.net; dkim=fail header.d=slice.bank.in"}
	result, err := service.Ingest(Input{Source: "cloudflare_email", Message: message})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "email_auth_failed" || result.ReviewCaseID != "review-case" || len(reviews.inputs) != 1 || reviews.inputs[0].Severity != "critical" {
		t.Fatalf("result=%+v reviews=%+v", result, reviews.inputs)
	}
	stored, _ := paymentService.Get(payment.ID)
	if stored.Status != domain.StatusPending {
		t.Fatalf("untrusted email changed payment to %s", stored.Status)
	}
	event, _ := app.FindRecordById("email_events", result.EventID)
	if event.GetString("processing_status") != "error" {
		t.Fatalf("event status=%s", event.GetString("processing_status"))
	}
}

func TestUnrelatedEmailIsIgnoredWithoutAuthenticationReview(t *testing.T) {
	service, _, _, now, reviews := emailTestService(t)
	message := Message{
		MessageID: "monthly-statement", From: "noreply@slice.bank.in",
		Subject: "Your monthly account statement", Body: "The statement is attached.", Date: *now,
	}
	result, err := service.Ingest(Input{Source: "cloudflare_email", Message: message})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "ignored_non_payment_email" || len(reviews.inputs) != 0 {
		t.Fatalf("result=%+v reviews=%+v", result, reviews.inputs)
	}
}

func TestDelayedEmailCannotConfirmReusedFingerprint(t *testing.T) {
	service, paymentService, _, now, _ := emailTestService(t)
	first, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
	if err != nil {
		t.Fatal(err)
	}
	oldMessageTime := now.Add(2 * time.Minute)
	*now = now.Add(24*time.Hour + 6*time.Minute + time.Second)
	second, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
	if err != nil {
		t.Fatal(err)
	}
	if first.PayablePaise != second.PayablePaise {
		t.Fatalf("test requires reused amount: %d != %d", first.PayablePaise, second.PayablePaise)
	}
	result, err := service.Ingest(Input{Source: "cloudflare_email", Message: creditMessage(oldMessageTime, "999988887777"), ReceivedAt: *now})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "review_required" || result.PaymentID != "" {
		t.Fatalf("stale result = %+v", result)
	}
	stored, _ := paymentService.Get(second.ID)
	if stored.Status != domain.StatusPending {
		t.Fatalf("reused payment status = %s", stored.Status)
	}
}
