import type { FastifyInstance } from "fastify";
import type { Services } from "../container.js";

export async function registerHealthRoute(app: FastifyInstance, services: Services): Promise<void> {
  app.get("/health", { config: { rateLimit: false } }, async () => ({
    status: "healthy",
    uptime: Math.round(process.uptime()),
    db: "ok",
    appwrite_reachable: await services.appwrite.reachable(),
  }));
}
