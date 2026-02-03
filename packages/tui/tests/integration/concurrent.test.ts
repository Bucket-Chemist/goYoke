/**
 * Concurrent Tool Calls Integration Tests
 * Tests modal queue behavior when multiple tools are called in parallel
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store/index.js";
import { askUserTool } from "../../src/mcp/tools/askUser.js";
import { confirmActionTool } from "../../src/mcp/tools/confirmAction.js";
import { requestInputTool } from "../../src/mcp/tools/requestInput.js";
import { selectOptionTool } from "../../src/mcp/tools/selectOption.js";
import type { ModalResponse } from "../../src/store/slices/modal.js";

describe("Concurrent Tool Calls", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
  });

  it("should queue multiple tool calls in FIFO order", async () => {
    // Spawn 3 tools simultaneously
    const promise1 = askUserTool.handler({ message: "Question 1" });
    const promise2 = askUserTool.handler({ message: "Question 2" });
    const promise3 = askUserTool.handler({ message: "Question 3" });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // All 3 should be in queue
    const queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(3);
    expect(queue[0]?.payload).toMatchObject({ message: "Question 1" });
    expect(queue[1]?.payload).toMatchObject({ message: "Question 2" });
    expect(queue[2]?.payload).toMatchObject({ message: "Question 3" });

    // Respond in order
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "Answer 1" });
    useStore.getState().dequeue(queue[1]!.id, { type: "ask", value: "Answer 2" });
    useStore.getState().dequeue(queue[2]!.id, { type: "ask", value: "Answer 3" });

    // All should resolve correctly
    const results = await Promise.all([promise1, promise2, promise3]);
    expect(results[0]?.content[0]?.text).toBe("Answer 1");
    expect(results[1]?.content[0]?.text).toBe("Answer 2");
    expect(results[2]?.content[0]?.text).toBe("Answer 3");

    // Queue should be empty
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle out-of-order responses correctly", async () => {
    // Spawn 3 tools
    const promise1 = askUserTool.handler({ message: "A" });
    const promise2 = askUserTool.handler({ message: "B" });
    const promise3 = askUserTool.handler({ message: "C" });

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(3);

    // Respond out of order: 2nd, 3rd, 1st
    useStore.getState().dequeue(queue[1]!.id, { type: "ask", value: "Response B" });
    useStore.getState().dequeue(queue[2]!.id, { type: "ask", value: "Response C" });
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "Response A" });

    // Each should get correct response
    const results = await Promise.all([promise1, promise2, promise3]);
    expect(results[0]?.content[0]?.text).toBe("Response A");
    expect(results[1]?.content[0]?.text).toBe("Response B");
    expect(results[2]?.content[0]?.text).toBe("Response C");

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle mixed tool types concurrently", async () => {
    // Different tools
    const askPromise = askUserTool.handler({ message: "Your name?" });
    const confirmPromise = confirmActionTool.handler({ action: "Delete file" });
    const inputPromise = requestInputTool.handler({ prompt: "Enter email" });
    const selectPromise = selectOptionTool.handler({
      message: "Choose one",
      options: [
        { label: "A", value: "a" },
        { label: "B", value: "b" },
      ],
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(4);

    // Verify queue contains all types
    expect(queue.map((m) => m.type)).toEqual(["ask", "confirm", "input", "select"]);

    // Respond to each
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "Alice" });
    useStore.getState().dequeue(queue[1]!.id, {
      type: "confirm",
      confirmed: true,
      cancelled: false,
    });
    useStore.getState().dequeue(queue[2]!.id, {
      type: "input",
      value: "alice@test.com",
    });
    useStore.getState().dequeue(queue[3]!.id, {
      type: "select",
      selected: "b",
      index: 1,
    });

    // All resolve correctly
    const [askResult, confirmResult, inputResult, selectResult] = await Promise.all([
      askPromise,
      confirmPromise,
      inputPromise,
      selectPromise,
    ]);

    expect(askResult.content[0]?.text).toBe("Alice");
    expect(JSON.parse(confirmResult.content[0]?.text ?? "{}")).toEqual({
      confirmed: true,
      cancelled: false,
    });
    expect(inputResult.content[0]?.text).toBe("alice@test.com");
    expect(JSON.parse(selectResult.content[0]?.text ?? "{}")).toEqual({
      selected: "b",
      index: 1,
    });

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should maintain independence between concurrent calls", async () => {
    // Spawn many tools
    const promises = Array.from({ length: 10 }, (_, i) =>
      askUserTool.handler({ message: `Question ${i}` })
    );

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(10);

    // Respond to all
    queue.forEach((modal, i) => {
      useStore.getState().dequeue(modal.id, {
        type: "ask",
        value: `Answer ${i}`,
      });
    });

    const results = await Promise.all(promises);

    // Each should get its corresponding answer
    results.forEach((result, i) => {
      expect(result.content[0]?.text).toBe(`Answer ${i}`);
    });

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle one cancellation without affecting others", async () => {
    const promise1 = askUserTool.handler({ message: "Q1" });
    const promise2 = confirmActionTool.handler({ action: "Action 2" });
    const promise3 = askUserTool.handler({ message: "Q3" });

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;

    // Respond to first and third, cancel second
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "A1" });
    useStore.getState().cancel(queue[1]!.id); // Cancel confirm
    useStore.getState().dequeue(queue[2]!.id, { type: "ask", value: "A3" });

    // First and third resolve successfully
    const result1 = await promise1;
    const result3 = await promise3;
    expect(result1.content[0]?.text).toBe("A1");
    expect(result3.content[0]?.text).toBe("A3");

    // Second rejects
    await expect(promise2).rejects.toThrow("Modal cancelled by user");

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle rapid sequential additions", async () => {
    const promises: Promise<{ content: Array<{ type: string; text: string }> }>[] = [];

    // Add 5 tools rapidly
    for (let i = 0; i < 5; i++) {
      promises.push(askUserTool.handler({ message: `Rapid ${i}` }));
    }

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(5);

    // Verify order preserved
    queue.forEach((modal, i) => {
      expect(modal.payload).toMatchObject({ message: `Rapid ${i}` });
    });

    // Respond to all
    queue.forEach((modal, i) => {
      useStore.getState().dequeue(modal.id, {
        type: "ask",
        value: `Response ${i}`,
      });
    });

    const results = await Promise.all(promises);
    results.forEach((result, i) => {
      expect(result.content[0]?.text).toBe(`Response ${i}`);
    });
  });

  it("should not interfere with queue when processing concurrently", async () => {
    const promise1 = askUserTool.handler({ message: "First" });
    const promise2 = askUserTool.handler({ message: "Second" });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // Respond to first
    let queue = useStore.getState().modalQueue;
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "First done" });

    await promise1;

    // Second should still be in queue
    queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(1);
    expect(queue[0]?.payload).toMatchObject({ message: "Second" });

    // Add third while second is pending
    const promise3 = askUserTool.handler({ message: "Third" });
    await new Promise((resolve) => setTimeout(resolve, 0));

    queue = useStore.getState().modalQueue;
    expect(queue).toHaveLength(2);
    expect(queue[0]?.payload).toMatchObject({ message: "Second" });
    expect(queue[1]?.payload).toMatchObject({ message: "Third" });

    // Complete both
    useStore.getState().dequeue(queue[0]!.id, { type: "ask", value: "Second done" });
    useStore.getState().dequeue(queue[1]!.id, { type: "ask", value: "Third done" });

    const [result2, result3] = await Promise.all([promise2, promise3]);
    expect(result2.content[0]?.text).toBe("Second done");
    expect(result3.content[0]?.text).toBe("Third done");

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle all concurrent tools completing at once", async () => {
    const promises = [
      askUserTool.handler({ message: "A" }),
      askUserTool.handler({ message: "B" }),
      askUserTool.handler({ message: "C" }),
    ];

    await new Promise((resolve) => setTimeout(resolve, 0));

    const queue = useStore.getState().modalQueue;

    // Respond to all at once
    queue[0] && useStore.getState().dequeue(queue[0].id, { type: "ask", value: "A" });
    queue[1] && useStore.getState().dequeue(queue[1].id, { type: "ask", value: "B" });
    queue[2] && useStore.getState().dequeue(queue[2].id, { type: "ask", value: "C" });

    // All should resolve
    const results = await Promise.all(promises);
    expect(results[0]?.content[0]?.text).toBe("A");
    expect(results[1]?.content[0]?.text).toBe("B");
    expect(results[2]?.content[0]?.text).toBe("C");

    expect(useStore.getState().modalQueue).toHaveLength(0);
  });
});
