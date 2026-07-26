import { useCallback, useEffect, useState } from "react";
import { Badge, Stat } from "../components/common";
import { api, pb } from "../pb";
import type { DashboardData } from "../types";
import { PaymentTable } from "./Payments";

export function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api<DashboardData>("/api/dashboard"));
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load dashboard");
    }
  }, []);

  useEffect(() => {
    void load();
    let disposed = false;
    let unsubscribe: (() => void) | undefined;
    void pb.collection("payments").subscribe("*", () => void load()).then((fn) => {
      if (disposed) void fn(); else unsubscribe = fn;
    });
    const timer = window.setInterval(() => void load(), 30_000);
    return () => { disposed = true; unsubscribe?.(); window.clearInterval(timer); };
  }, [load]);

  const stats = data?.stats ?? {};
  const connector = data?.connector;
  return <>
    {error && <p className="error banner">{error}</p>}
    <div className="grid four">
      <Stat label="Pending" value={stats.pending ?? 0} />
      <Stat label="Paid" value={stats.paid ?? 0} tone="good" />
      <Stat label="Late" value={stats.late ?? 0} tone="warn" />
      <Stat label="Expired" value={stats.expired ?? 0} />
    </div>
    <section className="card split">
      <div>
        <p className="eyebrow">GOOGLE MESSAGES</p>
        <h2>{connector?.enabled ? connector.state : "disabled"}</h2>
        <p className="muted">{connector?.lastError || (connector?.phoneResponsive ? "Phone responding" : "The legacy SMS webhook remains available")}</p>
      </div>
      <Badge status={connector?.connected ? "connected" : connector?.state ?? "disabled"} />
    </section>
    <section className="card">
      <div className="section-title"><h2>Recent payments</h2><span className="muted">Realtime updates</span></div>
      <PaymentTable limit={8} />
    </section>
  </>;
}
