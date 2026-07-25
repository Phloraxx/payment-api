# Payment API — PocketBase Rebuild

> **Status:** architecture/research branch. Implementation has not started yet.
>
> `main` remains the working prototype. This branch (`rebuild-pocketbase`) preserves that code while defining a clean rebuild from first principles.

A self-hosted hobby UPI payment-verification service. It creates a unique payable amount, watches the account's bank SMS through Google Messages, matches the exact amount and UPI reference, and exposes the result through an API and realtime UI.

This is **not** intended to hold funds or replace UPI/bank infrastructure. It is a personal payment-detection/orchestration layer built around an account that already receives UPI payments.

## Why rebuild

The prototype proved the core idea, but its architecture accumulated several shortcuts:

- payment allocation and expiry rely on in-memory state;
- pending payments are invalidated on restart;
- the current bank-SMS path can reduce an exact amount to its whole-rupee base before matching;
- SMS delivery depends on a custom Android relay application;
- custom WebSockets, admin authentication, logging databases and external replication duplicate functionality that PocketBase can already provide;
- the code is organised around `tickets` rather than a durable payment lifecycle.

The rebuild keeps the useful idea — **exact dynamic amount matching** — and discards the prototype-specific workarounds.

## Target architecture

```text
Android phone
    │
    │ bank SMS
    ▼
Google Messages
    │
    │ paired Messages-for-Web protocol
    ▼
libgm
    │
    ▼
┌──────────────────────────────────────────────┐
│              Payment API (Go)                │
│                                              │
│  PocketBase framework                        │
│   ├─ SQLite                                  │
│   ├─ migrations                              │
│   ├─ auth                                    │
│   ├─ REST/realtime                           │
│   ├─ logs/backups/admin `/_/`                │
│   └─ cron                                    │
│                                              │
│  custom Go                                   │
│   ├─ payment service                         │
│   ├─ exact-amount allocator                  │
│   ├─ SMS parser                              │
│   └─ Google Messages manager                 │
│                                              │
│  static React + Vite UI                      │
└──────────────────────────────────────────────┘
```

The deployment target is intentionally small:

- **one Go process**;
- **one PocketBase/SQLite database**;
- **one Docker container**;
- **one persistent `pb_data` volume**;
- no Redis, queue, Appwrite replica, separate logs database or custom WebSocket server.

## PocketBase's role

PocketBase is used as a Go framework, not forked and not treated as a separate service.

We keep PocketBase's built-in admin UI at `/_/` untouched. It is a developer/debug console for raw records, schema, logs, backups and cron jobs.

The normal application UI lives at `/` and is purpose-built for payments. It is compiled from React/Vite and served by the same Go application.

## Core data

The first implementation should need only two domain collections.

### `payments`

Stores the durable payment state:

- requested amount in integer paise;
- exact payable amount in integer paise;
- status (`pending`, `paid`, `expired`, `cancelled`);
- expiry and amount-reuse timestamps;
- RRN/UPI reference;
- payer UPI ID/name when available;
- creation and payment timestamps.

### `sms_events`

Stores the evidence and debugging trail:

- Google Messages message ID;
- sender and SMS body;
- phone/message timestamp and server ingestion timestamp;
- parsed amount, RRN and UPI ID;
- matched payment;
- processing status/error.

Google Messages authentication/session material is **not** a normal API collection. It should be stored as private connector state under the persistent data directory with restrictive permissions.

## Payment flow

```text
POST /api/payments { amount: 100 }
        │
        ▼
allocate exact amount transactionally
        │
        ├─ ₹100.00 available → ₹100.00
        ├─ otherwise          → ₹100.01
        └─ ...                → ₹100.99 / spillover block
        │
        ▼
payment = pending
        │
        ▼
customer pays exact amount
        │
        ▼
bank SMS → Google Messages → libgm
        │
        ▼
save sms_event first
        │
        ▼
parse amount + RRN + UPI ID
        │
        ▼
match exact payable_amount_paise
        │
        ▼
payment = paid
        │
        ├─ PocketBase realtime updates UI
        └─ optional outgoing webhook
```

All money is represented as integer paise. `₹100.37` is `10037`; no floating-point value participates in allocation or matching.

## Important design rules

1. **SQLite is the source of truth.** No payment state exists only in memory.
2. **Exact amount means exact amount.** A received `₹100.37` is matched as `10037`, never reduced to `₹100.00`.
3. **SMS is evidence, not an instruction.** The connector records an event; the payment service decides whether it matches.
4. **Duplicate delivery is normal.** Google message IDs and RRNs make processing idempotent.
5. **Expired amounts are quarantined before reuse.** A delayed SMS must not confirm a newer payment.
6. **External network work never happens inside a SQLite transaction.** Transactions stay short.
7. **PocketBase realtime replaces custom payment WebSockets.**
8. **PocketBase `/_/` is not the product UI.** We do not fork its frontend.
9. **libgm is a connector, not the payment model.** If it breaks, payment data remains valid and inspectable.
10. **Do not add abstractions until a second implementation needs them.** No generic provider/plugin framework in v1.

## Planned API

The exact response shapes will be finalised during implementation, but the intended surface is small:

```text
POST   /api/payments
GET    /api/payments/{id}
POST   /api/payments/{id}/cancel

GET    /api/connector/gmessages/status
POST   /api/connector/gmessages/pair
POST   /api/connector/gmessages/reconnect
DELETE /api/connector/gmessages/pair

GET    /api/health
```

PocketBase's native record/realtime APIs may be used by the authenticated dashboard for read-only views. Payment state-changing operations go through custom Go routes/services.

## UI

The custom UI should remain small:

- Dashboard
- Payments
- Payment details / test payment
- SMS events
- Google Messages connection/pairing
- Settings

PocketBase `/_/` remains available for low-level debugging instead of recreating database/log/schema tooling in the custom UI.

## Google Messages caveat

`libgm` is an open-source reverse-engineered Google Messages client, not an official Google API. Google can change the protocol. The first implementation milestone is therefore a connector feasibility test before we build the rest of the UI around it.

The phone remains part of the system: it must receive the SMS and have data/Wi-Fi for Google Messages for Web to sync. Connection loss must be visible, and reconnection/catch-up must be tested rather than assumed.

## Documentation

- [`RESEARCH.md`](RESEARCH.md) — verified capabilities, constraints and source references
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — target technical design and data flow
- [`PLAN.md`](PLAN.md) — implementation phases, acceptance criteria and test matrix

## Branch strategy

- `main` — preserve the current TypeScript/Fastify prototype until the rebuild is proven
- `rebuild-pocketbase` — architecture research, then clean Go/PocketBase implementation

The prototype is reference material, **not** a migration target. New code should be written around the new model rather than incrementally preserving old workarounds.
