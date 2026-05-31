import { useState } from "react";
import { createTestTicket, simulateWebhook } from "../api/tickets";
import type { Ticket } from "../types";

export function TestHarness() {
  const [amount, setAmount] = useState("100");
  const [sms, setSms] = useState("");
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [result, setResult] = useState("");
  return (
    <div className="page two-col">
      <section className="panel form-panel">
        <h2>Create Test Ticket</h2>
        <input value={amount} onChange={(event) => setAmount(event.target.value)} />
        <button onClick={() => void createTestTicket(amount).then(setTicket)}>Create</button>
        {ticket && (
          <div className="result">
            <strong>{ticket.ticketId}</strong>
            <span>Pay ₹{ticket.amount.toFixed(2)}</span>
          </div>
        )}
      </section>
      <section className="panel form-panel">
        <h2>Simulate SMS Webhook</h2>
        <textarea value={sms} onChange={(event) => setSms(event.target.value)} placeholder={'Generic: TICKET123456 paid ₹500.00 by John\nKotak: Received Rs. 500.00 from John (UPI Ref 12345678)'} />
        <button onClick={() => void simulateWebhook(sms).then((res) => setResult(`${res.action}: ${res.ticketId}`))}>Submit SMS</button>
        {result && <div className="result">{result}</div>}
      </section>
    </div>
  );
}
