import { afterEach, describe, expect, it } from "vitest";
import { withServices } from "./helpers.js";

let ctx: ReturnType<typeof withServices> | undefined;

afterEach(() => {
  ctx?.cleanup();
  ctx = undefined;
});

describe("payment matching", () => {
  it("parses and confirms generic ticket SMS", () => {
    ctx = withServices();
    const ticket = ctx.services.tickets.createTicket(100);
    const result = ctx.services.payments.confirmFromSms(`${ticket.id} SOURAV paid you ₹100.00 UPI Ref:606703736479`);
    expect(result.ticket.status).toBe("paid");
    expect(result.ticket.rrn).toBe("606703736479");
  });

  it("parses Kotak SMS by exact amount", () => {
    ctx = withServices();
    const ticket = ctx.services.tickets.createTicket(100);
    const result = ctx.services.payments.confirmFromSms(
      "Confirmed payment for Received Rs.100.00 in your Kotak Bank AC X4959 from user@oksbi on 08-03-26.UPI Ref:606703736480.",
    );
    expect(result.ticket.id).toBe(ticket.id);
    expect(result.ticket.upi_id).toBe("user@oksbi");
  });

  it("rejects duplicate RRNs", () => {
    ctx = withServices();
    const one = ctx.services.tickets.createTicket(100);
    const two = ctx.services.tickets.createTicket(100);
    ctx.services.payments.confirmFromSms(`${one.id} A paid you ₹100.00 UPI Ref:606703736481`);
    expect(() => ctx!.services.payments.confirmFromSms(`${two.id} B paid you ₹100.01 UPI Ref:606703736481`)).toThrow();
  });

  it("does not reuse paid decimals but reuses expired decimals", () => {
    ctx = withServices();
    const paid = ctx.services.tickets.createTicket(100);
    ctx.services.tickets.markPaid(paid.id, { rrn: "111122223333" });
    const next = ctx.services.tickets.createTicket(100);
    expect(next.amount).not.toBe(paid.amount);

    ctx.services.tickets.transition(next.id, "expired");
    expect(ctx.services.decimalPool.getSnapshot()[0]?.freeAmounts).toContain(next.amount);
  });
});
