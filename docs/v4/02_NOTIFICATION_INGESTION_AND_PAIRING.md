# 02 — Notification Ingestion and QR Pairing

## Principle: the phone is a sensor, PayGate is the brain

The Android app must not need to know the currently active collection profile. It observes payment-like notifications and sends a minimal signed snapshot to PayGate.

The server is responsible for interpreting the notification and deciding whether it belongs to Paytm, Kotak or a future source.

## v4.0 Android package allowlist

Only two notification packages are required initially:

```text
com.paytm.business
com.google.android.apps.messaging
```

GPay packages remain disabled in the v4.0 matching rollout. Slice is deferred.

This is intentionally narrower than the current relay allowlist.

## Generic on-device prefilter

The phone should avoid uploading unrelated notifications, especially personal messages.

A notification is relay-eligible only if all of the following are true:

1. package is allowlisted;
2. it contains a money marker such as `₹`, `Rs`, `INR` or a known equivalent;
3. it contains a monetary amount with exactly two paise digits;
4. the paise are non-zero (`.01` through `.99`), because PayGate never allocates `.00`;
5. it is not a group-summary notification;
6. it was posted after device enrollment and has not already been queued locally.

The prefilter is deliberately generic. It does not determine whether a notification is actually a credit. The server parser does that.

Examples that pass the cheap filter:

```text
₹100.37 Received from Rahul
INR 501.42 credited to A/c ...
Rs. 80.16 received via UPI
```

Examples that should not be relayed:

```text
₹100.00 debited ...
Your balance is ₹20,000
OTP 391204
Message from a friend
```

The server still validates direction and source after delivery.

## Relay wire payload

Keep the device payload small and source-neutral:

```json
{
  "event_id": "local-stable-event-id",
  "package_name": "com.paytm.business",
  "posted_at": "2026-09-01T00:00:00+05:30",
  "title": "...",
  "text": "...",
  "big_text": "...",
  "amount_hint": "100.37"
}
```

`amount_hint` is optional and exists only because the phone already performed a generic decimal-money filter. The server reparses and validates the amount from the notification text.

Do not add active profile, expected payment IDs or matching state to this payload.

## Server parser registry

Server receives the snapshot and evaluates source-specific parsers in a tiny registry.

### Paytm parser

Eligibility:

```text
package_name == com.paytm.business
```

The parser must verify that wording represents an incoming customer payment, then extract:

- amount
- payer name when present
- occurrence timestamp when present, otherwise notification timestamp

It normalizes to profile `paytm`.

### Kotak parser

Eligibility:

```text
package_name == com.google.android.apps.messaging
```

The parser evaluates notification text as a bank-credit SMS and recognizes Kotak wording/sender markers. It extracts:

- amount
- payer name if text exposes one
- payer UPI ID if text exposes one
- occurrence time if reliable, otherwise notification timestamp

It normalizes to profile `kotak`.

Unknown decimal-money Google Messages notifications are ignored, not matched to Kotak by default.

### Future GPay/Slice

Add parsers later behind explicit tests and rollout flags. Do not make GPay matching part of the v4.0 cutover.

## Normalized observation

Once parsed, the source-specific shape disappears:

```json
{
  "id": "obs_...",
  "source": "paytm_notification",
  "collection_profile": "paytm",
  "source_event_id": "...",
  "amount": "100.37",
  "payer_name": "Rahul",
  "payer_upi_id": null,
  "occurred_at": "...",
  "received_at": "..."
}
```

No UTR/RRN is required.

## Deduplication

The signed relay event ID is the ingestion idempotency boundary.

Server stores a unique `(device_id, event_id)` pair. A retry of the same local queue item returns the previous result rather than creating a second observation.

A payment observation also records its matched payment ID once matched, making repeated processing harmless.

## Why not direct SMS permissions

Do not request `READ_SMS`/`RECEIVE_SMS` for PayGate. Google Play restricts SMS permissions primarily to apps acting as the default SMS/Phone/Assistant handler. PayGate does not need to become the user's SMS app merely to observe a bank-credit notification.

The existing NotificationListenerService route is the simpler product and permission model.

References:

- Android `NotificationListenerService`: https://developer.android.com/reference/android/service/notification/NotificationListenerService.html
- Google Play SMS/Call Log policy: https://support.google.com/googleplay/android-developer/answer/16558241

## Remove server-side Google Messages/libgm

Target v4 removes:

- `internal/gmessages`
- Google Messages server session files
- Google-account cookies
- QR pairing to Google Messages
- Google Messages reauthentication routes
- libgm reconnect/background state
- connector-specific admin health UI

Before removal, run a short parity period where Kotak credits observed through Android notifications are compared to the existing libgm path. Once the Android path proves complete, disable libgm, observe production, then delete it.

## Seamless PayGate phone pairing

The current relay already has a strong ECDSA device identity and signed-request scheme. Preserve it; replace only the awkward enrollment UX.

### Pairing flow

```text
Web Settings > Devices > Connect phone
          |
          v
PayGate creates one-time pairing session (2 min TTL)
          |
          v
Dashboard renders QR
          |
          v
Phone system camera scans QR
          |
          v
HTTPS App Link opens PayGate Android
          |
          v
Android generates/reuses P-256 key in Android Keystore
          |
          v
POST one-time token + public key + device metadata
          |
          v
Server consumes token and stores public key
          |
          v
Paired
```

The QR should encode an HTTPS app link such as:

```text
https://pay.mulearnscet.in/device/pair/<single-use-random-token>
```

Using the phone's normal camera avoids adding an in-app camera permission solely for pairing.

### Pairing-session properties

- generated only by an authenticated admin
- at least 128 bits of cryptographic randomness
- single use
- short TTL (recommended 2 minutes)
- stores no admin password in the QR
- cannot be used to authenticate as the admin
- consumed atomically when the device public key is enrolled

### Device signing

Preserve the current public-key model:

- P-256/ECDSA private key stays in Android Keystore;
- device ID derives from the public key;
- server stores only public key and device metadata;
- each event/heartbeat signs method, path, timestamp and body hash;
- stale signatures are rejected;
- device can be revoked from Settings.

This is better than replacing the proven key scheme with a shared relay password.

OWASP recommends platform keystores for cryptographic key material: https://mas.owasp.org/MASTG/knowledge/android/MASVS-STORAGE/MASTG-KNOW-0047/

## Local queue behavior

The Android app keeps a small local SQLite queue:

```text
pending -> delivered
        -> retry
        -> failed
```

Requirements:

- durable across reboot/process death
- exponential bounded retry for network errors/5xx
- 4xx auth/signature errors become visible device-health errors rather than infinite retry
- successful event delivery is idempotent
- historical failed rows are inspectable but not blindly bulk-retried
- raw notification content is pruned after a short retention period

The background heartbeat and Doze watchdog from v0.4.x remain.