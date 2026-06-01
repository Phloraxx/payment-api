import pino from "pino";
import { buildApp } from "./app.js";
import { loadConfig } from "./config.js";
import { openDatabase, closeDatabase } from "./db/connection.js";
import { DecimalPoolService } from "./services/decimal.service.js";
import { TicketService } from "./services/ticket.service.js";
import { PaymentService } from "./services/payment.service.js";
import type { Ticket } from "../types/index.js";

const logger = pino({ level: process.env.LOG_LEVEL ?? "info" });
const config = loadConfig();
const db = openDatabase(config);
const decimalPool = new DecimalPoolService(db);
const tickets = new TicketService(db, config, decimalPool, logger);
const payments = new PaymentService(db, tickets);

const expired = db.prepare("UPDATE tickets SET status = 'expired', updated_at = datetime('now') WHERE status = 'pending'").run();
if (expired.changes > 0) logger.warn({ count: expired.changes }, "Expired stale pending tickets on startup");

const rows = db.prepare("SELECT base_amount, decimal_val, status FROM tickets").all() as Array<Pick<Ticket, "base_amount" | "decimal_val" | "status">>;
decimalPool.rebuild(rows);

const app = await buildApp(config, { db, decimalPool, tickets, payments, logger });

const close = async () => {
  try {
    logger.warn("Graceful shutdown started");
    await app.close();
    closeDatabase(db);
  } catch (err) {
    logger.error({ error: String(err) }, "Shutdown error");
    process.exit(1);
  }
  process.exit(0);
};

process.on("SIGTERM", () => void close());
process.on("SIGINT", () => void close());

await app.listen({ port: config.port, host: config.host });
logger.info({ port: config.port, host: config.host }, "Server listening");
