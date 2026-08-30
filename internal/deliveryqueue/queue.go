package deliveryqueue

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type Fields struct {
	Status        string
	Attempts      string
	NextAttemptAt string
	LockedAt      string
	LastAttemptAt string
	DeliveredAt   string
	LastError     string
	ResponseCode  string
}

type Queue struct {
	App            core.App
	Collection     string
	Fields         Fields
	MaxAttempts    int
	RetryDelays    []time.Duration
	StaleAfter     time.Duration
	ExhaustedAfter time.Duration
	ErrorMax       int
	StaleMessage   string
}

type FinishGuard func(*core.Record) bool

func (q Queue) Due(now time.Time, limit int) ([]*core.Record, error) {
	if limit <= 0 {
		limit = 50
	}
	filter := fmt.Sprintf(
		"(%s = 'pending' || %s = 'failed') && %s <= {:now}",
		q.Fields.Status, q.Fields.Status, q.Fields.NextAttemptAt,
	)
	sort := q.Fields.NextAttemptAt + ",created"
	return q.App.FindRecordsByFilter(
		q.Collection, filter, sort, limit, 0,
		dbx.Params{"now": filterDate(now)},
	)
}

func (q Queue) Claim(id string, now time.Time) (*core.Record, error) {
	var claimed *core.Record
	err := q.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById(q.Collection, id)
		if err != nil {
			return err
		}
		status := record.GetString(q.Fields.Status)
		if status != "pending" && status != "failed" {
			return nil
		}
		if next := record.GetDateTime(q.Fields.NextAttemptAt).Time(); !next.IsZero() && next.After(now) {
			return nil
		}
		record.Set(q.Fields.Status, "sending")
		record.Set(q.Fields.LockedAt, now)
		record.Set(q.Fields.LastAttemptAt, now)
		record.Set(q.Fields.Attempts, record.GetInt(q.Fields.Attempts)+1)
		if err := tx.Save(record); err != nil {
			return err
		}
		claimed = record.Clone()
		return nil
	})
	return claimed, err
}

func (q Queue) Finish(id string, now time.Time, statusCode int, deliveryErr error, guard FinishGuard) error {
	return q.App.RunInTransaction(func(tx core.App) error {
		record, err := tx.FindRecordById(q.Collection, id)
		if err != nil {
			return err
		}
		if guard != nil && !guard(record) {
			return nil
		}
		record.Set(q.Fields.LockedAt, "")
		if q.Fields.ResponseCode != "" {
			record.Set(q.Fields.ResponseCode, statusCode)
		}
		if deliveryErr == nil {
			record.Set(q.Fields.Status, "delivered")
			record.Set(q.Fields.DeliveredAt, now)
			record.Set(q.Fields.LastError, "")
			return tx.Save(record)
		}

		attempts := record.GetInt(q.Fields.Attempts)
		record.Set(q.Fields.LastError, truncate(deliveryErr.Error(), q.errorMax()))
		if attempts >= q.maxAttempts() {
			record.Set(q.Fields.Status, "exhausted")
			record.Set(q.Fields.NextAttemptAt, now.Add(q.exhaustedAfter()))
		} else {
			record.Set(q.Fields.Status, "failed")
			record.Set(q.Fields.NextAttemptAt, now.Add(q.retryDelay(attempts)))
		}
		return tx.Save(record)
	})
}

func (q Queue) RecoverStale(now time.Time, limit int) error {
	if limit <= 0 {
		limit = 50
	}
	stale := now.Add(-q.staleAfter())
	filter := fmt.Sprintf("%s = 'sending' && %s < {:stale}", q.Fields.Status, q.Fields.LockedAt)
	records, err := q.App.FindRecordsByFilter(
		q.Collection, filter, q.Fields.LockedAt, limit, 0,
		dbx.Params{"stale": filterDate(stale)},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set(q.Fields.Status, "failed")
		record.Set(q.Fields.LockedAt, "")
		record.Set(q.Fields.NextAttemptAt, now)
		record.Set(q.Fields.LastError, q.staleMessage())
		if err := q.App.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func (q Queue) retryDelay(attempt int) time.Duration {
	if len(q.RetryDelays) == 0 {
		return time.Minute
	}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(q.RetryDelays) {
		index = len(q.RetryDelays) - 1
	}
	return q.RetryDelays[index]
}
func (q Queue) maxAttempts() int {
	if q.MaxAttempts > 0 {
		return q.MaxAttempts
	}
	return 1
}

func (q Queue) staleAfter() time.Duration {
	if q.StaleAfter > 0 {
		return q.StaleAfter
	}
	return 2 * time.Minute
}

func (q Queue) exhaustedAfter() time.Duration {
	if q.ExhaustedAfter > 0 {
		return q.ExhaustedAfter
	}
	return 365 * 24 * time.Hour
}

func (q Queue) errorMax() int {
	if q.ErrorMax > 0 {
		return q.ErrorMax
	}
	return 4096
}

func (q Queue) staleMessage() string {
	if strings.TrimSpace(q.StaleMessage) != "" {
		return strings.TrimSpace(q.StaleMessage)
	}
	return "recovered stale delivery lease after restart"
}
func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
