import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Badge, formatDate } from "../components/common";
import { api, pb } from "../pb";
import type { RazorpayTestConfig, RazorpayTestOrder, RazorpayTestOrderResponse } from "../types";

type CheckoutSuccess = {
  razorpay_payment_id: string;
  razorpay_order_id: string;
  razorpay_signature: string;
};

type RazorpayOptions = {
  key: string;
  amount: number;
  currency: string;
  name: string;
  description: string;
  order_id: string;
  handler: (response: CheckoutSuccess) => void;
  modal?: { ondismiss?: () => void };
  retry?: { enabled: boolean };
  theme?: { color: string };
};

declare global {
  interface Window {
    Razorpay?: new (options: RazorpayOptions) => { open: () => void };
  }
}

let checkoutLoader: Promise<void> | null = null;

export function RazorpayTestPage({ notify }: { notify: (value: string) => void }) {
  const [config, setConfig] = useState<RazorpayTestConfig | null>(null);
  const [orders, setOrders] = useState<RazorpayTestOrder[]>([]);
  const [amount, setAmount] = useState("1.00");
  const [externalId, setExternalId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [idempotencyKey, setIdempotencyKey] = useState(() => crypto.randomUUID());

  const load = useCallback(async () => {
    try {
      const nextConfig = await api<RazorpayTestConfig>("/api/razorpay/test/config");
      setConfig(nextConfig);
      if (nextConfig.enabled) {
        const result = await pb.collection("razorpay_test_orders").getList<RazorpayTestOrder>(1, 100, { sort: "-created_at" });
        setOrders(result.items);
      } else {
        setOrders([]);
      }
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load Razorpay test rail");
    }
  }, []);

  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    if (!config?.enabled) return;
    let disposed = false;
    let unsubscribe: (() => void) | undefined;
    void pb.collection("razorpay_test_orders").subscribe("*", () => void load()).then((fn) => {
      if (disposed) void fn(); else unsubscribe = fn;
    });
    return () => { disposed = true; unsubscribe?.(); };
  }, [config?.enabled, load]);

  async function create(event: FormEvent) {
    event.preventDefault();
    if (!config?.enabled) return;
    const amountPaise = parseRupees(amount);
    if (amountPaise === null) {
      notify("Enter an amount between ₹1.00 and ₹1,00,000.00 with at most two decimal places.");
      return;
    }
    setBusy(true);
    try {
      const order = await api<RazorpayTestOrderResponse>("/api/razorpay/test/orders", {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ amountPaise, externalId: externalId.trim() || undefined }),
      });
      await openCheckout(order, async (response) => {
        try {
          const verified = await api<RazorpayTestOrderResponse>(`/api/razorpay/test/orders/${order.id}/verify`, {
            method: "POST",
            body: JSON.stringify(response),
          });
          notify(verified.status === "captured" ? "Razorpay test payment captured." : `Callback verified; provider status is ${verified.status}.`);
          await load();
        } catch (err) {
          notify(err instanceof Error ? err.message : "Razorpay callback verification failed.");
        }
      }, () => notify("Razorpay test checkout closed without a completed payment."));
      setExternalId("");
      setIdempotencyKey(crypto.randomUUID());
      await load();
    } catch (err) {
      notify(err instanceof Error ? err.message : "Could not create Razorpay test order.");
    } finally {
      setBusy(false);
    }
  }

  async function refresh(order: RazorpayTestOrder) {
    setBusy(true);
    try {
      const updated = await api<RazorpayTestOrderResponse>(`/api/razorpay/test/orders/${order.id}/refresh`, { method: "POST" });
      notify(`Razorpay status refreshed: ${updated.status}.`);
      await load();
    } catch (err) {
      notify(err instanceof Error ? err.message : "Could not refresh Razorpay status.");
    } finally {
      setBusy(false);
    }
  }

  if (error) return <p className="error banner">{error}</p>;
  if (!config) return <section className="card"><p className="empty">Loading Razorpay test rail…</p></section>;

  return <>
    <section className="card">
      <div className="section-title">
        <div>
          <p className="eyebrow">ISOLATED TEST RAIL</p>
          <h2>Razorpay Test Mode</h2>
          <p className="muted">Mock transactions only. This module is separate from PayGate’s SMS/DDM payment records.</p>
        </div>
        <Badge status={config.enabled ? "test" : "disabled"} />
      </div>
      {!config.enabled ? <div className="created">
        <strong>Disabled</strong>
        <span>Add Test Mode credentials to a staging deployment and set <code>RAZORPAY_TEST_ENABLED=true</code>.</span>
        <span>No Razorpay secret is sent to this browser.</span>
      </div> : <form className="inline-form" onSubmit={create}>
        <label>Test amount (₹)<input inputMode="decimal" value={amount} onChange={(event) => { setAmount(event.target.value); setIdempotencyKey(crypto.randomUUID()); }} placeholder="1.00" required /></label>
        <label>External reference<input value={externalId} onChange={(event) => { setExternalId(event.target.value); setIdempotencyKey(crypto.randomUUID()); }} maxLength={255} placeholder="Optional test label" /></label>
        <button className="primary" disabled={busy}>{busy ? "Working…" : "Create order & open Checkout"}</button>
      </form>}
      {config.enabled && <p className="muted">Checkout key: <code>{config.keyId}</code>. Use Razorpay’s Test Mode success/failure controls; no real money is deducted.</p>}
    </section>

    <section className="card">
      <div className="section-title"><h2>Test orders</h2><button className="ghost" onClick={() => void load()}>Refresh list</button></div>
      {!orders.length ? <p className="empty">No Razorpay test orders yet.</p> : <div className="table-wrap"><table>
        <thead><tr><th>Created</th><th>Amount</th><th>Local / Razorpay IDs</th><th>Status</th><th>Method</th><th>Action</th></tr></thead>
        <tbody>{orders.map((order) => <tr key={order.id}>
          <td>{formatDate(order.created_at)}</td>
          <td>₹{(order.amount / 100).toFixed(2)}</td>
          <td><strong>{order.id}</strong><small>{order.razorpay_order_id || "Provider order pending"}</small><small>{order.razorpay_payment_id || "No payment yet"}</small></td>
          <td><Badge status={order.status} />{order.error && <small className="error">{order.error}</small>}</td>
          <td>{order.payment_method || "—"}</td>
          <td><button disabled={busy || !order.razorpay_payment_id} onClick={() => void refresh(order)}>Fetch status</button></td>
        </tr>)}</tbody>
      </table></div>}
    </section>
  </>;
}

async function openCheckout(order: RazorpayTestOrderResponse, handler: (response: CheckoutSuccess) => void, dismissed: () => void) {
  await loadCheckoutScript();
  if (!window.Razorpay) throw new Error("Razorpay Checkout failed to initialize.");
  if (!order.razorpayOrderId) throw new Error("The server did not return a Razorpay order id.");
  const checkout = new window.Razorpay({
    key: order.keyId,
    amount: order.amountPaise,
    currency: order.currency,
    name: order.displayName,
    description: "PayGate isolated test transaction",
    order_id: order.razorpayOrderId,
    handler,
    modal: { ondismiss: dismissed },
    retry: { enabled: true },
    theme: { color: "#d8f36a" },
  });
  checkout.open();
}

function loadCheckoutScript(): Promise<void> {
  if (window.Razorpay) return Promise.resolve();
  if (checkoutLoader) return checkoutLoader;
  checkoutLoader = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://checkout.razorpay.com/v1/checkout.js";
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("Could not load Razorpay Checkout."));
    document.head.appendChild(script);
  });
  return checkoutLoader;
}

function parseRupees(value: string): number | null {
  const normalized = value.trim();
  if (!/^\d+(?:\.\d{1,2})?$/.test(normalized)) return null;
  const amount = Number(normalized);
  if (!Number.isFinite(amount) || amount < 1 || amount > 100_000) return null;
  return Math.round(amount * 100);
}
