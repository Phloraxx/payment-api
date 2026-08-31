#!/usr/bin/env bash
set -euo pipefail
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
VOLUME=main-payment-17aqux-pb-data
DEST=/home/drvij/paygate-backups/daily
LOCK=/tmp/paygate-backup-export.lock

# Important: never exec the PayGate binary while the production server owns
# pb_data. A second PocketBase process against the same SQLite directory can
# poison the live process' database handles. Export only completed backup ZIPs.
exec 9>"$LOCK"
flock -n 9 || exit 0
mkdir -p "$DEST"
uid=$(id -u drvij)
gid=$(id -g drvij)

docker run --rm -i \
  -e OWNER_UID="$uid" -e OWNER_GID="$gid" \
  -v "$VOLUME:/source:ro" \
  -v "$DEST:/dest" \
  python:3.12-alpine python - <<'PY'
import hashlib, os, pathlib, shutil, sqlite3, tempfile, zipfile
source=pathlib.Path('/source/backups')
dest=pathlib.Path('/dest')
files=sorted(source.glob('*.zip'), key=lambda p:p.stat().st_mtime, reverse=True)
if not files:
    raise SystemExit('no PocketBase backup archive found')
latest=files[0]
target=dest/latest.name
temp=dest/(latest.name+'.tmp')
shutil.copy2(latest,temp)
os.replace(temp,target)

with zipfile.ZipFile(target) as archive:
    bad=archive.testzip()
    if bad is not None:
        raise SystemExit(f'backup ZIP CRC failure at {bad}')
    with tempfile.TemporaryDirectory() as td:
        archive.extractall(td)
        checked=0
        for name in ('data.db','auxiliary.db'):
            db=pathlib.Path(td)/name
            if not db.exists():
                continue
            con=sqlite3.connect(f'file:{db}?mode=ro', uri=True)
            result=con.execute('PRAGMA integrity_check').fetchone()[0]
            con.close()
            if result != 'ok':
                raise SystemExit(f'{name} integrity_check={result}')
            checked += 1
        if checked != 2:
            raise SystemExit(f'expected 2 root SQLite databases, verified {checked}')

h=hashlib.sha256()
with target.open('rb') as f:
    for chunk in iter(lambda:f.read(1024*1024),b''):
        h.update(chunk)
sha=dest/(latest.name+'.sha256')
sha.write_text(f'{h.hexdigest()}  {latest.name}\n')
uid=int(os.environ['OWNER_UID']); gid=int(os.environ['OWNER_GID'])
for path in (target,sha):
    os.chmod(path,0o600); os.chown(path,uid,gid)

exports=sorted(dest.glob('*.zip'), key=lambda p:p.stat().st_mtime, reverse=True)
for old in exports[30:]:
    old.unlink(missing_ok=True)
    (dest/(old.name+'.sha256')).unlink(missing_ok=True)
print(f'verified {target.name} ({target.stat().st_size} bytes)')
PY

logger -t paygate-backup-export "verified and exported latest PayGate backup without starting a second PayGate process"
