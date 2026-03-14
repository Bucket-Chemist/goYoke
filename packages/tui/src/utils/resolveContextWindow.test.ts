import { describe, it, expect } from "vitest";
import { resolveContextWindow } from "./resolveContextWindow.js";

describe("resolveContextWindow", () => {
  it("returns 1M for models with [1m] suffix", () => {
    expect(resolveContextWindow("opus[1m]")).toBe(1_000_000);
    expect(resolveContextWindow("claude-opus-4-6[1m]")).toBe(1_000_000);
    expect(resolveContextWindow("sonnet[1m]")).toBe(1_000_000);
  });

  it("returns 200K for haiku models", () => {
    expect(resolveContextWindow("haiku")).toBe(200_000);
    expect(resolveContextWindow("claude-haiku-4-5")).toBe(200_000);
  });

  it("returns 200K as default", () => {
    expect(resolveContextWindow("opus")).toBe(200_000);
    expect(resolveContextWindow("sonnet")).toBe(200_000);
    expect(resolveContextWindow("")).toBe(200_000);
  });
});
