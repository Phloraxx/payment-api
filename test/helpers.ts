import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import pino from "pino";
import type { Config } from "../src/server/config.js";
import { closeDatabase, openDatabase } from "../src/server/db/connection.js";
import { DecimalPoolService } from "../src/server/services/decimal.service.js";
import { TicketService } from "../src/server/services/ticket.service.js";
import { PaymentService } from "../src/server/services/payment.service.js";

export function withServices() {
  const dir = mkdtempSync(join(tmpdir(), "pg-v2-"));
  const config: Config = {
    port: 0,
    host: "127.0.0.1",
    dataDir: dir,
    ticketTtlMinutes: 2,
    webhookSecret: "test-webhook-secret",
    upiId: "test@upi",
    upiPayeeName: "Test",
  };
  const logger = pino({ level: "silent" });
  const db = openDatabase(config);
  const decimalPool = new DecimalPoolService(db);
  const tickets = new TicketService(db, config, decimalPool, logger);
  const payments = new PaymentService(db, tickets);
  const rows = db.prepare("SELECT base_amount, decimal_val, status FROM tickets").all() as Array<{ base_amount: number; decimal_val: number; status: string }>;
  decimalPool.rebuild(rows);
  return {
    config,
    db,
    services: { db, decimalPool, tickets, payments, logger },
    cleanup: () => {
      closeDatabase(db);
      rmSync(dir, { recursive: true, force: true });
    },
  };
}

export async function withApp() {
  const context = withServices();
  const { buildApp } = await import("../src/server/app.js");
  const app = await buildApp(context.config, context.services);
  await app.ready();
  return {
    ...context,
    app,
    cleanup: async () => {
      await app.close();
      context.cleanup();
    },
  };
}
