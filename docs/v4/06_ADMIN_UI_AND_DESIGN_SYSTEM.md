# 06 — Admin UI and Design System

## UX goal

The web dashboard and Android app should feel like two views of the same PayGate product.

The operator should be able to answer these questions immediately:

- How much money came in today?
- How many payments succeeded?
- Which payments are pending/expired?
- Who paid a specific payment, when, and for how much?
- Which collection profile is active now?
- Is the Android payment monitor connected?
- Did a webhook fail?

Everything else is secondary detail.

## Primary navigation

```text
Overview
Payments
Activity
Settings
```

This replaces the current collection of operational pages around Reviews, Reconciliation, SMS/Email records and connector-specific state.

## Visual direction

Use a Razorpay-inspired **dark navy + blue** financial interface:

- deep navy, not pure black;
- clear blue primary actions;
- high-contrast white financial values;
- blue-gray secondary text;
- small-radius dense data tables on desktop;
- responsive card/list equivalents on Android/mobile;
- restrained use of gradients;
- status colors used only where status matters.

Do not copy Razorpay logos, illustrations, wording or exact screens.

Razorpay's own product surfaces emphasize configurable branded color and a strongly blue financial identity; use that as visual inspiration rather than as a component library.

Reference: https://razorpay.com/docs/payments/dashboard/account-settings/checkout-styling/

## Proposed color tokens

```text
--pg-bg:             #07111F
--pg-bg-subtle:      #091525
--pg-surface:        #0D1A2C
--pg-surface-2:      #12233B
--pg-border:         #213958
--pg-border-soft:    #182C47
--pg-primary:        #2F80FF
--pg-primary-hover:  #4A91FF
--pg-primary-soft:   #15345D
--pg-text:           #F7FAFF
--pg-text-secondary: #B1C0D5
--pg-text-muted:     #8096B5
--pg-success:        #2DCB8A
--pg-warning:        #F5B84B
--pg-danger:         #F0616A
```

Use blue as the product accent. Retire the current lime as the primary brand color for v4.

## Typography

Use a clean system-first sans stack to avoid font-loading fragility:

```text
Inter, ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif
```

Monetary values use tabular numerals where possible.

Hierarchy:

- amount/KPI: large, heavy, tight tracking
- page title: medium-large
- table primary value: 14–16px semibold
- metadata: 12–14px muted
- labels: compact, not all-caps everywhere

## Desktop shell

```text
+---------------------------------------------------------------+
| PayGate | Overview | Payments | Activity | Settings           |
+---------------------------------------------------------------+
|                                                               |
| page content                                                  |
|                                                               |
+---------------------------------------------------------------+
```

Prefer a compact top/side navigation with enough room for data. Avoid oversized marketing-style hero cards inside an operator dashboard.

## Overview

Top row:

```text
Collected today      Payments today      Pending      Expired
₹48,240.63           119                 3            2
```

Second row:

- volume trend by hour/day;
- success/pending/expired breakdown;
- active collection profile chip.

Then recent payments table.

A small status strip at the bottom/right can show:

```text
PayGate phone: Connected · last seen 1m ago
Webhook: Healthy
Backup: Last verified 6h ago
```

Do not turn system-health messages into the largest visual object on the page unless the system is actually unable to collect payments.

## Payments table

Columns:

```text
Payment
Name
Requested
Paid/Payable
Status
Payer
Created
Paid
```

Desktop table supports column sorting where useful.

Search field should search:

- PayGate payment ID
- external ID
- name/alias
- payer name
- payer UPI ID
- exact amount
- metadata text where indexed safely

Filters:

- status
- date range
- requested/payable amount range
- collection profile

Persist filter state in URL query parameters for shareable/reloadable admin views.

## Payment detail

Header:

```text
₹100.37                                  PAID
IEEE workshop registration
pay_01J...
```

Summary grid:

```text
Requested        ₹100.00
Payable/Paid     ₹100.37
Adjustment       ₹0.37
Payer            Rahul Kumar
UPI ID           rahul@okaxis
Paid at          1 Sep, 12:33 AM
Created          1 Sep, 12:30 AM
Collection       Paytm
External ID      reg_284
```

Timeline:

```text
12:30:00 Payment created
12:33:08 Incoming Paytm payment detected
12:33:08 Payment marked paid
12:33:09 Webhook delivered (200)
```

Technical notification text should be inside an expandable Activity entry, not clutter the payment summary.

## Edit payment

One edit drawer/modal instead of a separate manual-review workflow.

Editable fields:

```text
Status
Name
External ID
Payer name
Payer UPI ID
Paid at
Metadata
Internal note
```

When status changes, display exactly what will happen:

```text
Changing status from Pending to Paid will create a payment.paid webhook event.
```

Always append an immutable history record recording old/new values and timestamp.

## Activity

Activity is the home for everything that happened around money without forcing the operator to learn implementation terms.

Rows can be:

```text
Payment detected · Paytm · ₹100.37 · matched
Payment detected · Kotak · ₹501.42 · unmatched
Payment updated · pay_... · operator
Webhook delivered · payment.paid · 200
Webhook failed · payment.expired · 404
Device connected · Edge 60 Stylus
```

Filters can expose type/source/status when needed.

An unmatched payment observation is not a “Review Case”. It is simply an Activity item. The operator can inspect it and edit the relevant payment if necessary.

## Settings

### Collection

Show two profile cards initially:

```text
Paytm                  ACTIVE
UPI: ...
[Make active disabled]

Kotak
UPI: ...
[Make active]
```

Switch confirmation:

> New payments will use Kotak. Existing Paytm payments keep their current destination and remain matchable.

The active profile change must be one click + concise confirmation.

### Integrations

- merchant API key: create/rotate
- webhook destination URL
- webhook secret: create/rotate
- last successful webhook
- retry one selected failed delivery

### Device

```text
PayGate Android        Connected
Last seen              1 minute ago
Notification access    Enabled
Background             Unrestricted
App version            0.x / v4
```

Actions:

- Connect phone (shows pairing QR)
- Revoke device

No Google Messages connector pairing section in target v4.

### Security

- Change admin password
- Active admin sessions with revoke-all (optional if trivial)

## Login

Only one field:

```text
PayGate

Password
[••••••••••]

Continue
```

No username/email selector because the deployment has one operator identity.

Rate-limited failure messages remain generic.

## Responsive behavior

The web dashboard must be genuinely usable from a phone even though Android is the preferred mobile operator surface.

- tables collapse to transaction cards under narrow widths;
- filters become a bottom sheet/drawer;
- payment actions remain reachable without horizontal scrolling;
- keep amounts/status visible in the first card row.

## Accessibility

- maintain WCAG-level text contrast across dark surfaces;
- visible keyboard focus ring in blue/white;
- status must not be communicated by color alone;
- tables expose semantic headers;
- interactive rows must still have explicit buttons/links for keyboard/screen-reader access;
- respect reduced-motion preference.

## Anti-patterns to avoid

- giant “35 things need attention” hero as the first dashboard object;
- internal names such as `evidence_reference`, `relay processing status`, `reconciliation item`;
- excessive rounded cards around every line of data;
- decorative gradients behind dense tables;
- separate screens for each notification source;
- exposing Paytm/Kotak choice to merchant/customer checkout;
- light-theme fallback on Android.