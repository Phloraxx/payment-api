import helmet from "@fastify/helmet";
import rateLimit from "@fastify/rate-limit";
import Fastify from "fastify";
import type { Logger } from "pino";
import type Database from "better-sqlite3";
import type { Config } from "./config.js";
import type { DecimalPoolService } from "./services/decimal.service.js";
import type { TicketService } from "./services/ticket.service.js";
import type { PaymentService } from "./services/payment.service.js";
import { errorHandler } from "./middleware/error-handler.js";
import { registerRequestLogger } from "./middleware/request-logger.js";
import { registerHealthRoute } from "./routes/health.js";
import { registerTicketRoutes } from "./routes/ticket.js";
import { registerWebhookRoute } from "./routes/webhook.js";

export interface AppServices {
  db: Database.Database;
  logger: Logger;
  decimalPool: DecimalPoolService;
  tickets: TicketService;
  payments: PaymentService;
}

export async function buildApp(config: Config, services: AppServices) {
  const app = Fastify({ logger: false, trustProxy: true, bodyLimit: 64 * 1024 });
  app.setErrorHandler(errorHandler);
  await app.register(helmet, {
    contentSecurityPolicy: {
      directives: {
        defaultSrc: ["'self'"],
        scriptSrc: ["'self'"],
        styleSrc: ["'self'", "'unsafe-inline'"],
        imgSrc: ["'self'", "data:"],
        connectSrc: ["'self'"],
        frameAncestors: ["'none'"],
      },
    },
  });
  await app.register(rateLimit, { max: 100, timeWindow: "1 minute" });

  registerRequestLogger(app, services.logger);
  await registerHealthRoute(app);
  await registerTicketRoutes(app, services);
  await registerWebhookRoute(app, config, services);
  return app;
}
