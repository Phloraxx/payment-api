# Payment Gateway v2 — Zero-Fee UPI Payment Gateway

A UPI payment gateway for college events that uses **Dynamic Decimal Matching (DDM)** to identify payments without needing a third-party payment gateway or API. Works purely by parsing bank SMS notifications.

## How It Works

1. **Ticket creation** — When a ticket is created, the system allocates a unique precise amount (e.g., ₹100.03 instead of ₹100) from a pool of 100 decimal variations (0–99 paise).
2. **Student pays** — The student scans a UPI QR code or transfers to the displayed UPI ID using the exact displayed amount.
3. **Bank SMS arrives** — The system receives the bank's SMS notification via a webhook endpoint.
   - **Kotak Mahindra SMS** contains the amount and sender's UPI ID — the system matches by amount and marks the ticket paid.
   - **Generic SMS** (other banks) contains the ticket ID and sender's name — the system fills in the sender name only.
4. **Unique amount guarantees** — The unique decimal ensures no two payments can be confused, even when multiple tickets have the same rupee value.

## Architecture

- **Single Node.js process** — No microservices, no message queues, no external dependencies.
- **Fastify 5** — High-performance HTTP server with built-in rate limiting and schema validation.
- **SQLite** via `better-sqlite3` — Zero-config database, WAL mode for concurrent reads.
- **No external APIs** — No Razorpay, no PhonePe, no third-party payment gateway.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/ticket` | — | Create a ticket `{ amount: number }` |
| `GET` | `/api/status/:id` | — | Get ticket status |
| `POST` | `/api/webhook` | `x-webhook-secret` | Incoming SMS notification `{ sms: string }` |
| `GET` | `/health` | — | Health check |

### POST /api/ticket

```json
// Request
{ "amount": 100 }
```

```json
// Response
{ "ticketId": "TICKET17123456780000", "amount": 100 }
```

### GET /api/status/:id

```json
// Response
{ "status": "pending", "ticketId": "TICKET17123456780000", "amount": 100, "createdAt": "..." }
```

### POST /api/webhook

The webhook accepts SMS notifications from banks. Two formats are supported:

**Kotak Mahindra (marks ticket paid):**
```
Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736480.
```

**Generic (fills sender name, does NOT mark paid):**
```
TICKET17123456780000 SOURAV paid you ₹100.00 UPI Ref:606703736479
```

Requires the `x-webhook-secret` header (constant-time compared).

```json
// Request
{ "sms": "Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736480." }

// Response (Kotak)
{ "ticketId": "TICKET...", "status": "ok", "action": "marked_paid" }

// Response (Generic)
{ "ticketId": "TICKET...", "status": "ok", "action": "name_filled" }
```

### GET /health

```json
// Response
{ "status": "ok", "uptime": 1234 }
```

## Setup

Copy `.env.example` to `.env` and configure:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP server port |
| `HOST` | `0.0.0.0` | Bind address |
| `TICKET_TTL_MINUTES` | `2` | Minutes before a pending ticket auto-expires |
| `UPI_ID` | — | UPI ID for payment references (e.g., `college@upi`) |
| `UPI_PAYEE_NAME` | — | Payee name for ticket display |
| `WEBHOOK_SECRET` | random | Shared secret for SMS webhook authentication |

## Rate Limits

| Endpoint | Rate Limit |
|----------|------------|
| `POST /api/ticket` | 5 requests per minute per IP |
| `POST /api/webhook` | 30 requests per minute per IP |
| `GET /api/status/:id` | 60 requests per minute per IP |
| All others | 100 requests per minute per IP |

## Running

```bash
# Development (with auto-reload)
npm run dev

# Build for production
npm run build

# Start production
npm start

# Run tests
npm test

# Type check
npm run typecheck
```

### Docker

```bash
# Build
docker build -t payment-gateway-v2 .

# Run
docker run -d -p 3000:3000 -v payment_data:/app/data --env-file .env payment-gateway-v2

# With docker-compose
docker compose up -d
```

## Dynamic Decimal Matching (DDM) Algorithm

DDM prevents amount collisions when multiple tickets share the same rupee value.

### Pool Structure

Each base amount (price in paisa rounded down to the nearest rupee) has a pool of 100 decimal slots (0–99). For example, for ₹100 tickets:

- Base amount: `10000` paisa (100 × 100)
- Decimal slots: `0` through `99`
- Available amounts: `10000` (₹100.00), `10001` (₹100.01), … `10099` (₹100.99)

### Allocation

1. When `createTicket(100)` is called, the pool for base amount `10000` is checked.
2. The first free decimal slot is allocated (0, then 1, then 2, …).
3. If all 100 slots at the base amount are taken, allocation **spill over**s to the next integer block: `10100` (₹101.00), `10101`, etc.
4. The allocated slot is marked as **used** in the in-memory set so it won't be re-allocated.

### Release (Grace Period)

When a ticket is expired or cancelled, its decimal slot enters a **30-second grace period** before being returned to the free pool. This prevents rapid recycling of decimals that could confuse manual reconciliation.

- **Paid** tickets release their decimal immediately (the payment is confirmed).
- **Expired** tickets release after 30 seconds.
- **Cancelled** tickets release after 30 seconds.

### Recovery on Restart

On server startup, `rebuild()` scans all `pending` and `paid` tickets from the database and re-populates the in-memory pool. This ensures that the pool state is correctly recovered after a restart.

### Why Not a Random Decimal?

A deterministic sequential allocation is used rather than random because:

1. **Predictability** — The next free slot is always the smallest available, which minimises fragmentation.
2. **Debugging** — The allocation order is reproducible and easy to reason about.
3. **No collisions** — Sequential allocation is guaranteed to avoid collisions as long as freed slots are properly managed.

## Project Structure

```
src/
├── server/
│   ├── index.ts              # Entry point, startup, graceful shutdown
│   ├── config.ts             # Typed env configuration
│   ├── app.ts                # Fastify app assembly
│   ├── errors.ts             # AppError class and error codes
│   ├── money.ts              # Paisa conversion utilities
│   ├── db/
│   │   ├── connection.ts     # SQLite connection management
│   │   └── schema.ts         # Table definitions
│   ├── middleware/
│   │   ├── error-handler.ts  # Unified error responses
│   │   └── request-logger.ts # Request logging
│   ├── routes/
│   │   ├── health.ts         # Health check endpoint
│   │   ├── ticket.ts         # Ticket create + status endpoints
│   │   └── webhook.ts        # SMS webhook endpoint
│   └── services/
│       ├── decimal.service.ts    # DDM pool allocation and release
│       ├── payment.service.ts    # SMS parsing and payment confirmation
│       └── ticket.service.ts     # Ticket CRUD with prepared statements
├── types/
│   └── index.ts              # Shared TypeScript interfaces
└── test/                     # Vitest test suite
```

## Testing

```bash
# All tests
npm test

# Watch mode
npm run test:watch

# Coverage
npx vitest run --coverage
```

Tests cover:
- Decimal pool allocation, spillover, and recovery
- Payment SMS parsing (generic + Kotak formats)
- Route integration (ticket create, status check, webhook)
- Rejection of unauthorised webhook calls
- Duplicate RRN rejection
- Immediate reuse of paid decimals
