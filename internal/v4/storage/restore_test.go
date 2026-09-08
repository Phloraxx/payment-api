package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreDrillValidatesIsolatedBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "paygate.db")
	live, err := Open(ctx, livePath)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "backup.db")
	if err := live.BackupTo(ctx, backupPath); err != nil {
		live.Close()
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	report, err := RestoreDrill(ctx, backupPath, livePath, hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != schemaVersion || report.Payments != 0 || report.RelayEvents != 0 || report.WebhookDeliveries != 0 || report.RelayDevices != 0 {
		t.Fatalf("restore report = %+v", report)
	}
	if _, err := os.Stat(livePath); err != nil {
		t.Fatalf("live database was not preserved: %v", err)
	}
}

func TestRestoreDrillRejectsLivePathAndWrongHash(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "paygate.db")
	live, err := Open(ctx, livePath)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "backup.db")
	if err := live.BackupTo(ctx, backupPath); err != nil {
		live.Close()
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := RestoreDrill(ctx, backupPath, backupPath, ""); err == nil || !strings.Contains(err.Error(), "live database") {
		t.Fatalf("same path error = %v", err)
	}
	wrongHash := strings.Repeat("0", sha256.Size*2)
	if _, err := RestoreDrill(ctx, backupPath, livePath, wrongHash); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("wrong hash error = %v", err)
	}
}

func TestRestoreDrillRejectsTruncatedBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live.db")
	live, err := Open(ctx, livePath)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "backup.db")
	if err := live.BackupTo(ctx, backupPath); err != nil {
		live.Close()
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	truncatedPath := filepath.Join(dir, "truncated.db")
	if err := os.WriteFile(truncatedPath, raw[:len(raw)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreDrill(ctx, truncatedPath, livePath, ""); err == nil {
		t.Fatal("truncated backup was accepted")
	}
}

func TestRestoreDrillRejectsEmptyBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "empty.db")
	if err := os.WriteFile(backupPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreDrill(ctx, backupPath, filepath.Join(dir, "live.db"), ""); err == nil || !strings.Contains(err.Error(), "non-empty") {
		t.Fatalf("empty backup error = %v", err)
	}
}

func TestRestoreDrillRejectsHeaderOnlyDatabase(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "header-only.db")
	if err := os.WriteFile(backupPath, []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreDrill(ctx, backupPath, filepath.Join(dir, "live.db"), ""); err == nil {
		t.Fatal("header-only backup was accepted")
	}
}

func TestRestoreDrillRejectsMissingReservationUniqueness(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live.db")
	db, err := Open(ctx, livePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, `DROP INDEX uq_active_payable`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "backup.db")
	if err := db.BackupTo(ctx, backupPath); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreDrill(ctx, backupPath, livePath, ""); err == nil || !strings.Contains(err.Error(), "uq_active_payable") {
		t.Fatalf("missing uniqueness index error = %v", err)
	}
}

func TestRestoreDrillRejectsMissingGlobalAmountTrigger(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live.db")
	db, err := Open(ctx, livePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, `DROP TRIGGER trg_amount_reservations_global_unique_insert`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "backup.db")
	if err := db.BackupTo(ctx, backupPath); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreDrill(ctx, backupPath, livePath, ""); err == nil || !strings.Contains(err.Error(), "trg_amount_reservations_global_unique_insert") {
		t.Fatalf("missing global amount trigger error = %v", err)
	}
}

func buildV4RestoreFixture(t *testing.T, path string) (*sql.DB, *DB) {
	t.Helper()
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL) STRICT;`); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV1); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaV2); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(1,1),(2,2)`); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	db := &DB{SQL: raw, Path: path}
	if err := db.applyV3(ctx); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if err := db.runMigrationTx(ctx, 4, applyV4); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if err := db.ensureMultiRelayCompatibility(ctx); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if err := db.ensureRelayPayloadIntegrity(ctx); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	return raw, db
}

func finishRestoreFixture(t *testing.T, raw *sql.DB, path string) {
	t.Helper()
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, restoreFileMode); err != nil {
		t.Fatal(err)
	}
}

func installTransitionalRestoreState(ctx context.Context, db *DB, version int) error {
	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE amount_reservations ADD COLUMN global_unique_enforced INTEGER NOT NULL DEFAULT 1 CHECK(global_unique_enforced IN (0,1))`); err != nil {
		return err
	}
	if version >= 6 {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE relay_devices ADD COLUMN epoch_required INTEGER NOT NULL DEFAULT 0 CHECK(epoch_required IN (0,1))`); err != nil {
			return err
		}
	}
	if err := installGlobalAmountUniqueness(ctx, tx); err != nil {
		return err
	}
	for migration := 5; migration <= version; migration++ {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, migration, migration); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func installHistoricalMarkerAwareTransitionalState(ctx context.Context, db *DB, version int) error {
	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE amount_reservations ADD COLUMN global_unique_enforced INTEGER NOT NULL DEFAULT 1 CHECK(global_unique_enforced IN (0,1))`); err != nil {
		return err
	}
	if version >= 6 {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE relay_devices ADD COLUMN epoch_required INTEGER NOT NULL DEFAULT 0 CHECK(epoch_required IN (0,1))`); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
DROP INDEX IF EXISTS uq_active_profile_payable;
CREATE UNIQUE INDEX uq_active_payable ON amount_reservations(payable_amount_paise)
    WHERE released_at IS NULL AND global_unique_enforced=1;
CREATE TRIGGER trg_amount_reservations_global_unique_insert
BEFORE INSERT ON amount_reservations
WHEN NEW.global_unique_enforced<>1 OR (
    NEW.released_at IS NULL AND EXISTS (
        SELECT 1 FROM amount_reservations
        WHERE released_at IS NULL AND payable_amount_paise=NEW.payable_amount_paise
    )
)
BEGIN
    SELECT RAISE(ABORT, 'active payable amount already reserved');
END;
CREATE TRIGGER trg_amount_reservations_global_unique_update
BEFORE UPDATE OF payable_amount_paise,released_at,global_unique_enforced ON amount_reservations
WHEN (OLD.global_unique_enforced=1 AND NEW.global_unique_enforced<>1) OR (
    NEW.released_at IS NULL
    AND (NEW.global_unique_enforced=1 OR OLD.released_at IS NOT NULL OR NEW.payable_amount_paise<>OLD.payable_amount_paise)
    AND EXISTS (
        SELECT 1 FROM amount_reservations
        WHERE id<>NEW.id AND released_at IS NULL AND payable_amount_paise=NEW.payable_amount_paise
    )
)
BEGIN
    SELECT RAISE(ABORT, 'active payable amount already reserved');
END;`); err != nil {
		return err
	}
	for migration := 5; migration <= version; migration++ {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, migration, migration); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func TestRestoreDrillAcceptsHistoricalMarkerAwareTransitionalMigrations(t *testing.T) {
	for _, version := range []int{5, 7} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			backupPath := filepath.Join(dir, "backup.db")
			raw, db := buildV4RestoreFixture(t, backupPath)
			if err := installHistoricalMarkerAwareTransitionalState(ctx, db, version); err != nil {
				raw.Close()
				t.Fatal(err)
			}
			finishRestoreFixture(t, raw, backupPath)
			report, err := RestoreDrill(ctx, backupPath, filepath.Join(dir, "live.db"), "")
			if err != nil {
				t.Fatalf("restore historical marker-aware v%d backup: %v", version, err)
			}
			if report.SchemaVersion != schemaVersion {
				t.Fatalf("restored schema=%d want=%d", report.SchemaVersion, schemaVersion)
			}
		})
	}
}

func TestRestoreDrillAcceptsMarkerAwareInterruptedMigrations(t *testing.T) {
	for _, version := range []int{5, 6, 7} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			backupPath := filepath.Join(dir, "backup.db")
			raw, db := buildV4RestoreFixture(t, backupPath)
			if err := installTransitionalRestoreState(ctx, db, version); err != nil {
				raw.Close()
				t.Fatal(err)
			}
			finishRestoreFixture(t, raw, backupPath)
			report, err := RestoreDrill(ctx, backupPath, filepath.Join(dir, "live.db"), "")
			if err != nil {
				t.Fatalf("restore marker-aware v%d backup: %v", version, err)
			}
			if report.SchemaVersion != schemaVersion {
				t.Fatalf("restored schema=%d want=%d", report.SchemaVersion, schemaVersion)
			}
		})
	}
}

func TestRestoreDrillAcceptsLegacyV5GlobalIndexWithoutMarker(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "backup.db")
	raw, db := buildV4RestoreFixture(t, backupPath)
	tx, err := raw.BeginTx(ctx, nil)
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX uq_active_payable ON amount_reservations(payable_amount_paise) WHERE released_at IS NULL`); err != nil {
		tx.Rollback()
		raw.Close()
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(5,5)`); err != nil {
		tx.Rollback()
		raw.Close()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	_ = db
	finishRestoreFixture(t, raw, backupPath)
	report, err := RestoreDrill(ctx, backupPath, filepath.Join(dir, "live.db"), "")
	if err != nil {
		t.Fatalf("legacy v5 restore failed: %v", err)
	}
	if report.SchemaVersion != schemaVersion {
		t.Fatalf("restored schema=%d want=%d", report.SchemaVersion, schemaVersion)
	}
}
