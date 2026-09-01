import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, getActivity, getOverview } from "./api";
import type { ActivityEntry, Overview } from "./types";
import { Badge, Dot, ErrorNotice, SectionHead, Spinner, Stat, dateTime, money, relativeTime } from "./ui";

export function OverviewPage({ onOpenPayment }: { onOpenPayment: (id: string) => void }) {
  const [overview, setOverview] = useState<Overview>();
  const [activity, setActivity] = useState<ActivityEntry[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const load = useCallback(async () => {
    setError("");
    try {
      const [next, recent] = await Promise.all([getOverview(), getActivity(8)]);
      setOverview(next); setActivity(recent);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load PayGate overview."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void load(); const t = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 30_000); return () => clearInterval(t); }, [load]);

  const maxVolume = useMemo(() => Math.max(1, ...(overview?.volume.map((v) => v.amount_paise) ?? [1])), [overview]);
  if (loading && !overview) return <div className="page-loading"><Spinner/> Loading PayGate…</div>;
  return <>
    <SectionHead eyebrow="Live system" title="Overview" copy="Payment volume, collection routing, phone health and webhook delivery in one view." action={<button className="button button-secondary button-small" onClick={() => void load()}>Refresh</button>} />
    {error && <ErrorNotice message={error} />}
    {overview && <>
      <section className="stat-grid">
        <Stat label="Collected today" value={money(overview.collected_today_paise)} sub={`${overview.paid_today} paid`} />
        <Stat label="Payments today" value={overview.payments_today} sub={`${overview.pending} pending now`} />
        <Stat label="Expired today" value={overview.expired_today} sub="after grace" />
        <Stat label="Active collection" value={overview.active_profile?.label ?? "None"} sub={overview.active_profile?.id ?? "Configure a profile"} />
      </section>

      <section className="dashboard-grid">
        <article className="panel volume-panel">
          <div className="panel-head"><div><p className="eyebrow">Settled volume</p><h3>Last 7 days</h3></div><Badge tone="blue">INR</Badge></div>
          <div className="bar-chart" aria-label="Seven day payment volume">
            {overview.volume.map((point) => <div className="bar-item" key={point.date} title={`${point.date}: ${money(point.amount_paise)}`}>
              <div className="bar-track"><div className="bar-fill" style={{ height: `${Math.max(4, Math.round((point.amount_paise / maxVolume) * 100))}%` }} /></div>
              <strong>{point.payments}</strong><span>{new Date(`${point.date}T00:00:00`).toLocaleDateString("en-IN", { weekday: "short" })}</span>
            </div>)}
          </div>
        </article>

        <article className="panel health-panel">
          <div className="panel-head"><div><p className="eyebrow">System health</p><h3>Collection path</h3></div></div>
          <div className="health-row"><div><Dot ok={overview.relay.connected}/><div><strong>PayGate phone</strong><span>{overview.relay.name || "No active phone"}</span></div></div><Badge tone={overview.relay.connected ? "good" : "bad"}>{overview.relay.connected ? "Connected" : "Offline"}</Badge></div>
          <div className="health-meta"><span>Heartbeat</span><strong>{relativeTime(overview.relay.last_seen_at)}</strong></div>
          <div className="health-meta"><span>App</span><strong>{overview.relay.app_version || "—"}</strong></div>
          <div className="divider" />
          <div className="health-row"><div><Dot ok={overview.webhooks.exhausted === 0}/><div><strong>Merchant webhooks</strong><span>{overview.webhooks.pending} queued</span></div></div><Badge tone={overview.webhooks.exhausted ? "warn" : "good"}>{overview.webhooks.exhausted ? `${overview.webhooks.exhausted} exhausted` : "Healthy"}</Badge></div>
          <div className="health-meta"><span>Last delivered</span><strong>{relativeTime(overview.webhooks.last_delivered_at)}</strong></div>
        </article>
      </section>

      <section className="panel recent-panel">
        <div className="panel-head"><div><p className="eyebrow">Activity</p><h3>Latest events</h3></div><span className="muted">Auto-refreshes</span></div>
        <div className="activity-list compact">{activity.length ? activity.map((entry, i) => <button key={`${entry.at}-${i}`} className="activity-row" onClick={() => entry.payment_id && onOpenPayment(entry.payment_id)} disabled={!entry.payment_id}>
          <ActivityMark kind={entry.kind} status={entry.status} />
          <div className="activity-main"><strong>{entry.title}</strong><span>{entry.detail || entry.source || entry.status || "PayGate"}</span></div>
          {entry.amount_paise != null && <strong className="activity-amount">{money(entry.amount_paise)}</strong>}
          <time title={dateTime(entry.at)}>{relativeTime(entry.at)}</time>
        </button>) : <div className="empty-inline">No activity yet.</div>}</div>
      </section>
    </>}
  </>;
}

export function ActivityMark({ kind, status }: { kind: string; status?: string }) {
  const good = status === "paid" || status === "matched" || status === "corroborated" || status === "delivered";
  const bad = status === "exhausted" || status === "ambiguous";
  return <span className={`activity-mark ${good ? "good" : bad ? "bad" : kind === "webhook" ? "blue" : ""}`}>{kind === "webhook" ? "↗" : kind === "payment_detected" ? "₹" : "•"}</span>;
}
