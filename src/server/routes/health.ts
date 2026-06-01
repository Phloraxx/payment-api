import type { FastifyInstance } from "fastify";

export async function registerHealthRoute(app: FastifyInstance): Promise<void> {
  app.get("/health", { config: { rateLimit: false } }, async () => ({
    status: "healthy",
    uptime: Math.round(process.uptime()),
    db: "ok",
  }));
}
