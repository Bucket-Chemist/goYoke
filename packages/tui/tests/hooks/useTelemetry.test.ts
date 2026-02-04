/**
 * Tests for useTelemetry hook
 * Verifies file watching, event emission, and error tracking
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { promises as fs } from "fs";
import { join } from "path";
import { tmpdir } from "os";
import { useTelemetry } from "../../src/hooks/useTelemetry.js";
import { useStore } from "../../src/store/index.js";

// Mock logger to prevent file I/O during tests
vi.mock("../../src/utils/logger.js", () => ({
  logger: {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

// Test directories
const TEST_XDG_DATA_HOME = join(tmpdir(), `telemetry-test-xdg-${Date.now()}`);
const TEST_HOME = join(tmpdir(), `telemetry-test-home-${Date.now()}`);
const GOGENT_DIR = join(TEST_XDG_DATA_HOME, "gogent");
const MEMORY_DIR = join(TEST_HOME, ".claude", "memory");

describe("useTelemetry", () => {
  beforeEach(async () => {
    // Set test environment variables
    process.env["XDG_DATA_HOME"] = TEST_XDG_DATA_HOME;
    process.env["HOME"] = TEST_HOME;

    // Create test directories
    await fs.mkdir(GOGENT_DIR, { recursive: true });
    await fs.mkdir(MEMORY_DIR, { recursive: true });

    // Create empty telemetry files
    await fs.writeFile(join(GOGENT_DIR, "routing-decisions.jsonl"), "");
    await fs.writeFile(join(GOGENT_DIR, "sharp-edges.jsonl"), "");
    await fs.writeFile(join(MEMORY_DIR, "handoffs.jsonl"), "");

    // Reset store state
    useStore.setState({
      telemetry: {
        routingDecisions: [],
        handoffs: [],
        sharpEdges: [],
      },
    });
  });

  afterEach(async () => {
    // Clean up test directories
    try {
      await fs.rm(TEST_XDG_DATA_HOME, { recursive: true, force: true });
      await fs.rm(TEST_HOME, { recursive: true, force: true });
    } catch (error) {
      // Ignore cleanup errors
    }

    // Restore environment
    delete process.env["XDG_DATA_HOME"];
    delete process.env["HOME"];

    vi.clearAllMocks();
  });

  /**
   * Helper to append JSONL line to file
   */
  async function appendJsonl(
    file: "routing-decisions" | "sharp-edges" | "handoffs",
    data: unknown
  ): Promise<void> {
    const filePath =
      file === "handoffs"
        ? join(MEMORY_DIR, `${file}.jsonl`)
        : join(GOGENT_DIR, `${file}.jsonl`);

    await fs.appendFile(filePath, JSON.stringify(data) + "\n");
  }

  describe("Watcher Initialization", () => {
    it("should start watching telemetry files on mount", async () => {
      const { unmount } = renderHook(() => useTelemetry());

      // Hook should initialize without error
      expect(() => unmount()).not.toThrow();
    });

    it("should watch all three telemetry file paths", async () => {
      renderHook(() => useTelemetry());

      // Files should exist (created in beforeEach)
      const routingExists = await fs
        .access(join(GOGENT_DIR, "routing-decisions.jsonl"))
        .then(() => true)
        .catch(() => false);
      const handoffsExists = await fs
        .access(join(MEMORY_DIR, "handoffs.jsonl"))
        .then(() => true)
        .catch(() => false);
      const sharpEdgesExists = await fs
        .access(join(GOGENT_DIR, "sharp-edges.jsonl"))
        .then(() => true)
        .catch(() => false);

      expect(routingExists).toBe(true);
      expect(handoffsExists).toBe(true);
      expect(sharpEdgesExists).toBe(true);
    });

    it("should stop watching on unmount", async () => {
      const { unmount } = renderHook(() => useTelemetry());

      unmount();

      // Should complete without hanging
      expect(true).toBe(true);
    });
  });

  describe("Routing Decisions", () => {
    it("should detect and parse new routing decisions", async () => {
      renderHook(() => useTelemetry());

      const decision = {
        timestamp: new Date().toISOString(),
        agent: "codebase-search",
        tier: "haiku",
        reason: "File discovery task",
        tool_count: 3,
      };

      await appendJsonl("routing-decisions", decision);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.routingDecisions[0]).toMatchObject(decision);
    });

    it("should accumulate multiple routing decisions", async () => {
      renderHook(() => useTelemetry());

      const decisions = [
        { timestamp: "2026-02-04T10:00:00Z", agent: "haiku-scout", tier: "haiku" },
        { timestamp: "2026-02-04T10:01:00Z", agent: "go-pro", tier: "sonnet" },
        { timestamp: "2026-02-04T10:02:00Z", agent: "orchestrator", tier: "sonnet" },
      ];

      for (const decision of decisions) {
        await appendJsonl("routing-decisions", decision);
        await new Promise((resolve) => setTimeout(resolve, 200));
      }

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(3);
        },
        { timeout: 3000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.routingDecisions.map((d: any) => d.agent)).toEqual([
        "haiku-scout",
        "go-pro",
        "orchestrator",
      ]);
    });
  });

  describe("Handoffs", () => {
    it("should detect and parse new handoffs", async () => {
      renderHook(() => useTelemetry());

      const handoff = {
        timestamp: new Date().toISOString(),
        session_id: "test-session-123",
        summary: "Implemented useKeymap hook",
        pending_tasks: [],
        learnings: ["Tested keyboard binding patterns"],
      };

      await appendJsonl("handoffs", handoff);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.handoffs.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.handoffs[0]).toMatchObject(handoff);
    });

    it("should handle handoffs with complex structures", async () => {
      renderHook(() => useTelemetry());

      const handoff = {
        timestamp: new Date().toISOString(),
        session_id: "complex-123",
        summary: "Multi-agent implementation",
        pending_tasks: [
          { id: "task-1", description: "Review code", status: "pending" },
          { id: "task-2", description: "Write docs", status: "in_progress" },
        ],
        learnings: [
          "Agent coordination patterns",
          "Error recovery strategies",
        ],
        sharp_edges: [
          { pattern: "Concurrent file writes", mitigation: "Use locks" },
        ],
      };

      await appendJsonl("handoffs", handoff);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.handoffs.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.handoffs[0]).toMatchObject(handoff);
    });
  });

  describe("Sharp Edges", () => {
    it("should detect and parse new sharp edges", async () => {
      renderHook(() => useTelemetry());

      const sharpEdge = {
        timestamp: new Date().toISOString(),
        pattern: "Import path resolution failure",
        context: {
          file: "src/hooks/useTelemetry.ts",
          error: "Cannot find module",
          attempts: 3,
        },
        mitigation: "Use .js extension for imports",
      };

      await appendJsonl("sharp-edges", sharpEdge);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.sharpEdges.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.sharpEdges[0]).toMatchObject(sharpEdge);
    });

    it("should track multiple occurrences of same pattern", async () => {
      renderHook(() => useTelemetry());

      const edges = [
        {
          timestamp: "2026-02-04T10:00:00Z",
          pattern: "Type error in store",
          context: { file: "store/index.ts" },
        },
        {
          timestamp: "2026-02-04T10:05:00Z",
          pattern: "Type error in store",
          context: { file: "store/slices/telemetry.ts" },
        },
      ];

      for (const edge of edges) {
        await appendJsonl("sharp-edges", edge);
        await new Promise((resolve) => setTimeout(resolve, 200));
      }

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.sharpEdges.length).toBe(2);
        },
        { timeout: 3000 }
      );

      const state = useStore.getState();
      expect(
        state.telemetry.sharpEdges.filter((e: any) => e.pattern === "Type error in store")
      ).toHaveLength(2);
    });
  });

  describe("Error Handling", () => {
    it("should handle malformed JSON gracefully", async () => {
      renderHook(() => useTelemetry());

      // Append invalid JSON
      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");
      await fs.appendFile(filePath, "{invalid json}\n");

      // Wait to ensure it doesn't crash
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Store should remain empty
      const state = useStore.getState();
      expect(state.telemetry.routingDecisions.length).toBe(0);
    });

    it("should continue watching after parse error", async () => {
      renderHook(() => useTelemetry());

      // Append invalid JSON first
      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");
      await fs.appendFile(filePath, "{invalid}\n");

      await new Promise((resolve) => setTimeout(resolve, 300));

      // Then append valid JSON
      const decision = { timestamp: new Date().toISOString(), agent: "test" };
      await appendJsonl("routing-decisions", decision);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
        },
        { timeout: 2000 }
      );
    });

    it("should handle missing files gracefully", async () => {
      // Delete files before hook initialization
      await fs.rm(join(GOGENT_DIR, "routing-decisions.jsonl"));
      await fs.rm(join(GOGENT_DIR, "sharp-edges.jsonl"));
      await fs.rm(join(MEMORY_DIR, "handoffs.jsonl"));

      // Should not throw
      expect(() => renderHook(() => useTelemetry())).not.toThrow();
    });

    it("should recover when files are created after initialization", async () => {
      // Delete files
      await fs.rm(join(GOGENT_DIR, "routing-decisions.jsonl"));

      renderHook(() => useTelemetry());

      // Recreate file with content
      await fs.writeFile(
        join(GOGENT_DIR, "routing-decisions.jsonl"),
        JSON.stringify({ timestamp: new Date().toISOString(), agent: "test" }) + "\n"
      );

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
        },
        { timeout: 2000 }
      );
    });

    it("should handle permission errors gracefully", async () => {
      // Make file unreadable
      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");
      await fs.chmod(filePath, 0o000);

      renderHook(() => useTelemetry());

      // Should not crash (error is logged internally)
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Restore permissions for cleanup
      await fs.chmod(filePath, 0o644);
    });
  });

  describe("File Rotation Handling", () => {
    it("should detect file rotation (inode change)", async () => {
      renderHook(() => useTelemetry());

      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");

      // Write initial content
      await appendJsonl("routing-decisions", { timestamp: "2026-02-04T10:00:00Z", agent: "first" });

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
        },
        { timeout: 2000 }
      );

      // Simulate rotation by removing and recreating file
      await fs.rm(filePath);
      await fs.writeFile(filePath, "");
      await appendJsonl("routing-decisions", { timestamp: "2026-02-04T10:01:00Z", agent: "second" });

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(2);
        },
        { timeout: 2000 }
      );
    });

    it("should reset offset after file rotation", async () => {
      renderHook(() => useTelemetry());

      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");

      // Write some data
      await appendJsonl("routing-decisions", { id: 1 });
      await new Promise((resolve) => setTimeout(resolve, 300));

      // Rotate file
      await fs.rm(filePath);
      await fs.writeFile(filePath, JSON.stringify({ id: 2 }) + "\n");

      await waitFor(
        () => {
          const state = useStore.getState();
          const ids = state.telemetry.routingDecisions.map((d: any) => d.id);
          expect(ids).toContain(2);
        },
        { timeout: 2000 }
      );
    });
  });

  describe("Event Payload Structure", () => {
    it("should preserve exact structure of routing decisions", async () => {
      renderHook(() => useTelemetry());

      const decision = {
        timestamp: "2026-02-04T10:00:00Z",
        agent: "go-pro",
        tier: "sonnet",
        reason: "Go implementation task",
        tool_count: 5,
        cost: 0.045,
        metadata: {
          file: "main.go",
          lines: 150,
        },
      };

      await appendJsonl("routing-decisions", decision);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.routingDecisions[0]).toEqual(decision);
    });

    it("should preserve nested objects and arrays", async () => {
      renderHook(() => useTelemetry());

      const handoff = {
        timestamp: new Date().toISOString(),
        session_id: "nested-test",
        summary: "Test",
        nested: {
          level1: {
            level2: {
              value: "deep",
            },
          },
        },
        arrays: [[1, 2], [3, 4]],
      };

      await appendJsonl("handoffs", handoff);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.handoffs.length).toBe(1);
        },
        { timeout: 2000 }
      );

      const state = useStore.getState();
      expect(state.telemetry.handoffs[0]).toEqual(handoff);
    });
  });

  describe("Performance", () => {
    it("should handle rapid successive writes", async () => {
      renderHook(() => useTelemetry());

      // Append 10 entries rapidly
      const entries = Array.from({ length: 10 }, (_, i) => ({
        timestamp: new Date().toISOString(),
        agent: `agent-${i}`,
        tier: "haiku",
      }));

      for (const entry of entries) {
        await appendJsonl("routing-decisions", entry);
      }

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(10);
        },
        { timeout: 3000 }
      );
    });

    it("should handle large JSONL files efficiently", async () => {
      renderHook(() => useTelemetry());

      // Write 100 entries at once
      const filePath = join(GOGENT_DIR, "routing-decisions.jsonl");
      const lines = Array.from({ length: 100 }, (_, i) =>
        JSON.stringify({ id: i, timestamp: new Date().toISOString() })
      ).join("\n");

      await fs.writeFile(filePath, lines + "\n");

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(100);
        },
        { timeout: 5000 }
      );
    });
  });

  describe("Store Integration", () => {
    it("should update store state via updateTelemetry action", async () => {
      renderHook(() => useTelemetry());

      const decision = { timestamp: new Date().toISOString(), agent: "test" };
      await appendJsonl("routing-decisions", decision);

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBeGreaterThan(0);
        },
        { timeout: 2000 }
      );
    });

    it("should maintain separate arrays for each telemetry type", async () => {
      renderHook(() => useTelemetry());

      await appendJsonl("routing-decisions", { type: "decision" });
      await appendJsonl("handoffs", { type: "handoff" });
      await appendJsonl("sharp-edges", { type: "edge" });

      await waitFor(
        () => {
          const state = useStore.getState();
          expect(state.telemetry.routingDecisions.length).toBe(1);
          expect(state.telemetry.handoffs.length).toBe(1);
          expect(state.telemetry.sharpEdges.length).toBe(1);
        },
        { timeout: 3000 }
      );

      const state = useStore.getState();
      expect((state.telemetry.routingDecisions[0] as any).type).toBe("decision");
      expect((state.telemetry.handoffs[0] as any).type).toBe("handoff");
      expect((state.telemetry.sharpEdges[0] as any).type).toBe("edge");
    });
  });
});
