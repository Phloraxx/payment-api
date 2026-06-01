import type { FastifyInstance } from "fastify";
import { Type } from "@sinclair/typebox";
import type { AppServices } from "../app.js";
import { toTicketResponse } from "../services/ticket.service.js";

export async function registerTicketRoutes(app: FastifyInstance, services: AppServices): Promise<void> {
  app.post<{ Body: { amount: number | string } }>(
    "/api/ticket",
    {
      config: { rateLimit: { max: 5, timeWindow: "1 minute" } },
      schema: {
        body: Type.Object({ amount: Type.Union([Type.Number(), Type.String()]) }),
      },
    },
    async (request) => {
      const ticket = services.tickets.createTicket(request.body.amount);
      return toTicketResponse(ticket);
    },
  );

  app.get<{ Params: { id: string } }>(
    "/api/status/:id",
    { config: { rateLimit: { max: 60, timeWindow: "1 minute" } } },
    async (request) => {
      return toTicketResponse(services.tickets.getTicket(request.params.id));
    },
  );
}
