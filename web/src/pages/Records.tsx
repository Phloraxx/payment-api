import { useCallback, useEffect, useState } from "react";
import { formatDate } from "../components/common";
import { api } from "../api";

type OperationalRecord = { id: string; createdAt?: string; fields: Record<string, unknown> };
const smsFields = ["payment_account", "source", "source_event_id", "message_time", "sender", "body", "amount", "rrn", "upi_id", "payer_name", "processing_status", "matched_payment", "error"];
const emailFields = ["payment_account", "source", "source_event_id", "message_time", "received_at", "sender", "recipient", "subject", "body", "amount", "rrn", "upi_id", "payer_name", "processing_status", "matched_payment", "error"];
const auditFields = ["action", "actor_email", "entity_type", "entity_id", "summary", "details", "occurred_at"];
const webhookFields = ["event_id", "event", "payment", "status", "attempts", "response_code", "next_attempt_at", "last_attempt_at", "delivered_at", "last_error"];

export function SMSEvents() { return <Records kind="sms" title="SMS evidence" fields={smsFields} />; }
export function EmailEvents() { return <Records kind="email" title="Email payment evidence" fields={emailFields} />; }
export function AuditEvents() { return <Records kind="audit" title="Operator audit trail" fields={auditFields} />; }
export function WebhookDeliveries() { return <Records kind="webhooks" title="Outgoing webhook deliveries" fields={webhookFields} />; }

function Records({ kind, title, fields }: { kind: string; title: string; fields: string[] }) {
  const [records, setRecords] = useState<OperationalRecord[]>([]);
  const [error, setError] = useState("");
  const load = useCallback(async () => {
    try {
      const result = await api<{ records: OperationalRecord[] }>(`/api/operator/v2/records/${encodeURIComponent(kind)}?limit=100`);
      setRecords(result.records); setError("");
    } catch (err) { setError(err instanceof Error ? err.message : `Could not load ${kind}`); }
  }, [kind]);
  useEffect(() => { void load(); const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 10_000); return () => window.clearInterval(timer); }, [load]);
  return <section className="card">
    <div className="section-title"><h2>{title}</h2><button className="ghost" onClick={() => void load()}>Refresh</button></div>
    {error && <p className="error">{error}</p>}{!error && !records.length && <p className="empty">No records yet.</p>}
    <div className="record-list">{records.map((record) => <article key={record.id}><div className="record-head"><strong>{record.id}</strong><span>{formatDate(record.createdAt)}</span></div>{fields.map((field) => renderField(record.fields, field))}</article>)}</div>
  </section>;
}
function renderField(record: Record<string, unknown>, field: string) {
  const value = record[field]; if (value === undefined || value === null || value === "") return null;
  const display = field === "amount" && typeof value === "number" ? `₹${(value / 100).toFixed(2)} (${value} paise)` : typeof value === "object" ? JSON.stringify(value) : String(value);
  return <div className="record-field" key={field}><span>{field}</span><code>{display}</code></div>;
}
