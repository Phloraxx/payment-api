package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBusyTimeoutMS = 5000
	schemaVersion        = 2
)

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
		return fmt.Errorf("begin immediate transaction: %w", err)
	}
	done := false
	defer func() {
		if !done {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	if err := fn(&ImmediateTx{conn: conn}); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit immediate transaction: %w", err)
	}
	done = true
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
	if err := db.verifyPragmas(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	if err := db.migrate(ctx); err != nil {
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
