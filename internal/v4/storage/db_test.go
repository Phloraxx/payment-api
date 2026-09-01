package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenConfiguresSQLiteAndSchema(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	checks := map[string]string{
		"PRAGMA journal_mode": "wal",
		"PRAGMA synchronous":  "2",
		"PRAGMA foreign_keys": "1",
		"PRAGMA busy_timeout": "5000",
	}
	for query, want := range checks {
		var got string
		if err := db.SQL.QueryRowContext(ctx, query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if !strings.EqualFold(strings.TrimSpace(got), want) {
			t.Fatalf("%s = %q, want %q", query, got, want)
		}
	}

	var version int
	if err := db.SQL.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
}

func TestOpenMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paygate.db")
	first, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	var count int
	if err := second.SQL.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration row count = %d, want 1", count)
	}
}

func TestCollectionProfilesEnforceSingleActiveProfile(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)

	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
        VALUES('kotak','Kotak','merchant@kotak','kotak_sms',1,1,?,?)`, now, now); err == nil {
		t.Fatal("expected second active profile to violate unique partial index")
	}
}

func TestPaymentIdentityDoesNotUseNameOrExternalID(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)

	insertPayment(t, db.SQL, "pay_1", "Sourav P Bijoy", "evt_123", 10037, now)
	insertPayment(t, db.SQL, "pay_2", "Sourav P Bijoy", "evt_123", 10048, now+1)

	var count int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM payments WHERE name='Sourav P Bijoy' AND external_id='evt_123'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestActiveAmountReservationIsUniqueButHistoryIsRetained(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)
	insertPayment(t, db.SQL, "pay_1", "Person A", "evt_123", 10037, now)
	insertPayment(t, db.SQL, "pay_2", "Person B", "evt_123", 10037, now+1)

	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_1','paytm',10037,'pay_1',?,?,?)`, now, now+900_000, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_2','paytm',10037,'pay_2',?,?,?)`, now+1, now+900_001, now+1); err == nil {
		t.Fatal("expected duplicate active profile+amount reservation to fail")
	}

	if _, err := db.SQL.Exec(`UPDATE amount_reservations SET released_at=? WHERE id='res_1'`, now+900_000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_2','paytm',10037,'pay_2',?,?,?)`, now+900_001, now+1_800_001, now+900_001); err != nil {
		t.Fatalf("reuse after release should succeed: %v", err)
	}

	var count int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM amount_reservations WHERE collection_profile_id='paytm' AND payable_amount_paise=10037`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("reservation history count = %d, want 2", count)
	}
}

func TestForeignKeysAndStrictTypesAreEnforced(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)

	if _, err := db.SQL.Exec(`INSERT INTO payments(id,name,external_id,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
        VALUES('pay_bad','Person','evt',10000,10037,37,'missing','x@y','pending',?,?,?,?,?)`, now, now+300_000, now+600_000, now+900_000); err == nil {
		t.Fatal("expected missing collection profile foreign key to fail")
	}

	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
        VALUES('bad','Bad','x@y','paytm_notification','not-an-int',0,?,?)`, now, now); err == nil {
		t.Fatal("expected STRICT integer column to reject text")
	}
}

func insertProfile(t *testing.T, db *sql.DB, id string, active bool, now int64) {
	t.Helper()
	activeInt := 0
	if active {
		activeInt = 1
	}
	parser := "paytm_notification"
	if id == "kotak" {
		parser = "kotak_sms"
	}
	if _, err := db.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
        VALUES(?,?,?,?,1,?,?,?)`, id, id, id+"@upi", parser, activeInt, now, now); err != nil {
		t.Fatal(err)
	}
}

func insertPayment(t *testing.T, db *sql.DB, id, name, externalID string, payable int64, now int64) {
	t.Helper()
	requested := int64(10000)
	adjustment := payable - requested
	if _, err := db.Exec(`INSERT INTO payments(id,name,external_id,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
        VALUES(?,?,?,?,?,?, 'paytm','merchant@paytm','pending',?,?,?,?)`,
		id, name, externalID, requested, payable, adjustment, now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatal(err)
	}
}

func TestSafetyPragmasApplyToEveryPooledConnection(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	conns := make([]*sql.Conn, 0, 4)
	for i := 0; i < 4; i++ {
		conn, err := db.SQL.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	for i, conn := range conns {
		var foreignKeys, synchronous, busy int
		var journal string
		if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
			t.Fatalf("conn %d foreign_keys: %v", i, err)
		}
		if err := conn.QueryRowContext(ctx, `PRAGMA synchronous`).Scan(&synchronous); err != nil {
			t.Fatalf("conn %d synchronous: %v", i, err)
		}
		if err := conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busy); err != nil {
			t.Fatalf("conn %d busy_timeout: %v", i, err)
		}
		if err := conn.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journal); err != nil {
			t.Fatalf("conn %d journal_mode: %v", i, err)
		}
		if foreignKeys != 1 || synchronous != 2 || busy != defaultBusyTimeoutMS || !strings.EqualFold(journal, "wal") {
			t.Fatalf("conn %d unsafe pragmas: fk=%d sync=%d busy=%d journal=%s", i, foreignKeys, synchronous, busy, journal)
		}
	}
}

func TestPaymentAmountAndStatusConstraints(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)

	cases := []struct {
		name      string
		requested int64
		payable   int64
		adjust    int64
		status    string
		paidAt    any
	}{
		{"dot-zero-payable", 10000, 10100, 100, "pending", nil},
		{"requested-has-paise", 10001, 10037, 36, "pending", nil},
		{"adjustment-too-large", 10000, 10201, 201, "pending", nil},
		{"invalid-status", 10000, 10037, 37, "review", nil},
		{"paid-without-paid-at", 10000, 10037, 37, "paid", nil},
	}
	for i, tc := range cases {
		_, err := db.SQL.Exec(`INSERT INTO payments(id,name,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after,paid_at)
            VALUES(?,?,?,?,?,'paytm','merchant@paytm',?,?,?,?,?,?)`,
			"bad_"+string(rune('a'+i)), tc.name, tc.requested, tc.payable, tc.adjust, tc.status,
			now, now+300_000, now+600_000, now+900_000, tc.paidAt)
		if err == nil {
			t.Fatalf("%s: expected constraint failure", tc.name)
		}
	}

	if _, err := db.SQL.Exec(`INSERT INTO payments(id,name,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
        VALUES('overflow_ok','Person',10000,10199,199,'paytm','merchant@paytm','pending',?,?,?,?)`,
		now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatalf("valid second-bucket amount rejected: %v", err)
	}
}

func TestRelayEventAndIdempotencyUniqueness(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)
	insertPayment(t, db.SQL, "pay_1", "Person", "evt_1", 10037, now)

	if _, err := db.SQL.Exec(`INSERT INTO relay_devices(id,public_key_pem,enabled,enrolled_at) VALUES('device_1','pem',1,?)`, now); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status) VALUES(?,?,?,?,?,?,?)`
	if _, err := db.SQL.Exec(insert, "relay_1", "device_1", "source_1", "com.paytm.business", now, now, "received"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(insert, "relay_2", "device_1", "source_1", "com.paytm.business", now, now, "received"); err == nil {
		t.Fatal("expected duplicate device/source event identity to fail")
	}
	if _, err := db.SQL.Exec(`INSERT INTO payment_observations(id,relay_event_id,source,collection_profile_id,amount_paise,occurred_at,occurred_at_source,received_at,matched_payment_id,match_result) VALUES('obs_1','relay_1','paytm_notification','paytm',10037,?,'notification_posted_at',?,'pay_1','matched')`, now, now); err != nil {
		t.Fatalf("valid observation timestamp source rejected: %v", err)
	}
	if _, err := db.SQL.Exec(insert, "relay_3", "device_1", "source_2", "com.paytm.business", now, now, "received"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO payment_observations(id,relay_event_id,source,collection_profile_id,amount_paise,occurred_at,occurred_at_source,received_at,match_result) VALUES('obs_bad','relay_3','paytm_notification','paytm',10037,?,'notification',?,'unmatched')`, now, now); err == nil {
		t.Fatal("expected vague/unsupported observation timestamp source to fail")
	}

	keyHash := []byte("same-key")
	requestHash := []byte("same-request")
	if _, err := db.SQL.Exec(`INSERT INTO idempotency_keys(scope,key_hash,request_hash,payment_id,created_at,expires_at) VALUES(?,?,?,?,?,?)`, "api_key_1", keyHash, requestHash, "pay_1", now, now+86_400_000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO idempotency_keys(scope,key_hash,request_hash,payment_id,created_at,expires_at) VALUES(?,?,?,?,?,?)`, "api_key_1", keyHash, requestHash, "pay_1", now, now+86_400_000); err == nil {
		t.Fatal("expected duplicate idempotency key hash to fail")
	}
}

func TestWithImmediateTxCommitsAndRollsBack(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := int64(1_788_200_000_000)

	if err := db.WithImmediateTx(ctx, func(tx *ImmediateTx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
            VALUES('paytm','Paytm','merchant@paytm','paytm_notification',1,1,?,?)`, now, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	wantErr := sql.ErrNoRows
	err := db.WithImmediateTx(ctx, func(tx *ImmediateTx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES('should_rollback','yes',?)`, now); err != nil {
			return err
		}
		return wantErr
	})
	if err != wantErr {
		t.Fatalf("rollback error = %v, want %v", err, wantErr)
	}
	var value string
	if err := db.SQL.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='should_rollback'`).Scan(&value); err != sql.ErrNoRows {
		t.Fatalf("rolled-back row query error = %v, want sql.ErrNoRows", err)
	}
}
func TestOrdinaryReadTransactionDoesNotAcquireWriterLock(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	readTx, err := db.SQL.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	var version int
	if err := readTx.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}

	if err := db.WithImmediateTx(ctx, func(tx *ImmediateTx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES('writer_probe','ok',1)`)
		return err
	}); err != nil {
		t.Fatalf("ordinary read transaction blocked writer: %v", err)
	}
}
