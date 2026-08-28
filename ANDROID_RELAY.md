# PayGate Android Relay

`PayGate Relay` is an Android notification-evidence connector. The phone collects allowlisted notification metadata and sends signed evidence to PayGate; the phone never marks a payment successful.

## Security model

Each phone generates a P-256 ECDSA key in Android Keystore. The private key is non-exported. Enrollment sends only the public key and its SHA-256 fingerprint (`deviceId`). Each event request is signed over:

```text
POST
/api/relay/v1/events
<unix milliseconds>
<SHA-256 hex of exact request body>
```

Headers:

```text
X-PayGate-Relay-Device: <deviceId>
X-PayGate-Relay-Time: <unix milliseconds>
X-PayGate-Relay-Signature: <base64 ASN.1 DER ECDSA signature>
```

PayGate rejects disabled/unknown devices, invalid signatures, and request timestamps outside a five-minute window.

## Enrollment

Enrollment is closed by default. To pair a device, temporarily enable it and set a secret of at least 24 characters:

```env
ANDROID_RELAY_ENROLLMENT_ENABLED=true
ANDROID_RELAY_PAIRING_SECRET=replace-with-a-long-random-one-time-secret
```

The app calls `POST /api/relay/v1/enroll` with `X-Pairing-Secret`. After the intended phone has enrolled, set `ANDROID_RELAY_ENROLLMENT_ENABLED=false` and remove the pairing secret when practical. The route returns 404 while enrollment is disabled, and already-enrolled device signatures continue to work. The separate enable switch makes a stale secret inert after redeploys.

`relay_devices` stores the public key, device metadata, enabled flag, and last-seen time. Disabling a device immediately prevents future signed event ingestion.

## Event handling

`POST /api/relay/v1/events` accepts schema version 1 notification events from an allowlist:

- Paytm for Business: `com.paytm.business`
- Google Pay: `com.google.android.apps.nbu.paisa.user`
- Google Pay for Business: `com.google.android.apps.nbu.paisa.merchant`
- Google Messages: `com.google.android.apps.messaging`

All events are deduplicated by `(device, eventId)` and stored in `relay_events`.

Paytm for Business events are forwarded to the strict Paytm evidence parser. The parser uses the custom RemoteViews text exposed by AndroidX, for example:

```text
₹1.02 Received from Example User
Received on 27 Aug 2026 07:38 PM
```

The explicit `Received on ...` timestamp is parsed in `Asia/Kolkata` and is preferred as evidence occurrence time. Exact DDM amount and the existing payment-created-at guard still decide whether a payment can be matched.

Google Pay / GPay Business are observation-only until real success/refund/failure fixtures are collected and narrow parsers are tested. Google Messages notification relay is disabled by default in the Android app because the existing libgm connector is the primary SMS source and indiscriminate message-notification forwarding would expose unrelated SMS content.

## Heartbeat and fail-closed readiness

v0.3 sends the same signed canonical request to `POST /api/relay/v1/heartbeat`. The heartbeat reports app/Android version, notification-access state, listener-connected state, pending/failed queue counts and the last successful client delivery time and a bounded last client error. The operator API exposes aggregate/device status without exposing the private signing key.

`ANDROID_RELAY_STALE_AFTER` defaults to one hour. A Paytm `qr_only` checkout is created only when at least one enabled device has recent validated traffic. Once a v0.3 heartbeat exists, that device must also report both notification access and a connected `NotificationListenerService`. A missing/stale relay disables new Paytm checkouts but does not make Kotak/Slice or the PayGate process unhealthy.

The app schedules a network-constrained WorkManager safety heartbeat every 15 minutes and kicks immediate work after relevant notifications/listener state changes. Because periodic Android work is intentionally inexact, the server stale window is deliberately wider than one interval.

## Replay and privacy controls

The device records a pairing epoch and refuses to relay notifications older than that epoch (with a two-minute installation/pairing tolerance). The server independently compares notification post time with the persisted device enrollment boundary and stores older notifications as ignored evidence.

Production diagnostics are off by default. Disabled GPay/Messages sources are not extracted or persisted locally merely because their packages are allowlisted. Detailed extras and inflated RemoteViews dumps are kept only when the operator explicitly enables diagnostics; the normal Paytm path sends the narrow structured notification fields/custom text required for parsing.

Server raw Paytm notification and relay payloads default to 30-day retention (`PAYTM_NOTIFICATION_RAW_RETENTION`, `RELAY_RAW_RETENTION`). Matching status, amount, event IDs and payment linkage remain after raw text is redacted. The Android app prunes delivered/local-only history after seven days and failed local history after 30 days while never pruning pending/retrying events.


During the v0.2 → v0.3 rollout, an existing **enabled** relay device receives a bounded 48-hour heartbeat grace only if it had validated signed traffic within the preceding 24 hours. Disabled or long-idle devices and new enrollments receive no grace. The first signed heartbeat immediately switches the device to normal permission/listener/staleness readiness checks, and any operator enable/disable state change permanently clears migration grace for that device.
