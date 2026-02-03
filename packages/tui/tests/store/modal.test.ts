/**
 * Unit tests for modal slice
 * Tests: enqueue/dequeue flow, FIFO queueing, timeout, cancel, Promise resolution
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { useStore } from "../../src/store";
import type { ModalResponse } from "../../src/store/slices/modal";

describe("Modal Slice", () => {
  beforeEach(() => {
    // Clear modal queue before each test
    // Use dequeue instead of cancel to avoid unhandled rejection warnings
    const state = useStore.getState();
    state.modalQueue.forEach((modal) => {
      // Resolve with a default response instead of rejecting
      state.dequeue(modal.id, { type: modal.type, value: "__cleanup__" } as ModalResponse);
    });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("enqueue", () => {
    it("should enqueue a modal and return a Promise", async () => {
      const { enqueue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "What color?", options: ["Red", "Blue"] },
      });

      expect(promise).toBeInstanceOf(Promise);

      const { modalQueue } = useStore.getState();
      expect(modalQueue).toHaveLength(1);
      expect(modalQueue[0].type).toBe("ask");
      expect(modalQueue[0].payload).toEqual({
        message: "What color?",
        options: ["Red", "Blue"],
      });
      expect(modalQueue[0].id).toBeDefined();
    });

    it("should generate unique IDs for each modal", () => {
      const { enqueue } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "First" } });
      enqueue({ type: "confirm", payload: { action: "Delete" } });
      enqueue({ type: "input", payload: { prompt: "Name?" } });

      const { modalQueue } = useStore.getState();

      expect(modalQueue).toHaveLength(3);
      expect(modalQueue[0].id).not.toBe(modalQueue[1].id);
      expect(modalQueue[1].id).not.toBe(modalQueue[2].id);
      expect(modalQueue[0].id).not.toBe(modalQueue[2].id);
    });

    it("should maintain FIFO order", () => {
      const { enqueue } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "First" } });
      enqueue({ type: "confirm", payload: { action: "Second" } });
      enqueue({ type: "input", payload: { prompt: "Third" } });

      const { modalQueue } = useStore.getState();

      expect(modalQueue[0].type).toBe("ask");
      expect(modalQueue[1].type).toBe("confirm");
      expect(modalQueue[2].type).toBe("input");
    });

    it("should handle all modal types", () => {
      const { enqueue } = useStore.getState();

      enqueue({
        type: "ask",
        payload: { message: "Question", options: ["A", "B"] },
      });

      enqueue({
        type: "confirm",
        payload: { action: "Delete file", destructive: true },
      });

      enqueue({
        type: "input",
        payload: { prompt: "Enter name", placeholder: "John Doe" },
      });

      enqueue({
        type: "select",
        payload: {
          message: "Choose option",
          options: [
            { label: "First", value: "1" },
            { label: "Second", value: "2" },
          ],
        },
      });

      const { modalQueue } = useStore.getState();

      expect(modalQueue).toHaveLength(4);
      expect(modalQueue[0].type).toBe("ask");
      expect(modalQueue[1].type).toBe("confirm");
      expect(modalQueue[2].type).toBe("input");
      expect(modalQueue[3].type).toBe("select");
    });
  });

  describe("dequeue", () => {
    it("should resolve Promise with response when dequeued", async () => {
      const { enqueue, dequeue, modalQueue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "What color?" },
      });

      const modalId = useStore.getState().modalQueue[0].id;

      const response: ModalResponse = { type: "ask", value: "Blue" };
      dequeue(modalId, response);

      const result = await promise;

      expect(result).toEqual({ type: "ask", value: "Blue" });
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should remove modal from queue when dequeued", () => {
      const { enqueue, dequeue } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "First" } });
      enqueue({ type: "confirm", payload: { action: "Second" } });

      const firstId = useStore.getState().modalQueue[0].id;

      dequeue(firstId, { type: "ask", value: "Answer" });

      const { modalQueue } = useStore.getState();

      expect(modalQueue).toHaveLength(1);
      expect(modalQueue[0].type).toBe("confirm");
    });

    it("should handle dequeuing non-existent modal gracefully", () => {
      const { enqueue, dequeue } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "Test" } });

      // Attempt to dequeue non-existent ID
      dequeue("non-existent-id", { type: "ask", value: "Test" });

      // Should not crash, queue unchanged
      expect(useStore.getState().modalQueue).toHaveLength(1);
    });

    it("should handle different response types", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promises = [
        enqueue({ type: "ask", payload: { message: "Q1" } }),
        enqueue({ type: "confirm", payload: { action: "Delete" } }),
        enqueue({ type: "input", payload: { prompt: "Name?" } }),
        enqueue({
          type: "select",
          payload: {
            message: "Pick",
            options: [{ label: "A", value: "a" }],
          },
        }),
      ];

      const queue = useStore.getState().modalQueue;

      dequeue(queue[0].id, { type: "ask", value: "Answer" });
      dequeue(queue[1].id, { type: "confirm", confirmed: true, cancelled: false });
      dequeue(queue[2].id, { type: "input", value: "John" });
      dequeue(queue[3].id, { type: "select", selected: "a", index: 0 });

      const results = await Promise.all(promises);

      expect(results[0]).toEqual({ type: "ask", value: "Answer" });
      expect(results[1]).toEqual({ type: "confirm", confirmed: true, cancelled: false });
      expect(results[2]).toEqual({ type: "input", value: "John" });
      expect(results[3]).toEqual({ type: "select", selected: "a", index: 0 });
    });
  });

  describe("cancel", () => {
    it("should reject Promise when cancelled", async () => {
      const { enqueue, cancel } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "What color?" },
      });

      const modalId = useStore.getState().modalQueue[0].id;

      cancel(modalId);

      await expect(promise).rejects.toThrow("Modal cancelled by user");
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should remove modal from queue when cancelled", async () => {
      const { enqueue, cancel } = useStore.getState();

      const p1 = enqueue({ type: "ask", payload: { message: "First" } });
      enqueue({ type: "confirm", payload: { action: "Second" } });

      const firstId = useStore.getState().modalQueue[0].id;

      cancel(firstId);

      // Await the rejection to prevent unhandled rejection warning
      await expect(p1).rejects.toThrow("Modal cancelled by user");

      const { modalQueue } = useStore.getState();

      expect(modalQueue).toHaveLength(1);
      expect(modalQueue[0].type).toBe("confirm");
    });

    it("should handle cancelling non-existent modal gracefully", () => {
      const { enqueue, cancel } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "Test" } });

      // Attempt to cancel non-existent ID
      cancel("non-existent-id");

      // Should not crash, queue unchanged
      expect(useStore.getState().modalQueue).toHaveLength(1);
    });

    it("should allow cancelling middle modal in queue", async () => {
      const { enqueue, cancel } = useStore.getState();

      enqueue({ type: "ask", payload: { message: "First" } });
      const p2 = enqueue({ type: "confirm", payload: { action: "Second" } });
      enqueue({ type: "input", payload: { prompt: "Third" } });

      const middleId = useStore.getState().modalQueue[1].id;

      cancel(middleId);

      // Await the rejection to prevent unhandled rejection warning
      await expect(p2).rejects.toThrow("Modal cancelled by user");

      const { modalQueue } = useStore.getState();

      expect(modalQueue).toHaveLength(2);
      expect(modalQueue[0].type).toBe("ask");
      expect(modalQueue[1].type).toBe("input");
    });
  });

  describe("timeout", () => {
    it("should reject Promise after timeout", async () => {
      const { enqueue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "What color?" },
        timeout: 1000,
      });

      expect(useStore.getState().modalQueue).toHaveLength(1);

      // Fast-forward time
      vi.advanceTimersByTime(1000);

      await expect(promise).rejects.toThrow("Modal timeout after 1000ms");
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should remove modal from queue after timeout", async () => {
      const { enqueue } = useStore.getState();

      const p1 = enqueue({
        type: "ask",
        payload: { message: "Will timeout" },
        timeout: 500,
      });

      enqueue({ type: "confirm", payload: { action: "No timeout" } });

      expect(useStore.getState().modalQueue).toHaveLength(2);

      vi.advanceTimersByTime(500);

      // Await the rejection to prevent unhandled rejection warning
      await expect(p1).rejects.toThrow("Modal timeout after 500ms");

      expect(useStore.getState().modalQueue).toHaveLength(1);
      expect(useStore.getState().modalQueue[0].type).toBe("confirm");
    });

    it("should not timeout if dequeued before timeout", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "Quick response" },
        timeout: 5000,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Dequeue before timeout
      vi.advanceTimersByTime(1000);
      dequeue(modalId, { type: "ask", value: "Quick answer" });

      const result = await promise;

      expect(result).toEqual({ type: "ask", value: "Quick answer" });

      // Advance past timeout - should not reject
      vi.advanceTimersByTime(5000);

      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should not timeout if cancelled before timeout", async () => {
      const { enqueue, cancel } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "Will cancel" },
        timeout: 5000,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Cancel before timeout
      vi.advanceTimersByTime(1000);
      cancel(modalId);

      await expect(promise).rejects.toThrow("Modal cancelled by user");

      // Advance past timeout - should not double-reject
      vi.advanceTimersByTime(5000);

      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should handle timeout of 0 (no timeout)", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "No timeout" },
        timeout: 0,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Advance time arbitrarily - should never timeout
      vi.advanceTimersByTime(999999);

      expect(useStore.getState().modalQueue).toHaveLength(1);

      // Manual dequeue should still work
      dequeue(modalId, { type: "ask", value: "Answer" });

      const result = await promise;
      expect(result).toEqual({ type: "ask", value: "Answer" });
    });

    it("should handle undefined timeout (no timeout)", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "No timeout" },
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Advance time arbitrarily - should never timeout
      vi.advanceTimersByTime(999999);

      expect(useStore.getState().modalQueue).toHaveLength(1);

      // Manual dequeue should still work
      dequeue(modalId, { type: "ask", value: "Answer" });

      const result = await promise;
      expect(result).toEqual({ type: "ask", value: "Answer" });
    });

    it("should handle negative timeout (no timeout)", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue({
        type: "ask",
        payload: { message: "No timeout with negative value" },
        timeout: -1000,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Advance time arbitrarily - should never timeout
      vi.advanceTimersByTime(999999);

      expect(useStore.getState().modalQueue).toHaveLength(1);

      // Manual dequeue should still work
      dequeue(modalId, { type: "ask", value: "Answer" });

      const result = await promise;
      expect(result).toEqual({ type: "ask", value: "Answer" });
    });

    it("should not double-resolve when timeout and completion race", async () => {
      const { enqueue, dequeue } = useStore.getState();

      // Create a promise that will be resolved both by timeout and dequeue
      const promise = enqueue({
        type: "confirm",
        payload: { action: "Delete file" },
        timeout: 1000,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Advance time to exactly the timeout moment
      vi.advanceTimersByTime(999);

      // Simulate user completion happening at the same moment as timeout
      dequeue(modalId, { type: "confirm", confirmed: true, cancelled: false });

      // Complete the timeout
      vi.advanceTimersByTime(1);

      // Promise should resolve only once with user response (first to complete)
      const result = await promise;

      // The dequeue happened first (at 999ms), so it should win
      expect(result).toEqual({ type: "confirm", confirmed: true, cancelled: false });

      // Queue should be empty
      expect(useStore.getState().modalQueue).toHaveLength(0);

      // No error should be thrown (no double-resolution)
      // If there was a race condition, this test would fail with:
      // "Error: Promise already resolved"
    });

    it("should handle timeout winning the race condition", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue({
        type: "confirm",
        payload: { action: "Delete file" },
        timeout: 1000,
      });

      const modalId = useStore.getState().modalQueue[0].id;

      // Let timeout fire first
      vi.advanceTimersByTime(1000);

      // Try to dequeue after timeout - should be no-op
      dequeue(modalId, { type: "confirm", confirmed: true, cancelled: false });

      // Promise should resolve with timeout default (not confirmed)
      const result = await promise;
      expect(result).toEqual({ type: "confirm", confirmed: false, cancelled: true });

      // Queue should be empty (timeout already removed it)
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });
  });

  describe("queue behavior", () => {
    it("should handle multiple concurrent modals in FIFO order", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const p1 = enqueue({ type: "ask", payload: { message: "Q1" } });
      const p2 = enqueue({ type: "confirm", payload: { action: "Q2" } });
      const p3 = enqueue({ type: "input", payload: { prompt: "Q3" } });

      const queue = useStore.getState().modalQueue;
      expect(queue).toHaveLength(3);

      // Dequeue in order
      dequeue(queue[0].id, { type: "ask", value: "A1" });
      dequeue(queue[1].id, { type: "confirm", confirmed: true, cancelled: false });
      dequeue(queue[2].id, { type: "input", value: "A3" });

      const [r1, r2, r3] = await Promise.all([p1, p2, p3]);

      expect(r1).toEqual({ type: "ask", value: "A1" });
      expect(r2).toEqual({ type: "confirm", confirmed: true, cancelled: false });
      expect(r3).toEqual({ type: "input", value: "A3" });
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should handle dequeuing in reverse order", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const p1 = enqueue({ type: "ask", payload: { message: "Q1" } });
      const p2 = enqueue({ type: "ask", payload: { message: "Q2" } });
      const p3 = enqueue({ type: "ask", payload: { message: "Q3" } });

      const queue = useStore.getState().modalQueue;

      // Dequeue in reverse order
      dequeue(queue[2].id, { type: "ask", value: "A3" });
      dequeue(queue[1].id, { type: "ask", value: "A2" });
      dequeue(queue[0].id, { type: "ask", value: "A1" });

      const [r1, r2, r3] = await Promise.all([p1, p2, p3]);

      expect(r1).toEqual({ type: "ask", value: "A1" });
      expect(r2).toEqual({ type: "ask", value: "A2" });
      expect(r3).toEqual({ type: "ask", value: "A3" });
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should handle mixed cancel and dequeue operations", async () => {
      const { enqueue, dequeue, cancel } = useStore.getState();

      const p1 = enqueue({ type: "ask", payload: { message: "Q1" } });
      const p2 = enqueue({ type: "ask", payload: { message: "Q2" } });
      const p3 = enqueue({ type: "ask", payload: { message: "Q3" } });

      const queue = useStore.getState().modalQueue;

      // Cancel middle, dequeue others
      dequeue(queue[0].id, { type: "ask", value: "A1" });
      cancel(queue[1].id);
      dequeue(queue[2].id, { type: "ask", value: "A3" });

      const r1 = await p1;
      await expect(p2).rejects.toThrow("Modal cancelled by user");
      const r3 = await p3;

      expect(r1).toEqual({ type: "ask", value: "A1" });
      expect(r3).toEqual({ type: "ask", value: "A3" });
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });
  });

  describe("type safety", () => {
    it("should maintain type information through enqueue/dequeue", async () => {
      const { enqueue, dequeue } = useStore.getState();

      const promise = enqueue<{ message: string; options: string[] }>({
        type: "ask",
        payload: { message: "Pick one", options: ["A", "B", "C"] },
      });

      const queue = useStore.getState().modalQueue;
      const payload = queue[0].payload as { message: string; options: string[] };

      expect(payload.message).toBe("Pick one");
      expect(payload.options).toEqual(["A", "B", "C"]);

      dequeue(queue[0].id, { type: "ask", value: "B" });

      const result = await promise;
      expect(result.type).toBe("ask");
      if (result.type === "ask") {
        expect(result.value).toBe("B");
      }
    });
  });
});
