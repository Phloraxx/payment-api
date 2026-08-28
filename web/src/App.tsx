import { useEffect, useMemo, useState } from "react";
import { auth, refreshAuth as refreshOperatorAuth } from "./api";
import type { Page } from "./types";
import { Dashboard } from "./pages/Dashboard";
import { Login } from "./pages/Login";
import { Payments } from "./pages/Payments";
import { AuditEvents, EmailEvents, SMSEvents, WebhookDeliveries } from "./pages/Records";
import { AlertsPage, ReconciliationPage, RefundsPage, ReviewsPage } from "./pages/Operations";
import { Settings } from "./pages/Settings";
import { RazorpayTestPage } from "./pages/RazorpayTest";

const navGroups: Array<{ label: string; pages: Page[] }> = [
  { label: "Operate", pages: ["dashboard", "payments", "reviews", "refunds", "reconciliation"] },
  { label: "Evidence", pages: ["sms", "email", "razorpay_test"] },
  { label: "System", pages: ["alerts", "webhooks", "audit", "settings"] },
];
const pages: Page[] = navGroups.flatMap((group) => group.pages);

function pageFromHash(): Page {
  const value = window.location.hash.replace(/^#\/?/, "") as Page;
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
    const timer = window.setTimeout(() => setNotice(""), 5000);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const title = useMemo(() => label(page), [page]);
  if (!loggedIn) return <Login />;

  function navigate(next: Page) {
    window.location.hash = `/${next}`;
    setPage(next);
  }

  return <div className="shell">
    <aside>
      <div className="brand">PAY<span>GATE</span></div>
      <p className="muted">Operator console</p>
      <nav>{navGroups.map((group) => <div key={group.label}><p className="nav-group-label">{group.label}</p>{group.pages.map((item) => <button className={page === item ? "nav active" : "nav"} key={item} onClick={() => navigate(item)}>{label(item)}</button>)}</div>)}</nav>
      <div className="sidebar-bottom">
        <span className="operator">{auth.email}</span>
        <button className="signout" onClick={() => auth.clear()}>Sign out</button>
      </div>
    </aside>
    <main>
      <header><div><p className="eyebrow">PAYMENT OPERATIONS</p><h1>{title}</h1></div></header>
      {notice && <div className="notice" role="status" onClick={() => setNotice("")}>{notice}</div>}
      {page === "dashboard" && <Dashboard />}
      {page === "payments" && <Payments notify={setNotice} />}
      {page === "reviews" && <ReviewsPage notify={setNotice} />}
      {page === "reconciliation" && <ReconciliationPage notify={setNotice} />}
      {page === "sms" && <SMSEvents />}
      {page === "email" && <EmailEvents />}
      {page === "alerts" && <AlertsPage />}
      {page === "refunds" && <RefundsPage notify={setNotice} />}
      {page === "webhooks" && <WebhookDeliveries />}
      {page === "audit" && <AuditEvents />}
      {page === "razorpay_test" && <RazorpayTestPage notify={setNotice} />}
      {page === "settings" && <Settings notify={setNotice} />}
    </main>
  </div>;
}

function label(value: string) {
  if (value === "sms") return "SMS Events";
  if (value === "email") return "Email Events";
  if (value === "audit") return "Audit Trail";
  if (value === "razorpay_test") return "Razorpay Test";
  return value.charAt(0).toUpperCase() + value.slice(1);
}
