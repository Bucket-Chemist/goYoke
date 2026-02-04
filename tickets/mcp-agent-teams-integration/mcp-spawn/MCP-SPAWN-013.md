```yaml
---
id: MCP-SPAWN-013
title: Agent Relationship Validation Integration
description: Integrate spawn_agent with agents-index.json for relationship validation before spawning.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-008]
phase: 2
tags: [validation, relationships, agents-index, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-013: Agent Relationship Validation Integration

## Description

Integrate spawn_agent with the existing agents-index.json relationship fields to validate spawns before execution. This closes the gap between the agent-relationships-schema.json design and the actual implementation.

**Source**: agent-relationships-schema.json validation_rules, agent-relationships-examples.md §MCP spawn_agent Validation

## Why This Matters

agents-index.json already defines:
- `spawned_by`: Who can spawn this agent
- `can_spawn`: Who this agent can spawn
- `max_delegations`: Maximum children allowed

Without validation:
- Mozart could spawn agents not in its `can_spawn` list
- Agents could be spawned by unauthorized parents
- Resource limits (max_delegations) not enforced
- No warning when relationships are unexpected

## Task

1. Create agents-index.json loader with caching
2. Implement validateSpawnRelationship function
3. Integrate validation into spawn_agent before spawning
4. Handle errors (block) vs warnings (log but proceed)

## Files

- `packages/tui/src/spawn/agentConfig.ts` — Config loader
- `packages/tui/src/spawn/relationshipValidation.ts` — Validation logic
- `packages/tui/src/spawn/relationshipValidation.test.ts` — Tests
- `packages/tui/src/mcp/tools/spawnAgent.ts` — Integration

## Implementation

### Agent Config Loader (`packages/tui/src/spawn/agentConfig.ts`)

```typescript
import * as fs from "fs";
import * as path from "path";

/**
 * Relationship fields from agents-index.json
 */
export interface AgentRelationships {
  id: string;
  spawned_by?: string[];
  can_spawn?: string[];
  must_delegate?: boolean;
  min_delegations?: number;
  max_delegations?: number;
  inputs?: string[];
  outputs?: string[];
  outputs_to?: string[];
}

/**
 * Full agent config from agents-index.json
 */
export interface AgentConfig extends AgentRelationships {
  name: string;
  model: string;
  tier: number | string;
  triggers?: string[];
  tools?: string[];
  description?: string;
}

interface AgentsIndex {
  version: string;
  agents: AgentConfig[];
}

// Cache for agents-index.json
let cachedIndex: AgentsIndex | null = null;
let cacheTime: number = 0;
const CACHE_TTL_MS = 60000; // 1 minute

/**
 * Get the path to agents-index.json
 */
function getAgentsIndexPath(): string {
  // Check standard locations
  const locations = [
    path.join(process.cwd(), ".claude", "agents", "agents-index.json"),
    path.join(process.env.HOME || "", ".claude", "agents", "agents-index.json"),
  ];

  for (const loc of locations) {
    if (fs.existsSync(loc)) {
      return loc;
    }
  }

  throw new Error(
    "[agentConfig] agents-index.json not found. Checked: " + locations.join(", ")
  );
}

/**
 * Load agents-index.json with caching.
 */
export function loadAgentsIndex(): AgentsIndex {
  const now = Date.now();

  // Return cached if still valid
  if (cachedIndex && now - cacheTime < CACHE_TTL_MS) {
    return cachedIndex;
  }

  const indexPath = getAgentsIndexPath();
  const content = fs.readFileSync(indexPath, "utf-8");
  cachedIndex = JSON.parse(content) as AgentsIndex;
  cacheTime = now;

  return cachedIndex;
}

/**
 * Get config for a specific agent by ID.
 */
export function getAgentConfig(agentId: string): AgentConfig | null {
  const index = loadAgentsIndex();
  return index.agents.find((a) => a.id === agentId) || null;
}

/**
 * Clear the cache (for testing).
 */
export function clearAgentConfigCache(): void {
  cachedIndex = null;
  cacheTime = 0;
}
```

### Relationship Validation (`packages/tui/src/spawn/relationshipValidation.ts`)

```typescript
import { getAgentConfig, AgentConfig } from "./agentConfig";

export interface SpawnValidationResult {
  valid: boolean;
  errors: SpawnValidationError[];
  warnings: SpawnValidationWarning[];
}

export interface SpawnValidationError {
  code: string;
  message: string;
  field: string;
}

export interface SpawnValidationWarning {
  code: string;
  message: string;
  field: string;
}

/**
 * Validate spawn relationship between parent and child agent.
 *
 * Errors are blocking (spawn will fail).
 * Warnings are logged but spawn proceeds.
 *
 * @param parentType - Agent type of the parent (null if spawned by router)
 * @param childType - Agent type to spawn
 * @param currentChildCount - Number of children already spawned by parent
 */
export function validateSpawnRelationship(
  parentType: string | null | undefined,
  childType: string,
  currentChildCount: number = 0
): SpawnValidationResult {
  const errors: SpawnValidationError[] = [];
  const warnings: SpawnValidationWarning[] = [];

  const childConfig = getAgentConfig(childType);

  // Unknown child agent - allow with warning
  if (!childConfig) {
    warnings.push({
      code: "W_UNKNOWN_CHILD",
      message: `No config found for agent '${childType}' in agents-index.json`,
      field: "childType",
    });
    return { valid: true, errors, warnings };
  }

  // 1. Check spawned_by (who is allowed to spawn this child)
  if (childConfig.spawned_by && childConfig.spawned_by.length > 0) {
    const allowedParents = childConfig.spawned_by;

    // "any" means anyone can spawn
    if (!allowedParents.includes("any")) {
      // Router is represented as null parentType
      const parentIdentifier = parentType || "router";

      if (!allowedParents.includes(parentIdentifier)) {
        errors.push({
          code: "E_SPAWNED_BY_VIOLATION",
          message:
            `'${childType}' can only be spawned by [${allowedParents.join(", ")}], ` +
            `not '${parentIdentifier}'`,
          field: "spawned_by",
        });
      }
    }
  }

  // 2. Check can_spawn (is parent allowed to spawn this child)
  if (parentType) {
    const parentConfig = getAgentConfig(parentType);

    if (parentConfig) {
      // If parent has can_spawn defined, child must be in the list
      if (parentConfig.can_spawn && parentConfig.can_spawn.length > 0) {
        if (!parentConfig.can_spawn.includes(childType)) {
          errors.push({
            code: "E_CAN_SPAWN_VIOLATION",
            message:
              `'${parentType}' cannot spawn '${childType}'. ` +
              `Allowed: [${parentConfig.can_spawn.join(", ")}]`,
            field: "can_spawn",
          });
        }
      }

      // 3. Check max_delegations
      if (parentConfig.max_delegations !== undefined) {
        if (currentChildCount >= parentConfig.max_delegations) {
          errors.push({
            code: "E_MAX_DELEGATIONS_EXCEEDED",
            message:
              `'${parentType}' at max_delegations limit ` +
              `(${currentChildCount}/${parentConfig.max_delegations})`,
            field: "max_delegations",
          });
        }
      }
    } else {
      // Unknown parent - warn but allow
      warnings.push({
        code: "W_UNKNOWN_PARENT",
        message: `No config found for parent agent '${parentType}'`,
        field: "parentType",
      });
    }
  }

  // 4. Check invoked_by for additional context (warning only)
  if (childConfig.invoked_by) {
    const expectedInvoker = childConfig.invoked_by;

    // invoked_by can be: "router", "skill:<name>", "orchestrator:<id>", "any"
    if (expectedInvoker !== "any") {
      const actualInvoker = parentType ? `orchestrator:${parentType}` : "router";

      if (
        expectedInvoker !== actualInvoker &&
        expectedInvoker !== "router" &&
        !expectedInvoker.startsWith("skill:")
      ) {
        warnings.push({
          code: "W_INVOKED_BY_MISMATCH",
          message:
            `'${childType}' expects invoked_by='${expectedInvoker}', ` +
            `actual='${actualInvoker}'`,
          field: "invoked_by",
        });
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Format validation result for logging/display.
 */
export function formatValidationResult(result: SpawnValidationResult): string {
  const lines: string[] = [];

  if (result.valid) {
    lines.push("✅ Spawn validation passed");
  } else {
    lines.push("❌ Spawn validation failed");
  }

  if (result.errors.length > 0) {
    lines.push("\nErrors:");
    for (const err of result.errors) {
      lines.push(`  [${err.code}] ${err.message}`);
    }
  }

  if (result.warnings.length > 0) {
    lines.push("\nWarnings:");
    for (const warn of result.warnings) {
      lines.push(`  [${warn.code}] ${warn.message}`);
    }
  }

  return lines.join("\n");
}
```

### C1 Critical Fix: Atomic Child Count Management

**Problem:** Lines 247-254 have a TOCTOU (time-of-check-time-of-use) race condition. When multiple spawns occur concurrently, the check-then-spawn pattern allows exceeding limits.

**Solution:** Use a mutex to make validation + child registration atomic.

#### Updated `relationshipValidation.ts` with Atomic Operations

Add these imports and functions to the file:

```typescript
import { Mutex } from "async-mutex";

// Per-parent mutex map to allow parallel spawns from DIFFERENT parents
const parentMutexes = new Map<string, Mutex>();

function getParentMutex(parentId: string): Mutex {
  if (!parentMutexes.has(parentId)) {
    parentMutexes.set(parentId, new Mutex());
  }
  return parentMutexes.get(parentId)!;
}

/**
 * Validates spawn AND registers child atomically.
 * Returns validation result; if valid, child is already registered.
 */
export async function validateAndRegisterSpawn(
  parentId: string | null,
  parentType: string | null | undefined,
  childType: string,
  childId: string,
  store: AgentsStore
): Promise<SpawnValidationResult> {
  // No parent = router spawn, no locking needed
  if (!parentId) {
    return validateSpawnRelationship(parentType, childType, 0);
  }

  const mutex = getParentMutex(parentId);

  // Critical section: validate + register atomically
  return await mutex.runExclusive(async () => {
    const parent = store.get(parentId);
    const currentChildCount = parent?.childIds?.length || 0;

    const result = validateSpawnRelationship(
      parentType,
      childType,
      currentChildCount
    );

    if (result.valid) {
      // Register child INSIDE the lock
      store.addChild(parentId, childId);
    }

    return result;
  });
}

/**
 * Cleanup mutex when parent completes (prevent memory leak)
 */
export function cleanupParentMutex(parentId: string): void {
  parentMutexes.delete(parentId);
}
```

#### Required Dependency

Add to `packages/tui/package.json`:
```json
"dependencies": {
  "async-mutex": "^0.4.0"
}
```

#### Integration in `spawnAgent.ts`

Replace the validation block (around line 357-391 in current spec):

```typescript
// OLD (race-prone):
// const validation = validateSpawnRelationship(parentType, args.agent, currentChildCount);

// NEW (atomic):
import { validateAndRegisterSpawn, cleanupParentMutex } from "../../spawn/relationshipValidation";

const validation = await validateAndRegisterSpawn(
  parentId,
  parentType,
  args.agent,
  agentId,  // Pre-generated UUID
  getAgentsStore()
);

// If validation failed, child was NOT registered - safe to return error
if (!validation.valid) {
  return { content: [{ type: "text", text: JSON.stringify({ /* error */ }) }] };
}

// Child already registered - proceed with spawn
// On spawn failure, must REMOVE child from parent:
proc.on("error", () => {
  getAgentsStore().removeChild(parentId, agentId);
});
```

#### Concurrent Spawn Test Case

Add to `relationshipValidation.test.ts`:

```typescript
describe("concurrent spawn handling", () => {
  it("should not exceed max_delegations under concurrent spawns", async () => {
    // Parent with max_delegations: 2
    const parentId = "test-parent";
    const store = createMockStore({ maxDelegations: 2 });

    // Spawn 5 children concurrently
    const results = await Promise.all([
      validateAndRegisterSpawn(parentId, "orchestrator", "child", "c1", store),
      validateAndRegisterSpawn(parentId, "orchestrator", "child", "c2", store),
      validateAndRegisterSpawn(parentId, "orchestrator", "child", "c3", store),
      validateAndRegisterSpawn(parentId, "orchestrator", "child", "c4", store),
      validateAndRegisterSpawn(parentId, "orchestrator", "child", "c5", store),
    ]);

    const successes = results.filter(r => r.valid).length;
    const failures = results.filter(r => !r.valid).length;

    // Exactly 2 should succeed, 3 should fail
    expect(successes).toBe(2);
    expect(failures).toBe(3);
    expect(store.getChildCount(parentId)).toBe(2);
  });
});
```

### Integration into spawn_agent (`packages/tui/src/mcp/tools/spawnAgent.ts`)

```typescript
// Add imports at top
import {
  validateSpawnRelationship,
  formatValidationResult,
} from "../../spawn/relationshipValidation";

// Inside the spawn_agent handler, BEFORE spawning:

export const spawnAgent = tool(
  "spawn_agent",
  // ... description ...
  // ... schema ...
  async (args): Promise<{ content: Array<{ type: "text"; text: string }> }> => {
    const agentId = randomUUID();
    const registry = getProcessRegistry();
    const store = getAgentsStore();

    // Get parent info from store
    const parentId = args.parentId || process.env.GOGENT_PARENT_AGENT;
    const parentAgent = parentId ? store.get(parentId) : null;
    const parentType = parentAgent?.agentType;
    const currentChildCount = parentAgent?.childIds?.length || 0;

    // === RELATIONSHIP VALIDATION ===
    const validation = validateSpawnRelationship(
      parentType,
      args.agent,
      currentChildCount
    );

    // Log validation result
    if (!validation.valid || validation.warnings.length > 0) {
      console.log(formatValidationResult(validation));
    }

    // Block on validation errors
    if (!validation.valid) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                agentId: null,
                agent: args.agent,
                success: false,
                error: `Spawn validation failed: ${validation.errors
                  .map((e) => e.message)
                  .join("; ")}`,
                validationErrors: validation.errors,
                validationWarnings: validation.warnings,
              },
              null,
              2
            ),
          },
        ],
      };
    }

    // === END VALIDATION ===

    // Proceed with spawn...
    // (rest of existing implementation)
  }
);
```

### Tests (`packages/tui/src/spawn/relationshipValidation.test.ts`)

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  validateSpawnRelationship,
  formatValidationResult,
} from "./relationshipValidation";
import { clearAgentConfigCache } from "./agentConfig";

// Mock agents-index.json
vi.mock("fs", () => ({
  existsSync: vi.fn(() => true),
  readFileSync: vi.fn(() =>
    JSON.stringify({
      version: "test",
      agents: [
        {
          id: "mozart",
          name: "Mozart",
          model: "opus",
          tier: 3,
          can_spawn: ["einstein", "staff-architect-critical-review", "beethoven"],
          must_delegate: true,
          min_delegations: 3,
          max_delegations: 5,
        },
        {
          id: "einstein",
          name: "Einstein",
          model: "opus",
          tier: 3,
          spawned_by: ["mozart"],
          outputs_to: ["beethoven"],
        },
        {
          id: "beethoven",
          name: "Beethoven",
          model: "opus",
          tier: 3,
          spawned_by: ["mozart"],
          can_spawn: [],
        },
        {
          id: "review-orchestrator",
          name: "Review Orchestrator",
          model: "sonnet",
          tier: 2,
          can_spawn: ["backend-reviewer", "frontend-reviewer"],
          max_delegations: 4,
        },
        {
          id: "backend-reviewer",
          name: "Backend Reviewer",
          model: "haiku",
          tier: 1.5,
          spawned_by: ["review-orchestrator"],
        },
        {
          id: "codebase-search",
          name: "Codebase Search",
          model: "haiku",
          tier: 1,
          spawned_by: ["any"],
        },
      ],
    })
  ),
}));

describe("validateSpawnRelationship", () => {
  beforeEach(() => {
    clearAgentConfigCache();
  });

  describe("spawned_by validation", () => {
    it("should allow spawn when parent is in spawned_by list", () => {
      const result = validateSpawnRelationship("mozart", "einstein");

      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it("should block spawn when parent not in spawned_by list", () => {
      const result = validateSpawnRelationship("review-orchestrator", "einstein");

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_SPAWNED_BY_VIOLATION" })
      );
    });

    it("should allow spawn when spawned_by includes 'any'", () => {
      const result = validateSpawnRelationship("random-agent", "codebase-search");

      expect(result.valid).toBe(true);
    });

    it("should allow router to spawn when spawned_by includes 'router'", () => {
      // Add router to spawned_by for this test
      const result = validateSpawnRelationship(null, "codebase-search");

      expect(result.valid).toBe(true);
    });
  });

  describe("can_spawn validation", () => {
    it("should allow spawn when child is in parent can_spawn list", () => {
      const result = validateSpawnRelationship("mozart", "einstein");

      expect(result.valid).toBe(true);
    });

    it("should block spawn when child not in parent can_spawn list", () => {
      const result = validateSpawnRelationship("mozart", "backend-reviewer");

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_CAN_SPAWN_VIOLATION" })
      );
    });

    it("should allow spawn when parent has no can_spawn defined", () => {
      // Parent without can_spawn should allow anything
      const result = validateSpawnRelationship("backend-reviewer", "codebase-search");

      // backend-reviewer has no can_spawn, so no E_CAN_SPAWN error
      // but codebase-search has spawned_by: ["any"] so it's valid
      expect(result.errors.filter((e) => e.code === "E_CAN_SPAWN_VIOLATION")).toHaveLength(
        0
      );
    });
  });

  describe("max_delegations validation", () => {
    it("should allow spawn when under max_delegations", () => {
      const result = validateSpawnRelationship("mozart", "einstein", 2);

      expect(result.valid).toBe(true);
    });

    it("should block spawn when at max_delegations", () => {
      const result = validateSpawnRelationship("mozart", "beethoven", 5);

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_MAX_DELEGATIONS_EXCEEDED" })
      );
    });

    it("should block spawn when over max_delegations", () => {
      const result = validateSpawnRelationship("review-orchestrator", "backend-reviewer", 4);

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({
          code: "E_MAX_DELEGATIONS_EXCEEDED",
          message: expect.stringContaining("4/4"),
        })
      );
    });
  });

  describe("unknown agents", () => {
    it("should warn but allow unknown child agent", () => {
      const result = validateSpawnRelationship("mozart", "unknown-agent");

      // Should be valid (allow) but with warning
      expect(result.valid).toBe(true);
      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_CHILD" })
      );
    });

    it("should warn but allow unknown parent agent", () => {
      const result = validateSpawnRelationship("unknown-parent", "codebase-search");

      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_PARENT" })
      );
    });
  });
});

describe("formatValidationResult", () => {
  it("should format success result", () => {
    const result = { valid: true, errors: [], warnings: [] };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("✅ Spawn validation passed");
  });

  it("should format error result with details", () => {
    const result = {
      valid: false,
      errors: [
        { code: "E_TEST", message: "Test error", field: "test" },
      ],
      warnings: [],
    };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("❌ Spawn validation failed");
    expect(formatted).toContain("[E_TEST]");
    expect(formatted).toContain("Test error");
  });
});
```

## Acceptance Criteria

- [ ] agents-index.json loaded and cached (1 minute TTL)
- [ ] validateSpawnRelationship checks spawned_by, can_spawn, max_delegations
- [ ] Errors block spawn with clear message
- [ ] Warnings logged but spawn proceeds
- [ ] Integration with spawn_agent works correctly
- [ ] Race condition fix implemented with async-mutex
- [ ] Concurrent spawn test passes
- [ ] All tests pass: `npm test -- src/spawn/relationshipValidation.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file: `packages/tui/src/spawn/relationshipValidation.test.ts`
- [ ] Number of test functions: 13
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: Mozart spawns Einstein (allowed), Mozart spawns backend-reviewer (blocked)

## Schema Alignment

This ticket aligns mcp-spawning-v3 with agent-relationships-schema.json:

| Schema Field | Validated? | Behavior |
|--------------|------------|----------|
| `spawned_by` | ✅ | Block if parent not in list |
| `can_spawn` | ✅ | Block if child not in list |
| `max_delegations` | ✅ | Block if count exceeded |
| `invoked_by` | ✅ | Warning if mismatch |
| `inputs/outputs` | ❌ | Future: data flow validation |
| `outputs_to` | ❌ | Future: visualization |

