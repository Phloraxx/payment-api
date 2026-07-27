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

type QRPairResponse = { qrUrl: string; status: Connector };
type GooglePairResponse = { emoji: string; accountEmail: string; status: Connector };

const googleMessagesConfigURL = "https://accounts.google.com/AccountChooser?continue=https://messages.google.com/web/config";

export function Settings({ notify }: { notify: (value: string) => void }) {
  const [config, setConfig] = useState<SafeConfig | null>(null);
  const [connector, setConnector] = useState<Connector | null>(null);
  const [cookieData, setCookieData] = useState("");
  const [pairingEmoji, setPairingEmoji] = useState("");
  const [pairingAccount, setPairingAccount] = useState("");
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
      if (status.pairingMethod === "google" && status.state === "pairing") {
        setPairingEmoji(status.pairingEmoji ?? "");
        setPairingAccount(status.accountEmail ?? "");
      }
      const refreshingGoogleAuth = status.state === "reauth_required" || status.state === "reauthenticating";
      if (status.paired && !refreshingGoogleAuth) {
        setCookieData("");
        setPairingEmoji("");
        setQrUrl("");
        setQrImage("");
      }
    } catch (err) {
      notify(err instanceof Error ? err.message : "Could not load settings.");
    }
  }, [notify]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 3_000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  useEffect(() => {
    if (!qrUrl) { setQrImage(""); return; }
    void QRCode.toDataURL(qrUrl, { width: 420, margin: 2, errorCorrectionLevel: "L" }).then(setQrImage).catch(() => setQrImage(""));
  }, [qrUrl]);

  useEffect(() => {
    if (!qrUrl || connector?.state !== "pairing" || connector.pairingMethod !== "qr") return;
    const timer = window.setInterval(async () => {
      try {
        const result = await api<QRPairResponse>("/api/connector/gmessages/pair/qr/refresh", { method: "POST" });
        setQrUrl(result.qrUrl);
        setConnector(result.status);
      } catch (err) {
        notify(err instanceof Error ? err.message : "Could not refresh pairing QR.");
      }
    }, 20_000);
    return () => window.clearInterval(timer);
  }, [qrUrl, connector?.state, connector?.pairingMethod, notify]);

  async function startGooglePairing() {
    if (!cookieData.trim()) {
      notify("Paste the Google Messages cookie data or Copy-as-cURL request first.");
      return;
    }
    setBusy(true);
    try {
      const result = await api<GooglePairResponse>("/api/connector/gmessages/pair/google", {
        method: "POST",
        body: JSON.stringify({ cookieData }),
      });
      setCookieData("");
      setPairingEmoji(result.emoji);
      setPairingAccount(result.accountEmail);
      setConnector(result.status);
      notify("Pairing request sent. Tap the matching emoji in Google Messages on your phone.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Google account pairing could not start.");
    } finally {
      setBusy(false);
    }
  }

  async function refreshGoogleLogin() {
    if (!cookieData.trim()) {
      notify("Paste a fresh Google Messages Copy-as-cURL request first.");
      return;
    }
    setBusy(true);
    try {
      const status = await api<Connector>("/api/connector/gmessages/reauth/google", {
        method: "POST",
        body: JSON.stringify({ cookieData }),
      });
      setCookieData("");
      setConnector(status);
      notify("Google login refreshed. Reconnecting with the existing phone pairing.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Google login could not be refreshed.");
    } finally {
      setBusy(false);
    }
  }

  async function startQRPairing() {
    setBusy(true);
    try {
      const result = await api<QRPairResponse>("/api/connector/gmessages/pair/qr", { method: "POST" });
      setQrUrl(result.qrUrl);
      setConnector(result.status);
      notify("QR fallback pairing started.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "QR pairing could not start.");
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
    const pairing = connector?.state === "pairing";
    if (!window.confirm(pairing ? "Cancel the current Google Messages pairing?" : "Remove the stored Google Messages pairing from PayGate?")) return;
    setBusy(true);
    try {
      setConnector(await api<Connector>("/api/connector/gmessages/pair", { method: "DELETE" }));
      setCookieData(""); setPairingEmoji(""); setPairingAccount(""); setQrUrl(""); setQrImage("");
      notify(pairing ? "Pairing cancelled." : "Google Messages unpaired.");
    } catch (err) {
      notify(err instanceof Error ? err.message : "Unpair failed.");
    } finally { setBusy(false); }
  }

  const enabled = connector?.enabled ?? false;
  const pairing = connector?.state === "pairing";
  const googleReauth = connector?.paired && connector.pairingMethod === "google" &&
    (connector.state === "reauth_required" || connector.state === "reauthenticating");
  const displayState = connector?.state?.replaceAll("_", " ") ?? "loading";

  return <>
    <section className="card">
      <div className="section-title">
        <div><p className="eyebrow">GOOGLE MESSAGES</p><h2>{enabled ? displayState : "disabled"}</h2></div>
        <Badge status={connector?.connected ? "connected" : connector?.state ?? "disabled"} />
      </div>
      <p className="muted">{connector?.lastError || "Read-only SMS connector. Google account + emoji pairing is preferred; QR remains available as a fallback."}</p>
      <dl className="settings compact">
        <div><dt>Paired</dt><dd>{connector?.paired ? "Yes" : "No"}</dd></div>
        <div><dt>Pairing method</dt><dd>{connector?.pairingMethod || "—"}</dd></div>
        <div><dt>Google account</dt><dd>{connector?.accountEmail || pairingAccount || "—"}</dd></div>
        <div><dt>Phone responsive</dt><dd>{connector?.phoneResponsive ? "Yes" : "No / unknown"}</dd></div>
        <div><dt>Last connected</dt><dd>{formatDate(connector?.lastConnectedAt)}</dd></div>
        <div><dt>Last bank SMS</dt><dd>{formatDate(connector?.lastMessageAt)}</dd></div>
      </dl>

      {enabled && !connector?.paired && !pairing && <div className="google-pair-setup">
        <h3>Google account pairing</h3>
        <p className="muted">Open Google Messages Web in the same Google account, then in browser DevTools → Network reload the page, select the <code>config</code> request, and use <strong>Copy as cURL</strong>. Paste it below. A raw Cookie header or JSON cookie object also works.</p>
        <p><a href={googleMessagesConfigURL} target="_blank" rel="noreferrer">Open Google Messages account/config ↗</a></p>
        <textarea
          value={cookieData}
          onChange={(event) => setCookieData(event.target.value)}
          placeholder="Paste Copy-as-cURL, Cookie: SID=…; HSID=…; …, or cookie JSON"
          autoComplete="off"
          spellCheck={false}
          rows={7}
        />
        <p className="muted">Cookie values are never returned to the browser or written to application logs. PayGate persists them in the restricted session file only after pairing succeeds.</p>
        <div className="actions">
          <button className="primary" disabled={busy || !cookieData.trim()} onClick={() => void startGooglePairing()}>Start Google account pairing</button>
          <button disabled={busy} onClick={() => void startQRPairing()}>Use QR fallback</button>
        </div>
      </div>}

      {googleReauth && <div className="google-pair-setup">
        <h3>Refresh Google login</h3>
        <p className="muted">The phone pairing and encryption keys are still saved. Only the Google browser login expired, so no emoji or new device pairing is required.</p>
        <p className="muted">Open Google Messages Web with <strong>{connector?.accountEmail || "the already paired Google account"}</strong>, then DevTools → Network → reload → <code>config</code> → <strong>Copy as cURL</strong>. Paste the fresh request below.</p>
        <p><a href={googleMessagesConfigURL} target="_blank" rel="noreferrer">Open Google Messages account/config ↗</a></p>
        <textarea
          value={cookieData}
          onChange={(event) => setCookieData(event.target.value)}
          placeholder="Paste a fresh Copy-as-cURL request"
          autoComplete="off"
          spellCheck={false}
          rows={7}
        />
        <p className="muted">PayGate verifies that the cookies belong to the same Google account before replacing the stored authentication. Cookie values are never echoed back or written to application logs.</p>
        <div className="actions">
          <button className="primary" disabled={busy || !cookieData.trim()} onClick={() => void refreshGoogleLogin()}>{busy ? "Refreshing…" : "Refresh Google login"}</button>
        </div>
      </div>}

      {pairing && connector?.pairingMethod === "google" && <div className="pair-panel">
        <p className="eyebrow">PAIRING EMOJI</p>
        <div className="pair-emoji" aria-label={`Pairing emoji ${pairingEmoji || connector.pairingEmoji || ""}`}>{pairingEmoji || connector.pairingEmoji || "…"}</div>
        <h3>Tap this emoji on your phone</h3>
        <p>Google Messages → Device pairing. Choose the matching emoji when the pairing request appears.</p>
        <p className="muted">Keep Google Messages open and the phone online. This request expires automatically.</p>
      </div>}

      {qrImage && connector?.pairingMethod === "qr" && <div className="pair-panel"><img src={qrImage} alt="Google Messages pairing QR" /><p>Google Messages → Device pairing → Switch to QR pairing → scan this code.</p><p className="muted">The QR refreshes automatically before the token expires.</p></div>}

      <div className="actions">
        <button disabled={!enabled || !connector?.paired || googleReauth || busy} onClick={() => void reconnect()}>Reconnect</button>
        <button className="danger" disabled={!enabled || (!connector?.paired && !pairing) || busy} onClick={() => void unpair()}>{pairing ? "Cancel pairing" : "Unpair"}</button>
      </div>
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
