package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		payments, err := app.FindCollectionByNameOrId("payments")
		if err != nil {
			return err
		}
		if field, ok := payments.Fields.GetByName("payment_account").(*core.SelectField); ok {
			field.Values = []string{"kotak", "slice", "paytm"}
		}
		if payments.Fields.GetByName("evidence_source") == nil {
			payments.Fields.Add(&core.TextField{Name: "evidence_source", Max: 64})
		}
		if payments.Fields.GetByName("evidence_reference") == nil {
			payments.Fields.Add(&core.TextField{Name: "evidence_reference", Max: 255})
		}
		payments.AddIndex("idx_payments_evidence_reference_nonempty", true, "evidence_reference", "evidence_reference != ''")
		if err := app.Save(payments); err != nil {
			return err
		}

		if _, err := app.FindCollectionByNameOrId("notification_events"); err == nil {
			return nil
		}
		c := core.NewBaseCollection("notification_events")
		lockDomainWrites(c)
		c.Fields.Add(
			&core.SelectField{Name: "source", Values: []string{"macrodroid"}, Required: true},
			&core.TextField{Name: "source_event_id", Max: 255, Required: true},
			&core.TextField{Name: "app_package", Max: 255, Required: true},
			&core.TextField{Name: "app_name", Max: 255},
			&core.TextField{Name: "title", Max: 4096},
			&core.TextField{Name: "body", Max: 64 * 1024},
			&core.TextField{Name: "big_text", Max: 64 * 1024},
			&core.TextField{Name: "channel", Max: 255},
			&core.DateField{Name: "notification_time", Required: true},
			&core.SelectField{Name: "payment_account", Values: []string{"paytm"}, Required: true},
			&core.NumberField{Name: "amount", OnlyInt: true},
			&core.TextField{Name: "payer_name", Max: 255},
			&core.SelectField{Name: "processing_status", Values: []string{"received", "parsed", "matched", "duplicate", "unmatched", "ignored", "error"}, Required: true},
			&core.RelationField{Name: "matched_payment", CollectionId: payments.Id, MaxSelect: 1},
			&core.TextField{Name: "error", Max: 4096},
			&core.JSONField{Name: "raw_payload", MaxSize: 1 << 20},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		c.AddIndex("idx_notification_source_event", true, "source,source_event_id", "")
		c.AddIndex("idx_notification_processing", false, "processing_status,created", "")
		if err := app.Save(c); err != nil {
			return err
		}
		return nil
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("notification_events"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		payments, err := app.FindCollectionByNameOrId("payments")
		if err != nil {
			return nil
		}
		payments.RemoveIndex("idx_payments_evidence_reference_nonempty")
		payments.Fields.RemoveByName("evidence_source")
		payments.Fields.RemoveByName("evidence_reference")
		if field, ok := payments.Fields.GetByName("payment_account").(*core.SelectField); ok {
			field.Values = []string{"kotak", "slice"}
		}
		return app.Save(payments)
	})
}
