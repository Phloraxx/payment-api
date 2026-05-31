import type Database from "better-sqlite3";
import pino from "pino";
import type { LogEntry, LogLevel } from "../../types/index.js";

export type LogListener = (entry: LogEntry) => void;

export class LoggerService {
  private inserts = 0;
  private listeners = new Set<LogListener>();
  private readonly pino = pino({ level: process.env.LOG_LEVEL ?? "info" });
  private readonly insertStmt;
  private readonly listStmt;
  private readonly cleanupStmt;

  constructor(private readonly db: Database.Database) {
    this.insertStmt = db.prepare("INSERT INTO logs (level, message, meta) VALUES (?, ?, ?)");
    this.listStmt = db.prepare(`
      SELECT * FROM logs
      WHERE (? IS NULL OR level = ?)
        AND (? IS NULL OR message LIKE '%' || ? || '%' OR meta LIKE '%' || ? || '%')
      ORDER BY id DESC
      LIMIT ? OFFSET ?
    `);
    this.cleanupStmt = db.prepare(`
      DELETE FROM logs
      WHERE id NOT IN (SELECT id FROM logs ORDER BY id DESC LIMIT 10000)
    `);
  }

  onLog(listener: LogListener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  log(level: LogLevel, message: string, meta?: Record<string, unknown>): void {
    const metaText = meta ? JSON.stringify(meta) : null;
    const result = this.insertStmt.run(level, message, metaText);
    const entry = this.db.prepare("SELECT * FROM logs WHERE id = ?").get(result.lastInsertRowid) as LogEntry;
    this.inserts += 1;
    if (this.inserts % 500 === 0) {
      this.cleanupStmt.run();
    }
    this.pino[level]({ meta }, message);
    for (const listener of this.listeners) listener(entry);
  }

  info(message: string, meta?: Record<string, unknown>): void {
    this.log("info", message, meta);
  }

  warn(message: string, meta?: Record<string, unknown>): void {
    this.log("warn", message, meta);
  }

  error(message: string, meta?: Record<string, unknown>): void {
    this.log("error", message, meta);
  }

  debug(message: string, meta?: Record<string, unknown>): void {
    this.log("debug", message, meta);
  }

  list(params: { level?: string | undefined; q?: string | undefined; limit?: number | undefined; offset?: number | undefined }): LogEntry[] {
    const level = params.level || null;
    const q = params.q || null;
    const limit = Math.min(params.limit ?? 100, 500);
    const offset = params.offset ?? 0;
    return this.listStmt.all(level, level, q, q, q, limit, offset) as LogEntry[];
  }
}
