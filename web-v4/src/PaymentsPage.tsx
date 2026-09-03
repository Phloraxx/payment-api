import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, editPayment, getPayment, getProfiles, listPayments, retryWebhook } from "./api";
import type { Payment, PaymentDetail, PaymentStatus, Profile } from "./types";
import { Badge, Empty, ErrorNotice, Modal, SectionHead, Spinner, dateTime, money } from "./ui";

const PAGE_SIZE = 50;
const statuses: Array<{ id: "" | PaymentStatus; label: string }> = [
  { id: "", label: "All statuses" }, { id: "pending", label: "Pending" }, { id: "paid", label: "Paid" },
  { id: "expired", label: "Expired" }, { id: "cancelled", label: "Cancelled" },
];

export function PaymentsPage({ initialPaymentId, onInitialConsumed }: { initialPaymentId?: string; onInitialConsumed: () => void }) {
  const [items, setItems] = useState<Payment[]>([]);
  const [total, setTotal] = useState(0);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [profile, setProfile] = useState("");
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<string>();
  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const [result, profileItems] = await Promise.all([
        listPayments({ q: query, status, profile, limit: PAGE_SIZE, offset }), getProfiles(),
      ]);
      setItems(result.items); setTotal(result.total); setProfiles(profileItems);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load payments."); }
    finally { setLoading(false); }
  }, [query, status, profile, offset]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => { if (initialPaymentId) { setSelected(initialPaymentId); onInitialConsumed(); } }, [initialPaymentId, onInitialConsumed]);
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const page = Math.floor(offset / PAGE_SIZE) + 1;

  return <>
    <SectionHead eyebrow="Transactions" title="Payments" copy="Search every payment and correct business or payer information without touching immutable financial snapshots." action={<button className="button button-secondary button-small" onClick={() => void load()}>Refresh</button>} />
    <section className="filter-bar">
      <form onSubmit={(e) => { e.preventDefault(); setOffset(0); setQuery(q.trim()); }} className="search-box"><span aria-hidden="true">⌕</span><input aria-label="Search payments" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search name, payment ID, event ID, payer…"/><button className="text-button" type="submit">Search</button></form>
      <select aria-label="Filter payments by status" value={status} onChange={(e) => { setStatus(e.target.value); setOffset(0); }}>{statuses.map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}</select>
      <select aria-label="Filter payments by collection profile" value={profile} onChange={(e) => { setProfile(e.target.value); setOffset(0); }}><option value="">All profiles</option>{profiles.map((p) => <option key={p.id} value={p.id}>{p.label}</option>)}</select>
    </section>
    {error && <ErrorNotice message={error} />}
    <section className="panel table-panel">
      <div className="table-head"><span>{total.toLocaleString("en-IN")} payments</span><span>Page {page} of {pages}</span></div>
      {loading && !items.length ? <div className="page-loading"><Spinner/> Loading payments…</div> : !items.length ? <Empty title="No payments found" copy="Change the search or filters and try again." /> : <div className="payment-table-wrap"><table className="payment-table"><thead><tr><th>Payment</th><th>Exact amount</th><th>Status</th><th>Collection</th><th>Created</th><th/></tr></thead><tbody>
        {items.map((payment) => <tr key={payment.id} onClick={() => setSelected(payment.id)}>
          <td><strong>{payment.name}</strong><span>{payment.external_id || payment.id}</span></td>
          <td><strong className="money-cell">{money(payment.payable_amount_paise)}</strong><span>requested {money(payment.requested_amount_paise)}</span></td>
          <td><PaymentBadge status={payment.status}/></td>
          <td><strong>{profiles.find((p) => p.id === payment.collection_profile_id)?.label ?? payment.collection_profile_id}</strong><span>{payment.upi_id_snapshot}</span></td>
          <td><strong>{dateTime(payment.created_at)}</strong><span>{payment.payer_name || "—"}</span></td>
          <td><button className="icon-button" aria-label={`Open ${payment.id}`}>›</button></td>
        </tr>)}
      </tbody></table></div>}
      <div className="pagination"><button className="button button-secondary button-small" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>Previous</button><span>{Math.min(offset + 1, total || 0)}–{Math.min(offset + PAGE_SIZE, total)} of {total}</span><button className="button button-secondary button-small" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset(offset + PAGE_SIZE)}>Next</button></div>
    </section>
    {selected && <PaymentDrawer id={selected} profiles={profiles} onClose={() => setSelected(undefined)} onChanged={() => void load()} />}
  </>;
}

function PaymentDrawer({ id, profiles, onClose, onChanged }: { id: string; profiles: Profile[]; onClose: () => void; onChanged: () => void }) {
  const [detail, setDetail] = useState<PaymentDetail>();
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    setError("");
    try { setDetail(await getPayment(id)); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load payment."); }
  }, [id]);
  useEffect(() => { void load(); }, [load]);
  const payment = detail?.payment;
  return <Modal title={payment ? payment.name : "Payment"} onClose={onClose} wide>
    {!detail && !error && <div className="page-loading"><Spinner/> Loading payment…</div>}
    {error && <ErrorNotice message={error} />}
    {payment && detail && <>
      <div className="drawer-hero"><div><p className="eyebrow">Exact amount</p><strong>{money(payment.payable_amount_paise)}</strong><span>{payment.id}</span></div><PaymentBadge status={payment.status}/></div>
      <div className="detail-grid">
        <Detail label="Person identifier" value={payment.name}/><Detail label="Event ID" value={payment.external_id || "—"}/>
        <Detail label="Requested" value={money(payment.requested_amount_paise)}/><Detail label="Adjustment" value={money(payment.adjustment_paise)}/>
        <Detail label="Collection profile" value={profiles.find((p) => p.id === payment.collection_profile_id)?.label ?? payment.collection_profile_id}/><Detail label="Destination UPI" value={payment.upi_id_snapshot}/>
        <Detail label="Created" value={dateTime(payment.created_at)}/><Detail label="Payment window ends" value={dateTime(payment.expires_at)}/>
        <Detail label="Grace until" value={dateTime(payment.grace_until)}/><Detail label="Reusable after" value={dateTime(payment.reuse_after)}/>
        <Detail label="Paid at" value={dateTime(payment.paid_at)}/><Detail label="Observed payer" value={payment.payer_name || "—"}/>
        <Detail label="Payer UPI ID" value={payment.payer_upi_id || "—"}/><Detail label="Payee snapshot" value={payment.payee_name_snapshot || "—"}/>
      </div>
      <div className="drawer-actions"><button className="button button-primary" onClick={() => setEditing(true)}>Edit payment</button><button className="button button-secondary" onClick={() => void load()}>Refresh</button></div>
      {payment.internal_note && <div className="note-card"><span>Internal note</span><p>{payment.internal_note}</p></div>}
      <details className="metadata-block"><summary>Metadata</summary><pre>{JSON.stringify(payment.metadata ?? {}, null, 2)}</pre></details>
      <section className="timeline-section"><h4>Timeline</h4>{detail.history.length ? detail.history.map((item) => <div className="timeline-row" key={item.id}><i/><div><strong>{item.summary}</strong><span>{item.actor} · {dateTime(item.created_at)}</span></div></div>) : <p className="muted">No history recorded.</p>}</section>
      <section className="timeline-section"><h4>Webhook deliveries</h4>{detail.webhooks.length ? detail.webhooks.map((hook) => <div className="webhook-row" key={hook.id}><div><strong>{hook.event_type}</strong><span>{hook.last_http_status ? `HTTP ${hook.last_http_status} · ` : ""}{hook.attempts} attempt{hook.attempts === 1 ? "" : "s"}</span>{hook.last_error && <small>{hook.last_error}</small>}</div><div><Badge tone={hook.status === "delivered" ? "good" : hook.status === "exhausted" ? "bad" : "warn"}>{hook.status}</Badge>{hook.status === "exhausted" && <button className="text-button" disabled={busy} onClick={async () => { setBusy(true); try { await retryWebhook(hook.id); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not retry webhook."); } finally { setBusy(false); } }}>Retry</button>}</div></div>) : <p className="muted">No webhook deliveries for this payment.</p>}</section>
      {editing && <EditPaymentModal payment={payment} onClose={() => setEditing(false)} onSaved={async () => { setEditing(false); await load(); onChanged(); }} />}
    </>}
  </Modal>;
}

function EditPaymentModal({ payment, onClose, onSaved }: { payment: Payment; onClose: () => void; onSaved: () => Promise<void> }) {
  const [name, setName] = useState(payment.name);
  const [externalId, setExternalId] = useState(payment.external_id ?? "");
  const [status, setStatus] = useState<PaymentStatus>(payment.status);
  const [payerName, setPayerName] = useState(payment.payer_name ?? "");
  const [payerUPI, setPayerUPI] = useState(payment.payer_upi_id ?? "");
  const [note, setNote] = useState(payment.internal_note ?? "");
  const [metadata, setMetadata] = useState(JSON.stringify(payment.metadata ?? {}, null, 2));
  const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  async function save() {
    setError(""); let parsed: Record<string, unknown> | null;
    try { parsed = metadata.trim() ? JSON.parse(metadata) as Record<string, unknown> : {}; } catch { setError("Metadata must be valid JSON."); return; }
    setBusy(true);
    try {
      await editPayment(payment.id, { name: name.trim(), external_id: externalId.trim(), status, payer_name: payerName.trim(), payer_upi_id: payerUPI.trim(), internal_note: note.trim(), metadata: parsed });
      await onSaved();
    } catch (e) { setError(e instanceof ApiError ? e.message : "Could not update payment."); }
    finally { setBusy(false); }
  }
  return <Modal title="Edit payment" onClose={onClose}><div className="form-stack">
    <label><span>Person identifier</span><input value={name} onChange={(e) => setName(e.target.value)}/></label>
    <label><span>Event ID</span><input value={externalId} onChange={(e) => setExternalId(e.target.value)}/></label>
    <label><span>Status</span><select value={status} onChange={(e) => setStatus(e.target.value as PaymentStatus)}><option value="pending">Pending</option><option value="paid">Paid</option><option value="expired">Expired</option><option value="cancelled">Cancelled</option></select></label>
    <div className="two-col"><label><span>Observed payer name</span><input value={payerName} onChange={(e) => setPayerName(e.target.value)}/></label><label><span>Payer UPI ID</span><input value={payerUPI} onChange={(e) => setPayerUPI(e.target.value)}/></label></div>
    <label><span>Internal note</span><textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)}/></label>
    <label><span>Metadata JSON</span><textarea rows={7} className="mono-input" value={metadata} onChange={(e) => setMetadata(e.target.value)}/></label>
    <div className="immutable-note"><strong>Financial snapshots are immutable.</strong><span>Requested amount, exact payable amount, collection profile, destination UPI and reservation times cannot be rewritten.</span></div>
    {error && <ErrorNotice message={error}/>}<div className="form-actions"><button className="button button-secondary" onClick={onClose}>Cancel</button><button className="button button-primary" onClick={() => void save()} disabled={busy || !name.trim()}>{busy ? <><Spinner/> Saving…</> : "Save changes"}</button></div>
  </div></Modal>;
}

function Detail({ label, value }: { label: string; value: string }) { return <div className="detail"><span>{label}</span><strong>{value}</strong></div>; }
export function PaymentBadge({ status }: { status: PaymentStatus }) {
  const tone = status === "paid" ? "good" : status === "pending" ? "warn" : status === "cancelled" ? "neutral" : "neutral";
  return <Badge tone={tone}>{status}</Badge>;
}
