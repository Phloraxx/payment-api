import { useEffect, useState } from "react";
import { getPool } from "../api/tickets";
import { PoolGauge } from "../components/PoolGauge";
import type { PoolSnapshot } from "../types";

export function DecimalPool() {
  const [pools, setPools] = useState<PoolSnapshot[]>([]);
  useEffect(() => {
    void getPool().then((result) => setPools(result.pools));
  }, []);
  return (
    <div className="page">
      <div className="pool-list">
        {pools.map((pool) => (
          <PoolGauge key={pool.baseAmount} pool={pool} />
        ))}
        {pools.length === 0 && <section className="panel empty">No decimal pools active yet.</section>}
      </div>
    </div>
  );
}
