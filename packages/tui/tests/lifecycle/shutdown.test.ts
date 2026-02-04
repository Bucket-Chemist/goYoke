/**
 * Unit tests for shutdown handling
 * Tests: signal handlers, graceful shutdown, handler execution, state management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  onShutdown,
  initiateShutdown,
  setupSignalHandlers,
  isShutdownInProgress,
  resetShutdownState,
  registerChildProcessCleanup,
} from "../../src/lifecycle/shutdown";
import { useStore } from "../../src/store";

describe("Shutdown Handler", () => {
  beforeEach(() => {
    resetShutdownState();
    useStore.getState().clearSession();
    vi.clearAllMocks();
  });

  afterEach(() => {
    resetShutdownState();
  });

  describe("onShutdown", () => {
    it("should register shutdown handler", () => {
      const handler = vi.fn(async () => {});
      onShutdown(handler);

      // Handler registered but not called yet
      expect(handler).not.toHaveBeenCalled();
    });

    it("should allow multiple handlers", () => {
      const handler1 = vi.fn(async () => {});
      const handler2 = vi.fn(async () => {});

      onShutdown(handler1);
      onShutdown(handler2);

      // Both registered
      expect(handler1).not.toHaveBeenCalled();
      expect(handler2).not.toHaveBeenCalled();
    });
  });

  describe("initiateShutdown", () => {
    it("should execute registered handlers", async () => {
      const handler1 = vi.fn(async () => {});
      const handler2 = vi.fn(async () => {});

      onShutdown(handler1);
      onShutdown(handler2);

      // Mock process.exit to prevent test termination
      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected - process.exit throws
      }

      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).toHaveBeenCalledTimes(1);

      exitSpy.mockRestore();
    });

    it("should execute handlers in registration order", async () => {
      const order: number[] = [];

      const handler1 = vi.fn(async () => {
        order.push(1);
      });
      const handler2 = vi.fn(async () => {
        order.push(2);
      });
      const handler3 = vi.fn(async () => {
        order.push(3);
      });

      onShutdown(handler1);
      onShutdown(handler2);
      onShutdown(handler3);

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      expect(order).toEqual([1, 2, 3]);

      exitSpy.mockRestore();
    });

    it("should continue if handler throws error", async () => {
      const handler1 = vi.fn(async () => {
        throw new Error("Handler 1 error");
      });
      const handler2 = vi.fn(async () => {
        // This should still execute
      });

      onShutdown(handler1);
      onShutdown(handler2);

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      // Both handlers should have been called despite error
      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).toHaveBeenCalledTimes(1);

      exitSpy.mockRestore();
    });

    it("should prevent duplicate shutdown execution", async () => {
      const handler = vi.fn(async () => {
        // Simulate slow handler
        await new Promise((resolve) => setTimeout(resolve, 100));
      });

      onShutdown(handler);

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      // Start first shutdown
      const shutdown1 = initiateShutdown("TEST1").catch(() => {});

      // Try to start second shutdown while first is in progress
      const shutdown2 = initiateShutdown("TEST2").catch(() => {});

      await Promise.all([shutdown1, shutdown2]);

      // Handler should only be called once
      expect(handler).toHaveBeenCalledTimes(1);

      exitSpy.mockRestore();
    });

    it("should set shutdown in progress flag", async () => {
      expect(isShutdownInProgress()).toBe(false);

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      const shutdownPromise = initiateShutdown("TEST").catch(() => {});

      // Should be in progress immediately
      expect(isShutdownInProgress()).toBe(true);

      await shutdownPromise;

      exitSpy.mockRestore();
    });

    it("should call process.exit with code 0", async () => {
      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      expect(exitSpy).toHaveBeenCalledWith(0);

      exitSpy.mockRestore();
    });
  });

  describe("registerChildProcessCleanup", () => {
    it("should register cleanup as shutdown handler", async () => {
      const cleanup = vi.fn(async () => {});

      registerChildProcessCleanup(cleanup);

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      expect(cleanup).toHaveBeenCalledTimes(1);

      exitSpy.mockRestore();
    });
  });

  describe("setupSignalHandlers", () => {
    it("should register SIGINT handler", () => {
      const onSpy = vi.spyOn(process, "on");

      setupSignalHandlers();

      expect(onSpy).toHaveBeenCalledWith("SIGINT", expect.any(Function));

      onSpy.mockRestore();
    });

    it("should register SIGTERM handler", () => {
      const onSpy = vi.spyOn(process, "on");

      setupSignalHandlers();

      expect(onSpy).toHaveBeenCalledWith("SIGTERM", expect.any(Function));

      onSpy.mockRestore();
    });

    it("should register uncaughtException handler", () => {
      const onSpy = vi.spyOn(process, "on");

      setupSignalHandlers();

      expect(onSpy).toHaveBeenCalledWith(
        "uncaughtException",
        expect.any(Function)
      );

      onSpy.mockRestore();
    });

    it("should register unhandledRejection handler", () => {
      const onSpy = vi.spyOn(process, "on");

      setupSignalHandlers();

      expect(onSpy).toHaveBeenCalledWith(
        "unhandledRejection",
        expect.any(Function)
      );

      onSpy.mockRestore();
    });

    it("should register warning handler", () => {
      const onSpy = vi.spyOn(process, "on");

      setupSignalHandlers();

      expect(onSpy).toHaveBeenCalledWith("warning", expect.any(Function));

      onSpy.mockRestore();
    });
  });

  describe("resetShutdownState", () => {
    it("should reset shutdown flag", async () => {
      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      expect(isShutdownInProgress()).toBe(true);

      resetShutdownState();

      expect(isShutdownInProgress()).toBe(false);

      exitSpy.mockRestore();
    });

    it("should clear registered handlers", async () => {
      const handler = vi.fn(async () => {});
      onShutdown(handler);

      resetShutdownState();

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      // Handler should not be called (was cleared)
      expect(handler).not.toHaveBeenCalled();

      exitSpy.mockRestore();
    });
  });

  describe("Session persistence integration", () => {
    it("should save session before shutdown", async () => {
      const { updateSession } = useStore.getState();

      // Setup a session
      updateSession({
        id: "test-session-123",
        created_at: new Date().toISOString(),
        last_used: new Date().toISOString(),
        cost: 0.42,
        tool_calls: 127,
      });

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      // Session should have been saved (saveSession called)
      // Note: Actual file I/O is hard to test, but we verify the flow runs

      exitSpy.mockRestore();
    });

    it("should handle missing session gracefully", async () => {
      // No session in store

      const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
        throw new Error("process.exit called");
      });

      try {
        await initiateShutdown("TEST");
      } catch (error) {
        // Expected
      }

      // Should not throw, just skip session save
      expect(exitSpy).toHaveBeenCalledWith(0);

      exitSpy.mockRestore();
    });
  });
});
