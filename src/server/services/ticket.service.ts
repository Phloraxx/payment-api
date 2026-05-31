import type Database from "better-sqlite3";
import type { Ticket, TicketResponse, TicketStatus } from "../../types/index.js";
import { AppError, isSqliteUniqueError } from "../errors.js";
import { fromPaisa, toPaisa } from "../money.js";
import type { Config } from "../config.js";
import type { AppwriteService } from "./appwrite.service.js";
import type { DecimalPoolService } from "./decimal.service.js";
import type { LoggerService } from "./logger.service.js";

export class TicketService {
  private idCounter = 0;
  private readonly createStmt;
  private readonly getStmt;
  private readonly listStmt;
  private readonly updateStatusStmt;
  private readonly markPaidStmt;
  private readonly updateTicketStmt;

  constructor(
    private readonly db: Database.Database,
    private readonly config: Config,
    private readonly decimalPool: DecimalPoolService,
    private readonly logger: LoggerService,
    private readonly appwrite: AppwriteService,
  ) {
    this.createStmt = db.prepare(`
      INSERT INTO tickets (id, amount, status, base_amount, decimal_val, expires_at)
      VALUES (?, ?, 'pending', ?, ?, ?)
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
    const expiresAt = new Date(Date.now() + this.config.ticketTtlMinutes * 60_000).toISOString();
    this.createStmt.run(id, allocation.amount, allocation.baseAmount, allocation.decimalVal, expiresAt);
    const ticket = this.getTicket(id);
    this.logger.info("Ticket created", {
      ticketId: ticket.id,
      amount: ticket.amount,
      decimal: ticket.decimal_val,
      pool_free: this.decimalPool.getSnapshot().find((pool) => pool.baseAmount === ticket.base_amount)?.free,
    });
    this.appwrite.syncTicket(ticket);
    return ticket;
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

  all(): Ticket[] {
    return this.db.prepare("SELECT * FROM tickets ORDER BY created_at DESC").all() as Ticket[];
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
    this.appwrite.syncTicket(ticket);
    return ticket;
  }

  transition(id: string, status: Exclude<TicketStatus, "pending" | "paid">): Ticket {
    const existing = this.getTicket(id);
    if (existing.status !== "pending") {
      throw new AppError("TICKET_ALREADY_RESOLVED", "Ticket is already resolved.");
    }
    this.updateStatusStmt.run(status, id);
    const ticket = this.getTicket(id);
    this.decimalPool.release(ticket);
    this.logger.info(`Ticket ${status}`, { ticketId: ticket.id, amount: ticket.amount });
    this.appwrite.syncTicket(ticket);
    return ticket;
  }

  expirePending(): number {
    const pending = this.db.prepare("SELECT * FROM tickets WHERE status = 'pending'").all() as Ticket[];
    const tx = this.db.transaction((rows: Ticket[]) => {
      for (const ticket of rows) {
        this.updateStatusStmt.run("expired", ticket.id);
      }
    });
    tx(pending);
    this.decimalPool.rebuild();
    return pending.length;
  }

  expireDue(now = new Date()): Ticket[] {
    const due = this.db
      .prepare("SELECT * FROM tickets WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at <= ?")
      .all(now.toISOString()) as Ticket[];
    for (const ticket of due) {
      this.transition(ticket.id, "expired");
    }
    return due.map((ticket) => this.getTicket(ticket.id));
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
    const ticket = this.getTicket(id);
    this.decimalPool.release(ticket);
    this.logger.info("Payment confirmed", {
      ticketId: ticket.id,
      amount: ticket.amount,
      sender: ticket.sender_name,
      rrn: ticket.rrn,
      match_method: fields.matchMethod ?? "manual",
    });
    this.appwrite.syncTicket(ticket);
    return ticket;
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
    expiresAt: ticket.expires_at,
    createdAt: ticket.created_at,
    paidAt: ticket.paid_at,
    senderName: ticket.sender_name,
    rrn: ticket.rrn,
    upiId: ticket.upi_id,
  };
}
