package audit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type Actor struct {
	ID    string
	Email string
}

type Entry struct {
	Action     string
	Actor      Actor
	EntityType string
	EntityID   string
	Summary    string
	Details    any
	OccurredAt time.Time
}

type Service struct {
	App core.App
	Now func() time.Time
}

func NewService(app core.App) *Service {
	return &Service{App: app, Now: time.Now}
}

func (s *Service) Record(entry Entry) error {
	return s.RecordInApp(s.App, entry)
}

func (s *Service) RecordInApp(app core.App, entry Entry) error {
	entry.Action = strings.TrimSpace(entry.Action)
	entry.EntityType = strings.TrimSpace(entry.EntityType)
	entry.EntityID = strings.TrimSpace(entry.EntityID)
	entry.Summary = strings.TrimSpace(entry.Summary)
	if entry.Action == "" || entry.EntityType == "" || entry.EntityID == "" || entry.Summary == "" {
		return fmt.Errorf("audit action, entity type, entity id and summary are required")
	}
	if len(entry.Action) > 128 || len(entry.EntityType) > 64 || len(entry.EntityID) > 255 || len(entry.Summary) > 1024 {
		return fmt.Errorf("audit entry exceeds storage limits")
	}
	if entry.Details != nil {
		raw, err := json.Marshal(entry.Details)
		if err != nil || len(raw) > 1<<20 {
			return fmt.Errorf("audit details are invalid or exceed 1 MiB")
		}
	}
	at := entry.OccurredAt.UTC()
	if at.IsZero() {
		at = s.now()
	}
	collection, err := app.FindCollectionByNameOrId("audit_events")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("action", entry.Action)
	record.Set("actor_id", truncate(entry.Actor.ID, 255))
	record.Set("actor_email", truncate(entry.Actor.Email, 255))
	record.Set("entity_type", entry.EntityType)
	record.Set("entity_id", entry.EntityID)
	record.Set("summary", entry.Summary)
	record.Set("occurred_at", at)
	if entry.Details != nil {
		record.Set("details", entry.Details)
	}
	return app.Save(record)
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}
