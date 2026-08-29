package operatoradmin_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/operatoradmin"
	"github.com/Phloraxx/payment-api/internal/store"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func TestUpdatePaymentChangesProfileButPreservesFinancialTruthAndAudits(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	db := store.NewPocketBase(app)
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	var paymentID string
	err = db.Write(context.Background(), func(tx store.UnitOfWork) error {
		payment, err := tx.Payments().Create(store.NewPayment{
			Account: domain.PaymentAccountKotak, RequestedPaise: 50000, PayablePaise: 50017,
			CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(4 * time.Minute), ReuseAfter: now.Add(24 * time.Hour),
			ExternalID: "order-old", IdempotencyKey: "idem-immutable", Metadata: map[string]any{"old": true},
		})
		if err != nil {
			return err
		}
		payment.Status = domain.StatusPaid
		payment.RRN = "123456789012"
		payment.UPIId = "payer@upi"
		payment.PayerName = "Original Payer"
		payment.EvidenceSource = "kotak_sms"
		payment.EvidenceReference = "sms:123456789012"
		payment.PaidAt = now.Add(-30 * time.Second)
		payment.ResolvedAt = now.Add(-25 * time.Second)
		paymentID = payment.ID
		return tx.Payments().Save(payment)
	})
	if err != nil {
		t.Fatal(err)
	}

	var before *domain.Payment
	if err := db.View(context.Background(), func(tx store.UnitOfWork) error {
		var readErr error
		before, readErr = tx.Payments().Get(paymentID)
		return readErr
	}); err != nil {
		t.Fatal(err)
	}

	service := operatoradmin.Service{Store: db, Now: func() time.Time { return now }}
	updated, err := service.UpdatePayment(context.Background(), operatoradmin.UpdatePaymentInput{
		PaymentID: paymentID, Actor: audit.Actor{ID: "admin-1", Email: "admin@example.com"}, DisplayName: "Workshop registration",
		CustomerName: "Sourav P Bijoy", CustomerEmail: "sourav@example.com",
		CustomerPhone: "+91 90000 00000", Description: "IEEE workshop", AdminNote: "Verified by coordinator",
		Tags: []string{" IEEE ", "VIP", "ieee"}, CustomFields: map[string]any{"semester": "S7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "Workshop registration" || updated.CustomerName != "Sourav P Bijoy" {
		t.Fatalf("profile not updated: %+v", updated)
	}
	if !reflect.DeepEqual(updated.Tags, []string{"IEEE", "VIP"}) {
		t.Fatalf("tags=%v", updated.Tags)
	}

	assertFinancialTruthEqual(t, before, updated)
	audits, err := app.FindRecordsByFilter("audit_events", "entity_id = {:id}", "created", 10, 0, map[string]any{"id": paymentID})
	if err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].GetString("action") != "payment.profile.updated" || audits[0].GetString("actor_email") != "admin@example.com" {
		t.Fatalf("audit=%v", audits)
	}
}

func TestUpdatePaymentNoOpDoesNotCreateAudit(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	db := store.NewPocketBase(app)
	now := time.Now().UTC()
	var paymentID string
	if err := db.Write(context.Background(), func(tx store.UnitOfWork) error {
		payment, err := tx.Payments().Create(store.NewPayment{Account: domain.PaymentAccountKotak, RequestedPaise: 10000, PayablePaise: 10001, CreatedAt: now, ExpiresAt: now.Add(time.Minute), ReuseAfter: now.Add(time.Hour), IdempotencyKey: "idem-noop", Metadata: map[string]any{}})
		if err == nil {
			paymentID = payment.ID
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	service := operatoradmin.Service{Store: db}
	if _, err := service.UpdatePayment(context.Background(), operatoradmin.UpdatePaymentInput{PaymentID: paymentID, CustomFields: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	count, err := app.CountRecords("audit_events")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit count=%d", count)
	}
}

func TestUpdatePaymentValidatesProfileFields(t *testing.T) {
	service := operatoradmin.Service{Store: nil}
	_ = service
	cases := []operatoradmin.UpdatePaymentInput{
		{},
		{PaymentID: "p", CustomerEmail: "not-an-email"},
		{PaymentID: "p", Tags: make([]string, 33)},
	}
	for _, input := range cases {
		app, err := tests.NewTestApp()
		if err != nil {
			t.Fatal(err)
		}
		db := store.NewPocketBase(app)
		_, err = (&operatoradmin.Service{Store: db}).UpdatePayment(context.Background(), input)
		app.Cleanup()
		if err == nil {
			t.Fatalf("expected validation error for %+v", input)
		}
		if _, ok := err.(*operatoradmin.ValidationError); !ok {
			t.Fatalf("error=%T %v", err, err)
		}
	}
}

func assertFinancialTruthEqual(t *testing.T, before, after *domain.Payment) {
	t.Helper()
	if before.Account != after.Account || before.RequestedPaise != after.RequestedPaise || before.PayablePaise != after.PayablePaise ||
		before.Status != after.Status || !before.CreatedAt.Equal(after.CreatedAt) || !before.ExpiresAt.Equal(after.ExpiresAt) ||
		!before.ReuseAfter.Equal(after.ReuseAfter) || before.RRN != after.RRN || before.UPIId != after.UPIId ||
		before.PayerName != after.PayerName || before.EvidenceSource != after.EvidenceSource || before.EvidenceReference != after.EvidenceReference ||
		!before.PaidAt.Equal(after.PaidAt) || !before.ResolvedAt.Equal(after.ResolvedAt) || before.IdempotencyKey != after.IdempotencyKey ||
		before.ExternalID != after.ExternalID || !reflect.DeepEqual(before.Metadata, after.Metadata) {
		t.Fatalf("financial truth changed\nbefore=%+v\nafter=%+v", before, after)
	}
}
