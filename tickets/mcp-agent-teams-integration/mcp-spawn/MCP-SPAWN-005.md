```yaml
---
id: MCP-SPAWN-005
title: Process Registry and Cleanup
description: Implement global process registry for tracking spawned CLI processes and cleanup on shutdown.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-004]
phase: 1
tags: [infrastructure, lifecycle, phase-1, critical]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-005: Process Registry and Cleanup

## Description

Implement a global process registry that tracks all spawned CLI processes and ensures cleanup on TUI shutdown. Includes SIGTERM → SIGKILL escalation for stubborn processes.

**Source**: Staff-Architect Analysis §4.3.1, §4.3.2, Einstein Analysis §3.5.1

## Why This Matters

Without process registry:
- Orphan processes accumulate if TUI crashes
- No way to kill all spawned agents on Ctrl+C
- Memory leak from abandoned processes
- System resource exhaustion over time

## Task

1. Create ProcessRegistry class
2. Implement SIGTERM → SIGKILL escalation
3. Integrate with existing shutdown handlers
4. Add signal forwarding (Ctrl+C to children)

## Files

- `packages/tui/src/spawn/processRegistry.ts` — Registry implementation
- `packages/tui/src/spawn/processRegistry.test.ts` — Tests
- `packages/tui/src/lifecycle/shutdown.ts` — Integration (modify existing)

## Implementation

### Process Registry (`packages/tui/src/spawn/processRegistry.ts`)

```typescript
import { ChildProcess } from "child_process";
import { EventEmitter } from "events";

export interface ProcessInfo {
  id: string;
  process: ChildProcess;
  agentType: string;
  startTime: number;
  status: "running" | "terminating" | "terminated";
}

export interface ProcessRegistryEvents {
  registered: (info: ProcessInfo) => void;
  unregistered: (id: string, reason: "completed" | "killed" | "crashed") => void;
  allCleaned: () => void;
}

/**
 * Global registry for tracking spawned CLI processes.
 * Ensures cleanup on shutdown with SIGTERM → SIGKILL escalation.
 */
export class ProcessRegistry extends EventEmitter {
  private processes: Map<string, ProcessInfo> = new Map();
  private cleanupInProgress = false;
  private readonly gracePeriod: number;
  private readonly forceKillDelay: number;

  constructor(options?: { gracePeriod?: number; forceKillDelay?: number }) {
    super();
    this.gracePeriod = options?.gracePeriod ?? 5000; // 5s for graceful shutdown
    this.forceKillDelay = options?.forceKillDelay ?? 1000; // 1s before SIGKILL
  }

  /**
   * Register a spawned process for tracking.
   */
  register(id: string, process: ChildProcess, agentType: string): void {
    if (this.cleanupInProgress) {
      // Don't accept new processes during cleanup
      process.kill("SIGTERM");
      return;
    }

    const info: ProcessInfo = {
      id,
      process,
      agentType,
      startTime: Date.now(),
      status: "running",
    };

    this.processes.set(id, info);
    this.emit("registered", info);

    // Auto-unregister on process exit
    process.on("exit", (code, signal) => {
      const reason = signal ? "killed" : code === 0 ? "completed" : "crashed";
      this.unregister(id, reason);
    });
  }

  /**
   * Unregister a process (called automatically on exit).
   */
  unregister(
    id: string,
    reason: "completed" | "killed" | "crashed" = "completed"
  ): void {
    if (this.processes.has(id)) {
      this.processes.delete(id);
      this.emit("unregistered", id, reason);

      if (this.cleanupInProgress && this.processes.size === 0) {
        this.emit("allCleaned");
      }
    }
  }

  /**
   * Get info about a registered process.
   */
  get(id: string): ProcessInfo | undefined {
    return this.processes.get(id);
  }

  /**
   * Get all registered process IDs.
   */
  getAll(): string[] {
    return Array.from(this.processes.keys());
  }

  /**
   * Get count of active processes.
   */
  get size(): number {
    return this.processes.size;
  }

  /**
   * Kill a specific process by ID.
   */
  async kill(id: string): Promise<boolean> {
    const info = this.processes.get(id);
    if (!info || info.status !== "running") {
      return false;
    }

    return this.terminateProcess(info);
  }

  /**
   * Clean up all processes with graceful shutdown.
   * Returns promise that resolves when all processes are terminated.
   */
  async cleanupAll(): Promise<void> {
    if (this.cleanupInProgress) {
      return; // Already cleaning up
    }

    this.cleanupInProgress = true;

    if (this.processes.size === 0) {
      this.cleanupInProgress = false;
      this.emit("allCleaned");
      return;
    }

    // Send SIGTERM to all
    const terminations = Array.from(this.processes.values()).map((info) =>
      this.terminateProcess(info)
    );

    // Wait for all with timeout
    await Promise.race([
      Promise.all(terminations),
      new Promise<void>((resolve) =>
        setTimeout(() => {
          this.forceKillAll();
          resolve();
        }, this.gracePeriod)
      ),
    ]);

    this.cleanupInProgress = false;
  }

  /**
   * Terminate a single process with SIGTERM → SIGKILL escalation.
   */
  private async terminateProcess(info: ProcessInfo): Promise<boolean> {
    if (info.status !== "running") {
      return false;
    }

    info.status = "terminating";
    info.process.kill("SIGTERM");

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        if (!info.process.killed) {
          info.process.kill("SIGKILL");
        }
        info.status = "terminated";
        resolve(true);
      }, this.forceKillDelay);

      info.process.on("exit", () => {
        clearTimeout(timeout);
        info.status = "terminated";
        resolve(true);
      });
    });
  }

  /**
   * Force kill all remaining processes (SIGKILL).
   */
  private forceKillAll(): void {
    for (const info of this.processes.values()) {
      if (!info.process.killed) {
        info.process.kill("SIGKILL");
        info.status = "terminated";
      }
    }
    this.processes.clear();
  }

  /**
   * Forward a signal to all child processes.
   */
  forwardSignal(signal: NodeJS.Signals): void {
    for (const info of this.processes.values()) {
      if (info.status === "running" && !info.process.killed) {
        info.process.kill(signal);
      }
    }
  }
}

// Singleton instance
let globalRegistry: ProcessRegistry | null = null;

/**
 * Get the global process registry instance.
 */
export function getProcessRegistry(): ProcessRegistry {
  if (!globalRegistry) {
    globalRegistry = new ProcessRegistry();
  }
  return globalRegistry;
}

/**
 * Reset global registry (for testing).
 */
export function resetProcessRegistry(): void {
  if (globalRegistry) {
    globalRegistry.removeAllListeners();
  }
  globalRegistry = null;
}
```

### Shutdown Integration (`packages/tui/src/lifecycle/shutdown.ts` modification)

```typescript
// Add to existing shutdown.ts

import { getProcessRegistry } from "../spawn/processRegistry";

// In setupSignalHandlers():
export function setupSignalHandlers(): void {
  const registry = getProcessRegistry();

  // Forward SIGINT to children
  process.on("SIGINT", async () => {
    registry.forwardSignal("SIGINT");
    await registry.cleanupAll();
    process.exit(0);
  });

  // Forward SIGTERM to children
  process.on("SIGTERM", async () => {
    registry.forwardSignal("SIGTERM");
    await registry.cleanupAll();
    process.exit(0);
  });

  // Clean up on uncaught exception
  process.on("uncaughtException", async (err) => {
    console.error("Uncaught exception:", err);
    await registry.cleanupAll();
    process.exit(1);
  });
}

// Add to onShutdown callbacks:
onShutdown(async () => {
  const registry = getProcessRegistry();
  await registry.cleanupAll();
});
```

### Tests (`packages/tui/src/spawn/processRegistry.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawn, ChildProcess } from "child_process";
import {
  ProcessRegistry,
  getProcessRegistry,
  resetProcessRegistry,
} from "./processRegistry";

describe("ProcessRegistry", () => {
  let registry: ProcessRegistry;

  beforeEach(() => {
    registry = new ProcessRegistry({
      gracePeriod: 100, // Short for tests
      forceKillDelay: 50,
    });
  });

  afterEach(() => {
    resetProcessRegistry();
  });

  describe("register", () => {
    it("should track registered processes", () => {
      const mockProcess = createMockProcess();

      registry.register("test-1", mockProcess, "test-agent");

      expect(registry.size).toBe(1);
      expect(registry.get("test-1")).toBeDefined();
      expect(registry.get("test-1")?.agentType).toBe("test-agent");
    });

    it("should emit registered event", () => {
      const mockProcess = createMockProcess();
      const listener = vi.fn();
      registry.on("registered", listener);

      registry.register("test-1", mockProcess, "test-agent");

      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({ id: "test-1" })
      );
    });
  });

  describe("unregister", () => {
    it("should remove process from registry", () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");

      registry.unregister("test-1", "completed");

      expect(registry.size).toBe(0);
      expect(registry.get("test-1")).toBeUndefined();
    });

    it("should emit unregistered event", () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");
      const listener = vi.fn();
      registry.on("unregistered", listener);

      registry.unregister("test-1", "killed");

      expect(listener).toHaveBeenCalledWith("test-1", "killed");
    });
  });

  describe("kill", () => {
    it("should kill specific process", async () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");

      const result = await registry.kill("test-1");

      expect(result).toBe(true);
      expect(mockProcess.kill).toHaveBeenCalledWith("SIGTERM");
    });

    it("should return false for non-existent process", async () => {
      const result = await registry.kill("non-existent");

      expect(result).toBe(false);
    });
  });

  describe("cleanupAll", () => {
    it("should terminate all processes", async () => {
      const mock1 = createMockProcess();
      const mock2 = createMockProcess();
      registry.register("test-1", mock1, "agent1");
      registry.register("test-2", mock2, "agent2");

      await registry.cleanupAll();

      expect(mock1.kill).toHaveBeenCalled();
      expect(mock2.kill).toHaveBeenCalled();
    });

    it("should emit allCleaned when done", async () => {
      const listener = vi.fn();
      registry.on("allCleaned", listener);

      await registry.cleanupAll();

      expect(listener).toHaveBeenCalled();
    });
  });

  describe("forwardSignal", () => {
    it("should forward signal to all children", () => {
      const mock1 = createMockProcess();
      const mock2 = createMockProcess();
      registry.register("test-1", mock1, "agent1");
      registry.register("test-2", mock2, "agent2");

      registry.forwardSignal("SIGINT");

      expect(mock1.kill).toHaveBeenCalledWith("SIGINT");
      expect(mock2.kill).toHaveBeenCalledWith("SIGINT");
    });
  });
});

// Helper to create mock ChildProcess
function createMockProcess(): ChildProcess {
  const emitter = new (require("events").EventEmitter)();
  return {
    ...emitter,
    pid: Math.floor(Math.random() * 10000),
    killed: false,
    kill: vi.fn((signal) => {
      emitter.emit("exit", 0, signal);
      return true;
    }),
    stdin: { write: vi.fn(), end: vi.fn() },
    stdout: { on: vi.fn() },
    stderr: { on: vi.fn() },
  } as unknown as ChildProcess;
}
```

## Acceptance Criteria

- [ ] ProcessRegistry class implemented with all methods
- [ ] SIGTERM → SIGKILL escalation works (1s delay)
- [ ] Signal forwarding works (SIGINT, SIGTERM)
- [ ] Auto-unregister on process exit
- [ ] Integrated with existing shutdown handlers
- [ ] All tests pass: `npm test -- src/spawn/processRegistry.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/src/spawn/processRegistry.test.ts`
- [ ] Number of test functions: 9
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Integration tested with real processes (manual)

