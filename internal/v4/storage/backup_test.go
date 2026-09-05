package storage

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupToCreatesVerifiedStandaloneSQLiteCopy(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.SQL.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,parser,enabled,active,created_at,updated_at)
		VALUES('paytm','Paytm','paygate@paytm','paytm_notification',1,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "backup.db")
	if err := db.BackupTo(ctx, destination); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode=%o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".paygate-backup-") {
			t.Fatalf("backup left temporary SQLite artifact %q", entry.Name())
		}
	}
	backup, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: destination, RawQuery: "mode=ro"}).String())
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var label string
	if err := backup.QueryRowContext(ctx, `SELECT label FROM collection_profiles WHERE id='paytm'`).Scan(&label); err != nil {
		t.Fatal(err)
	}
	if label != "Paytm" {
		t.Fatalf("backup label=%q", label)
	}
}

func TestBackupToRefusesLiveDatabasePath(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if err := db.BackupTo(ctx, db.Path); err == nil {
		t.Fatal("expected live-database backup refusal")
	}
}
func TestBackupToRefusesExistingDestination(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	destination := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(destination, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := db.BackupTo(ctx, destination); err == nil {
		t.Fatal("expected existing-destination refusal")
	}
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "keep-me" {
		t.Fatalf("existing destination was modified: %q", contents)
	}
}
