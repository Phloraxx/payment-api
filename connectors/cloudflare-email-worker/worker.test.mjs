import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import test from "node:test";

import worker from "./worker.js";

const env = {
  PAYGATE_EMAIL_WEBHOOK_URL: "https://paygate.example/api/events/email",
  PAYGATE_EMAIL_WEBHOOK_SECRET: "email-webhook-secret-long-enough",
  PAYGATE_EMAIL_RECIPIENT: "payments@example.org",
  PAYGATE_EMAIL_FALLBACK_RECIPIENT: "payment-failures@example.net",
};

function message(raw, overrides = {}) {
  const rejected = [];
  const forwarded = [];
  return {
    from: "noreply@slice.bank.in",
    to: "payments@example.org",
    raw: new Response(raw).body,
    rawSize: new TextEncoder().encode(raw).byteLength,
    headers: new Headers({ "Message-ID": "<slice-event-1@example>" }),
    setReject: (reason) => rejected.push(reason),
    forward: async (recipient, headers) => forwarded.push({ recipient, headers }),
    rejected,
    forwarded,
    ...overrides,
  };
}

test("signs and forwards the exact JSON body", async (t) => {
  const originalFetch = globalThis.fetch;
  t.after(() => { globalThis.fetch = originalFetch; });
  let request;
  globalThis.fetch = async (url, init) => {
    request = { url, ...init };
    return new Response("ok", { status: 202 });
  };
  await worker.email(message("From: noreply@slice.bank.in\r\n\r\nReceived INR 100.01"), env);
  assert.equal(request.url, env.PAYGATE_EMAIL_WEBHOOK_URL);
  const timestamp = request.headers["X-PayGate-Timestamp"];
  const expected = createHmac("sha256", env.PAYGATE_EMAIL_WEBHOOK_SECRET)
    .update(`${timestamp}.${request.body}`)
    .digest("hex");
  assert.equal(request.headers["X-PayGate-Signature"], `sha256=${expected}`);
  const payload = JSON.parse(request.body);
  assert.equal(payload.sourceId, "slice-event-1@example");
  assert.equal(payload.envelopeFrom, "noreply@slice.bank.in");
  assert.equal(Buffer.from(payload.rawEmailBase64, "base64").toString(), "From: noreply@slice.bank.in\r\n\r\nReceived INR 100.01");
});

test("rejects an unexpected recipient without forwarding", async (t) => {
  const originalFetch = globalThis.fetch;
  t.after(() => { globalThis.fetch = originalFetch; });
  let called = false;
  globalThis.fetch = async () => { called = true; return new Response(); };
  const email = message("hello", { to: "other@example.org" });
  await worker.email(email, env);
  assert.equal(called, false);
  assert.equal(email.rejected.length, 1);
});

test("retries transient PayGate failures and forwards to the recovery inbox", async (t) => {
  const originalFetch = globalThis.fetch;
  const originalError = console.error;
  t.after(() => { globalThis.fetch = originalFetch; console.error = originalError; });
  console.error = () => {};
  let attempts = 0;
  globalThis.fetch = async () => { attempts += 1; return new Response("unavailable", { status: 503 }); };
  const email = message("hello");
  await worker.email(email, env);
  assert.equal(attempts, 3);
  assert.equal(email.forwarded.length, 1);
  assert.equal(email.forwarded[0].recipient, env.PAYGATE_EMAIL_FALLBACK_RECIPIENT);
  assert.equal(email.forwarded[0].headers.get("X-PayGate-Ingestion-Failed"), "true");
});

test("does not retry a permanent PayGate rejection", async (t) => {
  const originalFetch = globalThis.fetch;
  const originalError = console.error;
  t.after(() => { globalThis.fetch = originalFetch; console.error = originalError; });
  console.error = () => {};
  let attempts = 0;
  globalThis.fetch = async () => { attempts += 1; return new Response("unauthorized", { status: 401 }); };
  const email = message("hello");
  await worker.email(email, env);
  assert.equal(attempts, 1);
  assert.equal(email.forwarded.length, 1);
});
