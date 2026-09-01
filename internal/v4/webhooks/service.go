package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

const (
	defaultMaxAttempts = 8
	defaultBatchSize   = 50
	defaultLease       = 2 * time.Minute
)

var ErrInvalidConfig = errors.New("invalid webhook configuration")

type Config struct {
	Endpoint          string
	Secret            string
	AllowInsecureHTTP bool
}
type Service struct {
	DB          *storage.DB
	Config      Config
	HTTPClient  *http.Client
	Now         func() time.Time
	MaxAttempts int
	BatchSize   int
	Lease       time.Duration

	wake chan struct{}
	mu   sync.Mutex
}

type Delivery struct {
	ID       string
	Body     string
	Attempts int
}

func NewService(db *storage.DB, cfg Config) *Service {
	return &Service{
		DB: db, Config: cfg, HTTPClient: newHTTPClient(), Now: time.Now,
		MaxAttempts: defaultMaxAttempts, BatchSize: defaultBatchSize,
		Lease: defaultLease, wake: make(chan struct{}, 1),
	}
}

func (s *Service) Enabled() bool {
	return s != nil && strings.TrimSpace(s.Config.Endpoint) != "" && strings.TrimSpace(s.Config.Secret) != ""
}
func (s *Service) Wake() {
	if !s.Enabled() {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) Run(ctx context.Context) {
	if !s.Enabled() {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	s.Wake()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.wake:
		}
		_, _ = s.SendPending(ctx)
	}
}

func (s *Service) SendPending(ctx context.Context) (int, error) {
	if !s.Enabled() {
		return 0, nil
	}
	if err := s.ready(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	limit := s.BatchSize
	if limit <= 0 || limit > 500 {
		limit = defaultBatchSize
	}
	rows, err := s.DB.SQL.QueryContext(ctx, `SELECT id FROM webhook_deliveries
		WHERE status IN ('pending','retry') AND COALESCE(next_attempt_at,0) <= ?
		ORDER BY COALESCE(next_attempt_at,0), created_at, rowid LIMIT ?`, now.UnixMilli(), limit)
	if err != nil {
		return 0, fmt.Errorf("list due webhooks: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan due webhook: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate due webhooks: %w", err)
	}

	processed := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		delivery, err := s.claim(ctx, id, now)
		if err != nil {
			return processed, err
		}
		if delivery == nil {
			continue
		}
		s.deliver(ctx, *delivery)
		processed++
	}
	return processed, nil
}
func (s *Service) claim(ctx context.Context, id string, now time.Time) (*Delivery, error) {
	lease := s.Lease
	if lease <= 0 {
		lease = defaultLease
	}
	var out *Delivery
	err := s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		result, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries
			SET attempts=attempts+1,next_attempt_at=?
			WHERE id=? AND status IN ('pending','retry') AND COALESCE(next_attempt_at,0) <= ?`,
			now.Add(lease).UnixMilli(), id, now.UnixMilli())
		if err != nil {
			return fmt.Errorf("claim webhook %s: %w", id, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return nil
		}
		var delivery Delivery
		if err := tx.QueryRowContext(ctx, `SELECT id,payload_json,attempts FROM webhook_deliveries WHERE id=?`, id).
			Scan(&delivery.ID, &delivery.Body, &delivery.Attempts); err != nil {
			return fmt.Errorf("load claimed webhook %s: %w", id, err)
		}
		out = &delivery
		return nil
	})
	return out, err
}

func (s *Service) deliver(ctx context.Context, delivery Delivery) {
	now := s.now()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	signature := Sign(s.Config.Secret, timestamp, []byte(delivery.Body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Config.Endpoint, strings.NewReader(delivery.Body))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "PayGate/4")
		req.Header.Set("PayGate-Event-Id", delivery.ID)
		req.Header.Set("PayGate-Timestamp", timestamp)
		req.Header.Set("PayGate-Signature", "v1="+signature)
	}
	statusCode := 0
	retryable := true
	if err == nil {
		var response *http.Response
		response, err = s.client().Do(req)
		if response != nil {
			statusCode = response.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
			_ = response.Body.Close()
			if statusCode < 200 || statusCode >= 300 {
				err = fmt.Errorf("webhook returned HTTP %d", statusCode)
				retryable = retryableHTTPStatus(statusCode)
			}
		}
	}
	_ = s.finish(context.Background(), delivery, statusCode, retryable, err)
}

func (s *Service) finish(ctx context.Context, delivery Delivery, statusCode int, retryable bool, deliveryErr error) error {
	now := s.now()
	maxAttempts := s.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if deliveryErr == nil {
			_, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries SET status='delivered',next_attempt_at=NULL,last_http_status=?,last_error=NULL,delivered_at=? WHERE id=?`,
				nullableStatus(statusCode), now.UnixMilli(), delivery.ID)
			return err
		}

		message := truncate(deliveryErr.Error(), 4000)
		if !retryable || delivery.Attempts >= maxAttempts {
			_, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries SET status='exhausted',next_attempt_at=NULL,last_http_status=?,last_error=?,delivered_at=NULL WHERE id=?`,
				nullableStatus(statusCode), message, delivery.ID)
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries SET status='retry',next_attempt_at=?,last_http_status=?,last_error=?,delivered_at=NULL WHERE id=?`,
			now.Add(retryDelay(delivery.Attempts)).UnixMilli(), nullableStatus(statusCode), message, delivery.ID)
		return err
	})
}
func (s *Service) RetryOne(ctx context.Context, id string) error {
	if err := s.ready(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("webhook id is required")
	}
	result, err := s.DB.SQL.ExecContext(ctx, `UPDATE webhook_deliveries
		SET status='pending',attempts=0,next_attempt_at=?,last_http_status=NULL,last_error=NULL,delivered_at=NULL
		WHERE id=? AND status IN ('retry','exhausted')`, s.now().UnixMilli(), id)
	if err != nil {
		return fmt.Errorf("retry webhook: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return errors.New("webhook is not retryable")
	}
	s.Wake()
	return nil
}

func (s *Service) ready() error {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return errors.New("webhook storage is required")
	}
	endpoint := strings.TrimSpace(s.Config.Endpoint)
	secret := strings.TrimSpace(s.Config.Secret)
	if endpoint == "" || len(secret) < 32 {
		return fmt.Errorf("%w: endpoint and at least 32-byte secret are required", ErrInvalidConfig)
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%w: endpoint must be an absolute URL without credentials/query/fragment", ErrInvalidConfig)
	}
	if u.Scheme != "https" && !(s.Config.AllowInsecureHTTP && u.Scheme == "http") {
		return fmt.Errorf("%w: HTTPS endpoint is required", ErrInvalidConfig)
	}
	return nil
}

func Sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
func retryableHTTPStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests || status >= 500
}

func retryDelay(attempt int) time.Duration {
	delays := []time.Duration{
		time.Minute,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		6 * time.Hour,
		12 * time.Hour,
		24 * time.Hour,
	}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[index]
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *Service) client() *http.Client {
	if s.HTTPClient == nil {
		return newHTTPClient()
	}
	return s.HTTPClient
}
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func nullableStatus(status int) any {
	if status == 0 {
		return nil
	}
	return status
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
