import { useState } from "react";
import { ShieldCheck, Loader2 } from "lucide-react";
import { registerPasskey, verifyCode } from "../api/auth";

export function Setup({ onDone }: { onDone: () => void }) {
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState<string | null>(null);
  const [codeLoading, setCodeLoading] = useState(false);
  const [verified, setVerified] = useState(false);
  const [registerLoading, setRegisterLoading] = useState(false);
  const [registerError, setRegisterError] = useState<string | null>(null);

  const handleVerify = async () => {
    setCodeLoading(true);
    setCodeError(null);
    try {
      await verifyCode(code);
      setVerified(true);
    } catch (err) {
      setCodeError(err instanceof Error ? err.message : "Invalid code.");
    } finally {
      setCodeLoading(false);
    }
  };

  const handleRegister = async () => {
    setRegisterLoading(true);
    setRegisterError(null);
    try {
      await registerPasskey();
      onDone();
    } catch (err) {
      setRegisterError(err instanceof Error ? err.message : "Registration failed. Try again.");
    } finally {
      setRegisterLoading(false);
    }
  };

  return (
    <main className="auth-page">
      <section className="panel auth-card">
        <h1>First-Time Setup</h1>
        <p>Enter the one-time setup code, then register this device passkey.</p>
        {codeError && <p className="error-message">{codeError}</p>}
        <input value={code} onChange={(event) => setCode(event.target.value)} placeholder="SETUP-1234-ABCD" />
        <button onClick={handleVerify} disabled={codeLoading || verified}>
          {codeLoading ? <Loader2 size={18} className="spin" /> : null}
          {codeLoading ? "Verifying…" : verified ? "Code verified" : "Verify code"}
        </button>
        {registerError && <p className="error-message">{registerError}</p>}
        <button disabled={!verified || registerLoading} onClick={handleRegister}>
          {registerLoading ? <Loader2 size={18} className="spin" /> : <ShieldCheck size={18} />}
          {registerLoading ? "Registering…" : "Register passkey"}
        </button>
      </section>
    </main>
  );
}
