import { useEffect, useRef, useState } from "react";
import QRCode from "qrcode";
import { getSettings } from "../api/tickets";

interface QrDisplayProps {
  ticketId: string;
  amount: number;
}

function upiLink(ticketId: string, amount: number, upiId: string, payeeName: string): string {
  const am = amount.toFixed(2);
  return `upi://pay?pa=${upiId}&pn=${encodeURIComponent(payeeName)}&am=${am}&cu=INR&tn=${ticketId}`;
}

export function QrDisplay({ ticketId, amount }: QrDisplayProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [upiId, setUpiId] = useState("");
  const [payeeName, setPayeeName] = useState("");

  useEffect(() => {
    getSettings().then((s) => {
      setUpiId(s.upiId);
      setPayeeName(s.upiPayeeName);
    }).catch(() => {});
  }, []);

  const link = upiId ? upiLink(ticketId, amount, upiId, payeeName) : "";

  useEffect(() => {
    if (!canvasRef.current || !link) return;
    QRCode.toCanvas(canvasRef.current, link, {
      width: 200,
      margin: 2,
    });
  });
  return (
    <div className="panel" style={{ textAlign: "center" }}>
      <h3 style={{ margin: "0 0 12px" }}>Scan to Pay</h3>
      {link ? (
        <canvas ref={canvasRef} style={{ borderRadius: 8 }} />
      ) : (
        <div style={{ width: 200, height: 200, margin: "0 auto", background: "#1e293b", borderRadius: 8 }} />
      )}
      <div className="result" style={{ marginTop: 12, fontSize: 13, wordBreak: "break-all" }}>
        <strong style={{ display: "block", color: "#e2e8f0", fontSize: 15, marginBottom: 4 }}>
          ₹{amount.toFixed(2)}
        </strong>
        <span style={{ fontSize: 11, color: "#64748b" }}>{ticketId}</span>
      </div>
    </div>
  );
}
