export const paymentsSchema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS tickets (
  id          TEXT PRIMARY KEY,
  amount      INTEGER NOT NULL,
  status      TEXT NOT NULL DEFAULT 'pending'
              CHECK(status IN ('pending','paid','cancelled','expired')),
  base_amount INTEGER NOT NULL,
  decimal_val INTEGER NOT NULL,
  sender_name TEXT,
  rrn         TEXT UNIQUE,
  upi_id      TEXT,
  paid_at     TEXT,
  expires_at  TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tickets_status   ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_amount   ON tickets(amount);
CREATE INDEX IF NOT EXISTS idx_tickets_rrn      ON tickets(rrn);
CREATE INDEX IF NOT EXISTS idx_tickets_decimal  ON tickets(base_amount, decimal_val, status);
CREATE INDEX IF NOT EXISTS idx_tickets_created  ON tickets(created_at);

CREATE TABLE IF NOT EXISTS authenticators (
  id          TEXT PRIMARY KEY,
  public_key  TEXT NOT NULL,
  counter     INTEGER NOT NULL DEFAULT 0,
  device_name TEXT,
  transports  TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS one_time_codes (
  code        TEXT PRIMARY KEY,
  used        INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
`;

export const logsSchema = `
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS logs (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  level       TEXT NOT NULL CHECK(level IN ('info','warn','error','debug')),
  message     TEXT NOT NULL,
  meta        TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_logs_level   ON logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at);
`;
