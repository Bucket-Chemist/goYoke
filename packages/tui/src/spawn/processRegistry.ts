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
