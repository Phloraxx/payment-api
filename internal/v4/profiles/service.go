package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var (
	ErrProfileNotFound            = errors.New("collection profile not found")
	ErrProfileDisabled            = errors.New("collection profile is disabled")
	ErrCannotDisableActiveProfile = errors.New("cannot disable active collection profile")
	ErrInvalidProfile             = errors.New("invalid collection profile")
)

type Service struct {
	DB  *storage.DB
	Now func() time.Time
}
type Profile struct {
	ID        string
	Label     string
	UPIID     string
	PayeeName string
	Parser    string
	Enabled   bool
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpsertInput struct {
	ID        string
	Label     string
	UPIID     string
	PayeeName string
	Parser    string
	Enabled   bool
}

func NewService(db *storage.DB) *Service {
	return &Service{DB: db, Now: time.Now}
}

func (s *Service) Upsert(ctx context.Context, in UpsertInput) (Profile, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return Profile{}, errors.New("profile storage is required")
	}
	in.ID = strings.ToLower(strings.TrimSpace(in.ID))
	in.Label = strings.TrimSpace(in.Label)
	in.UPIID = strings.TrimSpace(in.UPIID)
	in.PayeeName = strings.TrimSpace(in.PayeeName)
	in.Parser = strings.TrimSpace(in.Parser)
	if err := validateUpsert(in); err != nil {
		return Profile{}, err
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	var out Profile
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		var active int
		err := tx.QueryRowContext(ctx, `SELECT active FROM collection_profiles WHERE id=?`, in.ID).Scan(&active)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			_, err = tx.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,payee_name,parser,enabled,active,created_at,updated_at)
				VALUES(?,?,?,?,?,?,0,?,?)`, in.ID, in.Label, in.UPIID, nullable(in.PayeeName), in.Parser, boolInt(in.Enabled), now.UnixMilli(), now.UnixMilli())
		case err != nil:
			return fmt.Errorf("read collection profile: %w", err)
		case active == 1 && !in.Enabled:
			return ErrCannotDisableActiveProfile
		default:
			_, err = tx.ExecContext(ctx, `UPDATE collection_profiles
				SET label=?,upi_id=?,payee_name=?,parser=?,enabled=?,updated_at=? WHERE id=?`,
				in.Label, in.UPIID, nullable(in.PayeeName), in.Parser, boolInt(in.Enabled), now.UnixMilli(), in.ID)
		}
		if err != nil {
			return fmt.Errorf("save collection profile: %w", err)
		}
		profile, err := getWith(ctx, tx, in.ID)
		if err != nil {
			return err
		}
		out = profile
		return nil
	})
	return out, err
}

func (s *Service) Activate(ctx context.Context, id string) (Profile, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return Profile{}, errors.New("profile storage is required")
	}
	id = strings.ToLower(strings.TrimSpace(id))
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	var out Profile
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		profile, err := getWith(ctx, tx, id)
		if err != nil {
			return err
		}
		if !profile.Enabled {
			return ErrProfileDisabled
		}
		if profile.Active {
			out = profile
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE collection_profiles SET active=0, updated_at=? WHERE active=1`, now.UnixMilli()); err != nil {
			return fmt.Errorf("deactivate collection profile: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE collection_profiles SET active=1, updated_at=? WHERE id=?`, now.UnixMilli(), id); err != nil {
			return fmt.Errorf("activate collection profile: %w", err)
		}
		out, err = getWith(ctx, tx, id)
		return err
	})
	return out, err
}

func (s *Service) Get(ctx context.Context, id string) (Profile, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return Profile{}, errors.New("profile storage is required")
	}
	return getWith(ctx, s.DB.SQL, strings.ToLower(strings.TrimSpace(id)))
}
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return nil, errors.New("profile storage is required")
	}
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT id,label,upi_id,COALESCE(payee_name,''),parser,enabled,active,created_at,updated_at
		FROM collection_profiles ORDER BY active DESC, label COLLATE NOCASE, id`)
	if err != nil {
		return nil, fmt.Errorf("list collection profiles: %w", err)
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collection profiles: %w", err)
	}
	return out, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type scanner interface {
	Scan(...any) error
}

func getWith(ctx context.Context, q queryRower, id string) (Profile, error) {
	row := q.QueryRowContext(ctx, `SELECT id,label,upi_id,COALESCE(payee_name,''),parser,enabled,active,created_at,updated_at
		FROM collection_profiles WHERE id=?`, id)
	profile, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("read collection profile: %w", err)
	}
	return profile, nil
}

func scanProfile(row scanner) (Profile, error) {
	var p Profile
	var enabled, active int
	var created, updated int64
	if err := row.Scan(&p.ID, &p.Label, &p.UPIID, &p.PayeeName, &p.Parser, &enabled, &active, &created, &updated); err != nil {
		return Profile{}, err
	}
	p.Enabled = enabled == 1
	p.Active = active == 1
	p.CreatedAt = time.UnixMilli(created).UTC()
	p.UpdatedAt = time.UnixMilli(updated).UTC()
	return p, nil
}

func validateUpsert(in UpsertInput) error {
	if in.ID == "" || len(in.ID) > 64 || strings.ContainsAny(in.ID, " \t\r\n") {
		return fmt.Errorf("%w: id must contain 1-64 non-space characters", ErrInvalidProfile)
	}
	if in.Label == "" || utf8.RuneCountInString(in.Label) > 120 {
		return fmt.Errorf("%w: label must contain 1-120 characters", ErrInvalidProfile)
	}
	if len(in.UPIID) < 3 || len(in.UPIID) > 255 || !strings.Contains(in.UPIID, "@") {
		return fmt.Errorf("%w: upi_id must be a valid VPA-like identifier", ErrInvalidProfile)
	}
	if utf8.RuneCountInString(in.PayeeName) > 120 {
		return fmt.Errorf("%w: payee_name is too long", ErrInvalidProfile)
	}
	switch in.Parser {
	case "paytm_notification", "kotak_sms":
	default:
		return fmt.Errorf("%w: unsupported parser %q", ErrInvalidProfile, in.Parser)
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
