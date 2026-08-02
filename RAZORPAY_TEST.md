# Razorpay Test Rail

This module is an isolated experiment. It does not modify PayGate's existing
SMS/DDM `payments` records and cannot be enabled with a Razorpay Live Mode key.

## What it implements

- server-side Razorpay Orders API calls;
- operator-only Standard Checkout launch;
- mandatory server-side checkout-signature verification;
- signed `payment.captured` and `payment.failed` webhooks;
- duplicate webhook-event protection using `X-Razorpay-Event-Id`;
- monotonic local state so a late failure cannot downgrade a captured payment;
- provider status refresh using the Fetch Payment API;
- separate `razorpay_test_orders` and `razorpay_test_events` collections;
- no storage of complete webhook payloads or customer payment details.

Only a `captured` order is treated as successfully paid. A verified browser
callback by itself remains `verification_pending` or `authorized` until the
provider status confirms capture.

## Start an isolated instance

```bash
cp .env.razorpay-test.example .env.razorpay-test
# Edit .env.razorpay-test locally; never commit it.
docker compose -f docker-compose.razorpay-test.yml up --build
```

The service binds to `127.0.0.1:3001` and uses the separate
`paygate_razorpay_test_data` volume. Do not point it at the production volume.

Create an operator account through PocketBase administration, sign in to the
PayGate operator UI, and open `#/razorpay_test`.

## Razorpay Dashboard setup

1. Switch the Razorpay Dashboard to **Test Mode**.
2. Generate Test Mode API keys.
3. Put the `rzp_test_...` Key ID and Key Secret into the staging environment.
4. Generate a separate random webhook secret.
5. Configure an HTTPS staging webhook URL:

```text
https://<staging-host>/api/razorpay/test/webhook
```

6. Subscribe only to:

```text
payment.captured
payment.failed
```

The connected ChatGPT Razorpay plugin is read-only and is not a substitute for
these API credentials or webhook configuration.

## Test flow

1. Create a ₹1.00 order from the operator page.
2. Checkout opens with the server-created Razorpay order ID.
3. Complete or fail the mock Test Mode payment.
4. The browser callback is signature-verified by PayGate.
5. PayGate fetches the payment state immediately.
6. The signed webhook independently confirms the final state.

Suggested scenarios:

- successful test UPI/payment;
- failed test payment;
- modified callback order, payment or signature;
- duplicate webhook event ID;
- `payment.failed` delivered after `payment.captured`;
- backend restart between checkout and webhook;
- webhook temporarily unavailable and later retried;
- same `Idempotency-Key` submitted twice;
- Live Mode key supplied to the test configuration (startup must fail).

## Deliberate limitations

- no Live Mode support;
- no refunds or captures initiated by PayGate;
- no generic payment-provider interface;
- no customer-facing production checkout route;
- no automatic migration of Razorpay test orders into normal PayGate payments;
- no raw webhook-payload retention.

## Public IEEE portal proxy

The approved public website is `https://pay.ieeesahrdaya.com`. The customer browser must not call the isolated Razorpay service directly. The maintained `payment-frontend` Hono server proxies only the customer-safe config/create/status/verify routes with a separate server API key. Razorpay sends the raw signed webhook through the same approved domain:

```text
https://pay.ieeesahrdaya.com/api/razorpay/test/webhook
```

The isolated Razorpay service accepts either an operator session or `PAYGATE_API_KEY` for config/order operations. The webhook remains authenticated exclusively by `X-Razorpay-Signature` over the original raw body.
