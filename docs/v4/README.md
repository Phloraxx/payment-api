# PayGate v4 — Unified Payment Gateway Redesign

Status: architecture and implementation plan only. This branch must not change production behavior.

## Product definition

PayGate is the payment gateway. It consists of one server product and one Android app:

- **PayGate server** — payment creation, collection-profile selection, UPI URI generation, payment matching, admin dashboard, API, webhooks, persistence and operations.
- **PayGate Android** — operator UI plus the background notification relay. It is one APK and one product, not a separate Relay app.
- **PayGate Frontend** — remains a separate integration/testing playground. It is not part of the PayGate product architecture and must not own routing, profile selection or matching logic.

## v4 decisions

1. The caller asks PayGate to create a payment using an amount, a human name/alias and optional external metadata. It does **not** select Paytm, Kotak or another collection route.
2. PayGate owns the active **collection profile**. v4.0 launches with Paytm and Kotak. GPay and Slice are future profiles/sources.
3. PayGate returns a canonical `upi://pay?...` URI string. The consuming frontend is free to render that string as a QR code. PayGate does not need to return SVG/PNG QR files.
4. The Android relay does not know which collection profile is active. It only observes allowlisted phone notifications that look like incoming PayGate payments and sends a minimal signed snapshot to the server.
5. v4.0 ingestion sources are **Paytm for Business notifications** and **Kotak credit SMS notifications as surfaced by Google Messages**. There is no server-side Google Messages/libgm connector in the target architecture.
6. The server parses source-specific notification wording, infers the collection profile, normalizes a payment observation and performs the match.
7. UTR/RRN is not required by the v4 matching model. The primary identity is reserved payable amount + collection profile + occurrence time + unique relay event ID.
8. Amount allocation can move beyond the requested rupee while preserving non-zero paise. The allocator always prefers the smallest free adjustment.
9. The lifecycle is 5 minutes active + 5 minutes grace + 5 minutes quarantine before a payable amount can be reused.
10. Admin UX is rebuilt around **Overview / Payments / Activity / Settings**. Manual Review, Evidence and Reconciliation are not first-class product concepts.
11. Web and Android use a Razorpay-inspired dark navy + blue visual language, without copying Razorpay screens or branding.
12. The target persistence layer is **direct SQLite owned by one Go server process**. PocketBase is removed from the target architecture; PostgreSQL is deferred until PayGate actually needs multiple database writers or horizontal scaling.

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

## Non-goals

- No hosted customer checkout in PayGate v4. The standalone PayGate Frontend remains the test/integration surface.
- No WebSocket requirement. Polling remains the default status mechanism.
- No GPay auto-matching in v4.0.
- No Slice support in v4.0.
- No direct Android SMS permission request.
- No multi-user/role system.
- No PostgreSQL migration unless scale requires it.
- No attempt to preserve PocketBase as a public product surface.

## Safety rule for implementation

The v4 rewrite must be introduced beside v3, tested against production-shaped data and cut over only after parity. The existing payment-verification invariants and current production service must remain available for rollback until the final migration is verified.