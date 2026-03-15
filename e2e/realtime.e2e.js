/* eslint-disable no-console */
const assert = require("node:assert/strict");
const WebSocket = require("ws");

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

function withTimeout(promise, ms, label) {
  let t;
  const timeout = new Promise((_, reject) => {
    t = setTimeout(() => reject(new Error(`Timeout after ${ms}ms: ${label}`)), ms);
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(t));
}

function baseToWs(baseUrl) {
  const u = new URL(baseUrl);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString().replace(/\/$/, "");
}

async function postJson(url, body) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    json = { raw: text };
  }
  return { res, json, text };
}

async function createTicket(baseUrl, amount) {
  const { res, json, text } = await postJson(`${baseUrl}/api/ticket`, { amount });
  assert.equal(res.status, 200, `createTicket failed: ${text}`);
  assert.ok(json.ticketId, "ticketId missing");
  assert.equal(json.status, "pending", "ticket not pending");
  return json;
}

function connectProxyWs(wsBaseUrl, ticketId) {
  const wsUrl = `${wsBaseUrl}/api/ws?ticketId=${encodeURIComponent(ticketId)}`;
  const ws = new WebSocket(wsUrl);

  const opened = new Promise((resolve, reject) => {
    ws.once("open", resolve);
    ws.once("error", reject);
  });
  return { wsUrl, ws, opened };
}

async function simulatePaymentViaWebhook(baseUrl, ticketId, amount) {
  const secret = process.env.WEBHOOK_SECRET || "REDACTED";
  const numeric = ticketId.replace(/^TICKET/i, "").trim();
  const body = `TICKET${numeric} SOURAV paid you ₹${amount}`;

  const { res, json, text } = await postJson(`${baseUrl}/api/webhook`, {
    secret_key: secret,
    body,
  });

  assert.equal(res.status, 200, `webhook failed: ${text}`);
  assert.ok(json, "webhook response missing");
  return json;
}

function waitForPaymentUpdate(ws, ticketId) {
  return new Promise((resolve, reject) => {
    const onMsg = (msg) => {
      const raw = msg.toString();
      let payload;
      try {
        payload = JSON.parse(raw);
      } catch {
        return;
      }

      if (payload?.type !== "payment_update") return;
      if (payload?.status !== "paid") return;
      ws.off("message", onMsg);
      ws.off("error", onErr);
      resolve(payload);
    };

    const onErr = (err) => {
      ws.off("message", onMsg);
      ws.off("error", onErr);
      reject(err);
    };

    ws.on("message", onMsg);
    ws.on("error", onErr);
  });
}

async function expectNoMessageFor(ms, ws) {
  return new Promise((resolve, reject) => {
    const onMsg = (msg) => reject(new Error(`Unexpected WS message: ${msg.toString()}`));
    const onErr = (err) => reject(err);
    ws.on("message", onMsg);
    ws.on("error", onErr);
    setTimeout(() => {
      ws.off("message", onMsg);
      ws.off("error", onErr);
      resolve();
    }, ms);
  });
}

async function run() {
  const baseUrl = (process.env.E2E_BASE_URL || "http://localhost:8787").replace(/\/$/, "");
  const wsBaseUrl = baseToWs(baseUrl);

  console.log(`[e2e] baseUrl=${baseUrl}`);
  console.log(`[e2e] wsBaseUrl=${wsBaseUrl}`);

  // --- Test 1: single ticket gets paid update ---
  console.log("[e2e] Test 1: single ticket realtime update");
  const t1 = await createTicket(baseUrl, 100);
  const c1 = connectProxyWs(wsBaseUrl, t1.ticketId);
  console.log(`[e2e] ws connect: ${c1.wsUrl}`);
  await withTimeout(c1.opened, 5000, "ws open (t1)");

  await sleep(500); // small buffer so the proxy has time to subscribe upstream
  const waitT1 = withTimeout(waitForPaymentUpdate(c1.ws, t1.ticketId), 20000, "payment_update (t1)");
  await simulatePaymentViaWebhook(baseUrl, t1.ticketId, t1.amount);
  const update1 = await waitT1;

  assert.equal(update1.status, "paid", "t1 status not paid");
  assert.ok(update1.paidAt, "t1 paidAt missing");

  c1.ws.close();
  console.log("[e2e] Test 1 passed");

  // --- Test 2: no cross-talk between two tickets ---
  console.log("[e2e] Test 2: cross-talk isolation");
  const ta = await createTicket(baseUrl, 101);
  const tb = await createTicket(baseUrl, 101);

  const ca = connectProxyWs(wsBaseUrl, ta.ticketId);
  const cb = connectProxyWs(wsBaseUrl, tb.ticketId);

  await withTimeout(ca.opened, 5000, "ws open (ta)");
  await withTimeout(cb.opened, 5000, "ws open (tb)");

  await sleep(500);
  const waitTa = withTimeout(waitForPaymentUpdate(ca.ws, ta.ticketId), 20000, "payment_update (ta)");
  await simulatePaymentViaWebhook(baseUrl, ta.ticketId, ta.amount);
  const updateA = await waitTa;
  assert.equal(updateA.status, "paid", "ta status not paid");

  // tb must NOT receive anything during this window
  await withTimeout(expectNoMessageFor(4000, cb.ws), 6000, "no cross-talk (tb)");

  ca.ws.close();
  cb.ws.close();
  console.log("[e2e] Test 2 passed");

  console.log("[e2e] All realtime tests passed");
}

run().catch((err) => {
  console.error("[e2e] FAILED:", err);
  process.exit(1);
});

