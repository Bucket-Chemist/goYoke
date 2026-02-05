```yaml
---
id: MCP-SPAWN-016
title: Impl-Manager Orchestrator Update
description: Update impl-manager to use spawn_agent for spawning implementation agents.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009, MCP-SPAWN-013]
phase: 2
tags: [orchestrator, implementation, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-016: Impl-Manager Orchestrator Update

## Description

Update the impl-manager orchestrator to use MCP spawn_agent for spawning implementation agents (go-pro, python-pro, typescript-pro, etc.) instead of attempting Task().

**Source**: agents-index.json impl-manager.can_spawn

## Why This Matters

impl-manager coordinates implementation tasks from specs.md, spawning language-specific agents to execute each task. At Level 1+, it cannot use Task() and must use MCP spawn_agent to spawn its children.

## Current can_spawn List

From agents-index.json:
```json
"can_spawn": [
  "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent",
  "python-pro", "python-ux",
  "r-pro", "r-shiny-pro",
  "typescript-pro", "react-pro",
  "codebase-search", "code-reviewer"
]
```

## Task

1. Update impl-manager agent definition to use spawn_agent
2. Implement intelligent agent selection based on file type
3. Add progress tracking for multi-agent implementations
4. Handle partial failures (continue if one agent fails)
5. Run mid-flight code-reviewer spawns for validation

## Files

- `~/.claude/agents/impl-manager/impl-manager.md` — Update agent instructions
- `packages/tui/tests/e2e/impl-manager-spawn.test.ts` — E2E tests

## Implementation

### Updated Impl-Manager Instructions

impl-manager should be instructed to use spawn_agent like this:

```
When implementing tasks from specs.md, use the MCP spawn_agent tool to delegate to language-specific agents:

// Determine agent based on file type
const agentMap = {
  ".go": "go-pro",
  ".py": "python-pro",
  ".ts": "typescript-pro",
  ".tsx": "react-pro",
  ".R": "r-pro"
};

// For CLI-specific Go files
if (filePath.includes("/cmd/") && ext === ".go") {
  agent = "go-cli";
}

// For TUI-specific Go files
if (filePath.includes("/tui/") && ext === ".go") {
  agent = "go-tui";
}

// Spawn the appropriate agent
mcp__gofortress__spawn_agent({
  agent: selectedAgent,
  description: `Implement task: ${taskSubject}`,
  prompt: `AGENT: ${selectedAgent}

IMPLEMENTATION TASK

Task ID: ${taskId}
Subject: ${taskSubject}

Files to modify:
${fileList}

Requirements:
${taskDescription}

Constraints:
- Follow conventions from ${conventionFile}
- Do not modify files outside the task scope
- Run tests after implementation`,
  model: "sonnet",
  timeout: 300000  // 5 minutes per task
})

// After implementation, spawn code-reviewer for validation
mcp__gofortress__spawn_agent({
  agent: "code-reviewer",
  description: "Validate implementation",
  prompt: `AGENT: code-reviewer

Review the changes made for task ${taskId}.
Check for:
- Convention compliance
- Obvious bugs
- Missing error handling`,
  model: "haiku",
  timeout: 60000
})
```

### Agent Selection Logic

```typescript
function selectImplementationAgent(filePath: string, taskContext: string): string {
  const ext = path.extname(filePath);

  // Go specializations
  if (ext === ".go") {
    if (filePath.includes("/cmd/") || taskContext.includes("cobra") || taskContext.includes("CLI")) {
      return "go-cli";
    }
    if (filePath.includes("/tui/") || taskContext.includes("bubbletea") || taskContext.includes("TUI")) {
      return "go-tui";
    }
    if (filePath.includes("/api/") || taskContext.includes("http client") || taskContext.includes("rate limit")) {
      return "go-api";
    }
    if (taskContext.includes("goroutine") || taskContext.includes("errgroup") || taskContext.includes("concurrent")) {
      return "go-concurrent";
    }
    return "go-pro";
  }

  // Python specializations
  if (ext === ".py") {
    if (taskContext.includes("PySide") || taskContext.includes("Qt") || taskContext.includes("GUI")) {
      return "python-ux";
    }
    return "python-pro";
  }

  // R specializations
  if (ext === ".R") {
    if (taskContext.includes("shiny") || taskContext.includes("reactive") || taskContext.includes("module")) {
      return "r-shiny-pro";
    }
    return "r-pro";
  }

  // TypeScript/React
  if (ext === ".tsx" || taskContext.includes("react") || taskContext.includes("component")) {
    return "react-pro";
  }
  if (ext === ".ts") {
    return "typescript-pro";
  }

  // Fallback to codebase-search for unknown
  return "codebase-search";
}
```

### Progress Tracking

```typescript
interface ImplProgress {
  totalTasks: number;
  completedTasks: number;
  failedTasks: number;
  currentTask: string | null;
  agentSpawns: Array<{
    taskId: string;
    agent: string;
    status: "pending" | "running" | "success" | "failed";
    duration?: number;
  }>;
}

// Write progress to .claude/tmp/impl-progress.json after each spawn
function updateProgress(progress: ImplProgress): void {
  fs.writeFileSync(
    ".claude/tmp/impl-progress.json",
    JSON.stringify(progress, null, 2)
  );
}
```

## Acceptance Criteria

- [ ] impl-manager uses spawn_agent instead of Task() for Level 2 spawning
- [ ] Correct agent selected based on file type and context
- [ ] All 13 agents in can_spawn list are properly invocable
- [ ] Progress tracking written to impl-progress.json
- [ ] Partial failures handled (continue with remaining tasks)
- [ ] Mid-flight code-reviewer validation working
- [ ] Full implementation workflow completes end-to-end
- [ ] All tests pass: `npm test -- tests/e2e/impl-manager-spawn.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/e2e/impl-manager-spawn.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing
- [ ] Coverage ≥80%

### Required Test Cases

```typescript
import { describe, it, expect } from "vitest";

describe("Impl-Manager MCP Spawning", () => {
  describe("agent selection", () => {
    it("should select go-pro for .go files", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        taskFile: "pkg/routing/validator.go",
      });

      expect(result.selectedAgent).toBe("go-pro");
    });

    it("should select go-cli for cmd/*.go files", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        taskFile: "cmd/gogent-validate/main.go",
      });

      expect(result.selectedAgent).toBe("go-cli");
    });

    it("should select go-tui for tui/*.go files", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        taskFile: "internal/tui/dashboard.go",
      });

      expect(result.selectedAgent).toBe("go-tui");
    });

    it("should select python-ux for PySide context", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        taskFile: "src/gui/main_window.py",
        taskContext: "Create QMainWindow with PySide6",
      });

      expect(result.selectedAgent).toBe("python-ux");
    });

    it("should select react-pro for .tsx files", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        taskFile: "src/components/Button.tsx",
      });

      expect(result.selectedAgent).toBe("react-pro");
    });
  });

  describe("spawn_agent usage", () => {
    it("should use MCP spawn_agent, not Task()", async () => {
      const result = await invokeImplManagerWithMockSpawn({
        verifyNoTaskCall: true,
      });

      expect(result.taskCalled).toBe(false);
      expect(result.spawnAgentCalled).toBe(true);
    });
  });

  describe("progress tracking", () => {
    it("should write progress to impl-progress.json", async () => {
      await invokeImplManagerWithMultipleTasks([
        { id: "1", file: "pkg/a.go" },
        { id: "2", file: "pkg/b.go" },
      ]);

      const progress = JSON.parse(
        fs.readFileSync(".claude/tmp/impl-progress.json", "utf-8")
      );

      expect(progress.totalTasks).toBe(2);
      expect(progress.agentSpawns).toHaveLength(2);
    });
  });

  describe("partial failure handling", () => {
    it("should continue if one task fails", async () => {
      const result = await invokeImplManagerWithFailure({
        failingTask: "task-2",
        workingTasks: ["task-1", "task-3"],
      });

      expect(result.completedTasks).toBe(2);
      expect(result.failedTasks).toBe(1);
      expect(result.overallSuccess).toBe(true);
    });
  });
});

// Helper functions use mock CLI infrastructure from MCP-SPAWN-003
async function invokeImplManagerWithMockSpawn(opts: any): Promise<any> {
  throw new Error("Implement with MCP-SPAWN-003 infrastructure");
}
```

## Relationship Validation

Per MCP-SPAWN-013, impl-manager's spawns will be validated against:

| Field | Value | Enforcement |
|-------|-------|-------------|
| `can_spawn` | 13 agents listed | Block if spawning unlisted agent |
| `must_delegate` | true | Block completion if no delegations |
| `min_delegations` | 1 | Block completion if < 1 spawn |

