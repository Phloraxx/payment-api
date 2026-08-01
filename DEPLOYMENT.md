# Production Deployment Record — 2026-08-01

## Release

- Reviewed source merged through PR #21.
- `main` commit: `99d1470`.
- Production service: `main-payment-17aqux`.
- Rollout used stop-first order so two processes never used the SQLite volume concurrently.
- Previous image retained locally as `main-payment-17aqux:pre-hardening-20260801`.

## Pre-cutover protection

Before schema migration, an online SQLite snapshot was taken from the live volume:

- `data.db`: `PRAGMA integrity_check = ok`;
- `auxiliary.db`: `PRAGMA integrity_check = ok`;
- Google Messages paired-session state included;
- archive SHA-256 generated;
- backup directory mode `700`, archive mode `600`.

The application now also creates PocketBase backup archives, verifies every ZIP entry and database presence daily, and performs a non-destructive temporary restore plus SQLite integrity checks monthly. A host cron exports the latest verified archive outside the Docker volume each day and retains 30 copies.

## Production verification

The following checks passed on the production network after migration:

- `/api/health` and `/api/paygate/health` healthy;
- Google Messages paired, connected and phone-responsive;
- unknown `/api/*` route returns JSON 404;
- retired `/api/webhook` route returns 404;
- authenticated payment create succeeds;
- harmless validation payment cancellation succeeds;
- public payment view does not expose RRN, payer name or UPI ID;
- hardened operator JavaScript bundle is served;
- backup creation succeeds;
- latest backup archive verification succeeds;
- temporary restore drill succeeds for both SQLite databases;
- stale Appwrite/cookie/legacy webhook secrets removed from the service.

A second forced task replacement then proved:

- database state persists;
- Google Messages reconnects without re-pairing;
- phone-responsive state returns;
- backup archives persist and re-verify;
- restore drill still passes;
- managed rate-limit rules remain idempotent across startup;
- Docker health returns to `healthy`.


## Post-cutover synthetic-data cleanup

A read-only scan of every operational table was performed for the acceptance, test and production-validation markers used during release testing. The production database contained only one synthetic row: a cancelled payment with external ID `production-hardening-validation`.

The row had no RRN and no references from SMS evidence, webhooks, review cases, refunds or reconciliation records. The service was scaled to zero, the isolated row was deleted in a single SQLite transaction, `PRAGMA integrity_check` returned `ok`, and the service was scaled back to one replica.

After restart:

- Google Messages returned to paired, connected and phone-responsive;
- all operational tables were rescanned and contained zero synthetic markers;
- the production database contained 15 payments and 8 Google Messages evidence events;
- no refund, audit, review, reconciliation or webhook acceptance records existed in production;
- a fresh backup `paygate_manual_20260801_110619.zip` was created;
- archive verification passed;
- temporary restore and integrity checks passed for both SQLite databases;
- the clean archive was exported to the protected host backup directory and its SHA-256 sidecar verified.

The earlier concern about an `ACCEPTANCE-REFUND-123` production record was incorrect: that record existed only in the disposable fresh-volume acceptance environment, not in production.

## Rollback

Application rollback:

```bash
docker service rollback main-payment-17aqux
```

The pre-hardening image tag and verified pre-cutover archive are retained. If database restoration is required, stop the service before replacing `/app/pb_data`; never restore while a task is writing to SQLite.

## External operational dependencies

Two optional integrations are implemented but intentionally not fabricated:

1. `OPERATOR_ALERT_WEBHOOK_URL` / `OPERATOR_ALERT_WEBHOOK_SECRET` need a real notification destination.
2. `PAYGATE_BACKUP_S3_*` needs real S3-compatible off-server credentials.

Until S3 is configured, backups exist in the persistent volume and in a separate protected host directory on the same server. This protects against container/volume deletion but not total server loss.

The remaining acceptance activity is a multi-day real-payment soak across UPI apps and delayed/offline-phone scenarios, recording transaction-to-SMS and SMS-to-verification latency. It does not block the deployed correctness safeguards, but it is required before reducing quarantine or claiming a delivery SLA.
