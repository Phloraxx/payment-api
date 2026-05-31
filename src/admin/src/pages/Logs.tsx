import { useCallback, useEffect, useState } from "react";
import { getLogs } from "../api/tickets";
import { useWebSocket } from "../hooks/useWebSocket";
import type { LogEntry } from "../types";

export function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [level, setLevel] = useState("");
  const load = useCallback(() => {
    void getLogs(level ? `?level=${level}` : "").then((result) => setLogs(result.logs));
  }, [level]);
  useEffect(load, [load]);
  useWebSocket("/api/admin/ws", (message) => {
    const typed = message as { type?: string; entry?: LogEntry };
    if (typed.type === "log_entry" && typed.entry) setLogs((current) => [typed.entry!, ...current].slice(0, 200));
  });
  return (
    <div className="page">
      <div className="toolbar">
        <select value={level} onChange={(event) => setLevel(event.target.value)}>
          <option value="">All levels</option>
          <option value="info">Info</option>
          <option value="warn">Warn</option>
          <option value="error">Error</option>
          <option value="debug">Debug</option>
        </select>
        <button onClick={load}>Refresh</button>
      </div>
      <section className="panel log-list">
        {logs.map((log) => (
          <article key={log.id}>
            <span className={`level ${log.level}`}>{log.level}</span>
            <strong>{log.message}</strong>
            <code>{log.meta}</code>
          </article>
        ))}
      </section>
    </div>
  );
}
