import type { FastifyInstance, FastifyReply } from "fastify";
import { Type } from "@sinclair/typebox";
import type { Config } from "../config.js";
import type { Services } from "../container.js";
import { currentSessionToken, requireAdmin } from "../middleware/auth.js";
import { toTicketResponse } from "../services/ticket.service.js";

export async function registerAdminRoutes(app: FastifyInstance, _config: Config, services: Services): Promise<void> {
  app.get("/api/admin/setup/status", async () => services.auth.setupStatus());

  app.post<{ Body: { code: string } }>(
    "/api/admin/setup/verify-code",
    {
      config: { rateLimit: { max: 10, timeWindow: "1 minute" } },
      schema: { body: Type.Object({ code: Type.String({ minLength: 1 }) }) },
    },
    async (request) => {
      services.auth.verifyCode(request.body.code);
      return { ok: true };
    },
  );

  app.get("/api/admin/register/begin", { config: { rateLimit: { max: 10, timeWindow: "1 minute" } } }, async () => ({
    publicKey: await services.auth.beginRegistration(),
  }));

  app.post<{ Body: { credential: unknown } }>(
    "/api/admin/register/complete",
    { schema: { body: Type.Object({ credential: Type.Any() }) } },
    async (request, reply) => {
      const token = await services.auth.completeRegistration(request.body.credential);
      setSession(reply, token);
      return { ok: true };
    },
  );

  app.get("/api/admin/login/begin", { config: { rateLimit: { max: 10, timeWindow: "1 minute" } } }, async () => {
    const result = await services.auth.beginLogin();
    return { requestId: result.requestId, publicKey: result.options };
  });

  app.post<{ Body: { requestId: string; assertion: unknown } }>(
    "/api/admin/login/complete",
    { schema: { body: Type.Object({ requestId: Type.String(), assertion: Type.Any() }) } },
    async (request, reply) => {
      const token = await services.auth.completeLogin(request.body.requestId, request.body.assertion);
      setSession(reply, token);
      return { ok: true };
    },
  );

  app.post("/api/admin/logout", async (request, reply) => {
    services.auth.revokeSession(currentSessionToken(request));
    reply.clearCookie("token", { path: "/api/admin" });
    return { ok: true };
  });

  app.addHook("preHandler", async (request) => {
    if (!request.url.startsWith("/api/admin/")) return;
    const publicPaths = [
      "/api/admin/setup/status",
      "/api/admin/setup/verify-code",
      "/api/admin/register/begin",
      "/api/admin/register/complete",
      "/api/admin/login/begin",
      "/api/admin/login/complete",
    ];
    if (publicPaths.some((path) => request.url.startsWith(path))) return;
    requireAdmin(services.auth, request);
  });

  app.get("/api/admin/session", async () => ({ ok: true }));

  app.get<{ Querystring: { status?: string; q?: string; limit?: string; offset?: string; export?: string } }>("/api/admin/tickets", async (request) => {
    const { status, q, limit, offset, export: exportFormat } = request.query;
    const tickets = services.tickets.list({
      status,
      q,
      limit: limit ? Number.parseInt(limit, 10) : undefined,
      offset: offset ? Number.parseInt(offset, 10) : undefined,
    });
    return { tickets: tickets.map(toTicketResponse), raw: exportFormat ? tickets : undefined };
  });

  app.get<{ Params: { id: string } }>("/api/admin/tickets/:id", async (request) => {
    return toTicketResponse(services.tickets.getTicket(request.params.id));
  });

  app.patch<{ Params: { id: string }; Body: { senderName?: string; rrn?: string; upiId?: string } }>(
    "/api/admin/tickets/:id",
    { schema: { body: Type.Object({ senderName: Type.Optional(Type.String()), rrn: Type.Optional(Type.String()), upiId: Type.Optional(Type.String()) }) } },
    async (request) => {
      return toTicketResponse(services.tickets.updateTicket(request.params.id, request.body));
    },
  );

  app.post<{ Params: { id: string } }>("/api/admin/tickets/:id/mark-paid", async (request) => {
    const ticket = services.tickets.markPaid(request.params.id, { matchMethod: "admin" });
    services.ws.broadcastTicketUpdate("paid", ticket);
    return toTicketResponse(ticket);
  });

  app.post<{ Params: { id: string } }>("/api/admin/tickets/:id/cancel", async (request) => {
    const ticket = services.tickets.transition(request.params.id, "cancelled");
    services.ws.broadcastTicketUpdate("cancelled", ticket);
    return toTicketResponse(ticket);
  });

  app.get("/api/admin/stats", async () => services.tickets.stats());
  app.get<{ Querystring: { level?: string; q?: string; limit?: string; offset?: string } }>("/api/admin/logs", async (request) => {
    const { level, q, limit, offset } = request.query;
    return {
      logs: services.logger.list({
        level,
        q,
        limit: limit ? Number.parseInt(limit, 10) : undefined,
        offset: offset ? Number.parseInt(offset, 10) : undefined,
      }),
    };
  });
  app.get("/api/admin/pool", async () => ({ pools: services.decimalPool.getSnapshot() }));
  app.get("/api/admin/settings", async () => ({
    port: _config.port,
    host: _config.host,
    publicBaseUrl: _config.publicBaseUrl,
    rpId: _config.rpId,
    ticketTtlMinutes: _config.ticketTtlMinutes,
    appwriteEnabled: _config.appwrite.enabled,
    appwriteEndpoint: _config.appwrite.endpoint ?? null,
  }));
  app.post("/api/admin/sync/full", async () => services.appwrite.fullSync(services.tickets.all()));
}

function setSession(reply: FastifyReply, token: string): void {
  reply.setCookie("token", token, {
    httpOnly: true,
    sameSite: "strict",
    path: "/api/admin",
    secure: reply.request.protocol === "https",
    signed: true,
  });
}
