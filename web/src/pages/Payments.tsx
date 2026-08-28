import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { Badge, formatDate, Modal } from "../components/common";
import { api } from "../api";
import type { OperatorPaymentDetail, OperatorPaymentSummary, PaymentCreateResponse } from "../types";

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
    } finally { setCreating(false); }
  }

  return <>
    <section className="card">
      <div className="section-title"><div><p className="eyebrow">CREATE PAYMENT</p><h2>Generate a DDM payable amount</h2></div></div>
      <form className="inline-form" onSubmit={create}>
        <label>Requested amount (whole INR)<input inputMode="numeric" pattern="[0-9]+" value={amount} onChange={(e) => { setAmount(e.target.value); retryIdempotencyKey.current = null; }} required /></label>
        <label>External/order ID (optional)<input maxLength={255} value={externalId} onChange={(e) => { setExternalId(e.target.value); retryIdempotencyKey.current = null; }} /></label>
        <button className="primary" disabled={creating}>{creating ? "Creating…" : "Create"}</button>
      </form>
      {created && <div className="created"><strong>₹{created.payableAmount}</strong><span>Requested ₹{created.requestedAmount} · expires {formatDate(created.expiresAt)}</span><code>{created.upiUri}</code></div>}
    </section>
    <section className="card">
      <div className="section-title"><h2>Payments</h2><span className="muted">Typed operator API · select a row for evidence</span></div>
      <PaymentTable limit={100} notify={notify} />
    </section>
  </>;
}

export function PaymentTable({ limit, notify }: { limit: number; notify?: (value: string) => void }) {
  const [records, setRecords] = useState<OperatorPaymentSummary[]>([]);
  const [selected, setSelected] = useState<OperatorPaymentDetail | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const result = await api<{ payments: OperatorPaymentSummary[] }>(`/api/operator/v2/payments?limit=${limit}`);
      setRecords(result.payments);
      if (selected) setSelected(await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(selected.id)}`));
      setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load payments"); }
  }, [limit, selected?.id]);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 5_000);
    return () => window.clearInterval(timer);
  }, [load]);

  async function open(payment: OperatorPaymentSummary) {
    try { setSelected(await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(payment.id)}`)); }
    catch (err) { notify?.(err instanceof Error ? err.message : "Could not load payment details."); }
  }
  async function cancel(payment: OperatorPaymentDetail) {
    if (!window.confirm("Cancel this pending payment? Evidence that occurred before cancellation remains protected by PayGate's matching rules.")) return;
    try {
      setSelected(await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(payment.id)}/cancel`, { method: "POST", body: "{}" }));
      notify?.("Payment cancelled.");
      await load();
    } catch (err) { notify?.(err instanceof Error ? err.message : "Cancel failed."); }
  }

  if (error) return <p className="error">{error}</p>;
  if (!records.length) return <p className="empty">No payments yet.</p>;
  return <>
    <div className="table-wrap"><table><thead><tr><th>Payment</th><th>Requested</th><th>Payable</th><th>Status</th><th>Expires</th></tr></thead>
      <tbody>{records.map((record) => <tr className="clickable" key={record.id} onClick={() => void open(record)}>
        <td><strong>{record.id}</strong><small>{record.paymentAccount}</small></td>
        <td>{formatPaise(record.requestedAmountPaise)}</td><td>{formatPaise(record.payableAmountPaise)}</td>
        <td><Badge status={record.status} /></td><td>{formatDate(record.expiresAt)}</td>
      </tr>)}</tbody></table></div>
    {selected && <Modal title={`Payment ${selected.id}`} onClose={() => setSelected(null)}>
      <dl className="detail-list">
        <Detail label="Status"><Badge status={selected.status} /></Detail>
        <Detail label="Account">{selected.paymentAccount}</Detail>
        <Detail label="Requested">{formatPaise(selected.requestedAmountPaise)}</Detail>
        <Detail label="Payable">{formatPaise(selected.payableAmountPaise)}</Detail>
        <Detail label="Created">{formatDate(selected.createdAt)}</Detail>
        <Detail label="Expires">{formatDate(selected.expiresAt)}</Detail>
        <Detail label="RRN">{selected.rrn || "—"}</Detail>
        <Detail label="Payer UPI">{selected.upiId || "—"}</Detail>
        <Detail label="Payer name">{selected.payerName || "—"}</Detail>
        <Detail label="Evidence">{selected.evidenceSource || "—"}</Detail>
        <Detail label="Paid at">{formatDate(selected.paidAt)}</Detail>
        <Detail label="External ID">{selected.externalId || "—"}</Detail>
      </dl>
      {selected.status === "pending" && <div className="actions"><button className="danger" onClick={() => void cancel(selected)}>Cancel payment</button></div>}
    </Modal>}
  </>;
}

function Detail({ label, children }: { label: string; children: ReactNode }) { return <div><dt>{label}</dt><dd>{children}</dd></div>; }
function formatPaise(value: number) { return `₹${(value / 100).toFixed(2)}`; }
