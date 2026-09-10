import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, editPayment, getDevices, getOverview, getPayment, getProfiles, listPayments, retryWebhook } from "./api";
import type { DeviceInfo, Overview, Payment, PaymentDetail, PaymentStatus, Profile } from "./types";
import { Badge, Empty, ErrorNotice, Modal, SectionHead, Spinner, dateTime, money, relativeTime } from "./ui";

const PAGE_SIZE = 25;
const statusFilters: Array<{ id: "" | PaymentStatus; label: string }> = [
  { id: "", label: "All" },
  { id: "paid", label: "Paid" },
  { id: "pending", label: "Pending" },
  { id: "expired", label: "Expired" },
  { id: "cancelled", label: "Cancelled" },
];

export function PaymentsPage({
  initialPaymentId,
  onInitialConsumed,
  onOpenSettings,
  onOpenActivity,
}: {
  initialPaymentId?: string;
  onInitialConsumed: () => void;
  onOpenSettings: () => void;
  onOpenActivity: () => void;
}) {
  const [items, setItems] = useState<Payment[]>([]);
  const [total, setTotal] = useState(0);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [overview, setOverview] = useState<Overview>();
  const [devices, setDevices] = useState<DeviceInfo[]>([]);
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | PaymentStatus>("");
  const [profile, setProfile] = useState("");
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [operationsError, setOperationsError] = useState("");
  const [selected, setSelected] = useState<string>();
  const paymentRequest = useRef(0);
  const operationsRequest = useRef(0);

  const loadPayments = useCallback(async (silent = false) => {
    const request = ++paymentRequest.current;
    if (silent) setRefreshing(true); else setLoading(true);
    setError("");
    try {
      const result = await listPayments({ q: query, status, profile, limit: PAGE_SIZE, offset });
      if (request !== paymentRequest.current) return;
      setItems(result.items);
      setTotal(result.total);
    } catch (e) {
      if (request !== paymentRequest.current) return;
      setError(e instanceof ApiError ? e.message : "Could not load payments.");
    } finally {
      if (request === paymentRequest.current) {
        setLoading(false);
        setRefreshing(false);
      }
    }
  }, [query, status, profile, offset]);

  const loadOperations = useCallback(async () => {
    const request = ++operationsRequest.current;
    setOperationsError("");
    const [profileResult, overviewResult, deviceResult] = await Promise.allSettled([getProfiles(), getOverview(), getDevices()]);
    if (request !== operationsRequest.current) return;
    let failures = 0;
    if (profileResult.status === "fulfilled") setProfiles(profileResult.value); else failures++;
    if (overviewResult.status === "fulfilled") setOverview(overviewResult.value); else failures++;
    if (deviceResult.status === "fulfilled") setDevices(deviceResult.value); else failures++;
    if (failures) setOperationsError("Some operational status data could not be refreshed.");
  }, []);

  const refreshAll = useCallback(async () => {
    await Promise.all([loadPayments(true), loadOperations()]);
  }, [loadPayments, loadOperations]);

  useEffect(() => { void loadPayments(); }, [loadPayments]);
  useEffect(() => { void loadOperations(); }, [loadOperations]);
  useEffect(() => {
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void loadPayments(true);
        void loadOperations();
      }
    }, 30_000);
    return () => window.clearInterval(timer);
  }, [loadPayments, loadOperations]);
  useEffect(() => {
    if (initialPaymentId) {
      setSelected(initialPaymentId);
      onInitialConsumed();
    }
  }, [initialPaymentId, onInitialConsumed]);

  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const page = Math.floor(offset / PAGE_SIZE) + 1;
  const unhealthyDevices = useMemo(() => devices.filter((device) => device.enabled && !device.operational), [devices]);
  const attentionCount = (overview?.unmatched_today ?? 0) + (overview?.expiring_soon ?? 0) + unhealthyDevices.length + (overview?.webhooks.exhausted ?? 0);

  return <div className="payments-overview-page">
    <SectionHead
      title="Payments"
      copy="Monitor and manage payment activity across your events."
      action={<button className="button button-secondary button-small" disabled={refreshing} onClick={() => void refreshAll()}>{refreshing ? "Refreshing…" : "Refresh"}</button>}
    />

    {error && <ErrorNotice message={error} />}
    {operationsError && <div className="notice notice-error">{operationsError}</div>}

    <section className="payments-summary" aria-label="Payment summary">
      <SummaryMetric value={money(overview?.collected_today_paise)} label="Collected today" />
      <SummaryMetric value={(overview?.paid_today ?? 0).toLocaleString("en-IN")} label="Confirmed today" />
      <SummaryMetric value={(overview?.pending ?? 0).toLocaleString("en-IN")} label="Pending" />
      <SummaryMetric value={(overview?.expired_today ?? 0).toLocaleString("en-IN")} label="Expired today" />
      <SummaryMetric value={(overview?.unmatched_today ?? 0).toLocaleString("en-IN")} label="Evidence to review" />
    </section>

    <div className="payments-operations-layout">
      <section className="payments-activity-section">
        <div className="payments-activity-head">
          <div><h3>Payment activity</h3><span>Today · {(overview?.payments_today ?? 0).toLocaleString("en-IN")} created</span></div>
          <form className="payments-search" onSubmit={(event) => { event.preventDefault(); setOffset(0); setQuery(q.trim()); }}>
            <span aria-hidden="true">⌕</span>
            <input aria-label="Search payments" value={q} onChange={(event) => setQ(event.target.value)} placeholder="Search registrant, event or payment ID" />
            {query && <button type="button" className="search-clear" onClick={() => { setQ(""); setQuery(""); setOffset(0); }}>×</button>}
          </form>
        </div>

        <div className="payments-toolbar">
          <div className="status-tabs" role="group" aria-label="Filter payments by status">
            {statusFilters.map((item) => <button key={item.id || "all"} className={status === item.id ? "active" : ""} onClick={() => { setStatus(item.id); setOffset(0); }}>{item.label}</button>)}
          </div>
          <select className="profile-filter" aria-label="Filter by collection profile" value={profile} onChange={(event) => { setProfile(event.target.value); setOffset(0); }}>
            <option value="">All destinations</option>
            {profiles.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
        </div>

        {loading && !items.length ? <div className="payments-loading"><Spinner /> Loading payments…</div> : !items.length ? <Empty title="No payments found" copy="Change the search or filters and try again." /> : <div className="overview-table-wrap">
          <table className="overview-payment-table">
            <thead><tr><th>Registrant</th><th>Event</th><th>Requested</th><th>Payable</th><th>Status</th><th>Evidence</th><th>Time</th><th aria-label="Actions" /></tr></thead>
            <tbody>{items.map((payment) => <tr key={payment.id} onClick={() => setSelected(payment.id)}>
              <td><strong>{payment.name || "Unnamed payment"}</strong><span className="payment-id-sub">{payment.id}</span></td>
              <td><span className="table-primary-soft">{payment.external_id || "—"}</span></td>
              <td><span className="requested-amount">{money(payment.requested_amount_paise)}</span></td>
              <td><strong className="payable-amount">{money(payment.payable_amount_paise)}</strong></td>
              <td><PaymentBadge status={payment.status} /></td>
              <td><EvidenceLabel payment={payment} /></td>
              <td><time title={dateTime(payment.paid_at || payment.created_at)}>{timeOnly(payment.paid_at || payment.created_at)}</time></td>
              <td><button type="button" className="row-arrow" aria-label={`Open payment ${payment.id}`} onClick={(event) => { event.stopPropagation(); setSelected(payment.id); }}>›</button></td>
            </tr>)}</tbody>
          </table>
        </div>}

        {total > PAGE_SIZE && <div className="overview-pagination">
          <span>{Math.min(offset + 1, total)}–{Math.min(offset + PAGE_SIZE, total)} of {total}</span>
          <div><button className="button button-secondary button-small" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>Previous</button><span>Page {page} of {pages}</span><button className="button button-secondary button-small" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset(offset + PAGE_SIZE)}>Next</button></div>
        </div>}
      </section>

      <aside className="payments-ops-rail">
        <section className="ops-rail-section attention-section">
          <div className="ops-rail-head"><div><h3>Needs attention</h3><span>{attentionCount ? `${attentionCount} signals` : "Nothing blocking operations"}</span></div></div>
          <AttentionRow count={overview?.unmatched_today ?? 0} label="Unmatched or ambiguous evidence" detail="Received today" action="Review activity" onClick={onOpenActivity} />
          <AttentionRow count={overview?.expiring_soon ?? 0} label="Payment windows ending soon" detail="Within the next 10 minutes" action="View pending" onClick={() => { setStatus("pending"); setOffset(0); }} />
          <AttentionRow count={unhealthyDevices.length} label="Relay devices needing attention" detail={unhealthyDevices[0]?.name || "All enabled relays are healthy"} action="Check relays" onClick={onOpenSettings} />
          {(overview?.webhooks.exhausted ?? 0) > 0 && <AttentionRow count={overview?.webhooks.exhausted ?? 0} label="Webhook deliveries exhausted" detail="Manual retry may be required" action="Open activity" onClick={onOpenActivity} />}
          {!attentionCount && <div className="all-clear"><span className="status-dot good" />All current checks are clear.</div>}
        </section>

        <section className="ops-rail-section detection-section">
          <div className="ops-rail-head"><div><h3>Payment detection</h3><span>Relay health and latest evidence</span></div><Badge tone={overview?.relay.connected ? "good" : "bad"}>{overview?.relay.connected ? "Healthy" : "Offline"}</Badge></div>
          <div className="detection-health"><span className={`status-dot ${overview?.relay.connected ? "good" : "bad"}`} /><div><strong>{overview?.relay.connected_devices ?? 0} of {overview?.relay.enabled_devices ?? 0} relay devices online</strong><span>{overview?.relay.name ? `Most recent: ${overview.relay.name}` : "No enabled relay reported"}</span></div></div>
          <div className="detection-latest"><span>Last payment notification</span><strong>{relativeTime(overview?.last_observation?.received_at)}</strong><small>{overview?.last_observation ? `${paymentAppLabel(overview.last_observation.package_name)}${overview.last_observation.device_name ? ` via ${overview.last_observation.device_name}` : ""}` : "No payment evidence received yet"}</small></div>
          <div className="detection-trust"><span>Evidence policy</span><strong>Trusted payment apps only</strong></div>
          <button className="rail-link" onClick={onOpenSettings}>View relay health <span>→</span></button>
        </section>
      </aside>
    </div>

    {selected && <PaymentDrawer id={selected} profiles={profiles} onClose={() => setSelected(undefined)} onChanged={() => void refreshAll()} />}
  </div>;
}

function SummaryMetric({ value, label }: { value: string; label: string }) {
  return <div className="summary-metric"><strong>{value}</strong><span>{label}</span></div>;
}

function AttentionRow({ count, label, detail, action, onClick }: { count: number; label: string; detail: string; action: string; onClick: () => void }) {
  if (count <= 0) return null;
  return <button className="attention-row" onClick={onClick}>
    <strong>{count}</strong><span><b>{label}</b><small>{detail}</small></span><em>{action} →</em>
  </button>;
}

function EvidenceLabel({ payment }: { payment: Payment }) {
  if (!payment.evidence_package) return <span className="evidence-empty">—</span>;
  return <span className="evidence-label"><i />{paymentAppLabel(payment.evidence_package)}</span>;
}

function paymentAppLabel(packageName?: string): string {
  switch (packageName) {
    case "in.org.npci.upiapp": return "BHIM";
    case "com.google.android.apps.nbu.paisa.merchant":
    case "com.google.android.apps.nbu.paisa.user": return "Google Pay";
    case "com.paytm.business": return "Paytm Business";
    case "net.one97.paytm": return "Paytm";
    case "com.phonepe.app": return "PhonePe";
    case "in.amazon.mShop.android.shopping": return "Amazon Pay";
    case "money.super.payments": return "super.money";
    case "com.kotak811mobilebankingapp.instantsavingsupiscanandpayrecharge": return "Kotak";
    default: return packageName ? "Payment app" : "—";
  }
}

function timeOnly(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "—";
  return new Intl.DateTimeFormat("en-IN", { hour: "numeric", minute: "2-digit" }).format(date);
}

function PaymentDrawer({ id, profiles, onClose, onChanged }: { id: string; profiles: Profile[]; onClose: () => void; onChanged: () => void }) {
  const [detail, setDetail] = useState<PaymentDetail>();
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    setError("");
    try { setDetail(await getPayment(id)); }
    catch (e) { setError(e instanceof ApiError ? e.message : "Could not load payment."); }
  }, [id]);
  useEffect(() => { void load(); }, [load]);
  const payment = detail?.payment;

  return <Modal title="Payment details" onClose={onClose} drawer>
    {!detail && !error && <div className="page-loading"><Spinner /> Loading payment…</div>}
    {error && <ErrorNotice message={error} />}
    {payment && detail && <>
      <div className="drawer-status-line"><PaymentBadge status={payment.status} /><span>{payment.paid_at ? dateTime(payment.paid_at) : dateTime(payment.created_at)}</span></div>
      <div className="drawer-payment-id"><span>Payment ID</span><strong>{payment.id}</strong></div>

      <DetailSection title="Registration">
        <DetailRow label="Event" value={payment.external_id || "—"} />
        <DetailRow label="Registrant" value={payment.name || "—"} />
      </DetailSection>

      <DetailSection title="Payment">
        <DetailRow label="Requested" value={money(payment.requested_amount_paise)} />
        <DetailRow label="Payable" value={money(payment.payable_amount_paise)} strong />
        <DetailRow label="Unique adjustment" value={`+ ${money(payment.adjustment_paise)}`} />
        <DetailRow label="Destination UPI" value={payment.upi_id_snapshot} />
      </DetailSection>

      <DetailSection title="Observed payment">
        <DetailRow label="Payer" value={payment.payer_name || "Not observed"} />
        <DetailRow label="Payer UPI" value={payment.payer_upi_id || "—"} />
        <DetailRow label="Evidence" value={payment.evidence_package ? `${paymentAppLabel(payment.evidence_package)} notification` : "—"} />
        <DetailRow label="Relay device" value={payment.evidence_device_name || "—"} />
        <DetailRow label="Observed at" value={dateTime(payment.evidence_received_at)} />
      </DetailSection>

      <DetailSection title="Timing">
        <DetailRow label="Created" value={dateTime(payment.created_at)} />
        <DetailRow label="Reservation ends" value={dateTime(payment.expires_at)} />
        <DetailRow label="Grace until" value={dateTime(payment.grace_until)} />
      </DetailSection>

      {payment.internal_note && <div className="note-card"><span>Internal note</span><p>{payment.internal_note}</p></div>}

      <section className="drawer-timeline"><h4>Timeline</h4>{detail.history.length ? detail.history.map((item) => <div className="drawer-timeline-row" key={item.id}><i /><div><strong>{item.summary}</strong><span>{item.actor} · {dateTime(item.created_at)}</span></div></div>) : <p className="muted">No history recorded.</p>}</section>

      <details className="drawer-technical"><summary>Webhook deliveries & metadata</summary>
        <div className="drawer-technical-body">
          <section><h4>Webhook deliveries</h4>{detail.webhooks.length ? detail.webhooks.map((hook) => <div className="webhook-row" key={hook.id}><div><strong>{hook.event_type}</strong><span>{hook.last_http_status ? `HTTP ${hook.last_http_status} · ` : ""}{hook.attempts} attempt{hook.attempts === 1 ? "" : "s"}</span>{hook.last_error && <small>{hook.last_error}</small>}</div><div><Badge tone={hook.status === "delivered" ? "good" : hook.status === "exhausted" ? "bad" : "warn"}>{hook.status}</Badge>{hook.status === "exhausted" && <button className="text-button" disabled={busy} onClick={async () => { setBusy(true); try { await retryWebhook(hook.id); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not retry webhook."); } finally { setBusy(false); } }}>Retry</button>}</div></div>) : <p className="muted">No webhook deliveries.</p>}</section>
          <pre>{JSON.stringify(payment.metadata ?? {}, null, 2)}</pre>
        </div>
      </details>

      <div className="drawer-actions"><button className="button button-secondary" onClick={() => setEditing(true)}>Edit payment</button><button className="text-button" onClick={() => void load()}>Refresh details</button></div>
      {editing && <EditPaymentModal payment={payment} onClose={() => setEditing(false)} onSaved={async () => { setEditing(false); await load(); onChanged(); }} />}
    </>}
  </Modal>;
}

function DetailSection({ title, children }: { title: string; children: React.ReactNode }) {
  return <section className="drawer-detail-section"><h4>{title}</h4><div>{children}</div></section>;
}
function DetailRow({ label, value, strong = false }: { label: string; value: string; strong?: boolean }) {
  return <div className="drawer-detail-row"><span>{label}</span><strong className={strong ? "emphasis" : ""}>{value}</strong></div>;
}

function EditPaymentModal({ payment, onClose, onSaved }: { payment: Payment; onClose: () => void; onSaved: () => Promise<void> }) {
  const [name, setName] = useState(payment.name);
  const [externalId, setExternalId] = useState(payment.external_id ?? "");
  const [status, setStatus] = useState<PaymentStatus>(payment.status);
  const [payerName, setPayerName] = useState(payment.payer_name ?? "");
  const [payerUPI, setPayerUPI] = useState(payment.payer_upi_id ?? "");
  const [note, setNote] = useState(payment.internal_note ?? "");
  const [metadata, setMetadata] = useState(JSON.stringify(payment.metadata ?? {}, null, 2));
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function save() {
    setError("");
    let parsed: Record<string, unknown> | null;
    try { parsed = metadata.trim() ? JSON.parse(metadata) as Record<string, unknown> : {}; }
    catch { setError("Metadata must be valid JSON."); return; }
    setBusy(true);
    try {
      await editPayment(payment.id, { name: name.trim(), external_id: externalId.trim(), status, payer_name: payerName.trim(), payer_upi_id: payerUPI.trim(), internal_note: note.trim(), metadata: parsed });
      await onSaved();
    } catch (e) { setError(e instanceof ApiError ? e.message : "Could not update payment."); }
    finally { setBusy(false); }
  }
  return <Modal title="Edit payment" onClose={onClose}><div className="form-stack">
    <label><span>Person identifier</span><input value={name} onChange={(e) => setName(e.target.value)} /></label>
    <label><span>Event ID</span><input value={externalId} onChange={(e) => setExternalId(e.target.value)} /></label>
    <label><span>Status</span><select value={status} onChange={(e) => setStatus(e.target.value as PaymentStatus)}><option value="pending">Pending</option><option value="paid">Paid</option><option value="expired">Expired</option><option value="cancelled">Cancelled</option></select></label>
    <div className="two-col"><label><span>Observed payer name</span><input value={payerName} onChange={(e) => setPayerName(e.target.value)} /></label><label><span>Payer UPI ID</span><input value={payerUPI} onChange={(e) => setPayerUPI(e.target.value)} /></label></div>
    <label><span>Internal note</span><textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)} /></label>
    <label><span>Metadata JSON</span><textarea rows={7} className="mono-input" value={metadata} onChange={(e) => setMetadata(e.target.value)} /></label>
    <div className="immutable-note"><strong>Financial snapshots are immutable.</strong><span>Requested amount, exact payable amount, collection profile, destination UPI and reservation times cannot be rewritten.</span></div>
    {error && <ErrorNotice message={error} />}<div className="form-actions"><button className="button button-secondary" onClick={onClose}>Cancel</button><button className="button button-primary" onClick={() => void save()} disabled={busy || !name.trim()}>{busy ? <><Spinner /> Saving…</> : "Save changes"}</button></div>
  </div></Modal>;
}

export function PaymentBadge({ status }: { status: PaymentStatus }) {
  const tone = status === "paid" ? "good" : status === "pending" ? "warn" : status === "expired" ? "bad" : "neutral";
  return <Badge tone={tone}>{status}</Badge>;
}
