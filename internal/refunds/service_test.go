package refunds

import (
	"errors"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type fakeWebhookScheduler struct {
	events []string
	wakes  int
}

func (f *fakeWebhookScheduler) ScheduleRefund(_ core.App, event string, _, _ *core.Record, _ time.Time) error {
	f.events = append(f.events, event)
	return nil
}
func (f *fakeWebhookScheduler) Wake() { f.wakes++ }

func refundTestService(t *testing.T) (*Service, *payments.Service, *tests.TestApp, *time.Time, audit.Actor, *fakeWebhookScheduler) {
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
	operator.Id = "refundop0000001"
	operator.SetEmail("refund@example.com")
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
	webhooks := &fakeWebhookScheduler{}
	service := NewService(app, auditService, webhooks)
	service.Now = func() time.Time { return now }
	return service, paymentService, app, &now, audit.Actor{ID: operator.Id, Email: operator.Email()}, webhooks
}

func createPaidPayment(t *testing.T, paymentService *payments.Service, now time.Time, rupees int64, rrn string) *domain.Payment {
	t.Helper()
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: rupees})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := paymentService.Match(domain.ParsedSMS{AmountPaise: payment.PayablePaise, RRN: rrn, OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}
	return matched.Payment
}

func TestRefundRequestIsAuditedIdempotentAndBounded(t *testing.T) {
	service, paymentService, app, now, actor, hooks := refundTestService(t)
	payment := createPaidPayment(t, paymentService, *now, 100, "100100100100")
	first, replayed, err := service.Request(RequestInput{
		PaymentID: payment.ID, AmountPaise: 5000, Reason: "Customer requested partial refund",
		ExternalID: "refund-order-1", IdempotencyKey: "refund-idem-1", Actor: actor,
	})
	if err != nil || replayed {
		t.Fatalf("first=%v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := service.Request(RequestInput{
		PaymentID: payment.ID, AmountPaise: 5000, Reason: "Customer requested partial refund",
		ExternalID: "refund-order-1", IdempotencyKey: "refund-idem-1", Actor: actor,
	})
	if err != nil || !replayed || second.Id != first.Id {
		t.Fatalf("second=%v replayed=%v err=%v", second, replayed, err)
	}
	_, _, err = service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: 5100, Reason: "Too much remaining", IdempotencyKey: "refund-idem-2", Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "REFUND_AMOUNT_EXCEEDS_AVAILABLE" {
		t.Fatalf("error=%v", err)
	}
	if len(hooks.events) != 1 || hooks.events[0] != "refund.requested" || hooks.wakes != 1 {
		t.Fatalf("events=%v wakes=%d", hooks.events, hooks.wakes)
	}
	if count, _ := app.CountRecords("audit_events"); count != 1 {
		t.Fatalf("audit count=%d", count)
	}
}

func TestRefundLifecycleRequiresReferenceAndRejectsTerminalTransitions(t *testing.T) {
	service, paymentService, app, now, actor, hooks := refundTestService(t)
	payment := createPaidPayment(t, paymentService, *now, 200, "200200200200")
	refund, _, err := service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: 10000, Reason: "Return", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(UpdateInput{RefundID: refund.Id, Status: "completed", Note: "Bank transfer completed.", Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "REFUND_REFERENCE_REQUIRED" {
		t.Fatalf("error=%v", err)
	}
	processing, err := service.Update(UpdateInput{RefundID: refund.Id, Status: "processing", Note: "Initiated in bank app.", Actor: actor})
	if err != nil || processing.GetString("status") != "processing" {
		t.Fatalf("processing=%v err=%v", processing, err)
	}
	completed, err := service.Update(UpdateInput{RefundID: refund.Id, Status: "completed", Reference: "RFND123456789", Note: "Verified bank transfer reference.", Actor: actor})
	if err != nil || completed.GetString("status") != "completed" {
		t.Fatalf("completed=%v err=%v", completed, err)
	}
	_, err = service.Update(UpdateInput{RefundID: refund.Id, Status: "cancelled", Note: "Cannot cancel completed refund.", Actor: actor})
	if !errors.As(err, &domainErr) || domainErr.Code != "INVALID_REFUND_TRANSITION" {
		t.Fatalf("terminal error=%v", err)
	}
	stored, _ := app.FindRecordById("refunds", refund.Id)
	if stored.GetString("reference") != "RFND123456789" || stored.GetDateTime("completed_at").IsZero() {
		t.Fatalf("stored reference=%s completed=%s", stored.GetString("reference"), stored.GetDateTime("completed_at"))
	}
	if len(hooks.events) != 3 || hooks.events[1] != "refund.processing" || hooks.events[2] != "refund.completed" {
		t.Fatalf("events=%v", hooks.events)
	}
}

func TestRefundCannotBeCreatedForPendingPayment(t *testing.T) {
	service, paymentService, _, _, actor, _ := refundTestService(t)
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 50})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: 1000, Reason: "Invalid", Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "PAYMENT_NOT_REFUNDABLE" {
		t.Fatalf("error=%v", err)
	}
}

func TestRefundEventsPersistThroughRealWebhookOutbox(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	users, _ := app.FindCollectionByNameOrId("users")
	operator := core.NewRecord(users)
	operator.Id = "realhookop00001"
	operator.SetEmail("hook@example.com")
	operator.SetPassword("test-password-123")
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	cfg := config.Config{
		PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour,
		OutgoingWebhookURL:    "https://example.invalid/webhook",
		OutgoingWebhookSecret: "outgoing-webhook-secret-123456",
	}
	outbox := webhooks.NewService(app, cfg)
	paymentService := payments.NewService(app, cfg, outbox)
	paymentService.Now = func() time.Time { return now }
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	auditService := audit.NewService(app)
	auditService.Now = func() time.Time { return now }
	service := NewService(app, auditService, outbox)
	service.Now = func() time.Time { return now }
	payment := createPaidPayment(t, paymentService, now, 125, "125125125125")
	refund, _, err := service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: 1000, Reason: "Webhook test", Actor: audit.Actor{ID: operator.Id, Email: operator.Email()}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(UpdateInput{RefundID: refund.Id, Status: "processing", Note: "Started", Actor: audit.Actor{ID: operator.Id, Email: operator.Email()}}); err != nil {
		t.Fatal(err)
	}
	deliveries, err := app.FindAllRecords("webhook_deliveries")
	if err != nil {
		t.Fatal(err)
	}
	var foundRequested, foundProcessing bool
	for _, delivery := range deliveries {
		switch delivery.GetString("event") {
		case "refund.requested":
			foundRequested = true
			if delivery.GetString("refund") != refund.Id {
				t.Fatalf("requested delivery refund=%s", delivery.GetString("refund"))
			}
		case "refund.processing":
			foundProcessing = true
			if delivery.GetString("refund") != refund.Id {
				t.Fatalf("processing delivery refund=%s", delivery.GetString("refund"))
			}
		}
	}
	if !foundRequested || !foundProcessing {
		t.Fatalf("events requested=%v processing=%v", foundRequested, foundProcessing)
	}
}

func TestFailedRefundCannotBeReactivatedBeyondRemainingAmount(t *testing.T) {
	service, paymentService, _, now, actor, _ := refundTestService(t)
	payment := createPaidPayment(t, paymentService, *now, 100, "909090909090")
	first, _, err := service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: payment.PayablePaise, Reason: "First attempt", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(UpdateInput{RefundID: first.Id, Status: "failed", Note: "Bank transfer failed", Actor: actor}); err != nil {
		t.Fatal(err)
	}
	second, _, err := service.Request(RequestInput{PaymentID: payment.ID, AmountPaise: payment.PayablePaise, Reason: "Replacement attempt", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(UpdateInput{RefundID: first.Id, Status: "processing", Note: "Unsafe retry", Actor: actor})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "REFUND_AMOUNT_EXCEEDS_AVAILABLE" {
		t.Fatalf("reactivation error=%v", err)
	}
	if _, err := service.Update(UpdateInput{RefundID: second.Id, Status: "cancelled", Note: "Replacement cancelled", Actor: actor}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(UpdateInput{RefundID: first.Id, Status: "processing", Note: "Retry after capacity released", Actor: actor}); err != nil {
		t.Fatalf("retry after capacity released: %v", err)
	}
}

func TestRefundIdempotencyIncludesMetadata(t *testing.T) {
	service, paymentService, _, now, actor, _ := refundTestService(t)
	payment := createPaidPayment(t, paymentService, *now, 50, "505050505050")
	input := RequestInput{PaymentID: payment.ID, AmountPaise: 1000, Reason: "Metadata test", IdempotencyKey: "refund-metadata-idem", Metadata: map[string]any{"order": "A"}, Actor: actor}
	if _, _, err := service.Request(input); err != nil {
		t.Fatal(err)
	}
	input.Metadata = map[string]any{"order": "B"}
	_, _, err := service.Request(input)
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "REFUND_IDEMPOTENCY_CONFLICT" {
		t.Fatalf("metadata idempotency error=%v", err)
	}
}
