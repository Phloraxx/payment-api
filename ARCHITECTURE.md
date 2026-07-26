# PayGate — Implemented Architecture

## 1. Scope

PayGate is a single-operator, self-hosted UPI payment-verification service. A payer sends money directly to the configured UPI destination. PayGate observes bank-credit evidence and correlates it with a previously created payment.

It is not a fund-custody, settlement or acquiring system. The current product boundary intentionally avoids multi-merchant tenancy and provider abstractions.

## 2. Runtime

```text
                       ┌────────────────────────────┐
                       │       Android phone        │
                       │ receives bank-credit SMS   │
                       └──────────────┬─────────────┘
                                      │
                  ┌───────────────────┴──────────────────┐
                  │                                      │
       Google Messages/libgm                    legacy Android relay
                  │                                      │
                  └───────────────────┬──────────────────┘
                                      ▼
                        ┌─────────────────────────┐
                        │      PayGate binary     │
                        │                         │
                        │ custom Go HTTP routes   │
                        │ payment service         │
                        │ SMS parser/ingestion    │
                        │ webhook outbox worker   │
                        │ optional libgm manager  │
                        │ PocketBase framework    │
                        │ React static assets     │
                        └────────────┬────────────┘
                                     │
                                     ▼
                              PocketBase SQLite
                               /app/pb_data
```

One process and one SQLite database are deliberate. This workload does not need Redis, Kafka, a second service or a separate logging database.

## 3. PocketBase boundary

PocketBase 0.39.9 supplies:

- SQLite connection and transactions;
- migrations/collection definitions;
- authentication;
- SSE realtime for the operator UI;
- cron scheduling;
- request logging;
- backups;
- the low-level `/_/` administration UI.

Custom Go code owns payment state transitions. PocketBase record APIs expose domain collections read-only to authenticated records from the `users` auth collection. Collection create/update/delete rules are locked.

The application UI is a separate embedded React/Vite build at `/`; PocketBase `/_/` is not modified.

## 4. Collections

### `users`

PocketBase auth collection for normal PayGate operator accounts. Public self-registration is disabled; accounts are created administratively.

### `payments`

Important fields:

| Field | Purpose |
|---|---|
| `created_at` | Business creation timestamp used for evidence eligibility |
| `requested_amount` | Whole requested amount in paise |
| `payable_amount` | Exact DDM amount in paise |
| `status` | pending/paid/expired/cancelled/late |
| `expires_at` | Pending deadline |
| `reuse_after` | Amount quarantine deadline |
| `rrn` | UPI reference, unique when non-empty |
| `upi_id` | Parsed payer UPI ID when available |
| `payer_name` | Parsed payer identity when available |
| `paid_at` | Best-known evidence occurrence time |
| `external_id` | Caller/order correlation |
| `idempotency_key` | Unique non-empty create key |
| `metadata` | Caller JSON metadata |

PocketBase's own `created` and `updated` autodate fields are also present. `created_at` is separate because business correlation must use an explicit timestamp written by the payment transaction rather than relying on framework metadata semantics.

### `sms_events`

Durable evidence/audit records:

- source: `android_webhook`, `gmessages`, or `manual`;
- source event ID;
- sender/body;
- original/effective message timestamp;
- parsed amount/RRN/UPI ID/payer name;
- processing status;
- matched payment relation;
- parsing/matching error;
- small raw connector metadata.

`(source, source_event_id)` is unique when the provider ID is non-empty.

### `webhook_deliveries`

Durable outbox/delivery state:

- stable event ID and event name;
- payment relation;
- destination URL;
- immutable request body;
- attempts/status;
- next attempt timestamp;
- sending lease timestamp;
- response code/error/delivery timestamps.

## 5. Money and DDM allocation

Money is never represented by floating point in the domain layer.

For requested whole rupees `R`:

```text
requestedPaise = R * 100
candidate = requestedPaise + suffix
suffix ∈ [1, 99]
```

`.00` is intentionally excluded. The maximum accepted requested amount reserves 99 paise of `int64` headroom so `requestedPaise + 99` cannot overflow.

Creation runs inside a SQLite transaction:

1. expire due pending payments;
2. resolve an existing idempotency key if supplied;
3. choose a randomized starting suffix;
4. probe all 99 suffixes cyclically;
5. a candidate is unavailable while any existing payment with that payable amount has `reuse_after > now`;
6. persist the first available candidate with `expires_at` and `reuse_after`;
7. fail with `AMOUNT_CAPACITY_EXHAUSTED` if all 99 are blocked.

The suffix start is injectable for deterministic tests but uses cryptographic randomness in production.

## 6. Lifecycle and quarantine

```text
            create
              │
              ▼
           pending
          /   │    \
       pay  expire cancel
        │      │      │
        ▼      ▼      ▼
      paid   expired cancelled
                \      /
                 \    /
             exact late credit
                    │
                    ▼
                   late
```

Expiry is persisted and evaluated both by scheduled work and before operations that depend on current state. No business truth depends on an in-memory timer.

`reuse_after` protects a fingerprint from immediate reassignment. Paid/cancelled/late transitions start a fresh quarantine window from the processing time. An expired payment retains the creation-time `expires_at + quarantine` reservation.

## 7. Evidence-time guard

Amount quarantine alone is insufficient once a suffix is eventually reused: a provider may reconnect and deliver an old historical message.

Every normalized SMS has `OccurredAt`. For Google Messages this comes from the provider message timestamp. The SMS service clamps missing/future timestamps to ingestion time.

Automatic matching requires:

```text
payment.created_at <= evidence.OccurredAt
```

Therefore an old catch-up message cannot confirm a payment that did not exist when the SMS occurred, even if the same amount has since been reused.

Legacy webhook clients that omit a timestamp cannot provide this protection and are treated as occurring at ingestion time. This is one reason the compatibility route is temporary.

## 8. SMS transaction

SMS ingestion is one database transaction for the evidence record and payment state transition:

1. validate source and storage-size bounds;
2. deduplicate `(source, source_event_id)`;
3. persist the raw SMS event as `received`;
4. ignore unrelated messages after keeping the audit record;
5. parse the bank-credit message;
6. require exact amount and RRN for automatic confirmation;
7. find an existing payment by RRN:
   - same amount → idempotent duplicate;
   - different amount → fail `RRN_AMOUNT_MISMATCH`;
8. exact-match an eligible pending payment;
9. otherwise exact-match an eligible expired/cancelled payment still quarantined and mark `late`;
10. update the SMS record with result/relation;
11. enqueue webhook work in the same transaction if required;
12. wake the delivery worker only after commit.

Network calls never occur inside the SQLite transaction.

## 9. Bank parser

The current parser is deliberately narrow and fail-closed around tested Kotak credit-message forms. It extracts:

- received INR amount;
- UPI RRN/reference;
- UPI ID where present;
- payer/from text where present.

OTP/debit/unrelated messages are ignored rather than interpreted as payment evidence. Derived text fields are bounded before persistence so an unusual SMS cannot cause a collection-validation 500.

Expanding to another bank should add concrete parser fixtures first rather than loosening the existing regex indiscriminately.

## 10. Google Messages connector

`internal/gmessages` wraps libgm behind an ingestion callback. The payment service itself has no dependency on libgm types.

Responsibilities:

- load/save pairing state under the persistent data directory;
- filesystem permissions: session file `0600`, private parent directory;
- connect/reconnect/backoff;
- monitor paired/connected/phone-responsive state;
- ignore outgoing messages;
- prefilter for bank-credit-like text before copying it into PayGate;
- preserve provider message ID and timestamp;
- process old/catch-up events through the same idempotent ingestion path;
- expose operator pairing/reconnect/unpair controls.

Pairing refuses to replace an already-valid session. The operator must unpair first.

The connector is optional. Payment/SMS records remain valid if Google's private protocol changes.

## 11. HTTP boundary

Custom routes:

```text
POST   /api/payments
GET    /api/payments/{id}
POST   /api/payments/{id}/cancel
POST   /api/events/sms
POST   /api/webhook                       # opt-in legacy compatibility
GET    /api/paygate/health
GET    /api/config                        # authenticated operator
GET    /api/dashboard                     # authenticated operator
GET    /api/connector/gmessages/status    # authenticated operator
POST   /api/connector/gmessages/pair
POST   /api/connector/gmessages/pair/refresh
POST   /api/connector/gmessages/reconnect
DELETE /api/connector/gmessages/pair
```

PocketBase owns `/api/health`, auth, records and realtime endpoints.

The SPA wildcard explicitly refuses `api` and `_` namespaces so malformed/unknown API calls cannot accidentally return `index.html` with status 200.

Request bodies have route-level limits in addition to collection field validation.

## 12. Authentication

State-changing payment API calls accept either:

- `Authorization: Bearer <PAYGATE_API_KEY>`; or
- a valid authenticated `users` PocketBase token used by the operator UI.

SMS ingestion uses a separate `X-Webhook-Secret`.

The migration compatibility route `/api/webhook` is separately gated by `LEGACY_SMS_WEBHOOK_ENABLED` and the old `WEBHOOK_SECRET`; it never substitutes for the primary `SMS_WEBHOOK_SECRET`.

PocketBase's built-in rate limiter is enabled by default by the PayGate startup configuration.

## 13. Outgoing webhook outbox

A state transition schedules a `webhook_deliveries` row inside the same transaction. The worker claims due rows with a transactional state change to `sending`, preventing two worker passes from intentionally delivering the same row simultaneously.

A stale sending lease is recovered after a process crash. Retry timing is persisted. A successful HTTP 2xx response marks the delivery `delivered`; failures are retried until the configured fixed attempt ceiling and then become `exhausted`.

The signature is HMAC-SHA256 over:

```text
unixTimestamp + "." + rawJSONBody
```

The stable event ID lets consumers handle network-level duplicate delivery safely.

## 14. UI/realtime

The UI reads `payments`, `sms_events` and `webhook_deliveries` through authenticated PocketBase record APIs and subscribes to PocketBase realtime events. It uses custom PayGate routes for state changes.

Authentication is refreshed periodically; a failed authenticated API response clears the local auth store instead of leaving the UI in a misleading signed-in state.

Payment-create UI retries keep the same idempotency key until form values change or a request succeeds.

## 15. Process lifecycle

At startup:

1. strict environment parsing;
2. PocketBase boot/migrations;
3. serve-time validation of required/strong secrets and URLs;
4. enable configured PocketBase rate limiting;
5. start webhook worker;
6. start libgm only when enabled and paired.

Cron jobs also check expirations and wake webhook delivery processing. The worker itself has an internal timer, so delayed delivery does not depend on a single cron tick.

At termination the root context is cancelled and the connector is disconnected cleanly.

## 16. Persistence and backups

All durable state lives under the configured PocketBase data directory. In the production image that is `/app/pb_data`.

A deployment without a persistent mount there is invalid. Recreating a task/container must not recreate the database.

The old prototype database is not migrated into the new schema automatically. Before the production cutover the old task data is separately backed up for rollback/reference.

## 17. Failure philosophy

PayGate fails closed where a false confirmation is possible:

- ambiguous exact amount → no payment confirmation;
- RRN/amount contradiction → error, no reassignment;
- old evidence predating a reused payment → unmatched;
- missing RRN → audit error, no automatic confirmation;
- unavailable Google Messages → persisted core remains intact;
- outgoing merchant webhook unavailable → payment state remains committed and delivery retries durably.

## 18. Project layout

```text
cmd/payment-api/          process/CLI wiring
internal/api/             HTTP routes and auth boundary
internal/config/          strict environment configuration
internal/domain/          domain types/errors
internal/gmessages/       libgm adapter/session lifecycle
internal/money/           integer-INR helpers
internal/payments/        allocation/state machine/matching
internal/sms/             parser and durable ingestion
internal/webhooks/        durable outbound delivery worker
internal/web/             embedded compiled UI
migrations/               PocketBase collection schema
web/                      React/Vite source
```

## 19. Core invariants

1. Payment amounts are integer paise.
2. Requested amount is whole rupees; DDM owns `.01`–`.99`.
3. SQLite is the source of truth.
4. Payment creation and amount reservation are one transaction.
5. Evidence never matches by whole-rupee base.
6. RRN cannot silently move between different amounts.
7. A message cannot confirm a payment created after that message occurred when an occurrence timestamp is known.
8. Reused fingerprints pass through quarantine.
9. External HTTP calls never run inside a payment/SMS transaction.
10. Domain writes are backend-owned.
11. Google Messages is a replaceable evidence connector, not the payment model.
12. `/app/pb_data` must be persistent in production.
