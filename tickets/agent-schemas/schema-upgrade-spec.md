# Subagent Schema Upgrade Specification v1.0

**Purpose**: Fully specified guide for upgrading the GOgent-Fortress subagent schema system to address identified gaps, inconsistencies, and enhancement opportunities.

**Scope**: `routing-schema.json`, `agents-index.json`, `pkg/routing/schema.go`, individual agent YAML files.

**Target Version**: 2.5.0

---

## Table of Contents

1. [P0: Go Struct Synchronization](#p0-go-struct-synchronization)
2. [P1: Staff Architect Tier Correction](#p1-staff-architect-tier-correction)
3. [P1: Version Description Cleanup](#p1-version-description-cleanup)
4. [P2: Parallelization Template Resolution](#p2-parallelization-template-resolution)
5. [P3: Missing Sharp Edges](#p3-missing-sharp-edges)
6. [P3: New Agent - Implementation Manager](#p3-new-agent---implementation-manager)
7. [P4: Agent Dependency Graph Standardization](#p4-agent-dependency-graph-standardization)
8. [P4: Cost Budget Standardization](#p4-cost-budget-standardization)
9. [P4: Thinking Budget Standardization](#p4-thinking-budget-standardization)
10. [Validation Checklist](#validation-checklist)

---

## P0: Go Struct Synchronization

### Problem

The Go struct `AgentSubagentMapping` in `pkg/routing/schema.go` is missing fields for agents added in v2.4.0. This breaks hook validation for these agents.

### Missing Agents

| Agent ID | Expected subagent_type | Currently in struct |
|----------|------------------------|---------------------|
| `typescript-pro` | `general-purpose` | ❌ Missing |
| `react-pro` | `general-purpose` | ❌ Missing |
| `backend-reviewer` | `Explore` | ❌ Missing |
| `frontend-reviewer` | `Explore` | ❌ Missing |
| `standards-reviewer` | `Explore` | ❌ Missing |
| `review-orchestrator` | `Plan` | ❌ Missing |
| `impl-manager` | `Plan` | ❌ Missing (NEW AGENT) |

### Required Changes

#### File: `pkg/routing/schema.go`

##### 1. Update `AgentSubagentMapping` struct (around line 194)

Add these fields after existing fields:

```go
type AgentSubagentMapping struct {
    Description                  string `json:"description"`
    // ... existing fields ...
    GoConcurrent                 string `json:"go-concurrent"`
    // ADD THESE:
    TypescriptPro                string `json:"typescript-pro"`
    ReactPro                     string `json:"react-pro"`
    BackendReviewer              string `json:"backend-reviewer"`
    FrontendReviewer             string `json:"frontend-reviewer"`
    StandardsReviewer            string `json:"standards-reviewer"`
    ReviewOrchestrator           string `json:"review-orchestrator"`
    ImplManager                  string `json:"impl-manager"`
    // ... rest of existing fields ...
    Orchestrator                 string `json:"orchestrator"`
    // ...
}
```

##### 2. Update `GetSubagentTypeForAgent()` mapping (around line 386)

Add to the mapping inside the function:

```go
mapping := map[string]string{
    // ... existing mappings ...
    "go-concurrent":                   s.AgentSubagentMapping.GoConcurrent,
    // ADD THESE:
    "typescript-pro":                  s.AgentSubagentMapping.TypescriptPro,
    "react-pro":                       s.AgentSubagentMapping.ReactPro,
    "backend-reviewer":                s.AgentSubagentMapping.BackendReviewer,
    "frontend-reviewer":               s.AgentSubagentMapping.FrontendReviewer,
    "standards-reviewer":              s.AgentSubagentMapping.StandardsReviewer,
    "review-orchestrator":             s.AgentSubagentMapping.ReviewOrchestrator,
    "impl-manager":                    s.AgentSubagentMapping.ImplManager,
    // ... rest of existing mappings ...
}
```

##### 3. Update validation in `Validate()` (around line 336)

Add the new mappings to the validation slice:

```go
mappings := []string{
    // ... existing ...
    s.AgentSubagentMapping.GoConcurrent,
    // ADD THESE:
    s.AgentSubagentMapping.TypescriptPro,
    s.AgentSubagentMapping.ReactPro,
    s.AgentSubagentMapping.BackendReviewer,
    s.AgentSubagentMapping.FrontendReviewer,
    s.AgentSubagentMapping.StandardsReviewer,
    s.AgentSubagentMapping.ReviewOrchestrator,
    s.AgentSubagentMapping.ImplManager,
    // ... rest ...
}
```

### Verification

After changes, run:
```bash
go build ./...
go test ./pkg/routing/...
```

Expected: All tests pass, no compile errors.

---

## P1: Staff Architect Tier Correction

### Problem

`staff-architect-critical-review` is incorrectly configured as Sonnet tier but should be Opus tier. This agent performs critical architectural review and needs Opus-level reasoning.

### Current State (INCORRECT)

**File**: `.claude/agents/staff-architect-critical-review/agent.yaml`
```yaml
model: sonnet           # WRONG
tier: 2                 # WRONG (tier 2 = sonnet)
thinking_budget: 16000  # Too low for Opus work
```

**File**: `.claude/agents/agents-index.json`
```json
{
  "id": "staff-architect-critical-review",
  "model": "sonnet",           // WRONG
  "tier": 2,                   // WRONG
  "thinking_budget": 16000,    // Too low
}
```

**File**: `.claude/routing-schema.json`
```json
"sonnet": {
  "agents": [..., "staff-architect-critical-review"]  // WRONG LOCATION
}
```

### Required Changes

#### File: `.claude/agents/staff-architect-critical-review/agent.yaml`

Change:
```yaml
id: staff-architect-critical-review
name: Staff Architect Critical Review
model: opus                    # CHANGED from sonnet
thinking: true
thinking_budget: 32000         # CHANGED from 16000
tier: 3                        # CHANGED from 2
category: review
subagent_type: Plan

# ... rest unchanged ...
```

#### File: `.claude/agents/agents-index.json`

Find the `staff-architect-critical-review` entry and update:
```json
{
  "id": "staff-architect-critical-review",
  "parallelization_template": "C",
  "name": "Staff Architect Critical Review",
  "model": "opus",              // CHANGED from "sonnet"
  "thinking": true,
  "thinking_budget": 32000,     // CHANGED from 16000
  "tier": 3,                    // CHANGED from 2
  "category": "review",
  "path": "staff-architect-critical-review",
  // ... triggers, tools unchanged ...
}
```

#### File: `.claude/routing-schema.json`

##### 1. Move agent from sonnet tier to opus tier

In `tiers.sonnet.agents` array, REMOVE:
```json
"sonnet": {
  "agents": ["python-pro", "python-ux", ..., "staff-architect-critical-review"]
                                              // ^^^ REMOVE THIS
}
```

In `tiers.opus.agents` array, ADD:
```json
"opus": {
  "agents": ["einstein", "planner", "architect", "staff-architect-critical-review"]
                                                 // ^^^ ADD THIS
}
```

##### 2. Verify task_invocation_allowlist

The `opus.task_invocation_allowlist` already includes this agent (correct):
```json
"task_invocation_allowlist": ["planner", "architect", "staff-architect-critical-review"]
```

This is correct because this agent CAN be invoked via Task() despite the general Opus blocking rule. It's an allowlisted planning agent.

##### 3. Update model_tiers in routing_rules

```json
"model_tiers": {
  "haiku": [...],
  "haiku_thinking": [...],
  "sonnet": ["python-pro", "python-ux", "r-pro", "r-shiny-pro", "orchestrator",
             "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent",
             "typescript-pro", "react-pro", "review-orchestrator"],
             // ^^^ staff-architect-critical-review REMOVED
  "opus": ["einstein", "planner", "architect", "staff-architect-critical-review"],
           // ^^^ staff-architect-critical-review ADDED
  "external": ["gemini-slave"]
}
```

### Verification

1. Run schema validation:
   ```bash
   go test ./pkg/routing/... -run TestSchemaValidation
   ```

2. Test Task invocation works:
   ```bash
   # The hook should allow Task(staff-architect-critical-review)
   # because it's in the allowlist
   ```

---

## P1: Version Description Cleanup

### Problem

The `agents-index.json` description field contains stale version info that doesn't match the `version` field.

### Current State

```json
{
  "version": "2.4.0",
  "description": "Agent index for Intent Gate routing and auto-activation. v2.3: Added TypeScript/React agents and code review specialists.",
  // ^^^ Says v2.3 but version is 2.4.0
}
```

### Required Changes

#### File: `.claude/agents/agents-index.json`

Update to:
```json
{
  "version": "2.5.0",
  "generated_at": "2026-02-01T00:00:00Z",
  "description": "Agent index for Intent Gate routing and auto-activation.",
}
```

Remove version history from description. The `version` field is the single source of truth. Historical changes should be tracked in git commits or a CHANGELOG.

#### File: `.claude/routing-schema.json`

Update version:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "version": "2.5.0",
  "description": "Routing schema for Claude Code tiered agent architecture",
  "updated": "2026-02-01",
}
```

#### File: `pkg/routing/schema.go`

Update expected version constant:
```go
const EXPECTED_SCHEMA_VERSION = "2.5.0"
```

---

## P2: Parallelization Template Resolution

### Problem

Every agent has a `parallelization_template` field with values A-F, but:
- No documentation of what A-F means
- No code uses this field
- Purpose is unclear

### Decision Required

**Option A: Document and implement**

If this is intended for agent swarm coordination, define the templates:

```json
"parallelization_templates": {
  "A": {
    "description": "Pure parallel - no dependencies",
    "max_concurrent": 5,
    "can_run_with": ["A", "B"]
  },
  "B": {
    "description": "Read-only parallel",
    "max_concurrent": 3,
    "can_run_with": ["A", "B", "C"]
  },
  "C": {
    "description": "Limited parallel - documentation/review",
    "max_concurrent": 2,
    "can_run_with": ["A", "B", "C"]
  },
  "D": {
    "description": "Implementation - serial preferred",
    "max_concurrent": 1,
    "can_run_with": ["A"]
  },
  "E": {
    "description": "Orchestration - exclusive",
    "max_concurrent": 1,
    "can_run_with": []
  },
  "F": {
    "description": "Deep analysis - exclusive, high priority",
    "max_concurrent": 1,
    "can_run_with": []
  }
}
```

**Option B: Remove as dead code**

Remove `parallelization_template` from all agents and `agents-index.json`.

### Recommended Action

**Option B** - Remove. The field is unused and adds confusion. If parallel agent coordination is needed in future, design it properly at that time.

### Changes for Option B

#### File: `.claude/agents/agents-index.json`

For EACH agent entry, remove the `parallelization_template` field:
```json
{
  "id": "memory-archivist",
  // "parallelization_template": "E",  // REMOVE THIS LINE
  "name": "Memory Archivist",
  // ...
}
```

This affects all 26 agents in the index.

---

## P3: Missing Sharp Edges

### Problem

Several agents lack `sharp-edges.yaml` files documenting known pitfalls.

### Agents Requiring Sharp Edges

| Agent | Priority | Reason |
|-------|----------|--------|
| `orchestrator` | High | Complex coordination, failure modes |
| `planner` | High | Strategic decisions, scope creep risk |
| `review-orchestrator` | Medium | Parallel reviewer coordination |
| `memory-archivist` | Medium | Data integrity, archive failures |
| `haiku-scout` | Low | Simple agent, few edge cases |
| `gemini-slave` | Medium | External API, rate limits, failures |

### Required Files

#### File: `.claude/agents/orchestrator/sharp-edges.yaml`

```yaml
# Sharp Edges for Orchestrator Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: scout-skip
    severity: high
    category: routing
    description: "Skipping scout for unknown-scope tasks"
    symptom: "Expensive agent spawned for simple task, wasted budget"
    solution: |
      ALWAYS run scout first when scope is unknown:
      1. Check if task mentions "module", "system", "refactor"
      2. If yes, spawn haiku-scout or gemini-slave scout
      3. Read scout_metrics.json before routing
    auto_inject: true

  - id: circular-escalation
    severity: critical
    category: coordination
    description: "Orchestrator spawns orchestrator (infinite loop)"
    symptom: "Stack overflow, runaway costs"
    solution: |
      NEVER spawn orchestrator from orchestrator.
      If sub-task needs coordination, break into atomic tasks
      and spawn implementation agents directly.
    auto_inject: true

  - id: background-task-orphan
    severity: high
    category: coordination
    description: "Background tasks not collected before synthesis"
    symptom: "Incomplete analysis, missing findings"
    solution: |
      ALWAYS call TaskOutput for every background task before
      generating final output. Track task_ids explicitly.
    auto_inject: true

  - id: compound-trigger-miss
    severity: medium
    category: routing
    description: "Not detecting compound triggers"
    symptom: "Wrong agent selected for multi-domain task"
    solution: |
      When 2+ tier patterns fire, route to orchestrator for
      coordination rather than picking one arbitrarily.
    auto_inject: true

  - id: escalation-loop
    severity: high
    category: failure-handling
    description: "Retrying failed approach without modification"
    symptom: "3 identical failures, sharp edge captured"
    solution: |
      After failure, MUST modify approach:
      - Different tool selection
      - Smaller scope
      - More context
      - Different agent
      Never retry identically.
    auto_inject: true
```

#### File: `.claude/agents/planner/sharp-edges.yaml`

```yaml
# Sharp Edges for Planner Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: scope-creep
    severity: high
    category: planning
    description: "Plan scope expands beyond original request"
    symptom: "Plan includes unrequested features, gold-plating"
    solution: |
      Anchor plan to original user request.
      If scope expansion seems necessary, ASK first.
      Use max_clarifying_questions: 2 limit.
    auto_inject: true

  - id: missing-constraints
    severity: high
    category: planning
    description: "Plan ignores stated constraints"
    symptom: "Implementation blocked by unconsidered constraint"
    solution: |
      Extract ALL constraints from request:
      - Time/budget constraints
      - Technology constraints
      - Compatibility requirements
      - Performance requirements
      List explicitly in strategy.md
    auto_inject: true

  - id: dependency-blindness
    severity: medium
    category: planning
    description: "Plan assumes non-existent dependencies available"
    symptom: "Implementation fails due to missing package/API"
    solution: |
      Verify dependencies exist before planning:
      - Check go.mod / package.json / requirements.txt
      - Verify API endpoints exist
      - Check for version compatibility
    auto_inject: true

  - id: strategy-without-risks
    severity: medium
    category: planning
    description: "Strategy omits risk analysis"
    symptom: "Implementation hits foreseeable blocker"
    solution: |
      Every strategy.md MUST include:
      - Known risks
      - Mitigation strategies
      - Fallback approaches
    auto_inject: true
```

#### File: `.claude/agents/review-orchestrator/sharp-edges.yaml`

```yaml
# Sharp Edges for Review Orchestrator Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: reviewer-timeout
    severity: medium
    category: coordination
    description: "Specialist reviewer times out or fails"
    symptom: "Incomplete review, missing domain coverage"
    solution: |
      If any reviewer fails:
      1. Note failure in summary
      2. Proceed with available results
      3. Add caveat about incomplete review
      4. Consider WARNING status due to uncertainty
    auto_inject: true

  - id: finding-duplication
    severity: low
    category: synthesis
    description: "Same issue reported by multiple reviewers"
    symptom: "Inflated issue count, redundant findings"
    solution: |
      Deduplicate findings before output:
      - Same file + same line + similar message = duplicate
      - Keep most specific finding
      - Note "flagged by multiple reviewers"
    auto_inject: true

  - id: severity-inflation
    severity: medium
    category: assessment
    description: "Marking issues as critical when they're warnings"
    symptom: "False BLOCK recommendations"
    solution: |
      Critical is ONLY for:
      - Security vulnerabilities
      - Data corruption risks
      - Memory leaks
      - Accessibility blockers

      Style issues are NEVER critical.
    auto_inject: true

  - id: missing-telemetry
    severity: high
    category: output
    description: "Review output missing telemetry fields"
    symptom: "Findings not captured in ML telemetry"
    solution: |
      ALWAYS include in output:
      - session_id (required)
      - status: BLOCKED/WARNING/APPROVE
      - summary: {critical, warnings, info}
      - findings array with required fields
    auto_inject: true
```

#### File: `.claude/agents/gemini-slave/sharp-edges.yaml`

```yaml
# Sharp Edges for Gemini Slave Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: rate-limit-hit
    severity: high
    category: external-api
    description: "Gemini API rate limit exceeded"
    symptom: "429 errors, requests rejected"
    solution: |
      Implement exponential backoff:
      - First retry: 1s
      - Second retry: 2s
      - Third retry: 4s
      - Then fail and fall back to haiku-scout
    auto_inject: true

  - id: context-overflow
    severity: medium
    category: input
    description: "Input exceeds even Gemini's context window"
    symptom: "API error, truncated analysis"
    solution: |
      Pre-filter before piping to gemini-slave:
      - Remove test files if not relevant
      - Remove generated code
      - Summarize large files
      Target: 500K tokens max
    auto_inject: true

  - id: json-parse-failure
    severity: high
    category: output
    description: "Gemini output not valid JSON when expected"
    symptom: "Parse error in downstream processing"
    solution: |
      For protocols expecting JSON (scout, mapper):
      - Validate JSON before writing to state file
      - If invalid, retry with explicit JSON instruction
      - If still invalid, extract manually or fail gracefully
    auto_inject: true

  - id: protocol-mismatch
    severity: medium
    category: invocation
    description: "Wrong protocol for task type"
    symptom: "Poor quality output, wrong format"
    solution: |
      Protocol selection:
      - scout: Pre-routing reconnaissance, JSON output
      - mapper: Dependency extraction, JSON output
      - debugger: Root cause analysis, markdown output
      - architect: Pattern review, markdown output
    auto_inject: true
```

---

## P3: New Agent - Implementation Manager

### Problem

There is a gap in the agent hierarchy between planning (architect) and implementation (go-pro, python-pro, etc.):

```
Planner → Architect → [GAP] → Implementation Agents → Review
           ↓
      specs.md + todos
           ↓
      ??? WHO ENFORCES ???
           ↓
      go-pro, python-pro, etc. (just implement)
```

No agent currently:
1. **Enforces** that implementations match specs.md
2. **Validates** architectural conventions DURING implementation (not after)
3. **Coordinates** multi-file implementations as a coherent unit
4. **Tracks** task completion against defined acceptance criteria
5. **Catches** convention drift before it becomes tech debt

### Solution: impl-manager Agent

**Archetype**: Senior Engineering Manager / Tech Lead

**Role**: Wrangles implementation task lists, enforces conventions during implementation, validates specs adherence, coordinates multi-agent implementations.

### Agent Definition

#### File: `.claude/agents/impl-manager/agent.yaml`

```yaml
id: impl-manager
name: Implementation Manager
model: sonnet
thinking:
  enabled: true
  budget: 14000
  budget_complex: 18000
tier: 2
category: coordination
subagent_type: Plan

triggers:
  - "implement from specs"
  - "execute todos"
  - "implement plan"
  - "run implementation"
  - "execute tasks"
  - "implement with validation"
  - "managed implementation"

tools:
  - Read
  - Glob
  - Grep
  - Task
  - Write
  - TaskCreate
  - TaskUpdate
  - TaskList
  - TaskGet

inputs:
  - .claude/tmp/specs.md
  - .claude/tmp/todos.json (or TaskList API)
  - conventions/*.md (auto-loaded based on language)

outputs:
  - .claude/tmp/impl-progress.json
  - .claude/tmp/impl-violations.jsonl

delegation:
  # Blocklist - can spawn implementation agents freely
  cannot_spawn:
    - impl-manager      # Prevents circular
    - orchestrator      # Different domain
    - planner           # Upstream
    - architect         # Upstream
    - einstein          # Must use escalation protocol

  # Resource management
  max_parallel: 3       # Controlled parallel implementation

  # Cost constraint - spawns multiple agents
  cost_ceiling: 1.50    # USD

focus_areas:
  - Specs-to-implementation traceability
  - Convention enforcement during implementation
  - Test coverage requirements
  - Task completion verification
  - Multi-file coherence

failure_tracking:
  max_attempts: 2
  on_max_reached: "escalate_to_orchestrator"

description: >
  Implementation coordinator. Enforces specs.md adherence, injects conventions
  into implementation agents, validates task completion against acceptance
  criteria, runs mid-flight reviews. Fills gap between architect and
  implementation agents.
```

#### File: `.claude/agents/impl-manager/CLAUDE.md`

```markdown
# Implementation Manager Agent Context

## Identity

You are the **Implementation Manager Agent** - the enforcer of implementation quality and specs adherence.

## Core Workflow

### 1. LOAD CONTEXT
- Read specs.md
- Read/create task list (TaskList API or .claude/tmp/todos.json)
- Load language conventions (conventions/*.md)
- Identify implementation agents needed

### 2. PRE-FLIGHT VALIDATION (per task)
Before spawning an implementation agent, verify:
- Is task well-defined? (acceptance criteria exist)
- Are dependencies met? (blocked_by resolved)
- Which conventions apply?
- What tests are required?

### 3. DISPATCH IMPLEMENTATION
Spawn implementation agent with:
- Task definition
- Relevant conventions (INJECTED into prompt)
- Test requirements
- Acceptance criteria

Track: task_id, agent, files_touched

### 4. MID-FLIGHT CHECKPOINT
After every 3 files modified or for complex changes:
- Spawn code-reviewer (quick check)
- Check for convention violations
- Verify changes align with task scope
- If violations: STOP, fix, continue

### 5. POST-TASK VALIDATION
Before marking task complete:
- Run tests (if applicable)
- Verify acceptance criteria met
- Check for scope creep
- Mark task complete OR report blockers

### 6. COHERENCE CHECK
After all tasks complete:
- Do implementations integrate correctly?
- Are there cross-cutting concerns missed?
- Final convention sweep
- Generate implementation report

## Spawning Pattern

ALWAYS inject conventions when spawning implementation agents:

```javascript
Task({
  description: "Implement user authentication handler",
  subagent_type: "general-purpose",
  model: "sonnet",
  prompt: `AGENT: go-pro

TASK: Implement user authentication handler per specs.md section 3.2

SPECS EXTRACT:
[Include relevant section from specs.md]

CONVENTIONS (MANDATORY):
[Include relevant sections from go.md]

ACCEPTANCE CRITERIA:
- [ ] Handler accepts POST /auth/login
- [ ] Returns JWT on success
- [ ] Returns 401 on failure with error message
- [ ] Logs authentication attempts

TEST REQUIREMENTS:
- Table-driven tests for success/failure cases
- Test invalid input handling

FILES TO CREATE/MODIFY:
- internal/handlers/auth.go
- internal/handlers/auth_test.go

CONSTRAINTS:
- Do NOT modify other handlers
- Follow existing error handling patterns`
})
```

## Convention Injection

For EVERY implementation spawn, include:

```
CONVENTIONS (MANDATORY):
You MUST follow these patterns:

[Paste relevant convention sections]

Violations will be caught by mid-flight review and require fixes.
```

## Progress Tracking

Write to `.claude/tmp/impl-progress.json`:

```json
{
  "session_id": "...",
  "specs_file": ".claude/tmp/specs.md",
  "total_tasks": 5,
  "completed_tasks": 2,
  "in_progress_tasks": 1,
  "blocked_tasks": 0,
  "tasks": [
    {
      "task_id": "1",
      "subject": "Implement auth handler",
      "status": "completed",
      "agent": "go-pro",
      "files_touched": ["internal/handlers/auth.go"],
      "tests_written": true,
      "conventions_verified": true
    }
  ],
  "violations": []
}
```

## Violation Logging

Append to `.claude/tmp/impl-violations.jsonl`:

```json
{"task_id": "2", "file": "internal/api/user.go", "line": 45, "violation": "Missing error wrapping", "convention": "go.md#error-handling", "severity": "warning"}
```

## Escalation Triggers

Escalate to orchestrator when:
- 2+ consecutive task failures
- Specs.md is incomplete/ambiguous
- Cross-module conflict detected
- Test failures indicate design issue

## Anti-Patterns

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| Spawning without conventions | ALWAYS inject conventions |
| Skipping mid-flight reviews | Review every 3 files |
| Marking task complete without tests | Verify test existence |
| Ignoring specs.md | Every task traces to specs section |
| Scope creep acceptance | Stop, ask user to update specs |
```

#### File: `.claude/agents/impl-manager/sharp-edges.yaml`

```yaml
# Sharp Edges for Implementation Manager Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: specs-drift
    severity: critical
    category: enforcement
    description: "Implementation deviates from specs.md"
    symptom: "Code doesn't match architectural decisions"
    solution: |
      Before spawning implementation agent:
      1. Extract relevant section from specs.md
      2. Include in agent prompt as CONSTRAINTS
      3. After implementation, verify against specs
      If deviation: STOP, ask user to update specs OR revert
    auto_inject: true

  - id: convention-bypass
    severity: high
    category: enforcement
    description: "Implementation agent ignores injected conventions"
    symptom: "Code doesn't follow project patterns"
    solution: |
      Inject conventions into EVERY implementation spawn:
      - Read conventions/{language}.md
      - Include as MANDATORY PATTERNS section
      - Post-implementation: spawn code-reviewer to verify
    auto_inject: true

  - id: test-gap
    severity: high
    category: quality
    description: "Implementation without corresponding tests"
    symptom: "New code lacks test coverage"
    solution: |
      For each implementation task, track:
      - Source files created/modified
      - Test files required (same count)
      If tests missing at task completion: BLOCK task
    auto_inject: true

  - id: scope-creep
    severity: medium
    category: enforcement
    description: "Task implementation exceeds defined scope"
    symptom: "Files modified outside task definition"
    solution: |
      Before task: record expected files
      After task: compare actual vs expected
      If unexpected files touched: review with user
    auto_inject: true

  - id: orphan-task
    severity: medium
    category: tracking
    description: "Task started but never completed"
    symptom: "Task stuck in 'in_progress' state"
    solution: |
      Track task start time
      If task in_progress > 10 minutes without update:
      - Check for errors
      - Escalate or mark blocked
    auto_inject: true

  - id: missing-acceptance-criteria
    severity: high
    category: validation
    description: "Task has no acceptance criteria to validate against"
    symptom: "Cannot determine if task is truly complete"
    solution: |
      Before starting task:
      1. Check for acceptance criteria in specs.md
      2. If missing, extract from task description
      3. If still unclear, ASK user before proceeding
      Never mark complete without criteria verification
    auto_inject: true

  - id: parallel-conflict
    severity: high
    category: coordination
    description: "Parallel implementations modify same files"
    symptom: "Merge conflicts, overwritten changes"
    solution: |
      Before spawning parallel agents:
      - Identify file ownership per task
      - Ensure no overlap
      - If overlap necessary, serialize those tasks
    auto_inject: true
```

### Schema Integration

#### File: `.claude/routing-schema.json`

Add to `tiers.sonnet.agents`:
```json
"sonnet": {
  "agents": [..., "impl-manager"]
}
```

Add to `agent_subagent_mapping`:
```json
"impl-manager": "Plan"
```

Add to `subagent_types.Plan.use_for`:
```json
"Plan": {
  "use_for": [..., "impl-manager"]
}
```

Add to `routing_rules.model_tiers.sonnet`:
```json
"sonnet": [..., "impl-manager"]
```

#### File: `.claude/agents/agents-index.json`

Add new entry:
```json
{
  "id": "impl-manager",
  "name": "Implementation Manager",
  "model": "sonnet",
  "thinking": true,
  "thinking_budget": 14000,
  "tier": 2,
  "category": "coordination",
  "path": "impl-manager",
  "triggers": [
    "implement from specs",
    "execute todos",
    "implement plan",
    "run implementation",
    "execute tasks",
    "implement with validation",
    "managed implementation"
  ],
  "tools": ["Read", "Glob", "Grep", "Task", "Write", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet"],
  "auto_activate": null,
  "inputs": [".claude/tmp/specs.md", "conventions/*.md"],
  "outputs": [".claude/tmp/impl-progress.json", ".claude/tmp/impl-violations.jsonl"],
  "sharp_edges_count": 7,
  "cost_per_invocation": "0.50-1.50",
  "description": "Implementation coordinator. Enforces specs.md adherence, injects conventions, validates task completion, runs mid-flight reviews."
}
```

### Hook Integration

#### File: `cmd/gogent-orchestrator-guard/main.go`

Update orchestrator detection to include impl-manager:

```go
// isOrchestratingAgent checks if agent spawns other agents
func isOrchestratingAgent(agentID string) bool {
    orchestrators := map[string]bool{
        "orchestrator":        true,
        "review-orchestrator": true,
        "impl-manager":        true,  // ADD THIS
    }
    return orchestrators[agentID]
}
```

### Workflow Integration

```
User Request
    ↓
orchestrator (if ambiguous)
    ↓
planner → strategy.md
    ↓
architect → specs.md + todos
    ↓
┌─────────────────────────────────┐
│ impl-manager (NEW)              │
│   ├── validates specs           │
│   ├── loads conventions         │
│   ├── spawns implementation     │
│   ├── injects conventions       │
│   ├── mid-flight reviews        │
│   ├── tracks progress           │
│   └── validates completion      │
└─────────────────────────────────┘
    ↓
review-orchestrator (final review)
    ↓
Done
```

### Telemetry Compatibility

| Telemetry System | Status | Notes |
|------------------|--------|-------|
| RoutingDecision | ✅ Compatible | Task() calls logged by gogent-validate |
| AgentCollaboration | ✅ Compatible | Parent→child relationships captured |
| AgentLifecycle | ✅ Compatible | spawn/complete events captured |
| ReviewFinding | ✅ Compatible | Mid-flight reviews captured |
| SharpEdgeHit | ✅ Compatible | Sharp edge violations captured |
| MLToolEvent | ✅ Compatible | Tool usage tracked |

**No new telemetry code required** - existing hooks handle all tracking.

### Test Coverage

Add to `pkg/routing/schema_test.go`:

```go
func TestGetSubagentTypeForAgent_ImplManager(t *testing.T) {
    schema, err := LoadSchema()
    require.NoError(t, err)

    subagentType, err := schema.GetSubagentTypeForAgent("impl-manager")
    require.NoError(t, err)
    assert.Equal(t, "Plan", subagentType)
}

func TestValidateAgentSubagentPair_ImplManager(t *testing.T) {
    schema, err := LoadSchema()
    require.NoError(t, err)

    // Valid pair
    assert.NoError(t, schema.ValidateAgentSubagentPair("impl-manager", "Plan"))

    // Invalid pair
    assert.Error(t, schema.ValidateAgentSubagentPair("impl-manager", "Explore"))
    assert.Error(t, schema.ValidateAgentSubagentPair("impl-manager", "general-purpose"))
}
```

---

## P4: Agent Dependency Graph Standardization

### Problem

Only `staff-architect-critical-review` has explicit delegation constraints. Other orchestrating agents lack this.

### Design Philosophy

**Use blocklists, not allowlists.**

| Approach | Problem |
|----------|---------|
| `can_spawn` allowlist | Inflexible, needs updates when adding agents, limits legitimate work |
| `max_spawns` hard limit | Arbitrary, prevents thorough analysis of large tasks |
| `cannot_spawn` blocklist | Future-proof, clear intent, only blocks what's forbidden |
| `cost_ceiling` | The REAL constraint - spending, not count |

**Orchestrating agents should spawn as many agents as needed to complete the task.** The constraints are:
1. **Circular prevention** - Cannot spawn self
2. **Tier violations** - Cannot spawn opus-tier via Task() (except allowlisted)
3. **Domain boundaries** - Review orchestrator shouldn't spawn implementation agents
4. **Cost ceiling** - The actual budget limit

### Template

```yaml
delegation:
  # Blocklist only - can spawn anything NOT listed
  cannot_spawn:
    - self              # Prevents circular (use actual agent name)
    - einstein          # Never via Task()

  # Resource management
  max_parallel: 4       # Concurrent agents (system resources, not total)

  # Cost is the real constraint
  cost_ceiling: 1.50    # USD - higher for orchestrators
```

**Note**: No `can_spawn` allowlist. No `max_spawns` limit. Orchestrators are trusted to spawn what they need within cost ceiling.

### Required Changes

#### File: `.claude/agents/orchestrator/agent.yaml`

Add:
```yaml
delegation:
  # Blocklist - can spawn anything else
  cannot_spawn:
    - orchestrator      # Prevents circular dependency
    - einstein          # Must use escalation protocol, not Task()
    - planner           # Planner is upstream (feeds INTO orchestrator via /plan)
    - architect         # Architect is downstream (orchestrator produces specs FOR architect)

  # Resource management
  max_parallel: 6       # Can run 6 agents concurrently

  # Cost constraint - orchestrator gets generous budget for complex coordination
  cost_ceiling: 2.00    # USD

  # Notes:
  # - CAN spawn any implementation agent (go-pro, python-pro, etc.)
  # - CAN spawn any scout (haiku-scout, codebase-search)
  # - CAN spawn reviewers for validation
  # - CAN spawn same agent multiple times for parallel work
  # - No artificial spawn count limit
```

#### File: `.claude/agents/review-orchestrator/agent.yaml`

Add:
```yaml
delegation:
  # Blocklist - can spawn anything else
  cannot_spawn:
    - review-orchestrator  # Prevents circular dependency
    - orchestrator         # Different responsibility domain
    - einstein             # Must use escalation protocol
    - architect            # Review doesn't produce architecture
    - planner              # Review doesn't produce strategy
    # Implementation agents NOT blocked - reviewer may need context

  # Resource management
  max_parallel: 4          # Run up to 4 reviewers concurrently

  # Cost constraint - reviews should be thorough but bounded
  cost_ceiling: 0.75       # USD

  # Notes:
  # - CAN spawn backend-reviewer multiple times (different files)
  # - CAN spawn frontend-reviewer multiple times (different components)
  # - CAN spawn standards-reviewer
  # - CAN spawn scouts for context gathering
  # - No artificial spawn count limit
```

#### File: `.claude/agents/impl-manager/agent.yaml`

Add (already included in agent definition above, but for completeness):
```yaml
delegation:
  # Blocklist - can spawn implementation agents freely
  cannot_spawn:
    - impl-manager      # Prevents circular dependency
    - orchestrator      # Different domain (coordination vs implementation)
    - planner           # Upstream (planner → architect → impl-manager)
    - architect         # Upstream (architect → impl-manager)
    - einstein          # Must use escalation protocol

  # Resource management
  max_parallel: 3       # Controlled parallel implementation

  # Cost constraint - spawns multiple agents
  cost_ceiling: 1.50    # USD

  # Notes:
  # - CAN spawn any implementation agent (go-pro, python-pro, etc.)
  # - CAN spawn code-reviewer for mid-flight checks
  # - CAN spawn scouts for context gathering
  # - Spawns with convention injection
  # - No artificial spawn count limit
```

#### File: `.claude/agents/architect/agent.yaml`

Add:
```yaml
delegation:
  # Blocklist - architect is more restricted (terminal node)
  cannot_spawn:
    - architect         # Prevents circular
    - planner           # Planner feeds architect, not reverse
    - orchestrator      # Orchestrator feeds architect, not reverse
    - einstein          # Must use escalation protocol
    # Implementation agents blocked - architect plans, doesn't implement
    - python-pro
    - python-ux
    - r-pro
    - r-shiny-pro
    - go-pro
    - go-cli
    - go-tui
    - go-api
    - go-concurrent
    - typescript-pro
    - react-pro

  # Resource management - architect does focused work
  max_parallel: 2

  # Cost constraint
  cost_ceiling: 0.50       # USD - architect is opus tier, expensive per call

  # Notes:
  # - CAN spawn scouts for reconnaissance
  # - CAN spawn codebase-search for context
  # - CAN spawn librarian for external docs
  # - Architect produces specs.md, doesn't execute implementation
```

#### File: `.claude/agents/planner/agent.yaml`

Add:
```yaml
delegation:
  # Blocklist - planner is upstream, very restricted
  cannot_spawn:
    - planner           # Prevents circular
    - architect         # Planner outputs TO architect, doesn't spawn
    - orchestrator      # Different workflow
    - einstein          # Must use escalation protocol
    # Implementation agents blocked - planner strategizes, doesn't implement
    - python-pro
    - python-ux
    - r-pro
    - r-shiny-pro
    - go-pro
    - go-cli
    - go-tui
    - go-api
    - go-concurrent
    - typescript-pro
    - react-pro

  # Resource management - planner does focused strategic work
  max_parallel: 2

  # Cost constraint
  cost_ceiling: 0.50       # USD - planner is opus tier

  # Notes:
  # - CAN spawn scouts for scope assessment
  # - CAN spawn codebase-search for context
  # - CAN spawn librarian for best practices research
  # - Planner produces strategy.md, delegates implementation planning to architect
```

#### File: `.claude/agents/staff-architect-critical-review/agent.yaml`

Update existing delegation block:
```yaml
delegation:
  # Blocklist
  cannot_spawn:
    - staff-architect-critical-review  # Prevents circular
    - architect         # Prevents review → planning loop
    - planner           # Review doesn't produce strategy
    - einstein          # Must use escalation protocol
    - orchestrator      # Different domain
    # No sonnet implementation agents - critical review is analysis only
    - python-pro
    - python-ux
    - r-pro
    - r-shiny-pro
    - go-pro
    - go-cli
    - go-tui
    - go-api
    - go-concurrent
    - typescript-pro
    - react-pro

  # Resource management - critical review is focused
  max_parallel: 2

  # Cost constraint (already had cost_ceiling: 0.20, update to reflect opus tier)
  cost_ceiling: 0.60       # USD - opus tier for deep analysis

  # Notes:
  # - CAN spawn haiku-scout for verification
  # - CAN spawn codebase-search for context
  # - Review is read-only analysis, doesn't spawn implementation
```

### Summary of Delegation Philosophy

| Agent | Role | Spawn Freedom | Cost Ceiling |
|-------|------|---------------|--------------|
| `orchestrator` | Coordination hub | High - any agent except circular/opus | $2.00 |
| `impl-manager` | Implementation coordination | High - implementation agents + reviewers | $1.50 |
| `review-orchestrator` | Review coordination | High - reviewers + scouts | $0.75 |
| `architect` | Implementation planning | Low - scouts only | $0.50 |
| `planner` | Strategic planning | Low - scouts only | $0.50 |
| `staff-architect-critical-review` | Plan review | Low - scouts only | $0.60 |

**Key insight**: The MORE coordinating an agent is, the MORE spawn freedom it needs. Terminal nodes (architect, planner) are restricted because they produce artifacts, not coordination.

---

## P4: Cost Budget Standardization

### Problem

Only `staff-architect-critical-review` has `cost_ceiling`. Other agents lack budget constraints.

### Enhancement

Add `cost_ceiling` to all agents based on tier AND role.

### Recommended Ceilings

#### By Tier (Base Values)

| Tier | Base Ceiling | Notes |
|------|--------------|-------|
| Haiku (tier 1) | $0.02 | Fast, cheap operations |
| Haiku+Thinking (tier 1.5) | $0.05 | Structured reasoning |
| Sonnet (tier 2) | $0.25 | Implementation work |
| Opus (tier 3) | $0.50 | Deep analysis |
| External | $0.01 | Gemini is cheap |

#### By Role (Overrides)

Orchestrating agents get higher budgets because they spawn other agents:

| Agent | Role | Cost Ceiling | Rationale |
|-------|------|--------------|-----------|
| `orchestrator` | Coordination hub | $2.00 | Spawns many agents for complex tasks |
| `impl-manager` | Implementation coordination | $1.50 | Spawns implementation agents + reviewers |
| `review-orchestrator` | Review coordination | $0.75 | Spawns multiple reviewers |
| `architect` | Implementation planning | $0.50 | Opus tier, focused work |
| `planner` | Strategic planning | $0.50 | Opus tier, focused work |
| `staff-architect-critical-review` | Plan review | $0.60 | Opus tier, may spawn scouts |
| `einstein` | Deep analysis | $1.00 | Opus tier, extended thinking |

#### Implementation Agents (Sonnet Tier)

All Sonnet implementation agents get $0.25 ceiling:
- `python-pro`, `python-ux`
- `r-pro`, `r-shiny-pro`
- `go-pro`, `go-cli`, `go-tui`, `go-api`, `go-concurrent`
- `typescript-pro`, `react-pro`

#### Haiku Agents

| Agent | Cost Ceiling |
|-------|--------------|
| `codebase-search` | $0.02 |
| `haiku-scout` | $0.02 |
| `scaffolder` | $0.05 |
| `tech-docs-writer` | $0.05 |
| `code-reviewer` | $0.05 |
| `librarian` | $0.05 |
| `memory-archivist` | $0.05 |
| `backend-reviewer` | $0.05 |
| `frontend-reviewer` | $0.05 |
| `standards-reviewer` | $0.05 |

#### External

| Agent | Cost Ceiling |
|-------|--------------|
| `gemini-slave` | $0.01 |

### Required Changes

Add to each `agent.yaml`:

```yaml
cost_ceiling: X.XX  # USD - see table above for value
```

Apply to all 26 agents with role-appropriate values from tables above.

---

## P4: Thinking Budget Standardization

### Problem

`go-pro` has task-specific thinking budgets but other implementation agents don't.

### Current (go-pro only)

```yaml
thinking:
  enabled: true
  budget: 10000
  budget_refactor: 14000
  budget_debug: 18000
```

### Enhancement

Apply to all Sonnet-tier implementation agents:

```yaml
thinking:
  enabled: true
  budget: 10000           # Default
  budget_refactor: 14000  # Refactoring tasks
  budget_debug: 18000     # Debugging tasks
  budget_security: 16000  # Security-related (optional)
```

### Agents to Update

- `python-pro`
- `python-ux`
- `r-pro`
- `r-shiny-pro`
- `go-cli`
- `go-tui`
- `go-api`
- `go-concurrent`
- `typescript-pro`
- `react-pro`

---

## Validation Checklist

### After All Changes

- [ ] Version bumped to 2.5.0 in all files
- [ ] `go build ./...` succeeds
- [ ] `go test ./pkg/routing/...` passes
- [ ] Schema loads without error: `go run ./cmd/gogent-validate/...`
- [ ] All new agents have valid subagent_type mappings
- [ ] `staff-architect-critical-review` is in opus tier
- [ ] No `parallelization_template` fields remain (if Option B chosen)
- [ ] All sharp-edges.yaml files created
- [ ] All delegation blocks added to orchestrating agents
- [ ] `impl-manager` agent created with agent.yaml, CLAUDE.md, sharp-edges.yaml
- [ ] `impl-manager` added to agents-index.json
- [ ] `impl-manager` added to routing-schema.json (sonnet tier, Plan subagent_type)
- [ ] `gogent-orchestrator-guard` updated to recognize impl-manager

### Files Modified

1. `pkg/routing/schema.go`
2. `.claude/routing-schema.json`
3. `.claude/agents/agents-index.json`
4. `.claude/agents/staff-architect-critical-review/agent.yaml`
5. `.claude/agents/orchestrator/agent.yaml`
6. `.claude/agents/orchestrator/sharp-edges.yaml` (NEW)
7. `.claude/agents/planner/agent.yaml`
8. `.claude/agents/planner/sharp-edges.yaml` (NEW)
9. `.claude/agents/architect/agent.yaml`
10. `.claude/agents/review-orchestrator/agent.yaml`
11. `.claude/agents/review-orchestrator/sharp-edges.yaml` (NEW)
12. `.claude/agents/gemini-slave/sharp-edges.yaml` (NEW)
13. All implementation agent YAML files (thinking budget updates)

---

## Implementation Order

1. **Phase 1 (P0)**: Go struct synchronization - unblocks validation
2. **Phase 2 (P1)**: Staff architect tier fix + version cleanup
3. **Phase 3 (P2)**: Parallelization template removal
4. **Phase 4 (P3)**: Sharp edges creation
5. **Phase 5 (P4)**: Delegation graphs + cost budgets + thinking budgets

Each phase should be a separate commit for clean rollback if needed.

---

---

## P5: Test File Updates

### Problem

Adding new agents to `pkg/routing/schema.go` requires corresponding test coverage.

### Required Changes

#### File: `pkg/routing/schema_test.go`

Add test cases for new agent mappings:

```go
func TestGetSubagentTypeForAgent_NewAgents(t *testing.T) {
    schema, err := LoadSchema()
    require.NoError(t, err)

    tests := []struct {
        agent          string
        expectedType   string
    }{
        {"typescript-pro", "general-purpose"},
        {"react-pro", "general-purpose"},
        {"backend-reviewer", "Explore"},
        {"frontend-reviewer", "Explore"},
        {"standards-reviewer", "Explore"},
        {"review-orchestrator", "Plan"},
        {"impl-manager", "Plan"},
    }

    for _, tc := range tests {
        t.Run(tc.agent, func(t *testing.T) {
            subagentType, err := schema.GetSubagentTypeForAgent(tc.agent)
            require.NoError(t, err)
            assert.Equal(t, tc.expectedType, subagentType)
        })
    }
}

func TestValidateAgentSubagentPair_NewAgents(t *testing.T) {
    schema, err := LoadSchema()
    require.NoError(t, err)

    // Valid pairs
    assert.NoError(t, schema.ValidateAgentSubagentPair("typescript-pro", "general-purpose"))
    assert.NoError(t, schema.ValidateAgentSubagentPair("react-pro", "general-purpose"))
    assert.NoError(t, schema.ValidateAgentSubagentPair("backend-reviewer", "Explore"))
    assert.NoError(t, schema.ValidateAgentSubagentPair("review-orchestrator", "Plan"))
    assert.NoError(t, schema.ValidateAgentSubagentPair("impl-manager", "Plan"))

    // Invalid pairs
    assert.Error(t, schema.ValidateAgentSubagentPair("typescript-pro", "Explore"))
    assert.Error(t, schema.ValidateAgentSubagentPair("backend-reviewer", "general-purpose"))
    assert.Error(t, schema.ValidateAgentSubagentPair("impl-manager", "Explore"))
}
```

---

## P5: Hook Binary Rebuild

### Required Commands

After modifying `pkg/routing/schema.go`, rebuild all hook binaries:

```bash
# Rebuild all hooks
go build -o ~/.local/bin/gogent-validate ./cmd/gogent-validate
go build -o ~/.local/bin/gogent-load-context ./cmd/gogent-load-context
go build -o ~/.local/bin/gogent-sharp-edge ./cmd/gogent-sharp-edge
go build -o ~/.local/bin/gogent-agent-endstate ./cmd/gogent-agent-endstate
go build -o ~/.local/bin/gogent-archive ./cmd/gogent-archive

# Or use Makefile if available
make install-hooks
```

### Verification

```bash
# Test that validate hook can load updated schema
echo '{"tool_name":"Task","tool_input":{"subagent_type":"general-purpose"}}' | gogent-validate
```

---

## P5: Missing Sharp Edges (Lower Priority)

### File: `.claude/agents/memory-archivist/sharp-edges.yaml`

```yaml
# Sharp Edges for Memory Archivist Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: incomplete-archive
    severity: medium
    category: data-integrity
    description: "Archive operation interrupted before completion"
    symptom: "Partial data in memory files, missing decisions/learnings"
    solution: |
      Use atomic writes:
      1. Write to temp file first
      2. Validate JSON/JSONL structure
      3. Rename to final location
      Never append directly to production files.
    auto_inject: true

  - id: duplicate-entries
    severity: low
    category: data-integrity
    description: "Same learning archived multiple times"
    symptom: "Bloated memory files, redundant context"
    solution: |
      Check for duplicates before archiving:
      - Hash content
      - Compare with recent entries
      - Skip if duplicate found
    auto_inject: true

  - id: stale-specs
    severity: medium
    category: input
    description: "Archiving outdated specs.md"
    symptom: "Memory contains obsolete decisions"
    solution: |
      Check specs.md timestamp before archiving.
      If older than 1 hour, confirm with user before archiving.
    auto_inject: true
```

### File: `.claude/agents/haiku-scout/sharp-edges.yaml`

```yaml
# Sharp Edges for Haiku Scout Agent

version: "1.0"
updated: "2026-02-01"

sharp_edges:
  - id: scope-underestimate
    severity: medium
    category: assessment
    description: "Underestimating scope due to shallow search"
    symptom: "Task routed to wrong tier, agent fails"
    solution: |
      Use multiple glob patterns:
      - *.go, *.py, *.ts for source
      - *_test.go, test_*.py for tests
      - go.mod, package.json for deps
      Don't rely on single pattern.
    auto_inject: true

  - id: output-format-mismatch
    severity: high
    category: output
    description: "Scout metrics not matching expected JSON schema"
    symptom: "calculate-complexity.sh fails to parse"
    solution: |
      Always output valid JSON matching schema:
      {
        "total_files": int,
        "total_lines": int,
        "estimated_tokens": int,
        "recommended_tier": "haiku"|"sonnet"|"external",
        "confidence": float
      }
    auto_inject: true
```

---

## P5: Agent CLAUDE.md Updates

### Agents Needing CLAUDE.md Content

The following agents have minimal or empty CLAUDE.md files that should be populated:

| Agent | Current State | Recommended Action |
|-------|---------------|-------------------|
| `typescript-pro` | Minimal | Add TS-specific patterns, common tasks |
| `react-pro` | Minimal | Add React/Ink patterns, hooks guidance |
| `backend-reviewer` | Empty | Add review focus areas, severity guidelines |
| `frontend-reviewer` | Empty | Add accessibility, UX review checklist |
| `standards-reviewer` | Empty | Add naming conventions, complexity checks |
| `planner` | Missing | Create with strategic planning guidance |

These can be populated incrementally as agents are used and patterns emerge.

---

## Updated Files Modified List

1. `pkg/routing/schema.go`
2. `pkg/routing/schema_test.go` (NEW TESTS)
3. `.claude/routing-schema.json`
4. `.claude/agents/agents-index.json`
5. `.claude/agents/staff-architect-critical-review/agent.yaml`
6. `.claude/agents/orchestrator/agent.yaml`
7. `.claude/agents/orchestrator/sharp-edges.yaml` (NEW)
8. `.claude/agents/planner/agent.yaml`
9. `.claude/agents/planner/sharp-edges.yaml` (NEW)
10. `.claude/agents/architect/agent.yaml`
11. `.claude/agents/review-orchestrator/agent.yaml`
12. `.claude/agents/review-orchestrator/sharp-edges.yaml` (NEW)
13. `.claude/agents/gemini-slave/sharp-edges.yaml` (NEW)
14. `.claude/agents/memory-archivist/sharp-edges.yaml` (NEW)
15. `.claude/agents/haiku-scout/sharp-edges.yaml` (NEW)
16. `.claude/agents/impl-manager/agent.yaml` (NEW AGENT)
17. `.claude/agents/impl-manager/CLAUDE.md` (NEW AGENT)
18. `.claude/agents/impl-manager/sharp-edges.yaml` (NEW AGENT)
19. `cmd/gogent-orchestrator-guard/main.go` (add impl-manager to orchestrator list)
20. All implementation agent YAML files (thinking budget updates)
21. Hook binaries (rebuild required)

---

## Updated Implementation Order

1. **Phase 1 (P0)**: Go struct synchronization - unblocks validation (includes impl-manager)
2. **Phase 2 (P1)**: Staff architect tier fix + version cleanup
3. **Phase 3 (P2)**: Parallelization template removal
4. **Phase 4 (P3)**: Sharp edges creation (all 6 files)
5. **Phase 5 (P3)**: impl-manager agent creation (agent.yaml, CLAUDE.md, sharp-edges.yaml)
6. **Phase 6 (P4)**: Delegation graphs + cost budgets + thinking budgets
7. **Phase 7 (P5)**: Test updates + hook rebuilds (includes gogent-orchestrator-guard update)
8. **Phase 8 (P5)**: Agent CLAUDE.md population (ongoing)

Each phase should be a separate commit for clean rollback if needed.

---

## End of Specification

This document provides complete implementation details for upgrading the subagent schema system from v2.4.0 to v2.5.0.
