import type Database from "better-sqlite3";
import type { Ticket } from "../../types/index.js";
import { AppError } from "../errors.js";

function baseAmountFromPaisa(value: number): number {
  return Math.floor(value / 100) * 100;
}

export class DecimalPoolService {
  private readonly pools = new Map<number, Set<number>>();

  constructor(private readonly db: Database.Database) {}

  rebuild(tickets: Pick<Ticket, "base_amount" | "decimal_val" | "status">[]): void {
    this.pools.clear();
    for (const t of tickets) {
      if (t.status !== "pending" && t.status !== "paid") continue;
      let set = this.pools.get(t.base_amount);
      if (!set) {
        set = new Set();
        this.pools.set(t.base_amount, set);
      }
      set.add(t.decimal_val);
    }
  }

  allocate(requestedPaisa: number): { amount: number; baseAmount: number; decimalVal: number } {
    const base = baseAmountFromPaisa(requestedPaisa);
    let set = this.pools.get(base);
    if (!set) {
      set = new Set();
      this.pools.set(base, set);
    }
    for (let i = 0; i < 100; i++) {
      if (!set.has(i)) {
        set.add(i);
        return { amount: base + i, baseAmount: base, decimalVal: i };
      }
    }
    let block = base + 100;
    let attempts = 0;
    while (attempts < 10_000) {
      set = this.pools.get(block);
      if (!set) {
        set = new Set();
        this.pools.set(block, set);
      }
      for (let i = 0; i < 100; i++) {
        if (!set.has(i)) {
          set.add(i);
          return { amount: block + i, baseAmount: block, decimalVal: i };
        }
      }
      block += 100;
      attempts++;
    }
    throw new AppError("POOL_EXHAUSTED", "No decimal slots available for this amount.");
  }

  release(baseAmount: number, decimalVal: number): void {
    const set = this.pools.get(baseAmount);
    if (set) {
      set.delete(decimalVal);
      if (set.size === 0) this.pools.delete(baseAmount);
    }
  }
}
