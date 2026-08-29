# PayGate v2 — Production Cutover Runbook

## Scope

This runbook covers the production transition for:
- API service `main-payment-17aqux`;
- customer frontend `main-payment-frontend-t1n1x8`;
- PayGate Android operator/relay app;
- domains `pay.mulearnscet.in`, `payment.mulearnscet.in`, and `pay.ieeesahrdaya.com`.

The database is SQLite on the persistent Docker volume mounted at `/app/pb_data`. Never run two PayGate API tasks against that volume at once.

## Preconditions

All of the following must be true before merging/deploying:
- API PR CI is green, including race, vet, staticcheck, govulncheck, and container build.
- Customer frontend PR CI is green and static-container smoke tests have passed.
- Android PR CI is green and the debug APK builds successfully.
- The latest production backup checksum verifies.
- `backup-verify` and `backup-restore-drill` pass on production.
- `scripts/v2-production-copy-acceptance.sh` passes against a fresh restored production backup.
- The production-copy alert test proves legacy webhook alerts aggregate without replay.
- Production health is green before cutover.
- `./scripts/v2-host-preflight.sh disabled` passes from the deployment host. The service image must be pinned; `:latest` is rejected.

## Environment normalization

The current production environment passes a non-printing v2 `Load()` + `ValidateServe()` dry-run unchanged. Do **not** normalize names before the first compatibility deployment; minimizing simultaneous changes is safer. After Phase A is healthy and before Phase B, normalize names in the deployment source of truth without printing or rotating existing values:
- copy `UPI_ID` to `KOTAK_UPI_ID`, then retire `UPI_ID`;
- copy `UPI_PAYEE_NAME` to `KOTAK_UPI_PAYEE_NAME`, then retire `UPI_PAYEE_NAME`;
- retain `PAYMENT_TTL` and retire the older `TICKET_TTL_MINUTES` fallback;
- because `LEGACY_SMS_WEBHOOK_ENABLED=false`, retire the unused `WEBHOOK_SECRET` after confirming `/api/webhook` remains 404;
- remove stale Appwrite variables, `COOKIE_SECRET`, `ONE_TIME_CODE`, `PUBLIC_BASE_URL`, and `RP_ID`;
- retain active PayGate API, SMS/email evidence, Android relay, outgoing webhook, Slice, Paytm, Google Messages, persistence, payment TTL/quarantine, and rate-limit settings;
- leave orchestrator-level `HOST`/`PORT` unchanged for the first cutover even though the PayGate binary does not consume them directly.

A non-printing config dry-run must pass again after normalization. `PAYGATE_CHECKOUT_ORIGINS` stays unset in Phase A. In Phase B set it only to `https://payment.mulearnscet.in,https://pay.ieeesahrdaya.com`.

## Phase A — API compatibility deployment

1. Create a fresh production backup and verify its archive checksum.
2. Run the non-destructive restore drill.
3. Record the current API image/task identity and verify the persistent volume mount. Keep the preserved pre-v2 image tag for rollback.
4. Keep `PAYGATE_CHECKOUT_ORIGINS` unset/empty for the first API deployment.
5. Keep Android relay enrollment closed.
6. Build/tag the v2 API with an immutable release identifier (prefer the commit SHA, for example `main-payment-17aqux:v2-<sha>`), then deploy that exact image with one replica and stop-first semantics. Never deploy production from `:latest`. Set `PAYGATE_EXPECTED_IMAGE` to that exact image when running host preflight.
7. Allow schema migrations to complete against the existing persistent volume.
8. Verify `/api/health` and `/api/paygate/health` immediately.
9. Verify Google Messages remains paired/connected and the Android relay remains ready.
10. Verify trusted payment creation remains authenticated and public checkout remains disabled.
11. Verify retired Google Messages QR routes return 404 for an authenticated operator. Current v1 keeps these routes behind auth and returns 401 anonymously; v2 removes them entirely.
12. Verify operator login and `/api/operator/v2/overview`.
13. Wait through at least one operational-alert cron pass and confirm:
   - exhausted webhook rows are unchanged;
   - legacy per-delivery webhook alerts are resolved;
   - exactly one aggregate `webhook:exhausted` alert is open while exhausted rows remain;
   - no historical delivery is replayed.

Rollback immediately if database health, payment reads, relay readiness, connector state, authentication, or outbox behavior regresses.

## Phase B — Enable browser checkout

After the API compatibility soak is clean:
1. Set `PAYGATE_CHECKOUT_ORIGINS` to the exact production customer origins only.
2. Redeploy/restart the API and run `./scripts/v2-host-preflight.sh enabled https://payment.mulearnscet.in` and again for `https://pay.ieeesahrdaya.com`; both must pass exact-origin CORS and denied-origin checks.
3. Confirm anonymous trusted `/api/payments` remains 401.
4. Confirm `/api/checkout/v2/payment-accounts` is reachable only from an allowed browser origin.
5. Confirm create/status quotas and `Retry-After` behavior.
6. Verify Kotak, Slice and Paytm account readiness before changing frontend traffic.

## Phase C — Static customer frontend

1. Retain the existing frontend image/task metadata for rollback.
2. Deploy the v2 static Nginx image from an immutable release tag/digest, not `:latest`.
3. Verify `/api/health` on the frontend container and SPA deep-link routing.
4. Verify CSP, HSTS, frame, referrer, content-type and asset-cache headers.
5. Verify both customer domains render the same static build.
6. Create a harmless checkout on each direct PayGate rail and cancel it if unpaid.
7. Run Razorpay Test end-to-end.
8. Keep Razorpay Live limited to the existing ₹1 pilot until separately approved.

Rollback the frontend independently if the UI or public checkout regresses; the API compatibility routes remain available during the initial soak.

## Phase D — Android app upgrade

1. Merge/release Android only after the required v2 operator endpoints are live.
2. Build the signed release APK with the existing package name and signing identity.
3. Install over the existing app; do not uninstall and do not clear app data.
4. Verify the relay device identity/pairing survives the upgrade.
5. Verify the foreground relay service starts and remains active independently of operator login.
6. Verify notification-listener access and battery-optimization exemption remain granted.
7. Verify signed heartbeat, pending queue and failed queue return healthy.
8. Sign in as an operator and verify Home, Payments, Reviews and Health.
9. Verify operator token refresh does not interrupt the relay runtime.
10. Verify the Google Messages shadow gate is visible but cannot change payment state.

## Real-payment acceptance

Perform one controlled transaction per verification rail:
- Kotak SMS;
- Slice email;
- Paytm notification while the phone is locked and Battery Saver is enabled;
- Razorpay Test;
- Razorpay Live ₹1 pilot when explicitly approved.

For each transaction verify exact amount, evidence identity, timestamps, one state transition, durable outbox behavior, no duplicate evidence, expected review behavior and audit history. Never use manual confirmation to rescue an automatic-match test.

## Google Messages retirement

Do not remove libgm during initial v2 cutover. Run the Android Google Messages path in observation-only shadow mode until the operator metric reaches the documented manual-review gate: at least 100 complete libgm samples, at least 100 parseable Android samples, 100% Android bank-reference coverage, 100% exact amount+reference parity, and zero libgm-only complete events.

Reaching the gate only permits a reviewed removal change. Remove libgm/session/reauth code in a later PR after a stable soak and a fresh backup.

## Rollback

Application rollback is preferred over database restoration because v2 migrations are additive during this cutover.

- API: use the Docker/Dokploy service rollback to the previous image, preserving `/app/pb_data`.
- Frontend: roll back independently to the previous frontend task/image.
- Android: retain the previous signed APK as an emergency downgrade artifact; do not clear app data.
- Database restore is last resort only. Scale the API to zero before replacing `/app/pb_data` and verify SQLite integrity before restart.

Immediate rollback triggers include duplicate RRN/reference/fingerprint allocation, incorrect amount matching, unexpected payment confirmation, missing outbox/audit mutations, database integrity failure, sustained SQLite lock regression, relay readiness regression, or a customer checkout that obscures the authoritative amount/status.
