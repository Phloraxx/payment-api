import { useEffect, useRef } from "react";
import QRCode from "qrcode";

const UPI_ID = "souravpbijoy-2@okcici";
const PAYEE_NAME = "MuLearn SCET";

interface QrDisplayProps {
  ticketId: string;
  amount: number;
}

function upiLink(ticketId: string, amount: number): string {
  const am = amount.toFixed(2);
  return `upi://pay?pa=${UPI_ID}&pn=${encodeURIComponent(PAYEE_NAME)}&am=${am}&cu=INR&tn=${ticketId}`;
}

export function QrDisplay({ ticketId, amount }: QrDisplayProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const link = upiLink(ticketId, amount);

  useEffect(() => {
    if (!canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, link, {
      width: 200,
      margin: 2,
      color: { dark: "#e2e8f0", light: "#12121a" },
    });
  }, [link]);

  return (
    <div className="panel" style={{ textAlign: "center" }}>
      <h3 style={{ margin: "0 0 12px" }}>Scan to Pay</h3>
      <canvas ref={canvasRef} style={{ borderRadius: 8 }} />
      <div className="result" style={{ marginTop: 12, fontSize: 13, wordBreak: "break-all" }}>
        <strong style={{ display: "block", color: "#e2e8f0", fontSize: 15, marginBottom: 4 }}>
          ₹{amount.toFixed(2)}
        </strong>
        <span style={{ fontSize: 11, color: "#64748b" }}>{ticketId}</span>
      </div>
    </div>
  );
}
