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
		if notifications, err := app.FindCollectionByNameOrId("notification_events"); err == nil {
			if field, ok := notifications.Fields.GetByName("source").(*core.SelectField); ok {
				field.Values = []string{"macrodroid", "android_relay"}
				if err := app.Save(notifications); err != nil {
					return err
				}
			}
		}
		devices, err := app.FindCollectionByNameOrId("relay_devices")
		if err != nil {
			devices = core.NewBaseCollection("relay_devices")
			lockDomainWrites(devices)
			devices.Fields.Add(
				&core.TextField{Name: "device_id", Max: 64, Required: true},
				&core.TextField{Name: "name", Max: 120, Required: true},
				&core.TextField{Name: "public_key_pem", Max: 8192, Required: true},
				&core.BoolField{Name: "enabled"},
				&core.TextField{Name: "app_version", Max: 64},
				&core.TextField{Name: "android_version", Max: 64},
				&core.TextField{Name: "device_model", Max: 255},
				&core.DateField{Name: "last_seen_at"},
				&core.AutodateField{Name: "created", OnCreate: true},
				&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
			)
			devices.AddIndex("idx_relay_devices_device_id", true, "device_id", "")
			if err := app.Save(devices); err != nil {
				return err
			}
		}
		if _, err := app.FindCollectionByNameOrId("relay_events"); err == nil {
			return nil
		}
		c := core.NewBaseCollection("relay_events")
		lockDomainWrites(c)
		c.Fields.Add(
			&core.RelationField{Name: "device", CollectionId: devices.Id, MaxSelect: 1, Required: true},
			&core.TextField{Name: "event_id", Max: 64, Required: true},
			&core.SelectField{Name: "kind", Values: []string{"notification"}, Required: true},
			&core.TextField{Name: "app_package", Max: 255, Required: true},
			&core.TextField{Name: "app_name", Max: 255},
			&core.TextField{Name: "notification_key", Max: 512},
			&core.NumberField{Name: "notification_id", OnlyInt: true},
			&core.TextField{Name: "notification_tag", Max: 255},
			&core.TextField{Name: "group_key", Max: 512},
			&core.BoolField{Name: "is_group_summary"},
			&core.DateField{Name: "post_time"},
			&core.DateField{Name: "notification_when"},
			&core.DateField{Name: "captured_at"},
			&core.TextField{Name: "channel_id", Max: 255},
			&core.TextField{Name: "category", Max: 255},
			&core.TextField{Name: "title", Max: 4096},
			&core.TextField{Name: "body", Max: 64 * 1024},
			&core.TextField{Name: "big_text", Max: 64 * 1024},
			&core.TextField{Name: "sub_text", Max: 4096},
			&core.TextField{Name: "summary_text", Max: 4096},
			&core.JSONField{Name: "text_lines", MaxSize: 128 * 1024},
			&core.JSONField{Name: "custom_texts", MaxSize: 128 * 1024},
			&core.SelectField{Name: "processing_status", Values: []string{"received", "observed", "forwarded", "ignored", "duplicate", "error"}, Required: true},
			&core.TextField{Name: "downstream_event_id", Max: 64},
			&core.RelationField{Name: "matched_payment", CollectionId: payments.Id, MaxSelect: 1},
			&core.JSONField{Name: "provider_result", MaxSize: 64 * 1024},
			&core.TextField{Name: "error", Max: 4096},
			&core.JSONField{Name: "raw_payload", MaxSize: 1 << 20},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		c.AddIndex("idx_relay_events_device_event", true, "device,event_id", "")
		c.AddIndex("idx_relay_events_processing", false, "processing_status,created", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("relay_events"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		if c, err := app.FindCollectionByNameOrId("relay_devices"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		if c, err := app.FindCollectionByNameOrId("notification_events"); err == nil {
			if field, ok := c.Fields.GetByName("source").(*core.SelectField); ok {
				field.Values = []string{"macrodroid"}
				return app.Save(c)
			}
		}
		return nil
	})
}
