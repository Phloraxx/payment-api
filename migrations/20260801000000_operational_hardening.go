package migrations

import (
	"database/sql"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		payments, err := app.FindCollectionByNameOrId("payments")
		if err != nil {
			return err
		}
		smsEvents, err := app.FindCollectionByNameOrId("sms_events")
		if err != nil {
			return err
		}
		webhookDeliveries, err := app.FindCollectionByNameOrId("webhook_deliveries")
		if err != nil {
			return err
		}

		if payments.Fields.GetByName("resolved_at") == nil {
			payments.Fields.Add(&core.DateField{Name: "resolved_at"})
			payments.AddIndex("idx_payments_resolved_at", false, "resolved_at", "resolved_at != ''")
			if err := app.Save(payments); err != nil {
				return err
			}
		}

		auditEvents, err := findOrCreateAuditEvents(app)
		if err != nil {
			return err
		}
		_ = auditEvents

		reviewCases, err := findOrCreateReviewCases(app, users.Id, payments.Id, smsEvents.Id)
		if err != nil {
			return err
		}
		_ = reviewCases

		runs, err := findOrCreateReconciliationRuns(app, users.Id)
		if err != nil {
			return err
		}
		entries, err := findOrCreateReconciliationEntries(app, runs.Id, payments.Id)
		if err != nil {
			return err
		}
		if reviewCases.Fields.GetByName("reconciliation_entry") == nil {
			reviewCases.Fields.Add(&core.RelationField{Name: "reconciliation_entry", CollectionId: entries.Id, MaxSelect: 1})
			reviewCases.AddIndex("idx_review_reconciliation_unique", true, "reconciliation_entry", "reconciliation_entry != ''")
			if err := app.Save(reviewCases); err != nil {
				return err
			}
		}
		if _, err := findOrCreateAlerts(app); err != nil {
			return err
		}
		refunds, err := findOrCreateRefunds(app, users.Id, payments.Id)
		if err != nil {
			return err
		}

		if webhookDeliveries.Fields.GetByName("refund") == nil {
			webhookDeliveries.Fields.Add(&core.RelationField{Name: "refund", CollectionId: refunds.Id, MaxSelect: 1})
		}
		if field, ok := webhookDeliveries.Fields.GetByName("event").(*core.SelectField); ok {
			field.Values = uniqueStrings(append(field.Values,
				"refund.requested", "refund.processing", "refund.completed", "refund.failed", "refund.cancelled",
			))
		}
		return app.Save(webhookDeliveries)
	}, func(app core.App) error {
		if webhookDeliveries, err := app.FindCollectionByNameOrId("webhook_deliveries"); err == nil {
			webhookDeliveries.Fields.RemoveByName("refund")
			if field, ok := webhookDeliveries.Fields.GetByName("event").(*core.SelectField); ok {
				field.Values = []string{"payment.paid", "payment.late", "payment.expired", "payment.cancelled"}
			}
			if err := app.Save(webhookDeliveries); err != nil {
				return err
			}
		}
		for _, name := range []string{"refunds", "alerts", "review_cases", "reconciliation_entries", "reconciliation_runs", "audit_events"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return err
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		if payments, err := app.FindCollectionByNameOrId("payments"); err == nil {
			payments.RemoveIndex("idx_payments_resolved_at")
			payments.Fields.RemoveByName("resolved_at")
			if err := app.Save(payments); err != nil {
				return err
			}
		}
		return nil
	})
}

func findOrCreateAuditEvents(app core.App) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("audit_events"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("audit_events")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.TextField{Name: "action", Max: 128, Required: true},
		&core.TextField{Name: "actor_id", Max: 255},
		&core.TextField{Name: "actor_email", Max: 255},
		&core.TextField{Name: "entity_type", Max: 64, Required: true},
		&core.TextField{Name: "entity_id", Max: 255, Required: true},
		&core.TextField{Name: "summary", Max: 1024, Required: true},
		&core.JSONField{Name: "details", MaxSize: 1 << 20},
		&core.DateField{Name: "occurred_at", Required: true},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	c.AddIndex("idx_audit_entity", false, "entity_type,entity_id,occurred_at", "")
	c.AddIndex("idx_audit_actor", false, "actor_id,occurred_at", "actor_id != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateReviewCases(app core.App, usersID, paymentsID, smsEventsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("review_cases"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("review_cases")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.SelectField{Name: "kind", Values: []string{"missing_rrn", "parse_error", "unmatched", "ambiguous", "rrn_conflict", "reconciliation_conflict", "manual"}, Required: true},
		&core.SelectField{Name: "status", Values: []string{"open", "resolved", "dismissed"}, Required: true},
		&core.SelectField{Name: "severity", Values: []string{"info", "warning", "critical"}, Required: true},
		&core.RelationField{Name: "sms_event", CollectionId: smsEventsID, MaxSelect: 1},
		&core.RelationField{Name: "payment", CollectionId: paymentsID, MaxSelect: 1},
		&core.JSONField{Name: "candidate_payment_ids", MaxSize: 64 * 1024},
		&core.TextField{Name: "reason", Max: 4096, Required: true},
		&core.SelectField{Name: "resolution", Values: []string{"manual_match", "dismissed", "duplicate", "not_payment", "corrected"}},
		&core.TextField{Name: "resolution_note", Max: 4096},
		&core.RelationField{Name: "resolved_by", CollectionId: usersID, MaxSelect: 1},
		&core.DateField{Name: "opened_at", Required: true},
		&core.DateField{Name: "resolved_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_review_open", false, "status,severity,opened_at", "status = 'open'")
	c.AddIndex("idx_review_sms_unique", true, "sms_event", "sms_event != ''")
	c.AddIndex("idx_review_payment", false, "payment,status", "payment != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateReconciliationRuns(app core.App, usersID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("reconciliation_runs"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("reconciliation_runs")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.TextField{Name: "filename", Max: 255, Required: true},
		&core.TextField{Name: "sha256", Max: 64, Required: true},
		&core.SelectField{Name: "status", Values: []string{"processing", "completed", "failed"}, Required: true},
		&core.NumberField{Name: "total_rows", OnlyInt: true},
		&core.NumberField{Name: "matched_rows", OnlyInt: true},
		&core.NumberField{Name: "unmatched_rows", OnlyInt: true},
		&core.NumberField{Name: "duplicate_rows", OnlyInt: true},
		&core.NumberField{Name: "conflict_rows", OnlyInt: true},
		&core.NumberField{Name: "invalid_rows", OnlyInt: true},
		&core.TextField{Name: "error", Max: 4096},
		&core.JSONField{Name: "summary", MaxSize: 1 << 20},
		&core.RelationField{Name: "created_by", CollectionId: usersID, MaxSelect: 1},
		&core.DateField{Name: "started_at", Required: true},
		&core.DateField{Name: "completed_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_reconciliation_runs", false, "status,started_at", "")
	c.AddIndex("idx_reconciliation_hash", false, "sha256,started_at", "")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateReconciliationEntries(app core.App, runsID, paymentsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("reconciliation_entries"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("reconciliation_entries")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.RelationField{Name: "run", CollectionId: runsID, MaxSelect: 1, Required: true},
		&core.NumberField{Name: "row_number", OnlyInt: true, Required: true},
		&core.DateField{Name: "transaction_time"},
		&core.NumberField{Name: "amount", OnlyInt: true},
		&core.TextField{Name: "rrn", Max: 64},
		&core.TextField{Name: "description", Max: 4096},
		&core.SelectField{Name: "status", Values: []string{"matched", "unmatched", "duplicate", "conflict", "invalid"}, Required: true},
		&core.RelationField{Name: "payment", CollectionId: paymentsID, MaxSelect: 1},
		&core.TextField{Name: "notes", Max: 4096},
		&core.JSONField{Name: "raw_row", MaxSize: 256 * 1024},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	c.AddIndex("idx_reconciliation_run_row", true, "run,row_number", "")
	c.AddIndex("idx_reconciliation_status", false, "run,status", "")
	c.AddIndex("idx_reconciliation_rrn", false, "rrn", "rrn != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateAlerts(app core.App) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("alerts"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("alerts")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.SelectField{Name: "kind", Values: []string{"connector_disconnected", "connector_reauth", "connector_unresponsive", "webhook_exhausted", "capacity_high", "backup_failed", "reconciliation_conflict"}, Required: true},
		&core.SelectField{Name: "status", Values: []string{"open", "resolved"}, Required: true},
		&core.SelectField{Name: "severity", Values: []string{"warning", "critical"}, Required: true},
		&core.TextField{Name: "dedupe_key", Max: 255, Required: true},
		&core.TextField{Name: "message", Max: 4096, Required: true},
		&core.JSONField{Name: "details", MaxSize: 1 << 20},
		&core.NumberField{Name: "occurrence_count", OnlyInt: true, Required: true},
		&core.DateField{Name: "first_seen_at", Required: true},
		&core.DateField{Name: "last_seen_at", Required: true},
		&core.DateField{Name: "resolved_at"},
		&core.TextField{Name: "notification_event_id", Max: 64},
		&core.SelectField{Name: "notification_status", Values: []string{"disabled", "pending", "sending", "delivered", "failed", "exhausted"}, Required: true},
		&core.NumberField{Name: "notification_attempts", OnlyInt: true},
		&core.DateField{Name: "notification_created_at"},
		&core.DateField{Name: "notification_next_attempt_at"},
		&core.DateField{Name: "notification_locked_at"},
		&core.DateField{Name: "notification_last_attempt_at"},
		&core.DateField{Name: "notification_delivered_at"},
		&core.TextField{Name: "notification_last_error", Max: 4096},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_alert_dedupe", true, "dedupe_key", "")
	c.AddIndex("idx_alert_open", false, "status,severity,last_seen_at", "status = 'open'")
	c.AddIndex("idx_alert_notification", false, "notification_status,notification_next_attempt_at", "")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateRefunds(app core.App, usersID, paymentsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("refunds"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("refunds")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.RelationField{Name: "payment", CollectionId: paymentsID, MaxSelect: 1, Required: true},
		&core.NumberField{Name: "amount", OnlyInt: true, Required: true},
		&core.SelectField{Name: "status", Values: []string{"requested", "processing", "completed", "failed", "cancelled"}, Required: true},
		&core.TextField{Name: "reason", Max: 4096},
		&core.TextField{Name: "reference", Max: 255},
		&core.TextField{Name: "external_id", Max: 255},
		&core.TextField{Name: "idempotency_key", Max: 255},
		&core.JSONField{Name: "metadata", MaxSize: 1 << 20},
		&core.RelationField{Name: "requested_by", CollectionId: usersID, MaxSelect: 1},
		&core.DateField{Name: "requested_at", Required: true},
		&core.DateField{Name: "completed_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_refunds_payment", false, "payment,status,requested_at", "")
	c.AddIndex("idx_refunds_idempotency", true, "idempotency_key", "idempotency_key != ''")
	c.AddIndex("idx_refunds_reference", true, "reference", "reference != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
