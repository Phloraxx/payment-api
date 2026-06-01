import Database from "better-sqlite3";
import { join } from "node:path";
import type { Config } from "../config.js";
import { schema } from "./schema.js";

export function openDatabase(config: Config): Database.Database {
  const db = new Database(join(config.dataDir, "app.db"));
  db.pragma("journal_mode = WAL");
  db.pragma("foreign_keys = ON");
  db.pragma("busy_timeout = 5000");
  db.exec(schema);
  const integrity = db.pragma("integrity_check", { simple: true });
  if (integrity !== "ok") {
    throw new Error(`database integrity_check failed: ${integrity}`);
  }
  return db;
}

export function closeDatabase(db: Database.Database): void {
  db.pragma("wal_checkpoint(TRUNCATE)");
  db.close();
}
