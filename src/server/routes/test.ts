import type { FastifyInstance } from "fastify";
import { Type } from "@sinclair/typebox";
import type { Services } from "../container.js";
import { requireAdmin } from "../middleware/auth.js";
import { toTicketResponse } from "../services/ticket.service.js";
import { processWebhookSms } from "../services/webhook-helper.js";

export async function registerTestRoutes(app: FastifyInstance, services: Services): Promise<void> {
  app.post(
    "/api/admin/test/ticket",
    { schema: { body: Type.Object({ amount: Type.Union([Type.Number(), Type.String()]) }) } },
    async (request) => {
      requireAdmin(services.auth, request);
      const ticket = services.tickets.createTicket((request.body as { amount: string | number }).amount);
      services.ws.broadcastTicketUpdate("created", ticket);
      return toTicketResponse(ticket);
    },
  );

  app.post(
    "/api/admin/test/webhook",
    { schema: { body: Type.Object({ sms: Type.String({ minLength: 1 }) }) } },
    async (request) => {
      requireAdmin(services.auth, request);
      const result = processWebhookSms(services, (request.body as { sms: string }).sms);
      return { status: "ok", ticketId: result.ticket.id, action: result.action, ticket: toTicketResponse(result.ticket) };
    },
  );
}
