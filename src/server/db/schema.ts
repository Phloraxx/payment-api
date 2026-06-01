export const schema = `
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
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tickets_status   ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_amount   ON tickets(amount);
CREATE INDEX IF NOT EXISTS idx_tickets_rrn      ON tickets(rrn);
CREATE INDEX IF NOT EXISTS idx_tickets_decimal  ON tickets(base_amount, decimal_val, status);
CREATE INDEX IF NOT EXISTS idx_tickets_created  ON tickets(created_at);
`;
