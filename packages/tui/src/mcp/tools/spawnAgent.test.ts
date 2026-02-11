import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
// @ts-expect-error — vitest resolves cross-rootDir imports; tsc can't follow into tests/
import { spawnMockClaude } from "../../../tests/mocks/spawnHelper";
import { resetProcessRegistry, getProcessRegistry } from "../../spawn/processRegistry.js";
import {
  buildCliArgs,
  parseCliOutput,
  getCurrentNestingLevel,
  validateNestingDepth,
} from "./spawnAgent.js";

// Mock agentConfig module
vi.mock("../../spawn/agentConfig.js", () => ({
  getAgentConfig: vi.fn((id: string) => {
    if (id === "einstein") {
      return {
        id: "einstein",
        name: "Einstein",
        model: "opus",
        tier: 3,
        tools: ["Read", "Write", "Glob", "Grep", "TaskGet"],
        cli_flags: {
          allowed_tools: ["Read", "Glob", "Grep"],
          additional_flags: ["--permission-mode delegate"],
        },
      };
    }
    if (id === "go-pro") {
      return {
        id: "go-pro",
        name: "Go Pro",
        model: "sonnet",
        tier: 2,
        tools: ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
        cli_flags: {
          allowed_tools: ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
        },
      };
    }
    return null;
  }),
}));

// Note: Full tests require mock CLI infrastructure from MCP-SPAWN-003

describe("spawn_agent tool", () => {
  beforeEach(() => {
    resetProcessRegistry();
  });

  afterEach(() => {
    resetProcessRegistry();
  });

  describe("buildCliArgs", () => {
    it("should include -p and --output-format json", () => {
      const args = buildCliArgs({});

      expect(args).toContain("-p");
      expect(args).toContain("--output-format");
      expect(args).toContain("json");
    });

    it("should include model when specified", () => {
      const args = buildCliArgs({ model: "opus" });

      expect(args).toContain("--model");
      expect(args).toContain("opus");
    });

    it("should include allowedTools when specified", () => {
      const args = buildCliArgs({ allowedTools: ["Read", "Glob", "Grep"] });

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });

    it("should use delegate permission mode", () => {
      const args = buildCliArgs({});

      expect(args).toContain("--permission-mode");
      expect(args).toContain("delegate");
    });

    it("should include maxBudget when specified", () => {
      const args = buildCliArgs({ maxBudget: 0.50 });

      expect(args).toContain("--max-budget-usd");
      expect(args).toContain("0.5");
    });

    it("should always include --allowedTools flag", () => {
      const args = buildCliArgs({});

      expect(args).toContain("--allowedTools");
    });

    it("should use config cli_flags when no caller tools provided", () => {
      const args = buildCliArgs({ agent: "einstein" });

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });

    it("should override config with caller-provided tools", () => {
      const args = buildCliArgs({
        agent: "einstein",
        allowedTools: ["Read", "Write"]
      });

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Write");
    });

    it("should use conservative fallback for unknown agent", () => {
      const args = buildCliArgs({ agent: "nonexistent-agent" });

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });

    it("should use conservative fallback when no agent or caller tools", () => {
      const args = buildCliArgs({});

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });

    it("should use config tools for go-pro agent", () => {
      const args = buildCliArgs({ agent: "go-pro" });

      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Write,Edit,Bash,Glob,Grep");
    });

    it("should combine model and config tools", () => {
      const args = buildCliArgs({ agent: "einstein", model: "opus" });

      expect(args).toContain("--model");
      expect(args).toContain("opus");
      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });
  });

  describe("parseCliOutput", () => {
    it("should parse valid JSON output", () => {
      const output = JSON.stringify({
        result: "Analysis complete",
        cost_usd: 0.05,
        num_turns: 3,
      });

      const parsed = parseCliOutput(output);

      expect(parsed.result).toBe("Analysis complete");
      expect(parsed.cost).toBe(0.05);
      expect(parsed.turns).toBe(3);
    });

    it("should return raw output for invalid JSON", () => {
      const output = "This is not JSON";
      const parsed = parseCliOutput(output);

      expect(parsed.result).toBe("This is not JSON");
    });

    it("should handle alternate JSON field names", () => {
      const output = JSON.stringify({
        output: "Result text",
        total_cost_usd: 0.12,
        num_turns: 5,
      });

      const parsed = parseCliOutput(output);

      expect(parsed.result).toBe("Result text");
      expect(parsed.cost).toBe(0.12);
      expect(parsed.turns).toBe(5);
    });
  });

  describe("getCurrentNestingLevel", () => {
    it("should return 0 when not set", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      delete process.env['GOGENT_NESTING_LEVEL'];

      expect(getCurrentNestingLevel()).toBe(0);

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });

    it("should return parsed level when set", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      process.env['GOGENT_NESTING_LEVEL'] = "2";

      expect(getCurrentNestingLevel()).toBe(2);

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });

    it("should return 0 for invalid nesting level", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      process.env['GOGENT_NESTING_LEVEL'] = "invalid";

      expect(getCurrentNestingLevel()).toBe(0);

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });
  });

  describe("nesting depth validation", () => {
    it("should reject spawn when at MAX_NESTING_DEPTH", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      process.env['GOGENT_NESTING_LEVEL'] = "10"; // MAX_NESTING_DEPTH

      const error = validateNestingDepth();

      expect(error).not.toBeNull();
      expect(error).toContain("Maximum nesting depth");
      expect(error).toContain("10");

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });

    it("should allow spawn when under MAX_NESTING_DEPTH", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      process.env['GOGENT_NESTING_LEVEL'] = "5";

      const error = validateNestingDepth();

      expect(error).toBeNull();

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });

    it("should allow spawn at level 0", () => {
      const originalEnv = process.env['GOGENT_NESTING_LEVEL'];
      delete process.env['GOGENT_NESTING_LEVEL'];

      const error = validateNestingDepth();

      expect(error).toBeNull();

      process.env['GOGENT_NESTING_LEVEL'] = originalEnv;
    });
  });

  // Integration tests with mock CLI
  describe("integration with mock CLI", () => {
    it("should handle successful spawn", async () => {
      const result = await spawnMockClaude(
        { behavior: "success", output: "Test output" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toContain("success");
    });

    it("should handle timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        "Test prompt",
        100 // 100ms timeout
      );

      expect(result.killed).toBe(true);
    });

    it("should handle error response", async () => {
      const result = await spawnMockClaude(
        { behavior: "error_max_turns" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(1);
    });
  });
});
