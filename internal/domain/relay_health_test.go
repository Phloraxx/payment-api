package domain

import (
	"testing"
	"time"
)

func TestRelayDeviceHealthRequiresPowerStateFromV031(t *testing.T) {
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	base := RelayDeviceHealth{
		Enabled: true, AppVersion: "0.3.1", LastSeenAt: now,
		LastHeartbeatAt: now, NotificationAccess: true, ListenerConnected: true,
		PowerHealthReported: true, BatteryOptimizationExempt: true, ForegroundServiceActive: true,
	}
	if !base.Ready(now, time.Hour) {
		t.Fatal("healthy v0.3.1 relay should be ready")
	}
	base.BatteryOptimizationExempt = false
	if base.Ready(now, time.Hour) {
		t.Fatal("battery-optimized v0.3.1 relay must fail closed")
	}
	base.AppVersion = "0.3.0"
	if !base.Ready(now, time.Hour) {
		t.Fatal("v0.3.0 compatibility should not require power-health fields")
	}
}
func TestRelayDeviceHealthGraceAndStaleness(t *testing.T) {
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	grace := RelayDeviceHealth{Enabled: true, HeartbeatGraceUntil: now.Add(time.Minute)}
	if !grace.Ready(now, time.Hour) {
		t.Fatal("bounded legacy grace should remain ready")
	}
	if grace.Ready(now.Add(2*time.Minute), time.Hour) {
		t.Fatal("expired legacy grace must fail closed")
	}
	current := RelayDeviceHealth{
		Enabled: true, AppVersion: "0.3.1", LastHeartbeatAt: now.Add(-2 * time.Hour), LastSeenAt: now.Add(-2 * time.Hour),
		NotificationAccess: true, ListenerConnected: true, PowerHealthReported: true,
		BatteryOptimizationExempt: true, ForegroundServiceActive: true,
	}
	if current.Ready(now, time.Hour) {
		t.Fatal("stale relay must not be ready")
	}
}

func TestRelayDeviceHealthPowerTelemetryCutoverGrace(t *testing.T) {
	now := time.Date(2026, 8, 29, 13, 30, 0, 0, time.UTC)
	health := RelayDeviceHealth{
		Enabled: true, AppVersion: "0.3.1", LastHeartbeatAt: now.Add(-time.Minute),
		LastSeenAt: now.Add(-time.Minute), HeartbeatGraceUntil: now.Add(2 * time.Hour),
		NotificationAccess: true, ListenerConnected: true,
	}
	if !health.Ready(now, time.Hour) {
		t.Fatal("fresh pre-power-telemetry heartbeat should receive bounded cutover grace")
	}
	health.NotificationAccess = false
	if health.Ready(now, time.Hour) {
		t.Fatal("cutover grace must still require notification access")
	}
	health.NotificationAccess = true
	if health.Ready(now.Add(2*time.Hour+time.Second), time.Hour) {
		t.Fatal("expired power telemetry grace must fail closed")
	}
}
