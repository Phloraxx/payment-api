import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate } from "../components/common";
import { api } from "../api";
import type { EvidenceShadowMetrics, OperatorAlertSummary, OperatorOverviewResponse } from "../types";

export function Health() {
  const [overview, setOverview] = useState<OperatorOverviewResponse | null>(null);
  const [alerts, setAlerts] = useState<OperatorAlertSummary[]>([]);
  const [shadow, setShadow] = useState<EvidenceShadowMetrics | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const [nextOverview, alertData, shadowData] = await Promise.all([
        api<OperatorOverviewResponse>("/api/operator/v2/overview?limit=1"),
        api<{ alerts: OperatorAlertSummary[] }>("/api/operator/v2/alerts?status=open&limit=50"),
        api<EvidenceShadowMetrics>("/api/operator/v2/evidence-shadow/google-messages?days=14"),
      ]);
      setOverview(nextOverview); setAlerts(alertData.alerts); setShadow(shadowData); setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not load health status"); }
  }, []);

  useEffect(() => { void load(); const timer = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 15_000); return () => window.clearInterval(timer); }, [load]);

  const connector = overview?.connector;
  const relay = overview?.relay;
  const backup = overview?.backup;
  const connectorReady = connector?.connected === true && connector?.phoneResponsive !== false;
  const relayReady = relay?.ready === true;
  const backupReady = backup?.enabled === true && !backup?.error;
  const systemReady = connectorReady && relayReady && alerts.length === 0;

  return <div className="health-layout">
    {error && <div className="soft-error">{error}<button onClick={() => void load()}>Retry</button></div>}
    <section className={systemReady ? "health-hero ready" : "health-hero"}>
      <span className="health-orb">{systemReady ? "✓" : "!"}</span><div><span>{systemReady ? "All core systems ready" : "PayGate needs attention"}</span><h2>{systemReady ? "Automatic verification can run safely." : "One or more verification paths need a check."}</h2><p>PayGate fails closed when a rail is unhealthy, so payment truth is never guessed.</p></div><button className="secondary-action" onClick={() => void load()}>Refresh</button>
    </section>

    <section className="health-grid">
      <HealthCard title="Kotak SMS" ready={connectorReady} state={connector?.state?.replaceAll("_", " ") || "unknown"} detail={connector?.lastError || (connectorReady ? `Phone connected · last message ${formatDate(connector?.lastMessageAt)}` : "Google Messages connector needs attention")} />
      <HealthCard title="Paytm relay" ready={relayReady} state={relayReady ? "ready" : "check"} detail={relay ? `${relay.activeDevices}/${relay.enabledDevices} active · ${relay.pendingQueueCount} queued · heartbeat ${formatDate(relay.lastHeartbeatAt ?? undefined)}` : "Android relay status unavailable"} />
      <HealthCard title="Recovery" ready={backupReady} state={backupReady ? "protected" : "check"} detail={backup?.error || (backup?.latest ? `Latest backup ${formatDate(backup.latest.modTime)}` : backup?.enabled ? "Waiting for first backup" : "Backup schedule disabled")} />
      <HealthCard title="Action queue" ready={alerts.length === 0} state={alerts.length ? `${alerts.length} open` : "clear"} detail={alerts.length ? "Operational alerts need review" : "No open infrastructure alerts"} />
    </section>

    <div className="health-split">
      <section className="panel"><div className="panel-heading"><div><span>Open alerts</span><h3>{alerts.length ? "Needs attention" : "All clear"}</h3></div><a href="#/alerts">History</a></div>{alerts.length === 0 ? <div className="empty-state compact">No operational alerts are open.</div> : <div className="health-alert-list">{alerts.slice(0, 8).map((alert) => <article key={alert.id}><div><strong>{human(alert.kind)}</strong><p>{alert.message}</p></div><Badge status={alert.severity} /></article>)}</div>}</section>
      <section className="panel"><div className="panel-heading"><div><span>Migration telemetry</span><h3>Google Messages parity</h3></div></div><p className="panel-copy">Android Messages remains shadow-only until the legacy Google Messages path has enough exact paired evidence to retire safely.</p>{shadow && <div className="parity-metrics"><Parity label="Window" value={`${shadow.windowDays} days`} /><Parity label="Exact paired" value={`${shadow.exactMatches} / ${shadow.libgmComplete}`} /><Parity label="Reference coverage" value={`${shadow.referenceCoveragePercent.toFixed(1)}%`} /><Parity label="Exact parity" value={`${shadow.exactParityPercent.toFixed(1)}%`} /></div>}<div className={shadow?.removalReady ? "migration-gate ready" : "migration-gate"}><span>{shadow?.removalReady ? "Ready" : "Collecting"}</span><p>{shadow ? human(shadow.removalGate) : "Metrics unavailable"}</p></div></section>
    </div>
  </div>;
}

function HealthCard({ title, ready, state, detail }: { title: string; ready: boolean; state: string; detail: string }) { return <article className={ready ? "health-card ready" : "health-card"}><div className="health-card-top"><span className={ready ? "readiness-dot ready" : "readiness-dot"} /><Badge status={ready ? "ready" : "warning"} /></div><h3>{title}</h3><strong>{state}</strong><p>{detail}</p></article>; }
function Parity({ label, value }: { label: string; value: string }) { return <div><span>{label}</span><strong>{value}</strong></div>; }
function human(value: string) { return value.replaceAll("_", " ").replace(/\b\w/g, (c) => c.toUpperCase()); }
