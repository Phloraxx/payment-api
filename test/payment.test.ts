import { afterEach, describe, expect, it } from "vitest";
import { withServices } from "./helpers.js";

let ctx: ReturnType<typeof withServices> | undefined;

afterEach(() => {
  ctx?.cleanup();
  ctx = undefined;
});

describe("payment matching", () => {
  it("parses generic SMS and fills sender name", () => {
    ctx = withServices();
    const ticket = ctx.services.tickets.createTicket(100);
    const result = ctx.services.payments.fillFromGenericSms(`${ticket.id} SOURAV paid you ₹100.00 UPI Ref:606703736479`);
    expect(result.action).toBe("name_filled");
    expect(result.ticket.sender_name).toBe("SOURAV");
    expect(result.ticket.status).toBe("pending");
  });

  it("parses bank SMS and marks ticket paid", () => {
    ctx = withServices();
    const ticket = ctx.services.tickets.createTicket(100);
    const result = ctx.services.payments.confirmFromBankSms(
      "Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736480.",
    );
    expect(result.ticket.id).toBe(ticket.id);
    expect(result.ticket.status).toBe("paid");
    expect(result.ticket.upi_id).toBe("user@oksbi");
  });

  it("rejects duplicate RRNs", () => {
    ctx = withServices();
    const one = ctx.services.tickets.createTicket(100);
    const two = ctx.services.tickets.createTicket(100);
    ctx.services.tickets.markPaid(one.id, { rrn: "606703736481" });
    expect(() => ctx!.services.tickets.markPaid(two.id, { rrn: "606703736481" })).toThrow();
  });

  it("reuses paid decimals immediately", () => {
    ctx = withServices();
    const first = ctx.services.tickets.createTicket(100);
    ctx.services.tickets.markPaid(first.id, { rrn: "111122223333" });
    const second = ctx.services.tickets.createTicket(100);
    expect(second.decimal_val).toBe(first.decimal_val);
    expect(second.amount).toBe(first.amount);
  });
});
