import { afterEach, describe, expect, it } from "vitest";
import { withApp } from "./helpers.js";

let ctx: Awaited<ReturnType<typeof withApp>> | undefined;

afterEach(async () => {
  await ctx?.cleanup();
  ctx = undefined;
});

describe("admin auth flow", () => {
  it("setup status reports needs_setup when no authenticator exists", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({ method: "GET", url: "/api/admin/setup/status" });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ needs_setup: true, has_one_time_code: true });
  });

  it("verify-code accepts the configured one-time code", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({
      method: "POST",
      url: "/api/admin/setup/verify-code",
      payload: { code: ctx.config.oneTimeCode },
    });
    expect(res.statusCode).toBe(200);
  });

  it("register/begin succeeds after code verification", async () => {
    ctx = await withApp();
    await ctx.app.inject({
      method: "POST",
      url: "/api/admin/setup/verify-code",
      payload: { code: ctx.config.oneTimeCode },
    });
    const res = await ctx.app.inject({ method: "GET", url: "/api/admin/register/begin" });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toHaveProperty("publicKey");
  });

  it("login/begin returns 401 when no authenticator is registered", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({ method: "GET", url: "/api/admin/login/begin" });
    expect(res.statusCode).toBe(401);
  });

  it("protected route returns 401 without cookie", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({ method: "GET", url: "/api/admin/session" });
    expect(res.statusCode).toBe(401);
  });

  it("verify-code rejects a wrong code", async () => {
    ctx = await withApp();
    const res = await ctx.app.inject({
      method: "POST",
      url: "/api/admin/setup/verify-code",
      payload: { code: "wrong-code" },
    });
    expect(res.statusCode).toBe(401);
  });
});
