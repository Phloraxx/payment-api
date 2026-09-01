# 04 — Public API and Webhooks

## API philosophy

The merchant asks PayGate for a payment. PayGate decides the collection profile, exact payable amount and destination UPI instruction.

The merchant does **not** choose Paytm/Kotak and does not know relay/matching internals.

The frontend/test application receives the canonical UPI URI string and renders the QR itself.

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
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026",
  "metadata": {
    "registration_id": "reg_284"
  }
}
```

## Exact field semantics

### `amount`

Requested whole-INR amount for the initial v4 contract.

PayGate owns the paise/verification adjustment and returns the exact payable amount.

Rules:

- must be positive;
- parsed as an integer, never floating point;
- server applies a configured upper bound to prevent abuse/overflow;
- requested paise support is a future explicit design, not an implicit float feature.

### `name`

Required merchant-supplied **human/payee/person identifier** for this payment.

Typical examples:

```text
Sourav P Bijoy
Rahul Kumar
Team Phoenix
```

It is **not** the event name and is **not** assumed to equal the actual bank/UPI sender name.

The notification-derived sender is returned separately as `payer.name` when available.

`name` is display/context data only. It does not participate in matching or uniqueness.

### `external_id`

The merchant/event-side **event ID**.

Example:

```text
evt_hardware_security_2026
```

Many payments may legitimately share the same `external_id` because many people can pay for one event.

Therefore:

- `external_id` is not unique;
- it is not an idempotency key;
- it is not used for matching;
- it is useful for filtering, reporting and webhooks.

If the integrating system has a registration/order/member-specific ID, it may include that in `metadata` and should store the returned PayGate payment ID in its own record.

### `metadata`

Optional bounded JSON object for integration context.

Examples:

```json
{
  "registration_id": "reg_284",
  "department": "CSE"
}
```

Do not use metadata as part of the payment-matching algorithm.

### `Idempotency-Key`

This is the request-uniqueness mechanism.

A frontend/backend retry with the same key and exact same request returns the original payment. Reusing the same key with different request content returns a conflict.

`name` and `external_id` may repeat; the idempotency key is what prevents accidental duplicate creation.

## Create response

Example only; the actual payable amount is randomized from the server-owned free pool:

```json
{
  "id": "pay_01J...",
  "object": "payment",
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026",
  "status": "pending",
  "currency": "INR",
  "requested_amount": "100.00",
  "payable_amount": "100.37",
  "adjustment": "0.37",
  "upi_uri": "upi://pay?pa=merchant%40paytm&pn=Merchant&am=101.37&cu=INR",
  "created_at": "2026-09-01T00:30:00+05:30",
  "expires_at": "2026-09-01T00:35:00+05:30",
  "grace_until": "2026-09-01T00:40:00+05:30",
  "paid_at": null
}
```

The response does not need to expose the active profile label to a normal merchant/customer integration. The UPI URI is the canonical payment instruction.

## QR boundary

PayGate returns only the UPI URI string.

It does **not** need:

```text
/payments/{id}/qr.svg
/payments/{id}/qr.png
hosted checkout URL
```

The consuming frontend turns `upi_uri` into a QR code and may also use the same URI for UPI intent/open-app behavior where appropriate.

## Money representation

Public response money fields use fixed two-decimal strings.

Internal storage uses integer paise.

Never use binary floating point for money or amount-allocation decisions.

## Get payment

```http
GET /v1/payments/{id}
Authorization: Bearer <merchant-api-key>
```

Pending response contains the original merchant context and payment instruction.

Paid response additionally contains best-effort notification-derived payer information:

```json
{
  "id": "pay_01J...",
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026",
  "status": "paid",
  "requested_amount": "100.00",
  "payable_amount": "100.37",
  "paid_at": "2026-09-01T00:33:11+05:30",
  "payer": {
    "name": "Bijoy P",
    "upi_id": "bijoy@okaxis"
  },
  "metadata": {
    "registration_id": "reg_284"
  }
}
```

This example intentionally shows `name != payer.name`; somebody else may pay for the named person.

`payer.name` and `payer.upi_id` are nullable independently.

## Status polling

The separate testing frontend should continue to use polling rather than WebSockets.

Recommended behavior:

- roughly every 2 seconds while the page is visible and payment is pending;
- immediate refresh on visibility regain;
- stop on `paid`, `expired`, or `cancelled`;
- slow down after repeated network errors.

No payment correctness depends on the browser staying open.

## Public browser status

For real merchant integrations, prefer merchant-backend proxying/authenticated status reads.

If a public status endpoint is retained for the test frontend, return only minimal non-sensitive fields and require a high-entropy unguessable payment identifier/token.

Never expose merchant metadata or notification-derived payer identity publicly without authorization.

## Cancel payment

```http
POST /v1/payments/{id}/cancel
Authorization: Bearer <merchant-api-key>
Idempotency-Key: <key>
```

Cancellation prevents continued customer use but does not immediately release the reserved payable amount. Reservation release follows the matching lifecycle.

## Admin APIs

Representative routes:

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

Exact route names may change; the product boundary should not.

## Merchant API keys

Merchant API keys remain separate from the singleton admin password and Android device key.

Store a verifier/hash, not plaintext. Plaintext is shown only at creation/rotation.

Do not build organizations, OAuth or role hierarchies for v4.0 unless a real use case appears.

## Webhooks

Minimal useful event vocabulary:

```text
payment.created
payment.paid
payment.expired
payment.cancelled
payment.updated
```

A webhook should contain merchant context plus best-effort payer information after payment.

Example:

```json
{
  "id": "evt_01J...",
  "type": "payment.paid",
  "created_at": "...",
  "data": {
    "payment": {
      "id": "pay_01J...",
      "name": "Sourav P Bijoy",
      "external_id": "evt_hardware_security_2026",
      "status": "paid",
      "requested_amount": "100.00",
      "payable_amount": "100.37",
      "paid_at": "...",
      "payer": {
        "name": "Bijoy P",
        "upi_id": "bijoy@okaxis"
      },
      "metadata": {
        "registration_id": "reg_284"
      }
    }
  }
}
```

The merchant should key its payment handling on PayGate payment ID/webhook event ID, not on `external_id`, because one event ID may have many payments.

## Webhook signing

Use HMAC-SHA256 over timestamp + exact raw body.

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

Consumers verify signature in constant time and reject stale timestamps according to the documented replay window.

## Durable outbox

Insert payment state mutation/history and outbound webhook row in the **same SQLite transaction**.

Delivery happens after commit.

Requirements:

- network failure never rolls back a paid payment;
- retry network errors and retryable 5xx using bounded exponential backoff;
- retain last HTTP status/error for Activity;
- event ID lets consumers dedupe;
- do not blindly replay every exhausted historical webhook;
- manual retry operates on one explicit event/payment.

## Idempotency edge cases

Required behavior:

- same key + same normalized request -> same payment/response;
- same key + changed amount -> conflict;
- same key + changed `name` -> conflict;
- same key + changed `external_id` -> conflict;
- different idempotency keys may create multiple legitimate payments for the same `name` and same event `external_id`;
- an idempotency record and created payment are committed atomically.