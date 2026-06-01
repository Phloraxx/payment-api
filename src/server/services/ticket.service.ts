import type Database from "better-sqlite3";
import type { Ticket, TicketResponse, TicketStatus } from "../../types/index.js";
import { AppError, isSqliteUniqueError } from "../errors.js";
import type { Logger } from "pino";
import { fromPaisa, toPaisa } from "../money.js";
import type { Config } from "../config.js";
import type { DecimalPoolService } from "./decimal.service.js";

export class TicketService {
  private idCounter = 0;
  private readonly createStmt;
  private readonly getStmt;
  private readonly listStmt;
  private readonly updateStatusStmt;
  private readonly markPaidStmt;
  private readonly updateTicketStmt;
  private readonly expiryTimers = new Map<string, NodeJS.Timeout>();
  private readonly graceTimers = new Map<string, NodeJS.Timeout>();
  private readonly releaseTimers = new Map<string, NodeJS.Timeout>();

  constructor(
    private readonly db: Database.Database,
    private readonly config: Config,
    private readonly decimalPool: DecimalPoolService,
    private readonly logger: Logger,
  ) {
    this.createStmt = db.prepare(`
      INSERT INTO tickets (id, amount, status, base_amount, decimal_val)
      VALUES (?, ?, 'pending', ?, ?)
    `);
    this.getStmt = db.prepare("SELECT * FROM tickets WHERE id = ?");
    this.listStmt = db.prepare(`
      SELECT * FROM tickets
      WHERE (? IS NULL OR status = ?)
        AND (? IS NULL OR id LIKE '%' || ? || '%' OR sender_name LIKE '%' || ? || '%' OR rrn LIKE '%' || ? || '%')
      ORDER BY created_at DESC
      LIMIT ? OFFSET ?
    `);
    this.updateStatusStmt = db.prepare(`
      UPDATE tickets
      SET status = ?, updated_at = datetime('now')
      WHERE id = ? AND status = 'pending'
    `);
    this.markPaidStmt = db.prepare(`
      UPDATE tickets
      SET status = 'paid',
          sender_name = COALESCE(?, sender_name),
          rrn = COALESCE(?, rrn),
          upi_id = COALESCE(?, upi_id),
          paid_at = COALESCE(?, datetime('now')),
          updated_at = datetime('now')
      WHERE id = ? AND status = 'pending'
    `);
    this.updateTicketStmt = db.prepare(`
      UPDATE tickets
      SET sender_name = COALESCE(?, sender_name),
          rrn = COALESCE(?, rrn),
          upi_id = COALESCE(?, upi_id),
          updated_at = datetime('now')
      WHERE id = ?
    `);
  }

  createTicket(rawAmount: number | string): Ticket {
    const requested = toPaisa(rawAmount);
    const allocation = this.decimalPool.allocate(requested);
    const id = `TICKET${Date.now()}${(this.idCounter++ % 10000).toString().padStart(4, "0")}`;
    this.createStmt.run(id, allocation.amount, allocation.baseAmount, allocation.decimalVal);
    const ticket = this.getTicket(id);
    const ms = this.config.ticketTtlMinutes * 60_000;
    const handle = setTimeout(() => this.onTtlReached(ticket), ms);
    this.expiryTimers.set(ticket.id, handle);
    this.logger.info({ ticketId: ticket.id, amount: ticket.amount, decimal: ticket.decimal_val }, "Ticket created");
    return ticket;
  }

  private onTtlReached(ticket: Ticket): void {
    this.expiryTimers.delete(ticket.id);
    const handle = setTimeout(() => {
      this.graceTimers.delete(ticket.id);
      const existing = this.getTicket(ticket.id);
      if (existing.status !== "pending") return;
      this.updateStatusStmt.run("expired", existing.id);
      this.logger.info({ ticketId: existing.id, amount: existing.amount }, "Ticket expired");
      const releaseHandle = setTimeout(() => {
        this.releaseTimers.delete(existing.id);
        this.decimalPool.release(existing.base_amount, existing.decimal_val);
      }, 30_000).unref();
      this.releaseTimers.set(existing.id, releaseHandle);
    }, 30_000).unref();
    this.graceTimers.set(ticket.id, handle);
  }

  getTicket(id: string): Ticket {
    const ticket = this.getStmt.get(id) as Ticket | undefined;
    if (!ticket) throw new AppError("TICKET_NOT_FOUND", "Ticket does not exist.");
    return ticket;
  }

  list(params: { status?: string | undefined; q?: string | undefined; limit?: number | undefined; offset?: number | undefined }): Ticket[] {
    const status = params.status || null;
    const q = params.q || null;
    const limit = Math.min(params.limit ?? 100, 500);
    const offset = params.offset ?? 0;
    return this.listStmt.all(status, status, q, q, q, q, limit, offset) as Ticket[];
  }

  updateTicket(id: string, fields: { senderName?: string | undefined; rrn?: string | undefined; upiId?: string | undefined }): Ticket {
    const existing = this.getTicket(id);
    try {
      this.updateTicketStmt.run(fields.senderName ?? null, fields.rrn ?? null, fields.upiId ?? null, id);
    } catch (error) {
      if (isSqliteUniqueError(error)) throw new AppError("RRN_DUPLICATE", "RRN has already been processed.");
      throw error;
    }
    const ticket = this.getTicket(existing.id);
    return ticket;
  }

  markPaid(
    id: string,
    fields: {
      senderName?: string | undefined;
      rrn?: string | undefined;
      upiId?: string | undefined;
      paidAt?: string | undefined;
      matchMethod?: string | undefined;
    } = {},
  ): Ticket {
    const existing = this.getTicket(id);
    if (existing.status !== "pending") {
      throw new AppError("TICKET_ALREADY_RESOLVED", "Ticket is already resolved.");
    }
    try {
      this.markPaidStmt.run(fields.senderName ?? null, fields.rrn ?? null, fields.upiId ?? null, fields.paidAt ?? null, id);
    } catch (error) {
      if (isSqliteUniqueError(error)) throw new AppError("RRN_DUPLICATE", "RRN has already been processed.");
      throw error;
    }
    this.clearTimers(id);
    const ticket = this.getTicket(id);
    this.decimalPool.release(ticket.base_amount, ticket.decimal_val);
    this.logger.info({
      ticketId: ticket.id,
      amount: ticket.amount,
      sender: ticket.sender_name,
      rrn: ticket.rrn,
      match_method: fields.matchMethod ?? "manual",
    }, "Payment confirmed");
    return ticket;
  }

  cancelTicket(id: string): Ticket {
    const existing = this.getTicket(id);
    if (existing.status !== "pending") throw new AppError("TICKET_ALREADY_RESOLVED", "Ticket is already resolved.");
    this.updateStatusStmt.run("cancelled", id);
    this.clearTimers(id);
    const ticket = this.getTicket(id);
    const handle = setTimeout(() => {
      this.releaseTimers.delete(id);
      this.decimalPool.release(ticket.base_amount, ticket.decimal_val);
    }, 30_000).unref();
    this.releaseTimers.set(id, handle);
    this.logger.info({ ticketId: ticket.id, amount: ticket.amount }, "Ticket cancelled");
    return ticket;
  }

  fillSenderName(id: string, senderName: string | undefined): Ticket {
    if (!senderName) return this.getTicket(id);
    this.updateTicketStmt.run(senderName, null, null, id);
    return this.getTicket(id);
  }

  private clearTimers(id: string): void {
    const e = this.expiryTimers.get(id);
    if (e) { clearTimeout(e); this.expiryTimers.delete(id); }
    const g = this.graceTimers.get(id);
    if (g) { clearTimeout(g); this.graceTimers.delete(id); }
    const r = this.releaseTimers.get(id);
    if (r) { clearTimeout(r); this.releaseTimers.delete(id); }
  }

  stats(): Record<string, number> {
    const rows = this.db
      .prepare("SELECT status, COUNT(*) as count, COALESCE(SUM(amount), 0) as total FROM tickets GROUP BY status")
      .all() as Array<{ status: TicketStatus; count: number; total: number }>;
    const stats: Record<string, number> = { total: 0, pending: 0, paid: 0, cancelled: 0, expired: 0, revenue: 0 };
    for (const row of rows) {
      stats[row.status] = row.count;
      stats.total = (stats.total ?? 0) + row.count;
      if (row.status === "paid") stats.revenue = row.total;
    }
    return stats;
  }
}

export function toTicketResponse(ticket: Ticket): TicketResponse {
  return {
    ticketId: ticket.id,
    amount: fromPaisa(ticket.amount),
    amountPaisa: ticket.amount,
    status: ticket.status,
    createdAt: ticket.created_at,
    paidAt: ticket.paid_at,
    senderName: ticket.sender_name,
    rrn: ticket.rrn,
    upiId: ticket.upi_id,
  };
}
