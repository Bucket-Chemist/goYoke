import { describe, it, expect } from "vitest";
import {
  ensureAgentV2,
  createAgent,
  isAgentV2,
  getAgentDepth,
  getAgentChildIds,
} from "./adapters.js";
import { AgentV1, Agent } from "./types.js";

describe("ensureAgentV2", () => {
  it("should upgrade V1 agent with defaults", () => {
    const v1Agent: AgentV1 = {
      id: "test-1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Test agent",
      startTime: 1000,
    };

    const v2Agent = ensureAgentV2(v1Agent);

    expect(v2Agent.agentType).toBe("Test agent");
    expect(v2Agent.epicId).toBe("legacy");
    expect(v2Agent.depth).toBe(1);
    expect(v2Agent.childIds).toEqual([]);
    expect(v2Agent.spawnMethod).toBe("task");
    expect(v2Agent.spawnedBy).toBe("router");
  });

  it("should return V2 agent unchanged", () => {
    const v2Agent: Agent = {
      id: "test-1",
      parentId: "parent-1",
      model: "opus",
      tier: "opus",
      status: "complete",
      startTime: 1000,
      agentType: "einstein",
      epicId: "braintrust-123",
      depth: 2,
      childIds: [],
      spawnMethod: "mcp-cli",
      spawnedBy: "mozart",
    };

    const result = ensureAgentV2(v2Agent);

    expect(result).toEqual(v2Agent);
    expect(result.spawnMethod).toBe("mcp-cli");
  });
});

describe("createAgent", () => {
  it("should create agent with all V2 fields", () => {
    const agent = createAgent({
      model: "haiku",
      tier: "haiku",
      description: "Test scout",
      agentType: "codebase-search",
      epicId: "explore-123",
      spawnMethod: "task",
    });

    expect(agent.id).toBeDefined();
    expect(agent.model).toBe("haiku");
    expect(agent.agentType).toBe("codebase-search");
    expect(agent.status).toBe("queued");
    expect(agent.childIds).toEqual([]);
    expect(agent.queuedAt).toBeDefined();
  });

  it("should use description as agentType fallback", () => {
    const agent = createAgent({
      model: "sonnet",
      tier: "sonnet",
      description: "Code implementation",
    });

    expect(agent.agentType).toBe("Code implementation");
  });
});

describe("isAgentV2", () => {
  it("should return true for V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      spawnMethod: "task",
    };

    expect(isAgentV2(agent)).toBe(true);
  });

  it("should return false for V1 agent", () => {
    const agent: AgentV1 = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(isAgentV2(agent)).toBe(false);
  });
});

describe("getAgentDepth", () => {
  it("should return depth from V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: "parent",
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      depth: 3,
      spawnMethod: "mcp-cli",
    };

    expect(getAgentDepth(agent)).toBe(3);
  });

  it("should return default depth for V1 agent", () => {
    const v1WithParent: AgentV1 = {
      id: "1",
      parentId: "parent",
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    const v1WithoutParent: AgentV1 = {
      id: "2",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(getAgentDepth(v1WithParent)).toBe(2);
    expect(getAgentDepth(v1WithoutParent)).toBe(1);
  });
});

describe("getAgentChildIds", () => {
  it("should return childIds from V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      childIds: ["child-1", "child-2"],
      spawnMethod: "task",
    };

    expect(getAgentChildIds(agent)).toEqual(["child-1", "child-2"]);
  });

  it("should return empty array for V1 agent", () => {
    const agent: AgentV1 = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(getAgentChildIds(agent)).toEqual([]);
  });
});
