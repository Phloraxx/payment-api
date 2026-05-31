import type Database from "better-sqlite3";
import type { Ticket } from "../../types/index.js";
import { baseAmountFromPaisa, decimalFromPaisa } from "../money.js";
import { AppError } from "../errors.js";

export interface PoolSnapshot {
  baseAmount: number;
  free: number;
  pending: number;
  paidReserved: number;
  reusable: number;
  nextFree: number | undefined;
  freeAmounts: number[];
}

export class DecimalPoolService {
  private readonly pools = new Map<number, number[]>();

  constructor(private readonly db: Database.Database) {}

  rebuild(): void {
    this.pools.clear();
    const rows = this.db.prepare("SELECT * FROM tickets").all() as Ticket[];
    const bases = new Set<number>(rows.map((row) => row.base_amount));
    if (bases.size === 0) return;
    for (const base of bases) {
      this.pools.set(base, this.buildFreeQueue(base, rows));
    }
  }

  allocate(requestedPaisa: number): { amount: number; baseAmount: number; decimalVal: number } {
    const requestedBase = baseAmountFromPaisa(requestedPaisa);
    let queue = this.pools.get(requestedBase);
    if (!queue) {
      queue = this.buildFreeQueue(requestedBase);
      this.pools.set(requestedBase, queue);
    }
    if (queue.length === 0) {
      const nextBase = this.findNextBlock(requestedBase);
      queue = this.buildFreeQueue(nextBase);
      this.pools.set(requestedBase, queue);
    }
    const amount = queue.shift();
    if (amount === undefined) {
      throw new AppError("POOL_EXHAUSTED", "No decimal slots available for this amount.");
    }
    return {
      amount,
      baseAmount: requestedBase,
      decimalVal: decimalFromPaisa(amount),
    };
  }

  release(ticket: Pick<Ticket, "base_amount" | "amount" | "status">): void {
    if (ticket.status === "paid") return;
    const queue = this.pools.get(ticket.base_amount) ?? [];
    if (!queue.includes(ticket.amount)) queue.push(ticket.amount);
    this.pools.set(ticket.base_amount, queue);
  }

  getSnapshot(): PoolSnapshot[] {
    const rows = this.db.prepare("SELECT * FROM tickets").all() as Ticket[];
    const bases = new Set<number>([...rows.map((row) => row.base_amount), ...this.pools.keys()]);
    return [...bases].sort((a, b) => a - b).map((base) => {
      const freeAmounts = this.pools.get(base) ?? this.buildFreeQueue(base, rows);
      return {
        baseAmount: base,
        free: freeAmounts.length,
        pending: rows.filter((row) => row.base_amount === base && row.status === "pending").length,
        paidReserved: rows.filter((row) => row.base_amount === base && row.status === "paid").length,
        reusable: rows.filter((row) => row.base_amount === base && ["expired", "cancelled"].includes(row.status)).length,
        nextFree: freeAmounts[0],
        freeAmounts,
      };
    });
  }

  private findNextBlock(base: number): number {
    let block = base;
    while (true) {
      const free = this.buildFreeQueue(block);
      if (free.length > 0) return block;
      block += 100;
    }
  }

  private buildFreeQueue(base: number, allRows?: Ticket[]): number[] {
    const rows = allRows ?? (this.db.prepare("SELECT * FROM tickets").all() as Ticket[]);
    const unavailable = new Set(
      rows
        .filter((row) => row.amount >= base && row.amount < base + 100 && ["pending", "paid"].includes(row.status))
        .map((row) => row.amount),
    );
    const free: number[] = [];
    for (let amount = base; amount < base + 100; amount += 1) {
      if (!unavailable.has(amount)) free.push(amount);
    }
    return free;
  }
}
