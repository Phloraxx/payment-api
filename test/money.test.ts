import { describe, expect, it } from "vitest";
import { fromPaisa, toPaisa } from "../src/server/money.js";

describe("money helpers", () => {
  it("converts decimal rupees to integer paisa", () => {
    expect(toPaisa(100)).toBe(10000);
    expect(toPaisa("100.03")).toBe(10003);
    expect(toPaisa("1.5")).toBe(150);
  });

  it("rejects invalid amounts", () => {
    expect(() => toPaisa("0")).toThrow();
    expect(() => toPaisa("10.999")).toThrow();
    expect(() => toPaisa("abc")).toThrow();
  });

  it("converts paisa to display amount", () => {
    expect(fromPaisa(10003)).toBe(100.03);
  });
});
