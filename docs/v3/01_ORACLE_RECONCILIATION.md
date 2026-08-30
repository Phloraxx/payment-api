# PayGate v3 — Oracle Worktree Reconciliation

## Compared states

Both implementations start from the same v2 foundation commit: `a89845b7`.

- Original Oracle worktree: `redesign/paygate-product-v3`, uncommitted.
- Recovery branch: `redesign/paygate-product-v3-recovery`.
- Comparison performed after the Oracle host returned online on 2026-08-30.

The Oracle worktree contained 14 modified tracked files and six untracked v3 files. It was treated as recovered design/code input, not as automatically authoritative source.

## Carried forward from Oracle

The following original ideas improved the recovery branch without weakening payment invariants:

- A dedicated **More** hub groups advanced money operations, evidence/delivery tools and system controls.
- More remains in primary navigation so mobile operators can reach advanced tools; advanced routes no longer disappear with the desktop sidebar.
- Payment search includes captured payer name, description and private admin note in addition to IDs, customer fields, UPI and evidence references.
- Status is available as an additional whitelisted payment sort.
- Protected IDs, references, UPI values and key amounts expose convenient copy actions.
- Effective payment-profile edits record before/after audit snapshots, while custom JSON is represented by a digest.

## Intentionally not carried forward

The recovery branch remains stricter where the Oracle draft conflicted with create/idempotency or validation guarantees:

- `externalId` is not editable after creation. Idempotent replay compares it with the original create request.
- Original create `metadata` is not editable after creation for the same replay-identity reason.
- The update API uses strict JSON decoding; unknown/protected keys fail with HTTP 400 instead of being ignored.
- Invalid pagination/filter input is rejected rather than silently clamped into another query.
- Oversized tags are rejected rather than silently dropped.
- Operator custom fields use bounded storage/request sizes instead of the broader draft limits.
- The recovery UI and Health implementation were retained where they had already passed desktop/mobile browser QA and the Oracle draft did not provide a stronger invariant or capability.

## Audit disposition

The Oracle draft's richer audit concept was retained but narrowed to safe editable profile data. Audit snapshots include display/customer fields, description, private note and tags. `customFields` is represented by a SHA-256 digest. Protected creation identity, financial truth and captured evidence are not duplicated into the profile-edit audit payload.

An unchanged form remains a no-op and must not emit an audit event.

## Scope boundary

This reconciliation closes the **web operator/admin v3** recovery gap. The Oracle product plan also described later customer-checkout and Android-operator redesign stages. Those are separate surfaces and should be implemented/reconciled independently after this admin PR is accepted; they must not be smuggled into the production API cutover without their own tests and rollout gates.
