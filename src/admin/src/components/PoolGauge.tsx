import type { PoolSnapshot } from "../types";

export function PoolGauge({ pool }: { pool: PoolSnapshot }) {
  const total = pool.free + pool.pending + pool.paidReserved + pool.reusable;
  const pendingPct = total ? Math.round((pool.pending / total) * 100) : 0;
  return (
    <section className="panel pool-gauge">
      <div>
        <span>Base Amount</span>
        <strong>₹{(pool.baseAmount / 100).toFixed(0)}</strong>
      </div>
      <div className="gauge" style={{ "--value": `${pendingPct}%` } as React.CSSProperties}>
        <b>{pendingPct}%</b>
      </div>
      <div className="pool-grid">
        <span>Free {pool.free}</span>
        <span>Pending {pool.pending}</span>
        <span>Paid reserved {pool.paidReserved}</span>
        <span>Reusable {pool.reusable}</span>
      </div>
    </section>
  );
}
