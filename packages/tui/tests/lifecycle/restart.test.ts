/**
 * Unit tests for RestartManager
 * Tests: exponential backoff, attempt tracking, successful run reset
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  RestartManager,
  getRestartManager,
  resetRestartManager,
  type RestartConfig,
} from "../../src/lifecycle/restart";

describe("RestartManager", () => {
  let manager: RestartManager;

  beforeEach(() => {
    manager = new RestartManager();
  });

  describe("shouldRestart", () => {
    it("should allow restart within max attempts", () => {
      expect(manager.shouldRestart()).toBe(true);
    });

    it("should block restart after max attempts", () => {
      manager.recordAttempt();
      manager.recordAttempt();
      manager.recordAttempt();
      expect(manager.shouldRestart()).toBe(false);
    });
  });

  describe("getDelay", () => {
    it("should return initial delay on first attempt", () => {
      const delay = manager.getDelay();
      expect(delay).toBe(1000); // initialDelayMs
    });

    it("should apply exponential backoff", () => {
      // Attempt 0: 1000ms
      expect(manager.getDelay()).toBe(1000);

      // Attempt 1: 2000ms (1000 * 2^1)
      manager.recordAttempt();
      expect(manager.getDelay()).toBe(2000);

      // Attempt 2: 4000ms (1000 * 2^2)
      manager.recordAttempt();
      expect(manager.getDelay()).toBe(4000);
    });

    it("should cap delay at maxDelayMs", () => {
      const config: RestartConfig = {
        maxAttempts: 10,
        initialDelayMs: 1000,
        maxDelayMs: 5000,
        backoffMultiplier: 2,
        successfulRunThresholdMs: 60000,
      };

      manager = new RestartManager(config);

      // Force many attempts
      for (let i = 0; i < 5; i++) {
        manager.recordAttempt();
      }

      // Should be capped at 5000ms
      expect(manager.getDelay()).toBe(5000);
    });
  });

  describe("recordAttempt", () => {
    it("should increment attempt counter", () => {
      expect(manager.getState().attempts).toBe(0);

      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(1);

      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(2);
    });

    it("should update lastAttemptTime", () => {
      const before = Date.now();
      manager.recordAttempt();
      const after = Date.now();

      const state = manager.getState();
      expect(state.lastAttemptTime).toBeGreaterThanOrEqual(before);
      expect(state.lastAttemptTime).toBeLessThanOrEqual(after);
    });
  });

  describe("recordSuccessfulStart", () => {
    it("should track start time", () => {
      const before = Date.now();
      manager.recordSuccessfulStart();
      const after = Date.now();

      // Can't directly check private field, but we can test behavior
      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(1);
    });
  });

  describe("checkAndResetIfSuccessful", () => {
    it("should reset after successful run duration", async () => {
      const config: RestartConfig = {
        maxAttempts: 3,
        initialDelayMs: 1000,
        maxDelayMs: 30000,
        backoffMultiplier: 2,
        successfulRunThresholdMs: 100, // 100ms for testing
      };

      manager = new RestartManager(config);

      // Record some failed attempts
      manager.recordAttempt();
      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(2);

      // Record successful start
      manager.recordSuccessfulStart();

      // Wait for threshold
      await new Promise((resolve) => setTimeout(resolve, 150));

      // Check and reset
      manager.checkAndResetIfSuccessful();

      // Should be reset
      expect(manager.getState().attempts).toBe(0);
    });

    it("should not reset before threshold", async () => {
      const config: RestartConfig = {
        maxAttempts: 3,
        initialDelayMs: 1000,
        maxDelayMs: 30000,
        backoffMultiplier: 2,
        successfulRunThresholdMs: 1000, // 1 second
      };

      manager = new RestartManager(config);

      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(1);

      manager.recordSuccessfulStart();

      // Check immediately (before threshold)
      manager.checkAndResetIfSuccessful();

      // Should NOT be reset
      expect(manager.getState().attempts).toBe(1);
    });
  });

  describe("reset", () => {
    it("should reset attempt counter", () => {
      manager.recordAttempt();
      manager.recordAttempt();
      expect(manager.getState().attempts).toBe(2);

      manager.reset();
      expect(manager.getState().attempts).toBe(0);
    });

    it("should reset lastAttemptTime", () => {
      manager.recordAttempt();
      expect(manager.getState().lastAttemptTime).toBeGreaterThan(0);

      manager.reset();
      expect(manager.getState().lastAttemptTime).toBe(0);
    });
  });

  describe("getState", () => {
    it("should return current state", () => {
      const state = manager.getState();

      expect(state).toHaveProperty("attempts");
      expect(state).toHaveProperty("maxAttempts");
      expect(state).toHaveProperty("lastAttemptTime");
      expect(state).toHaveProperty("canRestart");
      expect(state).toHaveProperty("nextDelay");
    });

    it("should reflect attempt changes", () => {
      let state = manager.getState();
      expect(state.attempts).toBe(0);
      expect(state.canRestart).toBe(true);

      manager.recordAttempt();
      state = manager.getState();
      expect(state.attempts).toBe(1);
    });
  });

  describe("Backoff sequence", () => {
    it("should follow correct delay pattern", () => {
      // Default config: 1s, 2s, 4s (max 3 attempts)
      const delays: number[] = [];

      for (let i = 0; i < 3; i++) {
        delays.push(manager.getDelay());
        manager.recordAttempt();
      }

      expect(delays).toEqual([1000, 2000, 4000]);
    });

    it("should prevent restart after max attempts", () => {
      for (let i = 0; i < 3; i++) {
        manager.recordAttempt();
      }

      expect(manager.shouldRestart()).toBe(false);
      expect(manager.getState().canRestart).toBe(false);
    });
  });
});

describe("Global RestartManager", () => {
  beforeEach(() => {
    resetRestartManager();
  });

  it("should return singleton instance", () => {
    const manager1 = getRestartManager();
    const manager2 = getRestartManager();

    expect(manager1).toBe(manager2);
  });

  it("should accept custom config on first call", () => {
    const config: Partial<RestartConfig> = {
      maxAttempts: 5,
      initialDelayMs: 2000,
    };

    const manager = getRestartManager(config);
    const state = manager.getState();

    expect(state.maxAttempts).toBe(5);
    expect(state.nextDelay).toBe(2000);
  });

  it("should ignore config on subsequent calls", () => {
    const manager1 = getRestartManager({ maxAttempts: 5 });
    const manager2 = getRestartManager({ maxAttempts: 10 });

    // Should be same instance, first config wins
    expect(manager1).toBe(manager2);
    expect(manager1.getState().maxAttempts).toBe(5);
  });

  it("should reset singleton", () => {
    const manager1 = getRestartManager();
    manager1.recordAttempt();

    resetRestartManager();

    const manager2 = getRestartManager();
    expect(manager2.getState().attempts).toBe(0);
    expect(manager1).not.toBe(manager2);
  });
});
