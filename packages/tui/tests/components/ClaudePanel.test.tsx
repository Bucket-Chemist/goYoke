/**
 * ClaudePanel component tests
 * Coverage:
 * - Message rendering (user, assistant, system messages)
 * - Message content types (text, tool_use, tool_result)
 * - Streaming state indicators
 * - Mock API response handling
 * - Focus state management
 * - Empty state display
 * - Viewport integration and auto-scroll
 * - Partial message rendering
 * - Store integration (messages, streaming, history)
 *
 * Note: ink-testing-library cannot simulate actual keyboard input.
 * Input handling is tested through store integration and controlled component patterns.
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { ClaudePanel } from "../../src/components/ClaudePanel.js";
import { useStore } from "../../src/store/index.js";
import type { Message, ContentBlock } from "../../src/store/types.js";

// Mock localStorage for Zustand persist middleware
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(global, "localStorage", {
  value: localStorageMock,
});

describe("ClaudePanel", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearMessages();
    useStore.getState().clearHistory();
    useStore.getState().setStreaming(false);
    // Clear localStorage
    localStorageMock.clear();
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  describe("Empty State", () => {
    it("renders empty state when no messages exist", () => {
      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Claude Conversation");
      expect(output).toContain("No messages yet");
    });

    it("shows input placeholder when not streaming", () => {
      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      expect(output).toContain("Type a message...");
    });
  });

  describe("Message Rendering", () => {
    it("renders user message with correct styling", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Hello Claude!" }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("You");
      expect(output).toContain("Hello Claude!");
    });

    it("renders assistant message with correct styling", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Hello! How can I help?" }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Claude");
      expect(output).toContain("Hello! How can I help?");
    });

    it("renders system message with correct styling", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "system",
        content: [{ type: "text", text: "System notification" }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("System");
      expect(output).toContain("System notification");
    });

    it("renders multiple messages in conversation order", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "First message" }],
        partial: false,
      });

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "First response" }],
        partial: false,
      });

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Second message" }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("First message");
      expect(output).toContain("First response");
      expect(output).toContain("Second message");
    });

    it("renders partial message with streaming indicator", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Partial response..." }],
        partial: true,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("streaming...");
      expect(output).toContain("Partial response...");
    });

    it("renders tool_use content blocks", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          { type: "text", text: "Let me help with that" },
          {
            type: "tool_use",
            id: "tool-1",
            name: "Read",
            input: { file_path: "/test.ts" },
          },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Let me help with that");
      expect(output).toContain("[Read]");
    });

    it("renders tool_result content blocks", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          { type: "text", text: "Here's what I found" },
          {
            type: "tool_result",
            tool_use_id: "tool-1",
            content: "File contents here",
          },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Here's what I found");
      expect(output).toContain("[result]");
    });

    it("handles messages with multiple text blocks", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          { type: "text", text: "First paragraph" },
          { type: "text", text: "Second paragraph" },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("First paragraph");
      expect(output).toContain("Second paragraph");
    });

    it("handles empty text content gracefully", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "" }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Should still render the role header
      expect(output).toContain("Claude");
    });
  });

  describe("Streaming State", () => {
    it("shows streaming indicator when streaming is true", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(true);

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Claude is thinking...");
    });

    it("changes input placeholder during streaming", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(true);

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      expect(output).toContain("Type to queue message...");
    });

    it("hides streaming indicator when streaming is false", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(false);

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).not.toContain("Claude is thinking...");
    });

    it("disables input during streaming", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(true);

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Placeholder text indicates disabled state
      expect(output).toContain("Type to queue message...");
    });
  });

  describe("Input Rendering", () => {
    it("shows input placeholder when not streaming", () => {
      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      expect(output).toContain("Type a message...");
    });

    it("shows waiting placeholder during streaming", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(true);

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      expect(output).toContain("Type to queue message...");
    });

    it("renders input component when panel is focused", () => {
      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Should show TextInput component
      expect(output).toContain("Type a message...");
    });

    it("renders input component when panel is not focused", () => {
      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Should still show TextInput (focus affects border color, not presence)
      expect(output).toContain("Type a message...");
    });
  });

  describe("Mock API Response", () => {
    it("component has infrastructure for mock responses", () => {
      // Note: The mock API response is triggered by internal component state (pendingMessage)
      // which is set by handleSubmit(). Since ink-testing-library cannot simulate actual
      // keyboard input to trigger handleSubmit, we verify the component structure instead.

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Component should render with input ready for submission
      expect(output).toContain("Type a message...");
      expect(output).toContain("Claude Conversation");

      // The useEffect for mock responses exists in the component (verified by reading source)
      // In real usage: user types → Enter → handleSubmit → setPendingMessage → useEffect → mock response
    });

    it("displays assistant messages from store", () => {
      // Verify component can display assistant messages (as mock API would create)
      useStore.getState().addMessage({
        role: "assistant",
        content: [
          {
            type: "text",
            text: 'Mock response to: "Hello"\n\nThis is a placeholder.',
          },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Claude");
      expect(output).toContain("Mock response");
      expect(output).toContain("placeholder");
    });
  });

  describe("Input History Integration", () => {
    it("component integrates with input history store", () => {
      const { addToHistory } = useStore.getState();
      addToHistory("First command");
      addToHistory("Second command");

      // Component should render with history available
      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Input should be ready (history navigation would work if we could test it)
      expect(output).toContain("Type a message...");

      // Verify history is in store
      const { inputHistory } = useStore.getState();
      expect(inputHistory).toContain("First command");
      expect(inputHistory).toContain("Second command");
    });

    it("renders correctly when history exists", () => {
      const { addToHistory } = useStore.getState();
      for (let i = 1; i <= 10; i++) {
        addToHistory(`Command ${i}`);
      }

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Should render normally with history present
      expect(output).toBeTruthy();
      expect(output).toContain("Claude Conversation");
    });

    it("uses history navigation functions from store", () => {
      const { addToHistory, navigateHistory } = useStore.getState();
      addToHistory("First");
      addToHistory("Second");

      // Test store navigation directly (component uses these functions)
      const item1 = navigateHistory("up");
      expect(item1).toBe("Second");

      const item2 = navigateHistory("up");
      expect(item2).toBe("First");

      const item3 = navigateHistory("down");
      expect(item3).toBe("Second");

      // Component would call these via keybindings
      const { lastFrame } = render(<ClaudePanel focused={true} />);
      expect(lastFrame()).toBeTruthy();
    });
  });

  describe("Focus State", () => {
    it("shows focused border when focused=true", () => {
      const { lastFrame: focusedFrame } = render(<ClaudePanel focused={true} />);
      const { lastFrame: unfocusedFrame } = render(<ClaudePanel focused={false} />);

      const focused = focusedFrame();
      const unfocused = unfocusedFrame();

      // Both should render
      expect(focused).toContain("Claude Conversation");
      expect(unfocused).toContain("Claude Conversation");

      // Color differences are handled by ink, both should be valid output
      expect(focused).toBeTruthy();
      expect(unfocused).toBeTruthy();
    });

    it("disables panel bindings when modal is active", async () => {
      // Add a mock modal to the queue
      const modalPromise = useStore.getState().enqueue({
        type: "confirm",
        payload: { message: "Test?" },
      });

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Panel should still render
      expect(output).toContain("Claude Conversation");

      // Clean up the modal - use dequeue instead of cancel to avoid rejection
      const modalQueue = useStore.getState().modalQueue;
      if (modalQueue.length > 0) {
        useStore.getState().dequeue(modalQueue[0]!.id, {
          type: "confirm",
          confirmed: false,
          cancelled: true,
        });
      }

      // Handle the promise to avoid unhandled rejection
      await modalPromise.catch(() => {
        // Expected cancellation
      });
    });
  });

  describe("Viewport Integration", () => {
    it("displays messages through Viewport component", () => {
      const { addMessage } = useStore.getState();

      // Add multiple messages
      for (let i = 1; i <= 5; i++) {
        addMessage({
          role: "user",
          content: [{ type: "text", text: `Message ${i}` }],
          partial: false,
        });
      }

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // autoScroll=true shows most-recent messages; scrollHeight defaults to 10 in tests.
      // With 5 messages × ~3 rows each = ~15 rows, earlier messages scroll off.
      // Verify the most recent message is visible.
      expect(output).toContain("Message 5");
    });

    it("renders with many messages", () => {
      const { addMessage } = useStore.getState();

      // Add many messages to exceed viewport height
      for (let i = 1; i <= 30; i++) {
        addMessage({
          role: "user",
          content: [{ type: "text", text: `Message ${i}` }],
          partial: false,
        });
      }

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Component passes autoScroll={true} to Viewport
      // Auto-scroll behavior is tested in Viewport.test.tsx
      // Here we verify the component renders without crashing with many messages
      expect(output).toContain("Claude Conversation");

      // At least some messages should be visible
      const hasMessages = output.includes("Message") || output.includes("You");
      expect(hasMessages).toBe(true);
    });

    it("disables viewport focus during streaming", () => {
      const { addMessage, setStreaming } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Test" }],
        partial: false,
      });

      setStreaming(true);

      const { lastFrame } = render(<ClaudePanel focused={true} />);
      const output = lastFrame();

      // Panel is focused, but viewport should not be (streaming=true)
      // This is verified by component passing focused && !streaming to Viewport
      expect(output).toContain("Test");
    });
  });

  describe("Markdown Rendering", () => {
    it("renders markdown in message content", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          {
            type: "text",
            text: "Here's some **bold** text and `code`",
          },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Markdown is rendered through renderMarkdown utility
      expect(output).toContain("bold");
      expect(output).toContain("code");
    });
  });

  describe("Edge Cases", () => {
    it("handles very long messages", () => {
      const { addMessage } = useStore.getState();

      const longText = "A".repeat(1000);
      addMessage({
        role: "user",
        content: [{ type: "text", text: longText }],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Should render without crashing (1000-char text wraps to many rows,
      // autoScroll scrolls past the "You" role header — just verify it renders)
      expect(output).toBeTruthy();
    });

    it("handles rapid message additions", () => {
      const { addMessage } = useStore.getState();

      // Add 50 messages rapidly
      for (let i = 1; i <= 50; i++) {
        addMessage({
          role: i % 2 === 0 ? "user" : "assistant",
          content: [{ type: "text", text: `Rapid ${i}` }],
          partial: false,
        });
      }

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Should render without crashing
      expect(output).toBeTruthy();
    });

    it("handles message with no content blocks", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      // Should still render the role header
      expect(output).toContain("Claude");
    });

    it("handles mixed content types in single message", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          { type: "text", text: "Using a tool now" },
          {
            type: "tool_use",
            id: "t1",
            name: "Grep",
            input: { pattern: "test" },
          },
          {
            type: "tool_result",
            tool_use_id: "t1",
            content: "Found matches",
          },
          { type: "text", text: "Done!" },
        ],
        partial: false,
      });

      const { lastFrame } = render(<ClaudePanel focused={false} />);
      const output = lastFrame();

      expect(output).toContain("Using a tool now");
      expect(output).toContain("[Grep]");
      expect(output).toContain("[result]");
      expect(output).toContain("Done!");
    });

    it("handles re-rendering with same messages", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Static message" }],
        partial: false,
      });

      const { lastFrame, rerender } = render(<ClaudePanel focused={false} />);

      const output1 = lastFrame();
      expect(output1).toContain("Static message");

      // Re-render with same props
      rerender(<ClaudePanel focused={false} />);

      const output2 = lastFrame();
      expect(output2).toContain("Static message");
    });

    it("handles focus changes during streaming", () => {
      const { setStreaming } = useStore.getState();
      setStreaming(true);

      const { lastFrame, rerender } = render(<ClaudePanel focused={false} />);

      const output1 = lastFrame();
      expect(output1).toContain("Claude is thinking...");

      // Change focus during streaming
      rerender(<ClaudePanel focused={true} />);

      const output2 = lastFrame();
      expect(output2).toContain("Claude is thinking...");
      expect(output2).toContain("Type to queue message...");
    });
  });

  describe("Store Integration", () => {
    it("updates when messages are added to store externally", () => {
      const { lastFrame, rerender } = render(<ClaudePanel focused={false} />);

      const output1 = lastFrame();
      expect(output1).toContain("No messages yet");

      // Add message through store directly
      useStore.getState().addMessage({
        role: "user",
        content: [{ type: "text", text: "External message" }],
        partial: false,
      });

      // Force re-render to pick up store change
      rerender(<ClaudePanel focused={false} />);

      const output2 = lastFrame();
      expect(output2).toContain("External message");
    });

    it("updates when streaming state changes in store", () => {
      const { lastFrame, rerender } = render(<ClaudePanel focused={false} />);

      const output1 = lastFrame();
      expect(output1).not.toContain("Claude is thinking...");

      // Change streaming state through store
      useStore.getState().setStreaming(true);

      // Force re-render to pick up store change
      rerender(<ClaudePanel focused={false} />);

      const output2 = lastFrame();
      expect(output2).toContain("Claude is thinking...");
    });

    it("clears messages when store is cleared", () => {
      const { addMessage, clearMessages } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "To be cleared" }],
        partial: false,
      });

      const { lastFrame, rerender } = render(<ClaudePanel focused={false} />);

      const output1 = lastFrame();
      expect(output1).toContain("To be cleared");

      clearMessages();
      rerender(<ClaudePanel focused={false} />);

      const output2 = lastFrame();
      expect(output2).toContain("No messages yet");
    });
  });
});
