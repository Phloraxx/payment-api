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

	"github.com/Phloraxx/payment-api/internal/deliveryqueue"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
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

func (s *Service) notificationQueue() deliveryqueue.Queue {
	return deliveryqueue.Queue{
		App: s.App, Collection: "alerts", MaxAttempts: maxNotificationAttempts,
		Fields: deliveryqueue.Fields{
			Status: "notification_status", Attempts: "notification_attempts",
			NextAttemptAt: "notification_next_attempt_at", LockedAt: "notification_locked_at",
			LastAttemptAt: "notification_last_attempt_at", DeliveredAt: "notification_delivered_at",
			LastError: "notification_last_error",
		},
		RetryDelays: []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 6 * time.Hour},
		StaleAfter:  2 * time.Minute, ExhaustedAfter: 365 * 24 * time.Hour, ErrorMax: 4096,
		StaleMessage: "recovered stale operator alert delivery lease after restart",
	}
}

func (s *Service) Open(input Input) (string, bool, error) {
	return s.upsert(input, true)
}

// EnsureOpen records a persistent operational condition without counting every
// periodic health scan as a new occurrence. A resolved condition increments
// once when it becomes active again; an already-open condition only refreshes
// its details/last-seen state.
func (s *Service) EnsureOpen(input Input) (string, bool, error) {
	return s.upsert(input, false)
}

func (s *Service) upsert(input Input, countRepeated bool) (string, bool, error) {
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
			count := record.GetInt("occurrence_count")
			if countRepeated || wasResolved {
				count++
			}
			record.Set("occurrence_count", count)
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
	queue := s.notificationQueue()
	if err := queue.RecoverStale(now, 50); err != nil {
		return 0, err
	}
	records, err := queue.Due(now, 50)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		claimed, err := queue.Claim(record.Id, now)
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
	return s.notificationQueue().Finish(id, s.now(), statusCode, deliveryErr, func(record *core.Record) bool {
		return record.GetString("notification_event_id") == eventID &&
			record.GetString("notification_status") == "sending"
	})
}

func (s *Service) CheckConnector(status gmessages.Status) error {
	if !status.Enabled {
		return s.resolveProblems("connector:reauth", "connector:disconnected", "connector:unresponsive")
	}
	if status.State == "reauth_required" {
		if _, _, err := s.EnsureOpen(Input{Kind: "connector_reauth", Severity: "critical", DedupeKey: "connector:reauth", Message: "Google Messages requires browser reauthentication", Details: map[string]any{"state": status.State, "lastError": status.LastError}}); err != nil {
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
		if _, _, err := s.EnsureOpen(Input{Kind: "connector_disconnected", Severity: "critical", DedupeKey: "connector:disconnected", Message: "Google Messages has remained disconnected for more than five minutes", Details: map[string]any{"state": status.State, "lastError": status.LastError}}); err != nil {
			return err
		}
	}
	if status.PhoneResponsive || !status.Paired {
		s.clearProblem("connector:unresponsive")
		if err := s.Resolve("connector:unresponsive"); err != nil {
			return err
		}
	} else if s.problemMature("connector:unresponsive", 15*time.Minute) {
		if _, _, err := s.EnsureOpen(Input{Kind: "connector_unresponsive", Severity: "warning", DedupeKey: "connector:unresponsive", Message: "Paired phone has not been responsive for more than fifteen minutes", Details: map[string]any{"state": status.State}}); err != nil {
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
		if _, _, err := s.EnsureOpen(Input{Kind: "capacity_high", Severity: severity, DedupeKey: key, Message: message, Details: pool}); err != nil {
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
	count, err := s.App.CountRecords("webhook_deliveries", dbx.NewExp("status = 'exhausted'"))
	if err != nil {
		return err
	}
	if err := s.resolveLegacyWebhookAlerts(); err != nil {
		return err
	}
	const dedupeKey = "webhook:exhausted"
	if count == 0 {
		return s.Resolve(dedupeKey)
	}

	exhausted, err := s.App.FindRecordsByFilter("webhook_deliveries", "status = 'exhausted'", "-updated", 1, 0)
	if err != nil {
		return err
	}
	details := map[string]any{"exhaustedCount": count}
	severity := "critical"
	message := fmt.Sprintf("%d outgoing webhook deliveries have exhausted their retry limit", count)
	if len(exhausted) == 1 {
		latest := exhausted[0]
		details["latestExhaustedAt"] = latest.GetDateTime("updated").String()
		details["latestEvent"] = latest.GetString("event")
		latestExhaustedAt := latest.GetDateTime("updated").Time()
		delivered, deliveredErr := s.App.FindRecordsByFilter("webhook_deliveries", "status = 'delivered' && delivered_at != ''", "-delivered_at", 1, 0)
		if deliveredErr != nil {
			return deliveredErr
		}
		if len(delivered) == 1 {
			latestDeliveredAt := delivered[0].GetDateTime("delivered_at").Time()
			details["latestDeliveredAt"] = delivered[0].GetDateTime("delivered_at").String()
			if !latestDeliveredAt.IsZero() && latestDeliveredAt.After(latestExhaustedAt) {
				details["transportRecovered"] = true
				severity = "warning"
				message = fmt.Sprintf("%d historical outgoing webhook deliveries remain exhausted; newer webhook deliveries have succeeded", count)
			}
		}
	}
	_, _, err = s.EnsureOpen(Input{
		Kind: "webhook_exhausted", Severity: severity, DedupeKey: dedupeKey,
		Message: message, Details: details,
	})
	return err
}

// resolveLegacyWebhookAlerts silently closes the per-delivery alerts generated
// by the pre-v2 scanner. They represent the same aggregate condition and must
// not emit hundreds of "resolved" notifications during remediation.
func (s *Service) resolveLegacyWebhookAlerts() error {
	records, err := s.App.FindRecordsByFilter(
		"alerts",
		"kind = 'webhook_exhausted' && dedupe_key != 'webhook:exhausted' && status = 'open'",
		"created", 0, 0,
	)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	now := s.now()
	return s.App.RunInTransaction(func(tx core.App) error {
		for _, item := range records {
			record, err := tx.FindRecordById("alerts", item.Id)
			if err != nil {
				return err
			}
			record.Set("status", "resolved")
			record.Set("resolved_at", now)
			record.Set("notification_status", "disabled")
			if err := tx.Save(record); err != nil {
				return err
			}
		}
		return nil
	})
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
