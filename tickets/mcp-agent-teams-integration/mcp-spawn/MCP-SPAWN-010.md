```yaml
---
id: MCP-SPAWN-010
title: Mozart Orchestrator Update
description: Update Mozart to use spawn_agent for spawning Einstein and Staff-Architect.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009, MCP-SPAWN-013]
phase: 2
tags: [orchestrator, braintrust, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-010: Mozart Orchestrator Update

## Description

Update the Mozart orchestrator (Braintrust skill) to use MCP spawn_agent for spawning Einstein and Staff-Architect instead of attempting Task().

**Source**: Einstein Analysis §3.6.1

## Task

1. Update Mozart prompt to use spawn_agent
2. Add parallel spawning for Einstein + Staff-Architect
3. Update error handling for spawn failures
4. Test full Braintrust workflow

## Files

- `~/.claude/skills/braintrust/SKILL.md` — Update Mozart instructions

## Implementation

### Updated Mozart Instructions

Mozart should be instructed to use spawn_agent like this:

```
When spawning Einstein and Staff-Architect, use the MCP spawn_agent tool:

// Spawn Einstein
mcp__gofortress__spawn_agent({
  agent: "einstein",
  description: "Theoretical analysis for Braintrust",
  prompt: `AGENT: einstein

BRAINTRUST WORKFLOW - THEORETICAL ANALYSIS

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000  // 10 minutes for complex analysis
})

// Spawn Staff-Architect (can be parallel with Einstein)
mcp__gofortress__spawn_agent({
  agent: "staff-architect-critical-review",
  description: "Practical review for Braintrust",
  prompt: `AGENT: staff-architect-critical-review

BRAINTRUST WORKFLOW - PRACTICAL REVIEW

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000
})
```

## Acceptance Criteria

- [ ] Mozart uses spawn_agent instead of Task() for Level 2 spawning
- [ ] Einstein spawns successfully via MCP
- [ ] Staff-Architect spawns successfully via MCP
- [ ] Both outputs collected correctly
- [ ] Beethoven synthesis works with collected outputs
- [ ] Full Braintrust workflow completes end-to-end
- [ ] All tests pass: `npm test -- tests/e2e/mozart-spawn.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/e2e/mozart-spawn.test.ts`
- [ ] Number of test functions: 5
- [ ] All tests passing
- [ ] Coverage ≥80%

### Required Test Cases (`packages/tui/tests/e2e/mozart-spawn.test.ts`)

```typescript
import { describe, it, expect } from "vitest";

describe("Mozart Orchestrator MCP Spawning", () => {
  describe("spawn_agent usage", () => {
    it("should spawn Einstein via MCP spawn_agent", async () => {
      const result = await invokeMozartWithMockSpawn({
        childAgent: "einstein",
        expectedInvocation: "mcp__gofortress__spawn_agent"
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("einstein");
    });

    it("should spawn Staff-Architect via MCP spawn_agent", async () => {
      const result = await invokeMozartWithMockSpawn({
        childAgent: "staff-architect-critical-review",
        expectedInvocation: "mcp__gofortress__spawn_agent"
      });

      expect(result.spawnCalled).toBe(true);
      expect(result.agentType).toBe("staff-architect-critical-review");
    });

    it("should NOT use Task() for Einstein/Staff-Architect spawning", async () => {
      const result = await invokeMozartWithMockSpawn({
        verifyNoTaskCall: true
      });

      expect(result.taskCalled).toBe(false);
    });
  });

  describe("parallel spawning", () => {
    it("should spawn Einstein and Staff-Architect in parallel", async () => {
      const results = await trackMozartSpawnOrder();

      // Both should start before either completes
      const einsteinStart = results.find(r => r.agent === "einstein")?.startTime;
      const staffStart = results.find(r => r.agent === "staff-architect-critical-review")?.startTime;
      const einsteinEnd = results.find(r => r.agent === "einstein")?.endTime;

      // Staff-Architect should start before Einstein ends (parallel)
      expect(staffStart).toBeLessThan(einsteinEnd!);
    });
  });

  describe("Beethoven synthesis", () => {
    it("should invoke Beethoven after Einstein and Staff-Architect complete", async () => {
      const timeline = await trackFullBraintrustTimeline();

      const beethovenStart = timeline.find(e => e.agent === "beethoven")?.startTime;
      const einsteinEnd = timeline.find(e => e.agent === "einstein")?.endTime;
      const staffEnd = timeline.find(e => e.agent === "staff-architect-critical-review")?.endTime;

      // Beethoven should start AFTER both others complete
      expect(beethovenStart).toBeGreaterThan(einsteinEnd!);
      expect(beethovenStart).toBeGreaterThan(staffEnd!);
    });
  });
});

// Helper functions use mock CLI infrastructure from MCP-SPAWN-003
async function invokeMozartWithMockSpawn(opts: any): Promise<any> {
  throw new Error("Implement with MCP-SPAWN-003 infrastructure");
}
```

