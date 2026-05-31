import type Database from "better-sqlite3";
import type { Ticket } from "../../types/index.js";
import { baseAmountFromPaisa, decimalFromPaisa } from "../money.js";
import { AppError } from "../errors.js";

const PENDING_RELEASE_MS = 5 * 60 * 1000;

interface PendingEntry {
  amount: number;
  releaseAt: number;
}

export interface PoolSnapshot {
  baseAmount: number;
  free: number;
  pending: number;
  paidReserved: number;
  reusable: number;
  pendingRelease: number;
  nextFree: number | undefined;
  freeAmounts: number[];
}

export class DecimalPoolService {
  private readonly pools = new Map<number, number[]>();
  private readonly pendingRelease = new Map<number, PendingEntry[]>();

  constructor(private readonly db: Database.Database) {}

  rebuild(): void {
    this.pools.clear();
    this.pendingRelease.clear();
    const rows = this.db.prepare("SELECT * FROM tickets").all() as Ticket[];
    const now = Date.now();
    for (const row of rows) {
      if (row.status === "pending") continue;
      const createdMs = new Date(row.created_at + "Z").getTime();
      const releaseAt = createdMs + PENDING_RELEASE_MS;
      if (releaseAt <= now) continue;
      this.scheduleRelease(row, now);
    }
    const bases = new Set<number>(rows.map((row) => row.base_amount));
    if (bases.size === 0) return;
    for (const base of bases) {
      this.pools.set(base, this.buildFreeQueue(base, rows));
    }
  }

  allocate(requestedPaisa: number): { amount: number; baseAmount: number; decimalVal: number } {
    const requestedBase = baseAmountFromPaisa(requestedPaisa);
    this.sweepPending(requestedBase);
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

  release(ticket: Pick<Ticket, "base_amount" | "amount" | "created_at">): void {
    this.scheduleRelease(ticket, Date.now());
  }

  getSnapshot(): PoolSnapshot[] {
    const rows = this.db.prepare("SELECT * FROM tickets").all() as Ticket[];
    const bases = new Set<number>([...rows.map((row) => row.base_amount), ...this.pools.keys()]);
    return [...bases].sort((a, b) => a - b).map((base) => {
      const freeAmounts = this.pools.get(base) ?? this.buildFreeQueue(base, rows);
      const pendingReleaseCount = (this.pendingRelease.get(base) ?? []).length;
      return {
        baseAmount: base,
        free: freeAmounts.length,
        pending: rows.filter((row) => row.base_amount === base && row.status === "pending").length,
        paidReserved: rows.filter((row) => row.base_amount === base && row.status === "paid").length,
        reusable: rows.filter((row) => row.base_amount === base && ["expired", "cancelled"].includes(row.status)).length,
        pendingRelease: pendingReleaseCount,
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

  private scheduleRelease(ticket: Pick<Ticket, "base_amount" | "amount" | "created_at">, now: number): void {
    const createdMs = new Date(ticket.created_at + "Z").getTime();
    const releaseAt = createdMs + PENDING_RELEASE_MS;
    if (releaseAt <= now) return;
    const base = ticket.base_amount;
    const queue = this.pendingRelease.get(base) ?? [];
    if (!queue.some((e) => e.amount === ticket.amount)) {
      queue.push({ amount: ticket.amount, releaseAt });
      this.pendingRelease.set(base, queue);
    }
  }

  private sweepPending(base: number): void {
    const pending = this.pendingRelease.get(base);
    if (!pending || pending.length === 0) return;
    const now = Date.now();
    const ready: number[] = [];
    const remaining: PendingEntry[] = [];
    for (const entry of pending) {
      if (entry.releaseAt <= now) {
        ready.push(entry.amount);
      } else {
        remaining.push(entry);
      }
    }
    if (remaining.length === 0) {
      this.pendingRelease.delete(base);
    } else {
      this.pendingRelease.set(base, remaining);
    }
    if (ready.length === 0) return;
    const pool = this.pools.get(base) ?? [];
    for (const amount of ready) {
      if (!pool.includes(amount)) pool.push(amount);
    }
    this.pools.set(base, pool);
  }
}