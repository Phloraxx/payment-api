import type { ReactNode } from "react";

export function money(paise?: number | null): string {
  if (paise == null || !Number.isFinite(paise)) return "—";
  return new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", minimumFractionDigits: 2 }).format(paise / 100);
}
export function dateTime(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "—";
  return new Intl.DateTimeFormat("en-IN", { dateStyle: "medium", timeStyle: "short" }).format(date);
}
export function relativeTime(value?: string | null): string {
  if (!value) return "Never";
  const diff = Date.now() - new Date(value).getTime();
  if (!Number.isFinite(diff)) return "Unknown";
  const abs = Math.abs(diff);
  if (abs < 60_000) return "Just now";
  if (abs < 3_600_000) return `${Math.round(abs / 60_000)}m ago`;
  if (abs < 86_400_000) return `${Math.round(abs / 3_600_000)}h ago`;
  return `${Math.round(abs / 86_400_000)}d ago`;
}
export function cx(...values: Array<string | false | null | undefined>): string { return values.filter(Boolean).join(" "); }

export function Badge({ children, tone = "neutral" }: { children: ReactNode; tone?: "neutral" | "good" | "warn" | "bad" | "blue" }) {
  return <span className={`badge badge-${tone}`}>{children}</span>;
}
export function Dot({ ok }: { ok: boolean }) { return <span className={cx("dot", ok ? "dot-good" : "dot-bad")} />; }
export function Spinner() { return <span className="spinner" aria-label="Loading" />; }
export function Empty({ title, copy }: { title: string; copy: string }) {
  return <div className="empty"><div className="empty-mark">PG</div><strong>{title}</strong><p>{copy}</p></div>;
}
export function ErrorNotice({ message }: { message: string }) { return <div className="notice notice-error">{message}</div>; }
export function SectionHead({ eyebrow, title, copy, action }: { eyebrow?: string; title: string; copy?: string; action?: ReactNode }) {
  return <div className="section-head"><div>{eyebrow && <p className="eyebrow">{eyebrow}</p>}<h2>{title}</h2>{copy && <p>{copy}</p>}</div>{action && <div className="section-action">{action}</div>}</div>;
}
export function Stat({ label, value, sub }: { label: string; value: ReactNode; sub?: ReactNode }) {
  return <div className="stat"><span>{label}</span><strong>{value}</strong>{sub && <small>{sub}</small>}</div>;
}
export function Modal({ title, children, onClose, wide = false }: { title: string; children: ReactNode; onClose: () => void; wide?: boolean }) {
  return <div className="modal-backdrop" role="presentation" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <section className={cx("modal", wide && "modal-wide")} role="dialog" aria-modal="true" aria-label={title}>
      <header><h3>{title}</h3><button className="icon-button" onClick={onClose} aria-label="Close">×</button></header>
      <div className="modal-body">{children}</div>
    </section>
  </div>;
}
export async function copyText(value: string): Promise<boolean> {
  try { await navigator.clipboard.writeText(value); return true; } catch { return false; }
}
