import type { Ticket } from "../../types/index.js";
import type { TicketService } from "./ticket.service.js";

export class ExpiryService {
  private timer: NodeJS.Timeout | undefined;

  constructor(
    private readonly tickets: TicketService,
    private readonly onExpire: (ticket: Ticket) => void,
  ) {}

  start(): void {
    this.stop();
    const tick = () => {
      for (const ticket of this.tickets.expireDue()) {
        this.onExpire(ticket);
      }
      this.timer = setTimeout(tick, 5_000).unref();
    };
    this.timer = setTimeout(tick, 5_000).unref();
  }

  stop(): void {
    if (this.timer) clearTimeout(this.timer);
    this.timer = undefined;
  }
}
