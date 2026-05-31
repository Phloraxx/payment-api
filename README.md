# Payment Gateway v2

Zero-fee UPI payment matching service for college events. SQLite-primary, passkey-secured admin dashboard, real-time WebSocket updates, and automated CI/CD to Dokploy.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Cloudflare Tunnel                   │
│                   pay.mulearnscet.in                 │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│              Fastify Server (Node.js 22)             │
│              Single process, no serverless           │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  Public  │  │  Admin   │  │  WebSocket (WS)   │  │
│  │  Routes  │  │  Routes  │  │  Real-time push   │  │
│  │  /api/   │  │  /api/   │  │  ticket updates   │  │
│  │  ticket  │  │  admin/  │  │  admin broadcasts │  │
│  │  status  │  │  pool    │  │  log streaming    │  │
│  │  webhook │  │  settings│  └───────────────────┘  │
│  └──────────┘  └──────────┘                          │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │              Services Layer                   │   │
│  │  DecimalPool │ TicketService │ PaymentService │   │
│  │  AuthService │ AppwriteSync  │ ExpiryService  │   │
│  │  Logger      │ WsManager     │                │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  ┌──────────────────┐  ┌──────────────────────────┐  │
│  │  payments.db     │  │  logs.db                 │  │
│  │  (SQLite WAL)    │  │  (SQLite WAL, ring buf)  │  │
│  │  tickets, auths, │  │  log entries, 10k cap    │  │
│  │  one_time_codes  │  └──────────────────────────┘  │
│  └──────────────────┘                                │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  Appwrite (optional, fire-and-forget sync)    │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  Admin SPA (React + Vite)                           │
│  8 pages: Overview, Tickets, Pool, Logs, Settings,  │
│  Test Harness, Decimal Pool, About                  │
│  Passkey auth (WebAuthn) via @simplewebauthn        │
│  Dark theme, frosted glass, Inter font              │
└─────────────────────────────────────────────────────┘
```

## Stack

| Layer | Technology |
|-------|-----------|
| Server | Node.js 22, Fastify 5, TypeScript (strict) |
| Database | SQLite 3 via better-sqlite3, WAL mode |
| Auth | WebAuthn passkeys via @simplewebauthn, signed HttpOnly cookies |
| Admin UI | React 19, Vite 6, lucide-react icons |
| Real-time | WebSocket via @fastify/websocket |
| QR Codes | qrcode canvas renderer (client-side) |
| Containers | Docker multi-stage, dumb-init, HEALTHCHECK |
| Deploy | GitHub Actions → Dokploy (auto-deploy) |

## Quick Start

```bash
# Install dependencies
npm install
npm --prefix src/admin install

# Copy env template
cp .env.example .env
# Edit .env with your values

# Run server (with auto-reload)
npm run dev

# Build admin SPA (for production)
npm run admin:build

# Run tests
npm test

# Type check
npm run typecheck
```

Visit `http://localhost:3000/health` to verify the server is running.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `3000` | No | HTTP server port |
| `HOST` | `0.0.0.0` | No | Bind address |
| `PUBLIC_BASE_URL` | `http://localhost:3000` | No | Public URL for WebAuthn |
| `RP_ID` | `localhost` | No | WebAuthn relying party (bare domain) |
| `TICKET_TTL_MINUTES` | `2` | No | Minutes before a pending ticket expires |
| `ONE_TIME_CODE` | auto-generated | No | First-run setup code for passkey registration |
| `COOKIE_SECRET` | random | No | Session cookie signing key |
| `WEBHOOK_SECRET` | random | No | Shared secret for SMS webhook |
| `UPI_ID` | — | No | UPI ID for QR generation (e.g. `user@okicici`) |
| `UPI_PAYEE_NAME` | — | No | Payee name shown on UPI QR |
| `APPWRITE_*` | — | No | Appwrite sync (optional, fire-and-forget) |

Set these in your production `.env` (gitignored) or as Docker environment variables.

## API Endpoints

### Public

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | — | Health check (uptime, DB, Appwrite reachable) |
| `POST` | `/api/ticket` | — | Create a payment ticket `{ amount: number }` |
| `GET` | `/api/status/:id` | — | Get ticket status |
| `GET` | `/api/ws?ticketId=X` | — | WebSocket — ticket status updates |
| `POST` | `/api/webhook` | `x-webhook-secret` | Incoming SMS webhook `{ sms: string }` |

### Admin (all require valid session cookie)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/setup/status` | Check if first-time setup is needed |
| `POST` | `/api/admin/setup/verify-code` | Verify one-time setup code `{ code }` |
| `GET` | `/api/admin/register/begin` | Begin WebAuthn registration |
| `POST` | `/api/admin/register/complete` | Complete WebAuthn registration |
| `GET` | `/api/admin/login/begin` | Begin WebAuthn login |
| `POST` | `/api/admin/login/complete` | Complete WebAuthn login |
| `POST` | `/api/admin/logout` | Clear session |
| `GET` | `/api/admin/session` | Check session validity |
| `GET` | `/api/admin/tickets` | List tickets (query: `status`, `q`, `limit`, `offset`) |
| `GET` | `/api/admin/tickets/:id` | Get single ticket |
| `PATCH` | `/api/admin/tickets/:id` | Update ticket (senderName, rrn, upiId) |
| `POST` | `/api/admin/tickets/:id/mark-paid` | Manually mark ticket as paid |
| `POST` | `/api/admin/tickets/:id/cancel` | Cancel a pending ticket |
| `GET` | `/api/admin/stats` | Aggregate ticket statistics |
| `GET` | `/api/admin/pool` | Decimal pool snapshot |
| `GET` | `/api/admin/logs` | Server logs (query: `level`, `q`, `limit`, `offset`) |
| `GET` | `/api/admin/settings` | Server configuration (UPI details, etc.) |
| `POST` | `/api/admin/sync/full` | Trigger full Appwrite resync |
| `GET` | `/api/admin/ws` | WebSocket — admin broadcast (auth required) |

### Test (admin auth required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/admin/test/ticket` | Create a test ticket `{ amount }` |
| `POST` | `/api/admin/test/webhook` | Simulate SMS webhook `{ sms }` |
| `GET` | `/api/admin/test/ws?ticketId=X` | WebSocket — test ticket updates |

## Dynamic Decimal Matching (DDM)

DDM prevents amount collisions when multiple tickets have the same rupee value.

**How it works:**

1. Each `baseAmount` (price in paisa, rounded to the nearest rupee) has a pool of 100 decimal slots (0–99).
2. When a ticket is created, it's allocated the next free decimal slot.
3. The final amount = `baseAmount + decimal` (e.g., ₹500.00, ₹500.01, ..., ₹500.99).
4. When a ticket is resolved (paid/cancelled/expired), the decimal enters a **5-minute delayed release** queue (from ticket creation time).
5. After 5 minutes, the decimal returns to the free pool.

**Why:** The SMS webhook matches by `TICKET ID + amount`. Without DDM, two simultaneous ₹500 tickets would be ambiguous. With DDM, ticket A is ₹500.00 and ticket B is ₹500.01 — the webhook matches the exact amount.

The Kotak format (no ticket ID) matches by amount only. If multiple pending tickets match, it's rejected — DDM prevents this collision.

## SMS Webhook

The webhook accepts bank SMS notifications and marks tickets as paid.

### SMS Formats

**Generic** (matches by ticket ID):
```
TICKET17802570916580000 paid ₹500.00 by John (UPI Ref 606703736479)
```

**Kotak** (matches by amount, no ticket ID):
```
Received Rs. 500.00 from user@oksbi (UPI Ref 606703736480)
```

### Matching Logic

1. Parse SMS to extract: `ticketId` (generic), `amount`, `senderName`, `rrn`, `upiId`
2. If **ticket ID found**: fetch that ticket, verify amount matches
3. If **no ticket ID (Kotak)**: find a pending ticket with matching amount (must be exactly 1 match)
4. Mark ticket as paid, store RRN (prevents duplicate processing), broadcast via WebSocket

### Authentication

Send the webhook secret in the `x-webhook-secret` header (constant-time compared):

```bash
curl -X POST https://pay.mulearnscet.in/api/webhook \
  -H "Content-Type: application/json" \
  -H "x-webhook-secret: your-secret-here" \
  -d '{"sms": "TICKET123 paid ₹500 by John (UPI Ref 606703736479)"}'
```

Rate limit: 30 requests/minute/IP.

## WebSocket Events

### Public (`/api/ws?ticketId=X`)

| Event | Payload | Trigger |
|-------|---------|---------|
| `expired` | `{ type, reason, ticketId }` | Ticket TTL reached |
| `payment_update` | `{ type, status, paidAt, senderName }` | Webhook or manual mark-paid |

### Admin (`/api/admin/ws`)

| Event | Payload | Trigger |
|-------|---------|---------|
| `ticket_update` | `{ type, action, ticket }` | Create, pay, cancel, expire |
| `log_entry` | `{ type, entry }` | Every server log entry |
| `shutdown` | `{ type, reason, reconnectMs }` | Graceful shutdown |

## Security

- **Auth**: WebAuthn (FIDO2 passkeys) — no passwords stored. Session via signed HttpOnly cookie.
- **One-time code**: Printed to stdout on first start (or set via env). Auto-expires after use.
- **Webhook**: Constant-time `safeEqual` comparison of shared secret from header.
- **Rate limiting**: Configurable per-route (global: 100/min, webhook: 30/min, ticket create: 5/min).
- **CSP**: Strict Content Security Policy — only `'self'` and Cloudflare insights allowed.
- **SQLite**: WAL mode with foreign keys. Prepared statements prevent injection.
- **Graceful shutdown**: Expire pending tickets, broadcast WS shutdown, checkpoint WAL.

## Admin Panel

8-page single-page application at `/admin/`:

| Page | Route | Description |
|------|-------|-------------|
| Overview | `/admin/` | Stats cards, pool gauge, recent logs |
| Tickets | `/admin/tickets` | Filterable table with detail drawer |
| Logs | `/admin/logs` | Filterable server log viewer |
| Pool | `/admin/pool` | Decimal pool visualization per base amount |
| Test Harness | `/admin/test` | Create tickets, simulate webhook, UPI QR scan |
| Settings | `/admin/settings` | Read-only server config display |
| Decimal Pool | — | Pool detail view |
| About | — | Version info |

First login: enter the one-time code from server startup → register passkey → done.

## UPI QR Codes

The Test Harness generates scannable UPI QR codes for test tickets:

```
upi://pay?pa=<UPI_ID>&pn=<PAYEE_NAME>&am=<AMOUNT>&cu=INR&tn=<TICKET_ID>
```

Configure via env vars (`UPI_ID`, `UPI_PAYEE_NAME`). QR rendered client-side with `qrcode` canvas library.

## Database Schema

### payments.db (primary)

```sql
-- Tickets
CREATE TABLE tickets (
  id          TEXT PRIMARY KEY,           -- e.g. TICKET1712345678000
  amount      INTEGER NOT NULL,           -- paisa (e.g. 50000 = ₹500.00)
  status      TEXT NOT NULL DEFAULT 'pending'
              CHECK(status IN ('pending','paid','cancelled','expired')),
  base_amount INTEGER NOT NULL,           -- amount rounded to rupee
  decimal_val INTEGER NOT NULL,           -- 0-99 decimal slot
  sender_name TEXT,
  rrn         TEXT UNIQUE,                -- UPI reference number
  upi_id      TEXT,
  paid_at     TEXT,
  expires_at  TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Passkey authenticators
CREATE TABLE authenticators (
  id          TEXT PRIMARY KEY,
  public_key  TEXT NOT NULL,
  counter     INTEGER NOT NULL DEFAULT 0,
  device_name TEXT,
  transports  TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- One-time setup codes
CREATE TABLE one_time_codes (
  code        TEXT PRIMARY KEY,
  used        INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### logs.db (ring buffer)

```sql
CREATE TABLE logs (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  level       TEXT NOT NULL CHECK(level IN ('info','warn','error','debug')),
  message     TEXT NOT NULL,
  meta        TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
```

Log cleanup runs every 500 inserts, keeping the latest 10,000 rows.

## Docker

```bash
# Build
docker build -t payment-gateway-v2 .

# Run with env file
docker run -d -p 3000:3000 -v payment_data:/app/data --env-file .env payment-gateway-v2

# Or with docker-compose
docker compose up -d
```

The Dockerfile uses multi-stage builds:
1. **admin-build**: Compiles the React SPA with Vite
2. **server-build**: Compiles TypeScript server with `tsc`
3. **production**: Minimal image with `dumb-init`, copies only `dist/` and `node_modules/`

## CI/CD

On every push to `main`:

1. **GitHub Actions** runs: `npm ci` → `npm run typecheck` → `npm test` (24 tests)
2. If tests pass, the **Deploy** job polls Dokploy's deployment status
3. **Dokploy** auto-deploys from the Dockerfile in the repo
4. The Deploy job reports success/failure based on Dokploy's build outcome

## Testing

```bash
# All tests
npm test

# Watch mode
npm run test:watch

# Coverage
npx vitest run --coverage
```

8 test files, 24 tests covering:
- Auth flow (WebAuthn setup, login, session)
- Decimal pool allocation and recovery
- Payment SMS parsing (generic + Kotak)
- Route integration (ticket create, status, webhook)
- Rate limiting
- Appwrite sync (disabled path)
- WebSocket broadcasting
- Money utilities (paisa conversion)

## Project Structure

```
src/
├── server/
│   ├── index.ts              # Entry point, startup sequence, graceful shutdown
│   ├── config.ts             # Typed env config with fallbacks
│   ├── app.ts                # Fastify app assembly, plugins
│   ├── container.ts          # Service dependency injection
│   ├── crypto.ts             # safeEqual constant-time comparison
│   ├── errors.ts             # AppError class, error codes
│   ├── money.ts              # Paisa conversion utilities
│   ├── db/
│   │   ├── connection.ts     # SQLite connections, WAL, seed codes
│   │   └── schema.ts         # CREATE TABLE definitions
│   ├── middleware/
│   │   ├── auth.ts           # Admin session verification
│   │   ├── error-handler.ts  # Unified error responses
│   │   └── request-logger.ts # HTTP request logging
│   ├── plugins/
│   │   └── static.ts         # Admin SPA static serving
│   ├── routes/
│   │   ├── admin.ts          # All admin CRUD + settings endpoints
│   │   ├── health.ts         # Health check
│   │   ├── ticket.ts         # Public ticket/status endpoints
│   │   ├── webhook.ts        # SMS webhook with header auth
│   │   ├── ws.ts             # WebSocket connection handlers
│   │   └── test.ts           # Admin test harness endpoints
│   ├── services/
│   │   ├── appwrite.service.ts    # Fire-and-forget Appwrite sync
│   │   ├── auth.service.ts        # WebAuthn passkey auth
│   │   ├── decimal.service.ts     # DDM pool with delayed release
│   │   ├── expiry.service.ts      # TTL enforcement (recursive setTimeout)
│   │   ├── logger.service.ts      # SQLite logging + pino stdout
│   │   ├── payment.service.ts     # SMS parsing + payment confirmation
│   │   ├── ticket.service.ts      # Ticket CRUD with cached statements
│   │   └── webhook-helper.ts      # Shared SMS processing
│   └── ws/
│       └── manager.ts        # WebSocket rooms, heartbeat, broadcast
├── admin/
│   └── src/
│       ├── api/              # API client functions
│       ├── components/       # Shared UI components
│       ├── pages/            # 8 admin pages
│       └── styles.css        # Apple-like dark theme
├── types/
│   └── index.ts              # Shared TypeScript interfaces
└── test/                     # 8 test files, 24 tests
```
