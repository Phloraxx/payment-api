package audit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
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
	App   core.App // transitional constructor dependency
	Store store.Database
	Now   func() time.Time
}

func NewService(app core.App) *Service {
	return &Service{App: app, Store: store.NewPocketBase(app), Now: time.Now}
}

func (s *Service) Record(entry Entry) error {
	return s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		return s.RecordUoW(uow, entry)
	})
}

func (s *Service) RecordUoW(uow store.UnitOfWork, entry Entry) error {
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
	return uow.Audit().Record(domain.AuditEvent{
		Action: entry.Action, ActorID: truncate(entry.Actor.ID, 255), ActorEmail: truncate(entry.Actor.Email, 255),
		EntityType: entry.EntityType, EntityID: entry.EntityID, Summary: entry.Summary, Details: entry.Details, OccurredAt: at,
	})
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
