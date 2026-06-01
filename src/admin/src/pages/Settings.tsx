import { useEffect, useState } from "react";
import { fullSync, getSettings } from "../api/tickets";
import type { Settings } from "../types";

export function Settings() {
  const [message, setMessage] = useState("");
  const [settings, setSettings] = useState<Settings>({} as Settings);
  useEffect(() => {
    void getSettings().then(setSettings);
  }, []);
  return (
    <div className="page">
      <section className="panel form-panel">
        <h2>Environment</h2>
        <dl className="settings-list">
          {Object.entries(settings).map(([key, value]) => (
            <div key={key}>
              <dt>{key}</dt>
              <dd>{String(value)}</dd>
            </div>
          ))}
        </dl>
      </section>
      <section className="panel form-panel">
        <h2>Appwrite Sync</h2>
        <p>SQLite remains the source of truth. Use full sync after Appwrite downtime.</p>
        <button onClick={() => void fullSync().then((result) => setMessage(`Attempted ${result.attempted}, failed ${result.failed}`))}>Re-sync All</button>
        {message && <div className="result">{message}</div>}
      </section>
    </div>
  );
}
