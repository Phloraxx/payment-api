package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
