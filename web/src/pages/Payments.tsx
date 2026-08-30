import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { Badge, formatDate, Modal } from "../components/common";
import { api } from "../api";
import type { OperatorPaymentDetail, OperatorPaymentDetailsUpdate, OperatorPaymentPage, OperatorPaymentSummary, PaymentCreateResponse } from "../types";

const PAGE_SIZE = 25;
const statuses = ["", "pending", "paid", "late", "expired", "cancelled"];
const accounts = ["", "kotak", "slice", "paytm"];

export function Payments({ notify }: { notify: (value: string) => void }) {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [account, setAccount] = useState("");
  const [sort, setSort] = useState("newest");
  const [offset, setOffset] = useState(0);
  const [page, setPage] = useState<OperatorPaymentPage>({ payments: [], total: 0, limit: PAGE_SIZE, offset: 0 });
  const [selected, setSelected] = useState<OperatorPaymentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset), sort });
    if (query.trim()) params.set("q", query.trim());
    if (status) params.set("status", status);
    if (account) params.set("account", account);
    try {
      setPage(await api<OperatorPaymentPage>(`/api/operator/v2/payments?${params}`));
      setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load payments"); }
    finally { setLoading(false); }
  }, [query, status, account, sort, offset]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), query.trim() ? 220 : 0);
    return () => window.clearTimeout(timer);
  }, [load]);
  useEffect(() => {
    const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 10_000);
    return () => window.clearInterval(timer);
  }, [load]);
  useEffect(() => {
    const raw = window.location.hash.split("?")[1] || "";
    const id = new URLSearchParams(raw).get("open");
    if (id) void openById(id);
  }, []);

  async function openById(id: string) {
    try { setSelected(await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(id)}`)); }
    catch (err) { notify(err instanceof Error ? err.message : "Could not load payment"); }
  }

  const from = page.total === 0 ? 0 : page.offset + 1;
  const to = Math.min(page.offset + page.payments.length, page.total);

  return <div className="payments-layout">
    <section className="payments-toolbar panel">
      <div className="payment-search">
        <span aria-hidden="true">⌕</span>
        <input value={query} onChange={(event) => { setQuery(event.target.value); setOffset(0); }} placeholder="Search payment, customer, payer, order, UPI, reference or note" aria-label="Search payments" />
        {query && <button onClick={() => { setQuery(""); setOffset(0); }} aria-label="Clear search">×</button>}
      </div>
      <button className="primary-action" onClick={() => setShowCreate(true)}><span>＋</span> New payment</button>
      <div className="filter-bar">
        <FilterSelect label="Status" value={status} onChange={(value) => { setStatus(value); setOffset(0); }} options={statuses} />
        <FilterSelect label="Account" value={account} onChange={(value) => { setAccount(value); setOffset(0); }} options={accounts} />
        <label className="filter-select"><span>Sort</span><select value={sort} onChange={(event) => { setSort(event.target.value); setOffset(0); }}><option value="newest">Newest</option><option value="oldest">Oldest</option><option value="amount_desc">Amount ↓</option><option value="amount_asc">Amount ↑</option><option value="status">Status</option></select></label>
        {(status || account || query || sort !== "newest") && <button className="clear-filters" onClick={() => { setStatus(""); setAccount(""); setQuery(""); setSort("newest"); setOffset(0); }}>Clear filters</button>}
      </div>
    </section>

    <section className="panel payments-table-panel">
      <div className="panel-heading payments-heading"><div><span>All payments</span><h3>{page.total.toLocaleString()} records</h3></div><button className="quiet-button" onClick={() => void load()} disabled={loading}>{loading ? "Refreshing…" : "Refresh"}</button></div>
      {error && <div className="soft-error">{error}<button onClick={() => void load()}>Retry</button></div>}
      {!error && !loading && page.payments.length === 0 && <div className="empty-state"><strong>No payments found</strong><span>Try a different search or clear a filter.</span></div>}
      {page.payments.length > 0 && <div className="payment-table-wrap"><table className="payment-table"><thead><tr><th>Payment</th><th>Customer</th><th>Amount</th><th>Account</th><th>Status</th><th>Created</th><th /></tr></thead><tbody>
        {page.payments.map((payment) => <PaymentRow key={payment.id} payment={payment} onOpen={() => void openById(payment.id)} />)}
      </tbody></table></div>}
      <div className="table-footer"><span>{from}–{to} of {page.total}</span><div><button disabled={loading || offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>← Previous</button><button disabled={loading || offset + PAGE_SIZE >= page.total} onClick={() => setOffset(offset + PAGE_SIZE)}>Next →</button></div></div>
    </section>

    {showCreate && <CreatePayment notify={notify} onClose={() => setShowCreate(false)} onCreated={async (payment) => { setShowCreate(false); await load(); await openById(payment.id); }} />}
    {selected && <PaymentDrawer payment={selected} notify={notify} onClose={() => setSelected(null)} onChanged={async (updated) => { setSelected(updated); await load(); }} />}
  </div>;
}

function PaymentRow({ payment, onOpen }: { payment: OperatorPaymentSummary; onOpen: () => void }) {
  const title = payment.displayName || payment.externalId || payment.customerName || payment.id;
  return <tr onClick={onOpen} tabIndex={0} onKeyDown={(event) => { if (event.key === "Enter") onOpen(); }}>
    <td><div className="table-identity"><span className="payment-avatar small">{title.slice(0, 1).toUpperCase()}</span><span><strong>{title}</strong><small>{payment.id}</small></span></div></td>
    <td><strong className="table-secondary-strong">{payment.customerName || "—"}</strong><small>{payment.externalId || "No external ID"}</small></td>
    <td><strong className="money-cell">{money(payment.payableAmountPaise)}</strong><small>requested {money(payment.requestedAmountPaise)}</small></td>
    <td><span className="account-pill">{payment.paymentAccount}</span></td>
    <td><Badge status={payment.status} /></td>
    <td><span className="date-cell">{shortDate(payment.createdAt)}</span></td>
    <td><button className="row-arrow" onClick={(event) => { event.stopPropagation(); onOpen(); }} aria-label="Open payment">→</button></td>
  </tr>;
}

function FilterSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: string[] }) {
  return <label className="filter-select"><span>{label}</span><select value={value} onChange={(event) => onChange(event.target.value)}>{options.map((item) => <option key={item || "all"} value={item}>{item ? item.charAt(0).toUpperCase() + item.slice(1) : `All ${label.toLowerCase()}s`}</option>)}</select></label>;
}

function CreatePayment({ notify, onClose, onCreated }: { notify: (value: string) => void; onClose: () => void; onCreated: (payment: PaymentCreateResponse) => Promise<void> }) {
  const [amount, setAmount] = useState("");
  const [externalId, setExternalId] = useState("");
  const [creating, setCreating] = useState(false);
  const retryKey = useRef<string | null>(null);

  async function create(event: FormEvent) {
    event.preventDefault();
    if (!/^\d+$/.test(amount) || Number(amount) <= 0) { notify("Enter a positive whole-rupee amount."); return; }
    setCreating(true);
    try {
      const key = retryKey.current ?? crypto.randomUUID();
      retryKey.current = key;
      const result = await api<PaymentCreateResponse>("/api/payments", { method: "POST", headers: { "Idempotency-Key": key }, body: JSON.stringify({ amount, externalId: externalId.trim() || undefined }) });
      retryKey.current = null;
      notify("Payment created.");
      await onCreated(result);
    } catch (err) { notify(err instanceof Error ? err.message : "Could not create payment"); }
    finally { setCreating(false); }
  }

  return <Modal title="New payment" onClose={onClose}>
    <form className="create-payment-form" onSubmit={create}>
      <div><p className="form-kicker">Requested amount</p><label className="large-money-input"><span>₹</span><input autoFocus inputMode="numeric" pattern="[0-9]*" placeholder="500" value={amount} onChange={(event) => { setAmount(event.target.value.replace(/\D/g, "")); retryKey.current = null; }} /></label><p>PayGate will add a safe paise marker automatically.</p></div>
      <label className="field"><span>Order / external ID <small>optional</small></span><input maxLength={255} value={externalId} onChange={(event) => { setExternalId(event.target.value); retryKey.current = null; }} placeholder="IEEE-REG-1024" /></label>
      <div className="modal-actions"><button type="button" className="secondary-action" onClick={onClose} disabled={creating}>Cancel</button><button className="primary-action" disabled={creating || !amount}>{creating ? "Creating…" : "Create payment"}</button></div>
    </form>
  </Modal>;
}

function PaymentDrawer({ payment, notify, onClose, onChanged }: { payment: OperatorPaymentDetail; notify: (value: string) => void; onClose: () => void; onChanged: (payment: OperatorPaymentDetail) => Promise<void> }) {
  const [tab, setTab] = useState<"overview" | "edit" | "financial">("overview");
  const [busy, setBusy] = useState(false);

  async function cancel() {
    if (!window.confirm("Cancel this pending payment? The original record and any evidence already received will remain protected.")) return;
    setBusy(true);
    try {
      const updated = await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(payment.id)}/cancel`, { method: "POST", body: "{}" });
      notify("Payment cancelled."); await onChanged(updated);
    } catch (err) { notify(err instanceof Error ? err.message : "Could not cancel payment"); }
    finally { setBusy(false); }
  }

  const title = payment.displayName || payment.externalId || payment.customerName || "Payment";
  return <Modal title={title} onClose={onClose} variant="drawer">
    <div className="drawer-summary"><div><span>Exact amount</span><strong>{money(payment.payableAmountPaise)}</strong></div><Badge status={payment.status} /></div>
    <div className="drawer-tabs"><button className={tab === "overview" ? "active" : ""} onClick={() => setTab("overview")}>Overview</button><button className={tab === "edit" ? "active" : ""} onClick={() => setTab("edit")}>Edit</button><button className={tab === "financial" ? "active" : ""} onClick={() => setTab("financial")}>Financial</button></div>
    {tab === "overview" && <PaymentOverview payment={payment} />}
    {tab === "edit" && <PaymentEditor payment={payment} notify={notify} onSaved={async (updated) => { await onChanged(updated); setTab("overview"); }} />}
    {tab === "financial" && <FinancialTruth payment={payment} />}
    {payment.status === "pending" && <div className="drawer-danger"><button disabled={busy} onClick={() => void cancel()}>{busy ? "Cancelling…" : "Cancel payment"}</button><p>Matching and creation-identity fields cannot be edited. Cancel and create a replacement instead.</p></div>}
  </Modal>;
}

function PaymentOverview({ payment }: { payment: OperatorPaymentDetail }) {
  return <div className="drawer-section-stack">
    <DetailSection title="Customer & purpose"><Detail label="Display name">{payment.displayName || "—"}</Detail><Detail label="Order / external ID" copyValue={payment.externalId}>{payment.externalId || "—"}</Detail><Detail label="Customer">{payment.customerName || "—"}</Detail><Detail label="Email" copyValue={payment.customerEmail}>{payment.customerEmail || "—"}</Detail><Detail label="Phone" copyValue={payment.customerPhone}>{payment.customerPhone || "—"}</Detail>{payment.description && <p className="detail-paragraph">{payment.description}</p>}</DetailSection>
    {payment.tags?.length > 0 && <DetailSection title="Tags"><div className="tag-row">{payment.tags.map((tag) => <span key={tag}>{tag}</span>)}</div></DetailSection>}
    {payment.adminNote && <div className="admin-note"><span>Private admin note</span><p>{payment.adminNote}</p></div>}
    <DetailSection title="Timeline"><Detail label="Created">{formatDate(payment.createdAt)}</Detail><Detail label="Expires">{formatDate(payment.expiresAt)}</Detail><Detail label="Paid">{formatDate(payment.paidAt)}</Detail><Detail label="Resolved">{formatDate(payment.resolvedAt)}</Detail></DetailSection>
    {Object.keys(payment.customFields || {}).length > 0 && <DetailSection title="Custom fields"><JsonPairs value={payment.customFields} /></DetailSection>}
  </div>;
}

function PaymentEditor({ payment, notify, onSaved }: { payment: OperatorPaymentDetail; notify: (value: string) => void; onSaved: (payment: OperatorPaymentDetail) => Promise<void> }) {
  const [form, setForm] = useState<OperatorPaymentDetailsUpdate>({ displayName: payment.displayName || "", customerName: payment.customerName || "", customerEmail: payment.customerEmail || "", customerPhone: payment.customerPhone || "", description: payment.description || "", adminNote: payment.adminNote || "", tags: payment.tags || [], customFields: payment.customFields || {} });
  const [tags, setTags] = useState((payment.tags || []).join(", "));
  const [customFields, setCustomFields] = useState(JSON.stringify(payment.customFields || {}, null, 2));
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const field = (key: keyof OperatorPaymentDetailsUpdate, value: string) => setForm((current) => ({ ...current, [key]: value }));

  async function save(event: FormEvent) {
    event.preventDefault(); setError("");
    let parsedCustom: Record<string, unknown>;
    try { parsedCustom = parseObject(customFields, "Custom fields"); }
    catch (err) { setError(err instanceof Error ? err.message : "Custom data is invalid"); return; }
    const body: OperatorPaymentDetailsUpdate = { ...form, tags: tags.split(",").map((tag) => tag.trim()).filter(Boolean), customFields: parsedCustom };
    setSaving(true);
    try {
      const updated = await api<OperatorPaymentDetail>(`/api/operator/v2/payments/${encodeURIComponent(payment.id)}/details`, { method: "PUT", body: JSON.stringify(body) });
      notify("Payment details saved and audited."); await onSaved(updated);
    } catch (err) { setError(err instanceof Error ? err.message : "Could not save payment details"); }
    finally { setSaving(false); }
  }

  return <form className="payment-editor" onSubmit={save}>
    <div className="editor-grid"><EditField label="Display name" value={form.displayName} onChange={(v) => field("displayName", v)} /><EditField label="Customer name" value={form.customerName} onChange={(v) => field("customerName", v)} /><EditField label="Customer email" value={form.customerEmail} onChange={(v) => field("customerEmail", v)} type="email" /><EditField label="Customer phone" value={form.customerPhone} onChange={(v) => field("customerPhone", v)} /><EditField label="Tags" value={tags} onChange={setTags} placeholder="event, vip, s7" /></div>
    <label className="field"><span>Description</span><textarea rows={3} value={form.description} onChange={(event) => field("description", event.target.value)} /></label>
    <label className="field"><span>Private admin note</span><textarea rows={4} value={form.adminNote} onChange={(event) => field("adminNote", event.target.value)} /></label>
    <button type="button" className="advanced-toggle" onClick={() => setShowAdvanced((value) => !value)}>{showAdvanced ? "Hide custom data" : "Show custom data"} <span>{showAdvanced ? "−" : "+"}</span></button>
    {showAdvanced && <div className="advanced-fields"><label className="field"><span>Custom fields <small>JSON object</small></span><textarea className="code-area" rows={7} value={customFields} onChange={(event) => setCustomFields(event.target.value)} /></label></div>}
    {error && <div className="form-error">{error}</div>}
    <div className="editor-footer"><p>Amounts, account, external ID, original metadata, idempotency, status and evidence are protected.</p><button className="primary-action" disabled={saving}>{saving ? "Saving…" : "Save changes"}</button></div>
  </form>;
}

function FinancialTruth({ payment }: { payment: OperatorPaymentDetail }) {
  return <div className="drawer-section-stack"><div className="protected-banner"><span>Protected financial record</span><p>These fields drive matching, uniqueness and audit guarantees. They cannot be silently edited.</p></div><DetailSection title="Payment"><Detail label="Requested amount" copyValue={money(payment.requestedAmountPaise)}>{money(payment.requestedAmountPaise)}</Detail><Detail label="Payable amount" copyValue={money(payment.payableAmountPaise)}>{money(payment.payableAmountPaise)}</Detail><Detail label="Account">{payment.paymentAccount}</Detail><Detail label="Status"><Badge status={payment.status} /></Detail></DetailSection><DetailSection title="Evidence"><Detail label="RRN / UTR" copyValue={payment.rrn}>{payment.rrn || "—"}</Detail><Detail label="Evidence reference" copyValue={payment.evidenceReference}>{payment.evidenceReference || "—"}</Detail><Detail label="Evidence source">{payment.evidenceSource || "—"}</Detail><Detail label="Detected payer">{payment.payerName || "—"}</Detail><Detail label="Detected UPI ID" copyValue={payment.upiId}>{payment.upiId || "—"}</Detail></DetailSection><DetailSection title="Creation identity"><Detail label="External ID" copyValue={payment.externalId}>{payment.externalId || "—"}</Detail><Detail label="Original metadata"><code>{JSON.stringify(payment.metadata ?? {})}</code></Detail><Detail label="Idempotency key" copyValue={payment.idempotencyKey}><code>{payment.idempotencyKey || "—"}</code></Detail></DetailSection><DetailSection title="Integrity"><Detail label="Reuse after">{formatDate(payment.reuseAfter)}</Detail><Detail label="Payment ID" copyValue={payment.id}><code>{payment.id}</code></Detail></DetailSection></div>;
}

function DetailSection({ title, children }: { title: string; children: ReactNode }) { return <section className="detail-section"><h4>{title}</h4><dl>{children}</dl></section>; }
function Detail({ label, children, copyValue }: { label: string; children: ReactNode; copyValue?: string }) { return <div className="detail-row"><dt>{label}</dt><dd><span>{children}</span>{copyValue && <button type="button" className="copy-button" onClick={() => void navigator.clipboard.writeText(copyValue)}>Copy</button>}</dd></div>; }
function EditField({ label, value, onChange, type = "text", placeholder = "" }: { label: string; value: string; onChange: (value: string) => void; type?: string; placeholder?: string }) { return <label className="field"><span>{label}</span><input type={type} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} /></label>; }
function JsonPairs({ value }: { value: Record<string, unknown> }) { return <div className="json-pairs">{Object.entries(value).map(([key, item]) => <div key={key}><span>{key}</span><code>{typeof item === "string" ? item : JSON.stringify(item)}</code></div>)}</div>; }
function parseObject(raw: string, label: string): Record<string, unknown> { const parsed = JSON.parse(raw || "{}"); if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error(`${label} must be a JSON object.`); return parsed as Record<string, unknown>; }
function money(value: number) { return `₹${(value / 100).toFixed(2)}`; }
function shortDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "2-digit" }); }
