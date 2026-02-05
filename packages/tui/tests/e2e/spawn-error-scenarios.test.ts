/**
 * E2E Tests: Spawn Agent Error Scenarios
 *
 * Comprehensive error handling tests for spawn_agent MCP tool.
 * Tests timeouts, validation failures, nesting violations, and cleanup.
 *
 * Source: MCP-SPAWN-012
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
 * Create a mock child process that times out (hangs forever)
 */
function createTimeoutMockProcess(): Partial<ChildProcess> {
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
  proc.kill = vi.fn((signal?: string) => {
    proc.killed = true;
    // Simulate process exit after kill
    setTimeout(() => {
      (proc as EventEmitter).emit("close", null, signal);
    }, 10);
    return true;
  });

  // Process never completes on its own - will be killed by timeout
  return proc;
}

/**
 * Create a mock child process that returns invalid JSON
 */
function createInvalidJsonMockProcess(): Partial<ChildProcess> {
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
  proc.kill = vi.fn(() => true);

  setTimeout(() => {
    stdout.emit("data", Buffer.from("This is not valid JSON {{{{"));
    setTimeout(() => {
      (proc as EventEmitter).emit("close", 0);
    }, 10);
  }, 50);

  return proc;
}

/**
 * Create a mock child process that exits with error code
 */
function createErrorMockProcess(errorMessage: string): Partial<ChildProcess> {
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
  proc.kill = vi.fn(() => true);

  setTimeout(() => {
    stderr.emit("data", Buffer.from(errorMessage));
    setTimeout(() => {
      (proc as EventEmitter).emit("close", 1); // Non-zero exit code
    }, 10);
  }, 50);

  return proc;
}

describe("Spawn Agent Error Scenarios", () => {
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

  describe("timeout handling", () => {
    it("should timeout after configured duration", async () => {
      vi.useFakeTimers();

      const mozartId = "test-mozart-timeout";
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
          return createTimeoutMockProcess() as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        // Start spawn with short timeout
        const promise = spawnAgent.handler({
          agent: "einstein",
          description: "Test timeout",
          prompt: "AGENT: einstein\n\nTest",
          model: "opus",
          timeout: 100, // Very short timeout
        });

        // Advance timers to trigger timeout
        await vi.advanceTimersByTimeAsync(100);

        const response = await promise;
        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("timed out");
        expect(result.error).toContain("100ms");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
        vi.useRealTimers();
      }
    }, 10000);

    it("should clean up processes on timeout", async () => {
      vi.useFakeTimers();

      const mozartId = "test-mozart-cleanup";
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
        let mockProc: Partial<ChildProcess> | null = null;
        const spawnMock = vi.mocked(cp.spawn);
        spawnMock.mockImplementation(() => {
          mockProc = createTimeoutMockProcess();
          return mockProc as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const promise = spawnAgent.handler({
          agent: "einstein",
          description: "Test cleanup",
          prompt: "Test",
          model: "opus",
          timeout: 100,
        });

        await vi.advanceTimersByTimeAsync(100);
        await promise;

        // Verify kill was called
        expect(mockProc?.kill).toHaveBeenCalled();
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
        vi.useRealTimers();
      }
    }, 10000);
  });

  describe("validation errors", () => {
    it("should fail when parent is not an allowed spawner", async () => {
      // Set parent as python-pro (not allowed to spawn Einstein)
      const pythonProId = "test-python-pro";
      process.env.GOGENT_PARENT_AGENT = pythonProId;
      process.env.GOGENT_NESTING_LEVEL = "1";

      useStore.getState().addAgent({
        id: pythonProId,
        parentId: null,
        model: "sonnet",
        tier: "sonnet",
        status: "running",
        description: "Test Python Pro",
        agentType: "python-pro", // Not in Mozart's allowed list
        startTime: Date.now(),
      });

      try {
        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test validation",
          prompt: "AGENT: einstein\n\nTest",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("validation failed");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
      }
    });

    it("should fail when parent is not set", async () => {
      // No parent set - invalid
      const { spawnAgent } = await import("../../src/mcp/tools/spawnAgent.js");

      const response = await spawnAgent.handler({
        agent: "einstein",
        description: "Test no parent",
        prompt: "AGENT: einstein\n\nTest",
        model: "opus",
        timeout: 1000,
      });

      const result: SpawnResult = JSON.parse(
        response.content[0]?.text ?? "{}"
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("validation failed");
    });

    it("should fail when nesting level exceeds maximum", async () => {
      const originalNesting = process.env.GOGENT_NESTING_LEVEL;
      process.env.GOGENT_NESTING_LEVEL = "10"; // MAX_NESTING_DEPTH

      try {
        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

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
        expect(result.errorCode).toBe("E_MAX_DEPTH_EXCEEDED");
      } finally {
        if (originalNesting !== undefined) {
          process.env.GOGENT_NESTING_LEVEL = originalNesting;
        } else {
          delete process.env.GOGENT_NESTING_LEVEL;
        }
      }
    });
  });

  describe("CLI errors", () => {
    it("should propagate CLI errors to parent", async () => {
      const mozartId = "test-mozart-cli-error";
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
          return createErrorMockProcess("Rate limit exceeded") as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test CLI error",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("Rate limit exceeded");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should handle invalid JSON in CLI output", async () => {
      const mozartId = "test-mozart-invalid-json";
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
          return createInvalidJsonMockProcess() as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test invalid JSON",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        // Should succeed but with unparsed output
        expect(result.success).toBe(true);
        expect(result.output).toContain("This is not valid JSON");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should handle process spawn errors", async () => {
      const mozartId = "test-mozart-spawn-error";
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

          proc.stdin = stdin;
          proc.stdout = new EventEmitter() as any;
          proc.stderr = new EventEmitter() as any;
          proc.kill = vi.fn(() => true);

          // Emit error event
          setTimeout(() => {
            (proc as EventEmitter).emit(
              "error",
              new Error("ENOENT: claude command not found")
            );
          }, 10);

          return proc as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test spawn error",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("Spawn error");
        expect(result.error).toContain("ENOENT");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });
  });

  describe("edge cases", () => {
    it("should handle large output with truncation", async () => {
      const mozartId = "test-mozart-truncation";
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
            // Emit 11MB of data (exceeds 10MB buffer limit)
            const largeChunk = "x".repeat(11 * 1024 * 1024);
            stdout.emit("data", Buffer.from(largeChunk));

            setTimeout(() => {
              (proc as EventEmitter).emit("close", 0);
            }, 10);
          }, 50);

          return proc as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        const response = await spawnAgent.handler({
          agent: "einstein",
          description: "Test truncation",
          prompt: "Test",
          model: "opus",
          timeout: 1000,
        });

        const result: SpawnResult = JSON.parse(
          response.content[0]?.text ?? "{}"
        );

        expect(result.truncated).toBe(true);
        expect(result.output).toContain("OUTPUT TRUNCATED");
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });

    it("should handle concurrent spawn failures", async () => {
      const mozartId = "test-mozart-concurrent-fail";
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
          return createErrorMockProcess("Concurrent spawn error") as any;
        });

        const { spawnAgent } = await import(
          "../../src/mcp/tools/spawnAgent.js"
        );

        // Spawn multiple agents that will all fail
        const promises = [
          spawnAgent.handler({
            agent: "einstein",
            description: "Test 1",
            prompt: "Test",
            model: "opus",
            timeout: 1000,
          }),
          spawnAgent.handler({
            agent: "staff-architect-critical-review",
            description: "Test 2",
            prompt: "Test",
            model: "opus",
            timeout: 1000,
          }),
          spawnAgent.handler({
            agent: "beethoven",
            description: "Test 3",
            prompt: "Test",
            model: "opus",
            timeout: 1000,
          }),
        ];

        const results = await Promise.all(promises);

        // All should fail
        for (const response of results) {
          const result: SpawnResult = JSON.parse(
            response.content[0]?.text ?? "{}"
          );
          expect(result.success).toBe(false);
        }
      } finally {
        delete process.env.GOGENT_PARENT_AGENT;
        delete process.env.GOGENT_NESTING_LEVEL;
        vi.mocked(cp.spawn).mockClear();
      }
    });
  });
});
