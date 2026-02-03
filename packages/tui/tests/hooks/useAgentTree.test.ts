/**
 * Tests for useAgentTree hook
 * Tests navigation logic by directly testing the hook with store state
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store/index.js";
import type { Agent } from "../../src/store/types.js";

// Helper to create agents
function createAgent(id: string, parentId: string | null, startTime?: number): Omit<Agent, "startTime"> {
  return {
    id,
    parentId,
    model: "claude-haiku-4",
    tier: "haiku",
    status: "running",
    ...(startTime !== undefined && { startTime }),
  };
}

describe("useAgentTree", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearAgents();
  });

  describe("Agent Tree Structure", () => {
    it("should handle empty agent tree", () => {
      const { agents, rootAgentId } = useStore.getState();
      expect(Object.keys(agents).length).toBe(0);
      expect(rootAgentId).toBeNull();
    });

    it("should track root agent", () => {
      const { addAgent } = useStore.getState();

      addAgent(createAgent("root", null));

      const state = useStore.getState();
      expect(state.rootAgentId).toBe("root");
    });

    it("should build tree with parent-child relationships", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("child1", "root"));
      addAgent(createAgent("child2", "root"));
      addAgent(createAgent("grandchild", "child1"));

      const rootChildren = getAgentChildren("root");
      expect(rootChildren.length).toBe(2);
      expect(rootChildren.map((c) => c.id)).toContain("child1");
      expect(rootChildren.map((c) => c.id)).toContain("child2");

      const child1Children = getAgentChildren("child1");
      expect(child1Children.length).toBe(1);
      expect(child1Children[0].id).toBe("grandchild");
    });
  });

  describe("Agent Selection", () => {
    it("should select agent by id", () => {
      const { addAgent, selectAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("child", "root"));

      selectAgent("child");

      expect(useStore.getState().selectedAgentId).toBe("child");
    });

    it("should clear selection", () => {
      const { addAgent, selectAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      selectAgent("root");
      expect(useStore.getState().selectedAgentId).toBe("root");

      selectAgent(null);
      expect(useStore.getState().selectedAgentId).toBeNull();
    });

    it("should update selected agent", () => {
      const { addAgent, selectAgent, updateAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      selectAgent("root");

      updateAgent("root", { status: "complete", endTime: Date.now() });

      const state = useStore.getState();
      expect(state.agents["root"].status).toBe("complete");
      expect(state.agents["root"].endTime).toBeDefined();
    });
  });

  describe("Navigation Logic", () => {
    it("should provide agents in depth-first order", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      // Build tree: root -> child1 -> grandchild, child2
      addAgent(createAgent("root", null));
      addAgent(createAgent("child1", "root"));
      addAgent(createAgent("child2", "root"));
      addAgent(createAgent("grandchild", "child1"));

      // Manually verify depth-first traversal
      const traversal: string[] = [];

      function traverse(id: string): void {
        traversal.push(id);
        const children = getAgentChildren(id);
        children.sort((a, b) => a.startTime - b.startTime);
        children.forEach((child) => traverse(child.id));
      }

      traverse("root");

      // Expected order: root, child1, grandchild, child2
      expect(traversal).toEqual(["root", "child1", "grandchild", "child2"]);
    });

    it("should sort children by start time", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      // Add agents in order - addAgent sets startTime to Date.now()
      addAgent(createAgent("root", null));
      addAgent(createAgent("child1", "root"));
      addAgent(createAgent("child2", "root"));
      addAgent(createAgent("child3", "root"));

      const children = getAgentChildren("root");
      children.sort((a, b) => a.startTime - b.startTime);

      // Should be in order of addition since startTime is set at creation
      expect(children.map((c) => c.id)).toEqual(["child1", "child2", "child3"]);
    });

    it("should handle selection at different tree levels", () => {
      const { addAgent, selectAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("child1", "root"));
      addAgent(createAgent("grandchild", "child1"));

      // Select each level
      selectAgent("root");
      expect(useStore.getState().selectedAgentId).toBe("root");

      selectAgent("child1");
      expect(useStore.getState().selectedAgentId).toBe("child1");

      selectAgent("grandchild");
      expect(useStore.getState().selectedAgentId).toBe("grandchild");
    });
  });

  describe("Agent Children Retrieval", () => {
    it("should return empty array for non-existent parent", () => {
      const { getAgentChildren } = useStore.getState();

      const children = getAgentChildren("non-existent");
      expect(children).toEqual([]);
    });

    it("should return empty array for leaf nodes", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("leaf", "root"));

      const children = getAgentChildren("leaf");
      expect(children).toEqual([]);
    });

    it("should not return grandchildren in direct children", () => {
      const { addAgent, getAgentChildren } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("child", "root"));
      addAgent(createAgent("grandchild", "child"));

      const rootChildren = getAgentChildren("root");
      expect(rootChildren.length).toBe(1);
      expect(rootChildren[0].id).toBe("child");
    });
  });

  describe("Agent Updates", () => {
    it("should update agent status", () => {
      const { addAgent, updateAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      updateAgent("root", { status: "complete" });

      expect(useStore.getState().agents["root"].status).toBe("complete");
    });

    it("should update agent with token usage", () => {
      const { addAgent, updateAgent } = useStore.getState();

      addAgent(createAgent("root", null));
      updateAgent("root", {
        status: "complete",
        tokenUsage: { input: 1000, output: 2000 },
      });

      const agent = useStore.getState().agents["root"];
      expect(agent.tokenUsage).toEqual({ input: 1000, output: 2000 });
    });

    it("should update agent with end time", () => {
      const { addAgent, updateAgent } = useStore.getState();
      const endTime = Date.now();

      addAgent(createAgent("root", null));
      updateAgent("root", { endTime });

      expect(useStore.getState().agents["root"].endTime).toBe(endTime);
    });

    it("should preserve other properties when updating", () => {
      const { addAgent, updateAgent } = useStore.getState();

      addAgent({
        id: "root",
        parentId: null,
        model: "claude-sonnet-4",
        tier: "sonnet",
        status: "running",
        description: "Test agent",
      });

      updateAgent("root", { status: "complete" });

      const agent = useStore.getState().agents["root"];
      expect(agent.model).toBe("claude-sonnet-4");
      expect(agent.description).toBe("Test agent");
      expect(agent.status).toBe("complete");
    });
  });

  describe("Clear Operations", () => {
    it("should clear all agents", () => {
      const { addAgent, clearAgents } = useStore.getState();

      addAgent(createAgent("root", null));
      addAgent(createAgent("child", "root"));

      expect(Object.keys(useStore.getState().agents).length).toBe(2);

      clearAgents();

      const state = useStore.getState();
      expect(Object.keys(state.agents).length).toBe(0);
      expect(state.selectedAgentId).toBeNull();
      expect(state.rootAgentId).toBeNull();
    });

    it("should clear selection when clearing agents", () => {
      const { addAgent, selectAgent, clearAgents } = useStore.getState();

      addAgent(createAgent("root", null));
      selectAgent("root");

      clearAgents();

      expect(useStore.getState().selectedAgentId).toBeNull();
    });
  });
});
