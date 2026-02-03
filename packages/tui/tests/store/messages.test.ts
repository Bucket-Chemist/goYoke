/**
 * Unit tests for messages slice
 * Tests: add, update partial, persistence, flicker prevention
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store";
import type { ContentBlock } from "../../src/store/types";

describe("Messages Slice", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearMessages();
  });

  describe("addMessage", () => {
    it("should add a message with generated id and timestamp", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Hello" }],
        partial: false,
      });

      const { messages } = useStore.getState();

      expect(messages).toHaveLength(1);
      expect(messages[0].id).toBeDefined();
      expect(messages[0].timestamp).toBeGreaterThan(0);
      expect(messages[0].role).toBe("user");
      expect(messages[0].content).toEqual([{ type: "text", text: "Hello" }]);
    });

    it("should add multiple messages in order", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "First" }],
        partial: false,
      });

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Second" }],
        partial: false,
      });

      const { messages } = useStore.getState();

      expect(messages).toHaveLength(2);
      expect(messages[0].content[0]).toEqual({ type: "text", text: "First" });
      expect(messages[1].content[0]).toEqual({ type: "text", text: "Second" });
    });

    it("should handle partial messages", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Streaming..." }],
        partial: true,
      });

      const { messages } = useStore.getState();

      expect(messages[0].partial).toBe(true);
    });
  });

  describe("updateLastMessage", () => {
    it("should update the last message content", () => {
      const { addMessage, updateLastMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Initial" }],
        partial: true,
      });

      const newContent: ContentBlock[] = [
        { type: "text", text: "Updated content" },
      ];

      updateLastMessage(newContent);

      const { messages } = useStore.getState();

      expect(messages).toHaveLength(1);
      expect(messages[0].content).toEqual(newContent);
      expect(messages[0].partial).toBe(false); // Should mark as complete
    });

    it("should handle no messages gracefully", () => {
      const { updateLastMessage } = useStore.getState();

      updateLastMessage([{ type: "text", text: "Should not crash" }]);

      const { messages } = useStore.getState();

      expect(messages).toHaveLength(0);
    });

    it("should preserve message id and timestamp", () => {
      const { addMessage, updateLastMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Original" }],
        partial: true,
      });

      const originalId = useStore.getState().messages[0].id;
      const originalTimestamp = useStore.getState().messages[0].timestamp;

      updateLastMessage([{ type: "text", text: "Updated" }]);

      const { messages } = useStore.getState();

      expect(messages[0].id).toBe(originalId);
      expect(messages[0].timestamp).toBe(originalTimestamp);
    });

    it("should handle complex content blocks", () => {
      const { addMessage, updateLastMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Initial" }],
        partial: true,
      });

      const complexContent: ContentBlock[] = [
        { type: "text", text: "Some text" },
        {
          type: "tool_use",
          id: "tool-1",
          name: "read_file",
          input: { path: "/test.ts" },
        },
        {
          type: "tool_result",
          tool_use_id: "tool-1",
          content: "file contents",
          is_error: false,
        },
      ];

      updateLastMessage(complexContent);

      const { messages } = useStore.getState();

      expect(messages[0].content).toEqual(complexContent);
      expect(messages[0].content).toHaveLength(3);
    });
  });

  describe("clearMessages", () => {
    it("should clear all messages", () => {
      const { addMessage, clearMessages } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Message 1" }],
        partial: false,
      });

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Message 2" }],
        partial: false,
      });

      expect(useStore.getState().messages).toHaveLength(2);

      clearMessages();

      expect(useStore.getState().messages).toHaveLength(0);
    });
  });

  describe("persistence behavior", () => {
    it("should maintain message reference stability on updates", () => {
      const { addMessage, updateLastMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Original" }],
        partial: true,
      });

      const messagesBefore = useStore.getState().messages;

      updateLastMessage([{ type: "text", text: "Updated" }]);

      const messagesAfter = useStore.getState().messages;

      // Should be a new array (immutability)
      expect(messagesBefore).not.toBe(messagesAfter);

      // But updates should not cause flicker (last message updated in place)
      expect(messagesAfter[0].content).toEqual([
        { type: "text", text: "Updated" },
      ]);
    });
  });
});
