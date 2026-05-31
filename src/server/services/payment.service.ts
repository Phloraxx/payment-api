import type Database from "better-sqlite3";
import type { ParsedSms, Ticket } from "../../types/index.js";
import { AppError } from "../errors.js";
import { toPaisa } from "../money.js";
import type { TicketService } from "./ticket.service.js";

export class PaymentService {
  constructor(
    private readonly db: Database.Database,
    private readonly tickets: TicketService,
  ) {}

  parseSms(sms: string): ParsedSms {
    const generic = sms.match(/(TICKET\d+).*?(?:₹|Rs\.?|INR)\s?(\d+(?:\.\d{1,2})?)/i);
    if (generic?.[1] && generic[2]) {
      const sender = sms.match(/TICKET\d+\s+([A-Za-z][A-Za-z .'-]{1,60}?)\s+paid/i)?.[1]?.trim();
      return {
        ticketId: generic[1],
        amount: toPaisa(generic[2]),
        senderName: sender,
        rrn: this.extractRrn(sms),
        upiId: this.extractUpi(sms),
        method: "generic",
      };
    }

    const kotak = sms.match(/(?:Received|payment for Received)\s+(?:Rs\.?|₹)\s?(\d+(?:\.\d{1,2})?)/i);
    if (kotak?.[1]) {
      return {
        amount: toPaisa(kotak[1]),
        rrn: this.extractRrn(sms),
        upiId: this.extractUpi(sms),
        method: "kotak",
      };
    }

    throw new AppError("INVALID_AMOUNT", 'Unrecognized SMS format. Expected: "TICKET123 paid ₹500 by Name" or "Received Rs. 500 from Name".');
  }

  confirmFromSms(sms: string): { ticket: Ticket; action: string; parsed: ParsedSms } {
    const parsed = this.parseSms(sms);
    let ticket: Ticket;
    if (parsed.ticketId) {
      ticket = this.tickets.getTicket(parsed.ticketId);
      if (ticket.amount !== parsed.amount) {
        throw new AppError("AMOUNT_MISMATCH", "SMS amount does not match ticket amount.");
      }
    } else {
      const matches = this.db
        .prepare("SELECT * FROM tickets WHERE amount = ? AND status = 'pending' ORDER BY created_at ASC LIMIT 2")
        .all(parsed.amount) as Ticket[];
      if (matches.length === 0) throw new AppError("TICKET_NOT_FOUND", "No pending ticket matches this payment amount.");
      if (matches.length > 1) throw new AppError("AMOUNT_MISMATCH", "Multiple pending tickets match this amount.");
      ticket = matches[0]!;
    }
    const paid = this.tickets.markPaid(ticket.id, {
      senderName: parsed.senderName,
      rrn: parsed.rrn,
      upiId: parsed.upiId,
      paidAt: new Date().toISOString(),
      matchMethod: parsed.method,
    });
    return { ticket: paid, action: "marked_paid", parsed };
  }

  private extractRrn(sms: string): string | undefined {
    return sms.match(/(?:UPI\s*Ref|RRN|Ref(?:erence)?)[\s:.#-]*(\d{8,20})/i)?.[1];
  }

  private extractUpi(sms: string): string | undefined {
    return sms.match(/[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+/)?.[0];
  }
}
