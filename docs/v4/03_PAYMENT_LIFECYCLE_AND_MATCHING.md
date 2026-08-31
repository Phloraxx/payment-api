# 03 — Payment Lifecycle and Matching

## Goals

The matching model should be easy to explain:

1. PayGate chooses one unique payable amount on the active collection profile.
2. The customer pays exactly that amount.
3. The phone reports an incoming decimal-money notification.
4. PayGate infers the profile and exact amount from the notification.
5. One eligible payment owns that profile+amount+time window, so PayGate marks it paid.

No UTR/RRN dependency is required.

## Amounts

Each payment stores two monetary values:

```text
requested_amount
payable_amount
```

Example:

```text
requested_amount = ₹100.00
payable_amount   = ₹100.37
adjustment       = ₹0.37
```

The API/UI must always display both requested and payable amount when they differ and explicitly identify the difference as the PayGate verification adjustment.

## Expand beyond one rupee's paise

The existing v3 allocator only has `.01` through `.99` for one requested rupee amount. v4 can move into the next rupee while keeping non-zero paise.

Allocation order for a ₹100 request:

```text
₹100.01 ... ₹100.99
₹101.01 ... ₹101.99
```

Never allocate `.00`.

The default v4 maximum adjustment is therefore **₹1.99**.

This gives up to 198 candidate payable values around one requested amount while bounding the customer's adjustment to less than ₹2.

The maximum adjustment should be configurable, but increasing it is an operator choice, not an automatic emergency behavior.

## Smallest-free allocator

Do not randomize the customer's adjustment unless there is a demonstrated reason.

Algorithm:

```text
profile = active profile
for candidate from requested+₹0.01 upward in ₹0.01 steps:
    skip every candidate ending in .00
    stop at requested+configured_max_adjustment
    if candidate is not reserved on this profile:
        reserve it
        return candidate
return PAYMENT_CAPACITY_TEMPORARILY_UNAVAILABLE
```

This minimizes the extra amount during normal load.

### Global uniqueness is profile-scoped

An amount is unavailable when another payment on the **same collection profile** has that payable amount and `reuse_after > now`.

The same payable amount may exist on Paytm and Kotak simultaneously because source/profile inference distinguishes them.

### Overlap between requested amounts is safe

Example:

```text
Payment A requested ₹100 -> candidate ₹101.01
Payment B requested ₹101 -> ₹101.01 is already reserved, so it receives another free candidate
```

The allocator checks the final payable amount globally within the profile, not just within the same requested amount.

## 5 / 5 / 5 lifecycle

Every payment gets three server timestamps:

```text
created_at
expires_at   = created_at + 5 minutes
grace_until  = created_at + 10 minutes
reuse_after  = created_at + 15 minutes
```

### Phase 1 — Active (0–5 min)

- status: `pending`
- frontend shows QR/payment instructions
- customer is expected to pay
- valid matching observation marks payment `paid`

### Phase 2 — Grace (5–10 min)

- frontend should say the normal payment window has ended and discourage a new payment
- PayGate still accepts an observation whose transaction/notification occurrence is in this grace period
- this handles a customer who was already completing the UPI flow as the timer ended

A grace-period match still resolves the payment as `paid`. `paid_after_expiry=true` can be retained internally if useful; it does not need to become a public status named `late`.

### Phase 3 — Quarantine (10–15 min)

- customer must not initiate payment using this old session
- a new PayGate payment cannot reuse the payable amount
- PayGate may still accept a **delayed relay delivery** only when the observation has a trustworthy occurrence time at or before `grace_until`
- if occurrence time is not trustworthy, fail closed and show the observation as unmatched Activity

### Reusable (15+ min)

The payable amount can return to the allocation pool.

An observation received after reuse must never be auto-matched to an old reservation unless its occurrence timestamp proves unambiguously which reservation it belongs to. The default simple behavior is to leave such an event unmatched rather than guess.

## Payment statuses

Public/product statuses:

```text
pending
paid
expired
cancelled
```

Avoid exposing `late`, `review`, `reconciled`, `evidence_pending`, etc. as product states.

Internal flags/history can record how the status was reached.

### Expiry

A payment becomes `expired` after `grace_until` if still pending.

The QR/frontend can visually end at `expires_at` (5 min), while the server waits until the grace window ends before declaring final expiry.

This distinction preserves a customer-friendly timer without losing payments that complete at the boundary.

## Observation eligibility

A normalized observation must have:

- supported/inferred collection profile
- exact positive amount with non-zero paise
- unique source event ID
- occurrence/notification time
- incoming-credit semantics confirmed by the source parser

Payer name and UPI ID are useful enrichment but are not required for matching.

## Auto-match algorithm

Given observation `O`:

```text
1. dedupe O by device/source event identity
2. parse/infer profile P
3. find payments where:
       collection_profile = P
       payable_amount = O.amount
       created_at <= O.occurred_at
       O.occurred_at <= grace_until
       reuse_after > safe_reference_time
4. if exactly one eligible payment exists:
       mark paid
       attach observation
       set payer fields when available
       enqueue payment.paid webhook
5. if none:
       store Activity as unmatched
6. if more than one:
       do not guess; store Activity as ambiguous
```

With correct profile-scoped amount reservation, step 6 should be exceptional and indicates a lifecycle/migration bug or insufficient occurrence-time confidence.

## No UTR/RRN requirement

v4 removes UTR/RRN from the matching contract.

If a parser happens to expose a bank reference in the future it may be stored as optional transaction metadata, but:

- creation does not depend on it;
- matching does not depend on it;
- the admin UI does not require it;
- integrations do not receive a promise that it exists.

The actual dedupe boundary is the signed relay event identity.

## Payer information

When available from notification text, store:

```text
payer_name
payer_upi_id
paid_at / occurred_at
```

Never fabricate a missing field.

A Paytm notification may provide only a payer name. A Kotak SMS may provide a UPI ID and/or name depending on wording. Null is valid.

## Operator edits

The operator can directly correct a payment without a separate Manual Review subsystem.

Editable:

- status
- name/alias
- external ID
- payer name
- payer UPI ID
- paid timestamp
- metadata
- internal note

Immutable after creation:

- PayGate payment ID
- original created timestamp
- requested amount
- generated payable amount
- collection-profile snapshot / destination UPI ID

Changing one of those immutable monetary/identity fields would invalidate the QR and amount reservation. Correct workflow is cancel + create a replacement.

Every edit creates a payment-history entry and schedules the appropriate webhook only when the externally meaningful state actually changes.

## Capacity behavior

If no payable amount is available within the configured maximum adjustment, creation fails cleanly with a retryable capacity error.

Do not silently charge an arbitrarily larger amount.

The dashboard should show current allocator pressure by profile so an operator can see whether the amount space is approaching saturation before increasing the configured cap.

## Tests required before cutover

- first candidate is smallest free non-zero-paise amount
- `.00` is never allocated
- allocator advances into next rupee after `.99`
- max adjustment is enforced
- uniqueness is profile-scoped
- adjacent requested amounts cannot collide
- active/grace/quarantine timestamps are correct
- profile switch does not change existing reservations
- on-time Paytm match
- grace Paytm match
- delayed-in-quarantine match with trustworthy occurrence time
- delayed event after reuse fails closed
- Kotak Messages notification normalizes/matches
- duplicate relay event is idempotent
- unmatched event cannot mutate payment
- ambiguous match fails closed
- operator status edit produces history/webhook once