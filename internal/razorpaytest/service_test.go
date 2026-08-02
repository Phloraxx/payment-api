package razorpaytest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

type fakeProviderClient struct {
	createCalls int
	order       ProviderOrder
	createErr   error
	payment     ProviderPayment
	fetchErr    error
}

func (f *fakeProviderClient) CreateOrder(_ context.Context, amount int64, receipt string) (ProviderOrder, error) {
	f.createCalls++
	if f.createErr != nil {
		return ProviderOrder{}, f.createErr
	}
	order := f.order
	if order.ID == "" {
		order = ProviderOrder{ID: "order_test_123", Amount: amount, Currency: "INR", Receipt: receipt, Status: "created"}
	}
	return order, nil
}

func (f *fakeProviderClient) FetchPayment(_ context.Context, paymentID string) (ProviderPayment, error) {
	if f.fetchErr != nil {
		return ProviderPayment{}, f.fetchErr
	}
	payment := f.payment
	if payment.ID == "" {
		payment.ID = paymentID
	}
	return payment, nil
}

func testService(t *testing.T) (*Service, *fakeProviderClient, *tests.TestApp, *time.Time) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	client := &fakeProviderClient{}
	service := NewService(app, client, "rzp_test_key", "checkout-secret-123456", "webhook-secret-123456789012", "PayGate Test")
	service.Now = func() time.Time { return now }
	return service, client, app, &now
}

func TestCreateIsIdempotentAndDoesNotCreateProviderDuplicates(t *testing.T) {
	service, client, _, _ := testService(t)
	first, replayed, err := service.Create(context.Background(), CreateInput{AmountPaise: 100, ExternalID: "order-1", IdempotencyKey: "idem-1"})
	if err != nil || replayed {
		t.Fatalf("first=%v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := service.Create(context.Background(), CreateInput{AmountPaise: 100, ExternalID: "order-1", IdempotencyKey: "idem-1"})
	if err != nil || !replayed || second.Id != first.Id {
		t.Fatalf("second=%v replayed=%v err=%v", second, replayed, err)
	}
	if client.createCalls != 1 {
		t.Fatalf("provider create calls=%d", client.createCalls)
	}
	_, _, err = service.Create(context.Background(), CreateInput{AmountPaise: 200, ExternalID: "order-1", IdempotencyKey: "idem-1"})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflict error=%v", err)
	}
}

func TestVerifyRejectsTamperingAndOnlyCapturedProviderStateMarksPaid(t *testing.T) {
	service, client, _, now := testService(t)
	order, _, err := service.Create(context.Background(), CreateInput{AmountPaise: 250, IdempotencyKey: "idem-verify"})
	if err != nil {
		t.Fatal(err)
	}
	paymentID := "pay_test_123"
	client.payment = ProviderPayment{
		ID: paymentID, OrderID: order.GetString("razorpay_order_id"), Amount: 250,
		Currency: "INR", Status: "captured", Captured: true, Method: "upi",
	}
	_, err = service.Verify(context.Background(), VerifyInput{
		LocalOrderID: order.Id, RazorpayOrderID: order.GetString("razorpay_order_id"),
		RazorpayPaymentID: paymentID, RazorpaySignature: "tampered",
	})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_SIGNATURE_INVALID" {
		t.Fatalf("tampered error=%v", err)
	}
	signature := checkoutSignature(service.KeySecret, order.GetString("razorpay_order_id"), paymentID)
	verified, err := service.Verify(context.Background(), VerifyInput{
		LocalOrderID: order.Id, RazorpayOrderID: order.GetString("razorpay_order_id"),
		RazorpayPaymentID: paymentID, RazorpaySignature: signature,
	})
	if err != nil {
		t.Fatal(err)
	}
	if verified.GetString("status") != "captured" || verified.GetString("payment_method") != "upi" || verified.GetDateTime("captured_at").Time() != *now {
		t.Fatalf("verified status=%s method=%s captured=%s", verified.GetString("status"), verified.GetString("payment_method"), verified.GetDateTime("captured_at"))
	}
}

func TestWebhookIsSignedDeduplicatedAndMonotonic(t *testing.T) {
	service, _, app, _ := testService(t)
	order, _, err := service.Create(context.Background(), CreateInput{AmountPaise: 500, IdempotencyKey: "idem-webhook"})
	if err != nil {
		t.Fatal(err)
	}
	captured := webhookBody(t, "payment.captured", ProviderPayment{
		ID: "pay_webhook_1", OrderID: order.GetString("razorpay_order_id"), Amount: 500,
		Currency: "INR", Status: "captured", Captured: true, Method: "upi",
	})
	result, err := service.IngestWebhook("evt_captured", Sign(service.WebhookSecret, captured), captured)
	if err != nil || !result.Processed || result.Status != "captured" {
		t.Fatalf("captured result=%+v err=%v", result, err)
	}
	duplicate, err := service.IngestWebhook("evt_captured", Sign(service.WebhookSecret, captured), captured)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	failed := webhookBody(t, "payment.failed", ProviderPayment{
		ID: "pay_webhook_1", OrderID: order.GetString("razorpay_order_id"), Amount: 500,
		Currency: "INR", Status: "failed", ErrorDescription: "late failed event",
	})
	if _, err := service.IngestWebhook("evt_failed_late", Sign(service.WebhookSecret, failed), failed); err != nil {
		t.Fatal(err)
	}
	stored, _ := app.FindRecordById("razorpay_test_orders", order.Id)
	if stored.GetString("status") != "captured" || stored.GetString("error") != "" || stored.GetString("provider_status") != "captured" {
		t.Fatalf("captured order was changed by stale failure: status=%s provider=%s error=%q", stored.GetString("status"), stored.GetString("provider_status"), stored.GetString("error"))
	}
	if count, _ := app.CountRecords("razorpay_test_events"); count != 2 {
		t.Fatalf("event count=%d", count)
	}
}

func TestWebhookRejectsInvalidSignatureAndPersistsMismatchAsFailed(t *testing.T) {
	service, _, app, _ := testService(t)
	order, _, err := service.Create(context.Background(), CreateInput{AmountPaise: 700, IdempotencyKey: "idem-mismatch"})
	if err != nil {
		t.Fatal(err)
	}
	body := webhookBody(t, "payment.captured", ProviderPayment{
		ID: "pay_mismatch", OrderID: order.GetString("razorpay_order_id"), Amount: 701,
		Currency: "INR", Status: "captured",
	})
	_, err = service.IngestWebhook("evt_invalid_sig", "invalid", body)
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_WEBHOOK_SIGNATURE_INVALID" {
		t.Fatalf("signature error=%v", err)
	}
	result, err := service.IngestWebhook("evt_mismatch", Sign(service.WebhookSecret, body), body)
	if err != nil || result.Status != "failed" {
		t.Fatalf("mismatch=%+v err=%v", result, err)
	}
	stored, _ := app.FindRecordById("razorpay_test_orders", order.Id)
	if stored.GetString("status") != "created" {
		t.Fatalf("mismatched event changed order to %s", stored.GetString("status"))
	}
}

func checkoutSignature(secret, orderID, paymentID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}

func webhookBody(t *testing.T, event string, payment ProviderPayment) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"event": event, "created_at": int64(1785672000),
		"payload": map[string]any{"payment": map[string]any{"entity": payment}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestCreateFailureCannotBeSilentlyRetriedWithSameIdempotencyKey(t *testing.T) {
	service, client, _, _ := testService(t)
	client.createErr = errors.New("provider timeout")
	_, _, err := service.Create(context.Background(), CreateInput{AmountPaise: 100, IdempotencyKey: "idem-timeout"})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_CREATE_FAILED" {
		t.Fatalf("first error=%v", err)
	}
	_, _, err = service.Create(context.Background(), CreateInput{AmountPaise: 100, IdempotencyKey: "idem-timeout"})
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_CREATE_STATE_UNKNOWN" {
		t.Fatalf("replay error=%v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("provider create calls=%d", client.createCalls)
	}
}

func TestWebhookRejectsEventIDReusedWithDifferentPayload(t *testing.T) {
	service, _, _, _ := testService(t)
	order, _, err := service.Create(context.Background(), CreateInput{AmountPaise: 900, IdempotencyKey: "idem-event-conflict"})
	if err != nil {
		t.Fatal(err)
	}
	first := webhookBody(t, "payment.captured", ProviderPayment{ID: "pay_conflict", OrderID: order.GetString("razorpay_order_id"), Amount: 900, Currency: "INR", Status: "captured"})
	if _, err := service.IngestWebhook("evt_same", Sign(service.WebhookSecret, first), first); err != nil {
		t.Fatal(err)
	}
	second := webhookBody(t, "payment.failed", ProviderPayment{ID: "pay_conflict", OrderID: order.GetString("razorpay_order_id"), Amount: 900, Currency: "INR", Status: "failed"})
	_, err = service.IngestWebhook("evt_same", Sign(service.WebhookSecret, second), second)
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RAZORPAY_TEST_EVENT_ID_CONFLICT" {
		t.Fatalf("error=%v", err)
	}
}
