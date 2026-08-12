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
		if payments.Fields.GetByName("payment_account") == nil {
			payments.Fields.Add(&core.SelectField{Name: "payment_account", Values: []string{"kotak", "slice"}})
			if err := app.Save(payments); err != nil {
				return err
			}
			records, err := app.FindAllRecords("payments")
			if err != nil {
				return err
			}
			for _, record := range records {
				record.Set("payment_account", "kotak")
				if err := app.Save(record); err != nil {
					return err
				}
			}
			if field, ok := payments.Fields.GetByName("payment_account").(*core.SelectField); ok {
				field.Required = true
			}
		}
		payments.AddIndex("idx_payments_account_amount", false, "payment_account,payable_amount,status,reuse_after", "")
		if err := app.Save(payments); err != nil {
			return err
		}

		for _, collectionName := range []string{"sms_events", "email_events"} {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return err
			}
			if collection.Fields.GetByName("payment_account") == nil {
				collection.Fields.Add(&core.SelectField{Name: "payment_account", Values: []string{"kotak", "slice"}})
				if err := app.Save(collection); err != nil {
					return err
				}
				account := "kotak"
				if collectionName == "email_events" {
					account = "slice"
				}
				records, err := app.FindAllRecords(collectionName)
				if err != nil {
					return err
				}
				for _, record := range records {
					record.Set("payment_account", account)
					if err := app.Save(record); err != nil {
						return err
					}
				}
				if field, ok := collection.Fields.GetByName("payment_account").(*core.SelectField); ok {
					field.Required = true
				}
			}
			collection.AddIndex("idx_"+collectionName+"_account", false, "payment_account,processing_status,created", "")
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, collectionName := range []string{"sms_events", "email_events"} {
			if collection, err := app.FindCollectionByNameOrId(collectionName); err == nil {
				collection.RemoveIndex("idx_" + collectionName + "_account")
				collection.Fields.RemoveByName("payment_account")
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}
		if payments, err := app.FindCollectionByNameOrId("payments"); err == nil {
			payments.RemoveIndex("idx_payments_account_amount")
			payments.Fields.RemoveByName("payment_account")
			return app.Save(payments)
		}
		return nil
	})
}
