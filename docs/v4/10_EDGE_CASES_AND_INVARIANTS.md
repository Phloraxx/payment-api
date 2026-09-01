# 10 — Edge Cases and Non-Negotiable Invariants

This document is the adversarial checklist for PayGate v4. If implementation behavior is unclear in an edge case, prefer failing closed over guessing that money belongs to a payment.

## 1. Merchant identity/context

### Same person pays twice

Two legitimate requests may contain:

```text
name = Sourav P Bijoy
external_id = evt_123
```

They are still distinct PayGate payments when created with different idempotency keys.

Neither `name` nor event `external_id` is unique.

### Many people pay for the same event

Expected normal case:

```text
Sourav P Bijoy   -> external_id evt_123
Rahul Kumar      -> external_id evt_123
Anu Thomas       -> external_id evt_123
```

SQLite must allow all three.

### Same name across different people

Names are not identity-proof. Duplicate names are valid and matching never uses `name`.

### Actual payer differs from merchant name

Example:

```text
name       = Student A
payer_name = Parent B
```

Do not overwrite merchant `name` with notification payer identity.

### Event ID changes after creation

Operator may correct `external_id` because it is context, but history records old/new values. It does not affect amount reservation or matching.

### Merchant retries creation

Same `Idempotency-Key` + same normalized request returns original payment.

Same key + changed amount/name/event/metadata returns conflict.

## 2. Amount allocation

### No `.00`

Automated PayGate payable amounts always have non-zero paise.

The Android cheap filter relies on this property, so allocator and prefilter tests must be linked.

### Random means cryptographic, not pseudo-random seed/time

Use Go `crypto/rand` to select among free values inside the **current lowest bucket**.

Do not seed `math/rand` with current time and call it sufficient.

### Base bucket must be exhausted before overflow

For a ₹100 request:

```text
first bucket:  ₹100.01…₹100.99
overflow:      ₹101.01…₹101.99
```

If even one eligible ₹100.xx value is free, PayGate must not allocate ₹101.xx.

### Random selection must still be unique

Randomness does not enforce ownership. SQLite active-reservation uniqueness does.

### Adjacent base amounts overlap only when overflow is used

A ₹100 request can reach ₹101.xx only when its ₹100.xx bucket is exhausted. A ₹101 request normally uses ₹101.xx directly. Final `(profile, payable_amount)` uniqueness is therefore still authoritative.

### Pool pressure

Within the current bucket, when preferred old/unrecent candidates are exhausted:

1. include recently released but currently free candidates in **that same bucket**;
2. randomize among them;
3. only move to the next rupee when the current bucket has no free candidate at all;
4. fail only when both configured buckets are actively unavailable.

Soft recent-use avoidance must never create fake bucket exhaustion or force premature overflow.

### Config cap changed downward

Existing payments/reservations above the new max remain valid and matchable.

The new cap applies only to future allocation.

### Config cap changed upward

New candidate space becomes available for future allocations. No existing payable amount changes.

### Concurrent creates

Two requests arriving simultaneously cannot both reserve the same value.

The write transaction + DB unique index is authoritative.

If insertion sees a uniqueness violation because another transaction won a candidate, safely retry candidate selection inside the bounded transaction/retry policy; never return duplicate ownership.

### Crash during allocation

Payment row, reservation and idempotency result are in one transaction.

After crash there must be either:

```text
all committed
```

or

```text
none committed
```

No orphan active reservation and no payment without its reservation.

## 3. Lifecycle/reuse

### Customer scans at 4:59 and pays at 5:10

Grace window exists specifically for this boundary. If trusted occurrence time is within grace, payment becomes paid.

### Payment notification delivered at 12 minutes for a payment at 3 minutes

If original notification/transaction occurrence is trustworthy and belongs to the old reservation, delayed relay delivery can still resolve the old payment.

### Notification arrives after amount was reused

If historical time uniquely identifies the old reservation, attach to old payment.

If time cannot distinguish old versus new use, no auto-match.

### Delayed SMS generated late by bank

This is harder than delayed relay delivery because Android `postTime` may reflect late bank delivery rather than transaction time.

If message text contains a reliable transaction timestamp, use it.

If it does not and the amount has since been reused, fail closed on ambiguity. Randomized allocation lowers collision probability but does not justify guessing.

### Cancelled payment paid anyway

Cancellation stops intended customer use but customer may already have scanned the QR.

Keep its amount reservation through normal reuse boundary.

If trusted occurrence time proves money arrived before/around cancellation according to defined policy, record appropriately; do not reuse the amount early.

### Paid payment receives duplicate/corroborating notifications

Exact retry of the same signed source event -> return the prior result idempotently.

A different source event can describe the same underlying credit. Example: Kotak SMS + GPay, or later Kotak SMS + Amazon Pay. Do not dedupe these on Android and do not hash text in an attempt to prove they are identical.

The first safe observation is `matched`. A later independent observation that resolves to the same historical payment reservation is `corroborated`. It attaches to the same payment and may fill missing payer information, but it creates **no second payment transition and no second merchant webhook**.

If the payable amount has been reused and the later source has only low-confidence timing, fail closed as `ambiguous`; never call it corroboration merely because the amount is equal.

## 4. Collection profile

### Switch Paytm -> Kotak with Paytm sessions in flight

New creates use Kotak.

Existing Paytm payment rows/reservations remain Paytm and continue matching from Paytm notifications.

### Switch back quickly

Profile switching must be atomic and history recorded. Existing payment snapshots remain unchanged regardless of switch frequency.

### Disable inactive profile with old payments

Disabled means no new creates.

Server parsers/matcher must still understand the profile long enough to process historical/in-flight payments.

### Destination UPI ID edited

New payments snapshot new destination.

Old payments retain old UPI snapshot and generated URI semantics.

Never rewrite old payment UPI destination after creation.

### No active ready profile

Payment creation fails clearly/temporarily unavailable.

Do not invent a fallback profile silently.

## 5. Android notification edge cases

### Paytm decimal debit

Phone may relay because cheap filter sees decimal money.

Server Paytm parser rejects it as non-incoming credit. No payment mutation.

### Google Messages balance message with decimals

Cheap filter may pass. Kotak parser must positively recognize incoming Kotak credit; otherwise ignore/unmatched.

### Multiple amounts in one notification

Example may contain transaction amount and balance.

Phone does not choose authority. Server source parser must know which token is the credited amount. If parser cannot determine reliably, no auto-match.

### Currency comma separators

Support normal Indian/standard formats such as:

```text
₹1,100.37
INR 10,000.42
```

Normalize separators before integer-paise conversion.

### Unicode/format variation

Notification text may use non-breaking spaces, line breaks or localized symbols. Normalize whitespace conservatively before parser matching while retaining bounded original text for short diagnostics.

### Group summary

Never relay Android group-summary notifications as money events.

### Notification rescan

Listener reconnect/active-notification rescan must reproduce same local event identity for unchanged relevant content.

### Notification update

If same Android key changes amount/payer materially, version as a new immutable event fingerprint rather than mutating an already delivered event.

### Phone reboot

Pending local events survive. Notification listener/foreground runtime/watchdog recover. No old pre-enrollment notification is newly relayed as fresh.

### Phone offline

Queue locally, preserve original notification post time, deliver later.

### Queue grows unexpectedly

Bound local DB retention/size. Health shows backlog. Do not drop newest payment events silently.

### Android notification content unavailable/truncated

Do not guess from amount hint alone. Event remains ignored/unmatched and parity gate fails if this systematically prevents Kotak detection.

## 6. Device identity/pairing

### QR scanned twice

Pairing token is single use. First successful enrollment consumes it; second fails.

### QR screenshot used after expiry

Reject expired token.

### Pairing request interrupted

Token consumption and device enrollment are atomic so failure cannot consume token without a device record or create device without consuming token.

### Replace phone

New phone must enroll before old phone is disabled. Final state has exactly one active relay device.

### Old phone comes online later

Its signed events are rejected after revoke/disable.

### App reinstalled

A reinstall may lose Keystore/app identity. Treat as a new phone and use Replace/Connect; do not silently reuse server identity based only on device model/name.

### Operator session expires

Relay ECDSA device credential is independent. Background payment monitoring continues.

## 7. Timestamp/clock edge cases

### Device clock slightly skewed

Request signing has a bounded tolerance. Health should expose actionable clock/signature failures.

### Notification timestamp in future

Do not trust an implausible future `postTime`. Mark timestamp confidence low/reject according to policy.

### Source text time has minute precision only

Represent an interval or conservative confidence rather than inventing seconds if exact-second precision matters to reservation overlap.

### Server receive time

Useful for ingestion ordering and diagnostics; not equivalent to transaction occurrence.

### DST/timezone

Persist server/domain times as UTC Unix milliseconds. Parse displayed bank times using explicit source timezone rules; do not let host locale silently reinterpret them.

## 8. Matching ambiguity

### Exactly one safe candidate

Auto-match.

### Zero candidates

Store unmatched Activity. No payment mutation.

### Multiple candidates

Store ambiguous Activity. No payment mutation.

### Low-confidence timestamp after amount reuse

Treat as ambiguous even if one *current* payment has that amount. Historical reuse must be considered.

### Notification amount equals current payment but predates creation

Cannot pay that payment.

### Notification source profile differs

Paytm observation cannot pay a Kotak reservation even if amount is identical.

## 9. Operator edits

### Admin marks pending -> paid manually

Append immutable history and enqueue one `payment.paid` webhook atomically.

### Admin edits payer name only

Append history; use `payment.updated` only if merchant integrations need that event. Do not emit `payment.paid` again.

### Admin tries to change payable amount

Reject. Cancel/create replacement.

### Admin tries to change requested amount

Reject after creation. Cancel/create replacement.

### Admin changes event ID/name

Allowed as context correction; append history.

### Admin reverses paid -> another state

This is financially dangerous. If v4 allows it, require explicit confirmation and document webhook semantics. Prefer a constrained status-transition table rather than arbitrary free text.

## 10. Webhook edge cases

### Payment committed, process dies before delivery

Outbox row was committed in same transaction, so worker sends it after restart.

### Endpoint timeout after consumer processed event

Retry may deliver duplicate. Consumer dedupes by webhook event ID.

### 4xx endpoint response

Record error. Retry policy should distinguish likely permanent configuration errors from transient rate-limit responses.

### Historical exhausted delivery

Do not blindly bulk replay. Operator retries one selected event after understanding downstream consequences.

### Webhook secret rotation

Define overlap/rotation semantics so in-flight queued events can still be verified or are re-signed with current secret deliberately. Do not silently make every queued event unverifiable.

## 11. SQLite concurrency/storage edge cases

### Second PayGate process

Process lock fails before live DB ownership. Second process exits closed.

### Short writer contention

Busy timeout can wait briefly.

### Long writer contention

Return safe retryable service error; log/alert operationally. Do not wait forever.

### Read handler accidentally performs network call inside transaction

Code review/test pattern must prevent long DB transactions around network I/O.

### Foreign key disabled on one pooled connection

Startup/connection configuration must set it for every connection. Tests should intentionally open pooled connections and prove enforcement.

### Typo in PRAGMA

SQLite may silently ignore unknown PRAGMA names. Startup reads back critical values and fails if unexpected.

### WAL grows

Monitor size/checkpoint health. Use in-process/passive checkpoint strategy if necessary rather than spawning a second PayGate runtime.

### Disk fills

Fail writes clearly and stop pretending payment state can be safely persisted. Health becomes critical. Never acknowledge a payment mutation that failed to commit.

### Filesystem is network-mounted

Startup/config should reject/document unsupported live DB path. WAL requires same-host semantics.

### Database file permissions wrong

Fail startup rather than creating an unexpected second DB somewhere else.

### Corruption/integrity failure

Do not auto-repair production destructively. Stop affected writes, preserve files, restore from verified backup according to runbook.

## 12. Backup/restore edge cases

### Backup while writes continue

Use SQLite Online Backup API; resulting completed file must pass integrity/foreign-key checks.

### Backup destination fails/disk full

Source database remains untouched. Failed staging artifact is never renamed/published as a valid backup.

### Host exporter runs while internal backup is incomplete

Exporter only recognizes atomically finalized completed artifact names/manifests.

### Restore drill accidentally points at live DB

Guardrails/path checks prevent it. Drills always use isolated paths/container.

### Backup contains secrets

Database backups include hashed credentials/session data and payment information. Treat backup files as sensitive; restrict permissions and encrypt remote/off-host storage where appropriate.

## 13. Migration edge cases

### V3 has statuses not present in v4

Migrator maps explicitly and reports counts. Never silently coerce unknown status.

### V3 has duplicate/odd external IDs

Allowed in v4 event-ID semantics.

### V3 requested/payable relationships outside new cap

Historical payment imports remain valid; new allocator cap is not retroactively applied.

### V3 current 24h reservations

Drain in-flight state before cutover where possible. Migrator must not accidentally turn an old active amount into a new free random candidate during cutover.

### V3 backup contains non-empty `data.db-wal`

Abort before creating the destination. A data-only extraction could otherwise omit uncheckpointed transactions. Migration accepts a missing WAL or an explicitly zero-length WAL only.

### Old merchant retries a v3 idempotency key after cutover

Migrated v3 keys are cutover tombstones bound to the historical payment. They deliberately do not pretend to be replayable v4 request fingerprints. A stale post-cutover retry fails closed with idempotency conflict and cannot create a second payment.

### Historical operational rows

Old webhook deliveries/reviews/SMS/relay/alert rows remain in the verified archive and are not activated in v4. This prevents a migration from replaying historical side effects.

### Migration interrupted

Destination new DB is disposable/recreated. Source v3 backup is never modified.

### New v4 writes after cutover

After meaningful v4 payments exist, rollback cannot simply restore old v3 database without reconciling/exporting the new writes.

## 14. Security edge cases

### Admin password brute force

Rate-limit/throttle password-only login and log security events without password content.

### Merchant API key leaked

Key can be rotated/revoked without changing admin password/device key/webhook secret.

### Android device key leaked/compromised

Revoke device and Replace phone. Merchant/admin credentials unaffected.

### Pairing token leaked

Short TTL + one use limits value; token cannot authenticate admin or merchant APIs.

### Notification payload contains HTML/script

Always render as escaped text. Never inject raw notification content into dashboard HTML.

## 15. Non-negotiable invariants summary

1. Only PayGate server changes authoritative payment state.
2. `name` and event `external_id` never match money and are never unique identity assumptions.
3. Merchant cannot choose collection profile.
4. Payment snapshots destination/profile at creation.
5. Automated payable amount is bounded, non-`.00`, randomized within the lowest available bucket, and actively unique within profile.
6. Randomness supplements; it does not replace hard reservation and fail-closed timestamp rules.
7. Android amount hint is never trusted as authoritative.
8. One signed relay event cannot produce duplicate payment transitions.
9. Reused amount + ambiguous time never auto-matches.
10. Payment state/history/outbox mutations are atomic SQLite transactions.
11. One PayGate process owns live SQLite.
12. No second maintenance/backup PayGate process opens live DB.
13. UTR/RRN is not required.
14. GPay/Amazon Pay/Slice do not auto-match in v4.0.
15. Multiple independent notification sources can corroborate one payment but can never create multiple `payment.paid` transitions/webhooks.
16. PocketBase/libgm are absent from final v4 runtime.
17. Every risky operator correction is visible in immutable history.
18. If PayGate cannot prove which payment owns money, it records Activity and does not guess.