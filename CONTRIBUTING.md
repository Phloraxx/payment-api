# Contributing to PayGate

PayGate is a small system with strong opinions, because the failure mode it guards against is confirming a payment that never happened. Changes are reviewed against the invariants first.

## Before changing code

Read, in this order:

- [`docs/v4/README.md`](docs/v4/README.md) — the v4 documentation index
- [`docs/v4/00_PRODUCT_VISION.md`](docs/v4/00_PRODUCT_VISION.md) — what PayGate is and is not
- [`docs/v4/03_PAYMENT_LIFECYCLE_AND_MATCHING.md`](docs/v4/03_PAYMENT_LIFECYCLE_AND_MATCHING.md) — reservation, grace, quarantine, matching
- [`docs/v4/07_DATA_STORAGE_SECURITY_OPERATIONS.md`](docs/v4/07_DATA_STORAGE_SECURITY_OPERATIONS.md) — storage, secrets, operations
- [`docs/v4/10_EDGE_CASES_AND_INVARIANTS.md`](docs/v4/10_EDGE_CASES_AND_INVARIANTS.md) — the list this file summarises

## Invariants

- The merchant API is provider-blind. A merchant asks for an amount and gets an instruction; it never selects Paytm, Kotak, or a relay.
- The Android client captures notifications and signs events. It never declares a payment successful.
- Source-specific parsing, profile inference, source trust, matching, deduplication and payment mutation are server-owned.
- Matching uses the snapshotted collection profile, exact payable amount, reservation lifecycle state, trustworthy occurrence time and unique signed relay-event identity.
- Only origin-bound evidence transitions a payment to paid. Generic and package-agnostic notifications remain evidence for operator confirmation.
- Exactly one PayGate process owns the live SQLite volume. The allocator and matcher fail closed when ownership is ambiguous.
- Admin password, merchant API keys, webhook signing secrets and Android device keys are separate credentials. Never collapse them into one.
- Plaintext secrets are not persisted when a hash or verifier is sufficient.
- Payer identity and raw notification detail stay out of unauthenticated and public payment views.
- Migrations must be rollback-safe. A rollback image and the previous data volume stay available until acceptance passes.

## Development

```sh
npm ci
npm run typecheck:v4
npm run build:v4

go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...

docker build -f Dockerfile.v4 -t paygate:v4 .
```

Run `-race` before opening a PR. The relay and matching paths are concurrent, and a passing non-race run proves very little.

## Pull requests

Include:

- the behaviour being changed and why;
- whether matching, evidence trust, lifecycle timing or webhook semantics changed;
- tests added or updated, including the failure case you are protecting against;
- migration and rollback implications;
- whether any merchant-visible response shape changed.

Commit messages follow Conventional Commits with a body explaining the reasoning: `feat:`, `fix:`, `test:`, `docs:`, `chore:`, `refactor:`, `ci:`.

## What not to send

- real notification payloads, payer VPAs, amounts tied to a live transaction, or relay enrollment codes;
- `.env` files, database dumps or backup archives;
- a patch that relaxes evidence requirements "to make the demo work";
- a change that adds a second process, cache or queue without addressing SQLite ownership.

## Reporting a security issue

Do not open a public issue. Follow [`SECURITY.md`](SECURITY.md).
