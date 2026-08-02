package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

const maxNotificationAttempts = 6

type Service struct {
	App core.App
	Now func() time.Time

	WebhookURL    string
	WebhookSecret string
	HTTPClient    *http.Client
	Logger        *slog.Logger

	mu           sync.Mutex
	problemSince map[string]time.Time
	wake         chan struct{}
}

type Input struct {
	Kind      string
	Severity  string
	DedupeKey string
	Message   string
	Details   any
}

func NewService(app core.App) *Service {
	return &Service{
		App: app, Now: time.Now,
		HTTPClient: newDeliveryHTTPClient(), Logger: slog.Default(),
		problemSince: map[string]time.Time{}, wake: make(chan struct{}, 1),
	}
}

func (s *Service) ConfigureWebhook(url, secret string) {
	s.WebhookURL = strings.TrimSpace(url)
	s.WebhookSecret = secret
}

func (s *Service) NotificationsEnabled() bool {
	return s != nil && s.WebhookURL != "" && s.WebhookSecret != ""
}

func (s *Service) Open(input Input) (string, bool, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.Severity = strings.TrimSpace(input.Severity)
	input.DedupeKey = strings.TrimSpace(input.DedupeKey)
	input.Message = strings.TrimSpace(input.Message)
	if input.Kind == "" || input.DedupeKey == "" || input.Message == "" || (input.Severity != "warning" && input.Severity != "critical") {
		return "", false, fmt.Errorf("invalid alert")
	}
	now := s.now()
	var id string
	var created bool
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindFirstRecordByData("alerts", "dedupe_key", input.DedupeKey)
		if err == nil {
			wasResolved := record.GetString("status") == "resolved"
			severityEscalated := record.GetString("severity") == "warning" && input.Severity == "critical"
			record.Set("kind", input.Kind)
			record.Set("status", "open")
			record.Set("severity", input.Severity)
			record.Set("message", truncate(input.Message, 4096))
			record.Set("details", input.Details)
			record.Set("last_seen_at", now)
			record.Set("resolved_at", "")
			record.Set("occurrence_count", record.GetInt("occurrence_count")+1)
			if s.NotificationsEnabled() {
				if wasResolved || severityEscalated || record.GetString("notification_status") == "disabled" {
					s.queueNotification(record, now)
					queued = true
				}
			} else {
				record.Set("notification_status", "disabled")
			}
			if err := tx.Save(record); err != nil {
				return err
			}
			id = record.Id
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		collection, err := tx.FindCollectionByNameOrId("alerts")
		if err != nil {
			return err
		}
		record = core.NewRecord(collection)
		record.Set("kind", input.Kind)
		record.Set("status", "open")
		record.Set("severity", input.Severity)
		record.Set("dedupe_key", input.DedupeKey)
		record.Set("message", truncate(input.Message, 4096))
		record.Set("details", input.Details)
		record.Set("occurrence_count", 1)
		record.Set("first_seen_at", now)
		record.Set("last_seen_at", now)
		if s.NotificationsEnabled() {
			s.queueNotification(record, now)
			queued = true
		} else {
			record.Set("notification_status", "disabled")
		}
		if err := tx.Save(record); err != nil {
			return err
		}
		id = record.Id
		created = true
		return nil
	})
	if queued {
		s.Wake()
	}
	return id, created, err
}

func (s *Service) Resolve(dedupeKey string) error {
	dedupeKey = strings.TrimSpace(dedupeKey)
	if dedupeKey == "" {
		return nil
	}
	now := s.now()
	queued := false
	err := s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindFirstRecordByData("alerts", "dedupe_key", dedupeKey)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if record.GetString("status") == "resolved" {
			return nil
		}
		record.Set("status", "resolved")
		record.Set("resolved_at", now)
		if s.NotificationsEnabled() {
			s.queueNotification(record, now)
			queued = true
		} else {
			record.Set("notification_status", "disabled")
		}
		return tx.Save(record)
	})
	if queued {
		s.Wake()
	}
	return err
}

func (s *Service) OpenCount() (int64, error) {
	return s.App.CountRecords("alerts", dbx.NewExp("status = 'open'"))
}

func (s *Service) Run(ctx context.Context) {
	if !s.NotificationsEnabled() {
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
		if _, err := s.SendPending(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.logger().Error("operator alert delivery pass failed", "error", err)
		}
	}
}

func (s *Service) Wake() {
	if !s.NotificationsEnabled() {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) SendPending(ctx context.Context) (int, error) {
	if !s.NotificationsEnabled() {
		return 0, nil
	}
	now := s.now()
	if err := s.recoverStaleNotifications(now); err != nil {
		return 0, err
	}
	records, err := s.App.FindRecordsByFilter(
		"alerts",
		"(notification_status = 'pending' || notification_status = 'failed') && notification_next_attempt_at <= {:now}",
		"notification_next_attempt_at,created", 50, 0,
		dbx.Params{"now": filterDate(now)},
	)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		claimed, err := s.claimNotification(record.Id, now)
		if err != nil {
			s.logger().Warn("failed to claim operator alert delivery", "alertId", record.Id, "error", err)
			continue
		}
		if claimed == nil {
			continue
		}
		s.deliverNotification(ctx, claimed)
		processed++
	}
	return processed, nil
}

func (s *Service) queueNotification(record *core.Record, now time.Time) {
	record.Set("notification_event_id", "alert_"+security.RandomString(24))
	record.Set("notification_status", "pending")
	record.Set("notification_created_at", now)
	record.Set("notification_attempts", 0)
	record.Set("notification_next_attempt_at", now)
	record.Set("notification_locked_at", "")
	record.Set("notification_last_attempt_at", "")
	record.Set("notification_delivered_at", "")
	record.Set("notification_last_error", "")
}

func (s *Service) claimNotification(id string, now time.Time) (*core.Record, error) {
	var claimed *core.Record
	err := s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById("alerts", id)
		if err != nil {
			return err
		}
		status := record.GetString("notification_status")
		if status != "pending" && status != "failed" {
			return nil
		}
		if next := record.GetDateTime("notification_next_attempt_at").Time(); !next.IsZero() && next.After(now) {
			return nil
		}
		record.Set("notification_status", "sending")
		record.Set("notification_locked_at", now)
		record.Set("notification_last_attempt_at", now)
		record.Set("notification_attempts", record.GetInt("notification_attempts")+1)
		if err := tx.Save(record); err != nil {
			return err
		}
		claimed = record.Clone()
		return nil
	})
	return claimed, err
}

func (s *Service) deliverNotification(ctx context.Context, record *core.Record) {
	eventID := record.GetString("notification_event_id")
	eventType := "operational.alert.opened"
	if record.GetString("status") == "resolved" {
		eventType = "operational.alert.resolved"
	}
	body, marshalErr := json.Marshal(map[string]any{
		"id": eventID, "type": eventType, "createdAt": record.GetDateTime("notification_created_at").Time().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{"alert": map[string]any{
			"id": record.Id, "kind": record.GetString("kind"), "status": record.GetString("status"),
			"severity": record.GetString("severity"), "message": record.GetString("message"),
			"details": record.Get("details"), "occurrenceCount": record.GetInt("occurrence_count"),
			"firstSeenAt": record.GetDateTime("first_seen_at").String(), "lastSeenAt": record.GetDateTime("last_seen_at").String(),
		}},
	})
	statusCode := 0
	var deliveryErr error = marshalErr
	if deliveryErr == nil {
		timestamp := strconv.FormatInt(s.now().Unix(), 10)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, strings.NewReader(string(body)))
		if err != nil {
			deliveryErr = err
		} else {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "PayGate/1.0")
			req.Header.Set("X-PayGate-Event-Id", eventID)
			req.Header.Set("X-PayGate-Timestamp", timestamp)
			req.Header.Set("X-PayGate-Signature", "v1="+webhooks.Sign(s.WebhookSecret, timestamp, body))
			response, err := s.client().Do(req)
			if response != nil {
				statusCode = response.StatusCode
				_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
				_ = response.Body.Close()
			}
			if err != nil {
				deliveryErr = err
			} else if statusCode < 200 || statusCode >= 300 {
				deliveryErr = fmt.Errorf("operator alert webhook returned HTTP %d", statusCode)
			}
		}
	}
	if err := s.finishNotification(record.Id, eventID, statusCode, deliveryErr); err != nil {
		s.logger().Error("failed to persist operator alert delivery result", "alertId", record.Id, "error", err)
	}
}

func (s *Service) finishNotification(id, eventID string, statusCode int, deliveryErr error) error {
	now := s.now()
	return s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById("alerts", id)
		if err != nil {
			return err
		}
		if record.GetString("notification_event_id") != eventID || record.GetString("notification_status") != "sending" {
			return nil // a newer open/resolved notification superseded this attempt
		}
		record.Set("notification_locked_at", "")
		if deliveryErr == nil {
			record.Set("notification_status", "delivered")
			record.Set("notification_delivered_at", now)
			record.Set("notification_last_error", "")
			return tx.Save(record)
		}
		attempts := record.GetInt("notification_attempts")
		record.Set("notification_last_error", truncate(deliveryErr.Error(), 4096))
		if attempts >= maxNotificationAttempts {
			record.Set("notification_status", "exhausted")
			record.Set("notification_next_attempt_at", now.Add(365*24*time.Hour))
		} else {
			record.Set("notification_status", "failed")
			record.Set("notification_next_attempt_at", now.Add(notificationRetryDelay(attempts)))
		}
		return tx.Save(record)
	})
}

func (s *Service) recoverStaleNotifications(now time.Time) error {
	records, err := s.App.FindRecordsByFilter(
		"alerts", "notification_status = 'sending' && notification_locked_at < {:stale}",
		"notification_locked_at", 50, 0, dbx.Params{"stale": filterDate(now.Add(-2 * time.Minute))},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set("notification_status", "failed")
		record.Set("notification_locked_at", "")
		record.Set("notification_next_attempt_at", now)
		record.Set("notification_last_error", "recovered stale operator alert delivery lease after restart")
		if err := s.App.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func notificationRetryDelay(attempt int) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 6 * time.Hour}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[index]
}

func (s *Service) CheckConnector(status gmessages.Status) error {
	if !status.Enabled {
		return s.resolveProblems("connector:reauth", "connector:disconnected", "connector:unresponsive")
	}
	if status.State == "reauth_required" {
		if _, _, err := s.Open(Input{Kind: "connector_reauth", Severity: "critical", DedupeKey: "connector:reauth", Message: "Google Messages requires browser reauthentication", Details: map[string]any{"state": status.State, "lastError": status.LastError}}); err != nil {
			return err
		}
	} else if err := s.Resolve("connector:reauth"); err != nil {
		return err
	}
	if status.Connected {
		s.clearProblem("connector:disconnected")
		if err := s.Resolve("connector:disconnected"); err != nil {
			return err
		}
	} else if status.Paired && s.problemMature("connector:disconnected", 5*time.Minute) {
		if _, _, err := s.Open(Input{Kind: "connector_disconnected", Severity: "critical", DedupeKey: "connector:disconnected", Message: "Google Messages has remained disconnected for more than five minutes", Details: map[string]any{"state": status.State, "lastError": status.LastError}}); err != nil {
			return err
		}
	}
	if status.PhoneResponsive || !status.Paired {
		s.clearProblem("connector:unresponsive")
		if err := s.Resolve("connector:unresponsive"); err != nil {
			return err
		}
	} else if s.problemMature("connector:unresponsive", 15*time.Minute) {
		if _, _, err := s.Open(Input{Kind: "connector_unresponsive", Severity: "warning", DedupeKey: "connector:unresponsive", Message: "Paired phone has not been responsive for more than fifteen minutes", Details: map[string]any{"state": status.State}}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CheckCapacity(snapshot payments.CapacitySnapshot) error {
	active := map[string]struct{}{}
	for _, pool := range snapshot.Pools {
		if pool.Level == "normal" {
			continue
		}
		key := fmt.Sprintf("capacity:%d", pool.RequestedAmountPaise)
		active[key] = struct{}{}
		severity := "warning"
		if pool.Level == "critical" {
			severity = "critical"
		}
		message := fmt.Sprintf("₹%s fingerprint pool is %.0f%% utilized (%d of 99 blocked)", pool.RequestedAmount, pool.UtilizationPercent, pool.Blocked)
		if _, _, err := s.Open(Input{Kind: "capacity_high", Severity: severity, DedupeKey: key, Message: message, Details: pool}); err != nil {
			return err
		}
	}
	records, err := s.App.FindRecordsByFilter("alerts", "kind = 'capacity_high' && status = 'open'", "created", 0, 0)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := active[record.GetString("dedupe_key")]; !ok {
			if err := s.Resolve(record.GetString("dedupe_key")); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) CheckWebhookExhaustion() error {
	records, err := s.App.FindRecordsByFilter("webhook_deliveries", "status = 'exhausted'", "-updated", 100, 0)
	if err != nil {
		return err
	}
	active := map[string]struct{}{}
	for _, record := range records {
		key := "webhook:" + record.Id
		active[key] = struct{}{}
		if _, _, err := s.Open(Input{Kind: "webhook_exhausted", Severity: "critical", DedupeKey: key, Message: "Outgoing webhook exhausted its retry limit", Details: map[string]any{"deliveryId": record.Id, "eventId": record.GetString("event_id"), "event": record.GetString("event"), "lastError": record.GetString("last_error")}}); err != nil {
			return err
		}
	}
	open, err := s.App.FindRecordsByFilter("alerts", "kind = 'webhook_exhausted' && status = 'open'", "created", 0, 0)
	if err != nil {
		return err
	}
	for _, record := range open {
		if _, ok := active[record.GetString("dedupe_key")]; !ok {
			if err := s.Resolve(record.GetString("dedupe_key")); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) problemMature(key string, delay time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	started, ok := s.problemSince[key]
	if !ok {
		s.problemSince[key] = now
		return delay <= 0
	}
	return now.Sub(started) >= delay
}

func (s *Service) clearProblem(key string) { s.mu.Lock(); delete(s.problemSince, key); s.mu.Unlock() }
func (s *Service) resolveProblems(keys ...string) error {
	for _, key := range keys {
		s.clearProblem(key)
		if err := s.Resolve(key); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
func newDeliveryHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (s *Service) client() *http.Client {
	if s.HTTPClient == nil {
		return newDeliveryHTTPClient()
	}
	return s.HTTPClient
}
func (s *Service) logger() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}
