# 08 — Migration and Implementation Plan

## Strategy

Do not rewrite production in place.

Build v4 beside the proven v3 system, validate against production-shaped data, then perform a controlled cutover with a complete rollback path.

The rewrite should reduce complexity in the final state without taking unnecessary migration risks.

## Phase 0 — Freeze the target design

Before implementation:

- approve the product boundaries in `docs/v4`;
- keep `payment-frontend` explicitly outside the PayGate product boundary;
- confirm v4.0 sources are Paytm + Kotak only;
- confirm GPay and Slice remain deferred;
- confirm amount adjustment default maximum is ₹1.99;
- confirm 5m active + 5m grace + 5m quarantine lifecycle;
- confirm direct SQLite target and PocketBase retirement;
- capture sanitized real Paytm and Kotak/Google Messages notification fixtures.

No production mutation is required for this phase.

## Phase 1 — Build the new storage/domain core offline

Create the v4 domain model without serving production traffic.

Implement:

- direct SQLite repository layer;
- schema/migrations;
- collection profiles;
- smallest-free amount allocator;
- payment lifecycle;
- payment history;
- relay events and normalized observations;
- webhook outbox;
- admin credentials/sessions;
- merchant API keys;
- pairing sessions and relay devices.

Use a production backup copy for read-only migration testing, never the live database.

### Required storage tests

- schema migration from empty database;
- transaction rollback on failures;
- unique profile+payable reservation behavior;
- profile switch transaction;
- event deduplication;
- webhook-outbox atomicity;
- SQLite WAL/busy-timeout startup checks;
- backup/restore against an isolated copy.

## Phase 2 — One-time v3 -> v4 data migrator

Build a dedicated offline migration command/tool that reads a verified v3/PocketBase backup and writes a **new** v4 SQLite file.

Do not mutate the v3 database in place.

Map at minimum:

- payments -> v4 payments;
- current account/UPI configuration -> collection profiles;
- relevant operator-visible history if available;
- webhook configuration/delivery history where useful;
- current relay device public key/enrollment -> v4 relay device;
- admin credential -> v4 singleton admin password hash or explicit reset/bootstrap path.

Legacy evidence/reconciliation records do not all need to become first-class v4 objects. Preserve only what is useful for payment history/audit and archive the old database for forensic reference.

### Migration validation report

The migrator must print/write:

```text
v3 payment count
v4 payment count
status counts before/after
latest payment IDs/timestamps
active profile
relay device identity
webhook destination presence
rows skipped/archive-only
integrity_check result
```

The report must contain no secrets or raw payer message bodies.

Run this migrator repeatedly against backups until deterministic.

## Phase 3 — Build v4 API beside v3

Implement new merchant contract:

```text
POST /v1/payments
GET  /v1/payments/{id}
POST /v1/payments/{id}/cancel
```

Implement idempotency and webhooks.

For migration compatibility, the v4 server may temporarily accept the existing Android relay protocol so the production phone does not have to be upgraded at exactly the same second as the backend cutover.

Do not expose `paymentAccount` in v4 payment creation.

## Phase 4 — Server parsers and Android-notification parity

### Paytm

Use sanitized real Paytm for Business notification captures to build parser fixtures.

Validate:

- incoming payment wording;
- amount;
- payer name if present;
- notification timestamp behavior;
- ignored non-payment Paytm notifications.

### Kotak via Google Messages

Capture sanitized Kotak credit SMS notifications as rendered by Google Messages.

Validate:

- generic Android prefilter sees them;
- server recognizes Kotak wording/sender;
- exact amount parsed;
- payer/UPI details parsed when present;
- unrelated bank/personal Google Messages notifications ignored.

### Temporary libgm parity period

Before deleting server libgm:

1. keep current libgm connector running read-only/normal during a short validation period;
2. enable Android Google Messages notification relay in observe mode;
3. compare Kotak credits seen by libgm against notification-relay observations;
4. verify no real Kotak credit is systematically missing;
5. promote Android Kotak observation to matching;
6. disable libgm;
7. observe production stability;
8. only then remove libgm code/routes/session state.

The parity period is temporary migration scaffolding, not part of final architecture.

## Phase 5 — Unified Android app redesign

Keep the existing application ID/signing certificate.

Build the new dark PayGate UI while preserving background runtime code first.

Recommended order:

1. theme/tokens;
2. Overview;
3. Payments/search/detail;
4. Activity;
5. Settings;
6. password-only login;
7. QR/App-Link pairing UX;
8. narrow relay allowlist to Paytm + Google Messages;
9. generic decimal-money prefilter;
10. preserve/retest queue + Doze watchdog.

Do not force the production phone to re-pair if its existing ECDSA key can be migrated/reused.

## Phase 6 — Rework admin web dashboard

Build Overview / Payments / Activity / Settings against v4 admin APIs.

Remove product surfaces for:

- Reviews;
- Reconciliation;
- Google Messages connector pairing;
- source-specific evidence tables;
- Razorpay test/live pages from the core v4 navigation unless still explicitly needed as a separate test tool.

Direct payment edit replaces Manual Review.

## Phase 7 — Update PayGate Frontend test application

The test frontend changes to prove the new integration contract.

Remove:

- payment-account selector;
- payment-account fetch;
- Paytm/Kotak/Slice-specific customer text;
- verification-method branches.

Keep:

- create test payment with amount + name + external ID;
- display requested/payable amount + adjustment;
- render QR from returned `upi_uri`;
- timer using `expires_at`;
- status polling;
- paid result displaying payer fields when available.

The frontend remains separately deployable and is not folded into PayGate server.

## Phase 8 — Staging/canary acceptance

Create a v4 staging environment from migrated backup data.

Acceptance scenarios:

### Payment creation

- Paytm active -> returned UPI URI points to Paytm snapshot;
- switch to Kotak -> new payment points to Kotak;
- existing Paytm payment remains Paytm after switch;
- frontend cannot specify/override profile.

### Amount allocator

- ₹100 creates lowest available `.xx` candidate;
- exhaustion of `100.01..100.99` rolls into `101.01`;
- `.00` never allocated;
- adjacent requested amounts do not collide;
- cap returns clean capacity error.

### Relay/matching

- real-format Paytm notification -> paid;
- real-format Kotak Messages notification -> paid;
- unrelated message -> ignored;
- `.00` notification -> filtered/not auto-matched;
- duplicate event -> idempotent;
- delayed delivery -> correct grace/quarantine behavior;
- unsupported GPay -> ignored/observed only until future release.

### Admin

- password-only login;
- search/filter payments;
- edit permitted fields;
- immutable fields cannot be edited;
- profile switch;
- QR phone pairing;
- revoke device;
- webhook configuration.

### Resilience

- phone locked + Doze;
- Battery Saver;
- network loss/recovery;
- server restart;
- Docker update stop-first;
- backup and restore drill.

## Phase 9 — Production cutover

Choose a quiet period.

### Pre-cutover

- all v4 CI green;
- production backup verified;
- rollback image/config prepared;
- v3/v4 migration dry run successful against latest backup;
- no unreviewed schema migration discrepancy;
- Android v4 APK signed with existing lineage and validated in-place on test device/build.

### Drain current payment reservations

Temporarily stop **new payment creation** while keeping v3 running long enough for current pending/grace/quarantine sessions to settle.

Preferred condition before database cutover:

```text
pending/grace reservations = 0
```

This avoids migrating an amount pool in motion.

### Cutover

1. take final verified v3 backup;
2. stop v3 PayGate API;
3. run deterministic offline migrator -> new v4 DB;
4. run SQLite integrity and migration validation;
5. start v4 one-replica stop-first service;
6. verify health/profile/device/webhook settings;
7. send synthetic/non-monetary relay heartbeat;
8. create controlled small payment and validate end to end;
9. re-enable normal payment creation.

Keep v3 image + final v3 DB backup untouched for rollback.

## Phase 10 — Android production upgrade

Install v4 in place.

Verify after upgrade:

- same package/signing lineage;
- existing device ECDSA key and device ID preserved;
- notification-listener grant preserved;
- battery exemption preserved;
- local queue preserved;
- foreground service/watchdog running;
- operator login works;
- Paytm and Kotak observations deliver;
- no GPay matching enabled.

## Phase 11 — Remove temporary compatibility

Only after stable v4 production:

- remove old `/api/payment-accounts` and v3 creation compatibility;
- remove old Android relay compatibility endpoint if replaced;
- remove libgm and Google Messages server sessions/routes;
- remove PocketBase dependency/runtime;
- remove old Review/Reconciliation UI code;
- archive the old payment-frontend account-selection code;
- delete stale migration worktrees only after final tagged release/backup.

## Rollback

If v4 cutover fails before accepting meaningful new v4 payments:

1. stop v4;
2. restore v3 service/image/config;
3. restore final v3 database backup if v4 used a separate DB file (preferred);
4. verify health and relay heartbeat;
5. reopen payment creation.

If meaningful v4 payments already occurred, do not blindly roll back the database. Export those payments/observations first and execute a deliberate reconciliation/migration back plan.

## Pull-request decomposition

Avoid one enormous implementation PR. Suggested sequence:

1. v4 SQLite schema/repositories + tests
2. profiles + amount allocator/lifecycle
3. relay protocol/observation parsers
4. payment API + idempotency
5. webhooks
6. admin auth/settings APIs
7. admin Payments/Overview UI
8. Activity/settings/pairing UI
9. Android dark redesign/navigation
10. Android prefilter + App-Link pairing
11. Kotak parity and libgm retirement
12. offline migration tool/cutover runbook
13. PayGate Frontend v4 integration update

Each PR should keep a focused rollback/review surface.

## Definition of done

v4 is complete only when:

- caller cannot choose Paytm/Kotak;
- PayGate returns canonical UPI URI and owns profile/amount selection;
- test frontend renders QR from that URI;
- Paytm + Kotak notification flows both work through the single Android app;
- server libgm is removed;
- GPay/Slice remain deliberately off unless separately accepted;
- 5/5/5 amount lifecycle is production-tested;
- smallest-free cross-rupee allocator is proven collision-safe;
- direct SQLite v4 DB is live with verified backups;
- PocketBase is no longer required by the production server;
- web/Android share Overview/Payments/Activity/Settings information architecture;
- password-only admin login is live;
- webhooks remain durable and signed;
- old production rollback artifacts/backups are retained for the agreed safety window.