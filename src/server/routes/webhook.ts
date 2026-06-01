import { timingSafeEqual } from "node:crypto";
import type { FastifyInstance } from "fastify";
import { Type } from "@sinclair/typebox";
import type { Config } from "../config.js";
import type { AppServices } from "../app.js";
import { AppError } from "../errors.js";
import { toTicketResponse } from "../services/ticket.service.js";

function safeEqual(expected: string, actual: string | undefined): boolean {
  if (!actual) return false;
  const a = Buffer.from(expected);
  const b = Buffer.from(actual);
  return a.length === b.length && timingSafeEqual(a, b);
}

export async function registerWebhookRoute(app: FastifyInstance, config: Config, services: AppServices): Promise<void> {
  app.post(
    "/api/webhook",
    {
      config: { rateLimit: { max: 30, timeWindow: "1 minute" } },
      schema: { body: Type.Object({ sms: Type.String({ minLength: 1 }) }) },
    },
    async (request) => {
      const header = request.headers["x-webhook-secret"];
      const secret = Array.isArray(header) ? header[0] : header;
      if (!secret || !safeEqual(config.webhookSecret, secret)) {
        services.logger.warn({ ip: request.ip, reason: "bad_secret" }, "Webhook auth failure");
        throw new AppError("WEBHOOK_UNAUTHORIZED", "Invalid webhook secret.");
      }
      const { sms } = request.body as { sms: string };
      const parsed = services.payments.parseSms(sms);
      if (parsed.method === "kotak") {
        const result = services.payments.confirmFromKotakSms(sms);
        return { status: "ok", ticketId: result.ticket.id, action: result.action, ticket: toTicketResponse(result.ticket) };
      }
      const result = services.payments.fillFromGenericSms(sms);
      return { status: "ok", ticketId: result.ticket.id, action: result.action, ticket: toTicketResponse(result.ticket) };
    },
  );
}

