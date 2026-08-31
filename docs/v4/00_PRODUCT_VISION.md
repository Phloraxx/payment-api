# 00 — Product Vision

## One product, two runtime artifacts

The name **PayGate** refers to the complete payment gateway, not only an HTTP API.

The finished system has only two first-class runtime artifacts:

1. **PayGate server**
2. **PayGate Android app**

The server contains the developer API, admin dashboard, payment state machine, collection-profile configuration, notification normalization, matching, webhooks and persistence.

The Android app is simultaneously the operator client and the background phone-notification relay. There must not be a separately managed “PayGate Relay” product.

The existing **payment-frontend** repo remains an integration playground used to test the PayGate API and customer QR/status UX. It is intentionally outside the product boundary.

## What the integrating application should know

An integrating backend should know only:

- requested amount
- a human-readable payment name/alias
- its own external ID
- optional metadata
- PayGate payment ID
- requested and payable amount
- canonical UPI URI
- payment status
- payer details after payment, when available
- timestamps

It must not know:

- whether Paytm or Kotak is active
- how phone notifications are captured
- whether the match originated from Paytm or Google Messages
- any relay-device identifier
- parser names
- reconciliation/evidence concepts

## What the customer-facing frontend should know

The frontend receives the canonical UPI URI and renders the QR itself. QR generation is a presentation concern.

Example boundary:

```text
Frontend/merchant backend
        |
        | amount=100, name="Workshop registration"
        v
      PayGate
        |
        | id, payable=100.37, upi_uri, timestamps
        v
Frontend renders QR and polls status
```

The frontend never chooses a collection profile.

## What the Android app should know

The Android background relay is intentionally dumb.

It should know:

- which Android packages are allowed to be observed
- whether a notification contains a plausible incoming money amount with non-zero paise
- the notification package, unique local event identity, timestamp and minimum text fields
- how to persist/retry a delivery
- how to sign requests to PayGate

It should not know:

- the currently active Paytm/Kotak collection profile
- which payment is expected
- PayGate amount allocation state
- payment matching rules
- source-specific business rules beyond the generic prefilter

Keeping the phone dumb means most parser changes are server releases instead of APK releases.

## v4.0 supported collection sources

### Paytm

- Receiving profile: Paytm
- Observation source: `com.paytm.business`
- Server parser: Paytm incoming-credit notification parser

### Kotak

- Receiving profile: Kotak
- Observation source: Google Messages notification (`com.google.android.apps.messaging`)
- Server parser: Kotak bank-credit SMS parser

Google Messages is only the Android notification surface. The target server does not log in to Google Messages and does not run libgm.

### Deferred

- GPay notifications
- Slice
- additional banks/payment apps

The ingestion architecture must make adding these a parser/profile addition rather than an Android architecture change.

## Admin product principles

The operator should think in terms of payments, not reconciliation machinery.

Primary information architecture:

```text
Overview
Payments
Activity
Settings
```

The operator must be able to:

- see total collections and transaction counts
- search every payment quickly
- filter by status/date/amount/profile
- open one payment and understand its full history
- edit allowed payment fields directly
- switch the active collection profile for future payments
- see incoming payment activity that did not match
- configure webhook delivery
- pair/revoke the Android device
- see whether notification monitoring is healthy

The operator should not need dedicated pages named Manual Review, Reconciliation, SMS Evidence, Paytm Evidence or Connector Management.

## Visual direction

The new interface should use a **Razorpay-inspired dark financial UI**:

- deep navy background
- layered blue-black surfaces
- clear electric-blue primary actions
- white/high-contrast monetary typography
- compact tables and transaction detail panels
- restrained status colors
- minimal decorative gradients

The design is inspiration, not a screen-for-screen Razorpay clone.

## Simplicity tests

Every proposed v4 component must pass these questions:

1. Does the merchant/customer need to know this exists?
2. Does it eliminate real ambiguity or failure?
3. Can it live inside an existing component instead?
4. Can this be a normal payment/activity field rather than a separate subsystem/UI?
5. Does it reduce or increase the number of things that must be deployed, paired or monitored?

If a feature fails those tests, omit it unless production evidence requires it.