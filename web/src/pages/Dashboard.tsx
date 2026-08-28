import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate } from "../components/common";
import { api } from "../api";
import type { OperatorOverviewResponse } from "../types";
import { PaymentTable } from "./Payments";

export function Dashboard() {
  const [data, setData] = useState<OperatorOverviewResponse | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api<OperatorOverviewResponse>("/api/operator/v2/overview?limit=8"));
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load dashboard");
    }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") void load();
    }, 10_000);
    return () => window.clearInterval(timer);
  }, [load]);

  const stats = data?.overview.paymentCounts ?? {};
  const connector = data?.connector;
  const capacityPools = data?.capacity?.pools?.slice(0, 6) ?? [];
  const backup = data?.backup;
  const relay = data?.relay;
  const openReviews = data?.overview.openReviews ?? 0;
  const openAlerts = data?.overview.openAlerts ?? 0;
  const attention = openReviews + openAlerts;
  const settled = (stats.paid ?? 0) + (stats.late ?? 0);

  return <>
    {error && <p className="error banner">{error}</p>}
    <section className="command-hero">
      <div>
        <p className="eyebrow">LIVE PAYMENT OPERATIONS</p>
        <h2>{attention ? `${attention} item${attention === 1 ? "" : "s"} need attention` : "All clear"}</h2>
        <p className="muted">PayGate is authoritative. Reviews and alerts are the only queues that require operator intervention.</p>
      </div>
      <div className="command-metrics">
        <div><span>Pending</span><strong>{stats.pending ?? 0}</strong></div>
        <div><span>Settled</span><strong>{settled}</strong></div>
        <div className={attention ? "attention" : ""}><span>Exceptions</span><strong>{attention}</strong></div>
      </div>
    </section>

    <div className="attention-grid">
      <a className={`attention-card ${openReviews ? "active" : "quiet"}`} href="#/reviews">
        <span>Review queue</span><strong>{openReviews}</strong><small>{openReviews ? "Evidence needs a decision" : "No payment evidence waiting"}</small>
      </a>
      <a className={`attention-card ${openAlerts ? "active" : "quiet"}`} href="#/alerts">
        <span>Operational alerts</span><strong>{openAlerts}</strong><small>{openAlerts ? "Infrastructure needs attention" : "No open operational alerts"}</small>
      </a>
      <div className="attention-card quiet"><span>Late settlements</span><strong>{stats.late ?? 0}</strong><small>Verified after the checkout window</small></div>
      <div className="attention-card quiet"><span>Expired</span><strong>{stats.expired ?? 0}</strong><small>No verified evidence in time</small></div>
    </div>

    <section className="rail-strip">
      <div className="rail-status">
        <div><p className="eyebrow">KOTAK / GOOGLE MESSAGES</p><strong>{connector?.enabled ? connector.state.replaceAll("_", " ") : "disabled"}</strong></div>
        <Badge status={connector?.connected ? "connected" : connector?.state ?? "disabled"} />
        <small>{connector?.lastError || (connector?.phoneResponsive ? `Phone responsive · last bank SMS ${formatDate(connector.lastMessageAt)}` : "Awaiting phone response")}</small>
      </div>
      <div className="rail-status">
        <div><p className="eyebrow">PAYTM / ANDROID RELAY</p><strong>{relay?.ready ? "ready" : "unavailable"}</strong></div>
        <Badge status={relay?.ready ? "connected" : "warning"} />
        <small>{relay ? `${relay.activeDevices}/${relay.enabledDevices} active · heartbeat ${formatDate(relay.lastHeartbeatAt ?? undefined)} · ${relay.pendingQueueCount} queued` : "Relay status unavailable"}</small>
      </div>
      <div className="rail-status">
        <div><p className="eyebrow">RECOVERY</p><strong>{backup?.enabled ? "protected" : "disabled"}</strong></div>
        <Badge status={backup?.offsite ? "offsite" : backup?.enabled ? "local" : "disabled"} />
        <small>{backup?.error || (backup?.latest ? `Latest backup ${formatDate(backup.latest.modTime)}` : backup?.enabled ? "Waiting for first backup" : "Backup schedule disabled")}</small>
      </div>
    </section>

    <div className="dashboard-split">
      <section className="card recent-panel">
        <div className="section-title"><div><p className="eyebrow">MONEY FLOW</p><h2>Recent payments</h2></div><a className="text-link" href="#/payments">View all</a></div>
        <PaymentTable limit={8} />
      </section>
      <section className="card capacity-panel">
        <div className="section-title"><div><p className="eyebrow">ALLOCATION</p><h2>Fingerprint capacity</h2></div></div>
        {!capacityPools.length ? <p className="empty">No active or quarantined pools.</p> : <div className="capacity-list compact">{capacityPools.map((pool) => <div className="capacity-row compact" key={pool.requestedAmountPaise}><div><strong>₹{pool.requestedAmount}</strong><small>{pool.pending} pending · {pool.available} free</small></div><Badge status={pool.level} /><progress className={`capacity-meter ${pool.level}`} max={100} value={Math.min(pool.utilizationPercent, 100)} /></div>)}</div>}
        <p className="muted capacity-note">Allocation warns at 70% and becomes critical at 95%. Quarantine prevents fingerprint reuse until evidence can no longer arrive safely.</p>
      </section>
    </div>
  </>;
}
