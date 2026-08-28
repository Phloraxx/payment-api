# PayGate v2 — Target Architecture

## Architecture style

PayGate remains a single deployable Go application backed by SQLite, with separate static/customer and operator/mobile clients. Internally it is a modular monolith with strict dependency direction and explicit transaction boundaries.

```text
HTTP / relay / provider adapters
            |
            v
     Application services
            |
            v
        Typed domain
            |
            v
    Unit of Work / repos
            |
            v
   SQLite/PocketBase adapter
```

Only the bottom adapter layer may know about PocketBase `core.App`, `core.Record`, collection names, or database field strings.

## Domain modules

### Payments
Owns allocation, idempotency, state transitions, fingerprint quarantine and payment queries. It does not parse SMS/email/notifications and does not make network calls.

### Evidence
Owns normalized evidence identity, validation and matching. Source adapters turn source-specific messages into `Evidence`; the matcher never receives raw SMS/email/notification payloads.

### Reviews
Owns ambiguous/unmatched/manual-resolution workflow. Manual linking cannot bypass amount, account, evidence uniqueness, creation-time or quarantine invariants.
### Refunds
Owns refund reservation, idempotency, provider/reference uniqueness and explicit state transitions. Execution remains manual/provider-specific until a provider adapter is explicitly approved.

### Reconciliation
Owns statement import, normalized statement entries, classification and discrepancy review. Parsing happens outside write transactions; persistence and classification happen in bounded batches.

### Relay health
Stores observed device facts. A pure policy evaluates those facts into `healthy`, `degraded`, `blocked` and reason codes. Readiness code consumes policy output rather than reproducing boolean logic.

### Operations
Owns alerts, durable delivery, backup status, retention and operational health. Alert events and alert conditions are separate concepts.

## Core value types

```go
type Money struct { Paise int64 }
type PaymentID string
type EvidenceID string
type AccountID string
type PaymentStatus uint8
type MatchOutcome uint8
```

String serialization belongs at HTTP/storage boundaries. The domain should prefer typed enums/value objects so invalid statuses and field-name typos cannot silently propagate.

## Payment aggregate

A payment contains requested/payable money, account, lifecycle timestamps, idempotency identity, external identity, and an optional applied evidence reference. Payer/evidence details may be materialized for operational convenience, but evidence remains a first-class record rather than an ad-hoc set of payment columns.
## Normalized evidence

```go
type Evidence struct {
    ID            EvidenceID
    Source        EvidenceSource
    Account       AccountID
    Amount        Money
    OccurredFrom  time.Time
    OccurredTo    time.Time
    Reference     string
    ReferenceKind ReferenceKind
    PayerName     string
    UPIID         string
}
```

Point-in-time evidence uses the same value for `OccurredFrom` and `OccurredTo`. Paytm minute-precision/notification evidence can use an interval. One matcher therefore handles all sources without a second notification-specific payment engine.

## Match algorithm

1. Validate source/account/amount/reference requirements.
2. Resolve duplicate evidence/reference assignment before candidate selection.
3. Clamp/validate occurrence interval against ingestion time according to source policy.
4. Query at most two candidate payments by account + exact payable amount + creation/time/quarantine constraints.
5. `0 candidates` -> unmatched/review depending on source policy.
6. `1 candidate` -> derive paid vs late from occurrence time and lifecycle timestamps.
7. `2 candidates` -> ambiguous; never choose by recency or heuristic.
8. Persist normalized evidence, payment mutation, audit entry and outbox event in one transaction.

Manual review supplies a selected payment ID to the same invariant evaluator. It does not use a separate permissive matching implementation.
## Unit of Work

Application services receive a `UnitOfWork` abstraction rather than `core.App`:

```go
type UnitOfWork interface {
    Payments() PaymentRepository
    Evidence() EvidenceRepository
    Reviews() ReviewRepository
    Outbox() OutboxRepository
    Audit() AuditRepository
}

type Transactor interface {
    Within(ctx context.Context, fn func(UnitOfWork) error) error
}
```

The initial implementation may wrap PocketBase transactions. A later plain-SQLite implementation can replace it without changing application/domain code.

## Durable outbox/delivery

Payment/refund events and operator notifications share claim, lease, retry and exhaustion mechanics. Destination-specific senders remain separate.

```text
outbox item -> available -> claimed -> sending
                         -> delivered
                         -> retry scheduled
                         -> exhausted
```

The payment transaction inserts an outbox item only. Network delivery is always asynchronous and outside that transaction.
## Alert semantics

Two APIs are required:

```go
RecordEvent(kind, severity, dedupe, details)
EnsureCondition(kind, severity, dedupe, active, details)
```

`RecordEvent` increments occurrence count because a new event occurred. `EnsureCondition` updates last-seen/details while a condition remains active but does not increment each scan. A resolved condition may increment/reopen once when it becomes active again.

## API surfaces

### Public checkout
Minimal endpoints for account availability, creating a checkout payment and reading public payment status. No raw evidence or operator data. Strong body/rate limits.

### Trusted integration
Server-to-server payment/refund endpoints authenticated independently from the public checkout surface.

### Evidence ingest
Signed/secret source-specific routes that normalize and pass evidence to the evidence application service.

### Relay
Device enrollment, signed events and heartbeat. Relay device credentials authorize relay operations only.

### Operator
Authenticated typed queries/actions for dashboard, payments, reviews, refunds, reconciliation, alerts, relay, delivery, backups and settings. This is the only administrative API used by web/mobile clients.

### Provider
Razorpay test/live webhooks and order operations implemented by a shared provider engine with mode-specific policy/credentials.
## Storage target

Preferred long-term tables/entities:
- `payments`
- `evidence` and source/raw evidence records
- `review_cases`
- `refunds`
- `reconciliation_runs` / `reconciliation_entries`
- `relay_devices` / `relay_events`
- `outbox`
- `alerts` / optional alert history
- `audit_events`
- Razorpay orders/events with test/live isolation
- operator identities/sessions

Do not collapse tables merely to reduce table count; consolidate duplicated behavior, not useful isolation.

## Background work

Jobs are bounded and idempotent:
- expire due payments in batches;
- claim/deliver outbox items;
- evaluate operational conditions;
- retention/redaction;
- backup verification and restore drill;
- reconciliation batches if an import is still processing.

No ordinary `GET` should expire unrelated payments or create unrelated webhook rows as a side effect.

## Google Messages exit strategy

The libgm connector remains available during migration, but Android Google Messages notification observation is shadow-compared against it. If measured miss rate, latency and parsed evidence quality satisfy the acceptance threshold, remove server-side pairing/reauth/QR/session machinery and delete unsupported QR fallback paths. The removal requires measured parity, not assumption.