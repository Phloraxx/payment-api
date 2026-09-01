package migratev3

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const legacyUnknownName = "Unknown (legacy)"

func normalizePayment(p legacyPayment, opts Options, now time.Time) (migratedPayment, bool, bool, error) {
	var out migratedPayment
	out.ID = strings.TrimSpace(p.ID)
	out.Requested, out.Payable = p.Requested, p.Payable
	out.Adjustment = p.Payable - p.Requested
	out.ProfileID = strings.ToLower(strings.TrimSpace(p.Account))
	out.CreatedAt, out.ExpiresAt, out.ReuseAfter = p.CreatedAt, p.ExpiresAt, p.ReuseAfter
	out.PayerName, out.PayerUPIID = strings.TrimSpace(p.PayerName), strings.TrimSpace(p.PayerUPIID)
	out.InternalNote = strings.TrimSpace(p.AdminNote)
	out.LegacyStatus = strings.ToLower(strings.TrimSpace(p.Status))
	out.LegacyIdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	out.LegacyResolvedAt = p.ResolvedAt
	if out.ID == "" {
		return out, false, false, errors.New("legacy payment id is empty")
	}
	if out.Requested <= 0 || out.Requested%100 != 0 || out.Payable <= out.Requested || out.Adjustment < 1 || out.Adjustment > 199 || out.Payable%100 == 0 {
		return out, false, false, fmt.Errorf("payment %s has invalid amount relationship", out.ID)
	}
	if !out.CreatedAt.Before(out.ExpiresAt) || !out.ExpiresAt.Before(out.ReuseAfter) {
		return out, false, false, fmt.Errorf("payment %s has invalid lifecycle timestamps", out.ID)
	}
	nameFilled := false
	out.Name = strings.TrimSpace(p.CustomerName)
	if out.Name == "" {
		out.Name, nameFilled = legacyUnknownName, true
	}
	if len([]rune(out.Name)) > 120 {
		out.Name = string([]rune(out.Name)[:120])
	}

	switch out.ProfileID {
	case "kotak":
		out.UPIIDSnapshot, out.PayeeSnapshot = strings.TrimSpace(opts.KotakUPIID), strings.TrimSpace(opts.KotakPayee)
	case "paytm":
		out.UPIIDSnapshot, out.PayeeSnapshot = strings.TrimSpace(opts.PaytmUPIID), strings.TrimSpace(opts.PaytmPayee)
	case "slice":
		out.UPIIDSnapshot, out.PayeeSnapshot = "unknown@legacy", "Slice (legacy)"
	default:
		return out, nameFilled, false, fmt.Errorf("payment %s has unsupported legacy account %q", out.ID, out.ProfileID)
	}
	if out.UPIIDSnapshot == "" {
		return out, nameFilled, false, fmt.Errorf("payment %s profile %s has no destination UPI snapshot", out.ID, out.ProfileID)
	}
	lateNormalized := false
	switch out.LegacyStatus {
	case "paid":
		out.Status = "paid"
	case "late":
		out.Status, lateNormalized = "paid", true
	case "expired", "cancelled":
		out.Status = out.LegacyStatus
	case "pending":
		out.Status = "pending"
	default:
		return out, nameFilled, lateNormalized, fmt.Errorf("payment %s has unsupported status %q", out.ID, out.LegacyStatus)
	}
	out.PaidAt = p.PaidAt
	if out.Status == "paid" && out.PaidAt == nil {
		return out, nameFilled, lateNormalized, fmt.Errorf("payment %s paid without paid_at", out.ID)
	}
	if out.Status != "paid" {
		out.PaidAt = nil
	}
	out.GraceUntil = out.ExpiresAt.Add(5 * time.Minute)
	if !out.GraceUntil.Before(out.ReuseAfter) {
		out.GraceUntil = out.ReuseAfter.Add(-time.Millisecond)
	}
	if !out.ExpiresAt.Before(out.GraceUntil) {
		return out, nameFilled, lateNormalized, fmt.Errorf("payment %s cannot derive v4 grace window", out.ID)
	}
	if out.Status == "pending" && !now.Before(out.GraceUntil) {
		out.Status = "expired"
	}

	metadata, _, err := migrateMetadata(p, &out)
	if err != nil {
		return out, nameFilled, lateNormalized, err
	}
	out.MetadataJSON = metadata
	return out, nameFilled, lateNormalized || false, nil
}

func migrateMetadata(p legacyPayment, out *migratedPayment) (string, bool, error) {
	var decoded any
	if err := json.Unmarshal(p.Metadata, &decoded); err != nil {
		return "", false, fmt.Errorf("payment %s has invalid metadata JSON: %w", p.ID, err)
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		root = map[string]any{"legacy_original_metadata": decoded}
	}
	eventRecovered := false
	if raw, ok := root["eventId"].(string); ok && strings.TrimSpace(raw) != "" {
		out.ExternalID = strings.TrimSpace(raw)
		eventRecovered = true
	}
	legacy := map[string]any{
		"source":                             "paygate_v3",
		"external_id":                        strings.TrimSpace(p.ExternalID),
		"status":                             out.LegacyStatus,
		"payment_account":                    out.ProfileID,
		"destination_snapshot_reconstructed": out.ProfileID != "slice",
	}
	addLegacyString(legacy, "display_name", p.DisplayName)
	addLegacyString(legacy, "customer_email", p.CustomerEmail)
	addLegacyString(legacy, "customer_phone", p.CustomerPhone)
	addLegacyString(legacy, "description", p.Description)

	if value, ok := decodeOptionalJSON(p.Tags); ok {
		legacy["tags"] = value
	}
	if value, ok := decodeOptionalJSON(p.CustomFields); ok {
		legacy["custom_fields"] = value
	}
	root["legacy_v3"] = legacy
	encoded, err := json.Marshal(root)
	if err != nil {
		return "", false, fmt.Errorf("payment %s encode migrated metadata: %w", p.ID, err)
	}
	if len(encoded) > 16<<10 {
		return "", false, fmt.Errorf("payment %s migrated metadata exceeds v4 limit", p.ID)
	}
	return string(encoded), eventRecovered, nil
}

func addLegacyString(dst map[string]any, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		dst[key] = value
	}
}

func decodeOptionalJSON(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return nil, false
	}
	return out, true
}

func idempotencyHashes(p migratedPayment) ([32]byte, [32]byte, error) {
	var zero [32]byte
	if p.LegacyIdempotencyKey == "" {
		return zero, zero, nil
	}
	requestHash := sha256.Sum256([]byte("paygate-v4-legacy-cutover:" + p.ID))
	keyHash := sha256.Sum256([]byte(p.LegacyIdempotencyKey))
	return requestHash, keyHash, nil
}
