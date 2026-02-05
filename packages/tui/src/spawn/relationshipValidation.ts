import { getAgentConfig } from "./agentConfig.js";
import { Mutex } from "async-mutex";
import type { Agent } from "../store/types.js";

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
 * Store interface for agent management
 */
export interface AgentsStore {
  get: (id: string) => Agent | undefined;
  addChild: (parentId: string, childId: string) => void;
  removeChild: (parentId: string, childId: string) => void;
}

// Per-parent mutex map to allow parallel spawns from DIFFERENT parents
const parentMutexes = new Map<string, Mutex>();

function getParentMutex(parentId: string): Mutex {
  if (!parentMutexes.has(parentId)) {
    parentMutexes.set(parentId, new Mutex());
  }
  return parentMutexes.get(parentId)!;
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
 * @param usingClaimedType - If true, parentType came from caller_type (not store).
 *                           Requires BIDIRECTIONAL validation: both spawned_by AND can_spawn must pass.
 */
export function validateSpawnRelationship(
  parentType: string | null | undefined,
  childType: string,
  currentChildCount: number = 0,
  usingClaimedType: boolean = false
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

  // If using claimed type, we REQUIRE bidirectional validation for security
  // Both spawned_by (child allows parent) AND can_spawn (parent allows child) must pass
  if (usingClaimedType && parentType) {
    const parentConfig = getAgentConfig(parentType);

    if (!parentConfig) {
      errors.push({
        code: "E_CLAIMED_TYPE_UNKNOWN",
        message:
          `Claimed caller_type '${parentType}' not found in agents-index.json. ` +
          `Cannot validate claimed identity.`,
        field: "caller_type",
      });
      return { valid: false, errors, warnings };
    }

    // Bidirectional check 1: Does child allow this parent? (spawned_by)
    if (childConfig.spawned_by && childConfig.spawned_by.length > 0) {
      if (!childConfig.spawned_by.includes("any") && !childConfig.spawned_by.includes(parentType)) {
        errors.push({
          code: "E_CLAIMED_TYPE_SPAWNED_BY_VIOLATION",
          message:
            `Claimed caller_type '${parentType}' is not in '${childType}'.spawned_by. ` +
            `Allowed: [${childConfig.spawned_by.join(", ")}]`,
          field: "caller_type",
        });
      }
    }

    // Bidirectional check 2: Does parent allow this child? (can_spawn)
    if (parentConfig.can_spawn && parentConfig.can_spawn.length > 0) {
      if (!parentConfig.can_spawn.includes(childType)) {
        errors.push({
          code: "E_CLAIMED_TYPE_CAN_SPAWN_VIOLATION",
          message:
            `Claimed caller_type '${parentType}' does not list '${childType}' in can_spawn. ` +
            `Allowed: [${parentConfig.can_spawn.join(", ")}]`,
          field: "caller_type",
        });
      }
    } else {
      // Parent has no can_spawn list but claiming to spawn - warn but allow
      warnings.push({
        code: "W_CLAIMED_TYPE_NO_CAN_SPAWN",
        message:
          `Claimed caller_type '${parentType}' has no can_spawn list defined. ` +
          `Allowing based on child's spawned_by only.`,
        field: "caller_type",
      });
    }

    // If bidirectional checks failed, return early
    if (errors.length > 0) {
      return { valid: false, errors, warnings };
    }

    // Bidirectional validation passed! Log for audit trail
    warnings.push({
      code: "I_CLAIMED_TYPE_ACCEPTED",
      message:
        `Using claimed caller_type '${parentType}' (Task-spawned agent). ` +
        `Bidirectional validation passed: can_spawn ✓, spawned_by ✓`,
      field: "caller_type",
    });

    // Continue to other checks but skip redundant spawned_by/can_spawn since we already did them
    return {
      valid: true,
      errors,
      warnings,
    };
  }

  // Standard validation path (parent from store or router)

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
 * Validates spawn AND registers child atomically.
 * Returns validation result; if valid, child is already registered.
 *
 * @param usingClaimedType - If true, parentType came from caller_type param (not store).
 *                           Requires bidirectional validation for security.
 */
export async function validateAndRegisterSpawn(
  parentId: string | null,
  parentType: string | null | undefined,
  childType: string,
  childId: string,
  store: AgentsStore,
  usingClaimedType: boolean = false
): Promise<SpawnValidationResult> {
  // No parent = router spawn, no locking needed
  if (!parentId) {
    return validateSpawnRelationship(parentType, childType, 0, usingClaimedType);
  }

  const mutex = getParentMutex(parentId);

  // Critical section: validate + register atomically
  return await mutex.runExclusive(async () => {
    const parent = store.get(parentId);
    const currentChildCount = parent?.childIds?.length || 0;

    const result = validateSpawnRelationship(
      parentType,
      childType,
      currentChildCount,
      usingClaimedType
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
