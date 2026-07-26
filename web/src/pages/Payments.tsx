import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { Badge, formatDate, Modal } from "../components/common";
import { api, pb } from "../pb";
import type { Payment, PaymentCreateResponse } from "../types";

export function Payments({ notify }: { notify: (value: string) => void }) {
  const [amount, setAmount] = useState("100");
  const [externalId, setExternalId] = useState("");
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<PaymentCreateResponse | null>(null);
  const retryIdempotencyKey = useRef<string | null>(null);

  async function create(event: FormEvent) {
    event.preventDefault();
    if (!/^\d+$/.test(amount) || Number(amount) <= 0) {
      notify("Requested amount must be a positive whole number of rupees.");
      return;
    }
    setCreating(true);
    try {
      const idempotencyKey = retryIdempotencyKey.current ?? crypto.randomUUID();
      retryIdempotencyKey.current = idempotencyKey;
      const result = await api<PaymentCreateResponse>("/api/payments", {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ amount, externalId: externalId.trim() || undefined }),
      });
      setCreated(result);
      retryIdempotencyKey.current = null;
      notify("Payment created.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Payment creation failed.");
    } finally {
      setCreating(false);
    }
  }

  return <>
    <section className="card">
      <div className="section-title"><div><p className="eyebrow">CREATE PAYMENT</p><h2>Generate a DDM payable amount</h2></div></div>
      <form className="inline-form" onSubmit={create}>
        <label>Requested amount (whole INR)<input inputMode="numeric" pattern="[0-9]+" value={amount} onChange={(e) => { setAmount(e.target.value); retryIdempotencyKey.current = null; }} required /></label>
        <label>External/order ID (optional)<input maxLength={255} value={externalId} onChange={(e) => { setExternalId(e.target.value); retryIdempotencyKey.current = null; }} /></label>
        <button className="primary" disabled={creating}>{creating ? "Creating…" : "Create"}</button>
      </form>
      {created && <div className="created">
        <strong>₹{created.payableAmount}</strong>
        <span>Requested ₹{created.requestedAmount} · expires {formatDate(created.expiresAt)}</span>
        <code>{created.upiUri}</code>
      </div>}
    </section>
    <section className="card">
      <div className="section-title"><h2>Payments</h2><span className="muted">Select a row for evidence and actions</span></div>
      <PaymentTable limit={100} notify={notify} />
    </section>
  </>;
}

export function PaymentTable({ limit, notify }: { limit: number; notify?: (value: string) => void }) {
  const [records, setRecords] = useState<Payment[]>([]);
  const [selected, setSelected] = useState<Payment | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const result = await pb.collection("payments").getList<Payment>(1, limit, { sort: "-created" });
      setRecords(result.items);
      if (selected) {
        const refreshed = result.items.find((item) => item.id === selected.id);
        if (refreshed) setSelected(refreshed);
      }
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load payments");
    }
  }, [limit, selected?.id]);

  useEffect(() => {
    void load();
    let disposed = false;
    let unsubscribe: (() => void) | undefined;
    void pb.collection("payments").subscribe("*", () => void load()).then((fn) => {
      if (disposed) void fn(); else unsubscribe = fn;
    });
    return () => { disposed = true; unsubscribe?.(); };
  }, [load]);

  async function cancel(payment: Payment) {
    try {
      await api(`/api/payments/${payment.id}/cancel`, { method: "POST" });
      notify?.("Payment cancelled.");
      await load();
    } catch (err) {
      notify?.(err instanceof Error ? err.message : "Cancel failed.");
    }
  }

  if (error) return <p className="error">{error}</p>;
  if (!records.length) return <p className="empty">No payments yet.</p>;

  return <>
    <div className="table-wrap"><table><thead><tr><th>Payment</th><th>Requested</th><th>Payable</th><th>Status</th><th>Expires</th></tr></thead>
      <tbody>{records.map((record) => <tr className="clickable" key={record.id} onClick={() => setSelected(record)}>
        <td><strong>{record.id}</strong><small>{record.external_id || "—"}</small></td>
        <td>₹{record.requested_amount / 100}</td><td>₹{(record.payable_amount / 100).toFixed(2)}</td>
        <td><Badge status={record.status} /></td><td>{formatDate(record.expires_at)}</td>
      </tr>)}</tbody></table></div>
    {selected && <Modal title={`Payment ${selected.id}`} onClose={() => setSelected(null)}>
      <dl className="detail-list">
        <Detail label="Status"><Badge status={selected.status} /></Detail>
        <Detail label="Requested">₹{selected.requested_amount / 100}</Detail>
        <Detail label="Payable">₹{(selected.payable_amount / 100).toFixed(2)}</Detail>
        <Detail label="Created">{formatDate(selected.created)}</Detail>
        <Detail label="Expires">{formatDate(selected.expires_at)}</Detail>
        <Detail label="Reuse after">{formatDate(selected.reuse_after)}</Detail>
        <Detail label="RRN">{selected.rrn || "—"}</Detail>
        <Detail label="Payer UPI">{selected.upi_id || "—"}</Detail>
        <Detail label="Payer name">{selected.payer_name || "—"}</Detail>
        <Detail label="Paid at">{formatDate(selected.paid_at)}</Detail>
        <Detail label="External ID">{selected.external_id || "—"}</Detail>
      </dl>
      {selected.status === "pending" && <div className="actions"><button className="danger" onClick={() => void cancel(selected)}>Cancel payment</button></div>}
    </Modal>}
  </>;
}

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return <div><dt>{label}</dt><dd>{children}</dd></div>;
}
