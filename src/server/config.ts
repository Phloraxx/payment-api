import { randomBytes } from "node:crypto";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
import "dotenv/config";

export interface Config {
  port: number;
  host: string;
  publicBaseUrl: string;
  rpId: string;
  dataDir: string;
  ticketTtlMinutes: number;
  oneTimeCode: string;
  cookieSecret: string;
  webhookSecret: string;
  appwrite: {
    endpoint?: string | undefined;
    projectId?: string | undefined;
    apiKey?: string | undefined;
    databaseId?: string | undefined;
    collectionId?: string | undefined;
    enabled: boolean;
  };
}

function env(name: string, fallback = ""): string {
  return process.env[name] ?? fallback;
}

export function loadConfig(): Config {
  const dataDir = resolve(env("DATA_DIR", "data"));
  mkdirSync(dataDir, { recursive: true });

  const cookieSecret = env("COOKIE_SECRET");
  const webhookSecret = env("WEBHOOK_SECRET");
  if (!cookieSecret || cookieSecret.startsWith("change-me")) {
    process.stderr.write("WARN: COOKIE_SECRET is missing or weak. Set a strong random value before production.\n");
  }
  if (!webhookSecret || webhookSecret.startsWith("change-me")) {
    process.stderr.write("WARN: WEBHOOK_SECRET is missing or weak. Set a strong random value before production.\n");
  }

  const appwrite = {
    endpoint: env("APPWRITE_ENDPOINT") || undefined,
    projectId: env("APPWRITE_PROJECT_ID") || undefined,
    apiKey: env("APPWRITE_API_KEY") || undefined,
    databaseId: env("APPWRITE_DATABASE_ID") || undefined,
    collectionId: env("APPWRITE_COLLECTION_ID") || undefined,
  };

  return {
    port: Number.parseInt(env("PORT", "3000"), 10),
    host: env("HOST", "0.0.0.0"),
    publicBaseUrl: env("PUBLIC_BASE_URL", "http://localhost:3000"),
    rpId: env("RP_ID", "localhost"),
    dataDir,
    ticketTtlMinutes: Number.parseInt(env("TICKET_TTL_MINUTES", "2"), 10),
    oneTimeCode: env("ONE_TIME_CODE") || `SETUP-${randomBytes(3).toString("hex").toUpperCase()}`,
    cookieSecret: cookieSecret || randomBytes(32).toString("hex"),
    webhookSecret: webhookSecret || randomBytes(24).toString("hex"),
    appwrite: {
      ...appwrite,
      enabled: Boolean(appwrite.endpoint && appwrite.projectId && appwrite.apiKey && appwrite.databaseId && appwrite.collectionId),
    },
  };
}
