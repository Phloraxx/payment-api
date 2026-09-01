package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const HeartbeatPath = "/api/v4/relay/heartbeat"

type HeartbeatInput struct {
	SchemaVersion             int    `json:"schema_version"`
	AppVersion                string `json:"app_version,omitempty"`
	AndroidVersion            string `json:"android_version,omitempty"`
	DeviceModel               string `json:"device_model,omitempty"`
	NotificationAccess        bool   `json:"notification_access"`
	ListenerConnected         bool   `json:"listener_connected"`
	BatteryOptimizationExempt bool   `json:"battery_optimization_exempt"`
	PowerSaveMode             bool   `json:"power_save_mode"`
	BackgroundRestricted      bool   `json:"background_restricted"`
	ForegroundService         bool   `json:"foreground_service"`
	PendingCount              int    `json:"pending_count"`
	FailedCount               int    `json:"failed_count"`
	LastSuccessfulDeliveryMS  int64  `json:"last_successful_delivery_at_ms,omitempty"`
	LastClientError           string `json:"last_client_error,omitempty"`
}
type HeartbeatResult struct {
	ReceivedAt time.Time `json:"received_at"`
}

func (s *Service) HeartbeatSigned(ctx context.Context, auth RequestAuth, rawBody []byte) (HeartbeatResult, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return HeartbeatResult{}, errors.New("relay storage is required")
	}
	if len(rawBody) == 0 || len(rawBody) > maxRawBodyBytes {
		return HeartbeatResult{}, relayError("RELAY_HEARTBEAT_TOO_LARGE", "relay heartbeat body is empty or too large", 400)
	}
	if strings.ToUpper(strings.TrimSpace(auth.Method)) != "POST" || auth.Path != HeartbeatPath {
		return HeartbeatResult{}, relayError("INVALID_RELAY_ENDPOINT", "relay signature is not for the v4 heartbeat endpoint", 401)
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	device, err := verifyRequest(ctx, s.DB, auth, rawBody, now)
	if err != nil {
		return HeartbeatResult{}, err
	}
	var input HeartbeatInput
	if err := json.Unmarshal(rawBody, &input); err != nil {
		return HeartbeatResult{}, relayError("INVALID_RELAY_HEARTBEAT", "relay heartbeat is not valid JSON", 400)
	}
	if input.SchemaVersion != SchemaVersion {
		return HeartbeatResult{}, relayError("UNSUPPORTED_RELAY_SCHEMA", "schema_version must be 1", 400)
	}
	input.AppVersion = trimRunes(input.AppVersion, 64)
	input.AndroidVersion = trimRunes(input.AndroidVersion, 64)
	input.DeviceModel = trimRunes(input.DeviceModel, 255)
	input.LastClientError = trimRunes(input.LastClientError, 512)
	if input.PendingCount < 0 || input.PendingCount > 1_000_000 || input.FailedCount < 0 || input.FailedCount > 1_000_000 {
		return HeartbeatResult{}, relayError("INVALID_RELAY_HEARTBEAT", "queue counts are outside the allowed range", 400)
	}
	var delivered any
	if input.LastSuccessfulDeliveryMS > 0 {
		t := time.UnixMilli(input.LastSuccessfulDeliveryMS).UTC()
		if t.After(now.Add(5*time.Minute)) || t.Year() < 2020 {
			return HeartbeatResult{}, relayError("INVALID_RELAY_HEARTBEAT", "last successful delivery time is invalid", 400)
		}
		delivered = t.UnixMilli()
	}
	result, err := s.DB.SQL.ExecContext(ctx, `UPDATE relay_devices SET
		last_seen_at=?,last_heartbeat_at=?,app_version=?,device_model=?,android_version=?,
		notification_access=?,listener_connected=?,battery_optimization_exempt=?,power_save_mode=?,
		background_restricted=?,foreground_service=?,pending_count=?,failed_count=?,last_successful_delivery_at=?,last_client_error=?
		WHERE id=? AND enabled=1`,
		now.UnixMilli(), now.UnixMilli(), nullableText(input.AppVersion), nullableText(input.DeviceModel), nullableText(input.AndroidVersion),
		boolInt(input.NotificationAccess), boolInt(input.ListenerConnected), boolInt(input.BatteryOptimizationExempt), boolInt(input.PowerSaveMode),
		boolInt(input.BackgroundRestricted), boolInt(input.ForegroundService), input.PendingCount, input.FailedCount,
		delivered, nullableText(input.LastClientError), device.ID)
	if err != nil {
		return HeartbeatResult{}, fmt.Errorf("persist relay heartbeat: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return HeartbeatResult{}, relayError("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	return HeartbeatResult{ReceivedAt: now}, nil
}
