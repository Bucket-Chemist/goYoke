import { describe, it, expect } from "vitest";
import { spawnMockClaude } from "./spawnHelper";

describe("Mock Claude CLI", () => {
  describe("success behavior", () => {
    it("should return valid JSON with success result", async () => {
      const result = await spawnMockClaude({ behavior: "success" });

      expect(result.exitCode).toBe(0);
      expect(result.killed).toBe(false);

      const output = JSON.parse(result.stdout);
      expect(output.type).toBe("result");
      expect(output.subtype).toBe("success");
      expect(output.cost_usd).toBeGreaterThan(0);
    });

    it("should accept custom output", async () => {
      const result = await spawnMockClaude({
        behavior: "success",
        output: "Custom test output",
      });

      const output = JSON.parse(result.stdout);
      expect(output.result).toBe("Custom test output");
    });
  });

  describe("error behaviors", () => {
    it("should return max_turns error with exit code 1", async () => {
      const result = await spawnMockClaude({ behavior: "error_max_turns" });

      expect(result.exitCode).toBe(1);
      const output = JSON.parse(result.stdout);
      expect(output.subtype).toBe("error_max_turns");
    });

    it("should return rate_limit error on stderr", async () => {
      const result = await spawnMockClaude({ behavior: "error_rate_limit" });

      expect(result.exitCode).toBe(1);
      expect(result.stderr).toContain("rate_limit");
    });
  });

  describe("timeout handling", () => {
    it("should kill hanging process after timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        undefined,
        200 // 200ms timeout
      );

      expect(result.killed).toBe(true);
      expect(result.duration).toBeLessThan(1000);
    }, 3000); // Allow 3s for test to complete (includes 1s SIGKILL escalation)
  });

  describe("invalid output handling", () => {
    it("should return invalid JSON for parsing tests", async () => {
      const result = await spawnMockClaude({ behavior: "invalid_json" });

      expect(result.exitCode).toBe(0);
      expect(() => JSON.parse(result.stdout)).toThrow();
    });

    it("should return partial output for crash simulation", async () => {
      const result = await spawnMockClaude({ behavior: "partial_output" });

      expect(result.stdout).toContain('{"type":');
      expect(() => JSON.parse(result.stdout)).toThrow();
    });
  });

  describe("stdin handling", () => {
    it("should accept stdin content", async () => {
      // Success mock ignores stdin but accepts it
      const result = await spawnMockClaude(
        { behavior: "success" },
        "Test prompt content"
      );

      expect(result.exitCode).toBe(0);
    });
  });
});
