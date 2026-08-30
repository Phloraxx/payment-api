package migrations

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

// Bridge already-heartbeating relay devices across the first server version
// that persists power-health telemetry. The old API accepted v0.3.1
// heartbeats but discarded those fields, so requiring them immediately after
// the server restart creates a false-negative readiness window.
func init() {
	pbmigrations.Register(func(app core.App) error {
		records, err := app.FindRecordsByFilter(
			"relay_devices",
			"enabled = true && power_health_reported = false",
			"created", 0, 0,
		)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		freshCutoff := now.Add(-time.Hour)
		graceUntil := now.Add(2 * time.Hour)
		for _, record := range records {
			lastHeartbeat := record.GetDateTime("last_heartbeat_at").Time()
			lastSeen := record.GetDateTime("last_seen_at").Time()
			if lastHeartbeat.IsZero() || lastSeen.IsZero() || lastSeen.Before(freshCutoff) {
				continue
			}
			if !record.GetBool("notification_access") || !record.GetBool("listener_connected") {
				continue
			}
			currentGrace := record.GetDateTime("heartbeat_grace_until").Time()
			if !currentGrace.IsZero() && currentGrace.After(now) {
				continue
			}
			record.Set("heartbeat_grace_until", graceUntil)
			if err := app.Save(record); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		// Operational grace is intentionally not rewound. It is bounded by time
		// and by the normal one-hour heartbeat freshness requirement.
		return nil
	})
}
