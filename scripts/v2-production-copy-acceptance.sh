#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 --source-copy /path/to/pb_data --binary /path/to/paygate" >&2
  exit 64
}

SOURCE=""
BINARY=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-copy) SOURCE="${2:-}"; shift 2 ;;
    --binary) BINARY="${2:-}"; shift 2 ;;
    *) usage ;;
  esac
done
[[ -n "$SOURCE" && -n "$BINARY" ]] || usage
[[ -d "$SOURCE" ]] || { echo "source copy not found: $SOURCE" >&2; exit 66; }
[[ -x "$BINARY" ]] || { echo "binary is not executable: $BINARY" >&2; exit 66; }

SOURCE="$(realpath "$SOURCE")"
case "$SOURCE" in
  /app/pb_data|/app/pb_data/*) echo "refusing live production data path" >&2; exit 70 ;;
esac
[[ -f "$SOURCE/data.db" ]] || { echo "data.db missing from source copy" >&2; exit 65; }
[[ -f "$SOURCE/.paygate-acceptance-copy" ]] || { echo "refusing unmarked data directory; create .paygate-acceptance-copy only in an offline/restored copy" >&2; exit 70; }

WORK="$(mktemp -d -t paygate-v2-acceptance-XXXXXX)"
trap 'rm -rf "$WORK"' EXIT
COPY="$WORK/pb_data"
cp -a "$SOURCE" "$COPY"
chmod -R u+rwX "$COPY"
DB="$COPY/data.db"

integrity() {
  local result
  result="$(sqlite3 "$DB" 'PRAGMA integrity_check;')"
  [[ "$result" == "ok" ]] || { echo "integrity_check failed: $result" >&2; exit 1; }
}

query_or_zero() {
  local sql="$1"
  sqlite3 "$DB" "$sql" 2>/dev/null || echo 0
}

snapshot() {
  local prefix="$1"
  {
    echo "payments=$(query_or_zero 'SELECT count(*) FROM payments;')"
    echo "sms_events=$(query_or_zero 'SELECT count(*) FROM sms_events;')"
    echo "email_events=$(query_or_zero 'SELECT count(*) FROM email_events;')"
    echo "notification_events=$(query_or_zero 'SELECT count(*) FROM notification_events;')"
    echo "refunds=$(query_or_zero 'SELECT count(*) FROM refunds;')"
    echo "reviews=$(query_or_zero 'SELECT count(*) FROM review_cases;')"
    echo "relay_devices=$(query_or_zero 'SELECT count(*) FROM relay_devices;')"
    echo "relay_events=$(query_or_zero 'SELECT count(*) FROM relay_events;')"
    echo "webhook_deliveries=$(query_or_zero 'SELECT count(*) FROM webhook_deliveries;')"
    echo "duplicate_rrn=$(query_or_zero "SELECT count(*) FROM (SELECT rrn FROM payments WHERE trim(coalesce(rrn,'')) <> '' GROUP BY rrn HAVING count(*) > 1);")"
    echo "duplicate_evidence_reference=$(query_or_zero "SELECT count(*) FROM (SELECT evidence_reference FROM payments WHERE trim(coalesce(evidence_reference,'')) <> '' GROUP BY evidence_reference HAVING count(*) > 1);")"
    echo "duplicate_idempotency=$(query_or_zero "SELECT count(*) FROM (SELECT idempotency_key FROM payments WHERE trim(coalesce(idempotency_key,'')) <> '' GROUP BY idempotency_key HAVING count(*) > 1);")"
  } > "$WORK/$prefix.txt"
}

integrity
snapshot before

"$BINARY" --dir="$COPY" migrate up >/dev/null

integrity
snapshot after

for key in payments sms_events email_events notification_events refunds reviews relay_devices relay_events webhook_deliveries duplicate_rrn duplicate_evidence_reference duplicate_idempotency; do
  before="$(grep "^${key}=" "$WORK/before.txt" | cut -d= -f2-)"
  after="$(grep "^${key}=" "$WORK/after.txt" | cut -d= -f2-)"
  if [[ "$before" != "$after" ]]; then
    echo "invariant changed during schema migration: $key $before -> $after" >&2
    exit 1
  fi
done

# v2 schema assertions.
shadow_status="$(sqlite3 "$DB" "SELECT count(*) FROM _collections c, json_each(c.fields) f WHERE c.name='relay_events' AND json_extract(f.value,'$.name')='processing_status' AND EXISTS (SELECT 1 FROM json_each(json_extract(f.value,'$.values')) v WHERE v.value='shadow_observed');")"
[[ "$shadow_status" == "1" ]] || { echo "relay_events.processing_status is missing shadow_observed" >&2; exit 1; }
operator_route_migration="$(query_or_zero "SELECT count(*) FROM _migrations WHERE file LIKE '20260828030000%';")"
[[ "$operator_route_migration" == "1" ]] || { echo "Google Messages shadow migration history is missing" >&2; exit 1; }

printf 'production-copy acceptance passed\n'
printf 'workspace was isolated and has been removed automatically\n'
cat "$WORK/after.txt"
