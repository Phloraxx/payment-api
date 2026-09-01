import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, getActivity } from "./api";
import type { ActivityEntry } from "./types";
import { ActivityMark } from "./OverviewPage";
import { Badge, Empty, ErrorNotice, SectionHead, Spinner, dateTime, money } from "./ui";

export function ActivityPage({ onOpenPayment }: { onOpenPayment: (id: string) => void }) {
  const [items, setItems] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [kind, setKind] = useState("");
  const [query, setQuery] = useState("");
  const load = useCallback(async () => {
    setError("");
    try { setItems(await getActivity(250)); }
    catch (e) { setError(e instanceof ApiError ? e.message : "Could not load activity."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void load(); }, [load]);
  const filtered = useMemo(() => items.filter((item) => {
    if (kind && item.kind !== kind) return false;
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return [item.title, item.detail, item.source, item.status, item.payment_id].some((value) => value?.toLowerCase().includes(needle));
  }), [items, kind, query]);
  return <>
    <SectionHead eyebrow="Audit stream" title="Activity" copy="Payment lifecycle changes, incoming phone observations and merchant webhook deliveries in chronological order." action={<button className="button button-secondary button-small" onClick={() => void load()}>Refresh</button>} />
    <section className="filter-bar"><div className="search-box"><span>⌕</span><input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search activity…"/></div><select value={kind} onChange={(e) => setKind(e.target.value)}><option value="">All activity</option><option value="payment">Payment changes</option><option value="payment_detected">Incoming payments</option><option value="webhook">Webhooks</option></select></section>
    {error && <ErrorNotice message={error}/>}
    <section className="panel activity-panel">{loading && !items.length ? <div className="page-loading"><Spinner/> Loading activity…</div> : !filtered.length ? <Empty title="No matching activity" copy="Try a different filter or search."/> : <div className="activity-list">{filtered.map((entry, i) => <button key={`${entry.at}-${i}-${entry.kind}`} className="activity-row activity-row-large" onClick={() => entry.payment_id && onOpenPayment(entry.payment_id)} disabled={!entry.payment_id}>
      <ActivityMark kind={entry.kind} status={entry.status}/>
      <div className="activity-main"><div><strong>{entry.title}</strong>{entry.status && <Badge tone={entry.status === "matched" || entry.status === "corroborated" || entry.status === "delivered" || entry.status === "paid" ? "good" : entry.status === "ambiguous" || entry.status === "exhausted" ? "bad" : "neutral"}>{entry.status}</Badge>}</div><span>{entry.detail || entry.source || "PayGate"}{entry.payment_id ? ` · ${entry.payment_id}` : ""}</span></div>
      {entry.amount_paise != null && <strong className="activity-amount">{money(entry.amount_paise)}</strong>}
      <time>{dateTime(entry.at)}</time>
    </button>)}</div>}</section>
  </>;
}
