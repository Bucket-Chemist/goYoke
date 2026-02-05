```yaml
---
id: MCP-SPAWN-017
title: Orchestrator Update for /plan Workflow
description: Update general orchestrator to use spawn_agent for spawning scouts, reviewers, and implementation agents.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009, MCP-SPAWN-013]
phase: 2
tags: [orchestrator, plan, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-017: Orchestrator Update for /plan Workflow

## Description

Update the general orchestrator agent to use MCP spawn_agent for spawning its children. The orchestrator is central to the `/plan` workflow and coordinates scouts, reviewers, and implementation agents.

**Source**: agents-index.json orchestrator.can_spawn, /plan skill workflow

## Why This Matters

The orchestrator handles:
- Ambiguous scope resolution
- Cross-module planning
- User interviews
- Design tradeoffs
- Debugging loops

It spawns multiple agent types and must use MCP spawn_agent at Level 1+ to maintain the agent hierarchy.

## Current can_spawn List

From agents-index.json:
```json
"can_spawn": [
  "codebase-search",
  "haiku-scout",
  "librarian",
  "code-reviewer",
  "architect",
  "go-pro",
  "python-pro",
  "typescript-pro",
  "react-pro"
]
```

## Task

1. Update orchestrator agent definition to use spawn_agent
2. Implement scout-first protocol via spawn_agent
3. Add spawn patterns for different workflow phases
4. Handle architect spawning for plan generation
5. Coordinate with /plan skill integration

## Files

- `~/.claude/agents/orchestrator/orchestrator.md` — Update agent instructions
- `~/.claude/skills/plan/SKILL.md` — Ensure compatibility
- `packages/tui/tests/e2e/orchestrator-spawn.test.ts` — E2E tests

## Implementation

### Updated Orchestrator Instructions

The orchestrator should use spawn_agent for all child spawning:

```
## Scout-First Protocol

When scope is unknown, spawn a scout first:

mcp__gofortress__spawn_agent({
  agent: "haiku-scout",
  description: "Assess scope before routing",
  prompt: `AGENT: haiku-scout

SCOUT REQUEST

Target: ${targetPath}
Question: ${userQuestion}

Gather:
- File count and types
- Total lines of code
- Language distribution
- Test coverage presence
- Complexity signals

Output to: .claude/tmp/scout_metrics.json`,
  model: "haiku",
  timeout: 30000  // 30 seconds max for scout
})

// Read scout results
const metrics = JSON.parse(fs.readFileSync(".claude/tmp/scout_metrics.json"));

// Route based on recommended_tier
if (metrics.recommended_tier === "opus") {
  // Generate GAP document for /braintrust
} else {
  // Continue with appropriate tier
}

## Research Phase

For codebase exploration:

mcp__gofortress__spawn_agent({
  agent: "codebase-search",
  description: "Find relevant code",
  prompt: `AGENT: codebase-search

Find all files related to: ${searchTopic}

Return:
- File paths
- Key functions/classes
- Entry points`,
  model: "haiku",
  timeout: 60000
})

For external documentation:

mcp__gofortress__spawn_agent({
  agent: "librarian",
  description: "Research library usage",
  prompt: `AGENT: librarian

Research: ${libraryName}

Find:
- Official documentation
- Best practices
- Common patterns
- Known issues`,
  model: "haiku",
  timeout: 120000
})

## Planning Phase

When ready to create implementation plan:

mcp__gofortress__spawn_agent({
  agent: "architect",
  description: "Create implementation plan",
  prompt: `AGENT: architect

CREATE IMPLEMENTATION PLAN

From scout report: .claude/tmp/scout_metrics.json
From strategy: .claude/tmp/strategy.md (if exists)

User goal: ${userGoal}

Produce:
1. specs.md at .claude/tmp/specs.md
2. TaskCreate calls for each implementation step

Follow phased approach with dependency mapping.`,
  model: "opus",
  timeout: 600000  // 10 minutes for complex planning
})

## Implementation Coordination

For implementation tasks, spawn language-specific agents:

// Go implementation
mcp__gofortress__spawn_agent({
  agent: "go-pro",
  description: "Implement Go code",
  prompt: `AGENT: go-pro

IMPLEMENT: ${taskDescription}

Files: ${fileList}
Conventions: go.md`,
  model: "sonnet",
  timeout: 300000
})

// Python implementation
mcp__gofortress__spawn_agent({
  agent: "python-pro",
  description: "Implement Python code",
  prompt: `AGENT: python-pro

IMPLEMENT: ${taskDescription}

Files: ${fileList}
Conventions: python.md`,
  model: "sonnet",
  timeout: 300000
})

// TypeScript/React implementation
mcp__gofortress__spawn_agent({
  agent: "typescript-pro",  // or react-pro for .tsx
  description: "Implement TypeScript code",
  prompt: `AGENT: typescript-pro

IMPLEMENT: ${taskDescription}

Files: ${fileList}
Conventions: typescript.md`,
  model: "sonnet",
  timeout: 300000
})

## Review Phase

After implementation:

mcp__gofortress__spawn_agent({
  agent: "code-reviewer",
  description: "Review implementation",
  prompt: `AGENT: code-reviewer

Review changes for: ${taskId}

Check:
- Convention compliance
- Obvious bugs
- Missing tests`,
  model: "haiku",
  timeout: 60000
})
```

### Workflow Integration with /plan Skill

The `/plan` skill invokes orchestrator at Level 1. The orchestrator then uses spawn_agent for all Level 2 spawns:

```
/plan workflow:
1. Router invokes orchestrator via Task() [Level 0 → Level 1]
2. Orchestrator spawns haiku-scout via spawn_agent [Level 1 → Level 2]
3. Orchestrator spawns architect via spawn_agent [Level 1 → Level 2]
4. Architect produces specs.md + TaskCreate calls
5. Orchestrator optionally spawns implementation agents [Level 1 → Level 2]
```

### State Management

Orchestrator reads and writes state files:

| File | Read By | Written By | Purpose |
|------|---------|------------|---------|
| `.claude/tmp/scout_metrics.json` | orchestrator, architect | haiku-scout | Scope assessment |
| `.claude/tmp/strategy.md` | architect | planner | High-level approach |
| `.claude/tmp/specs.md` | impl-manager, memory-archivist | architect | Implementation plan |

## Acceptance Criteria

- [ ] Orchestrator uses spawn_agent instead of Task() for Level 2 spawning
- [ ] Scout-first protocol works via spawn_agent
- [ ] All 9 agents in can_spawn list are properly invocable
- [ ] Architect spawning works for plan generation
- [ ] Implementation agent spawning works
- [ ] Code-reviewer spawning works for validation
- [ ] Full /plan workflow completes end-to-end
- [ ] All tests pass: `npm test -- tests/e2e/orchestrator-spawn.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/e2e/orchestrator-spawn.test.ts`
- [ ] Number of test functions: 7
- [ ] All tests passing
- [ ] Coverage ≥80%

### Required Test Cases

```typescript
import { describe, it, expect } from "vitest";

describe("Orchestrator MCP Spawning", () => {
  describe("scout-first protocol", () => {
    it("should spawn haiku-scout via spawn_agent", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        unknownScope: true,
      });

      expect(result.firstSpawn.agent).toBe("haiku-scout");
      expect(result.spawnMethod).toBe("mcp__gofortress__spawn_agent");
    });

    it("should read scout_metrics.json after scout completes", async () => {
      const result = await invokeOrchestratorWithScout();

      expect(result.scoutMetricsRead).toBe(true);
      expect(result.recommendedTier).toBeDefined();
    });
  });

  describe("research spawning", () => {
    it("should spawn codebase-search for code exploration", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "research",
        query: "find authentication handlers",
      });

      expect(result.spawnedAgent).toBe("codebase-search");
    });

    it("should spawn librarian for external docs", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "research",
        query: "how to use cobra library",
      });

      expect(result.spawnedAgent).toBe("librarian");
    });
  });

  describe("planning spawning", () => {
    it("should spawn architect for plan generation", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "planning",
      });

      expect(result.spawnedAgent).toBe("architect");
      expect(result.model).toBe("opus");
    });
  });

  describe("implementation spawning", () => {
    it("should spawn go-pro for Go implementation", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "implementation",
        file: "pkg/routing/validator.go",
      });

      expect(result.spawnedAgent).toBe("go-pro");
    });

    it("should spawn python-pro for Python implementation", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "implementation",
        file: "src/main.py",
      });

      expect(result.spawnedAgent).toBe("python-pro");
    });
  });

  describe("review spawning", () => {
    it("should spawn code-reviewer after implementation", async () => {
      const result = await invokeOrchestratorWithMockSpawn({
        phase: "review",
      });

      expect(result.spawnedAgent).toBe("code-reviewer");
      expect(result.model).toBe("haiku");
    });
  });
});

// Helper functions use mock CLI infrastructure from MCP-SPAWN-003
async function invokeOrchestratorWithMockSpawn(opts: any): Promise<any> {
  throw new Error("Implement with MCP-SPAWN-003 infrastructure");
}
```

## Relationship Validation

Per MCP-SPAWN-013, orchestrator's spawns will be validated against:

| Field | Value | Enforcement |
|-------|-------|-------------|
| `can_spawn` | 9 agents listed | Block if spawning unlisted agent |
| `scout_first` | true | Reminder injected if scout skipped |
| `must_delegate` | false | No minimum delegation requirement |

## Integration with Other Tickets

| Ticket | Relationship |
|--------|--------------|
| MCP-SPAWN-010 | Mozart uses similar patterns but for Braintrust |
| MCP-SPAWN-011 | Review-orchestrator uses similar patterns |
| MCP-SPAWN-016 | Impl-manager may be spawned BY orchestrator |
| MCP-SPAWN-013 | Validates all spawn relationships |

