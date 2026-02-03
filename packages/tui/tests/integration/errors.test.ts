/**
 * Error Handling Integration Tests
 * Tests timeout, cancellation, and error scenarios in MCP tool flow
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { askUserTool } from "../../src/mcp/tools/askUser.js";
import { confirmActionTool } from "../../src/mcp/tools/confirmAction.js";
import { requestInputTool } from "../../src/mcp/tools/requestInput.js";
import { selectOptionTool } from "../../src/mcp/tools/selectOption.js";

describe("Error Handling - Timeouts", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should timeout and return default value for ask_user", async () => {
    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: {
        message: "Question",
        defaultValue: "default answer",
      },
      timeout: 1000,
    });

    // Fast-forward time
    vi.advanceTimersByTime(1000);

    const response = await promise;
    expect(response).toEqual({
      type: "ask",
      value: "default answer",
    });

    // Queue should be empty after timeout
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should timeout and reject for ask_user without default", async () => {
    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: {
        message: "Question",
      },
      timeout: 1000,
    });

    vi.advanceTimersByTime(1000);

    await expect(promise).rejects.toThrow("Modal timeout after 1000ms");
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should timeout and return cancelled for confirm_action", async () => {
    const promise = useStore.getState().enqueue({
      type: "confirm",
      payload: {
        action: "Delete",
        destructive: true,
      },
      timeout: 2000,
    });

    vi.advanceTimersByTime(2000);

    const response = await promise;
    expect(response).toEqual({
      type: "confirm",
      confirmed: false,
      cancelled: true,
    });

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should timeout and reject for request_input", async () => {
    const promise = useStore.getState().enqueue({
      type: "input",
      payload: {
        prompt: "Enter value",
      },
      timeout: 1500,
    });

    vi.advanceTimersByTime(1500);

    await expect(promise).rejects.toThrow("Modal timeout after 1500ms");
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should timeout and reject for select_option", async () => {
    const promise = useStore.getState().enqueue({
      type: "select",
      payload: {
        message: "Choose",
        options: [{ label: "A", value: "a" }],
      },
      timeout: 3000,
    });

    vi.advanceTimersByTime(3000);

    await expect(promise).rejects.toThrow("Modal timeout after 3000ms");
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should clear timeout if user responds before timeout", async () => {
    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: {
        message: "Quick question",
      },
      timeout: 5000,
    });

    // Advance only 1 second
    vi.advanceTimersByTime(1000);

    // User responds
    const modal = useStore.getState().modalQueue[0];
    if (modal) {
      useStore.getState().dequeue(modal.id, {
        type: "ask",
        value: "Quick answer",
      });
    }

    const response = await promise;
    expect(response).toEqual({
      type: "ask",
      value: "Quick answer",
    });

    // Advance remaining time - should not cause issues
    vi.advanceTimersByTime(4000);

    // No lingering effects
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle timeout with concurrent modals", async () => {
    const promise1 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q1" },
      timeout: 1000,
    });

    const promise2 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q2" },
      timeout: 2000,
    });

    const promise3 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q3" },
      timeout: 3000,
    });

    // First times out at 1000ms
    vi.advanceTimersByTime(1000);
    await expect(promise1).rejects.toThrow("Modal timeout after 1000ms");

    let queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(2);

    // Second times out at 2000ms (total)
    vi.advanceTimersByTime(1000);
    await expect(promise2).rejects.toThrow("Modal timeout after 2000ms");

    queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(1);

    // Third times out at 3000ms (total)
    vi.advanceTimersByTime(1000);
    await expect(promise3).rejects.toThrow("Modal timeout after 3000ms");

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });
});

describe("Error Handling - Cancellation", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
  });

  it("should handle user cancellation via Escape", async () => {
    const promise = useStore.getState().enqueue({
      type: "confirm",
      payload: {
        action: "Dangerous action",
        destructive: true,
      },
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    const modal = useStore.getState().modalQueue[0];
    expect(modal).toBeDefined();

    // User presses Escape
    if (modal) {
      useStore.getState().cancel(modal.id);
    }

    await expect(promise).rejects.toThrow("Modal cancelled by user");
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle cancellation of first in queue", async () => {
    const promise1 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q1" },
    });

    const promise2 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q2" },
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // Cancel first
    const queue = useStore.getState().modalQueue;
    useStore.getState().cancel(queue[0]!.id);

    await expect(promise1).rejects.toThrow("Modal cancelled by user");

    // Second should still be in queue
    expect(useStore.getState().modalQueue).toHaveLength(1);
    expect(useStore.getState().modalQueue[0]?.payload).toMatchObject({
      message: "Q2",
    });

    // Complete second
    useStore.getState().dequeue(useStore.getState().modalQueue[0]!.id, {
      type: "ask",
      value: "A2",
    });

    const result2 = await promise2;
    expect(result2).toMatchObject({ type: "ask", value: "A2" });
  });

  it("should handle cancellation of middle modal", async () => {
    const promise1 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q1" },
    });
    const promise2 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q2" },
    });
    const promise3 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Q3" },
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;

    // Cancel middle one
    useStore.getState().cancel(queue[1]!.id);

    await expect(promise2).rejects.toThrow("Modal cancelled by user");

    // First and third still in queue
    expect(useStore.getState().modalQueue).toHaveLength(2);

    // Complete first and third
    const remainingQueue = useStore.getState().modalQueue;
    useStore.getState().dequeue(remainingQueue[0]!.id, { type: "ask", value: "A1" });
    useStore.getState().dequeue(remainingQueue[1]!.id, { type: "ask", value: "A3" });

    const [result1, result3] = await Promise.all([promise1, promise3]);
    expect(result1).toMatchObject({ type: "ask", value: "A1" });
    expect(result3).toMatchObject({ type: "ask", value: "A3" });
  });

  it("should clear timeout when modal is cancelled", async () => {
    vi.useFakeTimers();

    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Question" },
      timeout: 5000,
    });

    await Promise.resolve();

    // Cancel before timeout
    const modal = useStore.getState().modalQueue[0];
    if (modal) {
      useStore.getState().cancel(modal.id);
    }

    await expect(promise).rejects.toThrow("Modal cancelled by user");

    // Advance time - timeout should not fire
    vi.advanceTimersByTime(5000);

    // No additional errors
    expect(useStore.getState().modalQueue).toHaveLength(0);

    vi.useRealTimers();
  });

  it("should handle cancellation attempt on non-existent modal", () => {
    // Should not throw
    expect(() => {
      useStore.getState().cancel("non-existent-id");
    }).not.toThrow();

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });
});

describe("Error Handling - Invalid Responses", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
  });

  it("should handle dequeue with non-existent modal ID", () => {
    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Test" },
    });

    // Try to dequeue wrong ID
    useStore.getState().dequeue("wrong-id", { type: "ask", value: "Answer" });

    // Original modal still in queue
    expect(useStore.getState().modalQueue).toHaveLength(1);
  });

  it("should handle double-resolution protection", async () => {
    vi.useFakeTimers();

    const promise = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Test" },
      timeout: 1000,
    });

    await Promise.resolve();

    const modal = useStore.getState().modalQueue[0];

    // Respond normally
    if (modal) {
      useStore.getState().dequeue(modal.id, { type: "ask", value: "Response" });
    }

    const result = await promise;
    expect(result).toMatchObject({ type: "ask", value: "Response" });

    // Try to trigger timeout - should not cause issues
    vi.advanceTimersByTime(1000);

    // No duplicate resolution
    expect(useStore.getState().modalQueue).toHaveLength(0);

    vi.useRealTimers();
  });
});

// Tool validation tests are in tests/mcp/tools.test.ts
// These test schema validation which is already covered

describe("Error Handling - Edge Cases", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
  });

  it("should handle rapid cancel and new enqueue", async () => {
    const promise1 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "First" },
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // Cancel immediately
    const modal1 = useStore.getState().modalQueue[0];
    if (modal1) {
      useStore.getState().cancel(modal1.id);
    }

    // First should reject
    await expect(promise1).rejects.toThrow("Modal cancelled by user");

    // Enqueue new one
    const promise2 = useStore.getState().enqueue({
      type: "ask",
      payload: { message: "Second" },
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // Second should be in queue
    expect(useStore.getState().modalQueue).toHaveLength(1);

    // Complete second
    const modal2 = useStore.getState().modalQueue[0];
    if (modal2) {
      useStore.getState().dequeue(modal2.id, { type: "ask", value: "Answer" });
    }

    const result = await promise2;
    expect(result).toMatchObject({ type: "ask", value: "Answer" });
  });

  it("should handle all concurrent timeouts at same time", async () => {
    vi.useFakeTimers();

    const promises = [
      useStore.getState().enqueue({
        type: "ask",
        payload: { message: "Q1" },
        timeout: 1000,
      }),
      useStore.getState().enqueue({
        type: "ask",
        payload: { message: "Q2" },
        timeout: 1000,
      }),
      useStore.getState().enqueue({
        type: "ask",
        payload: { message: "Q3" },
        timeout: 1000,
      }),
    ];

    // All timeout at same time
    vi.advanceTimersByTime(1000);

    await Promise.allSettled(promises);

    // All should reject
    for (const promise of promises) {
      await expect(promise).rejects.toThrow();
    }

    expect(useStore.getState().modalQueue).toHaveLength(0);

    vi.useRealTimers();
  });
});
