package payments

import (
	"context"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/observations"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func insertRelayEvent(t *testing.T, db *storage.DB, id, sourceID, packageName string, posted, received time.Time) {
	t.Helper()
	if _, err := db.SQL.Exec(`INSERT OR IGNORE INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at) VALUES('device_1','Phone','pem',1,?)`, posted.Add(-time.Hour).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status) VALUES(?,'device_1',?,?,?,?,'received')`, id, sourceID, packageName, posted.UnixMilli(), received.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}

func paytmObservation(amount int64, at time.Time, source string) observations.Observation {
	return observations.Observation{
		Source: "paytm_notification", CollectionProfileID: "paytm", AmountPaise: amount,
		PayerName: "Rahul", OccurredAt: at.UTC(), OccurredAtSource: source,
	}
}
func TestApplyObservationMarksPendingPaidAtomically(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("match-1"))
	if err != nil {
		t.Fatal(err)
	}
	occurred := createdAt.Add(2 * time.Minute)
	received := occurred.Add(time.Second)
	insertRelayEvent(t, db, "relay_1", "source_1", observations.PaytmBusinessPackage, occurred, received)
	s.Now = func() time.Time { return received }

	result, err := s.ApplyObservation(ctx, "relay_1", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_posted_at"), received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "matched" || result.PaymentID != created.Payment.ID || !result.Transitioned || result.Replayed {
		t.Fatalf("match result = %+v", result)
	}
	got, err := s.Get(ctx, created.Payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payment.Status != "paid" || got.Payment.PaidAt == nil || !got.Payment.PaidAt.Equal(occurred) || got.Payment.PayerName != "Rahul" {
		t.Fatalf("paid payment = %+v", got.Payment)
	}
	assertCount(t, db.SQL, "payment_observations", 1)
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)

	replay, err := s.ApplyObservation(ctx, "relay_1", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_posted_at"), received)
	if err != nil || !replay.Replayed || replay.Result != "matched" || replay.PaymentID != created.Payment.ID {
		t.Fatalf("replay = %+v err=%v", replay, err)
	}
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)
}
func TestDifferentRelayEventForAlreadyPaidPaymentDoesNotTransitionTwice(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("match-duplicate"))
	if err != nil {
		t.Fatal(err)
	}
	occurred := createdAt.Add(2 * time.Minute)
	received := occurred.Add(time.Second)
	s.Now = func() time.Time { return received }
	insertRelayEvent(t, db, "relay_1", "source_1", observations.PaytmBusinessPackage, occurred, received)
	if _, err := s.ApplyObservation(ctx, "relay_1", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_posted_at"), received); err != nil {
		t.Fatal(err)
	}
	insertRelayEvent(t, db, "relay_2", "source_2", observations.PaytmBusinessPackage, occurred.Add(time.Second), received.Add(time.Second))
	second, err := s.ApplyObservation(ctx, "relay_2", paytmObservation(created.Payment.PayableAmountPaise, occurred.Add(time.Second), "notification_posted_at"), received.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if second.Result != "corroborated" || second.PaymentID != created.Payment.ID || second.Transitioned {
		t.Fatalf("second match = %+v", second)
	}
	assertCount(t, db.SQL, "payment_observations", 2)
	assertCount(t, db.SQL, "payment_history", 2)
	assertCount(t, db.SQL, "webhook_deliveries", 2)
}
func TestObservationCanPayExpiredPaymentWhenOccurrenceWasInsideReservation(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("expired-match"))
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return createdAt.Add(10 * time.Minute) }
	if count, err := s.ExpireDue(ctx, 100); err != nil || count != 1 {
		t.Fatalf("expire = %d %v", count, err)
	}
	occurred := createdAt.Add(8 * time.Minute)
	received := createdAt.Add(11 * time.Minute)
	insertRelayEvent(t, db, "relay_expired", "source_expired", observations.PaytmBusinessPackage, occurred, received)
	s.Now = func() time.Time { return received }
	result, err := s.ApplyObservation(ctx, "relay_expired", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_text"), received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "matched" || !result.Transitioned || result.PaymentID != created.Payment.ID {
		t.Fatalf("result = %+v", result)
	}
	got, _ := s.Get(ctx, created.Payment.ID)
	if got.Payment.Status != "paid" || got.Payment.PaidAt == nil || !got.Payment.PaidAt.Equal(occurred) {
		t.Fatalf("payment = %+v", got.Payment)
	}
}

func TestCancelledPaymentOnlyMatchesMoneyThatOccurredBeforeCancellation(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("cancel-match"))
	if err != nil {
		t.Fatal(err)
	}
	cancelAt := createdAt.Add(2 * time.Minute)
	s.Now = func() time.Time { return cancelAt }
	if _, err := s.Cancel(ctx, created.Payment.ID); err != nil {
		t.Fatal(err)
	}
	occurred := createdAt.Add(time.Minute)
	received := createdAt.Add(3 * time.Minute)
	insertRelayEvent(t, db, "relay_cancel_before", "source_cancel_before", observations.PaytmBusinessPackage, occurred, received)
	s.Now = func() time.Time { return received }
	result, err := s.ApplyObservation(ctx, "relay_cancel_before", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_text"), received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "matched" || !result.Transitioned {
		t.Fatalf("pre-cancel money = %+v", result)
	}
	got, _ := s.Get(ctx, created.Payment.ID)
	if got.Payment.Status != "paid" {
		t.Fatalf("payment status = %s", got.Payment.Status)
	}
}

func TestCancelledPaymentRejectsMoneyAfterCancellation(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("cancel-after"))
	if err != nil {
		t.Fatal(err)
	}
	cancelAt := createdAt.Add(2 * time.Minute)
	s.Now = func() time.Time { return cancelAt }
	if _, err := s.Cancel(ctx, created.Payment.ID); err != nil {
		t.Fatal(err)
	}
	occurred := createdAt.Add(3 * time.Minute)
	received := occurred.Add(time.Second)
	insertRelayEvent(t, db, "relay_cancel_after", "source_cancel_after", observations.PaytmBusinessPackage, occurred, received)
	s.Now = func() time.Time { return received }
	result, err := s.ApplyObservation(ctx, "relay_cancel_after", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_text"), received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "unmatched" || result.PaymentID != "" || result.Transitioned {
		t.Fatalf("post-cancel money = %+v", result)
	}
	got, _ := s.Get(ctx, created.Payment.ID)
	if got.Payment.Status != "cancelled" {
		t.Fatalf("payment status = %s", got.Payment.Status)
	}
}
func insertHistoricalReservation(t *testing.T, db *storage.DB, paymentID, profileID string, created time.Time, releasedAt *time.Time, status string) {
	t.Helper()
	if profileID == "kotak" {
		if _, err := db.SQL.Exec(`INSERT OR IGNORE INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at) VALUES('kotak','Kotak','kotak@upi','kotak_sms',1,0,?,?)`, created.UnixMilli(), created.UnixMilli()); err != nil {
			t.Fatal(err)
		}
	}
	reuse := created.Add(15 * time.Minute)
	if _, err := db.SQL.Exec(`INSERT INTO payments(id,name,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
		VALUES(?,'Person','{}',10000,10037,37,?,?,'`+status+`',?,?,?,?)`, paymentID, profileID, profileID+"@upi", created.UnixMilli(), created.Add(5*time.Minute).UnixMilli(), created.Add(10*time.Minute).UnixMilli(), reuse.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	var released any
	if releasedAt != nil {
		released = releasedAt.UnixMilli()
	}
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,released_at,last_used_at) VALUES(?,?,?,?,?,?,?,?)`, "res_"+paymentID, profileID, 10037, paymentID, created.UnixMilli(), reuse.UnixMilli(), released, created.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}

func TestKotakLowConfidenceLatestReuseFailsAmbiguous(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	base := time.UnixMilli(1_788_200_000_000).UTC()
	released := base.Add(15 * time.Minute)
	insertHistoricalReservation(t, db, "old", "kotak", base, &released, "expired")
	insertHistoricalReservation(t, db, "new", "kotak", base.Add(20*time.Minute), nil, "pending")
	occurred := base.Add(22 * time.Minute)
	received := occurred.Add(time.Second)
	insertRelayEvent(t, db, "relay_kotak_reuse", "source_kotak_reuse", observations.GoogleMessagesPackage, occurred, received)
	s := newTestService(t, db, received)
	obs := observations.Observation{Source: "kotak_sms", CollectionProfileID: "kotak", AmountPaise: 10037, OccurredAt: occurred, OccurredAtSource: "notification_posted_at"}
	result, err := s.ApplyObservation(ctx, "relay_kotak_reuse", obs, received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "ambiguous" || result.PaymentID != "" || result.Transitioned {
		t.Fatalf("reuse result = %+v", result)
	}
	var status string
	if err := db.SQL.QueryRow(`SELECT status FROM payments WHERE id='new'`).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("new payment status=%q err=%v", status, err)
	}
}
func TestTrustedHistoricalTimeCanMatchOldReservationAfterReuse(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	base := time.UnixMilli(1_788_200_000_000).UTC()
	released := base.Add(15 * time.Minute)
	insertHistoricalReservation(t, db, "old", "paytm", base, &released, "expired")
	insertHistoricalReservation(t, db, "new", "paytm", base.Add(20*time.Minute), nil, "pending")
	occurred := base.Add(5 * time.Minute)
	received := base.Add(25 * time.Minute)
	insertRelayEvent(t, db, "relay_old", "source_old", observations.PaytmBusinessPackage, received, received)
	s := newTestService(t, db, received)
	obs := paytmObservation(10037, occurred, "notification_text")
	result, err := s.ApplyObservation(ctx, "relay_old", obs, received)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != "matched" || result.PaymentID != "old" || !result.Transitioned {
		t.Fatalf("historical result = %+v", result)
	}
	var oldStatus, newStatus string
	if err := db.SQL.QueryRow(`SELECT status FROM payments WHERE id='old'`).Scan(&oldStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.SQL.QueryRow(`SELECT status FROM payments WHERE id='new'`).Scan(&newStatus); err != nil {
		t.Fatal(err)
	}
	if oldStatus != "paid" || newStatus != "pending" {
		t.Fatalf("statuses old=%s new=%s", oldStatus, newStatus)
	}
}

func TestPaytmMatchIgnoresCurrentActiveProfile(t *testing.T) {
	ctx := context.Background()
	db := openAllocatorDB(t)
	createdAt := time.UnixMilli(1_788_200_000_000).UTC()
	s := newTestService(t, db, createdAt)
	created, err := s.Create(ctx, validCreateInput("switch-match"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`UPDATE collection_profiles SET active=0 WHERE id='paytm'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at) VALUES('kotak','Kotak','kotak@upi','kotak_sms',1,1,?,?)`, createdAt.UnixMilli(), createdAt.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	occurred := createdAt.Add(time.Minute)
	received := occurred.Add(time.Second)
	insertRelayEvent(t, db, "relay_switched", "source_switched", observations.PaytmBusinessPackage, occurred, received)
	s.Now = func() time.Time { return received }
	result, err := s.ApplyObservation(ctx, "relay_switched", paytmObservation(created.Payment.PayableAmountPaise, occurred, "notification_text"), received)
	if err != nil || result.PaymentID != created.Payment.ID || result.Result != "matched" {
		t.Fatalf("switched match=%+v err=%v", result, err)
	}
}
