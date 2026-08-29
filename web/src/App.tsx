import { useEffect, useMemo, useState } from "react";
import { auth, refreshAuth as refreshOperatorAuth } from "./api";
import type { Page } from "./types";
import { Dashboard } from "./pages/Dashboard";
import { Health } from "./pages/Health";
import { Login } from "./pages/Login";
import { Payments } from "./pages/Payments";
import { AuditEvents, EmailEvents, SMSEvents, WebhookDeliveries } from "./pages/Records";
import { AlertsPage, ReconciliationPage, RefundsPage, ReviewsPage } from "./pages/Operations";
import { Settings } from "./pages/Settings";
import { RazorpayTestPage } from "./pages/RazorpayTest";

const primaryPages: Page[] = ["dashboard", "payments", "reviews", "health"];
const advancedPages: Page[] = ["reconciliation", "refunds", "sms", "email", "razorpay_test", "alerts", "webhooks", "audit", "settings"];
const pages = [...primaryPages, ...advancedPages];

const pageMeta: Record<Page, { label: string; eyebrow: string; description: string }> = {
  dashboard: { label: "Overview", eyebrow: "Today", description: "What needs your attention right now." },
  payments: { label: "Payments", eyebrow: "Money flow", description: "Find, create and manage every payment." },
  reviews: { label: "Action", eyebrow: "Needs a person", description: "Only the cases PayGate cannot decide safely." },
  health: { label: "Health", eyebrow: "System status", description: "The few things that must stay healthy for PayGate to work." },
  reconciliation: { label: "Reconciliation", eyebrow: "Advanced", description: "Compare bank statements without changing payment truth automatically." },
  refunds: { label: "Refunds", eyebrow: "Advanced", description: "Record and audit manual refund workflows." },
  sms: { label: "SMS evidence", eyebrow: "Advanced", description: "Raw operational SMS records." },
  email: { label: "Email evidence", eyebrow: "Advanced", description: "Raw operational bank-email records." },
  razorpay_test: { label: "Razorpay test", eyebrow: "Advanced", description: "Sandbox payment diagnostics." },
  alerts: { label: "Alerts", eyebrow: "Advanced", description: "Operational alert history." },
  webhooks: { label: "Webhooks", eyebrow: "Advanced", description: "Delivery-level diagnostics." },
  audit: { label: "Audit trail", eyebrow: "Advanced", description: "Immutable operator and system actions." },
  settings: { label: "Settings", eyebrow: "Advanced", description: "Low-frequency infrastructure controls." },
};

function pageFromHash(): Page {
  const value = window.location.hash.replace(/^#\/?/, "").split("?")[0] as Page;
  return pages.includes(value) ? value : "dashboard";
}

export function App() {
  const [loggedIn, setLoggedIn] = useState(auth.isValid);
  const [page, setPage] = useState<Page>(pageFromHash());
  const [notice, setNotice] = useState("");

  useEffect(() => auth.subscribe(() => setLoggedIn(auth.isValid)), []);
  useEffect(() => {
    if (!auth.token) return;
    const refreshSession = async () => {
      try { await refreshOperatorAuth(); } catch { auth.clear(); }
    };
    void refreshSession();
    const timer = window.setInterval(() => void refreshSession(), 10 * 60_000);
    return () => window.clearInterval(timer);
  }, [loggedIn]);
  useEffect(() => {
    const handler = () => setPage(pageFromHash());
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  }, []);
  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(() => setNotice(""), 4500);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const meta = useMemo(() => pageMeta[page], [page]);
  if (!loggedIn) return <Login />;

  function navigate(next: Page) {
    window.location.hash = `/${next}`;
    setPage(next);
  }

  return <div className="app-shell">
    <aside className="app-sidebar">
      <button className="brand-lockup" onClick={() => navigate("dashboard")} aria-label="PayGate overview">
        <span className="brand-mark">PG</span><span><strong>PayGate</strong><small>Operator</small></span>
      </button>
      <nav className="primary-nav" aria-label="Primary navigation">
        {primaryPages.map((item, index) => <button className={page === item ? "nav-item active" : "nav-item"} key={item} onClick={() => navigate(item)}>
          <span className="nav-index">0{index + 1}</span><span>{pageMeta[item].label}</span>{item === "reviews" && <span className="nav-signal" />}
        </button>)}
      </nav>
      <details className="advanced-nav" open={advancedPages.includes(page)}>
        <summary>Advanced</summary>
        <div>{advancedPages.map((item) => <button className={page === item ? "advanced-item active" : "advanced-item"} key={item} onClick={() => navigate(item)}>{pageMeta[item].label}</button>)}</div>
      </details>
      <div className="sidebar-account">
        <span className="account-avatar">{(auth.email || "O").slice(0, 1).toUpperCase()}</span>
        <span className="account-copy"><strong>{auth.email || "Operator"}</strong><small>Administrator</small></span>
        <button className="quiet-button" onClick={() => auth.clear()}>Sign out</button>
      </div>
    </aside>

    <main className="app-main">
      <header className="page-header">
        <div><p className="page-eyebrow">{meta.eyebrow}</p><h1>{meta.label}</h1><p>{meta.description}</p></div>
        <div className="live-chip"><span /> Live</div>
      </header>
      {notice && <button className="notice-toast" role="status" onClick={() => setNotice("")}>{notice}</button>}
      <div className="page-content">
        {page === "dashboard" && <Dashboard />}
        {page === "payments" && <Payments notify={setNotice} />}
        {page === "reviews" && <ReviewsPage notify={setNotice} />}
        {page === "health" && <Health />}
        {page === "reconciliation" && <ReconciliationPage notify={setNotice} />}
        {page === "refunds" && <RefundsPage notify={setNotice} />}
        {page === "sms" && <SMSEvents />}
        {page === "email" && <EmailEvents />}
        {page === "razorpay_test" && <RazorpayTestPage notify={setNotice} />}
        {page === "alerts" && <AlertsPage />}
        {page === "webhooks" && <WebhookDeliveries />}
        {page === "audit" && <AuditEvents />}
        {page === "settings" && <Settings notify={setNotice} />}
      </div>
    </main>
  </div>;
}
