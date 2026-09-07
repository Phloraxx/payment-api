package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const restoreFileMode os.FileMode = 0o600

// RestoreReport contains non-sensitive counts from an isolated backup validation.
type RestoreReport struct {
	SchemaVersion     int
	Payments          int
	RelayEvents       int
	WebhookDeliveries int
	RelayDevices      int
}

// RestoreDrill copies a standalone backup into a private temporary directory,
// opens it through the production storage path, and verifies integrity and
// representative tables. It never opens or writes the live database.
func RestoreDrill(ctx context.Context, backupPath, livePath, expectedSHA256 string) (report RestoreReport, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	backupPath, sourceInfo, expected, err := validateRestoreSource(backupPath, livePath, expectedSHA256)
	if err != nil {
		return RestoreReport{}, err
	}

	drillDir, err := os.MkdirTemp("", "paygate-restore-drill-")
	if err != nil {
		return RestoreReport{}, fmt.Errorf("create restore drill directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(drillDir); err == nil && removeErr != nil {
			err = fmt.Errorf("remove restore drill directory: %w", removeErr)
		}
	}()
	drillPath := filepath.Join(drillDir, "paygate.db")

	if err := copyRestoreSource(backupPath, sourceInfo, drillPath, expected); err != nil {
		return RestoreReport{}, err
	}
	if err := validateRestoreDatabase(ctx, drillPath); err != nil {
		return RestoreReport{}, fmt.Errorf("validate isolated restore: %w", err)
	}
	db, err := Open(ctx, drillPath)
	if err != nil {
		return RestoreReport{}, fmt.Errorf("open isolated restore: %w", err)
	}
	report, err = inspectRestoredDatabase(ctx, db)
	closeErr := db.Close()
	if err != nil {
		return RestoreReport{}, err
	}
	if closeErr != nil {
		return RestoreReport{}, fmt.Errorf("close isolated restore: %w", closeErr)
	}
	return report, nil
}

func validateRestoreSource(backupPath, livePath, expectedSHA256 string) (string, os.FileInfo, []byte, error) {
	if strings.TrimSpace(backupPath) == "" {
		return "", nil, nil, errors.New("restore backup path is required")
	}
	backupPath, err := filepath.Abs(strings.TrimSpace(backupPath))
	if err != nil {
		return "", nil, nil, fmt.Errorf("resolve restore backup path: %w", err)
	}
	info, err := os.Lstat(backupPath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("inspect restore backup: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", nil, nil, errors.New("restore backup must be a regular file, not a symlink")
	}
	if info.Size() <= 0 {
		return "", nil, nil, errors.New("restore backup must be non-empty")
	}
	if info.Mode().Perm() != restoreFileMode {
		return "", nil, nil, fmt.Errorf("restore backup permissions must be %04o, got %04o", restoreFileMode.Perm(), info.Mode().Perm())
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if sidecarInfo, sidecarErr := os.Lstat(backupPath + suffix); sidecarErr == nil {
			if sidecarInfo.Mode()&os.ModeSymlink != 0 || !sidecarInfo.Mode().IsRegular() {
				return "", nil, nil, fmt.Errorf("restore backup sidecar %s is not a regular file", suffix)
			}
			return "", nil, nil, fmt.Errorf("restore backup has sidecar %s; use a completed standalone backup", suffix)
		} else if !errors.Is(sidecarErr, os.ErrNotExist) {
			return "", nil, nil, fmt.Errorf("inspect restore backup sidecar %s: %w", suffix, sidecarErr)
		}
	}
	if strings.TrimSpace(livePath) != "" {
		livePath, err = filepath.Abs(strings.TrimSpace(livePath))
		if err != nil {
			return "", nil, nil, fmt.Errorf("resolve live database path: %w", err)
		}
		if filepath.Clean(livePath) == filepath.Clean(backupPath) {
			return "", nil, nil, errors.New("restore backup cannot be the live database")
		}
		liveInfo, statErr := os.Stat(livePath)
		if statErr == nil {
			if os.SameFile(info, liveInfo) {
				return "", nil, nil, errors.New("restore backup cannot be the live database")
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return "", nil, nil, fmt.Errorf("inspect live database path: %w", statErr)
		}
	}

	var expected []byte
	if value := strings.TrimSpace(expectedSHA256); value != "" {
		expected, err = hex.DecodeString(value)
		if err != nil || len(expected) != sha256.Size {
			return "", nil, nil, errors.New("restore SHA-256 must be 64 hexadecimal characters")
		}
	}
	return backupPath, info, expected, nil
}

func copyRestoreSource(sourcePath string, expectedInfo os.FileInfo, destination string, expectedHash []byte) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open restore backup: %w", err)
	}
	defer source.Close()
	openedInfo, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat restore backup: %w", err)
	}
	if !os.SameFile(expectedInfo, openedInfo) || openedInfo.Mode()&os.ModeSymlink != 0 || !openedInfo.Mode().IsRegular() {
		return errors.New("restore backup changed while it was being opened")
	}

	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, restoreFileMode)
	if err != nil {
		return fmt.Errorf("create isolated restore copy: %w", err)
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(destinationFile, hasher), source)
	if copyErr == nil {
		copyErr = destinationFile.Sync()
	}
	closeErr := destinationFile.Close()
	if copyErr != nil {
		return fmt.Errorf("copy restore backup after %d bytes: %w", written, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close isolated restore copy: %w", closeErr)
	}
	afterInfo, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat restore backup after copy: %w", err)
	}
	if !os.SameFile(expectedInfo, afterInfo) || afterInfo.Size() != expectedInfo.Size() ||
		!afterInfo.ModTime().Equal(expectedInfo.ModTime()) {
		return errors.New("restore backup changed while it was being copied")
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, sidecarErr := os.Lstat(sourcePath + suffix); sidecarErr == nil {
			return fmt.Errorf("restore backup sidecar %s appeared while it was being copied", suffix)
		} else if !errors.Is(sidecarErr, os.ErrNotExist) {
			return fmt.Errorf("inspect restore backup sidecar %s after copy: %w", suffix, sidecarErr)
		}
	}
	if len(expectedHash) > 0 && !strings.EqualFold(hex.EncodeToString(hasher.Sum(nil)), hex.EncodeToString(expectedHash)) {
		return errors.New("restore backup SHA-256 does not match the expected value")
	}
	return nil
}

func validateRestoreDatabase(ctx context.Context, path string) error {
	query := url.Values{}
	query.Set("mode", "ro")
	query.Set("_query_only", "true")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?"+query.Encode())
	if err != nil {
		return fmt.Errorf("open restore validation connection: %w", err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	if err := raw.PingContext(ctx); err != nil {
		return fmt.Errorf("ping restore validation connection: %w", err)
	}
	var integrity string
	if err := raw.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return fmt.Errorf("restore pre-migration integrity check: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(integrity)) != "ok" {
		return fmt.Errorf("restore pre-migration integrity check returned %q", integrity)
	}
	rows, err := raw.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return fmt.Errorf("read restore schema migrations before migration: %w", err)
	}
	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return fmt.Errorf("scan restore schema migration before migration: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate restore schema migrations before migration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close restore schema migrations before migration: %w", err)
	}
	if len(versions) == 0 {
		return errors.New("restore backup has no schema migrations")
	}
	for index, version := range versions {
		if version != index+1 {
			return fmt.Errorf("restore schema migrations are not contiguous at version %d", version)
		}
	}
	if versions[len(versions)-1] > schemaVersion {
		return fmt.Errorf("restore schema version %d is newer than supported %d", versions[len(versions)-1], schemaVersion)
	}
	type restoreExpectedColumn struct {
		name string
		kind string
		pk   bool
	}
	type restoreColumn struct {
		name    string
		kind    string
		pk      bool
		notNull bool
	}
	requiredColumns := map[string][]restoreExpectedColumn{
		"schema_migrations":    {{"version", "INTEGER", true}, {"applied_at", "INTEGER", false}},
		"collection_profiles":  {{"id", "TEXT", true}, {"label", "TEXT", false}, {"upi_id", "TEXT", false}, {"payee_name", "TEXT", false}, {"parser", "TEXT", false}, {"enabled", "INTEGER", false}, {"active", "INTEGER", false}, {"created_at", "INTEGER", false}, {"updated_at", "INTEGER", false}},
		"payments":             {{"id", "TEXT", true}, {"name", "TEXT", false}, {"external_id", "TEXT", false}, {"metadata_json", "TEXT", false}, {"requested_amount_paise", "INTEGER", false}, {"payable_amount_paise", "INTEGER", false}, {"adjustment_paise", "INTEGER", false}, {"currency", "TEXT", false}, {"collection_profile_id", "TEXT", false}, {"upi_id_snapshot", "TEXT", false}, {"payee_name_snapshot", "TEXT", false}, {"status", "TEXT", false}, {"created_at", "INTEGER", false}, {"expires_at", "INTEGER", false}, {"grace_until", "INTEGER", false}, {"reuse_after", "INTEGER", false}, {"paid_at", "INTEGER", false}, {"payer_name", "TEXT", false}, {"payer_upi_id", "TEXT", false}, {"internal_note", "TEXT", false}},
		"amount_reservations":  {{"id", "TEXT", true}, {"collection_profile_id", "TEXT", false}, {"payable_amount_paise", "INTEGER", false}, {"payment_id", "TEXT", false}, {"reserved_at", "INTEGER", false}, {"reserved_until", "INTEGER", false}, {"released_at", "INTEGER", false}, {"last_used_at", "INTEGER", false}},
		"idempotency_keys":     {{"scope", "TEXT", true}, {"key_hash", "BLOB", true}, {"request_hash", "BLOB", false}, {"payment_id", "TEXT", false}, {"created_at", "INTEGER", false}, {"expires_at", "INTEGER", false}},
		"relay_devices":        {{"id", "TEXT", true}, {"name", "TEXT", false}, {"public_key_pem", "TEXT", false}, {"enabled", "INTEGER", false}, {"enrolled_at", "INTEGER", false}, {"last_seen_at", "INTEGER", false}, {"last_heartbeat_at", "INTEGER", false}, {"app_version", "TEXT", false}, {"device_model", "TEXT", false}, {"android_version", "TEXT", false}},
		"pairing_sessions":     {{"id", "TEXT", true}, {"token_hash", "BLOB", false}, {"replace_existing", "INTEGER", false}, {"created_at", "INTEGER", false}, {"expires_at", "INTEGER", false}, {"consumed_at", "INTEGER", false}},
		"relay_events":         {{"id", "TEXT", true}, {"device_id", "TEXT", false}, {"source_event_id", "TEXT", false}, {"package_name", "TEXT", false}, {"posted_at", "INTEGER", false}, {"received_at", "INTEGER", false}, {"amount_hint_paise", "INTEGER", false}, {"title", "TEXT", false}, {"text", "TEXT", false}, {"big_text", "TEXT", false}, {"status", "TEXT", false}, {"error", "TEXT", false}},
		"payment_observations": {{"id", "TEXT", true}, {"relay_event_id", "TEXT", false}, {"source", "TEXT", false}, {"collection_profile_id", "TEXT", false}, {"amount_paise", "INTEGER", false}, {"payer_name", "TEXT", false}, {"payer_upi_id", "TEXT", false}, {"occurred_at", "INTEGER", false}, {"occurred_at_source", "TEXT", false}, {"received_at", "INTEGER", false}, {"matched_payment_id", "TEXT", false}, {"match_result", "TEXT", false}},
		"payment_history":      {{"id", "TEXT", true}, {"payment_id", "TEXT", false}, {"type", "TEXT", false}, {"actor", "TEXT", false}, {"summary", "TEXT", false}, {"changes_json", "TEXT", false}, {"created_at", "INTEGER", false}},
		"webhook_deliveries":   {{"id", "TEXT", true}, {"event_type", "TEXT", false}, {"payment_id", "TEXT", false}, {"payload_json", "TEXT", false}, {"status", "TEXT", false}, {"attempts", "INTEGER", false}, {"next_attempt_at", "INTEGER", false}, {"last_http_status", "INTEGER", false}, {"last_error", "TEXT", false}, {"created_at", "INTEGER", false}, {"delivered_at", "INTEGER", false}},
		"api_keys":             {{"id", "TEXT", true}, {"label", "TEXT", false}, {"secret_hash", "BLOB", false}, {"enabled", "INTEGER", false}, {"created_at", "INTEGER", false}, {"last_used_at", "INTEGER", false}},
		"admin_credentials":    {{"singleton", "INTEGER", true}, {"password_hash", "TEXT", false}, {"updated_at", "INTEGER", false}},
		"admin_sessions":       {{"token_hash", "BLOB", true}, {"created_at", "INTEGER", false}, {"expires_at", "INTEGER", false}, {"last_seen_at", "INTEGER", false}, {"revoked_at", "INTEGER", false}},
		"settings":             {{"key", "TEXT", true}, {"value", "TEXT", false}, {"updated_at", "INTEGER", false}},
	}
	if versions[len(versions)-1] >= 2 {
		requiredColumns["relay_devices"] = append(requiredColumns["relay_devices"],
			restoreExpectedColumn{"notification_access", "INTEGER", false},
			restoreExpectedColumn{"listener_connected", "INTEGER", false},
			restoreExpectedColumn{"battery_optimization_exempt", "INTEGER", false},
			restoreExpectedColumn{"power_save_mode", "INTEGER", false},
			restoreExpectedColumn{"background_restricted", "INTEGER", false},
			restoreExpectedColumn{"foreground_service", "INTEGER", false},
			restoreExpectedColumn{"pending_count", "INTEGER", false},
			restoreExpectedColumn{"failed_count", "INTEGER", false},
			restoreExpectedColumn{"last_successful_delivery_at", "INTEGER", false},
			restoreExpectedColumn{"last_client_error", "TEXT", false})
	}
	if versions[len(versions)-1] >= 6 {
		requiredColumns["relay_devices"] = append(requiredColumns["relay_devices"],
			restoreExpectedColumn{"epoch_required", "INTEGER", false})
	}
	if versions[len(versions)-1] >= 7 {
		requiredColumns["amount_reservations"] = append(requiredColumns["amount_reservations"],
			restoreExpectedColumn{"global_unique_enforced", "INTEGER", false})
	}
	requiredNotNull := map[string][]string{
		"schema_migrations":    {"applied_at"},
		"collection_profiles":  {"label", "upi_id", "parser", "enabled", "active", "created_at", "updated_at"},
		"payments":             {"name", "metadata_json", "requested_amount_paise", "payable_amount_paise", "adjustment_paise", "currency", "collection_profile_id", "upi_id_snapshot", "status", "created_at", "expires_at", "grace_until", "reuse_after"},
		"amount_reservations":  {"collection_profile_id", "payable_amount_paise", "payment_id", "reserved_at", "reserved_until", "last_used_at"},
		"idempotency_keys":     {"scope", "key_hash", "request_hash", "payment_id", "created_at", "expires_at"},
		"relay_devices":        {"public_key_pem", "enabled", "enrolled_at"},
		"pairing_sessions":     {"token_hash", "replace_existing", "created_at", "expires_at"},
		"relay_events":         {"device_id", "source_event_id", "package_name", "posted_at", "received_at", "status"},
		"payment_observations": {"relay_event_id", "source", "collection_profile_id", "amount_paise", "occurred_at", "occurred_at_source", "received_at", "match_result"},
		"payment_history":      {"payment_id", "type", "actor", "summary", "changes_json", "created_at"},
		"webhook_deliveries":   {"event_type", "payment_id", "payload_json", "status", "attempts", "created_at"},
		"api_keys":             {"label", "secret_hash", "enabled", "created_at"},
		"admin_credentials":    {"password_hash", "updated_at"},
		"admin_sessions":       {"created_at", "expires_at"},
		"settings":             {"value", "updated_at"},
	}
	if versions[len(versions)-1] >= 6 {
		requiredNotNull["relay_devices"] = append(requiredNotNull["relay_devices"], "epoch_required")
	}
	if versions[len(versions)-1] >= 7 {
		requiredNotNull["amount_reservations"] = append(requiredNotNull["amount_reservations"], "global_unique_enforced")
	}
	type restoreForeignKey struct {
		table    string
		from     string
		to       string
		onUpdate string
		onDelete string
	}
	requiredForeignKeys := map[string][]restoreForeignKey{
		"payments": {
			{"collection_profiles", "collection_profile_id", "id", "RESTRICT", "RESTRICT"},
		},
		"amount_reservations": {
			{"collection_profiles", "collection_profile_id", "id", "RESTRICT", "RESTRICT"},
			{"payments", "payment_id", "id", "RESTRICT", "RESTRICT"},
		},
		"idempotency_keys": {
			{"payments", "payment_id", "id", "RESTRICT", "RESTRICT"},
		},
		"relay_events": {
			{"relay_devices", "device_id", "id", "RESTRICT", "RESTRICT"},
		},
		"payment_observations": {
			{"relay_events", "relay_event_id", "id", "RESTRICT", "RESTRICT"},
			{"collection_profiles", "collection_profile_id", "id", "RESTRICT", "RESTRICT"},
			{"payments", "matched_payment_id", "id", "RESTRICT", "SET NULL"},
		},
		"payment_history": {
			{"payments", "payment_id", "id", "RESTRICT", "RESTRICT"},
		},
		"webhook_deliveries": {
			{"payments", "payment_id", "id", "RESTRICT", "RESTRICT"},
		},
	}
	requiredChecks := map[string]bool{
		"collection_profiles": true, "payments": true, "amount_reservations": true,
		"idempotency_keys": true, "relay_devices": true, "pairing_sessions": true,
		"relay_events": true, "payment_observations": true, "payment_history": true,
		"webhook_deliveries": true, "api_keys": true, "admin_credentials": true,
		"admin_sessions": true,
	}
	requiredCheckFragments := map[string][]string{
		"collection_profiles": {
			"LENGTH(TRIM(LABEL)) BETWEEN 1 AND 120",
			"LENGTH(TRIM(UPI_ID)) BETWEEN 3 AND 255",
			"ENABLED IN (0,1)",
			"ACTIVE IN (0,1)",
			"ACTIVE = 0 OR ENABLED = 1",
		},
		"payments": {
			"LENGTH(TRIM(NAME)) BETWEEN 1 AND 120",
			"REQUESTED_AMOUNT_PAISE > 0",
			"REQUESTED_AMOUNT_PAISE % 100 = 0",
			"PAYABLE_AMOUNT_PAISE > REQUESTED_AMOUNT_PAISE",
			"PAYABLE_AMOUNT_PAISE % 100 BETWEEN 1 AND 99",
			"PAYABLE_AMOUNT_PAISE = REQUESTED_AMOUNT_PAISE + ADJUSTMENT_PAISE",
			"ADJUSTMENT_PAISE BETWEEN 1 AND 199",
			"CURRENCY = 'INR'",
			"JSON_VALID(METADATA_JSON)",
			"STATUS IN ('PENDING','PAID','EXPIRED','CANCELLED')",
			"CREATED_AT < EXPIRES_AT AND EXPIRES_AT < GRACE_UNTIL AND GRACE_UNTIL < REUSE_AFTER",
			"STATUS = 'PAID' AND PAID_AT IS NOT NULL",
			"STATUS != 'PAID' AND PAID_AT IS NULL",
		},
		"amount_reservations": {
			"PAYABLE_AMOUNT_PAISE > 0",
			"PAYABLE_AMOUNT_PAISE % 100 BETWEEN 1 AND 99",
			"RESERVED_UNTIL > RESERVED_AT",
			"RELEASED_AT IS NULL OR RELEASED_AT >= RESERVED_AT",
		},
		"idempotency_keys": {"EXPIRES_AT > CREATED_AT"},
		"relay_devices":    {"ENABLED IN (0,1)"},
		"pairing_sessions": {
			"REPLACE_EXISTING IN (0,1)",
			"EXPIRES_AT > CREATED_AT",
			"CONSUMED_AT IS NULL OR (CONSUMED_AT >= CREATED_AT AND CONSUMED_AT <= EXPIRES_AT)",
		},
		"relay_events": {
			"AMOUNT_HINT_PAISE IS NULL OR (AMOUNT_HINT_PAISE > 0 AND AMOUNT_HINT_PAISE % 100 BETWEEN 1 AND 99)",
			"STATUS IN ('RECEIVED','PARSED','IGNORED','MATCHED','UNMATCHED','AMBIGUOUS','ERROR')",
		},
		"payment_observations": {
			"AMOUNT_PAISE > 0",
			"AMOUNT_PAISE % 100 BETWEEN 1 AND 99",
			"OCCURRED_AT_SOURCE IN ('NOTIFICATION_TEXT','NOTIFICATION_POSTED_AT','SERVER_RECEIVED_AT')",
		},
		"payment_history": {
			"ACTOR IN ('SYSTEM','ADMIN')",
			"JSON_VALID(CHANGES_JSON)",
		},
		"webhook_deliveries": {
			"EVENT_TYPE IN ('PAYMENT.CREATED','PAYMENT.PAID','PAYMENT.EXPIRED','PAYMENT.CANCELLED','PAYMENT.UPDATED')",
			"JSON_VALID(PAYLOAD_JSON)",
			"STATUS IN ('PENDING','RETRY','DELIVERED','EXHAUSTED')",
			"ATTEMPTS >= 0",
			"STATUS = 'DELIVERED' AND DELIVERED_AT IS NOT NULL",
			"STATUS != 'DELIVERED' AND DELIVERED_AT IS NULL",
		},
		"api_keys":          {"ENABLED IN (0,1)"},
		"admin_credentials": {"SINGLETON = 1"},
		"admin_sessions":    {"EXPIRES_AT > CREATED_AT"},
	}
	if versions[len(versions)-1] >= 6 {
		requiredCheckFragments["relay_devices"] = append(requiredCheckFragments["relay_devices"], "EPOCH_REQUIRED IN (0,1)")
	}
	if versions[len(versions)-1] >= 7 {
		requiredCheckFragments["amount_reservations"] = append(requiredCheckFragments["amount_reservations"], "GLOBAL_UNIQUE_ENFORCED IN (0,1)")
	}
	if versions[len(versions)-1] >= 3 {
		requiredCheckFragments["collection_profiles"] = append(requiredCheckFragments["collection_profiles"],
			"PARSER IN ('PAYTM_NOTIFICATION','KOTAK_SMS','LEGACY')",
			"PARSER != 'LEGACY' OR (ENABLED = 0 AND ACTIVE = 0)")
	} else {
		requiredCheckFragments["collection_profiles"] = append(requiredCheckFragments["collection_profiles"],
			"PARSER IN ('PAYTM_NOTIFICATION','KOTAK_SMS')")
	}
	if versions[len(versions)-1] >= 4 {
		requiredCheckFragments["payment_observations"] = append(requiredCheckFragments["payment_observations"],
			"LENGTH(TRIM(SOURCE)) BETWEEN 1 AND 64",
			"MATCH_RESULT IN ('MATCHED','CORROBORATED','UNMATCHED','AMBIGUOUS','IGNORED','ERROR')")
	} else {
		requiredCheckFragments["payment_observations"] = append(requiredCheckFragments["payment_observations"],
			"SOURCE IN ('PAYTM_NOTIFICATION','KOTAK_SMS')",
			"MATCH_RESULT IN ('MATCHED','UNMATCHED','AMBIGUOUS','IGNORED','ERROR')")
	}
	for table, columns := range requiredColumns {
		var createSQL sql.NullString
		if err := raw.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&createSQL); err != nil {
			return fmt.Errorf("read restore table %s definition: %w", table, err)
		}
		if !createSQL.Valid || !strings.Contains(strings.ToUpper(createSQL.String), "STRICT") {
			return fmt.Errorf("restore table %s is not a strict production table", table)
		}
		upperSQL := strings.ToUpper(createSQL.String)
		if requiredChecks[table] && !strings.Contains(upperSQL, "CHECK") {
			return fmt.Errorf("restore table %s is missing production checks", table)
		}
		for _, fragment := range requiredCheckFragments[table] {
			if !strings.Contains(upperSQL, fragment) {
				return fmt.Errorf("restore table %s is missing production check %s", table, fragment)
			}
		}
		tableRows, err := raw.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			return fmt.Errorf("inspect restore table %s columns: %w", table, err)
		}
		found := make(map[string]restoreColumn, len(columns))
		for tableRows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue sql.NullString
			if err := tableRows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				tableRows.Close()
				return fmt.Errorf("scan restore table %s columns: %w", table, err)
			}
			if _, duplicate := found[name]; duplicate {
				tableRows.Close()
				return fmt.Errorf("restore table %s has duplicate column %s", table, name)
			}
			if name == "payload_hash" && table == "relay_events" {
				if !strings.EqualFold(strings.TrimSpace(columnType), "BLOB") {
					tableRows.Close()
					return fmt.Errorf("restore table %s payload_hash must be BLOB", table)
				}
				found[name] = restoreColumn{name: name, kind: "BLOB", notNull: notNull != 0}
				continue
			}
			found[name] = restoreColumn{name: name, kind: strings.ToUpper(strings.TrimSpace(columnType)), pk: primaryKey > 0, notNull: notNull != 0}
		}
		if err := tableRows.Err(); err != nil {
			tableRows.Close()
			return fmt.Errorf("iterate restore table %s columns: %w", table, err)
		}
		if err := tableRows.Close(); err != nil {
			return fmt.Errorf("close restore table %s columns: %w", table, err)
		}
		for name := range found {
			known := false
			for _, column := range columns {
				if column.name == name {
					known = true
					break
				}
			}
			if !known && !(table == "relay_events" && name == "payload_hash") {
				return fmt.Errorf("restore table %s has unexpected column %s", table, name)
			}
		}
		for _, name := range requiredNotNull[table] {
			actual, ok := found[name]
			if !ok || !actual.notNull {
				return fmt.Errorf("restore table %s column %s must be NOT NULL", table, name)
			}
		}
		expectedCount := len(columns)
		if table == "relay_events" && found["payload_hash"].name != "" {
			expectedCount++
		}
		if len(found) != expectedCount {
			return fmt.Errorf("restore table %s has %d columns, want %d", table, len(found), expectedCount)
		}
		for _, column := range columns {
			actual, ok := found[column.name]
			if !ok {
				return fmt.Errorf("restore table %s is missing column %s", table, column.name)
			}
			if actual.kind != column.kind {
				return fmt.Errorf("restore table %s column %s has type %s, want %s", table, column.name, actual.kind, column.kind)
			}
			if actual.pk != column.pk {
				return fmt.Errorf("restore table %s column %s primary-key flag mismatch", table, column.name)
			}
		}
		if expected := requiredForeignKeys[table]; len(expected) > 0 {
			fkRows, err := raw.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
			if err != nil {
				return fmt.Errorf("inspect restore table %s foreign keys: %w", table, err)
			}
			actual := make(map[string]struct{}, len(expected))
			for fkRows.Next() {
				var id, seq int
				var referencedTable, from, to, onUpdate, onDelete, match string
				if err := fkRows.Scan(&id, &seq, &referencedTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
					fkRows.Close()
					return fmt.Errorf("scan restore table %s foreign keys: %w", table, err)
				}
				actual[strings.Join([]string{referencedTable, from, to, strings.ToUpper(onUpdate), strings.ToUpper(onDelete)}, "\x00")] = struct{}{}
			}
			if err := fkRows.Err(); err != nil {
				fkRows.Close()
				return fmt.Errorf("iterate restore table %s foreign keys: %w", table, err)
			}
			if err := fkRows.Close(); err != nil {
				return fmt.Errorf("close restore table %s foreign keys: %w", table, err)
			}
			if len(actual) != len(expected) {
				return fmt.Errorf("restore table %s has %d foreign keys, want %d", table, len(actual), len(expected))
			}
			for _, foreignKey := range expected {
				key := strings.Join([]string{foreignKey.table, foreignKey.from, foreignKey.to, foreignKey.onUpdate, foreignKey.onDelete}, "\x00")
				if _, ok := actual[key]; !ok {
					return fmt.Errorf("restore table %s is missing foreign key %s(%s) references %s(%s) ON UPDATE %s ON DELETE %s", table, table, foreignKey.from, foreignKey.table, foreignKey.to, foreignKey.onUpdate, foreignKey.onDelete)
				}
			}
		}
	}
	type restoreIndex struct {
		table     string
		unique    bool
		columns   []string
		fragments []string
	}
	requiredIndexes := map[string]restoreIndex{
		"uq_collection_profiles_one_active": {"collection_profiles", true, []string{"active"}, []string{"ACTIVE = 1", "WHERE"}},
		"idx_payments_external_id":          {"payments", false, []string{"external_id"}, []string{"EXTERNAL_ID"}},
		"idx_payments_status_created":       {"payments", false, []string{"status", "created_at"}, []string{"STATUS", "CREATED_AT"}},
		"idx_payments_profile_payable":      {"payments", false, []string{"collection_profile_id", "payable_amount_paise"}, []string{"COLLECTION_PROFILE_ID", "PAYABLE_AMOUNT_PAISE"}},
		"uq_active_profile_payable":         {"amount_reservations", true, []string{"collection_profile_id", "payable_amount_paise"}, []string{"COLLECTION_PROFILE_ID", "PAYABLE_AMOUNT_PAISE", "RELEASED_AT", "IS NULL", "WHERE"}},
		"idx_amount_reservations_history":   {"amount_reservations", false, []string{"collection_profile_id", "payable_amount_paise", "reserved_at"}, []string{"COLLECTION_PROFILE_ID", "PAYABLE_AMOUNT_PAISE", "RESERVED_AT"}},
		"idx_amount_reservations_release":   {"amount_reservations", false, []string{"released_at", "reserved_until"}, []string{"RELEASED_AT", "RESERVED_UNTIL"}},
		"idx_relay_events_received":         {"relay_events", false, []string{"received_at"}, []string{"RECEIVED_AT"}},
		"idx_observations_amount_time":      {"payment_observations", false, []string{"collection_profile_id", "amount_paise", "occurred_at"}, []string{"COLLECTION_PROFILE_ID", "AMOUNT_PAISE", "OCCURRED_AT"}},
		"idx_payment_history_payment":       {"payment_history", false, []string{"payment_id", "created_at"}, []string{"PAYMENT_ID", "CREATED_AT"}},
		"idx_webhook_delivery_queue":        {"webhook_deliveries", false, []string{"status", "next_attempt_at", "created_at"}, []string{"STATUS", "NEXT_ATTEMPT_AT", "CREATED_AT"}},
		"idx_admin_sessions_expiry":         {"admin_sessions", false, []string{"expires_at"}, []string{"EXPIRES_AT"}},
	}
	if versions[len(versions)-1] >= 4 {
		requiredIndexes["idx_observations_payment"] = restoreIndex{
			table: "payment_observations", unique: false, columns: []string{"matched_payment_id", "occurred_at"},
			fragments: []string{"MATCHED_PAYMENT_ID", "OCCURRED_AT", "IS NOT NULL", "WHERE"},
		}
	}
	if versions[len(versions)-1] >= 5 {
		delete(requiredIndexes, "uq_active_profile_payable")
		fragments := []string{"PAYABLE_AMOUNT_PAISE", "RELEASED_AT", "IS NULL", "WHERE"}
		if versions[len(versions)-1] >= 7 {
			fragments = append(fragments, "GLOBAL_UNIQUE_ENFORCED", "=1")
		}
		requiredIndexes["uq_active_payable"] = restoreIndex{
			table: "amount_reservations", unique: true, columns: []string{"payable_amount_paise"}, fragments: fragments,
		}
	}
	readIndexColumns := func(indexName string) ([]string, error) {
		quotedName := strings.ReplaceAll(indexName, "'", "''")
		rows, err := raw.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info('%s')", quotedName))
		if err != nil {
			return nil, err
		}
		var columns []string
		for rows.Next() {
			var seq, cid int
			var columnName sql.NullString
			if err := rows.Scan(&seq, &cid, &columnName); err != nil {
				rows.Close()
				return nil, err
			}
			if !columnName.Valid {
				rows.Close()
				return nil, fmt.Errorf("index %s contains an expression", indexName)
			}
			columns = append(columns, columnName.String)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		return columns, nil
	}
	sameIndexColumns := func(actual, expected []string) bool {
		if len(actual) != len(expected) {
			return false
		}
		for index := range expected {
			if !strings.EqualFold(actual[index], expected[index]) {
				return false
			}
		}
		return true
	}
	requiredUniqueColumns := map[string][]string{
		"relay_events":         {"device_id", "source_event_id"},
		"payment_observations": {"relay_event_id"},
		"amount_reservations":  {"payment_id"},
		"pairing_sessions":     {"token_hash"},
		"api_keys":             {"secret_hash"},
	}
	for table, expectedColumns := range requiredUniqueColumns {
		rows, err := raw.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", table))
		if err != nil {
			return fmt.Errorf("inspect restore table %s indexes: %w", table, err)
		}
		var uniqueNames []string
		for rows.Next() {
			var seq, unique, partial int
			var origin, indexName string
			if err := rows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
				rows.Close()
				return fmt.Errorf("scan restore table %s indexes: %w", table, err)
			}
			if unique != 0 {
				uniqueNames = append(uniqueNames, indexName)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate restore table %s indexes: %w", table, err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close restore table %s indexes: %w", table, err)
		}
		found := false
		for _, indexName := range uniqueNames {
			columns, err := readIndexColumns(indexName)
			if err != nil {
				return fmt.Errorf("inspect restore index %s: %w", indexName, err)
			}
			if sameIndexColumns(columns, expectedColumns) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("restore table %s is missing unique index on (%s)", table, strings.Join(expectedColumns, ", "))
		}
	}
	relayIndexRows, err := raw.QueryContext(ctx, `PRAGMA index_list(relay_devices)`)
	if err != nil {
		return fmt.Errorf("inspect relay device indexes: %w", err)
	}
	var relayUniqueNames []string
	for relayIndexRows.Next() {
		var seq, unique, partial int
		var origin, indexName string
		if err := relayIndexRows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
			relayIndexRows.Close()
			return fmt.Errorf("scan relay device indexes: %w", err)
		}
		if unique != 0 {
			relayUniqueNames = append(relayUniqueNames, indexName)
		}
	}
	if err := relayIndexRows.Err(); err != nil {
		relayIndexRows.Close()
		return fmt.Errorf("iterate relay device indexes: %w", err)
	}
	if err := relayIndexRows.Close(); err != nil {
		return fmt.Errorf("close relay device indexes: %w", err)
	}
	for _, indexName := range relayUniqueNames {
		columns, err := readIndexColumns(indexName)
		if err != nil {
			return fmt.Errorf("inspect relay device index %s: %w", indexName, err)
		}
		if sameIndexColumns(columns, []string{"enabled"}) {
			return errors.New("restore backup retains singleton relay_devices enabled uniqueness")
		}
	}
	requiredIndexDefinitions := map[string]string{
		"uq_collection_profiles_one_active": "CREATE UNIQUE INDEX uq_collection_profiles_one_active ON collection_profiles(active) WHERE active = 1",
		"idx_payments_external_id":          "CREATE INDEX idx_payments_external_id ON payments(external_id)",
		"idx_payments_status_created":       "CREATE INDEX idx_payments_status_created ON payments(status, created_at DESC)",
		"idx_payments_profile_payable":      "CREATE INDEX idx_payments_profile_payable ON payments(collection_profile_id, payable_amount_paise)",
		"uq_active_profile_payable":         "CREATE UNIQUE INDEX uq_active_profile_payable ON amount_reservations(collection_profile_id, payable_amount_paise) WHERE released_at IS NULL",
		"idx_amount_reservations_history":   "CREATE INDEX idx_amount_reservations_history ON amount_reservations(collection_profile_id, payable_amount_paise, reserved_at DESC)",
		"idx_amount_reservations_release":   "CREATE INDEX idx_amount_reservations_release ON amount_reservations(released_at, reserved_until)",
		"idx_relay_events_received":         "CREATE INDEX idx_relay_events_received ON relay_events(received_at DESC)",
		"idx_observations_amount_time":      "CREATE INDEX idx_observations_amount_time ON payment_observations(collection_profile_id, amount_paise, occurred_at)",
		"idx_payment_history_payment":       "CREATE INDEX idx_payment_history_payment ON payment_history(payment_id, created_at)",
		"idx_webhook_delivery_queue":        "CREATE INDEX idx_webhook_delivery_queue ON webhook_deliveries(status, next_attempt_at, created_at)",
		"idx_admin_sessions_expiry":         "CREATE INDEX idx_admin_sessions_expiry ON admin_sessions(expires_at)",
	}
	if versions[len(versions)-1] >= 4 {
		requiredIndexDefinitions["idx_observations_payment"] = "CREATE INDEX idx_observations_payment ON payment_observations(matched_payment_id, occurred_at) WHERE matched_payment_id IS NOT NULL"
	}
	if versions[len(versions)-1] >= 5 {
		delete(requiredIndexDefinitions, "uq_active_profile_payable")
		requiredIndexDefinitions["uq_active_payable"] = "CREATE UNIQUE INDEX uq_active_payable ON amount_reservations(payable_amount_paise) WHERE released_at IS NULL"
		if versions[len(versions)-1] >= 7 {
			requiredIndexDefinitions["uq_active_payable"] = "CREATE UNIQUE INDEX uq_active_payable ON amount_reservations(payable_amount_paise) WHERE released_at IS NULL AND global_unique_enforced=1"
		}
	}
	canonicalSQL := func(value string) string {
		return strings.Join(strings.Fields(strings.ToUpper(value)), " ")
	}
	requiredTriggers := map[string][]string{}
	if versions[len(versions)-1] >= 7 {
		requiredTriggers["trg_amount_reservations_global_unique_insert"] = []string{
			"BEFORE INSERT ON AMOUNT_RESERVATIONS", "NEW.GLOBAL_UNIQUE_ENFORCED<>1", "NEW.RELEASED_AT IS NULL",
			"PAYABLE_AMOUNT_PAISE=NEW.PAYABLE_AMOUNT_PAISE", "RAISE(ABORT",
		}
		requiredTriggers["trg_amount_reservations_global_unique_update"] = []string{
			"BEFORE UPDATE OF PAYABLE_AMOUNT_PAISE,RELEASED_AT,GLOBAL_UNIQUE_ENFORCED ON AMOUNT_RESERVATIONS",
			"OLD.GLOBAL_UNIQUE_ENFORCED=1", "NEW.GLOBAL_UNIQUE_ENFORCED<>1", "ID<>NEW.ID", "RAISE(ABORT",
		}
	}

	for name, expected := range requiredIndexes {
		var tableName, indexSQL string
		if err := raw.QueryRowContext(ctx, `SELECT tbl_name,sql FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&tableName, &indexSQL); err != nil {
			return fmt.Errorf("read restore index %s: %w", name, err)
		}
		if !strings.EqualFold(tableName, expected.table) {
			return fmt.Errorf("restore index %s belongs to table %s, want %s", name, tableName, expected.table)
		}
		indexRows, err := raw.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", expected.table))
		if err != nil {
			return fmt.Errorf("inspect restore index %s metadata: %w", name, err)
		}
		var metadataFound, metadataUnique bool
		for indexRows.Next() {
			var seq, unique, partial int
			var indexName, origin string
			if err := indexRows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
				indexRows.Close()
				return fmt.Errorf("scan restore index %s metadata: %w", name, err)
			}
			if indexName == name {
				metadataFound = true
				metadataUnique = unique != 0
			}
		}
		if err := indexRows.Err(); err != nil {
			indexRows.Close()
			return fmt.Errorf("iterate restore index %s metadata: %w", name, err)
		}
		if err := indexRows.Close(); err != nil {
			return fmt.Errorf("close restore index %s metadata: %w", name, err)
		}
		if !metadataFound {
			return fmt.Errorf("restore index %s is not listed by SQLite", name)
		}
		if expected.unique != metadataUnique {
			return fmt.Errorf("restore index %s uniqueness mismatch", name)
		}
		columns, err := readIndexColumns(name)
		if err != nil {
			return fmt.Errorf("inspect restore index %s columns: %w", name, err)
		}
		if !sameIndexColumns(columns, expected.columns) {
			return fmt.Errorf("restore index %s has columns (%s), want (%s)", name, strings.Join(columns, ", "), strings.Join(expected.columns, ", "))
		}
		upperIndexSQL := strings.ToUpper(indexSQL)
		if expected.unique != strings.Contains(upperIndexSQL, "CREATE UNIQUE INDEX") {
			return fmt.Errorf("restore index %s uniqueness mismatch", name)
		}
		for _, fragment := range expected.fragments {
			if !strings.Contains(upperIndexSQL, fragment) {
				return fmt.Errorf("restore index %s is missing definition fragment %s", name, fragment)
			}
		}
		if want := requiredIndexDefinitions[name]; want != "" && canonicalSQL(indexSQL) != canonicalSQL(want) {
			return fmt.Errorf("restore index %s definition mismatch", name)
		}
	}
	for name, fragments := range requiredTriggers {
		var tableName, triggerSQL string
		if err := raw.QueryRowContext(ctx, `SELECT tbl_name,sql FROM sqlite_master WHERE type='trigger' AND name=?`, name).Scan(&tableName, &triggerSQL); err != nil {
			return fmt.Errorf("read restore trigger %s: %w", name, err)
		}
		if !strings.EqualFold(tableName, "amount_reservations") {
			return fmt.Errorf("restore trigger %s belongs to table %s, want amount_reservations", name, tableName)
		}
		upperTriggerSQL := strings.ToUpper(triggerSQL)
		for _, fragment := range fragments {
			if !strings.Contains(upperTriggerSQL, fragment) {
				return fmt.Errorf("restore trigger %s is missing definition fragment %s", name, fragment)
			}
		}
	}
	return nil
}

func inspectRestoredDatabase(ctx context.Context, db *DB) (RestoreReport, error) {
	var report RestoreReport
	var integrity string
	if err := db.SQL.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return RestoreReport{}, fmt.Errorf("restore integrity check: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(integrity)) != "ok" {
		return RestoreReport{}, fmt.Errorf("restore integrity check returned %q", integrity)
	}
	rows, err := db.SQL.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return RestoreReport{}, fmt.Errorf("restore foreign-key check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return RestoreReport{}, errors.New("restore foreign-key check found violations")
	}
	if err := rows.Err(); err != nil {
		return RestoreReport{}, fmt.Errorf("iterate restore foreign-key check: %w", err)
	}

	if err := db.SQL.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&report.SchemaVersion); err != nil {
		return RestoreReport{}, fmt.Errorf("read restored schema version: %w", err)
	}
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM payments`:           &report.Payments,
		`SELECT COUNT(*) FROM relay_events`:       &report.RelayEvents,
		`SELECT COUNT(*) FROM webhook_deliveries`: &report.WebhookDeliveries,
		`SELECT COUNT(*) FROM relay_devices`:      &report.RelayDevices,
	} {
		if err := db.SQL.QueryRowContext(ctx, query).Scan(target); err != nil {
			return RestoreReport{}, fmt.Errorf("read restored table count: %w", err)
		}
	}
	return report, nil
}
