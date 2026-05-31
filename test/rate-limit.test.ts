import { afterEach, describe, expect, it } from "vitest";
import { withApp } from "./helpers.js";

let ctx: Awaited<ReturnType<typeof withApp>> | undefined;

afterEach(async () => {
  await ctx?.cleanup();
  ctx = undefined;
});

describe("rate limiting", () => {
  it("blocks the 6th POST /api/ticket within the time window", async () => {
    ctx = await withApp();
    const results: number[] = [];
    for (let i = 0; i < 6; i++) {
      const res = await ctx.app.inject({
        method: "POST",
        url: "/api/ticket",
        payload: { amount: 100 },
      });
      results.push(res.statusCode);
    }
    expect(results.slice(0, 5)).toEqual([200, 200, 200, 200, 200]);
    expect(results[5]).toBe(429);
  });

  it("returns RATE_LIMITED error code on 429", async () => {
    ctx = await withApp();
    let lastRes;
    for (let i = 0; i < 6; i++) {
      lastRes = await ctx.app.inject({
        method: "POST",
        url: "/api/ticket",
        payload: { amount: 100 },
      });
    }
    expect(lastRes!.statusCode).toBe(429);
    expect(lastRes!.json().error.code).toBe("RATE_LIMITED");
  });
});
