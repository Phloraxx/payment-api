package androidrelay

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
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

func (s *Service) Heartbeat(device *domain.RelayDevice, in HeartbeatInput) (HeartbeatResult, error) {
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
	var result HeartbeatResult
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		current, err := uow.Relay().Get(device.ID)
		if err != nil || current == nil || !current.Enabled {
			return domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
		}
		if in.LastSuccessfulDeliveryAtMs > 0 {
			lastDelivery := time.UnixMilli(in.LastSuccessfulDeliveryAtMs)
			if lastDelivery.After(now.Add(5 * time.Minute)) {
				return domain.New("INVALID_RELAY_HEARTBEAT", "last successful delivery time is in the future", 400)
			}
			current.LastClientDeliveryAt = lastDelivery
		}
		current.AppVersion = trimMax(in.AppVersion, 64)
		current.AndroidVersion = trimMax(in.AndroidVersion, 64)
		current.DeviceModel = trimMax(in.DeviceModel, 255)
		current.NotificationAccess = in.NotificationAccess
		current.ListenerConnected = in.ListenerConnected
		if in.BatteryOptimizationExempt != nil {
			current.BatteryOptimizationExempt = *in.BatteryOptimizationExempt
		}
		if in.PowerSaveMode != nil {
			current.PowerSaveMode = *in.PowerSaveMode
		}
		if in.BackgroundRestricted != nil {
			current.BackgroundRestricted = *in.BackgroundRestricted
		}
		if in.ForegroundService != nil {
			current.ForegroundServiceActive = *in.ForegroundService
		}
		current.PowerHealthReported = in.BatteryOptimizationExempt != nil && in.PowerSaveMode != nil && in.BackgroundRestricted != nil && in.ForegroundService != nil
		current.PendingCount = in.PendingCount
		current.FailedCount = in.FailedCount
		current.LastClientError = trimMax(in.LastClientError, 1024)
		current.LastHeartbeatAt = now
		current.LastSeenAt = now
		if err := uow.Relay().Save(current); err != nil {
			return err
		}
		result = HeartbeatResult{DeviceID: current.DeviceID, Enabled: current.Enabled, ServerTime: now.Format(time.RFC3339Nano)}
		return nil
	})
	return result, err
}

func (s *Service) Ready(staleAfter time.Duration) (bool, error) {
	var ready bool
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		ready, err = s.ReadyUoW(uow, staleAfter)
		return err
	})
	return ready, err
}

func (s *Service) ReadyUoW(uow store.UnitOfWork, staleAfter time.Duration) (bool, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	now := s.now()
	devices, err := uow.Relay().EnabledDevices(100)
	if err != nil {
		return false, err
	}
	for _, device := range devices {
		if device.Ready(now, staleAfter) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Status(staleAfter time.Duration) (Status, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	now := s.now()
	var status Status
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		devices, err := uow.Relay().All(100)
		if err != nil {
			return err
		}
		status.StaleAfterSeconds = int64(staleAfter / time.Second)
		for _, device := range devices {
			if !device.Enabled {
				continue
			}
			status.EnabledDevices++
			if status.LastSeenAt == nil && !device.LastSeenAt.IsZero() {
				status.LastSeenAt = timeValue(device.LastSeenAt)
			}
			if status.LastHeartbeatAt == nil && !device.LastHeartbeatAt.IsZero() {
				status.LastHeartbeatAt = timeValue(device.LastHeartbeatAt)
			}
			health := device.Health()
			if health.LegacyGraceActive(now) {
				status.ActiveDevices++
				status.LegacyGraceDevices++
			} else if health.CurrentReady(now, staleAfter) {
				status.ActiveDevices++
			}
			if !health.PowerReady() {
				status.PowerUnhealthyDevices++
			}
			status.PendingQueueCount += device.PendingCount
			status.FailedQueueCount += device.FailedCount
		}
		status.Ready = status.ActiveDevices > 0
		if latest, err := uow.RelayEvents().Latest(""); err == nil {
			status.LastEventAt = timeValue(latest.CreatedAt)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if matched, err := uow.RelayEvents().LatestMatched(""); err == nil {
			status.LastMatchedAt = timeValue(matched.CreatedAt)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		count, err := uow.RelayEvents().CountErrorsSince("", now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		status.RecentErrorCount = count
		return nil
	})
	return status, err
}

func (s *Service) Devices(staleAfter time.Duration) ([]DeviceStatus, error) {
	staleAfter = normalizeStaleAfter(staleAfter)
	now := s.now()
	var result []DeviceStatus
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		devices, err := uow.Relay().All(100)
		if err != nil {
			return err
		}
		result = make([]DeviceStatus, 0, len(devices))
		for _, device := range devices {
			stats, err := relayEventStatus(uow, device.ID, now)
			if err != nil {
				return err
			}
			health := device.Health()
			result = append(result, DeviceStatus{ID: device.ID, DeviceID: device.DeviceID, Name: device.Name, Enabled: device.Enabled, AppVersion: device.AppVersion, AndroidVersion: device.AndroidVersion, DeviceModel: device.DeviceModel, LastSeenAt: timeValue(device.LastSeenAt), LastHeartbeatAt: timeValue(device.LastHeartbeatAt), HeartbeatGraceUntil: timeValue(device.HeartbeatGraceUntil), NotificationAccess: device.NotificationAccess, ListenerConnected: device.ListenerConnected, PowerHealthReported: device.PowerHealthReported, BatteryOptimizationExempt: device.BatteryOptimizationExempt, PowerSaveMode: device.PowerSaveMode, BackgroundRestricted: device.BackgroundRestricted, ForegroundService: device.ForegroundServiceActive, PowerHealthy: health.PowerReady(), PendingCount: device.PendingCount, FailedCount: device.FailedCount, LastClientError: device.LastClientError, LastDeliveryAt: timeValue(device.LastClientDeliveryAt), LastEventAt: timeValue(stats.LastEventAt), LastMatchedAt: timeValue(stats.LastMatchedAt), LastMatchedPaymentID: stats.LastMatchedPaymentID, RecentErrorCount: stats.RecentErrorCount, Active: health.Ready(now, staleAfter)})
		}
		return nil
	})
	return result, err
}

func relayEventStatus(uow store.UnitOfWork, deviceRecordID string, now time.Time) (domain.RelayEventStats, error) {
	var stats domain.RelayEventStats
	if latest, err := uow.RelayEvents().Latest(deviceRecordID); err == nil {
		stats.LastEventAt = latest.CreatedAt
	} else if !errors.Is(err, sql.ErrNoRows) {
		return stats, err
	}
	if matched, err := uow.RelayEvents().LatestMatched(deviceRecordID); err == nil {
		stats.LastMatchedAt = matched.CreatedAt
		stats.LastMatchedPaymentID = matched.MatchedPaymentID
	} else if !errors.Is(err, sql.ErrNoRows) {
		return stats, err
	}
	count, err := uow.RelayEvents().CountErrorsSince(deviceRecordID, now.Add(-24*time.Hour))
	if err != nil {
		return stats, err
	}
	stats.RecentErrorCount = count
	return stats, nil
}

func (s *Service) SetEnabled(recordID string, enabled bool) (*domain.RelayDevice, error) {
	var result *domain.RelayDevice
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		result, err = s.SetEnabledUoW(uow, recordID, enabled)
		return err
	})
	return result, err
}

func (s *Service) SetEnabledUoW(uow store.UnitOfWork, recordID string, enabled bool) (*domain.RelayDevice, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return nil, domain.New("INVALID_RELAY_DEVICE_ID", "relay device id is required", 400)
	}
	device, err := uow.Relay().Get(recordID)
	if err != nil {
		return nil, domain.New("RELAY_DEVICE_NOT_FOUND", "relay device was not found", 404)
	}
	if device.Enabled != enabled {
		device.HeartbeatGraceUntil = time.Time{}
	}
	device.Enabled = enabled
	if err := uow.Relay().Save(device); err != nil {
		return nil, err
	}
	return device, nil
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
