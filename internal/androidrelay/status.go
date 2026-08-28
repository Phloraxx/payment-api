package androidrelay

import (
	"fmt"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const defaultStaleAfter = time.Hour

type HeartbeatInput struct {
	SchemaVersion              int    `json:"schemaVersion"`
	AppVersion                 string `json:"appVersion"`
	AndroidVersion             string `json:"androidVersion"`
	DeviceModel                string `json:"deviceModel"`
	NotificationAccess         bool   `json:"notificationAccess"`
	ListenerConnected          bool   `json:"listenerConnected"`
	BatteryOptimizationExempt  *bool  `json:"batteryOptimizationExempt"`
	PowerSaveMode              *bool  `json:"powerSaveMode"`
	BackgroundRestricted       *bool  `json:"backgroundRestricted"`
	ForegroundService          *bool  `json:"foregroundService"`
	PendingCount               int    `json:"pendingCount"`
	FailedCount                int    `json:"failedCount"`
	LastSuccessfulDeliveryAtMs int64  `json:"lastSuccessfulDeliveryAtMs"`
	LastClientError            string `json:"lastClientError"`
}

type HeartbeatResult struct {
	DeviceID   string `json:"deviceId"`
	Enabled    bool   `json:"enabled"`
	ServerTime string `json:"serverTime"`
}

type DeviceStatus struct {
	ID                        string `json:"id"`
	DeviceID                  string `json:"deviceId"`
	Name                      string `json:"name"`
	Enabled                   bool   `json:"enabled"`
	AppVersion                string `json:"appVersion"`
	AndroidVersion            string `json:"androidVersion"`
	DeviceModel               string `json:"deviceModel"`
	LastSeenAt                any    `json:"lastSeenAt"`
	LastHeartbeatAt           any    `json:"lastHeartbeatAt"`
	HeartbeatGraceUntil       any    `json:"heartbeatGraceUntil"`
	NotificationAccess        bool   `json:"notificationAccess"`
	ListenerConnected         bool   `json:"listenerConnected"`
	PowerHealthReported       bool   `json:"powerHealthReported"`
	BatteryOptimizationExempt bool   `json:"batteryOptimizationExempt"`
	PowerSaveMode             bool   `json:"powerSaveMode"`
	BackgroundRestricted      bool   `json:"backgroundRestricted"`
	ForegroundService         bool   `json:"foregroundService"`
	PowerHealthy              bool   `json:"powerHealthy"`
	PendingCount              int    `json:"pendingCount"`
	FailedCount               int    `json:"failedCount"`
	LastClientError           string `json:"lastClientError,omitempty"`
	LastDeliveryAt            any    `json:"lastDeliveryAt"`
	LastEventAt               any    `json:"lastEventAt"`
	LastMatchedAt             any    `json:"lastMatchedAt"`
	LastMatchedPaymentID      string `json:"lastMatchedPaymentId,omitempty"`
	RecentErrorCount          int64  `json:"recentErrorCount"`
	Active                    bool   `json:"active"`
}

type Status struct {
	Ready                 bool  `json:"ready"`
	EnabledDevices        int   `json:"enabledDevices"`
	ActiveDevices         int   `json:"activeDevices"`
	LegacyGraceDevices    int   `json:"legacyGraceDevices"`
	StaleAfterSeconds     int64 `json:"staleAfterSeconds"`
	LastSeenAt            any   `json:"lastSeenAt"`
	LastHeartbeatAt       any   `json:"lastHeartbeatAt"`
	LastEventAt           any   `json:"lastEventAt"`
	LastMatchedAt         any   `json:"lastMatchedAt"`
	RecentErrorCount      int64 `json:"recentErrorCount"`
	PendingQueueCount     int   `json:"pendingQueueCount"`
	FailedQueueCount      int   `json:"failedQueueCount"`
	PowerUnhealthyDevices int   `json:"powerUnhealthyDevices"`
}

func (s *Service) Heartbeat(device *core.Record, in HeartbeatInput) (HeartbeatResult, error) {
	if device == nil {
		return HeartbeatResult{}, domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	if in.SchemaVersion != 1 {
		return HeartbeatResult{}, domain.New("UNSUPPORTED_RELAY_SCHEMA", "schemaVersion must be 1", 400)
	}
	if in.PendingCount < 0 || in.PendingCount > 100000 || in.FailedCount < 0 || in.FailedCount > 100000 {
		return HeartbeatResult{}, domain.New("INVALID_RELAY_HEARTBEAT", "relay queue counts are invalid", 400)
	}
	if in.LastSuccessfulDeliveryAtMs < 0 {
		return HeartbeatResult{}, domain.New("INVALID_RELAY_HEARTBEAT", "last successful delivery time is invalid", 400)
	}
	now := s.now()
	if in.LastSuccessfulDeliveryAtMs > 0 {
		lastDelivery := time.UnixMilli(in.LastSuccessfulDeliveryAtMs)
		if lastDelivery.After(now.Add(5 * time.Minute)) {
			return HeartbeatResult{}, domain.New("INVALID_RELAY_HEARTBEAT", "last successful delivery time is in the future", 400)
		}
		device.Set("last_client_delivery_at", lastDelivery)
	}
	device.Set("app_version", trimMax(in.AppVersion, 64))
	device.Set("android_version", trimMax(in.AndroidVersion, 64))
	device.Set("device_model", trimMax(in.DeviceModel, 255))
	device.Set("notification_access", in.NotificationAccess)
	device.Set("listener_connected", in.ListenerConnected)
	if in.BatteryOptimizationExempt != nil {
		device.Set("battery_optimization_exempt", *in.BatteryOptimizationExempt)
	}
	if in.PowerSaveMode != nil {
		device.Set("power_save_mode", *in.PowerSaveMode)
	}
	if in.BackgroundRestricted != nil {
		device.Set("background_restricted", *in.BackgroundRestricted)
	}
	if in.ForegroundService != nil {
		device.Set("foreground_service_active", *in.ForegroundService)
	}
	device.Set("power_health_reported", in.BatteryOptimizationExempt != nil && in.PowerSaveMode != nil && in.BackgroundRestricted != nil && in.ForegroundService != nil)
	device.Set("pending_count", in.PendingCount)
	device.Set("failed_count", in.FailedCount)
	device.Set("last_client_error", trimMax(in.LastClientError, 1024))
	device.Set("last_heartbeat_at", now)
	device.Set("last_seen_at", now)
	if err := s.App.Save(device); err != nil {
		return HeartbeatResult{}, err
	}
	return HeartbeatResult{DeviceID: device.GetString("device_id"), Enabled: device.GetBool("enabled"), ServerTime: now.Format(time.RFC3339Nano)}, nil
}

func (s *Service) Ready(staleAfter time.Duration) (bool, error) {
	return s.ReadyInApp(s.App, staleAfter)
}

func (s *Service) ReadyInApp(app core.App, staleAfter time.Duration) (bool, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	now := s.now()
	cutoff := now.Add(-staleAfter)
	devices, err := app.FindRecordsByFilter("relay_devices", "enabled = true", "-last_seen_at", 100, 0)
	if err != nil {
		return false, err
	}
	for _, device := range devices {
		heartbeat := device.GetDateTime("last_heartbeat_at").Time()
		if heartbeat.IsZero() {
			graceUntil := device.GetDateTime("heartbeat_grace_until").Time()
			if !graceUntil.IsZero() && now.Before(graceUntil) {
				return true, nil
			}
			continue
		}
		if relayDeviceCurrentReady(device, cutoff) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Status(staleAfter time.Duration) (Status, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	now := s.now()
	cutoff := now.Add(-staleAfter)
	devices, err := s.App.FindRecordsByFilter("relay_devices", "enabled = true", "-last_seen_at", 0, 0)
	if err != nil {
		return Status{}, err
	}
	status := Status{EnabledDevices: len(devices), StaleAfterSeconds: int64(staleAfter / time.Second)}
	for _, device := range devices {
		seen := device.GetDateTime("last_seen_at").Time()
		heartbeat := device.GetDateTime("last_heartbeat_at").Time()
		if status.LastSeenAt == nil && !seen.IsZero() {
			status.LastSeenAt = timeValue(seen)
		}
		if status.LastHeartbeatAt == nil && !heartbeat.IsZero() {
			status.LastHeartbeatAt = timeValue(heartbeat)
		}
		// Existing v0.2 devices get a bounded migration grace so API deployment
		// cannot strand checkout before the phone is upgraded. New enrollments do
		// not get this grace. Once any heartbeat is received, normal stale/listener
		// readiness applies immediately.
		if heartbeat.IsZero() {
			graceUntil := device.GetDateTime("heartbeat_grace_until").Time()
			if !graceUntil.IsZero() && now.Before(graceUntil) {
				status.ActiveDevices++
				status.LegacyGraceDevices++
			}
		} else if relayDeviceCurrentReady(device, cutoff) {
			status.ActiveDevices++
		}
		if !relayDevicePowerReady(device) {
			status.PowerUnhealthyDevices++
		}
		status.PendingQueueCount += device.GetInt("pending_count")
		status.FailedQueueCount += device.GetInt("failed_count")
	}
	status.Ready = status.ActiveDevices > 0

	if latest, findErr := s.App.FindRecordsByFilter("relay_events", "", "-created", 1, 0); findErr == nil && len(latest) == 1 {
		status.LastEventAt = timeValue(latest[0].GetDateTime("created").Time())
	} else if findErr != nil {
		return Status{}, findErr
	}
	if matched, findErr := s.App.FindRecordsByFilter("relay_events", "matched_payment != ''", "-created", 1, 0); findErr == nil && len(matched) == 1 {
		status.LastMatchedAt = timeValue(matched[0].GetDateTime("created").Time())
	} else if findErr != nil {
		return Status{}, findErr
	}
	errorCutoff := filterDate(now.Add(-24 * time.Hour))
	errorCount, err := s.App.CountRecords("relay_events", dbx.NewExp("processing_status = 'error' AND created >= {:cutoff}", dbx.Params{"cutoff": errorCutoff}))
	if err != nil {
		return Status{}, err
	}
	status.RecentErrorCount = errorCount
	return status, nil
}

func (s *Service) Devices(staleAfter time.Duration) ([]DeviceStatus, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	cutoff := s.now().Add(-staleAfter)
	records, err := s.App.FindRecordsByFilter("relay_devices", "", "-last_seen_at,-created", 100, 0)
	if err != nil {
		return nil, err
	}
	result := make([]DeviceStatus, 0, len(records))
	for _, record := range records {
		seen := record.GetDateTime("last_seen_at").Time()
		heartbeat := record.GetDateTime("last_heartbeat_at").Time()
		graceUntil := record.GetDateTime("heartbeat_grace_until").Time()
		legacyGraceActive := heartbeat.IsZero() && !graceUntil.IsZero() && s.now().Before(graceUntil)
		powerHealthy := relayDevicePowerReady(record)
		lastEventAt, lastMatchedAt, lastMatchedPaymentID, recentErrorCount, statusErr := s.deviceEventStatus(record.Id, s.now())
		if statusErr != nil {
			return nil, statusErr
		}
		result = append(result, DeviceStatus{
			ID: record.Id, DeviceID: record.GetString("device_id"), Name: record.GetString("name"), Enabled: record.GetBool("enabled"),
			AppVersion: record.GetString("app_version"), AndroidVersion: record.GetString("android_version"), DeviceModel: record.GetString("device_model"),
			LastSeenAt: timeValue(seen), LastHeartbeatAt: timeValue(heartbeat), HeartbeatGraceUntil: timeValue(graceUntil), NotificationAccess: record.GetBool("notification_access"), ListenerConnected: record.GetBool("listener_connected"),
			PowerHealthReported: record.GetBool("power_health_reported"), BatteryOptimizationExempt: record.GetBool("battery_optimization_exempt"), PowerSaveMode: record.GetBool("power_save_mode"), BackgroundRestricted: record.GetBool("background_restricted"), ForegroundService: record.GetBool("foreground_service_active"), PowerHealthy: powerHealthy,
			PendingCount: record.GetInt("pending_count"), FailedCount: record.GetInt("failed_count"), LastClientError: record.GetString("last_client_error"),
			LastDeliveryAt: timeValue(record.GetDateTime("last_client_delivery_at").Time()),
			LastEventAt:    lastEventAt, LastMatchedAt: lastMatchedAt, LastMatchedPaymentID: lastMatchedPaymentID, RecentErrorCount: recentErrorCount,
			Active: record.GetBool("enabled") && (legacyGraceActive || relayDeviceCurrentReady(record, cutoff)),
		})
	}
	return result, nil
}

func (s *Service) deviceEventStatus(deviceRecordID string, now time.Time) (lastEventAt any, lastMatchedAt any, lastMatchedPaymentID string, recentErrorCount int64, err error) {
	params := dbx.Params{"device": deviceRecordID}
	latest, err := s.App.FindRecordsByFilter("relay_events", "device = {:device}", "-created", 1, 0, params)
	if err != nil {
		return nil, nil, "", 0, err
	}
	if len(latest) == 1 {
		lastEventAt = timeValue(latest[0].GetDateTime("created").Time())
	}
	matched, err := s.App.FindRecordsByFilter("relay_events", "device = {:device} && matched_payment != ''", "-created", 1, 0, params)
	if err != nil {
		return nil, nil, "", 0, err
	}
	if len(matched) == 1 {
		lastMatchedAt = timeValue(matched[0].GetDateTime("created").Time())
		lastMatchedPaymentID = matched[0].GetString("matched_payment")
	}
	errorCutoff := filterDate(now.Add(-24 * time.Hour))
	recentErrorCount, err = s.App.CountRecords("relay_events", dbx.NewExp("device = {:device} AND processing_status = 'error' AND created >= {:cutoff}", dbx.Params{"device": deviceRecordID, "cutoff": errorCutoff}))
	if err != nil {
		return nil, nil, "", 0, err
	}
	return lastEventAt, lastMatchedAt, lastMatchedPaymentID, recentErrorCount, nil
}

func (s *Service) SetEnabled(recordID string, enabled bool) (*core.Record, error) {
	return s.SetEnabledInApp(s.App, recordID, enabled)
}

func (s *Service) SetEnabledInApp(app core.App, recordID string, enabled bool) (*core.Record, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return nil, domain.New("INVALID_RELAY_DEVICE_ID", "relay device id is required", 400)
	}
	record, err := app.FindRecordById("relay_devices", recordID)
	if err != nil {
		return nil, domain.New("RELAY_DEVICE_NOT_FOUND", "relay device was not found", 404)
	}
	if record.GetBool("enabled") != enabled {
		// Operator state changes revoke migration-only compatibility. A device
		// that is explicitly disabled and later re-enabled must prove current
		// health with a v0.3 heartbeat rather than inheriting old rollout grace.
		record.Set("heartbeat_grace_until", "")
	}
	record.Set("enabled", enabled)
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func relayDeviceCurrentReady(record *core.Record, cutoff time.Time) bool {
	if record == nil || record.GetDateTime("last_heartbeat_at").Time().IsZero() {
		return false
	}
	seen := record.GetDateTime("last_seen_at").Time()
	return !seen.IsZero() && !seen.Before(cutoff) &&
		record.GetBool("notification_access") && record.GetBool("listener_connected") &&
		relayDevicePowerReady(record)
}

func relayDevicePowerReady(record *core.Record) bool {
	if record == nil || !requiresPowerHealth(record.GetString("app_version")) {
		return true
	}
	return record.GetBool("power_health_reported") &&
		record.GetBool("battery_optimization_exempt") &&
		!record.GetBool("background_restricted") &&
		record.GetBool("foreground_service_active")
}

func requiresPowerHealth(version string) bool {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "v"))
	parts := strings.SplitN(version, "-", 2)
	version = parts[0]
	numbers := strings.Split(version, ".")
	if len(numbers) < 3 {
		return false
	}
	major, minor, patch := 0, 0, 0
	if _, err := fmt.Sscanf(numbers[0]+"."+numbers[1]+"."+numbers[2], "%d.%d.%d", &major, &minor, &patch); err != nil {
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

func normalizeStaleAfter(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultStaleAfter
	}
	return value
}

func timeValue(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func filterDate(value time.Time) string {
	parsed, err := types.ParseDateTime(value.UTC())
	if err != nil {
		return value.UTC().Format(time.RFC3339Nano)
	}
	return parsed.String()
}
