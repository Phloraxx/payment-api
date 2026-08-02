package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

const maxAttempts = 8

type Service struct {
	App        core.App
	Config     config.Config
	HTTPClient *http.Client
	Logger     *slog.Logger
	Now        func() time.Time

	wake chan struct{}
}

func NewService(app core.App, cfg config.Config) *Service {
	return &Service{
		App:        app,
		Config:     cfg,
		HTTPClient: newDeliveryHTTPClient(),
		Logger:     slog.Default(),
		Now:        time.Now,
		wake:       make(chan struct{}, 1),
	}
}

func (s *Service) Enabled() bool {
	return s != nil && strings.TrimSpace(s.Config.OutgoingWebhookURL) != "" && s.Config.OutgoingWebhookSecret != ""
}

func (s *Service) Schedule(app core.App, event string, payment *core.Record, at time.Time) error {
	if payment == nil {
		return errors.New("payment is required for webhook scheduling")
	}
	return s.schedule(app, event, payment, nil, at)
}

func (s *Service) ScheduleRefund(app core.App, event string, payment, refund *core.Record, at time.Time) error {
	if payment == nil || refund == nil {
		return errors.New("payment and refund are required for refund webhook scheduling")
	}
	return s.schedule(app, event, payment, refund, at)
}

func (s *Service) schedule(app core.App, event string, payment, refund *core.Record, at time.Time) error {
	if !s.Enabled() {
		return nil
	}
	collection, err := app.FindCollectionByNameOrId("webhook_deliveries")
	if err != nil {
		return err
	}
	eventID := "evt_" + security.RandomString(24)
	data := map[string]any{
		"payment": map[string]any{
			"id":                   payment.Id,
			"requestedAmountPaise": payment.GetInt("requested_amount"),
			"payableAmountPaise":   payment.GetInt("payable_amount"),
			"status":               payment.GetString("status"),
			"rrn":                  payment.GetString("rrn"),
			"upiId":                payment.GetString("upi_id"),
			"payerName":            payment.GetString("payer_name"),
			"paidAt":               payment.GetDateTime("paid_at").String(),
			"externalId":           payment.GetString("external_id"),
		},
	}
	if refund != nil {
		data["refund"] = map[string]any{
			"id":          refund.Id,
			"amountPaise": refund.GetInt("amount"),
			"status":      refund.GetString("status"),
			"reason":      refund.GetString("reason"),
			"reference":   refund.GetString("reference"),
			"externalId":  refund.GetString("external_id"),
			"requestedAt": refund.GetDateTime("requested_at").String(),
			"completedAt": refund.GetDateTime("completed_at").String(),
		}
	}
	body, err := json.Marshal(map[string]any{
		"id": eventID, "type": event,
		"createdAt": at.UTC().Format(time.RFC3339Nano),
		"data":      data,
	})
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	record := core.NewRecord(collection)
	record.Set("event_id", eventID)
	record.Set("event", event)
	record.Set("payment", payment.Id)
	if refund != nil {
		record.Set("refund", refund.Id)
	}
	record.Set("url", s.Config.OutgoingWebhookURL)
	record.Set("body", string(body))
	record.Set("attempts", 0)
	record.Set("status", "pending")
	record.Set("next_attempt_at", at.UTC())
	return app.Save(record)
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
		if _, err := s.SendPending(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.logger().Error("webhook delivery pass failed", "error", err)
		}
	}
}

func (s *Service) SendPending(ctx context.Context) (int, error) {
	if !s.Enabled() {
		return 0, nil
	}
	now := s.now()
	if err := s.recoverStale(now); err != nil {
		return 0, err
	}
	records, err := s.App.FindRecordsByFilter(
		"webhook_deliveries",
		"(status = 'pending' || status = 'failed') && next_attempt_at <= {:now}",
		"next_attempt_at,created",
		50,
		0,
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
		claimed, err := s.claim(record.Id, now)
		if err != nil {
			s.logger().Warn("failed to claim webhook delivery", "id", record.Id, "error", err)
			continue
		}
		if claimed == nil {
			continue
		}
		s.deliver(ctx, claimed)
		processed++
	}
	return processed, nil
}

func (s *Service) claim(id string, now time.Time) (*core.Record, error) {
	var claimed *core.Record
	err := s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById("webhook_deliveries", id)
		if err != nil {
			return err
		}
		status := record.GetString("status")
		if status != "pending" && status != "failed" {
			return nil
		}
		if next := record.GetDateTime("next_attempt_at").Time(); !next.IsZero() && next.After(now) {
			return nil
		}
		record.Set("status", "sending")
		record.Set("locked_at", now)
		record.Set("last_attempt_at", now)
		record.Set("attempts", record.GetInt("attempts")+1)
		if err := tx.Save(record); err != nil {
			return err
		}
		claimed = record.Clone()
		return nil
	})
	return claimed, err
}

func (s *Service) deliver(ctx context.Context, record *core.Record) {
	body := record.GetString("body")
	timestamp := strconv.FormatInt(s.now().Unix(), 10)
	signature := Sign(s.Config.OutgoingWebhookSecret, timestamp, []byte(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, record.GetString("url"), strings.NewReader(body))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "PayGate/1.0")
		req.Header.Set("X-PayGate-Event-Id", record.GetString("event_id"))
		req.Header.Set("X-PayGate-Timestamp", timestamp)
		req.Header.Set("X-PayGate-Signature", "v1="+signature)
	}

	statusCode := 0
	if err == nil {
		var response *http.Response
		response, err = s.client().Do(req)
		if response != nil {
			statusCode = response.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
			_ = response.Body.Close()
			if statusCode < 200 || statusCode >= 300 {
				err = fmt.Errorf("webhook returned HTTP %d", statusCode)
			}
		}
	}
	if finishErr := s.finish(record.Id, statusCode, err); finishErr != nil {
		s.logger().Error("failed to persist webhook result", "id", record.Id, "error", finishErr)
	}
}

func (s *Service) finish(id string, statusCode int, deliveryErr error) error {
	now := s.now()
	return s.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById("webhook_deliveries", id)
		if err != nil {
			return err
		}
		record.Set("locked_at", "")
		record.Set("response_code", statusCode)
		if deliveryErr == nil {
			record.Set("status", "delivered")
			record.Set("delivered_at", now)
			record.Set("last_error", "")
			return tx.Save(record)
		}

		attempts := record.GetInt("attempts")
		record.Set("last_error", truncate(deliveryErr.Error(), 4000))
		if attempts >= maxAttempts {
			record.Set("status", "exhausted")
			// Keep a valid date for the required field; exhausted records aren't queried.
			record.Set("next_attempt_at", now.Add(365*24*time.Hour))
		} else {
			record.Set("status", "failed")
			record.Set("next_attempt_at", now.Add(retryDelay(attempts)))
		}
		return tx.Save(record)
	})
}

func (s *Service) recoverStale(now time.Time) error {
	stale := now.Add(-2 * time.Minute)
	records, err := s.App.FindRecordsByFilter(
		"webhook_deliveries",
		"status = 'sending' && locked_at < {:stale}",
		"locked_at",
		50,
		0,
		dbx.Params{"stale": filterDate(stale)},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set("status", "failed")
		record.Set("locked_at", "")
		record.Set("next_attempt_at", now)
		record.Set("last_error", "recovered stale delivery lease after restart")
		if err := s.App.Save(record); err != nil {
			return err
		}
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

func retryDelay(attempt int) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[index]
}

func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
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
