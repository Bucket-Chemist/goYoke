import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawn, ChildProcess } from "child_process";
import {
  ProcessRegistry,
  getProcessRegistry,
  resetProcessRegistry,
} from "./processRegistry.js";

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
  const EventEmitter = require("events").EventEmitter;
  const emitter = new EventEmitter();

  const mockProcess = Object.create(emitter);
  mockProcess.pid = Math.floor(Math.random() * 10000);
  mockProcess.killed = false;
  mockProcess.kill = vi.fn((signal) => {
    emitter.emit("exit", 0, signal);
    return true;
  });
  mockProcess.stdin = { write: vi.fn(), end: vi.fn() };
  mockProcess.stdout = { on: vi.fn() };
  mockProcess.stderr = { on: vi.fn() };

  return mockProcess as unknown as ChildProcess;
}
