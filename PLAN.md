# Payment Gateway v2 — Architecture Plan

A zero-fee UPI payment gateway for college events using **Dynamic Decimal Matching (DDM)**. Single-server, SQLite-primary, Appwrite-secondary.

---

## 1. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Docker Container                              │
│                                                                      │
│   Fastify Server (Node.js 22)                                        │
│   ┌─────────────────────────────────────────────────────────────┐    │
│   │  Routes                                                      │    │
│   │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐ │    │
│   │  │  Ticket  │ │ Webhook  │ │  Admin   │ │  WS Upgrade    │ │    │
│   │  │  Routes  │ │  Route   │ │  Routes  │ │  + Test Routes │ │    │
│   │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └───────┬────────┘ │    │
│   │       │            │            │                │           │    │
│   │  ┌────┴────────────┴────────────┴────────────────┴────────┐ │    │
│   │  │                    Services Layer                       │ │    │
│   │  │  TicketSvc   DecimalPoolSvc   PaymentSvc   ExpirySvc  │ │    │
│   │  │  SmsParser    AppwriteSync    Logger                    │ │    │
│   │  └──────────────────────────┬──────────────────────────────┘ │    │
│   │                             │                                 │    │
│   │  ┌──────────────────────────┴──────────────────────────────┐ │    │
│   │  │                     Data Layer                           │ │    │
│   │  │  better-sqlite3 (WAL mode, persistent Docker volume)    │ │    │
│   │  └─────────────────────────────────────────────────────────┘ │    │
│   └───────────────────────────────────────────────────────────────┘    │
│                                                                        │
│   Volumes:  ./data → /app/data                                         │
│             ├── payments.db                                            │
│             └── logs.db                                                │
│                                                                        │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │ fire-and-forget write
                      ┌────────┴────────┐
                      │    Appwrite      │  ← External apps read from here
                      │  (secondary DB)  │
                      └─────────────────┘
```

## 2. Technology Stack

| Layer | Choice | Rationale |
|---|---|---|
| Runtime | **Node.js 22** (Alpine Docker) | Single process, no cold starts |
| Web framework | **Fastify 5** + TypeScript | Fastest Node.js framework, plugin ecosystem |
| Primary DB | **better-sqlite3** (WAL mode) | Synchronous API, zero network overhead, crash-safe |
| Secondary DB | **Appwrite** (fire-and-forget sync) | External apps read from here; payment path never depends on it |
| WebSocket | **@fastify/websocket** | In-process pub/sub per ticket room |
| Validation | **TypeBox** | Schema → JSON Schema → validation + TypeScript types |
| Logging | **Pino** (Fastify default) | Structured JSON, fast, SQLite + WS stream |
| Admin auth | **Passkey (WebAuthn)** via `@simplewebauthn/server` + session cookie via `@fastify/cookie` | Biometric/PIN login. No password to leak, no brute-force. Cookie set after successful WebAuthn assertion. |
| Admin UI | **React + Vite + TypeScript** | Build step → static files served by Fastify |
| Container | **Docker** + Portainer | Portainer webhook for CI/CD |
| Testing | **Vitest** | Fast, native ESM |
| Health | **dumb-init** | Proper signal handling in Docker |

## 3. Database Schema

Two separate SQLite databases — no write contention between critical path and logging.

### `data/payments.db` — Core Payment Data

All monetary values stored as INTEGER paisa. ₹100.03 → `10003`. This avoids floating-point precision issues and enables exact comparisons.

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE tickets (
    id          TEXT PRIMARY KEY,           -- "TICKET" + UnixMs timestamp
    amount      INTEGER NOT NULL,           -- paisa: 10003 for ₹100.03
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK(status IN ('pending','paid','cancelled','expired')),
    base_amount INTEGER NOT NULL,           -- integer rupees in paisa: 10000 for ₹100
    decimal_val INTEGER NOT NULL,           -- 0-99 decimal for pool tracking
    sender_name TEXT,
    rrn         TEXT UNIQUE,                -- UPI reference number (dedup)
    upi_id      TEXT,
    paid_at     TEXT,                       -- ISO 8601
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_tickets_status   ON tickets(status);
CREATE INDEX idx_tickets_amount   ON tickets(amount);
CREATE INDEX idx_tickets_rrn      ON tickets(rrn);
CREATE INDEX idx_tickets_decimal  ON tickets(base_amount, decimal_val, status);

CREATE TABLE authenticators (
    id          TEXT PRIMARY KEY,           -- Base64URL credential ID
    public_key  TEXT NOT NULL,              -- Base64URL public key (COSEPublicKey)
    counter     INTEGER NOT NULL DEFAULT 0, -- Signature counter
    device_name TEXT,                       -- e.g. "Chrome on Windows"
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Single-use one-time code for first-time registration
CREATE TABLE one_time_codes (
    code        TEXT PRIMARY KEY,
    used        INTEGER NOT NULL DEFAULT 0, -- 0 = unused, 1 = used
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### `data/logs.db` — Separate Database for Logs

```sql
PRAGMA journal_mode = WAL;

CREATE TABLE logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    level       TEXT NOT NULL CHECK(level IN ('info','warn','error','debug')),
    message     TEXT NOT NULL,
    meta        TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_logs_level   ON logs(level);
CREATE INDEX idx_logs_created ON logs(created_at);

-- Ring buffer via periodic cleanup (not a trigger): every 500 inserts, keep max 10k rows
```

## 4. Decimal Allocation Engine (DDM Core)

All amounts stored as integer paisa internally. Conversion helpers:
- `toPaisa(100.03)` → `10003`
- `fromPaisa(10003)` → `100.03`

### In-Memory Data Structure

```typescript
class DecimalPool {
  // One free-list per base_amount (in paisa)
  // Key: 10000 for ₹100
  // Value: queue [10000, 10001, ..., 10099]
  private pools: Map<number, number[]>;

  // Currently allocated amounts
  private allocated: Map<string, TicketInfo>;
}
```

### State Machine

```
          ┌──────────┐
          │  FREE    │  ← Available in pool
          └────┬─────┘
               │ pop on create
          ┌────▼─────┐
          │ PENDING  │  ← TTL timer running (2 min)
          └────┬─────┘
               │
      ┌────────┼────────┐
      │        │        │
  paid ▼   expired ▼  cancelled ▼
┌──────┐  ┌────────┐ ┌──────────┐
│ PAID │  │EXPIRED │ │CANCELLED │
└──┬───┘  └───┬────┘ └────┬─────┘
   │          │           │
   └──────────┼───────────┘
              │ push back to pool
         ┌────▼─────┐
         │  FREE    │
         └──────────┘
```

### Allocation Rules

| Condition | Behavior |
|---|---|
| Free queue non-empty | Pop from front (e.g., `10003`) |
| Free queue empty | Increment integer by 100 paisa, add new block of 100 |
| Ticket paid/expired/cancelled | Push amount to back of free queue |
| Server restart | Mark all pending as expired, rebuild free queues |
| Pool full (all 100 in use) | New integer block created as safety valve |

### Lifecycle Example

```
Base: ₹100 → 10000 paisa
Pool: [10000, 10001, ..., 10099]

t=0:  Create → pop 10000 → pool: [10001..10099]
t=1:  Create → pop 10001 → pool: [10002..10099]
...
t=99: Create → pop 10099 → pool: []
t=100: Pool empty → new block: 10100..10199 → pop 10100
t=101: Pay for 10000 → push back → pool: [10101..10199, 10000]
t=102: Create → pop 10101 → pool: [10102..10199, 10000]

API returns: fromPaisa(10101) → 101.01
SMS lookup:  "₹101.01" → toPaisa(101.01) → 10101 → WHERE amount = 10101
```

## 5. API Endpoints

> **Amount format**: API uses decimal (e.g. `100.03`). Internally stored as integer paisa.
>
> **Admin auth**: Signed HttpOnly cookie via `@fastify/cookie`. Login sets `token=s:<signed>` with `HttpOnly; SameSite=Strict; Path=/api/admin`. Cookie is auto-sent on all admin requests and WebSocket upgrades.

### Public

| Method | Path | Auth | Request | Response |
|---|---|---|---|---|
| `POST` | `/api/ticket` | — | `{ amount: 100 }` | `{ ticketId, amount, expiresAt, createdAt }` |
| `GET` | `/api/status/:id` | — | — | `{ ticketId, amount, status, paidAt, senderName, rrn }` |
| `WS` | `/api/ws?ticketId=X` | — | — | `payment_update` / `expired` / `shutdown` events |

### Webhook (SMS)

| Method | Path | Auth | Request | Response |
|---|---|---|---|---|
| `POST` | `/api/webhook` | `X-Webhook-Secret` header | `{ sms: "..." }` | `{ status, ticketId, action }` |

**Generic SMS:**
```
TICKET1709123456789 SOURAV paid you ₹100.03
```

**Kotak SMS:**
```
Confirmed payment for Received Rs.100.03 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736479.
```

**Matching:**
1. Try generic: extract `TICKET(\d+)` + amount → lookup by ticketId
2. Try Kotak: extract `Rs.X.YY` → `SELECT * FROM tickets WHERE amount = ? AND status = 'pending' LIMIT 1`
3. Verify RRN not duplicate (UNIQUE constraint)
4. Mark paid, push WS event, free decimal, fire-and-forget Appwrite sync

### Admin (passkey + cookie-based auth)

First-time setup:
1. Admin visits `/admin` — no authenticators registered → redirects to `/admin/setup`
2. `GET /api/admin/setup/status` returns `{ needs_setup: true, has_one_time_code: true }`
3. Admin enters `ONE_TIME_CODE` → `POST /api/admin/setup/verify-code { code }` validates it
4. `GET /api/admin/register/begin` returns WebAuthn registration options (challenge, rp, user)
5. Browser calls `navigator.credentials.create(options)` → biometric/PIN prompt
6. `POST /api/admin/register/complete { credential }` verifies and stores public key → sets session cookie
7. One-time code is marked used

Subsequent logins:
1. `GET /api/admin/login/begin` returns WebAuthn authentication options (challenge + allowCredentials)
2. Browser calls `navigator.credentials.get(options)` → biometric/PIN prompt
3. `POST /api/admin/login/complete { assertion }` verifies signature → sets signed HttpOnly session cookie

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/admin/setup/status` | Check if setup is needed |
| `POST` | `/api/admin/setup/verify-code` | `{ code }` → validates one-time code |
| `GET` | `/api/admin/register/begin` | WebAuthn registration options (after code verified) |
| `POST` | `/api/admin/register/complete` | `{ credential }` → stores public key |
| `GET` | `/api/admin/login/begin` | WebAuthn assertion options |
| `POST` | `/api/admin/login/complete` | `{ assertion }` → verifies → sets cookie |
| `POST` | `/api/admin/logout` | Clears session cookie |
| `GET` | `/api/admin/session` | Check if cookie is valid (used by SPA on load) |
| `GET` | `/api/admin/tickets` | List + filter + paginate + export |
| `GET` | `/api/admin/tickets/:id` | Ticket detail |
| `PATCH` | `/api/admin/tickets/:id` | Update fields |
| `POST` | `/api/admin/tickets/:id/mark-paid` | Mark paid shortcut |
| `POST` | `/api/admin/tickets/:id/cancel` | Cancel shortcut |
| `GET` | `/api/admin/stats` | Aggregate stats |
| `GET` | `/api/admin/logs` | Query logs with filters |
| `GET` | `/api/admin/pool` | Current decimal pool state |
| `POST` | `/api/admin/sync/full` | Re-sync all tickets to Appwrite |
| `WS` | `/api/admin/ws` | Real-time events + log stream (cookie sent automatically) |

### Test (within admin, uses real API)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/admin/test/ticket` | Create test ticket (bypasses rate limits) |
| `POST` | `/api/admin/test/webhook` | Simulate SMS webhook |
| `WS` | `/api/admin/test/ws?ticketId=X` | Test WebSocket |

## 6. Rate Limiting

| Scope | Limit | Justification |
|---|---|---|
| **Ticket creation** | 5/min/IP | Prevents decimal pool exhaustion by bad actors. 5/min is enough for human-paced registration on campus NAT. |
| **Webhook** | 30/min/IP | Single phone sending SMS — generous buffer |
| **Status polling** | 60/min/IP | Prevents polling abuse |
| **Admin endpoints** | 30/min/IP (per session) | Admin dashboard |
| **Admin login / register** | 10/min/IP | Passkey auth — rate limit on WebAuthn challenge endpoints |
| **WebSocket connects** | 20/min/IP | Connection flood prevention |
| **Health check** | No limit | Required for Docker health checks |

All limits use `@fastify/rate-limit` — in-memory sliding window per IP.

## 7. WebSocket Architecture

```typescript
// Single process, all in-memory
class WsManager {
  ticketRooms: Map<string, Set<WebSocket>>;   // ticketId → connections
  adminSockets: Map<string, Set<WebSocket>>;   // admin sessions
  heartbeatTimers: Map<WebSocket, Timer>;
}
```

| Event | Source | Destination | Payload |
|---|---|---|---|
| `payment_update` | Webhook | Ticket room | `{ type, status, paidAt, senderName }` |
| `expired` | Expiry timer | Ticket room | `{ type, reason: "timeout" }` |
| `shutdown` | Graceful shutdown | All rooms | `{ type, reason: "restart", reconnectMs }` |
| `ticket_update` | Status change | Admin sockets | `{ type, action, ticket }` |
| `log_entry` | Logger | Admin sockets | `{ type, level, message, meta }` |

Heartbeat: Server pings every 30s. Closes connection after 30s no response.

## 8. Appwrite Sync (Fire-and-Forget)

SQLite is the source of truth. Appwrite is a convenience replica for external apps.

```
SQLite write (ticket create / update)
  ↓
Immediately fire Appwrite REST call (no await, no block)
  ↓
  Success → done
  Failure → log the error. Alerts visible in admin logs.
```

No queue, no worker, no polling. On crash, SQLite has all the data. Admin uses `POST /api/admin/sync/full` to re-sync if Appwrite was down.

## 9. Server Lifecycle

### Startup
1. Open SQLite, run `PRAGMA integrity_check`
2. Run migrations (CREATE TABLE IF NOT EXISTS)
3. Crash recovery: if any `pending` tickets exist → expire them
4. Build DecimalPool from scratch (all decimals free)
5. Seed `one_time_codes` from env if not already seeded (auto-generate if not set)
6. Register routes, WS handlers
7. Listen on PORT 3000
8. Health check returns 200 → Portainer marks healthy

### Graceful Shutdown
```
SIGTERM received
1. Fastify.close() → stop accepting new connections (503 for in-flight)
2. UPDATE tickets SET status='expired' WHERE status='pending'
3. Clear all expiry timers
4. Broadcast WS: { type: "shutdown", reason: "restart", reconnectMs: 3000 }
5. Close all WebSocket connections
6. SQLite WAL checkpoint
7. Exit (process.exit(0))
```

### Crash Recovery
On next startup: expire any stale pending tickets, rebuild pool from scratch.

### Health Check
```
GET /health → 200
{
  "status": "healthy",
  "uptime": 3600,
  "db": "ok",
  "appwrite_reachable": true,
  "pool": { "base_amount": 100, "pending": 23, "free": 77 }
}
```

## 10. Error Handling

```json
{
  "error": {
    "code": "POOL_EXHAUSTED",
    "message": "All payment slots are currently occupied. Please try again.",
    "details": { "pending_count": 100, "ttl_seconds": 120 }
  }
}
```

| Code | HTTP | Trigger |
|---|---|---|
| `INVALID_AMOUNT` | 400 | Missing, non-numeric, or ≤ 0 amount |
| `TICKET_NOT_FOUND` | 404 | ticketId doesn't exist |
| `POOL_EXHAUSTED` | 503 | All decimal slots in use |
| `RRN_DUPLICATE` | 409 | Payment with same RRN already processed |
| `AMOUNT_MISMATCH` | 400 | Webhook amount doesn't match ticket |
| `TICKET_ALREADY_RESOLVED` | 409 | Ticket already paid/cancelled/expired |
| `WEBHOOK_UNAUTHORIZED` | 401 | Bad webhook secret |
| `ADMIN_UNAUTHORIZED` | 401 | Invalid/missing/expired session cookie |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

## 11. Logging Architecture

### What Gets Logged

```
Every HTTP request:
  { level, time, req_id, method, path, status, duration_ms, ip, error }

Business events:
  - Ticket created:       { level, message, meta: { ticketId, amount, decimal, pool_free } }
  - Payment confirmed:    { level, message, meta: { ticketId, amount, sender, rrn, match_method } }
  - Decimal pool full:    { level, message, meta: { base_amount, pending_count, new_integer } }
  - Decimal freed:        { level, message, meta: { amount, reason } }
  - Ticket expired:       { level, message, meta: { ticketId, amount } }

Errors:
  - Appwrite sync failure:{ level, message, meta: { ticket_id, error } }
  - Webhook auth failure: { level, message, meta: { ip, reason } }
  - Invalid input:        { level, message, meta: { ip, validation_error } }
```

### Storage + Routing

```
Pino logger
  ├── Console (stdout, structured JSON) → Docker log collector
  ├── data/logs.db (separate DB, ring-buffer via periodic cleanup: every 500 inserts, keep max 10k rows) → Admin API
  └── WebSocket broadcast → Admin dashboard real-time stream
```

## 12. Admin Dashboard (React + Vite SPA)

Apple-like aesthetic with Tailwind CSS + Framer Motion. Inter font, dark mode default, frosted glass panels, spring animations.

### Pages

| Page | Route | Features |
|---|---|---|
| **Login** | `/` | Passkey WebAuthn authentication. Auto-redirect if cookie valid. |
| **Setup** | `/setup` | First-time passkey registration. One-time code entry → biometric/PIN prompt. |
| **Overview** | `/overview` | Stats cards (total tickets, pending, paid today, revenue). Decimal pool gauge (ring chart). Recent activity feed. |
| **Tickets** | `/tickets` | Searchable, filterable, sortable table with pagination. Row click → drawer with detail view. Export CSV/JSON. |
| **Decimal Pool** | `/pool` | Gauge visualization (count of free/pending/paid decimals). Table of all active decimals with status badges. |
| **Test Harness** | `/test` | Create ticket, simulate webhook, test WebSocket — all against real API endpoints. |
| **Logs** | `/logs` | Log table with level badges. Search + filter. Real-time stream via WebSocket. |
| **Settings** | `/settings` | Environment config display. Appwrite sync status + "Re-sync All" button. |

### Design System

```
Colors:
  bg-primary:    #0a0a0f  (deep black-blue)
  bg-secondary:  #12121a  (card backgrounds)
  bg-tertiary:   #1a1a2e  (hover states)
  accent:        #6366f1  (indigo-500 - primary action)
  accent-glow:   #818cf8  (indigo-400 - hover glow)
  text:          #f1f5f9  (slate-50)
  text-secondary:#94a3b8  (slate-400)
  border:        #1e293b  (slate-800)
  success:       #22c55e  (green-500)
  warning:       #f59e0b  (amber-500)
  error:         #ef4444  (red-500)

Typography:
  font-family: 'Inter', system-ui, sans-serif
  scale: 12 / 14 / 16 / 18 / 20 / 24 / 30

Components (Tailwind + Framer Motion):
  GlassCard:    bg-secondary/80 backdrop-blur-xl border border-white/5
  Button:       px-4 py-2 rounded-xl bg-accent hover:bg-accent-glow transition-all
                spring animation on tap (scale: 0.97)
  Input:        bg-tertiary/50 border-border rounded-xl focus:ring-accent
  Badge:        px-2 py-0.5 rounded-full text-xs font-medium
  Table:        glass card with sticky header, row hover bg-tertiary
  Sidebar:      fixed left, w-64, glass effect, animated icons (spring)
  Drawer:       slide-in from right, backdrop blur, spring animation
  Skeleton:     shimmer animation for loading states
```

### Build

```
# Dev
cd admin && vite dev              → localhost:5173 (proxied to :3000)

# Production
cd admin && vite build            → output: server/admin/public/

# Fastify serves it
app.register(fastifyStatic, { root: join(__dirname, 'admin/public') });
```

### WebAuthn Flow (Setup)

```
1. GET  /api/admin/setup/status          → { needs_setup: true }
2. POST /api/admin/setup/verify-code     → { code: "XXXX-XXXX" }
3. GET  /api/admin/register/begin        → { publicKey: CreationOptions }
4. Browser: navigator.credentials.create(publicKey)
5. POST /api/admin/register/complete     → { credential: ... } → sets cookie
```

### WebAuthn Flow (Login)

```
1. GET  /api/admin/login/begin           → { publicKey: RequestOptions }
2. Browser: navigator.credentials.get(publicKey)
3. POST /api/admin/login/complete        → { assertion: ... } → sets cookie
```

## 13. Project Structure

```
payment-gateway/
├── src/
│   ├── server/                       # Fastify backend
│   │   ├── index.ts                  # Entry point + graceful shutdown
│   │   ├── app.ts                    # Fastify bootstrap
│   │   ├── config.ts                 # Typed env config
│   │   ├── db/
│   │   │   ├── connection.ts         # Two DB connections: payments + logs
│   │   │   └── schema.ts             # CREATE TABLE statements
│   │   ├── routes/
│   │   │   ├── ticket.ts             # POST /api/ticket, GET /api/status/:id
│   │   │   ├── webhook.ts            # POST /api/webhook
│   │   │   ├── admin.ts              # All /api/admin/* routes
│   │   │   ├── health.ts             # GET /health
│   │   │   ├── ws.ts                 # WS upgrade handlers
│   │   │   └── test.ts               # /api/admin/test/* routes
│   │   ├── services/
│   │   │   ├── ticket.service.ts     # CRUD, expiry timers
│   │   │   ├── decimal.service.ts    # DecimalPool (in-memory + SQLite)
│   │   │   ├── payment.service.ts    # SMS parsing + matching
│   │   │   ├── appwrite.service.ts   # Fire-and-forget + full re-sync
│   │   │   ├── expiry.service.ts     # TTL scheduling + cleanup
│   │   │   ├── logger.service.ts     # Pino → logs.db + WS
│   │   │   └── auth.service.ts       # WebAuthn registration, verification, passkey management
│   │   ├── ws/
│   │   │   ├── manager.ts            # Connection pools, heartbeat
│   │   │   └── handlers.ts           # Ticket + Admin WS handlers
│   │   ├── middleware/
│   │   │   ├── auth.ts               # Verify signed HttpOnly cookie
│   │   │   ├── request-logger.ts     # Log all HTTP requests
│   │   │   └── error-handler.ts      # Global error handler
│   │   ├── plugins/
│   │   │   └── static.ts             # Serve admin SPA
│   │   └── admin/
│   │       └── public/               # Built React SPA output
│   ├── admin/                        # React + Vite SPA source
│   │   ├── index.html
│   │   ├── vite.config.ts
│   │   ├── package.json
│   │   └── src/
│   │       ├── main.tsx
│   │       ├── App.tsx
│   │       ├── api/
│   │       │   ├── client.ts
│   │       │   ├── tickets.ts
│   │       │   ├── logs.ts
│   │       │   └── auth.ts
│   │       ├── pages/
│   │       │   ├── Login.tsx
│   │       │   ├── Setup.tsx
│   │       │   ├── Overview.tsx
│   │       │   ├── Tickets.tsx
│   │       │   ├── DecimalPool.tsx
│   │       │   ├── Logs.tsx
│   │       │   ├── TestHarness.tsx
│   │       │   └── Settings.tsx
│   │       ├── components/
│   │       │   ├── GlassCard.tsx
│   │       │   ├── Sidebar.tsx
│   │       │   ├── StatsCard.tsx
│   │       │   ├── PoolGauge.tsx
│   │       │   ├── StatusBadge.tsx
│   │       │   └── DataTable.tsx
│   │       └── hooks/
│   │           ├── useWebSocket.ts
│   │           └── useWebAuthn.ts
│   └── types/
│       └── index.ts
├── data/                             # Docker volume (gitignored)
│   ├── payments.db
│   └── logs.db
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .github/workflows/deploy.yml
├── package.json
├── tsconfig.json
└── README.md
```

## 14. Docker + CI/CD

### Dockerfile

```dockerfile
# Stage 1: Build admin SPA
FROM node:22-alpine AS admin-build
WORKDIR /app
COPY src/admin/package*.json ./
RUN npm ci
COPY src/admin/ ./
RUN npm run build

# Stage 2: Build server
FROM node:22-alpine AS server-build
WORKDIR /app
COPY package*.json tsconfig*.json ./
RUN npm ci
COPY src/server/ ./src/server/
COPY src/types/ ./src/types/
RUN npm run build

# Stage 3: Production
FROM node:22-alpine
RUN apk add --no-cache dumb-init
WORKDIR /app
COPY --from=admin-build /app/dist/ ./admin/public/
COPY --from=server-build /app/dist/ ./dist/
COPY --from=server-build /app/node_modules/ ./node_modules/
COPY package*.json ./
EXPOSE 3000
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD node -e "fetch('http://localhost:3000/health').then(r=>process.exit(r.ok?0:1))"
CMD ["dumb-init", "node", "dist/server/index.js"]
```

### docker-compose.yml

```yaml
services:
  app:
    build: .
    ports: ["3000:3000"]
    volumes:
      - payment_data:/app/data
    environment:
      - PORT=3000
      - TICKET_TTL_MINUTES=2
      - ONE_TIME_CODE=${ONE_TIME_CODE}
      - COOKIE_SECRET=${COOKIE_SECRET}
      - WEBHOOK_SECRET=${WEBHOOK_SECRET}
      - APPWRITE_ENDPOINT=${APPWRITE_ENDPOINT}
      - APPWRITE_API_KEY=${APPWRITE_API_KEY}
      - APPWRITE_PROJECT_ID=${APPWRITE_PROJECT_ID}
      - APPWRITE_DATABASE_ID=${APPWRITE_DATABASE_ID}
      - APPWRITE_COLLECTION_ID=${APPWRITE_COLLECTION_ID}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "node", "-e", "fetch('http://localhost:3000/health').then(r=>process.exit(r.ok?0:1))"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  payment_data:
```

### GitHub Actions

```yaml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/${{ github.repository }}:latest
      - name: Trigger Portainer deploy
        run: curl -X POST "${{ secrets.PORTAINER_WEBHOOK_URL }}"
```

## 15. Security Checklist

| Concern | Mitigation |
|---|---|
| **SQL injection** | Parameterized queries (better-sqlite3) |
| **Timing attacks** | `crypto.timingSafeEqual` for all secret comparisons |
| **XSS** | React's default escaping + CSP headers |
| **Session theft** | HttpOnly + signed + SameSite=Strict cookie. Not accessible to JS. |
| **Webhook auth** | Shared secret via `X-Webhook-Secret` header only, constant-time comparison |
| **Passkey (WebAuthn)** | No password to leak. Biometric/PIN bound to device. Phishing-resistant. Public key stored in `authenticators` table. |
| **Decimal exhaustion** | 5/min/IP rate limit + 100-slot pool + 2min TTL |
| **Server crash** | WAL mode, expiry on restart, SQLite volume persists |
| **Dependencies** | Minimal surface: Fastify + plugins + better-sqlite3 + React |

## 16. Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | No | 3000 | HTTP server port |
| `HOST` | No | 0.0.0.0 | Bind address |
| `TICKET_TTL_MINUTES` | No | 2 | How long a pending ticket lives |
| `ONE_TIME_CODE` | Yes* | Auto-generated | One-time setup code for passkey registration (printed to logs on first start) |
| `COOKIE_SECRET` | Yes | — | Secret for signing admin session cookie |
| `WEBHOOK_SECRET` | Yes | — | Shared secret for SMS webhook (`X-Webhook-Secret` header) |
| `APPWRITE_ENDPOINT` | No | — | Appwrite server URL (if sync enabled) |
| `APPWRITE_PROJECT_ID` | No* | — | Appwrite project ID |
| `APPWRITE_API_KEY` | No* | — | Appwrite API key |
| `APPWRITE_DATABASE_ID` | No* | — | Appwrite database ID |
| `APPWRITE_COLLECTION_ID` | No* | — | Appwrite collection ID |

*Required only if Appwrite sync is configured.

---

_Plan generated from codebase analysis and design discussions — May 2026_
