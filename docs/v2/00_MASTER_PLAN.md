# PayGate v2 — Master Rewrite Plan

Status: implementation branch, not production
Canonical branches:
- API: `rewrite/paygate-v2-foundation`
- Checkout web: `rewrite/paygate-v2-ui`
- Android: `rewrite/paygate-v2-mobile`

## Goal

Rewrite PayGate into a smaller, typed, modular payment system without weakening any existing financial invariant. The rewrite must reduce framework coupling, duplicated evidence logic, duplicated Razorpay code, stringly-typed status handling, oversized transactions, UI complexity, and operator friction.

PayGate v2 remains a modular monolith. It does **not** introduce microservices, Kafka, Redis, distributed transactions, or a second primary database simply for architectural fashion.

## Non-negotiable payment invariants

1. Money is stored and compared as integer paise only.
2. Direct-UPI matching is exact amount + payment account + occurrence-time constrained.
3. Evidence identity is globally unique where the source provides a stable bank/reference identity.
4. Notification evidence must have a unique non-empty signed evidence reference.
5. Two or more plausible payment candidates is always an ambiguous/fail-closed result.
6. Evidence predating payment creation beyond the accepted tolerance can never pay that payment.
7. Amount fingerprints remain quarantined after expiry, cancellation, payment, or late payment.
8. Idempotency replay is resolved before new-payment readiness checks.
9. Payment mutation and durable outbound event creation are atomic.
10. External network calls never happen inside payment database transactions.
## Product surfaces

PayGate v2 is one product with three coordinated surfaces:

### 1. Checkout web
A public, mobile-first payment experience whose only job is to create and complete a payment safely. It should expose the minimum number of decisions, use progressive disclosure, and make the exact amount and payment state unmistakable.

### 2. Operator web
A typed administrative console for payments, reviews, refunds, reconciliation, alerts, relay health, capacity, backups, and settings. It must stop querying PocketBase collections directly.

### 3. PayGate Android app
The existing relay becomes the PayGate mobile application. The proven notification listener, durable local queue, WorkManager recovery, boot receiver, foreground runtime service, wake-lock-bounded delivery, and signed relay protocol remain. The UI becomes a modern operator app for health, payments, reviews and alerts.

## Architectural direction

The target dependency direction is:

`HTTP / source adapters -> application services -> typed domain -> repositories/UoW -> SQLite adapter`

Business code must not depend on PocketBase `core.Record`, record string keys, or `core.App` transaction objects. PocketBase may remain temporarily behind repository adapters during migration.

All payment evidence sources normalize into one domain `Evidence` model and one matching engine. SMS, authenticated email, Paytm notification, reconciliation and future sources are adapters, not parallel payment engines.
## Rewrite workstreams

### A. Domain and persistence
- Introduce typed `Money`, `Payment`, `Evidence`, `EvidenceRef`, `MatchOutcome`, `Refund`, `RelayHealth`, and typed status/ID values.
- Add repository interfaces and an explicit unit-of-work abstraction.
- Move PocketBase record mapping behind storage adapters.
- Remove application-layer `...InApp` variants as repository/UoW adoption reaches each module.
- Keep SQLite and WAL unless measured load demonstrates a real need to change.

### B. Evidence engine
- Normalize Kotak SMS, Slice email and Paytm notifications into one `Evidence` structure.
- Use one candidate selection algorithm for point-in-time and time-window evidence.
- Make manual review use the same matcher invariants with an operator-selected candidate.
- Preserve raw source events separately from normalized evidence and redact them by retention policy.
- Shadow-test Google Messages Android notification evidence against the server-side libgm connector before any connector removal. The Android path is observation-only and stores only non-raw parse metadata plus a hashed reference for correlation.
- QR pairing is retired from operator-facing HTTP/UI paths during the shadow period; Google-account pairing/reauth remains available until the libgm exit gate is satisfied.

### C. Durable delivery and alerts
- Keep the payment outbox concept.
- Consolidate webhook and operator notification retry/claim/lease mechanics into one durable delivery engine.
- Separate discrete alert events from persistent alert conditions.
- Replace periodic `Open()` calls for conditions with `EnsureCondition()` semantics that do not increment occurrence counts every scan.
- Persist health-condition timing instead of relying on process-local maturation maps.
### D. Payment lifecycle and jobs
- Stop running global expiry mutations as a side effect of ordinary reads and unrelated writes.
- Expire due payments in small bounded background batches.
- Let matching determine on-time versus late from persisted timestamps, not from whether an expiry cron happened first.
- Preserve `reuse_after` as the fingerprint allocation guard.
- Batch reconciliation persistence/classification instead of holding one write transaction across a large statement.
- Reject ambiguous statement dates unless a bank/import profile establishes the date convention.

### E. Razorpay
- Replace duplicated test/live implementations with one shared rail engine.
- Retain separate constructors, credentials, storage namespaces/tables and explicit mode checks to preserve blast-radius isolation.
- Express temporary pilot restrictions, such as fixed ₹1 live checkout, as policy/configuration rather than forked code.

### F. API and operator console
- Split API registration into public checkout, trusted integration, evidence ingest, relay, operator and provider modules.
- Introduce typed operator query endpoints and stop exposing database collection schemas to the browser.
- Use polling or a small SSE invalidation stream for live operator updates.
- Build stable response DTOs that deliberately exclude raw evidence unless a privileged detail endpoint requires it.

### G. Customer checkout
- Reduce the checkout to one primary action per state.
- Remove duplicated explanations, secondary controls and technical terminology from the main path.
- Keep payment account selection automatic by default; only show a chooser when multiple rails are ready and there is a user-relevant reason to choose.
- Design the pending screen around amount, QR/action, countdown and authoritative verification state.
### H. Android / PayGate mobile
- Keep the relay runtime engine but replace the raw Java settings/debug screen.
- Move the user-facing application to Kotlin + Jetpack Compose while allowing the Java relay engine to coexist during migration.
- Add authenticated operator sessions with device-bound secure token storage.
- Add home dashboard, payment search/list/detail, review queue, alert inbox, relay health and settings.
- Make dangerous actions require explicit confirmation and server-side authorization/audit.
- Keep relay pairing credentials distinct from operator login credentials.

## Explicit non-goals

- No direct bank-account automation beyond the trusted evidence sources already supported.
- No automatic ambiguous-payment resolution.
- No automatic refund execution unless a provider API and explicit policy are later approved.
- No replay of old exhausted webhooks merely because transport has recovered.
- No secret material in mobile diagnostics, operator DTOs, logs or public API responses.
- No breaking production migration without dual-read/dual-write or a tested backfill path.

## Success criteria

PayGate v2 is considered ready to replace v1 only when:
- all current payment invariant and edge-case tests pass against the v2 engine;
- shadow comparison shows equivalent outcomes for existing evidence sources;
- production can roll back to the v1 path without database restore;
- customer checkout has fewer primary decisions and fewer controls per state;
- operator web and Android both use the same typed operator API;
- no UI depends on PocketBase collection names or realtime record payloads;
- v2 operational alerts do not re-open/increment continuously for unchanged conditions;
- relay health, backups, outbox and reconciliation have deterministic recovery tests.