package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestGetReturnsCanonicalSnapshotURI(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	created, err := s.Create(context.Background(), validCreateInput("get-1"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.ID != created.Payment.ID || got.UPIURI != created.UPIURI {
		t.Fatalf("get result = %+v", got)
	}
	if _, err := s.Get(context.Background(), "missing"); !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("missing get error = %v", err)
	}
}
func TestCancelIsIdempotentAndKeepsReservationUntilReuseAfter(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("cancel-1"))
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return createdAt.Add(2 * time.Minute) }

	cancelled, err := s.Cancel(ctx, created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("status = %q", cancelled.Status)
	}
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)

	var released any
	var reservedUntil int64
	if err := db.SQL.QueryRow(`SELECT released_at,reserved_until FROM amount_reservations WHERE payment_id=?`, created.Payment.ID).Scan(&released, &reservedUntil); err != nil {
		t.Fatal(err)
	}
	if released != nil || reservedUntil != created.Payment.ReuseAfter.UnixMilli() {
		t.Fatalf("reservation released early: released=%v until=%d", released, reservedUntil)
	}

	if _, err := s.Cancel(ctx, created.Payment.ID); err != nil {
		t.Fatal(err)
	}
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)
}
func TestExpireDueWaitsThroughGraceAndKeepsReservation(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("expire-1"))
	if err != nil {
		t.Fatal(err)
	}

	s.Now = func() time.Time { return createdAt.Add(10*time.Minute - time.Millisecond) }
	if count, err := s.ExpireDue(ctx, 100); err != nil || count != 0 {
		t.Fatalf("pre-grace expiry = %d, %v", count, err)
	}
	s.Now = func() time.Time { return createdAt.Add(10 * time.Minute) }
	if count, err := s.ExpireDue(ctx, 100); err != nil || count != 1 {
		t.Fatalf("grace expiry = %d, %v", count, err)
	}

	got, err := s.Get(ctx, created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "expired" {
		t.Fatalf("status = %q", got.Payment.Status)
	}
	var released any
	if err := db.SQL.QueryRow(`SELECT released_at FROM amount_reservations WHERE payment_id=?`, created.Payment.ID).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != nil {
		t.Fatalf("expired payment reservation released before reuse_after: %v", released)
	}
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)
}
func TestCancelRejectsPaidPayment(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	created, err := s.Create(ctx, validCreateInput("paid-cancel"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`UPDATE payments SET status='paid',paid_at=? WHERE id=?`, now.Add(time.Minute).UnixMilli(), created.Payment.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(ctx, created.Payment.ID); !errors.Is(err, ErrPaymentTerminal) {
		t.Fatalf("cancel paid error = %v", err)
	}
	assertCount(t, db.SQL, "payment_history", 1)
	assertCount(t, db.SQL, "webhook_deliveries", 1)
}

func TestCancellationWebhookUsesDurableEventIDAndTransitionTime(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("cancel-payload"))
	if err != nil {
		t.Fatal(err)
	}
	cancelAt := createdAt.Add(2 * time.Minute)
	s.Now = func() time.Time { return cancelAt }
	if _, err := s.Cancel(ctx, created.Payment.ID); err != nil {
		t.Fatal(err)
	}

	var eventID, payloadJSON string
	if err := db.SQL.QueryRow(`SELECT id,payload_json FROM webhook_deliveries WHERE event_type='payment.cancelled' AND payment_id=?`, created.Payment.ID).Scan(&eventID, &payloadJSON); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"] != eventID || payload["type"] != "payment.cancelled" || payload["created_at"] != cancelAt.Format(time.RFC3339Nano) {
		t.Fatalf("webhook envelope = %#v", payload)
	}
}
