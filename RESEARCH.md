# PayGate — Research and Implementation Findings

This document records the technical facts that shaped the PocketBase/libgm rebuild and the implementation-specific problems discovered while validating it.

## 1. Prototype audit

The original TypeScript/Fastify service proved that a direct-to-UPI payment can be correlated from bank SMS evidence, but several prototype decisions were not safe foundations for a durable service.

Findings from the original implementation:

- amount state/expiry relied partly on process memory;
- restart behaviour could invalidate pending tickets;
- the advertised decimal matching path reduced a received amount to its whole-rupee base before selecting pending tickets, so two requests such as `₹100.01` and `₹100.02` could become ambiguous;
- database state lived in SQLite without the production deployment actually mounting its data directory persistently;
- the Android relay was an external dependency but the server interface itself only needed an SMS string plus shared secret;
- RRN uniqueness was a useful idea and was retained;
- integer money helpers were a useful idea and were retained.

The rebuild therefore kept exact-amount allocation, SMS parsing and RRN idempotency while replacing the state/runtime architecture.

## 2. PocketBase 0.39.9

PocketBase is embedded as a Go framework rather than deployed as a second process.

Upstream capabilities used directly by this project:

- `RunInTransaction` for short SQLite transactions;
- Go migrations and programmatically constructed collections;
- auth collections and API rules;
- custom HTTP routes through `OnServe`;
- SSE realtime record subscriptions;
- scheduled cron jobs;
- backup/log/admin facilities;
- a built-in rate-limit middleware/rule model;
- per-route request body limits.

Reference:

- https://pocketbase.io/docs/go-overview/
- https://pocketbase.io/docs/go-migrations/
- https://github.com/pocketbase/pocketbase

### Implementation-specific PocketBase findings

Several behaviours were verified against the exact v0.39.9 source/tests and then covered by PayGate tests:

1. Base collections only receive the system `id` field automatically; explicit `created`/`updated` autodate fields are required when the application wants them.
2. PocketBase filter date comparisons should receive PocketBase/RFC3339-formatted date values rather than relying on arbitrary Go `time.Time` binding. A validation experiment showed the same stored date failed a `<=` filter when bound as `time.Time` and matched when bound as a PocketBase-formatted date/string. PayGate therefore normalises every business date filter through one helper.
3. Collection index expressions are parsed/validated by PocketBase; a partial-index `IN (...)` expression used during the first migration draft was rejected and was replaced with an accepted boolean expression.
4. A wildcard static SPA route can still catch unknown API paths if it is not explicitly guarded. PayGate excludes `api` and `_` namespaces from its SPA fallback.
5. PocketBase rate-limit support exists but its settings must be enabled. PayGate enables configured rate limiting at serve time.

These are reasons the project pins and tests against a specific PocketBase version rather than assuming behaviour from older examples.

## 3. Why SQLite remains appropriate here

The current boundary is one operator and one service instance. SQLite gives:

- atomic allocation/state transitions;
- simple backup/recovery;
- very low deployment overhead;
- enough concurrency for this workload when transactions are kept short.

The application deliberately performs no external HTTP request inside a SQLite transaction. Outgoing webhooks use an outbox record and worker. Google Messages ingestion is normalised before entering payment matching.

A move to PostgreSQL would make sense if the product becomes a multi-merchant horizontally scaled service, but it would add operational complexity without solving a current requirement.

## 4. libgm / mautrix-gmessages

Pinned module:

```text
go.mau.fi/mautrix-gmessages v0.2605.0
```

Upstream source:

- https://github.com/mautrix/gmessages
- https://pkg.go.dev/go.mau.fi/mautrix-gmessages/pkg/libgm

The source contains the pieces needed for a standalone connector rather than browser-DOM automation:

- pairing;
- request crypto;
- protobuf schemas;
- private Google/Tachyon authentication;
- token refresh;
- long-poll/event handling;
- acknowledgements/session state;
- phone responsiveness events;
- old/catch-up message events.

`AuthData` contains sensitive cryptographic/device/session material including request crypto, refresh key, browser/mobile device identities, Tachyon auth token, IDs and cookies. PayGate therefore treats the JSON session like a credential, stores it outside normal API collections and enforces private filesystem permissions.

The connector is kept behind a small ingestion callback: payment-domain code never receives libgm protobuf types. This is a reliability boundary, not a claim that process/package separation changes licence obligations.

## 5. libgm licence

mautrix-gmessages/libgm is GNU AGPL-3.0. The upstream repository also contains special licence exceptions for named parties such as Beeper/Element; those exceptions do not automatically apply to this project.

Because this implementation directly links libgm into the PayGate binary, the rebuilt repository is distributed under AGPL terms (`LICENSE`, `NOTICE`). A future closed-source commercial distribution needs an actual licensing/legal decision, for example obtaining an appropriate licence or independently replacing the connector. It should not rely on an assumption that simply moving AGPL code to another process removes obligations.

## 6. Google Messages constraints

Google's official Messages-for-Web documentation describes a paired computer experience where the computer communicates through the phone. Important operational consequences:

- the phone remains part of the system;
- the phone needs cellular service to receive SMS;
- the phone/data path needs internet connectivity for Messages-for-Web synchronisation;
- background-data/battery restrictions can interfere with connectivity;
- paired devices expose sensitive message content and should be treated as trusted devices;
- pairings can become inactive/unpaired;
- only one computer is documented as active at a time even though multiple may be paired.

Official references:

- https://support.google.com/messages/answer/7611075
- https://support.google.com/messages/answer/9077245
- https://developer.android.com/training/monitoring-device-state/doze-standby
- https://developer.android.com/develop/connectivity/network-ops/data-saver

For a reliable installation, the intended phone setup is a dedicated/controlled Android device, stable power, stable Wi-Fi/mobile data, Google Messages allowed background/unrestricted data, and battery optimisation disabled for Messages where the device offers that control.

## 7. Latency

There is no published end-to-end Google Messages-for-Web latency SLA that PayGate can safely promise.

Total observed latency is the sum of:

```text
bank/core banking
→ mobile operator SMS delivery
→ phone Messages app
→ Google Messages relay
→ libgm event
→ PayGate persistence/parser/match
→ optional outgoing webhook
```

The final PayGate-local stages should be small under normal load; upstream bank/carrier/phone/network stages can be delayed or offline for an unbounded period.

This is why:

- payment state is durable;
- old/catch-up events are accepted and deduplicated;
- message occurrence time is retained;
- expired fingerprints are quarantined;
- an old event cannot confirm a payment created later;
- connector health is visible to the operator.

Real p50/p95/p99 latency must be measured with the actual bank/phone/network after live Google-account/emoji pairing (or QR fallback if required).

## 8. Why browser automation was rejected

Automating `messages.google.com/web` with Playwright/Puppeteer would couple payment verification to DOM structure, browser sessions and UI changes. libgm already exposes the underlying event/protocol implementation needed by the bridge ecosystem, so browser automation would add fragility rather than reduce it.

A custom clean-room protocol implementation is technically possible but would need to reproduce pairing, crypto, token refresh, protobuf/version tracking, long-polling, ack/session logic, catch-up handling and protocol changes. That is not justified for the current product while libgm works and the connector is isolated from the payment domain.

## 9. Evidence correlation research finding

A quarantine window by itself is not a complete stale-payment defence.

Scenario:

1. Payment A uses `₹100.01`.
2. A expires and its 24-hour quarantine eventually ends.
3. Payment B later reuses `₹100.01`.
4. Google Messages reconnects and emits an old SMS for A.

If matching only looks at current amount/status, that historical event can falsely confirm B.

The implemented fix is an evidence-time invariant:

```text
payment.created_at <= sms.OccurredAt
```

Provider timestamps are therefore domain-relevant evidence, not just observability metadata. A dedicated automated test reproduces reuse followed by delayed Google Messages catch-up and verifies B remains pending.

This timestamp guard solves delayed *historical delivery*, but not a payer who initiates a brand-new transfer from an old QR after the same amount has legitimately been reused. If the bank SMS exposes only amount and a new RRN, that new transfer is observationally identical to the current payment. No finite quarantine can remove this ambiguity forever; it only reduces its probability. Deployments needing cryptographic/provider-level correlation should move to official acquiring/bank APIs rather than pretending SMS/DDM supplies a guarantee it cannot.

## 10. RRN semantics

RRN is used as strong deduplication evidence:

- first exact amount + new RRN can resolve a payment;
- repeated RRN + same amount is idempotent;
- repeated RRN + different amount is an anomaly and does not resolve another payment.

The current single-UPI-account scope makes a global non-empty RRN uniqueness constraint practical. If PayGate becomes multi-account/multi-rail, the uniqueness scope should be revisited against the semantics of those official connectors rather than copied unchanged.

## 11. Legacy Android relay finding

Production inspection showed the old service uses the route `/api/webhook` and a legacy `WEBHOOK_SECRET`. The existing deployed secret is extremely weak, so promoting it to the new primary SMS secret would undermine the rebuild.

The compatibility strategy is therefore:

- primary product route: `/api/events/sms` + strong `SMS_WEBHOOK_SECRET`;
- old route: `/api/webhook` + old `WEBHOOK_SECRET`, available only when `LEGACY_SMS_WEBHOOK_ENABLED=true`;
- legacy route always identifies the source as `android_webhook`;
- log a startup warning while compatibility is enabled;
- remove/disable it after the phone relay is upgraded or Google Messages is proven.

This preserves cutover continuity without making the weak legacy credential the default security model.

## 12. Deployment finding

Inspection of the running Dokploy Swarm service showed the old production task has no persistent mount. Its SQLite file is therefore coupled to the task/container filesystem and can be lost on replacement.

A backup was taken before rebuild/cutover work. The new image uses `/app/pb_data`, and production acceptance explicitly requires task recreation with the same PocketBase state still present.

## 13. Current conclusion

The small, robust architecture for the present scope is:

```text
Go + embedded PocketBase/SQLite
+ exact persisted DDM payment service
+ durable SMS evidence
+ optional libgm adapter
+ legacy relay only as a migration bridge
+ durable signed webhook outbox
+ embedded React operator UI
+ one persistent pb_data volume
```

The remaining external uncertainty is not the payment/database architecture; it is live behaviour of Google's private Messages protocol on the real phone. That QR/device test is intentionally deferred and should be treated as a connector acceptance test, not as proof of the payment core.
