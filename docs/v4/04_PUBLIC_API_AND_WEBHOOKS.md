# 04 — Public API and Webhooks

## API philosophy

The merchant integration asks PayGate for a payment. PayGate decides the collection profile, payable amount and destination.

The API must never require a merchant to choose Paytm/Kotak or understand relay/matching internals.

## Create payment

```http
POST /v1/payments
Authorization: Bearer <merchant-api-key>
Idempotency-Key: <merchant-generated-key>
Content-Type: application/json
```

Request:

```json
{
  "amount": 100,
  "name": "IEEE workshop registration",
  "external_id": "reg_284",
  "metadata": {
    "event_id": "evt_12"
  }
}
```

### Request semantics

- `amount` — requested whole INR amount. v4 continues the current whole-rupee contract so PayGate owns paise for matching.
- `name` — human-readable alias shown in PayGate and returned to the integration. Required, trimmed, recommended max 120 Unicode characters.
- `external_id` — caller's ID. Optional but strongly recommended.
- `metadata` — optional bounded JSON object.

The request contains no payment-account/profile field.

## Create response

```json
{
  "id": "pay_01J...",
  "object": "payment",
  "name": "IEEE workshop registration",
  "external_id": "reg_284",
  "status": "pending",
  "currency": "INR",
  "requested_amount": "100.00",
  "payable_amount": "100.37",
  "adjustment": "0.37",
  "upi_uri": "upi://pay?pa=merchant%40paytm&pn=PayGate&am=100.37&cu=INR",
  "created_at": "2026-09-01T00:30:00+05:30",
  "expires_at": "2026-09-01T00:35:00+05:30",
  "grace_until": "2026-09-01T00:40:00+05:30",
  "paid_at": null
}
```

Do not return the active profile label to normal merchant/customer integrations. The UPI URI is the only payment-destination instruction they need.

The frontend is responsible for rendering `upi_uri` as a QR code.

## Why return strings for money

JSON floating point values are easy to mishandle. Public monetary responses should use fixed two-decimal strings.

Internal storage uses integer paise.

The request remains whole INR for the initial v4 contract because the current PayGate matching model reserves paise. If future merchant use cases require requested paise, redesign the allocator explicitly rather than accepting floats silently.

## Get payment

Authenticated merchant request:

```http
GET /v1/payments/{id}
Authorization: Bearer <merchant-api-key>
```

Pending response includes creation fields.

Paid response additionally includes best-effort payer information:

```json
{
  "id": "pay_01J...",
  "name": "IEEE workshop registration",
  "external_id": "reg_284",
  "status": "paid",
  "requested_amount": "100.00",
  "payable_amount": "100.37",
  "paid_at": "2026-09-01T00:33:11+05:30",
  "payer": {
    "name": "Rahul Kumar",
    "upi_id": "rahul@okaxis"
  }
}
```

`payer.name` and `payer.upi_id` are nullable independently.

## Lightweight status polling

The testing frontend should continue to poll instead of requiring WebSockets.

Recommended behavior:

- around 2 seconds while page is visible and payment is pending;
- immediate refresh when page regains visibility;
- stop on `paid`, `expired` or `cancelled`;
- exponential slowdown after connectivity errors.

For public browser polling, two options are acceptable:

1. frontend backend proxies the authenticated PayGate request (preferred for real merchant integrations); or
2. a limited public status endpoint keyed by the high-entropy payment ID returns only non-sensitive status fields.

If option 2 is retained, it must never expose payer identity or merchant metadata without merchant authentication.

## Cancel payment

```http
POST /v1/payments/{id}/cancel
Authorization: Bearer <merchant-api-key>
Idempotency-Key: <key>
```

Cancellation ends customer use immediately but keeps the payable amount reserved until the lifecycle's `reuse_after` boundary.

## Admin APIs

Admin routes are separate from merchant API keys, for example:

```text
POST  /admin/session
DELETE /admin/session
GET   /admin/overview
GET   /admin/payments
GET   /admin/payments/{id}
PATCH /admin/payments/{id}
GET   /admin/activity
GET   /admin/settings
PATCH /admin/settings/profile
POST  /admin/devices/pairing-session
GET   /admin/devices
DELETE /admin/devices/{id}
```

Exact route names can change during implementation; the product boundary should not.

## Merchant API keys

Merchant API keys remain separate from the one admin password.

Store only a keyed/hash representation sufficient for verification; show the plaintext key only when created/rotated.

Initial v4 can keep one merchant API key if that is all current integrations require. Do not build organizations, roles or OAuth.

## Webhooks

Webhooks remain a core integration mechanism.

Minimal event vocabulary:

```text
payment.created
payment.paid
payment.expired
payment.cancelled
payment.updated
```

`payment.updated` is for meaningful operator corrections that do not fit another state-transition event.

## Webhook envelope

```json
{
  "id": "evt_01J...",
  "type": "payment.paid",
  "created_at": "...",
  "data": {
    "payment": {
      "id": "pay_01J...",
      "name": "IEEE workshop registration",
      "external_id": "reg_284",
      "status": "paid",
      "requested_amount": "100.00",
      "payable_amount": "100.37",
      "paid_at": "...",
      "payer": {
        "name": "Rahul Kumar",
        "upi_id": "rahul@okaxis"
      },
      "metadata": {
        "event_id": "evt_12"
      }
    }
  }
}
```

## Webhook signing

Use a simple HMAC-SHA256 scheme with timestamped payload signing.

Example headers:

```text
PayGate-Event-Id: evt_...
PayGate-Timestamp: 1788210000
PayGate-Signature: v1=<hex-hmac>
```

Canonical bytes:

```text
<timestamp>.<raw-request-body>
```

Consumers verify HMAC with the webhook secret and reject timestamps outside a reasonable replay window.

## Outbox/retry behavior

Webhook rows live in a durable outbox table.

Requirements:

- insert payment state change and webhook row in the same database transaction;
- deliver asynchronously after commit;
- retry network errors/5xx with bounded exponential backoff;
- preserve response status and last error for admin Activity;
- never block payment matching while an integration endpoint is down;
- event ID makes consumer processing idempotent;
- operator can retry one exhausted webhook from the payment/activity detail if necessary, not a blind global bulk replay.

## Idempotency

Payment creation and other merchant writes accept `Idempotency-Key`.

Store the request fingerprint and resulting response/payment ID. Reusing a key with a different request returns a conflict; exact replay returns the original result.

Reference pattern: https://docs.stripe.com/api/idempotent_requests

## No QR/image API requirement

PayGate does not need endpoints such as:

```text
/payments/{id}/qr.svg
/payments/{id}/qr.png
```

The UPI URI string is canonical. The frontend/test application owns QR rendering.

This keeps PayGate transport/API responsibilities separate from presentation.