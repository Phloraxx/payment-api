import { useState, type FormEvent } from "react";
import { login } from "../api";

export function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await login(email, password);
    } catch {
      setError("Login failed. Check the operator credentials.");
    } finally {
      setBusy(false);
    }
  }

  return <div className="login">
    <form className="login-card" onSubmit={submit}>
      <div className="brand">PAY<span>GATE</span></div>
      <h1>Operator sign in</h1>
      <p className="muted">Payment verification, SMS evidence and connector health.</p>
      <label>Email<input type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus /></label>
      <label>Password<input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required /></label>
      {error && <p className="error">{error}</p>}
      <button className="primary" type="submit" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</button>
    </form>
  </div>;
}
