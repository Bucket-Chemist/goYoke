# Implementation Plan: Background Team Orchestration via `gogent-team-run`

> **Source**: KEEP sections from WEIGHTED-IMPLEMENTATION-PLAN.md Part IX
> **Date**: 2026-02-06
> **Status**: Foundational plan - pending architectural review
> **Approach**: Go binary (`gogent-team-run`) as background wave scheduler

---

## Table of Contents

1. [Phase 0: Foundation Fixes](#phase-0-foundation-fixes)
2. [Phase 1: Schema Design](#phase-1-schema-design)
   - [1A: Directory Structure](#1a-directory-structure)
   - [1B: Team Config JSON Schema (DEEP)](#1b-team-config-json-schema)
   - [1C: stdin JSON Schemas (DEEP)](#1c-stdin-json-schemas)
   - [1D: stdout JSON Schemas (DEEP)](#1d-stdout-json-schemas)
   - [1E: Prompt Envelope](#1e-prompt-envelope)
   - [1F: Default Team Templates](#1f-default-team-templates)
   - [1G: Inter-Wave Scripts](#1g-inter-wave-scripts)
3. [Phase 2: `gogent-team-run` Go Binary](#phase-2-gogent-team-run-go-binary)
4. [Phase 3: Slash Commands](#phase-3-slash-commands)
5. [Phase 4: Orchestrator Prompt Rewrites](#phase-4-orchestrator-prompt-rewrites)
6. [Quality Gates](#quality-gates)
7. [Risk Mitigations (from KEEP)](#risk-mitigations)
8. [Appendix: Failure Mode Catalog](#appendix-failure-mode-catalog)

---

## Phase 0: Foundation Fixes

**Source**: Braintrust KEEP - ProcessRegistry cleanup wiring (H4 finding)
**Effort**: 1-2 days
**Blocks**: All subsequent phases

### What & Why

ProcessRegistry has a fully implemented `cleanupAll()` method with graceful SIGTERM-then-SIGKILL escalation. The lifecycle module has `registerChildProcessCleanup()`. **But nobody ever called it.** Every spawned agent today survives a graceful TUI exit and becomes an orphan.

Additionally, dual signal handler registration exists: `index.tsx` registers SIGINT/SIGTERM handlers for terminal cleanup, then `shutdown.ts` registers its own handlers which overwrite the first set. Terminal state cleanup (cursor restoration, alternate buffer exit) never runs on graceful shutdown.

### Steps

1. **Wire ProcessRegistry cleanup to TUI shutdown**
   - File: `packages/tui/src/lifecycle/` or wherever session init occurs
   - Action: Call `registerChildProcessCleanup()` with `ProcessRegistry.cleanupAll()`
   - This is approximately a one-line fix

2. **Fix dual signal handler registration**
   - Files: `packages/tui/src/index.tsx`, `packages/tui/src/lifecycle/shutdown.ts`
   - Action: Consolidate signal handlers into a single registration chain
   - Terminal cleanup (cursor, alt buffer) must run before process cleanup

### Acceptance Criteria

- [ ] Spawning an agent, then gracefully exiting TUI → agent process terminates
- [ ] Terminal cursor and alternate buffer restored on graceful exit
- [ ] No orphan `claude` processes after TUI exit (verify with `ps aux | grep claude`)

---

## Phase 1: Schema Design

**Source**: Original Design KEEP items 1-5, 8, 10 + Braintrust KEEP items 2-4, 7
**Effort**: 2-3 days
**Blocks**: Phase 2 (Go binary needs schemas to read)

---

### 1A: Directory Structure

**Source**: Original Design KEEP - Directory structure (section 3.3)

```
packages/tui/.claude/sessions/
  {YYMMDD}.{sessionId}/
    teams/
      {timestamp}.{team-name}/
        config.json              # Team manifest (single source of truth)
        heartbeat                # Touched every 30s by gogent-team-run
        stdin_einstein.json      # Planner-written input for einstein
        stdin_staff-arch.json    # Planner-written input for staff-architect
        stdin_beethoven.json     # Planner-written input for beethoven
        stdout_einstein.json     # Agent-written output (einstein writes this itself)
        stdout_staff-arch.json   # Agent-written output
        stdout_beethoven.json    # Agent-written output
        pre-synthesis.md         # Optional: inter-wave jq extraction
```

**Conventions**:
- stdin files: `stdin_{member-name}.json` — written by planner LLM (or router for backgroundable workflows)
- stdout files: `stdout_{member-name}.json` — written by the agent itself using the Write tool
- `member-name` matches the `name` field in config.json `members[]`
- Team directory name: `{unix-timestamp}.{workflow-name}` (e.g., `1738856422.braintrust`)

**Schema template location** (for templates that get copied into team dirs):

```
.claude/schemas/
  teams/
    braintrust.json        # Default config template
    review.json            # Default config template
    implementation.json    # Default config template
  stdin/
    common.json            # Shared envelope schema
    einstein.json          # Agent-specific stdin
    staff-architect.json
    beethoven.json
    reviewer.json          # Shared for all 4 reviewer types
    worker.json            # Shared for go-pro, python-pro, etc.
  stdout/
    common.json            # Shared envelope schema
    einstein.json
    staff-architect.json
    beethoven.json
    reviewer.json
    worker.json
```

---

### 1B: Team Config JSON Schema

**Source**: Original Design KEEP - Team config JSON schema (section 3.4), simplified per WEIGHTED-PLAN recommendations. Braintrust KEEP - Budget ceiling, atomic writes, agent timeout.

This is the **single source of truth** for a running team. Only `gogent-team-run` writes to it. Agents and slash commands read it.

```json
{
  "$schema": "team-config-v1",
  "schema_version": "1.0.0",

  "team_name": "braintrust-1738856422",
  "workflow": "braintrust",
  "created_at": "2026-02-06T14:30:22Z",
  "session_id": "abc123",
  "trigger": "/braintrust",
  "team_dir": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust",

  "background_pid": null,
  "heartbeat_file": "heartbeat",
  "max_cost_usd": 15.00,

  "orchestrator": {
    "agent_type": "mozart",
    "phase": "planning",
    "status": "foreground",
    "started_at": "2026-02-06T14:30:22Z",
    "completed_at": null
  },

  "members": [
    {
      "name": "einstein",
      "agent_type": "einstein",
      "wave": 1,
      "role": "theoretical-analyst",
      "reports_to": "beethoven",
      "stdin_file": "stdin_einstein.json",
      "stdout_file": "stdout_einstein.json",
      "reads_from": [],

      "pid": null,
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "exit_code": null,
      "cost_usd": null,
      "timeout_ms": 600000,
      "retry_count": 0,
      "max_retries": 1,
      "error_message": null
    },
    {
      "name": "staff-arch",
      "agent_type": "staff-architect-critical-review",
      "wave": 1,
      "role": "practical-reviewer",
      "reports_to": "beethoven",
      "stdin_file": "stdin_staff-arch.json",
      "stdout_file": "stdout_staff-arch.json",
      "reads_from": [],

      "pid": null,
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "exit_code": null,
      "cost_usd": null,
      "timeout_ms": 600000,
      "retry_count": 0,
      "max_retries": 1,
      "error_message": null
    },
    {
      "name": "beethoven",
      "agent_type": "beethoven",
      "wave": 2,
      "role": "synthesizer",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_beethoven.json",
      "stdout_file": "stdout_beethoven.json",
      "reads_from": ["stdout_einstein.json", "stdout_staff-arch.json"],

      "pid": null,
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "exit_code": null,
      "cost_usd": null,
      "timeout_ms": 600000,
      "retry_count": 0,
      "max_retries": 1,
      "error_message": null
    }
  ],

  "tasks": [
    {
      "id": 1,
      "description": "Theoretical root cause analysis",
      "assigned_to": "einstein",
      "blocked_by": [],
      "wave": 1,
      "status": "pending"
    },
    {
      "id": 2,
      "description": "Practical implementation review",
      "assigned_to": "staff-arch",
      "blocked_by": [],
      "wave": 1,
      "status": "pending"
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
    "1": {
      "tasks": [1, 2],
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": null
    },
    "2": {
      "tasks": [3],
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": null
    }
  },

  "cost": {
    "total_usd": 0.0,
    "by_member": {},
    "budget_remaining_usd": 15.00
  },

  "timing": {
    "planning_started_at": "2026-02-06T14:30:22Z",
    "execution_started_at": null,
    "execution_completed_at": null,
    "total_duration_ms": null
  }
}
```

#### Field-by-Field Design Rationale

| Field | Type | Why It Exists | Who Reads It | Who Writes It |
|-------|------|---------------|--------------|---------------|
| `$schema` | string | Version identification for future migrations | Go binary, slash commands | Planner LLM |
| `schema_version` | semver string | Explicit version for Go binary compatibility checks | Go binary | Planner LLM |
| `team_name` | string | Human-readable identifier for `/teams` listing | Slash commands, user | Planner LLM |
| `workflow` | enum | Determines default behavior (interview?, inter-wave scripts?) | Go binary | Planner LLM |
| `created_at` | ISO 8601 | Audit trail | Slash commands | Planner LLM |
| `session_id` | string | Links team to TUI session for cost rollup | Slash commands | Planner LLM |
| `trigger` | string | What slash command created this team | `/team-status` display | Planner LLM |
| `team_dir` | abs path | Self-referential for agent self-orientation | Agents (via prompt) | Planner LLM |
| `background_pid` | int\|null | PID of `gogent-team-run` process for `/team-cancel` | `/team-cancel` | Go binary (on start) |
| `heartbeat_file` | string | Relative path to heartbeat file | Go binary (to touch), orphan cleanup | Planner LLM |
| `max_cost_usd` | float | Budget ceiling — Go binary aborts if exceeded | Go binary | Planner LLM (default from template) |
| `orchestrator.phase` | enum | `planning` → `executing` → `complete` → `failed` | `/team-status` | Go binary |
| `orchestrator.status` | enum | `foreground` → `background` | `/team-status` | Go binary |
| `members[].name` | string | File naming key: `stdin_{name}.json`, `stdout_{name}.json` | Go binary, agents | Planner LLM |
| `members[].agent_type` | string | Lookup key into `agents-index.json` for model + effortLevel | Go binary | Planner LLM |
| `members[].wave` | int | Which wave this member belongs to | Go binary (wave scheduling) | Planner LLM |
| `members[].role` | string | Human-readable description for `/team-status` | Slash commands | Planner LLM |
| `members[].reports_to` | string | Which member reads this member's output | Agent self-orientation | Planner LLM |
| `members[].stdin_file` | string | Relative path to stdin schema file | Go binary | Planner LLM |
| `members[].stdout_file` | string | Relative path to expected stdout file | Go binary (validates existence) | Planner LLM |
| `members[].reads_from` | string[] | Which stdout files this member needs | Agent self-orientation | Planner LLM |
| `members[].pid` | int\|null | OS PID of running claude process | Go binary (kill, wait) | Go binary |
| `members[].status` | enum | `pending` → `running` → `completed` → `failed` | Go binary, `/team-status` | Go binary |
| `members[].started_at` | ISO\|null | When process was spawned | `/team-status` | Go binary |
| `members[].completed_at` | ISO\|null | When process exited | `/team-status` | Go binary |
| `members[].exit_code` | int\|null | Process exit code | Go binary (retry decision) | Go binary |
| `members[].cost_usd` | float\|null | API cost for this member's session | `/team-status`, cost rollup | Go binary |
| `members[].timeout_ms` | int | Per-member timeout — Go binary kills after this | Go binary | Planner LLM (default from agents-index) |
| `members[].retry_count` | int | How many times this member has been retried | Go binary (retry logic) | Go binary |
| `members[].max_retries` | int | Maximum retries before marking failed | Go binary | Planner LLM (default: 1) |
| `members[].error_message` | string\|null | Last error message on failure | `/team-status`, debugging | Go binary |
| `tasks[].id` | int | Unique within team, referenced by `blocked_by` | Go binary, agents | Planner LLM |
| `tasks[].blocked_by` | int[] | Task IDs that must complete before this one starts | Go binary (wave validation) | Planner LLM |
| `tasks[].wave` | int | Computed from `blocked_by` DAG or explicit | Go binary | Planner LLM |
| `tasks[].status` | enum | Mirrors member status for the task | Go binary | Go binary |
| `waves.{N}.tasks` | int[] | Task IDs in this wave | Go binary | Planner LLM |
| `waves.{N}.status` | enum | `pending` → `running` → `completed` → `failed` | Go binary, `/team-status` | Go binary |
| `waves.{N}.on_complete_script` | string\|null | Bash script to run between this wave and next | Go binary | Planner LLM |
| `cost.total_usd` | float | Running total across all members | `/team-status`, budget check | Go binary |
| `cost.by_member` | map | Per-member cost breakdown | `/team-status` | Go binary |
| `cost.budget_remaining_usd` | float | `max_cost_usd - total_usd` | Go binary (budget gate) | Go binary |

#### What Was Dropped from Original Design (and Why)

| Original Field | Why Dropped |
|----------------|-------------|
| `agent_id` (UUID per member) | File-based coordination doesn't need it. `name` is the key. PIDs are for process management. |
| `output_format` field | Convention replaces it: `stdout_{name}.json`. No ambiguity. |
| `orchestrator.agent_id` | Same reason as member UUIDs. Not needed for file-based IPC. |

#### What Was Added from Braintrust KEEP

| New Field | Source | Purpose |
|-----------|--------|---------|
| `max_cost_usd` | Einstein 4.2 | Budget ceiling enforcement |
| `heartbeat_file` | Einstein 4.1 | Orphan detection |
| `members[].timeout_ms` | Staff-Arch F2 | Per-agent timeout (original had none) |
| `members[].retry_count` / `max_retries` | F1 retry logic | Track retry state |
| `members[].error_message` | F7 compliance | Debug failed agents |
| `cost.budget_remaining_usd` | Einstein 4.2 | Pre-spawn budget check |
| `schema_version` | Staff-Arch Layer 2 | CLI output format stability concern |

#### Atomic Write Contract

**Source**: Braintrust KEEP - Atomic config.json writes (Einstein 3.1)

`gogent-team-run` MUST write config.json atomically:

```go
func writeConfigAtomic(teamDir string, config *TeamConfig) error {
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal config: %w", err)
    }

    tmpFile := filepath.Join(teamDir, "config.json.tmp")
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return fmt.Errorf("write tmp: %w", err)
    }

    finalFile := filepath.Join(teamDir, "config.json")
    if err := os.Rename(tmpFile, finalFile); err != nil {
        return fmt.Errorf("rename: %w", err)
    }

    return nil
}
```

This prevents corruption if Go binary crashes mid-write. Readers (slash commands, agents) always see either the old complete file or the new complete file, never a partial write.

---

### 1C: stdin JSON Schemas

**Source**: Original Design KEEP - Structured I/O (section 5), Agent self-orientation (section 7)

stdin files are written by the **planner** (foreground LLM or router) before `gogent-team-run` starts. The Go binary reads them to construct the prompt it pipes to `claude -p`.

#### Design Principle

Every stdin file has a **common envelope** (identical structure across all agent types) plus an **agent-specific section** (`task_context`). The common envelope tells the agent:
- Who it is
- Where it lives (team context)
- What to read for instructions
- Where to write output
- What schema to follow

The agent-specific section tells it:
- What problem to solve
- What files to examine
- What focus areas to prioritize
- What anti-scope to avoid

#### Common Envelope

**Critical design rule**: Every file path in the stdin file MUST be absolute. Agents run headless via `claude -p` and use Read/Write tools that require absolute paths. The planner resolves all paths at stdin-creation time.

There are two path roots:
- **`team_dir`**: Where team artifacts live (config, stdin/stdout files, inter-wave outputs)
- **`project_root`**: Where the codebase lives (source files, .claude/ config)

The planner knows both when creating stdin files and resolves every reference.

```json
{
  "$schema": "stdin-v1",
  "schema_version": "1.0.0",

  "agent": {
    "type": "einstein",
    "name": "einstein",
    "role": "theoretical-analyst",
    "model": "opus",
    "effort_level": "high"
  },

  "paths": {
    "project_root": "/home/user/Documents/GOgent-Fortress",
    "team_dir": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust",
    "config": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/config.json",
    "my_stdin": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/stdin_einstein.json",
    "my_stdout": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/stdout_einstein.json"
  },

  "team": {
    "workflow": "braintrust",
    "wave": 1,
    "reports_to": "beethoven",
    "reads_from": []
  },

  "io": {
    "stdout_schema": "stdout-einstein-v1",

    "stdout_template": {
      "$schema": "stdout-einstein-v1",
      "schema_version": "1.0.0",
      "agent_type": "einstein",
      "task_id": 1,
      "status": "in_progress",
      "completed_at": null,
      "content": {
        "root_cause": {
          "summary": "",
          "reasoning_chain": [],
          "confidence": null
        },
        "first_principles": {
          "decomposition": [],
          "key_assumptions": [],
          "challenged_assumptions": []
        },
        "conceptual_frameworks": [],
        "novel_approaches": [],
        "open_questions": []
      },
      "metadata": {
        "files_read": [],
        "tools_used_count": null,
        "duration_ms": null,
        "self_assessment": {
          "confidence": null,
          "completeness": null,
          "caveats": []
        }
      }
    }
  },

  "task_context": {
    // Agent-specific — see below
  },

  "conventions": {
    "rules": ["agent-guidelines.md"],
    "language_conventions": []
  },

  "constraints": {
    "timeout_ms": 600000,
    "max_retries": 1,
    "budget_remaining_usd": 15.00
  }
}
```

#### How the Planner Resolves Paths

The planner (Mozart foreground LLM, or Router for fully-backgroundable workflows) is the
only entity that creates stdin files. It knows two roots:

```
project_root = process.cwd()  // or from GOGENT config
team_dir     = {session_dir}/teams/{timestamp}.{workflow}
```

Every path in the stdin file is resolved by the planner at creation time:

| Stdin Field | Resolution |
|-------------|------------|
| `paths.project_root` | `project_root` (absolute) |
| `paths.team_dir` | `team_dir` (absolute) |
| `paths.config` | `team_dir + "/config.json"` |
| `paths.my_stdin` | `team_dir + "/stdin_" + member.name + ".json"` |
| `paths.my_stdout` | `team_dir + "/stdout_" + member.name + ".json"` |
| `task_context.problem_brief.path` | `project_root + "/.claude/braintrust/..."` |
| `task_context.supplementary_files[].path` | `project_root + "/packages/tui/..."` |
| `task_context.upstream_outputs[].path` | `team_dir + "/stdout_einstein.json"` |
| `task_context.upstream_outputs[].summary_path` | `team_dir + "/pre-synthesis.md"` |

**The Go binary also resolves paths** when constructing the prompt envelope, using
`filepath.Join(teamDir, member.StdinFile)` etc. So the agent sees absolute paths in
BOTH the prompt envelope (piped to stdin) AND the stdin file (read via Read tool).

The agent never does path math. It copies paths directly into Read/Write tool calls.

#### Agent-Specific `task_context` Sections

**Einstein stdin** (`stdin_einstein.json`):

```json
{
  "task_context": {
    "problem_brief": {
      "path": "/home/user/Documents/GOgent-Fortress/.claude/braintrust/problem-brief-20260206.md",
      "summary": "Brief 1-2 sentence summary so agent knows what it's about before reading"
    },
    "supplementary_files": [
      {
        "path": "/home/user/Documents/GOgent-Fortress/packages/tui/src/mcp/tools/spawnAgent.ts",
        "reason": "Current agent spawning implementation"
      },
      {
        "path": "/home/user/Documents/GOgent-Fortress/packages/tui/src/spawn/processRegistry.ts",
        "reason": "Process tracking that would be bypassed"
      }
    ],
    "focus_areas": [
      "Root cause analysis using first principles",
      "Conceptual frameworks that illuminate the problem",
      "Novel approaches not considered in the problem brief",
      "Key assumptions that might be wrong"
    ],
    "anti_scope": [
      "Implementation details (that's Staff-Architect's job)",
      "Cost estimation",
      "Testing strategy"
    ],
    "output_guidance": {
      "max_sections": 5,
      "required_sections": ["root_cause", "first_principles", "novel_approaches"],
      "optional_sections": ["conceptual_frameworks", "open_questions"]
    }
  }
}
```

**Staff-Architect stdin** (`stdin_staff-arch.json`):

```json
{
  "task_context": {
    "problem_brief": {
      "path": "/home/user/Documents/GOgent-Fortress/.claude/braintrust/problem-brief-20260206.md",
      "summary": "Brief summary"
    },
    "implementation_artifacts": [
      {
        "path": "/home/user/Documents/GOgent-Fortress/.claude/tmp/specs.md",
        "reason": "Implementation plan to review"
      }
    ],
    "supplementary_files": [
      {
        "path": "/home/user/Documents/GOgent-Fortress/packages/tui/src/hooks/useClaudeQuery.ts",
        "reason": "Event loop concern - verify blocking claim"
      }
    ],
    "review_framework": "seven-layer",
    "focus_areas": [
      "Hidden assumptions that could break the design",
      "Dependency analysis (external + internal)",
      "Failure mode enumeration with severity",
      "Cost-benefit analysis",
      "Architecture smell detection",
      "Contractor readiness assessment"
    ],
    "anti_scope": [
      "Theoretical alternatives (that's Einstein's job)",
      "Implementation code",
      "Prompt engineering"
    ],
    "output_guidance": {
      "required_layers": ["assumptions", "dependencies", "failure_modes", "cost_benefit"],
      "optional_layers": ["testing_strategy", "architecture_smells", "contractor_readiness"]
    }
  }
}
```

**Beethoven stdin** (`stdin_beethoven.json`):

Note: Beethoven's `paths.reads_from` resolves the upstream stdout files to absolute paths.
The `team.reads_from` in the common envelope also lists these, but `upstream_outputs` here
adds the semantic context (who wrote it, what role, where the jq-extracted summary is).

```json
{
  "task_context": {
    "problem_brief": {
      "path": "/home/user/Documents/GOgent-Fortress/.claude/braintrust/problem-brief-20260206.md",
      "summary": "Brief summary"
    },
    "upstream_outputs": [
      {
        "agent": "einstein",
        "path": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/stdout_einstein.json",
        "role": "theoretical-analyst",
        "summary_path": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/pre-synthesis.md"
      },
      {
        "agent": "staff-arch",
        "path": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/stdout_staff-arch.json",
        "role": "practical-reviewer",
        "summary_path": "/home/user/.claude/sessions/260206.abc123/teams/1738856422.braintrust/pre-synthesis.md"
      }
    ],
    "synthesis_instructions": {
      "find_convergences": true,
      "resolve_divergences": true,
      "produce_unified_recommendations": true,
      "write_executive_summary": true,
      "identify_next_steps": true
    },
    "focus_areas": [
      "Where Einstein and Staff-Architect agree (high confidence items)",
      "Where they disagree (resolve with reasoned judgment)",
      "Unified recommendation list prioritized by impact",
      "Executive summary readable in 30 seconds"
    ],
    "anti_scope": [
      "Adding new analysis not present in either input",
      "Choosing sides without justification",
      "Rehashing the entire problem brief"
    ]
  }
}
```

**Reviewer stdin** (`stdin_backend-reviewer.json`, etc.):

```json
{
  "task_context": {
    "diff": {
      "content_path": ".claude/tmp/review-diff.patch",
      "base_branch": "main",
      "files_changed": ["pkg/routing/validator.go", "pkg/routing/events.go"]
    },
    "review_domain": "backend",
    "focus_areas": [
      "API design and consistency",
      "Error handling patterns",
      "Security vulnerabilities (OWASP top 10)",
      "Database query patterns",
      "Authentication/authorization"
    ],
    "severity_levels": ["critical", "high", "medium", "low", "nit"],
    "output_guidance": {
      "group_by": "severity",
      "include_line_references": true,
      "include_fix_suggestions": true
    }
  }
}
```

**Worker stdin** (`stdin_go-pro-1.json`, etc.):

```json
{
  "task_context": {
    "ticket": {
      "id": "IMPL-001",
      "title": "Implement auth handler",
      "description": "Full ticket description",
      "acceptance_criteria": [
        "Handler accepts JWT tokens",
        "Returns 401 on invalid token",
        "Extracts user_id from claims"
      ]
    },
    "target_files": [
      {
        "path": "pkg/auth/handler.go",
        "action": "create",
        "description": "Main auth handler"
      }
    ],
    "reference_files": [
      {
        "path": "pkg/auth/types.go",
        "reason": "Type definitions to use"
      }
    ],
    "conventions": ["go.md"],
    "blocked_by_outputs": [],
    "testing_requirements": {
      "table_driven": true,
      "coverage_target": "happy path + error cases"
    }
  }
}
```

#### How `gogent-team-run` Uses stdin Files

The Go binary does **not** pass the raw JSON to the Claude CLI. It reads the stdin file and constructs a **prompt envelope** (see section 1E) that tells the agent how to orient itself. The stdin file remains on disk for the agent to re-read if it loses context during a long session.

---

### 1D: stdout JSON Schemas

**Source**: Original Design KEEP - Structured I/O (section 5). Braintrust KEEP - Agent timeout handling, failure mode catalog.

stdout files are written by the **agents themselves** using Claude's Write tool during their session. The Go binary validates their existence after the agent exits and extracts metadata.

#### Design Principle

Agents write their own stdout files. This is superior to the Go binary parsing CLI output because:
1. Agents can write **partial results** before crashing
2. Agents incrementally update the file as they work
3. The Go binary doesn't need to parse free-form LLM output
4. The schema is enforced by prompt engineering, validated post-hoc

#### Common Envelope

Every stdout file follows this envelope:

```json
{
  "$schema": "stdout-{agent-type}-v1",
  "schema_version": "1.0.0",
  "agent_type": "einstein",
  "task_id": 1,
  "status": "completed",
  "completed_at": "2026-02-06T14:32:45Z",

  "content": {
    // Agent-specific structured content — see below
  },

  "metadata": {
    "files_read": [],
    "tools_used_count": null,
    "duration_ms": null,
    "self_assessment": {
      "confidence": "high",
      "completeness": "full",
      "caveats": []
    }
  }
}
```

| Field | Purpose | Who Fills It |
|-------|---------|-------------|
| `$schema` | Version identification | Agent (from stdin instructions) |
| `status` | `completed` \| `failed` \| `partial` | Agent (updates as it works) |
| `completed_at` | When the agent finished writing | Agent |
| `content` | The actual analysis/review/work | Agent |
| `metadata.files_read` | What the agent examined | Agent |
| `metadata.self_assessment` | Agent's confidence in its own output | Agent |

#### Agent-Specific `content` Sections

**Einstein stdout** (`stdout_einstein.json`):

```json
{
  "content": {
    "root_cause": {
      "summary": "One paragraph root cause identification",
      "reasoning_chain": [
        "Step 1: observed X",
        "Step 2: which implies Y",
        "Step 3: root cause is Z"
      ],
      "confidence": "high"
    },
    "first_principles": {
      "decomposition": [
        "Principle 1: description",
        "Principle 2: description"
      ],
      "key_assumptions": [
        {
          "assumption": "The event loop blocks during query()",
          "validity": "partially valid",
          "evidence": "async iterator yields between events"
        }
      ],
      "challenged_assumptions": [
        {
          "assumption": "What was assumed",
          "challenge": "Why it might be wrong",
          "implication": "What changes if it's wrong"
        }
      ]
    },
    "conceptual_frameworks": [
      {
        "framework": "Name of framework",
        "applicability": "How it applies here",
        "insights": ["Insight 1", "Insight 2"]
      }
    ],
    "novel_approaches": [
      {
        "approach": "Brief name",
        "rationale": "Why this could work",
        "risks": ["Risk 1", "Risk 2"],
        "prerequisites": ["What must be true first"],
        "estimated_effort": "rough effort"
      }
    ],
    "open_questions": [
      "Question that remains unanswered"
    ]
  }
}
```

**Staff-Architect stdout** (`stdout_staff-arch.json`):

```json
{
  "content": {
    "assumptions_layer": {
      "hidden_assumptions": [
        {
          "assumption": "What's assumed",
          "evidence": "Why we think this is assumed",
          "risk_if_wrong": "high",
          "validation_needed": "How to test"
        }
      ],
      "validated_assumptions": [
        {
          "assumption": "What was validated",
          "evidence": "How we validated"
        }
      ]
    },
    "dependency_layer": {
      "external_dependencies": [
        {
          "dependency": "Claude CLI JSON output format",
          "version_coupling": "tight",
          "stability": "unknown",
          "risk": "medium",
          "mitigation": "Parse defensively, pin version"
        }
      ],
      "internal_dependencies": [
        {
          "component": "agents-index.json",
          "coupling": "schema changes break binary",
          "risk": "medium"
        }
      ]
    },
    "failure_modes": [
      {
        "id": "F1",
        "mode": "Agent exits non-zero",
        "trigger": "Bug, timeout, OOM",
        "probability": "high",
        "impact": "medium",
        "detection": "Exit code",
        "recovery": "Retry once",
        "design_coverage": "yes"
      }
    ],
    "cost_benefit": {
      "implementation_cost": "14-19 days",
      "maintenance_cost": "Low - matches existing cmd/ pattern",
      "benefits": ["Interactive TUI", "Parallel execution"],
      "roi_assessment": "Positive if orchestration used weekly"
    },
    "architecture_smells": [
      {
        "smell": "Dual process management",
        "symptom": "Two systems tracking processes",
        "risk": "Orphans, split cost tracking",
        "recommendation": "Accept for now, unify in Phase 5"
      }
    ],
    "contractor_readiness": {
      "ready": false,
      "blockers": ["Failure mode handling unspecified"],
      "recommendations": ["Add acceptance criteria per phase"]
    }
  }
}
```

**Beethoven stdout** (`stdout_beethoven.json`):

```json
{
  "content": {
    "convergences": [
      {
        "topic": "What both analysts agree on",
        "einstein_position": "Einstein's take",
        "staff_architect_position": "Staff-Architect's take",
        "confidence": "high",
        "synthesis": "Unified statement"
      }
    ],
    "divergences": [
      {
        "topic": "Where they disagree",
        "einstein_position": "Einstein's take",
        "staff_architect_position": "Staff-Architect's take",
        "resolution": "How this was resolved",
        "resolution_rationale": "Why this resolution",
        "requires_user_judgment": false
      }
    ],
    "unified_recommendations": [
      {
        "id": 1,
        "recommendation": "What to do",
        "priority": "critical",
        "rationale": "Why",
        "effort": "rough estimate",
        "dissenting_view": "Any disagreement"
      }
    ],
    "executive_summary": "30-second readable summary of everything",
    "next_steps": [
      {
        "step": "What to do next",
        "owner": "who",
        "blocking": true
      }
    ],
    "decision_points": [
      {
        "decision": "What the user must decide",
        "options": ["Option A", "Option B"],
        "recommendation": "Which option and why"
      }
    ]
  }
}
```

**Reviewer stdout** (`stdout_backend-reviewer.json`, etc.):

```json
{
  "content": {
    "review_domain": "backend",
    "approval_status": "changes_requested",
    "findings": [
      {
        "severity": "high",
        "file": "pkg/routing/validator.go",
        "line": 42,
        "category": "security",
        "title": "SQL injection via unparameterized query",
        "description": "Detailed explanation",
        "suggestion": "Use parameterized query instead",
        "code_before": "db.Query(fmt.Sprintf(...))",
        "code_after": "db.Query(\"SELECT ... WHERE id = $1\", id)"
      }
    ],
    "summary": {
      "critical": 0,
      "high": 1,
      "medium": 3,
      "low": 2,
      "nit": 1,
      "overall_quality": "Good with minor issues"
    }
  }
}
```

**Worker stdout** (`stdout_go-pro-1.json`, etc.):

```json
{
  "content": {
    "work_done": {
      "files_created": [
        {
          "path": "pkg/auth/handler.go",
          "description": "JWT auth handler implementation",
          "lines_of_code": 85
        }
      ],
      "files_modified": [
        {
          "path": "pkg/auth/router.go",
          "description": "Added auth middleware to router",
          "changes_summary": "Added 3 lines to register middleware"
        }
      ],
      "tests_written": [
        {
          "path": "pkg/auth/handler_test.go",
          "test_count": 6,
          "coverage_areas": ["valid JWT", "expired JWT", "malformed JWT", "missing header"]
        }
      ]
    },
    "acceptance_criteria": [
      {
        "criterion": "Handler accepts JWT tokens",
        "met": true,
        "evidence": "TestValidJWT passes, handler extracts claims"
      },
      {
        "criterion": "Returns 401 on invalid token",
        "met": true,
        "evidence": "TestInvalidJWT, TestExpiredJWT both verify 401 response"
      }
    ],
    "issues_encountered": [
      "jwt-go v3 deprecated, used golang-jwt/jwt/v5 instead"
    ],
    "escalation_needed": false,
    "escalation_reason": null,
    "build_status": {
      "compiles": true,
      "tests_pass": true,
      "lint_clean": true
    }
  }
}
```

#### Stdout Validation by Go Binary

After an agent exits, `gogent-team-run` performs lightweight validation:

```go
func validateStdout(teamDir string, member *Member) error {
    path := filepath.Join(teamDir, member.StdoutFile)

    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("stdout file missing: %s", member.StdoutFile)
    }

    var envelope StdoutEnvelope
    if err := json.Unmarshal(data, &envelope); err != nil {
        return fmt.Errorf("stdout not valid JSON: %w", err)
    }

    if envelope.Schema == "" {
        return fmt.Errorf("stdout missing $schema field")
    }

    if envelope.Status == "" {
        return fmt.Errorf("stdout missing status field")
    }

    // Content validation is NOT done by Go binary.
    // It's prompt engineering's job to ensure quality.
    // Go binary only checks structural envelope.

    return nil
}
```

If validation fails, Go binary:
1. Marks member status as `"failed"`
2. Sets `error_message` to the validation error
3. Checks `retry_count < max_retries` → retry if possible
4. Otherwise continues wave (best-effort)

---

### 1E: Prompt Envelope

**Source**: Original Design KEEP - Agent self-orientation (section 7)

The Go binary constructs a prompt from the stdin file and pipes it to `claude -p`. This prompt is the agent's initial orientation — it tells the agent who it is, where to find its instructions, and where to write output.

#### Template

All `{...}` placeholders are resolved by the Go binary from config.json and the stdin file's `paths` block. The agent receives a prompt with zero relative paths — every file reference is absolute and directly usable with Read/Write tools.

```
AGENT: {agent_type}
ROLE: {role}
TEAM: {team_name} (workflow: {workflow})

---

## Step 1: Read Your Instructions

Your complete task instructions are at:

  {paths.my_stdin}

Read this file now. It contains:
- `paths` — absolute paths to every file you need
- `io.stdout_template` — the exact JSON structure you must write
- `task_context` — your specific task, focus areas, and context files

## Step 2: Write Your Output

Write your structured output to:

  {paths.my_stdout}

Use the Write tool with exactly this path. Your output MUST be valid JSON
matching the template in your stdin file under `io.stdout_template`.

**Workflow:**
1. First write: Copy the `io.stdout_template` from your stdin file, set
   `status: "in_progress"`, and write it to {paths.my_stdout}
2. As you work: Update the file with your findings (fill in content fields)
3. When done: Set `status: "completed"` and `completed_at` to current time
4. On failure: Set `status: "failed"` with whatever partial content you have

## Step 3: Read Context Files

Your stdin file lists all files to read under `task_context`. Every path
is absolute — use the Read tool directly on each path.
{reads_from_section}

## Key Paths

| What | Path |
|------|------|
| Your instructions | {paths.my_stdin} |
| Your output | {paths.my_stdout} |
| Team config | {paths.config} |
| Project root | {paths.project_root} |

## Self-Orientation Protocol

If you lose track of your task during a long session:
1. Read {paths.my_stdin} for your full instructions
2. Read {paths.config} to understand your role and team
3. Read {paths.my_stdout} to see what you've already written
4. Continue from where you left off

## Constraints

- Timeout: {timeout_ms}ms
- Budget remaining: ${budget_remaining_usd}
- Do not modify config.json (only gogent-team-run writes to it)
- Do not modify other members' files

---

Read your stdin file now and begin.
```

Where `{reads_from_section}` expands to:

```
- Upstream outputs to read:
  - {team_dir}/stdout_einstein.json (from einstein, theoretical-analyst)
  - {team_dir}/stdout_staff-arch.json (from staff-arch, practical-reviewer)
```

(Only present for wave 2+ members that have `reads_from` entries)

#### Go Binary Implementation

```go
func buildPromptEnvelope(teamDir, projectRoot string, config *TeamConfig, member *Member) string {
    // Resolve all paths to absolute — the agent must never see a relative path
    stdinPath := filepath.Join(teamDir, member.StdinFile)
    stdoutPath := filepath.Join(teamDir, member.StdoutFile)
    configPath := filepath.Join(teamDir, "config.json")

    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("AGENT: %s\n", member.AgentType))
    sb.WriteString(fmt.Sprintf("ROLE: %s\n", member.Role))
    sb.WriteString(fmt.Sprintf("TEAM: %s (workflow: %s)\n", config.TeamName, config.Workflow))
    sb.WriteString("\n---\n\n")

    // Step 1: Read instructions
    sb.WriteString("## Step 1: Read Your Instructions\n\n")
    sb.WriteString("Your complete task instructions are at:\n\n")
    sb.WriteString(fmt.Sprintf("  %s\n\n", stdinPath))
    sb.WriteString("Read this file now. It contains:\n")
    sb.WriteString("- `paths` — absolute paths to every file you need\n")
    sb.WriteString("- `io.stdout_template` — the exact JSON structure you must write\n")
    sb.WriteString("- `task_context` — your specific task, focus areas, and context files\n\n")

    // Step 2: Write output
    sb.WriteString("## Step 2: Write Your Output\n\n")
    sb.WriteString("Write your structured output to:\n\n")
    sb.WriteString(fmt.Sprintf("  %s\n\n", stdoutPath))
    sb.WriteString("Use the Write tool with exactly this path.\n")
    sb.WriteString("Copy `io.stdout_template` from your stdin file, fill in content, write it.\n")
    sb.WriteString("Start with status: \"in_progress\", update to \"completed\" when done.\n")
    sb.WriteString("On unrecoverable error, set status: \"failed\" with partial results.\n\n")

    // Step 3: Context files
    sb.WriteString("## Step 3: Read Context Files\n\n")
    sb.WriteString("Your stdin file lists all files to read under `task_context`.\n")
    sb.WriteString("Every path is absolute — use the Read tool directly.\n")

    if len(member.ReadsFrom) > 0 {
        sb.WriteString("\nUpstream outputs to read (from previous wave):\n")
        for _, rf := range member.ReadsFrom {
            sb.WriteString(fmt.Sprintf("  - %s\n", filepath.Join(teamDir, rf)))
        }
    }

    // Key paths table
    sb.WriteString("\n## Key Paths\n\n")
    sb.WriteString("| What | Path |\n")
    sb.WriteString("|------|------|\n")
    sb.WriteString(fmt.Sprintf("| Your instructions | %s |\n", stdinPath))
    sb.WriteString(fmt.Sprintf("| Your output | %s |\n", stdoutPath))
    sb.WriteString(fmt.Sprintf("| Team config | %s |\n", configPath))
    sb.WriteString(fmt.Sprintf("| Project root | %s |\n", projectRoot))

    // Self-orientation
    sb.WriteString("\n## Self-Orientation Protocol\n\n")
    sb.WriteString("If you lose track of your task during a long session:\n")
    sb.WriteString(fmt.Sprintf("1. Read %s for your full instructions\n", stdinPath))
    sb.WriteString(fmt.Sprintf("2. Read %s to understand your role and team\n", configPath))
    sb.WriteString(fmt.Sprintf("3. Read %s to see what you've already written\n", stdoutPath))
    sb.WriteString("4. Continue from where you left off\n\n")

    // Constraints
    sb.WriteString("## Constraints\n\n")
    sb.WriteString(fmt.Sprintf("- Timeout: %dms\n", member.TimeoutMs))
    sb.WriteString(fmt.Sprintf("- Budget remaining: $%.2f\n", config.Cost.BudgetRemainingUSD))
    sb.WriteString("- Do not modify config.json\n")
    sb.WriteString("- Do not modify other members' files\n\n")

    sb.WriteString("---\n\nRead your stdin file now and begin.\n")

    return sb.String()
}
```

---

### 1F: Default Team Templates

**Source**: Original Design KEEP - Orchestrator backgroundability profiles (section 3.2)

#### Braintrust Template (`.claude/schemas/teams/braintrust.json`)

```json
{
  "$schema": "team-config-v1",
  "schema_version": "1.0.0",
  "workflow": "braintrust",
  "max_cost_usd": 15.00,
  "heartbeat_file": "heartbeat",

  "members": [
    {
      "name": "einstein",
      "agent_type": "einstein",
      "wave": 1,
      "role": "theoretical-analyst",
      "reports_to": "beethoven",
      "stdin_file": "stdin_einstein.json",
      "stdout_file": "stdout_einstein.json",
      "reads_from": [],
      "timeout_ms": 600000,
      "max_retries": 1
    },
    {
      "name": "staff-arch",
      "agent_type": "staff-architect-critical-review",
      "wave": 1,
      "role": "practical-reviewer",
      "reports_to": "beethoven",
      "stdin_file": "stdin_staff-arch.json",
      "stdout_file": "stdout_staff-arch.json",
      "reads_from": [],
      "timeout_ms": 600000,
      "max_retries": 1
    },
    {
      "name": "beethoven",
      "agent_type": "beethoven",
      "wave": 2,
      "role": "synthesizer",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_beethoven.json",
      "stdout_file": "stdout_beethoven.json",
      "reads_from": ["stdout_einstein.json", "stdout_staff-arch.json"],
      "timeout_ms": 600000,
      "max_retries": 1
    }
  ],

  "tasks": [
    { "id": 1, "description": "Theoretical analysis", "assigned_to": "einstein", "blocked_by": [], "wave": 1 },
    { "id": 2, "description": "Practical review", "assigned_to": "staff-arch", "blocked_by": [], "wave": 1 },
    { "id": 3, "description": "Synthesis", "assigned_to": "beethoven", "blocked_by": [1, 2], "wave": 2 }
  ],

  "waves": {
    "1": { "tasks": [1, 2], "on_complete_script": "gogent-team-prepare-synthesis.sh" },
    "2": { "tasks": [3], "on_complete_script": null }
  }
}
```

**Background Profile**: `interview-then-background`. Mozart plans in foreground (~30s), then launches `gogent-team-run`.

#### Review Template (`.claude/schemas/teams/review.json`)

```json
{
  "$schema": "team-config-v1",
  "schema_version": "1.0.0",
  "workflow": "review",
  "max_cost_usd": 5.00,
  "heartbeat_file": "heartbeat",

  "members": [
    {
      "name": "backend-reviewer",
      "agent_type": "backend-reviewer",
      "wave": 1,
      "role": "backend",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_backend-reviewer.json",
      "stdout_file": "stdout_backend-reviewer.json",
      "reads_from": [],
      "timeout_ms": 300000,
      "max_retries": 0
    },
    {
      "name": "frontend-reviewer",
      "agent_type": "frontend-reviewer",
      "wave": 1,
      "role": "frontend",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_frontend-reviewer.json",
      "stdout_file": "stdout_frontend-reviewer.json",
      "reads_from": [],
      "timeout_ms": 300000,
      "max_retries": 0
    },
    {
      "name": "standards-reviewer",
      "agent_type": "standards-reviewer",
      "wave": 1,
      "role": "standards",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_standards-reviewer.json",
      "stdout_file": "stdout_standards-reviewer.json",
      "reads_from": [],
      "timeout_ms": 300000,
      "max_retries": 0
    },
    {
      "name": "architect-reviewer",
      "agent_type": "architect-reviewer",
      "wave": 1,
      "role": "architecture",
      "reports_to": "orchestrator",
      "stdin_file": "stdin_architect-reviewer.json",
      "stdout_file": "stdout_architect-reviewer.json",
      "reads_from": [],
      "timeout_ms": 300000,
      "max_retries": 0
    }
  ],

  "tasks": [
    { "id": 1, "description": "Backend review", "assigned_to": "backend-reviewer", "blocked_by": [], "wave": 1 },
    { "id": 2, "description": "Frontend review", "assigned_to": "frontend-reviewer", "blocked_by": [], "wave": 1 },
    { "id": 3, "description": "Standards review", "assigned_to": "standards-reviewer", "blocked_by": [], "wave": 1 },
    { "id": 4, "description": "Architecture review", "assigned_to": "architect-reviewer", "blocked_by": [], "wave": 1 }
  ],

  "waves": {
    "1": { "tasks": [1, 2, 3, 4], "on_complete_script": null }
  }
}
```

**Background Profile**: `fully-backgroundable`. Router reads git diff, fills stdin templates, launches `gogent-team-run` directly. No foreground LLM needed.

#### Implementation Template (`.claude/schemas/teams/implementation.json`)

```json
{
  "$schema": "team-config-v1",
  "schema_version": "1.0.0",
  "workflow": "implementation",
  "max_cost_usd": 20.00,
  "heartbeat_file": "heartbeat",

  "members": [],
  "tasks": [],
  "waves": {},

  "_note": "Members, tasks, and waves are dynamically populated from specs.md task DAG by the planner"
}
```

**Background Profile**: `fully-backgroundable`. Router reads specs.md, builds task DAG, populates members/tasks/waves, launches `gogent-team-run`.

---

### 1G: Inter-Wave Scripts

**Source**: Original Design KEEP - Inter-wave scripts (section 5)

Scripts run by `gogent-team-run` between waves. Configured via `waves.{N}.on_complete_script`. The Go binary calls them with the team directory as the first argument.

#### `gogent-team-prepare-synthesis.sh`

Purpose: Extract key sections from Wave 1 outputs into a curated markdown summary for Beethoven. This reduces Beethoven's context load from ~20K tokens (two full JSON files) to ~3K tokens (focused summary) + option to read full files for depth.

```bash
#!/usr/bin/env bash
# gogent-team-prepare-synthesis.sh
# Called by gogent-team-run between wave 1 and wave 2
# Extracts key sections from wave 1 outputs for efficient consumption
#
# Usage: gogent-team-prepare-synthesis.sh <team-dir>

set -euo pipefail

TEAM_DIR="$1"
EINSTEIN_OUT="$TEAM_DIR/stdout_einstein.json"
STAFFARCH_OUT="$TEAM_DIR/stdout_staff-arch.json"
OUTPUT="$TEAM_DIR/pre-synthesis.md"

{
  echo "# Pre-Synthesis Summary"
  echo ""
  echo "> Auto-generated by gogent-team-prepare-synthesis.sh"
  echo "> Full analyses available in stdout_einstein.json and stdout_staff-arch.json"
  echo ""

  echo "## Einstein: Root Cause"
  jq -r '.content.root_cause.summary // "No root cause identified"' "$EINSTEIN_OUT" 2>/dev/null || echo "(einstein output missing or malformed)"
  echo ""

  echo "## Einstein: Key Assumptions Challenged"
  jq -r '.content.first_principles.challenged_assumptions[]? | "- **\(.assumption)**: \(.challenge)"' "$EINSTEIN_OUT" 2>/dev/null || echo "(none)"
  echo ""

  echo "## Einstein: Novel Approaches"
  jq -r '.content.novel_approaches[]? | "- **\(.approach)**: \(.rationale)"' "$EINSTEIN_OUT" 2>/dev/null || echo "(none)"
  echo ""

  echo "## Staff-Architect: Critical Failure Modes"
  jq -r '.content.failure_modes[]? | select(.probability == "high" or .impact == "high") | "- **\(.mode)** [\(.probability) prob, \(.impact) impact]: \(.mitigation)"' "$STAFFARCH_OUT" 2>/dev/null || echo "(none)"
  echo ""

  echo "## Staff-Architect: Architecture Smells"
  jq -r '.content.architecture_smells[]?.smell' "$STAFFARCH_OUT" 2>/dev/null || echo "(none)"
  echo ""

  echo "## Staff-Architect: Contractor Readiness"
  jq -r '.content.contractor_readiness | "Ready: \(.ready // "unknown")\nBlockers: \(.blockers // [] | join(", "))"' "$STAFFARCH_OUT" 2>/dev/null || echo "(unknown)"
  echo ""

  echo "## Open Questions (Both)"
  echo "### From Einstein:"
  jq -r '.content.open_questions[]?' "$EINSTEIN_OUT" 2>/dev/null | sed 's/^/- /' || echo "- (none)"
  echo ""
  echo "### From Staff-Architect:"
  jq -r '.content.contractor_readiness.blockers[]?' "$STAFFARCH_OUT" 2>/dev/null | sed 's/^/- /' || echo "- (none)"

} > "$OUTPUT"

echo "Pre-synthesis summary written to $OUTPUT"
```

The script uses `2>/dev/null || echo "(fallback)"` patterns so it handles missing or malformed JSON gracefully — an agent that failed to write proper JSON doesn't crash the inter-wave script.

---

## Phase 2: `gogent-team-run` Go Binary

**Source**: Original Design KEEP - Wave-based execution (sections 4, 9), Signal handling (section 4), Cost tracking (section 3.4), effortLevel integration (section 4). Braintrust KEEP - Budget ceiling, orphan detection, atomic writes, agent timeout.
**Effort**: 5-7 days
**Blocks**: Phase 3 (slash commands need running teams to query)

### 2.1 Binary Location & Interface

```
cmd/gogent-team-run/main.go

Usage: gogent-team-run <team-dir>

Reads config.json from <team-dir>, executes waves, updates config.json.
Exits 0 on success, 1 on failure, 2 on budget exceeded.
```

Fits the existing `cmd/` pattern alongside `gogent-validate`, `gogent-sharp-edge`, `gogent-archive`, `gogent-load-context`, `gogent-agent-endstate`.

### 2.2 Core Loop (Pseudocode)

```
main(teamDir):
  config = readConfig(teamDir)
  config.background_pid = os.Getpid()
  config.orchestrator.phase = "executing"
  config.orchestrator.status = "background"
  config.timing.execution_started_at = now()
  writeConfigAtomic(teamDir, config)

  agentsIndex = loadAgentsIndex()   // .claude/agents/agents-index.json
  setupSignalHandlers()             // SIGTERM, SIGINT
  startHeartbeat(teamDir, 30s)      // Touch heartbeat file every 30s

  for waveNum in sorted(config.waves.keys()):
    wave = config.waves[waveNum]
    if wave.status == "completed":
      continue

    // Budget gate (Braintrust KEEP)
    if config.cost.budget_remaining_usd <= 0:
      config.orchestrator.phase = "failed"
      writeConfigAtomic(teamDir, config)
      exit(2)  // Budget exceeded

    wave.status = "running"
    wave.started_at = now()
    writeConfigAtomic(teamDir, config)

    // Spawn all members in this wave
    var wg sync.WaitGroup
    for member in membersInWave(config, waveNum):
      // Pre-spawn budget check
      if config.cost.budget_remaining_usd <= 0:
        member.status = "failed"
        member.error_message = "Budget exceeded before spawn"
        continue

      wg.Add(1)
      go spawnAndWait(teamDir, config, member, agentsIndex, &wg)

    wg.Wait()

    // Collect results, update config
    allFailed = true
    for member in membersInWave(config, waveNum):
      if member.status == "completed":
        allFailed = false
      // Update cost
      config.cost.total_usd += member.cost_usd
      config.cost.by_member[member.name] = member.cost_usd
      config.cost.budget_remaining_usd = config.max_cost_usd - config.cost.total_usd

    if allFailed:
      wave.status = "failed"
      config.orchestrator.phase = "failed"
      writeConfigAtomic(teamDir, config)
      exit(1)

    wave.status = "completed"
    wave.completed_at = now()
    writeConfigAtomic(teamDir, config)

    // Inter-wave script (Original Design KEEP)
    if wave.on_complete_script != "":
      runScript(teamDir, wave.on_complete_script)

  // All waves complete
  config.orchestrator.phase = "complete"
  config.timing.execution_completed_at = now()
  config.timing.total_duration_ms = timeSince(config.timing.execution_started_at)
  writeConfigAtomic(teamDir, config)
  exit(0)
```

### 2.3 Process Spawning Detail

```go
func spawnAndWait(teamDir string, config *TeamConfig, member *Member, agentsIndex *AgentsIndex, wg *sync.WaitGroup) {
    defer wg.Done()

    agentConfig := agentsIndex.Lookup(member.AgentType)

    // Build prompt envelope (section 1E)
    prompt := buildPromptEnvelope(teamDir, config, member)

    // Build CLI args
    args := []string{
        "-p",                          // Pipe mode: read prompt from stdin
        "--output-format", "json",     // JSON output for cost extraction
        "--model", agentConfig.Model,  // From agents-index.json
        "--permission-mode", "delegate", // Auto-approve tool use
    }

    cmd := exec.Command("claude", args...)
    cmd.Stdin = strings.NewReader(prompt)
    cmd.Dir = projectRoot  // So agent's Read/Write tools resolve relative to project

    // Environment (Original Design KEEP - effortLevel integration)
    env := os.Environ()
    if agentConfig.EffortLevel != "" {
        env = append(env, "CLAUDE_CODE_EFFORT_LEVEL="+agentConfig.EffortLevel)
    }
    env = append(env, fmt.Sprintf("GOGENT_NESTING_LEVEL=%d", 2))  // Team agents are level 2
    env = append(env, fmt.Sprintf("GOGENT_PARENT_AGENT=%s", config.Orchestrator.AgentType))
    env = append(env, fmt.Sprintf("GOGENT_TEAM_DIR=%s", teamDir))
    cmd.Env = env

    // Capture CLI stdout for cost extraction
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = os.Stderr  // Let agent stderr flow to team runner stderr

    // Update member state
    member.Status = "running"
    member.StartedAt = now()

    if err := cmd.Start(); err != nil {
        member.Status = "failed"
        member.ErrorMessage = fmt.Sprintf("spawn failed: %v", err)
        writeConfigAtomic(teamDir, config)  // Record PID
        return
    }

    member.PID = cmd.Process.Pid
    writeConfigAtomic(teamDir, config)  // Record PID for /team-cancel

    // Wait with timeout (Braintrust KEEP - agent timeout)
    done := make(chan error, 1)
    go func() { done <- cmd.Wait() }()

    select {
    case err := <-done:
        member.CompletedAt = now()
        member.ExitCode = cmd.ProcessState.ExitCode()

        if err != nil {
            member.Status = "failed"
            member.ErrorMessage = fmt.Sprintf("exit code %d", member.ExitCode)
        } else {
            member.Status = "completed"
        }

    case <-time.After(time.Duration(member.TimeoutMs) * time.Millisecond):
        cmd.Process.Kill()
        member.Status = "failed"
        member.ErrorMessage = fmt.Sprintf("timeout after %dms", member.TimeoutMs)
        member.CompletedAt = now()
        member.ExitCode = -1
    }

    // Extract cost from CLI JSON output
    member.CostUSD = extractCostFromCLIOutput(stdout.Bytes())

    // Validate stdout file exists (agent should have written it)
    if err := validateStdout(teamDir, member); err != nil {
        if member.Status == "completed" {
            // Agent exited 0 but didn't write valid stdout — suspicious
            member.Status = "failed"
            member.ErrorMessage = fmt.Sprintf("stdout validation: %v", err)
        }
    }

    // Retry logic (Braintrust KEEP)
    if member.Status == "failed" && member.RetryCount < member.MaxRetries {
        member.RetryCount++
        member.Status = "pending"
        member.PID = nil
        member.ErrorMessage = fmt.Sprintf("retrying (attempt %d/%d): %s",
            member.RetryCount+1, member.MaxRetries+1, member.ErrorMessage)
        // Re-spawn (recursive, but bounded by max_retries)
        spawnAndWait(teamDir, config, member, agentsIndex, wg)
        return
    }

    writeConfigAtomic(teamDir, config)
}
```

### 2.4 Signal Handling

**Source**: Original Design KEEP - Signal handling spec (section 4)

```go
func setupSignalHandlers(children []*exec.Cmd) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

    go func() {
        sig := <-sigCh

        // Forward to all children
        for _, child := range children {
            if child.Process != nil {
                child.Process.Signal(sig)
            }
        }

        // Grace period
        time.Sleep(5 * time.Second)

        // SIGKILL stragglers
        for _, child := range children {
            if child.Process != nil && !child.ProcessState.Exited() {
                child.Process.Kill()
            }
        }

        os.Exit(1)
    }()
}
```

### 2.5 Heartbeat & Orphan Detection

**Source**: Braintrust KEEP - Orphan detection protocol (Einstein 4.1)

```go
func startHeartbeat(teamDir string, interval time.Duration) {
    heartbeatPath := filepath.Join(teamDir, "heartbeat")

    go func() {
        for {
            os.WriteFile(heartbeatPath, []byte(time.Now().Format(time.RFC3339)), 0644)
            time.Sleep(interval)
        }
    }()
}
```

On next TUI session start, `gogent-load-context` (or a new cleanup hook) can check for stale heartbeat files:
- Heartbeat file exists but is older than 60s → team runner crashed
- Read config.json for children PIDs → kill them
- Mark team as `"failed"` with `"orphan cleanup"` message

### 2.6 Cost Extraction from CLI Output

```go
func extractCostFromCLIOutput(output []byte) float64 {
    // Claude CLI --output-format json wraps output in JSON
    // containing a "cost_usd" or similar field
    var cliOutput struct {
        CostUSD float64 `json:"cost_usd"`
        // Other fields we don't need
    }

    if err := json.Unmarshal(output, &cliOutput); err != nil {
        // Defensive: if CLI output format changes, don't crash
        return 0.0
    }

    return cliOutput.CostUSD
}
```

### 2.7 Reading agents-index.json

```go
type AgentConfig struct {
    ID          string `json:"id"`
    Model       string `json:"model"`
    EffortLevel string `json:"effortLevel"`
    // Other fields from agents-index.json
}

type AgentsIndex struct {
    Agents []AgentConfig `json:"agents"`
}

func (idx *AgentsIndex) Lookup(agentType string) *AgentConfig {
    for _, a := range idx.Agents {
        if a.ID == agentType {
            return &a
        }
    }
    return nil // Caller must handle
}
```

This mirrors how `spawnAgent.ts` reads agents-index.json, ensuring both spawn paths use the same agent configuration.

---

## Phase 3: Slash Commands

**Source**: Original Design KEEP - `/team-status` output format (section 8)
**Effort**: 2-3 days
**Blocks**: Phase 4 (orchestrators need to know commands exist for user instructions)

### 3.1 `/team-status`

**Reads**: config.json for each team directory in current session
**Action**: Display wave progress, member statuses, cost

Output format (from Original Design KEEP):

```
ACTIVE TEAMS (1)

braintrust-1738856422
  Trigger: /braintrust
  Phase: executing (background PID 54321)
  Wave 1 [COMPLETE]: einstein (2m14s, $1.82), staff-arch (1m58s, $1.23)
  Wave 2 [RUNNING]: beethoven (running 45s)
  Cost: $3.05 / $15.00 budget

COMPLETED TEAMS (1)

review-1738856800
  Trigger: /review
  Phase: complete (3m22s total)
  Wave 1: backend (31s), frontend (28s), standards (22s), architect (1m05s)
  Cost: $0.42
```

Implementation: Skill definition that reads config.json files from `sessions/{current}/teams/*/config.json`.

### 3.2 `/team-result`

**Reads**: stdout file of final-wave agent
**Action**: Display executive summary and key findings

For braintrust: reads `stdout_beethoven.json` → extracts `content.executive_summary` and `content.unified_recommendations`.

For review: reads all reviewer stdout files → aggregates findings by severity.

### 3.3 `/team-cancel`

**Reads**: config.json for background_pid
**Action**: Sends SIGTERM to `gogent-team-run` process, which cascades to children

```bash
kill -TERM $(jq -r '.background_pid' config.json)
```

### 3.4 `/teams`

**Reads**: Session teams directory
**Action**: Lists all teams with status summary

---

## Phase 4: Launch Sequence & Orchestrator Rewrites

**Source**: Original Design KEEP - Orchestrator backgroundability profiles (section 3.2)
**Effort**: 3-4 days
**Blocks**: Nothing (final phase)

### 4.0 The Launch Sequence (Critical Bridge)

This is the mechanism by which a planner LLM (Mozart) or the Router hands off to
the `gogent-team-run` Go binary. This is the bridge between "schemas are filled"
and "binary is running."

#### Why NOT `Bash({run_in_background: true})`

Claude Code's `run_in_background` parameter creates a tracked background task. This
causes two problems:

1. **Orchestrator-guard conflict**: `gogent-orchestrator-guard` (SubagentStop hook)
   blocks agent completion when background tasks remain uncollected. If Mozart
   launches a background Bash and then tries to return, the guard says
   "you have uncollected background tasks" and blocks it.

2. **Wrong tracking layer**: `run_in_background` returns a Claude `task_id` for
   `TaskOutput()` collection. But we don't WANT Claude to track this — the Go
   binary tracks itself via config.json. Two tracking systems for one process is
   exactly the split-brain problem the braintrust flagged.

#### The Correct Pattern: Process Detachment via `nohup`

The Go binary should be fully detached from Claude's process tree. It manages its
own lifecycle via config.json, heartbeat file, and signal handling.

```bash
# Launch command (used by Mozart or Router via Bash tool)
nohup gogent-team-run {team_dir} > {team_dir}/runner.log 2>&1 &
```

This:
- Returns immediately (Bash call completes in <1s)
- Process is detached from Claude's session (survives TUI exit by design)
- stdout/stderr captured to `runner.log` for debugging
- No Claude task_id created — no orchestrator-guard conflict
- Go binary writes its own PID to config.json on startup

#### Launch Verification

After the `nohup` command, the invoker (Mozart or Router) verifies the binary
actually started by checking config.json:

```
1. Bash: nohup gogent-team-run {team_dir} > {team_dir}/runner.log 2>&1 &
2. Sleep 1s (give binary time to start and write PID)
3. Read: {team_dir}/config.json
4. Check: background_pid is non-null AND orchestrator.phase == "executing"
5. If yes → "Team dispatched. Use /team-status to check progress."
6. If no → Read runner.log for error, report failure to user
```

In pseudocode for the LLM:

```javascript
// Mozart or Router does this
Bash({
  command: `nohup gogent-team-run "${teamDir}" > "${teamDir}/runner.log" 2>&1 & sleep 1 && cat "${teamDir}/config.json" | jq -r '.background_pid // "null"'`
})
// If output is a PID number → success
// If output is "null" → binary failed to start, read runner.log
```

#### Config.json State at Each Handoff Point

```
PLANNER WRITES (before launch):
  orchestrator.phase = "planning"    ← planner is still working
  orchestrator.status = "foreground"
  background_pid = null              ← no binary yet
  members[].status = "pending"       ← nobody spawned yet
  waves[].status = "pending"

GO BINARY STARTUP (first 100ms):
  orchestrator.phase = "executing"   ← binary took over
  orchestrator.status = "background"
  background_pid = 54321             ← binary's own PID
  timing.execution_started_at = now()

GO BINARY WAVE 1 START:
  waves["1"].status = "running"
  waves["1"].started_at = now()
  members[einstein].status = "running"
  members[einstein].pid = 54322
  members[staff-arch].status = "running"
  members[staff-arch].pid = 54323

... (binary continues updating as agents complete) ...

GO BINARY COMPLETE:
  orchestrator.phase = "complete"
  timing.execution_completed_at = now()
  timing.total_duration_ms = calculated
```

#### Runner Log

`{team_dir}/runner.log` captures the Go binary's own stdout/stderr. This is NOT
the agents' output (that goes to stdout files). This is the binary's operational log:

```
2026-02-06T14:30:23Z [INFO] gogent-team-run starting for team braintrust-1738856422
2026-02-06T14:30:23Z [INFO] PID 54321 written to config.json
2026-02-06T14:30:23Z [INFO] Starting wave 1 (2 tasks)
2026-02-06T14:30:23Z [INFO] Spawning einstein (PID 54322, model=opus, effort=high)
2026-02-06T14:30:23Z [INFO] Spawning staff-arch (PID 54323, model=opus, effort=high)
2026-02-06T14:30:23Z [INFO] Heartbeat started (30s interval)
2026-02-06T14:32:37Z [INFO] einstein completed (exit 0, cost $1.82, 134s)
2026-02-06T14:32:41Z [INFO] staff-arch completed (exit 0, cost $1.23, 138s)
2026-02-06T14:32:41Z [INFO] Wave 1 complete. Running inter-wave script...
2026-02-06T14:32:42Z [INFO] gogent-team-prepare-synthesis.sh completed
2026-02-06T14:32:42Z [INFO] Starting wave 2 (1 task)
2026-02-06T14:32:42Z [INFO] Spawning beethoven (PID 54324, model=opus, effort=high)
2026-02-06T14:34:15Z [INFO] beethoven completed (exit 0, cost $1.45, 93s)
2026-02-06T14:34:15Z [INFO] All waves complete. Total cost: $4.50. Duration: 232s
```

`/team-status` can tail this log for detailed progress. `/team-log` (from Phase 3)
reads this file directly.

---

### 4.1 Mozart (/braintrust) — Interview-Then-Background

**Current flow**: Mozart spawns Einstein → waits → spawns Staff-Arch → waits → spawns Beethoven → waits → returns synthesis. TUI frozen ~5 minutes.

**New flow**:

```
1. ROUTER spawns Mozart via Task() (foreground, ~30s)

2. MOZART (foreground LLM, inside Task):
   a. Interview: AskUserQuestion("What's driving this decision?")
   b. Scout: Task(haiku-scout) to assess scope
   c. Plan: Determine team composition

   d. Create team directory:
      Bash({ command: "mkdir -p {session_dir}/teams/{timestamp}.braintrust" })

   e. Write config.json from braintrust template:
      Write({ file_path: "{team_dir}/config.json", content: "..." })

   f. Write stdin files (absolute paths resolved here):
      Write({ file_path: "{team_dir}/stdin_einstein.json", content: "..." })
      Write({ file_path: "{team_dir}/stdin_staff-arch.json", content: "..." })
      Write({ file_path: "{team_dir}/stdin_beethoven.json", content: "..." })

   g. Launch Go binary (detached):
      Bash({
        command: "nohup gogent-team-run \"{team_dir}\" > \"{team_dir}/runner.log\" 2>&1 & sleep 1 && jq -r '.background_pid' \"{team_dir}/config.json\""
      })

   h. Verify PID is non-null in output

   i. Return: "Braintrust team dispatched (einstein + staff-architect → beethoven).
              Use /team-status to check progress."

3. ROUTER receives Mozart's return → passes message to user

4. TUI RETURNS TO USER (interactive, ~30s total)

5. gogent-team-run (detached, PID 54321):
   Wave 1: einstein + staff-arch in parallel
   Inter-wave: gogent-team-prepare-synthesis.sh
   Wave 2: beethoven
   Exit 0 → config.json updated to "complete"

6. USER (anytime):
   /team-status → reads config.json, shows progress
   /team-result → reads stdout_beethoven.json, shows synthesis
   /team-cancel → sends SIGTERM to background_pid from config.json
```

**Key detail**: Mozart uses Write tool (not the Go binary) to create config.json
and stdin files. This happens in the foreground Task() phase. The Go binary only
reads config.json and stdin files — it never needs to know how they were created.

### 4.2 Review-Orchestrator (/review) — Fully Backgroundable

**Current flow**: Review-orchestrator spawns 4 reviewers sequentially → synthesizes. TUI frozen ~4 minutes.

**New flow** (NO foreground LLM needed — Router handles directly):

```
1. ROUTER (foreground, ~5s):
   a. Compute git diff:
      Bash({ command: "git diff HEAD~1 --unified=3" })

   b. Create team directory:
      Bash({ command: "mkdir -p {session_dir}/teams/{timestamp}.review" })

   c. Write config.json from review template
   d. Write stdin files for all 4 reviewers (each gets the diff + their domain focus)
      — All paths resolved to absolute by the Router

   e. Launch Go binary (detached):
      Bash({
        command: "nohup gogent-team-run \"{team_dir}\" > \"{team_dir}/runner.log\" 2>&1 & sleep 1 && jq -r '.background_pid' \"{team_dir}/config.json\""
      })

   f. Verify PID
   g. Return: "Review team dispatched (4 reviewers in parallel).
              Use /team-status to check progress."

2. TUI RETURNS TO USER (near-instant, ~5s total)

3. gogent-team-run (detached):
   Wave 1: all 4 reviewers in parallel
   Exit 0

4. USER: /team-result → aggregated findings by severity
```

**Why no LLM needed**: The Router knows the review template, knows how to run
`git diff`, and can mechanically fill stdin files (each reviewer gets the same
diff with different `review_domain` and `focus_areas`). This is template-fill
work, not reasoning work.

### 4.3 Impl-Manager (/ticket) — Fully Backgroundable

**Current flow**: Impl-manager reads specs.md → spawns workers sequentially.

**New flow**:

```
1. ROUTER (foreground, ~10s):
   a. Read specs.md for task list and dependency graph
   b. Create team directory from implementation template
   c. Build task DAG: identify waves from blocked_by relationships
   d. Write config.json with tasks, waves, members
   e. Write stdin files for each worker agent
      — Each gets their ticket, target files, acceptance criteria
      — Paths resolved to absolute
   f. Launch Go binary (detached)
   g. Return: "Implementation team dispatched ({N} tasks across {M} waves).
              Use /team-status to check progress."

2. gogent-team-run (detached):
   Wave 1: Tasks with no blockers (e.g., auth handler)
   Wave 2: Tasks blocked by wave 1 (e.g., middleware)
   Wave 3: Tasks blocked by wave 2 (e.g., integration tests)
   Wave N: Final review pass
```

### 4.4 Orchestrator-Guard Exemption

The `gogent-orchestrator-guard` (SubagentStop hook) currently blocks completion
when uncollected background tasks exist. Team launches via `nohup` bypass this
entirely because `nohup ... &` inside a regular (non-background) Bash call
returns immediately with exit 0 — Claude doesn't track it as a background task.

However, if this changes or if the guard is enhanced to detect `nohup` patterns,
an explicit exemption should be added to the guard:

```go
// In gogent-orchestrator-guard
// Exempt team launches — they're intentionally detached
if strings.Contains(bashCommand, "gogent-team-run") {
    // This is a fire-and-forget team launch, not a forgotten background task
    return routing.AllowResponse()
}
```

This is a **Phase 2 detail** to validate during testing, not a separate work item.

---

## Quality Gates

**Source**: WEIGHTED-IMPLEMENTATION-PLAN Part X success criteria + Braintrust KEEP contractor readiness checklist

### Phase 0 → Phase 1
- [ ] Spawning an agent, then gracefully exiting TUI → agent terminates
- [ ] No orphan `claude` processes after TUI exit
- [ ] Terminal cursor and alt buffer restored on exit

### Phase 1 → Phase 2
- [ ] Team config schema validated by hand-filling a braintrust team config + stdin files
- [ ] Stdin files contain all information an agent needs (test by reading as a human)
- [ ] Stdout schema is writable by an LLM given the stdin instructions
- [ ] At least one complete set of templates (braintrust) fills correctly

### Phase 2 → Phase 3
- [ ] At least 3 successful `gogent-team-run` executions with real Claude CLI
- [ ] Budget ceiling prevents runaway (test with $0.50 budget)
- [ ] SIGTERM to team runner kills all children within 10 seconds
- [ ] Heartbeat file touched regularly, stale detection works
- [ ] Agent failure → retry once → failure correctly marked in config.json
- [ ] Atomic config.json writes verified (no corruption on kill -9)

### Phase 3 → Phase 4
- [ ] `/team-status` shows accurate wave progress and costs
- [ ] `/team-result` displays Beethoven's synthesis correctly
- [ ] `/team-cancel` gracefully stops running team
- [ ] `/teams` lists all teams in current session

### Phase 4 Complete (MVP)
- [ ] `/braintrust` dispatches Einstein + Staff-Architect in parallel (Wave 1)
- [ ] TUI returns to user within 30 seconds of invoking /braintrust
- [ ] Beethoven receives both outputs and synthesizes (Wave 2)
- [ ] At least 3 successful /braintrust runs with team pattern
- [ ] Cost tracking accurate to within 10% of actual API spend

---

## Risk Mitigations (from KEEP)

| Risk | Source | Mitigation | Implementation Point |
|------|--------|------------|---------------------|
| Go binary crash leaves orphans | Einstein 4.1 | Heartbeat file expiry → pidTracker recovery on next session | Phase 2: heartbeat goroutine |
| Agents don't comply with stdout schema | Staff-Arch F7 | Validate envelope only, fall back to raw. Iterate prompts. | Phase 2: validateStdout() |
| Cost runaway in background | Einstein 4.2 | Budget ceiling in config.json, checked before each spawn | Phase 2: budget gate in loop |
| CLI JSON output format changes | Staff-Arch Layer 2 | Parse defensively with fallbacks. Pin CLI version. | Phase 2: extractCostFromCLIOutput() |
| Config.json corruption on crash | Einstein 3.1 | Atomic writes (write tmp, rename) | Phase 1: writeConfigAtomic() |
| Agent hangs indefinitely | Staff-Arch F2 | Per-member timeout_ms with process kill | Phase 2: select with timeout |
| User forgets about background team | Design §8 | `/team-status` reminder. Future: status bar indicator. | Phase 3: slash command |

---

## Appendix: Failure Mode Catalog

**Source**: Braintrust KEEP - Failure mode catalog (Staff-Arch Layer 3)

| ID | Failure Mode | Probability | Impact | Detection | Recovery | Implemented In |
|----|--------------|-------------|--------|-----------|----------|----------------|
| F1 | Agent exits non-zero | HIGH | MEDIUM | Exit code | Retry once | Phase 2: retry logic |
| F2 | Agent hangs indefinitely | MEDIUM | HIGH | Timeout | Kill process | Phase 2: timeout select |
| F3 | All wave tasks fail | LOW | HIGH | All exit codes | Abort team | Phase 2: allFailed check |
| F4 | Go binary crash | LOW | CRITICAL | Heartbeat stale | Session cleanup | Phase 2: heartbeat |
| F5 | TUI crash/close | MEDIUM | CRITICAL | N/A | Heartbeat + pidTracker | Phase 0 + 2 |
| F6 | Config.json corruption | LOW | HIGH | JSON parse error | Atomic writes prevent | Phase 1: atomic write |
| F7 | Stdout schema violation | HIGH initially | MEDIUM | Envelope validation | Fall back to raw | Phase 2: validateStdout |
| F8 | Cost overrun | MEDIUM | HIGH | Budget tracking | Abort before spawn | Phase 2: budget gate |
| F9 | Disk full | LOW | MEDIUM | Write error | Log and abort | Phase 2: error handling |
| F10 | Hook bypass | NONE (resolved) | N/A | N/A | Hooks run unconditionally | Verified in H2 |

---

## Implementation Summary

| Phase | What | Effort | Key Deliverables |
|-------|------|--------|-----------------|
| **0** | Foundation fixes | 1-2 days | ProcessRegistry cleanup wiring, signal handler fix |
| **1** | Schema design | 2-3 days | Team config schema, stdin/stdout schemas, templates, prompt envelope, inter-wave scripts |
| **2** | Go binary | 5-7 days | `cmd/gogent-team-run/main.go` — wave scheduler with PID management, budget enforcement, heartbeat, signal handling |
| **3** | Slash commands | 2-3 days | `/team-status`, `/team-result`, `/team-cancel`, `/teams` skill definitions |
| **4** | Orchestrator rewrites | 3-4 days | Mozart, review-orchestrator, impl-manager prompt rewrites to use team pattern |
| **TOTAL** | | **13-19 days** | |
