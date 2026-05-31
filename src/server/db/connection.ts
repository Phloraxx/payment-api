import Database from "better-sqlite3";
import { join } from "node:path";
import { Config } from "../config.js";
import { logsSchema, paymentsSchema } from "./schema.js";

export interface DbBundle {
  payments: Database.Database;
  logs: Database.Database;
}

function openDb(path: string): Database.Database {
  const db = new Database(path);
  db.pragma("journal_mode = WAL");
  db.pragma("foreign_keys = ON");
  db.pragma("busy_timeout = 5000");
  return db;
}

export function openDatabases(config: Config): DbBundle {
  const payments = openDb(join(config.dataDir, "payments.db"));
  const logs = openDb(join(config.dataDir, "logs.db"));
  payments.exec(paymentsSchema);
  logs.exec(logsSchema);

  const integrity = payments.pragma("integrity_check", { simple: true });
  if (integrity !== "ok") {
    throw new Error(`payments.db integrity_check failed: ${integrity}`);
  }

  payments.prepare("DELETE FROM one_time_codes WHERE used = 0").run();
  payments
    .prepare("INSERT OR IGNORE INTO one_time_codes (code, used) VALUES (?, 0)")
    .run(config.oneTimeCode);

  return { payments, logs };
}

export function closeDatabases(db: DbBundle): void {
  db.payments.pragma("wal_checkpoint(TRUNCATE)");
  db.logs.pragma("wal_checkpoint(TRUNCATE)");
  db.payments.close();
  db.logs.close();
}
