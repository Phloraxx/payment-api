import type { FastifyInstance } from "fastify";
import { safeEqual } from "../crypto.js";
import { Type } from "@sinclair/typebox";
import type { Config } from "../config.js";
import type { Services } from "../container.js";
import { AppError } from "../errors.js";
import { toTicketResponse } from "../services/ticket.service.js";
import { processWebhookSms } from "../services/webhook-helper.js";

export async function registerWebhookRoute(app: FastifyInstance, config: Config, services: Services): Promise<void> {
  app.post(
    "/api/webhook",
    {
      config: { rateLimit: { max: 30, timeWindow: "1 minute" } },
      schema: { body: Type.Object({ sms: Type.String({ minLength: 1 }) }) },
    },
    async (request) => {
      const header = request.headers["x-webhook-secret"];
      const querySecret = (request.query as { secret?: string }).secret;
      const secret = Array.isArray(header) ? header[0] : (header ?? querySecret);
      if (!secret || !safeEqual(config.webhookSecret, secret)) {
        services.logger.warn("Webhook auth failure", { ip: request.ip, reason: "bad_secret" });
        throw new AppError("WEBHOOK_UNAUTHORIZED", "Invalid webhook secret.");
      }
      const { sms } = request.body as { sms: string };
      const result = processWebhookSms(services, sms);
      return { status: "ok", ticketId: result.ticket.id, action: result.action, ticket: toTicketResponse(result.ticket) };
    },
  );
}

