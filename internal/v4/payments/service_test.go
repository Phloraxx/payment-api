package payments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func newTestService(t *testing.T, db *storage.DB, now time.Time) *Service {
	t.Helper()
	s := NewService(db)
	s.Now = func() time.Time { return now }
	s.Allocator.Random = fixedIndex(36)
	counter := 0
	s.NewID = func(prefix string) (string, error) {
		counter++
		return fmt.Sprintf("%s_%d", prefix, counter), nil
	}
	return s
}

func validCreateInput(key string) CreateInput {
	return CreateInput{
		RequestedAmountPaise: 10000,
		Name:                 "Sourav P Bijoy",
		ExternalID:           "evt_hardware_security_2026",
		Metadata:             json.RawMessage(`{"registration_id":"reg_284"}`),
		IdempotencyScope:     "merchant_key_1",
		IdempotencyKey:       key,
	}
}

func TestCreatePaymentOwnsProfileAmountLifecycleAndUPIURI(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)

	result, err := s.Create(context.Background(), validCreateInput("create-1"))
	if err != nil {
		t.Fatal(err)
	}
	p := result.Payment
	if result.Replayed {
		t.Fatal("first create must not be replayed")
	}
	if p.Name != "Sourav P Bijoy" || p.ExternalID != "evt_hardware_security_2026" {
		t.Fatalf("identity fields = %q / %q", p.Name, p.ExternalID)
	}
	if p.RequestedAmountPaise != 10000 || p.PayableAmountPaise != 10037 || p.AdjustmentPaise != 37 {
		t.Fatalf("amounts = requested %d payable %d adjustment %d", p.RequestedAmountPaise, p.PayableAmountPaise, p.AdjustmentPaise)
	}
	if p.CollectionProfileID != "paytm" || p.UPIIDSnapshot != "merchant@paytm" {
		t.Fatalf("profile snapshot = %q %q", p.CollectionProfileID, p.UPIIDSnapshot)
	}
	if result.UPIURI != "upi://pay?am=100.37&cu=INR&pa=merchant%40paytm" {
		t.Fatalf("UPI URI = %q", result.UPIURI)
	}
	if !p.ExpiresAt.Equal(now.Add(5*time.Minute)) || !p.GraceUntil.Equal(now.Add(10*time.Minute)) || !p.ReuseAfter.Equal(now.Add(15*time.Minute)) {
		t.Fatalf("lifecycle = %s %s %s", p.ExpiresAt, p.GraceUntil, p.ReuseAfter)
	}

	assertCount(t, db.SQL, "payments", 1)
	assertCount(t, db.SQL, "amount_reservations", 1)
	assertCount(t, db.SQL, "payment_history", 1)
	assertCount(t, db.SQL, "webhook_deliveries", 1)
	assertCount(t, db.SQL, "idempotency_keys", 1)
}

func TestDestinationChangeAffectsOnlyFuturePayments(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)

	first, err := s.Create(ctx, validCreateInput("upi-before"))
	if err != nil {
		t.Fatal(err)
	}
	profileService := profiles.NewService(db)
	profileService.Now = func() time.Time { return now.Add(time.Second) }
	if _, err := profileService.UpdateDestination(ctx, "paytm", profiles.DestinationInput{UPIID: "newmerchant@upi", PayeeName: "New Merchant"}); err != nil {
		t.Fatal(err)
	}

	s.Allocator.Random = fixedIndex(37)
	second, err := s.Create(ctx, validCreateInput("upi-after"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Payment.UPIIDSnapshot != "merchant@paytm" {
		t.Fatalf("first payment snapshot = %q", first.Payment.UPIIDSnapshot)
	}
	if second.Payment.UPIIDSnapshot != "newmerchant@upi" || second.Payment.PayeeNameSnapshot != "New Merchant" {
		t.Fatalf("second payment destination = %q / %q", second.Payment.UPIIDSnapshot, second.Payment.PayeeNameSnapshot)
	}
	reloaded, err := s.Get(ctx, first.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Payment.UPIIDSnapshot != "merchant@paytm" {
		t.Fatalf("existing payment was rewritten to %q", reloaded.Payment.UPIIDSnapshot)
	}
}

func TestCreateIdempotencyExactReplayReturnsOriginalPayment(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	input := validCreateInput("same-key")

	first, err := s.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed {
		t.Fatal("expected second create to replay")
	}
	if second.Payment.ID != first.Payment.ID || second.Payment.PayableAmountPaise != first.Payment.PayableAmountPaise || second.UPIURI != first.UPIURI {
		t.Fatalf("replay changed payment: first=%+v second=%+v", first, second)
	}
	assertCount(t, db.SQL, "payments", 1)
	assertCount(t, db.SQL, "amount_reservations", 1)
	assertCount(t, db.SQL, "webhook_deliveries", 1)
}

func TestCreateIdempotencyCanonicalizesMetadata(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	firstInput := validCreateInput("json-key")
	firstInput.Metadata = json.RawMessage(`{"b":2,"a":1}`)
	secondInput := firstInput
	secondInput.Metadata = json.RawMessage(`{ "a": 1, "b": 2 }`)

	first, err := s.Create(context.Background(), firstInput)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(context.Background(), secondInput)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Payment.ID != first.Payment.ID {
		t.Fatalf("semantically identical metadata did not replay: %+v", second)
	}
}

func TestCreateIdempotencyConflictOnChangedRequest(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	input := validCreateInput("conflict-key")
	if _, err := s.Create(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.Name = "Different Person"
	if _, err := s.Create(context.Background(), input); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	assertCount(t, db.SQL, "payments", 1)
}

func TestExternalEventIDAndNameAreNotUniquePaymentIdentity(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	first := validCreateInput("person-1")
	second := validCreateInput("person-2")

	if _, err := s.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM payments WHERE name=? AND external_id=?`, first.Name, first.ExternalID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("same name/event payment count = %d, want 2", count)
	}
}

func TestIdempotencyKeyIsScopedToMerchantCredential(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, now)
	first := validCreateInput("shared-key")
	second := first
	second.IdempotencyScope = "merchant_key_2"

	a, err := s.Create(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if a.Payment.ID == b.Payment.ID || b.Replayed {
		t.Fatalf("different merchant scopes collided: %+v %+v", a, b)
	}
}

func TestProfileSwitchAffectsOnlyNewPayments(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,payee_name,parser,enabled,active,created_at,updated_at)
        VALUES('kotak','Kotak','merchant@kotak','PayGate Kotak','kotak_sms',1,0,?,?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	s := newTestService(t, db, now)

	first, err := s.Create(context.Background(), validCreateInput("profile-1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WithImmediateTx(context.Background(), func(tx *storage.ImmediateTx) error {
		if _, err := tx.ExecContext(context.Background(), `UPDATE collection_profiles SET active=0 WHERE id='paytm'`); err != nil {
			return err
		}
		_, err := tx.ExecContext(context.Background(), `UPDATE collection_profiles SET active=1 WHERE id='kotak'`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	secondInput := validCreateInput("profile-2")
	second, err := s.Create(context.Background(), secondInput)
	if err != nil {
		t.Fatal(err)
	}

	if first.Payment.CollectionProfileID != "paytm" || first.Payment.UPIIDSnapshot != "merchant@paytm" {
		t.Fatalf("first payment snapshot changed: %+v", first.Payment)
	}
	if second.Payment.CollectionProfileID != "kotak" || second.Payment.UPIIDSnapshot != "merchant@kotak" {
		t.Fatalf("second payment did not use Kotak: %+v", second.Payment)
	}
	if second.Payment.PayableAmountPaise != 10037 {
		t.Fatalf("profile-scoped amount should allow same exact amount, got %d", second.Payment.PayableAmountPaise)
	}
	if !strings.Contains(second.UPIURI, "pa=merchant%40kotak") || !strings.Contains(second.UPIURI, "pn=PayGate+Kotak") {
		t.Fatalf("Kotak UPI URI = %q", second.UPIURI)
	}
}

func TestCreateUsesOverflowOnlyAfterBaseBucketExhausted(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	fillActiveRange(t, db.SQL, 10001, 10099, -1, now)
	s := newTestService(t, db, now)

	result, err := s.Create(context.Background(), validCreateInput("overflow-create"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.PayableAmountPaise != 10137 {
		t.Fatalf("payable = %d, want 10137", result.Payment.PayableAmountPaise)
	}
}

func TestCreateFailsClosedWithoutActiveProfile(t *testing.T) {
	db := openAllocatorDB(t)
	if _, err := db.SQL.Exec(`UPDATE collection_profiles SET active=0 WHERE id='paytm'`); err != nil {
		t.Fatal(err)
	}
	s := newTestService(t, db, time.UnixMilli(1_788_200_000_000))
	if _, err := s.Create(context.Background(), validCreateInput("no-profile")); !errors.Is(err, ErrNoActiveProfile) {
		t.Fatalf("error = %v, want ErrNoActiveProfile", err)
	}
	assertCount(t, db.SQL, "payments", 0)
}

func TestCreateRollbackRestoresReservationStateOnLaterFailure(t *testing.T) {
	db := openAllocatorDB(t)
	now := time.UnixMilli(1_788_200_000_000).UTC()
	insertTestPayment(t, db.SQL, "old_pay", 10037, now.Add(-30*time.Minute))
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('old_res','paytm',10037,'old_pay',?,?,?)`, now.Add(-30*time.Minute).UnixMilli(), now.Add(-time.Minute).UnixMilli(), now.Add(-30*time.Minute).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	s := newTestService(t, db, now)
	s.NewID = func(string) (string, error) { return "", errors.New("synthetic id failure") }
	if _, err := s.Create(context.Background(), validCreateInput("rollback")); err == nil {
		t.Fatal("expected create failure")
	}

	var released sql.NullInt64
	if err := db.SQL.QueryRow(`SELECT released_at FROM amount_reservations WHERE id='old_res'`).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released.Valid {
		t.Fatalf("due-reservation release escaped rolled-back transaction: %d", released.Int64)
	}
	assertCount(t, db.SQL, "payments", 1) // only old_pay
	assertCount(t, db.SQL, "webhook_deliveries", 0)
	assertCount(t, db.SQL, "idempotency_keys", 0)
}

func TestWebhookPayloadUsesDurableDeliveryEventID(t *testing.T) {
	db := openAllocatorDB(t)
	s := newTestService(t, db, time.UnixMilli(1_788_200_000_000))
	result, err := s.Create(context.Background(), validCreateInput("webhook-id"))
	if err != nil {
		t.Fatal(err)
	}
	var eventID, payload string
	if err := db.SQL.QueryRow(`SELECT id,payload_json FROM webhook_deliveries WHERE payment_id=?`, result.Payment.ID).Scan(&eventID, &payload); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.ID != eventID {
		t.Fatalf("payload event ID %q != durable row ID %q", envelope.ID, eventID)
	}
}

func TestCreateRejectsNonObjectOrTrailingMetadata(t *testing.T) {
	db := openAllocatorDB(t)
	s := newTestService(t, db, time.UnixMilli(1_788_200_000_000))
	cases := []json.RawMessage{
		json.RawMessage(`[1,2]`),
		json.RawMessage(`{"a":1} {"b":2}`),
		json.RawMessage(`not-json`),
	}
	for i, raw := range cases {
		input := validCreateInput(fmt.Sprintf("bad-json-%d", i))
		input.Metadata = raw
		if _, err := s.Create(context.Background(), input); !errors.Is(err, ErrInvalidPaymentInput) {
			t.Fatalf("metadata %q error = %v", raw, err)
		}
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
