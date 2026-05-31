import { useState } from "react";
import { KeyRound, Loader2 } from "lucide-react";
import { loginPasskey } from "../api/auth";

export function Login({ onDone }: { onDone: () => void }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async () => {
    setLoading(true);
    setError(null);
    try {
      await loginPasskey();
      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Passkey login failed. Try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="auth-page">
      <section className="panel auth-card">
        <h1>Admin Login</h1>
        <p>Use the registered passkey to open the payment console.</p>
        {error && <p className="error-message">{error}</p>}
        <button onClick={handleLogin} disabled={loading}>
          {loading ? <Loader2 size={18} className="spin" /> : <KeyRound size={18} />}
          {loading ? "Verifying passkey…" : "Continue with passkey"}
        </button>
      </section>
    </main>
  );
}
