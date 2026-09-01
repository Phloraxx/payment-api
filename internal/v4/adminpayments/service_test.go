package adminpayments

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

type fixture struct {
	db       *storage.DB
	payments *payments.Service
	admin    *Service
	now      *time.Time
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	profileService := profiles.NewService(db)
	ctx := context.Background()
	if _, err := profileService.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "paygate@paytm", PayeeName: "PayGate",
		Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Upsert(ctx, profiles.UpsertInput{
		ID: "kotak", Label: "Kotak", UPIID: "paygate@kotak", PayeeName: "PayGate",
		Parser: "kotak_sms", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	paymentService := payments.NewService(db)
	paymentService.Now = func() time.Time { return now }
	adminService := NewService(db)
	adminService.Now = func() time.Time { return now }
	return fixture{db: db, payments: paymentService, admin: adminService, now: &now}
}

func (f fixture) create(t *testing.T, amount int64, name, external, idem string) payments.Payment {
	t.Helper()
	result, err := f.payments.Create(context.Background(), payments.CreateInput{
		RequestedAmountPaise: amount * 100, Name: name, ExternalID: external,
		IdempotencyScope: "test", IdempotencyKey: idem,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Payment
}
func TestListSearchFilterAndEventIDNonUnique(t *testing.T) {
	f := newFixture(t)
	one := f.create(t, 100, "Sourav P Bijoy", "evt_shared", "one")
	*f.now = f.now.Add(time.Minute)
	_ = f.create(t, 200, "Another Person", "evt_shared", "two")
	*f.now = f.now.Add(time.Minute)
	_ = f.create(t, 300, "Percent % Person", "evt_other", "three")

	result, err := f.admin.List(context.Background(), ListInput{ExternalID: "evt_shared"})
	if err != nil || result.Total != 2 {
		t.Fatalf("event filter result=%+v err=%v", result, err)
	}
	result, err = f.admin.List(context.Background(), ListInput{Query: "Sourav"})
	if err != nil || result.Total != 1 || result.Items[0].ID != one.ID {
		t.Fatalf("name search result=%+v err=%v", result, err)
	}
	result, err = f.admin.List(context.Background(), ListInput{Query: "100.00"})
	if err != nil || result.Total != 1 || result.Items[0].ID != one.ID {
		t.Fatalf("amount search result=%+v err=%v", result, err)
	}
	result, err = f.admin.List(context.Background(), ListInput{Query: "%"})
	if err != nil || result.Total != 1 || result.Items[0].Name != "Percent % Person" {
		t.Fatalf("escaped wildcard search result=%+v err=%v", result, err)
	}
}
func TestDetailIncludesHistoryAndWebhookTimeline(t *testing.T) {
	f := newFixture(t)
	payment := f.create(t, 100, "Sourav", "evt_1", "detail")
	detail, err := f.admin.Get(context.Background(), payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Payment.ID != payment.ID || len(detail.History) != 1 || detail.History[0].Type != "payment.created" {
		t.Fatalf("detail history = %+v", detail)
	}
	if len(detail.Webhooks) != 1 || detail.Webhooks[0].EventType != "payment.created" || detail.Webhooks[0].Status != "pending" {
		t.Fatalf("detail webhooks = %+v", detail.Webhooks)
	}
	if _, err := f.admin.Get(context.Background(), "pay_missing"); !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("missing payment error = %v", err)
	}
}

func TestEditMerchantVisibleFieldsCreatesUpdatedWebhook(t *testing.T) {
	f := newFixture(t)
	payment := f.create(t, 100, "Sourav", "evt_1", "edit-fields")
	originalPayable := payment.PayableAmountPaise
	name, external := "Sourav P Bijoy", "evt_2"
	payer, upi, note := "Bijoy P", "bijoy@okaxis", "checked manually"
	metadata := json.RawMessage(`{"registration_id":"reg_284"}`)
	updated, err := f.admin.Edit(context.Background(), payment.ID, EditInput{
		Name: &name, ExternalID: &external, Metadata: &metadata, PayerName: &payer, PayerUPIID: &upi, InternalNote: &note,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.ExternalID != external || updated.PayerName != payer || updated.PayerUPIID != upi || updated.InternalNote != note {
		t.Fatalf("updated payment = %+v", updated)
	}
	if updated.PayableAmountPaise != originalPayable || updated.CollectionProfileID != payment.CollectionProfileID || updated.UPIIDSnapshot != payment.UPIIDSnapshot {
		t.Fatal("operator edit mutated immutable payment identity")
	}
	detail, err := f.admin.Get(context.Background(), payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.History) != 2 || detail.History[1].Actor != "admin" || detail.History[1].Type != "payment.updated" {
		t.Fatalf("history = %+v", detail.History)
	}
	if len(detail.Webhooks) != 2 || detail.Webhooks[1].EventType != "payment.updated" {
		t.Fatalf("webhooks = %+v", detail.Webhooks)
	}

	note2 := "private note only"
	if _, err := f.admin.Edit(context.Background(), payment.ID, EditInput{InternalNote: &note2}); err != nil {
		t.Fatal(err)
	}
	detail, _ = f.admin.Get(context.Background(), payment.ID)
	if len(detail.History) != 3 || len(detail.Webhooks) != 2 {
		t.Fatalf("private-note edit should add history but no webhook: history=%d webhooks=%d", len(detail.History), len(detail.Webhooks))
	}
}
func TestEditStatusToPaidAndNoopIsIdempotent(t *testing.T) {
	f := newFixture(t)
	payment := f.create(t, 100, "Sourav", "evt_1", "edit-paid")
	status, payer, upi := "paid", "Bijoy P", "bijoy@okaxis"
	updated, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &status, PayerName: &payer, PayerUPIID: &upi})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "paid" || updated.PaidAt == nil || updated.PayerName != payer {
		t.Fatalf("paid update = %+v", updated)
	}
	detail, _ := f.admin.Get(context.Background(), payment.ID)
	if len(detail.History) != 2 || detail.History[1].Type != "payment.paid" || len(detail.Webhooks) != 2 || detail.Webhooks[1].EventType != "payment.paid" {
		t.Fatalf("paid transition detail = %+v", detail)
	}
	if _, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &status}); err != nil {
		t.Fatal(err)
	}
	after, _ := f.admin.Get(context.Background(), payment.ID)
	if len(after.History) != len(detail.History) || len(after.Webhooks) != len(detail.Webhooks) {
		t.Fatal("no-op paid edit created duplicate history/webhook")
	}
}

func TestUnsafeReopenAfterReservationRelease(t *testing.T) {
	f := newFixture(t)
	payment := f.create(t, 100, "Sourav", "evt_1", "unsafe-reopen")
	cancelled, err := f.payments.Cancel(context.Background(), payment.ID)
	if err != nil || cancelled.Status != "cancelled" {
		t.Fatalf("cancel = %+v err=%v", cancelled, err)
	}
	*f.now = payment.ReuseAfter.Add(time.Second)
	if _, err := f.db.SQL.Exec(`UPDATE amount_reservations SET released_at=? WHERE payment_id=?`, f.now.UnixMilli(), payment.ID); err != nil {
		t.Fatal(err)
	}
	pending := "pending"
	if _, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &pending}); !errors.Is(err, ErrUnsafeReopen) {
		t.Fatalf("unsafe reopen error = %v", err)
	}
	paid := "paid"
	corrected, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &paid})
	if err != nil || corrected.Status != "paid" {
		t.Fatalf("old payment correction to paid = %+v err=%v", corrected, err)
	}
}

func TestReopenWithinProtectedWindowClearsPaidAtAndUsesUpdatedWebhook(t *testing.T) {
	f := newFixture(t)
	payment := f.create(t, 100, "Sourav", "evt_1", "safe-reopen")
	paid := "paid"
	if _, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &paid}); err != nil {
		t.Fatal(err)
	}
	pending := "pending"
	reopened, err := f.admin.Edit(context.Background(), payment.ID, EditInput{Status: &pending})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Status != "pending" || reopened.PaidAt != nil {
		t.Fatalf("reopened payment = %+v", reopened)
	}
	detail, _ := f.admin.Get(context.Background(), payment.ID)
	if detail.Webhooks[len(detail.Webhooks)-1].EventType != "payment.updated" {
		t.Fatalf("reopen webhook = %+v", detail.Webhooks)
	}
}
