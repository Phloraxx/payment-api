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

Set a temporary secret of at least 24 characters:

```env
ANDROID_RELAY_PAIRING_SECRET=replace-with-a-long-random-one-time-secret
```

The app calls `POST /api/relay/v1/enroll` with `X-Pairing-Secret`. After the intended phone has enrolled, the pairing secret can be removed from the environment; enrolled-device event signatures continue to work.

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
