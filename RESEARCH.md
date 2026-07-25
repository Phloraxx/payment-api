# Payment API Rebuild — Research Notes

Research snapshot: **25 July 2026**.

This file records the upstream behaviour and constraints that the rebuild plan depends on. It is intentionally factual: when implementation starts, re-check version-sensitive details instead of assuming this snapshot remains current.

---

## 1. Current prototype audit

Repository audited: `Phloraxx/payment-api` on `main` before creating `rebuild-pocketbase`.

### Useful ideas worth preserving

- amounts are stored in integer paise;
- RRN has a database uniqueness constraint;
- Dynamic Decimal Matching gives concurrent payments distinguishable exact values;
- a bank SMS is treated as authoritative evidence;
- the API is intentionally small.

### Problems that justify a clean rebuild

#### Exact amount is currently lost in bank matching

Current `PaymentService.confirmFromBankSms()` converts the parsed bank amount to a base amount and then queries pending tickets by `base_amount`:

- source: [`src/server/services/payment.service.ts`](https://github.com/Phloraxx/payment-api/blob/main/src/server/services/payment.service.ts)
- helper: [`src/server/money.ts`](https://github.com/Phloraxx/payment-api/blob/main/src/server/money.ts)

`baseAmountFromPaisa()` floors to the whole-rupee block. Therefore `10001` and `10002` both become `10000` for the lookup. This contradicts the reason Dynamic Decimal Matching exists.

The rebuild matches the **exact payable amount**, e.g. `10037 == 10037`.

#### Important state is in memory

The current README documents:

- an in-memory `Map<base_amount, Set<taken_decimal>>`;
- per-ticket `setTimeout` handles;
- mass-expiry of pending tickets on restart because timer state is lost.

Source: [`README.md` on main](https://github.com/Phloraxx/payment-api/blob/main/README.md).

The rebuild stores payment validity/allocation state durably and derives expiry from timestamps.

#### Old plan accumulated duplicate infrastructure

The previous `PLAN.md` proposed, among other things:

- Appwrite secondary replication;
- custom WebSocket rooms/heartbeats;
- a separate SQLite logs database;
- custom passkey/WebAuthn administration;
- a custom expiry service;
- in-memory decimal pools;
- custom admin log streaming.

PocketBase already solves most of those infrastructure concerns. The rebuild removes them unless a real gap appears.

---

## 2. PocketBase: suitability as the application base

### Finding: PocketBase is explicitly designed to be embedded as a Go framework

PocketBase documentation states that it can be used as a Go framework/package to build custom portable applications and integrate arbitrary third-party Go libraries.

Sources:

- [Extending PocketBase](https://pocketbase.io/docs/use-as-framework/)
- [Extend with Go — Overview](https://pocketbase.io/docs/go-overview/)

This is the key reason **not** to fork PocketBase itself.

We can own `main.go`, routes and business logic while PocketBase remains an upgradable dependency.

### Finding: custom routes and lifecycle hooks are supported

The Go framework exposes route registration and application/event hooks.

Sources:

- [Extend with Go — Overview](https://pocketbase.io/docs/go-overview/)
- [Event hooks](https://pocketbase.io/docs/go-event-hooks/)

Implication:

- custom `/api/payments` routes are normal PocketBase extension code;
- connector start/stop can be integrated with app lifecycle;
- we do not need a second HTTP server/framework.

### Finding: transactions are built in

PocketBase exposes `app.RunInTransaction(fn)`.

Official docs explicitly note that:

- writes persist only if the callback succeeds;
- transaction code should use the provided `txApp`;
- only a single writer/transaction is allowed at a time;
- slow external operations should be kept outside transactions.

Source: [Extend with Go — Database](https://pocketbase.io/docs/go-database/).

Implication:

A short transaction is a good fit for exact-amount allocation:

```text
check candidate amount
insert payment
commit
```

We should never contact Google Messages or deliver webhooks while holding this transaction.

### Finding: realtime is already available through SSE

PocketBase's realtime API uses Server-Sent Events and publishes record create/update/delete events. Official SDKs manage reconnect/subscriptions.

Source: [API Realtime](https://pocketbase.io/docs/api-realtime/).

Implication:

The custom payment UI can subscribe to `payments` and `sms_events` directly. A custom WebSocket manager is unnecessary for v1.

### Finding: cron/scheduled jobs are built in

PocketBase exposes `app.Cron().Add/MustAdd`, runs jobs with `serve`, and shows registered jobs in the dashboard.

Source: [Extend with Go — Jobs scheduling](https://pocketbase.io/docs/go-jobs-scheduling/).

Implication:

Use one small periodic expiry/cleanup job. Do not create one timer per payment.

### Finding: migrations are built in and embeddable

PocketBase has Go migrations that can create/change collections and data, and the migrations become part of the final Go executable.

Source: [Extend with Go — Migrations](https://pocketbase.io/docs/go-migrations/).

Implication:

The schema should be committed as migrations rather than manually recreated through `/_/` on each deployment.

### Finding: PocketBase already includes admin/backup tooling

The upstream product provides:

- admin dashboard;
- logs;
- backup APIs/UI;
- cron inspection;
- SQL console for superusers.

Sources:

- [PocketBase](https://pocketbase.io/)
- [API Backups](https://pocketbase.io/docs/api-backups/)
- [API Crons](https://pocketbase.io/docs/api-crons/)
- [API SQL](https://pocketbase.io/docs/api-sql/)

Implication:

Do not rebuild raw collection/log/schema/backup screens into the normal payment UI.

### Version note

At this research snapshot, the public PocketBase docs show **v0.39.9**.

The current upstream `master` `go.mod` declares Go `1.25.0` and uses `modernc.org/sqlite`.

Sources:

- [PocketBase site/docs](https://pocketbase.io/)
- [`pocketbase/pocketbase` go.mod](https://github.com/pocketbase/pocketbase/blob/master/go.mod)

Do not blindly develop against `master`. Implementation should pin a specific released PocketBase version and use the Go version required by that release.

### Licence

PocketBase is MIT licensed.

Source: [`pocketbase/pocketbase` LICENSE.md](https://github.com/pocketbase/pocketbase/blob/master/LICENSE.md).

For this hobby project there is no reason to fork the PocketBase source merely to customise application behaviour.

---

## 3. libgm: what the source actually provides

Repository: [`mautrix/gmessages`](https://github.com/mautrix/gmessages).

The project is a Matrix ↔ Google Messages bridge, but the reusable Google Messages implementation lives under `pkg/libgm`.

### Finding: libgm stores the state required for persistent pairing

`libgm.AuthData` currently contains, among other fields:

- request encryption helper;
- refresh signing key;
- paired browser device;
- paired mobile device;
- Tachyon auth token/expiry;
- session/pairing identifiers;
- Google cookies.

Source: [`pkg/libgm/client.go`](https://github.com/mautrix/gmessages/blob/main/pkg/libgm/client.go).

Implication:

The connector session is highly sensitive and must be persisted as secret state, not logged or exposed as a normal public collection.

### Finding: event-driven long-poll connection exists

Current libgm source exposes:

- `SetEventHandler(...)`;
- `Connect()`;
- `ConnectBackground()`;
- `Disconnect()`;
- `IsConnected()`;
- `Reconnect()`.

`Connect()` starts the long-poll loop and acknowledgement handling. `Reconnect()` closes the existing poll and reconnects.

Source: [`pkg/libgm/client.go`](https://github.com/mautrix/gmessages/blob/main/pkg/libgm/client.go).

Implication:

A headless Go process can own the Messages-for-Web connection without Chromium/Playwright DOM scraping.

### Finding: upstream code persists and restores libgm session data

The upstream connector's login completion stores `client.AuthData` as session metadata and reconnects a client from it later.

Source: [`pkg/connector/login.go`](https://github.com/mautrix/gmessages/blob/main/pkg/connector/login.go).

Implication:

Persisting auth state and reconnecting after process restart is an intended/real upstream pattern, not an invented idea. We still must prove it on our actual phone/account.

### Finding: both QR and Google-account pairing logic exist upstream

The source contains:

- a QR login implementation using `StartLogin()` / `RefreshPhoneRelay()`;
- a Google-account pairing implementation using cookies plus emoji confirmation;
- session completion/persistence.

However, in the current bridge's advertised `LoginFlows`, the QR flow is commented out while the implementation remains in source.

Source: [`pkg/connector/login.go`](https://github.com/mautrix/gmessages/blob/main/pkg/connector/login.go).

Implication:

Do **not** design the final pairing UI around assumptions. Phase 0 must test the pairing flow that actually works in India with the target phone/account.

### Finding: upstream itself acknowledges phone/battery connectivity issues

The current login code includes a specific "phone not responding" error message advising that the phone be connected to the internet and that keeping the app open and/or disabling battery optimisation may be necessary.

Source: [`pkg/connector/login.go`](https://github.com/mautrix/gmessages/blob/main/pkg/connector/login.go).

This matches Google's own troubleshooting guidance and is why phone-state testing is part of Phase 0.

### Licence and maintenance state

`mautrix/gmessages` is AGPL-3.0 (with a separate exceptions file). The repository is actively maintained; GitHub lists release `v26.05` dated 16 May 2026 at this research snapshot.

Sources:

- [`mautrix/gmessages`](https://github.com/mautrix/gmessages)
- [`LICENSE`](https://github.com/mautrix/gmessages/blob/main/LICENSE)
- [`LICENSE.exceptions`](https://github.com/mautrix/gmessages/blob/main/LICENSE.exceptions)

For a private personal hobby deployment this is not the commercial-licensing concern it would be for a proprietary product, but the project should still retain/comply with the applicable open-source licence requirements if distributed.

---

## 4. Google Messages official constraints

Source: Google, not libgm.

### Phone internet is required

Google states that the phone needs Wi-Fi or a data connection for Messages for Web.

Source: [Check your messages on your computer or Android tablet](https://support.google.com/messages/answer/7611075?hl=en-IN).

Implication:

The phone may receive a cellular SMS while the data connection is unavailable, but the VPS cannot observe it through Messages for Web until connectivity resumes.

### Only one computer can be active at a time

Google states that the account can be paired with multiple devices but only one computer can be active at a time.

Source: [Google Messages for Web help](https://support.google.com/messages/answer/7611075?hl=en-IN).

Implication:

A dedicated payment phone is preferable. Normal use of Messages Web on another computer must be tested for interference with the headless connector.

### Inactive pairings may be removed

Google says accounts may be unpaired automatically if Messages for Web is not used for a few weeks.

Source: [Google Messages for Web help](https://support.google.com/messages/answer/7611075?hl=en-IN).

Implication:

Connector UI must support `unpaired` as a normal recoverable state.

### Google explicitly recommends background data and disabling battery optimisation when troubleshooting

Google's troubleshooting instructions say to:

- ensure strong phone/computer internet;
- enable Google Messages background data;
- disable battery optimisation if enabled.

Source: [Fix problems sending, receiving, or connecting to Google Messages](https://support.google.com/messages/answer/9077245?co=GENIE.Platform%3DDesktop&hl=en).

Implication:

Screen lock alone should not be treated as the failure mechanism. The real variables are Android/OEM background restrictions, battery mode and network availability. These need measurements on the actual device.

### QR pairing in India

Google's current India help page still describes QR pairing and notes that QR pairing is no longer available in the US.

Source: [Google Messages for Web help — India](https://support.google.com/messages/answer/7611075?hl=en-IN).

Implication:

QR pairing is worth testing first for a headless hobby service, but libgm/upstream behaviour is still the deciding factor.

---

## 5. What is known vs unknown about latency

### Known

- libgm uses a long-poll connection rather than browser DOM polling.
- SMS arrival itself depends on the mobile network/bank sender.
- Messages-for-Web forwarding depends on phone + internet + Google relay connectivity.
- our own database match path can be kept local and short.

### Unknown / must be measured

There is no useful Google Messages-for-Web latency SLA for this use case.

We should measure:

```text
message_at
connector_received_at
payment_matched_at
```

and classify tests by:

```text
screen locked/unlocked
charging/not charging
Battery Saver on/off
Wi-Fi/mobile data
normal/poor connectivity
live event/recovered after outage
```

Do not publish or hard-code assumptions such as "confirmation always takes 2 seconds".

---

## 6. Architecture decisions resulting from the research

### Decision A — use PocketBase as framework, not fork

Reason:

- upstream explicitly supports this use case;
- Go gives direct libgm integration;
- upgrades remain dependency upgrades rather than long-lived source merges;
- built-in infrastructure eliminates much of the prototype code.

### Decision B — one Go process by default

Reason:

- PocketBase and libgm are both Go;
- one-user hobby scale does not justify IPC/microservices;
- fewer deployment/failure boundaries.

### Decision C — keep a custom UI but leave PocketBase `/_/` untouched

Reason:

- normal payment workflows need a domain-specific UI;
- the upstream admin UI is already excellent for raw records/schema/logs/backups;
- modifying it would create upgrade maintenance for little benefit.

### Decision D — exact payment matching only

Reason:

The prototype's base-amount lookup defeats DDM when multiple `₹100.xx` payments are pending.

### Decision E — durable timestamps instead of in-memory timers

Reason:

A process restart should not alter payment truth.

### Decision F — retain SMS events

Reason:

SMS is the source evidence and the connector is unofficial. A persisted event trail is invaluable for parser failures, late messages, duplicates and latency debugging.

### Decision G — no queue/cache/database extras initially

Reason:

PocketBase/SQLite can perform the entire expected workload locally. Adding Redis/Postgres/message brokers would add operational complexity without solving an observed problem.

### Decision H — libgm feasibility spike before UI work

Reason:

It is the highest-risk external component and depends on real Google/phone behaviour that unit tests cannot prove.

---

## 7. Phase 0 live test matrix

Use a dedicated test payment/SMS where possible.

| Test | Phone | Network | Expected observation |
|---|---|---|---|
| Pair | unlocked | Wi-Fi | pairing completes and auth state saves |
| Restart | any | Wi-Fi | process restores session without repair |
| Normal SMS | unlocked | Wi-Fi | incoming event received |
| Locked SMS | locked | Wi-Fi | measure latency/reliability |
| Idle phone | locked/idle | Wi-Fi | measure after extended idle |
| Battery Saver | locked | Wi-Fi | measure degradation |
| Mobile data | locked | mobile | measure latency/reliability |
| Poor data | locked | weak/limited | measure delay/reconnect |
| VPS outage | any | phone online | determine catch-up after VPS returns |
| Phone data outage | any | no phone data | SMS may arrive; determine sync after data returns |
| Duplicate/reconnect | any | normal | verify duplicate source IDs/updates can be deduped |
| Other Messages Web client | any | normal | verify one-active-computer interaction |
| Long unattended run | locked/idle | normal | observe unpair/reconnect stability |

Record real results in this file or a later test log before declaring libgm production-ready for this hobby service.

---

## 8. Sources to re-check before implementation

PocketBase:

- https://pocketbase.io/docs/use-as-framework/
- https://pocketbase.io/docs/go-overview/
- https://pocketbase.io/docs/go-database/
- https://pocketbase.io/docs/go-migrations/
- https://pocketbase.io/docs/go-jobs-scheduling/
- https://pocketbase.io/docs/api-realtime/
- https://github.com/pocketbase/pocketbase

Google Messages/libgm:

- https://github.com/mautrix/gmessages
- https://github.com/mautrix/gmessages/blob/main/pkg/libgm/client.go
- https://github.com/mautrix/gmessages/blob/main/pkg/connector/login.go
- https://support.google.com/messages/answer/7611075?hl=en-IN
- https://support.google.com/messages/answer/9077245?co=GENIE.Platform%3DDesktop&hl=en

Prototype:

- https://github.com/Phloraxx/payment-api/tree/main
- https://github.com/Phloraxx/payment-api/blob/main/src/server/services/payment.service.ts
- https://github.com/Phloraxx/payment-api/blob/main/src/server/money.ts
- https://github.com/Phloraxx/payment-api/blob/main/PLAN.md

---

## 9. Research conclusion

The rebuild is feasible without forking PocketBase and without introducing extra infrastructure.

The architecture with the best simplicity/correctness ratio is currently:

```text
Go
+ PocketBase/SQLite
+ libgm
+ small React/Vite UI
+ one container
```

The one unresolved architectural risk is the real-world Google Messages connector behaviour. That is why the next engineering task is a **small libgm feasibility spike**, not the full payment rewrite.
