import QRCode from "qrcode";
import { useCallback, useEffect, useState } from "react";
import { Badge, formatDate } from "../components/common";
import { api } from "../api";
import type { BackupStatus, Connector, RelayDevice, RelayStatus } from "../types";

type SafeConfig = {
  upiId: string;
  upiPayeeName: string;
  defaultPaymentAccount: "kotak" | "slice" | "paytm";
  paymentAccounts: Array<{ id: "kotak" | "slice" | "paytm"; label: string; verification: "sms" | "email" | "notification"; flow: "upi_intent" | "qr_only" | "merchant_qr"; ready: boolean; unavailableReason?: string }>;
  paymentTtlSeconds: number;
  quarantineSeconds: number;
  webhookConfigured: boolean;
  rateLimitsEnabled: boolean;
  legacySMSWebhookEnabled: boolean;
  emailEvidenceEnabled: boolean;
  emailAllowedSender: string;
  retentionEnabled: boolean;
  smsRawRetentionSeconds: number;
  emailRawRetentionSeconds: number;
  reconciliationRawRetentionSeconds: number;
  auditRetentionSeconds: number;
  paytmNotificationRawRetentionSeconds: number;
  relayRawRetentionSeconds: number;
  androidRelayEnrollmentEnabled: boolean;
  androidRelayStaleAfterSeconds: number;
  backupEnabled: boolean;
  backupCron: string;
  backupMaxKeep: number;
  backupOffsite: boolean;
  operatorAlertWebhookConfigured: boolean;
  statementTimezone: string;
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
  const [backup, setBackup] = useState<BackupStatus | null>(null);
  const [backupBusy, setBackupBusy] = useState(false);
  const [relay, setRelay] = useState<RelayStatus | null>(null);
  const [relayDevices, setRelayDevices] = useState<RelayDevice[]>([]);
  const [relayBusy, setRelayBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [cfg, status, backupStatus, relayStatus, relayDeviceResult] = await Promise.all([
        api<SafeConfig>("/api/config"),
        api<Connector>("/api/connector/gmessages/status"),
        api<BackupStatus>("/api/paygate/backups/status"),
        api<RelayStatus>("/api/relay/status"),
        api<{ devices: RelayDevice[] }>("/api/relay/devices"),
      ]);
      setConfig(cfg);
      setConnector(status);
      setBackup(backupStatus);
      setRelay(relayStatus);
      setRelayDevices(relayDeviceResult.devices);
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

  async function setRelayEnabled(device: RelayDevice, enabled: boolean) {
    if (!window.confirm(`${enabled ? "Enable" : "Disable"} relay device ${device.name}?`)) return;
    setRelayBusy(true);
    try {
      await api<{ id: string; enabled: boolean }>(`/api/relay/devices/${encodeURIComponent(device.id)}/enabled`, {
        method: "POST",
        body: JSON.stringify({ enabled }),
      });
      notify(`Relay device ${enabled ? "enabled" : "disabled"}.`);
      await refresh();
    } catch (err) { notify(err instanceof Error ? err.message : "Relay device update failed."); }
    finally { setRelayBusy(false); }
  }

  async function createBackup() {
    setBackupBusy(true);
    try {
      const result = await api<{ name: string }>("/api/paygate/backups", { method: "POST" });
      notify(`Backup created: ${result.name}`);
      await refresh();
    } catch (err) { notify(err instanceof Error ? err.message : "Backup creation failed."); }
    finally { setBackupBusy(false); }
  }

  async function verifyBackup() {
    setBackupBusy(true);
    try {
      const result = await api<BackupStatus>("/api/paygate/backups/verify", { method: "POST" });
      setBackup(result);
      notify(result.latestVerified ? "Latest backup archive verified." : result.verificationError || "Backup verification failed.");
    } catch (err) { notify(err instanceof Error ? err.message : "Backup verification failed."); }
    finally { setBackupBusy(false); }
  }

  async function runRestoreDrill() {
    setBackupBusy(true);
    try {
      const result = await api<{ backupName: string; integrityChecked: number }>("/api/paygate/backups/restore-drill", { method: "POST" });
      notify(`Restore drill passed for ${result.backupName}: ${result.integrityChecked} database file(s) verified.`);
    } catch (err) { notify(err instanceof Error ? err.message : "Restore drill failed."); }
    finally { setBackupBusy(false); }
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
      <div className="section-title">
        <div><p className="eyebrow">ANDROID RELAY</p><h2>{relay?.ready ? "ready" : "unavailable"}</h2></div>
        <Badge status={relay?.ready ? "connected" : "warning"} />
      </div>
      <p className="muted">Paytm QR checkouts fail closed when no recently active relay device is available.</p>
      <dl className="settings compact">
        <div><dt>Active devices</dt><dd>{relay ? `${relay.activeDevices} / ${relay.enabledDevices}` : "—"}</dd></div>
        <div><dt>Power unhealthy</dt><dd>{relay?.powerUnhealthyDevices ?? 0}</dd></div>
        <div><dt>Last heartbeat</dt><dd>{formatDate(relay?.lastHeartbeatAt ?? undefined)}</dd></div>
        <div><dt>Last relay event</dt><dd>{formatDate(relay?.lastEventAt ?? undefined)}</dd></div>
        <div><dt>Last matched payment</dt><dd>{formatDate(relay?.lastMatchedAt ?? undefined)}</dd></div>
        <div><dt>Errors (24h)</dt><dd>{relay?.recentErrorCount ?? 0}</dd></div>
        <div><dt>Device queues</dt><dd>{relay ? `${relay.pendingQueueCount} pending · ${relay.failedQueueCount} failed` : "—"}</dd></div>
        <div><dt>Stale after</dt><dd>{relay ? `${Math.round(relay.staleAfterSeconds / 60)} min` : "—"}</dd></div>
      </dl>
      {!relayDevices.length ? <p className="empty">No relay devices enrolled.</p> : <div className="capacity-list">
        {relayDevices.map((device) => <div className="capacity-row" key={device.id}>
          <div>
            <strong>{device.name}</strong>
            <small>{device.deviceModel || "Android"} · app {device.appVersion || "unknown"} · fingerprint {device.deviceId ? `${device.deviceId.slice(0, 12)}…` : "unknown"}</small>
            <small>Last seen {formatDate(device.lastSeenAt ?? undefined)} · phone delivered {formatDate(device.lastDeliveryAt ?? undefined)} · last event {formatDate(device.lastEventAt ?? undefined)} · last match {formatDate(device.lastMatchedAt ?? undefined)}{!device.lastHeartbeatAt && device.heartbeatGraceUntil ? ` · legacy heartbeat grace until ${formatDate(device.heartbeatGraceUntil)}` : ""}</small>
            <small>Notifications {device.notificationAccess ? "allowed" : device.lastHeartbeatAt ? "blocked" : "not reported"} · listener {device.listenerConnected ? "connected" : device.lastHeartbeatAt ? "disconnected" : "not reported"} · queue {device.pendingCount} pending / {device.failedCount} failed · {device.recentErrorCount} server errors/24h{device.lastClientError ? ` · ${device.lastClientError}` : ""}</small>
            <small>Power {device.powerHealthReported ? (device.powerHealthy ? "ready" : "NOT ready") : "not required/reported"} · battery {device.batteryOptimizationExempt ? "unrestricted" : device.powerHealthReported ? "optimized" : "unknown"} · foreground {device.foregroundService ? "active" : device.powerHealthReported ? "inactive" : "unknown"} · saver {device.powerSaveMode ? "on" : "off"} · background {device.backgroundRestricted ? "RESTRICTED" : "allowed"}</small>
          </div>
          <Badge status={device.active ? "connected" : device.enabled ? "warning" : "disabled"} />
          <button className={device.enabled ? "danger" : ""} disabled={relayBusy} onClick={() => void setRelayEnabled(device, !device.enabled)}>{device.enabled ? "Disable" : "Enable"}</button>
        </div>)}
      </div>}
    </section>

    <section className="card">
      <p className="eyebrow">SAFE CONFIGURATION</p>
      {config ? <dl className="settings">
        <div><dt>Default account</dt><dd>{config.defaultPaymentAccount}</dd></div>
        <div><dt>Enabled UPI accounts</dt><dd>{config.paymentAccounts.map((account) => `${account.label} (${account.verification}${account.ready ? "" : ", unavailable"})`).join(" · ")}</dd></div>
        <div><dt>Legacy Kotak UPI ID</dt><dd>{config.upiId}</dd></div>
        <div><dt>Payee name</dt><dd>{config.upiPayeeName}</dd></div>
        <div><dt>Payment TTL</dt><dd>{config.paymentTtlSeconds}s</dd></div>
        <div><dt>Amount quarantine</dt><dd>{Math.round(config.quarantineSeconds / 3600)}h</dd></div>
        <div><dt>Outgoing webhook</dt><dd>{config.webhookConfigured ? "Configured" : "Disabled"}</dd></div>
        <div><dt>API rate limits</dt><dd>{config.rateLimitsEnabled ? "Enabled" : "Disabled"}</dd></div>
        <div><dt>Legacy /api/webhook</dt><dd>{config.legacySMSWebhookEnabled ? "Enabled (migration only)" : "Disabled"}</dd></div>
        <div><dt>Payment email evidence</dt><dd>{config.emailEvidenceEnabled ? `Enabled · ${config.emailAllowedSender}` : "Disabled"}</dd></div>
        <div><dt>Evidence retention</dt><dd>{config.retentionEnabled ? `SMS ${Math.round(config.smsRawRetentionSeconds / 86400)}d · email ${Math.round(config.emailRawRetentionSeconds / 86400)}d · Paytm ${Math.round(config.paytmNotificationRawRetentionSeconds / 86400)}d · relay ${Math.round(config.relayRawRetentionSeconds / 86400)}d · statements ${Math.round(config.reconciliationRawRetentionSeconds / 86400)}d · audit ${Math.round(config.auditRetentionSeconds / 86400)}d` : "Disabled"}</dd></div>
        <div><dt>Backup schedule</dt><dd>{config.backupEnabled ? `${config.backupCron} · keep ${config.backupMaxKeep}` : "Disabled"}</dd></div>
        <div><dt>Backup storage</dt><dd>{config.backupOffsite ? "S3-compatible offsite" : "Local persistent volume"}</dd></div>
        <div><dt>Operator alert webhook</dt><dd>{config.operatorAlertWebhookConfigured ? "Configured with signed retries" : "Dashboard only"}</dd></div>
        <div><dt>Relay enrollment</dt><dd>{config.androidRelayEnrollmentEnabled ? "OPEN — pair only intentionally" : "Closed"}</dd></div>
        <div><dt>Statement timezone</dt><dd>{config.statementTimezone || "Asia/Kolkata"}</dd></div>
      </dl> : <p className="empty">Loading configuration…</p>}
    </section>

    <section className="card">
      <div className="section-title"><div><p className="eyebrow">DISASTER RECOVERY</p><h2>Backups</h2></div><Badge status={backup?.latestVerified ? "verified" : backup?.enabled ? "configured" : "disabled"} /></div>
      <dl className="settings compact">
        <div><dt>Archives</dt><dd>{backup?.backupCount ?? 0}</dd></div>
        <div><dt>Latest</dt><dd>{backup?.latest?.name || "—"}</dd></div>
        <div><dt>Latest modified</dt><dd>{formatDate(backup?.latest?.modTime)}</dd></div>
        <div><dt>Offsite</dt><dd>{backup?.offsite ? "Yes" : "No"}</dd></div>
        <div><dt>Verification</dt><dd>{backup?.verificationError || (backup?.latestVerified ? "Passed" : "Not run")}</dd></div>
      </dl>
      <div className="actions"><button className="primary" disabled={backupBusy} onClick={() => void createBackup()}>{backupBusy ? "Working…" : "Create backup now"}</button><button disabled={backupBusy || !backup?.latest} onClick={() => void verifyBackup()}>Verify latest archive</button><button disabled={backupBusy || !backup?.latest} onClick={() => void runRestoreDrill()}>Run restore drill</button></div>
      <p className="muted">Verification downloads the archive, reads every ZIP entry and confirms a database file exists. A separate restore drill should still be run periodically on a temporary volume.</p>
    </section>
  </>;
}
