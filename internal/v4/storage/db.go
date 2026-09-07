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
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBusyTimeoutMS = 5000
	schemaVersion        = 7
)

var ErrBusy = errors.New("sqlite database busy")

type DB struct {
	SQL  *sql.DB
	Path string
}

// ImmediateTx is a short BEGIN IMMEDIATE transaction. It deliberately exposes
// only SQL execution primitives, not the underlying pooled connection.
type ImmediateTx struct {
	conn *sql.Conn
}

func (tx *ImmediateTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.conn.ExecContext(ctx, query, args...)
}

func (tx *ImmediateTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.conn.QueryContext(ctx, query, args...)
}

func (tx *ImmediateTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.conn.QueryRowContext(ctx, query, args...)
}

type sqliteCodeError interface {
	Code() int
}

func isSQLiteBusy(err error) bool {
	if errors.Is(err, ErrBusy) {
		return true
	}
	var coded sqliteCodeError
	if errors.As(err, &coded) {
		baseCode := coded.Code() & 0xff
		return baseCode == 5 || baseCode == 6
	}
	message := strings.ToUpper(err.Error())
	return strings.Contains(message, "SQLITE_BUSY") || strings.Contains(message, "SQLITE_LOCKED")
}

func wrapTransactionError(operation string, err error) error {
	if isSQLiteBusy(err) {
		return fmt.Errorf("%w: %s: %w", ErrBusy, operation, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// WithImmediateTx runs fn inside BEGIN IMMEDIATE on a dedicated pooled connection.
// Use it for short payment-critical write transactions. Network calls must never
// happen inside fn. Ordinary reads should use DB.SQL directly.
func (db *DB) WithImmediateTx(ctx context.Context, fn func(*ImmediateTx) error) (err error) {
	conn, err := db.SQL.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire sqlite connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return wrapTransactionError("begin immediate transaction", err)
	}
	done := false
	defer func() {
		if !done {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	if err := fn(&ImmediateTx{conn: conn}); err != nil {
		if isSQLiteBusy(err) {
			return wrapTransactionError("immediate transaction callback", err)
		}
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return wrapTransactionError("commit immediate transaction", err)
	}
	done = true
	return nil
}

const databaseFileMode os.FileMode = 0o600

func prepareDatabaseFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		file, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, databaseFileMode)
		if createErr != nil {
			return fmt.Errorf("create sqlite database: %w", createErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("close sqlite database: %w", closeErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect sqlite database: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("sqlite database must be a regular file")
	}
	if info.Mode().Perm() != databaseFileMode {
		if err := os.Chmod(path, databaseFileMode); err != nil {
			return fmt.Errorf("harden sqlite database permissions: %w", err)
		}
	}
	return nil
}

func verifyDatabaseSidecars(path string) error {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		sidecar := path + suffix
		info, err := os.Lstat(sidecar)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect sqlite %s file: %w", suffix, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("sqlite %s file must be a regular file", suffix)
		}
		if info.Mode().Perm() != databaseFileMode {
			if err := os.Chmod(sidecar, databaseFileMode); err != nil {
				return fmt.Errorf("harden sqlite %s permissions: %w", suffix, err)
			}
		}
	}
	return nil
}

func Open(ctx context.Context, path string) (*DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve sqlite path: %w", err)
	}
	if err := prepareDatabaseFile(abs); err != nil {
		return nil, err
	}
	if err := verifyDatabaseSidecars(abs); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(FULL)")
	q.Add("_pragma", "foreign_keys(ON)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", defaultBusyTimeoutMS))
	q.Set("_dqs", "false")

	dsn := "file:" + filepath.ToSlash(abs) + "?" + q.Encode()
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	raw.SetMaxOpenConns(8)
	raw.SetMaxIdleConns(8)
	raw.SetConnMaxLifetime(0)
	raw.SetConnMaxIdleTime(5 * time.Minute)

	db := &DB{SQL: raw, Path: abs}
	if err := raw.PingContext(ctx); err != nil {
		raw.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := verifyDatabaseSidecars(abs); err != nil {
		raw.Close()
		return nil, err
	}
	if err := db.verifyPragmas(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	if err := db.migrate(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	if err := db.ensureMultiRelayCompatibility(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	if err := verifyDatabaseSidecars(abs); err != nil {
		raw.Close()
		return nil, err
	}
	if err := db.ensureRelayPayloadIntegrity(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	if db == nil || db.SQL == nil {
		return nil
	}
	return db.SQL.Close()
}

func (db *DB) verifyPragmas(ctx context.Context) error {
	checks := []struct {
		query string
		want  string
	}{
		{"PRAGMA journal_mode", "wal"},
		{"PRAGMA synchronous", "2"},
		{"PRAGMA foreign_keys", "1"},
		{"PRAGMA busy_timeout", fmt.Sprint(defaultBusyTimeoutMS)},
	}
	for _, check := range checks {
		var got string
		if err := db.SQL.QueryRowContext(ctx, check.query).Scan(&got); err != nil {
			return fmt.Errorf("verify %s: %w", check.query, err)
		}
		if !strings.EqualFold(strings.TrimSpace(got), check.want) {
			return fmt.Errorf("verify %s: got %q want %q", check.query, got, check.want)
		}
	}
	return nil
}
