import { useEffect, useState } from "react";
import { getOverview, logout } from "./api";
import { Brand, PayGateMark } from "./Brand";
import { Login } from "./Login";
import { OverviewPage } from "./OverviewPage";
import { PaymentsPage } from "./PaymentsPage";
import { ActivityPage } from "./ActivityPage";
import { SettingsPage } from "./SettingsPage";
import { Spinner, cx } from "./ui";

type Tab = "overview" | "payments" | "activity" | "settings";
const tabs: Array<{ id: Tab; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "payments", label: "Payments" },
  { id: "activity", label: "Activity" },
  { id: "settings", label: "Settings" },
];

export function App() {
  const pairingPath = window.location.pathname.startsWith("/device/pair/");
  const [session, setSession] = useState<"checking" | "in" | "out">(pairingPath ? "out" : "checking");
  const [tab, setTab] = useState<Tab>("overview");
  const [openPaymentId, setOpenPaymentId] = useState<string>();

  useEffect(() => {
    if (pairingPath) return;
    let alive = true;
    getOverview().then(() => { if (alive) setSession("in"); }).catch(() => { if (alive) setSession("out"); });
    const unauthorized = () => setSession("out");
    window.addEventListener("paygate:unauthorized", unauthorized);
    return () => { alive = false; window.removeEventListener("paygate:unauthorized", unauthorized); };
  }, [pairingPath]);

  if (pairingPath) return <PairingLanding />;
  if (session === "checking") return <div className="boot"><PayGateMark className="boot-mark"/><Spinner/></div>;
  if (session === "out") return <Login onSuccess={() => setSession("in")} />;

  const openPayment = (id: string) => { setOpenPaymentId(id); setTab("payments"); };
  return <div className="app-shell">
    <aside className="sidebar">
      <Brand subtitle="Control plane" />
      <nav aria-label="PayGate navigation">{tabs.map((item) => <button key={item.id} aria-current={tab === item.id ? "page" : undefined} className={cx(tab === item.id && "active")} onClick={() => setTab(item.id)}>
        <NavIcon tab={item.id}/><span>{item.label}</span>
      </button>)}</nav>
      <div className="sidebar-foot">
        <span className="live-pill"><i/> System online</span>
        <small>PayGate v4 · private runtime</small>
      </div>
    </aside>
    <main className="content">
      <header className="topbar">
        <div className="mobile-brand"><Brand /></div>
        <div className="topbar-actions">
          <span className="live-pill topbar-live"><i/> Live</span>
          <span className="secure-chip">Secure admin session</span>
          <button className="text-button" onClick={() => void logout().finally(() => setSession("out"))}>Sign out</button>
        </div>
      </header>
      <div className="page-wrap">
        {tab === "overview" && <OverviewPage onOpenPayment={openPayment} />}
        {tab === "payments" && <PaymentsPage initialPaymentId={openPaymentId} onInitialConsumed={() => setOpenPaymentId(undefined)} />}
        {tab === "activity" && <ActivityPage onOpenPayment={openPayment} />}
        {tab === "settings" && <SettingsPage onSignedOut={() => setSession("out")} />}
      </div>
    </main>
    <nav className="bottom-nav" aria-label="PayGate mobile navigation">{tabs.map((item) => <button key={item.id} aria-current={tab === item.id ? "page" : undefined} className={cx(tab === item.id && "active")} onClick={() => setTab(item.id)}>
      <NavIcon tab={item.id}/><small>{item.label}</small>
    </button>)}</nav>
  </div>;
}

function NavIcon({ tab }: { tab: Tab }) {
  if (tab === "overview") return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 11.2 12 4l8 7.2v8.3a.5.5 0 0 1-.5.5H15v-6H9v6H4.5a.5.5 0 0 1-.5-.5Z"/></svg>;
  if (tab === "payments") return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="3"/><path d="M3 9h18M7 14h4"/></svg>;
  if (tab === "activity") return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 16h3l2-7 4 10 3-8 2 5h2"/></svg>;
  return <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19 13.5v-3l-2-.6a7 7 0 0 0-.7-1.6l1-1.8-2.1-2.1-1.8 1A7 7 0 0 0 11.5 4L11 2H8l-.6 2a7 7 0 0 0-1.6.7l-1.8-1L2 5.8l1 1.8A7 7 0 0 0 2.4 10l-2 .6v3l2 .6a7 7 0 0 0 .7 1.6l-1 1.8 2.1 2.1 1.8-1a7 7 0 0 0 1.6.7l.6 2h3l.6-2a7 7 0 0 0 1.6-.7l1.8 1 2.1-2.1-1-1.8a7 7 0 0 0 .7-1.6Z" transform="translate(2 -1) scale(.83)"/></svg>;
}

function PairingLanding() {
  const link = window.location.href;
  const [copied, setCopied] = useState(false);
  return <main className="login-page pairing-landing"><section className="login-card">
    <Brand subtitle="Phone pairing" />
    <div className="login-copy"><p className="eyebrow">Secure connection link</p><h1>Open this link on the PayGate phone.</h1><p>If PayGate is installed and Android has verified this domain, the app opens automatically. The pairing token is short-lived and one-use.</p></div>
    <button className="button button-primary" onClick={() => { window.location.href = link; }}>Open in PayGate</button>
    <button className="button button-secondary" onClick={() => void navigator.clipboard.writeText(link).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1300); })}>{copied ? "Copied" : "Copy pairing link"}</button>
    <p className="pairing-warning">Do not send this link to another person or device.</p>
  </section></main>;
}
