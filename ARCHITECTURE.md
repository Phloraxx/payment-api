# Payment API Rebuild — Architecture

This document defines the target architecture for the clean rebuild. `PLAN.md` defines implementation order; `RESEARCH.md` records the evidence behind these decisions.

## 1. Design goals

The system should be:

- simple enough for one person to understand and operate;
- durable across process/container restarts;
- correct about exact monetary amounts;
- resilient to duplicate and delayed SMS delivery;
- observable when the Android/Google Messages path is degraded;
- easy to self-host on the existing Dokploy server;
- small enough that PocketBase replaces infrastructure instead of becoming another dependency beside duplicate infrastructure.

This is a personal hobby service. It does not need multi-tenant/payment-company architecture.

---

## 2. System boundary

```text
                 UPI payment
Customer ─────────────────────────► bank account
                                        │
                                        │ credit SMS
                                        ▼
                                 Android phone
                                        │
                                 Google Messages
                                        │
                                  Messages Web
                                        │
                                        ▼
                                      libgm
                                        │
                                        ▼
┌───────────────────────────────────────────────────────┐
│                  Payment API process                  │
│                                                       │
│  GoogleMessagesManager ──► SmsService                 │
│                               │                       │
│                               ▼                       │
│                         PaymentService                │
│                               │                       │
│                    PocketBase / SQLite                │
│                      │              │                 │
│                  payments       sms_events            │
│                      │              │                 │
│                      └──── realtime ┘                 │
│                               │                       │
│                     custom REST routes                │
│                               │                       │
│                 React/Vite static frontend            │
│                                                       │
│       PocketBase `/_/` stays untouched/admin-only     │
└───────────────────────────────────────────────────────┘
```

The Android phone remains part of the payment-observation path. libgm replaces the custom relay application, not the phone/SIM.

---

## 3. Runtime architecture

### One Go process

The application embeds PocketBase as a Go package.

The process owns:

- HTTP server and custom routes;
- PocketBase admin/API/realtime;
- SQLite connection;
- migrations;
- scheduled expiry/cleanup jobs;
- payment business logic;
- SMS parsing;
- libgm client lifecycle;
- static frontend serving.

Do not split these into services unless a later requirement proves that necessary.

### One runtime container

```text
Docker container
├── payment-api binary
├── compiled frontend assets
└── /app/pb_data  ← persistent volume
```

The frontend build may use Node in a Docker build stage, but Node is not required in the final runtime image.

---

## 4. PocketBase responsibilities

Use PocketBase for functionality it already provides:

- SQLite/database access;
- collections and schema;
- Go migrations;
- authentication;
- record API for controlled reads;
- SSE realtime subscriptions;
- application logging;
- backups;
- health/admin plumbing;
- cron jobs;
- built-in superuser dashboard at `/_/`.

Do not fork PocketBase core or its admin frontend.

### UI separation

```text
/     custom payment UI
/_/   PocketBase superuser/debug UI
```

The custom UI is built against the PocketBase backend, not layered on or injected into the PocketBase admin frontend.

---

## 5. Domain model

Keep the first version deliberately small.

### 5.1 `payments`

A normal PocketBase collection controlled by custom payment routes for writes.

Suggested fields:

| Field | Type | Purpose |
|---|---|---|
| `requested_amount` | number/integer | Requested value in paise |
| `payable_amount` | number/integer | Exact allocated value in paise |
| `status` | select | `pending`, `paid`, `expired`, `cancelled` |
| `expires_at` | date | Payment validity deadline |
| `reuse_after` | date | Earliest time this exact amount may be allocated again |
| `rrn` | text | Bank UPI reference/RRN |
| `upi_id` | text | Payer UPI ID when available |
| `payer_name` | text | Payer name when available |
| `paid_at` | date | Confirmation time |
| `metadata` | JSON | Optional caller metadata/order identifier |
| `created` | system | PocketBase timestamp |
| `updated` | system | PocketBase timestamp |

Useful indexes:

- `payable_amount`;
- `status, expires_at`;
- `rrn` (unique when non-empty if schema/index behaviour allows the desired null/empty semantics);
- `created` for recent listings.

Business code must still explicitly handle duplicate RRNs even when a DB constraint exists.

### 5.2 `sms_events`

A durable evidence/debug collection.

| Field | Type | Purpose |
|---|---|---|
| `source` | select/text | `google_messages` initially |
| `source_message_id` | text | Stable connector dedupe key |
| `sender` | text | SMS sender/address |
| `body` | text | Raw SMS body |
| `message_at` | date | Source/phone timestamp when available |
| `received_at` | date | Time our process received/reconciled it |
| `processing_status` | select | `received`, `parsed`, `matched`, `ignored`, `error` |
| `parsed_amount` | number/integer | Parsed paise |
| `rrn` | text | Parsed RRN |
| `upi_id` | text | Parsed UPI ID |
| `payer_name` | text | Parsed payer name |
| `matched_payment` | relation | Optional relation to `payments` |
| `error` | text | Parse/match error or ignore reason |

`source + source_message_id` should be unique.

### 5.3 Auth collection

If the dashboard needs login, use one small PocketBase auth collection such as `operators`.

For a personal deployment this may contain a single account.

Do not create organisations, roles or permissions beyond what is actually needed.

### 5.4 Google Messages session

Do not expose libgm session/auth state as an ordinary collection.

Store the serialised connector session under a private path such as:

```text
pb_data/private/gmessages/session.json
```

Requirements:

- never served as a static file;
- restrictive file permissions;
- never logged;
- included intentionally in backup/restore testing;
- corruption/missing state must degrade to `unpaired`, not stop PocketBase/payment history from starting.

The session contains credentials/keys/tokens and must be treated as a secret.

---

## 6. Payment creation and amount allocation

### Money representation

All business logic uses integer paise.

```text
₹1.00    → 100
₹100.00  → 10000
₹100.37  → 10037
```

Convert request input at the API boundary and never use a floating-point amount afterward.

### Transactional allocator

`CreatePayment()` runs a short `RunInTransaction` callback.

Pseudo-flow:

```text
requested = 10000
candidate = 10000

for bounded candidate blocks:
    for decimal 00..99:
        candidate = block + decimal

        if candidate has pending payment:
            continue

        if candidate belongs to terminal payment with reuse_after > now:
            continue

        INSERT payment(candidate, pending, expires_at, reuse_after)
        COMMIT
        return payment

return allocation exhausted
```

SQLite's single-writer transaction model ensures two simultaneous creators cannot both pass the check and insert the same candidate when allocation stays inside one transaction.

### Spillover

After `₹100.00..₹100.99`, a bounded safety valve may use `₹101.00..₹101.99`, and so on.

The number of spillover blocks must be configurable/bounded so a bug cannot create unexpectedly high payable amounts.

---

## 7. Payment lifecycle

```text
                    ┌─────────────┐
              ┌────►│    paid     │
              │     └─────────────┘
              │
┌──────────┐  │     ┌─────────────┐
│ pending  │──┼────►│   expired   │
└──────────┘  │     └─────────────┘
              │
              │     ┌─────────────┐
              └────►│  cancelled  │
                    └─────────────┘
```

Only `pending` can automatically become `paid`.

### Expiry

A cron job marks stale pending payments expired, but timestamp checks also occur at business-operation time. A delayed cron job must not extend a payment accidentally.

### Amount quarantine

Terminal records retain `reuse_after`.

Until then, their exact amount remains unavailable for new allocation.

This prevents:

```text
old payment expires
        │
amount immediately reused
        │
old delayed SMS arrives
        │
WRONG new payment marked paid
```

The initial quarantine should be conservative and tuned from measured message delays/replays rather than guessed for maximum throughput.

---

## 8. SMS ingestion

### Rule: persist before processing

```text
incoming/recovered message
        │
        ▼
insert/find sms_events
        │
        ├─ duplicate source_message_id → no-op
        │
        ▼
parse message
        │
        ├─ irrelevant → ignored
        ├─ bad format → error
        └─ payment evidence
                 │
                 ▼
           MatchPayment()
```

This means a parser bug never destroys the only copy of the evidence.

### Bank parser

Start only with actual SMS formats observed by the project.

The first parser should extract:

- exact credited amount;
- RRN/reference;
- payer UPI ID when present;
- payer name when present.

No generic parser plugin framework is needed until a second genuinely different bank format exists.

---

## 9. Exact payment matching

Matching receives normalised evidence, not raw SMS.

Required checks:

1. parsed exact amount exists;
2. RRN is not already attached to another successful payment;
3. exactly one eligible `pending` payment has `payable_amount == parsed_amount`;
4. the payment has not expired at the evidence/processing boundary according to the chosen policy;
5. update the payment atomically to `paid`, attach evidence fields and `paid_at`;
6. update `sms_event.matched_payment` and processing status.

### Important invariant

```text
10037 matches 10037.
```

Never floor to `10000`, never infer by base rupee amount.

### Duplicate semantics

If the same source message or RRN is processed again, return a no-op/already-processed result rather than treating it as a fatal system error.

---

## 10. Google Messages connector

### libgm boundary

`GoogleMessagesManager` owns libgm. Payment code does not import or depend on Google-specific event structures.

```text
libgm event
   │
   ▼
GoogleMessagesManager
   │ converts to simple SMS input
   ▼
SmsService
```

### Responsibilities

- load/persist auth state;
- initiate pairing;
- connect/disconnect;
- reconnect with bounded exponential backoff;
- expose connection health;
- receive messages;
- reconcile/catch up after reconnect using the libgm behaviour proven in the Phase 0 spike;
- forward only SMS-relevant events to `SmsService`.

### Pairing

Do not hard-code the final UX until Phase 0.

Current upstream code contains both Google-account/emoji pairing logic and QR-pairing implementation, while Google documents QR pairing availability outside the US. The spike decides which flow is reliable on the actual Indian phone/account.

The final dashboard should reflect the tested flow rather than inventing a pairing abstraction first.

### Connector health

Expose at least:

```text
paired
connected
last_connected_at
last_disconnected_at
last_message_at
last_error
```

The frontend should visibly show degraded/disconnected/unpaired state.

---

## 11. HTTP/API boundary

### External/personal-project API

Initial routes:

```text
POST /api/payments
GET  /api/payments/{id}
POST /api/payments/{id}/cancel
GET  /api/health
```

Use one configured bearer key for state-changing API access in v1.

### Connector routes

Authenticated dashboard routes:

```text
GET    /api/connector/gmessages/status
POST   /api/connector/gmessages/pair
POST   /api/connector/gmessages/reconnect
DELETE /api/connector/gmessages/pair
```

Additional pairing-step routes may be added once the chosen libgm flow is proven.

### Generic PocketBase record APIs

Use them carefully:

- authenticated UI may read/list payments and SMS events;
- realtime subscriptions may read permitted records;
- direct generic create/update/delete for payment collections should be denied;
- payment transitions occur through Go service methods/custom routes.

This prevents a browser edit from bypassing payment invariants.

---

## 12. Realtime/UI

PocketBase SSE is the only realtime channel in v1.

```text
PaymentService updates `payments`
        │
        ▼
PocketBase record event
        │
        ▼
PocketBase JS SDK subscription
        │
        ▼
React UI updates
```

No Socket.IO/custom WebSocket manager.

### UI pages

Keep the first UI to approximately:

```text
Dashboard
Payments
Payment Detail / New Payment
SMS Events
Google Messages
Settings
```

The PocketBase dashboard at `/_/` handles schema, raw records, logs, backups and cron inspection. Do not duplicate those administrative capabilities unless the normal workflow truly requires them.

---

## 13. Scheduled work

Use PocketBase's built-in cron only for small periodic tasks such as:

- expire stale pending payments;
- optional cleanup/retention later;
- optional webhook retries later.

Do not schedule one timer/goroutine per payment.

The Google Messages connection itself is long-running connector lifecycle code, not a cron job.

---

## 14. Outgoing webhooks (optional v1.1)

Once payment detection is stable, support one destination configured by environment/settings.

A successful payment can emit:

```json
{
  "event": "payment.paid",
  "payment": {
    "id": "...",
    "requestedAmount": 10000,
    "payableAmount": 10037,
    "rrn": "..."
  }
}
```

Use an HMAC signature.

Only introduce a `webhook_deliveries` collection if retries/history are actually implemented.

---

## 15. Failure behaviour

### PocketBase/process restart

- payment state survives;
- pending payments remain pending until their real `expires_at`;
- no mass-expire-on-start behaviour;
- connector attempts session restore independently.

### Google Messages unavailable

- payment records remain valid;
- connector state becomes degraded;
- payment stays pending/ultimately expires;
- later recovered messages are processed idempotently if available.

### Session invalid/unpaired

- historical data remains available;
- connector reports `unpaired`;
- UI requests pairing;
- application does not crash-loop.

### Malformed bank SMS

- retained in `sms_events`;
- status `error`/`ignored` with reason;
- no payment state change.

### Duplicate SMS

- same source ID: no-op;
- same RRN under a different source message: no second confirmation.

---

## 16. Logging and observability

Do not create a second log database.

Use:

1. PocketBase/application logs for technical messages;
2. `sms_events` for payment evidence and parser/match debugging;
3. payment timestamps for lifecycle history.

Connector logs should include state transitions but must redact:

- cookies;
- Tachyon auth tokens;
- crypto keys;
- full serialised libgm `AuthData`.

Useful timings to collect from the start:

```text
message_at
received_at
parsed_at (optional)
paid_at
```

This allows actual latency measurement instead of assumptions about Google Messages.

---

## 17. Security model

Reasonable hobby-project baseline:

- HTTPS through existing reverse proxy/Cloudflare setup;
- operator authentication for the custom dashboard;
- superuser credentials only for `/_/`;
- generic payment collection writes disabled;
- bearer secret for external create/cancel APIs;
- connector auth state private on disk;
- `.env`/session files ignored by Git;
- no secrets in logs;
- CSRF/browser protections follow the authentication method chosen for the custom UI;
- raw SMS is considered sensitive data.

No custom WebAuthn stack in the first rebuild.

---

## 18. Backup model

The persistent state is intentionally compact:

```text
pb_data/
├── PocketBase SQLite/data
└── private/gmessages/session...
```

Use PocketBase backup capabilities and/or volume snapshots.

A restore is only trusted after testing:

- database restoration;
- migrations;
- payment history;
- connector session restoration;
- graceful re-pair request if connector credentials cannot be restored.

---

## 19. Project structure target

Keep structure shallow until code size forces more separation.

```text
payment-api/
├── cmd/
│   └── payment-api/
│       └── main.go
├── internal/
│   ├── config/
│   ├── payments/
│   │   ├── service.go
│   │   ├── allocator.go
│   │   └── money.go
│   ├── sms/
│   │   ├── service.go
│   │   └── kotak.go
│   ├── gmessages/
│   │   ├── manager.go
│   │   └── session.go
│   └── api/
│       └── routes.go
├── migrations/
├── web/
│   ├── package.json
│   └── src/
├── Dockerfile
├── README.md
├── PLAN.md
├── ARCHITECTURE.md
└── RESEARCH.md
```

Do not create repository/service/provider/interface layers merely for symmetry. Extract them only when tests or a second implementation make the boundary useful.

---

## 20. Architecture invariants

These should remain true as implementation evolves:

1. **Payment truth is durable SQLite data.**
2. **Exact payable amount is matched exactly.**
3. **No important payment timer/pool exists only in RAM.**
4. **Incoming SMS is persisted before matching.**
5. **Duplicate input is harmless.**
6. **Amounts are not reused until their quarantine allows it.**
7. **libgm failure does not corrupt payment state.**
8. **PocketBase admin remains upstream/unmodified.**
9. **PocketBase realtime is sufficient until proven otherwise.**
10. **One process/container is the default architecture.**
11. **External calls do not occur inside SQLite write transactions.**
12. **Complexity must be justified by an observed problem, not a hypothetical future scale.**
