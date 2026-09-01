import { useEffect, useState } from "react";
import { getOverview, logout } from "./api";
import { Login } from "./Login";
import { OverviewPage } from "./OverviewPage";
import { PaymentsPage } from "./PaymentsPage";
import { ActivityPage } from "./ActivityPage";
import { SettingsPage } from "./SettingsPage";
import { Spinner, cx } from "./ui";

type Tab = "overview" | "payments" | "activity" | "settings";
const tabs: Array<{ id: Tab; label: string; glyph: string }> = [
  { id: "overview", label: "Overview", glyph: "◫" },
  { id: "payments", label: "Payments", glyph: "₹" },
  { id: "activity", label: "Activity", glyph: "↗" },
  { id: "settings", label: "Settings", glyph: "⚙" },
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
  if (session === "checking") return <div className="boot"><div className="brand-mark">P</div><Spinner/></div>;
  if (session === "out") return <Login onSuccess={() => setSession("in")} />;

  const openPayment = (id: string) => { setOpenPaymentId(id); setTab("payments"); };
  return <div className="app-shell">
    <aside className="sidebar">
      <div className="brand-lockup"><div className="brand-mark">P</div><div><strong>PayGate</strong><span>Control plane</span></div></div>
      <nav>{tabs.map((item) => <button key={item.id} className={cx(tab === item.id && "active")} onClick={() => setTab(item.id)}><span>{item.glyph}</span>{item.label}</button>)}</nav>
      <div className="sidebar-foot"><span className="live-pill"><i/> v4</span><small>Direct SQLite runtime</small></div>
    </aside>
    <main className="content">
      <header className="topbar"><div><span className="mobile-brand">PayGate</span></div><div className="topbar-actions"><span className="secure-chip">Secure admin session</span><button className="text-button" onClick={() => void logout().finally(() => setSession("out"))}>Sign out</button></div></header>
      <div className="page-wrap">
        {tab === "overview" && <OverviewPage onOpenPayment={openPayment} />}
        {tab === "payments" && <PaymentsPage initialPaymentId={openPaymentId} onInitialConsumed={() => setOpenPaymentId(undefined)} />}
        {tab === "activity" && <ActivityPage onOpenPayment={openPayment} />}
        {tab === "settings" && <SettingsPage onSignedOut={() => setSession("out")} />}
      </div>
    </main>
    <nav className="bottom-nav">{tabs.map((item) => <button key={item.id} className={cx(tab === item.id && "active")} onClick={() => setTab(item.id)}><span>{item.glyph}</span><small>{item.label}</small></button>)}</nav>
  </div>;
}

function PairingLanding() {
  const link = window.location.href;
  const [copied, setCopied] = useState(false);
  return <main className="login-page pairing-landing"><section className="login-card">
    <div className="brand-lockup"><div className="brand-mark">P</div><div><strong>PayGate</strong><span>Phone pairing</span></div></div>
    <div className="login-copy"><p className="eyebrow">Secure connection link</p><h1>Open this link on the PayGate phone.</h1><p>If PayGate is installed and Android has verified this domain, the app opens automatically. The pairing token is short-lived and one-use.</p></div>
    <button className="button button-primary" onClick={() => { window.location.href = link; }}>Open in PayGate</button>
    <button className="button button-secondary" onClick={() => void navigator.clipboard.writeText(link).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1300); })}>{copied ? "Copied" : "Copy pairing link"}</button>
    <p className="pairing-warning">Do not send this link to another person or device.</p>
  </section></main>;
}
