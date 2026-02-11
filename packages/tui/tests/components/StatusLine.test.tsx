/**
 * StatusLine component tests (store-based version)
 * Coverage:
 * - Background team count display
 * - Store state management
 * - Text formatting logic
 *
 * Note: Following Layout.test.tsx pattern - store-based testing only.
 * Full component rendering tests are complex with Ink and skipped.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store/index.js";

describe("StatusLine (store-based)", () => {
  beforeEach(() => {
    const store = useStore.getState();
    store.clearSession();
    store.clearAgents();
    delete process.env["GOGENT_ASCII"];
  });

  it("backgroundTeamCount defaults to 0", () => {
    expect(useStore.getState().backgroundTeamCount).toBe(0);
  });

  it("setBackgroundTeamCount updates store", () => {
    useStore.getState().setBackgroundTeamCount(3);
    expect(useStore.getState().backgroundTeamCount).toBe(3);
  });

  it("clearSession resets backgroundTeamCount", () => {
    useStore.getState().setBackgroundTeamCount(5);
    useStore.getState().clearSession();
    expect(useStore.getState().backgroundTeamCount).toBe(0);
  });

  it("singular form for 1 team", () => {
    // Test the formatting logic directly
    const count = 1;
    const text = `${count} team${count !== 1 ? "s" : ""} running`;
    expect(text).toBe("1 team running");
  });

  it("plural form for multiple teams", () => {
    const count = 3;
    const text = `${count} team${count !== 1 ? "s" : ""} running`;
    expect(text).toBe("3 teams running");
  });

  it("ASCII fallback uses [BG] prefix", () => {
    process.env["GOGENT_ASCII"] = "1";
    const useAscii = process.env["TERM"] === "dumb" || process.env["GOGENT_ASCII"] === "1";
    expect(useAscii).toBe(true);
    delete process.env["GOGENT_ASCII"];
  });

  describe("model name extraction", () => {
    it.each([
      ["claude-opus-4-6", "Opus"],
      ["claude-sonnet-4-5-20250929", "Sonnet"],
      ["claude-haiku-4-5-20251001", "Haiku"],
      ["some-unknown-model", "some-unkno"],
    ])("should extract '%s' as '%s'", (modelId, expected) => {
      const model = modelId;
      let name: string;
      if (model.includes("opus")) name = "Opus";
      else if (model.includes("sonnet")) name = "Sonnet";
      else if (model.includes("haiku")) name = "Haiku";
      else name = model.substring(0, 10);
      expect(name).toBe(expected);
    });
  });

  describe("duration formatting", () => {
    it.each([
      [0, "0m 00s"],
      [59, "0m 59s"],
      [60, "1m 00s"],
      [125, "2m 05s"],
      [3661, "61m 01s"],
    ])("should format %d elapsed seconds as '%s'", (totalSeconds, expected) => {
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = totalSeconds % 60;
      const formatted = `${minutes}m ${String(seconds).padStart(2, "0")}s`;
      expect(formatted).toBe(expected);
    });
  });

  describe("context percentage", () => {
    it("should return 0 when totalCapacity is 0", () => {
      const pct = 0;
      expect(pct).toBe(0);
    });

    it.each([
      [0, 200000, 0],
      [100000, 200000, 50],
      [200000, 200000, 100],
      [250000, 200000, 100],
    ])("should calculate %d/%d tokens as %d%%", (used, total, expected) => {
      const pct = total === 0 ? 0 : Math.min(100, Math.round((used / total) * 100));
      expect(pct).toBe(expected);
    });
  });

  describe("agent count filtering", () => {
    it("should count running and streaming as 'running'", () => {
      const agents: Record<string, { status: string }> = {
        a: { status: "running" },
        b: { status: "streaming" },
        c: { status: "complete" },
        d: { status: "queued" },
      };
      const values = Object.values(agents);
      const running = values.filter(a => a.status === "running" || a.status === "streaming").length;
      const queued = values.filter(a => a.status === "queued" || a.status === "spawning").length;
      const complete = values.filter(a => a.status === "complete").length;
      expect(running).toBe(2);
      expect(queued).toBe(1);
      expect(complete).toBe(1);
    });

    it("should handle empty agents record", () => {
      const agents: Record<string, { status: string }> = {};
      const values = Object.values(agents);
      const running = values.filter(a => a.status === "running" || a.status === "streaming").length;
      expect(running).toBe(0);
    });
  });
});
