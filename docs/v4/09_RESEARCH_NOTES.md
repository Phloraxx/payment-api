# 09 — Research Notes

This document records the concrete evidence behind the v4 design so future implementation does not have to rediscover why the architecture changed.

## Current-code findings

### The current frontend selects the payment account

Current file:

```text
payment-frontend/src/server/paygate.ts
```

`createPayment(...)` currently accepts a `paymentAccount` and sends it to `/api/payments`.

That means the caller/frontend currently participates in Paytm/Kotak/Slice routing. v4 removes this parameter from normal payment creation.

### The current API also accepts `paymentAccount`

Current file:

```text
payment-api/internal/api/api.go
```

`createPaymentBody` contains:

```text
paymentAccount
```

and the API exposes `/api/payment-accounts`.

v4 replaces that with a server-owned active collection profile.

### The current API already generates the canonical UPI URI

Current file:

```text
payment-api/internal/payments/service.go
```

`CreateResponse(...)` constructs `upi://pay?...` from the selected account and payable amount.

This responsibility stays on PayGate.

### The current frontend renders the QR

Current file:

```text
payment-frontend/src/client/pages/PaymentPage.tsx
```

The page uses `QRCodeCanvas` and renders the server-returned UPI URI.

This is already the desired presentation boundary. v4 keeps QR generation in the frontend and removes the frontend's account/verification branching.

### The current frontend leaks internal payment route concepts

`PaymentPage.tsx` currently branches on fields such as:

- `paymentAccount`
- `paymentFlow`
- `verificationMethod`
- Paytm/Kotak/Slice labels

v4 removes these from the customer/test UX.

### The Android app already receives all relevant phone notification surfaces

Current file:

```text
paygate-relay-android/app/src/main/java/io/github/phloraxx/paygaterelay/RelayConfig.java
```

The current relay recognizes:

```text
com.paytm.business
com.google.android.apps.nbu.paisa.user
com.google.android.apps.nbu.paisa.merchant
com.google.android.apps.messaging
```

Therefore a separate server-side Google Messages login is not required merely to obtain Google Messages-visible bank SMS notifications.

For v4.0 the allowlist should be narrowed to Paytm Business + Google Messages. GPay matching and Slice are deferred.

### The server currently treats non-Paytm Android notifications as observation-only

Current file:

```text
payment-api/internal/androidrelay/service.go
```

The current code routes only Paytm notification events into Paytm matching; other allowlisted app notifications are saved as `observed_only`.

This confirms that promoting Kotak/Google Messages into the unified parser/matcher is a well-defined change rather than inventing a new transport.

### The current server contains a separate libgm Google Messages subsystem

Current file:

```text
payment-api/internal/gmessages/manager.go
```

The manager handles:

- libgm session state
- Google pairing
- cookies/reauthentication
- reconnect behavior
- Google Messages client events
- filtering bank-credit messages

This is substantial duplicated transport responsibility once Android notification delivery is proven for Kotak.

### Current matching is more RRN-dependent than v4 needs

Current bank SMS matching requires amount + RRN, while current Paytm notification matching already demonstrates a different idempotency model using notification evidence identity.

v4 uses signed relay event identity for deduplication and profile+amount+time for matching. UTR/RRN is removed from the required contract.

### Current decimal pool is only `.01` to `.99`

Current `payment-api/internal/payments/service.go` iterates 99 suffixes and adds one of them to the requested paise value.

v4 generalizes this to the next whole rupee while still skipping `.00` and preferring the smallest free candidate.

### Current lifecycle defaults are 5m + 24h quarantine

Current `.env.example` contains:

```text
PAYMENT_TTL=5m
PAYMENT_QUARANTINE=24h
```

v4 replaces this conservative amount hold with explicit 5m active + 5m grace + 5m quarantine, because profile+amount+time matching and the bounded relay-delivery window make a 24-hour payable-amount lock unnecessary for the target flow.

This change must be proven with delayed-notification tests before production.

### Existing relay authentication is already strong

Current `payment-api/internal/androidrelay/service.go` validates ECDSA signatures against an enrolled device public key. Device identity is tied to the public key fingerprint and requests include timestamp/body integrity.

v4 keeps this security model. QR pairing only improves enrollment UX.

## Android platform research

### NotificationListenerService is the correct primitive

Android documents `NotificationListenerService` as the system service that receives notification posted/removed callbacks after the user grants notification-listener access.

Reference:

https://developer.android.com/reference/android/service/notification/NotificationListenerService.html

This supports PayGate's existing model of observing Paytm and Google Messages notifications without becoming the user's SMS app.

### Direct SMS permissions would make the product more complicated

Google Play treats SMS permissions as highly sensitive and generally requires the app to be the active default SMS/Phone/Assistant handler for those permissions.

References:

https://support.google.com/googleplay/android-developer/answer/16558241

https://support.google.com/googleplay/android-developer/answer/10208820

Therefore v4 should **not** replace notification capture with `READ_SMS`/`RECEIVE_SMS`.

### Android Keystore remains appropriate for device keys

OWASP Mobile guidance recommends platform/hardware-backed Android Keystore for cryptographic key material.

Reference:

https://mas.owasp.org/MASTG/knowledge/android/MASVS-STORAGE/MASTG-KNOW-0047/

The current ECDSA private key should remain non-exportable in Android Keystore.

## UPI research

Razorpay's current UPI documentation notes that UPI Collect/manual VPA entry has been deprecated for most normal use cases effective 28 February 2026 and directs integrations toward UPI Intent or QR.

Reference:

https://razorpay.com/docs/payments/payment-methods/upi/

PayGate's canonical UPI URI + frontend-rendered QR model therefore remains aligned with current UPI payment UX.

Razorpay's UPI Intent documentation also describes pre-populating the payment details in the UPI app and using QR on desktop flows:

https://razorpay.com/docs/payments/payment-methods/upi/upi-intent/

PayGate does not need to copy Razorpay's SDK; the relevant design lesson is that the server supplies canonical payment instructions while presentation chooses QR/intent appropriately.

## Razorpay visual research

Razorpay's products consistently use a strong blue financial identity and allow checkout branding/background color customization.

Reference:

https://razorpay.com/docs/payments/dashboard/account-settings/checkout-styling/

The v4 design takes the **dark navy + vivid blue + dense financial data** direction as inspiration but defines independent PayGate tokens and does not copy Razorpay's branding/assets/screens.

## Database research

### PocketBase is SQLite plus a backend framework

PocketBase describes itself as an open-source backend containing embedded SQLite, auth management, realtime subscriptions, dashboard UI and REST-ish APIs.

It also currently warns that it remains under active development before v1 and is not recommended for production-critical applications unless the operator accepts migration/changelog work.

Reference:

https://pocketbase.io/docs/

For v4, the extra framework is no longer providing enough value to justify remaining in the payment-critical runtime.

### SQLite is suitable for the current server shape

SQLite's own guidance explicitly lists server-side databases as an appropriate use when application-specific requests go through one application server. It cautions primarily when many independent writers or multiple networked servers need simultaneous direct writes.

Reference:

https://www.sqlite.org/whentouse.html

That maps closely to PayGate's production invariant of one API/server process owning the database.

### WAL fits PayGate's reader/writer pattern

SQLite WAL allows concurrent readers and a writer, while still retaining one writer at a time and requiring same-host database access.

Reference:

https://www.sqlite.org/wal.html

This is appropriate for one PayGate service on the Oracle VM.

### Backups should use SQLite-aware APIs

SQLite documents its online Backup API for consistent backups of a running database.

Reference:

https://www.sqlite.org/backup.html

The v4 backup plan therefore has the already-running PayGate process create the consistent backup, with the host exporter copying only the completed artifact. This prevents a repeat of the earlier second-process/live-database incident.

## Authentication/session research

OWASP recommends opaque, unpredictable session identifiers with at least 128 bits of entropy and Secure/HttpOnly/SameSite cookie protections.

Reference:

https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html

This supports a simple password-only singleton admin login without inventing JWT/account-role complexity.

## Design conclusions from the research

The simplest architecture that still preserves PayGate's safety properties is:

```text
one PayGate server
  -> direct SQLite
  -> server-owned collection profile
  -> server-owned amount allocator
  -> server-owned matching/parsers
  -> signed webhooks

one PayGate Android APK
  -> NotificationListenerService
  -> generic decimal-money prefilter
  -> durable queue
  -> ECDSA-signed events
  -> operator UI
```

The standalone payment frontend remains only a test/reference consumer.

The following are intentionally excluded from v4.0:

```text
server libgm
GPay auto matching
Slice
merchant-selected payment account
browser-owned UPI routing
RRN/UTR requirement
manual-review product workflow
reconciliation product workflow
PostgreSQL
multi-user roles
WebSockets
server-rendered QR images
```

Each excluded feature can be reconsidered only after a concrete use case appears.