import type { FastifyError, FastifyReply, FastifyRequest } from "fastify";
import { AppError } from "../errors.js";

export function errorHandler(error: FastifyError | AppError, _request: FastifyRequest, reply: FastifyReply): void {
  if (error instanceof AppError) {
    reply.status(error.statusCode).send({
      error: {
        code: error.code,
        message: error.message,
        details: error.details,
      },
    });
    return;
  }

  if ((error as FastifyError).statusCode === 429) {
    reply.status(429).send({
      error: {
        code: "RATE_LIMITED",
        message: "Rate limit exceeded.",
      },
    });
    return;
  }

  const message = error instanceof Error ? error.stack ?? error.message : String(error);
  process.stderr.write(`UNHANDLED ERROR: ${message}\n`);
  reply.status(500).send({
    error: {
      code: "INTERNAL_ERROR",
      message: "Unexpected server error.",
    },
  });
}
