# Task Tools Reference

**Version:** 1.0
**Created:** 2026-02-02
**Purpose:** Document native Claude Code task management tools and agent access patterns

---

## Overview

Claude Code provides native task management tools that replace the legacy `write_todos` and `TodoWrite` patterns. This document defines which agents have access to which tools and why.

## Native Task Tools

| Tool | Purpose | Parameters |
|------|---------|------------|
| **TaskCreate** | Create new task | `subject`, `description`, `activeForm` |
| **TaskUpdate** | Modify existing task | `taskId`, `status`, `addBlockedBy`, `addBlocks` |
| **TaskList** | View all tasks | (none) |
| **TaskGet** | Get task details | `taskId` |

### TaskCreate

Creates a new task in the session task list.

```javascript
TaskCreate({
  subject: "Phase 1: Initialize database schema",
  description: "Create migration files for user and session tables...",
  activeForm: "Initializing database schema..."
})
```

**Fields:**
- `subject` - Brief imperative title (shown in task list)
- `description` - Full details, acceptance criteria, context
- `activeForm` - Present continuous form shown while in_progress

### TaskUpdate

Updates task status or dependencies.

```javascript
// Mark task as started
TaskUpdate({ taskId: "1", status: "in_progress" })

// Mark task as complete
TaskUpdate({ taskId: "1", status: "completed" })

// Set dependency (task 2 blocked by task 1)
TaskUpdate({ taskId: "2", addBlockedBy: ["1"] })
```

**Status values:** `pending` → `in_progress` → `completed` (or `deleted`)

### TaskList

Returns summary of all tasks with status and blocking relationships.

### TaskGet

Returns full details of a specific task by ID.

---

## Agent Tool Access Matrix

| Agent | TaskCreate | TaskUpdate | TaskList | TaskGet | Rationale |
|-------|:----------:|:----------:|:--------:|:-------:|-----------|
| **architect** | ✅ | ✅ | ❌ | ❌ | Creates tasks from plans, sets phase dependencies |
| **planner** | ❌ | ❌ | ❌ | ❌ | Creates strategy, not actionable tasks |
| **orchestrator** | ❌ | ❌ | ✅ | ✅ | Verifies tasks exist, reads details for coordination |
| **impl-manager** | ❌ | ✅ | ✅ | ✅ | Manages execution, updates task status |
| **go-pro** | ❌ | ✅ | ❌ | ✅ | Updates assigned task when complete |
| **python-pro** | ❌ | ✅ | ❌ | ✅ | Updates assigned task when complete |
| **r-pro** | ❌ | ✅ | ❌ | ✅ | Updates assigned task when complete |
| **code-reviewer** | ❌ | ❌ | ❌ | ❌ | Review only, no task management |
| **ticket skill** | ✅ | ✅ | ✅ | ✅ | Full lifecycle from ticket criteria |

### Design Principles

**Separation of Concerns:**
- **Planners CREATE** - architect converts specs.md phases to tasks
- **Coordinators READ** - orchestrator, impl-manager check task state
- **Workers UPDATE** - implementation agents mark tasks in_progress/completed

**Why architect doesn't need TaskList/TaskGet:**
- Architect creates NEW tasks from scratch based on specs.md
- It doesn't need to read existing tasks (clean slate per planning session)

**Why impl-manager doesn't have TaskCreate:**
- impl-manager executes existing tasks, doesn't create new ones
- If new tasks are discovered during implementation, escalate to architect

**Why implementation agents have limited access:**
- They receive a single task assignment
- They mark it in_progress when starting, completed when done
- They don't need to see the full task list

---

## Migration from Legacy Patterns

### Before (Legacy)

```javascript
// Fictional tool that didn't exist
write_todos([
  { title: "Task 1", ... },
  { title: "Task 2", ... }
])

// Also fictional
TodoWrite with items:
- Item 1
- Item 2
```

### After (Native Tools)

```javascript
// Create each task
TaskCreate({
  subject: "Task 1",
  description: "Details...",
  activeForm: "Working on Task 1..."
})

TaskCreate({
  subject: "Task 2",
  description: "Details...",
  activeForm: "Working on Task 2..."
})

// Set dependencies
TaskUpdate({
  taskId: "2",
  addBlockedBy: ["1"]
})
```

---

## Workflow Patterns

### Pattern 1: Architect Planning Flow

```
specs.md created
    ↓
For each phase:
    TaskCreate(phase task)
    ↓
TaskUpdate to set blockedBy relationships
    ↓
Tasks ready for impl-manager
```

### Pattern 2: Implementation Flow

```
impl-manager calls TaskList
    ↓
Finds pending task with no blockers
    ↓
TaskUpdate(taskId, status: "in_progress")
    ↓
Spawns implementation agent (go-pro, etc.)
    ↓
Agent completes work
    ↓
TaskUpdate(taskId, status: "completed")
    ↓
Loop to next task
```

### Pattern 3: Ticket Skill Flow

```
Parse ticket acceptance criteria
    ↓
For each criterion:
    TaskCreate(criterion as task)
    ↓
Implementation happens
    ↓
TaskUpdate to mark criteria complete
    ↓
Verify all tasks completed
    ↓
Ticket complete
```

---

## Files Updated in Migration

| File | Changes Made |
|------|--------------|
| `.claude/agents/architect/agent.md` | Replaced `write_todos` with `TaskCreate`/`TaskUpdate` |
| `.claude/agents/orchestrator/agent.md` | Updated expected architect output |
| `.claude/agents/impl-manager/agent.md` | Removed `todos.json` fallback |
| `.claude/agents/impl-manager/agent.yaml` | Removed `TaskCreate` (not its role) |
| `.claude/agents/agents-index.json` | Updated architect tools and output_artifacts |
| `.claude/skills/ticket/SKILL.md` | Replaced `TodoWrite` with `TaskCreate` |
| `.claude/skills/ticket/README.md` | Updated tracking references |
| `.claude/skills/explore/SKILL.md` | Updated architect invocation |
| `.claude/skills/explore/CLAUDE.md` | Updated key outputs |
| `.claude/skills/plan/SKILL.md` | Updated architect instructions |

---

## Troubleshooting

**"Task not found"**
- Verify task was created in current session
- Tasks don't persist across sessions

**"Agent can't create tasks"**
- Check agent's tool access in this matrix
- Only architect and ticket skill should create tasks

**"Circular dependency detected"**
- Review blockedBy relationships
- Tasks can't block themselves or create cycles

---

**Maintained By:** GOgent-Fortress System
**Related Docs:**
- `docs/ARCHITECTURE.md` - Full system architecture
- `.claude/agents/agents-index.json` - Agent definitions
