import type { Config } from "./config.js";
import type { DbBundle } from "./db/connection.js";
import { AppwriteService } from "./services/appwrite.service.js";
import { AuthService } from "./services/auth.service.js";
import { DecimalPoolService } from "./services/decimal.service.js";
import { ExpiryService } from "./services/expiry.service.js";
import { LoggerService } from "./services/logger.service.js";
import { PaymentService } from "./services/payment.service.js";
import { TicketService } from "./services/ticket.service.js";
import { WsManager } from "./ws/manager.js";

export interface Services {
  logger: LoggerService;
  decimalPool: DecimalPoolService;
  appwrite: AppwriteService;
  tickets: TicketService;
  payments: PaymentService;
  auth: AuthService;
  ws: WsManager;
  expiry: ExpiryService;
}

export function createServices(config: Config, db: DbBundle): Services {
  const logger = new LoggerService(db.logs);
  const decimalPool = new DecimalPoolService(db.payments);
  const appwrite = new AppwriteService(config, logger);
  const tickets = new TicketService(db.payments, config, decimalPool, logger, appwrite);
  const payments = new PaymentService(db.payments, tickets);
  const auth = new AuthService(db.payments, config);
  const ws = new WsManager(logger);
  const expiry = new ExpiryService(tickets, (ticket) => ws.broadcastExpired(ticket));
  return { logger, decimalPool, appwrite, tickets, payments, auth, ws, expiry };
}
