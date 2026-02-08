---
name: braintrust
description: >
  Multi-perspective deep analysis workflow. Invokes Mozart (orchestrator) who
  conducts clarification interview, spawns scouts for reconnaissance, then
  dispatches Einstein (theoretical) and Staff-Architect (practical) in parallel.
  Beethoven synthesizes orthogonal analyses into standardized output document.
replaces: einstein
version: 1.0.0
---

# Braintrust Skill v1.0

## Purpose

Braintrust is the premium analysis workflow for complex problems requiring both theoretical depth and practical rigor. It replaces the standalone `/einstein` skill with a multi-agent orchestrated approach.

**What this skill does:**

1. **Mozart** (Opus) - Intake, interview, scout, decompose problem
2. **Einstein** (Opus) - Theoretical analysis: root cause, frameworks, first principles
3. **Staff-Architect** (Opus) - Practical review: 7-layer framework, risks, implementation
4. **Beethoven** (Opus) - Synthesize orthogonal analyses into unified document

**What this skill does NOT do:**

- Implement code (analysis only, delegate execution after)
- Skip user confirmation (Mozart always confirms before heavy Opus spend)
- Produce vague outputs (standardized document format)

---

## Invocation

| Command                      | Behavior                            |
| ---------------------------- | ----------------------------------- |
| `/braintrust`                | Start with problem statement prompt |
| `/braintrust "question"`     | Quick mode with inline problem      |
| `/braintrust path/to/gap.md` | Process existing GAP document       |

---

## Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        /braintrust                                  │
│                     User Invocation                                 │
└───────────────────────────┬─────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MOZART (Opus)                                    │
│              Problem Decomposition Orchestrator                     │
│                                                                     │
│  Phase 1: INTAKE                                                    │
│    - Parse user input (question, GAP doc, or raw problem)           │
│                                                                     │
│  Phase 2: INTERVIEW (if needed)                                     │
│    - Conduct clarification interview (max 3 questions)              │
│                                                                     │
│  Phase 3: RECONNAISSANCE                                            │
│    - Spawn scouts (haiku) to gather scope metrics                   │
│    - Assemble Problem Brief                                         │
│                                                                     │
│  Phase 4: CONFIRMATION CHECKPOINT                                   │
│    - Present Problem Brief to user                                  │
│    - User approves before heavy Opus spend                          │
│                                                                     │
│  Phase 5: ORTHOGONAL DISPATCH (parallel)                            │
│    ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│    │  EINSTEIN (Opus)    │    │  STAFF-ARCHITECT-CR (Opus)      │  │
│    │  Theoretical        │    │  Practical/Implementation       │  │
│    │  - Root cause       │    │  - 7-layer review               │  │
│    │  - Conceptual model │    │  - Risk assessment              │  │
│    │  - Novel approaches │    │  - Failure modes                │  │
│    │  - First principles │    │  - Contractor readiness         │  │
│    └──────────┬──────────┘    └─────────────────┬───────────────┘  │
│               │                                  │                  │
│               └─────────────┬────────────────────┘                  │
│                             │                                       │
│  Phase 6: HANDOFF TO BEETHOVEN                                      │
│    - Collect both analyses                                          │
│    - Pass to Beethoven with Problem Brief                           │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   BEETHOVEN (Opus)                                  │
│                  Composer / Synthesizer                             │
│                                                                     │
│  - Identify convergences (where both agree)                         │
│  - Highlight divergences (where they disagree)                      │
│  - Resolve contradictions with higher-order reasoning               │
│  - Synthesize unified recommendation                                │
│  - Produce standardized Braintrust Analysis Document                │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│              BRAINTRUST ANALYSIS DOCUMENT                           │
│              .claude/braintrust/analysis-{timestamp}.md             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Interview Protocol

Mozart conducts a structured 4-question interview to configure the Braintrust team. This protocol is defined in TC-018 and ensures complete, validated team configurations.

### The Four Questions

| # | Question | When Asked | Maps To |
|---|----------|------------|---------|
| **Q1** | "What problem or question do you want the Braintrust to analyze?" | **ALWAYS** | `task.problem_statement` (all stdin files) |
| **Q2** | "Which files or areas of the codebase are relevant? (Or should I scout first?)" | **ALWAYS** | `context.relevant_files[]` OR `reads_from.scout_metrics` |
| **Q3** | "Should I include both Einstein and Staff-Architect, or just Einstein?" | **CONDITIONAL** (narrow scope) | `waves[].members[]` (full vs single-agent) |
| **Q4** | "Default budget is $5.00. Want to adjust?" | **CONDITIONAL** (cost concerns) | `budget_max_usd`, `budget_remaining_usd` |

### Decision Flow

```
Q1: Problem Statement (ALWAYS)
  └─► Capture problem (min 20 chars)

Q2: Scope (ALWAYS)
  ├─► Files provided → Read and validate → relevant_files[]
  ├─► "scout" → Spawn haiku scout → wait → scout_metrics path
  └─► "whole codebase" → Warn + recommend scout

Q3: Team Composition (CONDITIONAL)
  ├─► "both" (default) → Full: Einstein + Staff-Architect + Beethoven
  ├─► "just Einstein" → Single: Einstein only, skip synthesis
  └─► (not asked) → Default to full braintrust

Q4: Budget (CONDITIONAL)
  ├─► User specifies → Validate ($1-$50)
  └─► (not asked) → Default $5.00

Generate Problem Brief → Confirm → Generate config.json + stdin files → Launch
```

### Budget Ranges

| Configuration | Estimated Cost |
|--------------|----------------|
| Just Einstein | ~$1.50 |
| Full Braintrust (Einstein + Staff-Architect + Beethoven) | ~$4.50-$5.50 |
| Validation limits | Min $1.00, Max $50.00, Default $5.00 |

### Scout-First Path

When user responds "scout" or "don't know" to Q2:

1. Mozart spawns haiku scout via `Task(model: "haiku")`
2. Scout analyzes codebase (~10-30s depending on repo size)
3. Scout writes `.claude/tmp/scout_metrics.json`
4. Mozart reads scout output, extracts top 5 critical files
5. Scout metrics path included in `einstein.reads_from.scout_metrics`

**Full executable protocol:** See `.claude/agents/mozart/mozart.md` Phase 2

---

## Execution

When `/braintrust` is invoked, the Router spawns Mozart via Task(). Mozart conducts the interview, then dispatches execution either in the background (team-run) or foreground (MCP spawn), controlled by `settings.json -> "use_team_pattern"`.

### Step 1: Invoke Mozart via Task()

**IMPORTANT**: The Router (Level 0) IS allowed to use Task(). This ensures TUI interactivity (stdin/stdout) is preserved for the interview phase.

```javascript
Task({
  model: "opus",
  description: "Mozart: Braintrust problem decomposition",
  subagent_type: "Plan",
  prompt: `AGENT: mozart

BRAINTRUST INVOCATION

USER INPUT: {user_input}
INPUT TYPE: {raw_problem | gap_document | inline_question}

Execute Braintrust workflow:
1. Parse input
2. Interview if needed (max 4 questions per TC-018 protocol)
3. Spawn scouts for reconnaissance
4. Assemble Problem Brief
5. Confirm with user before proceeding
6. Check settings.json use_team_pattern flag:
   - If true: Generate team config + stdin files, launch gogent-team-run (background), return immediately
   - If false: Dispatch Einstein + Staff-Architect via mcp__gofortress__spawn_agent (foreground), collect, invoke Beethoven, return analysis path`,
});
```

### Step 2: Mozart Handles Orchestration

**Background mode** (`use_team_pattern: true` — default):

Mozart conducts interview (~30s), generates team configuration:
1. Creates team directory: `$GOGENT_SESSION_DIR/teams/{timestamp}.braintrust/`
2. Writes Problem Brief, config.json, and 3 stdin files
3. Launches `gogent-team-run` as background process
4. Returns to user immediately (~40s total)

Background execution continues autonomously:
- **Wave 1**: Einstein + Staff-Architect run in parallel (~5-8 min)
- **Inter-wave**: `gogent-team-prepare-synthesis` merges Wave 1 outputs → `pre-synthesis.md`
- **Wave 2**: Beethoven reads `pre-synthesis.md` and produces final synthesis

**Foreground mode** (`use_team_pattern: false`):

Mozart conducts interview, then orchestrates the full workflow synchronously:
- Spawns Einstein + Staff-Architect via `mcp__gofortress__spawn_agent` (parallel)
- Waits for both to complete (~6-9 min)
- Spawns Beethoven with collected analyses
- Returns final document path

### Step 3: Return to User

**Background mode:**
```
[Braintrust] Team dispatched.
[Braintrust] Track progress: /team-status
[Braintrust] View result when complete: /team-result
```

**Foreground mode:**
```
[Braintrust] Analysis complete.
[Braintrust] Output: .claude/braintrust/analysis-{timestamp}.md
[Braintrust] Agents: Mozart → Einstein + Staff-Architect → Beethoven
```

---

## Output Format

The final Braintrust Analysis document includes:

```markdown
# Braintrust Analysis: {Title}

## Executive Summary

## Problem Statement

## Analysis Perspectives

### Einstein (Theoretical)

### Staff-Architect (Practical)

## Convergence Points

## Divergence Resolution

## Unified Recommendations

## Implementation Pathway

## Risk Assessment

## Open Questions

## Appendix: Full Analyses

## Metadata
```

---

## Cost Model

| Agent                  | Typical Tokens         | Estimated Cost |
| ---------------------- | ---------------------- | -------------- |
| Mozart (orchestration) | 15,000 in / 5,000 out  | ~$0.75         |
| Scouts (2x haiku)      | 2,000 each             | ~$0.01         |
| Einstein               | 20,000 in / 8,000 out  | ~$1.10         |
| Staff-Architect        | 20,000 in / 6,000 out  | ~$1.00         |
| Beethoven              | 25,000 in / 10,000 out | ~$1.25         |
| **Total**              | -                      | **~$4.10**     |

**Note**: No cost ceiling. Quality over cost for Braintrust.

---

## State Files

### Foreground Mode
| File                                    | Written By | Read By                              | Purpose               |
| --------------------------------------- | ---------- | ------------------------------------ | --------------------- |
| `.claude/braintrust/problem-brief-*.md` | Mozart     | Einstein, Staff-Architect, Beethoven | Problem decomposition |
| `.claude/braintrust/analysis-*.md`      | Beethoven  | User                                 | Final output          |
| `.claude/braintrust/mozart-log-*.jsonl` | Mozart     | Telemetry                            | Workflow tracking     |
| `.claude/tmp/scout_metrics.json`        | Scouts     | Mozart                               | Reconnaissance data   |

### Team-Run Mode (Background)
| File                                    | Written By                       | Read By                | Purpose                      |
| --------------------------------------- | -------------------------------- | ---------------------- | ---------------------------- |
| `{team_dir}/config.json`               | Mozart                           | gogent-team-run        | Team execution configuration |
| `{team_dir}/problem-brief.md`          | Mozart                           | Agents (via stdin)     | Problem decomposition        |
| `{team_dir}/stdin_einstein.json`       | Mozart                           | gogent-team-run        | Einstein input               |
| `{team_dir}/stdin_staff-architect.json` | Mozart                          | gogent-team-run        | Staff-Architect input        |
| `{team_dir}/stdin_beethoven.json`      | Mozart                           | gogent-team-run        | Beethoven input              |
| `{team_dir}/stdout_einstein.json`      | gogent-team-run                  | prepare-synthesis      | Einstein structured output   |
| `{team_dir}/stdout_staff-arch.json`    | gogent-team-run                  | prepare-synthesis      | Staff-Architect output       |
| `{team_dir}/pre-synthesis.md`          | gogent-team-prepare-synthesis    | Beethoven (Read tool)  | Merged Wave 1 analyses       |
| `{team_dir}/stdout_beethoven.json`     | gogent-team-run                  | /team-result           | Final synthesis output       |
| `{team_dir}/runner.log`               | gogent-team-run                  | /team-status           | Execution log                |

`{team_dir}` = `$GOGENT_SESSION_DIR/teams/{timestamp}.braintrust/`

---

## Comparison: Einstein vs Braintrust

| Aspect       | Old /einstein           | New /braintrust                             |
| ------------ | ----------------------- | ------------------------------------------- |
| Agents       | 1 (Einstein)            | 4 (Mozart, Einstein, Staff-Arch, Beethoven) |
| Perspectives | Single deep analysis    | Theoretical + Practical orthogonal          |
| Interview    | None (GAP doc required) | Built-in clarification                      |
| Confirmation | Pre-flight check        | Full problem brief review                   |
| Output       | Analysis markdown       | Standardized synthesis document             |
| Cost         | ~$0.92                  | ~$4.10                                      |
| Use case     | Bounded escalations     | Complex problem workshopping                |

---

## When to Use Braintrust

**Use Braintrust for:**

- Complex architectural decisions
- Problems with both theoretical and practical dimensions
- Situations requiring multiple perspectives
- High-stakes decisions worth the cost
- Thought workshopping and whiteboarding
- Problems where implementation concerns matter

**Don't use Braintrust for:**

- Simple debugging (use regular agents)
- Single-perspective analysis (old einstein pattern still works via escalation)
- Time-sensitive issues (4 Opus agents take time)
- Well-understood problems (overkill)

---

## Migration from /einstein

The `/einstein` skill is replaced by `/braintrust`. However:

1. **Escalation protocol still works**: Agents can still generate GAP documents
2. **GAP documents as input**: `/braintrust path/to/gap.md` processes existing GAPs
3. **Legacy triggers preserved**: "escalate to einstein" still routes appropriately

For simple escalations that don't need full multi-perspective analysis, the GAP document pattern remains valid - just invoke `/braintrust` with the GAP path.

---

## Skill Metadata

```yaml
skill_id: braintrust
version: 1.0.0
replaces: einstein
tier: 3
model: opus (all agents)
agents_invoked:
  - mozart
  - einstein
  - staff-architect-critical-review
  - beethoven
cost_estimate: "$4-6 per invocation"
output_location: ".claude/braintrust/"
user_confirmation: required (Phase 4)
```
