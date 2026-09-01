package migratev3

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

const merchantIdempotencyScope = "merchant:v1"

func importAll(ctx context.Context, db *storage.DB, payments []migratedPayment, device *legacyDevice, opts Options, now time.Time) error {
	return db.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if err := insertProfiles(ctx, tx, payments, opts, now); err != nil {
			return err
		}
		for _, payment := range payments {
			if err := insertMigratedPayment(ctx, tx, payment, now); err != nil {
				return err
			}
		}
		if device != nil {
			if err := insertDevice(ctx, tx, *device); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertProfiles(ctx context.Context, tx *storage.ImmediateTx, payments []migratedPayment, opts Options, now time.Time) error {
	hasSlice := false
	for _, p := range payments {
		if p.ProfileID == "slice" {
			hasSlice = true
			break
		}
	}
	profiles := []struct {
		id, label, upi, payee, parser string
		enabled, active               int
	}{
		{"kotak", "Kotak", opts.KotakUPIID, opts.KotakPayee, "kotak_sms", 1, boolInt(opts.ActiveProfile == "kotak")},
		{"paytm", "Paytm", opts.PaytmUPIID, opts.PaytmPayee, "paytm_notification", 1, boolInt(opts.ActiveProfile == "paytm")},
	}
	if hasSlice {
		profiles = append(profiles, struct {
			id, label, upi, payee, parser string
			enabled, active               int
		}{"slice", "Slice (legacy)", "unknown@legacy", "Slice (legacy)", "legacy", 0, 0})
	}
	for _, p := range profiles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO collection_profiles(id,label,upi_id,payee_name,parser,enabled,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, p.id, p.label, p.upi, nullableText(p.payee), p.parser, p.enabled, p.active, now.UnixMilli(), now.UnixMilli()); err != nil {
			return fmt.Errorf("insert profile %s: %w", p.id, err)
		}
	}
	return nil
}
func insertMigratedPayment(ctx context.Context, tx *storage.ImmediateTx, p migratedPayment, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO payments(
		id,name,external_id,metadata_json,requested_amount_paise,payable_amount_paise,adjustment_paise,currency,
		collection_profile_id,upi_id_snapshot,payee_name_snapshot,status,created_at,expires_at,grace_until,reuse_after,
		paid_at,payer_name,payer_upi_id,internal_note)
		VALUES(?,?,?,?,?,?,?,'INR',?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, nullableText(p.ExternalID), p.MetadataJSON, p.Requested, p.Payable, p.Adjustment, p.ProfileID, p.UPIIDSnapshot,
		nullableText(p.PayeeSnapshot), p.Status, p.CreatedAt.UnixMilli(), p.ExpiresAt.UnixMilli(), p.GraceUntil.UnixMilli(), p.ReuseAfter.UnixMilli(),
		nullableTime(p.PaidAt), nullableText(p.PayerName), nullableText(p.PayerUPIID), nullableText(p.InternalNote)); err != nil {
		return fmt.Errorf("insert payment %s: %w", p.ID, err)
	}
	released := any(nil)
	if !now.Before(p.ReuseAfter) {
		released = p.ReuseAfter.UnixMilli()
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO amount_reservations(id,collection_profile_id,payable_amount_paise,payment_id,reserved_at,reserved_until,released_at,last_used_at)
		VALUES(?,?,?,?,?,?,?,?)`, "res_legacy_"+p.ID, p.ProfileID, p.Payable, p.ID, p.CreatedAt.UnixMilli(), p.ReuseAfter.UnixMilli(), released, p.CreatedAt.UnixMilli()); err != nil {
		return fmt.Errorf("insert reservation for %s: %w", p.ID, err)
	}
	if err := insertIdempotencyTombstone(ctx, tx, p, now); err != nil {
		return err
	}
	if err := insertHistory(ctx, tx, p); err != nil {
		return err
	}
	return nil
}

func insertIdempotencyTombstone(ctx context.Context, tx *storage.ImmediateTx, p migratedPayment, now time.Time) error {
	if p.LegacyIdempotencyKey == "" {
		return nil
	}
	requestHash, keyHash, err := idempotencyHashes(p)
	if err != nil {
		return fmt.Errorf("hash legacy idempotency for %s: %w", p.ID, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO idempotency_keys(scope,key_hash,request_hash,payment_id,created_at,expires_at) VALUES(?,?,?,?,?,?)`,
		merchantIdempotencyScope, keyHash[:], requestHash[:], p.ID, now.UnixMilli(), now.Add(7*24*time.Hour).UnixMilli()); err != nil {
		return fmt.Errorf("insert idempotency tombstone for %s: %w", p.ID, err)
	}
	return nil
}
func insertHistory(ctx context.Context, tx *storage.ImmediateTx, p migratedPayment) error {
	migrationChanges, _ := json.Marshal(map[string]any{"migration": map[string]any{"source": "paygate_v3", "legacy_status": p.LegacyStatus, "legacy_profile": p.ProfileID}})
	if _, err := tx.ExecContext(ctx, `INSERT INTO payment_history(id,payment_id,type,actor,summary,changes_json,created_at) VALUES(?,?,?,?,?,?,?)`,
		"hist_legacy_created_"+p.ID, p.ID, "payment.created", "system", "Imported from PayGate v3", string(migrationChanges), p.CreatedAt.UnixMilli()); err != nil {
		return fmt.Errorf("insert created history for %s: %w", p.ID, err)
	}
	if p.Status == "pending" {
		return nil
	}
	typeName, summary, at := "payment."+p.Status, "Payment "+p.Status, p.ExpiresAt
	if p.Status == "paid" && p.PaidAt != nil {
		at = *p.PaidAt
		summary = "Payment paid"
	}
	if (p.Status == "cancelled" || p.Status == "expired") && p.LegacyResolvedAt != nil {
		at = *p.LegacyResolvedAt
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO payment_history(id,payment_id,type,actor,summary,changes_json,created_at) VALUES(?,?,?,?,?,?,?)`,
		"hist_legacy_state_"+p.ID, p.ID, typeName, "system", summary, string(migrationChanges), at.UnixMilli()); err != nil {
		return fmt.Errorf("insert terminal history for %s: %w", p.ID, err)
	}
	return nil
}

func insertDevice(ctx context.Context, tx *storage.ImmediateTx, d legacyDevice) error {
	if d.DeviceID == "" || d.PublicKeyPEM == "" {
		return fmt.Errorf("legacy relay device identity is incomplete")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO relay_devices(
		id,name,public_key_pem,enabled,enrolled_at,last_seen_at,last_heartbeat_at,app_version,device_model,android_version,
		notification_access,listener_connected,battery_optimization_exempt,power_save_mode,background_restricted,foreground_service,
		pending_count,failed_count,last_successful_delivery_at,last_client_error)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.DeviceID, nullableText(d.Name), d.PublicKeyPEM, boolInt(d.Enabled), d.EnrolledAt.UnixMilli(), nullableTime(d.LastSeenAt), nullableTime(d.LastHeartbeatAt),
		nullableText(d.AppVersion), nullableText(d.DeviceModel), nullableText(d.AndroidVersion), boolInt(d.NotificationAccess), boolInt(d.ListenerConnected),
		boolInt(d.BatteryExempt), boolInt(d.PowerSave), boolInt(d.BackgroundRestricted), boolInt(d.ForegroundService), d.PendingCount, d.FailedCount,
		nullableTime(d.LastDeliveryAt), nullableText(d.LastClientError))
	if err != nil {
		return fmt.Errorf("insert relay device: %w", err)
	}
	return nil
}
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().UnixMilli()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
