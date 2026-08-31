# 08 — Migration and Implementation Plan

## Strategy

Do not rewrite production in place.

Build v4 beside v3, validate it against production-shaped backup data and real phone notification fixtures, then cut over with a rollback path.

The final architecture is simpler, but the migration must be deliberately conservative.

## Phase 0 — Freeze semantics

Before implementation, confirm these product rules:

- `name` = merchant-supplied person/payee identifier, not event title;
- `external_id` = merchant/event ID and is allowed to repeat across many payments;
- idempotency uses `Idempotency-Key`, not `external_id`;
- Paytm + Kotak only for v4.0;
- GPay and Slice deferred;
- merchant never selects collection profile;
- PayGate returns UPI URI string; frontend renders QR;
- no hosted checkout requirement;
- no UTR/RRN matching dependency;
- randomized bounded amount allocation;
- default max adjustment initially ₹1.99 unless explicitly changed;
- 5m active + 5m grace + 5m hard quarantine;
- soft recent-use avoidance after hard release;
- direct SQLite only; no PocketBase runtime;
- one active Android relay device for v4.0;
- Razorpay-inspired dark navy/blue UI.

Capture sanitized real notification fixtures before parser implementation.

## Phase 1 — Direct SQLite foundation

Build a new v4 storage package around:

```text
database/sql
modernc.org/sqlite
```

The current repository already includes `modernc.org/sqlite`, so v4 does not introduce a new database engine.

Implement:

- startup process lock;
- SQLite open/config verification;
- WAL;
- `synchronous=FULL`;
- `foreign_keys=ON`;
- bounded busy timeout;
- STRICT tables/check constraints where practical;
- schema migration table;
- direct repository/query layer;
- Online Backup API wrapper;
- backup integrity/foreign-key verification.

### Phase 1 tests

- startup refuses second PayGate process;
- critical PRAGMA values are queried/verified;
- schema v1 creates from empty DB;
- unknown future schema fails startup;
- migration rollback leaves source DB usable;
- unclean restart/WAL recovery test;
- online backup while writes continue;
- backup disk-full/error does not damage source;
- restore copy passes integrity/foreign-key checks.

## Phase 2 — Core payment tables/invariants

Implement:

- collection profiles;
- payments;
- amount reservations;
- idempotency keys;
- payment history;
- relay devices/pairing sessions;
- relay events;
- payment observations;
- webhook outbox;
- admin credentials/sessions/API keys/settings.

Critical DB invariants:

- one active profile;
- one active reservation per `(profile, payable_amount)`;
- one reservation per payment;
- unique relay event per `(device,event_id)`;
- unique merchant idempotency key within API-key scope;
- event `external_id` deliberately non-unique;
- foreign keys enforced.

## Phase 3 — Randomized allocator and lifecycle

Implement allocation inside a short write transaction:

1. resolve active profile;
2. release reservations due by `reuse_after`;
3. create bounded candidate list;
4. remove `.00`;
5. remove active reservations;
6. prefer candidates outside soft recent-use horizon;
7. cryptographically choose from preferred/free pool;
8. insert payment + reservation + idempotency result atomically.

Use Go `crypto/rand`.

### Allocator test matrix

- output never `.00`;
- candidate always inside cap;
- allocation is not sequential/predictable `.01`, `.02`, `.03`;
- adjacent requested amounts cannot collide;
- concurrent creates cannot receive same profile+amount;
- random selection retries safely if DB uniqueness is hit;
- soft recent-use avoidance prefers older free values;
- recent values remain usable if pool pressure requires them;
- hard reservation release timing exact;
- crash before commit leaves no partial payment/reservation;
- capacity exhaustion returns retryable error;
- profile switch during concurrent creation yields transactionally consistent snapshot.

## Phase 4 — Merchant API + webhook contract

Implement:

```text
POST /v1/payments
GET  /v1/payments/{id}
POST /v1/payments/{id}/cancel
```

Create request:

```json
{
  "amount": 100,
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026",
  "metadata": {
    "registration_id": "reg_284"
  }
}
```

Do not accept `paymentAccount`.

Implement:

- request normalization;
- idempotency request hash;
- fixed-string money response;
- canonical UPI URI from payment snapshot;
- payer info after payment when available;
- signed durable webhook outbox.

### API edge cases

- repeated event `external_id` creates separate legitimate payments with different idempotency keys;
- duplicate person `name` allowed;
- parent/friend payer does not overwrite merchant-supplied `name`;
- same idempotency key + same request replays original payment;
- same idempotency key + changed field conflicts;
- webhook consumer can dedupe by webhook event ID;
- public/browser status cannot expose payer/metadata accidentally.

## Phase 5 — Relay protocol and notification identity

Preserve existing Android device signing while simplifying payload/ingestion.

Implement/verify:

- stable local notification event identity;
- generic decimal-money prefilter;
- original notification `postTime` persistence;
- bounded text payload;
- server event dedupe;
- timestamp confidence/source;
- one active relay phone policy;
- QR/App-Link pairing session.

### Relay edge cases

- listener rescan sees same notification -> one event;
- notification retry after network outage -> one event;
- notification content update -> deterministic versioning, no mutation of delivered event;
- group summary ignored;
- `.00` ignored by prefilter;
- decimal debit/balance passes cheap filter but server rejects as non-credit;
- old pre-enrollment notification cannot replay;
- phone clock skew produces explicit health failure rather than silent match;
- relay offline > lifecycle -> event uses original post time, not resend time.

## Phase 6 — Paytm parser

Build from sanitized real Paytm Business notification captures.

Test:

- amount formats including comma separators;
- payer-name presence/absence;
- displayed transaction time if present;
- Android notification post time fallback;
- non-payment Paytm notifications;
- debit/refund/system notices;
- notification wording variants;
- duplicate/update behavior.

Paytm parser normalizes only confirmed incoming-credit wording to profile `paytm`.

## Phase 7 — Kotak through Google Messages

Capture sanitized real Kotak credit SMS notifications exactly as the Android NotificationListener sees them.

Test:

- sender/bank marker recognition;
- incoming-credit semantics;
- exact decimal amount;
- payer/UPI data when present;
- transaction time when present;
- Google Messages notification grouping/updates;
- unrelated personal messages;
- other-bank decimal messages;
- lock/Doze delivery;
- phone reboot.

### Temporary libgm parity gate

Before deleting server libgm:

1. keep current libgm path;
2. Android Google Messages path runs observe-only;
3. compare every real Kotak credit for a defined observation window;
4. investigate all misses or materially truncated notification bodies;
5. promote Android Kotak matching only when parity is acceptable;
6. disable libgm;
7. monitor production;
8. delete libgm code/routes/session state only after stable operation.

Final v4 contains no server-side Google Messages connector.

## Phase 8 — Matching engine

Implement matching against historical reservations, not merely the current active profile.

Matching inputs:

```text
profile
exact amount
occurred_at
occurred_at_source/confidence
unique relay event
```

### Critical delayed/reuse tests

- active payment match;
- grace match;
- delayed relay during quarantine with original post time;
- event delivered after amount release but occurrence belongs to old reservation;
- same amount reused later + high-confidence old timestamp -> old payment only;
- same amount reused later + ambiguous/low-confidence timestamp -> no auto-match;
- unknown decimal payment -> unmatched Activity;
- duplicate relay event -> no second paid webhook;
- inactive historical profile still matchable;
- active profile switch never affects match decision.

## Phase 9 — Admin auth and web dashboard

Implement singleton password-only admin authentication and:

```text
Overview
Payments
Activity
Settings
```

Payments must distinguish:

```text
Name/person       merchant context
Event ID          external_id
Actual payer      notification-derived identity
```

Add search/filter/payment edit/profile switch/webhook/device/backup health.

Do not expose Reviews/Reconciliation/Evidence as primary products.

## Phase 10 — Android unified dark app

Keep existing package ID/signing lineage and background services while replacing visible UI.

Order:

1. Razorpay-inspired dark design tokens;
2. Overview;
3. Payments/search/detail;
4. Activity;
5. Settings;
6. password-only login;
7. QR/App-Link Connect/Replace phone UX;
8. Paytm + Google Messages package allowlist;
9. generic decimal prefilter;
10. queue/heartbeat/Doze regression tests.

Do not require uninstall/clear-data/re-pair for the normal production upgrade.

## Phase 11 — PayGate Frontend test application

Remove:

- profile selector;
- account list fetch;
- verification-method branches;
- Paytm/Kotak customer logic.

Keep:

- input amount;
- input person `name`;
- event `external_id`;
- create payment;
- render QR from `upi_uri`;
- show exact randomized payable amount + adjustment clearly;
- 5-minute customer timer;
- status polling;
- paid result with actual payer details when available.

## Phase 12 — Offline v3 -> v4 migrator

Build a deterministic tool that reads a **verified v3 backup copy** and writes a new direct-SQLite v4 DB.

Never mutate live v3/PocketBase DB in place.

Map:

- payments;
- payment merchant context;
- account/UPI settings -> collection profiles;
- existing relay device public key;
- useful payment history/audit;
- webhook configuration/delivery records as needed;
- admin credential/bootstrap state.

Do not preserve every legacy evidence/reconciliation row as a first-class v4 concept. Archive the original v3 DB for forensic reference.

### Migration report

```text
v3 payment count
v4 payment count
status counts
latest IDs/timestamps
profile configuration
relay device identity
webhook config presence
rows archived/skipped
schema version
PRAGMA integrity_check
PRAGMA foreign_key_check
```

No secrets/raw personal message bodies in the report.

## Phase 13 — Staging acceptance

### Identity

- two people with same name create separate payments;
- many payments share same event `external_id`;
- event filtering works;
- actual payer may differ from merchant `name`.

### Profile

- Paytm active -> new UPI URI snapshots Paytm;
- switch to Kotak -> new payment snapshots Kotak;
- old Paytm session remains valid/matchable;
- caller cannot override profile.

### Random allocator

- statistically non-sequential across repeated controlled creations;
- no duplicate active profile+amount;
- no `.00`;
- cap respected;
- adjacent requested amounts handled;
- recent-use soft avoidance demonstrated;
- hard reservation expires at configured boundary.

### Relay

- real-format Paytm notification -> correct observation/payment;
- real-format Kotak Google Messages notification -> correct observation/payment;
- unrelated decimal message -> no payment mutation;
- duplicate/rescan -> idempotent;
- delayed/reused-amount ambiguity -> fail closed.

### SQLite

- concurrent API load;
- `SQLITE_BUSY` handling;
- server crash/restart;
- WAL recovery;
- migration failure;
- online backup under writes;
- restore drill;
- low disk simulation where feasible;
- second-process ownership lock.

### Android

- in-place upgrade;
- pairing state/key preserved;
- locked + Doze;
- Battery Saver;
- network loss/recovery;
- reboot;
- notification listener rebind;
- QR pairing replacement flow.

## Phase 14 — Production cutover

Choose a quiet period.

### Before cutover

- v4 CI green;
- final v3 backup verified;
- v4 migration dry run deterministic;
- parser parity complete;
- direct SQLite backup/restore drill complete;
- Android signed upgrade tested in place;
- rollback image/config/data prepared.

### Drain v3 amount state

Temporarily stop new payment creation while allowing existing pending/grace/quarantine reservations to settle.

Preferred:

```text
active in-flight reservations = 0
```

### Cutover

1. final verified v3 backup;
2. stop v3;
3. run offline migrator -> new `paygate.db`;
4. verify integrity, foreign keys, schema/report;
5. start one v4 process;
6. verify PRAGMA configuration/profile/device/webhook settings;
7. heartbeat/relay smoke test;
8. controlled low-value end-to-end payment;
9. re-enable normal creation.

Keep final v3 DB/image untouched for rollback window.

## Phase 15 — Cleanup

Only after stable production:

- remove old `/api/payment-accounts` decision surface;
- remove v3 compatibility endpoints no longer needed;
- remove `internal/gmessages`/libgm dependencies/session files;
- remove PocketBase/dbx runtime dependencies;
- remove Review/Reconciliation legacy UI;
- archive old payment-frontend profile-selection code;
- clean stale worktrees only after release/backup evidence is retained.

## Rollback

If cutover fails before meaningful v4 writes, stop v4 and restore v3 image/config/final DB backup.

If meaningful v4 payments occurred, never blindly replace v4 DB with old v3 state. Export/reconcile new payments first and execute a deliberate reverse migration/recovery plan.

## Suggested implementation PRs

1. direct SQLite connection/schema/migrations/backups
2. collection profiles + amount reservations + randomized allocator
3. payment domain/idempotency/history
4. merchant API + UPI URI contract
5. webhook outbox
6. relay pairing/signature/event identity
7. Paytm parser
8. Kotak Google Messages parser + parity tooling
9. matching engine/time-confidence rules
10. admin auth/settings APIs
11. admin Overview/Payments/Activity UI
12. Android dark redesign
13. Android prefilter + QR pairing
14. offline v3->v4 migration tool/runbook
15. test frontend v4 integration
16. production cutover/legacy cleanup

## Definition of done

V4 is done only when:

- `name` and event `external_id` semantics are correct everywhere;
- event ID is not treated as unique/idempotent;
- caller cannot select Paytm/Kotak;
- frontend renders QR from server UPI URI;
- randomized reservation allocator is collision-safe under concurrency;
- delayed/reused-amount ambiguity fails closed;
- Paytm + Kotak both flow through one Android app;
- server libgm is gone;
- direct SQLite is the only production data runtime;
- PocketBase is gone;
- backups/restores are verified;
- web/Android share Overview/Payments/Activity/Settings;
- password-only operator login is live;
- webhooks remain durable/signed;
- rollback artifacts are retained for the agreed safety window.