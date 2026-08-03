# Razorpay Live ₹1 Pilot

This is a separate Live Mode rail. It does not reuse Test Mode collections,
credentials, webhook events, or Docker volume. During the pilot, the backend
accepts only an exact ₹1 order.

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

## Pilot route

The portal deliberately does not link this route from the home page:

```text
https://pay.ieeesahrdaya.com/razorpay-live
```

The browser can create only ₹1. The portal and the isolated Live backend both
enforce that cap. Only provider state `captured` is displayed as successful.
