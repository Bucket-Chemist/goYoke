```yaml
---
id: MCP-SPAWN-015
title: Validation Integration Tests
description: Integration tests for relationship validation and delegation enforcement in real workflows.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-012, MCP-SPAWN-014]
phase: 3
tags: [testing, validation, integration, phase-3]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-015: Validation Integration Tests

## Description

Integration tests that verify relationship validation (MCP-SPAWN-013) and delegation enforcement (MCP-SPAWN-014) work correctly in real orchestrator workflows. These tests ensure the guardrails actually prevent invalid spawns and incomplete orchestrations.

**Source**: Critical review of MCP-SPAWN ticket series

## Why This Matters

MCP-SPAWN-012 tests happy paths (successful Braintrust and Review workflows). This ticket tests the **failure paths** - ensuring validation actually blocks invalid operations:

Without these tests:
- Relationship validation could silently fail
- Delegation enforcement could have edge cases
- Schema mismatches could cause runtime errors
- Recovery from validation failures untested

## Task

1. Test spawn blocked by `spawned_by` violation
2. Test spawn blocked by `can_spawn` violation
3. Test spawn blocked by `max_delegations` exceeded
4. Test completion blocked by `must_delegate` / `min_delegations`
5. Test warning-only scenarios (unknown agents)
6. Test recovery/retry after validation failure

## Files

- `packages/tui/tests/e2e/validation-spawn.test.ts` — Spawn validation tests
- `packages/tui/tests/e2e/validation-delegation.test.ts` — Delegation enforcement tests
- `packages/tui/tests/e2e/validation-recovery.test.ts` — Recovery scenario tests

## Implementation

### Spawn Validation Tests (`packages/tui/tests/e2e/validation-spawn.test.ts`)

```typescript
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { spawnMockClaude } from "../mocks/spawnHelper";
import { clearAgentConfigCache } from "../../src/spawn/agentConfig";

describe("Spawn Validation Integration", () => {
  beforeAll(() => {
    // Ensure agents-index.json is loaded fresh
    clearAgentConfigCache();
  });

  describe("spawned_by enforcement", () => {
    it("should block Einstein spawn from unauthorized parent", async () => {
      // Einstein can only be spawned by Mozart
      // Attempting spawn from review-orchestrator should fail
      const result = await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "einstein",
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("E_SPAWNED_BY_VIOLATION");
      expect(result.error).toContain("can only be spawned by");
    });

    it("should allow Einstein spawn from Mozart", async () => {
      const result = await simulateSpawn({
        parentType: "mozart",
        childType: "einstein",
      });

      expect(result.success).toBe(true);
      expect(result.validationErrors).toHaveLength(0);
    });

    it("should allow any parent for agents with spawned_by: ['any']", async () => {
      // codebase-search has spawned_by: ["any"]
      const result = await simulateSpawn({
        parentType: "random-orchestrator",
        childType: "codebase-search",
      });

      expect(result.success).toBe(true);
    });
  });

  describe("can_spawn enforcement", () => {
    it("should block spawn when child not in parent can_spawn list", async () => {
      // Mozart's can_spawn: ["einstein", "staff-architect-critical-review", "beethoven"]
      // Attempting to spawn backend-reviewer should fail
      const result = await simulateSpawn({
        parentType: "mozart",
        childType: "backend-reviewer",
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("E_CAN_SPAWN_VIOLATION");
      expect(result.error).toContain("cannot spawn");
    });

    it("should allow spawn when child is in parent can_spawn list", async () => {
      const result = await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "backend-reviewer",
      });

      expect(result.success).toBe(true);
    });
  });

  describe("max_delegations enforcement", () => {
    it("should block spawn when parent at max_delegations", async () => {
      // review-orchestrator has max_delegations: 4
      const result = await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "backend-reviewer",
        currentChildCount: 4,
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("E_MAX_DELEGATIONS_EXCEEDED");
      expect(result.error).toContain("4/4");
    });

    it("should allow spawn when under max_delegations", async () => {
      const result = await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "backend-reviewer",
        currentChildCount: 2,
      });

      expect(result.success).toBe(true);
    });
  });

  describe("warning scenarios", () => {
    it("should warn but allow unknown child agent", async () => {
      const result = await simulateSpawn({
        parentType: "mozart",
        childType: "unknown-experimental-agent",
      });

      // Should succeed with warning
      expect(result.success).toBe(true);
      expect(result.validationWarnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_CHILD" })
      );
    });

    it("should warn but allow unknown parent agent", async () => {
      const result = await simulateSpawn({
        parentType: "unknown-orchestrator",
        childType: "codebase-search", // has spawned_by: ["any"]
      });

      expect(result.success).toBe(true);
      expect(result.validationWarnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_PARENT" })
      );
    });
  });
});

// Helper to simulate spawn through spawn_agent
async function simulateSpawn(opts: {
  parentType: string;
  childType: string;
  currentChildCount?: number;
}): Promise<{
  success: boolean;
  error?: string;
  validationErrors?: Array<{ code: string; message: string }>;
  validationWarnings?: Array<{ code: string; message: string }>;
}> {
  // Implementation uses mock CLI and spawn_agent tool
  // Details depend on test infrastructure from MCP-SPAWN-003
  throw new Error("Implement with mock CLI infrastructure");
}
```

### Delegation Enforcement Tests (`packages/tui/tests/e2e/validation-delegation.test.ts`)

```typescript
import { describe, it, expect } from "vitest";

describe("Delegation Enforcement Integration", () => {
  describe("must_delegate enforcement", () => {
    it("should block Mozart completion with insufficient children", async () => {
      // Mozart requires min_delegations: 3
      const result = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 2,
        status: "complete",
      });

      expect(result.allowed).toBe(false);
      expect(result.reason).toContain("requires at least 3 delegations");
      expect(result.suggestion).toContain("Spawn more agents");
    });

    it("should allow Mozart completion with 3+ children", async () => {
      const result = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 3,
        status: "complete",
      });

      expect(result.allowed).toBe(true);
    });

    it("should allow Mozart completion with more than minimum", async () => {
      const result = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 5,
        status: "complete",
      });

      expect(result.allowed).toBe(true);
    });
  });

  describe("review-orchestrator enforcement", () => {
    it("should block completion with only 1 reviewer", async () => {
      // review-orchestrator requires min_delegations: 2
      const result = await simulateOrchestratorCompletion({
        agentType: "review-orchestrator",
        childCount: 1,
        status: "complete",
      });

      expect(result.allowed).toBe(false);
      expect(result.reason).toContain("requires at least 2 delegations");
    });

    it("should allow completion with 2+ reviewers", async () => {
      const result = await simulateOrchestratorCompletion({
        agentType: "review-orchestrator",
        childCount: 2,
        status: "complete",
      });

      expect(result.allowed).toBe(true);
    });
  });

  describe("non-orchestrator agents", () => {
    it("should allow go-pro completion without any children", async () => {
      // go-pro has must_delegate: false (or undefined)
      const result = await simulateOrchestratorCompletion({
        agentType: "go-pro",
        childCount: 0,
        status: "complete",
      });

      expect(result.allowed).toBe(true);
    });
  });

  describe("error status bypass", () => {
    it("should skip delegation check on error status", async () => {
      // If orchestrator errored, don't block completion
      const result = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 0,
        status: "error",
      });

      expect(result.allowed).toBe(true);
      expect(result.message).toContain("did not complete successfully");
    });

    it("should skip delegation check on timeout status", async () => {
      const result = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 1,
        status: "timeout",
      });

      expect(result.allowed).toBe(true);
    });
  });
});

// Helper to simulate orchestrator completion through hook
async function simulateOrchestratorCompletion(opts: {
  agentType: string;
  childCount: number;
  status: "complete" | "error" | "timeout";
}): Promise<{
  allowed: boolean;
  reason?: string;
  message?: string;
  suggestion?: string;
}> {
  // Implementation invokes gogent-orchestrator-guard hook
  // with simulated SubagentStop event
  throw new Error("Implement with hook test infrastructure");
}
```

### Recovery Tests (`packages/tui/tests/e2e/validation-recovery.test.ts`)

```typescript
import { describe, it, expect } from "vitest";

describe("Validation Recovery Scenarios", () => {
  describe("spawn validation failure recovery", () => {
    it("should allow retry with different (valid) agent after rejection", async () => {
      // First attempt: Mozart tries to spawn backend-reviewer (invalid)
      const attempt1 = await simulateSpawn({
        parentType: "mozart",
        childType: "backend-reviewer",
      });
      expect(attempt1.success).toBe(false);

      // Second attempt: Mozart spawns einstein (valid)
      const attempt2 = await simulateSpawn({
        parentType: "mozart",
        childType: "einstein",
      });
      expect(attempt2.success).toBe(true);
    });

    it("should track child count correctly after failed spawn attempts", async () => {
      // Failed spawns should NOT increment child count
      await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "einstein", // Not in can_spawn
        currentChildCount: 3,
      });

      // Child count should still be 3, not 4
      const result = await simulateSpawn({
        parentType: "review-orchestrator",
        childType: "backend-reviewer",
        currentChildCount: 3,
      });

      expect(result.success).toBe(true);
      // If count was wrongly incremented, this would hit max_delegations
    });
  });

  describe("delegation failure recovery", () => {
    it("should allow orchestrator to spawn more children after delegation block", async () => {
      // Mozart attempts completion with 2 children - blocked
      const completion1 = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 2,
        status: "complete",
      });
      expect(completion1.allowed).toBe(false);

      // Mozart spawns another child
      const spawn = await simulateSpawn({
        parentType: "mozart",
        childType: "beethoven",
        currentChildCount: 2,
      });
      expect(spawn.success).toBe(true);

      // Mozart attempts completion with 3 children - allowed
      const completion2 = await simulateOrchestratorCompletion({
        agentType: "mozart",
        childCount: 3,
        status: "complete",
      });
      expect(completion2.allowed).toBe(true);
    });
  });

  describe("concurrent validation", () => {
    it("should handle parallel spawn attempts correctly", async () => {
      // Simulate Mozart spawning Einstein and Staff-Architect in parallel
      const results = await Promise.all([
        simulateSpawn({
          parentType: "mozart",
          childType: "einstein",
          currentChildCount: 0,
        }),
        simulateSpawn({
          parentType: "mozart",
          childType: "staff-architect-critical-review",
          currentChildCount: 0,
        }),
      ]);

      // Both should succeed (no race condition on child count)
      expect(results.every((r) => r.success)).toBe(true);
    });
  });
});
```

## Acceptance Criteria

- [ ] Spawn blocked by `spawned_by` violation - tested and passes
- [ ] Spawn blocked by `can_spawn` violation - tested and passes
- [ ] Spawn blocked by `max_delegations` exceeded - tested and passes
- [ ] Completion blocked by `must_delegate` / `min_delegations` - tested and passes
- [ ] Warning scenarios (unknown agents) - tested and passes
- [ ] Recovery after validation failure - tested and passes
- [ ] Concurrent spawn handling - tested and passes
- [ ] All tests pass: `npm test -- tests/e2e/validation-*.test.ts`
- [ ] Code coverage ≥80% on validation paths

## Test Deliverables

- [ ] Test file: `packages/tui/tests/e2e/validation-spawn.test.ts`
- [ ] Test file: `packages/tui/tests/e2e/validation-delegation.test.ts`
- [ ] Test file: `packages/tui/tests/e2e/validation-recovery.test.ts`
- [ ] Number of test functions: 18
- [ ] All tests passing
- [ ] Coverage ≥80%

## Manual Verification Checklist

After automated tests pass, manually verify:

- [ ] Mozart → Einstein spawn: **allowed**
- [ ] Mozart → backend-reviewer spawn: **blocked with clear error**
- [ ] review-orchestrator at 4 children → 5th spawn: **blocked**
- [ ] Mozart completion with 2 children: **blocked with guidance**
- [ ] Mozart completion with 3 children: **allowed**
- [ ] Unknown agent spawn: **warning logged, spawn proceeds**

## Telemetry Verification

Check that validation events are logged correctly:

```bash
# Check spawn validation logs
tail -20 ~/.local/share/gogent/spawn-validation.jsonl

# Check delegation enforcement logs
tail -20 ~/.local/share/gogent/delegation-violations.jsonl
tail -20 ~/.local/share/gogent/delegation-success.jsonl
```

