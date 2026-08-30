package deliveryqueue

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func testQueue(t *testing.T) (*tests.TestApp, Queue) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	collection := core.NewBaseCollection("delivery_queue_test")
	collection.Fields.Add(
		&core.SelectField{Name: "status", Values: []string{"pending", "sending", "delivered", "failed", "exhausted"}, Required: true},
		&core.NumberField{Name: "attempts", OnlyInt: true},
		&core.DateField{Name: "next_attempt_at", Required: true},
		&core.DateField{Name: "locked_at"},
		&core.DateField{Name: "last_attempt_at"},
		&core.DateField{Name: "delivered_at"},
		&core.TextField{Name: "last_error", Max: 4096},
		&core.NumberField{Name: "response_code", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(collection); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}
	queue := Queue{
		App: app, Collection: collection.Name, MaxAttempts: 2,
		Fields: Fields{
			Status: "status", Attempts: "attempts", NextAttemptAt: "next_attempt_at",
			LockedAt: "locked_at", LastAttemptAt: "last_attempt_at", DeliveredAt: "delivered_at",
			LastError: "last_error", ResponseCode: "response_code",
		},
		RetryDelays: []time.Duration{time.Minute, 5 * time.Minute},
		StaleAfter:  time.Minute, ExhaustedAfter: 24 * time.Hour, ErrorMax: 100,
		StaleMessage: "stale lease recovered",
	}
	return app, queue
}

func newQueueRecord(t *testing.T, app *tests.TestApp, status string, next, locked time.Time) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("delivery_queue_test")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("status", status)
	record.Set("next_attempt_at", next)
	if !locked.IsZero() {
		record.Set("locked_at", locked)
	}
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestQueueClaimsRetriesAndExhausts(t *testing.T) {
	app, queue := testQueue(t)
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	record := newQueueRecord(t, app, "pending", now.Add(-time.Minute), time.Time{})

	due, err := queue.Due(now, 10)
	if err != nil || len(due) != 1 || due[0].Id != record.Id {
		t.Fatalf("due=%v err=%v", due, err)
	}
	claimed, err := queue.Claim(record.Id, now)
	if err != nil || claimed == nil || claimed.GetString("status") != "sending" || claimed.GetInt("attempts") != 1 {
		t.Fatalf("first claim=%v err=%v", claimed, err)
	}
	if err := queue.Finish(record.Id, now, 503, errors.New("temporary failure"), nil); err != nil {
		t.Fatal(err)
	}
	failed, _ := app.FindRecordById("delivery_queue_test", record.Id)
	if failed.GetString("status") != "failed" || failed.GetInt("response_code") != 503 {
		t.Fatalf("first failure status=%s code=%d", failed.GetString("status"), failed.GetInt("response_code"))
	}
	failed.Set("next_attempt_at", now)
	if err := app.Save(failed); err != nil {
		t.Fatal(err)
	}
	claimed, err = queue.Claim(record.Id, now)
	if err != nil || claimed == nil || claimed.GetInt("attempts") != 2 {
		t.Fatalf("second claim=%v err=%v", claimed, err)
	}
	if err := queue.Finish(record.Id, now, 500, errors.New("final failure"), nil); err != nil {
		t.Fatal(err)
	}
	exhausted, _ := app.FindRecordById("delivery_queue_test", record.Id)
	if exhausted.GetString("status") != "exhausted" {
		t.Fatalf("status=%s want exhausted", exhausted.GetString("status"))
	}
}

func TestQueueGuardAndStaleRecovery(t *testing.T) {
	app, queue := testQueue(t)
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	record := newQueueRecord(t, app, "pending", now.Add(-time.Minute), time.Time{})
	claimed, err := queue.Claim(record.Id, now)
	if err != nil || claimed == nil {
		t.Fatalf("claim err=%v", err)
	}
	if err := queue.Finish(record.Id, now, 204, nil, func(*core.Record) bool { return false }); err != nil {
		t.Fatal(err)
	}
	unchanged, _ := app.FindRecordById("delivery_queue_test", record.Id)
	if unchanged.GetString("status") != "sending" {
		t.Fatalf("guarded finish changed status to %s", unchanged.GetString("status"))
	}
	unchanged.Set("locked_at", now.Add(-2*time.Minute))
	if err := app.Save(unchanged); err != nil {
		t.Fatal(err)
	}
	if err := queue.RecoverStale(now, 10); err != nil {
		t.Fatal(err)
	}
	recovered, _ := app.FindRecordById("delivery_queue_test", record.Id)
	if recovered.GetString("status") != "failed" || recovered.GetString("last_error") != "stale lease recovered" {
		t.Fatalf("recovered status=%s error=%q", recovered.GetString("status"), recovered.GetString("last_error"))
	}
}
