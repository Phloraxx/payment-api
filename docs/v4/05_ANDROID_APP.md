# 05 — Unified Android App

## Product identity

The Android application is simply **PayGate**.

One APK has two responsibilities:

1. operator UI;
2. background payment-notification relay.

There is no separately installed Relay product.

Preserve the existing package/application ID, signing certificate, app data and device-key lineage so production upgrades in place.

## Navigation

```text
Overview
Payments
Activity
Settings
```

No primary tab named Action, Review, Reconciliation or Health.

## Overview

Show useful payment information first:

- collected today;
- paid count;
- pending count;
- recent payments;
- active collection profile for new payments;
- one compact phone-monitor status.

Do not make internal diagnostics the hero of the product.

## Payments

Search/filter/detail should use the same semantic fields as web:

```text
Name/person      merchant-supplied identifier
Event ID         external_id; may repeat
Actual payer     notification-derived identity
Requested
Payable/Paid
Status
Created/Paid time
```

`name` must never be mislabeled as the event title.

## Activity

Chronological stream for:

- Paytm/Kotak payment detection;
- matched/unmatched notifications;
- payment state changes;
- operator edits;
- webhook success/failure;
- profile/device changes;
- diagnostics behind expandable detail.

No separate review subsystem.

## Settings

```text
Collection
  active Paytm/Kotak profile
  destination configuration

Amount allocation
  maximum adjustment
  reservation/soft reuse policy (advanced)

Integrations
  merchant API key
  webhook URL/secret

PayGate device
  monitoring status
  notification access
  background/battery state
  last server contact
  Connect / Replace / Revoke

Security
  change admin password

Advanced
  failed local relay deliveries
  app/server version
```

## Visual system

Dark-only initial product UI, shared with web.

```text
background       #07111F
surface          #0C1728
surface_elevated #12213A
border           #213653
primary          #2F80FF
text             #F7FAFF
text_muted       #8FA4C1
success          #2DCB8A
warning          #F5B84B
danger           #F0616A
```

Razorpay is inspiration for financial density/blue identity, not a component copy.

Use large money typography, clear hierarchy and restrained surfaces instead of the current oversized lime warning style.

## Relay is independent of operator login

Payment monitoring must continue when:

- admin session expires;
- app UI closes;
- screen locks;
- device enters Doze;
- Android recreates the process;
- network disappears and returns.

Operator auth controls management UI only.

## NotificationListenerService

Initial allowlist:

```text
com.paytm.business
com.google.android.apps.messaging
```

GPay/Slice are deferred.

Phone applies only the generic non-`.00` decimal-money prefilter. It does not decide incoming-credit semantics/profile/payment match.

Do not request direct SMS permissions.

## Stable capture identity

The same Android notification can be seen repeatedly because of listener reconnects/rescans/updates.

Android must retain one stable local event identity for equivalent payment-relevant content.

Suggested local fields:

```text
event_id
package_name
notification_key
posted_at_ms
content_fingerprint
title/text/big_text (bounded)
amount_hint
state
attempts
last_http_status
last_error
created_at
delivered_at
```

A retry keeps the original `event_id` and `posted_at_ms`.

A genuinely changed payment-relevant notification update creates a new immutable version/fingerprint rather than rewriting a delivered event.

## Local queue

```text
pending
retry
failed
delivered
```

Insert before network delivery.

Requirements:

- durable across reboot/process death;
- bounded exponential retry;
- event delivery idempotent;
- 4xx device-auth failures visible rather than retried forever;
- successful delivered rows pruned on short retention;
- failed rows inspectable without blind bulk retry;
- raw notification content retained briefly.

Do not mirror the server payments DB locally. Operator screens fetch server data.

## Background reliability

Keep the proven v0.4.x mechanisms until a simpler alternative proves equally reliable:

- NotificationListenerService;
- foreground runtime service;
- WorkManager fallback;
- 15-minute `AlarmManager.setAndAllowWhileIdle()` watchdog;
- bounded wake lock only around real delivery;
- boot recovery;
- battery-optimization exemption monitoring.

No permanent aggressive wake lock.

## QR/App-Link pairing

### New phone

1. Open PayGate web Settings -> Device -> Connect phone.
2. Server creates a short-lived single-use pairing token and dashboard shows QR.
3. Scan QR with normal phone camera.
4. HTTPS App Link opens PayGate Android.
5. App generates/reuses P-256 ECDSA key in Android Keystore.
6. App submits token + public key + device metadata.
7. Server atomically consumes token and enrolls device.
8. App guides notification-listener access.
9. App verifies battery/background readiness.
10. Monitoring becomes Active.

No manual server URL/device ID/pairing secret in normal UX.

## One active phone

V4.0 assumes one payment-notification phone.

Pairing another should be a **Replace phone** action:

- new phone enrolls successfully first;
- server activates new device and revokes/disables old one atomically or in one controlled flow;
- old historical device record remains for audit;
- avoid two active devices delivering the same payment stream.

## Existing production phone

Upgrade must preserve:

- ECDSA private key;
- device ID;
- notification-listener grant;
- battery exemption;
- local pending/failed queue;
- signing lineage;
- app data.

No uninstall/clear-data/re-pair workaround.

## Device credential

Private ECDSA key remains non-exportable in Android Keystore and usable while locked for background signing.

Server stores public key only.

Admin login/token is separate from relay-device credential.

## Operator login

Single field:

```text
Password
[ Continue ]
```

A 401/expired operator session signs the UI out but does not stop relay/background enrollment.

## Payment detail

Example:

```text
₹100.37              PAID
Sourav P Bijoy
Event: evt_hardware_security_2026

Requested       ₹100.00
Adjustment      +₹1.37
Actual payer    Bijoy P
UPI             bijoy@okaxis
Collection      Paytm
```

If actual payer is unavailable, show that the notification did not provide it.

Editable:

- status;
- name/person;
- event `external_id`;
- payer fields;
- paid time;
- metadata/note.

Immutable:

- PayGate ID;
- created time;
- requested amount;
- generated payable amount;
- profile/destination snapshot.

## Activity examples

```text
Paytm payment detected · ₹100.37 · matched to Sourav P Bijoy
Kotak payment detected · ₹500.42 · unmatched
Payment updated · operator
Webhook delivered · 200
Phone replaced
```

Primary UI should not expose implementation words such as `evidence_reference`, `processing_status` or `reconciliation`.

## Tests

### Notification/queue

- allowlist Paytm + Google Messages only;
- decimal money with non-zero paise passes;
- `.00` is filtered;
- unrelated personal message never queues;
- duplicate listener rescan remains one row;
- retry preserves event ID/post time;
- content update versioning behaves deterministically;
- queue survives reboot/process death;
- auth 4xx fails visibly without infinite retry.

### Pairing/security

- valid one-time App Link enrollment;
- expired/used token rejected;
- existing key reused after app upgrade;
- Replace phone leaves one active relay device;
- revoke blocks signed relay requests;
- operator logout does not stop relay.

### Reliability

- locked + Doze heartbeat;
- Battery Saver;
- listener rebind;
- network loss/recovery;
- server restart;
- watchdog schedule/re-arm/cancel;
- no light-theme fallback.

### UX semantics

- `name` displayed as person/payee identifier;
- event `external_id` displayed separately;
- actual payer displayed separately and may differ;
- random payable adjustment clearly visible;
- immutable amount/profile fields cannot be edited.