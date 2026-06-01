import type { FastifyInstance } from "fastify";
import type { Logger } from "pino";

export function registerRequestLogger(app: FastifyInstance, logger: Logger): void {
  app.addHook("onResponse", (request, reply, done) => {
    logger.info({
      method: request.method,
      path: request.url,
      status: reply.statusCode,
      duration_ms: Math.round(reply.elapsedTime),
    }, "HTTP request");
    done();
  });
}
