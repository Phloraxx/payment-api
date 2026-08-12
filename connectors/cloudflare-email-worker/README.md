# Cloudflare Email Routing connector

This Worker delivers the original RFC 822 message to PayGate over a signed HTTPS request. PayGate independently checks the configured `From` address and trusted `Authentication-Results` (DKIM or DMARC) before treating the message as payment evidence.

1. Copy `wrangler.toml.example` to `wrangler.toml` and set the public PayGate URL, dedicated recipient address, and a separately monitored verified fallback inbox.
2. Generate one random secret of at least 24 characters. Put the same value in the Worker with `npx wrangler secret put PAYGATE_EMAIL_WEBHOOK_SECRET` and in PayGate as `PAYMENT_EMAIL_WEBHOOK_SECRET`. Do not commit it.
3. Deploy with `npx wrangler deploy`.
4. Verify the fallback inbox in Cloudflare Email Routing, then route only the dedicated payment recipient address to this Worker.
5. In Gmail, verify the dedicated Cloudflare recipient as a forwarding address, then create a narrow filter for `from:(noreply@slice.bank.in)` and Slice credit subjects. Do not forward the whole mailbox.
6. Send a small real Slice test payment and confirm the Email Events record shows trusted authentication and matches only a Slice payment.
7. Leave `PAYMENT_EMAIL_ENABLED=false` until that end-to-end test passes; then enable it and keep Kotak as the default until Slice is deliberately selected.

The Worker rejects wrong recipients and messages over 2 MiB. It makes up to three webhook attempts; if PayGate is still unavailable, it forwards the original message to the verified fallback inbox with `X-PayGate-Ingestion-Failed: true` for recovery. PayGate also applies its own body limit, five-minute anti-replay window, HMAC verification, Message-ID deduplication, sender authentication, timestamp eligibility, and exact amount/RRN matching.

To recover a message from the fallback inbox, download it as an original `.eml` file and run:

```bash
PAYGATE_EMAIL_WEBHOOK_URL=https://payments.example.org/api/events/email \
PAYMENT_EMAIL_WEBHOOK_SECRET='the-same-secret' \
node connectors/cloudflare-email-worker/replay-email.mjs failed-message.eml
```

The replay keeps the original Message-ID and bank `Date`, so event deduplication and stale-evidence protection still apply. Do not paste the secret or raw message into tickets, logs, or chat.
