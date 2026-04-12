---
name: plan-tickets
description: Comprehensive planning workflow with opus-tier agents. Scout -> Planner -> Architect -> Review -> Synthesis -> Tickets.
---

# Plan-Tickets Skill v1.0

## Purpose

Transform a goal into a comprehensive, critically-reviewed implementation plan with itemized tickets ready for the `/ticket` workflow.

**What this skill does:**

1. **Scout** — Assess scope before committing expensive resources
2. **Plan** — High-level strategy via planner (opus, 32K thinking)
3. **Architect** — Detailed implementation plan (opus, 32K thinking)
4. **Review** — 7-layer critical review (opus, 32K thinking)
5. **Resolve** — Einstein for critical issues (if needed)
6. **Synthesize** — Generate overview.md + itemized tickets

**What this skill does NOT do:**

- Implement code (delegates to language agents via `/ticket`)
- Skip the review phase (always runs staff-architect)
- Rush through planning (this is deliberate, thorough planning)

---

## Invocation

| Command                       | Behavior                                  |
| ----------------------------- | ----------------------------------------- |
| `/plan-tickets`               | Prompt for goal                           |
| `/plan-tickets [goal]`        | Start with stated goal                    |
| `/plan-tickets --skip-scout`  | Skip reconnaissance (when scope is known) |
| `/plan-tickets --resume`      | Resume from last checkpoint               |

---

## Prerequisites

**Required state:**

- Project directory with identifiable language (go.mod, pyproject.toml, etc.)
- Writeable `.gogent/sessions/` directory (auto-created by session hook)
- Session directory for artifacts (SESSION_DIR from session context)

**Optional inputs:**

- `SESSION_DIR/scout_metrics.json` (from prior scout)
- Existing tickets directory for integration

---

## Workflow

### Phase 1: Goal Capture

```
[plan-tickets] Starting comprehensive planning workflow.
[plan-tickets] Model tier: opus (32K thinking budget per phase)
```

**If no goal provided:**

```
What do you want to achieve? (One sentence describing the end state)
```

**After goal captured:**

```
[plan-tickets] Goal: "<user's goal>"
[plan-tickets] Proceeding to reconnaissance...
```

### Phase 2: Scout (Conditional)

**When to scout:**

- Scope is unknown ("implement X", "add feature Y")
- Goal mentions modules, systems, or architecture
- Could involve 5+ files

**When to skip:**

- User specified `--skip-scout`
- Goal mentions specific files
- Trivial scope

**Scout invocation:**

```bash
# Gather metrics
~/.claude/scripts/gather-scout-metrics.sh <target> > /tmp/bash_metrics.txt

# Analyze with Gemini
{
  cat /tmp/bash_metrics.txt
  echo "---FILES---"
  find . -type f \( -name "*.go" -o -name "*.py" -o -name "*.ts" \) | head -100
```

**Scout with fallback:**

```bash
else
    # Fallback: haiku-scout
    Task({subagent_type: "Explore", model: "haiku", prompt: "AGENT: haiku-scout\n\nAssess scope for: <goal>"})
fi
```

**Scout Output Schema:**

```json
{
  "scope_metrics": {
    "total_files": 47,
    "total_lines": 8234,
    "estimated_tokens": 32936
  },
  "complexity_signals": {
    "import_density": 0.15,
    "cross_file_dependencies": 12
  },
  "routing_recommendation": {
    "recommended_tier": "sonnet",
    "confidence": "high"
  }
}
```

**Output:**

```
[scout] Files: X | Lines: Y | Tokens: ~Z
[scout] Complexity: <score> | Recommended tier: opus
[scout] Proceeding to strategic planning...
```

### Phase 3: Planner (Opus)

**Invoke planner agent:**

```javascript
Task({
  description: "Create strategy for goal",
  subagent_type: "Plan",
  model: "opus",
  prompt: `AGENT: planner

TASK: Create strategic plan for the following goal.

GOAL: <user's stated goal>

SCOUT REPORT: <JSON from scout, or "Not available">

PROJECT CONTEXT:
- Language: <detected language>
- Conventions: <loaded conventions>

INSTRUCTIONS:
1. Analyze the goal and restate requirements clearly
2. Identify risks and unknowns
3. Formulate high-level strategic approach
4. Document constraints and scope boundaries
5. Define measurable success criteria
6. Write strategy to SESSION_DIR/strategy.md

If genuinely ambiguous, ask up to 2 clarifying questions via AskUserQuestion.
`,
});
```

**User checkpoint:**

```
[plan-tickets] Strategy complete.

Summary:
- Requirements: <count> identified
- Risks: <count> identified
- Approach: <one-line summary>

Strategy saved to: SESSION_DIR/strategy.md

Review strategy before proceeding? (y/n/edit)
```

**On response:**

- `y` or `yes` — Show strategy.md contents, then ask to proceed
- `n` or `no` — Proceed directly to architect
- `edit` — Allow user to modify, then re-validate

### Phase 4: Architect (Opus)

**Invoke architect agent:**

```javascript
Task({
  description: "Create implementation plan",
  subagent_type: "Plan",
  model: "opus",
  prompt: `AGENT: architect

TASK: Create detailed, phased implementation plan.

STRATEGY: <contents of SESSION_DIR/strategy.md>

SCOUT REPORT: <JSON if available>

INSTRUCTIONS:
1. Parse the strategy document
2. Map dependencies between components
3. Create ordered implementation phases
4. Assess risks per phase
5. Define validation criteria per phase
6. Write specs.md to SESSION_DIR/specs.md
7. Use TaskCreate to register tasks, TaskUpdate to set dependencies

CONSTRAINTS:
- Follow existing codebase patterns
- Each phase should be independently testable
- Include rollback considerations
`,
});
```

**Output:**

```
[plan-tickets] Implementation plan complete.

Phases: <count>
Files affected: <count>
Estimated tickets: <count>

Specs saved to: SESSION_DIR/specs.md

Proceeding to critical review...
```

### Phase 5: Staff Architect Review (Opus)

**Invoke staff-architect-critical-review:**

```javascript
Task({
  description: "Critical review of implementation plan",
  subagent_type: "Analyst",
  model: "opus",
  prompt: `AGENT: staff-architect-critical-review

TASK: Perform 7-layer critical review of implementation plan.

PLAN FILE: SESSION_DIR/specs.md
STRATEGY FILE: SESSION_DIR/strategy.md (for context)

REVIEW LAYERS:
1. Assumption Register - surface and challenge assumptions
2. Dependency Mapping - check for circular/hidden dependencies
3. Failure Mode Analysis - what if each phase fails?
4. Cost-Benefit Assessment - is complexity justified?
5. Testing Coverage - are all paths tested?
6. Architecture Smell Detection - identify anti-patterns
7. Contractor Readiness - can someone start Monday with zero questions?

OUTPUT:
- SESSION_DIR/review-critique.md
- SESSION_DIR/review-metadata.json

Be adversarial to the plan, but constructive to the outcome.
`,
});
```

**Process review result:**

```bash
# Read review verdict
gogent_session_dir="$(cat "$(git rev-parse --show-toplevel 2>/dev/null || echo .)/.gogent/current-session" 2>/dev/null)"
gogent_session_dir="${gogent_session_dir:-.gogent/sessions/$(date +%Y%m%d-%H%M%S)}"
verdict=$(jq -r '.verdict' "$gogent_session_dir/review-metadata.json")
critical_count=$(jq -r '.issue_counts.critical' "$gogent_session_dir/review-metadata.json")
```

**Branch based on verdict:**

| Verdict                   | Action                                                |
| ------------------------- | ----------------------------------------------------- |
| `APPROVE`                 | Proceed to synthesis                                  |
| `APPROVE_WITH_CONDITIONS` | Show conditions, ask user to acknowledge, proceed     |
| `CONCERNS`                | Show concerns, ask if user wants to proceed or revise |
| `CRITICAL_ISSUES`         | Escalate to Einstein or revise                        |

### Phase 5b: Einstein Resolution (Conditional)

**If `CRITICAL_ISSUES` detected:**

```
[plan-tickets] Critical issues found in review.
[plan-tickets] Issues:
- C-1: <issue summary>
- C-2: <issue summary>

Options:
1. Escalate to Einstein for deep analysis
2. Revise plan manually
3. Proceed anyway (not recommended)

Choice? (1/2/3)
```

**If Einstein escalation chosen:**

```
[plan-tickets] Generating GAP document for Einstein...
```

Generate GAP document with:

- Review critique as context
- Specific critical issues as questions
- Plan excerpts as relevant context

```
[plan-tickets] GAP document ready: SESSION_DIR/einstein-gap-<timestamp>.md

Run /einstein to resolve, then /plan-tickets --resume to continue.
```

**STOP and wait for user.**

### Phase 6: Synthesis

After review passes (APPROVE or APPROVE_WITH_CONDITIONS acknowledged):

**Generate overview.md:**

```markdown
# Implementation Plan: <Goal>

> Generated: <timestamp>
> Workflow: /plan-tickets v1.0
> Review Status: <verdict>

## Executive Summary

<2-3 sentences from strategy + specs>

## Strategic Approach

<From strategy.md>

## Implementation Phases

| Phase | Description | Tickets            | Dependencies |
| ----- | ----------- | ------------------ | ------------ |
| 1     | <name>      | PROJ-001, PROJ-002 | None         |
| 2     | <name>      | PROJ-003           | Phase 1      |
| ...   | ...         | ...                | ...          |

## Risk Register

| Risk                | Likelihood | Impact | Mitigation |
| ------------------- | ---------- | ------ | ---------- |
| <top 3 from review> | ...        | ...    | ...        |

## Review Summary

**Verdict:** <verdict>
**Critical Issues:** <count> (resolved: <count>)
**Major Issues:** <count>
**Commendations:** <count>

### Conditions (if any)

<list conditions that must be addressed>

## Success Criteria

<from strategy.md>

## Next Steps

1. Run `/ticket` to begin implementation
2. Address review conditions during implementation
3. Re-review after Phase <N> if significant changes

---

_Generated by /plan-tickets skill. Review critique: SESSION_DIR/review-critique.md_
```

**Generate tickets:**

For each phase in specs.md, generate ticket files:

```bash
# Create tickets directory if needed
mkdir -p tickets

# Generate tickets-index.json
# Generate individual PROJ-NNN.md files
```

**tickets-index.json Schema:**

```json
{
  "version": "1.0",
  "project": "<prefix>",
  "generated_by": "/plan-tickets v1.0",
  "generated_at": "<ISO timestamp>",
  "tickets": [
    {
      "id": "PROJ-001",
      "title": "...",
      "status": "pending",
      "phase": 1,
      "dependencies": [],
      "file": "tickets/PROJ-001.md"
    }
  ]
}
```

**Ticket Prefix Resolution:**

1. Read `.ticket-config.json` if exists → use `project_name` field
2. Parse `go.mod` module name → use last path component uppercase
3. Fallback: "PROJ"

**Dependency Mapping:**

- Phase 1 tickets: `dependencies: []`
- Phase N tickets (N>1): `dependencies: [all Phase N-1 ticket IDs]`

**Ticket format:**

```markdown
---
id: PROJ-001
title: "<Task title from specs.md>"
status: pending
dependencies: []
time_estimate: "<from specs.md or 'TBD'>"
phase: 1
tags: [plan-generated, phase-1]
needs_planning: false
---

# PROJ-001: <Title>

## Description

<From specs.md phase breakdown>

## Acceptance Criteria

- [ ] <Criterion 1>
- [ ] <Criterion 2>
- [ ] <Criterion 3>

## Files

- `path/to/file.go` — <what to modify>
- `path/to/new_file.go` — <what to create>

## Context

<Relevant notes from strategy/specs>

## Review Notes

<Any specific concerns from review for this ticket>

---

_Generated from: .claude/tmp/specs.md Phase <N>_
```

**Final output:**

```
[plan-tickets] Synthesis complete.

Generated:
- overview.md (executive summary)
- tickets/tickets-index.json (<count> tickets)
- tickets/PROJ-001.md through PROJ-<N>.md

Review status: <verdict>
Estimated cost of planning: $<total>

Ready for implementation. Run /ticket to begin.
```

---

## State Files

| File                               | Written By        | Read By                    | Purpose                      |
| ---------------------------------- | ----------------- | -------------------------- | ---------------------------- |
| `SESSION_DIR/strategy.md`          | planner           | architect, synthesis       | High-level strategy          |
| `SESSION_DIR/specs.md`             | architect         | staff-architect, synthesis | Detailed implementation plan |
| `SESSION_DIR/review-critique.md`   | staff-architect   | user, synthesis            | Critical review              |
| `SESSION_DIR/review-metadata.json` | staff-architect   | /plan-tickets workflow             | Review verdict and counts    |
| `SESSION_DIR/einstein-gap-*.md`    | /plan-tickets (if needed) | /einstein                  | Escalation context           |
| `overview.md`                      | synthesis         | user                       | Human-readable summary       |
| `tickets/*.md`                     | synthesis         | /ticket                    | Individual tickets           |
| `tickets/tickets-index.json`       | synthesis         | /ticket                    | Ticket registry              |

---

## Cost Model

| Phase     | Model        | Est. Tokens | Est. Cost      |
| --------- | ------------ | ----------- | -------------- |
| Scout     | Gemini/Haiku | 2-5K        | $0.01-0.02     |
| Planner   | Opus         | 15-30K      | $0.68-1.35     |
| Architect | Opus         | 20-40K      | $0.90-1.80     |
| Review    | Opus         | 15-30K      | $0.68-1.35     |
| Synthesis | N/A          | 0           | $0.00          |
| **Total** |              | 52-105K     | **$2.27-4.52** |

**Note:** Einstein resolution (if triggered) adds ~$0.90 per invocation.

**Cost Disclaimer:** Estimates assume efficient execution. Actual cost may be higher with:

- Extended user review time (context accumulates)
- Einstein escalation (+$0.90 per call)
- Retry loops on agent failures
- haiku-scout fallback instead of gemini (slightly higher)

---

## Checkpoints & Resume

The /plan-tickets skill saves state at each phase:

| Checkpoint      | State File                         | Resume Point        |
| --------------- | ---------------------------------- | ------------------- |
| After scout     | `SESSION_DIR/scout_metrics.json`   | Phase 3 (planner)   |
| After planner   | `SESSION_DIR/strategy.md`          | Phase 4 (architect) |
| After architect | `SESSION_DIR/specs.md`             | Phase 5 (review)    |
| After review    | `SESSION_DIR/review-metadata.json` | Phase 6 (synthesis) |
| Einstein needed | `SESSION_DIR/einstein-gap-*.md`    | After /einstein     |

**Resume command:**

```
/plan-tickets --resume
```

**Resume Detection Algorithm:**

```
Check in order (first match wins):
1. einstein-gap-*.md exists → resume at synthesis (post-Einstein)
2. review-metadata.json exists → resume at synthesis
3. specs.md exists → resume at review
4. strategy.md exists → resume at architect
5. scout_metrics.json exists → resume at planner
6. None → error "No checkpoint found. Run /plan-tickets to start fresh."
```

Detects last checkpoint and continues from there.

---

## Error Handling

| Error           | Recovery                                             |
| --------------- | ---------------------------------------------------- |
| Scout fails     | Fall back to haiku-scout, or skip scout with warning |
| Planner fails   | Retry once, then ask user for manual strategy input  |
| Architect fails | Retry with simpler constraints, or escalate          |
| Review fails    | Proceed with warning (review is advisory)            |
| Synthesis fails | Manual ticket creation from specs.md                 |

**Timeout Behavior:**

- Interactive mode: Wait indefinitely for user response
- Non-interactive mode: Default after 30s (proceed=yes, acknowledge=yes)

---

## Integration with /ticket

The tickets generated by /plan-tickets are fully compatible with /ticket skill:

```
/plan-tickets "Add user authentication"
  ↓
[generates overview.md + tickets/]
  ↓
/ticket
  ↓
[picks up PROJ-001, validates, begins implementation]
```

---

## When to Use /plan-tickets

**Use /plan-tickets when:**

- Starting a new feature with unclear scope
- Making architectural changes
- Planning multi-phase implementations
- You want review before committing to implementation
- The goal is complex enough to benefit from structured planning

**Don't use /plan-tickets when:**

- Fixing a simple bug (just fix it)
- Single-file changes (use /ticket directly)
- Documentation updates
- You already have a clear implementation plan

---

## Example Session

```
$ /plan-tickets "Add JWT authentication to the API"

[plan-tickets] Starting comprehensive planning workflow.
[plan-tickets] Model tier: opus (32K thinking budget per phase)
[plan-tickets] Goal: "Add JWT authentication to the API"

[scout] Files: 47 | Lines: 8,234 | Tokens: ~32K
[scout] Complexity: 7 | Recommended tier: sonnet
[scout] Proceeding to strategic planning...

[plan-tickets] Strategy complete.

Summary:
- Requirements: 5 identified
- Risks: 3 identified
- Approach: JWT with refresh tokens, middleware-based auth

Strategy saved to: SESSION_DIR/strategy.md

Review strategy before proceeding? (y/n/edit) n

[plan-tickets] Implementation plan complete.

Phases: 4
Files affected: 12
Estimated tickets: 8

Specs saved to: SESSION_DIR/specs.md

Proceeding to critical review...

[review] 7-layer critical review complete.

Verdict: APPROVE_WITH_CONDITIONS
Critical: 0 | Major: 2 | Minor: 4
Commendations: 3

Conditions:
- M-1: Add token rotation mechanism
- M-2: Include rate limiting on auth endpoints

Review saved to: SESSION_DIR/review-critique.md

Acknowledge conditions and proceed? (y/n) y

[plan-tickets] Synthesis complete.

Generated:
- overview.md (executive summary)
- tickets/tickets-index.json (8 tickets)
- tickets/AUTH-001.md through AUTH-008.md

Review status: APPROVE_WITH_CONDITIONS
Estimated cost of planning: $3.12

Ready for implementation. Run /ticket to begin.
```

---

**Skill Version:** 1.0
**Last Updated:** 2026-01-31
**Maintained By:** System
