package migrations

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		devices, err := app.FindCollectionByNameOrId("relay_devices")
		if err != nil {
			return err
		}
		if devices.Fields.GetByName("enrolled_at") == nil {
			devices.Fields.Add(&core.DateField{Name: "enrolled_at"})
		}
		if devices.Fields.GetByName("last_heartbeat_at") == nil {
			devices.Fields.Add(&core.DateField{Name: "last_heartbeat_at"})
		}
		if devices.Fields.GetByName("notification_access") == nil {
			devices.Fields.Add(&core.BoolField{Name: "notification_access"})
		}
		if devices.Fields.GetByName("listener_connected") == nil {
			devices.Fields.Add(&core.BoolField{Name: "listener_connected"})
		}
		if devices.Fields.GetByName("pending_count") == nil {
			devices.Fields.Add(&core.NumberField{Name: "pending_count", OnlyInt: true, Min: float64Ptr(0)})
		}
		if devices.Fields.GetByName("failed_count") == nil {
			devices.Fields.Add(&core.NumberField{Name: "failed_count", OnlyInt: true, Min: float64Ptr(0)})
		}
		if devices.Fields.GetByName("last_client_error") == nil {
			devices.Fields.Add(&core.TextField{Name: "last_client_error", Max: 1024})
		}
		if devices.Fields.GetByName("last_client_delivery_at") == nil {
			devices.Fields.Add(&core.DateField{Name: "last_client_delivery_at"})
		}
		if devices.Fields.GetByName("heartbeat_grace_until") == nil {
			devices.Fields.Add(&core.DateField{Name: "heartbeat_grace_until"})
		}
		devices.AddIndex("idx_relay_devices_enabled_seen", false, "enabled,last_seen_at", "")
		if err := app.Save(devices); err != nil {
			return err
		}
		// Backfill the enrollment boundary for already paired v0.2 devices.
		// Existing records were created only by the enrollment endpoint.
		records, err := app.FindRecordsByFilter("relay_devices", "enrolled_at = ''", "created", 0, 0)
		if err != nil {
			return err
		}
		migrationNow := time.Now().UTC()
		legacyGraceUntil := migrationNow.Add(48 * time.Hour)
		legacyActiveCutoff := migrationNow.Add(-24 * time.Hour)
		for _, record := range records {
			created := record.GetDateTime("created").Time()
			if !created.IsZero() {
				record.Set("enrolled_at", created)
			}
			lastSeen := record.GetDateTime("last_seen_at").Time()
			if record.GetBool("enabled") &&
				record.GetDateTime("last_heartbeat_at").Time().IsZero() &&
				!lastSeen.IsZero() && !lastSeen.Before(legacyActiveCutoff) {
				record.Set("heartbeat_grace_until", legacyGraceUntil)
			}
			if err := app.Save(record); err != nil {
				return err
			}
		}

		for _, name := range []string{"notification_events", "relay_events"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("raw_redacted_at") == nil {
				collection.Fields.Add(&core.DateField{Name: "raw_redacted_at"})
			}
			if name == "relay_events" {
				collection.AddIndex("idx_relay_events_device_created", false, "device,created", "")
				collection.AddIndex("idx_relay_events_device_match_created", false, "device,matched_payment,created", "")
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		if devices, err := app.FindCollectionByNameOrId("relay_devices"); err == nil {
			devices.RemoveIndex("idx_relay_devices_enabled_seen")
			for _, name := range []string{"enrolled_at", "last_heartbeat_at", "notification_access", "listener_connected", "pending_count", "failed_count", "last_client_error", "last_client_delivery_at", "heartbeat_grace_until"} {
				devices.Fields.RemoveByName(name)
			}
			if err := app.Save(devices); err != nil {
				return err
			}
		}
		for _, name := range []string{"notification_events", "relay_events"} {
			if collection, err := app.FindCollectionByNameOrId(name); err == nil {
				collection.Fields.RemoveByName("raw_redacted_at")
				if name == "relay_events" {
					collection.RemoveIndex("idx_relay_events_device_created")
					collection.RemoveIndex("idx_relay_events_device_match_created")
				}
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func float64Ptr(value float64) *float64 { return &value }
