# 03 — Payment Lifecycle and Matching

## Core rule

PayGate matches a payment using a server-reserved payable amount, collection profile, trusted time window and unique relay-event identity.

No UTR/RRN is required.

The merchant-facing `name` and `external_id` are context only and are **never matching keys**:

- `name` = merchant-supplied human/payee identifier (for example the registrant/person name);
- `external_id` = merchant/event identifier (for example the event ID) and may repeat across many payments;
- `payer_name` = actual sender/payer name observed later from a payment notification and may differ from `name`.

## Amount model

Each payment stores integer paise values:

```text
requested_amount_paise
payable_amount_paise
adjustment_paise
```

Normal example:

```text
requested  = ₹100.00
payable    = ₹100.37
adjustment = ₹0.37
```

A `₹101.xx` payable amount is used only after every free `₹100.01…₹100.99` candidate is unavailable.

The frontend must say clearly that the customer must pay the **exact PayGate amount** and display the adjustment whenever non-zero.

## Ordered random candidate buckets

For a whole-rupee request, v4 uses up to two ordered 99-value buckets by default.

For ₹100:

```text
Bucket 0 (always try first): ₹100.01 … ₹100.99
Bucket 1 (overflow only):    ₹101.01 … ₹101.99
```

Rules:

- `.00` is never generated;
- **never choose from Bucket 1 while any free candidate exists in Bucket 0**;
- randomness is used only to choose among free candidates inside the current bucket;
- final payable values are unique among currently reserved values on the same collection profile;
- adjacent requested amounts may still overlap once overflow is used, so uniqueness is enforced on the **final payable value**, not requested amount;
- default maximum adjustment is `₹1.99`; PayGate never silently keeps increasing the amount without limit.

## Randomized bucket allocator

Do **not** allocate `.01`, `.02`, `.03` sequentially, but also do **not** randomize across both rupees at once.

The goal is:

1. keep the customer inside the requested rupee whenever capacity exists;
2. make suffix reuse unpredictable inside that rupee;
3. use the next rupee only as overflow capacity.

Pseudo-flow:

```text
BEGIN IMMEDIATE
  release reservations whose hard reuse_after <= now

  for bucket in [requested_rupee, requested_rupee + 1]:
      free = all .01… .99 values in this bucket
             minus currently reserved values on this profile

      if free is empty:
          continue to next bucket

      preferred = free values not used in the soft recent-use horizon
      pool = preferred if preferred is non-empty else free
      cryptographically choose one candidate from pool
      create payment + active amount reservation atomically
      COMMIT
      return candidate

  ROLLBACK
  return PAYMENT_CAPACITY_TEMPORARILY_UNAVAILABLE
```

Use Go `crypto/rand`, not `math/rand`, for the candidate choice.

### Why retain a short hard quarantine if allocation is random?

Randomization reduces reuse probability; it does not make reuse impossible.

Therefore v4 keeps a short deterministic reservation window and combines it with random allocation. The two mechanisms protect different failure modes:

- hard reservation prevents immediate reuse while a normal/delayed notification is expected;
- random allocation makes reuse after release unpredictable;
- soft recent-use avoidance lowers the chance of quickly selecting the same released amount without blocking capacity.

### Soft recent-use avoidance

This is **not** another hard quarantine.

A released amount remains available if capacity is tight, but while many other free values exist PayGate should prefer values that have not been used recently.

Recommended initial soft horizon: configurable, default around a few hours. It affects selection preference only; it never makes creation fail by itself.

This gives most of the safety benefit of a long quarantine without holding the entire decimal pool for 24 hours.

## Database-enforced reservation

Use a dedicated `amount_reservations` table with a uniqueness constraint for active reservations:

```text
UNIQUE(collection_profile_id, payable_amount_paise)
WHERE released_at IS NULL
```

Allocation and payment creation happen inside one SQLite write transaction. If two requests arrive concurrently, only one can own a candidate.

Do not rely only on an application-level `SELECT` followed by `INSERT` outside one transaction.

## 5 / 5 / 5 hard lifecycle

Initial v4 lifecycle:

```text
created_at
expires_at   = created_at + 5m
grace_until  = created_at + 10m
reuse_after  = created_at + 15m
```

### Active — 0–5 minutes

- payment status is `pending`;
- customer is expected to pay;
- QR/UPI instruction is valid;
- matching observation resolves payment as `paid`.

### Grace — 5–10 minutes

- frontend stops encouraging a new payment and says the normal window ended;
- PayGate still accepts a payment whose trustworthy occurrence time falls inside this phase;
- a successful grace match is simply `paid`, optionally with an internal `paid_after_expiry` history flag.

### Quarantine — 10–15 minutes

- customer should not initiate payment using the old QR;
- the payable value remains reserved;
- a delayed relay delivery may still be attached to the old payment only when its occurrence time is trustworthy and proves it belongs to that payment;
- otherwise the observation stays unmatched in Activity rather than guessing.

### Reusable — 15+ minutes

The hard reservation may be released and the value returns to the random free pool.

The payment row and historical reservation are retained, so PayGate can reason about previous use when evaluating a delayed observation.

## Timestamp confidence

The server must distinguish **when money likely moved** from **when PayGate received the relay request**.

Normalized observations should record:

```text
occurred_at
occurred_at_source
notification_posted_at
server_received_at
```

Suggested `occurred_at_source` values:

```text
message_text
notification_posted
server_received
```

Preference:

1. source-specific transaction time parsed from trusted notification/message text;
2. Android notification `postTime`;
3. server receive time only as a last-resort diagnostic value.

`server_received` time alone must not be sufficient to auto-match a delayed event after an amount has been reused.

## Reuse/collision safety rule

If a payable value has only one historical reservation compatible with the trusted occurrence time, it may match that reservation.

If the same value has been reused and the observation time is missing, implausible or too low-confidence to distinguish the reservations:

```text
DO NOT AUTO-MATCH
```

Store the observation as unmatched/ambiguous Activity. The operator can correct the payment directly if necessary.

This is the final defense against extremely delayed SMS/notification delivery.

## Auto-match algorithm

For normalized observation `O`:

```text
1. dedupe signed relay event
2. parse source and infer collection profile P
3. validate incoming-credit semantics
4. validate amount > 0 and paise != 00
5. derive occurred_at + confidence/source
6. find historical payments on P with exact payable amount
7. restrict candidates to payments whose lifecycle can contain O.occurred_at
8. if exactly one candidate is safe and not yet paid:
      mark paid
      attach observation as `matched`
      copy payer enrichment when present
      append payment history
      enqueue one payment.paid webhook in same transaction
9. if exactly one safe candidate is already paid and already has a confirming observation:
      attach this independent observation as `corroborated`
      optionally enrich missing payer fields
      do not create another payment transition/history/webhook
10. if zero candidates:
      save unmatched Activity
11. if multiple/uncertain candidates:
      fail closed; save ambiguous Activity
```

The **currently active collection profile is irrelevant to matching**. A Paytm observation can still pay an older Paytm payment after the operator has switched new payment creation to Kotak.

## Relay amount hint is not trusted

Android may send an `amount_hint` because it already found a decimal-looking token for its cheap prefilter.

The server parser must re-parse and validate the notification text. `amount_hint` is never authoritative for marking a payment paid.

## Payment statuses

Public/product statuses remain:

```text
pending
paid
expired
cancelled
```

Do not expose `late`, `review`, `reconciled`, `evidence_pending`, etc. as product states.

## Expiry behavior

A customer-facing timer ends at `expires_at` (5m).

The server may keep the payment internally eligible through `grace_until` (10m), then transitions unresolved `pending` payments to `expired`.

The amount reservation survives until `reuse_after` (15m).

## Cancellation

Cancellation stops customer use immediately but does **not** free the amount immediately.

Keep the reservation until at least its original `reuse_after`, because a customer may already have scanned the QR or a delayed notification may still arrive.

## Profile switch

Switching Paytm ↔ Kotak affects new payment creation only.

Existing payments retain:

- collection profile snapshot;
- destination UPI ID snapshot;
- payee-name snapshot;
- reservation.

Disabled/inactive profiles must remain parseable/matchable for historical in-flight payments until their lifecycle has settled.

## Payer information

When available, store separately from merchant-supplied `name`:

```text
payer_name
payer_upi_id
paid_at
```

Never overwrite `name` automatically with the notification payer name. A parent/friend may pay for the named attendee.

## No UTR/RRN requirement

UTR/RRN is absent from the v4 matching contract and UI requirements.

If a future parser happens to expose a bank reference, it may be stored as optional opaque metadata, but matching, creation, webhooks and admin correction must not depend on it.

## Capacity behavior

If no free candidate exists inside the configured adjustment cap, fail creation with a retryable capacity error.

Do not:

- reuse an active reservation;
- allocate `.00`;
- increase the adjustment above the configured cap automatically;
- fall back to another collection profile without an explicit product decision.

## Required edge-case tests

### Allocation/concurrency

- random candidate is always within the bounded pool;
- `.00` is never generated;
- repeated allocations are not deterministic sequential increments;
- active reservation uniqueness is database-enforced;
- two concurrent create requests cannot receive the same profile+payable value;
- adjacent requested amounts cannot collide;
- soft recent-use avoidance prefers older values but never causes artificial exhaustion;
- hard capacity exhaustion returns a clean retryable error;
- crash/rollback between candidate selection and commit leaves no orphan reservation.

### Identity/context

- duplicate `name` values are allowed;
- duplicate `external_id` event IDs are allowed;
- neither `name` nor `external_id` participates in matching;
- idempotency key replay returns the original payment;
- same idempotency key with changed request conflicts.

### Lifecycle/time

- active/grace/quarantine boundaries are exact;
- cancellation does not release the amount early;
- delayed relay delivery with original notification post time matches the old reservation;
- reused amount + low-confidence delayed timestamp fails closed;
- future/skewed device timestamps are clamped/rejected according to policy;
- server restart does not lose reservation state.

### Source/profile

- Paytm observation matches only Paytm-profile reservations;
- Kotak observation matches only Kotak-profile reservations;
- profile switch does not alter existing payments;
- unrelated decimal Paytm/Google Messages notification becomes ignored/unmatched, never paid;
- duplicate relay event is idempotent;
- two independent observations of the same already-paid reservation produce `matched` then `corroborated`, with one payment webhook total;
- a delayed duplicate after amount reuse with low-confidence time fails ambiguous rather than attaching to the new payment;
- unsupported GPay/Amazon Pay/Slice cannot auto-match in v4.0; their future parser rollout must map the source to a collection profile confidently.

### Operator/webhook

- operator correction creates immutable history;
- externally meaningful state transition enqueues exactly one webhook event;
- webhook failure cannot roll back an already committed payment.