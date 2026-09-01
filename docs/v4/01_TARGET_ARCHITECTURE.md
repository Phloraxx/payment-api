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
                         | Payments         |
                         | Profiles         |
                         | Amount allocator |
                         | Relay verifier   |
                         | Parsers          |
                         | Matcher          |
                         | Admin dashboard  |
                         | Webhook outbox   |
                         +---------+--------+
                                   |
                              direct SQLite
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

The separate payment frontend remains a test/reference consumer. It is not merged into PayGate and it does not select profiles or perform matching.

There is no server-side Google Messages/libgm connector in the target architecture.

## Server modules

The server is one Go process/image with small domain modules.

### `payments`

Owns:

- payment creation;
- merchant context (`name`, event `external_id`, metadata);
- product status transitions;
- operator edits;
- immutable payment history integration.

`name` is the merchant-supplied person/payee identifier. `external_id` is the merchant/event ID and is not unique.

### `profiles`

Owns Paytm/Kotak collection profiles and one active profile for **new** payments.

Each created payment snapshots:

```text
collection_profile_id
upi_id_snapshot
payee_name_snapshot
```

Switching the active profile never changes existing payments or old matching eligibility.

### `allocator`

Owns bounded randomized payable-amount allocation.

Responsibilities:

- build candidate values from requested amount through configured max adjustment;
- never use `.00`;
- remove active profile+amount reservations;
- prefer free values outside the soft recent-use horizon;
- cryptographically select a candidate;
- create payment + active reservation atomically inside one SQLite write transaction;
- return capacity failure rather than exceeding the configured cap.

### `relay`

Owns:

- short-lived QR pairing sessions;
- enrolled Android public key;
- signed request verification;
- one active relay-device policy for v4.0;
- heartbeat/device health;
- event deduplication.

### `observations`

Parses signed notification snapshots into source-neutral payment observations.

```text
id
relay_event_id
source
collection_profile_id
amount_paise
payer_name?
payer_upi_id?
occurred_at
occurred_at_source
notification_posted_at
server_received_at
matched_payment_id?
match_result
```

UTR/RRN is not required.

### `matching`

Matches normalized observations using:

```text
inferred profile
+ exact payable amount
+ historical reservation/lifecycle
+ trustworthy occurrence time
+ unique relay-event identity
```

The current active profile is not used for matching.

### `webhooks`

Durable SQLite outbox.

Externally meaningful payment transition and webhook row are committed in one database transaction. Network delivery occurs after commit.

### `admin`

Owns:

- password-only singleton operator session;
- Overview/Payments/Activity/Settings APIs;
- direct payment correction;
- profile switch;
- device pairing/replacement/revoke;
- webhook and operational settings.

### `storage`

Small explicit repository/query layer using `database/sql` + `modernc.org/sqlite`.

No PocketBase Records or PocketBase lifecycle hooks appear in the v4 domain model.

## Collection profiles

Initial profiles:

```text
paytm
kotak
```

Representative fields:

```text
id
label
upi_id
payee_name
parser
enabled
active
created_at
updated_at
```

The server enforces one active enabled profile.

Creation flow:

```text
POST /v1/payments
       |
       v
validate amount/name/event context
       |
       v
BEGIN IMMEDIATE
       |
       +--> resolve active profile
       +--> release due amount reservations
       +--> choose random free `.01…₹N.99` amount in base bucket
       +--> only if base bucket exhausted, choose random free `₹(N+1).01…₹(N+1).99`
       +--> insert payment
       +--> insert reservation
       +--> insert idempotency result
       |
     COMMIT
       |
       v
build canonical UPI URI from payment snapshot
```

## Profile-scoped amount ownership

A payable amount may exist simultaneously on different profiles:

```text
Paytm -> ₹100.37
Kotak -> ₹100.37
```

That is safe because notification source/parser identifies the collection profile.

Within one profile, only one unreleased reservation may own an exact payable amount.

The uniqueness is enforced by SQLite, not just by Go queries.

## Overlapping requested ranges

Different requested amounts can still produce overlapping final candidate values **after overflow is used**.

Example:

```text
₹100 request: ₹101.37 is possible only after every free ₹100.xx value is unavailable
₹101 request: ₹101.37 is a normal base-bucket candidate
```

Therefore reservation uniqueness is based on:

```text
(collection_profile_id, payable_amount_paise)
```

not requested amount.

## Android/server responsibility split

### Android

- package allowlist;
- cheap generic decimal-money prefilter;
- capture notification package/key/text/post time;
- stable local event ID;
- local durable retry queue;
- ECDSA request signing;
- heartbeat/background survival.

### Server

- source-specific incoming-credit parsing;
- profile inference;
- authoritative amount parse;
- timestamp confidence;
- payer extraction;
- matching;
- payment mutation/history/webhook scheduling.

The server never trusts Android `amount_hint` as payment authority.

## PayGate Frontend boundary

The separate frontend's v4 job is only:

1. send amount + person `name` + event `external_id` through its backend;
2. receive PayGate payment response;
3. render `upi_uri` as QR;
4. prominently show the exact payable amount/adjustment;
5. poll status;
6. render paid/expired/cancelled result.

It must not:

- fetch/profile-select Paytm/Kotak;
- derive UPI destination itself;
- infer verification source;
- match notifications;
- depend on UTR/RRN.

## Deployment

```text
Oracle/Docker host
  PayGate: 1 replica, stop-first
  local paygate.db + WAL/SHM
  completed-backup exporter

Android phone
  one PayGate APK
```

Production invariant remains one PayGate process owning the live SQLite database.

## What disappears from final v4

- merchant/browser profile selector;
- `/api/payment-accounts` decision surface;
- server-side libgm Google Messages runtime;
- Google session cookies/reauth pairing;
- separate Relay product identity;
- PocketBase runtime/domain records;
- UTR/RRN matching requirement;
- Manual Review/Reconciliation as primary product workflows;
- server-rendered QR images;
- hosted checkout requirement.

## Future-source extension rule

Adding GPay/Slice later should require only:

1. allowlist a new Android package if necessary;
2. add server parser + sanitized fixtures;
3. map parser to a collection profile/source;
4. pass the same observation/matching pipeline.

It must not require changing merchant payment creation or Android/server ownership boundaries.