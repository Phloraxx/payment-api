package migratev3

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func loadPayments(ctx context.Context, db *sql.DB) ([]legacyPayment, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,external_id,metadata,status,payment_account,customer_name,payer_name,upi_id,
		display_name,customer_email,customer_phone,description,admin_note,idempotency_key,tags,custom_fields,
		requested_amount,payable_amount,created_at,expires_at,reuse_after,paid_at,resolved_at
		FROM payments ORDER BY created_at,id`)
	if err != nil {
		return nil, fmt.Errorf("read legacy payments: %w", err)
	}
	defer rows.Close()
	var out []legacyPayment
	for rows.Next() {
		var p legacyPayment
		var metadata, tags, customFields, created, expires, reuse, paid, resolved sql.NullString
		if err := rows.Scan(&p.ID, &p.ExternalID, &metadata, &p.Status, &p.Account, &p.CustomerName, &p.PayerName, &p.PayerUPIID,
			&p.DisplayName, &p.CustomerEmail, &p.CustomerPhone, &p.Description, &p.AdminNote, &p.IdempotencyKey, &tags, &customFields,
			&p.Requested, &p.Payable, &created, &expires, &reuse, &paid, &resolved); err != nil {
			return nil, fmt.Errorf("scan legacy payment: %w", err)
		}
		p.Metadata = json.RawMessage(strings.TrimSpace(metadata.String))
		if len(p.Metadata) == 0 {
			p.Metadata = json.RawMessage(`{}`)
		}
		p.Tags = json.RawMessage(strings.TrimSpace(tags.String))
		p.CustomFields = json.RawMessage(strings.TrimSpace(customFields.String))
		if p.CreatedAt, err = parsePBTime(created.String); err != nil {
			return nil, fmt.Errorf("payment %s created_at: %w", p.ID, err)
		}
		if p.ExpiresAt, err = parsePBTime(expires.String); err != nil {
			return nil, fmt.Errorf("payment %s expires_at: %w", p.ID, err)
		}
		if p.ReuseAfter, err = parsePBTime(reuse.String); err != nil {
			return nil, fmt.Errorf("payment %s reuse_after: %w", p.ID, err)
		}
		if p.PaidAt, err = parseOptionalPBTime(paid.String); err != nil {
			return nil, fmt.Errorf("payment %s paid_at: %w", p.ID, err)
		}
		if p.ResolvedAt, err = parseOptionalPBTime(resolved.String); err != nil {
			return nil, fmt.Errorf("payment %s resolved_at: %w", p.ID, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func loadEnabledDevice(ctx context.Context, db *sql.DB) (*legacyDevice, error) {
	var enabledCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relay_devices WHERE enabled=1`).Scan(&enabledCount); err != nil {
		return nil, fmt.Errorf("count enabled relay devices: %w", err)
	}
	if enabledCount > 1 {
		return nil, fmt.Errorf("legacy source has %d enabled relay devices; expected at most one", enabledCount)
	}
	row := db.QueryRowContext(ctx, `SELECT device_id,name,public_key_pem,enabled,COALESCE(NULLIF(enrolled_at,''),created),
		last_seen_at,last_heartbeat_at,app_version,device_model,android_version,notification_access,listener_connected,
		battery_optimization_exempt,power_save_mode,background_restricted,foreground_service_active,pending_count,failed_count,
		last_client_delivery_at,last_client_error FROM relay_devices WHERE enabled=1 ORDER BY updated DESC LIMIT 1`)
	var d legacyDevice
	var enabled, notification, listener, battery, powerSave, restricted, foreground int
	var enrolled, seen, heartbeat, delivery string
	err := row.Scan(&d.DeviceID, &d.Name, &d.PublicKeyPEM, &enabled, &enrolled, &seen, &heartbeat, &d.AppVersion, &d.DeviceModel, &d.AndroidVersion,
		&notification, &listener, &battery, &powerSave, &restricted, &foreground, &d.PendingCount, &d.FailedCount, &delivery, &d.LastClientError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read legacy relay device: %w", err)
	}
	d.Enabled = enabled == 1
	d.NotificationAccess, d.ListenerConnected = notification == 1, listener == 1
	d.BatteryExempt, d.PowerSave = battery == 1, powerSave == 1
	d.BackgroundRestricted, d.ForegroundService = restricted == 1, foreground == 1
	if d.EnrolledAt, err = parsePBTime(enrolled); err != nil {
		return nil, fmt.Errorf("relay enrolled_at: %w", err)
	}
	if d.LastSeenAt, err = parseOptionalPBTime(seen); err != nil {
		return nil, err
	}
	if d.LastHeartbeatAt, err = parseOptionalPBTime(heartbeat); err != nil {
		return nil, err
	}
	if d.LastDeliveryAt, err = parseOptionalPBTime(delivery); err != nil {
		return nil, err
	}
	return &d, nil
}

func parsePBTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	layouts := []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid PocketBase timestamp %q", value)
}

func parseOptionalPBTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parsePBTime(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func archivedOnlyCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
	out := map[string]int{}
	for _, table := range []string{"notification_events", "relay_events", "sms_events", "email_events", "review_cases", "alerts", "webhook_deliveries"} {
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&exists); err != nil {
			return nil, err
		}
		if exists == 0 {
			continue
		}
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			return nil, fmt.Errorf("count archived-only %s: %w", table, err)
		}
		if count > 0 {
			out[table] = count
		}
	}
	return out, nil
}
