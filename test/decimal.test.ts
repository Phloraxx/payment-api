import { afterEach, describe, expect, it } from "vitest";
import { withServices } from "./helpers.js";

let ctx: ReturnType<typeof withServices> | undefined;

afterEach(() => {
  ctx?.cleanup();
  ctx = undefined;
});

describe("decimal allocation", () => {
  it("allocates 100 unique slots then moves into the next integer block", () => {
    ctx = withServices();
    const tickets = Array.from({ length: 101 }, () => ctx!.services.tickets.createTicket(100));
    expect(new Set(tickets.map((ticket) => ticket.amount)).size).toBe(101);
    expect(tickets[0]!.amount).toBe(10000);
    expect(tickets[99]!.amount).toBe(10099);
    expect(tickets[100]!.amount).toBe(10100);
  });

  it("expires pending tickets on recovery", () => {
    ctx = withServices();
    ctx.services.tickets.createTicket(100);
    expect(ctx.services.tickets.expirePending()).toBe(1);
    expect(ctx.services.tickets.list({ status: "expired" })).toHaveLength(1);
  });
});
