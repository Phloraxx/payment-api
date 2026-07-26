package migrations

import (
	"database/sql"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		users, err := findOrCreateUsers(app)
		if err != nil {
			return err
		}
		_ = users

		payments, err := findOrCreatePayments(app)
		if err != nil {
			return err
		}
		if _, err := findOrCreateSMSEvents(app, payments.Id); err != nil {
			return err
		}
		if _, err := findOrCreateWebhookDeliveries(app, payments.Id); err != nil {
			return err
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"webhook_deliveries", "sms_events", "payments", "users"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				continue
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}

func findOrCreateUsers(app core.App) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("users"); err == nil {
		return c, nil
	}
	c := core.NewAuthCollection("users")
	c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	// Dashboard accounts are created by a superuser; public self-registration is disabled.
	c.CreateRule = nil
	c.UpdateRule = nil
	c.DeleteRule = nil
	c.ListRule = nil
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreatePayments(app core.App) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("payments"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("payments")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.DateField{Name: "created_at", Required: true},
		&core.NumberField{Name: "requested_amount", OnlyInt: true, Required: true},
		&core.NumberField{Name: "payable_amount", OnlyInt: true, Required: true},
		&core.SelectField{Name: "status", Values: []string{"pending", "paid", "expired", "cancelled", "late"}, Required: true},
		&core.DateField{Name: "expires_at", Required: true},
		&core.DateField{Name: "reuse_after", Required: true},
		&core.TextField{Name: "rrn", Max: 64},
		&core.TextField{Name: "upi_id", Max: 255},
		&core.TextField{Name: "payer_name", Max: 255},
		&core.DateField{Name: "paid_at"},
		&core.TextField{Name: "external_id", Max: 255},
		&core.TextField{Name: "idempotency_key", Max: 255},
		&core.JSONField{Name: "metadata", MaxSize: 1 << 20},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_payments_created_at", false, "created_at", "")
	c.AddIndex("idx_payments_payable", false, "payable_amount", "")
	c.AddIndex("idx_payments_allocation", false, "payable_amount,status,reuse_after", "")
	c.AddIndex("idx_payments_expiry", false, "status,expires_at", "status = 'pending'")
	c.AddIndex("idx_payments_rrn_nonempty", true, "rrn", "rrn != ''")
	c.AddIndex("idx_payments_idempotency_nonempty", true, "idempotency_key", "idempotency_key != ''")
	c.AddIndex("idx_payments_external_id", false, "external_id", "external_id != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateSMSEvents(app core.App, paymentsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("sms_events"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("sms_events")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.SelectField{Name: "source", Values: []string{"android_webhook", "gmessages", "manual"}, Required: true},
		&core.TextField{Name: "source_event_id", Max: 255},
		&core.TextField{Name: "sender", Max: 255},
		&core.TextField{Name: "body", Max: 64 * 1024, Required: true},
		&core.DateField{Name: "message_time"},
		&core.NumberField{Name: "amount", OnlyInt: true},
		&core.TextField{Name: "rrn", Max: 64},
		&core.TextField{Name: "upi_id", Max: 255},
		&core.TextField{Name: "payer_name", Max: 255},
		&core.SelectField{Name: "processing_status", Values: []string{"received", "parsed", "matched", "duplicate", "unmatched", "ignored", "error"}, Required: true},
		&core.RelationField{Name: "matched_payment", CollectionId: paymentsID, MaxSelect: 1},
		&core.TextField{Name: "error", Max: 4096},
		&core.JSONField{Name: "raw_payload", MaxSize: 1 << 20},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_sms_source_event_nonempty", true, "source,source_event_id", "source_event_id != ''")
	c.AddIndex("idx_sms_processing", false, "processing_status,created", "")
	c.AddIndex("idx_sms_rrn", false, "rrn", "rrn != ''")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func findOrCreateWebhookDeliveries(app core.App, paymentsID string) (*core.Collection, error) {
	if c, err := app.FindCollectionByNameOrId("webhook_deliveries"); err == nil {
		return c, nil
	}
	c := core.NewBaseCollection("webhook_deliveries")
	lockDomainWrites(c)
	c.Fields.Add(
		&core.TextField{Name: "event_id", Max: 64, Required: true},
		&core.SelectField{Name: "event", Values: []string{"payment.paid", "payment.late", "payment.expired", "payment.cancelled"}, Required: true},
		&core.RelationField{Name: "payment", CollectionId: paymentsID, MaxSelect: 1, Required: true},
		&core.URLField{Name: "url", Required: true},
		&core.TextField{Name: "body", Max: 1 << 20, Required: true},
		&core.NumberField{Name: "attempts", OnlyInt: true},
		&core.SelectField{Name: "status", Values: []string{"pending", "sending", "delivered", "failed", "exhausted"}, Required: true},
		&core.DateField{Name: "next_attempt_at", Required: true},
		&core.DateField{Name: "locked_at"},
		&core.DateField{Name: "last_attempt_at"},
		&core.DateField{Name: "delivered_at"},
		&core.NumberField{Name: "response_code", OnlyInt: true},
		&core.TextField{Name: "last_error", Max: 4096},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_webhook_due", false, "status,next_attempt_at", "status = 'pending' OR status = 'failed'")
	c.AddIndex("idx_webhook_event_id", true, "event_id", "")
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func lockDomainWrites(c *core.Collection) {
	readRule := "@request.auth.id != '' && @request.auth.collectionName = 'users'"
	c.ListRule = types.Pointer(readRule)
	c.ViewRule = types.Pointer(readRule)
	c.CreateRule = nil
	c.UpdateRule = nil
	c.DeleteRule = nil
}
