import { useEffect, useMemo, useState } from "react";
import { pb } from "./pb";
import type { Page } from "./types";
import { Dashboard } from "./pages/Dashboard";
import { Login } from "./pages/Login";
import { Payments } from "./pages/Payments";
import { AuditEvents, SMSEvents, WebhookDeliveries } from "./pages/Records";
import { AlertsPage, ReconciliationPage, RefundsPage, ReviewsPage } from "./pages/Operations";
import { Settings } from "./pages/Settings";

const pages: Page[] = ["dashboard", "payments", "reviews", "reconciliation", "sms", "alerts", "refunds", "webhooks", "audit", "settings"];

function pageFromHash(): Page {
  const value = window.location.hash.replace(/^#\/?/, "") as Page;
  return pages.includes(value) ? value : "dashboard";
}

export function App() {
  const [loggedIn, setLoggedIn] = useState(pb.authStore.isValid);
  const [page, setPage] = useState<Page>(pageFromHash());
  const [notice, setNotice] = useState("");

  useEffect(() => pb.authStore.onChange(() => setLoggedIn(pb.authStore.isValid)), []);
  useEffect(() => {
    if (!pb.authStore.token) return;
    const refreshAuth = async () => {
      try { await pb.collection("users").authRefresh(); } catch { pb.authStore.clear(); }
    };
    void refreshAuth();
    const timer = window.setInterval(() => void refreshAuth(), 10 * 60_000);
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
      <nav>{pages.map((item) => <button className={page === item ? "nav active" : "nav"} key={item} onClick={() => navigate(item)}>{label(item)}</button>)}</nav>
      <div className="sidebar-bottom">
        <span className="operator">{pb.authStore.record?.email}</span>
        <button className="signout" onClick={() => pb.authStore.clear()}>Sign out</button>
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
      {page === "alerts" && <AlertsPage />}
      {page === "refunds" && <RefundsPage notify={setNotice} />}
      {page === "webhooks" && <WebhookDeliveries />}
      {page === "audit" && <AuditEvents />}
      {page === "settings" && <Settings notify={setNotice} />}
    </main>
  </div>;
}

function label(value: string) {
  if (value === "sms") return "SMS Events";
  if (value === "audit") return "Audit Trail";
  return value.charAt(0).toUpperCase() + value.slice(1);
}
