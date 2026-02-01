# GOgent-Fortress Framework - Combined Export

This file contains all framework configuration for deep research review.

---

## Core Configuration

### CLAUDE.md (Router Identity)
```markdown
# Claude Code - GOgent-Fortress Configuration

```
                   ⚡  GOgent Fortress  ⚡
                    ___________________
                   /\  ╔══════════╗  /\
                  /  \ ║ CONTEXT  ║ /  \
              /\ /    \║  VAULT   ║/    \/\
             /  /  /\  ╚══════════╝  /\  \  \
            /  /  /  \______________/  \  \  \
        /\ /  /  / /\ ║█║█║█║█║█║█║ /\ \  \  \/\
       /  /  /  / /  \║█║█║█║█║█║█║/  \ \  \  \  \
     /__/__/__/_/_/__\══════════════/__\_\_\__\__\__\
    |█████████████████|  ROUTER  |█████████████████|
    |█ DISPATCH █ DELEGATE █ VERIFY █ RETURN █████|
    |█████████████████████████████████████████████|

    ⚡ You are a ROUTER, not an implementer ⚡
```

---

## Core Identity

**You are a request ROUTER.** Your job:
1. **Classify** incoming requests
2. **Dispatch** to the appropriate handler
3. **Verify** results meet requirements
4. **Return** to user

**You implement directly ONLY when:**
- Trivial edits (typos, single-line fixes)
- No agent applies to the request
- User explicitly says "do it directly"

---

## Session Init (First Response Only)

**On first response of every session, output:**

```
[Session Init] {language}. {conventions}. Router ready.
```

**Examples:**
- `[Session Init] Go. go.md. Router ready.`
- `[Session Init] Python. python.md. Router ready.`
- `[Session Init] R + Shiny. R.md, R-shiny.md. Router ready.`
- `[Session Init] Home. None. Router ready.`

The `gogent-load-context` hook injects language detection and conventions automatically. This output confirms you received and processed that context.

**Then address the user's request.**

---

## System Facts

| Fact | Value |
|------|-------|
| OS | Arch Linux / CachyOS |
| Python | Externally managed (PEP 668) |
| Python execution | `uv run python` or `~/.generic-python/bin/python` |
| Config location | `~/Documents/GOgent-Fortress/.claude/` |
| Schema version | `routing-schema.json` v2.2.0 |
| Symlink | `~/.claude → ~/Documents/GOgent-Fortress/.claude` |

---

## Active Hooks (ENFORCED)

These Go binaries run automatically. You cannot bypass them.

| Event | Binary | What It Does |
|-------|--------|--------------|
| **SessionStart** | `gogent-load-context` | Detects language, loads conventions, restores handoff, injects git context |
| **PreToolUse (Task)** | `gogent-validate` | Blocks Task(opus), validates subagent_type, checks delegation ceiling, logs violations |
| **PostToolUse** | `gogent-sharp-edge` | Counts tools, reminds routing (every 10), tracks failures, captures sharp edges (3+), logs ML telemetry |
| **SubagentStop** | `gogent-agent-endstate` | Records decision outcomes, logs agent collaborations |
| **SessionEnd** | `gogent-archive` | Generates handoff, archives metrics, captures learnings |

**What hooks enforce:**
- ✅ Task(opus) is BLOCKED → use `/einstein` instead
- ✅ Wrong subagent_type → BLOCKED with corrective message
- ✅ 3+ consecutive failures → Sharp edge captured, execution blocked
- ✅ Every 10 tools → Routing compliance reminder injected

**What hooks DON'T enforce (your responsibility):**
- Choosing the right agent for a request
- Scouting before large tasks
- Post-delegation verification

---

## Routing Decision Flow

```
Request arrives
    │
    ├─► Is it a slash command (/explore, /einstein, etc.)?
    │       YES → Execute the skill
    │
    ├─► Does it match an agent trigger? (see Agent Dispatch Table)
    │       YES → Route to that agent via Task()
    │
    ├─► Is it exploration/research with unknown scope?
    │       YES → Use Task(subagent_type: "Explore")
    │
    ├─► Is it trivial (typo, config tweak, single line)?
    │       YES → Handle directly
    │
    └─► Ambiguous?
            → Ask ONE clarifying question, then route
```

**Output format when routing:**
```
[ROUTING] → agent-name (reason)
```

---

## Slash Commands (Skills)

| Command | What It Does |
|---------|--------------|
| `/explore` | Structured codebase exploration with scout → architect flow |
| `/einstein` | Deep analysis with Opus (bypasses Task blocking) |
| `/review` | Multi-domain code review with severity-grouped findings |
| `/review-plan` | Critical 7-layer review of implementation plans |
| `/ticket` | Ticket-driven implementation workflow |
| `/init-auto` | Initialize project with CLAUDE.md scaffold |
| `/benchmark` | Run gold standard prompts, generate compliance report |
| `/benchmark-meta` | Analyze benchmark trends across commits |
| `/memory-improvement` | Audit system memory, find gaps |
| `/explore-add` | Add custom skill to spawner system |
| `/dummies-guide` | Explain the config system |

---

## Agent Dispatch Table

**Source of truth:** `agents-index.json`

### Tier 1: Haiku (Fast, Cheap)

| Trigger Patterns | Agent | subagent_type |
|------------------|-------|---------------|
| where is, find, which files, grep, locate | `codebase-search` | Explore |
| assess scope, count lines, how big is | `haiku-scout` | Explore |

### Tier 1.5: Haiku + Thinking (Structured Reasoning)

| Trigger Patterns | Agent | subagent_type |
|------------------|-------|---------------|
| scaffold, boilerplate, new class, template | `scaffolder` | general-purpose |
| readme, document, API docs, mermaid, diagram | `tech-docs-writer` | general-purpose |
| review this, code review, spot check | `code-reviewer` | Explore |
| review backend, api review, security review | `backend-reviewer` | Explore |
| review frontend, component review, ui review | `frontend-reviewer` | Explore |
| review standards, code quality, naming review | `standards-reviewer` | Explore |
| how to use, library, best practice, docs | `librarian` | Explore |
| archive session, wrap up, save memory | `memory-archivist` | general-purpose |

### Tier 2: Sonnet (Implementation)

| Trigger Patterns | Agent | subagent_type |
|------------------|-------|---------------|
| Python: implement, refactor, class, test | `python-pro` | general-purpose |
| PySide6, Qt, GUI, widget | `python-ux` | general-purpose |
| Go: implement, struct, test, go build | `go-pro` | general-purpose |
| Cobra, CLI, subcommand, flags | `go-cli` | general-purpose |
| Bubbletea, TUI, lipgloss, tea.Model | `go-tui` | general-purpose |
| HTTP client, API, rate limit, retry | `go-api` | general-purpose |
| Concurrency, goroutine, errgroup, channel | `go-concurrent` | general-purpose |
| R: implement, S4, tidyverse, dplyr | `r-pro` | general-purpose |
| Shiny, reactive, module | `r-shiny-pro` | general-purpose |
| typescript, ts code, type system, generics | `typescript-pro` | general-purpose |
| react, component, hook, useState, ink | `react-pro` | general-purpose |
| code review, full review, review changes | `review-orchestrator` | Plan |
| Ambiguous scope, synthesize, think through | `orchestrator` | Plan |
| Create plan, break down, dependency analysis | `architect` | Plan |
| Review plan, critical review | `staff-architect-critical-review` | Explore |

### Tier 3: Opus (Deep Analysis)

| Trigger | Handler | Notes |
|---------|---------|-------|
| einstein, deep analysis | `/einstein` skill | **NOT via Task()** - blocked by hook |

### External: Gemini

| Trigger Patterns | Handler | Notes |
|------------------|---------|-------|
| full codebase, cross-module, large context | `gemini-slave` | Via Bash, not Task() |

---

## Task() Invocation Pattern

```javascript
Task({
  description: "Brief description",
  subagent_type: "[from dispatch table]",  // ENFORCED by gogent-validate
  model: "haiku" | "sonnet",
  prompt: `AGENT: [agent-id]

TASK: [atomic goal]
CONTEXT: [relevant files, patterns]
EXPECTED OUTPUT: [deliverable]
CONSTRAINTS: [what not to do]`
})
```

**If gogent-validate blocks your Task():**
- Check the error message - it tells you the correct subagent_type
- Fix and retry

---

## Gemini Slave (Special Case)

Uses Bash, NOT Task():

```bash
# Gather files and pipe to gemini-slave
cat file1.go file2.go | gemini-slave mapper "Extract entry points and dependencies"
```

| Protocol | Output | Use When |
|----------|--------|----------|
| `mapper` | JSON structure | Reduce files to critical paths |
| `debugger` | Root cause analysis | Cross-module error tracing |
| `architect` | Patterns/anti-patterns | Module review |
| `scout` | Scope metrics | Pre-routing assessment |

---

## Workflow Patterns

### Pattern 1: Scout → Route → Execute

For unknown scope:
```
1. [SCOUTING] Spawn haiku-scout or gemini-slave scout
2. Read .claude/tmp/scout_metrics.json
3. Route based on recommended_tier
4. Execute via appropriate agent
```

### Pattern 2: Gemini → Orchestrator → Architect

For large-context analysis:
```
1. gemini-slave (Bash) → produces report
2. orchestrator (Task) → synthesizes findings
3. architect (Task) → creates implementation plan
```

### Pattern 3: Einstein Escalation

When orchestrator fails 3x or problem is intractable:
```
1. Generate GAP document to .claude/tmp/einstein-gap-{timestamp}.md
2. Output: "🚨 Run /einstein to process"
3. STOP - wait for user
```

---

## ML Telemetry (Captured Automatically)

gogent-sharp-edge logs every routing decision:

| Data Point | Location |
|------------|----------|
| Routing decisions | `$XDG_DATA_HOME/gogent-fortress/routing-decisions.jsonl` |
| Decision outcomes | `$XDG_DATA_HOME/gogent-fortress/routing-decision-updates.jsonl` |
| Agent collaborations | `$XDG_DATA_HOME/gogent-fortress/agent-collaborations.jsonl` |

**Export for analysis:**
```bash
gogent-ml-export routing-decisions --output=decisions.jsonl
gogent-ml-export stats
```

---

## GOgent Utilities

| Command | Purpose |
|---------|---------|
| `gogent-archive list` | List archived sessions |
| `gogent-archive stats` | Session statistics |
| `gogent-archive sharp-edges` | View captured sharp edges |
| `gogent-aggregate` | Cross-session analysis |
| `gogent-ml-export stats` | ML telemetry summary |

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOGENT_MAX_FAILURES` | 3 | Failures before sharp edge capture |
| `GOGENT_REMINDER_THRESHOLD` | 10 | Tools between routing reminders |
| `GOGENT_FLUSH_THRESHOLD` | 20 | Tools between auto-flush |
| `XDG_DATA_HOME` | `~/.local/share` | ML telemetry location |

---

## Session Lifecycle

### Start (Automatic via gogent-load-context)
- Detects project language
- Loads conventions (`~/.claude/conventions/`)
- Restores handoff from previous session
- Injects git context

### During Session
- Every tool: ML telemetry logged
- Every 10 tools: Routing reminder injected
- Every 20+ tools: Pending learnings auto-flushed
- On failures: Sharp edge tracking

### End (Automatic via gogent-archive)
- Handoff generated to `memory/handoffs.jsonl`
- Human-readable summary to `memory/last-handoff.md`
- Metrics captured

---

## Best Practices (Not Enforced)

### Scout Before Committing Resources

When scope is unknown (mentions "module", "refactor", "all files"):
```
[SCOUTING] Assessing scope before routing...
```

### Verify After Delegation

After agent returns:
```
[Verifying] Checking result meets requirements...
✓ Output received
✓ Deliverable complete
✓ No obvious errors
```

### Compound Triggers → Orchestrator

When 2+ agent triggers fire:
```
[Compound Triggers] synthesis + implementation + documentation
[ROUTING] → orchestrator (multi-domain coordination)
```

---

## Escape Hatches

| Situation | Action |
|-----------|--------|
| User says "do it directly" | Skip routing, execute |
| Single-line fix | Execute directly |
| Hook blocks incorrectly | User can override (rare) |
| No agent fits | Handle as general exploration |
| Urgent/time-sensitive | Note deviation, proceed |

---

## Reference Documents

| Document | Purpose |
|----------|---------|
| `routing-schema.json` | Source of truth for tiers, agents, thresholds |
| `agents-index.json` | Complete agent definitions with triggers |
| `conventions/*.md` | Language-specific coding conventions |
| `rules/LLM-guidelines.md` | Multi-model strategy, anti-patterns |
| `rules/agent-behavior.md` | Behavioral guidelines for all agents |
| `ARCHITECTURE.md` | Full system architecture (in repo root) |

---

## Quick Reference Card

```
ROUTER CHECKLIST:
□ Slash command? → Execute skill
□ Agent trigger? → Route to agent
□ Large scope? → Scout first
□ Exploration? → Task(Explore)
□ Trivial? → Handle directly
□ Ambiguous? → Ask ONE question

BLOCKED BY HOOKS:
✗ Task(model: "opus") → use /einstein
✗ Wrong subagent_type → check dispatch table
✗ 3+ failures → stop, sharp edge captured

OUTPUT FORMATS:
[Session Init] {lang}. {conventions}. Router ready.
[ROUTING] → agent (reason)
[SCOUTING] Assessing scope...
[Verifying] Checking result...
```
```

### routing-schema.json
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "version": "2.4.0",
  "description": "Routing schema for Claude Code tiered agent architecture - v2.4: Added TypeScript/React agents and review specialists",
  "updated": "2026-02-01",
  
  "tiers": {
    "haiku": {
      "description": "Mechanical work - fast, cheap, no reasoning required",
      "model": "haiku",
      "thinking": false,
      "max_thinking_budget": 0,
      "cost_per_1k_tokens": 0.0005,
      "patterns": [
        "count", "list", "find", "search", "glob", "grep", 
        "format", "lint", "locate", "where is", "which files",
        "look for", "find all", "how many"
      ],
      "tools": ["Read", "Glob", "Grep", "Bash"],
      "thresholds": {
        "max_files": 5,
        "max_lines": 500,
        "max_tokens_estimate": 10000
      },
      "agents": ["codebase-search", "haiku-scout"]
    },
    
    "haiku_thinking": {
      "description": "Structured reasoning - Haiku with thinking for templated tasks",
      "model": "haiku",
      "thinking": true,
      "max_thinking_budget": 6000,
      "cost_per_1k_tokens": 0.001,
      "patterns": [
        "scaffold", "boilerplate", "template", "stub",
        "document", "readme", "docstring",
        "review", "check", "spot check", "quick review",
        "research", "library", "best practice", "how to use"
      ],
      "tools": ["Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch"],
      "thresholds": {
        "max_files": 10,
        "max_lines": 1000,
        "max_tokens_estimate": 25000
      },
      "agents": ["scaffolder", "tech-docs-writer", "code-reviewer", "librarian", "memory-archivist", "backend-reviewer", "frontend-reviewer", "standards-reviewer"]
    },
    
    "sonnet": {
      "description": "Reasoning work - implementation, refactoring, debugging",
      "model": "sonnet",
      "thinking": true,
      "max_thinking_budget": 16000,
      "cost_per_1k_tokens": 0.009,
      "patterns": [
        "implement", "refactor", "debug", "test", "fix",
        "create class", "add function", "write test",
        "optimize", "async", "type hints",
        "analyze", "architect", "synthesize", "plan",
        "ambiguous", "cross-module", "design decision",
        "golang", "go build", "cobra", "bubbletea", "tui",
        "goroutine", "errgroup", "http client", "api client"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Glob", "Grep", "Task"],
      "thresholds": {
        "max_files": 20,
        "max_lines": 5000,
        "max_tokens_estimate": 50000
      },
      "agents": ["python-pro", "python-ux", "r-pro", "r-shiny-pro", "orchestrator", "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent", "typescript-pro", "react-pro", "review-orchestrator", "staff-architect-critical-review"]
    },
    
    "opus": {
      "description": "Deep analysis - restricted invocation, allowlist for planning agents",
      "model": "opus",
      "thinking": true,
      "max_thinking_budget": 32000,
      "cost_per_1k_tokens": 0.045,
      "patterns": [
        "einstein", "deep analysis", "security audit",
        "architectural decision", "complex tradeoff",
        "synthesize findings", "cross-domain",
        "novel problem", "nuclear option",
        "plan", "strategy", "comprehensive planning"
      ],
      "tools": ["*"],
      "invocation": "/einstein for standalone, Task() for allowlisted planning agents",
      "task_invocation_blocked": true,
      "task_invocation_allowlist": ["planner", "architect", "staff-architect-critical-review"],
      "escalation_protocol": "escalate_to_einstein",
      "thresholds": {
        "max_files": null,
        "max_lines": null,
        "max_tokens_estimate": null
      },
      "agents": ["einstein", "planner", "architect"]
    },

    "external": {
      "description": "External context engine - Gemini for 1M+ token windows and fast scouting",
      "model": "gemini-2.0-flash",
      "thinking": false,
      "cost_per_1k_tokens": 0.0001,
      "patterns": [
        "large context", "entire codebase", "full module",
        "10+ files", "cross-module trace", "architectural review",
        "map the codebase", "trace through", "assess scope"
      ],
      "tools": ["Bash"],
      "invocation": "cat [files] | gemini-slave [protocol] \"[instruction]\"",
      "protocols": {
        "scout": {"model": "gemini-2.0-flash", "output": "json"},
        "mapper": {"model": "gemini-2.0-pro", "output": "json"},
        "debugger": {"model": "gemini-2.0-pro", "output": "markdown"},
        "architect": {"model": "gemini-2.0-pro", "output": "markdown"},
        "memory-audit": {"model": "gemini-2.0-pro", "output": "json"},
        "benchmark-audit": {"model": "gemini-2.0-pro", "output": "json"}
      },
      "thresholds": {
        "min_files": 10,
        "min_lines": 2000,
        "min_tokens_estimate": 50000
      },
      "agents": ["gemini-slave"]
    }
  },

  "tier_levels": {
    "description": "Numeric levels for tier comparison in delegation ceiling",
    "haiku": 1,
    "haiku_thinking": 2,
    "sonnet": 3,
    "opus": 4,
    "external": 0
  },

  "delegation_ceiling": {
    "description": "Controls which agents can be spawned via Task(), independent of analysis context",
    "file": ".claude/tmp/max_delegation",
    "set_by": "calculate-complexity.sh",
    "enforced_by": "validate-routing.sh",
    "values": ["haiku", "haiku_thinking", "sonnet"],
    "note": "opus never allowed via Task() regardless of ceiling",
    "override": "--force-delegation=<tier>",
    "calculation": {
      "haiku": "Simple queries, no reasoning required",
      "haiku_thinking": "Structured tasks, documentation, review (default)",
      "sonnet": "Implementation, multi-file edits, security-sensitive"
    }
  },

  "scout_protocol": {
    "description": "Pre-routing reconnaissance to assess scope before committing resources",
    "primary": "gemini-slave scout",
    "fallback": "haiku-scout",
    "selection_logic": {
      "haiku_scout": {
        "max_files": 3,
        "max_tokens": 5000,
        "reason": "Lower latency for very small scopes, preserves context hygiene for anything larger"
      },
      "gemini_scout": {
        "min_files": 4,
        "min_tokens": 5001,
        "reason": "Aggressive offloading: massive context window, higher rate limits, lower cost, zero context pollution"
      }
    },
    "cost_per_call_estimate": 0.001,
    "invocation": "find [target] -type f \\( -name \"*.py\" -o -name \"*.md\" \\) | gemini-slave scout \"[task]\"",
    "output_schema": {
      "scope_metrics": ["total_files", "total_lines", "estimated_tokens"],
      "complexity_signals": ["import_density", "cross_file_dependencies"],
      "routing_recommendation": ["recommended_tier", "confidence", "clarification_needed"]
    },
    "when_to_use": [
      "Unknown scope (refactor X, improve Y)",
      "Task mentions modules or systems",
      "Could involve 5+ files"
    ],
    "when_to_skip": [
      "User specified exact files",
      "Single file operation",
      "Trivial task (typo, config change)"
    ]
  },
  
  "escalation_rules": {
    "haiku_to_haiku_thinking": [
      "reasoning required but simple domain",
      "structured output needed",
      "template-following task"
    ],
    "haiku_to_sonnet": [
      "multi-file edit required",
      "complex reasoning needed",
      "error after 2 attempts",
      "implementation task detected"
    ],
    "sonnet_to_opus": {
      "triggers": [
        "architectural decision required",
        "security concern detected",
        "3+ file dependencies with complex interaction",
        "3 consecutive failures on same task",
        "user explicitly requests deep analysis"
      ],
      "action": "DO NOT use Task(opus). Generate GAP document instead.",
      "protocol": "escalate_to_einstein",
      "output_path": ".claude/tmp/einstein-gap-{timestamp}.md",
      "notification": "🚨 Run /einstein to process this escalation"
    },
    "any_to_external": [
      "context exceeds 50K tokens",
      "10+ files need simultaneous analysis",
      "cross-module debugging required",
      "full codebase understanding needed"
    ]
  },
  
  "compound_triggers": {
    "description": "When 2+ patterns from different tiers fire, escalate to orchestrator",
    "examples": [
      ["large context", "synthesize"],
      ["implement", "architectural review"],
      ["debug", "cross-module"]
    ],
    "action": "Route to orchestrator for coordination"
  },
  
  "cost_thresholds": {
    "scout_max_cost": 0.01,
    "exploration_max_cost": 0.20,
    "description": "Cost ceilings for pre-execution phases"
  },
  
  "override": {
    "flag": "--force-tier=<tier>",
    "description": "User escape hatch when routing produces false positives",
    "valid_tiers": ["haiku", "haiku_thinking", "sonnet", "opus", "external"],
    "audit_log": "/tmp/claude-routing-violations.jsonl"
  },

  "subagent_types": {
    "description": "Defines tool capabilities for each subagent_type used in Task invocations",
    "Explore": {
      "description": "Read-only codebase exploration and analysis",
      "tools": ["Read", "Glob", "Grep", "Bash"],
      "allows_write": false,
      "respects_agent_yaml": false,
      "use_for": ["codebase-search", "haiku-scout", "code-reviewer", "librarian", "backend-reviewer", "frontend-reviewer", "standards-reviewer"],
      "rationale": "Exploration tasks should be non-destructive reconnaissance"
    },
    "general-purpose": {
      "description": "Full tool access following agent.yaml definitions",
      "tools": ["*"],
      "allows_write": true,
      "respects_agent_yaml": true,
      "use_for": ["scaffolder", "tech-docs-writer", "python-pro", "python-ux", "r-pro", "r-shiny-pro", "memory-archivist", "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent", "typescript-pro", "react-pro"],
      "rationale": "Implementation and documentation agents need write permissions"
    },
    "Bash": {
      "description": "Command execution specialist for external processes",
      "tools": ["Bash", "Read"],
      "allows_write": false,
      "respects_agent_yaml": false,
      "use_for": ["gemini-slave"],
      "rationale": "External context engines use shell piping, not file modification"
    },
    "Plan": {
      "description": "Architecture and planning mode with controlled write access",
      "tools": ["Read", "Glob", "Grep", "Write", "Task", "AskUserQuestion"],
      "allows_write": true,
      "respects_agent_yaml": false,
      "use_for": ["orchestrator", "architect", "planner", "review-orchestrator"],
      "rationale": "Planning agents need to write plans and spawn other agents"
    }
  },

  "delegation_rules": {
    "description": "Task() is always allowed regardless of session tier. It's delegation, not direct work.",
    "task_always_allowed": true,
    "tier_restrictions_apply_to": [
      "Read", "Write", "Edit", "Bash", "Glob", "Grep", "WebFetch", "WebSearch"
    ],
    "tier_restrictions_do_not_apply_to": [
      "Task"
    ],
    "rationale": "Session tier indicates what MODEL to use for direct analysis. It does not restrict which agents can be spawned. Spawned agents operate at their own tier."
  },

  "agent_subagent_mapping": {
    "description": "Maps each agent to its required subagent_type (enforced by validate-routing.sh)",
    "codebase-search": "Explore",
    "haiku-scout": "Explore",
    "code-reviewer": "Explore",
    "librarian": "Explore",
    "tech-docs-writer": "general-purpose",
    "scaffolder": "general-purpose",
    "memory-archivist": "general-purpose",
    "python-pro": "general-purpose",
    "python-ux": "general-purpose",
    "r-pro": "general-purpose",
    "r-shiny-pro": "general-purpose",
    "go-pro": "general-purpose",
    "go-cli": "general-purpose",
    "go-tui": "general-purpose",
    "go-api": "general-purpose",
    "go-concurrent": "general-purpose",
    "typescript-pro": "general-purpose",
    "react-pro": "general-purpose",
    "backend-reviewer": "Explore",
    "frontend-reviewer": "Explore",
    "standards-reviewer": "Explore",
    "review-orchestrator": "Plan",
    "orchestrator": "Plan",
    "architect": "Plan",
    "planner": "Plan",
    "einstein": "general-purpose",
    "gemini-slave": "Bash",
    "staff-architect-critical-review": "Plan"
  },

  "blocked_patterns": {
    "description": "Patterns that should NEVER be used",
    "patterns": [
      {
        "pattern": "Task.*model.*opus",
        "reason": "Opus invocation via Task tool causes 60K+ token inheritance",
        "alternative": "Use escalate_to_einstein protocol → /einstein slash command",
        "cost_impact": "$2.38 wasted per call (72% overhead)"
      }
    ]
  },

  "direct_impl_check": {
    "description": "Detects when router writes implementation code directly instead of delegating",
    "enabled": true,
    "write_threshold_lines": 50,
    "edit_threshold_lines": 30,
    "implementation_extensions": [".go", ".py", ".r", ".ts", ".js"],
    "implementation_paths": ["internal/", "pkg/", "cmd/", "lib/", "src/"],
    "excluded_patterns": ["*_test.go", "testdata/*", "*.gen.go", "*_generated.go"]
  },

  "meta_rules": {
    "documentation_theater": {
      "description": "Enforcement belongs in hooks, not documentation",
      "detection_patterns": [
        "MUST NOT",
        "NEVER use",
        "is BLOCKED",
        "SHALL NOT",
        "ALWAYS .* instead",
        "FORBIDDEN",
        "PROHIBITED"
      ],
      "target_files": ["**/CLAUDE.md", "**/agent.md"],
      "enforcement": "detect-documentation-theater.sh",
      "guidance": "~/.claude/rules/LLM-guidelines.md §Enforcement Architecture"
    }
  }
}
```

## Agents Index
```json
{
  "version": "2.3.0",
  "generated_at": "2026-02-01T00:00:00Z",
  "description": "Agent index for Intent Gate routing and auto-activation. v2.3: Added TypeScript/React agents and code review specialists.",
  "agents": [
    {
      "id": "memory-archivist",
      "parallelization_template": "E",
      "name": "Memory Archivist",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 4000,
      "tier": 1.5,
      "category": "memory",
      "path": "memory-archivist",
      "triggers": [
        "task complete",
        "end session",
        "wrap up",
        "archive session",
        "capture learnings",
        "save memory",
        "archive specs",
        "save plan"
      ],
      "tools": ["Read", "Write", "Glob", "save_memory"],
      "auto_activate": null,
      "inputs": [".claude/tmp/specs.md", ".claude/memory/pending-learnings.jsonl"],
      "outputs": [".claude/memory/decisions/", ".claude/memory/sharp-edges/"],
      "description": "The Historian. Archives specs.md and session learnings to structured memory for RAG queries."
    },
    {
      "id": "codebase-search",
      "parallelization_template": "A",
      "name": "Codebase Search",
      "model": "haiku",
      "thinking": false,
      "tier": 1,
      "category": "task",
      "path": "codebase-search",
      "triggers": [
        "where is",
        "find the",
        "which files",
        "locate",
        "search for",
        "find all",
        "grep for",
        "look for"
      ],
      "tools": ["Glob", "Grep", "Read"],
      "auto_activate": null,
      "description": "Fast file and code discovery. Haiku-tier for mechanical extraction."
    },
    {
      "id": "librarian",
      "parallelization_template": "A",
      "name": "Librarian",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 4000,
      "tier": 1.5,
      "category": "task",
      "path": "librarian",
      "triggers": [
        "how do I use",
        "best practice",
        "library",
        "documentation",
        "example of",
        "what's the",
        "official docs",
        "how does this library"
      ],
      "tools": ["WebFetch", "WebSearch", "Bash", "Read", "Grep"],
      "auto_activate": null,
      "auto_fire": ["External library mentioned", "Unfamiliar package"],
      "sharp_edges_count": 4,
      "description": "External research with GitHub permalinks and documentation fetching."
    },
    {
      "id": "scaffolder",
      "parallelization_template": "B",
      "name": "Scaffolder",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 4000,
      "tier": 1.5,
      "category": "task",
      "path": "scaffolder",
      "triggers": [
        "create skeleton",
        "scaffold",
        "boilerplate",
        "new class",
        "new module",
        "new test file",
        "generate template",
        "stub out"
      ],
      "tools": ["Read", "Write", "Glob"],
      "auto_activate": null,
      "description": "Generate boilerplate code following conventions. Haiku-tier for fast scaffolding."
    },
    {
      "id": "tech-docs-writer",
      "parallelization_template": "C",
      "name": "Tech Docs Writer",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 6000,
      "tier": 1.5,
      "category": "documentation",
      "path": "tech-docs-writer",
      "triggers": [
        "write readme",
        "document",
        "API docs",
        "add docstrings",
        "create documentation",
        "update docs",
        "architecture guide",
        "visual guide",
        "mermaid",
        "infographic",
        "diagram"
      ],
      "tools": ["Read", "Write", "Edit", "Glob", "Grep", "Bash"],
      "auto_activate": null,
      "description": "Technical documentation specialist. README, API docs, architecture guides."
    },
    {
      "id": "code-reviewer",
      "parallelization_template": "C",
      "name": "Code Reviewer",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 6000,
      "tier": 1.5,
      "category": "review",
      "path": "code-reviewer",
      "triggers": [
        "review this",
        "check this code",
        "any issues with",
        "code review",
        "quick review",
        "spot check"
      ],
      "tools": ["Read", "Glob", "Grep"],
      "auto_activate": null,
      "description": "Routine code review for style, simple bugs, and obvious improvements."
    },
    {
      "id": "python-pro",
      "parallelization_template": "D",
      "name": "Python Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "python-pro",
      "triggers": [
        "implement",
        "refactor",
        "optimize",
        "create class",
        "add function",
        "write test",
        "async",
        "type hints",
        "python"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "languages": ["Python"]
      },
      "conventions_required": ["python.md"],
      "sharp_edges_count": 6,
      "description": "Expert Python development with modern patterns. Auto-activated for Python projects."
    },
    {
      "id": "python-ux",
      "parallelization_template": "D",
      "name": "Python UX (PySide6)",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "python-ux",
      "triggers": [
        "create widget",
        "add dialog",
        "signal slot",
        "QML",
        "model view",
        "Qt",
        "PySide",
        "PySide6",
        "GUI",
        "desktop app",
        "QThread"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["PySide6", "PyQt", "QWidget", "QMainWindow", "Qt"]
      },
      "conventions_required": ["python.md"],
      "sharp_edges_count": 7,
      "description": "PySide6/Qt6 GUI development expert. Desktop application UI specialist."
    },
    {
      "id": "r-pro",
      "parallelization_template": "D",
      "name": "R Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "r-pro",
      "triggers": [
        "implement",
        "refactor",
        "S4 class",
        "R6",
        "vectorize",
        "parallel",
        "test",
        "tidyverse",
        "bioconductor",
        "dplyr",
        "ggplot"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "languages": ["R"]
      },
      "conventions_required": ["R.md"],
      "sharp_edges_count": 6,
      "description": "Expert R development with S4 OOP, tidyverse, and bioinformatics patterns."
    },
    {
      "id": "r-shiny-pro",
      "parallelization_template": "D",
      "name": "R Shiny Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "r-shiny-pro",
      "triggers": [
        "create module",
        "reactive",
        "observe",
        "render",
        "shinyFiles",
        "state management",
        "bookmark",
        "shiny",
        "module"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "languages": ["R+Shiny"],
        "patterns": ["shiny", "app.R", "mod_", "reactive", "shinyFiles"]
      },
      "conventions_required": ["R.md", "R-shiny.md"],
      "sharp_edges_count": 6,
      "description": "Shiny application expert with module architecture and R6/S4 state management."
    },
    {
      "id": "go-pro",
      "parallelization_template": "D",
      "name": "GO Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "go-pro",
      "triggers": [
        "implement",
        "refactor",
        "optimize",
        "create struct",
        "add function",
        "write test",
        "golang",
        "go code",
        "go module",
        "go build"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "languages": ["Go"],
        "file_patterns": ["go.mod", "*.go"]
      },
      "conventions_required": ["go.md"],
      "sharp_edges_count": 8,
      "description": "Expert GO development. Auto-activated for GO projects. Single-binary desktop distribution focus."
    },
    {
      "id": "go-cli",
      "parallelization_template": "D",
      "name": "GO CLI (Cobra)",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "go-cli",
      "triggers": [
        "cli command",
        "cobra",
        "subcommand",
        "command line",
        "flags",
        "shell completion",
        "viper config",
        "add command",
        "cli application"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["**/cmd/**/main.go", "**/cli/**/*.go"]
      },
      "conventions_required": ["go.md", "go-cobra.md"],
      "sharp_edges_count": 5,
      "description": "Cobra CLI specialist. Factory pattern, Viper config, shell completion."
    },
    {
      "id": "go-tui",
      "parallelization_template": "D",
      "name": "GO TUI (Bubbletea)",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 12000,
      "tier": 2,
      "category": "language",
      "path": "go-tui",
      "triggers": [
        "tui",
        "bubbletea",
        "terminal ui",
        "lipgloss",
        "bubbles",
        "tea.Model",
        "dashboard",
        "status display"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["**/tui/**/*.go", "**/ui/**/*.go"]
      },
      "conventions_required": ["go.md", "go-bubbletea.md"],
      "sharp_edges_count": 6,
      "description": "Bubbletea TUI specialist. MVU architecture, Lipgloss styling, component composition."
    },
    {
      "id": "go-api",
      "parallelization_template": "D",
      "name": "GO API (HTTP Client)",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "go-api",
      "triggers": [
        "http client",
        "api client",
        "api integration",
        "rate limit",
        "retry logic",
        "backoff",
        "sse streaming",
        "llm api",
        "rest client"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["**/api/**/*.go", "**/client/**/*.go"]
      },
      "conventions_required": ["go.md"],
      "sharp_edges_count": 5,
      "description": "HTTP client specialist. Timeouts, retries, rate limiting, SSE streaming."
    },
    {
      "id": "go-concurrent",
      "parallelization_template": "D",
      "name": "GO Concurrent",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 12000,
      "tier": 2,
      "category": "language",
      "path": "go-concurrent",
      "triggers": [
        "concurrency",
        "goroutine",
        "worker pool",
        "errgroup",
        "semaphore",
        "parallel",
        "channel",
        "fan-out",
        "fan-in",
        "race condition"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["**/worker/**/*.go", "**/pool/**/*.go"]
      },
      "conventions_required": ["go.md"],
      "sharp_edges_count": 7,
      "description": "Concurrency specialist. Worker pools, errgroup, semaphores, channels, graceful shutdown."
    },
    {
      "id": "orchestrator",
      "parallelization_template": "E",
      "name": "Orchestrator",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 16000,
      "tier": 2,
      "category": "architecture",
      "path": "orchestrator",
      "triggers": [
        "ambiguous scope",
        "cross-module planning",
        "user interview",
        "design decision",
        "debugging loop",
        "think through",
        "analyze",
        "architect",
        "synthesize",
        "synthesis",
        "review findings",
        "triage",
        "interpret results"
      ],
      "tools": ["Read", "Glob", "Grep", "Task", "Bash"],
      "auto_activate": null,
      "scout_first": true,
      "description": "Handles ambiguous scope, cross-module planning, user interviews, design tradeoffs, and debugging loops. Uses Scout-First Protocol."
    },
    {
      "id": "architect",
      "parallelization_template": "E",
      "name": "Architect",
      "model": "opus",
      "thinking": true,
      "thinking_budget": 32000,
      "thinking_budget_complex": 48000,
      "tier": 3,
      "category": "architecture",
      "path": "architect",
      "triggers": [
        "create a plan",
        "implementation plan",
        "break this down",
        "what order should",
        "dependency analysis",
        "architectural review",
        "refactor strategy",
        "dependency map",
        "from scout report",
        "from strategy"
      ],
      "tools": ["Read", "Write", "Glob", "Grep"],
      "auto_activate": null,
      "output_artifacts": {
        "required": ["specs.md", "write_todos"],
        "specs_location": ".claude/tmp/specs.md"
      },
      "input_sources": [".claude/tmp/scout_metrics.json", ".claude/tmp/strategy.md"],
      "description": "Implementation planner (opus tier). Produces specs.md + write_todos. Reads scout reports and strategy documents for context."
    },
    {
      "id": "planner",
      "parallelization_template": "E",
      "name": "Planner",
      "model": "opus",
      "thinking": true,
      "thinking_budget": 32000,
      "tier": 3,
      "category": "architecture",
      "path": "planner",
      "triggers": [
        "plan",
        "strategy",
        "requirements",
        "what should we build",
        "how should we approach",
        "strategic planning",
        "comprehensive plan"
      ],
      "tools": ["Read", "Glob", "Grep", "Write", "AskUserQuestion"],
      "auto_activate": null,
      "output_artifacts": {
        "required": ["strategy.md"],
        "strategy_location": ".claude/tmp/strategy.md"
      },
      "input_sources": [".claude/tmp/scout_metrics.json"],
      "max_clarifying_questions": 2,
      "description": "Strategic planner (opus tier). Transforms goals into requirements, risks, and high-level approach. Produces strategy.md for architect."
    },
    {
      "id": "einstein",
      "parallelization_template": "F",
      "name": "Einstein",
      "model": "opus",
      "thinking": true,
      "thinking_budget": 32000,
      "tier": 3,
      "category": "analysis",
      "path": "einstein",
      "triggers": [
        "einstein",
        "call einstein",
        "orchestrator fails 3x same task",
        "deep analysis"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
      "auto_activate": null,
      "description": "Last-resort escalation for issues unresolvable by orchestrator. Receives full diagnostic context."
    },
    {
      "id": "gemini-slave",
      "parallelization_template": null,
      "name": "Gemini Slave",
      "model": "external",
      "tier": "external",
      "category": "context",
      "path": "gemini-slave",
      "triggers": [
        "analyze entire",
        "trace through",
        "full codebase",
        "all files in",
        "deep dive",
        "root cause across",
        "architectural review",
        "map the codebase",
        "multiple files",
        "entire module",
        "large context",
        "cross-module",
        "assess scope",
        "scout"
      ],
      "tools": ["Bash"],
      "auto_activate": null,
      "invocation": "Bash (gemini-slave wrapper), NOT Task tool",
      "protocols": ["mapper", "debugger", "architect", "benchmark-audit", "memory-audit", "scout"],
      "state_files": {
        "scout_output": ".claude/tmp/scout_metrics.json",
        "complexity_score": ".claude/tmp/complexity_score"
      },
      "description": "Large-context analysis subagent with 1M+ token window. Includes scout protocol for pre-routing reconnaissance."
    },
    {
      "id": "staff-architect-critical-review",
      "parallelization_template": "C",
      "name": "Staff Architect Critical Review",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 16000,
      "tier": 2,
      "category": "review",
      "path": "staff-architect-critical-review",
      "triggers": [
        "review plan",
        "critical review",
        "review implementation plan",
        "validate specs"
      ],
      "tools": ["Read", "Glob", "Grep", "Task", "Write"],
      "auto_activate": null,
      "inputs": [".claude/tmp/specs.md"],
      "outputs": [".claude/tmp/review-critique.md", ".claude/tmp/review-metadata.json"],
      "sharp_edges_count": 6,
      "cost_per_invocation": "0.15-0.25",
      "description": "Critical review of implementation plans using 7-layer framework. Invoked via /review-plan command or automatically in /plan workflow."
    },
    {
      "id": "haiku-scout",
      "parallelization_template": "A",
      "name": "Haiku Scout",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 2000,
      "tier": 1,
      "category": "reconnaissance",
      "path": "haiku-scout",
      "triggers": [
        "assess scope",
        "count lines",
        "estimate complexity",
        "pre-route",
        "scout",
        "how big is",
        "how many files"
      ],
      "tools": ["Read", "Glob", "Grep", "Bash"],
      "auto_activate": null,
      "parallel_safe": true,
      "swarm_compatible": true,
      "output_format": "json",
      "output_file": ".claude/tmp/scout_metrics.json",
      "cost_ceiling_usd": 0.02,
      "fallback_for": "gemini-slave scout",
      "description": "Fallback scout when Gemini unavailable. Writes to .claude/tmp/scout_metrics.json for complexity calculation."
    },
    {
      "id": "typescript-pro",
      "parallelization_template": "D",
      "name": "TypeScript Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "typescript-pro",
      "triggers": [
        "typescript",
        "ts code",
        "type system",
        "implement",
        "refactor",
        "strict types",
        "generics",
        "interface",
        "enum",
        "type guard"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "languages": ["TypeScript"],
        "file_patterns": ["*.ts", "*.tsx"]
      },
      "conventions_required": ["typescript.md"],
      "sharp_edges_count": 10,
      "description": "Expert TypeScript development with strict types and modern patterns. Auto-activated for TypeScript projects."
    },
    {
      "id": "react-pro",
      "parallelization_template": "D",
      "name": "React Pro",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 10000,
      "tier": 2,
      "category": "language",
      "path": "react-pro",
      "triggers": [
        "react",
        "component",
        "hook",
        "useState",
        "useEffect",
        "ink",
        "tui component",
        "zustand",
        "context",
        "memo",
        "callback"
      ],
      "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
      "auto_activate": {
        "patterns": ["*.tsx", "use*.ts"]
      },
      "conventions_required": ["typescript.md", "react.md"],
      "sharp_edges_count": 11,
      "description": "React specialist with hooks, Ink TUI components, and state management. Auto-activated for React patterns."
    },
    {
      "id": "backend-reviewer",
      "parallelization_template": "C",
      "name": "Backend Reviewer",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 6000,
      "tier": 1.5,
      "category": "review",
      "path": "backend-reviewer",
      "triggers": [
        "review backend",
        "api review",
        "backend patterns",
        "data layer review",
        "review api"
      ],
      "tools": ["Read", "Glob", "Grep"],
      "auto_activate": null,
      "conventions_required": ["go.md", "python.md", "R.md", "typescript.md"],
      "sharp_edges_count": 10,
      "description": "Backend code review specialist. API design, data patterns, error handling, concurrency."
    },
    {
      "id": "frontend-reviewer",
      "parallelization_template": "C",
      "name": "Frontend Reviewer",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 6000,
      "tier": 1.5,
      "category": "review",
      "path": "frontend-reviewer",
      "triggers": [
        "review frontend",
        "component review",
        "ui review",
        "react review",
        "ink review"
      ],
      "tools": ["Read", "Glob", "Grep"],
      "auto_activate": null,
      "conventions_required": ["typescript.md", "react.md"],
      "sharp_edges_count": 10,
      "description": "Frontend code review specialist. Component architecture, state management, hooks, performance."
    },
    {
      "id": "standards-reviewer",
      "parallelization_template": "C",
      "name": "Standards Reviewer",
      "model": "haiku",
      "thinking": true,
      "thinking_budget": 6000,
      "tier": 1.5,
      "category": "review",
      "path": "standards-reviewer",
      "triggers": [
        "review standards",
        "code quality",
        "naming review",
        "style review",
        "convention review"
      ],
      "tools": ["Read", "Glob", "Grep"],
      "auto_activate": null,
      "sharp_edges_count": 10,
      "description": "Code standards review specialist. Naming, organization, documentation, consistency."
    },
    {
      "id": "review-orchestrator",
      "parallelization_template": "E",
      "name": "Review Orchestrator",
      "model": "sonnet",
      "thinking": true,
      "thinking_budget": 12000,
      "tier": 2,
      "category": "review",
      "path": "review-orchestrator",
      "triggers": [
        "code review",
        "review changes",
        "full review",
        "/review",
        "orchestrate review"
      ],
      "tools": ["Read", "Glob", "Grep", "Task", "Write"],
      "auto_activate": null,
      "sharp_edges_count": 0,
      "description": "Coordinates multi-domain code review. Spawns specialized reviewers, synthesizes findings."
    }
  ],
  "routing_rules": {
    "intent_gate": {
      "description": "Pre-classify every message before routing",
      "types": [
        {"type": "trivial", "signal": "Single file, known location", "action": "Direct tools only"},
        {"type": "explicit", "signal": "Specific file/line, clear command", "action": "Execute directly"},
        {"type": "exploratory", "signal": "How does X work?", "action": "Fire codebase-search + librarian"},
        {"type": "open-ended", "signal": "Improve, Refactor", "action": "Scout first, then route"},
        {"type": "ambiguous", "signal": "Unclear scope", "action": "Scout first, clarify if needed"}
      ]
    },
    "scout_first_protocol": {
      "description": "Spawn scout before committing expensive resources",
      "triggers": [
        "unknown scope",
        "mentions module/system/architecture",
        "refactor/implement/add feature",
        "could involve 5+ files"
      ],
      "skip_when": [
        "user specified exact files",
        "single file operation",
        "trivial task (typo, config)"
      ],
      "primary": "gemini-slave scout",
      "fallback": "haiku-scout",
      "output": ".claude/tmp/scout_metrics.json"
    },
    "complexity_routing": {
      "description": "Code-enforced tier selection based on complexity score",
      "calculator": ".claude/scripts/calculate-complexity.sh",
      "thresholds": {
        "haiku": {"max_score": 2},
        "sonnet": {"min_score": 2, "max_score": 10},
        "external": {"min_score": 10}
      },
      "force_external_if": "tokens > 50000"
    },
    "auto_fire": {
      "external_library_mentioned": "librarian",
      "multiple_modules_involved": "codebase-search",
      "implementation_python": "python-pro",
      "implementation_r": "r-pro",
      "implementation_shiny": "r-shiny-pro",
      "implementation_go": "go-pro",
      "implementation_go_cli": "go-cli",
      "implementation_go_tui": "go-tui",
      "pyside6_detected": "python-ux",
      "high_context_detected": "gemini-slave"
    },
    "model_tiers": {
      "haiku": ["codebase-search", "scaffolder", "librarian", "tech-docs-writer", "code-reviewer", "haiku-scout", "memory-archivist"],
      "haiku_thinking": ["backend-reviewer", "frontend-reviewer", "standards-reviewer"],
      "sonnet": ["python-pro", "python-ux", "r-pro", "r-shiny-pro", "orchestrator", "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent", "typescript-pro", "react-pro", "review-orchestrator"],
      "opus": ["einstein", "planner", "architect", "staff-architect-critical-review"],
      "external": ["gemini-slave"]
    }
  },
  "state_management": {
    "description": "File-based state passing between agents",
    "tmp_directory": ".claude/tmp/",
    "files": {
      "scout_metrics.json": {
        "written_by": ["gemini-slave scout", "haiku-scout"],
        "read_by": ["calculate-complexity.sh", "architect"],
        "ttl_minutes": 60
      },
      "complexity_score": {
        "written_by": ["calculate-complexity.sh"],
        "read_by": ["validate-routing.sh"],
        "ttl_minutes": 60
      },
      "recommended_tier": {
        "written_by": ["calculate-complexity.sh"],
        "read_by": ["validate-routing.sh"],
        "ttl_minutes": 60
      },
      "specs.md": {
        "written_by": ["architect"],
        "read_by": ["memory-archivist"],
        "ttl_minutes": null,
        "archived_to": ".claude/memory/decisions/"
      }
    },
    "cleanup": {
      "trigger": "memory-archivist or session end",
      "action": "delete files older than ttl"
    }
  }
}
```

## Rules
### agent-behavior.md
```markdown
---
paths:
  - "**/*"
alwaysApply: true
---

# Agent Behavioral Guidelines

**Audience:** This document instructs Claude (the agent) on optimal behavior patterns.
**Scope:** All sessions, all project types.

---

## 1. Coding Discipline

These principles apply to all coding work, before routing considerations.

### 1.1 Surface Assumptions Before Committing

Before implementing non-trivial code:

**State what you're assuming.** If the request could reasonably be interpreted
multiple ways, name them. Don't silently pick one and hope it's right.

**Push back when warranted.** If a simpler approach exists, say so. If the
requested approach has significant tradeoffs, surface them. Technical honesty
serves the user better than compliance.

**Stop when genuinely confused.** Name what's unclear. One targeted question
beats three attempts that miss the mark.

This doesn't mean interrogating the user on every detail. Use judgment:
- Obvious intent → proceed
- Ambiguous with reasonable default → state assumption, proceed
- Ambiguous with no clear default → ask

### 1.2 Solve the Actual Problem

Write code that addresses what was asked. Resist the pull toward:

- **Speculative features** - functionality nobody requested
- **Premature abstraction** - generalizing one-off code "for later"
- **Defensive overkill** - error handling for scenarios that can't occur

That said, use judgment. Sometimes a small abstraction genuinely clarifies.
Sometimes adjacent error handling prevents real bugs. The test isn't "was
this explicitly requested?" but "does this serve the user's actual goal?"

**The checks:**

1. If you wrote 200 lines and it could be 50, rewrite it.

2. Ask yourself: "Would a senior engineer say this is overcomplicated?"
   If yes, simplify.

3. If you added code beyond what was asked, ask why. "Might be useful
   someday" → reconsider. "Genuinely makes this clearer or more correct"
   → proceed.

### 1.3 Parallelize Independent Operations

When executing a task, identify operations that don't depend on each other
and execute them in a single message with multiple tool calls.

**How to parallelize:** Include multiple tool invocations in the same response.
The runtime executes them concurrently and returns all results together.

**Parallelize (no dependencies between calls):**
```javascript
// GOOD: Single message, multiple independent reads
Read({file_path: "/src/auth/handler.go"})
Read({file_path: "/src/auth/middleware.go"})
Read({file_path: "/src/auth/types.go"})
// All three execute concurrently, results return together
```

**Don't parallelize (output informs next input):**
```javascript
// These MUST be sequential - each depends on the previous
Read({file_path: "/src/config.go"})        // Need content first
// ...wait for result...
Edit({file_path: "/src/config.go", ...})   // Edit depends on read
// ...wait for result...
Bash({command: "go build ./..."})          // Build depends on edit
```

**Common parallelizable patterns:**

| Scenario | Parallel Calls |
|----------|----------------|
| Understanding a module | Read 3-5 related files |
| Finding patterns | Multiple Grep/Glob searches |
| Validating changes | Bash(go vet) + Bash(go test) + Bash(golangci-lint) |
| Exploring structure | Glob for *.go + Glob for *_test.go + Read go.mod |

**The check:** Before making a tool call, ask: "Do I need the result of a
previous call to determine this call's parameters?" If no → batch it.

---

## 2. Routing Discipline

### 2.1 Always Check Before Acting

Before using ANY tool, verify:
1. Does this task match a Key Trigger in CLAUDE.md?
2. Is the current tier appropriate for this work?
3. Should I scout first to assess scope?

**Reference:** `~/.claude/routing-schema.json` is the source of truth for tier thresholds.

### 2.2 Tier Selection Matrix

| Task Complexity | Context Size | Tier | Action |
|-----------------|--------------|------|--------|
| Mechanical (count, find, grep) | Any | Haiku | Direct or codebase-search |
| Structured output (docs, scaffold) | <1000 lines | Haiku+Thinking | Delegate to specialist |
| Reasoning required | <5000 tokens | Sonnet | Delegate to implementation agent |
| Multi-source synthesis | Any | Sonnet (orchestrator) | Coordinate multiple agents |
| Novel/complex/security | Any | Opus (einstein) | **Generate GAP doc** (see 4.4) |
| >10 files or >50K tokens | Large | External (Gemini) | Pipe to gemini-slave first |

### 2.2.1 Go Implementation Agents (Sonnet Tier)

| Trigger Patterns | Agent | Use For |
|------------------|-------|---------|
| implement, struct, interface, go build | `go-pro` | Core Go implementation |
| Cobra, CLI, subcommand, flags | `go-cli` | CLI applications |
| Bubbletea, TUI, lipgloss, tea.Model | `go-tui` | Terminal interfaces |
| HTTP client, API, rate limit, retry | `go-api` | HTTP clients/servers |
| goroutine, errgroup, channel, mutex | `go-concurrent` | Concurrent patterns |

These agents understand Go idioms: explicit error handling, small interfaces, composition over inheritance, table-driven tests.

### 2.3 Scout Before Commit

**When scope is unknown:**
```
[SCOUTING] Unknown scope detected. Spawning haiku-scout...
```

Then wait for scout results before selecting tier. This prevents $0.50 Opus calls on $0.02 Haiku work.

---

## 3. Parallel Agent Management

### 3.1 Background vs Foreground

| Pattern | Use When | Mechanism |
|---------|----------|-----------|
| **Foreground (default)** | Next step depends on this output | `Task({...})` |
| **Background** | Independent work, will collect later | `Bash({..., run_in_background: true})` |
| **Parallel foreground** | Multiple independent, need all before continuing | Multiple `Task()` in same message |

### 3.2 MANDATORY: Background Task Collection

**Enforcement:** `gogent-orchestrator-guard` (SubagentStop hook) blocks orchestrator completion when background tasks remain uncollected.

**If you spawn background tasks, you MUST:**

1. Track every task_id returned
2. Before ANY final output or synthesis:
   ```javascript
   TaskOutput({task_id: "bg-task-1", block: true})
   TaskOutput({task_id: "bg-task-2", block: true})
   ```
3. NEVER conclude orchestration with uncollected background tasks

**Violation Pattern (BLOCKED by hook):**
```javascript
Bash({..., run_in_background: true})  // Spawned
Bash({..., run_in_background: true})  // Spawned
// ... do other work ...
// Output synthesis WITHOUT calling TaskOutput → BLOCKED by gogent-orchestrator-guard
```

### 3.3 Fan-Out, Fan-In Pattern

For parallel information gathering:

```javascript
// 1. FAN-OUT: Spawn all tasks
const task1 = Task({...})  // Returns task_id
const task2 = Task({...})  // Returns task_id  
const task3 = Task({...})  // Returns task_id

// 2. FAN-IN: Collect all results (MANDATORY)
const result1 = TaskOutput({task_id: task1, block: true})
const result2 = TaskOutput({task_id: task2, block: true})
const result3 = TaskOutput({task_id: task3, block: true})

// 3. SYNTHESIZE: Only after all collected
// Now proceed with synthesis
```

---

## 4. Failure Handling

### 4.1 Automatic Escalation Triggers

| Condition | Action |
|-----------|--------|
| 2 failures on same file | Warning injected (via hook) |
| 3 failures on same file | Sharp edge captured, escalation prompted |
| Agent returns error | Retry with modified approach ONCE |
| Retry also fails | Escalate to next tier |

### 4.2 Retry with Modification

When an approach fails, do NOT retry identically. Modify:
- Different tool selection
- Smaller scope
- More context provided
- Different agent

**Bad:**
```
[Attempt 1] Edit file X → Error
[Attempt 2] Edit file X → Same error  // WRONG: identical retry
```

**Good:**
```
[Attempt 1] Edit file X → Error
[Analysis] Error suggests type mismatch
[Attempt 2] Read file X first, then Edit with correct types
```

### 4.3 Sharp Edge Protocol

When a debugging loop is detected:
1. STOP current approach
2. Document the pattern (auto-logged by hook)
3. Consider: What assumption was wrong?
4. Either:
   - Fix assumption and retry differently, OR
   - Escalate to higher tier with context

### 4.4 Escalate to Einstein Protocol

**Enforcement:** `gogent-validate` (Go binary, PreToolUse hook) **blocks** `Task(model: "opus")` calls. Must use `/einstein` slash command.

#### Trigger Conditions

Escalate to Einstein when:

| Condition | Detection |
|-----------|-----------|
| **3+ consecutive failures** | Same file/function, same error class |
| **Architectural decision required** | Solution requires cross-module tradeoffs |
| **Complexity exceeds Sonnet tier** | Scout returns `recommended_tier: opus` |
| **User explicitly requests** | "call einstein", "deep analysis needed" |
| **Novel problem** | No pattern in sharp-edges.yaml applies |

#### Escalation Procedure

1. **STOP** current execution
2. **Generate GAP document** using template at `~/.claude/schemas/einstein-gap.md`
3. **Write** to `.claude/tmp/einstein-gap-{timestamp}.md`
4. **Output notification**:
   ```
   [ESCALATED] GAP document ready: .claude/tmp/einstein-gap-{timestamp}.md

   🚨 Run `/einstein` to process this escalation.

   Summary:
   - Problem: {brief_problem}
   - Attempts: {attempt_count}
   - Blocker: {primary_blocker}
   ```
5. **WAIT** for user to run `/einstein`

#### What NOT to Do

- ❌ **DO NOT** invoke Einstein via Task tool (hook will block it)
- ❌ **DO NOT** continue attempting after 3 failures (you're looping)
- ❌ **DO NOT** generate incomplete GAP documents (garbage in = garbage out)
- ❌ **DO NOT** escalate trivial problems (use code-reviewer for sanity checks first)

#### GAP Document Quality Checklist

Before writing the GAP document, verify:

- [ ] Problem statement is specific (not "it doesn't work")
- [ ] All attempts are logged with actual error messages
- [ ] Relevant file excerpts are included (not just paths)
- [ ] Constraints are explicit (not assumed)
- [ ] Question is answerable from provided context
- [ ] Anti-scope prevents scope creep

**Reference:** See `orchestrator/agent.md` for complete GAP generation code example.

---

## 5. Hook Awareness

### 5.1 Active Hooks

The following Go binaries run as hooks and inject context automatically:

| Binary | Event | What You'll See |
|--------|-------|-----------------|
| `gogent-load-context` | SessionStart | Routing schema, previous handoff, git context |
| `gogent-validate` | PreToolUse (Task) | Block/allow decision, subagent_type enforcement |
| `gogent-sharp-edge` | PostToolUse | Tool counter, routing reminders (every 10), failure tracking |
| `gogent-agent-endstate` | SubagentStop | Decision outcomes, tier-specific follow-up prompts |
| `gogent-orchestrator-guard` | SubagentStop | Background task collection enforcement |
| `gogent-archive` | SessionEnd | Handoff generation, metrics capture |

### 5.2 Responding to Hook Injections

When you see `additionalContext` in a hook response:
- READ the injected guidance
- FOLLOW the recommendations
- Do NOT ignore or dismiss

---

## 6. Memory & Learning

### 6.1 Knowledge Compounding

When you discover something worth remembering:

1. **Sharp Edges** (errors, gotchas): Auto-captured by hook → review at session end
2. **Decisions** (architectural choices): Document in `memory/decisions/`
3. **Patterns** (successful approaches): Propose addition to conventions

### 6.2 Session Handoff

At session end:
- Pending learnings are archived automatically
- Handoff document generated at `memory/last-handoff.md`
- Next session receives this context via `gogent-load-context` hook

### 6.3 Evolution Cycle

```
Work → Detect patterns → Capture to memory → 
Weekly audit (Gemini) → Propose config updates → 
Benchmark test → If improved: commit
```

---

## 7. Cost Optimization

### 7.1 Token Budget Awareness

| Tier | Thinking Budget | Cost/1K tokens |
|------|-----------------|----------------|
| Haiku | 0 or 2-4K | $0.0005 |
| Haiku+Thinking | 4-6K | $0.001 |
| Sonnet | 10-16K | $0.009 |
| Opus | 16-32K | $0.045 |

### 7.2 Cost-Saving Patterns

1. **Scout before expensive work**: $0.02 scout can prevent $0.50 mis-routing
2. **Haiku for mechanical**: Never use Sonnet for grep/find/count
3. **Gemini for large context**: Cheaper than Sonnet for >50K tokens
4. **Batch similar operations**: One agent call with multiple files > multiple calls

### 7.3 Delegation Overhead Threshold

If task is <$0.01 of work, do it directly rather than delegating. Delegation itself costs tokens.

---

## 8. Output Quality

### 8.1 Self-Verification

Before returning output to user:
1. Does it answer the actual question?
2. Does it follow relevant conventions?
3. Are there obvious errors?
4. Would a quick code-reviewer pass help?

### 8.2 Critic Pattern (Optional)

For important outputs, invoke quick review:
```javascript
Task({
  model: "haiku",
  prompt: "Review this output for obvious errors: [output]"
})
```

Cost: ~$0.005. Worth it for user-facing deliverables.

---

## 9. Anti-Patterns

### 9.1 FORBIDDEN Behaviors

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|------------------|
| Retrying identically after failure | Wastes tokens, won't help | Modify approach |
| Using Sonnet for file search | 50x cost waste | Use Haiku/codebase-search |
| Spawning background tasks without collecting | Orphaned work | Always call TaskOutput |
| Ignoring hook injections | Misses guidance | Read and follow |
| Skipping scout on unknown scope | Potential mis-routing | Scout first |
| Large context without Gemini | Context overflow | Pipe to gemini-slave |

### 9.2 WARNING Behaviors

| Behavior | Risk | Mitigation |
|----------|------|------------|
| >3 agents in one task | Coordination complexity | Consider orchestrator |
| Opus for routine work | Cost | Verify Opus triggers present |
| Direct file editing without reading | Context gaps | Read first |

---

## 10. Checklist: Before Completing Task

- [ ] All background tasks collected?
- [ ] Routing tier was appropriate?
- [ ] No obvious errors in output?
- [ ] Sharp edges documented if any?
- [ ] Conventions followed?
- [ ] User's actual question answered?

---

**Remember:** Your effectiveness is bounded by coding discipline and routing discipline. Overcomplicated code or wrong tier = wasted effort + suboptimal output.
```

### LLM-guidelines.md
```markdown
---
paths:
  - "**/*"
---

# Guidelines for Maximizing Claude's Effectiveness

This document defines best practices for leveraging Claude models across all coding tasks. These patterns apply regardless of language (Python, R) or domain (ML, cloud, data science).

---

## Core Principles

### Context is Everything
Claude's effectiveness scales with context quality. Provide:
- **Complete type/class definitions** before asking for methods
- **Representative data samples** (structure, not full datasets)
- **Full error tracebacks** when debugging
- **Related code** that new code must integrate with
- **Constraints** (performance, memory, compatibility)

### Explicit Over Implicit
Never assume Claude knows your project conventions:
- Reference specific rule files: "Follow the conventions in R.md"
- State expected return types explicitly
- Specify edge cases that must be handled
- Declare performance requirements upfront

### Iterative Refinement Over One-Shot
Complex tasks benefit from staged approaches:
1. Skeleton/interface first
2. Implementation second
3. Tests third
4. Optimization fourth

---

## Task Specification Patterns

### The Complete Specification Pattern
For non-trivial functions, provide:
```
TASK: [What to build]
INPUTS: [Types, shapes, constraints]
OUTPUTS: [Expected return type/structure]
EDGE CASES: [What to handle]
INTEGRATES WITH: [Existing code/classes]
CONSTRAINTS: [Performance, memory, compatibility]
EXAMPLE: [Representative input → expected output]
```

### The Debugging Pattern
When asking Claude to debug:
```
OBSERVED: [What happened - include full error]
EXPECTED: [What should happen]
CONTEXT: [Relevant code, recent changes]
ATTEMPTED: [What you already tried]
```

### The Review Pattern
When requesting code review:
```
REVIEW AGAINST: [Specific rules/criteria]
FOCUS AREAS: [Performance, security, style, etc.]
CONSTRAINTS: [What cannot change]
```

---

## Leveraging Claude Code Features

### Plan Mode for Architecture
**MUST** use plan mode (`EnterPlanMode`) for:
- New feature implementations with multiple valid approaches
- Architectural decisions (state management, data flow)
- Multi-file refactoring
- Any task where user preference matters

Plan mode allows exploration before commitment.

### Parallel Agents for Research
Use the `Task` tool with multiple agents when:
- Researching best practices across sources
- Running multiple specialized code reviews
- Exploring different implementation approaches
- Gathering context from multiple codebases

```
Example: "Launch parallel agents to research Bubbletea tea.Model patterns
and lipgloss styling conventions, then synthesize recommendations"
```

### Todo Tracking for Complex Tasks
For multi-step implementations:
- Break down into discrete, verifiable steps
- Track progress explicitly
- Mark completions as you go
- Add discovered sub-tasks dynamically

---

## Verification and Self-Review

### Request Self-Review
After Claude generates code, ask:
- "Review this against the [language].md conventions"
- "Identify edge cases this doesn't handle"
- "What are the performance characteristics?"
- "Generate test cases for this function"

### Staged Verification
1. **Correctness**: "Does this logic handle [specific case]?"
2. **Style**: "Does this follow our naming conventions?"
3. **Performance**: "What's the complexity? Any bottlenecks?"
4. **Security**: "Any injection risks or data leaks?"
5. **Tests**: "Generate testthat/pytest cases for edge conditions"

### The Rubber Duck Pattern
Ask Claude to explain complex generated code:
- Forces verification of logic
- Surfaces implicit assumptions
- Identifies documentation gaps

---

## Domain-Specific Patterns

### Machine Learning (PyTorch/TensorFlow)

**Model Architecture Specification:**
```
ARCHITECTURE: [Model type - CNN, Transformer, etc.]
INPUT SHAPE: [Batch, channels, height, width] or [Batch, seq_len, features]
OUTPUT SHAPE: [Expected output dimensions]
LAYERS: [Key layer specifications]
LOSS: [Loss function and any weighting]
DEVICE: [CPU, CUDA, TPU considerations]
```

**Training Loop Requests:**
- Specify batch size, gradient accumulation steps
- State mixed precision requirements (fp16/bf16)
- Define checkpointing strategy
- Specify distributed training needs (DDP, FSDP)

**Data Pipeline Requests:**
- Input data format and source
- Augmentation requirements
- Preprocessing steps
- Memory constraints (streaming vs. in-memory)

**Debugging ML Code:**
- Always include tensor shapes at failure point
- Provide device placement info
- Include gradient flow concerns
- State memory usage observations

### Cloud Storage (GCS)

**Authentication Context:**
- Service account vs. user credentials
- Environment (local dev, Cloud Run, GKE, Vertex AI)
- Required IAM permissions

**Data Transfer Patterns:**
- Batch size for parallel operations
- Resumable upload requirements
- Streaming vs. download-then-process
- Cost considerations (egress, operations)

**Path Handling:**
- Always use `gs://bucket/path` URI format
- Clarify blob vs. prefix operations
- State whether recursive operations needed

### Large Data Processing

**Memory Constraints:**
- State available memory
- Specify chunking requirements
- Indicate streaming needs
- Define acceptable memory/speed tradeoffs

**Parallelization Context:**
- Number of available cores/workers
- I/O vs. CPU bound nature
- Shared state requirements
- Progress tracking needs

### Go Development (Primary Language)

**Code Structure Specification:**
```
PACKAGE: [package name and responsibility]
IMPORTS: [expected dependencies]
TYPES: [structs, interfaces to define]
FUNCTIONS: [public API with signatures]
ERROR HANDLING: [error types, wrapping strategy]
TESTS: [table-driven test expectations]
```

**Interface Design:**
- Keep interfaces small (1-3 methods)
- Define interfaces at the point of use, not implementation
- Accept interfaces, return concrete types
- Example: `io.Reader` over custom `DataSource`

**Error Handling Patterns:**
- Always handle errors explicitly (no `_` for errors)
- Wrap errors with context: `fmt.Errorf("loading config: %w", err)`
- Define sentinel errors for expected conditions
- Use error types for errors that need inspection

**Concurrency Specification:**
```
PATTERN: [fan-out/fan-in, worker pool, pipeline]
GOROUTINES: [number and responsibility]
SYNCHRONIZATION: [channels, mutex, errgroup]
CANCELLATION: [context.Context usage]
ERROR PROPAGATION: [how errors surface]
```

**Common Go Agents:**
| Agent | Use For | Key Patterns |
|-------|---------|--------------|
| `go-pro` | Core implementation | Idiomatic Go, error handling |
| `go-cli` | CLI apps (Cobra) | Flags, subcommands, help text |
| `go-tui` | TUI apps (Bubbletea) | tea.Model, tea.Cmd, lipgloss |
| `go-api` | HTTP clients/servers | Context, retry, rate limiting |
| `go-concurrent` | Concurrency | errgroup, channels, context |

**Testing Patterns:**
- Table-driven tests for multiple cases
- Subtests with `t.Run()` for organization
- Test helpers with `t.Helper()` marking
- `testdata/` directory for fixtures
- Example: `TestParseConfig/valid_json`, `TestParseConfig/missing_field`

**GOgent-Fortress Specific:**
- All hook binaries read JSON from STDIN with 5s timeout
- Output JSON to STDOUT for Claude Code consumption
- Use `pkg/routing` for schema validation
- Use `pkg/session` for handoff operations
- JSONL files are append-only (never rewrite)

---

## Anti-Patterns to Avoid

### Vague Specifications
| Bad | Good |
|-----|------|
| "Make it faster" | "Reduce complexity from O(n²) to O(n log n)" |
| "Handle errors" | "Catch ConnectionError, retry 3x with exponential backoff" |
| "Add logging" | "Log at INFO level with structured fields: user_id, operation, duration" |
| "Make it work with big data" | "Must handle 10M rows with <8GB RAM using chunked processing" |

### Missing Context
| Bad | Good |
|-----|------|
| "Fix this function" | "Fix this function [full code]. Error: [full traceback]. Expected: [behavior]" |
| "Add a method to MyClass" | "Add a method to MyClass [class definition]. Must integrate with [related code]" |
| "Optimize the model" | "Optimize: current 2.3s/batch on V100, target <1s. Bottleneck appears in attention layer" |

### Skipping Verification
| Bad | Good |
|-----|------|
| Accept first output | "Review this against our style guide before I use it" |
| Assume edge cases handled | "What edge cases does this not handle?" |
| Trust performance claims | "Profile this and identify actual bottlenecks" |

---

## Prompting Strategies by Task Type

### New Feature Implementation
1. Start with plan mode for architecture
2. Define interfaces/contracts first
3. Implement core logic
4. Add error handling
5. Generate tests
6. Self-review against rules

### Bug Fixing
1. Provide full error context
2. Share relevant code sections
3. State what you've tried
4. Ask for root cause analysis
5. Request fix with explanation
6. Ask for regression test

### Code Review
1. Specify review criteria explicitly
2. Request structured feedback (severity, location, suggestion)
3. Ask for specific rule violations
4. Request improvement alternatives

### Refactoring
1. State refactoring goals (performance, readability, testability)
2. Define what must not change (API, behavior)
3. Request incremental changes
4. Ask for verification strategy

### Performance Optimization
1. Provide profiling data
2. State performance targets
3. Specify acceptable tradeoffs
4. Request complexity analysis
5. Ask for benchmarking approach

---

## Context Window Optimization

### What to Include
- Full class/type definitions for code being modified
- Representative sample data (structure, not volume)
- Error messages with complete tracebacks
- Related functions that must integrate
- Relevant configuration/constants

### What to Summarize
- Large datasets → representative samples + schema
- Long files → relevant sections + structure overview
- History → key decisions and constraints

### What to Reference
- Rule files by name: "per go.md conventions"
- Previous conversation context: "as discussed above"
- External docs: use WebFetch tool or MCP-provided fetch tools

---

## Multi-Model Strategy

### CRITICAL: Tiered Model Routing

Use `Task(model: "haiku")` or `Task(model: "sonnet")` to delegate work to cheaper models. Only keep quality-critical tasks in Opus.

### When to Use Different Models

| Task Type | Model | Rationale |
|-----------|-------|-----------|
| **OPUS (Quality Critical)** | | |
| Interview/requirements gathering | Opus | Quality of questions determines outcome |
| Planning/architecture | Opus | Complex tradeoffs need depth |
| Cross-domain synthesis | Opus | Connecting 5+ sources needs reasoning |
| Conflict judgment | Opus | Requires nuanced assessment |
| MCP API calls | Opus | Direct API, delegation overhead exceeds savings |
| **SONNET (Reasoning, Familiar)** | | |
| Go implementation (go-pro, go-tui, go-cli) | Sonnet | Needs reasoning, follows Go idioms |
| Code understanding | Sonnet | Needs reasoning but standard patterns |
| Core implementation | Sonnet | Following established patterns |
| Single-domain analysis | Sonnet | Focused analysis, not cross-cutting |
| Documentation generation | Sonnet | Structured output with reasoning |
| Concurrency design (go-concurrent) | Sonnet | Channel patterns, error propagation |
| **HAIKU (Mechanical Work)** | | |
| File discovery (glob, find, ls) | Haiku | Pure file operations |
| Pattern extraction (grep, regex) | Haiku | Mechanical matching |
| Keyword extraction | Haiku | Text parsing |
| Result formatting | Haiku | Structured output, no reasoning |
| Skill/index loading | Haiku | File reading |
| Boilerplate generation | Haiku | Template following |
| Sharp edge detection | Haiku | Pattern matching against known list |
| Code review (style only) | Haiku | Convention checking, no design judgment |

### Routing Enforcement

When using `/explore` or similar workflows:

1. **ALWAYS announce routing** with `[ROUTING] → Model (reason)`
2. **Use Task tool** with explicit `model: "haiku"` or `model: "sonnet"`
3. **Never use Glob/Grep/Read directly** for exploration - spawn Haiku scouts
4. **Stay in Opus** only for interview, planning, synthesis, and judgment

### Cost Impact

Aggressive tiered routing saves ~70% on exploration workflows:
- Haiku: ~$0.0005/1k tokens (50x cheaper than Opus)
- Sonnet: ~$0.009/1k tokens (5x cheaper than Opus)
- Opus: ~$0.045/1k tokens (baseline)

### Parallel Agent Patterns

For complex research tasks, launch multiple Haiku scouts in parallel:
```
- Haiku Scout 1: File discovery (glob patterns)
- Haiku Scout 2: Pattern extraction (grep)
- Haiku Scout 3: Code snippet extraction
→ Sonnet Analyst: Synthesize findings
→ Opus Main: Make architectural decisions
```

For Go implementation tasks, consider:
```
- Haiku Scout: Find existing patterns (grep for similar interfaces)
- go-pro (Sonnet): Implement core logic
- code-reviewer (Haiku): Verify conventions
```

---

## Effective Feedback Loops

### When Output Isn't Right
Instead of: "That's wrong, try again"
Use: "The output [specific issue]. The constraint is [constraint]. Consider [hint]"

### Building on Previous Output
Instead of: Starting fresh
Use: "Keep [what worked], but modify [specific part] to [desired change]"

### Incremental Complexity
Instead of: Full implementation request
Use:
1. "Create the interface/skeleton"
2. "Implement the core logic"
3. "Add error handling for [cases]"
4. "Optimize [specific bottleneck]"

---

## Checklist: Before Asking Claude

- [ ] Have I provided complete type/class definitions?
- [ ] Have I included representative examples?
- [ ] Have I stated constraints explicitly?
- [ ] Have I referenced relevant rule files?
- [ ] Have I specified what "done" looks like?
- [ ] Am I using plan mode for complex tasks?
- [ ] Should I break this into smaller requests?

---

## Enforcement Architecture

### The Anti-Pattern: Documentation Theater

**Definition:** Adding imperative enforcement language ("MUST NOT", "NEVER", "BLOCKED") to CLAUDE.md or other documentation files, creating the illusion of enforcement without any actual mechanism.

**Why it fails:**
- Text instructions are probabilistic suggestions, not deterministic rules
- Attention to early instructions degrades over long conversations
- No mechanism exists to actually BLOCK a tool call via text
- Creates false confidence that behavioral problems are "solved"
- CLAUDE.md becomes bloated with unenforceable imperatives

### The Correct Pattern: Declarative → Programmatic → Reference

**Three components, in order:**

1. **Declarative Rule** (`routing-schema.json`)
   - Single source of truth for what's allowed/blocked
   - Parsed by hooks at runtime
   - Example: `"task_invocation_blocked": true`

2. **Programmatic Enforcement** (Go hook binary, e.g., `gogent-validate`)
   - Actually runs before/after tool use
   - Can block, warn, or modify behavior
   - Example: Check schema rule, return `routing.BlockResponse()` with reason

3. **Reference Documentation** (`CLAUDE.md`)
   - Points to enforcement, doesn't replace it
   - Example: "Blocked by gogent-validate (PreToolUse hook)"
   - Provides context for WHY, not enforcement of WHAT

### Decision Tree: Where Does This Go?

```
Is this enforcement of a behavior?
│
├─ YES: Can it be detected programmatically?
│   │
│   ├─ YES: What kind of enforcement?
│   │   │
│   │   ├─ Block action → routing-schema.json rule
│   │   │                 + gogent-validate check (Go binary)
│   │   │                 + CLAUDE.md reference
│   │   │
│   │   ├─ Require action → Hook injects reminder at trigger
│   │   │                   + CLAUDE.md documents workflow
│   │   │
│   │   └─ Warn on pattern → PreToolUse hook with warning
│   │                        + CLAUDE.md notes the check
│   │
│   └─ NO: Is it methodology guidance?
│       │
│       ├─ YES → LLM-guidelines.md (this file)
│       │
│       └─ NO → agent-behavior.md or conventions/*.md
│
└─ NO: Is this describing existing system behavior?
    │
    ├─ YES → CLAUDE.md (gates, workflows, triggers)
    │
    └─ NO → Probably doesn't need to be written
```

### What Goes Where: Quick Reference

| Need | ❌ Wrong | ✅ Right |
|------|----------|----------|
| Block a tool pattern | "You MUST NOT use X" in CLAUDE.md | `routing-schema.json` rule + `gogent-validate` enforcement + CLAUDE.md reference |
| Require pre-check | "ALWAYS check Y first" in CLAUDE.md | Hook injects reminder at trigger point |
| Prevent anti-pattern | "NEVER do Z" in CLAUDE.md | This section in LLM-guidelines.md + warning hook |
| Document workflow | Gates 1-5 in CLAUDE.md | ✅ Appropriate (this IS documentation) |
| Agent-specific rule | In CLAUDE.md | `agents/*/sharp-edges.yaml` or `agent.yaml` |

### Pre-Commit Checklist for CLAUDE.md Edits

Before adding enforcement-style language to CLAUDE.md:

- [ ] Is this DESCRIPTION of existing behavior, or ENFORCEMENT of new behavior?
- [ ] If enforcement: Is it implemented in a hook FIRST?
- [ ] Does CLAUDE.md text REFERENCE the hook (file + line), not REPLACE it?
- [ ] Are there any new "MUST", "NEVER", "BLOCKED" without corresponding code?
- [ ] Would this still work if the LLM ignores this paragraph?

If any answer is wrong, implement programmatic enforcement first.

### What CLAUDE.md IS For

✅ **Appropriate content:**
- Gates (workflow checkpoints with structure)
- Trigger tables (pattern → agent mapping)
- System constraints (Arch Linux, Python paths)
- References ("See hook X for enforcement")
- Context loading (conventions, skills)

❌ **Inappropriate content:**
- Behavioral blocking ("MUST NOT use X")
- Imperative requirements without enforcement
- Rules that depend on LLM "remembering"
- Anything that fails silently when ignored

### Example: Correct vs Incorrect

**Scenario:** Need to prevent Task(opus) invocations

❌ **Incorrect (documentation theater):**
```markdown
## Gate 6: Einstein Protection

**You MUST NOT invoke Einstein via Task tool.**
**This is BLOCKED. Use /einstein slash command instead.**
```

✅ **Correct (layered enforcement):**

1. `routing-schema.json`:
```json
"opus": {
  "task_invocation_blocked": true,
  "blocked_reason": "60K+ token inheritance overhead"
}
```

2. `cmd/gogent-validate/main.go`:
```go
if event.Task != nil && event.Task.Model == "opus" {
    return routing.BlockResponse(
        "Task(opus) blocked by gogent-validate. Use /einstein instead.",
    )
}
```

3. `CLAUDE.md`:
```markdown
## Gate 6: Einstein Escalation

Einstein invocation via Task tool is blocked by `gogent-validate` (PreToolUse hook).
See `routing-schema.json` → `opus.task_invocation_blocked`.

When Einstein triggers fire, use `escalate_to_einstein` protocol instead.
Reference: `~/.claude/skills/einstein/SKILL.md`
```

The CLAUDE.md version describes and references; it doesn't pretend to enforce.

---

**Remember:** Claude's output quality is bounded by input quality. Invest in context.
```

## Conventions
### go-bubbletea.md
```markdown
# GO Bubbletea TUI Conventions - Lisan al-Gaib

## Overview

Bubbletea implements The Elm Architecture (TEA/MVU) for terminal UIs. These conventions ensure professional TUIs with proper state management, component composition, and styling.

## The Elm Architecture

### Core Principles

1. **Model**: All application state in a single struct
2. **View**: Pure function that renders state to string (no side effects)
3. **Update**: Pure function that handles messages and returns new state
4. **Commands**: The ONLY way to perform I/O

### Basic Structure

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// Model holds ALL application state
type Model struct {
    width    int
    height   int
    items    []string
    cursor   int
    selected map[int]struct{}
    loading  bool
    err      error
}

// Init returns initial commands to run
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        fetchItems,      // Load initial data
        tea.EnterAltScreen, // Use alternate screen buffer
    )
}

// Update handles messages and returns new model + commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
        
    case tea.KeyMsg:
        return m.handleKey(msg)
        
    case itemsLoadedMsg:
        m.items = msg.items
        m.loading = false
        return m, nil
        
    case errMsg:
        m.err = msg.err
        m.loading = false
        return m, nil
    }
    
    return m, nil
}

// View renders the UI - MUST be fast, NO I/O
func (m Model) View() string {
    if m.loading {
        return "Loading..."
    }
    if m.err != nil {
        return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
    }
    return m.renderList()
}
```

## Messages and Commands

### Custom Message Types

```go
// Define message types for your data flows
type itemsLoadedMsg struct {
    items []string
}

type errMsg struct {
    err error
}

type statusUpdateMsg string

type tickMsg time.Time
```

### Commands (The ONLY Way to Do I/O)

```go
// Commands are functions that return a Msg
func fetchItems() tea.Msg {
    items, err := api.FetchItems()
    if err != nil {
        return errMsg{err}
    }
    return itemsLoadedMsg{items}
}

// Returning commands from Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            // Return command to process selected item
            return m, processItem(m.items[m.cursor])
        case "r":
            // Return command to refresh
            m.loading = true
            return m, fetchItems
        }
    }
    return m, nil
}

// Command factory
func processItem(item string) tea.Cmd {
    return func() tea.Msg {
        result, err := api.Process(item)
        if err != nil {
            return errMsg{err}
        }
        return processCompleteMsg{result}
    }
}
```

### NEVER Modify State in Goroutines

```go
// WRONG: Race condition, undefined behavior
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        go func() {
            data := fetchData()
            m.data = data  // BUG: Modifying model outside Update!
        }()
    }
    return m, nil
}

// CORRECT: Return command, let Bubbletea manage goroutine
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        return m, fetchDataCmd
    }
    return m, nil
}

func fetchDataCmd() tea.Msg {
    data := fetchData()
    return dataLoadedMsg{data}
}
```

## Keyboard Handling

### Pattern for Key Messages

```go
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Global keys (work in any mode)
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    }
    
    // Mode-specific keys
    switch m.mode {
    case modeNormal:
        return m.handleNormalKey(msg)
    case modeInput:
        return m.handleInputKey(msg)
    case modeHelp:
        return m.handleHelpKey(msg)
    }
    
    return m, nil
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
    case "down", "j":
        if m.cursor < len(m.items)-1 {
            m.cursor++
        }
    case "enter":
        return m, processSelected(m.items[m.cursor])
    case "/":
        m.mode = modeInput
        m.input = ""
    case "?":
        m.mode = modeHelp
    }
    return m, nil
}
```

### Special Key Sequences

```go
switch msg.Type {
case tea.KeyCtrlC:
    return m, tea.Quit
case tea.KeyEsc:
    m.mode = modeNormal
case tea.KeyEnter:
    return m.submitInput()
case tea.KeyBackspace:
    if len(m.input) > 0 {
        m.input = m.input[:len(m.input)-1]
    }
case tea.KeyRunes:
    m.input += string(msg.Runes)
}
```

## Component Composition

### Parent-Child Pattern

```go
// Child component
type ListComponent struct {
    items    []string
    cursor   int
    focused  bool
}

func (l ListComponent) Update(msg tea.Msg) (ListComponent, tea.Cmd) {
    if !l.focused {
        return l, nil
    }
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if l.cursor > 0 {
                l.cursor--
            }
        case "down", "j":
            if l.cursor < len(l.items)-1 {
                l.cursor++
            }
        case "enter":
            // Return custom message for parent to handle
            return l, func() tea.Msg {
                return itemSelectedMsg{l.items[l.cursor]}
            }
        }
    }
    return l, nil
}

func (l ListComponent) View() string {
    var b strings.Builder
    for i, item := range l.items {
        cursor := " "
        if l.focused && i == l.cursor {
            cursor = "â–¸"
        }
        b.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
    }
    return b.String()
}

// Parent model
type Model struct {
    list    ListComponent
    detail  DetailComponent
    focus   string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    // Handle component-specific messages
    switch msg := msg.(type) {
    case itemSelectedMsg:
        m.detail = NewDetailComponent(msg.item)
        return m, nil
    case tea.KeyMsg:
        if msg.String() == "tab" {
            // Toggle focus
            if m.focus == "list" {
                m.focus = "detail"
                m.list.focused = false
                m.detail.focused = true
            } else {
                m.focus = "list"
                m.list.focused = true
                m.detail.focused = false
            }
            return m, nil
        }
    }
    
    // Update focused component
    var cmd tea.Cmd
    if m.focus == "list" {
        m.list, cmd = m.list.Update(msg)
        cmds = append(cmds, cmd)
    } else {
        m.detail, cmd = m.detail.Update(msg)
        cmds = append(cmds, cmd)
    }
    
    return m, tea.Batch(cmds...)
}
```

## Lipgloss Styling

### Style Definitions

```go
var (
    // Colors
    subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
    highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
    special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
    
    // Styles
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(highlight).
        Padding(0, 1)
    
    itemStyle = lipgloss.NewStyle().
        PaddingLeft(2)
    
    selectedItemStyle = lipgloss.NewStyle().
        PaddingLeft(2).
        Foreground(special).
        Bold(true)
    
    statusBarStyle = lipgloss.NewStyle().
        Foreground(subtle).
        Padding(0, 1)
    
    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#626262"))
    
    // Box styles
    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(subtle).
        Padding(1, 2)
)
```

### Responsive Layout

```go
func (m Model) View() string {
    // Calculate available space
    contentWidth := m.width - 4  // Account for borders
    contentHeight := m.height - 6  // Account for header/footer
    
    // Build components with dynamic sizing
    header := titleStyle.Width(m.width).Render("My App")
    
    content := boxStyle.
        Width(contentWidth).
        Height(contentHeight).
        Render(m.renderContent())
    
    footer := statusBarStyle.Width(m.width).Render(m.status)
    
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### Layout Helpers

```go
// Horizontal layout
row := lipgloss.JoinHorizontal(lipgloss.Top,
    leftPanel.Render(leftContent),
    rightPanel.Render(rightContent),
)

// Vertical layout
column := lipgloss.JoinVertical(lipgloss.Left,
    header,
    content,
    footer,
)

// Centering
centered := lipgloss.Place(
    m.width, m.height,
    lipgloss.Center, lipgloss.Center,
    content,
)

// Inline styling
text := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#FF0000")).
    Render("Error!")
```

## Spinners and Progress

### Spinner Component

```go
import "github.com/charmbracelet/bubbles/spinner"

type Model struct {
    spinner  spinner.Model
    loading  bool
}

func NewModel() Model {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    return Model{spinner: s, loading: true}
}

func (m Model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m Model) View() string {
    if m.loading {
        return fmt.Sprintf("%s Loading...", m.spinner.View())
    }
    return "Done!"
}
```

### Progress Bar

```go
import "github.com/charmbracelet/bubbles/progress"

type Model struct {
    progress progress.Model
    percent  float64
}

func NewModel() Model {
    p := progress.New(
        progress.WithDefaultGradient(),
        progress.WithWidth(40),
    )
    return Model{progress: p}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case progressMsg:
        m.percent = float64(msg)
        return m, nil
    case progress.FrameMsg:
        pm, cmd := m.progress.Update(msg)
        m.progress = pm.(progress.Model)
        return m, cmd
    }
    return m, nil
}

func (m Model) View() string {
    return m.progress.ViewAs(m.percent)
}
```

## Text Input

```go
import "github.com/charmbracelet/bubbles/textinput"

type Model struct {
    input    textinput.Model
    mode     string
}

func NewModel() Model {
    ti := textinput.New()
    ti.Placeholder = "Enter text..."
    ti.Focus()
    ti.CharLimit = 156
    ti.Width = 40
    return Model{input: ti, mode: "input"}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEnter:
            value := m.input.Value()
            m.input.Reset()
            return m, processInput(value)
        case tea.KeyEsc:
            m.mode = "normal"
            m.input.Blur()
        }
    }
    
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    return m.input.View()
}
```

## Ticking/Animation

```go
type tickMsg time.Time

func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m Model) Init() tea.Cmd {
    return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tickMsg:
        m.currentTime = time.Time(msg)
        return m, tickCmd()  // Continue ticking
    }
    return m, nil
}
```

## Program Options

### Starting the Program

```go
func main() {
    p := tea.NewProgram(
        NewModel(),
        tea.WithAltScreen(),        // Use alternate screen buffer
        tea.WithMouseCellMotion(),  // Enable mouse support
    )
    
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Sending Messages from Outside

```go
func main() {
    p := tea.NewProgram(NewModel())
    
    // Send message from another goroutine
    go func() {
        time.Sleep(5 * time.Second)
        p.Send(externalUpdateMsg{data: "new data"})
    }()
    
    p.Run()
}
```

## Testing TUI Components

```go
func TestModelUpdate(t *testing.T) {
    m := NewModel()
    
    // Simulate key press
    newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
    updatedM := newModel.(Model)
    
    assert.Equal(t, 1, updatedM.cursor)
}

func TestModelView(t *testing.T) {
    m := Model{
        items:  []string{"a", "b", "c"},
        cursor: 0,
    }
    
    view := m.View()
    assert.Contains(t, view, "â–¸ a")  // Selected indicator
    assert.Contains(t, view, "  b")  // Non-selected
}
```

## Sharp Edges

### 1. View Must Be Fast

```go
// WRONG: I/O in View
func (m Model) View() string {
    data, _ := os.ReadFile("data.txt")  // BUG: Blocking I/O
    return string(data)
}

// CORRECT: Load data via commands, render from state
func (m Model) View() string {
    return m.data  // Already loaded in state
}
```

### 2. Don't Forget tea.Batch

```go
// WRONG: Lost command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd1, cmd2 tea.Cmd
    m.spinner, cmd1 = m.spinner.Update(msg)
    m.list, cmd2 = m.list.Update(msg)
    return m, cmd1  // BUG: cmd2 lost!
}

// CORRECT: Batch all commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)
    
    m.list, cmd = m.list.Update(msg)
    cmds = append(cmds, cmd)
    
    return m, tea.Batch(cmds...)
}
```

### 3. WindowSizeMsg on Startup

```go
// The first message is often WindowSizeMsg
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true  // Now safe to render
    }
    // ...
}

func (m Model) View() string {
    if !m.ready {
        return "Initializing..."  // Don't render until we have dimensions
    }
    // ...
}
```

### 4. Alt Screen Buffer

```go
// Use alt screen to preserve terminal history
p := tea.NewProgram(model, tea.WithAltScreen())

// Exit cleanly restores original screen
// Don't use os.Exit() - let Run() return normally
```
```

### go-cobra.md
```markdown
# GO Cobra CLI Conventions - Lisan al-Gaib

## Overview

Cobra is the standard CLI framework for GO. These conventions ensure professional-grade CLI applications with proper configuration, error handling, and user experience.

## Project Structure

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go           # Minimal entrypoint
├── internal/
│   └── cli/
│       ├── root.go           # Root command + global config
│       ├── serve/
│       │   └── command.go    # Subcommand factory
│       ├── config/
│       │   └── command.go
│       └── version/
│           └── command.go
├── go.mod
└── go.sum
```

## Main Entry Point

### Minimal main.go

```go
// cmd/myapp/main.go
package main

import (
    "os"
    "myapp/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Root Command

### Standard Pattern

```go
// internal/cli/root.go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
)

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "A professional CLI tool",
    Long: `MyApp - A comprehensive tool for X.

Complete documentation at https://myapp.example.com`,
    
    // PersistentPreRunE runs before ANY subcommand
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return initConfig()
    },
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Global flags (available to all subcommands)
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.myapp/config.toml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    
    // Bind to viper
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
    
    // Add subcommands
    rootCmd.AddCommand(serve.NewCommand())
    rootCmd.AddCommand(config.NewCommand())
    rootCmd.AddCommand(version.NewCommand())
}

func initConfig() error {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            return fmt.Errorf("find home directory: %w", err)
        }
        
        configDir := filepath.Join(home, ".myapp")
        viper.AddConfigPath(configDir)
        viper.SetConfigName("config")
        viper.SetConfigType("toml")
    }
    
    // Environment variables
    viper.SetEnvPrefix("MYAPP")
    viper.AutomaticEnv()
    
    // Read config (ignore if not found)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("read config: %w", err)
        }
    }
    
    return nil
}
```

## Subcommand Pattern

### Factory Function Pattern

```go
// internal/cli/serve/command.go
package serve

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

func NewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "serve",
        Short: "Start the server",
        Long:  `Start the HTTP server on the specified port.`,
        Example: `  myapp serve --port 8080
  myapp serve --config /path/to/config.toml`,
        
        RunE: runServe,
    }
    
    // Local flags (only for this command)
    cmd.Flags().IntP("port", "p", 8080, "port to listen on")
    cmd.Flags().String("host", "localhost", "host to bind to")
    
    // Bind local flags to viper
    viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
    viper.BindPFlag("server.host", cmd.Flags().Lookup("host"))
    
    return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
    // CRITICAL: Silence usage on runtime errors
    cmd.SilenceUsage = true
    
    // Get values from viper (respects flag > env > config > default)
    port := viper.GetInt("server.port")
    host := viper.GetString("server.host")
    
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigCh
        fmt.Println("\nShutting down...")
        cancel()
    }()
    
    // Start server
    return startServer(ctx, host, port)
}
```

## Viper Integration

### CRITICAL: Configuration Priority

```go
// Viper priority (highest to lowest):
// 1. Explicit flag value (--port 8080)
// 2. Environment variable (MYAPP_SERVER_PORT=8080)
// 3. Config file value
// 4. Default value

// WRONG: Using flag directly (ignores config file)
port, _ := cmd.Flags().GetInt("port")

// CORRECT: Using viper (respects full priority chain)
port := viper.GetInt("server.port")
```

### Binding Flags to Viper

```go
// In NewCommand():
cmd.Flags().Int("port", 8080, "port number")
viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))

// In RunE:
port := viper.GetInt("server.port")

// Environment variable: MYAPP_SERVER_PORT (automatic with SetEnvPrefix)
```

### Config File Structure

```toml
# ~/.myapp/config.toml

[server]
port = 8080
host = "0.0.0.0"

[api]
key = "sk-..."
timeout = "30s"

[logging]
level = "info"
format = "json"
```

## Error Handling

### RunE vs Run

```go
// CORRECT: Use RunE for proper error propagation
RunE: func(cmd *cobra.Command, args []string) error {
    cmd.SilenceUsage = true  // Don't show usage on runtime errors
    
    if err := doWork(); err != nil {
        return fmt.Errorf("work failed: %w", err)
    }
    return nil
},

// WRONG: Using Run with os.Exit
Run: func(cmd *cobra.Command, args []string) {
    if err := doWork(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)  // Skips cleanup, bad practice
    }
},
```

### SilenceUsage Pattern

```go
RunE: func(cmd *cobra.Command, args []string) error {
    // Set this FIRST - prevents usage output on runtime errors
    cmd.SilenceUsage = true
    
    // Now do work...
    result, err := process(args)
    if err != nil {
        // Error shown, but NOT usage help
        return err
    }
    
    fmt.Println(result)
    return nil
},
```

## Argument Validation

### Built-in Validators

```go
cmd := &cobra.Command{
    Use:   "delete [id]",
    Args:  cobra.ExactArgs(1),  // Exactly 1 arg required
    RunE:  runDelete,
}

// Available validators:
// cobra.NoArgs              - No arguments allowed
// cobra.ExactArgs(n)        - Exactly n arguments
// cobra.MinimumNArgs(n)     - At least n arguments
// cobra.MaximumNArgs(n)     - At most n arguments
// cobra.RangeArgs(min, max) - Between min and max arguments
// cobra.OnlyValidArgs       - Must be in ValidArgs list
```

### Custom Validation

```go
cmd := &cobra.Command{
    Use:  "process [file]",
    Args: func(cmd *cobra.Command, args []string) error {
        if len(args) != 1 {
            return fmt.Errorf("requires exactly one file argument")
        }
        
        if _, err := os.Stat(args[0]); os.IsNotExist(err) {
            return fmt.Errorf("file %q does not exist", args[0])
        }
        
        return nil
    },
    RunE: runProcess,
}
```

## Completion Support

### Enable Shell Completion

```go
func init() {
    rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate shell completion script",
    Long: `Generate shell completion script for the specified shell.

To load completions:

Bash:
  $ source <(myapp completion bash)
  # Or add to ~/.bashrc

Zsh:
  $ myapp completion zsh > "${fpath[1]}/_myapp"

Fish:
  $ myapp completion fish > ~/.config/fish/completions/myapp.fish

PowerShell:
  PS> myapp completion powershell | Out-String | Invoke-Expression
`,
    ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
    Args:      cobra.ExactValidArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        switch args[0] {
        case "bash":
            return rootCmd.GenBashCompletion(os.Stdout)
        case "zsh":
            return rootCmd.GenZshCompletion(os.Stdout)
        case "fish":
            return rootCmd.GenFishCompletion(os.Stdout, true)
        case "powershell":
            return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
        default:
            return fmt.Errorf("unknown shell: %s", args[0])
        }
    },
}
```

### Dynamic Completion

```go
cmd := &cobra.Command{
    Use: "select [item]",
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        if len(args) != 0 {
            return nil, cobra.ShellCompDirectiveNoFileComp
        }
        
        // Return dynamic suggestions
        items := []string{"alpha", "beta", "gamma"}
        return items, cobra.ShellCompDirectiveNoFileComp
    },
}
```

## Output Formatting

### JSON Output Flag

```go
var outputFormat string

func init() {
    rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text|json)")
}

func printResult(result interface{}) error {
    switch outputFormat {
    case "json":
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(result)
    case "text":
        fmt.Printf("%+v\n", result)
        return nil
    default:
        return fmt.Errorf("unknown output format: %s", outputFormat)
    }
}
```

### Progress Output

```go
// Use stderr for progress, stdout for results
fmt.Fprintln(os.Stderr, "Processing...")

// Clear progress line
fmt.Fprint(os.Stderr, "\r                    \r")

// Final result to stdout
fmt.Fprintln(os.Stdout, result)
```

## Testing Commands

### Test Helper

```go
func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
    buf := new(bytes.Buffer)
    root.SetOut(buf)
    root.SetErr(buf)
    root.SetArgs(args)
    
    err = root.Execute()
    return buf.String(), err
}

func TestServeCommand(t *testing.T) {
    output, err := executeCommand(rootCmd, "serve", "--port", "9090")
    require.NoError(t, err)
    assert.Contains(t, output, "Starting server")
}
```

## Version Command

### Standard Pattern

```go
// Set at build time with ldflags
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("myapp %s\n", version)
        fmt.Printf("  commit: %s\n", commit)
        fmt.Printf("  built:  %s\n", date)
    },
}

// Build with:
// go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Sharp Edges

### 1. Flag Binding Timing

```go
// WRONG: Binding in RunE (too late, viper already initialized)
RunE: func(cmd *cobra.Command, args []string) error {
    viper.BindPFlag("port", cmd.Flags().Lookup("port"))  // TOO LATE
    ...
}

// CORRECT: Binding in init() or NewCommand()
func NewCommand() *cobra.Command {
    cmd := &cobra.Command{...}
    cmd.Flags().Int("port", 8080, "port")
    viper.BindPFlag("port", cmd.Flags().Lookup("port"))  // CORRECT
    return cmd
}
```

### 2. Persistent vs Local Flags

```go
// Persistent: Available to this command AND all subcommands
rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

// Local: Only available to THIS command
serveCmd.Flags().Int("port", 8080, "port")
```

### 3. Required Flags

```go
cmd.Flags().String("api-key", "", "API key (required)")
cmd.MarkFlagRequired("api-key")

// Better UX: Custom validation with helpful message
RunE: func(cmd *cobra.Command, args []string) error {
    apiKey := viper.GetString("api-key")
    if apiKey == "" {
        return fmt.Errorf("API key required. Set via --api-key flag, MYAPP_API_KEY env var, or config file")
    }
    ...
}
```

### 4. Hidden Commands

```go
// For internal/debug commands
debugCmd := &cobra.Command{
    Use:    "debug",
    Hidden: true,  // Won't show in help
    ...
}
```
```

### go.md
```markdown
# GO Conventions - Lisan al-Gaib

## System Constraints (CRITICAL)

**This system targets desktop distribution. All GO code must:**
1. Compile to single binary with zero runtime dependencies
2. Cross-compile for darwin/amd64, darwin/arm64, windows/amd64, linux/amd64
3. Embed all static assets using `go:embed`
4. Never require users to install GO toolchain

## Project Structure

### Start Simple, Add Complexity Only When Needed

```
# Minimum viable (single binary)
myproject/
  go.mod
  main.go

# Add internal/ for private packages (compiler-enforced)
myproject/
  main.go
  internal/
    config/config.go
    handlers/handlers.go
  go.mod

# Add cmd/ only for multiple binaries
myproject/
  cmd/
    api/main.go
    worker/main.go
  internal/
    shared/
    api/
    worker/
  go.mod
```

**Rules:**
- `internal/` - Private packages, cannot be imported externally
- `pkg/` - ONLY if explicitly sharing code as library (rarely needed)
- `cmd/` - ONLY for multiple binaries
- Never use `golang-standards/project-layout` structure blindly

### Embedding Static Files

```go
// CORRECT: Package-level embed
//go:embed templates/*.html static/
var content embed.FS

// CORRECT: Single file
//go:embed version.txt
var version string

// WRONG: Embedding in function (won't compile)
func loadTemplates() {
    //go:embed templates/  // ERROR
}

// PREFER: //go:embed dirname over //go:embed dirname/* (latter includes dotfiles)
```

## Error Handling

### Wrapping Errors

```go
// CORRECT: Wrap with context using %w
if err := db.Query(ctx, query); err != nil {
    return fmt.Errorf("query users table: %w", err)
}

// CORRECT: Check specific errors
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}

// CORRECT: Extract typed errors
var apiErr *APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
    return handleRateLimit(apiErr)
}

// WRONG: String comparison
if err.Error() == "not found" {  // NEVER
    // ...
}

// WRONG: Bare error return
return err  // Add context!
```

### Sentinel Errors

```go
// Define at package level
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
    ErrRateLimited   = errors.New("rate limited")
)

// Custom error types with Unwrap
type ValidationError struct {
    Field   string
    Message string
    Err     error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed on %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
    return e.Err
}
```

### Panic Rules

```go
// CORRECT: Panic for programming errors only
func MustCompile(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

// WRONG: Panic for expected conditions
func GetUser(id int) *User {
    user, err := db.GetUser(id)
    if err != nil {
        panic(err)  // NEVER - return error instead
    }
    return user
}
```

## Concurrency Patterns

### Context Propagation

```go
// CORRECT: Accept context as first parameter
func (s *Service) ProcessTask(ctx context.Context, task Task) error {
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Pass context to all downstream calls
    result, err := s.api.Fetch(ctx, task.URL)
    if err != nil {
        return fmt.Errorf("fetch: %w", err)
    }
    
    return s.store.Save(ctx, result)
}

// WRONG: Ignoring context
func (s *Service) ProcessTask(task Task) error {
    result, _ := s.api.Fetch(context.Background(), task.URL)  // WRONG
    // ...
}
```

### Worker Pool Pattern

```go
type WorkerPool struct {
    numWorkers int
    jobs       chan Job
    results    chan Result
    wg         sync.WaitGroup
}

func NewWorkerPool(n int, bufferSize int) *WorkerPool {
    return &WorkerPool{
        numWorkers: n,
        jobs:       make(chan Job, bufferSize),
        results:    make(chan Result, bufferSize),
    }
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.numWorkers; i++ {
        wp.wg.Add(1)
        go wp.worker(ctx, i)
    }
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
    defer wp.wg.Done()
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-wp.jobs:
            if !ok {
                return
            }
            result := wp.process(job)
            select {
            case wp.results <- result:
            case <-ctx.Done():
                return
            }
        }
    }
}

func (wp *WorkerPool) Submit(job Job) {
    wp.jobs <- job
}

func (wp *WorkerPool) Close() {
    close(wp.jobs)
    wp.wg.Wait()
    close(wp.results)
}
```

### errgroup for Coordinated Operations

```go
import "golang.org/x/sync/errgroup"

func FetchAll(ctx context.Context, urls []string) ([]Result, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(urls))
    
    for i, url := range urls {
        i, url := i, url  // CRITICAL: Capture loop variables
        g.Go(func() error {
            result, err := fetch(ctx, url)
            if err != nil {
                return fmt.Errorf("fetch %s: %w", url, err)
            }
            results[i] = result
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

### Semaphore for Rate Limiting

```go
import "golang.org/x/sync/semaphore"

func ProcessWithLimit(ctx context.Context, tasks []Task, maxConcurrent int64) error {
    sem := semaphore.NewWeighted(maxConcurrent)
    g, ctx := errgroup.WithContext(ctx)
    
    for _, task := range tasks {
        task := task  // Capture
        
        if err := sem.Acquire(ctx, 1); err != nil {
            return fmt.Errorf("acquire semaphore: %w", err)
        }
        
        g.Go(func() error {
            defer sem.Release(1)
            return processTask(ctx, task)
        })
    }
    
    return g.Wait()
}
```

## HTTP Clients

### Never Use Default Client

```go
// WRONG: No timeout, can hang forever
resp, err := http.Get(url)

// CORRECT: Configured client with timeouts
func NewHTTPClient() *http.Client {
    return &http.Client{
        Timeout: 120 * time.Second,
        Transport: &http.Transport{
            DialContext: (&net.Dialer{
                Timeout:   10 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext,
            TLSHandshakeTimeout:   10 * time.Second,
            ResponseHeaderTimeout: 30 * time.Second,
            MaxIdleConns:          100,
            MaxIdleConnsPerHost:   10,
            IdleConnTimeout:       90 * time.Second,
            ForceAttemptHTTP2:     true,
        },
    }
}
```

### Exponential Backoff with Jitter

```go
func CalculateBackoff(attempt int, base, max time.Duration) time.Duration {
    delay := float64(base) * math.Pow(2.0, float64(attempt))
    jitter := delay * (0.5 + rand.Float64())  // Â±50% randomization
    if time.Duration(jitter) > max {
        return max
    }
    return time.Duration(jitter)
}

func RetryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }
        
        backoff := CalculateBackoff(attempt, 100*time.Millisecond, 30*time.Second)
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
    }
    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

## Testing

### Table-Driven Tests

```go
func TestParseConfig(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *Config
        wantErr  bool
    }{
        {
            name:     "valid config",
            input:    `{"port": 8080}`,
            expected: &Config{Port: 8080},
        },
        {
            name:    "invalid JSON",
            input:   `{invalid}`,
            wantErr: true,
        },
        {
            name:     "empty config uses defaults",
            input:    `{}`,
            expected: &Config{Port: 3000},
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result, err := ParseConfig([]byte(tc.input))
            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### Parallel Tests with Variable Capture

```go
func TestConcurrent(t *testing.T) {
    tests := []struct{
        name string
        input int
    }{
        {"case1", 1},
        {"case2", 2},
    }
    
    for _, tc := range tests {
        tc := tc  // CRITICAL: Capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            // test logic using tc
        })
    }
}
```

### Run With Race Detector

```bash
# ALWAYS run in development
go test -race ./...

# CI should fail on race conditions
go test -race -count=1 ./...
```

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Package | lowercase, single word | `http`, `config`, `agent` |
| Exported | PascalCase | `Client`, `NewServer`, `Config` |
| Unexported | camelCase | `config`, `parseInput`, `client` |
| Receiver | 1-2 letter abbreviation | `func (c *Client) Do()` |
| Interface | -er suffix for single method | `Reader`, `Writer`, `Stringer` |
| Getters | No "Get" prefix | `func (u *User) Name() string` |
| Initialisms | Consistent case | `userID`, `httpClient`, `apiURL` |

### Avoid Stuttering

```go
// BAD: user.UserService
package user
type UserService struct{}

// GOOD: user.Service
package user
type Service struct{}
```

## Documentation

### Doc Comments Start With Name

```go
// Client is an HTTP client for the Claude API.
// Its zero value is not usable; use NewClient instead.
type Client struct {
    // APIKey is the authentication key for the API.
    // Required.
    APIKey string
    
    // Timeout specifies a time limit for requests.
    // Zero means no timeout.
    Timeout time.Duration
}

// NewClient creates a Client with the given API key.
// It returns an error if apiKey is empty.
func NewClient(apiKey string) (*Client, error)
```

## Linting Configuration

### .golangci.yml

```yaml
linters:
  enable:
    - errcheck       # Check error returns
    - govet          # Go vet checks
    - staticcheck    # Comprehensive static analysis
    - gosimple       # Simplification suggestions
    - ineffassign    # Detect ineffectual assignments
    - bodyclose      # HTTP response body closure
    - gosec          # Security issues
    - gofmt          # Format checking
    - goimports      # Import organization

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
  govet:
    enable-all: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

## Build Commands

### Makefile Template

```makefile
BINARY_NAME=lisan
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: build build-all clean test lint

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/${BINARY_NAME}

build-all:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ./cmd/${BINARY_NAME}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ./cmd/${BINARY_NAME}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ./cmd/${BINARY_NAME}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ./cmd/${BINARY_NAME}

clean:
	rm -f ${BINARY_NAME}
	rm -rf dist/

test:
	go test -race -v ./...

lint:
	golangci-lint run
```

## Sharp Edges

### Common Gotchas

1. **Loop variable capture in goroutines**
   ```go
   // WRONG: All goroutines see same value
   for _, item := range items {
       go func() {
           process(item)  // BUG: item changes
       }()
   }
   
   // CORRECT: Capture variable
   for _, item := range items {
       item := item  // Capture
       go func() {
           process(item)
       }()
   }
   ```

2. **Nil slice vs empty slice**
   ```go
   var s []int        // nil slice, json: null
   s := []int{}       // empty slice, json: []
   s := make([]int,0) // empty slice, json: []
   ```

3. **defer in loops**
   ```go
   // WRONG: Defers accumulate until function returns
   for _, file := range files {
       f, _ := os.Open(file)
       defer f.Close()  // Won't close until function ends
   }
   
   // CORRECT: Use anonymous function
   for _, file := range files {
       func() {
           f, _ := os.Open(file)
           defer f.Close()
           // ... use f ...
       }()
   }
   ```

4. **Channel closing**
   ```go
   // Only sender should close channels
   // Never close from receiver side
   // Closing twice causes panic
   ```

5. **Context cancellation**
   ```go
   // Always check context in long-running operations
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
   }
   ```
```

### python.md
```markdown
---
paths:
  - "**/*.py"
---

# Agent Guidelines for Python Code Quality

This document provides guidelines for maintaining high-quality Python code. These rules MUST be followed by all AI coding agents and contributors.

## Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:
- Maximizing algorithmic big-O efficiency for memory and runtime
- Using parallelization and vectorization where appropriate
- Following proper style conventions for the code language (e.g., maximizing code reuse (DRY))
- No extra code beyond what is absolutely necessary to solve the problem the user provides (i.e., no technical debt)

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

---

## Python Version and Modern Features

**MUST** target Python 3.11+ for new projects; 3.12+ preferred.

### Python 3.11+ Features to Adopt
- **Self type** for methods returning `self` (replaces verbose TypeVar patterns)
- **ExceptionGroup** and `except*` for handling multiple concurrent errors
- **tomllib** for TOML parsing (built-in, replaces external toml packages)
- **asyncio.TaskGroup** for structured concurrency (replaces `asyncio.gather()`)

### Python 3.12+ Features to Adopt
- **PEP 695 type parameter syntax**: `class Stack[T]:` instead of `TypeVar`
- **F-string improvements**: nested quotes, multi-line expressions, comments allowed
- **`@override` decorator** for explicit method overriding
- **`type` statement** for type aliases: `type Vector[T] = list[T]`

### Python 3.13+ Features (When Available)
- **TypeIs** for type narrowing (prefer over TypeGuard)
- **ReadOnly** TypedDict fields for immutable configurations
- **`@deprecated`** decorator for marking deprecated APIs

**Example - Modern Generic Syntax (Python 3.12+):**
```python
from typing import Self

# PEP 695 syntax - PREFERRED
class Builder[T]:
    def add(self, item: T) -> Self:
        self._items.append(item)
        return self

# Type alias
type JSON = dict[str, "JSON"] | list["JSON"] | str | int | float | bool | None
```

---

## Preferred Tools

### Package Management
- **MUST** use `uv` for Python package management and virtual environments
- **MUST** commit `uv.lock` to version control for reproducibility
- **MUST** use `uv run` to execute Python commands in the virtual environment
- **MUST** add dependencies with `uv add` and `uv add --dev` for dev dependencies
```bash
uv init --package myproject  # Create project with src layout
uv add requests polars       # Add dependencies
uv add --dev pytest ruff mypy  # Add dev dependencies
uv sync                       # Install all dependencies
uv run pytest                 # Run commands in venv
```

### Environment: Arch Linux / CachyOS (CRITICAL)
On this system, **pip is incompatible with the system Python** due to Arch's externally-managed Python policy.

**Running Python:**
- **MUST** use a virtual environment for all Python execution
- **NEVER** run `pip install` or bare `python` on system Python—it will fail

**General-purpose venv:** `~/.generic-python/`
```bash
# Direct invocation (preferred for scripts)
~/.generic-python/bin/python script.py

# Interactive use
genpy                    # Bash alias to activate the venv
python                   # Then run interactively
```

**When to use which:**
| Scenario | Command |
|----------|---------|
| Running a script | `~/.generic-python/bin/python script.py` |
| Interactive REPL | `genpy` then `python` |
| Project with `pyproject.toml` | `uv run python ...` |
| Installing packages | `uv add package` (in project) |

### Development Tools
- **Ruff** for linting and formatting (replaces Black, isort, flake8)
- **mypy** (or **Pyright**) for static type checking
- **pytest** for testing with **pytest-xdist** for parallel execution
- **pre-commit** for automated checks before commits
- **tqdm** for progress bars in long-running loops
- **orjson** for fast JSON loading/dumping
- Use `logger.error` instead of `print` for error reporting

### Data Science Tools
- **ALWAYS** use `polars` instead of `pandas` for data frame manipulation
- Use **lazy evaluation** (`pl.scan_csv()`) for files >50MB
- **NEVER** print DataFrame row count or schema alongside the DataFrame (redundant)
- **NEVER** ingest more than 10 rows of a DataFrame at a time in context
- For pandas (when required): enable `dtype_backend="pyarrow"` and `copy_on_write`

**Example - Polars Lazy Evaluation:**
```python
import polars as pl

# PREFERRED: Lazy evaluation with query optimization
df = (
    pl.scan_csv("large_file.csv")
    .filter(pl.col("value") > 100)
    .group_by("category")
    .agg(pl.col("amount").sum())
    .collect()  # Execute optimized query at the end
)

# For larger-than-memory: use streaming
result = lazy_df.collect(streaming=True)
```

### Database Guidelines
- Do not denormalize unless explicitly prompted
- Use appropriate datatypes: `DATETIME/TIMESTAMP` for dates, `ARRAY` for nested fields
- **NEVER** save arrays as `TEXT/STRING`

---

## Code Style and Formatting

- **MUST** use meaningful, descriptive variable and function names
- **MUST** follow PEP 8 style guidelines
- **MUST** use 4 spaces for indentation (never tabs)
- **NEVER** use emoji or unicode emulating emoji (checkmarks, X marks) except in tests
- Use snake_case for functions/variables, PascalCase for classes, UPPER_CASE for constants
- Limit line length to 88 characters (Ruff formatter standard)

---

## Type Hints

### Requirements
- **MUST** use type hints for all function signatures (parameters and return values)
- **MUST** run mypy with `--strict` and resolve all type errors
- **NEVER** use `Any` type unless absolutely necessary—prefer `object` or union types
- **MUST** include `# type: ignore[error-code]` with specific codes, never bare ignores

### Modern Type Hint Patterns
```python
from typing import Self, TypedDict, Protocol, Literal, Final, TypeIs
from collections.abc import Sequence, Mapping

# Use Self for method chaining
class QueryBuilder:
    def where(self, condition: str) -> Self:
        return self

# Use TypedDict for structured dicts
class Config(TypedDict):
    host: str
    port: int
    debug: bool

# Use Protocol for structural subtyping
class Closeable(Protocol):
    def close(self) -> None: ...

# Use Literal for constrained values
Mode = Literal["read", "write", "append"]

# Use Final for constants
MAX_RETRIES: Final = 3

# Accept broad types, return specific types
def process(items: Sequence[str], config: Mapping[str, int]) -> list[str]:
    return [item.upper() for item in items]
```

### Type Hints to Avoid
- **NEVER** use `typing.List`, `typing.Dict`—use built-in `list`, `dict`
- **NEVER** use `Optional[X]`—use `X | None`
- **NEVER** use `Union[X, Y]`—use `X | Y`
- **NEVER** use old TypeVar syntax when PEP 695 is available (Python 3.12+)

---

## Documentation

- **MUST** include docstrings for all public functions, classes, and methods
- **MUST** document function parameters, return values, and exceptions raised
- Keep comments up-to-date with code changes
- Include examples in docstrings for complex functions
```python
def calculate_total(items: list[dict], tax_rate: float = 0.0) -> float:
    """Calculate the total cost of items including tax.

    Args:
        items: List of item dictionaries with 'price' keys
        tax_rate: Tax rate as decimal (e.g., 0.08 for 8%)

    Returns:
        Total cost including tax

    Raises:
        ValueError: If items is empty or tax_rate is negative

    Example:
        >>> calculate_total([{"price": 10}, {"price": 20}], 0.1)
        33.0
    """
```

---

## Error Handling

- **NEVER** silently swallow exceptions without logging
- **NEVER** use bare `except:` clauses
- **MUST** catch specific exceptions rather than broad exception types
- **MUST** use context managers (`with` statements) for resource cleanup
- Provide meaningful error messages

### Exception Groups (Python 3.11+)
Use `ExceptionGroup` and `except*` for concurrent operations:
```python
import asyncio

async def fetch_all(urls: list[str]) -> list[str]:
    try:
        async with asyncio.TaskGroup() as tg:
            tasks = [tg.create_task(fetch(url)) for url in urls]
        return [t.result() for t in tasks]
    except* ValueError as eg:
        for exc in eg.exceptions:
            logger.error(f"Validation error: {exc}")
        raise
    except* ConnectionError as eg:
        for exc in eg.exceptions:
            logger.error(f"Connection error: {exc}")
        raise
```

---

## Function Design

- **MUST** keep functions focused on a single responsibility
- **NEVER** use mutable objects (lists, dicts) as default argument values
- Limit function parameters to 5 or fewer
- Return early to reduce nesting

---

## Class Design

- **MUST** keep classes focused on a single responsibility
- **MUST** keep `__init__` simple; avoid complex logic
- Use `@dataclass(slots=True)` for simple data containers (memory efficient)
- Prefer composition over inheritance
- Use `@property` for computed attributes
- Use `@override` decorator when overriding parent methods (Python 3.12+)
```python
from dataclasses import dataclass
from typing import override

@dataclass(slots=True)
class Point:
    x: float
    y: float

class Animal:
    def speak(self) -> str:
        return "..."

class Dog(Animal):
    @override
    def speak(self) -> str:
        return "Woof!"
```

---

## Concurrency and Parallelization

### asyncio (I/O-bound tasks)
- **MUST** use `asyncio.TaskGroup` instead of `asyncio.gather()` for structured concurrency
- **MUST** use `asyncio.timeout()` for bounded async operations
- **MUST** let `CancelledError` propagate—do not swallow it
- **MUST** use `asyncio.to_thread()` for blocking operations in async code
```python
import asyncio

async def fetch_with_timeout(urls: list[str]) -> list[str]:
    async with asyncio.timeout(30.0):
        async with asyncio.TaskGroup() as tg:
            tasks = [tg.create_task(fetch(url)) for url in urls]
        return [t.result() for t in tasks]
```

### Multiprocessing (CPU-bound tasks)
- **MUST** use `ProcessPoolExecutor` for CPU-bound work
- **MUST** use context managers for executor lifecycle
- **MUST** use `if __name__ == "__main__":` guard
- **MUST** check `future.result()` to surface errors
```python
from concurrent.futures import ProcessPoolExecutor, as_completed

def parallel_process(items: list) -> list:
    with ProcessPoolExecutor() as executor:
        futures = {executor.submit(process, item): item for item in items}
        results = []
        for future in as_completed(futures):
            try:
                results.append(future.result())
            except Exception as e:
                logger.error(f"Failed: {futures[future]}: {e}")
        return results
```

### Concurrency Selection Guide
| Workload | Solution |
|----------|----------|
| I/O-bound (network, files) | `asyncio.TaskGroup` or `ThreadPoolExecutor` |
| CPU-bound (computation) | `ProcessPoolExecutor` |
| ML data preprocessing | `joblib` with loky backend |
| Large datasets | `Polars` (parallel by default) or `Dask` |

---

## Testing

### pytest Configuration
- **MUST** use pytest as the testing framework
- **MUST** use `pytest-xdist` for parallel test execution (`pytest -n auto`)
- **MUST** use `pytest-cov` with branch coverage (`--cov-branch`)
- **MUST** mock external dependencies (APIs, databases, file systems)
- **NEVER** run tests without saving them as discrete files first
- **NEVER** delete files created as part of testing
```bash
pytest -n auto --cov=src --cov-branch --cov-fail-under=80
```

### Property-Based Testing with Hypothesis
Use Hypothesis for testing invariants and discovering edge cases:
```python
from hypothesis import given, strategies as st

@given(st.lists(st.integers()))
def test_sorted_is_idempotent(lst: list[int]) -> None:
    result = sorted(lst)
    assert sorted(result) == result
    assert len(result) == len(lst)

@given(st.text(), st.text())
def test_string_concatenation_length(s1: str, s2: str) -> None:
    assert len(s1 + s2) == len(s1) + len(s2)
```

### Testing Best Practices
- Follow the Arrange-Act-Assert pattern
- Use `pytest.raises` for exception testing
- Use `pytest.mark.parametrize` for multiple inputs
- Use `autospec=True` when mocking to catch API mismatches
- **NEVER** write tests that depend on execution order

---

## Performance Optimization

### Profiling First
- **MUST** profile before optimizing—never optimize without data
- Use `cProfile` for function-level profiling
- Use `py-spy` for production profiling (zero code changes)
- Use `scalene` for combined CPU/memory/GPU profiling
```bash
# Generate flamegraph
py-spy record -o flame.svg -- python script.py

# Scalene profiling
scalene script.py
```

### Optimization Techniques
- Use NumPy/Polars vectorization over Python loops
- Use `@functools.cache` or `@lru_cache` for pure functions with repeated calls
- Use `__slots__` for classes with many instances
- Use generators for large sequence processing
- **NEVER** concatenate strings in loops—use `"".join()` or `io.StringIO`

### Memory Management
- Use generators instead of list comprehensions for large datasets
- Use `__slots__` for memory-constrained classes
- Process large files in chunks
- Use `numpy.memmap` for large numeric arrays

---

## Security

### Secrets Management
- **NEVER** store secrets, API keys, or passwords in code
- **MUST** use environment variables via `python-dotenv` or `pydantic-settings`
- **MUST** add `.env` to `.gitignore`
- **NEVER** print or log URLs containing API keys
- **NEVER** log sensitive information (passwords, tokens, PII)

### Input Validation
- **MUST** validate all external inputs with Pydantic
- **MUST** use parameterized queries for all database operations
- **MUST** use `os.path.basename()` for user-provided filenames
- **NEVER** use `pickle.loads()` with untrusted data (RCE vulnerability)
- **NEVER** use `yaml.load()`—use `yaml.safe_load()`
- **NEVER** use `eval()`, `exec()`, or `subprocess` with `shell=True` on user input

### Dependency Security
- **MUST** run `pip-audit` in CI pipeline
- **MUST** run `bandit` for static security analysis
- **MUST** use pre-commit hooks for secret scanning (detect-secrets, gitleaks)
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/PyCQA/bandit
    rev: 1.7.9
    hooks:
      - id: bandit
        args: [-r, -ll, -x, tests]
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.22.1
    hooks:
      - id: gitleaks
```

---

## Project Structure

Use the **src layout** for all packages:
```
myproject/
├── src/
│   └── mypackage/
│       ├── __init__.py
│       ├── py.typed          # For mypy
│       └── module.py
├── tests/
│   ├── __init__.py
│   └── test_module.py
├── pyproject.toml
├── uv.lock
├── .python-version
├── .pre-commit-config.yaml
└── README.md
```

---

## Configuration (pyproject.toml)
```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "mypackage"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=8.0",
    "pytest-cov>=4.0",
    "pytest-xdist>=3.0",
    "hypothesis>=6.0",
    "ruff>=0.5",
    "mypy>=1.10",
    "pre-commit>=3.7",
    "bandit>=1.7",
    "pip-audit>=2.7",
]

[tool.ruff]
target-version = "py311"
line-length = 88
src = ["src", "tests"]

[tool.ruff.lint]
select = ["E", "W", "F", "I", "B", "C4", "UP", "ARG", "SIM", "TCH", "PTH", "RUF"]
ignore = ["E501"]

[tool.ruff.lint.isort]
known-first-party = ["mypackage"]

[tool.mypy]
python_version = "3.11"
strict = true
warn_return_any = true
warn_unused_ignores = true
show_error_codes = true

[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "-ra --strict-markers"
asyncio_mode = "auto"

[tool.coverage.run]
source = ["src"]
branch = true

[tool.coverage.report]
fail_under = 80
show_missing = true
exclude_lines = ["pragma: no cover", "if TYPE_CHECKING:"]
```

---

## Imports and Dependencies

- **MUST** avoid wildcard imports (`from module import *`)
- **MUST** document dependencies in `pyproject.toml`
- Organize imports: standard library, third-party, local imports
- Ruff handles import sorting automatically

---

## Version Control

- **MUST** write clear, descriptive commit messages
- **NEVER** commit commented-out code; delete it
- **NEVER** commit debug print statements or breakpoints
- **NEVER** commit credentials or sensitive data

---

## Before Committing Checklist

- [ ] All tests pass (`uv run pytest -n auto`)
- [ ] Type checking passes (`uv run mypy src`)
- [ ] Linter and formatter pass (`uv run ruff check . && uv run ruff format .`)
- [ ] Security scan passes (`uv run bandit -r src && uv run pip-audit`)
- [ ] All functions have docstrings and type hints
- [ ] No commented-out code or debug statements
- [ ] No hardcoded credentials

---

**Remember:** Prioritize clarity and maintainability over cleverness.
```

### react.md
```markdown
# React Conventions

## System Constraints (CRITICAL)

**This system targets React 18+ with functional components and TypeScript.**

All React code must:
1. Use functional components exclusively - no class components
2. Include proper TypeScript types for all props and state
3. Follow React 18+ patterns (concurrent features, automatic batching)
4. Be compatible with strict mode
5. Never mutate state directly

## Component Patterns

### Component Structure

```typescript
// CORRECT: Complete component with types
interface UserCardProps {
  user: User;
  onEdit?: (user: User) => void;
  className?: string;
}

function UserCard({ user, onEdit, className }: UserCardProps): JSX.Element {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleEdit = useCallback(() => {
    onEdit?.(user);
  }, [onEdit, user]);

  return (
    <div className={className}>
      <h3>{user.name}</h3>
      <button onClick={() => setIsExpanded(!isExpanded)}>
        {isExpanded ? "Collapse" : "Expand"}
      </button>
      {isExpanded && <UserDetails user={user} onEdit={handleEdit} />}
    </div>
  );
}

// WRONG: Missing types
function UserCard({ user, onEdit }) {  // Implicit any
  // ...
}

// WRONG: Props destructuring without interface
function UserCard(props: { user: User; onEdit: Function }) {
  // Don't use Function type - too broad
}
```

### Props Best Practices

```typescript
// CORRECT: Specific prop types
interface ButtonProps {
  variant: "primary" | "secondary" | "danger";
  size?: "small" | "medium" | "large";
  disabled?: boolean;
  onClick: (event: React.MouseEvent<HTMLButtonElement>) => void;
  children: React.ReactNode;
}

// CORRECT: Extending HTML element props
interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

function Input({ label, error, ...inputProps }: InputProps): JSX.Element {
  return (
    <div>
      <label>{label}</label>
      <input {...inputProps} />
      {error && <span className="error">{error}</span>}
    </div>
  );
}

// CORRECT: Component props with generics
interface ListProps<T> {
  items: T[];
  renderItem: (item: T) => React.ReactNode;
  keyExtractor: (item: T) => string;
}

function List<T>({ items, renderItem, keyExtractor }: ListProps<T>) {
  return (
    <ul>
      {items.map(item => (
        <li key={keyExtractor(item)}>{renderItem(item)}</li>
      ))}
    </ul>
  );
}

// Usage
<List
  items={users}
  renderItem={user => <UserCard user={user} />}
  keyExtractor={user => user.id}
/>
```

### Composition Pattern

```typescript
// CORRECT: Compound components with context
interface TabsContextValue {
  activeTab: string;
  setActiveTab: (id: string) => void;
}

const TabsContext = React.createContext<TabsContextValue | null>(null);

function useTabs(): TabsContextValue {
  const context = useContext(TabsContext);
  if (!context) {
    throw new Error("useTabs must be used within Tabs");
  }
  return context;
}

interface TabsProps {
  defaultTab: string;
  children: React.ReactNode;
}

function Tabs({ defaultTab, children }: TabsProps): JSX.Element {
  const [activeTab, setActiveTab] = useState(defaultTab);

  const value = useMemo(
    () => ({ activeTab, setActiveTab }),
    [activeTab]
  );

  return (
    <TabsContext.Provider value={value}>
      <div className="tabs">{children}</div>
    </TabsContext.Provider>
  );
}

interface TabListProps {
  children: React.ReactNode;
}

function TabList({ children }: TabListProps): JSX.Element {
  return <div className="tab-list">{children}</div>;
}

interface TabProps {
  id: string;
  children: React.ReactNode;
}

function Tab({ id, children }: TabProps): JSX.Element {
  const { activeTab, setActiveTab } = useTabs();

  return (
    <button
      className={activeTab === id ? "active" : ""}
      onClick={() => setActiveTab(id)}
    >
      {children}
    </button>
  );
}

interface TabPanelProps {
  id: string;
  children: React.ReactNode;
}

function TabPanel({ id, children }: TabPanelProps): JSX.Element | null {
  const { activeTab } = useTabs();

  if (activeTab !== id) {
    return null;
  }

  return <div className="tab-panel">{children}</div>;
}

// Attach subcomponents
Tabs.List = TabList;
Tabs.Tab = Tab;
Tabs.Panel = TabPanel;

// Usage
<Tabs defaultTab="profile">
  <Tabs.List>
    <Tabs.Tab id="profile">Profile</Tabs.Tab>
    <Tabs.Tab id="settings">Settings</Tabs.Tab>
  </Tabs.List>
  <Tabs.Panel id="profile">Profile content</Tabs.Panel>
  <Tabs.Panel id="settings">Settings content</Tabs.Panel>
</Tabs>
```

### Render Props Pattern

```typescript
interface MouseTrackerProps {
  children: (position: { x: number; y: number }) => React.ReactNode;
}

function MouseTracker({ children }: MouseTrackerProps): JSX.Element {
  const [position, setPosition] = useState({ x: 0, y: 0 });

  useEffect(() => {
    const handleMouseMove = (event: MouseEvent): void => {
      setPosition({ x: event.clientX, y: event.clientY });
    };

    window.addEventListener("mousemove", handleMouseMove);
    return () => window.removeEventListener("mousemove", handleMouseMove);
  }, []);

  return <>{children(position)}</>;
}

// Usage
<MouseTracker>
  {({ x, y }) => (
    <div>
      Mouse position: {x}, {y}
    </div>
  )}
</MouseTracker>
```

### Higher-Order Components (Use Sparingly)

```typescript
// PREFER: Custom hooks over HOCs
// Only use HOCs for cross-cutting concerns like error boundaries

function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>
): React.ComponentType<P> {
  return function WithErrorBoundary(props: P): JSX.Element {
    return (
      <ErrorBoundary>
        <Component {...props} />
      </ErrorBoundary>
    );
  };
}

// BETTER: Use hooks instead
function useErrorHandler(): {
  error: Error | null;
  resetError: () => void;
} {
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const handleError = (event: ErrorEvent): void => {
      setError(event.error);
    };

    window.addEventListener("error", handleError);
    return () => window.removeEventListener("error", handleError);
  }, []);

  const resetError = useCallback(() => setError(null), []);

  return { error, resetError };
}
```

## Hooks Mastery

### useState Rules

```typescript
// CORRECT: Proper state initialization
const [count, setCount] = useState(0);
const [user, setUser] = useState<User | null>(null);

// CORRECT: Lazy initialization for expensive computation
const [data, setData] = useState(() => {
  return computeExpensiveValue();
});

// CORRECT: Functional updates for state based on previous state
const incrementCount = () => setCount(prev => prev + 1);

// WRONG: Mutating state
const [items, setItems] = useState<string[]>([]);
items.push("new");  // NEVER mutate state
setItems(items);

// CORRECT: Immutable state updates
setItems(prev => [...prev, "new"]);

// CORRECT: Object state updates
interface FormState {
  name: string;
  email: string;
}

const [form, setForm] = useState<FormState>({ name: "", email: "" });

const updateField = (field: keyof FormState, value: string): void => {
  setForm(prev => ({ ...prev, [field]: value }));
};

// WRONG: Multiple related state values
const [firstName, setFirstName] = useState("");
const [lastName, setLastName] = useState("");
const [email, setEmail] = useState("");

// CORRECT: Single state object
interface UserForm {
  firstName: string;
  lastName: string;
  email: string;
}

const [form, setForm] = useState<UserForm>({
  firstName: "",
  lastName: "",
  email: "",
});
```

### useEffect Rules

```typescript
// CORRECT: Effect with dependencies
useEffect(() => {
  fetchUser(userId);
}, [userId]);  // Re-run when userId changes

// CORRECT: Cleanup function
useEffect(() => {
  const subscription = api.subscribe(userId);

  return () => {
    subscription.unsubscribe();
  };
}, [userId]);

// CORRECT: Multiple effects for different concerns
useEffect(() => {
  // Track page view
  analytics.trackPageView(pathname);
}, [pathname]);

useEffect(() => {
  // Fetch data
  fetchData();
}, [dataId]);

// WRONG: Missing dependencies
useEffect(() => {
  fetchUser(userId);  // userId used but not in deps
}, []);  // ESLint error

// WRONG: Infinite loop
useEffect(() => {
  setCount(count + 1);  // Updates count, triggers effect again
}, [count]);

// CORRECT: Conditional effect execution
useEffect(() => {
  if (shouldFetch) {
    fetchData();
  }
}, [shouldFetch]);

// CORRECT: Abort controller for cleanup
useEffect(() => {
  const controller = new AbortController();

  async function fetchData(): Promise<void> {
    try {
      const response = await fetch(url, { signal: controller.signal });
      const data = await response.json();
      setData(data);
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        return;  // Ignore abort errors
      }
      setError(error);
    }
  }

  fetchData();

  return () => controller.abort();
}, [url]);
```

### useCallback

```typescript
// CORRECT: Memoize callbacks passed to child components
interface ChildProps {
  onUpdate: (value: string) => void;
}

const Child = React.memo(({ onUpdate }: ChildProps) => {
  // Component implementation
});

function Parent(): JSX.Element {
  const [value, setValue] = useState("");

  // Without useCallback, new function created on every render
  // causing Child to re-render even with React.memo
  const handleUpdate = useCallback((newValue: string) => {
    setValue(newValue);
    api.save(newValue);
  }, []);  // Empty deps because setValue is stable

  return <Child onUpdate={handleUpdate} />;
}

// WRONG: Overusing useCallback
function Component(): JSX.Element {
  // Not needed - this callback doesn't go to child components
  const handleClick = useCallback(() => {
    console.log("Clicked");
  }, []);

  return <button onClick={handleClick}>Click</button>;
}

// CORRECT: Don't wrap unless passed to memoized components
function Component(): JSX.Element {
  const handleClick = (): void => {
    console.log("Clicked");
  };

  return <button onClick={handleClick}>Click</button>;
}

// CORRECT: Include all dependencies
function SearchInput({ onSearch }: { onSearch: (q: string) => void }) {
  const [query, setQuery] = useState("");

  const handleSearch = useCallback(() => {
    onSearch(query);
  }, [query, onSearch]);  // Include all used values

  return (
    <input
      value={query}
      onChange={e => setQuery(e.target.value)}
      onKeyDown={e => e.key === "Enter" && handleSearch()}
    />
  );
}
```

### useMemo

```typescript
// CORRECT: Memoize expensive computations
function DataTable({ data }: { data: Item[] }): JSX.Element {
  const sortedData = useMemo(() => {
    return [...data].sort((a, b) => a.name.localeCompare(b.name));
  }, [data]);

  return <Table data={sortedData} />;
}

// WRONG: Memoizing cheap operations
function Component({ count }: { count: number }): JSX.Element {
  const doubled = useMemo(() => count * 2, [count]);  // Unnecessary
  return <div>{doubled}</div>;
}

// CORRECT: Memoize referential equality
function Component({ filters }: { filters: string[] }): JSX.Element {
  // Create stable object reference
  const filterConfig = useMemo(() => ({
    include: filters,
    caseSensitive: true,
  }), [filters]);

  return <FilteredList config={filterConfig} />;
}

// CORRECT: Memoize derived complex state
function useFilteredUsers(users: User[], query: string): User[] {
  return useMemo(() => {
    if (!query) return users;

    const lowerQuery = query.toLowerCase();
    return users.filter(user =>
      user.name.toLowerCase().includes(lowerQuery) ||
      user.email.toLowerCase().includes(lowerQuery)
    );
  }, [users, query]);
}
```

### useRef

```typescript
// CORRECT: DOM reference
function TextInput(): JSX.Element {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  return <input ref={inputRef} />;
}

// CORRECT: Store mutable value without triggering re-render
function Timer(): JSX.Element {
  const intervalRef = useRef<number | null>(null);
  const [count, setCount] = useState(0);

  useEffect(() => {
    intervalRef.current = window.setInterval(() => {
      setCount(c => c + 1);
    }, 1000);

    return () => {
      if (intervalRef.current !== null) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  return <div>Count: {count}</div>;
}

// CORRECT: Store previous value
function usePrevious<T>(value: T): T | undefined {
  const ref = useRef<T>();

  useEffect(() => {
    ref.current = value;
  }, [value]);

  return ref.current;
}

function Component({ count }: { count: number }): JSX.Element {
  const prevCount = usePrevious(count);

  return (
    <div>
      Current: {count}, Previous: {prevCount ?? "none"}
    </div>
  );
}

// WRONG: Using ref for state that should trigger re-render
function Counter(): JSX.Element {
  const countRef = useRef(0);

  const increment = (): void => {
    countRef.current++;  // Component won't re-render
  };

  return <div>{countRef.current}</div>;  // Stale value
}
```

### Custom Hooks

```typescript
// CORRECT: Custom hook for data fetching
interface UseFetchResult<T> {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  refetch: () => void;
}

function useFetch<T>(url: string): UseFetchResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [trigger, setTrigger] = useState(0);

  useEffect(() => {
    const controller = new AbortController();

    async function fetchData(): Promise<void> {
      setIsLoading(true);
      setError(null);

      try {
        const response = await fetch(url, { signal: controller.signal });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const json = await response.json();
        setData(json);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          return;
        }
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setIsLoading(false);
      }
    }

    fetchData();

    return () => controller.abort();
  }, [url, trigger]);

  const refetch = useCallback(() => {
    setTrigger(t => t + 1);
  }, []);

  return { data, error, isLoading, refetch };
}

// CORRECT: Custom hook for form state
interface UseFormOptions<T> {
  initialValues: T;
  onSubmit: (values: T) => void | Promise<void>;
  validate?: (values: T) => Partial<Record<keyof T, string>>;
}

function useForm<T extends Record<string, any>>({
  initialValues,
  onSubmit,
  validate,
}: UseFormOptions<T>) {
  const [values, setValues] = useState<T>(initialValues);
  const [errors, setErrors] = useState<Partial<Record<keyof T, string>>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = useCallback((field: keyof T, value: any) => {
    setValues(prev => ({ ...prev, [field]: value }));
    setErrors(prev => ({ ...prev, [field]: undefined }));
  }, []);

  const handleSubmit = useCallback(async (
    event: React.FormEvent
  ): Promise<void> => {
    event.preventDefault();

    if (validate) {
      const validationErrors = validate(values);
      if (Object.keys(validationErrors).length > 0) {
        setErrors(validationErrors);
        return;
      }
    }

    setIsSubmitting(true);
    try {
      await onSubmit(values);
    } finally {
      setIsSubmitting(false);
    }
  }, [values, validate, onSubmit]);

  const reset = useCallback(() => {
    setValues(initialValues);
    setErrors({});
  }, [initialValues]);

  return {
    values,
    errors,
    isSubmitting,
    handleChange,
    handleSubmit,
    reset,
  };
}

// Usage
function LoginForm(): JSX.Element {
  const { values, errors, isSubmitting, handleChange, handleSubmit } = useForm({
    initialValues: { email: "", password: "" },
    onSubmit: async (values) => {
      await login(values.email, values.password);
    },
    validate: (values) => {
      const errors: Partial<Record<keyof typeof values, string>> = {};
      if (!values.email) errors.email = "Required";
      if (!values.password) errors.password = "Required";
      return errors;
    },
  });

  return (
    <form onSubmit={handleSubmit}>
      <input
        value={values.email}
        onChange={e => handleChange("email", e.target.value)}
      />
      {errors.email && <span>{errors.email}</span>}
      <button disabled={isSubmitting}>Submit</button>
    </form>
  );
}
```

## State Management

### Zustand Patterns

**This project uses Zustand for global state.**

```typescript
import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import { immer } from "zustand/middleware/immer";

// CORRECT: Typed store with actions
interface User {
  id: string;
  name: string;
  email: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  updateUser: (updates: Partial<User>) => void;
}

const useAuthStore = create<AuthState>()(
  devtools(
    persist(
      (set) => ({
        user: null,
        isAuthenticated: false,

        login: async (email, password) => {
          const user = await api.login(email, password);
          set({ user, isAuthenticated: true });
        },

        logout: () => {
          set({ user: null, isAuthenticated: false });
        },

        updateUser: (updates) => {
          set((state) => ({
            user: state.user ? { ...state.user, ...updates } : null,
          }));
        },
      }),
      { name: "auth-storage" }
    )
  )
);

// CORRECT: Using immer for complex state updates
interface TodoState {
  todos: Todo[];
  addTodo: (text: string) => void;
  toggleTodo: (id: string) => void;
  removeTodo: (id: string) => void;
}

const useTodoStore = create<TodoState>()(
  immer((set) => ({
    todos: [],

    addTodo: (text) =>
      set((state) => {
        state.todos.push({ id: nanoid(), text, completed: false });
      }),

    toggleTodo: (id) =>
      set((state) => {
        const todo = state.todos.find((t) => t.id === id);
        if (todo) {
          todo.completed = !todo.completed;
        }
      }),

    removeTodo: (id) =>
      set((state) => {
        state.todos = state.todos.filter((t) => t.id !== id);
      }),
  }))
);

// CORRECT: Selective subscription to avoid unnecessary re-renders
function UserProfile(): JSX.Element {
  // Only re-renders when user changes
  const user = useAuthStore((state) => state.user);

  return <div>{user?.name}</div>;
}

// CORRECT: Subscribing to multiple values with shallow comparison
import { shallow } from "zustand/shallow";

function TodoStats(): JSX.Element {
  const { total, completed } = useTodoStore(
    (state) => ({
      total: state.todos.length,
      completed: state.todos.filter((t) => t.completed).length,
    }),
    shallow
  );

  return <div>{completed}/{total} completed</div>;
}

// WRONG: Subscribing to entire store
function TodoList(): JSX.Element {
  const state = useTodoStore();  // Re-renders on ANY state change
  return <div>{state.todos.length}</div>;
}
```

### Context API (Use Sparingly)

```typescript
// CORRECT: Context for dependency injection, not frequent updates
interface ThemeContextValue {
  theme: "light" | "dark";
  toggleTheme: () => void;
}

const ThemeContext = React.createContext<ThemeContextValue | null>(null);

function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}

function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<"light" | "dark">("light");

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === "light" ? "dark" : "light"));
  }, []);

  const value = useMemo(() => ({ theme, toggleTheme }), [theme, toggleTheme]);

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

// WRONG: Context for frequently updating data
// Use Zustand or other state management instead
```

## Performance

### React.memo

```typescript
// CORRECT: Memoize expensive components
interface ListItemProps {
  item: Item;
  onSelect: (id: string) => void;
}

const ListItem = React.memo(({ item, onSelect }: ListItemProps) => {
  return (
    <div onClick={() => onSelect(item.id)}>
      {item.name}
    </div>
  );
});

// CORRECT: Custom comparison function
const ListItem = React.memo(
  ({ item, onSelect }: ListItemProps) => {
    return <div onClick={() => onSelect(item.id)}>{item.name}</div>;
  },
  (prev, next) => {
    // Only re-render if item.id changed
    return prev.item.id === next.item.id;
  }
);

// WRONG: Memoizing everything
const SimpleButton = React.memo(({ onClick }: { onClick: () => void }) => {
  return <button onClick={onClick}>Click</button>;
});
// Not worth the overhead for simple components
```

### Code Splitting

```typescript
// CORRECT: Route-based code splitting
import { lazy, Suspense } from "react";

const Dashboard = lazy(() => import("./pages/Dashboard"));
const Settings = lazy(() => import("./pages/Settings"));

function App(): JSX.Element {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
}

// CORRECT: Component-based code splitting
const HeavyChart = lazy(() => import("./components/HeavyChart"));

function Analytics(): JSX.Element {
  const [showChart, setShowChart] = useState(false);

  return (
    <div>
      <button onClick={() => setShowChart(true)}>Show Chart</button>
      {showChart && (
        <Suspense fallback={<div>Loading chart...</div>}>
          <HeavyChart />
        </Suspense>
      )}
    </div>
  );
}
```

### List Virtualization

```typescript
// CORRECT: Use virtual scrolling for long lists
import { FixedSizeList } from "react-window";

interface VirtualListProps {
  items: Item[];
}

function VirtualList({ items }: VirtualListProps): JSX.Element {
  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => (
    <div style={style}>
      {items[index].name}
    </div>
  );

  return (
    <FixedSizeList
      height={600}
      itemCount={items.length}
      itemSize={50}
      width="100%"
    >
      {Row}
    </FixedSizeList>
  );
}
```

## Testing

### Component Testing with Vitest + Testing Library

```typescript
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";

// CORRECT: Test user interactions
describe("LoginForm", () => {
  it("calls onSubmit with form values", async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();

    render(<LoginForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText(/email/i), "test@example.com");
    await user.type(screen.getByLabelText(/password/i), "password123");
    await user.click(screen.getByRole("button", { name: /submit/i }));

    expect(onSubmit).toHaveBeenCalledWith({
      email: "test@example.com",
      password: "password123",
    });
  });

  it("displays validation error", async () => {
    const user = userEvent.setup();
    render(<LoginForm onSubmit={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: /submit/i }));

    expect(screen.getByText(/email is required/i)).toBeInTheDocument();
  });
});

// CORRECT: Test async behavior
describe("UserList", () => {
  it("loads and displays users", async () => {
    const users = [
      { id: "1", name: "Alice" },
      { id: "2", name: "Bob" },
    ];

    vi.mocked(api.getUsers).mockResolvedValue(users);

    render(<UserList />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("Alice")).toBeInTheDocument();
    });

    expect(screen.getByText("Bob")).toBeInTheDocument();
  });
});

// CORRECT: Test custom hooks
import { renderHook } from "@testing-library/react";

describe("useFetch", () => {
  it("fetches data", async () => {
    const data = { id: 1, name: "Test" };
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify(data))
    );

    const { result } = renderHook(() => useFetch("/api/data"));

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual(data);
    expect(result.current.error).toBeNull();
  });
});
```

## Ink-Specific Patterns

**This project uses Ink for terminal UI components.**

### Basic Ink Components

```typescript
import { Box, Text, useInput, useApp } from "ink";
import { useState } from "react";

// CORRECT: Terminal text display
function Header(): JSX.Element {
  return (
    <Box borderStyle="round" borderColor="cyan" padding={1}>
      <Text bold color="cyan">
        Application Title
      </Text>
    </Box>
  );
}

// CORRECT: Layout with Box
function Dashboard(): JSX.Element {
  return (
    <Box flexDirection="column" padding={1}>
      <Box marginBottom={1}>
        <Text bold>Dashboard</Text>
      </Box>
      <Box borderStyle="single" padding={1}>
        <Text>Content goes here</Text>
      </Box>
    </Box>
  );
}
```

### Ink Input Handling

```typescript
// CORRECT: useInput hook
function Menu(): JSX.Element {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const items = ["Option 1", "Option 2", "Option 3"];
  const { exit } = useApp();

  useInput((input, key) => {
    if (key.upArrow) {
      setSelectedIndex((prev) => Math.max(0, prev - 1));
    }

    if (key.downArrow) {
      setSelectedIndex((prev) => Math.min(items.length - 1, prev + 1));
    }

    if (key.return) {
      console.log(`Selected: ${items[selectedIndex]}`);
    }

    if (input === "q") {
      exit();
    }
  });

  return (
    <Box flexDirection="column">
      {items.map((item, index) => (
        <Text key={item} inverse={index === selectedIndex}>
          {item}
        </Text>
      ))}
    </Box>
  );
}
```

### Ink Constraints

```typescript
// WRONG: Using DOM-specific props
function BadInkComponent(): JSX.Element {
  return (
    <Box className="container">  {/* className doesn't work in Ink */}
      <Text onClick={() => {}}>Click</Text>  {/* No onClick in terminal */}
    </Box>
  );
}

// CORRECT: Ink-specific styling
function GoodInkComponent(): JSX.Element {
  return (
    <Box borderStyle="round" padding={1} borderColor="green">
      <Text color="green" bold>
        Success
      </Text>
    </Box>
  );
}

// CORRECT: Handle terminal dimensions
import { useStdout } from "ink";

function ResponsiveBox(): JSX.Element {
  const { stdout } = useStdout();
  const width = stdout.columns;

  return (
    <Box width={width - 4} borderStyle="single">
      <Text>Responsive content</Text>
    </Box>
  );
}
```

## Sharp Edges

### Stale Closures

```typescript
// WRONG: Stale closure in setInterval
function Timer(): JSX.Element {
  const [count, setCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setCount(count + 1);  // Always uses initial count (0)
    }, 1000);

    return () => clearInterval(interval);
  }, []);  // Empty deps - closure captures initial count

  return <div>{count}</div>;
}

// CORRECT: Use functional update
function Timer(): JSX.Element {
  const [count, setCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setCount((prev) => prev + 1);  // Uses current state
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return <div>{count}</div>;
}
```

### Dependency Array Issues

```typescript
// WRONG: Missing dependencies
function UserProfile({ userId }: { userId: string }): JSX.Element {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUser(userId).then(setUser);
  }, []);  // Missing userId - won't refetch on change

  return <div>{user?.name}</div>;
}

// CORRECT: Include all dependencies
useEffect(() => {
  fetchUser(userId).then(setUser);
}, [userId]);

// WRONG: Object/array in dependencies
function Component({ config }: { config: Config }): JSX.Element {
  useEffect(() => {
    applyConfig(config);
  }, [config]);  // New object on every render - infinite loop

  return <div />;
}

// CORRECT: Use primitive dependencies or useMemo
function Component({ config }: { config: Config }): JSX.Element {
  const configJson = JSON.stringify(config);

  useEffect(() => {
    applyConfig(JSON.parse(configJson));
  }, [configJson]);  // String comparison works

  return <div />;
}
```

### Infinite Render Loops

```typescript
// WRONG: State update in render
function Component(): JSX.Element {
  const [count, setCount] = useState(0);

  setCount(count + 1);  // NEVER - infinite loop

  return <div>{count}</div>;
}

// WRONG: useEffect without dependencies
function Component(): JSX.Element {
  const [data, setData] = useState(null);

  useEffect(() => {
    setData({ value: Math.random() });
  });  // No deps array - runs after every render

  return <div>{data?.value}</div>;
}

// CORRECT: Effect with proper dependencies
function Component(): JSX.Element {
  const [data, setData] = useState(null);

  useEffect(() => {
    setData({ value: Math.random() });
  }, []);  // Empty array - runs once

  return <div>{data?.value}</div>;
}
```

### Cleanup Function Gotchas

```typescript
// WRONG: Missing cleanup
function Component(): JSX.Element {
  useEffect(() => {
    const subscription = api.subscribe(handleData);
    // Missing cleanup - memory leak
  }, []);

  return <div />;
}

// CORRECT: Proper cleanup
function Component(): JSX.Element {
  useEffect(() => {
    const subscription = api.subscribe(handleData);

    return () => {
      subscription.unsubscribe();
    };
  }, []);

  return <div />;
}

// WRONG: Async cleanup
function Component(): JSX.Element {
  useEffect(() => {
    return async () => {  // WRONG - cleanup can't be async
      await api.cleanup();
    };
  }, []);

  return <div />;
}

// CORRECT: Handle async in cleanup
function Component(): JSX.Element {
  useEffect(() => {
    return () => {
      void api.cleanup();  // Fire and forget
      // Or use an IIFE if you need to handle the promise
      (async () => {
        await api.cleanup();
      })();
    };
  }, []);

  return <div />;
}
```

### Key Prop Issues

```typescript
// WRONG: Using index as key
function List({ items }: { items: string[] }): JSX.Element {
  return (
    <ul>
      {items.map((item, index) => (
        <li key={index}>{item}</li>  // Breaks on reorder
      ))}
    </ul>
  );
}

// CORRECT: Use stable identifier
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.id}>{item.name}</li>
      ))}
    </ul>
  );
}

// WRONG: Non-unique keys
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.category}>{item.name}</li>  // Multiple items per category
      ))}
    </ul>
  );
}

// CORRECT: Compound key if no unique ID
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item, index) => (
        <li key={`${item.category}-${index}`}>{item.name}</li>
      ))}
    </ul>
  );
}
```
```

### R-golem.md
```markdown
---
paths:
  - "**/inst/golem-config.yml"
  - "**/R/app_ui.R"
  - "**/R/app_server.R"
  - "**/dev/*.R"
---

# Golem Development Standards

This document provides guidelines for building production-grade Shiny apps with {golem}. These conventions extend the core R and Shiny rules, enforcing the "App-as-a-Package" philosophy.

**Prerequisite**: This file assumes `~/.claude/rules/R.md` and `~/.claude/rules/R-shiny.md` are also loaded.

---

## 1. Project Architecture (App-as-Package)

The application MUST be structured as an R package.

### Directory Structure

```
myapp/
├── DESCRIPTION           # Package metadata (REQUIRED)
├── NAMESPACE             # Generated by roxygen2
├── R/                    # ALL functional R code (FLAT - no subdirs!)
│   ├── app_config.R      # App configuration
│   ├── app_server.R      # Main server
│   ├── app_ui.R          # Main UI + golem_add_external_resources()
│   ├── run_app.R         # App launcher
│   ├── mod_*.R           # Modules
│   ├── fct_*.R           # Business logic functions
│   └── utils_*.R         # Utility functions
├── inst/
│   ├── app/www/          # Static assets (CSS, JS, images)
│   └── golem-config.yml  # Environment configuration
├── dev/                  # Development scripts
│   ├── 01_start.R        # Initial setup
│   ├── 02_dev.R          # Development utilities
│   ├── 03_deploy.R       # Deployment scripts
│   └── run_dev.R         # Development runner
├── tests/
│   └── testthat/
├── man/                  # Generated documentation
└── *.Rproj
```

### CRITICAL: Flat R/ Directory

**NO subdirectories are allowed inside `R/`**. This is an R package constraint.

**Organization Strategy**: Use file naming prefixes to replace folders:

| Old Pattern | Golem Pattern |
|-------------|---------------|
| `R/shiny/modules/proteomics/qc.R` | `R/mod_proteomics_qc.R` |
| `R/shiny/utils/helpers.R` | `R/utils_helpers.R` |
| `R/analysis/statistics.R` | `R/fct_statistics.R` |

### File Naming Conventions

| Prefix | Purpose | Example |
|--------|---------|---------|
| `mod_` | Shiny modules (UI + Server) | `mod_data_upload.R` |
| `fct_` | Business logic functions | `fct_data_processing.R` |
| `utils_` | Utility/helper functions | `utils_validation.R` |
| `app_` | Core app files | `app_ui.R`, `app_server.R` |

---

## 2. Module Creation

### Using golem::add_module()

**ALWAYS** use golem's module generator:

```r
# In dev/02_dev.R or console
golem::add_module(name = "data_upload", with_test = TRUE)
```

This creates:
- `R/mod_data_upload.R` with UI and Server functions
- `tests/testthat/test-mod_data_upload.R`

### Module Template

```r
#' data_upload UI Function
#'
#' @description A shiny Module for uploading data files.
#'
#' @param id Internal parameter for shiny.
#'
#' @noRd
#' @importFrom shiny NS tagList
mod_data_upload_ui <- function(id) {
    ns <- shiny::NS(id)
    shiny::tagList(
        shiny::wellPanel(
            # UI components
        )
    )
}

#' data_upload Server Function
#'
#' @param id Internal parameter for shiny.
#' @param workflow_data Reactive values object for shared state.
#'
#' @noRd
mod_data_upload_server <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns

        # 1. Access shared state
        # data <- workflow_data$data_raw

        # 2. Core logic (use S4 methods where appropriate)

        # 3. Update shared state
        # workflow_data$data_raw <- processed_data
    })
}
```

### Integration with workflow_data

Pass the reactive values object to all modules:

```r
# In app_server.R
app_server <- function(input, output, session) {
    # Initialize shared state
    workflow_data <- shiny::reactiveValues(
        data_raw = NULL
        , data_processed = NULL
        , config = list()
    )

    # Call modules with workflow_data
    mod_data_upload_server("data_upload", workflow_data)
    mod_analysis_server("analysis", workflow_data)
}
```

---

## 3. Resource Management

### File Access

**NEVER** use relative paths:

```r
# BAD
read.csv("./data/file.csv")
read.csv("data/file.csv")

# GOOD - Use app_sys() wrapper for system.file()
read.csv(app_sys("extdata", "file.csv"))

# For www assets
css_path <- app_sys("app", "www", "style.css")
```

### Adding External Assets

**Do NOT** write `<script>` or `<link>` tags manually:

```r
# BAD - Manual tag in UI
shiny::tags$head(
    shiny::tags$link(rel = "stylesheet", href = "custom.css")
)

# GOOD - Use golem helpers in dev/02_dev.R
golem::add_css_file("custom")     # Creates inst/app/www/custom.css
golem::add_js_file("handlers")    # Creates inst/app/www/handlers.js
golem::add_sass_file("theme")     # Creates inst/app/www/theme.sass
```

### Bundle Resources

Ensure `golem_add_external_resources()` in `R/app_ui.R` calls `bundle_resources()`:

```r
#' Add external resources to the app
#'
#' @noRd
golem_add_external_resources <- function() {
    golem::add_resource_path("www", app_sys("app/www"))

    shiny::tags$head(
        golem::bundle_resources(
            path = app_sys("app/www")
            , app_title = "My App"
        )
        # Add custom meta tags here if needed
    )
}
```

---

## 4. Configuration

### golem-config.yml

Store environment-specific configuration in `inst/golem-config.yml`:

```yaml
default:
  golem_name: myapp
  golem_version: 0.1.0
  app_prod: no

production:
  app_prod: yes
  db_host: "prod-db.example.com"

staging:
  app_prod: no
  db_host: "staging-db.example.com"
```

### Accessing Configuration

```r
# Get config value (uses GOLEM_CONFIG_ACTIVE env var)
db_host <- golem::get_golem_config("db_host")

# Check if production mode
if (golem::get_golem_config("app_prod")) {
    # Production-specific code
}

# Set active config via environment variable
Sys.setenv(GOLEM_CONFIG_ACTIVE = "production")
```

### Secrets Management

**NEVER** store secrets in golem-config.yml. Use environment variables:

```r
# In .Renviron (not committed to git)
DB_PASSWORD=secret123

# In R code
db_password <- Sys.getenv("DB_PASSWORD")
```

### Global Variables

**STRICTLY FORBIDDEN**: Do not create objects in the global environment.

```r
# BAD - Global variable in R/ file
my_global_data <- read.csv("data.csv")

# GOOD - Use workflow_data or golem-config
# Access via get_golem_config() or pass through workflow_data
```

---

## 5. Development Workflow

### Package Dependencies

**NEVER** use `library()` inside app functions:

```r
# BAD
my_function <- function(data) {
    library(dplyr)
    data |> filter(x > 0)
}

# GOOD - Use explicit namespaces
my_function <- function(data) {
    data |> dplyr::filter(x > 0)
}

# GOOD - Use @importFrom in roxygen
#' @importFrom dplyr filter mutate
my_function <- function(data) {
    data |> filter(x > 0) |> mutate(y = x * 2)
}
```

Add dependencies properly:

```r
# In dev/02_dev.R or console
usethis::use_package("dplyr")
usethis::use_package("ggplot2", type = "Suggests")  # For optional deps
```

### Documentation

```r
# Internal functions (not exported)
#' Process data helper
#'
#' @param data Input data frame
#' @return Processed data frame
#'
#' @noRd
process_helper <- function(data) {
    # ...
}

# Refresh namespace and reload app
golem::document_and_reload()
```

### Development Runner

Use `dev/run_dev.R` for development:

```r
# dev/run_dev.R
# Set options here
options(shiny.reactlog = TRUE)
options(golem.app.prod = FALSE)

# Run the app
run_app()
```

---

## 6. Testing

### Test Organization

```
tests/
└── testthat/
    ├── test-mod_data_upload.R    # Module tests
    ├── test-fct_processing.R     # Function tests
    ├── test-app.R                # Integration tests
    └── _snaps/                   # Snapshot test files
```

### Testing Modules with testServer

```r
test_that("data_upload module processes files correctly", {
    # Create mock workflow_data
    mock_workflow <- shiny::reactiveValues(data_raw = NULL)

    shiny::testServer(
        mod_data_upload_server
        , args = list(workflow_data = mock_workflow)
        , {
            # Simulate file upload
            session$setInputs(file = list(datapath = test_path("fixtures/test.csv")))
            session$setInputs(upload = 1)

            # Check results
            expect_s4_class(mock_workflow$data_raw, "MyDataClass")
            expect_equal(nrow(mock_workflow$data_raw), 100)
        }
    )
})
```

### Integration Testing with shinytest2

```r
test_that("Full app workflow works", {
    skip_on_cran()

    app <- shinytest2::AppDriver$new(
        app_dir = system.file(package = "myapp")
        , name = "integration_test"
    )

    # Test workflow
    app$upload_file(file = test_path("fixtures/test.csv"))
    app$click("process")
    app$wait_for_idle()

    # Verify results
    app$expect_values()

    app$stop()
})
```

### Test Fixtures

Store test data in `tests/testthat/fixtures/`:

```r
# Access in tests
test_file <- testthat::test_path("fixtures", "test_data.csv")
```

---

## 7. Deployment

### Docker

Generate Dockerfile with golem:

```r
# In dev/03_deploy.R
golem::add_dockerfile()                    # Standard Dockerfile
golem::add_dockerfile_with_renv()          # With renv for reproducibility
golem::add_dockerfile_with_renv_shinyproxy()  # For ShinyProxy
```

### GitHub Actions

Add CI/CD workflow:

```r
# In dev/02_dev.R
usethis::use_github_action_check_standard()
```

Example workflow for Shiny apps (`.github/workflows/check.yml`):

```yaml
name: R-CMD-check

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  R-CMD-check:
    runs-on: ubuntu-latest
    env:
      GITHUB_PAT: ${{ secrets.GITHUB_TOKEN }}
      R_KEEP_PKG_SOURCE: yes

    steps:
      - uses: actions/checkout@v4

      - uses: r-lib/actions/setup-r@v2
        with:
          use-public-rspm: true

      - uses: r-lib/actions/setup-r-dependencies@v2
        with:
          extra-packages: any::rcmdcheck
          needs: check

      - uses: r-lib/actions/check-r-package@v2

  shinytest2:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: r-lib/actions/setup-r@v2
      - uses: r-lib/actions/setup-r-dependencies@v2
      - name: Run shinytest2
        run: |
          shinytest2::test_app()
        shell: Rscript {0}
```

### dev/03_deploy.R Patterns

```r
# Deploy to shinyapps.io
rsconnect::deployApp(
    appDir = "."
    , appName = golem::get_golem_name()
    , account = "your_account"
)

# Deploy to Posit Connect
rsconnect::deployApp(
    appDir = "."
    , server = "connect.example.com"
)

# Build for Docker
golem::add_dockerfile_with_renv()
# Then: docker build -t myapp .
```

---

## 8. Adaptation Reference

| Traditional Shiny | Golem Implementation |
|-------------------|---------------------|
| `R/shiny/modules/{type}/` | `R/mod_{type}_{name}.R` (flat structure) |
| `workflow_data` global | Passed as argument: `mod_server(id, workflow_data)` |
| Local CSV/assets | Move to `inst/extdata/` or `inst/app/www/` |
| `source("file.R")` | Remove. Functions auto-loaded by package namespace. |
| `library(pkg)` in code | Use `pkg::function()` or `@importFrom` |
| Relative file paths | Use `app_sys()` wrapper |
| Manual CSS/JS includes | Use `golem::add_css_file()`, `add_js_file()` |

---

## Code Style (Strict)

In addition to core R style rules:

- **Assignment**: `<-` only (never `=` for assignment)
- **Commas**: Leading commas in multi-line lists/arguments
- **Namespaces**: Explicit prefixes mandatory (`shiny::`, `dplyr::`)
- **Documentation**: All modules use `#' @noRd` for internal functions

```r
# Correct style example
mod_example_server <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns

        result <- shiny::reactive({
            shiny::req(workflow_data$data_raw)

            workflow_data$data_raw |>
                dplyr::filter(
                    value > 0
                    , category %in% input$selected_categories
                ) |>
                dplyr::summarise(
                    mean_value = mean(value, na.rm = TRUE)
                    , n = dplyr::n()
                )
        })

        return(result)
    })
}
```

---

## Before Committing Checklist

- [ ] `R/` directory is flat (no subdirectories)
- [ ] All modules created with `golem::add_module()`
- [ ] No `library()` calls in R/ files
- [ ] No relative file paths (use `app_sys()`)
- [ ] Dependencies added with `usethis::use_package()`
- [ ] `golem::document_and_reload()` runs without errors
- [ ] `devtools::check()` passes
- [ ] Tests exist for modules and functions
- [ ] Secrets in environment variables (not golem-config.yml)

---

**Remember:** Golem enforces package structure discipline. This pays dividends in maintainability, testability, and deployment flexibility.
```

### R.md
```markdown
---
paths:
  - "**/*.R"
  - "**/*.Rmd"
  - "**/*.qmd"
---

# Agent Guidelines for R Code Quality

This document provides guidelines for maintaining high-quality R code. These rules MUST be followed by all AI coding agents and contributors.

## Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:
- Maximizing algorithmic big-O efficiency for memory and runtime
- Using parallelization and vectorization where appropriate
- Following proper style conventions (tidyverse style guide, DRY principle)
- No extra code beyond what is absolutely necessary to solve the problem the user provides (i.e., no technical debt)

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

---

## R Version and Modern Features

**MUST** target R 4.1+ for new projects; R 4.3+ preferred.

### R 4.1+ Features to Adopt
- **Native pipe `|>`** instead of magrittr `%>%`
- **Lambda syntax** `\(x) x + 1` instead of `function(x) x + 1`
- **`...names()`** for extracting names from `...` arguments

### R 4.2+ Features to Adopt
- **Improved placeholder** `_` in pipes (with named arguments)
- **`chooseOpsMethod()`** for better S4/S3 method dispatch

### R 4.3+ Features to Adopt
- **`toTitleCase()` improvements** for string handling
- **Native R serialization v3** for better cross-platform compatibility

**Example - Modern R Syntax:**
```r
# PREFERRED: Native pipe with lambda
processed_data <- raw_data |>
    filter(\(x) !is.na(x$value)) |>
    mutate(log_value = log2(value + 1))

# PREFERRED: Lambda in apply functions
results <- lapply(data_list, \(df) {
    df |>
        filter(q_value < 0.05) |>
        arrange(desc(log2fc))
})
```

---

## Preferred Tools

### Package Management
- **MUST** use `renv` for dependency management and reproducibility
- **MUST** commit `renv.lock` to version control
- **MUST** run `renv::snapshot()` after adding/updating packages
- **MUST** use `renv::restore()` to recreate environments
```r
renv::init()            # Initialize renv in project
renv::install("dplyr")  # Install packages
renv::snapshot()        # Lock dependencies
renv::restore()         # Restore from lockfile
```

### Development Tools
- **styler** for code formatting (`styler::style_file()`, `styler::style_dir()`)
- **lintr** for static code analysis
- **roxygen2** for documentation (mandatory for all functions)
- **testthat** for testing with **testthat::test_dir()** for batch execution
- **logger** for structured logging (replaces `print`/`message`)
- **profvis** for profiling before optimization
- **covr** for test coverage analysis

### Data Manipulation Tools
- **dplyr/tidyr** for tidy data manipulation
- **purrr** for functional programming on lists/vectors
- **data.table** for performance-critical operations on very large tables

### Tidy Evaluation with String Variables (CRITICAL)

`{{ }}` is for **symbols passed as function arguments**, NOT string variables. For strings, use:

| Verb | String Variable Pattern |
|------|------------------------|
| `pull()` | `df[[var]]` (base R only) |
| `rename()` | `rename(new = !!rlang::sym(var))` |
| `group_by()` | `group_by(across(all_of(var)))` |
| `filter()`/`mutate()` | `.data[[var]]` works |
| `select()` | `all_of(var)` |

**MUST** namespace-prefix tidyselect helpers in packages: `dplyr::all_of()`, not `all_of()`.

**Default to base R** (`df[[col]]`) for simple extraction—no tidy eval complexity.

### S4 Method Calls (CRITICAL)

**MUST** verify parameter names in `R/allGenerics.R` before calling S4 methods. Common traps:
- Full descriptive names: `ruv_number_k` not `k`
- British spelling: `normalisation_method` not `normalization_method`
- Prefixed names: `itsd_aggregation` not `aggregation`

### UI-to-Function Mapping

UI inputs may control **one** parameter while others need hardcoded values. **MUST** read function docs to identify which parameters are type selectors (hardcode) vs user choices (wire to UI).

### Proteomics/Bioinformatics Tools
- **SummarizedExperiment** as base container for omics data
- **MultiAssayExperiment** for multi-omics integration
- **Biobase** for legacy ExpressionSet compatibility
- **arrow** for out-of-memory data handling

---

## Code Style and Formatting

- **MUST** use meaningful, descriptive variable and function names
- **MUST** follow tidyverse style guide strictly
- **MUST** use 4 spaces for indentation (never tabs)
- **MUST** use `<-` for assignment, `=` only for function arguments
- **NEVER** use emoji or unicode emulating emoji except in tests
- Limit line length to 80-100 characters

### Naming Conventions
| Element | Convention | Example |
|---------|------------|---------|
| Functions | camelCase (verbs) | `normalizeProteomicsData()` |
| Variables | snake_case (nouns) | `sample_annotation` |
| Constants | UPPER_SNAKE_CASE | `MAX_Q_VALUE` |
| S4 Classes | PascalCase | `ProteomicsData` |
| R6 Classes | PascalCase | `ExperimentManager` |

### Leading Comma Convention
**MUST** place commas at the beginning of new lines in multi-line constructs:
```r
# CORRECT: Leading commas
my_list <- list(
    item1 = "apple"
    , item2 = "banana"
    , item3 = "cherry"
)

df |>
    select(
        sample_id
        , protein_id
        , intensity
    ) |>
    filter(intensity > 0)
```

---

## Documentation

### roxygen2 Requirements
- **MUST** include roxygen2 documentation for ALL functions (exported and internal)
- **MUST** document `@param`, `@return`, description for every function
- **MUST** use `@export`, `@examples`, `@importFrom` appropriately

### Critical roxygen2 Rules

**NEVER add roxygen comments to `allGenerics.R`:**
```r
# BAD - Will cause parsing errors
#' @export
setGeneric("normalizeData", function(object, ...) standardGeneric("normalizeData"))

# GOOD - Only bare setGeneric calls
setGeneric("normalizeData", function(object, ...) standardGeneric("normalizeData"))
```

**Tag order (recommended):**
1. `@title` - Brief title
2. `@name` - Explicit topic name (required for S4 methods)
3. Description paragraph(s)
4. `@param` - Parameter documentation
5. `@return` - Return value documentation
6. `@importFrom` - Package imports
7. `@export` - Export directive
8. `@examples` - Usage examples

**Tag Conflicts - `@describeIn` and `@name` are mutually exclusive:**
```r
# BAD - Will cause error: "@describeIn can not be used with @name"
#' @describeIn plotPca Method for MetaboliteAssayData
#' @name plotPca,MetaboliteAssayData-method
#' @export
setMethod(f = "plotPca", ...)

# GOOD - Use @name and @title instead
#' @title Plot PCA for MetaboliteAssayData
#' @name plotPca,MetaboliteAssayData-method
#' @export
setMethod(f = "plotPca", ...)
```

**Inheritance Tags - Referenced topics MUST exist:**
- `@inheritParams` and `@inheritDoc` require the referenced topic to exist
- Verify referenced function/topic is documented before using inheritance
- Consider explicit parameter documentation if reference is unclear

**Duplicate Tags:**
- **Only one `@export` per function/method** - multiple tags cause issues

**S4 Method Documentation:**
```r
#' @title Normalize Between Samples for ProteomicsData
#' @name normaliseBetweenSamples,ProteomicsData-method
#' @param theObject Object of class ProteomicsData
#' @param normalisation_method Method to use for normalization
#' @return Modified ProteomicsData object with normalized values
#' @importFrom limma normalizeCyclicLoess
#' @export
setMethod(
    f = "normaliseBetweenSamples"
    , signature = "ProteomicsData"
    , definition = function(theObject, normalisation_method = NULL) {
        # Implementation
    }
)
```

### Code Commenting Philosophy
- **Explain the "Why", not the "What":** Focus on rationale and scientific reasoning
- **Target audience:** Assume biologist/analyst reader
- Use section dividers for complex functions: `# --- Section Name ---`
- **NEVER** commit commented-out code; delete it

```r
# --- Filter Low Variance Features ---
# Remove features with low variance across samples, as they are less likely
# to be informative for downstream differential analysis or clustering.
# Using IQR as a robust variance measure resistant to outliers.
low_variance_threshold <- 0.1
features_to_keep <- calculateFeatureIQR(data_matrix) > low_variance_threshold
filtered_matrix <- data_matrix[features_to_keep, ]
```

---

## S4 Object-Oriented Programming

### Core S4 Principles
- **MUST** use S4 classes inheriting from `SummarizedExperiment` for omics data
- **MUST** use accessor methods instead of direct slot access (`@`)
- **MUST** implement `setValidity` for all custom classes
- **MUST** use dedicated constructor functions with validation

### S4 Class Hierarchy Pattern
```r
# --- Base Class Definition ---
setClass(
    "QuantitativeOmicsData"
    , contains = "SummarizedExperiment"
    , slots = c(
        processing_log = "list"
        , analysis_params = "list"
    )
)

setValidity("QuantitativeOmicsData", function(object) {
    errors <- character()
    if (!is.list(object@processing_log)) {
        errors <- c(errors, "processing_log must be a list")
    }
    if (length(errors) == 0) TRUE else errors
})

# --- Derived Class ---
setClass(
    "ProteomicsData"
    , contains = "QuantitativeOmicsData"
    , slots = c(
        protein_groups = "character"
        , quantification_method = "character"
    )
)

# --- Constructor Function ---
createProteomicsData <- function(
    assay_matrix
    , col_data
    , row_data
    , quantification_method = "LFQ"
) {
    stopifnot(
        is.matrix(assay_matrix)
        , is.data.frame(col_data) || is(col_data, "DataFrame")
        , nrow(col_data) == ncol(assay_matrix)
    )

    se <- SummarizedExperiment(
        assays = list(intensity = assay_matrix)
        , colData = col_data
        , rowData = row_data
    )

    new(
        "ProteomicsData"
        , se
        , processing_log = list()
        , analysis_params = list()
        , protein_groups = rownames(assay_matrix)
        , quantification_method = quantification_method
    )
}
```

### S4 Generics and Methods Pattern
```r
# In allGenerics.R - NO roxygen comments
setGeneric("normalizeData", function(object, method, ...) {
    standardGeneric("normalizeData")
})

# In methods-normalize.R - Full documentation
#' @title Normalize Data for ProteomicsData
#' @name normalizeData,ProteomicsData-method
#' @param object ProteomicsData object
#' @param method Normalization method: "median", "quantile", "vsn"
#' @return Normalized ProteomicsData object
#' @export
setMethod(
    f = "normalizeData"
    , signature = "ProteomicsData"
    , definition = function(object, method = "median", ...) {
        # Implementation specific to proteomics
        assay_data <- assay(object, "intensity")
        normalized <- switch(
            method
            , median = .normalizeMedian(assay_data)
            , quantile = .normalizeQuantile(assay_data)
            , vsn = .normalizeVsn(assay_data)
            , stop("Unknown method: ", method)
        )
        assay(object, "intensity") <- normalized
        object@processing_log <- c(
            object@processing_log
            , list(list(step = "normalize", method = method, time = Sys.time()))
        )
        object
    }
)
```

### S4 Type Checking (CRITICAL)

When checking if an object is an S4 object, **MUST** use the correct functions:

```r
# CORRECT: Use isS4() to check if something is an S4 object
if (!isS4(my_object)) {
    stop("Expected an S4 object")
}

# CORRECT: Use is() or inherits() to check specific class
if (!methods::is(my_object, "ProteomicsData")) {
    stop("Expected a ProteomicsData object")
}

# WRONG: "S4" is NOT a class name - this will ALWAYS return FALSE
if (!methods::is(my_object, "S4")) {  # BUG! "S4" is not a class
    stop("This check always fails for valid S4 objects")
}
```

**Key distinction:**
- `isS4(x)` - Checks the object's **type system** (is it an S4 object?)
- `methods::is(x, "ClassName")` - Checks **class inheritance** (does it inherit from ClassName?)
- `inherits(x, "ClassName")` - Also checks class inheritance (works for S3 and S4)

The string `"S4"` is not a class that objects inherit from—it's a type system designation. S4 objects inherit from their specific classes (e.g., `"ProteomicsData"`, `"SummarizedExperiment"`), not from `"S4"`.

---

## R6 State Management

### When to Use R6
- **Use R6 for:** Mutable state, complex workflows, caching, connection management
- **Use S4 for:** Data containers, method dispatch, Bioconductor integration

### R6 Pattern for Large Data Workflows
```r
#' @title Experiment Manager
#' @description R6 class for managing proteomics analysis workflows with caching
#' @export
ExperimentManager <- R6::R6Class(
    "ExperimentManager"
    , public = list(
        #' @field data ProteomicsData object
        data = NULL

        #' @field cache List of cached computations
        , cache = NULL

        #' @description Initialize manager with data
        #' @param proteomics_data ProteomicsData object
        , initialize = function(proteomics_data) {
            stopifnot(is(proteomics_data, "ProteomicsData"))
            self$data <- proteomics_data
            self$cache <- list()
            private$.log_step("initialized")
        }

        #' @description Run CPU-intensive operation with caching
        #' @param operation Character name of operation
        #' @param fn Function to execute
        #' @param ... Arguments passed to fn
        , run_cached = function(operation, fn, ...) {
            cache_key <- digest::digest(list(operation, ...))
            if (!is.null(self$cache[[cache_key]])) {
                logger::log_info("Cache hit for {operation}")
                return(self$cache[[cache_key]])
            }
            logger::log_info("Computing {operation}")
            result <- fn(self$data, ...)
            self$cache[[cache_key]] <- result
            private$.log_step(operation)
            result
        }

        #' @description Clear cache
        , clear_cache = function() {
            self$cache <- list()
            invisible(self)
        }
    )
    , private = list(
        .log = list()

        , .log_step = function(step) {
            private$.log <- c(
                private$.log
                , list(list(step = step, time = Sys.time()))
            )
        }
    )
)
```

---

## Concurrency and Parallelization

### Package Selection
| Workload | Solution |
|----------|----------|
| Independent list operations | `future.apply::future_lapply()` |
| Tidyverse-style parallel map | `furrr::future_map()` |
| Bioconductor parallel | `BiocParallel::bplapply()` |
| CPU-bound matrix ops | `RcppParallel` or `data.table` |
| Very large data chunks | `future` with `multisession` plan |

### future/future.apply Setup
```r
library(future)
library(future.apply)

# Cross-platform parallel backend (MUST use multisession on Windows)
plan(multisession, workers = parallel::detectCores() - 1)

# IMPORTANT: Always set seed for reproducibility
results <- future_lapply(
    data_list
    , process_fn
    , future.seed = TRUE  # MUST set for reproducibility
)

# Cleanup when done (good practice)
plan(sequential)
```

### furrr for Tidyverse Workflows
```r
library(furrr)

# Set plan BEFORE using furrr functions
plan(multisession, workers = 4)

# Parallel map with progress bar
results <- future_map(
    data_list
    , slow_fn
    , .progress = TRUE
    , .options = furrr_options(seed = TRUE)
)

# Parallel map with data frame output
results_df <- future_map_dfr(
    split_data
    , \(chunk) {
        chunk |>
            filter(q_value < 0.05) |>
            summarise(mean_fc = mean(log2fc))
    }
    , .id = "group"
    , .options = furrr_options(seed = TRUE)
)
```

### Wrapper Functions for CPU-Intensive S4 Operations
**MUST** create simple wrappers for parallelizing operations on large S4 objects:

```r
#' @title Parallel Wrapper for CPU-Intensive Operations
#' @description Execute function in parallel across data chunks
#' @param object S4 object (ProteomicsData, etc.)
#' @param fn Function to apply to each chunk
#' @param chunk_by Column in rowData to split by (e.g., "protein_group")
#' @param workers Number of parallel workers
#' @param ... Additional arguments passed to fn
#' @return Combined results
#' @export
parallelApplyS4 <- function(
    object
    , fn
    , chunk_by = NULL
    , workers = parallel::detectCores() - 1
    , ...
) {
    stopifnot(is(object, "SummarizedExperiment"))

    # Set up parallel backend
    oplan <- plan(multisession, workers = workers)
    on.exit(plan(oplan), add = TRUE)

    # Split data into chunks
    if (is.null(chunk_by)) {
        # Split by row indices
        n_rows <- nrow(object)
        chunk_size <- ceiling(n_rows / workers)
        indices <- split(seq_len(n_rows), ceiling(seq_len(n_rows) / chunk_size))
        chunks <- lapply(indices, \(idx) object[idx, ])
    } else {
        # Split by grouping variable
        groups <- rowData(object)[[chunk_by]]
        chunks <- lapply(unique(groups), \(g) object[groups == g, ])
    }

    # Execute in parallel with progress
    logger::log_info("Processing {length(chunks)} chunks across {workers} workers")

    results <- future_lapply(
        chunks
        , fn
        , ...
        , future.seed = TRUE
    )

    results
}

# Usage example for proteomics normalization
normalizeChunkedProteomics <- function(prot_data, method = "vsn") {
    # Wrapper for parallel VSN normalization on large datasets
    results <- parallelApplyS4(
        prot_data
        , fn = \(chunk) {
            # CPU-intensive normalization per chunk
            assay_data <- assay(chunk, "intensity")
            normalized <- vsn::justvsn(assay_data)
            assay(chunk, "intensity") <- normalized
            chunk
        }
        , workers = 4
    )

    # Combine results back into single object
    do.call(rbind, results)
}
```

### Parallel Pattern for Large Matrix Operations
```r
#' @title Parallel Correlation Matrix for Large Proteomics Data
#' @param prot_data ProteomicsData object
#' @param method Correlation method
#' @return Correlation matrix
#' @export
parallelCorrelation <- function(prot_data, method = "pearson") {
    mat <- assay(prot_data, "intensity")
    n_samples <- ncol(mat)

    # Only parallelize if matrix is large enough
    if (n_samples < 50) {
        return(cor(mat, method = method, use = "pairwise.complete.obs"))
    }

    plan(multisession, workers = parallel::detectCores() - 1)
    on.exit(plan(sequential), add = TRUE)

    # Compute correlations in parallel by column blocks
    col_pairs <- combn(seq_len(n_samples), 2, simplify = FALSE)

    cor_values <- future_sapply(
        col_pairs
        , \(pair) cor(mat[, pair[1]], mat[, pair[2]], method = method, use = "complete.obs")
        , future.seed = TRUE
    )

    # Reconstruct symmetric matrix
    cor_mat <- matrix(1, nrow = n_samples, ncol = n_samples)
    for (i in seq_along(col_pairs)) {
        pair <- col_pairs[[i]]
        cor_mat[pair[1], pair[2]] <- cor_values[i]
        cor_mat[pair[2], pair[1]] <- cor_values[i]
    }

    dimnames(cor_mat) <- list(colnames(mat), colnames(mat))
    cor_mat
}
```

### BiocParallel for Bioconductor Workflows
```r
library(BiocParallel)

# Register parallel backend
register(MulticoreParam(workers = 4))  # Unix/Mac
# register(SnowParam(workers = 4))     # Windows

# Use with Bioconductor functions that support BPPARAM
results <- bplapply(
    split_data
    , processChunk
    , BPPARAM = MulticoreParam(workers = 4)
)
```

### Memory-Conscious Parallelization
```r
#' @title Memory-Safe Parallel Processing for Large S4 Objects
#' @description Process large objects in memory-efficient chunks
processLargeObject <- function(object, fn, chunk_size = 1000) {
    n_features <- nrow(object)
    n_chunks <- ceiling(n_features / chunk_size)

    # Use sequential futures to avoid memory duplication
    plan(sequential)

    results <- vector("list", n_chunks)

    for (i in seq_len(n_chunks)) {
        start_idx <- (i - 1) * chunk_size + 1
        end_idx <- min(i * chunk_size, n_features)

        # Process chunk
        chunk <- object[start_idx:end_idx, ]
        results[[i]] <- fn(chunk)

        # Force garbage collection between chunks
        rm(chunk)
        gc()

        logger::log_debug("Processed chunk {i}/{n_chunks}")
    }

    do.call(rbind, results)
}
```

---

## Error Handling

- **NEVER** silently swallow exceptions without logging
- **MUST** use `stopifnot()` for input validation
- **MUST** use `tryCatch()` with specific error handling
- **MUST** log errors with `logger::log_error()`

### Error Handling Pattern
```r
processData <- function(object, method) {
    # Input validation
    stopifnot(
        is(object, "ProteomicsData")
        , is.character(method)
        , length(method) == 1
    )

    tryCatch(
        {
            result <- performAnalysis(object, method)
            logger::log_info("Analysis completed successfully")
            result
        }
        , error = function(e) {
            # NEVER use {} interpolation in logger inside tryCatch
            logger::log_error(paste("Analysis failed:", e$message))
            stop(e)
        }
        , warning = function(w) {
            logger::log_warn(paste("Warning during analysis:", w$message))
            invokeRestart("muffleWarning")
        }
    )
}
```

---

## Function Design

- **MUST** keep functions focused on a single responsibility
- **MUST** keep functions under 50-75 lines
- **NEVER** use mutable default arguments
- Prefer pure functions (no side effects)
- Return early to reduce nesting
- Limit parameters to 5-7; use config objects for more

```r
# GOOD: Early return pattern
validateInput <- function(data, threshold) {
    if (is.null(data)) {
        return(NULL)
    }
    if (!is.numeric(threshold)) {
        stop("threshold must be numeric")
    }
    if (threshold < 0 || threshold > 1) {
        stop("threshold must be between 0 and 1")
    }

    # Main logic only reached with valid inputs
    data[data > threshold]
}
```

---

## Testing (testthat)

### Requirements
- **MUST** use testthat for all testing
- **MUST** create test fixtures with `set.seed()` for reproducibility
- **MUST** test S4 class validity and method dispatch
- **NEVER** delete test files or fixtures

### Test Structure
```r
# tests/testthat/test-normalize.R
test_that("normalizeData median method works correctly", {
    # Arrange
    set.seed(42)
    test_data <- createTestProteomicsData(n_proteins = 100, n_samples = 10)

    # Act
    result <- normalizeData(test_data, method = "median")

    # Assert
    expect_s4_class(result, "ProteomicsData")
    expect_equal(ncol(result), ncol(test_data))
    expect_true(all(!is.na(assay(result, "intensity"))))
})

test_that("normalizeData fails with invalid method", {
    test_data <- createTestProteomicsData()

    expect_error(
        normalizeData(test_data, method = "invalid")
        , regexp = "Unknown method"
    )
})
```

---

## Performance Optimization

### Profiling First
- **MUST** profile before optimizing with `profvis`
- **NEVER** optimize without profiling data
```r
profvis::profvis({
    result <- expensiveOperation(large_data)
})
```

### Optimization Techniques
- Use vectorized operations over loops
- Use matrices for numeric data (avoid data frames in hot paths)
- Use `data.table` for large table operations
- Use `memoise` for caching expensive pure functions
- Use `Rcpp` for critical bottlenecks

### Memory Management
- Remove large unused objects: `rm(large_obj); gc()`
- Monitor memory: `lobstr::obj_size()`, `pryr::mem_used()`
- Use `arrow` for larger-than-memory datasets
- Process in chunks for very large matrices

---

## Reproducibility

### Seed Management
- **MUST** use `set.seed()` before ALL stochastic operations
- Document seed values in analysis parameters
- Use `set.seed()` in test fixtures

### Dependency Management
- **MUST** use `renv` for all projects
- **MUST** commit `renv.lock` to version control
```r
renv::init()      # Initialize
renv::snapshot()  # Lock current state
renv::restore()   # Recreate environment
```

### Parameter Tracking
- Track all analysis parameters in config files or script headers
- Log parameters with results
- Use the `processing_log` slot in S4 objects

### Session Info
- **MUST** save session info with results for reproducibility
```r
# Save with results
session_info <- sessionInfo()
# Or more detailed
session_info <- devtools::session_info()
```

### Version Control
- **MUST** use Git for ALL code, scripts, `renv.lock`, config files
- **MUST** commit often with descriptive messages
- **NEVER** commit `.Rhistory`, `.RData`, or large data files

---

## MultiOmics-Specific Guidelines

### Interoperability
- Maintain consistent sample and feature identifiers across omics layers
- Use stable IDs (e.g., Ensembl gene IDs, UniProt accessions)
- Document ID sources and mappings
- Use standard `colData`/`rowData` column names where applicable

### Integration Tools
- Use `MultiAssayExperiment` for managing linked assays
- Consider `mixOmics` for advanced integration analysis
- Document data merging/joining strategies (inner/outer joins)
- Document rationale for normalization methods

### Missing Values
- **Understand missingness type:** MNAR (Missing Not At Random) vs MAR/MCAR
- Visualize patterns with `visdat`, `naniar`
- Choose and document appropriate imputation methods
- Consider type-specific defaults via S4 methods for `imputeMissingValues`
```r
# Visualize missing data patterns
visdat::vis_miss(assay_df)
naniar::gg_miss_upset(assay_df)
```

---

## Quality Control Standards

### Documentation Requirements
- **MUST** justify all filtering thresholds in comments/logs
- **MUST** document normalization choices and rationale

### Visualization
- Generate plots (PCA, density, boxplots, heatmaps) before/after each major QC step
- Use generic plotting functions with S4 dispatch (e.g., `plotPCA(object, ...)`)

### Impact Tracking
- Log features/samples removed at each step
- Track changes in data distribution
- Use the `processing_log` slot in S4 objects
```r
# Track processing step
object@processing_log <- c(
    object@processing_log
    , list(list(
        step = "filter_low_counts"
        , features_removed = sum(!keep_features)
        , threshold = min_count
        , time = Sys.time()
    ))
)
```

---

## Statistical Best Practices

### Multiple Testing Correction
- **MUST** correct p-values using `p.adjust()` when testing multiple hypotheses
- Prefer method="BH" (Benjamini-Hochberg) for FDR control
- Report BOTH raw and adjusted p-values
```r
results$p_adjusted <- p.adjust(results$p_value, method = "BH")
```

### Effect Sizes
- **MUST** report effect sizes alongside p-values
- Use log2 fold change for expression data
- Use Cohen's d for group comparisons
- Visualize with volcano plots
```r
# Volcano plot pattern
ggplot(results, aes(x = log2fc, y = -log10(p_adjusted))) +
    geom_point(aes(color = significant)) +
    geom_hline(yintercept = -log10(0.05), linetype = "dashed")
```

---

## External Resources and Caching

### API Requests
- **MUST** use `httr2` for robust HTTP requests
- Configure retries, error handling, user-agent, timeout
```r
library(httr2)

response <- request("https://api.example.com/data") |>
    req_retry(max_tries = 3, backoff = ~ 2) |>
    req_timeout(seconds = 30) |>
    req_user_agent("MyPackage/1.0") |>
    req_error(is_error = \(resp) resp_status(resp) >= 400) |>
    req_perform()
```

### Caching Strategies
- Use `memoise` for function-level caching of expensive operations
- Use RDS caching for intermediate results
- Provide cache invalidation mechanisms
```r
library(memoise)

# Memoize expensive function
fetchAnnotations <- memoise(function(ids) {
    # Expensive API call or computation
    biomaRt::getBM(...)
})

# Simple RDS caching pattern
getCachedResult <- function(cache_path, compute_fn, force_refresh = FALSE) {
    if (!force_refresh && file.exists(cache_path)) {
        return(readRDS(cache_path))
    }
    result <- compute_fn()
    saveRDS(result, cache_path)
    result
}
```

---

## Package and Dependency Management

### Loading Packages
- Use `pacman::p_load()` for convenient loading/installation
- Use `conflicted::conflict_prefer()` to resolve namespace conflicts
```r
# Convenient loading with auto-install
pacman::p_load(dplyr, tidyr, ggplot2, SummarizedExperiment)

# Explicit conflict resolution
library(conflicted)
conflict_prefer("filter", "dplyr")
conflict_prefer("select", "dplyr")
conflict_prefer("lag", "dplyr")
```

### Namespace Conflicts
- Be explicit with `package::function()` when conflicts exist
- Document known conflicts in project README
- Use `conflicted` package to force explicit resolution

---

## Security

- **NEVER** store secrets, API keys, or passwords in code
- **NEVER** use `eval(parse(text = user_input))`
- **NEVER** print or log URLs containing API keys
- **MUST** validate all external inputs
- **MUST** add `.env` and credentials files to `.gitignore`

---

## Shiny Application Development

**MUST** use explicit namespaces for all Shiny functions:
```r
# BAD - Will fail if packages are detached/reloaded
tabItem(tabName = "home", h3("Welcome"), br())

# GOOD - Always works
shinydashboard::tabItem(
    tabName = "home"
    , shiny::h3("Welcome")
    , shiny::br()
)
```

### Logger Bug in Reactive Contexts
**NEVER** use `{}` interpolation in logger calls inside error handlers or reactive contexts:
```r
# BAD - Will cause error
log_error("Error: {e$message}")

# GOOD - Safe in all contexts
log_error(paste("Error:", e$message))
```

---

## Logging

### logger Package Setup
- **MUST** use `logger` for structured logging instead of `print`/`message`
- Configure appropriate log levels, appenders, and layout
```r
library(logger)

# Configure logging
log_threshold(INFO)
log_appender(appender_tee(file = "analysis.log"))
log_layout(layout_glue_colors)

# Use appropriate levels
log_debug("Detailed debugging info")
log_info("Processing started")
log_warn("Missing values detected, using defaults")
log_error("Failed to load file")
log_fatal("Unrecoverable error, aborting")
```

### Log Levels
| Level | Use Case |
|-------|----------|
| DEBUG | Detailed tracing, variable values |
| INFO | Normal operation milestones |
| WARN | Unexpected but recoverable situations |
| ERROR | Failures that don't stop execution |
| FATAL | Unrecoverable errors |

### Logger Interpolation Bug (CRITICAL)
The `logger` package has a known issue with string interpolation in certain contexts:
- **NEVER** use `{}` interpolation in `tryCatch` error handlers
- **NEVER** use `{}` interpolation in Shiny reactive contexts
- **ALWAYS** use `paste()` or `sprintf()` in these contexts
```r
# Safe contexts for interpolation:
log_info("Processing {n_samples} samples")  # OK in regular functions

# Unsafe contexts - use paste():
tryCatch(
    { risky_operation() }
    , error = function(e) {
        log_error(paste("Failed:", e$message))  # MUST use paste()
    }
)
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| `df[,1]` (numeric indexing) | `df$col` or `df[["col"]]` |
| `attach()` / `detach()` | Use explicit references |
| `rm(list = ls())` | Never use; restart R session |
| `setwd()` | Use `here::here()` or relative paths |
| Bare `%>%` in packages | Use `\|>` or import from magrittr |
| `source()` for shared code | Create proper packages |
| Deep nesting (>3 levels) | Refactor into smaller functions |

---

## Project Structure

```
myproject/
├── R/
│   ├── allClasses.R          # S4 class definitions
│   ├── allGenerics.R         # setGeneric() calls ONLY
│   ├── methods-normalize.R   # Method implementations
│   └── utils.R               # Helper functions
├── tests/
│   └── testthat/
│       ├── helper-fixtures.R # Test data generators
│       └── test-normalize.R  # Tests
├── man/                      # Generated by roxygen2
├── vignettes/
├── DESCRIPTION
├── NAMESPACE                 # Generated by roxygen2
├── renv.lock                 # Dependency lock file
├── .Rprofile                 # renv activation
└── .gitignore
```

---

## Before Committing Checklist

- [ ] All tests pass (`devtools::test()`)
- [ ] R CMD check passes (`devtools::check()`)
- [ ] Documentation builds (`devtools::document()`)
- [ ] Code formatted (`styler::style_pkg()`)
- [ ] Linter passes (`lintr::lint_package()`)
- [ ] All functions have roxygen2 documentation
- [ ] No commented-out code or debug statements
- [ ] `set.seed()` used before all stochastic operations
- [ ] `renv::snapshot()` if dependencies changed
- [ ] No hardcoded credentials or file paths

---

**Remember:** Prioritize clarity and maintainability over cleverness.
```

### R-shiny.md
```markdown
---
paths:
  - "**/app.R"
  - "**/ui.R"
  - "**/server.R"
  - "**/R/mod_*.R"
  - "**/www/**"
---

# Shiny Application Development Standards

This document provides guidelines for building production-grade Shiny applications. These conventions extend the core R rules and MUST be followed for all Shiny projects.

---

## Core Architectural Principles

### 1. Module-Based Architecture

All non-trivial Shiny apps MUST use modular architecture:

- **Applets/Modules**: Self-contained UI + Server pairs that encapsulate functionality
- **Single Responsibility**: Each module handles one feature or workflow step
- **Composability**: Modules can be nested and combined

```r
# Module UI - takes id, returns tagList
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::tagList(
        shiny::textInput(ns("input"), "Enter value")
        , shiny::actionButton(ns("submit"), "Submit")
    )
}

# Module Server - uses moduleServer pattern
myModuleServer <- function(id, shared_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns
        # Module logic here
    })
}
```

### 2. R6/S4 Hybrid State Management (Recommended Pattern)

For complex apps with undo/revert functionality, use R6 as a state tracker for S4 objects:

**Division of Labor:**
- **S4 Classes**: Represent all core data structures; handle validation, transformation, computation
- **R6 Class (StateManager)**: Track snapshots of S4 objects; enable undo/revert; NO transformation logic

```r
# R6 State Manager
StateManager <- R6::R6Class("StateManager",
    public = list(
        states = list()

        , saveState = function(state_name, s4_object, config = NULL, description = "") {
            self$states[[state_name]] <- list(
                data = s4_object
                , config = config
                , description = description
                , timestamp = Sys.time()
            )
            invisible(self)
        }

        , getState = function(state_name) {
            if (!state_name %in% names(self$states)) {
                stop("State '", state_name, "' not found")
            }
            self$states[[state_name]]$data
        }

        , listStates = function() {
            names(self$states)
        }
    )
)

# Usage in server
state_manager <- StateManager$new()
state_manager$saveState("after_load", my_s4_object, description = "Initial data")
# Later: revert with state_manager$getState("after_load")
```

**Anti-Pattern**: Do NOT rewrite S4 methods in R6 or store data in new R6 formats. The hybrid model leverages strengths of both systems.

### 3. Centralized Reactive Data Flow

Use a single `reactiveValues` object as the central data bus:

```r
# In main server
workflow_data <- shiny::reactiveValues(
    data_raw = NULL
    , data_processed = NULL
    , config = list()
    , state_manager = StateManager$new()
    , tab_status = list()
)

# Pass to modules
myModuleServer("module_id", workflow_data)
```

**Rules:**
- Modules MUST NOT use the global environment for data sharing
- All shared state flows through `workflow_data`
- Each module reads from and writes to `workflow_data`

---

## Module Construction Standards

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Module UI Function | `moduleNameUI` | `qualityControlUI` |
| Module Server Function | `moduleNameServer` | `qualityControlServer` |
| Module ID | snake_case | `"quality_control"` |

### Namespace Handling

**CRITICAL**: Always use namespaced IDs in modules:

```r
# UI: Use ns() for ALL input/output IDs
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::tagList(
        shiny::textInput(ns("user_input"), "Label")  # CORRECT
        # shiny::textInput("user_input", "Label")    # WRONG - not namespaced
    )
}

# Server: Use session$ns for dynamic UI
myModuleServer <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns

        output$dynamic_ui <- shiny::renderUI({
            shiny::selectInput(ns("dynamic_select"), "Choose", choices = c("A", "B"))
        })
    })
}
```

### Return Patterns

Modules that compute artifacts MUST return them as reactives:

```r
designMatrixServer <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        # Compute design matrix
        design_matrix <- shiny::reactive({
            shiny::req(workflow_data$data_raw)
            build_design_matrix(workflow_data$data_raw, input$factors)
        })

        # Return the reactive for parent to use
        return(design_matrix)
    })
}

# Parent server
design_result <- designMatrixServer("design", workflow_data)
shiny::observe({
    workflow_data$design_matrix <- design_result()
})
```

---

## UI/UX Conventions

### Layout Structure

- **Main Sections**: Wrap major applet UIs in `shiny::wellPanel()`
- **Tabbed Content**: Use `shiny::tabsetPanel()` for sub-steps within a workflow stage
- **Inputs/Actions**: Group related inputs and action buttons in nested `wellPanel` (left/top)
- **Outputs**: Display plots, tables in the main panel area

```r
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::wellPanel(
        shiny::fluidRow(
            shiny::column(3,
                shiny::wellPanel(
                    shiny::selectInput(ns("method"), "Method", choices = c("A", "B"))
                    , shiny::actionButton(ns("run"), "Run Analysis")
                )
            )
            , shiny::column(9,
                shiny::plotOutput(ns("main_plot"), height = "600px")
            )
        )
    )
}
```

### File/Directory Selection

**ALWAYS** use `shinyFiles` for file and directory inputs:

```r
# UI
shinyFiles::shinyFilesButton(
    ns("file_select")
    , label = "Select File"
    , title = "Choose a file"
    , multiple = FALSE
)

# Server
volumes <- c(Home = fs::path_home(), getVolumes()())
shinyFiles::shinyFileChoose(input, "file_select", roots = volumes)

shiny::observeEvent(input$file_select, {
    file_path <- shinyFiles::parseFilePaths(volumes, input$file_select)$datapath
    # Use file_path
})
```

**NEVER** use base `shiny::fileInput()` for production apps - it lacks native file system access.

### Resizable Plots

For complex plots, use `shinyjqui::jqui_resizable()`:

```r
shiny::column(9,
    shinyjqui::jqui_resizable(
        shiny::plotOutput(ns("my_plot"), height = "600px", width = "100%")
    )
)
```

**MUST** define initial `height` and `width` to prevent collapsing.

### Explicit Namespaces

**MUST** use explicit package prefixes for all Shiny functions:

```r
# CORRECT
shiny::fluidRow(
    shiny::column(6, shiny::textInput(ns("x"), "X"))
    , shiny::column(6, shiny::actionButton(ns("go"), "Go"))
)

# WRONG - Will fail if packages are detached/reloaded
fluidRow(
    column(6, textInput(ns("x"), "X"))
)
```

---

## Testing with shinytest2

### Setup

```r
# Install
install.packages("shinytest2")

# Create test file
usethis::use_test("app")
```

### Snapshot Testing

```r
library(shinytest2)

test_that("App launches and basic interaction works", {
    app <- AppDriver$new(app_dir = ".", name = "basic_test")

    # Take initial snapshot
    app$expect_screenshot()

    # Interact with app
    app$set_inputs(method = "B")
    app$click("run")

    # Wait for computation
    app$wait_for_idle()

    # Verify output
    app$expect_screenshot()

    app$stop()
})
```

### Testing Modules in Isolation

```r
test_that("Module server logic works", {
    shiny::testServer(myModuleServer, args = list(workflow_data = mock_data), {
        # Set input
        session$setInputs(method = "A")

        # Trigger action
        session$setInputs(run = 1)

        # Check output
        expect_equal(output$result, expected_value)
    })
})
```

---

## Debugging

### reactlog for Reactive Dependencies

Enable reactive logging to visualize dependency graphs:

```r
# Before running app
options(shiny.reactlog = TRUE)

# Run app, then press Ctrl+F3 (Cmd+F3 on Mac) to open reactlog
shiny::runApp()

# Or programmatically
reactlog::reactlog_show()
```

### Logger Bug Workaround

**CRITICAL**: The `logger` package has issues with `{}` interpolation in reactive contexts:

```r
# BAD - Will cause error in tryCatch or reactive
tryCatch({
    risky_operation()
}, error = function(e) {
    logger::log_error("Error: {e$message}")  # FAILS
})

# GOOD - Use paste() in reactive/error contexts
tryCatch({
    risky_operation()
}, error = function(e) {
    logger::log_error(paste("Error:", e$message))  # SAFE
})
```

### Debug Print Pattern

```r
# Enable verbose mode via option
options(myapp.debug = TRUE)

debug_log <- function(...) {
    if (isTRUE(getOption("myapp.debug"))) {
        message("[DEBUG] ", ...)
    }
}

# Use in server
debug_log("Processing started, n_rows:", nrow(data))
```

---

## Performance

### bindCache for Expensive Computations

Cache reactive results based on inputs:

```r
expensive_result <- shiny::reactive({
    shiny::req(input$dataset, input$method)
    perform_expensive_computation(input$dataset, input$method)
}) |>
    shiny::bindCache(input$dataset, input$method)
```

### Async with promises/future

For long-running operations, use async to avoid blocking:

```r
library(promises)
library(future)
plan(multisession)

observeEvent(input$run_analysis, {
    # Show loading state
    output$status <- renderText("Processing...")

    future_promise({
        # Long-running computation (runs in separate R process)
        heavy_computation(workflow_data$data_raw)
    }) %...>% (function(result) {
        # Update UI with result (back in main process)
        workflow_data$result <- result
        output$status <- renderText("Complete!")
    }) %...!% (function(error) {
        # Handle errors
        output$status <- renderText(paste("Error:", error$message))
    })
})
```

### Plot Caching

Cache rendered plots:

```r
output$main_plot <- shiny::renderPlot({
    shiny::req(workflow_data$data_processed)
    create_complex_plot(workflow_data$data_processed)
}) |>
    shiny::bindCache(workflow_data$data_processed)
```

### Throttle/Debounce Reactive Inputs

Limit how often reactives fire:

```r
# Debounce: Wait for input to settle (good for text input)
search_debounced <- shiny::debounce(reactive(input$search_text), 500)

# Throttle: Limit frequency (good for sliders)
slider_throttled <- shiny::throttle(reactive(input$slider), 250)
```

---

## Error Handling

### validate/need Pattern

Provide user-friendly error messages:

```r
output$analysis_result <- shiny::renderPlot({
    shiny::validate(
        shiny::need(input$dataset, "Please select a dataset")
        , shiny::need(nrow(workflow_data$data_raw) > 0, "Dataset is empty")
        , shiny::need(input$method %in% c("A", "B", "C"), "Invalid method selected")
    )

    # Only runs if all validations pass
    create_plot(workflow_data$data_raw, input$method)
})
```

### req() for Silent Validation

Use `req()` when you want to silently stop execution without error:

```r
output$plot <- shiny::renderPlot({
    shiny::req(workflow_data$data_processed)  # Silently waits if NULL
    shiny::req(input$show_plot)               # Silently waits if FALSE

    create_plot(workflow_data$data_processed)
})
```

### safeError for User-Facing Errors

Wrap errors that should be shown to users:

```r
output$result <- shiny::renderTable({
    tryCatch({
        process_data(input$file)
    }, error = function(e) {
        stop(shiny::safeError(paste("Could not process file:", e$message)))
    })
})
```

---

## Security

### Input Sanitization

**NEVER** trust user input directly:

```r
# BAD - SQL injection risk
query <- paste0("SELECT * FROM users WHERE name = '", input$username, "'")

# GOOD - Use parameterized queries
query <- DBI::sqlInterpolate(conn, "SELECT * FROM users WHERE name = ?", input$username)
```

### XSS Prevention in renderUI

**NEVER** use `htmlOutput` with unsanitized user input:

```r
# BAD - XSS vulnerability
output$user_content <- shiny::renderUI({
    shiny::HTML(input$user_text)  # User could inject <script> tags
})

# GOOD - Escape HTML
output$user_content <- shiny::renderUI({
    shiny::tags$p(input$user_text)  # Automatically escaped
})

# GOOD - Explicit sanitization if HTML is needed
output$user_content <- shiny::renderUI({
    shiny::HTML(htmltools::htmlEscape(input$user_text))
})
```

### Session Security

```r
# Access session info
session$clientData$url_hostname
session$clientData$url_protocol

# Clean up on session end
session$onSessionEnded(function() {
    # Close database connections
    # Clean up temp files
    # Log session end
})
```

---

## Accessibility

### ARIA Labels

Add accessibility labels to interactive elements:

```r
shiny::actionButton(
    ns("submit")
    , "Submit"
    , `aria-label` = "Submit the form"
)

shiny::tags$div(
    role = "region"
    , `aria-labelledby` = ns("section_title")
    , shiny::h3(id = ns("section_title"), "Results")
    , shiny::tableOutput(ns("results_table"))
)
```

### Keyboard Navigation

Ensure interactive elements are keyboard-accessible:

```r
# Use standard HTML elements that are naturally keyboard-accessible
shiny::actionButton()  # Focusable, activates with Enter/Space
shiny::selectInput()   # Arrow key navigation

# For custom interactive elements, add tabindex
shiny::tags$div(
    tabindex = "0"
    , role = "button"
    , `aria-pressed` = "false"
    , onclick = sprintf("Shiny.setInputValue('%s', true)", ns("custom_btn"))
    , onkeydown = "if(event.key === 'Enter') this.click()"
    , "Custom Button"
)
```

---

## Bookmarking

Enable state serialization for shareable URLs:

### Enable Bookmarking

```r
# In UI
shiny::bookmarkButton()

# In server
shiny::enableBookmarking(store = "url")  # or "server" for complex state
```

### Custom Bookmark State

```r
# Exclude certain inputs from bookmarking
shiny::setBookmarkExclude(c("password", "temp_input"))

# Add custom state
shiny::onBookmark(function(state) {
    state$values$custom_data <- workflow_data$processed
})

shiny::onRestore(function(state) {
    workflow_data$processed <- state$values$custom_data
})
```

---

## Shiny Gotchas (CRITICAL)

### Grid Plot Rendering

| Problem | Solution |
|---------|----------|
| `arrangeGrob()` errors with "cannot open Rplots.pdf" | Wrap with `pdf(NULL)` ... `dev.off()` |
| `grid.arrange()` draws immediately, nothing stored | Use `arrangeGrob()` to create grob object |
| Grob doesn't appear in `renderPlot()` | Use `grid::grid.draw(grob)`, not `print()` |

### Dynamic UI Race Conditions

**NEVER** use `renderUI()` to generate output IDs dynamically—causes timing mismatches where outputs bind to non-existent elements.

**MUST** use slot-based static IDs:
```r
# WRONG - dynamic IDs from data
output_id <- paste0("plot_", assay_name)  # Race condition

# CORRECT - static slot IDs, resolve names inside render function
output$plot_assay1 <- renderImage({ ... })  # Bind at startup
output$plot_assay2 <- renderImage({ ... })
```

**MUST** populate reactive values in the SAME function that generates dependent content—not in separate `observe()` blocks.

### Data Type Preservation

**Matrix roundtrip coerces ID columns to character.** Capture type BEFORE conversion, restore AFTER:
```r
original_type <- class(df[[id_col]])[1]
# ... matrix operations ...
if (original_type %in% c("numeric", "integer")) {
    result[[id_col]] <- as.numeric(result[[id_col]])
}
```

**NEVER** use `sapply(df, is.numeric)` to detect sample columns—grabs metadata columns too. Pass explicit column names from design matrix.

### S4 File Integrity

After editing large S4 class files, **MUST** verify no truncation: `wc -l R/func_*_s4_objects.R`. Restore from git if line count drops unexpectedly.

---

## Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| Global variables for state | Use `reactiveValues` passed to modules |
| `library()` in app code | Use `package::function()` or `@importFrom` |
| `source()` for modules | Use proper module pattern |
| Relative file paths | Use `here::here()` or proper resource management |
| Blocking long operations | Use `future`/`promises` for async |
| `print()` for debugging | Use `logger` or structured debug functions |
| `fileInput()` for production | Use `shinyFiles` for native file access |
| Direct slot access (`@`) | Use accessor methods for S4 objects |

---

## Before Committing Checklist

- [ ] All modules use `NS(id)` and `session$ns` correctly
- [ ] `reactiveValues` used for shared state (no globals)
- [ ] Explicit package namespaces (`shiny::`, `shinydashboard::`)
- [ ] `validate()`/`need()` for user-facing validation
- [ ] Long operations use async patterns
- [ ] No hardcoded file paths
- [ ] `shinytest2` tests for critical workflows
- [ ] Logger uses `paste()` in reactive/error contexts

---

**Remember:** Modular architecture and proper state management are the foundation of maintainable Shiny apps.
```

### typescript.md
```markdown
# TypeScript Conventions

## System Constraints (CRITICAL)

**This system targets TypeScript 5.0+ with strict mode ALWAYS enabled.**

All TypeScript code must:
1. Pass `tsc --noEmit` with zero errors under strict configuration
2. Never use `any` type - prefer `unknown`, proper narrowing, or generics
3. Enable all strict flags in tsconfig.json
4. Use modern ES2022+ features (top-level await, private class fields, etc.)

## Type System Mastery

### Strict Configuration (Non-Negotiable)

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true,
    "noPropertyAccessFromIndexSignature": true,
    "noFallthroughCasesInSwitch": true,
    "forceConsistentCasingInFileNames": true,
    "skipLibCheck": false,
    "allowUnusedLabels": false,
    "allowUnreachableCode": false,
    "verbatimModuleSyntax": true
  }
}
```

### Type vs Interface

**Rule:** Use `type` for unions, intersections, primitives, tuples. Use `interface` for object shapes that may be extended.

```typescript
// CORRECT: Use type for unions and complex compositions
type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

type ID = string | number;
type Coordinates = [number, number];

// CORRECT: Use interface for extensible object shapes
interface User {
  id: string;
  name: string;
  email: string;
}

interface AdminUser extends User {
  permissions: string[];
}

// WRONG: Using interface for unions
interface Status = "pending" | "active";  // Won't compile

// WRONG: Using type when extension is expected
type BaseConfig = { port: number };
type ExtendedConfig = BaseConfig & { host: string };  // Works but interface is clearer
```

### Discriminated Unions

**Always include a discriminant field for tagged unions.**

```typescript
// CORRECT: Discriminated union with exhaustive checking
type ApiResponse<T> =
  | { status: "loading" }
  | { status: "success"; data: T }
  | { status: "error"; error: Error };

function handleResponse<T>(response: ApiResponse<T>): void {
  switch (response.status) {
    case "loading":
      console.log("Loading...");
      break;
    case "success":
      console.log(response.data);  // TypeScript knows data exists
      break;
    case "error":
      console.error(response.error);  // TypeScript knows error exists
      break;
    default:
      // Exhaustiveness check - will error if new status added
      const _exhaustive: never = response;
      throw new Error(`Unhandled status: ${_exhaustive}`);
  }
}

// WRONG: Union without discriminant
type BadResponse<T> =
  | { data: T }
  | { error: Error };
// Can't tell which is which at runtime
```

### Branded Types (Nominal Typing)

Use branded types to prevent mixing semantically different primitive types.

```typescript
// Define branded types
type UserId = string & { readonly __brand: "UserId" };
type Email = string & { readonly __brand: "Email" };
type URL = string & { readonly __brand: "URL" };

// Constructor functions with validation
function createUserId(id: string): UserId {
  if (!id.match(/^usr_[a-zA-Z0-9]+$/)) {
    throw new Error(`Invalid user ID: ${id}`);
  }
  return id as UserId;
}

function createEmail(email: string): Email {
  if (!email.includes("@")) {
    throw new Error(`Invalid email: ${email}`);
  }
  return email as Email;
}

// Usage
function getUserById(id: UserId): User { /* ... */ }

const id = createUserId("usr_123");
const email = createEmail("user@example.com");

getUserById(id);  // OK
getUserById(email);  // Type error - can't pass Email where UserId expected
getUserById("usr_123");  // Type error - must use constructor
```

### Conditional Types

```typescript
// Extract function parameter types
type FirstParameter<T> = T extends (first: infer P, ...args: any[]) => any
  ? P
  : never;

type SecondParameter<T> = T extends (
  first: any,
  second: infer P,
  ...args: any[]
) => any
  ? P
  : never;

// Example usage
function example(a: string, b: number): void {}
type First = FirstParameter<typeof example>;  // string
type Second = SecondParameter<typeof example>;  // number

// Conditional return types
type UnwrapPromise<T> = T extends Promise<infer U> ? U : T;

type A = UnwrapPromise<Promise<string>>;  // string
type B = UnwrapPromise<number>;           // number

// Recursive conditional types
type Awaited<T> = T extends Promise<infer U>
  ? Awaited<U>
  : T;

type Nested = Awaited<Promise<Promise<Promise<string>>>>;  // string
```

### Mapped Types

```typescript
// Make all properties optional recursively
type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

// Make all properties readonly recursively
type DeepReadonly<T> = {
  readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

// Make specific properties required
type RequireKeys<T, K extends keyof T> = T & Required<Pick<T, K>>;

interface Config {
  host?: string;
  port?: number;
  debug?: boolean;
}

type ProductionConfig = RequireKeys<Config, "host" | "port">;
// { host: string; port: number; debug?: boolean }

// Remove null and undefined from all properties
type NonNullableProps<T> = {
  [P in keyof T]: NonNullable<T[P]>;
};
```

### Template Literal Types

```typescript
// String pattern matching
type HTTPMethod = "GET" | "POST" | "PUT" | "DELETE";
type Endpoint = `/api/${string}`;
type Route = `${HTTPMethod} ${Endpoint}`;

const route: Route = "GET /api/users";  // OK
const invalid: Route = "PATCH /api/users";  // Error: PATCH not in HTTPMethod

// Derive related types
type EventName = "click" | "focus" | "blur";
type HandlerName = `on${Capitalize<EventName>}`;
// "onClick" | "onFocus" | "onBlur"

// CSS property names
type CSSProperty = "background-color" | "font-size" | "margin-top";
type CSSInJS = {
  [K in CSSProperty as `${K}` | Uncapitalize<K>]: string;
};
// { "background-color": string; "font-size": string; ... }
```

### Utility Types Mastery

```typescript
// Standard utilities
type User = {
  id: string;
  name: string;
  email: string;
  password: string;
  createdAt: Date;
};

// Pick specific properties
type UserPublic = Pick<User, "id" | "name" | "email">;

// Omit specific properties
type UserUpdate = Omit<User, "id" | "createdAt">;

// Make all properties optional
type UserPartial = Partial<User>;

// Make all properties required
type UserRequired = Required<User>;

// Create record type
type UserRoles = Record<string, "admin" | "user" | "guest">;

// Extract return type
function getUser(): User { /* ... */ }
type GetUserReturn = ReturnType<typeof getUser>;  // User

// Extract parameter types
type GetUserParams = Parameters<typeof getUserById>;  // [UserId]

// Custom utility: Nullable
type Nullable<T> = T | null;

// Custom utility: ValueOf (get union of all property values)
type ValueOf<T> = T[keyof T];
type UserValue = ValueOf<User>;  // string | Date
```

## Modern Patterns

### Never Use `any`

```typescript
// WRONG: Using any
function parseJSON(json: string): any {
  return JSON.parse(json);
}

// CORRECT: Use unknown and narrow
function parseJSON(json: string): unknown {
  return JSON.parse(json);
}

// Then narrow the type
function getUser(json: string): User {
  const data = parseJSON(json);

  // Type guard
  if (isUser(data)) {
    return data;
  }
  throw new Error("Invalid user data");
}

function isUser(value: unknown): value is User {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    "name" in value &&
    "email" in value
  );
}

// CORRECT: Use generics
function identity<T>(value: T): T {
  return value;
}

// CORRECT: Accept specific union
function log(value: string | number | boolean): void {
  console.log(value);
}
```

### Type Guards and Narrowing

```typescript
// Built-in type guards
function processValue(value: string | number): string {
  if (typeof value === "string") {
    return value.toUpperCase();  // TypeScript knows it's string
  }
  return value.toFixed(2);  // TypeScript knows it's number
}

// Custom type guards
interface Cat {
  type: "cat";
  meow(): void;
}

interface Dog {
  type: "dog";
  bark(): void;
}

type Animal = Cat | Dog;

function isCat(animal: Animal): animal is Cat {
  return animal.type === "cat";
}

function handleAnimal(animal: Animal): void {
  if (isCat(animal)) {
    animal.meow();  // TypeScript knows it's Cat
  } else {
    animal.bark();  // TypeScript knows it's Dog
  }
}

// Assertion functions
function assertIsString(value: unknown): asserts value is string {
  if (typeof value !== "string") {
    throw new Error(`Expected string, got ${typeof value}`);
  }
}

function processInput(input: unknown): string {
  assertIsString(input);
  return input.toUpperCase();  // TypeScript knows input is string
}
```

### Type-Only Imports

**Always use type-only imports when importing only types.**

```typescript
// CORRECT: Type-only import
import type { User, Config } from "./types";
import { createUser } from "./user";

// CORRECT: Mixed import
import { type User, createUser } from "./user";

// WRONG: Regular import for types only
import { User, Config } from "./types";  // May cause circular dependency issues
```

### Const Assertions

```typescript
// CORRECT: Use const assertion for literal types
const config = {
  apiUrl: "https://api.example.com",
  timeout: 5000,
  retries: 3,
} as const;

type Config = typeof config;
// {
//   readonly apiUrl: "https://api.example.com";
//   readonly timeout: 5000;
//   readonly retries: 3;
// }

// CORRECT: Use for tuple types
const point = [10, 20] as const;
type Point = typeof point;  // readonly [10, 20]

// CORRECT: Use for enum-like objects
const Status = {
  PENDING: "pending",
  ACTIVE: "active",
  INACTIVE: "inactive",
} as const;

type StatusValue = typeof Status[keyof typeof Status];
// "pending" | "active" | "inactive"

// WRONG: Without const assertion
const badConfig = {
  apiUrl: "https://api.example.com",  // Type: string (too broad)
  timeout: 5000,  // Type: number (too broad)
};
```

### Satisfies Operator (TypeScript 4.9+)

```typescript
// CORRECT: Validate type without widening
type Colors = "red" | "green" | "blue";

const favoriteColors = {
  alice: "red",
  bob: "green",
  charlie: "blue",
} satisfies Record<string, Colors>;

favoriteColors.alice;  // Type: "red" (literal, not Colors)

// WRONG: Using type annotation widens types
const badColors: Record<string, Colors> = {
  alice: "red",  // Type: Colors (widened)
};

// CORRECT: Validate array elements
const endpoints = [
  { path: "/api/users", method: "GET" },
  { path: "/api/posts", method: "POST" },
] satisfies Array<{ path: string; method: "GET" | "POST" }>;

endpoints[0].method;  // Type: "GET" (literal preserved)
```

## Error Handling

### Result Pattern

**Never throw errors for expected failure cases. Use Result type.**

```typescript
type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

// Helper constructors
function Ok<T>(value: T): Result<T, never> {
  return { ok: true, value };
}

function Err<E>(error: E): Result<never, E> {
  return { ok: false, error };
}

// Usage
function parseNumber(input: string): Result<number, string> {
  const num = Number(input);
  if (Number.isNaN(num)) {
    return Err(`Invalid number: ${input}`);
  }
  return Ok(num);
}

// Chain operations
function divide(a: number, b: number): Result<number, string> {
  if (b === 0) {
    return Err("Division by zero");
  }
  return Ok(a / b);
}

function calculate(input: string): Result<number, string> {
  const numResult = parseNumber(input);
  if (!numResult.ok) {
    return numResult;
  }

  return divide(100, numResult.value);
}

// Pattern match on result
const result = calculate("5");
if (result.ok) {
  console.log(`Result: ${result.value}`);
} else {
  console.error(`Error: ${result.error}`);
}
```

### Custom Error Classes

```typescript
// Base error class with context
abstract class AppError extends Error {
  abstract readonly code: string;
  readonly timestamp: Date;
  readonly context?: Record<string, unknown>;

  constructor(message: string, context?: Record<string, unknown>) {
    super(message);
    this.name = this.constructor.name;
    this.timestamp = new Date();
    this.context = context;
    Error.captureStackTrace(this, this.constructor);
  }
}

class ValidationError extends AppError {
  readonly code = "VALIDATION_ERROR" as const;

  constructor(
    message: string,
    public readonly field: string,
    public readonly value: unknown
  ) {
    super(message, { field, value });
  }
}

class NotFoundError extends AppError {
  readonly code = "NOT_FOUND" as const;

  constructor(
    message: string,
    public readonly resource: string,
    public readonly id: string
  ) {
    super(message, { resource, id });
  }
}

class UnauthorizedError extends AppError {
  readonly code = "UNAUTHORIZED" as const;
}

// Type guard for error handling
function isAppError(error: unknown): error is AppError {
  return error instanceof AppError;
}

// Usage
try {
  throw new NotFoundError("User not found", "User", "123");
} catch (error) {
  if (error instanceof NotFoundError) {
    console.error(`${error.code}: ${error.message}`);
    console.error(`Resource: ${error.resource}, ID: ${error.id}`);
  } else if (isAppError(error)) {
    console.error(`${error.code}: ${error.message}`);
  } else {
    console.error("Unknown error:", error);
  }
}
```

### Exhaustive Checking

**Always use exhaustive checking for union types.**

```typescript
type Status = "idle" | "loading" | "success" | "error";

function handleStatus(status: Status): string {
  switch (status) {
    case "idle":
      return "Ready";
    case "loading":
      return "Loading...";
    case "success":
      return "Done!";
    case "error":
      return "Failed";
    default:
      // This ensures TypeScript errors if a new status is added
      const _exhaustive: never = status;
      throw new Error(`Unhandled status: ${_exhaustive}`);
  }
}

// Alternative: function-based exhaustive check
function assertUnreachable(value: never): never {
  throw new Error(`Unexpected value: ${value}`);
}

function handleStatus2(status: Status): string {
  switch (status) {
    case "idle":
      return "Ready";
    case "loading":
      return "Loading...";
    case "success":
      return "Done!";
    case "error":
      return "Failed";
    default:
      return assertUnreachable(status);
  }
}
```

## Async Patterns

### Promise Best Practices

```typescript
// CORRECT: Proper error handling
async function fetchUser(id: string): Promise<User> {
  try {
    const response = await fetch(`/api/users/${id}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return await response.json();
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to fetch user: ${error.message}`);
    }
    throw error;
  }
}

// WRONG: Unhandled promise rejection
async function badFetch(id: string): Promise<User> {
  const response = await fetch(`/api/users/${id}`);
  return response.json();  // No error handling
}

// CORRECT: Concurrent requests
async function fetchMultiple(ids: string[]): Promise<User[]> {
  const promises = ids.map(id => fetchUser(id));
  return Promise.all(promises);
}

// CORRECT: Fail-fast with Promise.all
async function fetchWithDependencies(
  userId: string
): Promise<{ user: User; posts: Post[] }> {
  const [user, posts] = await Promise.all([
    fetchUser(userId),
    fetchUserPosts(userId),
  ]);
  return { user, posts };
}

// CORRECT: Continue on partial failure with Promise.allSettled
async function fetchAllUsers(ids: string[]): Promise<User[]> {
  const results = await Promise.allSettled(ids.map(fetchUser));

  return results
    .filter((result): result is PromiseFulfilledResult<User> =>
      result.status === "fulfilled"
    )
    .map(result => result.value);
}
```

### AbortController Pattern

```typescript
// CORRECT: Cancelable fetch with timeout
async function fetchWithTimeout<T>(
  url: string,
  timeoutMs: number,
  signal?: AbortSignal
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      signal: signal ?? controller.signal,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    return await response.json();
  } finally {
    clearTimeout(timeout);
  }
}

// Usage with external abort signal
const controller = new AbortController();

setTimeout(() => controller.abort(), 5000);

try {
  const data = await fetchWithTimeout<User>(
    "/api/user",
    3000,
    controller.signal
  );
  console.log(data);
} catch (error) {
  if (error instanceof DOMException && error.name === "AbortError") {
    console.log("Request aborted");
  } else {
    console.error("Request failed:", error);
  }
}
```

### Async Iteration

```typescript
// CORRECT: Async generator for pagination
async function* fetchAllPages<T>(
  baseUrl: string
): AsyncGenerator<T[], void, undefined> {
  let page = 1;
  let hasMore = true;

  while (hasMore) {
    const response = await fetch(`${baseUrl}?page=${page}`);
    const data: { items: T[]; hasMore: boolean } = await response.json();

    yield data.items;

    hasMore = data.hasMore;
    page++;
  }
}

// Usage
for await (const users of fetchAllPages<User>("/api/users")) {
  console.log(`Fetched ${users.length} users`);
  users.forEach(processUser);
}

// CORRECT: Async iterator pattern
class DataStream<T> {
  constructor(private readonly source: AsyncIterable<T>) {}

  async *filter(predicate: (item: T) => boolean): AsyncGenerator<T> {
    for await (const item of this.source) {
      if (predicate(item)) {
        yield item;
      }
    }
  }

  async *map<U>(mapper: (item: T) => U): AsyncGenerator<U> {
    for await (const item of this.source) {
      yield mapper(item);
    }
  }

  async collect(): Promise<T[]> {
    const result: T[] = [];
    for await (const item of this.source) {
      result.push(item);
    }
    return result;
  }
}
```

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Variable | camelCase | `userName`, `totalCount` |
| Function | camelCase | `fetchUser`, `calculateTotal` |
| Class | PascalCase | `User`, `HttpClient` |
| Interface | PascalCase | `User`, `Config` |
| Type Alias | PascalCase | `Result`, `ApiResponse` |
| Enum | PascalCase | `Status`, `HttpMethod` |
| Enum Member | UPPER_CASE or PascalCase | `PENDING`, `Success` |
| Generic Type Param | Single uppercase letter or PascalCase | `T`, `TKey`, `TValue` |
| Constant | UPPER_SNAKE_CASE | `MAX_RETRIES`, `API_URL` |
| Private Field | #prefix or _prefix | `#apiKey`, `_cache` |
| Boolean | is/has/can prefix | `isLoading`, `hasError`, `canSubmit` |

### Avoid Abbreviations

```typescript
// WRONG: Unclear abbreviations
function procUsr(usr: Usr): void {}

// CORRECT: Full words
function processUser(user: User): void {}

// ACCEPTABLE: Common abbreviations
type ID = string;
type URL = string;
type HTTP = "http" | "https";
```

## Tooling Configuration

### tsconfig.json (Complete)

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "allowImportingTsExtensions": true,
    "allowJs": false,
    "checkJs": false,
    "jsx": "react-jsx",
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "removeComments": true,
    "noEmit": true,
    "importHelpers": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
    "noImplicitThis": true,
    "alwaysStrict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "allowUnusedLabels": false,
    "allowUnreachableCode": false,
    "exactOptionalPropertyTypes": true,
    "skipLibCheck": false,
    "verbatimModuleSyntax": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "**/*.test.ts"]
}
```

### ESLint Configuration (typescript-eslint)

```javascript
// eslint.config.js
import eslint from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  eslint.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    languageOptions: {
      parserOptions: {
        project: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unsafe-assignment": "error",
      "@typescript-eslint/no-unsafe-member-access": "error",
      "@typescript-eslint/no-unsafe-call": "error",
      "@typescript-eslint/no-unsafe-return": "error",
      "@typescript-eslint/explicit-function-return-type": "error",
      "@typescript-eslint/explicit-module-boundary-types": "error",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/await-thenable": "error",
      "@typescript-eslint/no-misused-promises": "error",
      "@typescript-eslint/require-await": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { prefer: "type-imports" },
      ],
      "@typescript-eslint/consistent-type-exports": "error",
      "@typescript-eslint/no-import-type-side-effects": "error",
    },
  }
);
```

## Sharp Edges

### Type Coercion Gotchas

```typescript
// WRONG: Truthy/falsy assumptions
function getLength(arr: string[] | null): number {
  if (arr) {
    return arr.length;  // What about empty array?
  }
  return 0;
}

// CORRECT: Explicit null check
function getLength(arr: string[] | null): number {
  if (arr === null) {
    return 0;
  }
  return arr.length;
}

// WRONG: Implicit number coercion
const value = "5";
const result = value * 2;  // 10, but type is number (confusing)

// CORRECT: Explicit conversion
const value = "5";
const result = Number(value) * 2;
```

### Module Resolution Pitfalls

```typescript
// WRONG: Implicit any from missing types
import express from "express";  // May be 'any' if @types/express not installed

// CORRECT: Ensure types are installed
// npm install --save-dev @types/express

// WRONG: Circular dependency
// user.ts
import { Post } from "./post";
export interface User {
  posts: Post[];
}

// post.ts
import { User } from "./user";
export interface Post {
  author: User;  // Circular!
}

// CORRECT: Use type-only import or separate types file
// types.ts
export interface User {
  posts: Post[];
}
export interface Post {
  author: User;
}
```

### Declaration Merging Confusion

```typescript
// WRONG: Accidental namespace/interface merge
interface User {
  id: string;
}

namespace User {
  export function create(id: string): User {
    return { id };
  }
}

// This works but is confusing
const user = User.create("123");

// CORRECT: Separate namespace or use class
class UserFactory {
  static create(id: string): User {
    return { id };
  }
}
```

### Index Signature Traps

```typescript
// WRONG: Unsafe object access
interface Config {
  [key: string]: string;
}

const config: Config = { apiUrl: "https://api.example.com" };
const value = config.missingKey;  // Type: string (but actually undefined)

// CORRECT: Enable noUncheckedIndexedAccess
// With this flag, value is: string | undefined

// CORRECT: Use Record with explicit keys
type ConfigKey = "apiUrl" | "timeout" | "retries";
type Config = Record<ConfigKey, string>;

// Or use optional properties
interface Config {
  apiUrl: string;
  timeout?: string;
  retries?: string;
}
```

### Type Parameter Constraints

```typescript
// WRONG: Unconstrained generic loses information
function firstElement<T>(arr: T[]): T | undefined {
  return arr[0];
}

const nums = [1, 2, 3];
const first = firstElement(nums);  // Type: number | undefined (correct)

const value = firstElement([]);  // Type: never | undefined (useless)

// CORRECT: Constrain or provide default
function firstElement<T = never>(arr: T[]): T | undefined {
  return arr[0];
}

// CORRECT: Use extends for constraints
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
  return obj[key];
}

const user = { id: "1", name: "Alice" };
const name = getProperty(user, "name");  // Type: string
getProperty(user, "invalid");  // Type error
```

### Promise Type Issues

```typescript
// WRONG: Promise<void> swallows errors
async function process(): Promise<void> {
  throw new Error("Failed");  // Error silently swallowed if not awaited
}

process();  // No error, promise rejected but not handled

// CORRECT: Mark as floating or always await
void process();  // Explicit void
// or
process().catch(console.error);  // Handle rejection
// or
await process();  // Await the promise

// WRONG: Forgetting await in async function
async function getData(): Promise<number> {
  return fetchNumber();  // Returns Promise<Promise<number>>
}

// CORRECT: Always await promises
async function getData(): Promise<number> {
  return await fetchNumber();
}
```

### Enum Pitfalls

```typescript
// WRONG: Numeric enums allow invalid values
enum Status {
  Idle,
  Loading,
  Success,
}

const status: Status = 99;  // Valid but nonsense

// CORRECT: Use string enums or const objects
enum Status {
  Idle = "idle",
  Loading = "loading",
  Success = "success",
}

const status: Status = 99;  // Type error

// BETTER: Use const object
const Status = {
  Idle: "idle",
  Loading: "loading",
  Success: "success",
} as const;

type Status = typeof Status[keyof typeof Status];
```
```

