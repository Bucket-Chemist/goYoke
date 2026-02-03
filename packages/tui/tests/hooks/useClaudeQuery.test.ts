/**
 * Tests for useClaudeQuery hook
 * Verifies store integration and error classification
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { useStore } from "../../src/store/index.js";

// Mock the Claude SDK query function
vi.mock("@anthropic-ai/claude-agent-sdk", () => ({
  query: vi.fn(),
}));

// Mock MCP server
vi.mock("../../src/mcp/server.js", () => ({
  mcpServer: {
    name: "test-server",
    version: "1.0.0",
    tools: [],
  },
}));

describe("useClaudeQuery", () => {
  beforeEach(() => {
    // Reset store state
    useStore.setState({
      messages: [],
      agents: {},
      selectedAgentId: null,
      rootAgentId: null,
      sessionId: null,
      totalCost: 0,
      tokenCount: { input: 0, output: 0 },
      streaming: false,
    });

    // Clear all mocks
    vi.clearAllMocks();
  });

  describe("Store Integration", () => {
    it("should initialize store with default state", () => {
      const state = useStore.getState();

      expect(state.messages).toEqual([]);
      expect(state.sessionId).toBeNull();
      expect(state.totalCost).toBe(0);
      expect(state.streaming).toBe(false);
    });

    it("should add messages to store", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Hello" }],
        partial: false,
      });

      const state = useStore.getState();
      expect(state.messages.length).toBe(1);
      expect(state.messages[0].role).toBe("user");
      expect(state.messages[0].content[0]).toMatchObject({
        type: "text",
        text: "Hello",
      });
    });

    it("should update session data", () => {
      const { updateSession } = useStore.getState();

      updateSession({
        id: "test-session-123",
        cost: 0.005,
      });

      const state = useStore.getState();
      expect(state.sessionId).toBe("test-session-123");
      expect(state.totalCost).toBe(0.005);
    });

    it("should increment cost", () => {
      const { incrementCost } = useStore.getState();

      incrementCost(0.001);
      incrementCost(0.002);

      const state = useStore.getState();
      expect(state.totalCost).toBe(0.003);
    });

    it("should add tokens", () => {
      const { addTokens } = useStore.getState();

      addTokens({ input: 100, output: 200 });
      addTokens({ input: 50, output: 100 });

      const state = useStore.getState();
      expect(state.tokenCount.input).toBe(150);
      expect(state.tokenCount.output).toBe(300);
    });

    it("should manage streaming state", () => {
      const { setStreaming } = useStore.getState();

      setStreaming(true);
      expect(useStore.getState().streaming).toBe(true);

      setStreaming(false);
      expect(useStore.getState().streaming).toBe(false);
    });
  });

  describe("Error Classification", () => {
    it("should identify network errors", () => {
      const error = new Error("ECONNREFUSED: Connection refused");
      expect(error.message).toContain("ECONNREFUSED");
    });

    it("should identify authentication errors", () => {
      const error = new Error("401: Invalid API key");
      expect(error.message).toContain("401");
    });

    it("should identify rate limit errors", () => {
      const error = new Error("429: Rate limit exceeded");
      expect(error.message).toContain("429");
    });

    it("should identify server errors", () => {
      const error = new Error("500: Internal server error");
      expect(error.message).toContain("500");
    });
  });

  describe("Message State Management", () => {
    it("should update last message", () => {
      const { addMessage, updateLastMessage } = useStore.getState();

      // Add initial message
      addMessage({
        role: "assistant",
        content: [{ type: "text", text: "Hello" }],
        partial: true,
      });

      // Update it
      updateLastMessage([{ type: "text", text: "Hello! How can I help?" }]);

      const state = useStore.getState();
      expect(state.messages.length).toBe(1);
      expect(state.messages[0].content[0]).toMatchObject({
        type: "text",
        text: "Hello! How can I help?",
      });
      expect(state.messages[0].partial).toBe(false);
    });

    it("should handle tool_use content blocks", () => {
      const { addMessage } = useStore.getState();

      addMessage({
        role: "assistant",
        content: [
          {
            type: "tool_use",
            id: "tool-123",
            name: "ask_user",
            input: { message: "What is your name?" },
          },
        ],
        partial: false,
      });

      const state = useStore.getState();
      expect(state.messages[0].content[0]).toMatchObject({
        type: "tool_use",
        id: "tool-123",
        name: "ask_user",
        input: { message: "What is your name?" },
      });
    });

    it("should clear messages", () => {
      const { addMessage, clearMessages } = useStore.getState();

      addMessage({
        role: "user",
        content: [{ type: "text", text: "Test" }],
        partial: false,
      });

      expect(useStore.getState().messages.length).toBe(1);

      clearMessages();

      expect(useStore.getState().messages.length).toBe(0);
    });
  });

  describe("Agent Management", () => {
    it("should add agents", () => {
      const { addAgent } = useStore.getState();

      addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet-4",
        tier: "sonnet",
        status: "running",
        description: "Test agent",
      });

      const state = useStore.getState();
      expect(state.agents["agent-1"]).toBeDefined();
      expect(state.agents["agent-1"].model).toBe("claude-sonnet-4");
      expect(state.agents["agent-1"].tier).toBe("sonnet");
    });

    it("should update agents", () => {
      const { addAgent, updateAgent } = useStore.getState();

      addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet-4",
        tier: "sonnet",
        status: "running",
      });

      updateAgent("agent-1", {
        status: "complete",
        endTime: Date.now(),
      });

      const state = useStore.getState();
      expect(state.agents["agent-1"].status).toBe("complete");
      expect(state.agents["agent-1"].endTime).toBeDefined();
    });

    it("should track root agent", () => {
      const { addAgent } = useStore.getState();

      addAgent({
        id: "root-agent",
        parentId: null,
        model: "claude-sonnet-4",
        tier: "sonnet",
        status: "running",
      });

      const state = useStore.getState();
      expect(state.rootAgentId).toBe("root-agent");
    });

    it("should get agent children", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      addAgent({
        id: "parent",
        parentId: null,
        model: "claude-sonnet-4",
        tier: "sonnet",
        status: "running",
      });

      addAgent({
        id: "child-1",
        parentId: "parent",
        model: "claude-haiku-4",
        tier: "haiku",
        status: "running",
      });

      addAgent({
        id: "child-2",
        parentId: "parent",
        model: "claude-haiku-4",
        tier: "haiku",
        status: "running",
      });

      const children = getAgentChildren("parent");
      expect(children.length).toBe(2);
      expect(children.map((c) => c.id)).toContain("child-1");
      expect(children.map((c) => c.id)).toContain("child-2");
    });
  });

  describe("Session Management", () => {
    it("should clear session", () => {
      const { updateSession, incrementCost, addTokens, clearSession } =
        useStore.getState();

      updateSession({ id: "test-session" });
      incrementCost(0.01);
      addTokens({ input: 100, output: 200 });

      clearSession();

      const state = useStore.getState();
      expect(state.sessionId).toBeNull();
      expect(state.totalCost).toBe(0);
      expect(state.tokenCount.input).toBe(0);
      expect(state.tokenCount.output).toBe(0);
    });
  });
});
