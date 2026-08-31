# 05 — Unified Android App

## Product identity

The Android application is simply **PayGate**.

It has two responsibilities inside one APK:

1. operator UI;
2. background notification relay.

There must not be a second separately installed “Relay” app.

Preserve the existing Android application ID and signing lineage during the redesign so production can update in place without uninstalling, clearing data or re-pairing.

## Navigation

Target primary navigation:

```text
Overview
Payments
Activity
Settings
```

No primary tab named Action, Review, Reconciliation or Health.

### Overview

Show:

- collected today
- paid transaction count
- pending count
- recent payments
- active collection profile
- one compact monitoring-state indicator

Do not lead with internal alert counts.

### Payments

- search
- filter
- payment rows with amount/name/status/time
- payment detail
- direct edit for allowed fields
- history timeline

### Activity

A chronological operational stream:

- incoming payment observations
- matched/unmatched notifications
- payment state changes
- operator edits
- webhook success/failure
- device connect/disconnect events when useful

Technical details can live behind an expandable row.

### Settings

Sections:

```text
Collection
  Active profile
  Paytm destination
  Kotak destination

Integrations
  Merchant API key
  Webhook URL/secret

PayGate device
  Monitoring status
  Notification access
  Battery/background status
  Last server contact
  Re-pair / revoke

Security
  Change admin password

Advanced
  local failed relay deliveries
  app/server version
```

## Theme

Dark-only initial design, matching the web dashboard's Razorpay-inspired navy/blue system.

Suggested Android tokens:

```text
background       #07111F
surface          #0C1728
surface_elevated #12213A
border           #213653
primary          #2F80FF
primary_hover    #4A91FF
text             #F7FAFF
text_muted       #8FA4C1
success          #2DCB8A
warning          #F5B84B
danger           #F0616A
```

These are PayGate tokens, not copied Razorpay source values.

Use large monetary typography, compact secondary metadata and restrained cards rather than the current oversized lime warning hero.

## Relay lifecycle remains independent of operator login

Background payment observation must continue when:

- admin session expires;
- user closes the PayGate UI;
- screen is locked;
- device is in Doze;
- Android process is recreated;
- network temporarily disappears.

Operator authentication controls dashboard access only.

## Notification access

Continue using `NotificationListenerService`.

Initial allowlist:

```text
com.paytm.business
com.google.android.apps.messaging
```

The service should capture only notifications passing the generic decimal-money prefilter described in `02_NOTIFICATION_INGESTION_AND_PAIRING.md`.

Do not request direct SMS permissions.

## Local storage

The Android app needs only a small local database for relay delivery state and recent diagnostics.

Suggested tables:

```text
relay_events
  event_id
  package_name
  posted_at
  title/text/big_text (bounded)
  amount_hint
  state
  attempts
  last_http_status
  last_error
  created_at
  delivered_at
```

Do not locally mirror the PayGate payments database. Operator payment data comes from the server.

## Queue state

```text
pending
retry
failed
delivered
```

A new notification that passes the prefilter is inserted before network delivery. This makes process death safe.

Delivery is idempotent by event ID.

## Background reliability

Keep the proven v0.4.x model:

- notification listener
- foreground runtime service
- WorkManager fallback
- 15-minute `AlarmManager.setAndAllowWhileIdle()` heartbeat/watchdog
- bounded wake lock only around real delivery work
- boot recovery
- battery-optimization exemption status

Do not add a permanent aggressive wake lock.

## Pairing UX

### New phone

1. Install/open PayGate.
2. Intro screen explains that notification access is required to detect incoming payments.
3. User scans the QR generated from PayGate web Settings using the phone's normal camera.
4. Android App Link opens PayGate with the one-time token.
5. PayGate generates its device signing key in Android Keystore if one does not exist.
6. App enrolls public key with the one-time token.
7. App guides the user to enable notification access.
8. App checks/request battery optimization exemption.
9. Monitoring state becomes Active.

No server URL, pairing secret or device ID should normally be typed by the user.

### Existing production phone

Migration must preserve:

- existing ECDSA private key
- device ID
- notification-listener grant
- battery exemption
- local pending/failed event database
- app data

If a v4 migration can reuse the current enrolled public key, it must do so rather than forcing QR re-pairing.

## Device credential storage

Keep the ECDSA private key non-exportable in Android Keystore. The server stores the public key.

Do not configure the relay signing key to require an unlock for every signature; background delivery has to function while the device is locked. The operator UI can optionally use biometrics later, but relay signing must remain background-capable.

## Operator authentication

Login screen contains only:

```text
Password
[ Continue ]
```

Internally the server has one fixed administrator identity.

The Android app stores the returned operator session credential using platform-protected app storage. Operator token expiry must sign the UI out without touching relay enrollment.

## Payment detail editing

Use the same fields/rules as web:

Editable:

- status
- name
- external ID
- payer name
- payer UPI ID
- paid time
- metadata/note

Immutable:

- PayGate ID
- created time
- requested amount
- payable amount
- collection-profile/destination snapshot

Destructive/status edits should use a confirmation sheet with a concise description of resulting webhook behavior.

## Activity detail for an incoming notification

Operator-facing language:

```text
Kotak payment detected
₹100.37
Rahul / rahul@upi
00:33
Matched to pay_...
```

or:

```text
Incoming payment not matched
₹100.37
Kotak
00:33
```

Do not expose terms such as `evidence_reference`, `processing_status` or `reconciliation` in the primary UI.

## Tests

- package allowlist
- generic non-zero decimal-money filter
- `.00` notification rejected by prefilter
- unrelated Google Messages notification never queued
- event persisted before delivery
- duplicate event ID remains one local row
- listener/service recovery after reboot
- heartbeat alarm re-arm/cancel
- pairing App Link parsing
- one-time enrollment success/failure
- existing key is reused across app upgrade
- operator logout does not stop relay
- dark theme screens render without light-theme fallback
- server unreachable -> retry -> recovery
- 401 device auth -> visible failed state, no infinite retry