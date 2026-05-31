import { afterEach, describe, expect, it } from "vitest";
import { withApp } from "./helpers.js";

let ctx: Awaited<ReturnType<typeof withApp>> | undefined;

afterEach(async () => {
  await ctx?.cleanup();
  ctx = undefined;
});

describe("Appwrite sync", () => {
  it("fullSync returns 0 attempted when appwrite is disabled", async () => {
    ctx = await withApp();
    const result = await ctx.services.appwrite.fullSync([]);
    expect(result).toEqual({ attempted: 0, failed: 0 });
  });

  it("POST /api/admin/sync/full returns expected shape with valid session", async () => {
    ctx = await withApp();
    const token = ctx.services.auth.createSession();
    const signed = ctx.app.signCookie(token);
    const res = await ctx.app.inject({
      method: "POST",
      url: "/api/admin/sync/full",
      cookies: { token: signed },
    });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ attempted: 0, failed: 0 });
  });
});
