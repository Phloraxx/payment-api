import { useCallback, useEffect, useState } from "react";
import type { RecordModel } from "pocketbase";
import { formatDate } from "../components/common";
import { pb } from "../pb";

const smsFields = ["source", "source_event_id", "message_time", "sender", "body", "amount", "rrn", "upi_id", "payer_name", "processing_status", "matched_payment", "error"];
const auditFields = ["action", "actor_email", "entity_type", "entity_id", "summary", "details", "occurred_at"];
const webhookFields = ["event_id", "event", "payment", "status", "attempts", "response_code", "next_attempt_at", "last_attempt_at", "delivered_at", "last_error"];

export function SMSEvents() {
  return <Records collection="sms_events" title="SMS evidence" fields={smsFields} />;
}

export function AuditEvents() {
  return <Records collection="audit_events" title="Operator audit trail" fields={auditFields} />;
}

export function WebhookDeliveries() {
  return <Records collection="webhook_deliveries" title="Outgoing webhook deliveries" fields={webhookFields} />;
}

function Records({ collection, title, fields }: { collection: string; title: string; fields: string[] }) {
  const [records, setRecords] = useState<RecordModel[]>([]);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const result = await pb.collection(collection).getList(1, 100, { sort: "-created" });
      setRecords(result.items);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : `Could not load ${collection}`);
    }
  }, [collection]);

  useEffect(() => {
    void load();
    let disposed = false;
    let unsubscribe: (() => void) | undefined;
    void pb.collection(collection).subscribe("*", () => void load()).then((fn) => {
      if (disposed) void fn(); else unsubscribe = fn;
    });
    return () => { disposed = true; unsubscribe?.(); };
  }, [collection, load]);

  return <section className="card">
    <div className="section-title"><h2>{title}</h2><button className="ghost" onClick={() => void load()}>Refresh</button></div>
    {error && <p className="error">{error}</p>}
    {!error && !records.length && <p className="empty">No records yet.</p>}
    <div className="record-list">{records.map((record) => <article key={record.id}>
      <div className="record-head"><strong>{record.id}</strong><span>{formatDate(String(record.created || ""))}</span></div>
      {fields.map((field) => renderField(record, field))}
    </article>)}</div>
  </section>;
}

function renderField(record: RecordModel, field: string) {
  const value = record[field];
  if (value === undefined || value === null || value === "") return null;
  const display = field === "amount" && typeof value === "number"
    ? `₹${(value / 100).toFixed(2)} (${value} paise)`
    : typeof value === "object" ? JSON.stringify(value) : String(value);
  return <div className="record-field" key={field}><span>{field}</span><code>{display}</code></div>;
}
