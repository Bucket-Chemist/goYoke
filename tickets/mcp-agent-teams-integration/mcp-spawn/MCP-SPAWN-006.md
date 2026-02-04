```yaml
---
id: MCP-SPAWN-006
title: Store Interface Extension (Backward Compatible)
description: Extend the Agent interface with hierarchy fields while maintaining backward compatibility with existing code.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-004]
phase: 1
tags: [store, types, phase-1]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-006: Store Interface Extension (Backward Compatible)

## Description

Extend the existing Agent interface with hierarchy and spawning fields while maintaining backward compatibility. All new fields are optional to avoid breaking existing code.

**Source**: Staff-Architect Analysis §4.1.4, §4.6.3

## Why This Matters

The current Agent interface has 10 fields. The proposed has 24. A breaking change would require updating all components simultaneously. The adapter pattern allows incremental migration.

## Task

1. Extend Agent interface with optional fields
2. Create AgentV1 type alias for legacy compatibility
3. Implement ensureAgentV2 adapter function
4. Update store slice to handle new fields

## Files

- `packages/tui/src/store/types.ts` — Type extensions
- `packages/tui/src/store/adapters.ts` — Adapter functions
- `packages/tui/src/store/adapters.test.ts` — Tests

## Implementation

### Type Extensions (`packages/tui/src/store/types.ts`)

```typescript
/**
 * Legacy Agent interface (V1) - DO NOT MODIFY existing fields.
 * Kept for backward compatibility reference.
 */
export interface AgentV1 {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: {
    input: number;
    output: number;
  };
}

/**
 * Extended Agent interface (V2) - All new fields are OPTIONAL.
 * This maintains backward compatibility with V1.
 */
export interface Agent extends AgentV1 {
  // Hierarchy (optional for V1 compatibility)
  agentType?: string;
  epicId?: string;
  depth?: number;
  childIds?: string[];

  // Spawning metadata (optional)
  spawnMethod?: "task" | "mcp-cli";
  spawnedBy?: string;
  prompt?: string;

  // Process info (for MCP-CLI spawns)
  pid?: number;
  queuedAt?: number;

  // Extended status (compatible with V1 status)
  // V1 status values still valid, these are additions
  // "queued" | "streaming" | "timeout" are new options

  // Output (optional)
  output?: string;
  streamBuffer?: string;
  error?: string;

  // Extended metrics (optional)
  cost?: number;
  turns?: number;
  toolCalls?: number;
}

/**
 * Status values - union of V1 and V2
 */
export type AgentStatus =
  | "queued"      // New: waiting to spawn
  | "spawning"   // V1: CLI starting
  | "running"    // V1: executing
  | "streaming"  // New: producing output
  | "complete"   // V1: finished successfully
  | "error"      // V1: failed
  | "timeout";   // New: exceeded time limit

/**
 * Spawn method discriminator
 */
export type SpawnMethod = "task" | "mcp-cli";

/**
 * Input for creating a new agent
 */
export interface CreateAgentInput {
  // Required
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  description: string;

  // Optional hierarchy
  parentId?: string | null;
  agentType?: string;
  epicId?: string;
  spawnMethod?: SpawnMethod;
  prompt?: string;
}
```

### Adapter Functions (`packages/tui/src/store/adapters.ts`)

```typescript
import { Agent, AgentV1, CreateAgentInput } from "./types";
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
```

### Tests (`packages/tui/src/store/adapters.test.ts`)

```typescript
import { describe, it, expect } from "vitest";
import {
  ensureAgentV2,
  createAgent,
  isAgentV2,
  getAgentDepth,
  getAgentChildIds,
} from "./adapters";
import { AgentV1, Agent } from "./types";

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
```

## Acceptance Criteria

- [ ] Agent interface extended with all V2 fields (all optional)
- [ ] AgentV1 type alias preserved for documentation
- [ ] ensureAgentV2 adapter correctly upgrades V1 agents
- [ ] createAgent produces valid V2 agents
- [ ] Helper functions (isAgentV2, getAgentDepth, getAgentChildIds) work correctly
- [ ] All tests pass: `npm test -- src/store/adapters.test.ts`
- [ ] Code coverage ≥80%
- [ ] Existing components still compile without changes

## Test Deliverables

- [ ] Test file created: `packages/tui/src/store/adapters.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing
- [ ] Coverage ≥80%

