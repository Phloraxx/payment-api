import { createHmac } from "node:crypto";
import { readFile } from "node:fs/promises";

const [file] = process.argv.slice(2);
const url = process.env.PAYGATE_EMAIL_WEBHOOK_URL;
const secret = process.env.PAYMENT_EMAIL_WEBHOOK_SECRET;
if (!file || !url || !secret) {
  console.error("Usage: PAYGATE_EMAIL_WEBHOOK_URL=https://... PAYMENT_EMAIL_WEBHOOK_SECRET=... node replay-email.mjs message.eml");
  process.exit(2);
}

const raw = await readFile(file);
if (raw.byteLength > 2 * 1024 * 1024) {
  throw new Error("Email exceeds PayGate's 2 MiB limit");
}
const messageIdMatch = raw.toString("utf8").match(/^Message-ID:\s*<?([^>\r\n]+)>?/im);
if (!messageIdMatch) {
  throw new Error("Email has no Message-ID and cannot be replayed safely");
}
const body = JSON.stringify({
  sourceId: messageIdMatch[1].trim(),
  envelopeFrom: "",
  envelopeTo: "",
  receivedAt: new Date().toISOString(),
  rawEmailBase64: raw.toString("base64"),
});
const timestamp = Math.floor(Date.now() / 1000).toString();
const signature = createHmac("sha256", secret).update(`${timestamp}.${body}`).digest("hex");
const response = await fetch(url, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-PayGate-Timestamp": timestamp,
    "X-PayGate-Signature": `sha256=${signature}`,
  },
  body,
});
const responseBody = await response.text();
if (!response.ok) {
  throw new Error(`Replay failed with HTTP ${response.status}: ${responseBody.slice(0, 1000)}`);
}
console.log(responseBody);
