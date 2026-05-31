import { useEffect, useState } from "react";
import { logout, session, setupStatus } from "./api/auth";
import { Sidebar, type PageId } from "./components/Sidebar";
import { DecimalPool } from "./pages/DecimalPool";
import { Login } from "./pages/Login";
import { Logs } from "./pages/Logs";
import { Overview } from "./pages/Overview";
import { Settings } from "./pages/Settings";
import { Setup } from "./pages/Setup";
import { TestHarness } from "./pages/TestHarness";
import { Tickets } from "./pages/Tickets";

type AuthState = "loading" | "setup" | "login" | "ready";

export function App() {
  const [auth, setAuth] = useState<AuthState>("loading");
  const [page, setPage] = useState<PageId>("overview");

  useEffect(() => {
    void setupStatus().then((status) => {
      if (status.needs_setup) {
        setAuth("setup");
        return;
      }
      void session()
        .then(() => setAuth("ready"))
        .catch(() => setAuth("login"));
    });
  }, []);

  if (auth === "loading") return <main className="auth-page">Loading...</main>;
  if (auth === "setup") return <Setup onDone={() => setAuth("ready")} />;
  if (auth === "login") return <Login onDone={() => setAuth("ready")} />;

  return (
    <div className="shell">
      <Sidebar
        page={page}
        onPage={setPage}
        onLogout={() => {
          void logout().finally(() => setAuth("login"));
        }}
      />
      <main className="content">
        <header>
          <h1>{title(page)}</h1>
        </header>
        {page === "overview" && <Overview />}
        {page === "tickets" && <Tickets />}
        {page === "pool" && <DecimalPool />}
        {page === "test" && <TestHarness />}
        {page === "logs" && <Logs />}
        {page === "settings" && <Settings />}
      </main>
    </div>
  );
}

function title(page: PageId): string {
  return {
    overview: "Overview",
    tickets: "Tickets",
    pool: "Decimal Pool",
    test: "Test Harness",
    logs: "Logs",
    settings: "Settings",
  }[page];
}
