import { afterEach, describe, expect, it } from "vitest";
import { withApp } from "./helpers.js";

let ctx: Awaited<ReturnType<typeof withApp>> | undefined;

afterEach(async () => {
  await ctx?.cleanup();
  ctx = undefined;
});

describe("HTTP routes", () => {
  it("creates a ticket, reads status, and confirms via webhook", async () => {
    ctx = await withApp();
    const create = await ctx.app.inject({
      method: "POST",
      url: "/api/ticket",
      payload: { amount: 100 },
    });
    expect(create.statusCode).toBe(200);
    const ticket = create.json<{ ticketId: string; amount: number }>();
    expect(ticket.amount).toBe(100);

    const status = await ctx.app.inject({ method: "GET", url: `/api/status/${ticket.ticketId}` });
    expect(status.statusCode).toBe(200);
    expect(status.json<{ status: string }>().status).toBe("pending");

    const webhook = await ctx.app.inject({
      method: "POST",
      url: "/api/webhook",
      headers: { "x-webhook-secret": ctx.config.webhookSecret },
      payload: { sms: `${ticket.ticketId} SOURAV paid you ₹100.00 UPI Ref:606703736499` },
    });
    expect(webhook.statusCode).toBe(200);
    expect(webhook.json<{ ticketId: string }>().ticketId).toBe(ticket.ticketId);
  });

  it("rejects webhook calls without the shared secret", async () => {
    ctx = await withApp();
    const response = await ctx.app.inject({
      method: "POST",
      url: "/api/webhook",
      payload: { sms: "Confirmed payment for Received Rs.100.00 UPI Ref:606703736500." },
    });
    expect(response.statusCode).toBe(401);
  });
});
