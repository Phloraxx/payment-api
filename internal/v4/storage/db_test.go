package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestActiveAmountReservationIsGloballyUniqueButHistoryIsRetained(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)
	insertProfile(t, db.SQL, "paytm", true, now)
	insertProfile(t, db.SQL, "kotak", false, now)
	insertPayment(t, db.SQL, "pay_1", "Person A", "evt_123", 10037, now)
	insertPayment(t, db.SQL, "pay_2", "Person B", "evt_123", 10037, now+1)
	if _, err := db.SQL.Exec(`UPDATE payments SET collection_profile_id='kotak' WHERE id='pay_2'`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_1','paytm',10037,'pay_1',?,?,?)`, now, now+900_000, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_2','kotak',10037,'pay_2',?,?,?)`, now+1, now+900_001, now+1); err == nil {
		t.Fatal("expected duplicate active payable amount to fail")
	}

	if _, err := db.SQL.Exec(`UPDATE amount_reservations SET released_at=? WHERE id='res_1'`, now+900_000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at)
        VALUES('res_2','kotak',10037,'pay_2',?,?,?)`, now+900_001, now+1_800_001, now+900_001); err != nil {
		t.Fatalf("reuse after release should succeed: %v", err)
	}

	var count int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM amount_reservations WHERE payable_amount_paise=10037`).Scan(&count); err != nil {
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
		{"adjustment-too-large", 10000, 10601, 601, "pending", nil},
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
        VALUES('overflow_ok','Person',10000,10599,599,'paytm','merchant@paytm','pending',?,?,?,?)`,
		now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatalf("valid sixth-bucket amount rejected: %v", err)
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
func TestOpenMigratesV4CrossProfileDuplicateReservationsWithoutRewritingAmounts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "paygate-v4-duplicates.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL) STRICT;`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV2); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(1,1),(2,2)`); err != nil {
		t.Fatal(err)
	}
	v4 := &DB{SQL: raw, Path: path}
	if err := v4.applyV3(ctx); err != nil {
		t.Fatal(err)
	}
	if err := v4.runMigrationTx(ctx, 4, applyV4); err != nil {
		t.Fatal(err)
	}

	now := int64(1_788_200_000_000)
	if _, err := raw.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at) VALUES
		('paytm','Paytm','paytm@upi','paytm_notification',1,1,?,?),
		('kotak','Kotak','kotak@upi','kotak_sms',1,0,?,?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO payments(id,name,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
		VALUES(?,?,'{}',10000,10037,37,?,?,'pending',?,?,?,?)`
	if _, err := raw.ExecContext(ctx, insert, "pay_paytm", "Paytm payer", "paytm", "paytm@upi", now, now+300_000, now+600_000, now+900_000); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, insert, "pay_kotak", "Kotak payer", "kotak", "kotak@upi", now+1, now+300_001, now+600_001, now+900_001); err != nil {
		t.Fatal(err)
	}
	reservation := `INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at) VALUES(?,?,?,?,?,?,?)`
	if _, err := raw.ExecContext(ctx, reservation, "res_paytm", "paytm", 10037, "pay_paytm", now, now+900_000, now); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, reservation, "res_kotak", "kotak", 10037, "pay_kotak", now+1, now+900_001, now+1); err != nil {
		t.Fatalf("v4 should allow cross-profile duplicate amount: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("upgrade with live v4 duplicates failed: %v", err)
	}
	defer db.Close()
	var version int
	if err := db.SQL.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version=%d want=%d", version, schemaVersion)
	}
	rows, err := db.SQL.QueryContext(ctx, `SELECT payment_id,payable_amount_paise,global_unique_enforced FROM amount_reservations ORDER BY payment_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var paymentID string
		var amount int64
		var enforced int
		if err := rows.Scan(&paymentID, &amount, &enforced); err != nil {
			t.Fatal(err)
		}
		if amount != 10037 || enforced != 0 {
			t.Fatalf("grandfathered reservation %s amount=%d enforced=%d", paymentID, amount, enforced)
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen != 2 {
		t.Fatalf("grandfathered reservations=%d want=2", seen)
	}

	if _, err := db.SQL.ExecContext(ctx, insert, "pay_blocked", "Blocked", "paytm", "paytm@upi", now+2, now+300_002, now+600_002, now+900_002); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, reservation, "res_blocked", "paytm", 10037, "pay_blocked", now+2, now+900_002, now+2); err == nil {
		t.Fatal("new reservation reused a grandfathered live amount")
	}
	if _, err := db.SQL.ExecContext(ctx, `UPDATE payments SET payable_amount_paise=10048,adjustment_paise=48 WHERE id='pay_blocked'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, reservation, "res_new", "paytm", 10048, "pay_blocked", now+2, now+900_002, now+2); err != nil {
		t.Fatalf("new globally unique reservation failed: %v", err)
	}
	var enforced int
	if err := db.SQL.QueryRowContext(ctx, `SELECT global_unique_enforced FROM amount_reservations WHERE id='res_new'`).Scan(&enforced); err != nil {
		t.Fatal(err)
	}
	if enforced != 1 {
		t.Fatalf("new reservation enforcement=%d want=1", enforced)
	}
}

func TestOpenWidensHistoricalPaymentAdjustmentConstraint(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "paygate-old-adjustment.db")
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	var createSQL string
	if err := raw.QueryRowContext(ctx, `SELECT sql FROM sqlite_schema WHERE type='table' AND name='payments'`).Scan(&createSQL); err != nil {
		t.Fatal(err)
	}
	oldSQL := strings.Replace(createSQL,
		"adjustment_paise INTEGER NOT NULL CHECK(adjustment_paise BETWEEN 1 AND 599)",
		"adjustment_paise INTEGER NOT NULL CHECK(adjustment_paise BETWEEN 1 AND 199)", 1)
	if oldSQL == createSQL {
		t.Fatal("current payments schema did not contain widened adjustment constraint")
	}
	if _, err := raw.ExecContext(ctx, "PRAGMA writable_schema=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `UPDATE sqlite_schema SET sql=? WHERE type='table' AND name='payments'`, oldSQL); err != nil {
		t.Fatal(err)
	}
	var schemaNumber int
	if err := raw.QueryRowContext(ctx, "PRAGMA schema_version").Scan(&schemaNumber); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, fmt.Sprintf("PRAGMA schema_version=%d", schemaNumber+1)); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, "PRAGMA writable_schema=OFF"); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen historical constraint database: %v", err)
	}
	defer db.Close()
	if err := db.SQL.QueryRowContext(ctx, `SELECT sql FROM sqlite_schema WHERE type='table' AND name='payments'`).Scan(&createSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(createSQL, "adjustment_paise BETWEEN 1 AND 599") {
		t.Fatalf("payments constraint was not widened: %s", createSQL)
	}
}

func TestOpenMigratesV1DatabaseToRelayHealthSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "paygate-v1.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL) STRICT;`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(1,1)`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.SQL.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version=%d want=%d", version, schemaVersion)
	}
	rows, err := db.SQL.QueryContext(ctx, `PRAGMA table_info(relay_devices)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "notification_access" {
			found = true
		}
	}
	if !found {
		t.Fatal("relay health columns were not added")
	}
}

func TestOpenMigratesV2ProfilesToDisabledLegacySupport(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "paygate-v2.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL) STRICT;`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV2); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(1,1),(2,2)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
		VALUES('kotak','Kotak','merchant@kotak','kotak_sms',1,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO payments(id,name,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,
		collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after)
		VALUES('pay_old','Unknown (legacy)','{}',10000,10001,1,'kotak','merchant@kotak','expired',1000,301000,601000,901000)`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var profile string
	if err := db.SQL.QueryRowContext(ctx, `SELECT collection_profile_id FROM payments WHERE id='pay_old'`).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if profile != "kotak" {
		t.Fatalf("payment profile=%q", profile)
	}
	if _, err := db.SQL.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
		VALUES('slice','Slice (legacy)','legacy@slice','legacy',0,0,3,3)`); err != nil {
		t.Fatalf("disabled legacy profile rejected: %v", err)
	}
	if _, err := db.SQL.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
		VALUES('bad','Bad legacy','legacy@bad','legacy',1,0,3,3)`); err == nil {
		t.Fatal("enabled legacy profile should be rejected")
	}
	rows, err := db.SQL.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign key violations after v3 profile rebuild")
	}
}

func TestObservationSchemaSupportsCorroborationAndFutureSources(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC().UnixMilli()
	if _, err := db.SQL.Exec(`INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at) VALUES('kotak','Kotak','merchant@kotak','kotak_sms',1,1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO payments(id,name,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,collection_profile_id,upi_id_snapshot,status,created_at,expires_at,grace_until,reuse_after,paid_at) VALUES('pay_future','Legacy','{}',10000,10037,37,'kotak','merchant@kotak','paid',?,?,?,?,?)`, now, now+300000, now+600000, now+900000, now+1000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO relay_devices(id,public_key_pem,enabled,enrolled_at) VALUES('device_future','pem',1,?)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status) VALUES('relay_future','device_future','src_future','future.package',?,?,'matched')`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO payment_observations(id,relay_event_id,source,collection_profile_id,amount_paise,occurred_at,occurred_at_source,received_at,matched_payment_id,match_result) VALUES('obs_future','relay_future','amazonpay_notification','kotak',10037,?,'notification_posted_at',?,'pay_future','corroborated')`, now, now); err != nil {
		t.Fatalf("future source/corroboration rejected: %v", err)
	}
}

func TestMultiRelayCompatibilityKeepsSchemaRollbackReadable(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version int
	if err := db.SQL.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version=%d want=%d for rollback compatibility", version, schemaVersion)
	}

	var indexes int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='uq_relay_devices_one_enabled'`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes != 0 {
		t.Fatalf("singleton relay index still present: %d", indexes)
	}
}

func TestRollbackV4SQLShapeRemainsOperational(t *testing.T) {
	db := openTestDB(t)
	now := int64(1_788_200_000_000)

	var version int
	if err := db.SQL.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 4 {
		t.Fatalf("rollback schema version=%d want=4", version)
	}

	insertProfile(t, db.SQL, "paytm", true, now)
	insertProfile(t, db.SQL, "kotak", false, now)
	insertPayment(t, db.SQL, "pay_v4_a", "A", "evt", 10037, now)
	insertPayment(t, db.SQL, "pay_v4_b", "B", "evt", 10037, now+1)
	if _, err := db.SQL.Exec(`UPDATE payments SET collection_profile_id='kotak',upi_id_snapshot='kotak@upi' WHERE id='pay_v4_b'`); err != nil {
		t.Fatal(err)
	}

	oldInsert := `INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,last_used_at) VALUES(?,?,?,?,?,?,?)`
	if _, err := db.SQL.Exec(oldInsert, "res_v4_a", "paytm", 10037, "pay_v4_a", now, now+900_000, now); err != nil {
		t.Fatalf("old-v4 reservation insert failed: %v", err)
	}
	if _, err := db.SQL.Exec(oldInsert, "res_v4_b", "kotak", 10037, "pay_v4_b", now+1, now+900_001, now+1); err == nil {
		t.Fatal("old-v4 SQL bypassed global active amount guard")
	}

	if _, err := db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at,app_version,device_model,android_version) VALUES('v4-device','Phone','pem',1,?,'v4','model','android')`, now); err != nil {
		t.Fatalf("old-v4 relay insert failed: %v", err)
	}
	var enabled int
	var enrolled int64
	if err := db.SQL.QueryRow(`SELECT enabled,enrolled_at FROM relay_devices WHERE id='v4-device'`).Scan(&enabled, &enrolled); err != nil {
		t.Fatalf("old-v4 relay read failed: %v", err)
	}
	if enabled != 1 || enrolled != now {
		t.Fatalf("old-v4 relay read enabled=%d enrolled=%d", enabled, enrolled)
	}
}
