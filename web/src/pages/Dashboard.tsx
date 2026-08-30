import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate } from "../components/common";
import { api } from "../api";
import type { OperatorOverviewResponse, OperatorPaymentSummary } from "../types";

export function Dashboard() {
  const [data, setData] = useState<OperatorOverviewResponse | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api<OperatorOverviewResponse>("/api/operator/v2/overview?limit=6"));
      setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load PayGate overview"); }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 10_000);
    return () => window.clearInterval(timer);
  }, [load]);

  const stats = data?.overview.paymentCounts ?? {};
  const reviews = data?.overview.openReviews ?? 0;
  const alerts = data?.overview.openAlerts ?? 0;
  const attention = reviews + alerts;
  const relayReady = data?.relay?.ready === true;
  const connectorReady = data?.connector?.connected === true;
  const settled = (stats.paid ?? 0) + (stats.late ?? 0);

  return <div className="overview-layout">
    {error && <div className="soft-error">{error}<button onClick={() => void load()}>Retry</button></div>}
    <section className={attention ? "overview-hero attention" : "overview-hero"}>
      <div className="overview-hero-copy">
        <span className="hero-status"><i /> {attention ? "Attention needed" : "PayGate is ready"}</span>
        <h2>{attention ? `${attention} ${attention === 1 ? "thing needs" : "things need"} a check.` : "Everything looks good."}</h2>
        <p>{attention ? "PayGate has stopped where it should. Open Action to make the decisions that cannot be automated safely." : "Payments are being watched automatically. You only need to step in when PayGate asks."}</p>
        {attention > 0 && <a className="hero-action" href="#/reviews">Open Action <span>→</span></a>}
      </div>
      <div className="hero-number"><strong>{stats.pending ?? 0}</strong><span>pending now</span></div>
    </section>

    <section className="metric-row">
      <Metric label="Settled" value={settled} detail="verified payments" />
      <Metric label="Pending" value={stats.pending ?? 0} detail="waiting for evidence" />
      <Metric label="Action" value={attention} detail="needs a person" tone={attention ? "warn" : ""} />
      <Metric label="Expired" value={stats.expired ?? 0} detail="closed without match" />
    </section>

    <div className="overview-grid">
      <section className="panel recent-payments-panel">
        <div className="panel-heading"><div><span>Recent activity</span><h3>Payments</h3></div><a href="#/payments">View all</a></div>
        <div className="recent-list">
          {(data?.overview.recentPayments ?? []).map((payment) => <RecentPayment key={payment.id} payment={payment} />)}
          {data && data.overview.recentPayments.length === 0 && <div className="empty-state compact">No payments yet.</div>}
        </div>
      </section>
      <section className="panel readiness-panel">
        <div className="panel-heading"><div><span>Automatic verification</span><h3>Readiness</h3></div><a href="#/health">Health</a></div>
        <ReadinessRow label="Kotak SMS" ready={connectorReady} detail={connectorReady ? "Google Messages connected" : "Connector needs attention"} />
        <ReadinessRow label="Paytm" ready={relayReady} detail={relayReady ? "Android relay ready" : "Phone relay needs attention"} />
        <ReadinessRow label="Recovery" ready={data?.backup?.enabled === true && !data?.backup?.error} detail={data?.backup?.enabled ? (data.backup.error || "Backups enabled") : "Backup schedule disabled"} />
        <p className="readiness-footnote">Payment matching remains fail-closed if a verification rail is not healthy.</p>
      </section>
    </div>
  </div>;
}

function Metric({ label, value, detail, tone = "" }: { label: string; value: number; detail: string; tone?: string }) {
  return <article className={`metric-tile ${tone}`}><span>{label}</span><strong>{value}</strong><small>{detail}</small></article>;
}

function RecentPayment({ payment }: { payment: OperatorPaymentSummary }) {
  const name = payment.displayName || payment.externalId || payment.customerName || payment.id;
  return <a className="recent-payment" href={`#/payments?open=${encodeURIComponent(payment.id)}`}>
    <div className="payment-avatar">{name.slice(0, 1).toUpperCase()}</div>
    <div className="recent-payment-main"><strong>{name}</strong><span>{payment.customerName || payment.id} · {formatDate(payment.createdAt)}</span></div>
    <div className="recent-payment-end"><strong>{money(payment.payableAmountPaise)}</strong><Badge status={payment.status} /></div>
  </a>;
}

function ReadinessRow({ label, ready, detail }: { label: string; ready: boolean; detail: string }) {
  return <div className="readiness-row"><span className={ready ? "readiness-dot ready" : "readiness-dot"} /><div><strong>{label}</strong><small>{detail}</small></div><span className={ready ? "readiness-state ready" : "readiness-state"}>{ready ? "Ready" : "Check"}</span></div>;
}

function money(paise: number) { return `₹${(paise / 100).toFixed(2)}`; }
