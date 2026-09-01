package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotInitialized     = errors.New("admin password is not initialized")
	ErrAlreadyInitialized = errors.New("admin password is already initialized")
	ErrInvalidSession     = errors.New("invalid admin session")
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrInvalidInput       = errors.New("invalid authentication input")
)

type ArgonParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltBytes   uint32
	KeyBytes    uint32
}

type Service struct {
	DB         *storage.DB
	Now        func() time.Time
	Random     io.Reader
	SessionTTL time.Duration
	Argon      ArgonParams
}

type AdminSession struct {
	Token     string
	ExpiresAt time.Time
}

type APIKey struct {
	ID        string
	Label     string
	Secret    string
	CreatedAt time.Time
}

type APIKeyInfo struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func NewService(db *storage.DB) *Service {
	return &Service{
		DB:         db,
		Now:        time.Now,
		Random:     rand.Reader,
		SessionTTL: 24 * time.Hour,
		Argon: ArgonParams{
			MemoryKiB:   64 * 1024,
			Iterations:  3,
			Parallelism: 2,
			SaltBytes:   16,
			KeyBytes:    32,
		},
	}
}

func (s *Service) BootstrapPassword(ctx context.Context, password string) error {
	if err := s.ready(); err != nil {
		return err
	}
	encoded, err := s.hashPassword(password)
	if err != nil {
		return err
	}
	now := s.now().UnixMilli()
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_credentials`).Scan(&count); err != nil {
			return fmt.Errorf("read admin credential state: %w", err)
		}
		if count != 0 {
			return ErrAlreadyInitialized
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO admin_credentials(singleton,password_hash,updated_at) VALUES(1,?,?)`, encoded, now)
		if err != nil {
			return fmt.Errorf("store admin password: %w", err)
		}
		return nil
	})
}
func (s *Service) SetPassword(ctx context.Context, password string) error {
	if err := s.ready(); err != nil {
		return err
	}
	encoded, err := s.hashPassword(password)
	if err != nil {
		return err
	}
	now := s.now().UnixMilli()
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO admin_credentials(singleton,password_hash,updated_at) VALUES(1,?,?)
			ON CONFLICT(singleton) DO UPDATE SET password_hash=excluded.password_hash,updated_at=excluded.updated_at`, encoded, now)
		if err != nil {
			return fmt.Errorf("store admin password: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions`); err != nil {
			return fmt.Errorf("revoke admin sessions: %w", err)
		}
		return nil
	})
}

func (s *Service) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	if err := s.ready(); err != nil {
		return err
	}
	encodedNew, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}
	now := s.now().UnixMilli()
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var encodedCurrent string
		if err := tx.QueryRowContext(ctx, `SELECT password_hash FROM admin_credentials WHERE singleton=1`).Scan(&encodedCurrent); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotInitialized
			}
			return fmt.Errorf("read admin password: %w", err)
		}
		ok, err := verifyPassword(encodedCurrent, currentPassword)
		if err != nil {
			return fmt.Errorf("verify admin password: %w", err)
		}
		if !ok {
			return ErrInvalidCredentials
		}
		if _, err := tx.ExecContext(ctx, `UPDATE admin_credentials SET password_hash=?,updated_at=? WHERE singleton=1`, encodedNew, now); err != nil {
			return fmt.Errorf("store admin password: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions`); err != nil {
			return fmt.Errorf("revoke admin sessions: %w", err)
		}
		return nil
	})
}

func (s *Service) CreateAdminSession(ctx context.Context, password string) (AdminSession, error) {
	if err := s.ready(); err != nil {
		return AdminSession{}, err
	}
	var encoded string
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT password_hash FROM admin_credentials WHERE singleton=1`).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminSession{}, ErrNotInitialized
	}
	if err != nil {
		return AdminSession{}, fmt.Errorf("read admin password: %w", err)
	}
	ok, err := verifyPassword(encoded, password)
	if err != nil {
		return AdminSession{}, fmt.Errorf("verify admin password: %w", err)
	}
	if !ok {
		return AdminSession{}, ErrInvalidCredentials
	}
	token, err := s.randomToken("pg_admin_", 32)
	if err != nil {
		return AdminSession{}, fmt.Errorf("generate admin session: %w", err)
	}
	hash := sha256.Sum256([]byte(token))
	now := s.now()
	ttl := s.SessionTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expires := now.Add(ttl)
	_, err = s.DB.SQL.ExecContext(ctx, `INSERT INTO admin_sessions(token_hash,created_at,expires_at,last_seen_at) VALUES(?,?,?,?)`,
		hash[:], now.UnixMilli(), expires.UnixMilli(), now.UnixMilli())
	if err != nil {
		return AdminSession{}, fmt.Errorf("store admin session: %w", err)
	}
	return AdminSession{Token: token, ExpiresAt: expires}, nil
}

func (s *Service) AuthenticateAdminSession(ctx context.Context, token string) error {
	if err := s.ready(); err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "pg_admin_") || len(token) < 32 {
		return ErrInvalidSession
	}
	hash := sha256.Sum256([]byte(token))
	now := s.now()
	var expiresAt int64
	var revokedAt sql.NullInt64
	var lastSeen sql.NullInt64
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT expires_at,revoked_at,last_seen_at FROM admin_sessions WHERE token_hash=?`, hash[:]).
		Scan(&expiresAt, &revokedAt, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) || revokedAt.Valid || now.UnixMilli() >= expiresAt {
		return ErrInvalidSession
	}
	if err != nil {
		return fmt.Errorf("read admin session: %w", err)
	}
	if !lastSeen.Valid || now.UnixMilli()-lastSeen.Int64 >= int64(5*time.Minute/time.Millisecond) {
		_, _ = s.DB.SQL.ExecContext(ctx, `UPDATE admin_sessions SET last_seen_at=? WHERE token_hash=? AND revoked_at IS NULL`, now.UnixMilli(), hash[:])
	}
	return nil
}
func (s *Service) RevokeAdminSession(ctx context.Context, token string) error {
	if err := s.ready(); err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	result, err := s.DB.SQL.ExecContext(ctx, `UPDATE admin_sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL`, s.now().UnixMilli(), hash[:])
	if err != nil {
		return fmt.Errorf("revoke admin session: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return ErrInvalidSession
	}
	return nil
}

func (s *Service) CreateAPIKey(ctx context.Context, label string) (APIKey, error) {
	if err := s.ready(); err != nil {
		return APIKey{}, err
	}
	label = strings.TrimSpace(label)
	if label == "" || len([]rune(label)) > 120 {
		return APIKey{}, fmt.Errorf("%w: API key label must contain 1-120 characters", ErrInvalidInput)
	}
	idPart, err := s.randomBytes(12)
	if err != nil {
		return APIKey{}, fmt.Errorf("generate API key id: %w", err)
	}
	secretPart, err := s.randomBytes(32)
	if err != nil {
		return APIKey{}, fmt.Errorf("generate API key secret: %w", err)
	}
	id := "key_" + base64.RawURLEncoding.EncodeToString(idPart)
	secret := "pg_live_" + id + "_" + base64.RawURLEncoding.EncodeToString(secretPart)
	hash := sha256.Sum256([]byte(secret))
	now := s.now()
	_, err = s.DB.SQL.ExecContext(ctx, `INSERT INTO api_keys(id,label,secret_hash,enabled,created_at) VALUES(?,?,?,1,?)`, id, label, hash[:], now.UnixMilli())
	if err != nil {
		return APIKey{}, fmt.Errorf("store API key: %w", err)
	}
	return APIKey{ID: id, Label: label, Secret: secret, CreatedAt: now}, nil
}
func (s *Service) AuthenticateAPIKey(ctx context.Context, secret string) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	secret = strings.TrimSpace(secret)
	if !strings.HasPrefix(secret, "pg_live_key_") || len(secret) < 48 {
		return "", ErrInvalidAPIKey
	}
	hash := sha256.Sum256([]byte(secret))
	var id string
	var lastUsed sql.NullInt64
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT id,last_used_at FROM api_keys WHERE secret_hash=? AND enabled=1`, hash[:]).Scan(&id, &lastUsed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidAPIKey
	}
	if err != nil {
		return "", fmt.Errorf("read API key: %w", err)
	}
	now := s.now().UnixMilli()
	if !lastUsed.Valid || now-lastUsed.Int64 >= int64(5*time.Minute/time.Millisecond) {
		_, _ = s.DB.SQL.ExecContext(ctx, `UPDATE api_keys SET last_used_at=? WHERE id=? AND enabled=1`, now, id)
	}
	return id, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, id string) error {
	if err := s.ready(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	result, err := s.DB.SQL.ExecContext(ctx, `UPDATE api_keys SET enabled=0 WHERE id=? AND enabled=1`, id)
	if err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return ErrInvalidAPIKey
	}
	return nil
}

func (s *Service) ListAPIKeys(ctx context.Context) ([]APIKeyInfo, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT id,label,enabled,created_at,last_used_at FROM api_keys ORDER BY created_at DESC,id`)
	if err != nil {
		return nil, fmt.Errorf("list API keys: %w", err)
	}
	defer rows.Close()
	var out []APIKeyInfo
	for rows.Next() {
		var info APIKeyInfo
		var enabled int
		var created int64
		var lastUsed sql.NullInt64
		if err := rows.Scan(&info.ID, &info.Label, &enabled, &created, &lastUsed); err != nil {
			return nil, fmt.Errorf("scan API key: %w", err)
		}
		info.Enabled = enabled == 1
		info.CreatedAt = time.UnixMilli(created).UTC()
		if lastUsed.Valid {
			value := time.UnixMilli(lastUsed.Int64).UTC()
			info.LastUsedAt = &value
		}
		out = append(out, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate API keys: %w", err)
	}
	return out, nil
}

func (s *Service) ready() error {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return errors.New("authentication storage is required")
	}
	return nil
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
func (s *Service) hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	params := s.Argon
	if params == (ArgonParams{}) {
		params = NewService(nil).Argon
	}
	if params.MemoryKiB < 8*1024 || params.MemoryKiB > 1024*1024 || params.Iterations < 1 || params.Iterations > 10 ||
		params.Parallelism < 1 || params.Parallelism > 16 || params.SaltBytes < 8 || params.SaltBytes > 64 ||
		params.KeyBytes < 16 || params.KeyBytes > 64 {
		return "", errors.New("invalid Argon2 parameters")
	}
	salt, err := s.randomBytes(int(params.SaltBytes))
	if err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyBytes)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, params.MemoryKiB, params.Iterations,
		params.Parallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func verifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid password hash format")
	}
	versionText := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionText)
	if err != nil || version != argon2.Version {
		return false, errors.New("unsupported password hash version")
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, errors.New("invalid password hash parameters")
	}
	if memory < 8*1024 || memory > 1024*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 16 {
		return false, errors.New("invalid password hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false, errors.New("invalid password hash salt")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 64 {
		return false, errors.New("invalid password hash value")
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 1024 {
		return fmt.Errorf("%w: password must contain 8-1024 bytes", ErrInvalidInput)
	}
	return nil
}

func (s *Service) randomToken(prefix string, bytes int) (string, error) {
	raw, err := s.randomBytes(bytes)
	if err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Service) randomBytes(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("random byte length must be positive")
	}
	r := s.Random
	if r == nil {
		r = rand.Reader
	}
	out := make([]byte, size)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	return out, nil
}
