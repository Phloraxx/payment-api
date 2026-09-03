import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, getActivity, getOverview, listPayments } from "./api";
import type { ActivityEntry, DailyVolume, Overview, Payment } from "./types";
import { Badge, Dot, ErrorNotice, SectionHead, Spinner, dateTime, money, relativeTime } from "./ui";

const statusColors: Record<string, string> = {
  paid: "#28e68f",
  pending: "#f6c85f",
  expired: "#637287",
  cancelled: "#ff6f7d",
};

export function OverviewPage({ onOpenPayment }: { onOpenPayment: (id: string) => void }) {
  const [overview, setOverview] = useState<Overview>();
  const [activity, setActivity] = useState<ActivityEntry[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const load = useCallback(async () => {
    setError("");
    try {
      const [next, recent, paymentPage] = await Promise.all([getOverview(), getActivity(6), listPayments({ limit: 5 })]);
      setOverview(next); setActivity(recent); setPayments(paymentPage.items);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load PayGate overview."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void load(); const t = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 30_000); return () => clearInterval(t); }, [load]);

  const volumeTotal = useMemo(() => overview?.volume.reduce((sum, item) => sum + item.amount_paise, 0) ?? 0, [overview]);
  const paymentTotal = useMemo(() => Object.values(overview?.status_counts ?? {}).reduce((sum, count) => sum + count, 0), [overview]);
  const donut = useMemo(() => donutGradient(overview?.status_counts ?? {}), [overview]);
  if (loading && !overview) return <div className="page-loading"><Spinner/> Loading PayGate…</div>;

  return <>
    <SectionHead eyebrow="Live operations" title="Overview" copy="Payment volume, settlement state and collection health — without exposing routing to callers." action={<button className="button button-secondary button-small" onClick={() => void load()}>Refresh</button>} />
    {error && <ErrorNotice message={error} />}
    {overview && <>
      <section className="stat-grid">
        <MetricCard label="Collected today" value={money(overview.collected_today_paise)} detail={`${overview.paid_today} settled`} accent="green" />
        <MetricCard label="Payments today" value={overview.payments_today.toLocaleString("en-IN")} detail={`${overview.pending} pending`} accent="teal" />
        <MetricCard label="7-day volume" value={money(volumeTotal)} detail={`${overview.volume.reduce((sum, item) => sum + item.payments, 0)} payments`} accent="aqua" />
        <MetricCard label="Collection profile" value={overview.active_profile?.label ?? "Not configured"} detail={overview.active_profile?.id ?? "Needs setup"} accent="green" />
      </section>

      <section className="dashboard-grid dashboard-grid-primary">
        <article className="panel volume-panel">
          <div className="panel-head"><div><p className="eyebrow">Payment volume</p><h3>Last 7 days</h3><p>Settled amount by day</p></div><Badge tone="blue">INR</Badge></div>
          <AreaChart points={overview.volume} />
        </article>

        <article className="panel status-panel">
          <div className="panel-head"><div><p className="eyebrow">Payment state</p><h3>Status breakdown</h3><p>All migrated + live payments</p></div></div>
          <div className="donut-wrap">
            <div className="donut" style={{ background: donut }}><div><strong>{paymentTotal.toLocaleString("en-IN")}</strong><span>Total</span></div></div>
            <div className="status-legend">{["paid", "pending", "expired", "cancelled"].map((status) => {
              const count = overview.status_counts[status] ?? 0;
              const pct = paymentTotal ? Math.round((count / paymentTotal) * 1000) / 10 : 0;
              return <div key={status}><i style={{ background: statusColors[status] }}/><span>{status}</span><strong>{count}</strong><small>{pct}%</small></div>;
            })}</div>
          </div>
        </article>

        <article className="panel health-panel">
          <div className="panel-head"><div><p className="eyebrow">System health</p><h3>Collection path</h3><p>What must stay healthy</p></div></div>
          <HealthRow label="PayGate phone" detail={overview.relay.name || "No active phone"} ok={overview.relay.connected} state={overview.relay.connected ? "Online" : "Offline"} />
          <div className="health-meta"><span>Heartbeat</span><strong>{relativeTime(overview.relay.last_seen_at)}</strong></div>
          <div className="health-meta"><span>App version</span><strong>{overview.relay.app_version || "—"}</strong></div>
          <div className="divider" />
          <HealthRow label="Merchant webhooks" detail={`${overview.webhooks.pending} queued`} ok={overview.webhooks.exhausted === 0} state={overview.webhooks.exhausted ? `${overview.webhooks.exhausted} exhausted` : "Healthy"} />
          <div className="health-meta"><span>Last delivered</span><strong>{relativeTime(overview.webhooks.last_delivered_at)}</strong></div>
        </article>
      </section>

      <section className="dashboard-grid dashboard-grid-secondary">
        <article className="panel recent-payments-panel">
          <div className="panel-head"><div><p className="eyebrow">Recent payments</p><h3>Latest transactions</h3></div><span className="muted">Newest first</span></div>
          <div className="payment-compact-list">{payments.length ? payments.map((payment) => <button key={payment.id} onClick={() => onOpenPayment(payment.id)}>
            <span className="payment-avatar">{(payment.name || payment.external_id || "P").slice(0, 1).toUpperCase()}</span>
            <span className="payment-compact-main"><strong>{payment.name || payment.external_id || payment.id}</strong><small>{payment.external_id || payment.collection_profile_id} · {relativeTime(payment.created_at)}</small></span>
            <span className="payment-compact-end"><strong>{money(payment.payable_amount_paise)}</strong><Badge tone={statusTone(payment.status)}>{payment.status}</Badge></span>
          </button>) : <div className="empty-inline">No payments yet.</div>}</div>
        </article>

        <article className="panel ops-panel">
          <div className="panel-head"><div><p className="eyebrow">Collection</p><h3>Active profile</h3></div><Badge tone={overview.active_profile ? "good" : "warn"}>{overview.active_profile ? "Active" : "Missing"}</Badge></div>
          <div className="profile-focus"><div className="profile-focus-icon">₹</div><div><strong>{overview.active_profile?.label ?? "No profile"}</strong><span>{overview.active_profile?.id ?? "Configure a destination before taking payments"}</span></div></div>
          <div className="ops-grid"><div><span>Pending now</span><strong>{overview.pending}</strong></div><div><span>Expired today</span><strong>{overview.expired_today}</strong></div><div><span>Webhook queue</span><strong>{overview.webhooks.pending}</strong></div><div><span>Exhausted</span><strong>{overview.webhooks.exhausted}</strong></div></div>
        </article>
      </section>

      <section className="panel recent-panel activity-panel">
        <div className="panel-head"><div><p className="eyebrow">Recent activity</p><h3>What PayGate just did</h3></div><span className="muted">Auto-refreshes</span></div>
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

function MetricCard({ label, value, detail, accent }: { label: string; value: string; detail: string; accent: string }) {
  return <article className={`stat stat-${accent}`}><div className="stat-icon"><span/></div><span>{label}</span><strong>{value}</strong><small>{detail}</small><div className="stat-spark"><i/><i/><i/><i/><i/><i/></div></article>;
}

function AreaChart({ points }: { points: DailyVolume[] }) {
  const width = 720, height = 230, padX = 18, padY = 24;
  const max = Math.max(1, ...points.map((p) => p.amount_paise));
  const coords = points.map((point, i) => ({
    x: padX + (points.length <= 1 ? 0 : i * ((width - padX * 2) / (points.length - 1))),
    y: height - padY - (point.amount_paise / max) * (height - padY * 2),
  }));
  const line = coords.length ? `M ${coords.map((p) => `${p.x} ${p.y}`).join(" L ")}` : "";
  const area = coords.length ? `${line} L ${coords.at(-1)!.x} ${height - padY} L ${coords[0].x} ${height - padY} Z` : "";
  return <div className="area-chart-wrap">
    <svg className="area-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Seven day settled payment volume">
      <defs><linearGradient id="volume-fill" x1="0" y1="0" x2="0" y2="1"><stop stopColor="#20e995" stopOpacity=".34"/><stop offset="1" stopColor="#20e995" stopOpacity="0"/></linearGradient><linearGradient id="volume-line" x1="0" y1="0" x2="1" y2="0"><stop stopColor="#57ff72"/><stop offset=".55" stopColor="#18e5a3"/><stop offset="1" stopColor="#08b5ca"/></linearGradient></defs>
      {[.25,.5,.75,1].map((v) => <line key={v} x1={padX} x2={width-padX} y1={padY + (height-padY*2)*v} y2={padY + (height-padY*2)*v} className="chart-gridline"/>)}
      {area && <path d={area} fill="url(#volume-fill)"/>}{line && <path d={line} fill="none" stroke="url(#volume-line)" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round"/>}
      {coords.map((p,i) => <circle key={i} cx={p.x} cy={p.y} r="4" className="chart-dot"/>)}
    </svg>
    <div className="chart-labels">{points.map((point) => <span key={point.date}><strong>{point.payments}</strong><small>{new Date(`${point.date}T00:00:00`).toLocaleDateString("en-IN", { weekday: "short" })}</small></span>)}</div>
  </div>;
}

function HealthRow({ label, detail, ok, state }: { label: string; detail: string; ok: boolean; state: string }) {
  return <div className="health-row"><div><Dot ok={ok}/><div><strong>{label}</strong><span>{detail}</span></div></div><Badge tone={ok ? "good" : "bad"}>{state}</Badge></div>;
}

function donutGradient(counts: Record<string, number>): string {
  const ordered = ["paid", "pending", "expired", "cancelled"];
  const total = ordered.reduce((sum, key) => sum + (counts[key] ?? 0), 0);
  if (!total) return "conic-gradient(#16252b 0 100%)";
  let current = 0;
  const parts = ordered.map((key) => {
    const start = current;
    current += ((counts[key] ?? 0) / total) * 100;
    return `${statusColors[key]} ${start}% ${current}%`;
  });
  return `conic-gradient(${parts.join(",")})`;
}

function statusTone(status: string): "neutral" | "good" | "warn" | "bad" | "blue" {
  if (status === "paid") return "good";
  if (status === "pending") return "warn";
  if (status === "cancelled") return "bad";
  return "neutral";
}

export function ActivityMark({ kind, status }: { kind: string; status?: string }) {
  const good = status === "paid" || status === "matched" || status === "corroborated" || status === "delivered";
  const bad = status === "exhausted" || status === "ambiguous";
  return <span className={`activity-mark ${good ? "good" : bad ? "bad" : kind === "webhook" ? "blue" : ""}`}>{kind === "webhook" ? "↗" : kind === "payment_detected" ? "₹" : "•"}</span>;
}
