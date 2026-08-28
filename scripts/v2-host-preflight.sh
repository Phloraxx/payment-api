#!/usr/bin/env bash
set -euo pipefail

SERVICE="${PAYGATE_SERVICE:-main-payment-17aqux}"
API_ORIGIN="${PAYGATE_ORIGIN:-https://pay.mulearnscet.in}"
BACKUP_DIR="${PAYGATE_BACKUP_DIR:-/home/drvij/paygate-backups/daily}"
CHECKOUT="${1:-disabled}"
ALLOWED_ORIGIN="${2:-https://payment.mulearnscet.in}"

fail() { echo "preflight failed: $*" >&2; exit 1; }
code() { curl -sS -o /tmp/paygate-v2-preflight-body -w '%{http_code}' --max-time 10 "$@"; }

health="$(curl -fsS --max-time 10 "$API_ORIGIN/api/paygate/health")"
python3 - "$health" <<'PY'
import json,sys
body=json.loads(sys.argv[1])
assert body.get('db') == 'ok', body
assert body.get('ready') is True, body
assert body.get('relay',{}).get('ready') is True, body
print('live_health=ready')
PY

[[ "$(code -X POST -H "Content-Type: application/json" --data "{}" "$API_ORIGIN/api/payments")" == "401" ]] || fail "trusted payments endpoint is not anonymous-401"
qr_code="$(code -X POST "$API_ORIGIN/api/connector/gmessages/pair/qr")"
[[ "$qr_code" == "401" || "$qr_code" == "404" ]] || fail "unexpected QR route status: $qr_code"

service_json="$(docker service inspect "$SERVICE")"
python3 - "$service_json" <<'PY'
import json,sys
svc=json.loads(sys.argv[1])[0]
mode=svc['Spec'].get('Mode',{}).get('Replicated',{})
assert mode.get('Replicas') == 1, mode
mounts=svc['Spec']['TaskTemplate']['ContainerSpec'].get('Mounts',[])
assert any(m.get('Type')=='volume' and m.get('Target')=='/app/pb_data' for m in mounts), mounts
print('service_shape=single_replica_persistent_volume')
PY

case "$CHECKOUT" in
  disabled)
    [[ "$(code "$API_ORIGIN/api/checkout/v2/payment-accounts")" == "404" ]] || fail "checkout should be disabled"
    ;;
  enabled)
    headers="$(mktemp -t paygate-v2-cors-XXXXXX)"
    checkout_code="$(curl -sS -D "$headers" -o /tmp/paygate-v2-preflight-body -w '%{http_code}' --max-time 10 -H "Origin: $ALLOWED_ORIGIN" "$API_ORIGIN/api/checkout/v2/payment-accounts")"
    [[ "$checkout_code" == "200" ]] || fail "allowed-origin checkout returned HTTP $checkout_code"
    grep -Fqi "Access-Control-Allow-Origin: $ALLOWED_ORIGIN" "$headers" || fail "checkout CORS allow-origin mismatch"
    rm -f "$headers"
    [[ "$(code -H "Origin: https://denied.invalid" "$API_ORIGIN/api/checkout/v2/payment-accounts")" == "404" ]] || fail "foreign checkout origin was not denied"
    ;;
  *) fail "usage: $0 [disabled|enabled] [allowed-origin]" ;;
esac

latest_sidecar="$(find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.zip.sha256' -printf '%T@ %p\n' | sort -nr | head -n1 | cut -d' ' -f2-)"
[[ -n "$latest_sidecar" && -f "$latest_sidecar" ]] || fail "no backup checksum sidecar found"
(
  cd "$BACKUP_DIR"
  sha256sum -c "$(basename "$latest_sidecar")" >/dev/null
)
archive="${latest_sidecar%.sha256}"
unzip -t "$archive" >/dev/null
work="$(mktemp -d -t paygate-v2-preflight-XXXXXX)"
trap 'rm -rf "$work" /tmp/paygate-v2-preflight-body' EXIT
unzip -q "$archive" -d "$work"
for db in data.db auxiliary.db; do
  [[ -f "$work/$db" ]] || fail "$db missing from latest backup"
  [[ "$(sqlite3 "$work/$db" 'PRAGMA integrity_check;')" == "ok" ]] || fail "$db integrity check failed"
done

echo "backup_restore_integrity=ok"
echo "checkout_state=$CHECKOUT"
echo "preflight=passed"
