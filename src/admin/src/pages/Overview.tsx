import { useEffect, useState } from "react";
import { getLogs, getPool, getStats } from "../api/tickets";
import { PoolGauge } from "../components/PoolGauge";
import { StatsCard } from "../components/StatsCard";
import type { LogEntry, PoolSnapshot } from "../types";

export function Overview() {
  const [stats, setStats] = useState<Record<string, number>>({});
  const [pools, setPools] = useState<PoolSnapshot[]>([]);
  const [logs, setLogs] = useState<LogEntry[]>([]);

  useEffect(() => {
    void Promise.all([getStats(), getPool(), getLogs("?limit=8")]).then(([statsResult, poolResult, logResult]) => {
      setStats(statsResult);
      setPools(poolResult.pools);
      setLogs(logResult.logs);
    });
  }, []);

  return (
    <div className="page">
      <div className="stats-grid">
        <StatsCard label="Total Tickets" value={stats.total ?? 0} />
        <StatsCard label="Pending" value={stats.pending ?? 0} />
        <StatsCard label="Paid" value={stats.paid ?? 0} />
        <StatsCard label="Revenue" value={`₹${((stats.revenue ?? 0) / 100).toFixed(2)}`} />
      </div>
      {pools[0] && <PoolGauge pool={pools[0]} />}
      <section className="panel">
        <h2>Recent Activity</h2>
        <ul className="activity">
          {logs.map((log) => (
            <li key={log.id}>
              <span className={`level ${log.level}`}>{log.level}</span>
              {log.message}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
