# 07 — Data Storage, Security and Operations

## Database decision

### Important distinction

PocketBase is not an alternative database to SQLite. PocketBase is a backend framework that uses SQLite underneath.

Current PayGate is therefore effectively:

```text
Go + PocketBase framework + SQLite
```

The v4 target is:

```text
Go + small PayGate domain/repository layer + SQLite
```

### Why remove PocketBase

PocketBase provides useful auth/admin/realtime features, but v4 does not need most of them:

- one operator, not a general auth/user system;
- custom payment API;
- custom admin dashboard;
- no realtime subscription requirement;
- custom migrations/domain rules already dominate the current code.

PocketBase's own documentation currently states that it is still pre-v1 and is not recommended for production-critical applications unless the operator accepts compatibility/migration work.

Reference: https://pocketbase.io/docs/

For a payment gateway, removing this unnecessary framework dependency is a reasonable v4 simplification.

### Why keep SQLite

SQLite remains appropriate because:

- one PayGate server process owns the database;
- clients never connect to SQLite directly over the network;
- write volume is small/moderate;
- transactions are short;
- operational simplicity matters;
- a single-file database makes verified backup/rollback straightforward.

SQLite explicitly documents server-side application databases as an appropriate use when the application server owns the database; it recommends client/server databases when many independent writers or multiple servers need concurrent writes.

Reference: https://www.sqlite.org/whentouse.html

### Why not PostgreSQL now

PostgreSQL would add:

- another daemon/container;
- credentials/networking;
- schema/backup tooling;
- restore/upgrade operations;
- more moving parts on the Oracle VM.

It does not solve a demonstrated PayGate problem today.

Migrate to PostgreSQL only when one of these becomes true:

- PayGate requires multiple server replicas that can write concurrently;
- sustained write contention becomes measurable;
- database size/analytics outgrow the single-host model;
- high availability requires database failover independent of the app host.

## SQLite configuration

Use one database file on local durable storage.

On every connection configure/verify:

```text
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=<bounded milliseconds>;
```

WAL allows readers and the writer to proceed concurrently, while still permitting only one writer at a time. This matches the single-server architecture.

Reference: https://www.sqlite.org/wal.html

Keep transactions small and never hold a write transaction while making network calls.

## Server process ownership

Production invariant:

```text
exactly one PayGate process owns the live database
```

Continue the existing stop-first deployment rule.

Do not restore the old pattern where a second CLI invocation opens the production database for maintenance.

Maintenance either:

- runs inside the already-running PayGate process through explicit internal/admin operations; or
- works exclusively on an offline backup/archive.

## Proposed v4 schema

### `payments`

```text
id TEXT PRIMARY KEY
name TEXT NOT NULL
external_id TEXT
metadata_json TEXT
requested_amount_paise INTEGER NOT NULL
payable_amount_paise INTEGER NOT NULL
adjustment_paise INTEGER NOT NULL
currency TEXT NOT NULL DEFAULT 'INR'
collection_profile_id TEXT NOT NULL
upi_id_snapshot TEXT NOT NULL
payee_name_snapshot TEXT
status TEXT NOT NULL
created_at INTEGER NOT NULL
expires_at INTEGER NOT NULL
grace_until INTEGER NOT NULL
reuse_after INTEGER NOT NULL
paid_at INTEGER
payer_name TEXT
payer_upi_id TEXT
internal_note TEXT
```

Indexes:

- `(collection_profile_id, payable_amount_paise, reuse_after)`
- `status`
- `created_at`
- `external_id`
- normalized/search helper indexes after actual query profiling

### `collection_profiles`

```text
id TEXT PRIMARY KEY
label TEXT NOT NULL
upi_id TEXT NOT NULL
payee_name TEXT
parser TEXT NOT NULL
enabled INTEGER NOT NULL
active INTEGER NOT NULL
created_at INTEGER NOT NULL
updated_at INTEGER NOT NULL
```

Application transaction enforces one active enabled profile.

### `relay_devices`

```text
id TEXT PRIMARY KEY            -- public-key fingerprint
name TEXT
public_key_pem TEXT NOT NULL
enabled INTEGER NOT NULL
enrolled_at INTEGER NOT NULL
last_seen_at INTEGER
last_heartbeat_at INTEGER
app_version TEXT
device_model TEXT
android_version TEXT
```

### `pairing_sessions`

```text
id TEXT PRIMARY KEY
token_hash BLOB UNIQUE NOT NULL
created_at INTEGER NOT NULL
expires_at INTEGER NOT NULL
consumed_at INTEGER
```

Store only a hash of the QR pairing token.

### `relay_events`

```text
id TEXT PRIMARY KEY
device_id TEXT NOT NULL
source_event_id TEXT NOT NULL
package_name TEXT NOT NULL
posted_at INTEGER NOT NULL
received_at INTEGER NOT NULL
amount_hint_paise INTEGER
title TEXT
text TEXT
big_text TEXT
status TEXT NOT NULL
error TEXT
UNIQUE(device_id, source_event_id)
```

Raw text is bounded and short-retention.

### `payment_observations`

```text
id TEXT PRIMARY KEY
relay_event_id TEXT UNIQUE NOT NULL
source TEXT NOT NULL
collection_profile_id TEXT NOT NULL
amount_paise INTEGER NOT NULL
payer_name TEXT
payer_upi_id TEXT
occurred_at INTEGER NOT NULL
received_at INTEGER NOT NULL
matched_payment_id TEXT
match_result TEXT NOT NULL
```

### `payment_history`

Immutable append-only timeline of payment transitions/operator edits.

```text
id
payment_id
type
actor            system/admin
summary
changes_json
created_at
```

### `webhook_deliveries`

Durable outbox + delivery history:

```text
id
event_type
payment_id
payload_json
status
attempts
next_attempt_at
last_http_status
last_error
created_at
delivered_at
```

### `api_keys`

Store API-key identifier + cryptographic hash, not plaintext secret.

### `admin_credentials`

Singleton row containing only the password hash and update timestamp.

### `admin_sessions`

Opaque random sessions; store a hash of each token server-side with expiry/revocation metadata.

### `settings`

Small key/value or singleton config table only for settings that must be mutable from UI (webhook configuration, allocator cap, etc.). Do not turn this into an untyped dumping ground.

## Admin password

User-facing authentication is password-only.

Initial bootstrap:

1. if no admin credential exists, require a one-time `PAYGATE_BOOTSTRAP_PASSWORD`/setup path;
2. immediately hash with Argon2id and store only the hash;
3. allow password changes through Settings;
4. invalidate existing admin sessions after a password change.

Do not have a username/email field in the UI.

## Sessions

Use high-entropy opaque random session IDs, not a custom data-bearing token format.

Web:

```text
__Host-paygate_session=<opaque token>;
Secure;
HttpOnly;
SameSite=Strict;
Path=/
```

Store only the token hash in SQLite.

OWASP recommends unpredictable session identifiers with at least 128 bits of entropy and Secure/HttpOnly/SameSite cookie protections.

Reference: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html

## Device security

Preserve ECDSA request signing and Android Keystore private-key storage.

The device signing key must be usable while the screen is locked because payment monitoring is a background capability.

Server stores public keys only.

Pairing token is separate, short-lived and single-use.

## API keys and webhook secrets

### Merchant API key

- generate cryptographically random value;
- display plaintext only at creation/rotation;
- store verifier/hash, not plaintext;
- support one key initially if sufficient.

### Webhook secret

PayGate must sign outbound webhooks, so it needs access to the secret value.

Encrypt mutable webhook secrets at rest using a server master key supplied through deployment secrets/environment. Do not store the master key in SQLite/backups.

## CSRF/XSS

Admin web routes use same-origin cookies and explicit CSRF protection for state-changing requests.

Use a strict Content Security Policy where practical. Avoid injecting raw notification/message text as HTML.

## Backups

Do not raw-copy the database file while WAL writes are active.

Use the SQLite Online Backup API from the live PayGate process to create a consistent backup database/archive, then let the host exporter copy/archive that completed artifact.

Reference: https://www.sqlite.org/backup.html

Recommended flow:

```text
PayGate internal scheduler
  -> SQLite online backup to staging path
  -> integrity_check on backup
  -> atomic rename to completed backup

host exporter
  -> copies only completed backup artifact
  -> checksum
  -> retention/upload
```

This prevents another executable from opening live storage.

## Restore drill

Restore drills always use a copy in an isolated directory/container.

Validation:

- `PRAGMA integrity_check` is `ok`;
- required tables/migrations present;
- payment/profile counts plausible;
- latest payment/history/webhook rows readable;
- isolated PayGate instance can start against restored copy.

Never point a restore-drill process at production `paygate.db`.

## Retention

Suggested defaults:

- payments/history: long-lived unless business policy says otherwise;
- normalized observations: 90 days or according to operational need;
- raw notification excerpts: 30 days maximum by default;
- delivered local Android relay rows: short retention (for example 7 days);
- failed Android rows: longer bounded retention for diagnostics;
- admin sessions: purge after expiry;
- consumed/expired pairing sessions: purge quickly;
- old webhook bodies: bounded retention after successful delivery.

Personal notification text should not become permanent storage merely because it passed through the relay.

## Observability

Expose simple health fields:

```text
db=ok
active_profile=paytm
relay_device=healthy
last_heartbeat_at=...
pending_webhooks=...
backup_last_verified_at=...
```

Operational logs should use payment/event IDs and error codes, not full raw SMS/notification bodies by default.