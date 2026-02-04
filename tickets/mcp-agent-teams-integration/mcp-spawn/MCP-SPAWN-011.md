```yaml
---
id: MCP-SPAWN-011
title: Review-Orchestrator Update
description: Update review-orchestrator to use spawn_agent for parallel reviewer spawning.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009, MCP-SPAWN-013]
phase: 2
tags: [orchestrator, review, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-011: Review-Orchestrator Update

## Description

Update the review-orchestrator to use MCP spawn_agent for spawning parallel reviewers (backend, frontend, standards, architect).

**Source**: Staff-Architect Analysis v2 §Part 6

## Task

1. Update review-orchestrator prompt to use spawn_agent
2. Spawn reviewers in parallel
3. Collect all results (handle partial failures)
4. Test full review workflow

## Acceptance Criteria

- [ ] review-orchestrator uses spawn_agent for reviewers
- [ ] Parallel spawning works (all reviewers start simultaneously)
- [ ] Partial failures handled (continue if 1 reviewer fails)
- [ ] All findings collected and synthesized
- [ ] Full /review workflow completes
- [ ] All tests pass: `npm test -- tests/e2e/review-orchestrator-spawn.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/e2e/review-orchestrator-spawn.test.ts`
- [ ] Number of test functions: 5
- [ ] All tests passing
- [ ] Coverage ≥80%

### Required Test Cases (`packages/tui/tests/e2e/review-orchestrator-spawn.test.ts`)

```typescript
import { describe, it, expect } from "vitest";

describe("Review-Orchestrator MCP Spawning", () => {
  describe("parallel reviewer spawning", () => {
    it("should spawn backend-reviewer via MCP spawn_agent", async () => {
      const result = await invokeReviewOrchestratorWithMockSpawn({
        childAgent: "backend-reviewer",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("backend-reviewer");
    });

    it("should spawn frontend-reviewer via MCP spawn_agent", async () => {
      const result = await invokeReviewOrchestratorWithMockSpawn({
        childAgent: "frontend-reviewer",
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("frontend-reviewer");
    });

    it("should spawn all reviewers in parallel", async () => {
      const timeline = await trackReviewerSpawnOrder();

      // All reviewers should start within 100ms of each other
      const startTimes = timeline.map(r => r.startTime);
      const maxDiff = Math.max(...startTimes) - Math.min(...startTimes);

      expect(maxDiff).toBeLessThan(100);
    });
  });

  describe("partial failure handling", () => {
    it("should continue if one reviewer fails", async () => {
      const result = await invokeReviewOrchestratorWithFailure({
        failingAgent: "frontend-reviewer",
        workingAgents: ["backend-reviewer", "standards-reviewer"],
      });

      // Should still collect results from working reviewers
      expect(result.collectedCount).toBe(2);
      expect(result.failedCount).toBe(1);
      expect(result.overallSuccess).toBe(true);
    });
  });

  describe("findings collection", () => {
    it("should collect and synthesize all reviewer findings", async () => {
      const result = await invokeFullReviewWorkflow();

      expect(result.findings).toBeDefined();
      expect(result.findings.backend).toBeDefined();
      expect(result.findings.frontend).toBeDefined();
      expect(result.synthesisComplete).toBe(true);
    });
  });
});

// Helper functions use mock CLI infrastructure from MCP-SPAWN-003
async function invokeReviewOrchestratorWithMockSpawn(opts: any): Promise<any> {
  throw new Error("Implement with MCP-SPAWN-003 infrastructure");
}
```

