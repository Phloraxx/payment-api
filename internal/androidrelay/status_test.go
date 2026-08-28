package androidrelay

import (
	"testing"
	"time"

	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRelayStatusAndHeartbeatTrackReadiness(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }

	status, err := service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if status.Ready || status.EnabledDevices != 0 || status.ActiveDevices != 0 {
		t.Fatalf("unexpected empty status: %+v", status)
	}

	collection, err := app.FindCollectionByNameOrId("relay_devices")
	if err != nil {
		t.Fatal(err)
	}
	device := core.NewRecord(collection)
	device.Set("device_id", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	device.Set("name", "Test phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now)
	device.Set("heartbeat_grace_until", now.Add(48*time.Hour))
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}

	status, err = service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ActiveDevices != 1 {
		t.Fatalf("legacy heartbeat grace should be ready: %+v", status)
	}

	if _, err := service.Heartbeat(typedRelayDevice(t, service, device.Id), HeartbeatInput{SchemaVersion: 1, AppVersion: "0.3.0", NotificationAccess: false}); err != nil {
		t.Fatal(err)
	}
	status, _ = service.Status(time.Hour)
	if status.Ready {
		t.Fatalf("heartbeat without notification access must make relay unavailable: %+v", status)
	}

	if _, err := service.Heartbeat(typedRelayDevice(t, service, device.Id), HeartbeatInput{SchemaVersion: 1, AppVersion: "0.3.0", NotificationAccess: true, ListenerConnected: true, PendingCount: 2, FailedCount: 1, LastSuccessfulDeliveryAtMs: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	status, _ = service.Status(time.Hour)
	if !status.Ready || status.ActiveDevices != 1 || status.LastHeartbeatAt == nil || status.PendingQueueCount != 2 || status.FailedQueueCount != 1 {
		t.Fatalf("heartbeat should restore readiness: %+v", status)
	}
	devices, err := service.Devices(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || !devices[0].Active || !devices[0].ListenerConnected || devices[0].PendingCount != 2 || devices[0].FailedCount != 1 || devices[0].LastDeliveryAt == nil {
		t.Fatalf("device status = %+v", devices)
	}
	if devices[0].RecentErrorCount != 0 || devices[0].LastEventAt != nil || devices[0].LastMatchedAt != nil {
		t.Fatalf("empty device event status = %+v", devices[0])
	}
}

func TestLegacyHeartbeatGraceAllowsIdleMigratedDevice(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	device.Set("name", "Legacy phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now.Add(-24*time.Hour))
	device.Set("heartbeat_grace_until", now.Add(time.Hour))
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	ready, err := service.Ready(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("migrated v0.2 device should remain ready during bounded heartbeat grace")
	}
	status, err := service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.LegacyGraceDevices != 1 {
		t.Fatalf("legacy status = %+v", status)
	}
}

func TestLegacyHeartbeatGraceExpiresFailClosed(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	device.Set("name", "Expired legacy phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now.Add(-24*time.Hour))
	device.Set("heartbeat_grace_until", now)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	ready, err := service.Ready(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("legacy relay must fail closed at the grace boundary")
	}
	status, err := service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if status.Ready || status.LegacyGraceDevices != 0 || status.ActiveDevices != 0 {
		t.Fatalf("expired legacy status = %+v", status)
	}
}

func TestRelayDeviceStateChangeClearsLegacyGrace(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 4, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "acacacacacacacacacacacacacacacacacacacacacacacacacacacacacacacac")
	device.Set("name", "Grace revocation phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now.Add(-time.Hour))
	device.Set("heartbeat_grace_until", now.Add(24*time.Hour))
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	if ready, err := service.Ready(time.Hour); err != nil || !ready {
		t.Fatalf("expected legacy grace before revocation, ready=%v err=%v", ready, err)
	}
	disabled, err := service.SetEnabled(device.Id, false)
	if err != nil {
		t.Fatal(err)
	}
	if !disabled.HeartbeatGraceUntil.IsZero() {
		t.Fatal("disabling a device must clear legacy heartbeat grace")
	}
	if _, err := service.SetEnabled(device.Id, true); err != nil {
		t.Fatal(err)
	}
	ready, err := service.Ready(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("re-enabled device must require a real heartbeat after grace revocation")
	}
}

func TestHeartbeatRejectsFutureDeliveryTimestamp(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	device.Set("name", "Phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	_, err = service.Heartbeat(typedRelayDevice(t, service, device.Id), HeartbeatInput{SchemaVersion: 1, LastSuccessfulDeliveryAtMs: now.Add(6 * time.Minute).UnixMilli()})
	if err == nil {
		t.Fatal("expected future delivery timestamp to be rejected")
	}
}

func TestRelayStatusMarksStaleDeviceInactive(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	device.Set("name", "Stale phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now.Add(-2*time.Hour))
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if status.Ready || status.EnabledDevices != 1 || status.ActiveDevices != 0 {
		t.Fatalf("stale device status = %+v", status)
	}
}

func TestV031PowerHealthGatesReadinessButAllowsPowerSaver(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 7, 30, 0, 0, time.UTC)
	service := NewService(app, nil)
	service.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "abababababababababababababababababababababababababababababababab")
	device.Set("name", "Always-on phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}

	heartbeat := func(exempt, saver, restricted, foreground bool) {
		t.Helper()
		if _, err := service.Heartbeat(typedRelayDevice(t, service, device.Id), HeartbeatInput{
			SchemaVersion:             1,
			AppVersion:                "0.3.1",
			NotificationAccess:        true,
			ListenerConnected:         true,
			BatteryOptimizationExempt: boolPointer(exempt),
			PowerSaveMode:             boolPointer(saver),
			BackgroundRestricted:      boolPointer(restricted),
			ForegroundService:         boolPointer(foreground),
		}); err != nil {
			t.Fatal(err)
		}
	}

	heartbeat(true, true, false, true)
	status, err := service.Status(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ActiveDevices != 1 || status.PowerUnhealthyDevices != 0 {
		t.Fatalf("power saver should remain ready when v0.3.1 is exempt and foreground: %+v", status)
	}
	devices, err := service.Devices(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || !devices[0].PowerHealthReported || !devices[0].PowerHealthy || !devices[0].BatteryOptimizationExempt || !devices[0].PowerSaveMode || !devices[0].ForegroundService {
		t.Fatalf("unexpected power health: %+v", devices)
	}

	heartbeat(false, true, false, true)
	status, _ = service.Status(time.Hour)
	if status.Ready || status.PowerUnhealthyDevices != 1 {
		t.Fatalf("battery-optimized v0.3.1 must fail closed: %+v", status)
	}

	heartbeat(true, true, true, true)
	status, _ = service.Status(time.Hour)
	if status.Ready {
		t.Fatalf("background-restricted v0.3.1 must fail closed: %+v", status)
	}

	heartbeat(true, true, false, false)
	status, _ = service.Status(time.Hour)
	if status.Ready {
		t.Fatalf("v0.3.1 without foreground runtime must fail closed: %+v", status)
	}

	heartbeat(true, true, false, true)
	status, _ = service.Status(time.Hour)
	if !status.Ready {
		t.Fatalf("healthy always-on state should recover readiness: %+v", status)
	}
}

func boolPointer(value bool) *bool { return &value }
