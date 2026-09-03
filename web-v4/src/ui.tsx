import { useEffect, useRef } from "react";
import type { KeyboardEvent, ReactNode } from "react";

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
export function Dot({ ok }: { ok: boolean }) { return <span aria-hidden="true" className={cx("dot", ok ? "dot-good" : "dot-bad")} />; }
export function Spinner() { return <span className="spinner" role="status" aria-label="Loading" />; }
export function Empty({ title, copy }: { title: string; copy: string }) {
  return <div className="empty"><div className="empty-mark">PG</div><strong>{title}</strong><p>{copy}</p></div>;
}
export function ErrorNotice({ message }: { message: string }) { return <div className="notice notice-error" role="alert">{message}</div>; }
export function SectionHead({ eyebrow, title, copy, action }: { eyebrow?: string; title: string; copy?: string; action?: ReactNode }) {
  return <div className="section-head"><div>{eyebrow && <p className="eyebrow">{eyebrow}</p>}<h2>{title}</h2>{copy && <p>{copy}</p>}</div>{action && <div className="section-action">{action}</div>}</div>;
}
export function Stat({ label, value, sub }: { label: string; value: ReactNode; sub?: ReactNode }) {
  return <div className="stat"><span>{label}</span><strong>{value}</strong>{sub && <small>{sub}</small>}</div>;
}
export function Modal({ title, children, onClose, wide = false }: { title: string; children: ReactNode; onClose: () => void; wide?: boolean }) {
  const dialogRef = useRef<HTMLElement>(null);
  const focusable = () => dialogRef.current ? Array.from(dialogRef.current.querySelectorAll<HTMLElement>('button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')) : [];
  useEffect(() => {
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    focusable()[0]?.focus();
    return () => { if (previous?.isConnected) previous.focus(); };
  }, []);
  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); onClose(); return; }
    if (event.key !== "Tab") return;
    const items = focusable();
    event.stopPropagation();
    if (!items.length) { event.preventDefault(); return; }
    const first = items[0], last = items[items.length - 1], active = document.activeElement;
    if (event.shiftKey && (active === first || !dialogRef.current?.contains(active))) { event.preventDefault(); last.focus(); }
    else if (!event.shiftKey && active === last) { event.preventDefault(); first.focus(); }
  };
  return <div className="modal-backdrop" role="presentation" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <section ref={dialogRef} onKeyDown={onKeyDown} className={cx("modal", wide && "modal-wide")} role="dialog" aria-modal="true" aria-label={title}>
      <header><h3>{title}</h3><button type="button" className="icon-button" onClick={onClose} aria-label="Close">×</button></header>
      <div className="modal-body">{children}</div>
    </section>
  </div>;
}
export async function copyText(value: string): Promise<boolean> {
  try { await navigator.clipboard.writeText(value); return true; } catch { return false; }
}
