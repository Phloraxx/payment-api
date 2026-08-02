# PayGate

Self-hosted UPI payment verification for applications that receive money directly into a configured UPI/bank account.

PayGate creates a unique payable amount using a paise fingerprint, observes bank-credit SMS evidence, matches the **exact** amount and UPI reference, persists the payment lifecycle in PocketBase/SQLite, and exposes status through an HTTP API, realtime operator UI and optional signed outgoing webhooks.

PayGate does **not** custody, route or settle funds. The payer pays the configured UPI account directly. SMS-based verification is an evidence mechanism, not a bank/acquirer API and not a settlement guarantee.

## Runtime

The production deployment is intentionally one small service:

```text
Android phone / bank SMS
        │
        ├── Google Messages → libgm ──┐
        │                             │
        └── legacy Android relay ─────┤
                                      ▼
                         ┌──────────────────────────┐
                         │       PayGate (Go)       │
                         │                          │
                         │ PocketBase 0.39.9        │
                         │ SQLite + migrations      │
                         │ payment/SMS services     │
                         │ webhook outbox/worker    │
                         │ optional libgm manager   │
                         │ React/Vite operator UI   │
                         └────────────┬─────────────┘
                                      │
                                  /app/pb_data
                              persistent volume
```

There is no Redis, external queue, second database or custom realtime server. PocketBase provides SQLite, auth, SSE realtime, cron, logs, backups and its raw admin UI at `/_/`.

## Payment model

A caller requests a **whole rupee** amount. PayGate reserves one of 99 paise fingerprints for that rupee value.

Example:

```text
requested ₹100
possible payable amounts: ₹100.01 ... ₹100.99
```

`.00` is never allocated and allocation never spills into the next rupee. All money is integer paise internally.

The reservation is transactional and persisted. A payment remains `pending` until it is paid, cancelled or expires. Resolved/expired amounts are quarantined before reuse so a delayed SMS cannot normally confirm a newer payment.

An additional stale-evidence guard compares the SMS/provider occurrence timestamp with the payment's persisted `created_at`: an old Google Messages catch-up event that predates a reused payment cannot confirm that newer payment.

A finite quarantine cannot make amount-only verification mathematically unique forever: someone who deliberately pays an old QR **after** its amount has eventually been reused creates a new bank transaction at the current time and is indistinguishable from a new payer if the bank evidence exposes only amount/RRN. PayGate therefore fails closed where it can, keeps a long configurable quarantine, and treats official bank/acquirer references as the long-term path when stronger correlation is required.

Statuses:

- `pending`
- `paid`
- `expired`
- `cancelled`
- `late` — an exact credit arrived after expiry/cancellation but while that amount was still quarantined

## API

### Create a payment

```http
POST /api/payments
Authorization: Bearer <PAYGATE_API_KEY>
Idempotency-Key: <unique key>
Content-Type: application/json

{
  "amount": 100,
  "externalId": "order-123",
  "metadata": {"cart": "abc"}
}
```

`amount` may also be an integer string such as `"100"`. Fractional requested amounts are rejected.

Example response:

```json
{
  "id": "...",
  "paymentId": "...",
  "requestedAmount": 100,
  "requestedAmountPaise": 10000,
  "payableAmount": "100.37",
  "payableAmountPaise": 10037,
  "status": "pending",
  "expiresAt": "...",
  "paidAt": null,
  "externalId": "order-123",
  "upiUri": "upi://pay?..."
}
```

The same `Idempotency-Key` and identical parameters return the original payment. Reusing a key with different parameters returns `IDEMPOTENCY_CONFLICT`.

### Read public payment status

```http
GET /api/payments/{id}
```

This deliberately returns the limited public payment view. Bank evidence such as RRN, payer UPI ID and payer name is not exposed here.

### Cancel

```http
POST /api/payments/{id}/cancel
Authorization: Bearer <PAYGATE_API_KEY>
```

### Primary SMS ingestion

```http
POST /api/events/sms
X-Webhook-Secret: <SMS_WEBHOOK_SECRET>
Content-Type: application/json

{
  "sms": "Received Rs.100.37 ... UPI Ref:123456789012",
  "source": "android_webhook",
  "sourceId": "provider-message-id",
  "sender": "bank-sender",
  "timestamp": "2026-07-25T12:34:56Z"
}
```

`source` must be `android_webhook`, `gmessages` or `manual`. Supplying `sourceId` and the original `timestamp` is strongly recommended for durable deduplication and stale-message protection.

### Legacy Android relay

`POST /api/webhook` accepts the old `{ "sms": "..." }` payload only when `LEGACY_SMS_WEBHOOK_ENABLED=true`. It authenticates with the separate legacy `WEBHOOK_SECRET` and always records the source as `android_webhook`.

This compatibility route exists for migration only. The production default is disabled. Rotate the old relay to `SMS_WEBHOOK_SECRET` and `/api/events/sms`, then disable the legacy route.

### Operational APIs

Dashboard-authenticated operations:

- `GET /api/capacity` — active/quarantined suffix-pool utilization;
- `POST /api/review-cases/{id}/resolve` — audited review resolution/manual match;
- `POST /api/reconciliation/import` — multipart CSV/TSV/XLSX statement import;
- `GET /api/paygate/backups/status` — redacted archive status;
- `POST /api/paygate/backups` — create a backup now;
- `POST /api/paygate/backups/verify` — verify the latest archive;
- `POST /api/paygate/backups/restore-drill` — temporary extraction plus SQLite integrity checks.

API-key or dashboard-authenticated refund operations:

- `POST /api/refunds` with an `Idempotency-Key`;
- `POST /api/refunds/{id}/status`.

Refund endpoints record operator/bank evidence only; they never initiate a bank transfer.

### Health

- `GET /api/health` — PocketBase liveness endpoint, used by the container healthcheck.
- `GET /api/paygate/health` — PayGate readiness, database state and a redacted connector summary.

Unknown `/api/*` paths remain JSON 404 responses; the React SPA fallback never converts API errors into HTML 200 responses.

## Matching and idempotency

Bank SMS processing follows these rules:

1. persist the SMS evidence event;
2. parse exact amount, RRN and optional payer information;
3. reject automatic matching if amount or RRN is missing;
4. treat an already-seen RRN with the same amount as an idempotent duplicate;
5. treat the same RRN with a different amount as `RRN_AMOUNT_MISMATCH`;
6. use the bank/provider occurrence timestamp to decide whether the transaction was on time, so delayed SMS delivery alone does not turn an on-time payment into `late`;
7. match only the exact payable paise amount and eligible evidence window;
8. if no on-time payment matches, check an expired/cancelled payment still in quarantine and mark a genuinely late transaction `late`;
9. persist a review case instead of silently assigning incomplete, unmatched, contradictory or ambiguous evidence.

Google Messages catch-up messages retain their provider message timestamp. Legacy relays that omit a timestamp are treated as arriving at ingestion time, so upgrading the legacy relay to send timestamps is recommended.

## Outgoing webhooks

Configure `OUTGOING_WEBHOOK_URL` and `OUTGOING_WEBHOOK_SECRET` to receive payment lifecycle events.

Events currently include:

- `payment.paid`
- `payment.late`
- `payment.expired`
- `payment.cancelled`
- `refund.requested`
- `refund.processing`
- `refund.completed`
- `refund.failed`
- `refund.cancelled`

Delivery records are written transactionally to `webhook_deliveries`; network I/O happens only after the transaction commits. The worker uses durable retries and recovers stale `sending` leases after process restarts.

Headers:

```text
X-PayGate-Event-Id: <stable event id>
X-PayGate-Timestamp: <unix seconds>
X-PayGate-Signature: v1=<hex HMAC-SHA256>
```

Signature input:

```text
<timestamp>.<raw request body>
```

Consumers should verify the HMAC with `OUTGOING_WEBHOOK_SECRET` and deduplicate by event ID.

## Operator UI

The React UI at `/` provides:

- dashboard and payment statistics;
- payment creation;
- realtime payment list/details;
- cancellation;
- SMS evidence records;
- persistent evidence review and audited manual matching;
- CSV/TSV/XLSX bank-statement reconciliation;
- fingerprint-pool capacity monitoring;
- operational alerts and signed notification delivery state;
- manual refund lifecycle records;
- backup creation, verification and temporary restore drills;
- outgoing webhook delivery records;
- Google Messages connector status, Google-account/emoji pairing and QR fallback controls;
- safe non-secret configuration status.

Operator accounts use the PocketBase `users` auth collection. Domain collections are read-only through PocketBase APIs for authenticated `users`; state-changing payment operations go through custom Go handlers. Direct domain writes are locked.

PocketBase's own `/_/` interface remains available to superusers for low-level administration, logs, backups and schema inspection.

## Google Messages connector

The optional connector uses `go.mau.fi/mautrix-gmessages/pkg/libgm`.

Set:

```text
GMESSAGES_ENABLED=true
```

The connector:

- restores a persisted session from `pb_data/gmessages/session.json`;
- stores the session with restrictive filesystem permissions;
- connects using libgm's event-driven relay/long-poll implementation;
- forwards only incoming messages that resemble supported bank-credit SMS text;
- records the Google message ID and original timestamp;
- reconnects/backoffs on connection failures;
- reports paired/connected/phone-responsive state;
- supports the current Google-account + emoji (Gaia) pairing flow;
- accepts the upstream-required browser cookie set as cookie JSON, a raw Cookie header, or a DevTools Copy-as-cURL request;
- keeps QR pairing as a fallback and automatically refreshes short-lived QR data.

Google-account pairing is the primary path. Browser cookie input is never logged or echoed; it remains transient until pairing succeeds, after which libgm's AuthData (including the cookies required for account reauthentication) is stored in `pb_data/gmessages/session.json` with restrictive permissions.

Starting a new pairing is refused while another pairing is active or a valid session is already paired; explicitly cancel/unpair first.

The production connector has completed real-phone Google-account/emoji pairing, persisted its session across restarts and remained connected through the expected Tachyon authentication refresh boundary. This is strong acceptance evidence, but the private Google Messages protocol can still change; operational alerts and a reconciliation safety net remain necessary.

## Configuration

Copy `.env.example` and provide secrets through your deployment system rather than committing them.

Required in normal serve mode:

```text
UPI_ID=
PAYGATE_API_KEY=           # minimum 24 characters
SMS_WEBHOOK_SECRET=        # minimum 24 characters
```

Common optional values:

```text
UPI_PAYEE_NAME=PayGate
PAYMENT_TTL=5m
PAYMENT_QUARANTINE=24h
PAYGATE_RATE_LIMITS_ENABLED=true
GMESSAGES_ENABLED=false
GMESSAGES_SESSION_PATH=
OUTGOING_WEBHOOK_URL=
OUTGOING_WEBHOOK_SECRET=
OPERATOR_ALERT_WEBHOOK_URL=
OPERATOR_ALERT_WEBHOOK_SECRET=
STATEMENT_TIMEZONE=Asia/Kolkata
PAYGATE_RETENTION_ENABLED=true
SMS_RAW_RETENTION=2160h
RECONCILIATION_RAW_RETENTION=8760h
AUDIT_RETENTION=17520h
PAYGATE_BACKUP_CRON="0 3 * * *"
PAYGATE_BACKUP_MAX_KEEP=14
PB_DATA_DIR=./pb_data
```

Legacy prototype variables `UPI_NAME`, `TICKET_TTL_MINUTES`, `AMOUNT_QUARANTINE_HOURS` and `PAYMENT_WEBHOOK_*` remain understood where documented in `.env.example`. Invalid booleans/durations fail startup instead of silently falling back.

`PAYGATE_TEST_MODE=true` is for controlled tests only. It skips required production credentials, but duration, URL, timezone and feature-specific secret validation still applies.

## Docker

```bash
cp .env.example .env
# edit .env
docker compose up -d --build
```

The only state that must survive container replacement is mounted at:

```text
/app/pb_data
```

Do not deploy this image without a persistent volume/bind mount there. PocketBase SQLite, migrations, operator accounts, SMS evidence, outgoing webhook state, backups and the Google Messages session all depend on that directory.

## First operator account

PocketBase can create the first superuser through `/_/`, or from the container CLI with PocketBase's `superuser upsert` command. After that, create a normal record in the `users` auth collection for the PayGate operator UI.

## Tests and CI

Local validation:

```bash
npm ci
npm run typecheck
npm run build

gofmt -w cmd internal migrations
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...

docker build -t paygate .
```

CI performs frontend install/typecheck/build, Go formatting, unit/integration tests, race tests, vet and a production container build. Deployment is intentionally not automatic: the persistent-volume and environment cutover is an operator action.

## Security notes

- Treat `PAYGATE_API_KEY`, SMS secrets, PocketBase auth tokens and the libgm session as credentials.
- The Google Messages session can represent a paired Messages-for-Web client and is stored outside normal collections with restrictive permissions.
- Keep `/app/pb_data` private and backed up.
- Use HTTPS at the reverse proxy.
- PocketBase rate limits are enabled by default in PayGate.
- Do not leave the weak legacy `/api/webhook` compatibility route enabled after migration.
- A false-positive payment confirmation is worse than a delayed/manual review; matching therefore fails closed on ambiguity.
- SMS delivery and the private Google Messages protocol have no end-to-end latency/SLA guarantee.

## Licence

This repository directly links against `libgm`, which is AGPL-3.0. The rebuilt project is therefore distributed under the GNU Affero General Public License; see `LICENSE` and `NOTICE`.

A future proprietary/commercial distribution needs a separate licensing review rather than assuming architectural separation removes AGPL obligations.

## More detail

- `ARCHITECTURE.md` — implemented system design and invariants
- `PLAN.md` — implementation/acceptance status
- `OPERATIONS.md` — evidence review, reconciliation, refunds, alerts, backups and incident runbook
- `RESEARCH.md` — technical research and constraints behind the design
- `IMPLEMENTATION_SPEC.md` — rebuild requirements used during implementation
