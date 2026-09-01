package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func openTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func testProfile(id string, enabled bool) UpsertInput {
	parser := "paytm_notification"
	if id == "kotak" {
		parser = "kotak_sms"
	}
	return UpsertInput{ID: id, Label: id, UPIID: id + "@upi", PayeeName: "PayGate", Parser: parser, Enabled: enabled}
}
func TestActivateSwitchesAtomically(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	svc.Now = func() time.Time { return time.UnixMilli(1_788_200_000_000).UTC() }

	if _, err := svc.Upsert(ctx, testProfile("paytm", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, testProfile("kotak", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Activate(ctx, "kotak")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.ID != "kotak" {
		t.Fatalf("active profile = %+v", got)
	}

	profiles, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || profiles[0].ID != "kotak" || !profiles[0].Active || profiles[1].Active {
		t.Fatalf("profiles after switch = %+v", profiles)
	}
}
func TestDisabledProfileCannotBeActivatedOrDisableCurrentActive(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)

	if _, err := svc.Upsert(ctx, testProfile("paytm", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, testProfile("paytm", false)); !errors.Is(err, ErrCannotDisableActiveProfile) {
		t.Fatalf("disable active error = %v", err)
	}
	current, err := svc.Get(ctx, "paytm")
	if err != nil {
		t.Fatal(err)
	}
	if !current.Enabled || !current.Active {
		t.Fatalf("active profile changed after rejected disable: %+v", current)
	}

	if _, err := svc.Upsert(ctx, testProfile("kotak", false)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Activate(ctx, "kotak"); !errors.Is(err, ErrProfileDisabled) {
		t.Fatalf("activate disabled error = %v", err)
	}
}
func TestProfileSwitchDoesNotMutateExistingPaymentSnapshot(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	now := int64(1_788_200_000_000)

	if _, err := svc.Upsert(ctx, testProfile("paytm", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, testProfile("kotak", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO payments(id,name,external_id,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,payee_name_snapshot,status,created_at,expires_at,grace_until,reuse_after)
		VALUES('pay_1','Sourav','evt_1',10000,10037,37,'paytm','paytm@upi','PayGate','pending',?,?,?,?)`, now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Activate(ctx, "kotak"); err != nil {
		t.Fatal(err)
	}
	var profileID, upiID string
	if err := db.SQL.QueryRow(`SELECT collection_profile_id,upi_id_snapshot FROM payments WHERE id='pay_1'`).Scan(&profileID, &upiID); err != nil {
		t.Fatal(err)
	}
	if profileID != "paytm" || upiID != "paytm@upi" {
		t.Fatalf("payment snapshot changed: profile=%q upi=%q", profileID, upiID)
	}
}
func TestProfileValidationAndNotFound(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)

	badUPI := testProfile("paytm", true)
	badUPI.UPIID = "not-a-vpa"
	if _, err := svc.Upsert(ctx, badUPI); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("bad upi error = %v", err)
	}
	badParser := testProfile("paytm", true)
	badParser.Parser = "gpay"
	if _, err := svc.Upsert(ctx, badParser); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("bad parser error = %v", err)
	}
	if _, err := svc.Get(ctx, "missing"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("missing get error = %v", err)
	}
	if _, err := svc.Activate(ctx, "missing"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("missing activate error = %v", err)
	}
}
