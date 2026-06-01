import { randomBytes } from "node:crypto";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
import "dotenv/config";

export interface Config {
  port: number;
  host: string;
  dataDir: string;
  ticketTtlMinutes: number;
  webhookSecret: string;
  upiId: string;
  upiPayeeName: string;
}

function env(name: string, fallback = ""): string {
  return process.env[name] ?? fallback;
}

export function loadConfig(): Config {
  const dataDir = resolve(env("DATA_DIR", "data"));
  mkdirSync(dataDir, { recursive: true });

  const webhookSecret = env("WEBHOOK_SECRET");
  if (!webhookSecret || webhookSecret.startsWith("change-me")) {
    process.stderr.write("WARN: WEBHOOK_SECRET is missing or weak. Set a strong random value before production.\n");
  }
  if (!env("UPI_ID")) {
    process.stderr.write("WARN: UPI_ID is not set. UPI payment references will not be available.\n");
  }

  return {
    port: Number.parseInt(env("PORT", "3000"), 10),
    host: env("HOST", "0.0.0.0"),
    dataDir,
    ticketTtlMinutes: Number.parseInt(env("TICKET_TTL_MINUTES", "2"), 10),
    webhookSecret: webhookSecret || randomBytes(24).toString("hex"),
    upiId: env("UPI_ID"),
    upiPayeeName: env("UPI_PAYEE_NAME"),
  };
}
