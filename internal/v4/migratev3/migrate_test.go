package migratev3

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
	_ "modernc.org/sqlite"
)

func TestRunMigratesLegacyPaymentsDeviceAndTombstones(t *testing.T) {
	archive := makeLegacyArchive(t, false)
	destination := filepath.Join(t.TempDir(), "v4.db")
	now := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	report, err := Run(context.Background(), Options{
		SourceZIP: archive, Destination: destination, ActiveProfile: "kotak",
		KotakUPIID: "merchant@kotak", PaytmUPIID: "merchant@paytm", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.MigratedPayments != 3 || report.LateNormalizedPaid != 1 || report.EventIDsRecovered != 2 || !report.MigratedDevice {
		t.Fatalf("report=%+v", report)
	}
	db, err := storage.Open(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	assertMigratedFixture(t, db)
}
func assertMigratedFixture(t *testing.T, db *storage.DB) {
	t.Helper()
	var count int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM payments`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("payments=%d err=%v", count, err)
	}
	var status, eventID, name string
	if err := db.SQL.QueryRow(`SELECT status,COALESCE(external_id,''),name FROM payments WHERE id='late1'`).Scan(&status, &eventID, &name); err != nil {
		t.Fatal(err)
	}
	if status != "paid" || eventID != "evt_1" || name != legacyUnknownName {
		t.Fatalf("late migration status=%s event=%s name=%s", status, eventID, name)
	}
	var parser string
	var enabled, active int
	if err := db.SQL.QueryRow(`SELECT parser,enabled,active FROM collection_profiles WHERE id='slice'`).Scan(&parser, &enabled, &active); err != nil {
		t.Fatal(err)
	}
	if parser != "legacy" || enabled != 0 || active != 0 {
		t.Fatalf("slice profile=%s/%d/%d", parser, enabled, active)
	}
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM webhook_deliveries`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("webhooks=%d err=%v", count, err)
	}
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM idempotency_keys`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("idempotency=%d err=%v", count, err)
	}
	var deviceID string
	if err := db.SQL.QueryRow(`SELECT id FROM relay_devices WHERE enabled=1`).Scan(&deviceID); err != nil {
		t.Fatal(err)
	}
	if deviceID != "device-fingerprint" {
		t.Fatalf("device=%q", deviceID)
	}
}

func TestMigratedLegacyIdempotencyKeyFailsClosedOnChangedRequest(t *testing.T) {
	archive := makeLegacyArchive(t, false)
	destination := filepath.Join(t.TempDir(), "v4.db")
	now := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	if _, err := Run(context.Background(), Options{SourceZIP: archive, Destination: destination, ActiveProfile: "kotak", KotakUPIID: "merchant@kotak", PaytmUPIID: "merchant@paytm", Now: func() time.Time { return now }}); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := payments.NewService(db)
	_, err = svc.Create(context.Background(), payments.CreateInput{RequestedAmountPaise: 10000, Name: "Different person", ExternalID: "evt_1", Metadata: []byte(`{"eventId":"evt_1"}`), IdempotencyScope: merchantIdempotencyScope, IdempotencyKey: "idem-late"})
	if !errors.Is(err, payments.ErrIdempotencyConflict) {
		t.Fatalf("changed retry err=%v", err)
	}
}
func TestRunRejectsUnverifiedSourceAndExistingDestination(t *testing.T) {
	archive := makeLegacyArchive(t, false)
	if err := os.Remove(archive + ".sha256"); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Options{SourceZIP: archive, Destination: filepath.Join(t.TempDir(), "v4.db"), ActiveProfile: "kotak", KotakUPIID: "m@kotak", PaytmUPIID: "m@paytm"})
	if err == nil {
		t.Fatal("unverified source should fail")
	}

	archive = makeLegacyArchive(t, false)
	destination := filepath.Join(t.TempDir(), "exists.db")
	if err := os.WriteFile(destination, []byte("do not replace"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), Options{SourceZIP: archive, Destination: destination, ActiveProfile: "kotak", KotakUPIID: "m@kotak", PaytmUPIID: "m@paytm"})
	if err == nil {
		t.Fatal("existing destination should fail")
	}
	raw, _ := os.ReadFile(destination)
	if string(raw) != "do not replace" {
		t.Fatal("existing destination was modified")
	}
}

func TestRunRejectsNonEmptySourceWAL(t *testing.T) {
	archive := makeLegacyArchive(t, false, []byte("uncheckpointed transaction"))
	destination := filepath.Join(t.TempDir(), "v4.db")
	_, err := Run(context.Background(), Options{
		SourceZIP: archive, Destination: destination, ActiveProfile: "kotak",
		KotakUPIID: "m@kotak", PaytmUPIID: "m@paytm",
	})
	if err == nil || !strings.Contains(err.Error(), "non-empty data.db-wal") {
		t.Fatalf("non-empty WAL error=%v", err)
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("destination exists after WAL preflight failure: %v", statErr)
	}
}

func TestRunRejectsActiveReservationCollisionBeforeCreatingDestination(t *testing.T) {
	archive := makeLegacyArchive(t, true)
	destination := filepath.Join(t.TempDir(), "v4.db")
	now := time.Date(2026, 8, 31, 10, 5, 0, 0, time.UTC)
	_, err := Run(context.Background(), Options{SourceZIP: archive, Destination: destination, ActiveProfile: "kotak", KotakUPIID: "m@kotak", PaytmUPIID: "m@paytm", Now: func() time.Time { return now }})
	if err == nil {
		t.Fatal("active collision should fail")
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("destination exists after preflight failure: %v", statErr)
	}
}

func makeLegacyArchive(t *testing.T, activeCollision bool, wal ...[]byte) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	paymentsDDL := `CREATE TABLE payments(
		id TEXT PRIMARY KEY,external_id TEXT,metadata JSON,status TEXT,payment_account TEXT,customer_name TEXT,payer_name TEXT,upi_id TEXT,
		display_name TEXT,customer_email TEXT,customer_phone TEXT,description TEXT,admin_note TEXT,idempotency_key TEXT,tags JSON,custom_fields JSON,
		requested_amount NUMERIC,payable_amount NUMERIC,created_at TEXT,expires_at TEXT,reuse_after TEXT,paid_at TEXT,resolved_at TEXT);`
	deviceDDL := `CREATE TABLE relay_devices(
		device_id TEXT,name TEXT,public_key_pem TEXT,enabled INTEGER,enrolled_at TEXT,created TEXT,last_seen_at TEXT,last_heartbeat_at TEXT,
		app_version TEXT,device_model TEXT,android_version TEXT,notification_access INTEGER,listener_connected INTEGER,battery_optimization_exempt INTEGER,
		power_save_mode INTEGER,background_restricted INTEGER,foreground_service_active INTEGER,pending_count INTEGER,failed_count INTEGER,
		last_client_delivery_at TEXT,last_client_error TEXT,updated TEXT);`
	if _, err := db.Exec(paymentsDDL + deviceDDL); err != nil {
		t.Fatal(err)
	}
	insertLegacyFixtures(t, db, activeCollision)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "backup.zip")
	writeZipWithDataDB(t, archive, dbPath, wal...)
	writeSHA256Sidecar(t, archive)
	return archive
}

func insertLegacyFixtures(t *testing.T, db *sql.DB, activeCollision bool) {
	t.Helper()
	insert := func(id, account, status string, requested, payable int64, created, expires, reuse, paid, resolved, metadata, externalID, idem string) {
		_, err := db.Exec(`INSERT INTO payments(id,external_id,metadata,status,payment_account,customer_name,payer_name,upi_id,
			display_name,customer_email,customer_phone,description,admin_note,idempotency_key,tags,custom_fields,
			requested_amount,payable_amount,created_at,expires_at,reuse_after,paid_at,resolved_at)
			VALUES(?,?,?,?,?,'','Payer','payer@upi','','','','','',?,NULL,NULL,?,?,?,?,?,?,?)`,
			id, externalID, metadata, status, account, idem, requested, payable, created, expires, reuse, paid, resolved)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("late1", "kotak", "late", 10000, 10037, "2026-08-30 10:00:00.000Z", "2026-08-30 10:05:00.000Z", "2026-08-30 10:15:00.000Z", "2026-08-30 10:03:00.000Z", "2026-08-30 10:03:00.000Z", `{"eventId":"evt_1"}`, "legacy:registration:1", "idem-late")
	insert("expired1", "paytm", "expired", 20000, 20042, "2026-08-30 11:00:00.000Z", "2026-08-30 11:05:00.000Z", "2026-08-30 11:15:00.000Z", "", "2026-08-30 11:10:00.000Z", `{"eventId":"evt_2"}`, "legacy:registration:2", "idem-expired")
	insert("slice1", "slice", "paid", 30000, 30077, "2026-08-30 12:00:00.000Z", "2026-08-30 12:05:00.000Z", "2026-08-30 12:15:00.000Z", "2026-08-30 12:02:00.000Z", "2026-08-30 12:02:00.000Z", `{}`, "legacy:registration:3", "idem-slice")
	if activeCollision {
		insert("pending-a", "kotak", "pending", 40000, 40037, "2026-08-31 10:00:00.000Z", "2026-08-31 10:05:00.000Z", "2026-08-31 10:15:00.000Z", "", "", `{}`, "", "idem-a")
		insert("pending-b", "kotak", "pending", 40000, 40037, "2026-08-31 10:01:00.000Z", "2026-08-31 10:06:00.000Z", "2026-08-31 10:16:00.000Z", "", "", `{}`, "", "idem-b")
	}
	_, err := db.Exec(`INSERT INTO relay_devices(device_id,name,public_key_pem,enabled,enrolled_at,created,last_seen_at,last_heartbeat_at,app_version,device_model,android_version,
		notification_access,listener_connected,battery_optimization_exempt,power_save_mode,background_restricted,foreground_service_active,pending_count,failed_count,last_client_delivery_at,last_client_error,updated)
		VALUES('device-fingerprint','Phone','pem',1,'2026-08-27 12:00:00.000Z','2026-08-27 12:00:00.000Z','2026-09-01 02:58:00.000Z','2026-09-01 02:58:00.000Z','0.4.2','Edge 60','16',1,1,1,0,0,1,0,0,'2026-09-01 02:57:00.000Z','','2026-09-01 02:58:00.000Z')`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeZipWithDataDB(t *testing.T, archive, dbPath string, wal ...[]byte) {
	t.Helper()
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	entry, err := zw.Create("data.db")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(raw); err != nil {
		t.Fatal(err)
	}
	if len(wal) > 0 {
		walEntry, err := zw.Create("data.db-wal")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := walEntry.Write(wal[0]); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeSHA256Sidecar(t *testing.T, archive string) {
	t.Helper()
	raw, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	content := hex.EncodeToString(sum[:]) + "  " + filepath.Base(archive) + "\n"
	if err := os.WriteFile(archive+".sha256", []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
