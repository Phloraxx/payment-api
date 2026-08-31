# 09 — Research Notes

This file records why the v4 plan changed so implementation does not have to rediscover the same conclusions.

## Current-code findings

### Current frontend chooses payment account

`payment-frontend/src/server/paygate.ts` currently sends `paymentAccount` to PayGate.

V4 removes that merchant/frontend responsibility.

### Current API accepts payment account

`payment-api/internal/api/api.go` contains `paymentAccount` in the create body and exposes `/api/payment-accounts`.

V4 replaces that public decision with one server-owned active collection profile for new payments.

### Current API already builds canonical UPI URI

`payment-api/internal/payments/service.go` builds `upi://pay?...` from account destination + payable amount.

That responsibility stays server-side.

### Current frontend already renders QR

`payment-frontend/src/client/pages/PaymentPage.tsx` uses `QRCodeCanvas` on the server-returned URI.

That presentation boundary is retained. V4 does not need server QR image endpoints or hosted checkout.

### Current Android already observes required notification surfaces

Current relay recognizes:

```text
com.paytm.business
com.google.android.apps.nbu.paisa.user
com.google.android.apps.nbu.paisa.merchant
com.google.android.apps.messaging
```

V4.0 narrows matching rollout to:

```text
Paytm Business
Google Messages -> Kotak
```

GPay/Slice are deferred.

### Current server routes non-Paytm Android events observation-only

`internal/androidrelay/service.go` currently sends only Paytm notification events into Paytm matching; other allowlisted Android notifications are `observed_only`.

Promoting Kotak-via-Google-Messages into the unified server parser/matcher is therefore an extension of the existing transport, not a new relay.

### Current server duplicates Google Messages transport through libgm

`internal/gmessages/manager.go` handles Google account/session pairing, cookies, reconnect, reauth and message ingestion.

Once Android Google Messages notification parity is proven for Kotak, this duplicate server-side transport can be removed.

### Current bank SMS matching requires RRN, Paytm does not

Current SMS matching requires amount + RRN. Paytm notification matching already proves idempotency can instead use unique notification evidence identity.

V4 therefore removes UTR/RRN from the required matching contract.

### Current amount allocator is already partially randomized

Current `payments.Service` picks a cryptographically random suffix start using Go `crypto/rand`, then scans the 99 suffixes for a free candidate.

The first v4 planning draft accidentally changed this into a deterministic smallest-free allocator. That was the wrong direction.

V4 restores and generalizes the useful property:

```text
cryptographically random choice among bounded free values
```

while expanding the pool beyond one rupee and adding database-enforced active reservations.

### Current default quarantine is 24h

`.env.example` currently has:

```text
PAYMENT_TTL=5m
PAYMENT_QUARANTINE=24h
```

V4 targets a short 5m active + 5m grace + 5m hard quarantine, then random reuse with soft recent-use avoidance.

This must be validated specifically against delayed phone notifications before production.

### Existing relay authentication is strong

Current Android relay service validates ECDSA signatures against an enrolled device public key and binds identity to the public-key fingerprint.

V4 keeps this model and changes only enrollment UX to one-time QR/App-Link pairing.

## Merchant identity semantics discovered during design review

The merchant `name` field is not an event label. It identifies the person/payee/subject attached to this payment.

`external_id` is the merchant's event ID.

That means `external_id` is intentionally **non-unique**: one event can have many PayGate payments.

Consequences:

- never make `external_id` UNIQUE in SQLite;
- never use it as idempotency identity;
- never use it for matching;
- merchant/test frontend stores returned PayGate payment ID per registration/payment;
- `Idempotency-Key` is the duplicate-create boundary;
- actual notification-derived payer identity remains separate from merchant `name`.

## Randomized amount-allocation research/conclusions

### Why sequential amounts are undesirable

Sequential `.01`, `.02`, `.03` allocation makes reuse predictable and repeatedly concentrates traffic on the same low suffixes after reservations expire.

That is unnecessary because PayGate does not need human-predictable decimals.

### Why randomization is not enough by itself

Random allocation only lowers the probability that a newly created payment reuses the exact amount belonging to a very delayed old notification.

It cannot make that probability zero.

Therefore final design combines:

1. 5m active;
2. 5m grace;
3. 5m hard no-reuse quarantine;
4. cryptographic random candidate selection after release;
5. soft recent-use avoidance when alternatives exist;
6. occurrence-time reasoning against historical reservations;
7. fail-closed behavior when reused amount + timestamp ambiguity cannot be resolved.

### Why soft recent-use avoidance is useful

A long 24h quarantine reduces usable amount capacity whether or not old notifications actually exist.

A soft horizon changes only **preference**:

- old/unrecent free values are preferred;
- recently released values remain usable if the pool is pressured.

This improves stale-collision resistance without creating artificial capacity exhaustion.

### Cross-rupee pool

With default max adjustment ₹1.99 and a whole-rupee requested amount, candidate values include 198 non-`.00` amounts.

Adjacent requested amounts overlap, so active uniqueness must be based on final profile+payable amount, not requested amount.

## Android research

### NotificationListenerService is the right primitive

Android provides `NotificationListenerService` for notification posted/removed callbacks after user-granted notification access.

https://developer.android.com/reference/android/service/notification/NotificationListenerService.html

### Direct SMS permissions add unnecessary policy complexity

Google Play heavily restricts SMS/Call Log permissions and generally expects eligible/default handlers.

https://support.google.com/googleplay/android-developer/answer/16558241

PayGate does not need to become an SMS client to observe the Google Messages notification that already exists.

### Device key storage

Keep Android Keystore for the ECDSA private key.

https://mas.owasp.org/MASTG/knowledge/android/MASVS-STORAGE/MASTG-KNOW-0047/

## UPI research

Current UPI product direction favors QR/Intent rather than legacy manual-collect flows.

PayGate's server-generated canonical UPI URI + frontend QR rendering remains aligned with that model.

Razorpay references used during planning:

- https://razorpay.com/docs/payments/payment-methods/upi/
- https://razorpay.com/docs/payments/payment-methods/upi/upi-intent/

## SQLite research

### SQLite is the database; PocketBase is a framework above it

Current PayGate effectively has:

```text
Go + PocketBase + SQLite
```

Target is:

```text
Go + database/sql + modernc.org/sqlite + SQLite
```

The current repository already depends directly on:

```text
modernc.org/sqlite v1.54.0
modernc.org/libc v1.74.1
```

So removing PocketBase does not require introducing a different database engine.

`modernc.org/sqlite` is CGO-free and exposes SQLite's online Backup API. Its v1.54.0 release upgraded the embedded SQLite engine to SQLite 3.53.3.

References:

- https://pkg.go.dev/modernc.org/sqlite
- https://gitlab.com/cznic/sqlite/-/blob/v1.54.0/driver.go

### One application server is a documented SQLite use case

SQLite says a server-side application database is appropriate when SQL is executed by the application server on the same machine as the database file, even though end users are remote.

https://www.sqlite.org/whentouse.html

That maps directly to PayGate's one-server-process architecture.

### WAL does not mean multiple writers

WAL lets readers continue while a writer is active, but SQLite still serializes writers.

https://www.sqlite.org/wal.html

That is fine for PayGate because writes are short payment/event transactions rather than long analytical jobs.

### `BEGIN IMMEDIATE` is useful for critical writes

SQLite documents that `BEGIN IMMEDIATE` acquires the write transaction up front, so contention appears before business logic has performed a read-then-write upgrade.

https://sqlite.org/lang_transaction.html

Use it for allocation/matching/profile-switch/pairing mutations while normal reads remain simple/deferred.

### Bounded busy timeout

SQLite provides a busy timeout so short lock contention can wait rather than immediately return `SQLITE_BUSY`.

https://sqlite.org/c3ref/busy_timeout.html

This is a bounded resilience mechanism, not permission to hold long write transactions.

### `synchronous=FULL` is the safer payment setting

SQLite's PRAGMA documentation explains that WAL + `synchronous=FULL` adds an extra WAL sync after each commit and provides stronger durability across power loss than NORMAL.

https://www.sqlite.org/pragma.html#pragma_synchronous

PayGate should prefer this durability unless measured production performance proves it unacceptable.

### STRICT tables reduce accidental SQLite type flexibility

SQLite supports per-table STRICT typing.

https://www.sqlite.org/stricttables.html

V4 should use STRICT domain tables plus CHECK/UNIQUE/foreign-key constraints so bugs fail at the database boundary.

### Partial unique indexes fit active state

SQLite partial indexes apply only to rows satisfying a WHERE clause.

https://www.sqlite.org/partialindex.html

Useful invariants:

```text
one active collection profile
one unreleased reservation per profile+payable amount
```

### Online Backup API is the correct live-backup primitive

SQLite's Backup API copies a consistent snapshot while source read locks are held only during page copying.

https://www.sqlite.org/backup.html

This avoids raw-copying a changing DB/WAL pair and avoids launching a second PayGate runtime against live storage.

### `PRAGMA optimize` is current planner-maintenance guidance

SQLite 3.46+ recommends `PRAGMA optimize` instead of broad manual ANALYZE scheduling.

https://www.sqlite.org/pragma.html#pragma_optimize

### SQLite PRAGMA gotcha

Unknown PRAGMA names may be silently ignored.

Therefore PayGate startup should query/verify critical effective settings rather than merely execute strings and assume success.

https://www.sqlite.org/pragma.html

## Authentication/session research

Singleton password-only operator UX can remain secure if the hidden implementation separates credentials:

- Argon2id admin password hash;
- opaque admin session token;
- merchant API key;
- Android device private key;
- webhook signing secret.

OWASP session guidance:

https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html

## Final research-derived architecture

```text
PayGate server
  -> direct SQLite
  -> server-owned active profile
  -> randomized bounded amount reservations
  -> server notification parsers
  -> timestamp-aware matching
  -> durable signed webhooks

PayGate Android
  -> Paytm + Google Messages notifications
  -> cheap decimal-money prefilter
  -> stable local event ID
  -> durable queue
  -> ECDSA signed relay
  -> operator UI
```

Excluded from v4.0:

```text
PocketBase runtime
server libgm
GPay auto-match
Slice
merchant profile selection
hosted checkout
server QR images
UTR/RRN requirement
manual-review/reconciliation products
PostgreSQL
multi-user roles
WebSockets
```