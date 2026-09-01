# 02 — Notification Ingestion and QR Pairing

## Principle: phone is a sensor, PayGate is the brain

The Android app does not know the active collection profile and does not match payments.

It observes a tiny package allowlist, applies a cheap privacy-preserving decimal-money prefilter, stores a minimal notification snapshot locally and sends that snapshot to PayGate with a device signature.

The server decides:

- whether the notification is actually an incoming credit;
- whether it is Paytm or Kotak;
- what amount/payer/time can be trusted;
- whether any payment should change state.

## v4.0 package allowlist

Only:

```text
com.paytm.business
com.google.android.apps.messaging
```

GPay and Slice stay disabled for matching in v4.0.

This keeps the Android architecture generic while intentionally limiting production source risk.

## Generic on-device prefilter

Relay only when all are true:

1. package is allowlisted;
2. notification is not a group summary;
3. visible/extracted notification text contains a plausible INR/money token;
4. at least one amount-shaped token has two paise digits;
5. at least one candidate has non-zero paise (`.01`–`.99`);
6. notification event is not already queued locally;
7. notification was posted after the current device enrollment epoch.

Examples that should pass the cheap filter:

```text
₹101.37 Received from Rahul
INR 500.42 credited to A/c ...
Rs. 80.16 received via UPI
₹1,100.73 credited ...
```

Examples that should not be uploaded:

```text
₹100.00 debited ...
Your balance is ₹20,000.00
OTP 391204
ordinary personal chat
```

However, the phone does **not** need source-specific debit/credit grammar. If a decimal money notification passes the cheap filter but is actually a debit/balance message, the server parser rejects it.

## Never trust the phone's amount hint

Android may attach an optional `amount_hint` only because it already parsed enough to decide whether the notification is worth sending.

The server must re-parse the raw bounded text and derive the authoritative amount independently.

A malicious/buggy phone cannot mark a payment paid merely by sending a fabricated amount hint.

## Relay wire payload

Keep it source-neutral:

```json
{
  "event_id": "local-stable-event-id",
  "package_name": "com.paytm.business",
  "notification_key": "...",
  "posted_at_ms": 1788201000000,
  "title": "...",
  "text": "...",
  "big_text": "...",
  "amount_hint": "101.37"
}
```

Do not send:

- active profile;
- expected payment IDs;
- merchant/event identifiers;
- matching result guesses;
- UTR/RRN requirements.

## Stable local event identity

Android notification listeners can see the same notification more than once because of:

- notification updates;
- listener reconnect/rescan;
- process restart;
- retry after network failure.

The app must create/reuse a stable local event ID so the same notification does not become multiple server events.

At minimum the local capture layer should dedupe using a fingerprint derived from stable notification identity such as:

```text
package + Android notification key + original post time + normalized relevant content
```

Then store one local queue row with its own immutable UUID/ID.

Do not generate a fresh relay event ID every time active notifications are rescanned.

Server additionally enforces unique `(device_id, event_id)`.

## Notification updates

If an app updates the same notification with materially different payment-relevant content, treat it carefully:

- identical relevant content -> same event/retry;
- cosmetic/non-payment changes -> ignore;
- genuinely different amount/payer on the same Android key -> create a new local event version/fingerprint rather than mutating an already delivered event.

A delivered event is immutable.

## Server parser registry

### Paytm

Eligibility starts with:

```text
package_name == com.paytm.business
```

Parser verifies incoming-payment wording and extracts best-effort:

- exact amount;
- payer name;
- transaction/notification occurrence time if present.

Normalizes to profile `paytm`.

### Kotak through Google Messages

Eligibility starts with:

```text
package_name == com.google.android.apps.messaging
```

Parser must positively recognize a Kotak incoming-credit SMS/message before assigning profile `kotak`.

It extracts best-effort:

- exact amount;
- payer name;
- payer UPI ID when present;
- transaction time when text contains one.

Unknown decimal-money Google Messages notifications are ignored/unmatched; they are not treated as Kotak by default.

### GPay / Amazon Pay / Slice

Deferred for v4.0. Their notifications may be enabled only in a future explicit rollout with real sanitized fixtures and parser tests.

The normalized observation schema is intentionally not tied to a fixed source enum, so adding `gpay_notification`, `amazonpay_notification` or another vetted source later is a parser/routing change rather than a SQLite schema redesign.

A future app-level notification source does **not** need to know the currently active collection profile. The server parser/routing layer must determine which collection profile(s) that source can represent. If the source cannot be mapped confidently to one profile, it must remain unmatched/ambiguous rather than guessing.

## Normalized observation

Example:

```json
{
  "id": "obs_...",
  "source": "paytm_notification",
  "collection_profile": "paytm",
  "source_event_id": "...",
  "amount": "101.37",
  "payer_name": "Rahul",
  "payer_upi_id": null,
  "occurred_at": "...",
  "occurred_at_source": "notification_posted",
  "notification_posted_at": "...",
  "received_at": "..."
}
```

No UTR/RRN required.

## Timestamp handling

There are multiple clocks/timestamps:

- Android notification `postTime`;
- optional timestamp inside message text;
- device request-signature timestamp;
- server receive time.

Do not conflate them.

Matching preference:

1. source-specific transaction time parsed from trusted text;
2. Android notification post time;
3. server receive time only for diagnostics/low-confidence handling.

Server should detect implausible values such as:

- notification time far in the future;
- notification time predating device enrollment by an impossible amount;
- device signing clock outside allowed request tolerance.

Low-confidence time + reused payable amount must fail closed instead of auto-matching.

## Queue/delivery delay

A phone may be offline/dozing for minutes or longer.

The local row retains the **original notification post time**; retrying later does not replace it with the current delivery time.

Therefore a payment can still be matched to the correct historical reservation when the original occurrence time is trustworthy.

## Server dedupe and immutability

The signed relay event is an append/ingest object.

After accepted ingestion:

- same event ID retry returns prior result;
- event text/time is not silently rewritten;
- normalized observation keeps one source event;
- repeated processing cannot generate a second `payment.paid` transition/webhook.

### Cross-source corroboration

One real UPI credit may create more than one phone notification. Example: a Kotak account credit can produce both a Kotak SMS surfaced by Google Messages and a GPay notification; a future Amazon Pay integration could create another observation for the same money.

PayGate must not try to collapse those notifications on the phone. Each source event remains independently signed and stored. The **payment** is the dedupe anchor:

1. first safe observation -> `matched`; if needed, transition payment to `paid` and enqueue exactly one `payment.paid` webhook;
2. later independent observation for the same historical reservation -> `corroborated`, attach to the same payment and optionally enrich missing payer fields;
3. exact retry of the same relay event ID -> replay prior result without a second observation;
4. reused amount + insufficient timestamp confidence -> `ambiguous`, never assume it corroborates the newest payment.

This makes duplicate handling provider-agnostic and does not require UTR/RRN or fragile notification-text hashes.

## Privacy

Android should not upload unrelated personal notifications.

Rules:

- allowlist only required apps;
- cheap decimal-money filter before persistence/upload;
- bounded title/text/big-text sizes;
- short local retention;
- short server raw-event retention;
- no notification bodies in normal operational logs.

## Why no direct SMS permission

Do not request `READ_SMS`/`RECEIVE_SMS`.

Google Play heavily restricts SMS permissions unless the app is an eligible/default handler. PayGate already has the simpler NotificationListenerService path.

References:

- https://developer.android.com/reference/android/service/notification/NotificationListenerService.html
- https://support.google.com/googleplay/android-developer/answer/16558241

## Retire server-side libgm

Final v4 removes:

- `internal/gmessages`;
- server Google Messages login/session;
- cookies/reauthentication;
- QR pairing to Google Messages;
- libgm reconnect/client state;
- Google Messages connector UI.

Before deletion, run a short parity period using real Kotak transaction captures:

1. current libgm path remains available;
2. Android Google Messages notifications are captured in observe mode;
3. compare event coverage/timestamps/text completeness;
4. investigate every miss;
5. only enable Kotak auto-match after parity is convincing;
6. disable libgm;
7. observe production;
8. remove libgm code/state.

If Google Messages notifications do not reliably expose enough Kotak content on the real phone, do **not** pretend parity succeeded; solve that concrete transport problem before deleting the fallback.

## Seamless QR pairing

Preserve the current ECDSA device identity and replace only enrollment UX.

```text
Web Settings > Device > Connect phone
          |
          v
server creates one-time pairing session
          |
          v
dashboard shows QR
          |
          v
phone camera scans HTTPS App Link
          |
          v
PayGate Android opens
          |
          v
Android generates/reuses P-256 key in Android Keystore
          |
          v
POST token + public key + device metadata
          |
          v
server atomically consumes token and enrolls device
          |
          v
connected
```

Example App Link:

```text
https://pay.mulearnscet.in/device/pair/<single-use-token>
```

## Pairing-token rules

- only authenticated admin can create;
- cryptographically random, at least 128 bits;
- short TTL, e.g. 2 minutes;
- single-use;
- only token hash stored server-side;
- contains no admin password/API key/device private key;
- consumption and device enrollment happen atomically;
- failed/expired/used token cannot be replayed.

## One active relay phone in v4.0

Because the product currently needs one phone, keep the operational model simple:

```text
one active payment relay device
```

Pairing a second phone should require an explicit **Replace device** flow that revokes/disables the old relay only after the new enrollment succeeds.

This avoids two phones delivering duplicate notifications for the same bank/payment stream.

Historical device records may remain for audit.

## Device signing

Preserve:

- P-256/ECDSA private key in Android Keystore;
- device ID derived from public key fingerprint;
- public key stored server-side;
- signature covers method/path/timestamp/body hash;
- stale request signatures rejected;
- device revoke available from Settings.

Pairing does not make the admin password a relay credential.

## Local queue

State concept:

```text
pending -> delivered
        -> retry
        -> failed
```

Requirements:

- durable across process death/reboot;
- queue insertion before attempting network delivery;
- bounded exponential retry for network/5xx;
- auth/signature 4xx becomes visible health error rather than infinite retry;
- immutable local event identity;
- retries preserve original notification post time;
- history inspectable without blind bulk retry;
- old delivered/raw content pruned.

The existing foreground runtime, WorkManager fallback and Doze alarm watchdog remain unless testing proves a simpler equally reliable mechanism.