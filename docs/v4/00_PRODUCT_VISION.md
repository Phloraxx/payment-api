# 00 — Product Vision

## One product, two runtime artifacts

**PayGate** is the payment gateway, not merely an HTTP API.

The final system has two first-class runtime artifacts:

1. **PayGate server**
2. **PayGate Android app**

The server owns developer API, payment state, collection profiles, amount reservation, notification parsing, matching, direct SQLite persistence, admin dashboard and webhooks.

The Android app is both operator client and background phone-notification sensor. There is no separately managed Relay product.

The existing **payment-frontend** remains a separate integration/testing playground and is intentionally outside the PayGate product boundary.

## Merchant-facing mental model

The integrating system asks:

> Create a ₹100 PayGate payment for this person, related to this event.

Example:

```json
{
  "amount": 100,
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026"
}
```

PayGate answers with:

- PayGate payment ID;
- requested amount;
- randomly reserved exact payable amount;
- adjustment;
- canonical UPI URI;
- status/lifecycle timestamps;
- payer details later when a notification exposes them.

## `name` is not the event name

`name` is the merchant-supplied **human/payee/person identifier** attached to the payment.

It is not used to decide whether money was received.

Actual notification-derived sender data is separate:

```text
name        = person the merchant associates with the payment
payer_name  = person/account name observed from the actual payment notification
```

They may differ. For example, a parent may pay for a student's registration.

## `external_id` is the event ID

`external_id` is the event identifier supplied by the integrating system.

It may repeat across many PayGate payments because one event can have many attendees/payers.

Therefore it is context/filtering data, not:

- a primary key;
- a unique constraint;
- an idempotency key;
- a matching key.

The integration stores the returned PayGate payment ID for each payment. `Idempotency-Key` prevents accidental duplicate creation.

## What the integrating application should know

It should know only:

- amount;
- person/payee `name`;
- event `external_id`;
- optional metadata;
- PayGate payment ID;
- requested/payable amount and adjustment;
- canonical UPI URI;
- payment status;
- payer details after payment when available;
- timestamps.

It must not know:

- whether Paytm or Kotak is currently active;
- which Android package produced confirmation;
- parser/relay concepts;
- amount-pool state;
- Google Messages internals.

## Frontend responsibility

PayGate returns the canonical UPI string.

```text
merchant/test frontend
       |
       | amount + name + event ID
       v
    PayGate
       |
       | payment ID + exact payable + upi_uri
       v
frontend renders QR and polls payment status
```

The frontend never chooses the collection profile and never constructs a destination UPI ID itself.

## Android responsibility

The phone is a trustworthy sensor, not the payment engine.

It knows:

- allowed notification packages;
- a cheap generic decimal-money prefilter;
- notification package/key/text/post time;
- durable local queue/retry;
- device request signing;
- heartbeat/background survival.

It does **not** know:

- active profile;
- payment candidates;
- amount reservation state;
- payment matching rules;
- whether a particular notification ultimately pays anything.

## v4.0 sources

### Paytm

```text
Paytm for Business notification
-> PayGate Android
-> signed relay event
-> PayGate server Paytm parser
-> profile paytm
```

### Kotak

```text
Kotak bank credit SMS
-> Google Messages notification
-> PayGate Android
-> signed relay event
-> PayGate server Kotak parser
-> profile kotak
```

There is no final server-side Google Messages/libgm login.

### Deferred

- Google Pay matching;
- Slice;
- other banks/payment apps.

Their future addition must be a parser/profile extension, not another relay architecture.

## Amount identity principle

PayGate deliberately generates a non-`.00` exact amount and reserves that amount within the receiving profile.

V4 selection is randomized across the bounded free pool rather than sequential `.01`, `.02`, `.03` assignment.

A short hard lifecycle plus soft recent-use avoidance protects against delayed notifications without a 24-hour capacity lock.

Matching uses:

```text
profile + exact payable amount + trustworthy occurrence time + unique relay event
```

not merchant name, event ID or UTR/RRN.

## Admin product principles

Primary navigation:

```text
Overview
Payments
Activity
Settings
```

Operator should be able to:

- see collection totals and payment counts;
- search/filter every payment;
- open one payment and understand it immediately;
- edit allowed fields directly;
- switch the active collection profile for future payments;
- inspect unmatched incoming payment activity;
- pair/replace/revoke the Android phone;
- see relay/webhook/backup health.

The operator should not have to understand separate products called Manual Review, Reconciliation, Evidence or Google Messages Connector.

## Visual direction

Use a Razorpay-inspired dark financial interface:

- deep navy/blue-black base;
- vivid blue primary action;
- white high-contrast money typography;
- dense but readable transaction tables;
- restrained status colors;
- minimal decoration.

Use Razorpay as information/visual inspiration, not as a copied brand or screen design.

## Simplicity tests

Every v4 feature must answer:

1. Does payment correctness require it?
2. Does merchant/operator UX require it?
3. Can SQLite enforce this invariant instead of custom recovery code?
4. Can this be a field/history event instead of another subsystem?
5. Does it reduce deployed/pairing/operational moving parts?

If not, omit it until a real production use case proves otherwise.