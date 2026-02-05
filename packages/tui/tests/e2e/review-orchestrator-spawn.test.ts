/**
 * E2E Tests: Review-Orchestrator MCP Spawning
 *
 * Tests the review-orchestrator workflow where it spawns multiple specialized
 * reviewers (backend, frontend, standards, architect) in parallel via the MCP
 * spawn_agent tool.
 *
 * Source: MCP-SPAWN-011
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { resetProcessRegistry } from "../../src/spawn/processRegistry.js";
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
 * Result from invoking review-orchestrator with mock spawns
 */
interface MockReviewResult {
  spawnCalled: boolean;
  agentType?: string;
  result?: SpawnResult;
}

/**
 * Options for mock review-orchestrator invocation
 */
interface MockReviewOptions {
  childAgent?: string;
  expectedInvocation?: string;
}

/**
 * Partial failure test result
 */
interface PartialFailureResult {
  collectedCount: number;
  failedCount: number;
  overallSuccess: boolean;
  findings: Record<string, unknown>;
}

/**
 * Full workflow result
 */
interface FullWorkflowResult {
  findings: {
    backend?: unknown;
    frontend?: unknown;
    standards?: unknown;
    architect?: unknown;
  };
  synthesisComplete: boolean;
}

/**
 * Create a mock child process that simulates Claude CLI
 */
function createMockChildProcess(success: boolean, output: string, delay = 50): Partial<ChildProcess> {
  const proc = new EventEmitter() as Partial<ChildProcess>;

  // Mock stdin, stdout, stderr as EventEmitters with write/end methods
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
      stdout.emit("data", Buffer.from(output));
    } else {
      stderr.emit("data", Buffer.from(output));
    }

    setTimeout(() => {
      (proc as EventEmitter).emit("close", success ? 0 : 1);
      (proc as EventEmitter).emit("exit", success ? 0 : 1);
    }, 10);
  }, delay);

  return proc;
}

/**
 * Helper: Invoke spawn_agent with mocks and track invocation
 */
async function invokeReviewOrchestratorWithMockSpawn(
  opts: MockReviewOptions
): Promise<MockReviewResult> {
  const childAgent = opts.childAgent || "backend-reviewer";
  let spawnCalled = false;
  let agentType: string | undefined;

  // Mock the parent agent as review-orchestrator to pass validation
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const reviewOrchId = "test-review-orchestrator-id";

  // Set parent as review-orchestrator
  process.env.GOGENT_PARENT_AGENT = reviewOrchId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  // Register review-orchestrator in the store using the correct Agent structure
  const store = useStore.getState();
  store.addAgent({
    id: reviewOrchId,
    parentId: null,
    model: "sonnet",
    tier: "sonnet",
    status: "running",
    description: "Test review-orchestrator",
    agentType: "review-orchestrator",
    startTime: Date.now(),
  });

  // Mock child_process.spawn
  const mockOutput = JSON.stringify({
    type: "result",
    subtype: "success",
    cost_usd: 0.001,
    total_cost_usd: 0.001,
    duration_ms: 50,
    num_turns: 1,
    result: JSON.stringify({
      findings: [
        {
          severity: "warning",
          file: "src/test.ts",
          line: 42,
          message: "Test finding from mock reviewer",
        },
      ],
    }),
    session_id: "mock-session-123",
  });

  const spawnMock = vi.mocked(cp.spawn);
  spawnMock.mockImplementation(() => {
    return createMockChildProcess(true, mockOutput) as any;
  });

  try {
    // Import spawnAgent AFTER mocking to ensure it uses the mocked spawn
    const { spawnAgent } = await import("../../src/mcp/tools/spawnAgent.js");

    // Call spawn_agent tool with the child agent
    const response = await spawnAgent.handler({
      agent: childAgent,
      description: `Test spawn of ${childAgent}`,
      prompt: `AGENT: ${childAgent}\n\nTest review prompt`,
      model: "haiku",
      timeout: 1000,
    });

    spawnCalled = true;
    agentType = childAgent;

    // Parse the result
    const resultText = response.content[0]?.text;
    const result: SpawnResult = resultText ? JSON.parse(resultText) : null;

    return {
      spawnCalled,
      agentType,
      result,
    };
  } finally {
    // Clear spawn mock
    spawnMock.mockClear();

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
    useStore.getState().clearAgents();
  }
}

/**
 * Helper: Track spawn order for parallel spawning test
 */
async function trackReviewerSpawnOrder(): Promise<
  Array<{ agent: string; startTime: number; endTime: number }>
> {
  // Mock the parent agent as review-orchestrator
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const reviewOrchId = "test-review-orchestrator-id";

  process.env.GOGENT_PARENT_AGENT = reviewOrchId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  const store = useStore.getState();
  store.addAgent({
    id: reviewOrchId,
    parentId: null,
    model: "sonnet",
    tier: "sonnet",
    status: "running",
    description: "Test review-orchestrator",
    agentType: "review-orchestrator",
    startTime: Date.now(),
  });

  // Mock child_process.spawn
  const mockOutput = JSON.stringify({
    type: "result",
    subtype: "success",
    cost_usd: 0.001,
    total_cost_usd: 0.001,
    duration_ms: 50,
    num_turns: 1,
    result: "Mock review complete",
    session_id: "mock-session-123",
  });

  const spawnMock = vi.mocked(cp.spawn);
  spawnMock.mockImplementation(() => {
    return createMockChildProcess(true, mockOutput) as any;
  });

  // Import spawnAgent AFTER mocking
  const { spawnAgent } = await import("../../src/mcp/tools/spawnAgent.js");

  try {
    // Spawn backend-reviewer
    const backendStart = Date.now();
    const backendPromise = spawnAgent.handler({
      agent: "backend-reviewer",
      description: "Backend review",
      prompt: "AGENT: backend-reviewer\n\nReview...",
      model: "haiku",
      timeout: 500,
    });

    // Spawn frontend-reviewer in parallel (don't await yet)
    const frontendStart = Date.now();
    const frontendPromise = spawnAgent.handler({
      agent: "frontend-reviewer",
      description: "Frontend review",
      prompt: "AGENT: frontend-reviewer\n\nReview...",
      model: "haiku",
      timeout: 500,
    });

    // Spawn standards-reviewer in parallel
    const standardsStart = Date.now();
    const standardsPromise = spawnAgent.handler({
      agent: "standards-reviewer",
      description: "Standards review",
      prompt: "AGENT: standards-reviewer\n\nReview...",
      model: "haiku",
      timeout: 500,
    });

    // Wait for all to complete
    const [backendResult, frontendResult, standardsResult] = await Promise.all([
      backendPromise,
      frontendPromise,
      standardsPromise,
    ]);

    const backendEnd = Date.now();
    const frontendEnd = Date.now();
    const standardsEnd = Date.now();

    // Convert to results
    return [
      {
        agent: "backend-reviewer",
        startTime: backendStart,
        endTime: backendEnd,
      },
      {
        agent: "frontend-reviewer",
        startTime: frontendStart,
        endTime: frontendEnd,
      },
      {
        agent: "standards-reviewer",
        startTime: standardsStart,
        endTime: standardsEnd,
      },
    ];
  } finally {
    spawnMock.mockClear();

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
 * Helper: Invoke review-orchestrator with partial failure
 */
async function invokeReviewOrchestratorWithFailure(opts: {
  failingAgent: string;
  workingAgents: string[];
}): Promise<PartialFailureResult> {
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const reviewOrchId = "test-review-orchestrator-id";

  process.env.GOGENT_PARENT_AGENT = reviewOrchId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  const store = useStore.getState();
  store.addAgent({
    id: reviewOrchId,
    parentId: null,
    model: "sonnet",
    tier: "sonnet",
    status: "running",
    description: "Test review-orchestrator",
    agentType: "review-orchestrator",
    startTime: Date.now(),
  });

  // Mock spawn - working agents succeed, failing agent fails
  const successOutput = JSON.stringify({
    type: "result",
    subtype: "success",
    cost_usd: 0.001,
    total_cost_usd: 0.001,
    duration_ms: 50,
    num_turns: 1,
    result: "Mock review complete",
    session_id: "mock-session-123",
  });

  const spawnMock = vi.mocked(cp.spawn);
  let callCount = 0;
  spawnMock.mockImplementation(() => {
    callCount++;
    // Last call is the failing agent
    const isFailingAgent = callCount > opts.workingAgents.length;
    if (isFailingAgent) {
      // Return a process that times out (delays longer than timeout)
      return createMockChildProcess(false, "Timeout", 200) as any;
    }
    return createMockChildProcess(true, successOutput) as any;
  });

  // Import spawnAgent AFTER mocking
  const { spawnAgent } = await import("../../src/mcp/tools/spawnAgent.js");

  try {
    const promises: Promise<unknown>[] = [];
    let collectedCount = 0;
    let failedCount = 0;
    const findings: Record<string, unknown> = {};

    // Spawn working agents
    for (const agent of opts.workingAgents) {
      const promise = spawnAgent.handler({
        agent,
        description: `Review by ${agent}`,
        prompt: `AGENT: ${agent}\n\nReview...`,
        model: "haiku",
        timeout: 500,
      }).then((result) => {
        const resultData: SpawnResult = JSON.parse(result.content[0]?.text ?? "{}");
        if (resultData.success) {
          collectedCount++;
          findings[agent] = { findings: ["mock finding"] };
        }
      });
      promises.push(promise);
    }

    // Spawn failing agent
    const failPromise = spawnAgent.handler({
      agent: opts.failingAgent,
      description: `Review by ${opts.failingAgent}`,
      prompt: `AGENT: ${opts.failingAgent}\n\nReview...`,
      model: "haiku",
      timeout: 100, // Very short timeout to force failure
    }).then((result) => {
      const resultData: SpawnResult = JSON.parse(result.content[0]?.text ?? "{}");
      if (!resultData.success) {
        failedCount++;
      }
    });
    promises.push(failPromise);

    // Wait for all (some may fail)
    await Promise.allSettled(promises);

    return {
      collectedCount,
      failedCount,
      overallSuccess: collectedCount > 0,
      findings,
    };
  } finally {
    spawnMock.mockClear();

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
 * Helper: Invoke full review workflow
 */
async function invokeFullReviewWorkflow(): Promise<FullWorkflowResult> {
  const originalParentEnv = process.env.GOGENT_PARENT_AGENT;
  const originalNestingEnv = process.env.GOGENT_NESTING_LEVEL;
  const reviewOrchId = "test-review-orchestrator-id";

  process.env.GOGENT_PARENT_AGENT = reviewOrchId;
  process.env.GOGENT_NESTING_LEVEL = "1";

  const store = useStore.getState();
  store.addAgent({
    id: reviewOrchId,
    parentId: null,
    model: "sonnet",
    tier: "sonnet",
    status: "running",
    description: "Test review-orchestrator",
    agentType: "review-orchestrator",
    startTime: Date.now(),
  });

  // Mock spawn
  const mockOutput = JSON.stringify({
    type: "result",
    subtype: "success",
    cost_usd: 0.001,
    total_cost_usd: 0.001,
    duration_ms: 50,
    num_turns: 1,
    result: "Mock review complete",
    session_id: "mock-session-123",
  });

  const spawnMock = vi.mocked(cp.spawn);
  spawnMock.mockImplementation(() => {
    return createMockChildProcess(true, mockOutput) as any;
  });

  // Import spawnAgent AFTER mocking
  const { spawnAgent } = await import("../../src/mcp/tools/spawnAgent.js");

  try {
    const findings: FullWorkflowResult["findings"] = {};

    // Spawn all reviewers in parallel
    const reviewers = [
      "backend-reviewer",
      "frontend-reviewer",
      "standards-reviewer",
      "architect-reviewer",
    ];

    const promises = reviewers.map(async (agent) => {
      const result = await spawnAgent.handler({
        agent,
        description: `Review by ${agent}`,
        prompt: `AGENT: ${agent}\n\nReview...`,
        model: agent === "architect-reviewer" ? "sonnet" : "haiku",
        timeout: 1000,
      });

      const resultData: SpawnResult = JSON.parse(result.content[0]?.text ?? "{}");
      if (resultData.success) {
        findings[agent.replace("-reviewer", "") as keyof typeof findings] = {
          findings: ["mock finding"],
        };
      }
    });

    // Wait for all reviewers
    await Promise.all(promises);

    // Check if synthesis would be complete (all reviewers returned)
    const synthesisComplete = Object.keys(findings).length === reviewers.length;

    return {
      findings,
      synthesisComplete,
    };
  } finally {
    spawnMock.mockClear();

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

describe("Review-Orchestrator MCP Spawning", () => {
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

  describe("parallel reviewer spawning", () => {
    it("should spawn backend-reviewer via MCP spawn_agent", async () => {
      const result = await invokeReviewOrchestratorWithMockSpawn({
        childAgent: "backend-reviewer",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("backend-reviewer");
      expect(result.result).toBeDefined();
      expect(result.result?.success).toBe(true);
    });

    it("should spawn frontend-reviewer via MCP spawn_agent", async () => {
      const result = await invokeReviewOrchestratorWithMockSpawn({
        childAgent: "frontend-reviewer",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("frontend-reviewer");
      expect(result.result).toBeDefined();
      expect(result.result?.success).toBe(true);
    });

    it("should spawn all reviewers in parallel", async () => {
      const timeline = await trackReviewerSpawnOrder();

      expect(timeline).toHaveLength(3);

      const backend = timeline.find((r) => r.agent === "backend-reviewer");
      const frontend = timeline.find((r) => r.agent === "frontend-reviewer");
      const standards = timeline.find((r) => r.agent === "standards-reviewer");

      expect(backend).toBeDefined();
      expect(frontend).toBeDefined();
      expect(standards).toBeDefined();

      // All reviewers should start within 100ms of each other (parallel execution)
      const startTimes = timeline.map((r) => r.startTime);
      const maxDiff = Math.max(...startTimes) - Math.min(...startTimes);

      expect(maxDiff).toBeLessThan(100);
    });
  });

  describe("partial failure handling", () => {
    it("should continue if one reviewer fails", async () => {
      const result = await invokeReviewOrchestratorWithFailure({
        failingAgent: "frontend-reviewer",
        workingAgents: ["backend-reviewer", "standards-reviewer"],
      });

      // Should still collect results from working reviewers
      expect(result.collectedCount).toBeGreaterThanOrEqual(1);
      expect(result.failedCount).toBeGreaterThanOrEqual(1);
      expect(result.overallSuccess).toBe(true);

      // Should have findings from working reviewers
      expect(Object.keys(result.findings).length).toBeGreaterThan(0);
    });
  });

  describe("findings collection", () => {
    it("should collect and synthesize all reviewer findings", async () => {
      const result = await invokeFullReviewWorkflow();

      expect(result.findings).toBeDefined();
      expect(result.findings.backend).toBeDefined();
      expect(result.findings.frontend).toBeDefined();
      expect(result.findings.standards).toBeDefined();
      expect(result.findings.architect).toBeDefined();
      expect(result.synthesisComplete).toBe(true);
    });
  });
});
