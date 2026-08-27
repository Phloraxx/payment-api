package paytmnotification

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func paytmTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC)
	cfg := config.Config{PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour, TestMode: true, PaytmQRPayload: "paytm-static-merchant-qr", PaytmNotificationWebhookSecret: "paytm-notification-secret-long-enough"}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	service := NewService(app, paymentService)
	service.Now = func() time.Time { return now }
	return service, paymentService, app, &now
}

func TestIngestMatchesPaytmNotificationByExactDDMAmount(t *testing.T) {
	service, paymentService, app, now := paytmTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 1, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Ingest(Input{SourceEventID: "com.paytm.business:1787763600000", AppPackage: PaytmBusinessPackage, AppName: "Paytm for Business", Title: "Payment received", Body: "₹1.01 paid by Test User", NotificationTime: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "matched" || result.Action != "marked_paid" || result.PaymentID != payment.ID {
		t.Fatalf("result = %+v", result)
	}
	stored, err := paymentService.Get(payment.ID)
	if err != nil || stored.Status != domain.StatusPaid || stored.PayerName != "Test User" || stored.EvidenceSource != "paytm_notification" {
		t.Fatalf("stored = %+v err=%v", stored, err)
	}
	event, err := app.FindRecordById("notification_events", result.EventID)
	if err != nil || event.GetString("processing_status") != "matched" || event.GetString("matched_payment") != payment.ID {
		t.Fatalf("event = %+v err=%v", event, err)
	}
}

func TestIngestDedupesAndRejectsWrongApp(t *testing.T) {
	service, paymentService, app, _ := paytmTestService(t)
	_, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 1, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{SourceEventID: "evt-1", AppPackage: PaytmBusinessPackage, Body: "₹1.01 paid by User"}
	first, err := service.Ingest(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Ingest(input)
	if err != nil || !second.Duplicate || second.Action != "duplicate_event" || second.EventID != first.EventID {
		t.Fatalf("duplicate = %+v err=%v", second, err)
	}
	if count, _ := app.CountRecords("notification_events"); count != 1 {
		t.Fatalf("notification count = %d; want 1", count)
	}
	if _, err := service.Ingest(Input{SourceEventID: "evt-wrong", AppPackage: "com.example.fake", Body: "₹1.01 paid by User"}); err == nil {
		t.Fatal("wrong app package was accepted")
	}
}

func TestDelayedOldNotificationCannotConfirmReusedDDMAmount(t *testing.T) {
	service, paymentService, _, now := paytmTestService(t)
	first, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 10, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(time.Minute)
	*now = now.Add(24*time.Hour + 6*time.Minute + time.Second)
	second, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 10, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	if first.PayablePaise != second.PayablePaise {
		t.Fatalf("test needs reused amount: %d vs %d", first.PayablePaise, second.PayablePaise)
	}
	result, err := service.Ingest(Input{SourceEventID: "late-old-event", AppPackage: PaytmBusinessPackage, Body: "₹10.01 paid by Old User", NotificationTime: oldTime})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmatched" {
		t.Fatalf("old notification result = %+v", result)
	}
	stored, _ := paymentService.Get(second.ID)
	if stored.Status != domain.StatusPending {
		t.Fatalf("new payment incorrectly became %s", stored.Status)
	}
}
