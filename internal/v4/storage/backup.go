package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	sqlite "modernc.org/sqlite"
)

type onlineBackupConn interface {
	NewBackup(string) (*sqlite.Backup, error)
}

func (db *DB) BackupTo(ctx context.Context, destination string) (err error) {
	if db == nil || db.SQL == nil || strings.TrimSpace(db.Path) == "" {
		return errors.New("sqlite storage is required")
	}
	destination, err = filepath.Abs(strings.TrimSpace(destination))
	if err != nil || destination == "." {
		return errors.New("valid backup destination is required")
	}
	if filepath.Clean(destination) == filepath.Clean(db.Path) {
		return errors.New("backup destination cannot be the live database")
	}
	if _, statErr := os.Stat(destination); statErr == nil {
		return errors.New("backup destination already exists")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect backup destination: %w", statErr)
	}
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".paygate-backup-*.db")
	if err != nil {
		return fmt.Errorf("create backup temp file: %w", err)
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("prepare backup temp path: %w", err)
	}
	defer func() {
		if err != nil {
			_ = removeBackupTempArtifacts(tempPath)
		}
	}()

	source, err := db.SQL.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire backup source connection: %w", err)
	}
	defer source.Close()
	destinationURI := (&url.URL{Scheme: "file", Path: tempPath}).String()
	if err := source.Raw(func(driverConn any) error {
		conn, ok := driverConn.(onlineBackupConn)
		if !ok {
			return errors.New("sqlite driver does not expose online backup")
		}
		backup, err := conn.NewBackup(destinationURI)
		if err != nil {
			return fmt.Errorf("start sqlite online backup: %w", err)
		}
		more, stepErr := backup.Step(-1)
		finishErr := backup.Finish()
		if stepErr != nil {
			return fmt.Errorf("copy sqlite backup: %w", stepErr)
		}
		if more {
			return errors.New("sqlite online backup did not finish")
		}
		if finishErr != nil {
			return fmt.Errorf("finish sqlite online backup: %w", finishErr)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := verifyBackupFile(ctx, tempPath); err != nil {
		return err
	}
	if err := syncFile(tempPath); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		return fmt.Errorf("set backup permissions: %w", err)
	}
	// tempPath is created in the destination directory, so a hard link gives
	// us an atomic no-replace publish on the same filesystem. os.Rename would
	// overwrite a destination that appeared after the initial existence check.
	if err := os.Link(tempPath, destination); err != nil {
		return fmt.Errorf("publish backup without overwrite: %w", err)
	}
	if err := removeBackupTempArtifacts(tempPath); err != nil {
		_ = os.Remove(destination)
		return fmt.Errorf("remove published backup temp artifacts: %w", err)
	}
	if err := syncDir(dir); err != nil {
		return err
	}
	return nil
}

func removeBackupTempArtifacts(path string) error {
	var errs []error
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		artifact := path + suffix
		if err := os.Remove(artifact); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s: %w", filepath.Base(artifact), err))
		}
	}
	return errors.Join(errs...)
}

func verifyBackupFile(ctx context.Context, path string) error {
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	check, err := sql.Open("sqlite", uri)
	if err != nil {
		return fmt.Errorf("open backup for verification: %w", err)
	}
	defer check.Close()
	var integrity string
	if err := check.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return fmt.Errorf("verify backup integrity: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(integrity)) != "ok" {
		return fmt.Errorf("backup integrity check returned %q", integrity)
	}
	rows, err := check.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("verify backup foreign keys: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("backup foreign key check found violations")
	}
	return rows.Err()
}
func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open backup for sync: %w", err)
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync backup: %w", err)
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open backup directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync backup directory: %w", err)
	}
	return nil
}
