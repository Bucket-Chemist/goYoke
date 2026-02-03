/**
 * E2E Integration Tests
 * Tests full conversation flow: user message → Claude response → tool call → modal → continue
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { askUserTool } from "../../src/mcp/tools/askUser.js";
import { confirmActionTool } from "../../src/mcp/tools/confirmAction.js";
import type { ModalResponse } from "../../src/store/slices/modal.js";

/**
 * Simulate a conversation turn where Claude calls a tool
 */
describe("E2E Flow - Complete Conversation", () => {
  beforeEach(() => {
    // Clear relevant store slices
    useStore.getState().clearMessages();
    useStore.setState({
      modalQueue: [],
    });
  });

  it("should handle full message → tool call → response flow", async () => {
    // 1. User sends message (simulated)
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: "Ask the user what their favorite color is" }],
      partial: false,
    });

    expect(useStore.getState().messages).toHaveLength(1);

    // 2. Claude would respond with tool use (simulated)
    // In real flow, this comes from SDK stream
    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "tool_use",
          id: "tool_1",
          name: "ask_user",
          input: { message: "What is your favorite color?" },
        },
      ],
      partial: false,
    });

    expect(useStore.getState().messages).toHaveLength(2);

    // 3. Tool executes and enqueues modal
    const toolPromise = askUserTool.handler({
      message: "What is your favorite color?",
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // Modal should be in queue
    const modal = useStore.getState().modalQueue[0];
    expect(modal).toBeDefined();
    expect(modal?.type).toBe("ask");

    // 4. User responds via modal
    if (modal) {
      useStore.getState().dequeue(modal.id, {
        type: "ask",
        value: "Purple",
      });
    }

    // 5. Tool returns result
    const toolResult = await toolPromise;
    expect(toolResult.content[0]?.text).toBe("Purple");

    // 6. Tool result would be sent back to Claude (simulated)
    useStore.getState().addMessage({
      role: "user",
      content: [
        {
          type: "text",
          text: `Tool result: ${toolResult.content[0]?.text}`,
        },
      ],
      partial: false,
    });

    // 7. Claude's final response (simulated)
    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "text",
          text: "The user's favorite color is Purple.",
        },
      ],
      partial: false,
    });

    // Verify complete conversation
    const messages = useStore.getState().messages;
    expect(messages).toHaveLength(4);
    expect(messages[0]?.role).toBe("user");
    expect(messages[1]?.role).toBe("assistant");
    expect(messages[2]?.role).toBe("user"); // Tool result
    expect(messages[3]?.role).toBe("assistant");

    // Queue should be empty
    expect(useStore.getState().modalQueue).toHaveLength(0);
  });

  it("should handle confirmation flow in conversation", async () => {
    // User asks Claude to delete something
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: "Delete config.yaml" }],
      partial: false,
    });

    // Claude calls confirm_action
    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "tool_use",
          id: "tool_2",
          name: "confirm_action",
          input: { action: "Delete config.yaml", destructive: true },
        },
      ],
      partial: false,
    });

    // Tool executes
    const toolPromise = confirmActionTool.handler({
      action: "Delete config.yaml",
      destructive: true,
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // User confirms
    const modal = useStore.getState().modalQueue[0];
    if (modal) {
      useStore.getState().dequeue(modal.id, {
        type: "confirm",
        confirmed: true,
        cancelled: false,
      });
    }

    const toolResult = await toolPromise;
    const response = JSON.parse(toolResult.content[0]?.text ?? "{}");
    expect(response.confirmed).toBe(true);

    // Tool result sent to Claude
    useStore.getState().addMessage({
      role: "user",
      content: [
        {
          type: "text",
          text: `Confirmation: ${toolResult.content[0]?.text}`,
        },
      ],
      partial: false,
    });

    // Claude completes action
    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "text",
          text: "File deleted successfully.",
        },
      ],
      partial: false,
    });

    expect(useStore.getState().messages).toHaveLength(4);
  });

  it("should handle user cancellation in conversation flow", async () => {
    // Start flow
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: "Reset the database" }],
      partial: false,
    });

    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "tool_use",
          id: "tool_3",
          name: "confirm_action",
          input: { action: "Reset database", destructive: true },
        },
      ],
      partial: false,
    });

    const toolPromise = confirmActionTool.handler({
      action: "Reset database",
      destructive: true,
    });

    await new Promise((resolve) => setTimeout(resolve, 0));

    // User cancels (Escape)
    const modal = useStore.getState().modalQueue[0];
    if (modal) {
      useStore.getState().dequeue(modal.id, {
        type: "confirm",
        confirmed: false,
        cancelled: true,
      });
    }

    const toolResult = await toolPromise;
    const response = JSON.parse(toolResult.content[0]?.text ?? "{}");
    expect(response.cancelled).toBe(true);

    // Claude acknowledges cancellation
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: toolResult.content[0]?.text ?? "" }],
      partial: false,
    });

    useStore.getState().addMessage({
      role: "assistant",
      content: [
        {
          type: "text",
          text: "Database reset cancelled.",
        },
      ],
      partial: false,
    });

    expect(useStore.getState().messages).toHaveLength(4);
  });

  it("should maintain conversation state across multiple tool calls", async () => {
    // First tool call
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: "What's your name?" }],
      partial: false,
    });

    const tool1Promise = askUserTool.handler({
      message: "What is your name?",
    });

    // Use setTimeout to respond
    setTimeout(() => {
      const modal = useStore.getState().modalQueue[0];
      if (modal) {
        useStore.getState().dequeue(modal.id, { type: "ask", value: "Alice" });
      }
    }, 10);

    await tool1Promise;
    useStore.getState().addMessage({
      role: "assistant",
      content: [{ type: "text", text: "Nice to meet you, Alice!" }],
      partial: false,
    });

    // Second tool call
    useStore.getState().addMessage({
      role: "user",
      content: [{ type: "text", text: "What's your email?" }],
      partial: false,
    });

    const tool2Promise = askUserTool.handler({
      message: "What is your email?",
    });

    // Use setTimeout to respond
    setTimeout(() => {
      const modal = useStore.getState().modalQueue[0];
      if (modal) {
        useStore.getState().dequeue(modal.id, { type: "ask", value: "alice@test.com" });
      }
    }, 10);

    await tool2Promise;
    useStore.getState().addMessage({
      role: "assistant",
      content: [{ type: "text", text: "Got it, alice@test.com" }],
      partial: false,
    });

    // Verify full conversation history
    const messages = useStore.getState().messages;
    expect(messages).toHaveLength(4);
    expect(messages[0]?.content[0]).toMatchObject({
      type: "text",
      text: "What's your name?",
    });
    expect(messages[3]?.content[0]).toMatchObject({
      type: "text",
      text: "Got it, alice@test.com",
    });
  });

  it("should handle streaming response with tool call", async () => {
    // Start streaming
    useStore.getState().setStreaming(true);

    // Partial text response
    useStore.getState().addMessage({
      role: "assistant",
      content: [{ type: "text", text: "Let me ask the user..." }],
      partial: true,
    });

    let messages = useStore.getState().messages;
    expect(messages).toHaveLength(1);
    expect(messages[0]?.partial).toBe(true);

    // Tool use arrives in stream - updateLastMessage sets partial: false
    useStore.getState().updateLastMessage([
      { type: "text", text: "Let me ask the user..." },
      {
        type: "tool_use",
        id: "tool_stream",
        name: "ask_user",
        input: { message: "Confirm?" },
      },
    ]);

    messages = useStore.getState().messages;
    expect(messages).toHaveLength(1);
    expect(messages[0]?.content).toHaveLength(2);
    expect(messages[0]?.partial).toBe(false); // updateLastMessage sets partial: false

    // Tool executes
    const toolPromise = askUserTool.handler({ message: "Confirm?" });

    setTimeout(() => {
      const modal = useStore.getState().modalQueue[0];
      if (modal) {
        useStore.getState().dequeue(modal.id, { type: "ask", value: "Yes" });
      }
    }, 10);

    await toolPromise;

    // Stream completes
    useStore.getState().setStreaming(false);

    const finalMessages = useStore.getState().messages;
    expect(finalMessages[0]?.partial).toBe(false);
    expect(finalMessages).toHaveLength(1);
  });
});
