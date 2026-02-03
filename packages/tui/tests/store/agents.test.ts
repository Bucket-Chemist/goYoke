/**
 * Unit tests for agents slice
 * Tests: tree operations, parent/child relationships, agent lifecycle
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store";
import type { Agent } from "../../src/store/types";

describe("Agents Slice", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearAgents();
  });

  describe("addAgent", () => {
    it("should add an agent with auto-generated startTime", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-3-5-sonnet",
        tier: "sonnet",
        status: "running",
      });

      const agent = useStore.getState().agents.get("agent-1");
      expect(agent).toBeDefined();
      expect(agent?.startTime).toBeGreaterThan(0);
      expect(agent?.model).toBe("claude-3-5-sonnet");
      expect(agent?.tier).toBe("sonnet");
    });

    it("should set rootAgentId for first agent with no parent", () => {
      useStore.getState().addAgent({
        id: "root-agent",
        parentId: null,
        model: "claude-3-5-sonnet",
        tier: "sonnet",
        status: "running",
      });

      expect(useStore.getState().rootAgentId).toBe("root-agent");
    });

    it("should not change rootAgentId for child agents", () => {
      useStore.getState().addAgent({
        id: "root-agent",
        parentId: null,
        model: "claude-3-5-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "child-agent",
        parentId: "root-agent",
        model: "claude-3-5-haiku",
        tier: "haiku",
        status: "running",
      });

      expect(useStore.getState().rootAgentId).toBe("root-agent");
    });

    it("should handle multiple agents", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-opus",
        tier: "opus",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "agent-2",
        parentId: "agent-1",
        model: "claude-sonnet",
        tier: "sonnet",
        status: "spawning",
      });

      const { agents } = useStore.getState();
      expect(agents.size).toBe(2);
      expect(agents.get("agent-2")?.parentId).toBe("agent-1");
    });
  });

  describe("updateAgent", () => {
    it("should update agent status", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().updateAgent("agent-1", { status: "complete", endTime: Date.now() });

      const agent = useStore.getState().agents.get("agent-1");
      expect(agent?.status).toBe("complete");
      expect(agent?.endTime).toBeDefined();
    });

    it("should update token usage", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().updateAgent("agent-1", {
        tokenUsage: { input: 1000, output: 500 },
      });

      const agent = useStore.getState().agents.get("agent-1");
      expect(agent?.tokenUsage).toEqual({ input: 1000, output: 500 });
    });

    it("should handle non-existent agent gracefully", () => {
      useStore.getState().updateAgent("non-existent", { status: "complete" });

      expect(useStore.getState().agents.size).toBe(0);
    });

    it("should preserve other agent properties on partial update", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
        description: "Test agent",
      });

      useStore.getState().updateAgent("agent-1", { status: "complete" });

      const agent = useStore.getState().agents.get("agent-1");
      expect(agent?.description).toBe("Test agent");
      expect(agent?.model).toBe("claude-sonnet");
      expect(agent?.status).toBe("complete");
    });
  });

  describe("selectAgent", () => {
    it("should select an agent", () => {
      useStore.getState().selectAgent("agent-1");

      expect(useStore.getState().selectedAgentId).toBe("agent-1");
    });

    it("should allow deselecting by passing null", () => {
      useStore.getState().selectAgent("agent-1");
      useStore.getState().selectAgent(null);

      expect(useStore.getState().selectedAgentId).toBeNull();
    });
  });

  describe("getAgentChildren", () => {
    it("should return all children of an agent", () => {
      useStore.getState().addAgent({
        id: "parent",
        parentId: null,
        model: "claude-opus",
        tier: "opus",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "child-1",
        parentId: "parent",
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "child-2",
        parentId: "parent",
        model: "claude-haiku",
        tier: "haiku",
        status: "complete",
      });

      useStore.getState().addAgent({
        id: "other",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      const children = useStore.getState().getAgentChildren("parent");

      expect(children).toHaveLength(2);
      expect(children.map((c) => c.id)).toContain("child-1");
      expect(children.map((c) => c.id)).toContain("child-2");
      expect(children.map((c) => c.id)).not.toContain("other");
    });

    it("should return empty array for agent with no children", () => {
      useStore.getState().addAgent({
        id: "lonely",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      const children = useStore.getState().getAgentChildren("lonely");

      expect(children).toHaveLength(0);
    });

    it("should return empty array for non-existent agent", () => {
      const children = useStore.getState().getAgentChildren("non-existent");

      expect(children).toHaveLength(0);
    });
  });

  describe("clearAgents", () => {
    it("should clear all agents and reset state", () => {
      useStore.getState().addAgent({
        id: "agent-1",
        parentId: null,
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().selectAgent("agent-1");

      useStore.getState().clearAgents();

      const state = useStore.getState();
      expect(state.agents.size).toBe(0);
      expect(state.selectedAgentId).toBeNull();
      expect(state.rootAgentId).toBeNull();
    });
  });

  describe("agent tree structure", () => {
    it("should maintain parent-child relationships correctly", () => {
      // Build a tree:
      //     root
      //     ├── child-1
      //     │   └── grandchild-1
      //     └── child-2

      useStore.getState().addAgent({
        id: "root",
        parentId: null,
        model: "claude-opus",
        tier: "opus",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "child-1",
        parentId: "root",
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "child-2",
        parentId: "root",
        model: "claude-sonnet",
        tier: "sonnet",
        status: "running",
      });

      useStore.getState().addAgent({
        id: "grandchild-1",
        parentId: "child-1",
        model: "claude-haiku",
        tier: "haiku",
        status: "complete",
      });

      const rootChildren = useStore.getState().getAgentChildren("root");
      expect(rootChildren).toHaveLength(2);

      const child1Children = useStore.getState().getAgentChildren("child-1");
      expect(child1Children).toHaveLength(1);
      expect(child1Children[0].id).toBe("grandchild-1");

      const child2Children = useStore.getState().getAgentChildren("child-2");
      expect(child2Children).toHaveLength(0);
    });
  });
});
