# PayGate v2 — Migration and Rollout Plan

## Principle

This is a strangler rewrite, not a big-bang replacement. Production remains on the current v1 path until each v2 slice has characterization tests, shadow comparison where possible, reversible schema changes, and a rollback path that does not require restoring the database.

## Phase 0 — Stabilize v1 before extraction

- Merge/fix relay power-health compatibility.
- Replace webhook exhaustion alert scanning with aggregate condition semantics.
- [done in rewrite branch] Remove unsupported Google Messages QR fallback from operator-facing HTTP/UI paths; retain Google-account pairing/reauth during parity measurement.
- Add characterization tests for payment allocation, cancellation, late evidence, stale evidence, duplicate RRN/reference, idempotency, refunds and reconciliation.
- Capture production-like anonymized fixture cases for every evidence source.

Exit: v1 behavior is frozen by tests and known operational bugs are not being copied into v2.

## Phase 1 — Typed domain foundation

- Add typed money/payment/evidence/outcome types without changing persistence.
- Add repository/UoW interfaces and PocketBase-backed adapters.
- Move one read-only operator query through the repository layer first.
- Move payment record serialization/mapping into one adapter location.
- Prevent new direct `core.Record` access outside adapters with package/lint conventions.

Exit: application services can execute through repositories while the live schema remains unchanged.
## Phase 2 — Evidence normalization in shadow mode

- Add normalized `Evidence` and source adapter interfaces.
- Run Kotak SMS, Slice email and Paytm notification fixtures through both v1 and v2 match decision logic.
- Persist v2 normalized evidence in shadow-only storage or test fixtures first; do not mutate production payment state from v2 yet.
- For Google Messages, reuse relay/SMS event history for shadow comparison and store only non-raw Android parse metadata plus a hashed bank reference; require the documented strict parity gate before manual libgm removal review.
- Compare candidate IDs, outcome, paid/late classification and rejection reason.
- Require exact parity on all invariant cases before enabling v2 writes.

Exit: v2 matcher produces the same safe decision as v1 for all known/fixture cases and property tests.

## Phase 3 — Transactional v2 write path

- Enable v2 payment/evidence mutation behind an environment/feature flag.
- Keep current schema initially and write through the repository adapter.
- Preserve existing outbox/audit rows so downstream integrations remain unchanged.
- Start with one evidence source, then expand source by source.
- Record comparison telemetry without raw sensitive evidence.

Rollback: disable v2 matcher flag; v1 reads the same persisted payments/evidence columns.

## Phase 4 — Operator API decoupling

- Add typed `/api/operator/v2/*` query/action endpoints.
- Port dashboard, payments, reviews, refunds, reconciliation, alerts, relay, delivery, backups and settings.
- Web operator console and Android app consume these endpoints only.
- Replace PocketBase realtime record subscriptions with polling/SSE invalidations.
- Once no client reads PocketBase collections directly, collection schema can evolve independently.

Exit: browser/mobile clients have no collection-name or PocketBase realtime dependency.
## Phase 5 — Customer checkout v2

- Ship the redesigned checkout against the existing compatible payment contract first.
- Measure create errors, abandonment between create/payment, completion latency and status-refresh errors.
- Add direct public checkout API only after equivalent limits/abuse controls are implemented in Go.
- If the Node/Hono BFF is removed, keep the old frontend deployment available for immediate DNS/route rollback.

## Phase 6 — PayGate mobile

- Release the modern app UI while preserving the existing relay package/application ID and signing certificate.
- Maintain existing relay pairing/device identity across upgrade.
- Introduce operator authentication independently from relay identity.
- Start mobile management read-only; add reviewed actions one by one with audit and confirmation.
- Do not allow mobile UI/background service lifecycle changes to stop the relay when the operator logs out.

## Phase 7 — Consolidation

- Shared Razorpay engine replaces duplicated test/live business code while retaining credential/storage isolation.
- Shared durable delivery engine replaces duplicated webhook/operator notification retry mechanics.
- Batched expiry/reconciliation replace oversized/global transaction behavior.
- Configuration is decomposed into validated sub-configs.
- Legacy source aliases/routes are removed after usage counters reach zero.

## Phase 8 — Optional PocketBase removal

Only consider this after operator clients and application services are fully decoupled. Implement typed SQLite repositories (prefer explicit SQL/sqlc-style generated access) and migration tooling. Keep schema compatibility or provide a proven one-way migration plus backup/restore rehearsal.
## Required release gates for every phase

1. `gofmt`, unit/integration tests, `go vet`, staticcheck, govulncheck and relevant race tests pass.
2. Frontend/mobile contract tests cover old and new server compatibility where a rolling upgrade is possible.
3. Database migrations are additive until all old readers are removed.
4. A fresh verified backup exists before any production schema/data remediation.
5. Migration scripts have a test that runs from a realistic pre-migration database.
6. Payment/evidence duplicate constraints are checked before and after rollout.
7. No production data reset, automatic historical replay, or destructive backfill.
8. Health/readiness must degrade safely if a required verification source is unhealthy.
9. Metrics/logs used for shadow comparison must not contain raw payer evidence or secrets.
10. Every operator mutation creates an audit entry containing actor, action and target identity.

## Cutover strategy

Prefer percentage/source-scoped feature flags over host-level blue/green for the payment matcher because both implementations need to observe the same SQLite state. Public web UI can be blue/green independently.

Suggested matcher progression: fixtures -> test DB -> shadow production decisions -> one source/write path -> all direct UPI sources -> operator/manual flows -> old path disabled -> old path deleted after soak.

## Rollback triggers

Immediate rollback for any of:
- candidate/outcome mismatch against established invariant fixtures;
- duplicate evidence/reference or duplicate active fingerprint allocation;
- unexpected increase in ambiguous/unmatched evidence;
- payment creation success but missing durable outbox/audit mutation;
- elevated SQLite busy/write-lock time attributable to v2;
- mobile/operator action bypassing authorization or audit;
- checkout regression that obscures exact amount, expiry or authoritative verification status.

Deletion of old code occurs only after a separate stable soak period; rollback must remain possible before deletion.