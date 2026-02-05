/**
 * E2E Tests: Cost Tracking and Attribution
 *
 * Tests cost extraction from CLI output and aggregation to parent session.
 * Per MCP-SPAWN-012 R10: Cost Attribution Strategy
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { resetProcessRegistry } from "../../src/spawn/processRegistry.js";
import {
  getSessionCostTracker,
  resetSessionCostTracker,
} from "../../src/cost/tracker.js";
import type { SpawnResult } from "../../src/mcp/tools/spawnAgent.js";
import { EventEmitter } from "events";
import type { ChildProcess } from "child_process";
import * as cp from "child_process";

// Mock child_process at module level
vi.mock("child_process", async (importOriginal) => {
  const actual = await importOriginal<typeof import("child_process")>();
  return {
    ...actual,
    spawn: vi.fn(),
  };
});

/**
 * Create a mock child process that simulates Claude CLI with cost data
 */
function createMockChildProcessWithCost(
  success: boolean,
  cost: number,
  inputTokens: number,
  outputTokens: number,
  numTurns: number,
  delay = 50
): Partial<ChildProcess> {
  const proc = new EventEmitter() as Partial<ChildProcess>;

  const stdin = new EventEmitter() as any;
  stdin.write = vi.fn();
  stdin.end = vi.fn();

  const stdout = new EventEmitter() as any;
  const stderr = new EventEmitter() as any;

  proc.stdin = stdin;
  proc.stdout = stdout;
  proc.stderr = stderr;
  proc.pid = Math.floor(Math.random() * 10000);
  proc.killed = false;
  proc.kill = vi.fn(() => {
    proc.killed = true;
    return true;
  });

  // Simulate async process execution
  setTimeout(() => {
    if (success) {
      const output = JSON.stringify({
        type: "result",
        subtype: "success",
        cost_usd: cost,
        total_cost_usd: cost,
        input_tokens: inputTokens,
        output_tokens: outputTokens,
        duration_ms: delay,
        num_turns: numTurns,
        result: "Mock analysis complete",
        session_id: `mock-session-${Date.now()}`,
      });
      stdout.emit("data", Buffer.from(output));
    } else {
      stderr.emit("data", Buffer.from("Error"));
    }

    setTimeout(() => {
      (proc as EventEmitter).emit("close", success ? 0 : 1);
      (proc as EventEmitter).emit("exit", success ? 0 : 1);
    }, 10);
  }, delay);

  return proc;
}

describe("Cost Tracking and Attribution", () => {
  beforeEach(() => {
    resetProcessRegistry();
    resetSessionCostTracker();
    useStore.setState({ modalQueue: [], messages: [] });
    useStore.getState().clearAgents();
  });

  afterEach(() => {
    resetProcessRegistry();
    useStore.getState().clearAgents();
    vi.restoreAllMocks();
  });

  describe("CLI output parsing", () => {
    it("should extract cost data from CLI JSON output", async () => {
      const { parseCliOutput } = await import(
        "../../src/mcp/tools/spawnAgent.js"
      );

      const cliOutput = JSON.stringify({
        type: "result",
        subtype: "success",
        cost_usd: 0.0523,
        input_tokens: 12500,
        output_tokens: 3200,
        num_turns: 5,
        result: "Analysis complete",
      });

      const parsed = parseCliOutput(cliOutput);

      expect(parsed.cost).toBe(0.0523);
      expect(parsed.inputTokens).toBe(12500);
      expect(parsed.outputTokens).toBe(3200);
      expect(parsed.turns).toBe(5);
      expect(parsed.result).toBe("Analysis complete");
    });

    it("should handle alternative cost field name (total_cost_usd)", async () => {
      const { parseCliOutput } = await import(
        "../../src/mcp/tools/spawnAgent.js"
      );

      const cliOutput = JSON.stringify({
        total_cost_usd: 0.042,
        num_turns: 3,
        result: "Done",
      });

      const parsed = parseCliOutput(cliOutput);
      expect(parsed.cost).toBe(0.042);
    });

    it("should handle missing cost fields gracefully", async () => {
      const { parseCliOutput } = await import(
        "../../src/mcp/tools/spawnAgent.js"
      );

      const cliOutput = JSON.stringify({
        result: "No cost data",
      });

      const parsed = parseCliOutput(cliOutput);
      expect(parsed.cost).toBeUndefined();
      expect(parsed.inputTokens).toBeUndefined();
      expect(parsed.outputTokens).toBeUndefined();
    });

    it("should handle invalid JSON by returning raw output", async () => {
      const { parseCliOutput } = await import(
        "../../src/mcp/tools/spawnAgent.js"
      );

      const invalidJson = "This is not JSON {{{{";
      const parsed = parseCliOutput(invalidJson);

      expect(parsed.result).toBe(invalidJson);
      expect(parsed.cost).toBeUndefined();
    });
  });

  describe("spawn cost aggregation", () => {
    it("should aggregate spawn costs to parent session", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      // Mock the parent agent as Mozart
      const mozartId = "test-mozart-cost";
      process.env.GOGENT_PARENT_AGENT = mozartId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      const store = useStore.getState();
      store.addAgent({
        id: mozartId,
        parentId: null,
        model: "opus",
        tier: "opus",
        status: "running",
        description: "Test Mozart",
        agentType: "mozart",
        startTime: Date.now(),
      });

      try {
        // Mock spawn with cost data
        const spawnMock = vi.mocked(cp.spawn);
        spawnMock.mockImplementation(() => {
          return createMockChildProcessWithCost(
            true,
            0.0523, // cost
            12500, // input tokens
            3200, // output tokens
            5 // turns
          ) as any;
        });

        // Import spawnAgent AFTER mocking
        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        // Spawn agent
        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test spawn",
          prompt: "AGENT: einstein\n\nTest",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        // Verify cost was extracted
        expect(result.cost).toBe(0.0523);

        // Verify cost was added to tracker
        const costs = tracker.getSpawnCosts();
        expect(costs.length).toBe(1);
        expect(costs[0]?.agentType).toBe("einstein");
        expect(costs[0]?.cost).toBe(0.0523);
        expect(costs[0]?.tokens.input).toBe(12500);
        expect(costs[0]?.tokens.output).toBe(3200);
        expect(costs[0]?.turns).toBe(5);
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should include spawn costs in session total", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      // Add some direct costs
      tracker.addDirectCost(0.05);

      const initialTotal = tracker.getSessionTotal();
      expect(initialTotal).toBe(0.05);

      // Mock parent
      const mozartId = "test-mozart-total";
      process.env.GOGENT_PARENT_AGENT = mozartId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      useStore.getState().addAgent({
        id: mozartId,
        parentId: null,
        model: "opus",
        tier: "opus",
        status: "running",
        description: "Test Mozart",
        agentType: "mozart",
        startTime: Date.now(),
      });

      try {
        const spawnMock = vi.mocked(cp.spawn);
        spawnMock.mockImplementation(() => {
          return createMockChildProcessWithCost(
            true,
            0.03, // spawn cost
            5000,
            1500,
            2
          ) as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        await spawnAgent.handler({
          agent: "einstein",
          description: "Test",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        // Total should include both direct and spawn costs
        const finalTotal = tracker.getSessionTotal();
        expect(finalTotal).toBe(0.08); // 0.05 + 0.03
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should aggregate multiple spawn costs", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      const mozartId = "test-mozart-multiple";
      process.env.GOGENT_PARENT_AGENT = mozartId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      useStore.getState().addAgent({
        id: mozartId,
        parentId: null,
        model: "opus",
        tier: "opus",
        status: "running",
        description: "Test Mozart",
        agentType: "mozart",
        startTime: Date.now(),
      });

      try {
        const spawnMock = vi.mocked(cp.spawn);
        let callCount = 0;
        const costs = [0.05, 0.04, 0.03];

        spawnMock.mockImplementation(() => {
          const cost = costs[callCount++] || 0.01;
          return createMockChildProcessWithCost(
            true,
            cost,
            10000,
            2000,
            3
          ) as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        // Spawn Einstein
        await spawnAgent.handler({
          agent: "einstein",
          description: "Theoretical",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        // Spawn Staff-Architect
        await spawnAgent.handler({
          agent: "staff-architect-critical-review",
          description: "Practical",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        // Spawn Beethoven
        await spawnAgent.handler({
          agent: "beethoven",
          description: "Synthesis",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        // Check aggregation
        const spawnCosts = tracker.getSpawnCosts();
        expect(spawnCosts.length).toBe(3);

        const total = tracker.getSessionTotal();
        expect(total).toBe(0.12); // 0.05 + 0.04 + 0.03
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should group costs by agent type in summary", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      // Manually add spawn costs
      tracker.addSpawnCost({
        agentId: "einstein-1",
        agentType: "einstein",
        cost: 0.05,
        tokens: { input: 10000, output: 2000 },
        turns: 5,
      });

      tracker.addSpawnCost({
        agentId: "einstein-2",
        agentType: "einstein",
        cost: 0.04,
        tokens: { input: 8000, output: 1500 },
        turns: 3,
      });

      tracker.addSpawnCost({
        agentId: "beethoven-1",
        agentType: "beethoven",
        cost: 0.03,
        tokens: { input: 5000, output: 1000 },
        turns: 2,
      });

      const summary = tracker.formatSummary();

      // Should group by agent type
      expect(summary).toContain("einstein: $0.0900 (2x)");
      expect(summary).toContain("beethoven: $0.0300 (1x)");
      expect(summary).toContain("Total: $0.1200");
    });
  });

  describe("session cost summary", () => {
    it("should generate formatted summary with spawn costs", () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      tracker.addDirectCost(0.12);
      tracker.addSpawnCost({
        agentId: "e1",
        agentType: "einstein",
        cost: 0.05,
        tokens: { input: 10000, output: 2000 },
        turns: 5,
      });
      tracker.addSpawnCost({
        agentId: "s1",
        agentType: "staff-architect-critical-review",
        cost: 0.04,
        tokens: { input: 8000, output: 1500 },
        turns: 4,
      });
      tracker.addSpawnCost({
        agentId: "b1",
        agentType: "beethoven",
        cost: 0.03,
        tokens: { input: 6000, output: 1200 },
        turns: 3,
      });

      const summary = tracker.formatSummary();

      expect(summary).toContain("Session Cost Summary:");
      expect(summary).toContain("Router direct costs: $0.1200");
      expect(summary).toContain("Spawn costs:");
      expect(summary).toContain("Total: $0.2400");
    });

    it("should handle empty spawn costs", () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      tracker.addDirectCost(0.05);

      const summary = tracker.formatSummary();

      expect(summary).toContain("Router direct costs: $0.0500");
      expect(summary).toContain("(none)");
      expect(summary).toContain("Total: $0.0500");
    });

    it("should include duration when session ends", () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      tracker.addDirectCost(0.05);
      tracker.end();

      const summary = tracker.formatSummary();
      expect(summary).toContain("Duration:");
    });
  });

  describe("cost tracking with failures", () => {
    it("should not add cost if spawn fails", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      const mozartId = "test-mozart-fail";
      process.env.GOGENT_PARENT_AGENT = mozartId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      useStore.getState().addAgent({
        id: mozartId,
        parentId: null,
        model: "opus",
        tier: "opus",
        status: "running",
        description: "Test Mozart",
        agentType: "mozart",
        startTime: Date.now(),
      });

      try {
        const spawnMock = vi.mocked(cp.spawn);
        spawnMock.mockImplementation(() => {
          return createMockChildProcessWithCost(
            false, // failure
            0.05,
            10000,
            2000,
            5
          ) as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test fail",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );
        expect(result.success).toBe(false);

        // Cost should NOT be added for failed spawns
        const costs = tracker.getSpawnCosts();
        expect(costs.length).toBe(0);

        const total = tracker.getSessionTotal();
        expect(total).toBe(0);
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should not add cost if cost field is missing", async () => {
      const tracker = getSessionCostTracker();
      tracker.reset();

      const mozartId = "test-mozart-nocost";
      process.env.GOGENT_PARENT_AGENT = mozartId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      useStore.getState().addAgent({
        id: mozartId,
        parentId: null,
        model: "opus",
        tier: "opus",
        status: "running",
        description: "Test Mozart",
        agentType: "mozart",
        startTime: Date.now(),
      });

      try {
        const spawnMock = vi.mocked(cp.spawn);
        spawnMock.mockImplementation(() => {
          const proc = new EventEmitter() as Partial<ChildProcess>;
          const stdin = new EventEmitter() as any;
          stdin.write = vi.fn();
          stdin.end = vi.fn();
          const stdout = new EventEmitter() as any;
          const stderr = new EventEmitter() as any;

          proc.stdin = stdin;
          proc.stdout = stdout;
          proc.stderr = stderr;
          proc.pid = 12345;
          proc.killed = false;
          proc.kill = vi.fn(() => true);

          setTimeout(() => {
            // Output without cost field
            const output = JSON.stringify({
              result: "Done",
              // NO cost_usd field
            });
            stdout.emit("data", Buffer.from(output));
            setTimeout(() => {
              (proc as EventEmitter).emit("close", 0);
            }, 10);
          }, 50);

          return proc as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        await spawnAgent.handler({
          agent: "einstein",
          description: "Test no cost",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        // Cost should NOT be added
        const costs = tracker.getSpawnCosts();
        expect(costs.length).toBe(0);
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });
  });
});
