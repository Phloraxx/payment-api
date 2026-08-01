package audit

import (
	"testing"
	"time"

	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRecordPersistsAttributableAuditEntry(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	service := NewService(app)
	service.Now = func() time.Time { return now }
	if err := service.Record(Entry{
		Action: "review.resolve", Actor: Actor{ID: "user-1", Email: "operator@example.com"},
		EntityType: "review_case", EntityID: "case-1", Summary: "Manually matched bank evidence",
		Details: map[string]any{"paymentId": "payment-1"},
	}); err != nil {
		t.Fatal(err)
	}
	records, err := app.FindAllRecords("audit_events")
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%d err=%v", len(records), err)
	}
	if got := records[0].GetString("actor_email"); got != "operator@example.com" {
		t.Fatalf("actor_email=%q", got)
	}
	if got := records[0].GetString("action"); got != "review.resolve" {
		t.Fatalf("action=%q", got)
	}
}
