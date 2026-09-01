import { useState } from "react";
import { ApiError, login } from "./api";
import { ErrorNotice, Spinner } from "./ui";

export function Login({ onSuccess }: { onSuccess: () => void }) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!password) return;
    setBusy(true); setError("");
    try { await login(password); setPassword(""); onSuccess(); }
    catch (e) { setError(e instanceof ApiError ? e.message : "Could not sign in."); }
    finally { setBusy(false); }
  }
  return <main className="login-page">
    <section className="login-card">
      <div className="brand-lockup"><div className="brand-mark">P</div><div><strong>PayGate</strong><span>Payment infrastructure</span></div></div>
      <div className="login-copy"><p className="eyebrow">Operator access</p><h1>One gateway.<br/>One control plane.</h1><p>Sign in with the PayGate admin password. The phone relay runs independently of this dashboard session.</p></div>
      <form onSubmit={submit} className="login-form">
        <label><span>Admin password</span><input autoFocus type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Enter password" /></label>
        {error && <ErrorNotice message={error} />}
        <button className="button button-primary" disabled={busy || !password}>{busy ? <><Spinner/> Signing in…</> : "Sign in to PayGate"}</button>
      </form>
      <footer>Direct SQLite · signed relay · durable webhooks</footer>
    </section>
  </main>;
}
