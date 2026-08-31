# 01 — Target Architecture

## System shape

```text
                         Merchant / test frontend
                                  |
                         HTTPS developer API
                                  |
                                  v
                         +------------------+
                         |     PayGate      |
                         |------------------|
                         | Payment service  |
                         | Profile service  |
                         | Observation      |
                         | parsers          |
                         | Match engine     |
                         | Admin dashboard  |
                         | Webhook outbox   |
                         | Auth/devices     |
                         +---------+--------+
                                   |
                                SQLite
                                   ^
                                   |
                         signed relay events
                                   |
                         +---------+--------+
                         | PayGate Android  |
                         |------------------|
                         | Operator UI      |
                         | Notification NLS |
                         | Local queue      |
                         | Device signer    |
                         | Doze watchdog    |
                         +------------------+
```

There is no server-side Google Messages connector in the target architecture.

## Server modules

The server remains one process and one deployable image, but code is separated into small domain modules:

### `payments`

Owns payment creation, status transitions, amount reservations and payment edits.

### `profiles`

Owns collection profiles and the single active profile used for **new** payments.

A payment snapshots its selected profile/destination when created. Switching the active profile never changes existing payments.

### `relay`

Owns device pairing, enrolled public keys, signed relay request verification, heartbeat and event deduplication.

### `observations`

Owns normalization of incoming notification snapshots into payment observations.

A normalized observation contains only payment-relevant data:

```text
id
source
collection_profile
amount
payer_name?
payer_upi_id?
occurred_at
received_at
source_event_id
notification_excerpt?
matched_payment_id?
```

UTR/RRN is not a required field in v4.

### `matching`

Matches normalized observations against reserved payments by:

- inferred collection profile
- exact payable amount
- payment/observation time windows
- unique observation identity

### `webhooks`

Durable outbox for merchant events. Payment creation and webhook scheduling occur transactionally so a paid payment cannot lose its outbound event.

### `admin`

Password-only operator authentication, dashboard APIs, payment editing, profile settings, device settings and webhook settings.

### `storage`

Small repository layer over SQLite. The domain does not expose database table/record objects to handlers.

## Collection profiles

Initial table/model:

```text
collection_profiles
-------------------
id                    paytm / kotak
label                 Paytm / Kotak
enabled               true/false
upi_id                 destination VPA
payee_name             UPI payee name
parser                 paytm_notification / kotak_sms
active                 exactly one enabled row is active
created_at
updated_at
```

The API never accepts a `paymentAccount`/profile selector from normal payment-creation clients.

Creation sequence:

```text
POST payment
    |
    v
read active profile
    |
    v
reserve payable amount inside that profile
    |
    v
snapshot destination/profile on payment
    |
    v
build canonical UPI URI
```

A profile switch is atomic and applies to subsequent creations only.

## Why profile-scoped amount reservation

The same exact payable amount can safely exist simultaneously on different collection destinations because incoming observations identify/infer the destination profile.

Example:

```text
Paytm  -> ₹100.37
Kotak  -> ₹100.37
```

These are not ambiguous because the Paytm app notification and Kotak bank-message notification normalize to different profile IDs.

Within one profile, however, an unexpired reservation must own a payable amount exclusively.

## Android/server responsibility split

### Android

- package allowlist
- generic decimal-money prefilter
- notification snapshot capture
- local durable delivery queue
- request signing
- heartbeat/background survival

### Server

- source-specific parsing
- incoming-vs-outgoing interpretation
- profile inference
- amount extraction validation
- payer extraction
- payment matching
- status mutation

This split is deliberate. Notification wording changes should usually require a server parser update, not an APK update.

## PayGate Frontend boundary

The separate frontend remains a test/reference integration and can continue to have its own deployment/repo.

Its correct v4 job is:

1. ask its backend to create a PayGate payment;
2. receive the PayGate response;
3. render `upi_uri` as a QR;
4. display requested/payable amount and expiry;
5. poll PayGate status;
6. demonstrate paid/expired UX.

It must not:

- list collection profiles to the customer;
- choose Paytm/Kotak;
- derive a UPI destination itself;
- apply matching logic;
- know whether the source is notification or SMS.

## Deployment

Production target remains deliberately small:

```text
Docker/Swarm host
  PayGate service: 1 replica, stop-first
  SQLite file: local durable volume
  host backup/export job

Android
  one PayGate APK
```

The one-server-writer rule remains until the database is intentionally migrated to a client/server DB.

## What is removed from the target architecture

- browser-side profile selection
- `/api/payment-accounts` as a merchant decision endpoint
- server libgm/Google Messages session
- Google Messages pairing/reauth UI
- source-specific relay toggles in normal Android UX
- manual-review subsystem as a product workflow
- reconciliation subsystem as a primary UI workflow
- separate PayGate Relay product identity
- RRN/UTR as a match requirement
- PocketBase record objects as the domain model

## Future extension rule

Adding GPay or Slice later must be possible by adding:

1. an allowlisted Android package only if a new app package is needed;
2. one server parser;
3. one collection profile/source mapping;
4. tests with captured sanitized notification fixtures.

It must not require redesigning payment creation or the Android/server protocol.