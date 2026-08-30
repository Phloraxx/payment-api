import type { ReactNode } from "react";

export function Badge({ status }: { status?: string }) {
  const value = status || "unknown";
  return <span className={`badge ${value}`}>{value}</span>;
}

export function Stat({ label, value, tone = "" }: { label: string; value: number; tone?: string }) {
  return <div className={`card stat ${tone}`}><span>{label}</span><strong>{value}</strong></div>;
}

export function Modal({ title, onClose, children, variant = "dialog" }: { title: string; onClose: () => void; children: ReactNode; variant?: "dialog" | "drawer" }) {
  return <div className={`modal-backdrop ${variant}`} role="presentation" onMouseDown={onClose}>
    <section className={`modal ${variant}`} role="dialog" aria-modal="true" aria-label={title} onMouseDown={(event) => event.stopPropagation()}>
      <div className="section-title"><h2>{title}</h2><button className="ghost" onClick={onClose}>Close</button></div>
      {children}
    </section>
  </div>;
}

export function formatDate(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
