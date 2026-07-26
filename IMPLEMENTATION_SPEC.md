# PayGate Rebuild — Implementation Specification

This specification defined the clean implementation on `rebuild-pocketbase`. The old Node/Fastify implementation was prototype/reference material and is removed from the rebuilt tree once equivalent behaviour is covered by tests.

## Product boundary

Personal, self-hosted UPI payment verification for projects owned by one operator. Money goes directly to the configured UPI account. The service does not hold or settle funds.

## Required stack

- Go 1.25.x.
- PocketBase v0.39.9 embedded as the Go application framework.
- PocketBase SQLite as the only database.
- React + Vite + TypeScript operator UI embedded in the Go binary.
- `go.mau.fi/mautrix-gmessages`/libgm v0.2605.0 as an optional Google Messages connector.
- One application process/container and one persistent `pb_data` volume.

## DDM invariants

- API `amount` is a positive whole number of INR rupees. Reject paise/fractional requested amounts.
- All domain money is integer paise.
- `requested_amount = amount * 100`.
- Allocate `payable_amount` only in `requested_amount + 1..99`; `.00` is excluded.
- Never spill into the next whole rupee. If all 99 suffixes are unavailable, return `AMOUNT_CAPACITY_EXHAUSTED`.
- Reserve enough `int64` headroom that adding suffix 99 cannot overflow.
- Exact `payable_amount` is the payment correlation key. Never floor to a base amount for matching.
- Allocation is transactional/durable; there is no in-memory decimal pool.
- Default payment TTL is 5 minutes, configurable.
- Every resolved/expired amount enters configurable quarantine (default 24 hours) before reuse.
- At creation, `reuse_after = expires_at + quarantine`. Paying/cancelling/late-paying resets it to processing time + quarantine.
- A late SMS for a still-quarantined expired/cancelled payment attaches to that old payment as `late`.
- Provider message occurrence time is retained. Evidence may not confirm a payment created after the evidence occurred.
- Duplicate source events and duplicate same-amount RRNs are idempotent.
- Same RRN with a different amount is an anomaly and must not confirm another payment.
- Restart must not expire valid payments or lose allocation state.

## Data model

PocketBase migrations create:

1. `users` auth collection for the operator dashboard.
2. `payments`: explicit business `created_at`, requested/payable amount, status (`pending|paid|expired|cancelled|late`), expiry/quarantine, RRN, payer evidence, `paid_at`, external ID, idempotency key, metadata.
3. `sms_events`: source (`android_webhook|gmessages|manual`), provider ID, sender/body, message time, parsed evidence, processing status, matched payment, error/raw connector metadata.
4. `webhook_deliveries`: durable outgoing webhook outbox/retries.

Use non-empty uniqueness for RRN, idempotency key and `(source, source_event_id)`. Authenticated records from the `users` collection may read domain records; direct record writes remain locked so business invariants cannot be bypassed.

## SMS/payment processing

- Strictly parse the tested Kotak bank-credit formats first.
- Extract exact amount, RRN/UPI Ref and optional UPI ID/payer name.
- Require exact amount and RRN for automatic paid/late transitions.
- Persist SMS evidence before/with processing.
- Matching and payment transition occur transactionally.
- Bound request/derived fields before PocketBase validation.
- `POST /api/events/sms` is the primary strong-secret endpoint.
- `POST /api/webhook` is an explicitly enabled migration-only compatibility route for the old Android relay.

## HTTP API

- `POST /api/payments`: create; API key or operator auth; `Idempotency-Key` supported.
- `GET /api/payments/{id}`: public limited status; sensitive bank evidence omitted.
- `POST /api/payments/{id}/cancel`: API key or operator auth.
- `POST /api/events/sms`: primary SMS secret; legacy `{sms}` and richer metadata shapes supported.
- `POST /api/webhook`: old route only when `LEGACY_SMS_WEBHOOK_ENABLED=true`, authenticated with separate `WEBHOOK_SECRET`.
- `GET /api/health`: PocketBase liveness used by the container healthcheck.
- `GET /api/paygate/health`: PayGate DB/readiness and redacted connector summary.
- `GET /api/config`: authenticated safe configuration/connector status.
- `GET /api/dashboard`: authenticated summary.
- connector status, Google-account/emoji pair, QR fallback/refresh, reconnect and unpair routes are dashboard-authenticated.

The SPA must never swallow unknown `/api/*` or `/_/*` routes.

Create response includes requested amount, payable amount, expiry, payment ID and a UPI URI with configured `pa`, `pn`, exact `am`, `cu=INR`, and payment ID in `tr`/`tn`. Verification does not depend on `tr`.

## Authentication and secrets

- External API: `Authorization: Bearer $PAYGATE_API_KEY`.
- Primary SMS: `X-Webhook-Secret: $SMS_WEBHOOK_SECRET`.
- Legacy route: separate `WEBHOOK_SECRET`, opt-in only.
- Dashboard: PocketBase `users`; the SPA never receives PocketBase superuser credentials.
- `UPI_ID`, API key and primary SMS secret are required in non-test mode.
- Primary API/SMS secrets have a minimum strength requirement.
- Never log secret values.
- Optional outgoing webhook requires valid absolute URL + HMAC secret.
- Invalid boolean/duration environment values fail startup instead of silently defaulting.
- PocketBase rate limits are enabled by default through PayGate configuration.

## Outgoing webhooks

When configured, persist/send `payment.paid`, `payment.late`, `payment.expired` and `payment.cancelled`. Sign `${timestamp}.${rawBody}` using HMAC-SHA256 and expose stable event ID/timestamp/signature headers. Retry failures durably with bounded backoff. Restart must not lose pending deliveries; stale sending leases must recover.

## Realtime and UI

Use PocketBase realtime for authenticated operator views. No custom WebSocket server.

UI pages: Login, Dashboard, Payments/create/details, SMS Events, Webhook Deliveries, Settings/connector. PocketBase `/_/` remains available for raw administration/debugging.

## libgm connector

libgm is optional infrastructure, not the payment model.

- Persist AuthData under `pb_data/gmessages/session.json` with restrictive permissions.
- Prefer Google-account/Gaia pairing: validate the upstream-required Google cookie set, fetch config, display the derived emoji, wait for phone confirmation, then persist the completed session.
- Cookie input may be cookie JSON, a raw Cookie header, or a DevTools Copy-as-cURL request; values must never be logged or echoed.
- Keep QR pairing as a fallback and refresh short-lived QR tokens.
- Refuse accidental replacement of an existing valid pairing or another pairing already in progress; cancel/unpair first.
- When enabled/paired, connect on serve, persist token refreshes, process incoming text `WrappedMessage` events into the same SMS service, and reconnect/back off on failure.
- Keep provider message ID and original timestamp, including catch-up/old events.
- Report paired/connected/phone-responsive/timestamp/error state to authenticated operator views.
- Application remains healthy when unpaired/offline.
- Connector is read-only; do not send SMS/RCS.
- Real phone Google-account/emoji pairing and reconnect persistence remain a device acceptance test; QR is fallback only.

## Expiry/background work

Persist timestamps as truth. PocketBase cron checks expiry and wakes webhook retry processing. Lazy expiry in create/match/status paths prevents correctness from depending on cron timing. The webhook worker also runs its own periodic wake cycle.

## Deployment

- Multi-stage Node/Go build with a non-root distroless runtime.
- Listen on port 3000 for existing Dokploy routing.
- Runtime data directory `/app/pb_data` **must** be a Dokploy persistent mount before switching `main`.
- Healthcheck uses PocketBase `/api/health`.
- Keep an external backup of the old prototype DB; do not silently mix schemas.
- The weak old Android secret may be retained only behind the opt-in migration route until the relay is upgraded, then removed.

## Required validation

Tests cover money boundaries, all 99 DDM slots, concurrency/exhaustion, restart persistence, exact matching, expiry/quarantine/late payments, delayed catch-up after amount reuse, duplicate/contradictory RRN, provider dedupe, parser variants, auth/redaction/body limits, config strictness, collection rules, webhook HMAC/retries/claim recovery, connector session handling and API namespace behaviour.

Frontend must typecheck/build. Final source must pass uncached tests, race tests, `go vet`, formatting, `git diff --check`, dependency audit and a production Docker build.

Before moving `main`, run the final image on a fresh temporary volume and exercise health, API auth/validation, creation/idempotency, SMS matching, duplicate handling, redaction, container recreation and database persistence. Review the final diff multiple times for stale prototype code, secret leakage, dead code, unsafe direct writes, matching errors and deployment persistence mistakes.
