# PayGate v3 production rollout — 2026-08-30

## Scope

This record captures the gated production rollout of the PayGate v3 API/operator platform, customer checkout, and Android relay/operator app.

Production services:
- API: `main-payment-17aqux`
- Customer frontend: `main-payment-frontend-t1n1x8`
- API domain: `https://pay.mulearnscet.in`
- Customer domains: `https://payment.mulearnscet.in` and `https://pay.ieeesahrdaya.com`

## Released source and images

- API reviewed source tree: `67bcc8f5e2d178aca251bc0dc26f72b1326909dd`
- API canonical main merge: `9acce4fc93090c9a7fa245f7a20c8d446e36fef5`
- Both commits have identical Git trees: `02f64e29556fa180e40e9744718f3f3ed43ee331`
- API production image: `main-payment-17aqux:v3-67bcc8f5`
- Customer frontend merge: `e08d599a23b901a640d1be228d1d66dc44465cad`
- Customer production image: `main-payment-frontend-t1n1x8:v3-e08d599a`
- Android main merge: `0bcf2156d6ee7bd4afc18b423005f8b921eabb27`
- Android release: `0.4.0-alpha1`, versionCode `8`

## Backup and rollback gates

Before the v3 API deployment, PayGate created a fresh production backup through its own backup command:

- backup: `paygate_manual_20260830_071509.zip`
- size: `3815559` bytes
- exported checksum: verified
- restore drill: passed for both active database files

Rollback metadata was captured before service changes. API and frontend remained independently rollbackable, and the API deployment retained one replica with stop-first semantics.

## Pre-deploy v2 SQLite incident

The mandatory baseline correctly blocked deployment when the still-running v2 process reported `db:error` and checkout readiness returned HTTP 502.

Logs showed repeated SQLite `disk I/O error (522)` on ordinary reads. The fresh online backup and restore drill still passed, which indicated the stored databases were readable and internally consistent.

The exact same pinned v2 image was restarted with stop-first semantics. Fresh database handles immediately restored:
- `db=ok`
- checkout readiness HTTP 200
- connector connected
- no new SQLite 522 errors in the replacement task

No database restore or direct live-database mutation was required.

## API and customer acceptance

The v3 API candidate passed immediate and post-cron production gates:
- database healthy
- connector connected
- relay ready
- checkout origin allowlist preserved
- denied origins still rejected
- trusted write API remained private
- new operator endpoints remained private anonymously
- no new task/database errors

The customer frontend passed both-domain verification with the same static assets, security headers, SPA routing, and checkout API connectivity.

Harmless unpaid production smoke sessions were created and immediately cancelled on Kotak, Slice, and Paytm. Public payment-status responses remained redacted. No real payment was made.

## Android in-place upgrade

The installed phone was reached over Tailscale ADB at `100.99.14.85:5555` and independently verified before upgrade:
- package: `io.github.phloraxx.paygaterelay`
- previous version: `0.3.1`, versionCode `7`
- previous installed APK SHA-256: `a971b4ffa09e41877bc1b5858b1a427d743f0ad62ba5249dfd640012f4161e72`
- stable signer SHA-256: `412b8f66c06acd93958c4dd11caa3214ba950059c00e57159d5673f94700d44a`

The verified v0.4 release APK SHA-256 was `b55ef0f50309f3ac370b1ad6d8a32168b372dc03df6dd47ee2d6d0fbd06b687b`.

The update used only `adb install -r`. It preserved:
- first-install lineage and device identity
- notification-listener access
- battery-optimization exemption
- background execution allowance
- foreground relay service
- pairing and signed heartbeat

After upgrade the same relay device reported `0.4.0-alpha1`, `power_health_reported=1`, `battery_optimization_exempt=1`, `background_restricted=0`, `foreground_service_active=1`, and no client error.

## Locked + Battery Saver acceptance

With the phone locked and dozing, Battery Saver was enabled for a controlled acceptance check. The app remained battery-whitelisted and its foreground relay/listener services stayed active.

The app launcher path was invoked without bypassing keyguard; `PayGateActivity` called the relay's normal `ensureRunning()`/`kick()` path. The server received a fresh signed heartbeat while keyguard remained active, and persisted telemetry reported `power_save_mode=1` with the foreground service active and no client error.

The relay also has a 15-minute WorkManager safety pass. Autonomous-heartbeat observation should be checked after that interval when validating long-idle behavior.

## Historical local queue item

One historical failed local item remains on the phone: a GPay observation that previously received HTTP 401. It is not a pending Paytm relay item. Pending queue is zero. The record was intentionally retained rather than deleting history to make the counter zero.

## Remaining acceptance

- visually exercise authenticated Android Home / Payments / Action / Health after normal user unlock
- verify operator token refresh does not interrupt relay operation
- observe the 15-minute background safety heartbeat while locked/Battery Saver is active
- restore Battery Saver to the user's original off state after the controlled test
- continue ordinary production soak monitoring
- Razorpay Live remains out of scope without explicit approval
