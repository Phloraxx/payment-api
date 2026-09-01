package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func openAuthDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func testService(t *testing.T) (*Service, *storage.DB, *time.Time) {
	t.Helper()
	db := openAuthDB(t)
	now := time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC)
	s := NewService(db)
	s.Now = func() time.Time { return now }
	s.SessionTTL = 30 * time.Minute
	s.Argon = ArgonParams{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}
	return s, db, &now
}

func TestBootstrapAndAdminSession(t *testing.T) {
	s, db, _ := testService(t)
	ctx := context.Background()
	if err := s.BootstrapPassword(ctx, "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	if err := s.BootstrapPassword(ctx, "another password"); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("second bootstrap error = %v", err)
	}
	if _, err := s.CreateAdminSession(ctx, "wrong password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v", err)
	}
	session, err := s.CreateAdminSession(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(session.Token, "pg_admin_") {
		t.Fatalf("token = %q", session.Token)
	}
	if err := s.AuthenticateAdminSession(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(session.Token))
	var stored []byte
	if err := db.SQL.QueryRow(`SELECT token_hash FROM admin_sessions`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, hash[:]) {
		t.Fatal("admin session hash mismatch")
	}
	var plaintextCount int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE CAST(token_hash AS TEXT)=?`, session.Token).Scan(&plaintextCount); err != nil {
		t.Fatal(err)
	}
	if plaintextCount != 0 {
		t.Fatal("admin session token stored in plaintext")
	}
	if err := s.RevokeAdminSession(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if err := s.AuthenticateAdminSession(ctx, session.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("revoked session error = %v", err)
	}
}

func TestSessionExpiryAndPasswordChangeRevokesSessions(t *testing.T) {
	s, _, now := testService(t)
	ctx := context.Background()
	if err := s.BootstrapPassword(ctx, "initial password"); err != nil {
		t.Fatal(err)
	}
	session, err := s.CreateAdminSession(ctx, "initial password")
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(31 * time.Minute)
	if err := s.AuthenticateAdminSession(ctx, session.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expired session error = %v", err)
	}
	*now = now.Add(-31 * time.Minute)
	second, err := s.CreateAdminSession(ctx, "initial password")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassword(ctx, "replacement password"); err != nil {
		t.Fatal(err)
	}
	if err := s.AuthenticateAdminSession(ctx, second.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("session after password change = %v", err)
	}
	if _, err := s.CreateAdminSession(ctx, "initial password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password error = %v", err)
	}
	if _, err := s.CreateAdminSession(ctx, "replacement password"); err != nil {
		t.Fatalf("new password login: %v", err)
	}
}

func TestAPIKeyLifecycleAndHashedStorage(t *testing.T) {
	s, db, now := testService(t)
	ctx := context.Background()
	key, err := s.CreateAPIKey(ctx, "IEEE frontend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key.Secret, "pg_live_key_") || key.ID == "" {
		t.Fatalf("key = %+v", key)
	}
	id, err := s.AuthenticateAPIKey(ctx, key.Secret)
	if err != nil || id != key.ID {
		t.Fatalf("authenticate key id=%q err=%v", id, err)
	}
	var stored []byte
	if err := db.SQL.QueryRow(`SELECT secret_hash FROM api_keys WHERE id=?`, key.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte(key.Secret))
	if !bytes.Equal(stored, want[:]) {
		t.Fatal("API key hash mismatch")
	}
	var plaintextCount int
	if err := db.SQL.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE CAST(secret_hash AS TEXT)=?`, key.Secret).Scan(&plaintextCount); err != nil {
		t.Fatal(err)
	}
	if plaintextCount != 0 {
		t.Fatal("API key stored in plaintext")
	}
	*now = now.Add(6 * time.Minute)
	if _, err := s.AuthenticateAPIKey(ctx, key.Secret); err != nil {
		t.Fatal(err)
	}
	keys, err := s.ListAPIKeys(ctx)
	if err != nil || len(keys) != 1 || keys[0].LastUsedAt == nil {
		t.Fatalf("keys=%+v err=%v", keys, err)
	}
	if err := s.RevokeAPIKey(ctx, key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthenticateAPIKey(ctx, key.Secret); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("revoked key error = %v", err)
	}
}
func TestValidationAndNotInitialized(t *testing.T) {
	s, _, _ := testService(t)
	ctx := context.Background()
	if _, err := s.CreateAdminSession(ctx, "anything valid"); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("not initialized error = %v", err)
	}
	if err := s.BootstrapPassword(ctx, "short"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("short password error = %v", err)
	}
	if _, err := s.CreateAPIKey(ctx, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty label error = %v", err)
	}
	if _, err := s.AuthenticateAPIKey(ctx, "not-a-key"); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("malformed key error = %v", err)
	}
}

func TestMalformedPasswordHashFailsSafely(t *testing.T) {
	s, db, _ := testService(t)
	ctx := context.Background()
	if _, err := db.SQL.Exec(`INSERT INTO admin_credentials(singleton,password_hash,updated_at) VALUES(1,'$argon2id$bad',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAdminSession(ctx, "anything valid"); err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("malformed stored hash should be an internal validation error, got %v", err)
	}
}
