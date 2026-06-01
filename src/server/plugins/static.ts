import fastifyStatic from "@fastify/static";
import type { FastifyInstance } from "fastify";
import { existsSync } from "node:fs";
import { resolve } from "node:path";

export async function registerStatic(app: FastifyInstance): Promise<void> {
  const root = existsSync(resolve("src/server/admin/public")) ? resolve("src/server/admin/public") : resolve("admin/public");
  if (!existsSync(root)) return;
  await app.register(fastifyStatic, {
    root,
    prefix: "/admin/",
  });
  app.setNotFoundHandler((request, reply) => {
    if (request.url.startsWith("/admin")) {
      return reply.sendFile("index.html", root);
    }
    return reply.status(404).send({ error: { code: "TICKET_NOT_FOUND", message: "Route not found." } });
  });
}
