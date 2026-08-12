# Razorpay Live Event Payments

This is a separate Live Mode rail. It does not reuse Test Mode collections,
credentials, webhook events, or Docker volume. The backend accepts server-created
orders from ₹1 through ₹1,00,000; Razorpay may apply a lower account-specific
maximum.

## Required protected values

```text
RAZORPAY_LIVE_ENABLED=true
RAZORPAY_LIVE_KEY_ID=rzp_live_...
RAZORPAY_LIVE_KEY_SECRET=...
RAZORPAY_LIVE_WEBHOOK_SECRET=...
PAYGATE_API_KEY=<separate internal portal key>
```

The public portal reaches the service only over the private Dokploy network.
The service must not publish a host port.

## Live webhook

Configure in the Razorpay Dashboard while switched to Live Mode:

```text
https://pay.ieeesahrdaya.com/api/razorpay/live/webhook
```

Subscribe to `payment.authorized`, `payment.captured`, and `payment.failed`.
Use a separate webhook secret, not the API Key Secret.

## Live route

The portal deliberately does not link this route from the home page:

```text
https://pay.ieeesahrdaya.com/razorpay-live
```

The browser never supplies the trusted event amount. The calling application
creates the order server-to-server and the isolated Live backend enforces the
amount range. Only provider state `captured` is treated as successful.
