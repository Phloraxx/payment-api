# PayGate v4 — Unified Payment Gateway Redesign

Status: architecture/research/implementation plan only. This branch must not change production behavior.

## Product definition

PayGate consists of two first-class runtime artifacts:

- **PayGate server** — developer API, payment creation, collection-profile selection, randomized payable-amount reservation, matching, admin dashboard, webhooks, direct SQLite persistence and operations.
- **PayGate Android** — operator UI plus background notification relay in the same APK/product.
- **PayGate Frontend** — separate integration/testing playground. It renders QR from PayGate's UPI URI and polls status, but it is not part of the PayGate runtime architecture.

## Correct merchant field semantics

A create request contains context such as:

```json
{
  "amount": 100,
  "name": "Sourav P Bijoy",
  "external_id": "evt_hardware_security_2026"
}
```

- `name` is the merchant-supplied person/payee identifier for the payment. It is not the event name and is not used for matching.
- `external_id` is the merchant/event-side **event ID**. It is deliberately non-unique because one event can have many payments.
- PayGate's own payment ID + `Idempotency-Key` provide payment/request identity.
- notification-derived `payer_name`/`payer_upi_id` are separate and may differ from `name`.

## v4 decisions

1. Caller never selects Paytm/Kotak. PayGate owns the active **collection profile** for new payments.
2. v4.0 supports Paytm and Kotak. GPay and Slice are deferred.
3. PayGate snapshots profile/destination onto each payment, so switching the active profile never changes existing sessions.
4. PayGate returns the canonical `upi://pay?...` string. The frontend renders the QR; PayGate does not need SVG/PNG/hosted-checkout output.
5. Android does not know the active profile or expected payment. It relays a minimal signed notification snapshot from allowlisted packages when it contains a plausible non-`.00` money value.
6. Server parses source-specific wording, infers Paytm/Kotak, validates incoming-credit semantics and performs matching.
7. Target v4 has no server-side Google Messages/libgm connector. Kotak arrives through the phone's Google Messages notification.
8. UTR/RRN is not part of v4 matching.
9. Payable amounts are selected **cryptographically at random from the bounded free pool**, not sequentially from `.01` upward.
10. Default candidate space may span requested amount through `+₹1.99`, skipping every `.00`; exact cap stays configurable.
11. The allocator combines 5m active + 5m grace + 5m hard quarantine with **soft recent-use avoidance** after release. Recent values are avoided when alternatives exist but remain available under pressure.
12. Exact profile+payable active ownership is database-enforced in SQLite.
13. Admin UX is rebuilt around **Overview / Payments / Activity / Settings** rather than Review/Evidence/Reconciliation products.
14. Web and Android use a Razorpay-inspired dark navy + blue design language.
15. Authentication is password-only for the singleton operator; merchant API keys, webhook secrets and Android device keys remain separate credentials.
16. Persistence is **direct SQLite only** through Go/`database/sql` + `modernc.org/sqlite`. PocketBase is removed from the target runtime.
17. Production keeps one PayGate process, one local SQLite database, WAL, `synchronous=FULL`, foreign keys, bounded busy timeout, database constraints and SQLite Online Backup API.

## Documents

- [00_PRODUCT_VISION.md](00_PRODUCT_VISION.md)
- [01_TARGET_ARCHITECTURE.md](01_TARGET_ARCHITECTURE.md)
- [02_NOTIFICATION_INGESTION_AND_PAIRING.md](02_NOTIFICATION_INGESTION_AND_PAIRING.md)
- [03_PAYMENT_LIFECYCLE_AND_MATCHING.md](03_PAYMENT_LIFECYCLE_AND_MATCHING.md)
- [04_PUBLIC_API_AND_WEBHOOKS.md](04_PUBLIC_API_AND_WEBHOOKS.md)
- [05_ANDROID_APP.md](05_ANDROID_APP.md)
- [06_ADMIN_UI_AND_DESIGN_SYSTEM.md](06_ADMIN_UI_AND_DESIGN_SYSTEM.md)
- [07_DATA_STORAGE_SECURITY_OPERATIONS.md](07_DATA_STORAGE_SECURITY_OPERATIONS.md)
- [08_MIGRATION_AND_IMPLEMENTATION_PLAN.md](08_MIGRATION_AND_IMPLEMENTATION_PLAN.md)
- [09_RESEARCH_NOTES.md](09_RESEARCH_NOTES.md)
- [10_EDGE_CASES_AND_INVARIANTS.md](10_EDGE_CASES_AND_INVARIANTS.md)

## Non-goals

- no hosted customer checkout in PayGate v4;
- no WebSocket requirement;
- no GPay or Slice auto-matching in v4.0;
- no direct Android SMS permission;
- no UTR/RRN dependency;
- no multi-user/role system;
- no PostgreSQL;
- no PocketBase runtime;
- no server-rendered QR images;
- no merchant-controlled collection-profile selection.

## Safety rule for implementation

V4 is built beside v3 and cut over only after deterministic migration, parser parity, SQLite backup/restore testing, Android in-place upgrade testing and end-to-end payment acceptance.

The final design should be simpler than v3, but migration must not sacrifice payment correctness or rollbackability.