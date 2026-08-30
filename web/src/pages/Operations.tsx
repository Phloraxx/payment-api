import { useCallback, useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Badge, formatDate, Modal } from "../components/common";
import { api } from "../api";
import type { OperatorAlertSummary, OperatorEvidenceDetail, OperatorReconciliationEntry, OperatorReconciliationRun, OperatorRefund, OperatorReviewDetail, OperatorReviewSummary } from "../types";

export function ReviewsPage({ notify }: { notify: (value: string) => void }) {
  const [records, setRecords] = useState<OperatorReviewSummary[]>([]);
  const [selected, setSelected] = useState<OperatorReviewDetail | null>(null);
  const [error, setError] = useState("");
  const [showResolved, setShowResolved] = useState(false);

  const load = useCallback(async () => {
    try {
      const status = showResolved ? "" : "&status=open";
      const result = await api<{ reviews: OperatorReviewSummary[] }>(`/api/operator/v2/reviews?limit=100${status}`);
      setRecords(result.reviews);
      if (selected) setSelected(await api<OperatorReviewDetail>(`/api/operator/v2/reviews/${encodeURIComponent(selected.id)}`));
      setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load review cases"); }
  }, [showResolved, selected?.id]);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 8_000);
    return () => window.clearInterval(timer);
  }, [load]);

  async function open(review: OperatorReviewSummary) {
    try { setSelected(await api<OperatorReviewDetail>(`/api/operator/v2/reviews/${encodeURIComponent(review.id)}`)); }
    catch (err) { notify(err instanceof Error ? err.message : "Could not load review details."); }
  }

  return <section className="card">
    <div className="section-title"><div><p className="eyebrow">FAIL-CLOSED EVIDENCE</p><h2>Attention required</h2><p className="muted">Unmatched or incomplete evidence is never assigned automatically.</p></div><label className="toggle"><input type="checkbox" checked={showResolved} onChange={(event) => setShowResolved(event.target.checked)} /> Show resolved</label></div>
    {error && <p className="error">{error}</p>}{!error && !records.length && <p className="empty">No review cases need attention.</p>}
    <div className="table-wrap"><table><thead><tr><th>Opened</th><th>Type</th><th>Severity</th><th>Payment</th><th>Status</th></tr></thead><tbody>{records.map((record) => <tr className="clickable" key={record.id} onClick={() => void open(record)}><td>{formatDate(record.openedAt)}</td><td>{record.kind}</td><td><Badge status={record.severity} /></td><td><strong>{record.paymentId || "—"}</strong><small>{record.reason}</small></td><td><Badge status={record.status} /></td></tr>)}</tbody></table></div>
    {selected && <ReviewModal review={selected} notify={notify} onClose={() => setSelected(null)} onResolved={async (message) => { notify(message); setSelected(null); await load(); }} />}
  </section>;
}

function ReviewModal({ review, notify, onClose, onResolved }: { review: OperatorReviewDetail; notify: (value: string) => void; onClose: () => void; onResolved: (message: string) => Promise<void> }) {
  const candidates = review.candidatePaymentIds ?? [];
  const [action, setAction] = useState(review.status === "open" ? "manual_match" : review.resolution || "dismissed");
  const [paymentId, setPaymentId] = useState(review.paymentId || candidates[0] || "");
  const [bankReference, setBankReference] = useState("");
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  async function resolve(event: FormEvent) {
    event.preventDefault(); setSaving(true);
    try {
      await api(`/api/operator/v2/reviews/${review.id}/resolve`, { method: "POST", body: JSON.stringify({ action, paymentId: paymentId || undefined, bankReference: bankReference || undefined, note }) });
      await onResolved(action === "manual_match" ? "Evidence matched and audited." : "Review case resolved and audited.");
    } catch (err) { notify(err instanceof Error ? err.message : "Review resolution failed."); }
    finally { setSaving(false); }
  }
  return <Modal title={`Review ${review.id}`} onClose={onClose}>
    <dl className="detail-list"><Detail label="Status"><Badge status={review.status} /></Detail><Detail label="Kind">{review.kind}</Detail><Detail label="Reason">{review.reason}</Detail><Detail label="Opened">{formatDate(review.openedAt)}</Detail><Detail label="Candidates">{candidates.length ? candidates.join(", ") : "—"}</Detail></dl>
    {review.evidence && <EvidenceCard record={review.evidence} />}
    {review.status === "open" ? <form className="stack-form" onSubmit={resolve}><label>Resolution<select value={action} onChange={(event) => setAction(event.target.value)}><option value="manual_match">Manually match exact evidence</option><option value="duplicate">Duplicate evidence</option><option value="not_payment">Not a checkout payment</option><option value="corrected">Corrected externally</option><option value="dismissed">Dismiss</option></select></label>{action === "manual_match" && <><label>Payment ID<input value={paymentId} onChange={(event) => setPaymentId(event.target.value)} required /></label><label>Bank RRN / UTR<input value={bankReference} onChange={(event) => setBankReference(event.target.value)} maxLength={64} /></label></>}<label>Resolution note<textarea value={note} onChange={(event) => setNote(event.target.value)} minLength={3} maxLength={4096} required /></label><button className="primary" disabled={saving}>{saving ? "Saving…" : "Resolve with audit trail"}</button></form> : <dl className="detail-list compact"><Detail label="Resolution">{review.resolution || "—"}</Detail><Detail label="Note">{review.resolutionNote || "—"}</Detail><Detail label="Resolved">{formatDate(review.resolvedAt)}</Detail></dl>}
  </Modal>;
}

function EvidenceCard({ record }: { record: OperatorEvidenceDetail }) {
  const fields: Array<[string, unknown]> = [["source", record.source], ["sender", record.sender], ["subject", record.subject], ["occurred", record.occurredAt], ["amount", record.amountPaise], ["rrn", record.reference], ["upi_id", record.upiId], ["payer_name", record.payerName], ["description", record.description], ["status", record.status], ["notes", record.notes]];
  return <div className="evidence-card"><p className="eyebrow">{record.kind.toUpperCase()} EVIDENCE</p>{fields.map(([field, raw]) => { if (raw === undefined || raw === null || raw === "" || raw === 0) return null; const value = field === "amount" && typeof raw === "number" ? `₹${(raw / 100).toFixed(2)}` : String(raw); return <div className="record-field" key={field}><span>{field}</span><code>{value}</code></div>; })}</div>;
}

export function ReconciliationPage({ notify }: { notify: (value: string) => void }) {
  const [runs, setRuns] = useState<OperatorReconciliationRun[]>([]);
  const [selected, setSelected] = useState<OperatorReconciliationRun | null>(null);
  const [entries, setEntries] = useState<OperatorReconciliationEntry[]>([]);
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const result = await api<{ runs: OperatorReconciliationRun[] }>("/api/operator/v2/reconciliation?limit=100");
      setRuns(result.runs); setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load reconciliation runs"); }
  }, []);
  useEffect(() => { void load(); const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 15_000); return () => window.clearInterval(timer); }, [load]);

  async function upload(event: FormEvent) {
    event.preventDefault();
    if (!file) return;
    setUploading(true);
    try {
      const form = new FormData(); form.append("statement", file);
      const result = await api<{ runId: string; conflictRows: number; reviewCases: number }>("/api/reconciliation/import", { method: "POST", body: form });
      notify(`Statement imported. ${result.conflictRows} conflicts and ${result.reviewCases} review cases.`);
      setFile(null); await load();
    } catch (err) { notify(err instanceof Error ? err.message : "Statement import failed."); }
    finally { setUploading(false); }
  }

  async function openRun(run: OperatorReconciliationRun) {
    setSelected(run);
    try {
      const result = await api<{ entries: OperatorReconciliationEntry[] }>(`/api/operator/v2/reconciliation/${encodeURIComponent(run.id)}/entries?limit=250`);
      setEntries(result.entries);
    } catch (err) { notify(err instanceof Error ? err.message : "Could not load reconciliation entries."); }
  }

  return <>
    <section className="card"><div className="section-title"><div><p className="eyebrow">BANK SAFETY NET</p><h2>Import statement</h2><p className="muted">CSV, TSV or XLSX. Imports report discrepancies and create reviews; they never mark payments paid automatically.</p></div></div>
      <form className="inline-form" onSubmit={upload}><label>Statement file<input className="file-input" type="file" accept=".csv,.tsv,.txt,.xlsx" onChange={(event) => setFile(event.target.files?.[0] ?? null)} required /></label><button className="primary" disabled={!file || uploading}>{uploading ? "Importing…" : "Import and reconcile"}</button></form>
    </section>
    <section className="card"><div className="section-title"><h2>Reconciliation runs</h2><button className="ghost" onClick={() => void load()}>Refresh</button></div>
      {error && <p className="error">{error}</p>}{!error && !runs.length && <p className="empty">No statements imported.</p>}
      <div className="table-wrap"><table><thead><tr><th>File</th><th>Status</th><th>Matched</th><th>Conflicts</th><th>Unmatched</th><th>Completed</th></tr></thead><tbody>
        {runs.map((run) => <tr className="clickable" key={run.id} onClick={() => void openRun(run)}><td><strong>{run.filename}</strong><small>{run.id}</small></td><td><Badge status={run.status} /></td><td>{run.matchedRows}</td><td>{run.conflictRows}</td><td>{run.unmatchedRows}</td><td>{formatDate(run.completedAt)}</td></tr>)}
      </tbody></table></div>
    </section>
    {selected && <Modal title={`Reconciliation ${selected.filename}`} onClose={() => { setSelected(null); setEntries([]); }}>
      <div className="record-list">{entries.map((entry) => <article key={entry.id}><div className="record-head"><strong>Row {String(entry.rowNumber)}</strong><Badge status={String(entry.status)} /></div><div className="record-field"><span>Amount</span><code>₹{(Number(entry.amountPaise) / 100).toFixed(2)}</code></div><div className="record-field"><span>RRN / UTR</span><code>{String(entry.reference || "—")}</code></div><div className="record-field"><span>Payment</span><code>{String(entry.paymentId || "—")}</code></div><div className="record-field"><span>Notes</span><code>{String(entry.notes || "—")}</code></div></article>)}</div>
    </Modal>}
  </>;
}

export function AlertsPage() {
  const [records, setRecords] = useState<OperatorAlertSummary[]>([]);
  const [error, setError] = useState("");
  const load = useCallback(async () => {
    try { const result = await api<{ alerts: OperatorAlertSummary[] }>("/api/operator/v2/alerts?limit=100"); setRecords(result.alerts); setError(""); }
    catch (err) { setError(err instanceof Error ? err.message : "Could not load alerts"); }
  }, []);
  useEffect(() => { void load(); const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 10_000); return () => window.clearInterval(timer); }, [load]);
  return <section className="card"><div className="section-title"><div><p className="eyebrow">OPERATIONAL MONITORING</p><h2>Alerts</h2></div><button className="ghost" onClick={() => void load()}>Refresh</button></div>
    {error && <p className="error">{error}</p>}{!error && !records.length && <p className="empty">No alerts recorded.</p>}
    <div className="record-list">{records.map((record) => <article key={record.id}><div className="record-head"><strong>{record.kind}</strong><span><Badge status={record.status} /> <Badge status={record.severity} /></span></div><p>{record.message}</p><div className="record-field"><span>Occurrences</span><code>{record.occurrenceCount}</code></div><div className="record-field"><span>First seen</span><code>{formatDate(record.firstSeenAt)}</code></div><div className="record-field"><span>Last seen</span><code>{formatDate(record.lastSeenAt)}</code></div><div className="record-field"><span>Notification</span><code>{record.notificationStatus || "disabled"}{record.notificationAttempts ? ` · ${record.notificationAttempts} attempt(s)` : ""}</code></div>{record.notificationLastError && <div className="record-field"><span>Delivery error</span><code>{record.notificationLastError}</code></div>}</article>)}</div>
  </section>;
}

export function RefundsPage({ notify }: { notify: (value: string) => void }) {
  const [records, setRecords] = useState<OperatorRefund[]>([]);
  const [selected, setSelected] = useState<OperatorRefund | null>(null);
  const [paymentId, setPaymentId] = useState("");
  const [amountPaise, setAmountPaise] = useState("");
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try { const result = await api<{ refunds: OperatorRefund[] }>("/api/operator/v2/refunds?limit=100"); setRecords(result.refunds); }
    catch (err) { notify(err instanceof Error ? err.message : "Could not load refunds"); }
  }, [notify]);
  useEffect(() => { void load(); const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 15_000); return () => window.clearInterval(timer); }, [load]);

  async function create(event: FormEvent) {
    event.preventDefault(); setSaving(true);
    try {
      await api("/api/refunds", { method: "POST", headers: { "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ paymentId, amountPaise: Number(amountPaise), reason }) });
      notify("Refund request recorded. PayGate has not moved money; complete it through the bank and record the reference.");
      setPaymentId(""); setAmountPaise(""); setReason(""); await load();
    } catch (err) { notify(err instanceof Error ? err.message : "Refund request failed."); }
    finally { setSaving(false); }
  }

  return <><section className="card"><div className="section-title"><div><p className="eyebrow">MANUAL MONEY MOVEMENT</p><h2>Record refund request</h2><p className="muted">This records and notifies. It does not transfer funds.</p></div></div>
    <form className="inline-form" onSubmit={create}><label>Paid payment ID<input value={paymentId} onChange={(event) => setPaymentId(event.target.value)} required /></label><label>Amount in paise<input type="number" min="1" step="1" value={amountPaise} onChange={(event) => setAmountPaise(event.target.value)} required /></label><label>Reason<input value={reason} onChange={(event) => setReason(event.target.value)} maxLength={4096} /></label><button className="primary" disabled={saving}>{saving ? "Recording…" : "Record request"}</button></form>
  </section><section className="card"><div className="section-title"><h2>Refund lifecycle</h2></div>{!records.length && <p className="empty">No refunds recorded.</p>}<div className="table-wrap"><table><thead><tr><th>Refund</th><th>Payment</th><th>Amount</th><th>Status</th><th>Reference</th></tr></thead><tbody>{records.map((record) => <tr className="clickable" key={record.id} onClick={() => setSelected(record)}><td><strong>{record.id}</strong><small>{formatDate(record.requestedAt)}</small></td><td>{record.paymentId}</td><td>₹{(record.amountPaise / 100).toFixed(2)}</td><td><Badge status={record.status} /></td><td>{record.reference || "—"}</td></tr>)}</tbody></table></div></section>
    {selected && <RefundModal refund={selected} notify={notify} onClose={() => setSelected(null)} onSaved={async (message) => { notify(message); setSelected(null); await load(); }} />}</>;
}

function RefundModal({ refund, notify, onClose, onSaved }: { refund: OperatorRefund; notify: (value: string) => void; onClose: () => void; onSaved: (message: string) => Promise<void> }) {
  const [status, setStatus] = useState(refund.status === "requested" ? "processing" : "completed");
  const [reference, setReference] = useState(refund.reference || "");
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const terminal = refund.status === "completed" || refund.status === "cancelled";
  async function save(event: FormEvent) {
    event.preventDefault(); setSaving(true);
    try { await api(`/api/refunds/${refund.id}/status`, { method: "POST", body: JSON.stringify({ status, reference, note }) }); await onSaved("Refund lifecycle updated and audited."); }
    catch (err) { notify(err instanceof Error ? err.message : "Refund update failed."); }
    finally { setSaving(false); }
  }
  return <Modal title={`Refund ${refund.id}`} onClose={onClose}><dl className="detail-list"><Detail label="Payment">{refund.paymentId}</Detail><Detail label="Amount">₹{(refund.amountPaise / 100).toFixed(2)}</Detail><Detail label="Current status"><Badge status={refund.status} /></Detail><Detail label="Reason">{refund.reason || "—"}</Detail><Detail label="Reference">{refund.reference || "—"}</Detail></dl>
    {!terminal && <form className="stack-form" onSubmit={save}><label>New status<select value={status} onChange={(event) => setStatus(event.target.value)}><option value="processing">Processing</option><option value="completed">Completed</option><option value="failed">Failed</option><option value="cancelled">Cancelled</option></select></label><label>Bank refund reference<input value={reference} onChange={(event) => setReference(event.target.value)} required={status === "completed"} /></label><label>Audit note<textarea value={note} onChange={(event) => setNote(event.target.value)} maxLength={4096} /></label><button className="primary" disabled={saving}>{saving ? "Saving…" : "Update lifecycle"}</button></form>}
  </Modal>;
}

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return <div><dt>{label}</dt><dd>{children}</dd></div>;
}
