const MAX_RAW_BYTES = 2 * 1024 * 1024;

export default {
  async email(message, env) {
    if (!env.PAYGATE_EMAIL_WEBHOOK_URL || !env.PAYGATE_EMAIL_WEBHOOK_SECRET || !env.PAYGATE_EMAIL_RECIPIENT || !env.PAYGATE_EMAIL_FALLBACK_RECIPIENT) {
      throw new Error("PayGate email worker is not fully configured");
    }
    if (message.to.toLowerCase() !== env.PAYGATE_EMAIL_RECIPIENT.toLowerCase()) {
      message.setReject("Unknown payment notification recipient");
      return;
    }
    if (message.rawSize > MAX_RAW_BYTES) {
      message.setReject("Payment notification exceeds 2 MiB");
      return;
    }

    const raw = new Uint8Array(await new Response(message.raw).arrayBuffer());
    if (raw.byteLength > MAX_RAW_BYTES) {
      message.setReject("Payment notification exceeds 2 MiB");
      return;
    }
    const messageId = (message.headers.get("Message-ID") || "").trim().replace(/^<|>$/g, "");
    const body = JSON.stringify({
      sourceId: messageId,
      envelopeFrom: message.from,
      envelopeTo: message.to,
      receivedAt: new Date().toISOString(),
      rawEmailBase64: bytesToBase64(raw),
    });
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const signature = await sign(env.PAYGATE_EMAIL_WEBHOOK_SECRET, `${timestamp}.${body}`);
    let failure = "network error";
    for (let attempt = 1; attempt <= 3; attempt += 1) {
      try {
        const response = await fetch(env.PAYGATE_EMAIL_WEBHOOK_URL, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-PayGate-Timestamp": timestamp,
            "X-PayGate-Signature": `sha256=${signature}`,
          },
          body,
        });
        if (response.ok) return;
        failure = `HTTP ${response.status}`;
        if (response.status < 500 && response.status !== 408 && response.status !== 429) break;
      } catch (error) {
        failure = error instanceof Error ? error.message : "network error";
      }
    }

    console.error(`PayGate email ingestion failed: ${failure}`);
    const fallbackHeaders = new Headers();
    fallbackHeaders.set("X-PayGate-Ingestion-Failed", "true");
    fallbackHeaders.set("X-PayGate-Ingestion-Error", failure.slice(0, 200));
    await message.forward(env.PAYGATE_EMAIL_FALLBACK_RECIPIENT, fallbackHeaders);
  },
};

async function sign(secret, value) {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const bytes = new Uint8Array(await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(value)));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function bytesToBase64(bytes) {
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
  }
  return btoa(binary);
}
