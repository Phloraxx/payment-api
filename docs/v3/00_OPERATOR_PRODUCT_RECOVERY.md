# PayGate v3 Operator Product Recovery

## Purpose

This document defines the recovered v3 operator-console scope built from the exact v2 API base `a89845b7`. It is intentionally narrower than a generic finance back office: common payment work is simple, while evidence, reconciliation, refunds and infrastructure remain available without dominating the primary UI.

The recovery branch is not a production cutover branch. Before any merge or deployment, compare it with the stranded/original v3 worktree when that environment is available and resolve differences explicitly.

## Product hierarchy

The primary operator navigation is:

1. **Overview** — current payment state and only the attention that matters now.
2. **Payments** — search, create, inspect and manage every payment.
3. **Action** — fail-closed cases that require a human decision.
4. **Health** — verification rails, recovery readiness and open operational alerts.

Advanced tools remain available for reconciliation, refunds, raw SMS/email evidence, Razorpay test mode, alert history, webhook delivery diagnostics, audit history and low-frequency settings.

## Payment management contract

Operators can search across payment ID, external/order ID, display name, customer details, RRN/UTR, UPI ID and evidence reference. Search values are bound parameters in a fixed PocketBase filter expression; user input is never concatenated into the filter grammar.
The payment list supports validated status/account filters, a sort whitelist, total counts and bounded pagination. Summary rows expose business context but not sensitive matching evidence; evidence is shown only in the authenticated payment detail surface.

Editable operator-owned fields are limited to:

- display name
- customer name, email and phone
- description
- private admin note
- tags
- custom fields

Every effective profile change is saved and audited in the same database transaction. The audit record stores the names of changed fields, not customer values. Saving an unchanged form is a no-op and does not create audit noise.

## Immutable payment truth

The operator profile endpoint must never mutate fields that participate in payment matching, uniqueness, lifecycle truth or create-request identity. The following remain read-only:

- payment account
- requested and payable amount
- payment status and lifecycle timestamps
- reuse/quarantine window
- RRN/UTR, payer identity and evidence source/reference
- idempotency key
- external/order ID
- original create metadata

`externalId` and original `metadata` are specifically protected because create idempotency replay compares them with the original request. Editing either after creation could turn a legitimate retry into an idempotency conflict.
The HTTP update boundary uses a strict JSON decoder. Unknown keys — including attempts to submit protected fields — are rejected with HTTP 400 instead of being silently discarded.

## Storage and migration

The v3 migration is additive. It adds only operator-owned profile fields and search indexes to `payments`; existing payment/evidence fields and indexes are not rewritten. Rollback removes only those v3 additions.

The SQLite single-writer invariant remains unchanged. Do not run a second API instance against the same persistent data directory during testing, rollout or restore operations.

## Acceptance gates

A recovery commit is acceptable only when all of the following pass:

- `npm test`
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- adversarial API tests for authentication and protected-field rejection
- desktop and 390 px mobile browser QA with no horizontal overflow or runtime errors
- secret/temp-artifact review of the source diff

Production rollout remains separate from source acceptance. Use a pinned immutable image, preserve one writer, keep automatic deployment disabled during manual cutover, and retain all v2 matching/evidence fail-closed invariants.

## Recovery reconciliation

When the original Oracle v3 worktree becomes available, compare it against this branch file-by-file. Prefer tested invariant-preserving behavior over unverified recovered code. Any additional original changes must pass the same acceptance gates before being carried forward.
