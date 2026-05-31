import { afterEach, describe, expect, it } from "vitest";
import { withApp } from "./helpers.js";

let ctx: Awaited<ReturnType<typeof withApp>> | undefined;

afterEach(async () => {
  await ctx?.cleanup();
  ctx = undefined;
});

describe("WebSocket", () => {
  it("broadcastTicketUpdate does not throw", async () => {
    ctx = await withApp();
    const ticket = ctx.services.tickets.createTicket(100);
    expect(() => ctx.services.ws.broadcastTicketUpdate("paid", ticket)).not.toThrow();
  });

  it("/api/ws appears in the route tree", async () => {
    ctx = await withApp();
    const routes = ctx.app.printRoutes({ commonPrefix: false });
    expect(routes).toContain("/api/ws");
  });

  it("/api/admin/ws requires admin auth (returns 401 without cookie)", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({ method: "GET", url: "/api/admin/ws" });
    expect(res.statusCode).toBe(401);
  });
});
