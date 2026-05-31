import type { FastifyInstance } from "fastify";
import type { Services } from "../container.js";
import { requireAdmin } from "../middleware/auth.js";

export async function registerWsRoutes(app: FastifyInstance, services: Services): Promise<void> {
  app.get<{ Querystring: { ticketId?: string } }>("/api/ws", { websocket: true, config: { rateLimit: { max: 20, timeWindow: "1 minute" } } }, (socket, request) => {
    const ticketId = request.query.ticketId;
    if (!ticketId) {
      socket.close(1008, "ticketId required");
      return;
    }
    services.ws.addTicketSocket(ticketId, socket);
  });

  app.get("/api/admin/ws", { websocket: true, config: { rateLimit: { max: 20, timeWindow: "1 minute" } } }, (socket, request) => {
    try {
      requireAdmin(services.auth, request);
      services.ws.addAdminSocket(socket);
    } catch {
      socket.close(1008, "unauthorized");
    }
  });

  app.get<{ Querystring: { ticketId?: string } }>("/api/admin/test/ws", { websocket: true }, (socket, request) => {
    try {
      requireAdmin(services.auth, request);
      const ticketId = request.query.ticketId;
      if (!ticketId) {
        socket.close(1008, "ticketId required");
        return;
      }
      services.ws.addTicketSocket(ticketId, socket);
    } catch {
      socket.close(1008, "unauthorized");
    }
  });
}
