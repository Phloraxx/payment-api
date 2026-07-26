import QRCode from "qrcode";
import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate } from "../components/common";
import { api } from "../pb";
import type { Connector } from "../types";

type SafeConfig = {
  upiId: string;
  upiPayeeName: string;
  paymentTtlSeconds: number;
  quarantineSeconds: number;
  webhookConfigured: boolean;
  rateLimitsEnabled: boolean;
  legacySMSWebhookEnabled: boolean;
  connector: Connector;
};

type PairResponse = { qrUrl: string; status: Connector };

export function Settings({ notify }: { notify: (value: string) => void }) {
  const [config, setConfig] = useState<SafeConfig | null>(null);
  const [connector, setConnector] = useState<Connector | null>(null);
  const [qrUrl, setQrUrl] = useState("");
  const [qrImage, setQrImage] = useState("");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [cfg, status] = await Promise.all([
        api<SafeConfig>("/api/config"),
        api<Connector>("/api/connector/gmessages/status"),
      ]);
      setConfig(cfg);
      setConnector(status);
      if (status.state !== "pairing" && status.paired) {
        setQrUrl("");
        setQrImage("");
      }
    } catch (err) {
      notify(err instanceof Error ? err.message : "Could not load settings.");
    }
  }, [notify]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 10_000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  useEffect(() => {
    if (!qrUrl) { setQrImage(""); return; }
    void QRCode.toDataURL(qrUrl, { width: 420, margin: 2, errorCorrectionLevel: "L" }).then(setQrImage).catch(() => setQrImage(""));
  }, [qrUrl]);

  useEffect(() => {
    if (!qrUrl || connector?.state !== "pairing") return;
    const timer = window.setInterval(async () => {
      try {
        const result = await api<PairResponse>("/api/connector/gmessages/pair/refresh", { method: "POST" });
        setQrUrl(result.qrUrl);
        setConnector(result.status);
      } catch (err) {
        notify(err instanceof Error ? err.message : "Could not refresh pairing QR.");
      }
    }, 20_000);
    return () => window.clearInterval(timer);
  }, [qrUrl, connector?.state, notify]);

  async function startPairing() {
    setBusy(true);
    try {
      const result = await api<PairResponse>("/api/connector/gmessages/pair", { method: "POST" });
      setQrUrl(result.qrUrl);
      setConnector(result.status);
      notify("Pairing started. Scan the QR from Google Messages.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Pairing could not start.");
    } finally {
      setBusy(false);
    }
  }

  async function reconnect() {
    setBusy(true);
    try {
      setConnector(await api<Connector>("/api/connector/gmessages/reconnect", { method: "POST" }));
      notify("Reconnect requested.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Reconnect failed.");
    } finally { setBusy(false); }
  }

  async function unpair() {
    if (!window.confirm("Remove the stored Google Messages pairing from PayGate?")) return;
    setBusy(true);
    try {
      setConnector(await api<Connector>("/api/connector/gmessages/pair", { method: "DELETE" }));
      setQrUrl(""); setQrImage("");
      notify("Google Messages unpaired.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Unpair failed.");
    } finally { setBusy(false); }
  }

  const enabled = connector?.enabled ?? false;
  return <>
    <section className="card">
      <div className="section-title">
        <div><p className="eyebrow">GOOGLE MESSAGES</p><h2>{enabled ? connector?.state ?? "loading" : "disabled"}</h2></div>
        <Badge status={connector?.connected ? "connected" : connector?.state ?? "disabled"} />
      </div>
      <p className="muted">{connector?.lastError || "Read-only SMS connector. The authenticated Android SMS webhook remains available as fallback."}</p>
      <dl className="settings compact">
        <div><dt>Paired</dt><dd>{connector?.paired ? "Yes" : "No"}</dd></div>
        <div><dt>Phone responsive</dt><dd>{connector?.phoneResponsive ? "Yes" : "No / unknown"}</dd></div>
        <div><dt>Last connected</dt><dd>{formatDate(connector?.lastConnectedAt)}</dd></div>
        <div><dt>Last bank SMS</dt><dd>{formatDate(connector?.lastMessageAt)}</dd></div>
      </dl>
      <div className="actions">
        <button className="primary" disabled={!enabled || connector?.paired || busy} onClick={() => void startPairing()}>Start QR pairing</button>
        <button disabled={!enabled || !connector?.paired || busy} onClick={() => void reconnect()}>Reconnect</button>
        <button className="danger" disabled={!connector?.paired || busy} onClick={() => void unpair()}>Unpair</button>
      </div>
      {qrImage && <div className="pair-panel"><img src={qrImage} alt="Google Messages pairing QR" /><p>Google Messages → Device pairing → Switch to QR pairing → scan this code.</p><p className="muted">The QR refreshes automatically before the token expires.</p></div>}
    </section>

    <section className="card">
      <p className="eyebrow">SAFE CONFIGURATION</p>
      {config ? <dl className="settings">
        <div><dt>UPI ID</dt><dd>{config.upiId}</dd></div>
        <div><dt>Payee name</dt><dd>{config.upiPayeeName}</dd></div>
        <div><dt>Payment TTL</dt><dd>{config.paymentTtlSeconds}s</dd></div>
        <div><dt>Amount quarantine</dt><dd>{Math.round(config.quarantineSeconds / 3600)}h</dd></div>
        <div><dt>Outgoing webhook</dt><dd>{config.webhookConfigured ? "Configured" : "Disabled"}</dd></div>
        <div><dt>API rate limits</dt><dd>{config.rateLimitsEnabled ? "Enabled" : "Disabled"}</dd></div>
        <div><dt>Legacy /api/webhook</dt><dd>{config.legacySMSWebhookEnabled ? "Enabled (migration only)" : "Disabled"}</dd></div>
      </dl> : <p className="empty">Loading configuration…</p>}
    </section>
  </>;
}
