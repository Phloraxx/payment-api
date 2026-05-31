import type { FastifyInstance } from "fastify";
import type { LoggerService } from "../services/logger.service.js";

export function registerRequestLogger(app: FastifyInstance, logger: LoggerService): void {
  app.addHook("onResponse", (request, reply, done) => {
    logger.info("HTTP request", {
      req_id: request.id,
      method: request.method,
      path: request.url,
      status: reply.statusCode,
      duration_ms: Math.round(reply.elapsedTime),
      ip: request.ip,
    });
    done();
  });
}
