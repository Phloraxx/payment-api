package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func openAllocatorDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := int64(1_788_200_000_000)
	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
        VALUES('paytm','Paytm','merchant@paytm','paytm_notification',1,1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	return db
}

func fixedIndex(index int) RandomIndex {
	return func(max int) (int, error) {
		if index >= max {
			return 0, fmt.Errorf("fixed index %d >= %d", index, max)
		}
		return index, nil
	}
}

func TestAllocatorRandomizesInsideBaseBucket(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)
	allocator := NewAllocator()
	allocator.Random = fixedIndex(36)

	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10037 {
		t.Fatalf("got %d, want 10037", got)
	}
}

func TestAllocatorNeverOverflowsWhileBaseBucketHasOneFreeValue(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)
	fillActiveRange(t, db.SQL, 10001, 10099, 10077, now)

	allocator := NewAllocator()
	allocator.Random = fixedIndex(0)
	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10077 {
		t.Fatalf("got %d, want the only free base-bucket value 10077", got)
	}
}

func TestAllocatorUsesOverflowOnlyAfterBaseBucketExhausted(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)
	fillActiveRange(t, db.SQL, 10001, 10099, -1, now)

	allocator := NewAllocator()
	allocator.Random = fixedIndex(36)
	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10137 {
		t.Fatalf("got %d, want 10137 from overflow bucket", got)
	}
}

func TestSoftRecentUseCannotForceEarlyOverflow(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)

	// Create released recent history for every base-bucket value. The whole base
	// bucket is still free, so soft avoidance must fall back to these values
	// instead of moving to ₹101.xx.
	fillReleasedRange(t, db.SQL, 10001, 10099, now.Add(-time.Minute))

	allocator := NewAllocator()
	allocator.SoftHorizon = 4 * time.Hour
	allocator.Random = fixedIndex(98)
	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10099 {
		t.Fatalf("got %d, want base-bucket value 10099", got)
	}
}

func TestAllocatorPrefersNotRecentlyUsedValueWithinSameBucket(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)

	// Make every base value active except 10037 and 10048. 10037 was just
	// released; 10048 has no history. Preferred pool must contain only 10048.
	fillActiveRange(t, db.SQL, 10001, 10099, 10037, now)
	releaseActive(t, db.SQL, 10048) // make 10048 free with old history
	insertReleasedHistory(t, db.SQL, 10037, now.Add(-time.Minute), "recent_10037")
	if _, err := db.SQL.Exec(`UPDATE amount_reservations SET last_used_at=? WHERE payable_amount_paise=10048`, now.Add(-24*time.Hour).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	allocator := NewAllocator()
	allocator.SoftHorizon = 4 * time.Hour
	allocator.Random = fixedIndex(0)
	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10048 {
		t.Fatalf("got %d, want older free amount 10048", got)
	}
}

func TestAllocatorReleasesDueReservationBeforeChoosing(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)
	fillActiveRange(t, db.SQL, 10001, 10099, -1, now)
	if _, err := db.SQL.Exec(`UPDATE amount_reservations SET reserved_at=?, reserved_until=?, last_used_at=? WHERE payable_amount_paise=10037`, now.Add(-20*time.Minute).UnixMilli(), now.Add(-time.Second).UnixMilli(), now.Add(-20*time.Minute).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	allocator := NewAllocator()
	allocator.SoftHorizon = 4 * time.Hour
	allocator.Random = fixedIndex(0)
	var got int64
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var err error
		got, err = allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 10037 {
		t.Fatalf("got %d, want newly released 10037 without premature overflow", got)
	}
}

func TestAllocatorReturnsCapacityOnlyWhenBothBucketsFull(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	now := time.UnixMilli(1_788_200_000_000)
	fillActiveRange(t, db.SQL, 10001, 10099, -1, now)
	fillActiveRange(t, db.SQL, 10101, 10199, -1, now)

	allocator := NewAllocator()
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		_, err := allocator.Select(ctx, tx, "paytm", 10000, now)
		return err
	})
	if !errors.Is(err, ErrPaymentCapacity) {
		t.Fatalf("error = %v, want ErrPaymentCapacity", err)
	}
}

func TestAllocatorRejectsNonWholeRequestedAmount(t *testing.T) {
	db := openAllocatorDB(t)
	ctx := context.Background()
	allocator := NewAllocator()
	err := db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		_, err := allocator.Select(ctx, tx, "paytm", 10001, time.Now())
		return err
	})
	if err == nil {
		t.Fatal("expected non-whole requested amount to fail")
	}
}

func fillActiveRange(t *testing.T, db *sql.DB, start, end, leaveFree int64, now time.Time) {
	t.Helper()
	for amount := start; amount <= end; amount++ {
		if amount == leaveFree {
			continue
		}
		id := fmt.Sprintf("p_%d", amount)
		insertTestPayment(t, db, id, amount, now)
		if _, err := db.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
            VALUES(?,?,?,?,?,?,?)`, "r_"+id, "paytm", amount, id, now.UnixMilli(), now.Add(15*time.Minute).UnixMilli(), now.UnixMilli()); err != nil {
			t.Fatal(err)
		}
	}
}

func fillReleasedRange(t *testing.T, db *sql.DB, start, end int64, lastUsed time.Time) {
	t.Helper()
	for amount := start; amount <= end; amount++ {
		insertReleasedHistory(t, db, amount, lastUsed, fmt.Sprintf("hist_%d", amount))
	}
}

func insertReleasedHistory(t *testing.T, db *sql.DB, amount int64, lastUsed time.Time, suffix string) {
	t.Helper()
	id := "p_" + suffix
	insertTestPayment(t, db, id, amount, lastUsed.Add(-time.Hour))
	reservedAt := lastUsed.Add(-20 * time.Minute).UnixMilli()
	releasedAt := lastUsed.Add(-5 * time.Minute).UnixMilli()
	if _, err := db.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,released_at,last_used_at)
        VALUES(?,?,?,?,?,?,?,?)`, "r_"+suffix, "paytm", amount, id, reservedAt, releasedAt, releasedAt, lastUsed.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}

func releaseActive(t *testing.T, db *sql.DB, amount int64) {
	t.Helper()
	if _, err := db.Exec(`UPDATE amount_reservations SET released_at=reserved_until WHERE payable_amount_paise=?`, amount); err != nil {
		t.Fatal(err)
	}
}

func insertTestPayment(t *testing.T, db *sql.DB, id string, payable int64, created time.Time) {
	t.Helper()
	requested := int64(10000)
	adjustment := payable - requested
	if adjustment <= 0 || adjustment > 199 {
		t.Fatalf("invalid test payable %d", payable)
	}
	now := created.UnixMilli()
	if _, err := db.Exec(`INSERT INTO payments(id,name,external_id,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
        VALUES(?,?,?,?,?,?,'paytm','merchant@paytm','pending',?,?,?,?)`, id, "Person", "evt_1", requested, payable, adjustment,
		now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatal(err)
	}
}
