# 07 — Direct SQLite, Security and Operations

## Decision: SQLite only

The v4 production data layer is:

```text
Go
+ database/sql
+ modernc.org/sqlite
+ one local SQLite database
```

There is **no PocketBase runtime or PocketBase domain model** in v4.

This is not a new database engine for the project: the current `go.mod` already carries `modernc.org/sqlite` through the existing stack. v4 uses SQLite directly and removes the framework layer around it.

Current repository version at planning time:

```text
modernc.org/sqlite v1.54.0
```

Pin the exact driver/libc versions in `go.mod` and update deliberately through normal dependency PRs.

## Why SQLite is the right database here

SQLite's own guidance explicitly supports the model where an application server and its database file are on the same machine and remote users talk to the application server rather than SQLite directly.

That is exactly PayGate:

```text
Internet / Android
        |
    PayGate Go server
        |
 local paygate.db
```

PayGate has one authoritative server writer, modest transaction volume and strong value from a small operational footprint.

SQLite documentation:

- appropriate use: https://www.sqlite.org/whentouse.html
- transactions: https://sqlite.org/lang_transaction.html
- WAL: https://www.sqlite.org/wal.html
- backup: https://www.sqlite.org/backup.html

A client/server database becomes justified only if the architecture changes to multiple independent writable PayGate servers or sustained write contention that SQLite cannot meet.

## Process ownership invariant

Production keeps the strongest lesson from the v3 SQLite incident:

```text
exactly one PayGate server process owns the live database
```

Keep a non-blocking process/file lock acquired **before** opening the database for normal service.

No cron job, backup verifier, migration helper or CLI command may start a second PayGate runtime against the live DB.

Maintenance is either:

- performed by the running server through a narrowly scoped internal operation; or
- performed on a completed backup copy.

## Filesystem rule

`paygate.db`, `paygate.db-wal` and `paygate.db-shm` live on local durable storage attached to the PayGate host.

Do not put the live SQLite database on NFS/SMB/network-mounted storage. WAL assumes all participating processes are on the same host/shared-memory environment.

## SQLite startup configuration

For a payment gateway, prefer durability over shaving a few milliseconds from commits.

Required/verified configuration:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

Why:

- WAL allows normal reads to proceed while a writer is active, but SQLite still intentionally has one writer at a time;
- `synchronous=FULL` in WAL adds a WAL sync for each commit and provides the stronger power-loss durability appropriate for payment state;
- foreign keys are connection-scoped and must be enabled on every connection;
- a bounded busy timeout absorbs brief writer contention instead of failing immediately with `SQLITE_BUSY`.

SQLite notes that unknown PRAGMAs may be silently ignored, so startup must **query and verify** critical effective settings rather than assuming a typo succeeded.

Do not use `synchronous=OFF` or memory journaling for production payment data.

References:

- https://www.sqlite.org/pragma.html
- https://sqlite.org/c3ref/busy_timeout.html

## Connection policy

Use a small bounded `database/sql` pool; do not open an unbounded number of SQLite connections.

Initial implementation should remain conservative and be load-tested before tuning.

Rules:

- normal read queries should be short and do not need explicit transactions unless a consistent multi-query snapshot is required;
- write transactions are short;
- never hold a write transaction across HTTP calls, Android communication, webhook delivery, DNS, filesystem uploads or other network I/O;
- close rows/statements promptly;
- retry `SQLITE_BUSY` only within a bounded policy; never spin forever.

## Write transaction policy

Critical state mutations should acquire the write lock at the beginning of the transaction rather than read first and discover contention halfway through.

SQLite provides `BEGIN IMMEDIATE` for this purpose.

Use it for operations such as:

- payment creation + amount reservation + idempotency row;
- payment match + history + webhook outbox;
- collection-profile switch;
- operator state correction + history + webhook outbox;
- pairing-token consume + device enrollment.

Do not globally turn every read-only transaction into `BEGIN IMMEDIATE`.

`modernc.org/sqlite` supports `_txlock=immediate`; implementation may use a narrowly scoped writer helper/connection so normal reads remain deferred/non-transactional.

Reference: https://sqlite.org/lang_transaction.html

## Schema rules

Use SQLite `STRICT` tables for v4 domain tables where practical.

SQLite STRICT tables enforce declared storage types and reduce accidental type coercion.

Reference: https://www.sqlite.org/stricttables.html

General rules:

- money: INTEGER paise only;
- timestamps: INTEGER Unix milliseconds UTC;
- booleans: INTEGER with `CHECK(value IN (0,1))`;
- enums/statuses: TEXT with explicit `CHECK` constraints;
- JSON: TEXT validated/normalized by the application;
- IDs/tokens: generated in Go with cryptographic randomness; no semantic data encoded into IDs;
- foreign keys with explicit delete/update behavior;
- use UNIQUE constraints for correctness, not only lookup code.

## Proposed schema

### `payments`

```text
id TEXT PRIMARY KEY
name TEXT NOT NULL                  -- merchant-supplied person/payee identifier
external_id TEXT                    -- merchant event ID; deliberately non-unique
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
payer_name TEXT                     -- actual notification-derived sender
payer_upi_id TEXT
internal_note TEXT
```

Important constraints:

- requested/payable amounts > 0;
- payable - requested = adjustment;
- payable paise component is never `.00` for automatically allocated payments;
- status restricted to product states;
- lifecycle timestamps ordered correctly.

`external_id` is indexed for event filtering but is **not unique**.

### `amount_reservations`

Keep reservation concurrency separate from historical payment rows:

```text
id TEXT PRIMARY KEY
collection_profile_id TEXT NOT NULL
payable_amount_paise INTEGER NOT NULL
payment_id TEXT NOT NULL UNIQUE
reserved_at INTEGER NOT NULL
reserved_until INTEGER NOT NULL
released_at INTEGER
last_used_at INTEGER NOT NULL
```

Database-enforced active uniqueness:

```sql
CREATE UNIQUE INDEX uq_active_profile_amount
ON amount_reservations(collection_profile_id, payable_amount_paise)
WHERE released_at IS NULL;
```

Before allocation, due reservations can be marked released inside the same write transaction.

Historical rows remain available for recent-use avoidance and delayed-notification reasoning.

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

Enforce one active profile with a partial unique index:

```sql
CREATE UNIQUE INDEX uq_one_active_profile
ON collection_profiles(active)
WHERE active = 1;
```

Application logic additionally requires the active profile to be enabled and ready.

### `idempotency_keys`

Do not overload event `external_id` for idempotency.

```text
merchant_key_id
idempotency_key
request_hash
payment_id
created_at
expires_at
UNIQUE(merchant_key_id, idempotency_key)
```

Creation of the idempotency row and payment occurs atomically.

### `relay_devices`

```text
id TEXT PRIMARY KEY
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

v4.0 should prefer **one active payment-relay device** to avoid duplicate phone streams. Pairing a replacement phone should be an explicit operator action.

### `pairing_sessions`

```text
id TEXT PRIMARY KEY
token_hash BLOB UNIQUE NOT NULL
created_at INTEGER NOT NULL
expires_at INTEGER NOT NULL
consumed_at INTEGER
```

Token plaintext never needs to be stored.

### `relay_events`

```text
id TEXT PRIMARY KEY
device_id TEXT NOT NULL
source_event_id TEXT NOT NULL
package_name TEXT NOT NULL
notification_posted_at INTEGER NOT NULL
server_received_at INTEGER NOT NULL
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
occurred_at_source TEXT NOT NULL
received_at INTEGER NOT NULL
matched_payment_id TEXT
match_result TEXT NOT NULL
```

### `payment_history`

Append-only timeline:

```text
id
payment_id
type
actor
summary
changes_json
created_at
```

Do not update/delete history in normal product flows.

### `webhook_deliveries`

Durable outbox + delivery record:

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

Store key identifier + verifier/hash. Do not persist plaintext merchant API keys.

### `admin_credentials`

Singleton password hash row.

### `admin_sessions`

Opaque session-token hash + expiry/revocation metadata.

### `settings`

Only mutable product settings that genuinely belong in the DB, such as:

- active profile linkage/config where not represented relationally;
- allocator max adjustment;
- soft recent-use horizon;
- webhook endpoint configuration.

Do not create an untyped dumping ground for domain state.

### `schema_migrations`

```text
version INTEGER PRIMARY KEY
name TEXT NOT NULL
applied_at INTEGER NOT NULL
checksum TEXT NOT NULL
```

Server refuses to start if the DB schema is newer than the binary understands.

## Indexing strategy

Start with indexes dictated by real access paths/invariants:

- active reservation unique index above;
- `payments(created_at)`;
- `payments(status, created_at)`;
- `payments(external_id, created_at)`;
- `payments(name)` only if search needs it;
- `payment_observations(collection_profile_id, amount_paise, occurred_at)`;
- `relay_events(device_id, source_event_id)` unique;
- pending webhook due index `(status, next_attempt_at)`;
- payment-history `(payment_id, created_at)`.

Do not pre-create dozens of speculative indexes. Use query plans and actual dashboard traffic.

SQLite supports partial indexes and they are useful for small hot subsets such as active reservations/pending work: https://www.sqlite.org/partialindex.html

## Migrations

Rules:

1. every schema change has a monotonic migration version;
2. migrations run before serving traffic;
3. migration runs inside a transaction when SQLite permits;
4. create a verified pre-migration backup for production upgrades;
5. destructive table rewrites are tested on production-sized backup copies;
6. never automatically downgrade a schema;
7. binary refuses unknown future schema versions.

The v3 -> v4 conversion is a one-time offline migrator into a **new database file**, not an in-place PocketBase mutation.

## WAL/checkpoint management

SQLite's default WAL autocheckpoint is around 1000 pages and is a reasonable starting point. Do not invent aggressive checkpoint tuning before observing real WAL size/latency.

Monitor WAL growth.

A periodic in-process `PRAGMA wal_checkpoint(PASSIVE)` may be used if required; PASSIVE does not wait for readers/writers to finish.

Never run a separate PayGate CLI just to checkpoint the live DB.

References:

- https://www.sqlite.org/wal.html
- https://www.sqlite.org/pragma.html#pragma_wal_checkpoint

## Query planner maintenance

SQLite 3.46+ recommends `PRAGMA optimize` rather than manually running broad `ANALYZE` jobs.

For long-lived connections, run the documented startup/periodic optimize pattern conservatively, especially after index/schema changes.

Reference: https://www.sqlite.org/pragma.html#pragma_optimize

## Backups

Never `cp paygate.db` while the database is actively changing and assume that is a complete WAL-aware backup.

Use SQLite's Online Backup API from the already-running PayGate process.

`modernc.org/sqlite` exposes an online `Backup` API, so v4 can do this without spawning another database-owning process.

Flow:

```text
PayGate scheduler/internal operation
  -> open destination backup DB
  -> SQLite online backup in bounded page steps
  -> finish/close destination
  -> PRAGMA integrity_check on destination
  -> fsync/atomic rename to completed artifact

host exporter
  -> sees completed artifact only
  -> checksum
  -> retention/off-host upload
```

SQLite documents that the source is only read-locked while pages are copied, allowing normal users to continue between backup steps.

Reference: https://www.sqlite.org/backup.html

## Backup verification

At minimum verify on the backup copy:

```sql
PRAGMA integrity_check;
PRAGMA foreign_key_check;
```

Plus application checks:

- schema migration version expected;
- payment count plausible;
- latest payments/history readable;
- collection profile state valid;
- one active relay device invariant valid;
- pending webhook counts readable.

Do not run a full integrity scan on every request/health check.

## Restore drills

Always restore to an isolated path/container.

Never point a second test process at live `paygate.db`.

Restore acceptance:

- integrity check `ok`;
- foreign-key check clean;
- expected migration version;
- application starts;
- representative payment/history/webhook queries work;
- test writes can commit in the isolated copy.

## Sensitive-data retention

Keep payment/history records according to business/audit needs, but aggressively bound notification content.

Recommended policy to validate during implementation:

- relay raw title/text/bigText: days, not permanent;
- normalized payer fields: retained with payment when operationally useful;
- unmatched raw notification data: short retention;
- delivered local Android queue: short retention;
- failed local rows: longer but bounded diagnostics retention;
- pairing sessions: purge quickly after use/expiry;
- admin sessions: purge after expiry/revocation;
- delivered webhook bodies: bounded retention after operational window.

If we need stronger deletion hygiene for notification text, evaluate `PRAGMA secure_delete=FAST` against write cost and WAL/backup behavior rather than assuming row deletion instantly removes every forensic copy.

Reference: https://www.sqlite.org/pragma.html#pragma_secure_delete

## Security boundaries

### Admin

One password, hashed with Argon2id. Password change invalidates existing admin sessions.

Web session cookie:

```text
Secure
HttpOnly
SameSite=Strict
Path=/admin
```

### Android device

Keep ECDSA signing and Android Keystore private key. Pairing token is short-lived and single-use; server stores public key only.

### Merchant API

Separate merchant API key. Never reuse the admin password or Android device credential.

### Webhook

Separate signing secret. Payment status mutation and outbox row are atomic; delivery is asynchronous.

## Operational health

Useful health fields:

```text
db=ok
schema_version=...
active_profile=...
active_reservations=...
relay_device=healthy
last_heartbeat_at=...
pending_webhooks=...
backup_last_verified_at=...
wal_size=...
```

Do not log full notification bodies by default.

## SQLite failure/edge cases that must be tested

- two concurrent payment creations contend for writer lock;
- writer gets `SQLITE_BUSY` beyond timeout and API returns safe retryable failure;
- crash during payment+reservation transaction leaves neither partial payment nor orphan active reservation;
- crash after payment commit but before webhook worker runs still leaves durable outbox row;
- server restart reopens WAL cleanly;
- unclean shutdown with WAL recovers without state mismatch;
- long dashboard read does not hold a transaction while waiting on external work;
- migration failure rolls back and old DB remains untouched;
- backup while writes continue produces a valid snapshot;
- backup destination disk-full fails without damaging source;
- host disk-low condition stops accepting unsafe new work before corruption risk;
- database file permissions prevent unrelated users/processes from writing it;
- only local filesystem paths are accepted for the live DB;
- unknown/corrupt schema version fails startup closed;
- foreign-key/STRICT/CHECK violations are surfaced as programming errors, not silently normalized.

## V4 container/runtime boundary — 2026-09-01

`Dockerfile.v4` is the sole production-image path. It builds only the v4 dashboard plus `paygate-v4` and `paygate-v4-migrate`, then runs the server as the distroless `nonroot` user. The pre-v4 image definition was removed after cutover.

The v4 image has a built-in `paygate-v4 healthcheck` command that calls its own `/health` route over loopback. A disposable image acceptance run verified Docker reports the container `healthy` with a fresh temporary SQLite database.

The embedded v4 dashboard applies CSP, frame denial, no-referrer, noindex and no-store headers. Static SPA fallback is mounted only after the health, merchant, admin, relay and Digital Asset Links routes, so a frontend route cannot shadow a server API.

Only a tiny placeholder `internal/v4web/dist/index.html` belongs in Git. CI/container builds replace it with generated hashed assets before compiling the production v4 binary; generated `dist/assets/*` remain ignored.
