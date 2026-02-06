# Parallel Orchestration Architecture

> **Einstein Analysis** | 2026-02-06
> **Scope**: Team-based parallel agent spawning with structured I/O and background orchestration
> **Status**: Design proposal for exploration

---

## 1. Problem Statement

### Current State: The TUI Freeze

Orchestrator agents (mozart, impl-manager, review-orchestrator) spawn subagents **sequentially**, and every spawn blocks the TUI. A braintrust analysis that could run einstein + staff-architect in parallel instead waits ~2min for einstein, then ~2min for staff-architect, then ~1min for beethoven = **5 minutes of frozen TUI**.

This is the #1 UX failure of the current system. The user cannot:
- Work on other code while waiting
- Monitor progress of the orchestration
- Run a second orchestration concurrently (e.g. `/review` while `/braintrust` runs)
- Cancel gracefully and see partial results

Additionally:
- No shared state between teammates (agents can't discover each other)
- No structured I/O contracts (agents produce free-form text)
- No post-hoc visibility into what was spawned, what was asked, what was returned
- No dependency tracking between spawned agents

### The Critical Insight: Orchestrators Don't Write Code

Orchestrators (mozart, review-orchestrator, impl-manager) do not use Claude's `Write` or `Edit` tools on implementation files. They:
1. **Read** files for context
2. **Plan** what agents to spawn and with what prompts
3. **Coordinate** by spawning children and collecting results
4. **Synthesize** by passing results to downstream agents

Steps 2-4 are coordination, not implementation. The actual file writing during orchestration is:
- Creating team config directories
- Writing config.json with member/task status
- Copying schema templates
- Collecting stdout from child processes

**All of this can be done by a Go binary using native `os.WriteFile()`** - completely outside Claude's tool permission system. This means orchestration execution can run as a **background Bash process** while the TUI stays interactive.

### Desired State

- Orchestrators plan foreground (brief LLM phase), execute background (Go binary)
- `gogent-team-run` handles spawning, PID monitoring, wave advancement, config updates
- TUI returns to user immediately after planning phase
- Multiple teams can run concurrently
- Structured JSON schemas define input/output contracts per agent type
- Post-hoc tooling can reconstruct what happened from persisted artifacts

---

## 2. Current Architecture Analysis

### What Already Works

| Component | Status | Location |
|-----------|--------|----------|
| `spawn_agent` MCP tool | Working | `packages/tui/src/mcp/tools/spawnAgent.ts` |
| Process registry (PID tracking) | Working | `packages/tui/src/spawn/processRegistry.ts` |
| Relationship validation | Working | `packages/tui/src/spawn/relationshipValidation.ts` |
| Agent config lookup + effortLevel | Working | `packages/tui/src/spawn/agentConfig.ts` |
| Cost tracking per spawn | Working | Built into spawnAgent.ts |
| `Bash({run_in_background: true})` | Working | Native Claude Code feature |

### What's Missing

| Component | Gap |
|-----------|-----|
| Team config file | No shared state between teammates |
| Task DAG with blocked_by | No dependency tracking between spawned agents |
| `gogent-team-run` Go binary | No background wave scheduler |
| `gogent-team-init` Go binary | No team directory bootstrapping |
| Structured I/O schemas | No input/output contracts |
| Session directory structure | No persistence layer for team artifacts |
| `/team-status`, `/team-result` | No user-facing team monitoring commands |

### Key Constraint: Why Background Works

Claude's `Write`/`Edit` tools require foreground permission flow. But `gogent-team-run` is a **Go binary invoked via `Bash({run_in_background: true})`**. It writes files through Go's native `os.WriteFile()`, spawns `claude` CLI processes via `exec.Command()`, and monitors PIDs through OS signals. None of this touches Claude's tool permission system.

The spawned `claude` CLI processes (einstein, beethoven, etc.) DO have full tool access - they run as independent CLI sessions with `--permission-mode delegate`. They read and write files normally within their own sessions. The Go binary just orchestrates when they start and collects their JSON output.

---

## 3. Canonical Architecture: Background-First Orchestration

### 3.1 The Three-Phase Pattern

Every orchestrated workflow follows the same pattern:

```
Phase 1: PLAN (Foreground LLM, ~30s)
  Orchestrator LLM: interview → scout → plan team → write config + schemas
  TUI: Interactive (user sees planning progress)

Phase 2: EXECUTE (Background Go binary, minutes)
  gogent-team-run: spawn waves → monitor PIDs → advance → collect stdout
  TUI: FREE (user works on other things)

Phase 3: DELIVER (User-initiated)
  /team-status: Check progress anytime
  /team-result: Read final output when complete
  TUI: Interactive (user reads results when ready)
```

### 3.2 Orchestrator Backgroundability Profiles

Not all orchestrators are equal. The key variable is whether they need user interaction:

| Orchestrator | Interview Phase? | Background Profile |
|---|---|---|
| **mozart** (/braintrust) | Yes (1-3 questions + confirmation) | Foreground planning (~30s), then background execution |
| **review-orchestrator** (/review) | No (reads diff directly) | **Fully backgroundable** - no foreground LLM needed |
| **impl-manager** (/ticket) | No (reads specs.md + task list) | **Fully backgroundable** - no foreground LLM needed |

For fully-backgroundable orchestrators, the router can skip the planning LLM entirely:
1. Router reads the input (diff, specs.md)
2. Router runs `gogent-team-init` to create team dir from default template
3. Router runs `gogent-team-run` in background
4. TUI returns immediately

For interview-required orchestrators (mozart):
1. Router spawns Mozart LLM foreground (~30s for interview + planning)
2. Mozart writes team config + schemas
3. Mozart runs `gogent-team-run` in background
4. Mozart returns, TUI is free

### 3.3 Directory Structure

```
packages/tui/.claude/sessions/
  {YYMMDD}.{sessionId}/
    teams/
      {timestamp}.{team-name}/
        config.json           # Team manifest (members, tasks, deps, waves)
        stdin_einstein.json   # Input contract for einstein (filled by planner)
        stdin_staff-arch.json # Input contract for staff-architect
        stdin_beethoven.json  # Input contract for beethoven
        stdout_einstein.json  # Output from einstein (filled by agent, collected by scheduler)
        stdout_staff-arch.json
        stdout_beethoven.json
        pre-synthesis.md      # Optional: bash-extracted summary between waves
```

### 3.4 Team Config Schema

```json
{
  "$schema": "team-config-v1",
  "team_name": "braintrust-20260206-143022",
  "created_at": "2026-02-06T14:30:22Z",
  "session_id": "abc123",
  "trigger": "/braintrust",
  "background_pid": 54321,

  "orchestrator": {
    "agent_id": "uuid-mozart",
    "agent_type": "mozart",
    "phase": "executing",
    "status": "background"
  },

  "members": [
    {
      "name": "einstein",
      "agent_id": "uuid-einstein",
      "agent_type": "einstein",
      "pid": 12346,
      "role": "theoretical-analyst",
      "reports_to": "beethoven",
      "output_format": "stdout_einstein.json",
      "reads_from": ["problem-brief.md"],
      "prompt_file": "stdin_einstein.json",
      "tasks_assigned": [1],
      "status": "running",
      "started_at": "2026-02-06T14:30:25Z",
      "completed_at": null,
      "exit_code": null
    },
    {
      "name": "staff-architect",
      "agent_id": "uuid-staff-arch",
      "agent_type": "staff-architect-critical-review",
      "pid": 12347,
      "role": "practical-reviewer",
      "reports_to": "beethoven",
      "output_format": "stdout_staff-arch.json",
      "reads_from": ["problem-brief.md"],
      "prompt_file": "stdin_staff-arch.json",
      "tasks_assigned": [2],
      "status": "running",
      "started_at": "2026-02-06T14:30:25Z",
      "completed_at": null,
      "exit_code": null
    },
    {
      "name": "beethoven",
      "agent_id": null,
      "agent_type": "beethoven",
      "pid": null,
      "role": "synthesizer",
      "reports_to": "orchestrator",
      "output_format": "stdout_beethoven.json",
      "reads_from": ["stdout_einstein.json", "stdout_staff-arch.json"],
      "prompt_file": "stdin_beethoven.json",
      "tasks_assigned": [3],
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "exit_code": null
    }
  ],

  "tasks": [
    {
      "id": 1,
      "description": "Theoretical root cause analysis",
      "assigned_to": "einstein",
      "blocked_by": [],
      "wave": 1,
      "status": "in_progress"
    },
    {
      "id": 2,
      "description": "Practical implementation review",
      "assigned_to": "staff-architect",
      "blocked_by": [],
      "wave": 1,
      "status": "in_progress"
    },
    {
      "id": 3,
      "description": "Synthesize orthogonal analyses",
      "assigned_to": "beethoven",
      "blocked_by": [1, 2],
      "wave": 2,
      "status": "pending"
    }
  ],

  "waves": {
    "1": { "tasks": [1, 2], "status": "running", "started_at": "2026-02-06T14:30:25Z" },
    "2": { "tasks": [3], "status": "pending", "started_at": null }
  },

  "cost": {
    "total_usd": 0.0,
    "by_agent": {}
  }
}
```

Key additions vs. earlier draft:
- `background_pid`: PID of the `gogent-team-run` process itself (for `/team-cancel`)
- `orchestrator.phase`: tracks whether we're in planning/executing/complete
- `orchestrator.status`: "foreground" during planning, "background" during execution
- `cost`: rolling cost aggregation updated by scheduler from CLI JSON output

---

## 4. The Background Execution Engine: `gogent-team-run`

### Why a Go Binary (Not LLM, Not Bash Script)

| Option | Problem |
|--------|---------|
| LLM orchestration | Blocks TUI, expensive tokens for mechanical work |
| Bash script | Fragile PID management, no structured error handling |
| **Go binary** | Native process management, `os.WriteFile()` for config updates, signal handling, structured JSON parsing |

The Go binary is the **only component that writes to config.json**. This eliminates concurrent write problems entirely. Agents write their stdout JSON files; the scheduler reads those files and updates config.json.

### Specification

```
gogent-team-run <team-dir>

Lifecycle:
1. Read config.json from <team-dir>
2. Read agents-index.json for model + effortLevel per agent
3. For current wave (lowest wave with status != "completed"):
   a. For each task in wave with status == "pending":
      - Read stdin_{agent}.json for prompt content
      - Build CLI args: claude -p --output-format json --model {model}
      - Set env: CLAUDE_CODE_EFFORT_LEVEL={effortLevel}, GOGENT_TEAM_DIR={team-dir}
      - Spawn process, record PID in config.json
      - Set task status = "in_progress", member status = "running"
   b. Wait for all PIDs in wave to exit (os.Process.Wait)
   c. For each completed process:
      - Parse stdout JSON → write to stdout_{agent}.json
      - Extract cost from CLI output → update config.json cost
      - Update member: exit_code, completed_at, status
      - Update task: status = "completed" or "failed"
   d. Set wave status = "completed"
   e. Run optional inter-wave script if configured (e.g. gogent-team-prepare-synthesis.sh)
4. Repeat for next wave
5. All waves complete → set orchestrator.phase = "complete", exit 0

Signals:
  SIGTERM → Forward to all child processes, wait 5s, SIGKILL stragglers, exit 1
  SIGUSR1 → Write current status to stderr (for /team-status to capture)
  SIGUSR2 → Abort current wave, skip to completion with partial results

Error handling:
  Child exits non-zero → Set task status = "failed", retry once
  Retry also fails → Mark as "failed", continue wave (best-effort)
  All tasks in wave failed → Set wave status = "failed", skip subsequent waves, exit 1
```

### effortLevel Integration

`gogent-team-run` reads `agents-index.json` for each agent it spawns, just like `spawnAgent.ts` does:

```go
// Lookup effort level for this agent type
agentConfig := loadAgentConfig(agentType)
env := os.Environ()
if agentConfig.EffortLevel != "" {
    env = append(env, "CLAUDE_CODE_EFFORT_LEVEL="+agentConfig.EffortLevel)
}
```

This means the effort level config we added to agents-index.json today serves BOTH spawn paths:
- **MCP spawn_agent** (spawnAgent.ts) - for live orchestration within LLM sessions
- **gogent-team-run** (Go binary) - for background team execution

### Process Monitoring

The Go binary manages child processes directly via `exec.Command` and `os.Process.Wait`. It does NOT use the TypeScript ProcessRegistry (which lives in the TUI process). However, it writes PIDs to config.json, which the TUI can read via `/team-status`.

For integration with the TUI's ProcessRegistry (optional, Phase 5):
- `gogent-team-run` could write to a Unix socket that the TUI listens on
- Or: TUI reads config.json periodically when `/team-status` is invoked
- Recommendation: Keep it simple. TUI reads config.json. No socket needed.

---

## 5. Structured I/O Schemas

### Design Principle

Each agent type has a **default stdin schema** (what it receives) and a **default stdout schema** (what it produces). The planning phase (foreground LLM or `gogent-team-init`) copies templates into the team directory and fills in the stdin. Agents fill in stdout before completing. `gogent-team-run` collects stdout after each wave.

**The key benefit for background execution**: Because stdout is structured JSON, the Go binary can validate completeness without understanding content. And downstream agents (beethoven) can parse upstream outputs programmatically, or bash scripts can pre-process with jq between waves.

### stdin Schema (Planner → Agent)

```json
{
  "$schema": "stdin-v1",
  "agent_type": "einstein",
  "team_config": "./config.json",
  "task_id": 1,

  "context": {
    "problem_brief_path": ".claude/braintrust/problem-brief-20260206.md",
    "supplementary_files": [
      "packages/tui/src/mcp/tools/spawnAgent.ts",
      "packages/tui/src/spawn/processRegistry.ts"
    ]
  },

  "instructions": {
    "focus_areas": ["root cause", "first principles", "novel approaches"],
    "anti_scope": ["implementation details", "cost estimation"],
    "output_schema": "stdout_einstein.json",
    "max_sections": 5
  },

  "handoff": {
    "reports_to": "beethoven",
    "output_file": "stdout_einstein.json",
    "on_failure": "write partial output with status=failed"
  }
}
```

### stdout Schemas (Agent → Scheduler → Next Agent)

**Einstein stdout:**
```json
{
  "$schema": "stdout-einstein-v1",
  "agent_type": "einstein",
  "task_id": 1,
  "completed_at": null,
  "status": "pending",

  "analysis": {
    "root_cause": {
      "summary": "",
      "reasoning_chain": [],
      "confidence": null
    },
    "conceptual_frameworks": [
      {
        "framework": "",
        "applicability": "",
        "insights": []
      }
    ],
    "first_principles": {
      "decomposition": [],
      "key_assumptions": [],
      "challenged_assumptions": []
    },
    "novel_approaches": [
      {
        "approach": "",
        "rationale": "",
        "risks": [],
        "prerequisites": []
      }
    ],
    "open_questions": []
  },

  "metadata": {
    "tokens_used": null,
    "thinking_budget_used": null,
    "duration_ms": null,
    "files_read": []
  }
}
```

**Staff-Architect stdout:**
```json
{
  "$schema": "stdout-staff-architect-v1",
  "agent_type": "staff-architect-critical-review",
  "task_id": 2,
  "completed_at": null,
  "status": "pending",

  "review": {
    "assumptions_layer": {
      "hidden_assumptions": [],
      "validated_assumptions": [],
      "severity": null
    },
    "dependency_layer": {
      "external_dependencies": [],
      "internal_dependencies": [],
      "risk_rating": null
    },
    "failure_modes": [
      {
        "mode": "",
        "probability": "",
        "impact": "",
        "mitigation": ""
      }
    ],
    "cost_benefit": {
      "implementation_cost": "",
      "maintenance_cost": "",
      "benefits": [],
      "roi_assessment": ""
    },
    "testing_strategy": {
      "critical_tests": [],
      "edge_cases": [],
      "integration_concerns": []
    },
    "architecture_smells": [],
    "contractor_readiness": {
      "ready": null,
      "blockers": [],
      "recommendations": []
    }
  },

  "metadata": {
    "tokens_used": null,
    "duration_ms": null,
    "files_read": []
  }
}
```

**Beethoven stdout:**
```json
{
  "$schema": "stdout-beethoven-v1",
  "agent_type": "beethoven",
  "task_id": 3,
  "completed_at": null,
  "status": "pending",

  "synthesis": {
    "convergences": [
      {
        "topic": "",
        "einstein_position": "",
        "staff_architect_position": "",
        "confidence": "high"
      }
    ],
    "divergences": [
      {
        "topic": "",
        "einstein_position": "",
        "staff_architect_position": "",
        "resolution": "",
        "requires_user_judgment": false
      }
    ],
    "unified_recommendations": [
      {
        "recommendation": "",
        "priority": "",
        "rationale": "",
        "dissenting_view": ""
      }
    ],
    "executive_summary": "",
    "next_steps": []
  },

  "metadata": {
    "inputs_received": [],
    "tokens_used": null,
    "duration_ms": null
  }
}
```

**Work Agent stdout (go-pro, python-pro, etc.):**
```json
{
  "$schema": "stdout-worker-v1",
  "agent_type": "",
  "task_id": null,
  "status": "pending",

  "work_done": {
    "files_created": [],
    "files_modified": [],
    "tests_written": [],
    "acceptance_criteria": [
      { "criterion": "", "met": null, "evidence": "" }
    ]
  },

  "issues_encountered": [],
  "escalation_needed": false,
  "escalation_reason": null,

  "metadata": {
    "tokens_used": null,
    "duration_ms": null
  }
}
```

### Inter-Wave Processing: The Bash Script Angle

Because agents produce structured JSON, `gogent-team-run` can run an optional bash script between waves to pre-process outputs for the next wave. This is configured in config.json:

```json
"waves": {
  "1": { "tasks": [1, 2], "on_complete_script": "gogent-team-prepare-synthesis.sh" },
  "2": { "tasks": [3] }
}
```

Example pre-synthesis script:

```bash
#!/usr/bin/env bash
# gogent-team-prepare-synthesis.sh
# Run by gogent-team-run between wave 1 and wave 2
# Extracts key sections from wave 1 outputs for beethoven's efficient consumption

TEAM_DIR="$1"
EINSTEIN_OUT="$TEAM_DIR/stdout_einstein.json"
STAFFARCH_OUT="$TEAM_DIR/stdout_staff-arch.json"

{
  echo "# Pre-Synthesis Summary"
  echo ""
  echo "## Einstein: Root Cause"
  jq -r '.analysis.root_cause.summary // "No root cause identified"' "$EINSTEIN_OUT"
  echo ""
  echo "## Einstein: Novel Approaches"
  jq -r '.analysis.novel_approaches[] | "- **\(.approach)**: \(.rationale)"' "$EINSTEIN_OUT"
  echo ""
  echo "## Staff-Architect: Failure Modes"
  jq -r '.review.failure_modes[] | "- **\(.mode)** (\(.probability) prob, \(.impact) impact): \(.mitigation)"' "$STAFFARCH_OUT"
  echo ""
  echo "## Staff-Architect: Architecture Smells"
  jq -r '.review.architecture_smells[]' "$STAFFARCH_OUT"
  echo ""
  echo "## Open Questions (Both)"
  jq -r '.analysis.open_questions[]' "$EINSTEIN_OUT"
  jq -r '.review.contractor_readiness.blockers[]' "$STAFFARCH_OUT"
} > "$TEAM_DIR/pre-synthesis.md"
```

This reduces beethoven's context load. Instead of reading two full JSON files, beethoven reads a curated markdown summary plus the original files for depth where needed.

---

## 6. Orchestrator-Specific Defaults

### Schema Template Location

```
.claude/schemas/
  teams/
    braintrust.json     # Default team config for /braintrust
    review.json         # Default team config for /review
    implementation.json # Default team config for /ticket (impl-manager)
  stdin/
    einstein.json
    staff-architect.json
    beethoven.json
    backend-reviewer.json
    frontend-reviewer.json
    standards-reviewer.json
    architect-reviewer.json
    worker.json         # Generic for go-pro, python-pro, etc.
  stdout/
    einstein.json
    staff-architect.json
    beethoven.json
    backend-reviewer.json
    frontend-reviewer.json
    standards-reviewer.json
    architect-reviewer.json
    worker.json
```

### Braintrust Default Config

```json
{
  "workflow": "braintrust",
  "requires_interview": true,
  "background_profile": "interview-then-background",
  "default_members": [
    { "agent_type": "einstein", "role": "theoretical-analyst", "wave": 1 },
    { "agent_type": "staff-architect-critical-review", "role": "practical-reviewer", "wave": 1 },
    { "agent_type": "beethoven", "role": "synthesizer", "wave": 2, "blocked_by_wave": 1 }
  ],
  "inter_wave_scripts": {
    "1": "gogent-team-prepare-synthesis.sh"
  }
}
```

### Review Default Config

```json
{
  "workflow": "review",
  "requires_interview": false,
  "background_profile": "fully-backgroundable",
  "default_members": [
    { "agent_type": "backend-reviewer", "role": "backend", "wave": 1 },
    { "agent_type": "frontend-reviewer", "role": "frontend", "wave": 1 },
    { "agent_type": "standards-reviewer", "role": "standards", "wave": 1 },
    { "agent_type": "architect-reviewer", "role": "architecture", "wave": 1 }
  ],
  "inter_wave_scripts": {}
}
```

Note: `/review` is **single-wave, all-parallel**. All 4 reviewers run simultaneously in wave 1. No wave 2 needed because the review-orchestrator synthesizes findings after collection, or a synthesis agent could be added as wave 2.

### Implementation Default Config

```json
{
  "workflow": "implementation",
  "requires_interview": false,
  "background_profile": "fully-backgroundable",
  "default_members": [],
  "task_dag_from": ".claude/tmp/specs.md"
}
```

Note: impl-manager dynamically builds the task DAG from specs.md. The default config provides the workflow template; the planner fills in members and tasks.

---

## 7. Agent Self-Orientation

### The Re-Orientation Problem

Long-running agents can lose context mid-session. The team config and stdin schema serve as persistent anchors that survive context compression.

Each agent spawned by `gogent-team-run` receives in its prompt:

```
TEAM CONFIG: {path to config.json}
YOUR STDIN: {path to stdin_{agent}.json}
YOUR STDOUT: {path to stdout_{agent}.json}

If you lose track of your task:
1. Read your stdin file for your full instructions
2. Read config.json to understand your role and who you report to
3. Write your findings to your stdout file using the schema provided
```

The Go binary constructs this prompt prefix from config.json before passing it to the `claude` CLI via stdin.

### Inter-Agent Discovery

An agent can read config.json to find teammates:

```
"Who else is on my team?" → config.json members[]
"What is my task?" → config.json tasks[] filtered by assigned_to
"What am I blocked by?" → config.json tasks[].blocked_by
"Where do I write output?" → config.json members[].output_format
"Who reads my output?" → config.json members[] where reads_from includes my output
"What wave am I in?" → config.json tasks[].wave
"Is my wave the current one?" → config.json waves[N].status
```

This is particularly useful for beethoven, which needs to know where einstein and staff-architect wrote their outputs. Rather than receiving their full outputs in-prompt (expensive), beethoven reads the file paths from config.json and loads them itself.

---

## 8. User-Facing Commands

### New Slash Commands

| Command | Action | Reads From |
|---------|--------|------------|
| `/team-status` | Display all active teams, their waves, member statuses | config.json for each team dir |
| `/team-status <team-name>` | Detailed view of specific team | Specific config.json |
| `/team-result` | Display most recent completed team's final output | Latest stdout of final-wave agent |
| `/team-result <team-name>` | Display specific team's result | Specific stdout file |
| `/team-cancel` | Cancel most recent active team | Send SIGTERM to background_pid |
| `/team-cancel <team-name>` | Cancel specific team | Send SIGTERM to specific background_pid |
| `/team-log <agent>` | Show stderr/output from specific team member | Process stderr capture |
| `/teams` | List all teams in current session | Session teams/ directory |

### Example `/team-status` Output

```
ACTIVE TEAMS (2)

braintrust-20260206-143022
  Trigger: /braintrust
  Phase: executing (background PID 54321)
  Wave 1 [COMPLETE]: einstein (2m14s), staff-architect (1m58s)
  Wave 2 [RUNNING]: beethoven (running 45s)
  Cost: $3.42

review-20260206-144105
  Trigger: /review
  Phase: executing (background PID 54389)
  Wave 1 [RUNNING]: backend-reviewer (52s), frontend-reviewer (48s),
                     standards-reviewer (31s), architect-reviewer (running)
  Cost: $0.18
```

### Example `/team-result` Output

```
TEAM RESULT: braintrust-20260206-143022

Executive Summary:
  [beethoven's synthesis.executive_summary]

Unified Recommendations:
  1. [recommendation with priority]
  2. [recommendation with priority]

Convergences (high confidence): 3
Divergences (requires judgment): 1

Full analysis: .claude/sessions/260206.abc123/teams/143022.braintrust/stdout_beethoven.json
```

---

## 9. Concrete Flows

### Flow A: /braintrust (Interview-Then-Background)

```
User: /braintrust "should we use gRPC or REST for the agent spawn protocol?"

1. ROUTER (foreground):
   - Recognizes /braintrust trigger
   - Spawns Mozart LLM via Task() (foreground, brief)

2. MOZART LLM (foreground, ~30s):
   - Interview: AskUserQuestion("What's driving this decision?")
   - Scout: Task(haiku-scout) to assess codebase scope
   - Plan: Determine team composition, write Problem Brief
   - Create: gogent-team-init braintrust → team directory
   - Fill: stdin_einstein.json, stdin_staff-arch.json, stdin_beethoven.json
   - Launch: Bash({
       command: "gogent-team-run ./teams/143022.braintrust",
       run_in_background: true
     })
   - Return: "Team dispatched. Use /team-status to check progress."

3. TUI RETURNS TO USER (interactive)

4. gogent-team-run (background PID 54321):
   Wave 1:
     - Spawn: claude --model opus < stdin_einstein.json (PID 54322)
       env: CLAUDE_CODE_EFFORT_LEVEL=high
     - Spawn: claude --model opus < stdin_staff-arch.json (PID 54323)
       env: CLAUDE_CODE_EFFORT_LEVEL=high
     - Wait for both PIDs
     - Collect: stdout → stdout_einstein.json, stdout_staff-arch.json
     - Update config.json (task statuses, costs)
     - Run: gogent-team-prepare-synthesis.sh (optional)

   Wave 2:
     - Spawn: claude --model opus < stdin_beethoven.json (PID 54324)
       env: CLAUDE_CODE_EFFORT_LEVEL=high
     - Wait for PID
     - Collect: stdout → stdout_beethoven.json
     - Update config.json

   Exit 0. config.json orchestrator.phase = "complete"

5. USER (anytime):
   /team-status → sees wave progress
   /team-result → reads beethoven's synthesis
```

### Flow B: /review (Fully Backgroundable)

```
User: /review

1. ROUTER (foreground, ~5s):
   - Recognizes /review trigger
   - Reads git diff (no LLM needed)
   - Runs: gogent-team-init review → team directory
   - Fills stdin schemas with diff content (mechanical, router can do this directly)
   - Launch: Bash({
       command: "gogent-team-run ./teams/144105.review",
       run_in_background: true
     })
   - Return: "Review team dispatched (4 reviewers). Use /team-status to check."

2. TUI RETURNS TO USER (interactive) - near-instant

3. gogent-team-run (background):
   Wave 1 (all parallel):
     - Spawn: claude --model haiku < stdin_backend-reviewer.json
     - Spawn: claude --model haiku < stdin_frontend-reviewer.json
     - Spawn: claude --model haiku < stdin_standards-reviewer.json
     - Spawn: claude --model sonnet < stdin_architect-reviewer.json
     - Wait for all 4
     - Collect stdout files

   Exit 0.

4. USER: /team-result → aggregated review findings
```

### Flow C: /ticket (Fully Backgroundable, Multi-Wave)

```
User: /ticket implement auth system

1. ROUTER (foreground, ~10s):
   - Reads specs.md, builds task DAG from TODO items
   - Creates team with go-pro agents for each task
   - Identifies dependencies (auth handler before middleware before tests)
   - Launch: gogent-team-run in background

2. gogent-team-run (background):
   Wave 1: go-pro implements auth handler (no blockers)
   Wave 2: go-pro implements middleware (blocked by wave 1)
   Wave 3: go-pro writes integration tests (blocked by wave 2)
   Wave 4: code-reviewer reviews all changes
```

---

## 10. Implementation Phases

### Phase 1: Background Engine + Team Config (Foundation)

**The enabling phase.** Everything else depends on this.

**Deliverables:**
- Team config JSON schema (as specified in 3.4)
- `gogent-team-init` Go binary: creates team directory, copies schema templates
- `gogent-team-run` Go binary: wave scheduler with PID management, config.json updates, cost tracking, effortLevel injection, signal handling
- Integration with `agents-index.json` for model + effortLevel per agent

**Effort**: High (two Go binaries with process management)
**Risk**: Medium (process management is well-understood in Go, but edge cases exist)
**Unlocks**: Background orchestration for ALL orchestrator types

### Phase 2: Structured I/O Schemas (Contracts)

**Deliverables:**
- Default stdin/stdout JSON schemas for all agent types in `.claude/schemas/`
- Default team config templates for braintrust, review, implementation
- Inter-wave bash scripts (gogent-team-prepare-synthesis.sh)
- Update leaf agent prompts to write structured stdout

**Effort**: Medium (schema design + prompt engineering)
**Risk**: Low-Medium (agents must comply with schema, may need iteration)
**Unlocks**: Structured inter-agent data flow, jq-based tooling

### Phase 3: Slash Commands + TUI Integration (User Experience)

**Deliverables:**
- `/team-status`, `/team-result`, `/team-cancel`, `/team-log`, `/teams` slash commands
- Status bar integration showing active background teams
- Notification when a team completes (if TUI supports notifications)

**Effort**: Medium (skill definitions + TUI status bar work)
**Risk**: Low (reads from config.json, no complex logic)
**Unlocks**: User visibility into background work

### Phase 4: Orchestrator Prompt Rewrites (Migration)

**Deliverables:**
- Rewrite mozart.md: planning-only foreground, hands off to gogent-team-run
- Rewrite review-orchestrator.md: fully backgroundable via router
- Rewrite impl-manager.md: fully backgroundable via router
- Update /braintrust, /review, /ticket skills to use new flow

**Effort**: Medium (prompt engineering, skill updates)
**Risk**: Medium (behavioral changes to established workflows, needs testing)
**Unlocks**: The full vision - interactive TUI with background teams

### Phase 5: Bash Tooling + Advanced Features (Developer Experience)

**Deliverables:**
- `gogent-team-prepare-synthesis.sh` and other inter-wave scripts
- `gogent-team-report.sh`: generate human-readable session summary
- jq-based extraction scripts for common patterns
- ProcessRegistry integration (optional: TUI live-updates from config.json)
- Multiple concurrent teams support testing

**Effort**: Low-Medium
**Risk**: Low

---

## 11. Open Questions

### Q1: Locking on config.json

**Resolved.** Only `gogent-team-run` writes to config.json. Agents write their own stdout files and exit. The scheduler monitors PIDs via `os.Process.Wait()` and updates config.json synchronously between operations. No concurrent write problem.

### Q2: How Does Beethoven Get Its Input?

**Resolved.** Beethoven's stdin schema contains file paths to einstein and staff-architect's stdout files. Beethoven reads them itself via Claude's Read tool. Optionally, `gogent-team-prepare-synthesis.sh` runs between waves to produce a curated `pre-synthesis.md` that reduces beethoven's context load.

The Go binary does NOT do prompt engineering. It reads the stdin schema file and pipes it to `claude` CLI via stdin. The schema already contains everything the agent needs, written by the planning LLM.

### Q3: What If an Agent Fails Mid-Wave?

**Strategy: Best-effort with retry-once.**

1. Agent exits non-zero → `gogent-team-run` retries once with same stdin
2. Retry also fails → Mark task as "failed", member as "failed"
3. Continue wave - let other members complete
4. Next wave receives partial results. Downstream agents (beethoven) handle gaps gracefully
5. If ALL tasks in a wave fail → abort remaining waves, exit 1, set orchestrator.phase = "failed"

### Q4: Cost Tracking Across Background Teams

Each CLI process outputs JSON with `cost_usd`. `gogent-team-run` parses this and accumulates in config.json `cost.total_usd` and `cost.by_agent`. The TUI session tracker reads these on `/team-status`.

### Q5: Multiple Concurrent Teams

**Yes, by design.** Each team gets its own directory and `gogent-team-run` process. The TUI can show multiple active teams via `/teams`. ProcessRegistry handles cleanup on session end. SIGTERM to the TUI cascades to all `gogent-team-run` processes via process group.

### Q6: How Does the Router Create Team Config Without an LLM?

For fully-backgroundable orchestrators (review, implementation), the router needs to:
1. Read the input (git diff, specs.md)
2. Fill in stdin templates with file paths and content
3. Set up config.json with the default task DAG

This is mechanical work. Two options:
- **Router LLM does it directly** (current Sonnet, ~10s, trivial cost)
- **`gogent-team-init` Go binary does it** (instant, zero LLM cost, but less flexible)

**Recommendation**: Start with router LLM for flexibility. Migrate to Go binary for well-established workflows once templates stabilize.

---

## 12. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Agents don't comply with stdout schema | High (initially) | Medium | Validate output, fall back to raw text. Iterate prompts. |
| PID monitoring misses edge cases (zombies, OOM kills) | Medium | High | Use Go's `exec.Command.Wait()`, handle all exit conditions |
| Background orchestration loses user context | Low | Medium | Team config preserves full context, agent re-orientation protocol |
| Inter-wave bash scripts too brittle | Low | Low | Optional optimization, not required path. jq is robust for JSON. |
| Wave scheduling deadlock (circular deps) | Low | High | Validate DAG at planning time, reject cycles in gogent-team-init |
| User forgets about background team | Medium | Low | TUI status bar shows active teams, notification on completion |
| Cost runaway in background | Medium | High | config.json tracks rolling cost, gogent-team-run can enforce ceiling |

---

## 13. Summary

### The Core Architectural Move

**Separate orchestration planning (LLM, foreground) from orchestration execution (Go binary, background).**

- LLMs plan teams, write prompts, fill schemas (~30s foreground)
- `gogent-team-run` spawns processes, monitors PIDs, advances waves, collects outputs (minutes, background)
- TUI returns to user immediately after planning

### Why Background Works

Orchestrators don't use Claude's `Write`/`Edit` tools on implementation files. The coordination work (config.json updates, stdout collection, wave advancement) is done by `gogent-team-run` using Go's native `os.WriteFile()` - completely outside Claude's tool permission system. The spawned `claude` CLI processes have full tool access within their own sessions.

### What This Unlocks

1. **Parallelism**: Independent agents run simultaneously within a wave
2. **Interactivity**: TUI returns immediately, users work on other code while teams execute
3. **Concurrency**: Multiple teams run simultaneously (`/braintrust` + `/review`)
4. **Visibility**: config.json updated in real-time, `/team-status` shows progress
5. **Structured I/O**: JSON schemas enable bash/jq pre-processing between waves
6. **Audit trail**: Team directory preserves every prompt sent and every output received
7. **Cost control**: Rolling cost tracking per team, enforceable ceilings

### What Stays the Same

- Agent definitions in agents-index.json (just add effortLevel, already done)
- spawn_agent MCP tool (still works for live orchestration within LLM sessions)
- ProcessRegistry (still manages TUI-spawned processes)
- Agent prompts (updated, not replaced - add schema awareness)
- Relationship validation (gogent-team-run validates parent-child from agents-index.json)
