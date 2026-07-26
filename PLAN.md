# PayGate — Rebuild Implementation and Acceptance Status

This document records what was implemented and what must be proven before/after the production cutover.

## Completed implementation

### Application base

- [x] Replace Fastify/Node backend with Go.
- [x] Embed PocketBase 0.39.9 as the application framework.
- [x] Use one PocketBase SQLite database.
- [x] Add explicit Go migrations.
- [x] Preserve PocketBase `/_/` unchanged.
- [x] Embed a React/Vite operator UI into the Go binary.
- [x] Build a single non-root distroless production image.
- [x] Persist runtime data under `/app/pb_data`.

### Payment core

- [x] Store money as integer paise.
- [x] Accept only positive whole-rupee requested amounts.
- [x] Reserve `.01`–`.99`; never allocate `.00`.
- [x] Never spill DDM into the next rupee.
- [x] Reserve int64 headroom for the largest request.
- [x] Allocate transactionally in SQLite.
- [x] Randomize the starting suffix in production.
- [x] Persist payment expiry/quarantine timestamps.
- [x] Remove in-memory business timers/pools.
- [x] Exact-match full payable paise.
- [x] Add `pending`, `paid`, `expired`, `cancelled`, `late` states.
- [x] Add idempotent create keys.
- [x] Quarantine paid/cancelled/expired/late fingerprints.
- [x] Reject ambiguous automatic matches.
- [x] Deduplicate RRN.
- [x] Detect same-RRN/different-amount contradictions.
- [x] Reject old evidence that predates a reused payment.

### SMS ingestion

- [x] Persist `sms_events` evidence.
- [x] Scope provider dedupe to `(source, source_event_id)`.
- [x] Preserve provider/message timestamp.
- [x] Clamp missing/future timestamps safely.
- [x] Parse tested Kotak bank-credit variants.
- [x] Ignore unrelated/OTP messages.
- [x] Require amount and RRN for automatic confirmation.
- [x] Bound input/derived fields before PocketBase validation.
- [x] Add strong primary `/api/events/sms` secret.
- [x] Add explicit opt-in `/api/webhook` migration compatibility.

### Google Messages

- [x] Integrate libgm as an optional connector.
- [x] Persist/restore libgm AuthData.
- [x] Restrict session file/directory permissions.
- [x] Use provider MessageID and timestamp.
- [x] Ignore outgoing messages.
- [x] Privacy-prefilter bank-credit-like messages before ingestion.
- [x] Handle connected/degraded/phone-response events.
- [x] Back off reconnect attempts.
- [x] Add Google-account/Gaia emoji pairing API and UI.
- [x] Accept/validate upstream-required cookie JSON, raw Cookie headers and DevTools Copy-as-cURL input without logging values.
- [x] Keep QR pairing as a fallback and auto-refresh short-lived QR data.
- [x] Refuse accidental re-pair over an existing session or another pairing in progress.
- [ ] Complete a real phone Google-account/emoji pairing and verify reconnect/session persistence.

### API/security

- [x] API-key or operator-auth protection for payment writes.
- [x] Public limited status endpoint with bank evidence redacted.
- [x] Strict JSON decoding/unknown-field rejection.
- [x] Request body limits.
- [x] External ID/idempotency key/metadata bounds.
- [x] Strict startup environment parsing.
- [x] Minimum primary secret length.
- [x] URL validation for configured public/outgoing URLs.
- [x] PocketBase rate limits enabled by default.
- [x] Unknown `/api/*` routes remain API 404s, not SPA HTML.
- [x] `users`-only domain read rules; direct record writes locked.

### Outgoing webhooks

- [x] Durable delivery table/outbox.
- [x] Enqueue in the same payment transaction.
- [x] Network delivery after commit.
- [x] Stable event ID.
- [x] Timestamped HMAC-SHA256 signature.
- [x] Transactional claim.
- [x] Persisted retry schedule.
- [x] Stale `sending` lease recovery after restart.
- [x] Exhaustion state after retry ceiling.

### UI

- [x] Operator login.
- [x] Dashboard stats.
- [x] Realtime payment list.
- [x] Payment creation.
- [x] Payment detail/evidence.
- [x] Cancellation.
- [x] SMS evidence view.
- [x] Outgoing webhook-delivery view.
- [x] Connector health/settings.
- [x] Google-account/emoji pairing UI plus QR fallback rendering/refresh.
- [x] Periodic auth refresh and 401 sign-out.
- [x] UI create retries preserve idempotency key.

## Automated verification

The final branch must pass all of these after the last code change:

```bash
npm ci
npm audit
npm run typecheck
npm run build

test -z "$(gofmt -l cmd internal migrations)"
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
git diff --check

docker build --pull -t paygate-rebuild:test .
```

Important automated scenarios include:

- all 99 fingerprints and exhaustion;
- concurrent allocation uniqueness;
- no `.00`/spillover;
- amount overflow boundary;
- idempotent creation/conflict;
- exact match;
- duplicate/contradictory RRN;
- expiry and late payment;
- quarantine and post-quarantine reuse;
- delayed old SMS after amount reuse;
- source-event dedupe;
- source validation and field bounds;
- OTP/unrelated SMS handling;
- durable webhook HMAC/retry/concurrent claim;
- session file permissions;
- libgm timestamp/message extraction;
- API auth, redaction, request limits and route namespace behaviour;
- migration access rules.

## Fresh-container acceptance test

Before replacing `main`, use a **new temporary Docker volume** with the final image and prove:

1. clean startup and migrations;
2. `/api/health` healthy;
3. `/api/paygate/health` healthy;
4. compiled UI served at `/`;
5. unknown `/api/...` is a 404;
6. unauthenticated payment create denied;
7. fractional requested amount rejected;
8. authenticated payment create succeeds;
9. same idempotency key replays the same payment;
10. exact SMS event changes it to paid;
11. duplicate provider/RRN event is idempotent;
12. public status does not expose RRN/payer evidence;
13. database survives container stop/removal/recreation on the same volume;
14. Docker health transitions to healthy after recreation.

Live Google Messages device pairing is excluded from the generic container acceptance pass and is validated separately with the real phone.

## Production cutover checklist

### Protect the old deployment

- [x] Identify current `main-payment-17aqux` service.
- [x] Discover that the old service has no persistent mount.
- [x] Back up the prototype task data before changes.
- [ ] Reconfirm backup path/readability immediately before cutover.

### Prepare Dokploy

- [ ] Add a persistent volume or bind mount to `/app/pb_data`.
- [ ] Preserve the current UPI destination/payee configuration.
- [ ] Generate a new strong `PAYGATE_API_KEY`.
- [ ] Generate a new strong `SMS_WEBHOOK_SECRET`.
- [ ] Keep the old `WEBHOOK_SECRET` only if the current Android relay must survive the first cutover.
- [ ] Set `LEGACY_SMS_WEBHOOK_ENABLED=true` only for that transition.
- [x] Enable Google Messages only after production core cutover; live account/emoji pairing remains pending.
- [ ] Confirm no secret is printed into logs/history during the change.

### Branch/cutover

- [ ] Final independent diff/review pass.
- [ ] Commit the rebuild on `rebuild-pocketbase`.
- [ ] Push and confirm CI.
- [ ] Advance/merge `main` only after CI and fresh-container acceptance are green.
- [ ] Trigger Dokploy deployment for the `main` commit.
- [ ] Verify service health, UI and API through the actual production route.
- [ ] Create a harmless validation payment if appropriate.
- [ ] Recreate/restart the production task and prove its PocketBase state persists.
- [ ] Verify Docker/Dokploy reports healthy after restart.

## Post-cutover migration cleanup

Once the new Android endpoint or Google Messages path is confirmed:

- rotate/update the Android relay to `/api/events/sms` with `SMS_WEBHOOK_SECRET` and timestamps/provider IDs;
- set `LEGACY_SMS_WEBHOOK_ENABLED=false`;
- remove the old weak `WEBHOOK_SECRET` from Dokploy;
- pair/test Google Messages with the real phone using Google-account/emoji pairing (QR only as fallback);
- measure ingestion/matching latency and missed-event rate before treating libgm as the primary source.

## Definition of v1 done

For this rebuild, v1 is considered complete when:

- the final source/tests/container are green;
- `main` contains the reviewed rebuild;
- Dokploy runs that `main` build;
- `/app/pb_data` is persistent across a production task replacement;
- the API/UI/payment/SMS legacy path are verified in production;
- no known correctness/security blocker remains except the explicitly deferred real-phone QR validation.
