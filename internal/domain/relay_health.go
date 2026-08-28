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

type RelayDevice struct {
	ID                        string
	DeviceID                  string
	Name                      string
	PublicKeyPEM              string
	Enabled                   bool
	AppVersion                string
	AndroidVersion            string
	DeviceModel               string
	EnrolledAt                time.Time
	LastSeenAt                time.Time
	LastHeartbeatAt           time.Time
	HeartbeatGraceUntil       time.Time
	NotificationAccess        bool
	ListenerConnected         bool
	PowerHealthReported       bool
	BatteryOptimizationExempt bool
	PowerSaveMode             bool
	BackgroundRestricted      bool
	ForegroundServiceActive   bool
	PendingCount              int
	FailedCount               int
	LastClientError           string
	LastClientDeliveryAt      time.Time
	CreatedAt                 time.Time
}

func (d RelayDevice) Health() RelayDeviceHealth {
	return RelayDeviceHealth{
		Enabled: d.Enabled, AppVersion: d.AppVersion, LastSeenAt: d.LastSeenAt,
		LastHeartbeatAt: d.LastHeartbeatAt, HeartbeatGraceUntil: d.HeartbeatGraceUntil,
		NotificationAccess: d.NotificationAccess, ListenerConnected: d.ListenerConnected,
		PowerHealthReported: d.PowerHealthReported, BatteryOptimizationExempt: d.BatteryOptimizationExempt,
		BackgroundRestricted: d.BackgroundRestricted, ForegroundServiceActive: d.ForegroundServiceActive,
	}
}

type RelayEvent struct {
	ID                string
	DeviceRecordID    string
	EventID           string
	Kind              string
	AppPackage        string
	AppName           string
	NotificationKey   string
	NotificationID    int
	NotificationTag   string
	GroupKey          string
	IsGroupSummary    bool
	PostTime          time.Time
	NotificationWhen  time.Time
	CapturedAt        time.Time
	ChannelID         string
	Category          string
	Title             string
	Body              string
	BigText           string
	SubText           string
	SummaryText       string
	TextLines         []string
	CustomTexts       []string
	ProcessingStatus  string
	DownstreamEventID string
	MatchedPaymentID  string
	ProviderResult    any
	Error             string
	RawPayload        any
	CreatedAt         time.Time
}

type RelayEventStats struct {
	LastEventAt          time.Time
	LastMatchedAt        time.Time
	LastMatchedPaymentID string
	RecentErrorCount     int64
}
