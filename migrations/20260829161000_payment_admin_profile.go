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
		fields := []core.Field{
			&core.TextField{Name: "display_name", Max: 255},
			&core.TextField{Name: "customer_name", Max: 255},
			&core.TextField{Name: "customer_email", Max: 254},
			&core.TextField{Name: "customer_phone", Max: 64},
			&core.TextField{Name: "description", Max: 4096},
			&core.TextField{Name: "admin_note", Max: 4096},
			&core.JSONField{Name: "tags", MaxSize: 64 * 1024},
			&core.JSONField{Name: "custom_fields", MaxSize: 256 * 1024},
		}
		for _, field := range fields {
			if payments.Fields.GetByName(field.GetName()) == nil {
				payments.Fields.Add(field)
			}
		}
		payments.AddIndex("idx_payments_customer_email", false, "customer_email", "customer_email != ''")
		payments.AddIndex("idx_payments_customer_phone", false, "customer_phone", "customer_phone != ''")
		return app.Save(payments)
	}, func(app core.App) error {
		payments, err := app.FindCollectionByNameOrId("payments")
		if err != nil {
			return nil
		}
		payments.RemoveIndex("idx_payments_customer_email")
		payments.RemoveIndex("idx_payments_customer_phone")
		for _, name := range []string{
			"display_name", "customer_name", "customer_email", "customer_phone",
			"description", "admin_note", "tags", "custom_fields",
		} {
			payments.Fields.RemoveByName(name)
		}
		return app.Save(payments)
	})
}
