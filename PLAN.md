# Payment API Rebuild — Implementation Plan

> Branch: `rebuild-pocketbase`
>
> Goal: rebuild the prototype as a small, durable, self-hosted hobby payment-verification service using **Go + PocketBase + SQLite + libgm + React/Vite**.

This plan intentionally optimises for simplicity. The project is personal infrastructure, not a commercial payment processor. We should still preserve the properties that matter for payment correctness: durable state, exact matching, idempotency, conservative amount reuse, observable connector health and reproducible tests.

---

## 1. Decisions already made

### Keep

- Dynamic exact-amount matching (`₹100.00` → `₹100.xx`).
- Integer-paise money representation.
- Bank SMS as the authoritative evidence for a UPI credit.
- RRN/UPI reference deduplication.
- A simple API that other personal projects can call.
- A custom dashboard.
- Docker/Dokploy deployment.

### Replace

| Prototype | Rebuild |
|---|---|
| Node/Fastify backend | Go application embedding PocketBase |
| `better-sqlite3` directly | PocketBase SQLite/database APIs |
| in-memory decimal pool | transactional database allocation |
| per-ticket `setTimeout` | timestamps + PocketBase cron |
| custom WebSocket manager | PocketBase realtime SSE |
| custom Android relay app | Google Messages + libgm |
| custom passkey/admin stack | PocketBase auth for app UI; PocketBase superuser UI for raw admin |
| separate logs DB | PocketBase logs + `sms_events` domain audit records |
| Appwrite replica | removed |
| manual schema setup | PocketBase Go migrations |

### Explicit non-goals for v1

Do **not** add these unless a concrete need appears:

- PostgreSQL
- Redis
- RabbitMQ/NATS/Kafka
- microservices
- Kubernetes
- Appwrite/Supabase replication
- generic provider/plugin architecture
- multi-merchant organisations/roles
- OAuth/API-key management UI
- custom WebSocket infrastructure
- custom PocketBase admin fork
- GraphQL
- separate analytics database
- distributed locks

---

## 2. First principle: prove libgm before rebuilding everything

The highest-risk dependency is not PocketBase; it is the unofficial Google Messages protocol.

Before spending time on the payment UI or replacing the prototype, build a small disposable Go spike that proves the actual phone path.

### Phase 0 — Google Messages feasibility spike

Create a tiny command under something like `cmd/gm-spike/`.

It must prove:

1. PocketBase is not involved yet.
2. Initialise a `libgm.Client`.
3. Pair the Android phone.
4. Persist `libgm.AuthData`.
5. Restart the process and reconnect without repairing.
6. Receive an incoming SMS event.
7. Identify the message body, sender/message ID and timestamp we need.
8. Disconnect network on the VPS, receive an SMS on the phone, restore network and test whether the message is recovered/caught up.
9. Lock the phone and repeat.
10. Enable Android Battery Saver and repeat.
11. Test with weak/slow mobile data if practical.
12. Leave the connector running for at least a realistic unattended period and observe reconnect behaviour.

### Record measurements

For every test SMS record:

- phone/SMS timestamp if available;
- connector event timestamp;
- process reconnect timestamp;
- whether event was live or recovered;
- duplicate count;
- phone lock state;
- battery mode;
- Wi-Fi/mobile-data state.

### Phase 0 acceptance criteria

Do not proceed with libgm as the primary connector until all of these are true:

- pairing works on the actual Android phone;
- auth can be persisted and restored;
- restart reconnect works;
- at least one real bank SMS is received and parsed;
- duplicate/replayed messages can be identified;
- connection loss is detectable;
- recovery behaviour is understood and documented;
- the one-active-computer Google Messages limitation is acceptable for this phone.

If the spike fails, retain the Android relay as the connector and still continue with the PocketBase rebuild. The payment architecture must not depend on libgm succeeding.

---

## 3. Phase 1 — Clean application skeleton

Only after Phase 0 provides enough confidence.

### Backend

Create a new Go application around PocketBase:

```text
cmd/payment-api/main.go
internal/
  config/
  payments/
  sms/
  gmessages/
  api/
migrations/
web/
```

Initial responsibilities:

- create `pocketbase.New()`;
- register Go migrations;
- register custom API routes;
- register the expiry/cleanup cron;
- serve the compiled frontend;
- expose a health endpoint;
- start/stop the Google Messages manager with application lifecycle.

### Frontend

Use only:

- React
- Vite
- TypeScript
- PocketBase JS SDK

No Next.js/SSR/Redux/GraphQL.

### Deployment

Target one runtime container:

```text
payment-api
  ├─ single Go binary
  ├─ embedded/served web assets
  └─ /app/pb_data persistent volume
```

A multi-stage Docker build may use Node only to compile the frontend.

### Acceptance criteria

- application boots with an empty `pb_data` directory;
- migrations run automatically;
- PocketBase `/_/` works;
- custom `/` UI is served;
- `/api/health` reports healthy;
- restart preserves data;
- Docker image runs in Dokploy with one volume.

---

## 4. Phase 2 — Durable payment model

### Collection: `payments`

Minimum fields:

```text
id                    PocketBase record id
requested_amount      integer paise
payable_amount        integer paise
status                pending | paid | expired | cancelled
expires_at            datetime
reuse_after           datetime
rrn                    text, optional
upi_id                 text, optional
payer_name             text, optional
paid_at                datetime, optional
metadata               JSON, optional
created                PocketBase created timestamp
updated                PocketBase updated timestamp
```

Indexes should support:

- status + expiry scans;
- exact payable amount lookup;
- RRN lookup/uniqueness where practical;
- recent payments listing.

### Money rules

- API may accept rupees as a decimal string/number for convenience.
- Convert immediately to integer paise.
- Internal payment code only accepts integer paise.
- Never use float equality for allocation or matching.

### Payment lifecycle

```text
          create
            │
            ▼
         pending
       /    │     \
      /     │      \
   paid   expired  cancelled
```

No in-memory state is authoritative.

### Expiry

A payment is expired based on `expires_at`, not a Go timer.

A PocketBase cron job periodically marks stale `pending` payments `expired` and sets/maintains `reuse_after`.

Every operation that tries to pay/cancel a payment must also validate its current timestamps; correctness must not depend on the cron firing at an exact second.

---

## 5. Phase 3 — Exact amount allocator

The allocator is part of `PaymentService` and runs in a short SQLite transaction.

### Inputs

```text
requested_amount_paise
now
payment TTL
reuse/quarantine policy
```

### Algorithm

For a request of `10000` paise:

1. Start with block `10000..10099`.
2. For each candidate exact amount:
   - candidate must not have a current `pending` payment;
   - candidate must not be inside another payment's `reuse_after` quarantine window.
3. Select the first safe candidate.
4. Insert the new `payments` record inside the same transaction.
5. If all 100 candidates are unavailable, move to the next block (`10100..10199`) as a safety valve.
6. Bound spillover with a configured maximum so a bug cannot allocate arbitrarily high amounts.

Because PocketBase/SQLite permits a single writer transaction at a time, the check-and-insert sequence is naturally serialised as long as the entire allocation is inside one short transaction.

### Important difference from the prototype

Received `₹100.37` is `10037`.

Matching is:

```text
WHERE payable_amount = 10037
AND status = 'pending'
```

It is **never** transformed to `10000` and matched by a base amount.

### Reuse policy

Do not optimise this prematurely.

Initial rule:

- pending amount: unavailable;
- terminal payment: unavailable until `reuse_after`;
- `reuse_after` duration is configurable;
- start conservative and tune from real observed SMS delay/replay data.

The system must prefer failing to auto-match over falsely confirming the wrong payment.

### Tests

- first candidate allocation;
- sequential allocation;
- two concurrent creates cannot receive the same exact amount;
- 100-slot spillover;
- configured spillover bound;
- amount remains unavailable during quarantine;
- amount becomes reusable afterward;
- exact amount lookup does not confuse `10001` and `10002`.

---

## 6. Phase 4 — SMS event model and parser

### Collection: `sms_events`

Minimum fields:

```text
id
source                 google_messages
source_message_id      unique/dedup key
sender
body
message_at
received_at
processing_status      received | parsed | matched | ignored | error
parsed_amount           integer paise, optional
rrn                     optional
upi_id                  optional
payer_name              optional
matched_payment         relation -> payments, optional
error                   optional
created
updated
```

### Processing rule

Always save the event before attempting to match it.

```text
libgm event
   │
   ▼
insert/find sms_event by source_message_id
   │
   ├─ already processed → no-op
   │
   ▼
parse
   │
   ├─ irrelevant SMS → ignored
   ├─ parse error     → error
   └─ payment evidence
            │
            ▼
       PaymentService.Match
```

### Parser design

Start with the real bank format currently used by the project (Kotak) and only add parsers when real samples exist.

Do not build a plugin registry until there is a second bank implementation.

Parser output should be a small value object:

```text
amount_paise
rrn
upi_id
payer_name
message_time
```

### Idempotency

Two independent dedupe keys are useful:

1. Google/source message ID prevents processing the same message event repeatedly.
2. RRN prevents the same UPI transaction from confirming multiple payments.

Duplicate delivery should be a successful no-op, not an exceptional crash path.

---

## 7. Phase 5 — Google Messages manager

Integrate libgm directly in the Go process.

### Responsibilities

`GoogleMessagesManager` owns only connector concerns:

- load/save private auth/session state;
- pair/unpair;
- connect;
- reconnect with backoff;
- expose health/status;
- receive events;
- hand SMS messages to `SmsService`;
- request/reconcile updates after reconnection if supported by the tested libgm flow.

It must **not** contain payment matching logic.

### Connector state

Keep small operational state such as:

```text
paired
connected
last_connected_at
last_disconnected_at
last_event_at
last_error
```

This can be exposed through a custom status endpoint and UI.

Sensitive `libgm.AuthData` should not be a normal public PocketBase collection. Store it under the persistent data directory with restrictive file permissions. If a practical encryption-at-rest key is introduced, keep the key outside the data file/backup.

### Connection lifecycle

```text
application boot
    │
    ├─ no session → disconnected/unpaired
    │
    └─ session exists
          │
          ▼
       connect
          │
     ┌────┴────┐
     │         │
  healthy   failure
     │         │
 events      backoff
     │         │
     └──── reconnect
```

### Do not hide degraded state

If the phone or Google relay is unavailable, the UI must clearly report it. A payment remaining pending is safer than pretending the connector is healthy.

---

## 8. Phase 6 — API

Keep custom write routes small.

### Initial external API

```text
POST   /api/payments
GET    /api/payments/{id}
POST   /api/payments/{id}/cancel
GET    /api/health
```

Protect personal-project API writes with one configured bearer API key initially. Do not build an API-key management subsystem yet.

### Dashboard/internal API

```text
GET    /api/connector/gmessages/status
POST   /api/connector/gmessages/pair
POST   /api/connector/gmessages/reconnect
DELETE /api/connector/gmessages/pair
```

The dashboard may use PocketBase collection APIs for authenticated read-only listing and realtime subscriptions.

Do not allow the browser to directly create/update payment records through generic PocketBase record APIs. State transitions belong to `PaymentService`.

---

## 9. Phase 7 — Custom UI

PocketBase's admin dashboard remains untouched at `/_/`.

Our UI should cover only normal workflows:

### Dashboard

- connector status;
- pending/paid counts;
- recent payments;
- most recent SMS event/error.

### Payments

- create test/payment;
- list/filter payments;
- payment detail;
- exact requested vs payable amount;
- status and timestamps;
- RRN/UPI ID when available;
- cancel pending payment.

### SMS Events

- recent events;
- parsed fields;
- matched payment;
- ignored/error reason.

This is primarily a debugging surface, so keep it practical rather than decorative.

### Google Messages

- paired/unpaired;
- connected/disconnected;
- pair flow;
- reconnect;
- unpair;
- last connected/event/error timestamps.

### Settings

Initially show configuration read-only where useful. Do not duplicate every PocketBase setting from `/_/`.

### Realtime

Use PocketBase collection subscriptions for payment/event updates. Do not create a parallel WebSocket protocol.

---

## 10. Phase 8 — Optional outgoing webhook

Only add after payment detection is reliable.

Use one configured destination initially:

```text
PAYMENT_WEBHOOK_URL
PAYMENT_WEBHOOK_SECRET
```

On a transition to `paid`:

```json
{
  "event": "payment.paid",
  "payment": { "id": "...", "amount": 10037, "rrn": "..." }
}
```

Sign with HMAC and include an event/delivery ID.

If retries become necessary, introduce a `webhook_deliveries` collection and PocketBase cron retry loop **then**. Do not create it before there is an outgoing webhook implementation.

---

## 11. Testing strategy

### Unit tests

- rupee/paise parsing;
- exact amount allocation;
- amount quarantine;
- SMS parser against captured, redacted real samples;
- state transitions;
- RRN/source-message dedupe.

### Database integration tests

Use a temporary PocketBase data directory and real SQLite transactions.

Test:

- migrations;
- create payment;
- exact-match confirmation;
- expiry after restart;
- allocation race/concurrency;
- duplicate SMS event;
- duplicate RRN;
- late/expired payment does not confirm a new payment.

### Connector tests

Split into:

- pure tests around our manager/session code with fakes where practical;
- manual/live tests against the actual phone because the Google protocol cannot be meaningfully validated only with mocks.

### End-to-end acceptance test

```text
1. Start from clean pb_data.
2. Pair phone.
3. Create payment for ₹100.
4. Receive payable amount, e.g. ₹100.03.
5. Pay exactly ₹100.03.
6. Real bank SMS arrives on Android.
7. libgm receives/reconciles SMS.
8. sms_event is persisted.
9. exact pending payment is marked paid.
10. React UI updates through PocketBase realtime.
11. Restart container.
12. payment and connector session remain available.
```

---

## 12. Failure scenarios that must be tested

| Failure | Expected behaviour |
|---|---|
| API process crashes | SQLite keeps payment state; restart does not expire valid payments solely because of restart |
| Phone locked | SMS path should continue if Google Messages remains connected; measure actual behaviour |
| Battery Saver | degraded/delayed behaviour is measured and surfaced, not assumed |
| Phone internet lost | connector becomes stale/disconnected; pending payments remain pending |
| VPS internet lost | reconnect later and reconcile/catch up where libgm allows |
| Google session expires/unpairs | status becomes unpaired; pairing is required again |
| duplicate Google event | no-op after existing `source_message_id` |
| duplicate bank SMS/RRN | cannot pay a second payment |
| delayed SMS after payment expiry | event is retained; must not falsely pay a newer payment using a quarantined amount |
| malformed SMS | retained as ignored/error for debugging |
| PocketBase realtime disconnect | browser SDK reconnects; payment state is still queryable from DB |

---

## 13. Security boundaries

This is a hobby service, but it handles sensitive SMS/session material.

Minimum safeguards:

- TLS at the reverse proxy;
- custom dashboard requires PocketBase user authentication or equivalent private access;
- payment create/cancel API protected by a bearer secret;
- `payments`/`sms_events` generic write rules locked down;
- libgm session file not served by PocketBase/static routes;
- session file permissions restricted;
- secrets excluded from Git;
- logs never print Google auth tokens/cookies;
- raw SMS retention configurable later if desired;
- PocketBase superuser `/_/` treated as a privileged debug interface.

Do not implement custom WebAuthn for v1. PocketBase auth or external private access is enough.

---

## 14. Backups and recovery

Use PocketBase's own backup capability plus normal volume backups.

Recovery test must verify:

1. restore `pb_data`;
2. payment records return;
3. migrations agree with restored schema;
4. Google Messages session restoration works if its private state is included;
5. if the session cannot be restored, the payment database still starts cleanly and the UI asks for re-pairing.

Do not let connector-session corruption prevent access to historical payments.

---

## 15. Recommended implementation order

Do not build horizontally across every feature. Finish vertical slices.

### Milestone A — prove connector

- libgm spike
- pairing
- persistence
- real SMS
- reconnect testing
- latency notes

### Milestone B — prove payment core

- PocketBase skeleton
- migrations
- `payments`
- transactional allocator
- exact matching with synthetic SMS events
- expiry/quarantine
- tests

### Milestone C — connect the two

- `sms_events`
- GoogleMessagesManager
- real SMS ingestion
- idempotency
- connector status

### Milestone D — usable UI

- auth
- dashboard
- payment create/list/detail
- SMS events
- connector pairing/status
- realtime

### Milestone E — self-host comfortably

- Docker
- Dokploy
- healthcheck
- backup/restore test
- optional outgoing webhook

---

## 16. Definition of v1 done

v1 is done when:

- one container deploys successfully;
- PocketBase is the durable source of truth;
- a payment can be created with an exact unique payable amount;
- payment state survives restart;
- bank SMS is received through the chosen connector;
- exact amount + RRN matching marks the correct payment paid;
- duplicate/replayed SMS cannot double-pay;
- late SMS cannot pay a newly reused amount during quarantine;
- the UI shows payment status in realtime;
- Google Messages health and pairing state are visible;
- the built-in PocketBase admin UI remains untouched;
- backup and restore has been manually tested;
- the old Android relay is no longer required for normal operation if libgm passes Phase 0.

Anything beyond that is v2 work, not a reason to delay the clean rebuild.
