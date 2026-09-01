import { useCallback, useEffect, useState } from "react";
import { QRCodeCanvas } from "qrcode.react";
import {
  ApiError, activateProfile, changePassword, createApiKey, createPairingSession, getApiKeys, getDevice,
  getProfiles, getWebhookSettings, revokeApiKey, revokeDevice, saveProfile, saveWebhook,
} from "./api";
import type { ApiKeyInfo, DeviceInfo, PairingSession, Profile, WebhookSettings } from "./types";
import { Badge, Dot, ErrorNotice, Modal, SectionHead, Spinner, copyText, dateTime, relativeTime } from "./ui";

export function SettingsPage({ onSignedOut }: { onSignedOut: () => void }) {
  return <>
    <SectionHead eyebrow="Configuration" title="Settings" copy="Control where PayGate receives money, how merchants are notified, and which phone is trusted." />
    <div className="settings-stack">
      <ProfilesSettings />
      <WebhookSettingsPanel />
      <ApiKeysPanel />
      <DevicePanel />
      <PasswordPanel onSignedOut={onSignedOut} />
    </div>
  </>;
}

function ProfilesSettings() {
  const [items, setItems] = useState<Profile[]>([]); const [error, setError] = useState(""); const [editing, setEditing] = useState<Profile | "new">(); const [busy, setBusy] = useState("");
  const load = useCallback(async () => { try { setItems(await getProfiles()); setError(""); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load collection profiles."); } }, []);
  useEffect(() => { void load(); }, [load]);
  async function activate(id: string) { setBusy(id); setError(""); try { await activateProfile(id); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not activate profile."); } finally { setBusy(""); } }
  return <section className="panel settings-panel"><div className="panel-head"><div><p className="eyebrow">Money destination</p><h3>Collection profiles</h3><p>Merchants never choose a rail. PayGate uses the one active profile when it creates a payment.</p></div><button className="button button-secondary button-small" onClick={() => setEditing("new")}>Add profile</button></div>
    {error && <ErrorNotice message={error}/>}<div className="profile-grid">{items.map((profile) => <article className={`profile-card ${profile.active ? "active" : ""}`} key={profile.id}>
      <div className="profile-top"><div><Badge tone={profile.active ? "blue" : profile.enabled ? "good" : "neutral"}>{profile.active ? "Active" : profile.enabled ? "Enabled" : "Disabled"}</Badge><h4>{profile.label}</h4></div><span className="profile-id">{profile.id}</span></div>
      <dl><div><dt>UPI ID</dt><dd>{profile.upi_id}</dd></div><div><dt>Payee</dt><dd>{profile.payee_name || "—"}</dd></div><div><dt>Input</dt><dd>{parserLabel(profile.parser)}</dd></div></dl>
      <div className="profile-actions">{profile.parser !== "legacy" && <button className="text-button" onClick={() => setEditing(profile)}>Edit</button>}{profile.enabled && !profile.active && <button className="button button-primary button-small" disabled={busy === profile.id} onClick={() => void activate(profile.id)}>{busy === profile.id ? <Spinner/> : "Make active"}</button>}{profile.parser === "legacy" && <span className="muted">Historical only</span>}</div>
    </article>)}</div>
    {editing && <ProfileModal profile={editing === "new" ? undefined : editing} onClose={() => setEditing(undefined)} onSaved={async () => { setEditing(undefined); await load(); }}/>}
  </section>;
}

function ProfileModal({ profile, onClose, onSaved }: { profile?: Profile; onClose: () => void; onSaved: () => Promise<void> }) {
  const [id, setId] = useState(profile?.id ?? ""); const [label, setLabel] = useState(profile?.label ?? ""); const [upi, setUpi] = useState(profile?.upi_id ?? ""); const [payee, setPayee] = useState(profile?.payee_name ?? ""); const [parser, setParser] = useState(profile?.parser ?? "paytm_notification"); const [enabled, setEnabled] = useState(profile?.enabled ?? true); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function save() { setBusy(true); setError(""); try { await saveProfile({ id: id.trim(), label: label.trim(), upi_id: upi.trim(), payee_name: payee.trim(), parser, enabled }); await onSaved(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not save profile."); } finally { setBusy(false); } }
  return <Modal title={profile ? `Edit ${profile.label}` : "Add collection profile"} onClose={onClose}><div className="form-stack">
    <label><span>Profile ID</span><input value={id} disabled={Boolean(profile)} onChange={(e) => setId(e.target.value.toLowerCase().replace(/\s/g, ""))} placeholder="paytm"/></label>
    <label><span>Label</span><input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="Paytm"/></label>
    <label><span>Destination UPI ID</span><input value={upi} onChange={(e) => setUpi(e.target.value)} placeholder="merchant@upi"/></label>
    <label><span>Payee name</span><input value={payee} onChange={(e) => setPayee(e.target.value)} placeholder="PayGate"/></label>
    <label><span>Incoming signal</span><select value={parser} onChange={(e) => setParser(e.target.value)}><option value="paytm_notification">Paytm Business notification</option><option value="kotak_sms">Kotak SMS via Google Messages</option></select></label>
    <label className="toggle-line"><input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)}/><span>Allow new payments on this profile</span></label>
    {error && <ErrorNotice message={error}/>}<div className="form-actions"><button className="button button-secondary" onClick={onClose}>Cancel</button><button className="button button-primary" disabled={busy || !id || !label || !upi} onClick={() => void save()}>{busy ? <><Spinner/> Saving…</> : "Save profile"}</button></div>
  </div></Modal>;
}

function WebhookSettingsPanel() {
  const [settings, setSettings] = useState<WebhookSettings>(); const [endpoint, setEndpoint] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const [secret, setSecret] = useState<string>();
  useEffect(() => { void getWebhookSettings().then((v) => { setSettings(v); setEndpoint(v.endpoint || ""); }).catch((e) => setError(e instanceof ApiError ? e.message : "Could not load webhook settings.")); }, []);
  async function save(rotate: boolean) { setBusy(true); setError(""); try { const result = await saveWebhook(endpoint.trim(), rotate); setSettings(result.webhook); if (result.signing_secret) setSecret(result.signing_secret); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not update webhook."); } finally { setBusy(false); } }
  return <section className="panel settings-panel"><div className="panel-head"><div><p className="eyebrow">Merchant notifications</p><h3>Webhook</h3><p>Durable signed delivery. The signing secret is shown only when first generated or rotated.</p></div>{settings && <Badge tone={settings.enabled ? "good" : "neutral"}>{settings.enabled ? "Enabled" : "Disabled"}</Badge>}</div>
    <div className="settings-form-row"><label><span>HTTPS endpoint</span><input value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder="https://example.com/paygate/webhook"/></label><button className="button button-primary" disabled={busy} onClick={() => void save(false)}>{busy ? <Spinner/> : "Save"}</button></div>
    <div className="inline-actions"><button className="text-button danger-text" disabled={busy || !settings?.secret_configured} onClick={() => { if (window.confirm("Rotate the signing secret? The merchant must be updated immediately.")) void save(true); }}>Rotate signing secret</button><button className="text-button" disabled={busy || (!endpoint && !settings?.enabled)} onClick={() => { if (window.confirm("Disable webhook delivery?")) { setEndpoint(""); void saveWebhook("", false).then((r) => setSettings(r.webhook)).catch((e) => setError(e instanceof ApiError ? e.message : "Could not disable webhook.")); } }}>Disable webhook</button></div>
    {error && <ErrorNotice message={error}/>} {secret && <SecretModal title="Webhook signing secret" secret={secret} onClose={() => setSecret(undefined)} warning="Store this in the merchant backend now. PayGate will not show it again."/>}
  </section>;
}

function ApiKeysPanel() {
  const [items, setItems] = useState<ApiKeyInfo[]>([]); const [label, setLabel] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const [secret, setSecret] = useState<string>();
  const load = useCallback(async () => { try { setItems(await getApiKeys()); setError(""); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load API keys."); } }, []);
  useEffect(() => { void load(); }, [load]);
  async function create() { if (!label.trim()) return; setBusy(true); setError(""); try { const key = await createApiKey(label.trim()); setLabel(""); setSecret(key.secret); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not create API key."); } finally { setBusy(false); } }
  return <section className="panel settings-panel"><div className="panel-head"><div><p className="eyebrow">Merchant access</p><h3>API keys</h3><p>Server-to-server credentials for creating and reading payments. Secrets are shown once.</p></div></div>
    <div className="settings-form-row"><label><span>New key label</span><input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="IEEE events frontend"/></label><button className="button button-primary" disabled={busy || !label.trim()} onClick={() => void create()}>{busy ? <Spinner/> : "Create key"}</button></div>
    {error && <ErrorNotice message={error}/>}<div className="key-list">{items.map((key) => <div className="key-row" key={key.id}><div><strong>{key.label}</strong><span>{key.id} · created {dateTime(key.created_at)}{key.last_used_at ? ` · used ${relativeTime(key.last_used_at)}` : " · never used"}</span></div><button className="text-button danger-text" onClick={() => { if (window.confirm(`Revoke ${key.label}? Existing integrations using it will stop working.`)) void revokeApiKey(key.id).then(load).catch((e) => setError(e instanceof ApiError ? e.message : "Could not revoke API key.")); }}>Revoke</button></div>)}</div>
    {secret && <SecretModal title="New merchant API key" secret={secret} onClose={() => setSecret(undefined)} warning="Copy this key into the merchant server now. It cannot be recovered later."/>}
  </section>;
}

function DevicePanel() {
  const [device, setDevice] = useState<DeviceInfo | null>(); const [error, setError] = useState(""); const [pairing, setPairing] = useState<PairingSession>(); const [busy, setBusy] = useState(false);
  const load = useCallback(async () => { try { setDevice(await getDevice()); setError(""); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not load phone status."); } }, []);
  useEffect(() => { void load(); const t = setInterval(() => { if (document.visibilityState === "visible") void load(); }, 30_000); return () => clearInterval(t); }, [load]);
  async function pair() { setBusy(true); setError(""); try { setPairing(await createPairingSession(Boolean(device))); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not create pairing link."); } finally { setBusy(false); } }
  return <section className="panel settings-panel"><div className="panel-head"><div><p className="eyebrow">Notification relay</p><h3>PayGate phone</h3><p>One trusted Android device captures allowlisted decimal-money notifications and signs them to PayGate.</p></div>{device && <Badge tone={isDeviceHealthy(device) ? "good" : "warn"}>{isDeviceHealthy(device) ? "Healthy" : "Needs attention"}</Badge>}</div>
    {error && <ErrorNotice message={error}/>} {device ? <><div className="device-card"><div className="device-title"><div className="phone-glyph">▯</div><div><strong>{device.name || device.device_model || "PayGate phone"}</strong><span>{device.device_model || "Android"} · Android {device.android_version || "—"} · PayGate {device.app_version || "—"}</span></div></div><div className="device-heartbeat"><Dot ok={Boolean(device.last_heartbeat_at)}/><span>Heartbeat {relativeTime(device.last_heartbeat_at)}</span></div></div>
      <div className="prereq-grid"><Prereq label="Notification access" value={device.notification_access}/><Prereq label="Listener connected" value={device.listener_connected}/><Prereq label="Battery unrestricted" value={device.battery_optimization_exempt}/><Prereq label="Foreground service" value={device.foreground_service}/><Prereq label="Power saver" value={device.power_save_mode === undefined ? undefined : !device.power_save_mode}/><Prereq label="Background allowed" value={device.background_restricted === undefined ? undefined : !device.background_restricted}/></div>
      <div className="queue-strip"><span>Local pending <strong>{device.pending_count ?? "—"}</strong></span><span>Local failed <strong>{device.failed_count ?? "—"}</strong></span><span>Last upload <strong>{relativeTime(device.last_successful_delivery_at)}</strong></span></div>
      {device.last_client_error && <div className="notice notice-error">Phone reports: {device.last_client_error}</div>}
    </> : <div className="empty-inline">No active PayGate phone is connected.</div>}
    <div className="inline-actions"><button className="button button-primary" disabled={busy} onClick={() => void pair()}>{busy ? <Spinner/> : device ? "Connect replacement phone" : "Connect phone"}</button>{device && <button className="text-button danger-text" onClick={() => { if (window.confirm("Revoke this phone now? Payment notifications will stop until another phone is paired.")) void revokeDevice(device.id).then(load).catch((e) => setError(e instanceof ApiError ? e.message : "Could not revoke phone.")); }}>Revoke phone</button>}</div>
    {pairing && <PairingModal pairing={pairing} onClose={() => { setPairing(undefined); void load(); }}/>}
  </section>;
}

function PairingModal({ pairing, onClose }: { pairing: PairingSession; onClose: () => void }) {
  const url = pairing.pairing_url || `${window.location.origin}/device/pair/${pairing.token}`; const [copied, setCopied] = useState(false);
  return <Modal title={pairing.replace_existing ? "Connect replacement phone" : "Connect PayGate phone"} onClose={onClose}><div className="pairing-modal"><div className="qr-card"><QRCodeCanvas value={url} size={232} level="M" bgColor="#ffffff" fgColor="#07111f" marginSize={2}/></div><p>Scan this QR with the Android phone. The verified PayGate App Link should open the existing app directly.</p><div className="pairing-expiry"><span>Expires</span><strong>{dateTime(pairing.expires_at)}</strong></div><button className="button button-secondary" onClick={() => void copyText(url).then((ok) => { if (ok) { setCopied(true); setTimeout(() => setCopied(false), 1300); } })}>{copied ? "Copied" : "Copy pairing link"}</button><div className="immutable-note"><strong>Short-lived and one-use.</strong><span>Do not send this link to another person. The old phone stays active until replacement pairing succeeds.</span></div></div></Modal>;
}

function PasswordPanel({ onSignedOut }: { onSignedOut: () => void }) {
  const [current, setCurrent] = useState(""); const [next, setNext] = useState(""); const [confirm, setConfirm] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function change() { if (next !== confirm) { setError("New passwords do not match."); return; } setBusy(true); setError(""); try { await changePassword(current, next); onSignedOut(); } catch (e) { setError(e instanceof ApiError ? e.message : "Could not change password."); } finally { setBusy(false); } }
  return <section className="panel settings-panel"><div className="panel-head"><div><p className="eyebrow">Operator access</p><h3>Admin password</h3><p>PayGate has one operator password. Changing it signs out every active web and Android dashboard session; the phone relay itself keeps running.</p></div></div>
    <div className="password-grid"><label><span>Current password</span><input type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)}/></label><label><span>New password</span><input type="password" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)}/></label><label><span>Confirm new password</span><input type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)}/></label></div>
    {error && <ErrorNotice message={error}/>}<div className="inline-actions"><button className="button button-primary" disabled={busy || !current || !next || !confirm} onClick={() => void change()}>{busy ? <Spinner/> : "Change password"}</button></div>
  </section>;
}

function SecretModal({ title, secret, warning, onClose }: { title: string; secret: string; warning: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false); return <Modal title={title} onClose={onClose}><div className="secret-modal"><p>{warning}</p><code>{secret}</code><button className="button button-primary" onClick={() => void copyText(secret).then((ok) => { if (ok) setCopied(true); })}>{copied ? "Copied" : "Copy secret"}</button><button className="button button-secondary" onClick={onClose}>I have stored it</button></div></Modal>;
}
function Prereq({ label, value }: { label: string; value?: boolean }) { return <div className="prereq"><Dot ok={value === true}/><span>{label}</span><strong>{value === undefined ? "Unknown" : value ? "OK" : "Check"}</strong></div>; }
function isDeviceHealthy(device: DeviceInfo): boolean { return device.notification_access === true && device.listener_connected === true && device.battery_optimization_exempt === true && device.foreground_service === true && device.background_restricted !== true; }
function parserLabel(value: string): string { return value === "paytm_notification" ? "Paytm Business notification" : value === "kotak_sms" ? "Kotak SMS" : "Historical"; }
