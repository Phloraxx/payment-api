import cookie from "@fastify/cookie";
import helmet from "@fastify/helmet";
import rateLimit from "@fastify/rate-limit";
import websocket from "@fastify/websocket";
import Fastify from "fastify";
import type { Config } from "./config.js";
import type { Services } from "./container.js";
import { errorHandler } from "./middleware/error-handler.js";
import { registerRequestLogger } from "./middleware/request-logger.js";
import { registerStatic } from "./plugins/static.js";
import { registerAdminRoutes } from "./routes/admin.js";
import { registerHealthRoute } from "./routes/health.js";
import { registerTestRoutes } from "./routes/test.js";
import { registerTicketRoutes } from "./routes/ticket.js";
import { registerWebhookRoute } from "./routes/webhook.js";
import { registerWsRoutes } from "./routes/ws.js";

export async function buildApp(config: Config, services: Services) {
  const app = Fastify({ logger: false, trustProxy: true, bodyLimit: 64 * 1024 });
  app.setErrorHandler(errorHandler);
  await app.register(helmet, {
    contentSecurityPolicy: {
      directives: {
        defaultSrc: ["'self'"],
        scriptSrc: ["'self'"],
        styleSrc: ["'self'", "'unsafe-inline'"],
        imgSrc: ["'self'", "data:"],
        connectSrc: ["'self'", "ws:", "wss:"],
        frameAncestors: ["'none'"],
      },
    },
  });
  await app.register(cookie, { secret: config.cookieSecret });
  await app.register(rateLimit, { max: 100, timeWindow: "1 minute" });
  await app.register(websocket);

  registerRequestLogger(app, services.logger);
  await registerHealthRoute(app, services);
  await registerTicketRoutes(app, services);
  await registerWebhookRoute(app, config, services);
  await registerAdminRoutes(app, config, services);
  await registerTestRoutes(app, services);
  await registerWsRoutes(app, services);
  await registerStatic(app);
  return app;
}
