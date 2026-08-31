# PayGate SQLite single-process incident — 2026-08-31

## Summary

Production checkout degraded again on 2026-08-31 because the live PayGate process began returning SQLite `disk I/O error (522)` on ordinary reads. The database files themselves remained restorable and passed integrity checks.

The strongest trigger correlation is the host backup-export cron. The production API task started at `00:30:08Z`. PocketBase created its normal in-process backup at `03:00:00Z`. The host export ran at `03:20` and invoked `/app/paygate backup-verify` with `docker exec`. The first live-process SQLite 522 errors appeared at `03:26:00Z`.

`backup-verify` is not a standalone archive reader. Before Cobra dispatch, the binary constructs a full PocketBase app and all PayGate services against `pb_data`. Therefore the host cron started a second PocketBase process against the same SQLite directory while the production server was active.

## Evidence

- Production data volume remained readable.
- `@auto_pb_backup_acme_20260831030000.zip` passed ZIP CRC validation.
- Restored root `data.db` returned `PRAGMA integrity_check = ok`.
- Restored root `auxiliary.db` returned `PRAGMA integrity_check = ok`.
- Restarting the exact same pinned API image immediately restored `db=ok` without restoring or mutating the databases.
- The failure repeated after another day with the same host-side backup-export pattern.

This makes database corruption unlikely and stale/poisoned handles caused by concurrent PocketBase processes the primary incident mechanism. The new process lock is an additional fail-closed guard even if another trigger is later discovered.

## Production mitigation

The host exporter no longer executes the PayGate binary. It now:

1. mounts the PayGate volume read-only;
2. copies only the newest completed backup ZIP;
3. validates ZIP CRCs;
4. extracts the archive into a temporary directory;
5. requires both active root SQLite databases;
6. runs `PRAGMA integrity_check` on the restored copies;
7. writes a SHA-256 sidecar; and
8. retains the newest 30 exported archives.

The production host script is versioned as `scripts/v3-host-backup-export.sh` and is byte-identical to `/home/drvij/bin/paygate-backup-export.sh` at the time of this fix.

## Binary hardening

PayGate now takes a non-blocking exclusive OS lock on `.paygate-process.lock` inside the configured data directory before constructing PocketBase. A second PayGate process using the same data directory fails before opening PocketBase or SQLite.

The standalone healthcheck remains outside this lock and does not open the data directory.

Tests prove a second acquisition fails and that releasing the first lock permits a new process to acquire it.
