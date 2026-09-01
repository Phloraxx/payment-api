package operator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

type Service struct {
	DB       *storage.DB
	Now      func() time.Time
	Location *time.Location
}

type DailyVolume struct {
	Date        string `json:"date"`
	AmountPaise int64  `json:"amount_paise"`
	Payments    int    `json:"payments"`
}

type Overview struct {
	CollectedTodayPaise int64           `json:"collected_today_paise"`
	PaymentsToday       int             `json:"payments_today"`
	PaidToday           int             `json:"paid_today"`
	Pending             int             `json:"pending"`
	ExpiredToday        int             `json:"expired_today"`
	StatusCounts        map[string]int  `json:"status_counts"`
	Volume              []DailyVolume   `json:"volume"`
	ActiveProfile       *ProfileSummary `json:"active_profile"`
	Relay               RelaySummary    `json:"relay"`
	Webhooks            WebhookSummary  `json:"webhooks"`
}
type ProfileSummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RelaySummary struct {
	Connected  bool       `json:"connected"`
	Name       string     `json:"name,omitempty"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	AppVersion string     `json:"app_version,omitempty"`
}

type WebhookSummary struct {
	Pending         int        `json:"pending"`
	Exhausted       int        `json:"exhausted"`
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
}

type ActivityEntry struct {
	At          time.Time `json:"at"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status,omitempty"`
	Source      string    `json:"source,omitempty"`
	Title       string    `json:"title"`
	PaymentID   string    `json:"payment_id,omitempty"`
	AmountPaise *int64    `json:"amount_paise,omitempty"`
	Detail      string    `json:"detail,omitempty"`
}

func NewService(db *storage.DB) *Service {
	return &Service{DB: db, Now: time.Now, Location: time.FixedZone("IST", 5*60*60+30*60)}
}

func (s *Service) ready() error {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return errors.New("operator storage is required")
	}
	return nil
}
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	if err := s.ready(); err != nil {
		return Overview{}, err
	}
	now := s.now()
	start, end := localDayBounds(now, s.location())
	out := Overview{StatusCounts: map[string]int{}}

	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COALESCE(SUM(payable_amount_paise),0),COUNT(*) FROM payments
		WHERE status='paid' AND paid_at>=? AND paid_at<?`, start.UnixMilli(), end.UnixMilli()).Scan(&out.CollectedTodayPaise, &out.PaidToday); err != nil {
		return Overview{}, fmt.Errorf("read paid-today overview: %w", err)
	}
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE created_at>=? AND created_at<?`, start.UnixMilli(), end.UnixMilli()).Scan(&out.PaymentsToday); err != nil {
		return Overview{}, fmt.Errorf("read payments-today overview: %w", err)
	}
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE status='pending'`).Scan(&out.Pending); err != nil {
		return Overview{}, fmt.Errorf("read pending overview: %w", err)
	}
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE status='expired' AND grace_until>=? AND grace_until<?`, start.UnixMilli(), end.UnixMilli()).Scan(&out.ExpiredToday); err != nil {
		return Overview{}, fmt.Errorf("read expired-today overview: %w", err)
	}
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT status,COUNT(*) FROM payments GROUP BY status`)
	if err != nil {
		return Overview{}, fmt.Errorf("read status counts: %w", err)
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return Overview{}, fmt.Errorf("scan status count: %w", err)
		}
		out.StatusCounts[status] = count
	}
	if err := rows.Close(); err != nil {
		return Overview{}, err
	}
	if err := rows.Err(); err != nil {
		return Overview{}, fmt.Errorf("iterate status counts: %w", err)
	}

	volume, err := s.volumeTrend(ctx, now, 7)
	if err != nil {
		return Overview{}, err
	}
	out.Volume = volume
	profile, err := s.activeProfile(ctx)
	if err != nil {
		return Overview{}, err
	}
	out.ActiveProfile = profile
	if err := s.loadRelay(ctx, &out.Relay); err != nil {
		return Overview{}, err
	}
	if err := s.loadWebhookSummary(ctx, &out.Webhooks); err != nil {
		return Overview{}, err
	}
	return out, nil
}
func (s *Service) volumeTrend(ctx context.Context, now time.Time, days int) ([]DailyVolume, error) {
	loc := s.location()
	if days <= 0 {
		days = 7
	}
	startDay := now.In(loc).AddDate(0, 0, -(days - 1))
	start, _ := localDayBounds(startDay, loc)
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT paid_at,payable_amount_paise FROM payments WHERE status='paid' AND paid_at>=? ORDER BY paid_at`, start.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("read volume trend: %w", err)
	}
	defer rows.Close()
	byDate := make(map[string]*DailyVolume, days)
	for i := 0; i < days; i++ {
		date := startDay.AddDate(0, 0, i).Format("2006-01-02")
		byDate[date] = &DailyVolume{Date: date}
	}
	for rows.Next() {
		var paidAt, amount int64
		if err := rows.Scan(&paidAt, &amount); err != nil {
			return nil, fmt.Errorf("scan volume trend: %w", err)
		}
		date := time.UnixMilli(paidAt).In(loc).Format("2006-01-02")
		if bucket := byDate[date]; bucket != nil {
			bucket.AmountPaise += amount
			bucket.Payments++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate volume trend: %w", err)
	}
	out := make([]DailyVolume, 0, days)
	for i := 0; i < days; i++ {
		date := startDay.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, *byDate[date])
	}
	return out, nil
}
func (s *Service) activeProfile(ctx context.Context) (*ProfileSummary, error) {
	var profile ProfileSummary
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT id,label FROM collection_profiles WHERE active=1 AND enabled=1`).Scan(&profile.ID, &profile.Label)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read active profile: %w", err)
	}
	return &profile, nil
}

func (s *Service) loadRelay(ctx context.Context, out *RelaySummary) error {
	var name string
	var lastSeen sql.NullInt64
	var appVersion sql.NullString
	err := s.DB.SQL.QueryRowContext(ctx, `SELECT COALESCE(name,''),COALESCE(last_heartbeat_at,last_seen_at),app_version FROM relay_devices WHERE enabled=1 LIMIT 1`).
		Scan(&name, &lastSeen, &appVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read relay summary: %w", err)
	}
	out.Name = name
	out.AppVersion = appVersion.String
	if lastSeen.Valid {
		value := time.UnixMilli(lastSeen.Int64).UTC()
		out.LastSeenAt = &value
		age := s.now().Sub(value)
		out.Connected = age >= -5*time.Minute && age <= time.Hour
	}
	return nil
}

func (s *Service) loadWebhookSummary(ctx context.Context, out *WebhookSummary) error {
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE status IN ('pending','retry')`).Scan(&out.Pending); err != nil {
		return fmt.Errorf("read pending webhook count: %w", err)
	}
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE status='exhausted'`).Scan(&out.Exhausted); err != nil {
		return fmt.Errorf("read exhausted webhook count: %w", err)
	}
	var delivered sql.NullInt64
	if err := s.DB.SQL.QueryRowContext(ctx, `SELECT MAX(delivered_at) FROM webhook_deliveries WHERE status='delivered'`).Scan(&delivered); err != nil {
		return fmt.Errorf("read last webhook delivery: %w", err)
	}
	if delivered.Valid {
		value := time.UnixMilli(delivered.Int64).UTC()
		out.LastDeliveredAt = &value
	}
	return nil
}
func (s *Service) Activity(ctx context.Context, limit int) ([]ActivityEntry, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	entries := make([]ActivityEntry, 0, limit*2)
	if err := s.appendPaymentHistory(ctx, &entries, limit); err != nil {
		return nil, err
	}
	if err := s.appendObservations(ctx, &entries, limit); err != nil {
		return nil, err
	}
	if err := s.appendWebhooks(ctx, &entries, limit); err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].At.Equal(entries[j].At) {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].At.After(entries[j].At)
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (s *Service) appendPaymentHistory(ctx context.Context, out *[]ActivityEntry, limit int) error {
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT created_at,type,actor,summary,payment_id FROM payment_history ORDER BY created_at DESC,rowid DESC LIMIT ?`, limit)
	if err != nil {
		return fmt.Errorf("read payment activity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var at int64
		var entry ActivityEntry
		if err := rows.Scan(&at, &entry.Status, &entry.Source, &entry.Title, &entry.PaymentID); err != nil {
			return fmt.Errorf("scan payment activity: %w", err)
		}
		entry.At = time.UnixMilli(at).UTC()
		entry.Kind = "payment"
		*out = append(*out, entry)
	}
	return rows.Err()
}
func (s *Service) appendObservations(ctx context.Context, out *[]ActivityEntry, limit int) error {
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT received_at,match_result,source,COALESCE(matched_payment_id,''),amount_paise,COALESCE(payer_name,'')
		FROM payment_observations ORDER BY received_at DESC,rowid DESC LIMIT ?`, limit)
	if err != nil {
		return fmt.Errorf("read observation activity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var at, amount int64
		var entry ActivityEntry
		var payer string
		if err := rows.Scan(&at, &entry.Status, &entry.Source, &entry.PaymentID, &amount, &payer); err != nil {
			return fmt.Errorf("scan observation activity: %w", err)
		}
		entry.At = time.UnixMilli(at).UTC()
		entry.Kind = "payment_detected"
		entry.Title = "Incoming payment detected"
		entry.AmountPaise = &amount
		entry.Detail = payer
		*out = append(*out, entry)
	}
	return rows.Err()
}

func (s *Service) appendWebhooks(ctx context.Context, out *[]ActivityEntry, limit int) error {
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT COALESCE(delivered_at,created_at),status,event_type,payment_id,last_http_status,COALESCE(last_error,'')
		FROM webhook_deliveries ORDER BY COALESCE(delivered_at,created_at) DESC,rowid DESC LIMIT ?`, limit)
	if err != nil {
		return fmt.Errorf("read webhook activity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var at int64
		var entry ActivityEntry
		var httpStatus sql.NullInt64
		if err := rows.Scan(&at, &entry.Status, &entry.Source, &entry.PaymentID, &httpStatus, &entry.Detail); err != nil {
			return fmt.Errorf("scan webhook activity: %w", err)
		}
		entry.At = time.UnixMilli(at).UTC()
		entry.Kind = "webhook"
		entry.Title = entry.Source
		if httpStatus.Valid {
			if entry.Detail != "" {
				entry.Detail = fmt.Sprintf("HTTP %d · %s", httpStatus.Int64, entry.Detail)
			} else {
				entry.Detail = fmt.Sprintf("HTTP %d", httpStatus.Int64)
			}
		}
		*out = append(*out, entry)
	}
	return rows.Err()
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *Service) location() *time.Location {
	if s.Location == nil {
		return time.FixedZone("IST", 5*60*60+30*60)
	}
	return s.Location
}

func localDayBounds(now time.Time, loc *time.Location) (time.Time, time.Time) {
	local := now.In(loc)
	startLocal := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return startLocal.UTC(), startLocal.AddDate(0, 0, 1).UTC()
}
