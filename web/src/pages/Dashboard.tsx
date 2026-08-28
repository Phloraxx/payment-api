import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate, Stat } from "../components/common";
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
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load dashboard"); }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 10_000);
    return () => window.clearInterval(timer);
  }, [load]);

  const stats = data?.overview.paymentCounts ?? {};
  const connector = data?.connector;
  const capacityPools = data?.capacity?.pools?.slice(0, 8) ?? [];
  const backup = data?.backup;
  const relay = data?.relay;
  return <>
    {error && <p className="error banner">{error}</p>}
    <div className="grid six">
      <Stat label="Pending" value={stats.pending ?? 0} />
      <Stat label="Paid" value={stats.paid ?? 0} tone="good" />
      <Stat label="Late" value={stats.late ?? 0} tone="warn" />
      <Stat label="Expired" value={stats.expired ?? 0} />
      <Stat label="Open reviews" value={data?.overview.openReviews ?? 0} tone={(data?.overview.openReviews ?? 0) > 0 ? "warn" : ""} />
      <Stat label="Open alerts" value={data?.overview.openAlerts ?? 0} tone={(data?.overview.openAlerts ?? 0) > 0 ? "warn" : ""} />
    </div>
    <div className="grid two">
      <section className="card split"><div><p className="eyebrow">GOOGLE MESSAGES</p><h2>{connector?.enabled ? connector.state.replaceAll("_", " ") : "disabled"}</h2><p className="muted">{connector?.lastError || (connector?.phoneResponsive ? `Phone responding · last bank SMS ${formatDate(connector.lastMessageAt)}` : "Waiting for phone response")}</p></div><Badge status={connector?.connected ? "connected" : connector?.state ?? "disabled"} /></section>
      <section className="card split"><div><p className="eyebrow">ANDROID RELAY</p><h2>{relay?.ready ? "ready" : "unavailable"}</h2><p className="muted">{relay ? `${relay.activeDevices}/${relay.enabledDevices} active · ${relay.powerUnhealthyDevices} power-unhealthy · heartbeat ${formatDate(relay.lastHeartbeatAt ?? undefined)} · queue ${relay.pendingQueueCount}/${relay.failedQueueCount}` : "Relay status unavailable"}</p></div><Badge status={relay?.ready ? "connected" : "warning"} /></section>
      <section className="card split"><div><p className="eyebrow">BACKUPS</p><h2>{backup?.enabled ? `${backup.backupCount} available` : "disabled"}</h2><p className="muted">{backup?.error || (backup?.latest ? `Latest ${backup.latest.name} · ${formatDate(backup.latest.modTime)}` : backup?.enabled ? "No backup created yet" : "Configure a backup cron")}</p></div><Badge status={backup?.offsite ? "offsite" : backup?.enabled ? "local" : "disabled"} /></section>
    </div>
    <section className="card">
      <div className="section-title"><div><p className="eyebrow">99-SUFFIX POOLS</p><h2>Fingerprint capacity</h2></div><span className="muted">70% warning · 95% critical</span></div>
      {!capacityPools.length ? <p className="empty">No active or quarantined fingerprint pools.</p> : <div className="capacity-list">{capacityPools.map((pool) => <div className="capacity-row" key={pool.requestedAmountPaise}><div><strong>₹{pool.requestedAmount}</strong><small>{pool.pending} pending · {pool.quarantined} quarantined · {pool.available} available</small></div><progress className={`capacity-meter ${pool.level}`} max={100} value={Math.min(pool.utilizationPercent, 100)} /><Badge status={pool.level} /></div>)}</div>}
    </section>
    <section className="card"><div className="section-title"><h2>Recent payments</h2><span className="muted">Typed API · 5s refresh</span></div><PaymentTable limit={8} /></section>
  </>;
}
