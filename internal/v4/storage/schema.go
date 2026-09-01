package storage

import (
	"context"
	"database/sql"
	"fmt"
)

func (db *DB) migrate(ctx context.Context) error {
	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
) STRICT;
`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var current int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current > schemaVersion {
		return fmt.Errorf("database schema %d is newer than supported %d", current, schemaVersion)
	}
	if current < 1 {
		if err := applyV1(ctx, tx); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(1, unixepoch('subsec') * 1000)`); err != nil {
			return fmt.Errorf("record schema v1: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migration: %w", err)
	}
	return nil
}

func applyV1(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, schemaV1); err != nil {
		return fmt.Errorf("apply schema v1: %w", err)
	}
	return nil
}

const schemaV1 = `
CREATE TABLE collection_profiles (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL CHECK(length(trim(label)) BETWEEN 1 AND 120),
    upi_id TEXT NOT NULL CHECK(length(trim(upi_id)) BETWEEN 3 AND 255),
    payee_name TEXT,
    parser TEXT NOT NULL CHECK(parser IN ('paytm_notification','kotak_sms')),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    active INTEGER NOT NULL DEFAULT 0 CHECK(active IN (0,1)),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK(active = 0 OR enabled = 1)
) STRICT;
CREATE UNIQUE INDEX uq_collection_profiles_one_active ON collection_profiles(active) WHERE active = 1;

CREATE TABLE payments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 120),
    external_id TEXT,
    metadata_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(metadata_json)),
    requested_amount_paise INTEGER NOT NULL CHECK(requested_amount_paise > 0 AND requested_amount_paise % 100 = 0),
    payable_amount_paise INTEGER NOT NULL CHECK(payable_amount_paise > requested_amount_paise AND payable_amount_paise % 100 BETWEEN 1 AND 99),
    adjustment_paise INTEGER NOT NULL CHECK(adjustment_paise BETWEEN 1 AND 199),
    currency TEXT NOT NULL DEFAULT 'INR' CHECK(currency = 'INR'),
    collection_profile_id TEXT NOT NULL REFERENCES collection_profiles(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    upi_id_snapshot TEXT NOT NULL,
    payee_name_snapshot TEXT,
    status TEXT NOT NULL CHECK(status IN ('pending','paid','expired','cancelled')),
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    grace_until INTEGER NOT NULL,
    reuse_after INTEGER NOT NULL,
    paid_at INTEGER,
    payer_name TEXT,
    payer_upi_id TEXT,
    internal_note TEXT,
    CHECK(payable_amount_paise = requested_amount_paise + adjustment_paise),
    CHECK(created_at < expires_at AND expires_at < grace_until AND grace_until < reuse_after),
    CHECK((status = 'paid' AND paid_at IS NOT NULL) OR (status != 'paid' AND paid_at IS NULL))
) STRICT;
CREATE INDEX idx_payments_external_id ON payments(external_id);
CREATE INDEX idx_payments_status_created ON payments(status, created_at DESC);
CREATE INDEX idx_payments_profile_payable ON payments(collection_profile_id, payable_amount_paise);

CREATE TABLE amount_reservations (
    id TEXT PRIMARY KEY,
    collection_profile_id TEXT NOT NULL REFERENCES collection_profiles(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    payable_amount_paise INTEGER NOT NULL CHECK(payable_amount_paise > 0 AND payable_amount_paise % 100 BETWEEN 1 AND 99),
    payment_id TEXT NOT NULL UNIQUE REFERENCES payments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    reserved_at INTEGER NOT NULL,
    reserved_until INTEGER NOT NULL CHECK(reserved_until > reserved_at),
    released_at INTEGER,
    last_used_at INTEGER NOT NULL,
    CHECK(released_at IS NULL OR released_at >= reserved_at)
) STRICT;
CREATE UNIQUE INDEX uq_active_profile_payable ON amount_reservations(collection_profile_id, payable_amount_paise) WHERE released_at IS NULL;
CREATE INDEX idx_amount_reservations_history ON amount_reservations(collection_profile_id, payable_amount_paise, reserved_at DESC);
CREATE INDEX idx_amount_reservations_release ON amount_reservations(released_at, reserved_until);

CREATE TABLE idempotency_keys (
    scope TEXT NOT NULL,
    key_hash BLOB NOT NULL,
    request_hash BLOB NOT NULL,
    payment_id TEXT NOT NULL REFERENCES payments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL CHECK(expires_at > created_at),
    PRIMARY KEY(scope, key_hash)
) STRICT;

CREATE TABLE relay_devices (
    id TEXT PRIMARY KEY,
    name TEXT,
    public_key_pem TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    enrolled_at INTEGER NOT NULL,
    last_seen_at INTEGER,
    last_heartbeat_at INTEGER,
    app_version TEXT,
    device_model TEXT,
    android_version TEXT
) STRICT;
CREATE UNIQUE INDEX uq_relay_devices_one_enabled ON relay_devices(enabled) WHERE enabled = 1;

CREATE TABLE pairing_sessions (
    id TEXT PRIMARY KEY,
    token_hash BLOB NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL CHECK(expires_at > created_at),
    consumed_at INTEGER,
    CHECK(consumed_at IS NULL OR (consumed_at >= created_at AND consumed_at <= expires_at))
) STRICT;

CREATE TABLE relay_events (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES relay_devices(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    source_event_id TEXT NOT NULL,
    package_name TEXT NOT NULL,
    posted_at INTEGER NOT NULL,
    received_at INTEGER NOT NULL,
    amount_hint_paise INTEGER CHECK(amount_hint_paise IS NULL OR (amount_hint_paise > 0 AND amount_hint_paise % 100 BETWEEN 1 AND 99)),
    title TEXT,
    text TEXT,
    big_text TEXT,
    status TEXT NOT NULL CHECK(status IN ('received','parsed','ignored','matched','unmatched','ambiguous','error')),
    error TEXT,
    UNIQUE(device_id, source_event_id)
) STRICT;
CREATE INDEX idx_relay_events_received ON relay_events(received_at DESC);

CREATE TABLE payment_observations (
    id TEXT PRIMARY KEY,
    relay_event_id TEXT NOT NULL UNIQUE REFERENCES relay_events(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    source TEXT NOT NULL CHECK(source IN ('paytm_notification','kotak_sms')),
    collection_profile_id TEXT NOT NULL REFERENCES collection_profiles(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    amount_paise INTEGER NOT NULL CHECK(amount_paise > 0 AND amount_paise % 100 BETWEEN 1 AND 99),
    payer_name TEXT,
    payer_upi_id TEXT,
    occurred_at INTEGER NOT NULL,
    occurred_at_source TEXT NOT NULL CHECK(occurred_at_source IN ('notification_text','notification_posted_at','server_received_at')),
    received_at INTEGER NOT NULL,
    matched_payment_id TEXT REFERENCES payments(id) ON UPDATE RESTRICT ON DELETE SET NULL,
    match_result TEXT NOT NULL CHECK(match_result IN ('matched','unmatched','ambiguous','ignored','error'))
) STRICT;
CREATE INDEX idx_observations_amount_time ON payment_observations(collection_profile_id, amount_paise, occurred_at);

CREATE TABLE payment_history (
    id TEXT PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES payments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    type TEXT NOT NULL,
    actor TEXT NOT NULL CHECK(actor IN ('system','admin')),
    summary TEXT NOT NULL,
    changes_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(changes_json)),
    created_at INTEGER NOT NULL
) STRICT;
CREATE INDEX idx_payment_history_payment ON payment_history(payment_id, created_at);

CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL CHECK(event_type IN ('payment.created','payment.paid','payment.expired','payment.cancelled','payment.updated')),
    payment_id TEXT NOT NULL REFERENCES payments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    payload_json TEXT NOT NULL CHECK(json_valid(payload_json)),
    status TEXT NOT NULL CHECK(status IN ('pending','retry','delivered','exhausted')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts >= 0),
    next_attempt_at INTEGER,
    last_http_status INTEGER,
    last_error TEXT,
    created_at INTEGER NOT NULL,
    delivered_at INTEGER,
    CHECK((status = 'delivered' AND delivered_at IS NOT NULL) OR (status != 'delivered' AND delivered_at IS NULL))
) STRICT;
CREATE INDEX idx_webhook_delivery_queue ON webhook_deliveries(status, next_attempt_at, created_at);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    secret_hash BLOB NOT NULL UNIQUE,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    created_at INTEGER NOT NULL,
    last_used_at INTEGER
) STRICT;

CREATE TABLE admin_credentials (
    singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
    password_hash TEXT NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE admin_sessions (
    token_hash BLOB PRIMARY KEY,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL CHECK(expires_at > created_at),
    last_seen_at INTEGER,
    revoked_at INTEGER
) STRICT;
CREATE INDEX idx_admin_sessions_expiry ON admin_sessions(expires_at);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;
`
