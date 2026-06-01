<div align="center">

# Payment API

**v0.1.0**

A zero-fee UPI payment gateway that uses **Dynamic Decimal Matching** to resolve payments by parsing bank SMS notifications. Built with Fastify, SQLite, and TypeScript.

</div>

![Node](https://img.shields.io/badge/node-%3E%3D22-339933?logo=node.js)
![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?logo=typescript)
![Fastify](https://img.shields.io/badge/Fastify-5-000000?logo=fastify)
![SQLite](https://img.shields.io/badge/SQLite-better_sqlite3-003B57?logo=sqlite)

---

## Table of Contents

- [Architecture](#architecture)
- [Data Flow](#data-flow)
- [Dynamic Decimal Matching](#dynamic-decimal-matching)
- [SMS Processing](#sms-processing)
- [Timer & Expiry System](#timer--expiry-system)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Running](#running)

---

## Architecture

```
Fastify Server
│
├── Routes ────────▶ Services ──────────▶ SQLite (app.db)
│   POST /api/ticket    TicketService        tickets table
│   GET  /api/status    DecimalPool
│   POST /api/webhook   PaymentService
│   GET  /api/health
│
├── Middleware
│   Rate limit (@fastify/rate-limit)
│   Request logger (Pino)
│   Error handler (unified JSON responses)
│
├── In-Memory State
│   Map<base_amount, Set<taken_decimal>>
│   Map<ticket_id, setTimeout>  (expiry timers)
```

### Components

| Component | Role |
|-----------|------|
| **Fastify** | HTTP server with built-in rate limiting (`@fastify/rate-limit`), schema validation (`@sinclair/typebox`), and security headers (`@fastify/helmet`) |
| **TicketService** | Ticket CRUD via prepared statements. Manages per-ticket `setTimeout` handles for TTL expiry and decimal release |
| **DecimalPoolService** | In-memory pool of taken decimal values. `allocate()` finds the first free decimal 0-99, `release()` removes from the taken set |
| **PaymentService** | Parses incoming SMS using regex, dispatches to `confirmFromBankSms()` or `fillFromGenericSms()` |
| **SQLite** | Single `tickets` table in WAL mode. Synchronous driver (`better-sqlite3`) — no connection pool overhead |

---

## Data Flow

### Ticket Creation

```
POST /api/ticket { amount: 100 }

  1. toPaisa(100) → 10000 paisa
  2. DecimalPool.allocate(10000)
     → base = 10000
     → scan 0..99, decimal 00 is free
     → mark 00 as taken
     → return { amount: 10000, baseAmount: 10000, decimalVal: 0 }
  3. INSERT INTO tickets (...) VALUES ('TICKET...', 10000, 'pending', 10000, 0)
  4. setTimeout(() => onTtlReached(ticket), 2 * 60_000)
  5. Response: { ticketId: 'TICKET...', amount: 100, status: 'pending' }

ticket.amount = 10000 paisa = ₹100.00
```

### Payment Confirmation

```
Two SMSes arrive at POST /api/webhook { sms: "..." }

  BANK SMS (settlement notification):
    "Received Rs.100.00 from user@paytm UPI Ref:123456789"
    → parseSms → method: "bank", amount: 10000
    → confirmFromBankSms
      → SELECT WHERE base_amount = 10000 AND status = 'pending'
      → markPaid(ticket)
        → UPDATE status = 'paid', rrn = '123456789'
        → clearTimers() (cancel expiry + grace timers)
        → DecimalPool.release(10000, 0)
    → Response: { action: "marked_paid", ticketId: "TICKET..." }

  GENERIC SMS (UPI app notification, includes ticketId):
    "TICKET17123456780000 SOURAV paid you ₹100.00 UPI Ref:123456789"
    → parseSms → method: "generic", ticketId: "TICKET...", senderName: "SOURAV"
    → fillFromGenericSms
      → UPDATE sender_name = 'SOURAV' WHERE id = 'TICKET...'
    → Response: { action: "name_filled", ticketId: "TICKET..." }
```

The bank SMS is the authoritative payment signal. The generic SMS only fills the payer's name. Either can arrive first — `fillSenderName` works regardless of payment status.

---

## Dynamic Decimal Matching

### Problem

UPI payments only provide a transaction amount and reference number. Without a payment gateway callback, there is no way to know which customer paid for which ticket when multiple tickets share the same price.

### Solution

Replace the standard `₹100.00` amount with a unique `₹100.xx` amount drawn from a pool of 100 decimal variations. The exact amount encodes which ticket was paid.

### Pool Structure

```
Map<base_amount, Set<taken_decimal>>

base 10000 (₹100):  Set{ 00, 03, 07, 15 }    → 96 free slots
base 10100 (₹101):  Set{ 01, 02 }              → 98 free slots
base 10200 (₹102):  Set{ }                     → 100 free slots (untouched)
```

The pool stores only **taken** decimals. Free decimals are anything in 0..99 not in the set.

### Allocation

```
allocate(10000 paisa):
  1. base = baseAmountFromPaisa(10000) → 10000
  2. set = pools.get(10000) ?? new Set()
  3. for i = 0..99:
       if !set.has(i): set.add(i); return { amount: 10000 + i, baseAmount: 10000, decimalVal: i }
  4. All 100 taken → spillover
     for block = 10100, 10200, ...:
       set = pools.get(block) ?? new Set()
       if set.size < 100: return allocateFromBlock(block)
  5. throw POOL_EXHAUSTED
```

Sequential allocation (0, 1, 2, ...) is used rather than random to minimise fragmentation.

### Release Semantics

| Trigger | Decimal Release | Rationale |
|---------|----------------|-----------|
| **Paid** | Immediate | RRN deduplication in the DB prevents the same decimal from being double-matched |
| **Expired** | 30s delay | Prevents rapid recycling: a delayed SMS could match a freshly re-allocated decimal |
| **Cancelled** | 30s delay | Same anti-race protection |

### Spillover

When all 100 decimal slots for a base amount are taken, the next integer block is used. For example, ticket #101 for `₹100` will be allocated `₹101.00`. The price drift is bounded by the number of concurrent tickets at that price point.

### Startup Recovery

On server restart:

```sql
UPDATE tickets SET status = 'expired' WHERE status = 'pending';
SELECT base_amount, decimal_val, status FROM tickets;
```

1. All pending tickets are mass-expired (TTL state was in-memory and lost)
2. The pool is rebuilt from remaining `pending` and `paid` ticket decimals
3. No per-ticket timers are restored — fresh timers are created for new tickets

---

## SMS Processing

### SMS Formats

**Bank SMS** (settlement notification, no ticket ID):
```
Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959
from user@oksbi on 08-03-26.UPI Ref:606703736480.
```
→ Extracts `amount` (10000 paisa), `rrn`, `upi_id` → matches by `base_amount` → marks paid

**Generic** (UPI app notification, includes ticket ID):
```
TICKET17123456780000 SOURAV paid you ₹100.00 UPI Ref:606703736479
```
→ Extracts `ticketId`, `senderName` → matches by ID → fills `sender_name`

### RRN Deduplication

The `rrn` column has a `UNIQUE` constraint. If two webhook calls arrive with the same RRN (duplicate delivery), the second `UPDATE` throws a constraint error, which is caught and surfaced as `RRN_DUPLICATE`. This prevents the same transaction from marking two different tickets as paid.

---

## Timer & Expiry System

Each ticket has three associated timers managed in-memory:

```
Ticket Created
│
├── 2 min TTL ──────────▶ onTtlReached()
│                           │
│                           ├── 30s grace period
│                           │   ticket stays 'pending', bank SMS can still arrive
│                           │
│                           ▼
│                        graceExpired()
│                           │
│                           ├── UPDATE status = 'expired'
│                           └── 30s ──▶ DecimalPool.release()
│
├── Paid (via bank SMS) ──▶ clearTimers()
│                            └── DecimalPool.release()  (immediate)
│
└── Cancelled ─────────────▶ clearTimers()
                             └── 30s ──▶ DecimalPool.release()
```

On server restart, all timers are lost. The startup routine mass-expires any remaining pending tickets to maintain consistency:

```sql
UPDATE tickets SET status = 'expired', updated_at = datetime('now')
WHERE status = 'pending';
```

---

## API Reference

### `POST /api/ticket`

Create a payment ticket. A unique decimal amount is allocated from the pool.

```
Rate limit: 5 requests/minute/IP
```

**Request:**
```json
{ "amount": 100 }
```

**Response `200`:**
```json
{
  "ticketId": "TICKET17123456780000",
  "amount": 100,
  "amountPaisa": 10000,
  "status": "pending",
  "createdAt": "2026-06-01T12:00:00"
}
```

**Errors:**
| Code | Status | Condition |
|------|--------|-----------|
| `INVALID_AMOUNT` | 400 | Amount is not a positive number with ≤2 decimals |
| `POOL_EXHAUSTED` | 503 | All decimal slots for this and adjacent blocks are taken |

---

### `GET /api/status/:id`

Get the current status of a ticket.

```
Rate limit: 60 requests/minute/IP
```

**Response `200`:**
```json
{
  "ticketId": "TICKET17123456780000",
  "amount": 100,
  "amountPaisa": 10000,
  "status": "paid",
  "createdAt": "2026-06-01T12:00:00",
  "paidAt": "2026-06-01T12:02:30",
  "senderName": "SOURAV",
  "rrn": "606703736479",
  "upiId": "user@paytm"
}
```

`status` is one of: `pending`, `paid`, `cancelled`, `expired`.

**Errors:**
| Code | Status | Condition |
|------|--------|-----------|
| `TICKET_NOT_FOUND` | 404 | No ticket with the given ID |

---

### `POST /api/webhook`

Receive a bank SMS notification. Requires the `X-Webhook-Secret` header.

```
Rate limit: 30 requests/minute/IP
```

**Headers:**
```
X-Webhook-Secret: <shared-secret>
```

**Request:**
```json
{ "sms": "Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736480." }
```

**Response `200` (Bank SMS):**
```json
{
  "status": "ok",
  "ticketId": "TICKET17123456780000",
  "action": "marked_paid",
  "ticket": { "ticketId": "...", "amount": 100, "status": "paid", ... }
}
```

**Response `200` (Generic):**
```json
{
  "status": "ok",
  "ticketId": "TICKET17123456780000",
  "action": "name_filled",
  "ticket": { "ticketId": "...", "amount": 100, "status": "pending", ... }
}
```

**Errors:**
| Code | Status | Condition |
|------|--------|-----------|
| `WEBHOOK_UNAUTHORIZED` | 401 | Missing or invalid `X-Webhook-Secret` |
| `TICKET_NOT_FOUND` | 404 | Bank SMS: no pending ticket matches the amount. Generic: no ticket with the given ID |
| `AMOUNT_MISMATCH` | 400 | Bank SMS: multiple pending tickets match the same amount |
| `RRN_DUPLICATE` | 409 | The RRN has already been processed for a different ticket |
| `INVALID_AMOUNT` | 400 | Unrecognised SMS format |

---

### `GET /health`

```
Rate limit: none
```

**Response `200`:**
```json
{ "status": "healthy", "uptime": 3600, "db": "ok" }
```

---

## Rate Limits

| Endpoint | Limit | Scope |
|----------|-------|-------|
| `POST /api/ticket` | 5 requests / minute | Per IP |
| `GET /api/status/:id` | 60 requests / minute | Per IP |
| `POST /api/webhook` | 30 requests / minute | Per IP |
| `GET /health` | Unlimited | — |
| All others | 100 requests / minute | Per IP |

Limits are enforced by `@fastify/rate-limit` using an in-memory sliding window.

---

## Database Schema

### `tickets`

```sql
CREATE TABLE IF NOT EXISTS tickets (
  id          TEXT PRIMARY KEY,
  amount      INTEGER NOT NULL,
  status      TEXT NOT NULL DEFAULT 'pending'
              CHECK(status IN ('pending','paid','cancelled','expired')),
  base_amount INTEGER NOT NULL,
  decimal_val INTEGER NOT NULL,
  sender_name TEXT,
  rrn         TEXT UNIQUE,
  upi_id      TEXT,
  paid_at     TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
```

| Column | Type | Notes |
|--------|------|-------|
| `id` | TEXT | Format: `TICKET{timestamp}{counter}`, e.g. `TICKET17123456780000` |
| `amount` | INTEGER | Total amount in paisa (base + decimal), e.g. `10003` = ₹100.03 |
| `status` | TEXT | `pending` / `paid` / `cancelled` / `expired` |
| `base_amount` | INTEGER | Floor to nearest rupee in paisa, e.g. `10000` for ₹100.xx |
| `decimal_val` | INTEGER | The allocated decimal 0-99 |
| `rrn` | TEXT | UPI reference number, unique across all tickets |

### Indexes

```sql
idx_tickets_status   ON tickets(status)
idx_tickets_amount   ON tickets(amount)
idx_tickets_rrn      ON tickets(rrn)
idx_tickets_decimal  ON tickets(base_amount, decimal_val, status)
idx_tickets_created  ON tickets(created_at)
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP server port |
| `HOST` | `0.0.0.0` | Bind address |
| `TICKET_TTL_MINUTES` | `2` | Time before a pending ticket expires (in-memory timer) |
| `UPI_ID` | — | UPI ID shown on tickets (e.g. `college@upi`) |
| `UPI_PAYEE_NAME` | — | Payee name for ticket display |
| `WEBHOOK_SECRET` | random | Shared secret for `X-Webhook-Secret` header verification |
| `DATA_DIR` | `data` | Directory for the SQLite database file |
| `LOG_LEVEL` | `info` | Pino log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal` |

---

## Project Structure

```
src/
├── server/
│   ├── index.ts              # Entry point, startup, graceful shutdown
│   ├── config.ts             # Environment variable loader
│   ├── app.ts                # Fastify app assembly (routes, plugins)
│   ├── errors.ts             # AppError class + status code map
│   ├── money.ts              # Paisa/rupee conversion utilities
│   ├── db/
│   │   ├── connection.ts     # SQLite open/close with WAL pragmas
│   │   └── schema.ts         # CREATE TABLE statements
│   ├── middleware/
│   │   ├── error-handler.ts  # Unified error response format
│   │   └── request-logger.ts # Pino request logging
│   ├── routes/
│   │   ├── health.ts         # GET /health
│   │   ├── ticket.ts         # POST /api/ticket, GET /api/status/:id
│   │   └── webhook.ts        # POST /api/webhook
│   └── services/
│       ├── decimal.service.ts    # DDM pool: allocate, release, rebuild
│       ├── payment.service.ts    # SMS parsing, bank/generic dispatch
│       └── ticket.service.ts     # Ticket CRUD, timer management
├── types/
│   └── index.ts              # Ticket, TicketResponse, ParsedSms interfaces
└── test/                     # Vitest test suite
    ├── helpers.ts
    ├── decimal.test.ts
    ├── money.test.ts
    ├── payment.test.ts
    └── routes.test.ts
```

---

## Running

```bash
# Install
npm install

# Development (with file watching)
npm run dev

# TypeScript check
npm run typecheck

# Tests
npm test

# Production build
npm run build

# Start production
npm start
```

### Docker

```bash
# Build
docker build -t ddm-payment-gateway .

# Run
docker run -d \
  -p 3000:3000 \
  -v payment_data:/app/data \
  --env-file .env \
  ddm-payment-gateway

# With docker-compose
docker compose up -d
```

### Testing

```
npm test            # Run all tests
npm run test:watch  # Watch mode
```

The test suite covers:

- Decimal pool allocation, spillover, and recovery
- SMS parsing for both bank and generic formats
- Route integration (create ticket, status check, webhook)
- Webhook authentication rejection
- Duplicate RRN rejection
- Immediate reuse of paid decimals
