# PayGate Operations Runbook

This runbook covers evidence review, statement reconciliation, alerts, refunds, retention, backup verification and production acceptance. PayGate remains fail-closed: an operator may choose the payment associated with reviewed evidence, but cannot override exact amount equality, bank-reference uniqueness or stale-evidence guards.

## 1. Daily operator checks

Enrollment must remain disabled (`ANDROID_RELAY_ENROLLMENT_ENABLED=false`) except during a deliberate Android device-pairing window.

1. Open the dashboard and confirm Google Messages is `connected` when that rail is enabled.
2. Confirm **Android Relay** is `ready` when Paytm is enabled: at least one active device, recent heartbeat, notification access allowed, listener connected, and no failed queue that needs investigation.
3. Confirm **Open reviews** and **Open alerts** are zero, or inspect every item.
4. Check fingerprint capacity. Investigate warning pools at 70% and critical pools at 95%.
5. Confirm the latest backup exists. Archive verification runs daily; failures create a critical alert.
6. Review exhausted outgoing payment/refund webhooks.

An optional signed operator-alert webhook can push open/resolved operational alerts to an external monitoring or notification system. Configure `OPERATOR_ALERT_WEBHOOK_URL` and `OPERATOR_ALERT_WEBHOOK_SECRET` together. Notifications are durable, deduplicated and retried; repeated health checks do not send repeated notifications for the same open alert.

## 2. Evidence review

Review cases are created for:

- bank-credit-like SMS text that cannot be parsed;
- bank-credit-like email that fails trusted DKIM/DMARC authentication or parsing;
- a credit amount with no usable RRN/UTR;
- exact credits that match no eligible payment;
- ambiguous amount reuse;
- a bank reference already associated with a different amount;
- statement reconciliation conflicts.

### Manual match procedure

1. Inspect the original SMS, email or statement row and its provider/bank timestamp.
2. Confirm the exact payable amount shown by PayGate.
3. Confirm or enter the bank RRN/UTR/reference.
4. Select the intended payment ID.
5. Add a meaningful resolution note explaining the independent evidence used.
6. Submit **Manually match exact evidence**.

The server still rejects the action if:

- the evidence amount differs from the payment payable amount;
- the RRN is already assigned to another payment;
- the bank evidence predates the payment;
- the payment is already resolved by different evidence.

Every resolution writes an immutable `audit_events` entry with actor, action, entity and details. Original evidence records are not deleted or rewritten beyond adding the reviewed reference/match result.

## 3. Bank statement reconciliation

Supported inputs:

- CSV and TSV;
- XLSX first worksheet;
- up to 10 MiB, 10,000 credit rows and 64 columns.

The importer recognizes common date, credit/deposit, narration, type and RRN/UPI-reference headings. Debit rows are ignored. XLSX archives are checked for unsafe paths, excessive entries and decompression expansion before parsing.

Bank exports that omit a timezone are interpreted using `STATEMENT_TIMEZONE`, which defaults to `Asia/Kolkata`. Set this explicitly if the receiving account's statement timestamps use a different timezone. Evidence eligibility is determined from the bank transaction time, not the later import/review time.
PayGate allows a two-second tolerance only when comparing evidence against payment creation time, because SMS and statement timestamps may be rounded to whole seconds while payment records retain sub-second precision. Expiry, cancellation and fingerprint-reuse boundaries are not relaxed.

Classification:

- `matched` — bank RRN and exact amount already match a PayGate payment;
- `duplicate` — the same RRN appears more than once in the imported statement;
- `conflict` — same RRN/different amount, one amount candidate without a linked RRN, or multiple plausible historical candidates;
- `unmatched` — no PayGate payment matches the credit;
- `invalid` — required row values could not be parsed.

Imports are report-only. They never mark payments paid automatically. Conflicting and UPI-like unmatched rows create review cases. Importing identical file bytes twice is rejected by SHA-256 hash.

## 4. Refund lifecycle

PayGate records refund work but does not move money.

1. Create a refund only for a `paid` or `late` payment.
2. Enter the amount in paise and the reason.
3. Move the record to `processing` when the bank transfer is initiated.
4. Enter the actual bank refund reference and mark it `completed` only after verification.

The sum of active/completed refund records cannot exceed the received payment amount. Completed and cancelled records are terminal. Request and status events are signed and delivered through the normal durable outgoing webhook outbox:

- `refund.requested`
- `refund.processing`
- `refund.completed`
- `refund.failed`
- `refund.cancelled`

Consumers must still verify the signature and deduplicate the stable event ID.

## 5. Fingerprint capacity

Each whole-rupee requested amount has 99 paise fingerprints (`.01` through `.99`). Pending and quarantined records block reuse.

- warning: 70 blocked fingerprints;
- critical: 95 blocked fingerprints;
- exhausted: all 99 blocked.

Do not shorten quarantine merely to increase throughput. First measure real SMS delay and catch-up behavior. Prevent clients from repeatedly creating abandoned sessions and use idempotency keys for create retries.

## 6. Evidence retention

Defaults:

- raw SMS identity/body fields: 90 days;
- raw email sender/recipient/subject/body/authentication fields: 90 days;
- raw statement narration/row content: 365 days;
- operator audit events: 730 days.

After the message window, PayGate removes raw body/payload, sender and payer identity fields while preserving source, provider ID, bank timestamp, amount, RRN and processing result. Email subject, recipient and authentication headers are also redacted. Statement rows retain amount, RRN, timestamp and reconciliation status after raw narration is removed.

Configure with:

```text
PAYGATE_RETENTION_ENABLED=true
SMS_RAW_RETENTION=2160h
EMAIL_RAW_RETENTION=2160h
RECONCILIATION_RAW_RETENTION=8760h
AUDIT_RETENTION=17520h
```

## 7. Backups and restore drills

Automatic backups use PocketBase's transactionally generated `pb_data` archive and can be stored locally or in S3-compatible offsite storage.

```text
PAYGATE_BACKUP_CRON="0 3 * * *"
PAYGATE_BACKUP_MAX_KEEP=14
```

For offsite storage configure all `PAYGATE_BACKUP_S3_*` variables. Secrets must be injected by the deployment system and never committed.

Protection levels:

1. scheduled backup creation;
2. daily ZIP verification that reads every entry and confirms a database file exists;
3. monthly non-destructive restore drill into a temporary directory, followed by SQLite `PRAGMA integrity_check` on every restored database.

Manual commands:

```bash
payment-api backup-create
payment-api backup-verify
payment-api backup-restore-drill
```

The same actions are available from Operator UI → Settings. A drill does not replace the production volume and does not modify live data.

## 8. Google Messages incident response

### Payment email delivery failure

The Cloudflare Email Worker retries the PayGate webhook three times. A remaining failure is forwarded to the configured verified recovery inbox with `X-PayGate-Ingestion-Failed: true`. Download the original `.eml` and use `connectors/cloudflare-email-worker/replay-email.mjs`; this preserves Message-ID deduplication and the original bank timestamp. After the replay is verified in Email Events, delete the recovery copy according to the same 90-day-or-shorter evidence policy. Never copy the raw message or webhook secret into a ticket or chat.

### `reauth_required`

The phone pairing/encryption keys may still be valid. Use the operator UI's Google login refresh with a fresh same-account Google Messages Web `config` Copy-as-cURL request. Do not unpair unless same-account reauthentication fails.

### Disconnected

- wait through the five-minute alert grace period;
- confirm the phone is online and Google Messages is open/responding;
- use **Reconnect**;
- inspect redacted container logs for repeated auth or relay errors.

### Phone unresponsive

A warning opens after 15 minutes. Confirm battery optimization, connectivity and Google Messages background access on the paired phone.

Never paste cookies or session data into issue trackers, logs or chat transcripts. The connector session is a credential and must remain inside the protected persistent volume.

## 9. Production soak test

Before relying on PayGate for real orders, run controlled payments over several days covering:

- GPay, PhonePe, Paytm and another available UPI app;
- delayed SMS while the payment itself occurred before expiry;
- phone offline and later catch-up;
- container restart and task replacement;
- duplicate provider events and duplicate RRN;
- wrong amount, missing RRN and unrecognized wording;
- payment after expiry;
- two concurrent sessions for the same whole-rupee price;
- outgoing webhook failure/retry;
- statement reconciliation and manual review;
- real Slice email delivery, DKIM/DMARC pass, replay rejection, duplicate Message-ID, and rejection of any attempt to match Slice email evidence to Kotak or Kotak SMS evidence to Slice;
- Paytm Relay heartbeat loss, revoked Notification Access, listener disconnect/reconnect, device disable/re-enable, app update with old active notifications still in the shade, and offline queue catch-up;
- partial refund and completed refund reference.

Record payer action time, bank transaction time, SMS arrival time, ingestion time and final verification time. Do not reduce quarantine until the worst observed delay is understood.

## 10. Release acceptance

A release is production-ready only when:

- frontend typecheck/build pass;
- Go formatting, unit/integration tests, race tests and vet pass;
- migration up/down is tested on a fresh database;
- a production image builds and starts on a new temporary volume;
- liveness/readiness, auth rejection, payment create, SMS match, review, reconciliation, refund, backup and restore-drill flows pass;
- the temporary container is recreated on the same volume and state persists;
- the current production volume is backed up and verified before schema cutover.


During the v0.2 → v0.3 rollout, an existing **enabled** relay device receives a bounded 48-hour heartbeat grace only if it had validated signed traffic within the preceding 24 hours. Disabled or long-idle devices and new enrollments receive no grace. The first signed heartbeat immediately switches the device to normal permission/listener/staleness readiness checks, and any operator enable/disable state change permanently clears migration grace for that device.
