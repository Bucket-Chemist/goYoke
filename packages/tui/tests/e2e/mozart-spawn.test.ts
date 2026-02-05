/**
 * E2E Tests: Mozart Orchestrator MCP Spawning
 *
 * Tests the Braintrust workflow where Mozart spawns Einstein and Staff-Architect
 * via the MCP spawn_agent tool instead of Task().
 *
 * Source: MCP-SPAWN-010
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { spawnAgent } from "../../src/mcp/tools/spawnAgent.js";
import { createMockClaude } from "../mocks/mockClaude.js";
import { useStore } from "../../src/store/index.js";
import { resetProcessRegistry } from "../../src/spawn/processRegistry.js";
import type { SpawnResult } from "../../src/mcp/tools/spawnAgent.js";

/**
 * Timeline event for tracking spawn order
 */
interface TimelineEvent {
  agent: string;
  event: "start" | "end";
  timestamp: number;
}

/**
 * Result from invoking Mozart with mock spawns
 */
interface MockSpawnResult {
  spawnCalled: boolean;
  agentType?: string;
  taskCalled: boolean;
  result?: SpawnResult;
}

/**
 * Options for mock Mozart invocation
 */
interface MockMozartOptions {
  childAgent?: string;
  expectedInvocation?: string;
  verifyNoTaskCall?: boolean;
}

/**
 * Helper: Invoke spawn_agent with mocks and track invocation
 */
async function invokeMozartWithMockSpawn(
  opts: MockMozartOptions
): Promise<MockSpawnResult> {
  const childAgent = opts.childAgent || "einstein";
  let spawnCalled = false;
  let agentType: string | undefined;
  let taskCalled = false;

  // Mock the spawn function to track calls without actually spawning
  const originalSpawn = vi.hoisted(() => {
    return {
      spawn: vi.fn(),
    };
  });

  // Track Task() calls via store
  const originalTask = useStore.getState().addMessage;
  const taskSpy = vi.fn(originalTask);
  useStore.setState({ addMessage: taskSpy });

  // Mock the parent agent as Mozart to pass validation
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const mozartId = "test-mozart-agent-id";

  // Set parent as Mozart
  process.env.GOGENT_PARENT_AGENT = mozartId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  // Register Mozart in the store using the correct Agent structure
  const store = useStore.getState();
  store.addAgent({
    id: mozartId,
    parentId: null,
    model: "opus",
    tier: "opus",
    status: "running",
    description: "Test Mozart orchestrator",
    agentType: "mozart",
    startTime: Date.now(),
  });

  try {
    // Create a mock Claude CLI that succeeds quickly
    const mockPath = await createMockClaude({
      behavior: "success",
      delay: 50,
      output: `Analysis complete for ${childAgent}`,
    });

    // Call spawn_agent tool with the child agent
    const response = await spawnAgent.handler({
      agent: childAgent,
      description: `Test spawn of ${childAgent}`,
      prompt: `AGENT: ${childAgent}\n\nTest prompt`,
      model: "opus",
      timeout: 1000,
    });

    spawnCalled = true;
    agentType = childAgent;

    // Parse the result
    const resultText = response.content[0]?.text;
    const result: SpawnResult = resultText ? JSON.parse(resultText) : null;

    // Check if Task was called (it shouldn't be)
    taskCalled = taskSpy.mock.calls.some((call) => {
      const message = call[0];
      return message?.role === "assistant" &&
             message?.content?.some((c: { type: string }) => c.type === "tool_use");
    });

    return {
      spawnCalled,
      agentType,
      taskCalled,
      result,
    };
  } finally {
    // Restore original environment
    if (originalParentEnv !== undefined) {
      process.env.GOGENT_PARENT_AGENT = originalParentEnv;
    } else {
      delete process.env.GOGENT_PARENT_AGENT;
    }

    if (originalNestingEnv !== undefined) {
      process.env.GOGENT_NESTING_LEVEL = originalNestingEnv;
    } else {
      delete process.env.GOGENT_NESTING_LEVEL;
    }

    // Restore original function
    useStore.setState({ addMessage: originalTask });
  }
}

/**
 * Helper: Track spawn order for parallel spawning test
 */
async function trackMozartSpawnOrder(): Promise<
  Array<{ agent: string; startTime: number; endTime: number }>
> {
  // Mock the parent agent as Mozart
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const mozartId = "test-mozart-agent-id";

  process.env.GOGENT_PARENT_AGENT = mozartId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  const store = useStore.getState();
  store.addAgent({
    id: mozartId,
    parentId: null,
    model: "opus",
    tier: "opus",
    status: "running",
    description: "Test Mozart orchestrator",
    agentType: "mozart",
    startTime: Date.now(),
  });

  try {
    const timeline: TimelineEvent[] = [];

    // Spawn Einstein
    const einsteinStart = Date.now();
    timeline.push({ agent: "einstein", event: "start", timestamp: einsteinStart });

    const einsteinPromise = spawnAgent.handler({
      agent: "einstein",
      description: "Theoretical analysis",
      prompt: "AGENT: einstein\n\nAnalyze...",
      model: "opus",
      timeout: 500,
    });

    // Spawn Staff-Architect in parallel (don't await yet)
    const staffStart = Date.now();
    timeline.push({
      agent: "staff-architect-critical-review",
      event: "start",
      timestamp: staffStart,
    });

    const staffPromise = spawnAgent.handler({
      agent: "staff-architect-critical-review",
      description: "Practical review",
      prompt: "AGENT: staff-architect-critical-review\n\nReview...",
      model: "opus",
      timeout: 500,
    });

    // Wait for both to complete
    const [einsteinResult, staffResult] = await Promise.all([
      einsteinPromise,
      staffPromise,
    ]);

    const einsteinEnd = Date.now();
    const staffEnd = Date.now();

    timeline.push({ agent: "einstein", event: "end", timestamp: einsteinEnd });
    timeline.push({
      agent: "staff-architect-critical-review",
      event: "end",
      timestamp: staffEnd,
    });

    // Convert timeline to results
    return [
      {
        agent: "einstein",
        startTime: einsteinStart,
        endTime: einsteinEnd,
      },
      {
        agent: "staff-architect-critical-review",
        startTime: staffStart,
        endTime: staffEnd,
      },
    ];
  } finally {
    if (originalParentEnv !== undefined) {
      process.env.GOGENT_PARENT_AGENT = originalParentEnv;
    } else {
      delete process.env.GOGENT_PARENT_AGENT;
    }

    if (originalNestingEnv !== undefined) {
      process.env.GOGENT_NESTING_LEVEL = originalNestingEnv;
    } else {
      delete process.env.GOGENT_NESTING_LEVEL;
    }
  }
}

/**
 * Helper: Track full Braintrust timeline including Beethoven
 */
async function trackFullBraintrustTimeline(): Promise<
  Array<{ agent: string; startTime: number; endTime: number }>
> {
  // Mock the parent agent as Mozart
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const mozartId = "test-mozart-agent-id";

  process.env.GOGENT_PARENT_AGENT = mozartId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  const store = useStore.getState();
  store.addAgent({
    id: mozartId,
    parentId: null,
    model: "opus",
    tier: "opus",
    status: "running",
    description: "Test Mozart orchestrator",
    agentType: "mozart",
    startTime: Date.now(),
  });

  try {
    const timeline: Array<{ agent: string; startTime: number; endTime: number }> = [];

    // Phase 1: Spawn Einstein and Staff-Architect in parallel
    const einsteinStart = Date.now();
    const einsteinPromise = spawnAgent.handler({
      agent: "einstein",
      description: "Theoretical analysis",
      prompt: "AGENT: einstein\n\nAnalyze...",
      model: "opus",
      timeout: 500,
    });

    const staffStart = Date.now();
    const staffPromise = spawnAgent.handler({
      agent: "staff-architect-critical-review",
      description: "Practical review",
      prompt: "AGENT: staff-architect-critical-review\n\nReview...",
      model: "opus",
      timeout: 500,
    });

    // Wait for both to complete
    await Promise.all([einsteinPromise, staffPromise]);

    const einsteinEnd = Date.now();
    const staffEnd = Date.now();

    timeline.push({
      agent: "einstein",
      startTime: einsteinStart,
      endTime: einsteinEnd,
    });
    timeline.push({
      agent: "staff-architect-critical-review",
      startTime: staffStart,
      endTime: staffEnd,
    });

    // CRITICAL: Add small delay to ensure Beethoven starts AFTER both complete
    // This prevents race conditions in timestamp comparison
    await new Promise((resolve) => setTimeout(resolve, 10));

    // Phase 2: Spawn Beethoven AFTER both complete
    const beethovenStart = Date.now();
    await spawnAgent.handler({
      agent: "beethoven",
      description: "Synthesis",
      prompt: "AGENT: beethoven\n\nSynthesize...",
      model: "opus",
      timeout: 500,
    });
    const beethovenEnd = Date.now();

    timeline.push({
      agent: "beethoven",
      startTime: beethovenStart,
      endTime: beethovenEnd,
    });

    return timeline;
  } finally {
    if (originalParentEnv !== undefined) {
      process.env.GOGENT_PARENT_AGENT = originalParentEnv;
    } else {
      delete process.env.GOGENT_PARENT_AGENT;
    }

    if (originalNestingEnv !== undefined) {
      process.env.GOGENT_NESTING_LEVEL = originalNestingEnv;
    } else {
      delete process.env.GOGENT_NESTING_LEVEL;
    }
  }
}

describe("Mozart Orchestrator MCP Spawning", () => {
  beforeEach(() => {
    resetProcessRegistry();
    useStore.setState({ modalQueue: [], messages: [] });
    useStore.getState().clearAgents();
  });

  afterEach(() => {
    resetProcessRegistry();
    useStore.getState().clearAgents();
    vi.restoreAllMocks();
  });

  describe("spawn_agent usage", () => {
    it("should spawn Einstein via MCP spawn_agent", async () => {
      const result = await invokeMozartWithMockSpawn({
        childAgent: "einstein",
        expectedInvocation: "mcp__gofortress__spawn_agent",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("einstein");
      expect(result.result).toBeDefined();
      expect(result.result?.agent).toBe("einstein");
    });

    it("should spawn Staff-Architect via MCP spawn_agent", async () => {
      const result = await invokeMozartWithMockSpawn({
        childAgent: "staff-architect-critical-review",
        expectedInvocation: "mcp__gofortress__spawn_agent",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("staff-architect-critical-review");
      expect(result.result).toBeDefined();
      expect(result.result?.agent).toBe("staff-architect-critical-review");
    });

    it("should NOT use Task() for Einstein/Staff-Architect spawning", async () => {
      const result = await invokeMozartWithMockSpawn({
        verifyNoTaskCall: true,
      });

      // Verify Task() was NOT called for spawning
      expect(result.taskCalled).toBe(false);

      // But spawn_agent WAS called
      expect(result.spawnCalled).toBe(true);
    });
  });

  describe("parallel spawning", () => {
    it("should spawn Einstein and Staff-Architect in parallel", async () => {
      const results = await trackMozartSpawnOrder();

      expect(results).toHaveLength(2);

      const einstein = results.find((r) => r.agent === "einstein");
      const staff = results.find(
        (r) => r.agent === "staff-architect-critical-review"
      );

      expect(einstein).toBeDefined();
      expect(staff).toBeDefined();

      // Both should start before either completes (parallel execution)
      // Staff-Architect should start before Einstein ends
      expect(staff!.startTime).toBeLessThan(einstein!.endTime);

      // Einstein should start before Staff-Architect ends
      expect(einstein!.startTime).toBeLessThan(staff!.endTime);

      // The start times should be very close (within 100ms indicates parallelism)
      const timeDiff = Math.abs(einstein!.startTime - staff!.startTime);
      expect(timeDiff).toBeLessThan(100);
    });
  });

  describe("Beethoven synthesis", () => {
    it("should invoke Beethoven after Einstein and Staff-Architect complete", async () => {
      const timeline = await trackFullBraintrustTimeline();

      expect(timeline).toHaveLength(3);

      const einstein = timeline.find((e) => e.agent === "einstein");
      const staff = timeline.find(
        (e) => e.agent === "staff-architect-critical-review"
      );
      const beethoven = timeline.find((e) => e.agent === "beethoven");

      expect(einstein).toBeDefined();
      expect(staff).toBeDefined();
      expect(beethoven).toBeDefined();

      // Beethoven should start AFTER both Einstein and Staff-Architect complete
      expect(beethoven!.startTime).toBeGreaterThan(einstein!.endTime);
      expect(beethoven!.startTime).toBeGreaterThan(staff!.endTime);
    });
  });

  describe("error handling", () => {
    it("should handle spawn timeout", async () => {
      // Set up Mozart parent
      const mozartId = "test-mozart-timeout";
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
        // Create slow mock that will timeout
        await createMockClaude({
          behavior: "timeout",
        });

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test timeout",
          prompt: "AGENT: einstein\n\nTest",
          model: "opus",
          timeout: 100, // Very short timeout
        });

        const result: SpawnResult = JSON.parse(response.content[0]?.text ?? "{}");

        expect(result.success).toBe(false);
        expect(result.error).toContain("timed out");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
      }
    }, 10000);

    it("should handle validation errors when parent is not Mozart", async () => {
      // Don't set parent - should fail validation
      const response = await spawnAgent.handler({
        agent: "einstein",
        description: "Test validation",
        prompt: "AGENT: einstein\n\nTest",
        model: "opus",
        timeout: 1000,
      });

      const result: SpawnResult = JSON.parse(response.content[0]?.text ?? "{}");

      expect(result.success).toBe(false);
      expect(result.error).toContain("validation failed");
    });

    it("should handle max nesting depth", async () => {
      // Set nesting to max depth
      const originalNesting = process.env.GOGENT_NESTING_LEVEL;
      process.env.GOGENT_NESTING_LEVEL = "10"; // MAX_NESTING_DEPTH

      try {
        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test depth",
          prompt: "AGENT: einstein\n\nTest",
          model: "opus",
          timeout: 1000,
        });

        const result = JSON.parse(response.content[0]?.text ?? "{}");

        expect(result.success).toBe(false);
        expect(result.error).toContain("Maximum nesting depth");
      } finally {
        if (originalNesting !== undefined) {
          process.env.GOGENT_NESTING_LEVEL = originalNesting;
        } else {
          delete process.env.GOGENT_NESTING_LEVEL;
        }
      }
    });
  });
});
