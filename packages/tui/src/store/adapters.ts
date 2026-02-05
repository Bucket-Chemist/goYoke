import { Agent, AgentV1, CreateAgentInput } from "./types.js";
import { randomUUID } from "crypto";

/**
 * Ensures an agent has all V2 fields with sensible defaults.
 * Use when you need to work with extended fields on potentially V1 data.
 */
export function ensureAgentV2(agent: AgentV1 | Agent): Agent {
  // If already has V2 fields, return as-is
  if ("spawnMethod" in agent && agent.spawnMethod !== undefined) {
    return agent as Agent;
  }

  // Upgrade V1 to V2 with defaults
  return {
    ...agent,
    agentType: agent.description || "unknown",
    epicId: "legacy",
    depth: 1,
    childIds: [],
    spawnMethod: "task",
    spawnedBy: "router",
    queuedAt: agent.startTime,
  };
}

/**
 * Creates a new agent with all V2 fields populated.
 */
export function createAgent(input: CreateAgentInput): Agent {
  const now = Date.now();
  const id = randomUUID();

  return {
    // V1 fields
    id,
    parentId: input.parentId ?? null,
    model: input.model,
    tier: input.tier,
    status: "queued",
    description: input.description,
    startTime: now,

    // V2 fields
    agentType: input.agentType || input.description,
    epicId: input.epicId || "default",
    depth: input.parentId ? 2 : 1, // Will be calculated properly by caller
    childIds: [],
    spawnMethod: input.spawnMethod || "task",
    spawnedBy: input.parentId || "router",
    prompt: input.prompt,
    queuedAt: now,
  };
}

/**
 * Check if agent is V2 (has extended fields)
 */
export function isAgentV2(agent: AgentV1 | Agent): agent is Agent {
  return "spawnMethod" in agent && agent.spawnMethod !== undefined;
}

/**
 * Safely get depth, defaulting to 1 for V1 agents
 */
export function getAgentDepth(agent: AgentV1 | Agent): number {
  if ("depth" in agent && typeof agent.depth === "number") {
    return agent.depth;
  }
  return agent.parentId ? 2 : 1;
}

/**
 * Safely get childIds, defaulting to empty array for V1 agents
 */
export function getAgentChildIds(agent: AgentV1 | Agent): string[] {
  if ("childIds" in agent && Array.isArray(agent.childIds)) {
    return agent.childIds;
  }
  return [];
}
