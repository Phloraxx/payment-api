# Security policy

PayGate decides whether money arrived. It sits on a trust boundary between an Android phone reading payment notifications, a server that owns matching, and merchant systems that act on the result.

The design position is deliberate: a false-positive confirmation is worse than a delayed or manual one. Only origin-bound evidence may transition a payment to paid. Anything that breaks that rule is a security bug, not a matching bug.

## Reporting a vulnerability

Use GitHub's **Report a vulnerability** flow for this repository. Private vulnerability reporting is enabled, so details reach the maintainer without a public issue.

If that flow is unavailable, open a minimal issue asking for a private contact path. Do **not** put exploit details in the public issue.

## Never include in a report

- real payer UPI IDs, VPAs, names or phone numbers;
- live payment amounts paired with timestamps that identify a transaction;
- merchant API keys, webhook signing secrets or the admin password;
- relay enrollment codes, pairing epochs or device private keys;
- unredacted notification payloads or screenshots from a live device.

Synthetic amounts, `example.com` hosts and redacted payloads are preferred.

## Security-relevant behaviour

Reports are especially useful for:

- a payment reaching `paid` without origin-bound evidence — for example through a generic, package-agnostic or replayed notification;
- a relay event accepted after revocation, after a pairing-epoch change, or from a different enrollment;
- amount matching that accepts an over-precise, malformed or ambiguous payable value;
- duplicate or replayed webhook delivery that mutates payment state twice;
- cross-merchant or cross-profile data exposure in the merchant API, admin UI or webhook payload;
- payer identity appearing in an unauthenticated or public view;
- plaintext secrets persisted where a hash or verifier would do;
- SQLite ownership ambiguity that lets two writers mutate the live database;
- a migration or restore path that silently drops or reinterprets existing payment state.

## Supported versions

The v4 line is the supported security target. The retired v3/PocketBase runtime is not patched; upgrade instead.

## If a credential may be exposed

Revoke first, investigate second. Rotate the merchant API key or webhook secret, revoke the affected relay enrollment, and treat any payment confirmed after the exposure as unverified until re-checked against the provider.

## Out of scope

- the payer's own bank or UPI app behaviour;
- notification delivery failures caused by Android Doze or OEM battery policy on an unconfigured device;
- anything requiring an already-compromised host with read access to the live data volume.
