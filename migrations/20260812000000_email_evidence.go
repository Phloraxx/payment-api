package migrations

import (
	"database/sql"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		payments, err := app.FindCollectionByNameOrId("payments")
		if err != nil {
			return err
		}
		emailEvents, err := findOrCreateEmailEvents(app, payments.Id)
		if err != nil {
			return err
		}
		reviews, err := app.FindCollectionByNameOrId("review_cases")
		if err != nil {
			return err
		}
		if reviews.Fields.GetByName("email_event") == nil {
			reviews.Fields.Add(&core.RelationField{Name: "email_event", CollectionId: emailEvents.Id, MaxSelect: 1})
			reviews.AddIndex("idx_review_email_unique", true, "email_event", "email_event != ''")
		}
		if field, ok := reviews.Fields.GetByName("kind").(*core.SelectField); ok {
			field.Values = uniqueStrings(append(field.Values, "email_auth_failed"))
		}
		return app.Save(reviews)
	}, func(app core.App) error {
		if reviews, err := app.FindCollectionByNameOrId("review_cases"); err == nil {
			reviews.RemoveIndex("idx_review_email_unique")
			reviews.Fields.RemoveByName("email_event")
			if field, ok := reviews.Fields.GetByName("kind").(*core.SelectField); ok {
				values := make([]string, 0, len(field.Values))
				for _, value := range field.Values {
					if value != "email_auth_failed" {
						values = append(values, value)
					}
				}
				field.Values = values
			}
			if err := app.Save(reviews); err != nil {
				return err
			}
		}
		collection, err := app.FindCollectionByNameOrId("email_events")
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}

func findOrCreateEmailEvents(app core.App, paymentsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("email_events"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("email_events")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.SelectField{Name: "source", Values: []string{"cloudflare_email", "manual"}, Required: true},
		&core.TextField{Name: "source_event_id", Max: 255},
		&core.TextField{Name: "envelope_sender", Max: 255},
		&core.TextField{Name: "recipient", Max: 255},
		&core.TextField{Name: "sender", Max: 255},
		&core.TextField{Name: "subject", Max: 1024},
		&core.TextField{Name: "body", Max: 64 * 1024},
		&core.DateField{Name: "message_time"},
		&core.DateField{Name: "received_at", Required: true},
		&core.TextField{Name: "auth_result", Max: 8192},
		&core.NumberField{Name: "amount", OnlyInt: true},
		&core.TextField{Name: "rrn", Max: 64},
		&core.TextField{Name: "upi_id", Max: 255},
		&core.TextField{Name: "payer_name", Max: 255},
		&core.SelectField{Name: "processing_status", Values: []string{"received", "parsed", "matched", "duplicate", "unmatched", "ignored", "error"}, Required: true},
		&core.RelationField{Name: "matched_payment", CollectionId: paymentsID, MaxSelect: 1},
		&core.TextField{Name: "error", Max: 4096},
		&core.JSONField{Name: "raw_payload", MaxSize: 256 * 1024},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_email_source_event_nonempty", true, "source,source_event_id", "source_event_id != ''")
	c.AddIndex("idx_email_processing", false, "processing_status,created", "")
	c.AddIndex("idx_email_rrn", false, "rrn", "rrn != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}
