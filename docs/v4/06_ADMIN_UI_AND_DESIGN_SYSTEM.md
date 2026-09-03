# 06 — Admin UI and Design System

## UX goal

Web dashboard and Android app are two views of the same PayGate product.

The operator should immediately answer:

- How much money came in today?
- How many payments succeeded/pending/expired?
- Which person/payment is this?
- Who actually paid it, when and for how much?
- Which event does it belong to?
- Which collection profile is active for new payments?
- Is the PayGate phone connected?
- Did a webhook fail?

Everything else is secondary.

## Primary navigation

```text
Overview
Payments
Activity
Settings
```

No primary products named Review, Reconciliation, Evidence, SMS Events or Connector Management.

## Visual direction

PayGate uses an original **charcoal + emerald/teal** financial interface:

- near-black charcoal backgrounds rather than flat black;
- emerald primary actions and healthy-state emphasis;
- teal/cyan secondary accents for charts and routing information;
- high-contrast monetary typography;
- muted green-gray metadata;
- dense desktop financial tables;
- responsive mobile cards and a four-item bottom navigation;
- restrained gradients, borders and status colors.

The approved PayGate gateway mark, web dashboard concept and mobile concept are the visual source of truth. External fintech dashboards are reference material for information hierarchy only; do not copy logos, wording or exact screens.

## Color tokens

```text
--pg-bg:             #050B0F
--pg-bg-deep:        #03080B
--pg-surface:        #0C171C
--pg-surface-2:      #102027
--pg-border:         rgba(167, 211, 206, .11)
--pg-primary:        #31E78F
--pg-primary-bright: #58FF72
--pg-secondary:      #10D7A7
--pg-accent:         #0AB6C8
--pg-text:           #EDF6F5
--pg-text-secondary: #C5D4D3
--pg-text-muted:     #8CA09F
--pg-success:        #31E78F
--pg-warning:        #F6C85F
--pg-danger:         #FF6F7D
```

## Typography

Use a robust system-first sans stack:

```text
Inter, ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif
```

Use tabular numerals for money/timestamps where possible.

## Overview

Suggested first row:

```text
Collected today      Payments today      Pending      Expired
₹48,240.63           119                 3            2
```

Then:

- volume trend;
- status breakdown;
- active collection profile;
- recent payments.

Small operational strip:

```text
PayGate phone: Connected · last seen 1m ago
Webhook: Healthy
Backup: Verified 6h ago
```

Operational health should not dominate the page unless collection is actually impaired.

## Payments table

Suggested columns:

```text
Person / Name
Payment ID
Event ID
Requested
Payable/Paid
Status
Actual payer
Created
Paid
```

Important semantics:

- **Person / Name** = merchant-supplied `name`, e.g. `Sourav P Bijoy`;
- **Event ID** = merchant-supplied `external_id`, e.g. `evt_hardware_security_2026`;
- **Actual payer** = notification-derived payer name and may be a different person.

Search should cover:

- PayGate payment ID;
- person/name;
- event `external_id`;
- optional metadata such as registration ID;
- payer name;
- payer UPI ID;
- exact requested/payable amount;
- status/date/profile.

Filters:

- status;
- date range;
- requested/payable amount range;
- event ID;
- collection profile.

Event ID is not unique; filtering one event should naturally show many payments.

Persist filter state in URL query parameters on web.

## Payment detail

Example:

```text
₹100.37                                    PAID
Sourav P Bijoy
pay_01J...

Event ID          evt_hardware_security_2026
Requested         ₹100.00
Payable/Paid      ₹100.37
Adjustment        ₹1.37
Actual payer      Bijoy P
Payer UPI ID      bijoy@okaxis
Paid at           1 Sep, 12:33 AM
Created           1 Sep, 12:30 AM
Collection        Paytm
```

The interface should make `name` versus actual payer visually unambiguous.

If payer data is unavailable, show `Not provided by notification` rather than inventing a value.

Timeline:

```text
12:30:00 Payment created · Paytm · ₹100.37 reserved
12:33:08 Incoming Paytm payment detected
12:33:08 Payment marked paid
12:33:09 payment.paid webhook delivered · 200
```

Technical raw notification text belongs in an expandable Activity/detail section, not the primary summary.

## Random payable amount UX

The operator and test frontend must make the exact amount understandable without exposing allocator jargon.

Preferred wording:

```text
Requested     ₹100.00
Pay exactly   ₹100.37
Adjustment    +₹1.37
```

Do not call the random amount a fee unless the business actually treats it as a fee.

Settings may expose allocator configuration such as maximum adjustment, but normal Payments screens should simply show what was requested and what was payable.

## Edit payment

Use one edit drawer/modal rather than a manual-review subsystem.

Editable:

```text
Status
Name
Event ID (external_id)
Payer name
Payer UPI ID
Paid at
Metadata
Internal note
```

Immutable:

```text
PayGate payment ID
Created time
Requested amount
Generated payable amount
Profile/destination snapshot
```

Changing immutable monetary identity would invalidate the issued payment instruction. Correct flow is cancel + replacement payment.

When status changes, preview the side effect:

```text
Pending -> Paid will append history and queue payment.paid webhook.
```

Every admin edit appends immutable history.

## Activity

Activity represents everything around money without forcing implementation vocabulary.

Examples:

```text
Payment detected · Paytm · ₹100.37 · matched to Sourav P Bijoy
Payment detected · Kotak · ₹501.42 · unmatched
Payment updated · pay_... · operator
Webhook delivered · payment.paid · 200
Webhook failed · payment.expired · 404
Device connected · Edge 60 Stylus
Profile changed · Paytm -> Kotak
```

An ambiguous/unmatched observation is an Activity item, not a separate Review product.

## Settings

### Collection

```text
Paytm                  ACTIVE
UPI: ...

Kotak
UPI: ...
[Make active]
```

Switch confirmation:

> New payments will use Kotak. Existing Paytm payments keep their destination and remain matchable.

### Amount allocation

Expose only settings that are operationally meaningful:

```text
Maximum adjustment        ₹1.99
Hard reservation          15 minutes
Soft recent-use avoidance configurable
```

Avoid exposing random seeds/pool internals.

### Integrations

- merchant API key create/rotate;
- webhook URL;
- webhook signing secret rotate;
- latest webhook health;
- retry one selected exhausted event.

### Device

```text
PayGate Android        Connected
Last seen              1 minute ago
Notification access    Enabled
Background             Unrestricted
App version            ...
```

Actions:

- Connect/Replace phone -> pairing QR;
- Revoke device.

No Google Messages connector pairing section in final v4.

### Storage / backup

Show human-level health only:

```text
Database               Healthy
Last backup            ...
Last verified backup   ...
```

Do not expose WAL checkpoint controls in the normal product UI.

### Security

- change admin password;
- revoke active sessions if needed.

## Login

One field:

```text
PayGate

Password
[••••••••••]

Continue
```

No username/email.

## Responsive behavior

Web dashboard must remain usable on a phone:

- tables become payment cards;
- filters become drawer/sheet;
- amount/status/name stay visible without horizontal scrolling;
- actions remain reachable.

Android remains the preferred mobile operator client.

## Accessibility

- WCAG-level contrast on dark surfaces;
- visible keyboard focus;
- status not color-only;
- semantic table headers;
- explicit controls rather than click-only rows;
- reduced-motion support;
- monetary values readable at large text sizes.

## Anti-patterns

Avoid:

- giant attention hero cards;
- light-theme fallback for the product UI;
- internal terms such as evidence reference/reconciliation state;
- a separate page per notification source;
- treating event `external_id` as unique;
- labeling merchant `name` as an event title;
- hiding the random payable adjustment;
- letting the admin edit the already-issued payable amount;
- exposing Paytm/Kotak routing to customer/merchant checkout.

## Implementation checkpoint — 2026-09-01

The standalone v4 web dashboard is now implemented as a separate React/Vite bundle embedded only in `paygate-v4`. It does not replace or alter the legacy v3 PocketBase-facing bundle before cutover.

Implemented operator surfaces:

- password-only login using an HttpOnly web cookie;
- Overview with money/payment totals, seven-day volume, active profile, phone and webhook health;
- Payments with search/filter/pagination, complete detail, immutable financial snapshots, safe edits, history and per-delivery webhook retry;
- Activity with payment, observation and webhook events including `matched`, `corroborated` and `ambiguous`;
- Settings for collection profiles, webhook endpoint/secret rotation, merchant API keys, phone state/replacement pairing, revocation and admin-password change.

The browser never stores an admin bearer token. Web authentication uses a `Secure`, `HttpOnly`, `SameSite=Strict`, `Path=/admin` cookie. Android continues to use the explicit bearer-token response intended for the mobile client.

The dashboard is deliberately lightweight: no router framework, charting framework or state-management library was added. SPA fallback is handled by the embedded Go static handler, while API routes retain precedence.

The production-shaped disposable-container smoke verified `/health`, dashboard serving, API protection, cookie login, authenticated Overview, `/device/pair/<token>` browser fallback, Android Digital Asset Links and the built-in Docker healthcheck. No production deployment occurred during this validation.
