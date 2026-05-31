import type { Services } from "../container.js";
import type { Ticket } from "../../types/index.js";

export function processWebhookSms(services: Services, sms: string): { ticket: Ticket; action: string } {
  const result = services.payments.confirmFromSms(sms);
  services.ws.broadcastTicket(result.ticket.id, {
    type: "payment_update",
    status: result.ticket.status,
    paidAt: result.ticket.paid_at,
    senderName: result.ticket.sender_name,
  });
  services.ws.broadcastTicketUpdate("paid", result.ticket);
  return result;
}
