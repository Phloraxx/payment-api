package domain

import (
	"fmt"
	"strings"
	"time"
)

type RelayDeviceHealth struct {
	Enabled                   bool
	AppVersion                string
	LastSeenAt                time.Time
	LastHeartbeatAt           time.Time
	HeartbeatGraceUntil       time.Time
	NotificationAccess        bool
	ListenerConnected         bool
	PowerHealthReported       bool
	BatteryOptimizationExempt bool
	BackgroundRestricted      bool
	ForegroundServiceActive   bool
}

func (h RelayDeviceHealth) LegacyGraceActive(now time.Time) bool {
	return h.Enabled && h.LastHeartbeatAt.IsZero() && !h.HeartbeatGraceUntil.IsZero() && now.Before(h.HeartbeatGraceUntil)
}
func (h RelayDeviceHealth) PowerReady() bool {
	if !relayPowerHealthRequired(h.AppVersion) {
		return true
	}
	return h.PowerHealthReported && h.BatteryOptimizationExempt && !h.BackgroundRestricted && h.ForegroundServiceActive
}

func (h RelayDeviceHealth) CurrentReady(now time.Time, staleAfter time.Duration) bool {
	if !h.Enabled || h.LastHeartbeatAt.IsZero() {
		return false
	}
	if staleAfter <= 0 {
		staleAfter = time.Hour
	}
	if h.LastSeenAt.IsZero() || h.LastSeenAt.Before(now.Add(-staleAfter)) {
		return false
	}
	return h.NotificationAccess && h.ListenerConnected && h.PowerReady()
}

func (h RelayDeviceHealth) Ready(now time.Time, staleAfter time.Duration) bool {
	return h.LegacyGraceActive(now) || h.CurrentReady(now, staleAfter)
}
func relayPowerHealthRequired(version string) bool {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "v"))
	version = strings.SplitN(version, "-", 2)[0]
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major, minor, patch := 0, 0, 0
	if _, err := fmt.Sscanf(parts[0]+"."+parts[1]+"."+parts[2], "%d.%d.%d", &major, &minor, &patch); err != nil {
		return false
	}
	if major != 0 {
		return major > 0
	}
	if minor != 3 {
		return minor > 3
	}
	return patch >= 1
}
