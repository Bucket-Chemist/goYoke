# Untracked Hooks - Migration Planning Document

**Status**: Planning Document (Multi-Session)
**Last Updated**: 2026-01-18
**Purpose**: Track hooks from `~/.claude/hooks` not yet covered in migration plan

---

## Overview

This document identifies hooks from the Bash-based system (`~/.claude/hooks/`) that are **not yet covered** in weeks 4-7 of the migration plan. These hooks require translation to Go as part of the complete GOgent-Fortress migration.

**Hooks Already Covered** (Weeks 1-7):
- ✅ `validate-routing.sh` → GOgent-001 to 025 (Weeks 1-3)
- ✅ `session-archive.sh` → GOgent-026 to 033 (Week 4)
- ✅ `sharp-edge-detector.sh` → GOgent-034 to 040 (Week 5)
- ✅ Integration tests → GOgent-094 to 100 (Week 6)
- ✅ Deployment and cutover → GOgent-101 to 108 (Week 7)

**Total Tickets So Far**: GOgent-000 to GOgent-108 (109 tickets)

---

## Untracked Hooks Inventory

### Critical Priority (Must Have)

These hooks are essential for system functionality and should be implemented first.

#### 1. `load-routing-context.sh`

**Trigger**: SessionStart (startup, resume)
**Function**: Injects routing schema and session handoff context at session initialization
**Complexity**: Medium-High (15-20 LOC logic, JSON parsing, file I/O)

**What It Does**:
- Loads routing schema summary from `~/.claude/routing-schema.json`
- For resume sessions, loads previous session handoff from `.claude/memory/last-handoff.md`
- Checks for pending learnings in `.claude/memory/pending-learnings.jsonl`
- Detects git branch and uncommitted changes
- Auto-detects project type (Python, R, R+Shiny, JavaScript)
- Initializes tool counter for attention-gate
- Injects all context as `additionalContext` in SessionStart response

**Why Critical**:
- First hook that fires in every session
- Sets up context that other hooks depend on (tool counter, routing schema)
- Without this, agents lose routing awareness and previous session context

**Translation Scope**: ~7 tickets (event parsing, schema loading, handoff formatting, project detection, git integration, response generation, CLI)

---

#### 2. `agent-endstate.sh`

**Trigger**: SubagentStop
**Function**: Fires tier-specific follow-up actions when subagents complete
**Complexity**: Medium (case-switch logic, transcript parsing)

**What It Does**:
- Detects which agent type completed (orchestrator, architect, einstein, python-pro, r-pro, code-reviewer, haiku-scout)
- For orchestrator/architect: prompts for TODO updates and architectural decision capture
- For einstein/opus: prompts for insight extraction and sharp-edge updates
- For implementation agents: prompts for test verification
- For code-reviewer: prompts for issue delegation
- For haiku-scout: prompts for routing decision based on scope
- Logs endstate to `/tmp/claude-agent-endstates.jsonl`

**Why Critical**:
- Prevents work from being lost when agents complete
- Enforces knowledge compounding discipline
- Essential for orchestration workflows

**Translation Scope**: ~6 tickets (SubagentStop event parsing, agent detection, response templates, logging, integration tests, CLI)

---

#### 3. `attention-gate.sh`

**Trigger**: PostToolUse (all tools)
**Function**: Routing reminders every N tool calls + periodic pending learnings flush
**Complexity**: Medium (counter management, conditional logic)

**What It Does**:
- Maintains tool call counter in `/tmp/claude-tool-counter`
- Every 10 tool calls (configurable): injects routing compliance reminder
- Every 20 tool calls (configurable): flushes pending learnings if > 5 entries
- Auto-archives pending learnings to `.claude/memory/sharp-edges/auto-flush-{timestamp}.jsonl`
- Creates markdown summaries for RAG indexing
- Prevents data loss on SIGINT (Ctrl+C) sessions

**Why Critical**:
- Prevents instruction degradation over long sessions
- Prevents sharp edge data loss
- Keeps agent aligned with routing schema

**Translation Scope**: ~6 tickets (counter management, reminder injection, auto-flush logic, archive generation, integration tests, CLI)

---

#### 4. `orchestrator-completion-guard.sh`

**Trigger**: SubagentStop (specifically for orchestrator/architect)
**Function**: Blocks orchestrator completion if background tasks uncollected
**Complexity**: Medium-High (transcript parsing, heuristic detection)

**What It Does**:
- Detects if completing agent is orchestrator/architect type
- Scans transcript for `run_in_background: true` calls
- Counts `TaskOutput` calls to detect collection
- If `spawned > collected`: blocks completion with `decision: block`
- Provides explicit remediation steps in `additionalContext`

**Why Critical**:
- Prevents "orphaned" background tasks in orchestration workflows
- Enforces fan-out/fan-in discipline (from LLM-guidelines.md)
- Direct implementation of "Background Task Collection" pattern

**Translation Scope**: ~6 tickets (SubagentStop parsing, transcript analysis, task tracking, blocking response, integration tests, CLI)

---

### High Priority (Should Have)

Important for system quality but not blocking.

#### 5. `detect-documentation-theater.sh`

**Trigger**: PreToolUse (Write/Edit on CLAUDE.md files)
**Function**: Warns when enforcement-style language added to docs without programmatic backing
**Complexity**: Medium (pattern matching, file path detection)

**What It Does**:
- Triggers on Write/Edit operations targeting `CLAUDE.md` files
- Scans content for enforcement patterns: "MUST NOT", "BLOCKED", "NEVER use", "FORBIDDEN", etc.
- Does NOT block - injects warning with friction
- Reminds agent to implement enforcement in hooks first
- References LLM-guidelines.md enforcement architecture section

**Why Important**:
- Prevents documentation theater anti-pattern
- Enforces "programmatic enforcement > text instructions" principle
- Keeps CLAUDE.md maintainable

**Translation Scope**: ~6 tickets (PreToolUse parsing, file path extraction, content scanning, pattern matching, warning response, CLI)

---

#### 6. `benchmark-logger.sh`

**Trigger**: PostToolUse (specific tools, or manual invocation)
**Function**: Logs performance benchmarks for system optimization
**Complexity**: Low-Medium (timing, JSONL logging)

**What It Does**:
- Logs tool execution times
- Tracks hook response times
- Records model tier usage
- Outputs to `/tmp/claude-benchmarks.jsonl`
- Used by `/benchmark` skill for compliance audits

**Why Important**:
- Enables cost optimization analysis
- Tracks routing efficiency
- Supports tier-appropriate delegation verification

**Translation Scope**: ~4 tickets (event parsing, timing capture, JSONL logging, CLI)

---

### Investigation Required

#### 7. `stop-gate.sh`

**Trigger**: Unknown (needs investigation)
**Function**: Unknown (not documented in routing schema or CLAUDE.md)
**Complexity**: Unknown

**Status**: File exists but purpose unclear. May be experimental, deprecated, or test-only.

**Translation Scope**: 1 ticket for investigation + 2-4 tickets for implementation (if needed)

---

### Test/Utility Hooks (Lower Priority)

These are development/testing utilities and may not need production translation.

#### 8. `test-input-capture.sh`

**Purpose**: Captures hook input for test corpus development
**Status**: Testing utility
**Translation**: May not be needed (test harness in GOgent-094 provides equivalent)

---

#### 9. `zz-test-logger.sh`

**Purpose**: Development logging utility
**Status**: Testing utility
**Translation**: Not needed for production

---

#### 10. `zzz-corpus-logger` (binary)

**Purpose**: Event corpus logging (already a binary)
**Status**: Already compiled Go binary
**Translation**: Already in target language, may need integration

---

## Impact on Existing Plans

### Week 6 (Integration Tests) - Needs Refactoring

**Current Scope**: GOgent-094 to 100 (validate-routing, session-archive, sharp-edge tests)

**Needs Addition**:
- Integration tests for `load-routing-context` hook
- Integration tests for `agent-endstate` hook
- Integration tests for `attention-gate` hook
- Integration tests for `orchestrator-completion-guard` hook
- Integration tests for `detect-documentation-theater` hook
- Integration tests for `benchmark-logger` hook

**Recommendation**: Week 6 should be expanded or split to accommodate new hook testing.

---

### Week 7 (Deployment/Cutover) - Needs Refactoring

**Current Scope**: GOgent-101 to 108 (installation, parallel testing, cutover)

**Needs Addition**:
- Parallel testing should include ALL hooks (not just first 3)
- Cutover script needs to handle ALL hook symlinks
- Rollback procedure needs to cover ALL hooks
- Post-cutover validation expanded for new hooks

**Recommendation**: Update existing tickets to reference "all hooks" rather than specific three.

---

## Proposed Weekly Plans

Based on complexity estimates, the following new weekly plans are recommended:

### Week 8: Session Initialization & Context Loading

**Scope**: `load-routing-context.sh` translation
**Tickets**: GOgent-056 to GOgent-062 (7 tickets)
**Time Estimate**: ~11 hours

**Coverage**:
- SessionStart event parsing
- Routing schema loading and formatting
- Handoff document loading
- Pending learnings detection
- Git status integration
- Project type detection
- CLI binary build

---

### Week 9: Agent Workflow Hooks

**Scope**: `agent-endstate.sh` + `attention-gate.sh` translation
**Tickets**: GOgent-063 to GOgent-074 (12 tickets)
**Time Estimate**: ~18 hours

**Coverage**:
- agent-endstate: 6 tickets
  - SubagentStop parsing
  - Agent type detection
  - Tier-specific responses
  - Decision logging
  - Integration tests
  - CLI build

- attention-gate: 6 tickets
  - Tool counter management
  - Reminder injection
  - Auto-flush logic
  - Archive generation
  - Integration tests
  - CLI build

---

### Week 10: Advanced Enforcement

**Scope**: `orchestrator-completion-guard.sh` + `detect-documentation-theater.sh` translation
**Tickets**: GOgent-075 to GOgent-086 (12 tickets)
**Time Estimate**: ~18 hours

**Coverage**:
- orchestrator-guard: 6 tickets
  - Transcript parsing
  - Background task detection
  - Blocking response
  - Remediation guidance
  - Integration tests
  - CLI build

- doc-theater: 6 tickets
  - PreToolUse parsing
  - File path detection
  - Pattern matching
  - Warning response
  - Integration tests
  - CLI build

---

### Week 11: Observability & Remaining

**Scope**: `benchmark-logger.sh` + `stop-gate.sh` investigation
**Tickets**: GOgent-087 to GOgent-093 (7 tickets)
**Time Estimate**: ~10 hours

**Coverage**:
- benchmark-logger: 4 tickets
  - Event timing
  - JSONL logging
  - Integration tests
  - CLI build

- stop-gate investigation: 3 tickets
  - Function investigation
  - Translation (if needed)
  - Integration tests (if needed)

---

## Ticket Count Summary

| Week | Hook(s) | Ticket Range | Count | Est. Hours |
|------|---------|--------------|-------|------------|
| Week 8 | load-routing-context | GOgent-056 to 062 | 7 | 11 |
| Week 9 | agent-endstate + attention-gate | GOgent-063 to 074 | 12 | 18 |
| Week 10 | orchestrator-guard + doc-theater | GOgent-075 to 086 | 12 | 18 |
| Week 11 | benchmark + stop-gate | GOgent-087 to 093 | 7 | 10 |
| **Total New** | **7 hooks** | **GOgent-056 to 093** | **38** | **57** |

**Combined Project Total**: GOgent-000 to GOgent-093 = **94 tickets**, ~142.5 hours

---

## Integration Test Refactoring (Week 6)

Week 6 should be updated to include integration tests for new hooks:

**Recommended Addition** (GOgent-100b to GOgent-100g):
- GOgent-100b: Integration tests for load-routing-context (1.5h)
- GOgent-100c: Integration tests for agent-endstate (1.5h)
- GOgent-100d: Integration tests for attention-gate (1.5h)
- GOgent-100e: Integration tests for orchestrator-guard (1.5h)
- GOgent-100f: Integration tests for doc-theater (1h)
- GOgent-100g: Integration tests for benchmark-logger (1h)

**Total Addition**: 6 tickets, ~8.5 hours

---

## Deployment Refactoring (Week 7)

Week 7 tickets should be updated to reference "all hooks" rather than specific three:

**Updates Needed**:
- GOgent-101: Installation script should install ALL 7 hook binaries
- GOgent-102: Parallel testing should run ALL hooks side-by-side
- GOgent-104: Cutover script should handle ALL hook symlinks
- GOgent-105: Rollback should restore ALL hooks
- GOgent-108: Post-cutover validation should verify ALL hooks

**No new tickets needed** - just scope expansion of existing tickets.

---

## Critical Path Impact

Adding weeks 8-11 extends critical path:

**Original Critical Path** (ending at GOgent-108):
```
GOgent-000 → GOgent-001 → ... → GOgent-108 (3 weeks)
```

**Extended Critical Path** (ending at GOgent-093):
```
GOgent-000 → GOgent-001 → ... → GOgent-093 (7 weeks)
```

However, **weeks 8-11 can partially parallelize** with earlier work if multiple contributors are available.

---

## Recommendations

### Option A: Sequential (Conservative)

Complete existing weeks 1-7, then proceed with weeks 8-11.

**Pros**: Low risk, clean dependencies
**Cons**: 7-week timeline

---

### Option B: Parallel (Aggressive)

After week 1 foundation is complete, parallelize:
- Track 1: Weeks 2-3 (session-archive, sharp-edge, tests)
- Track 2: Weeks 8-9 (load-routing-context, agent-endstate, attention-gate)
- Track 3: Week 10-11 (orchestrator-guard, doc-theater, benchmark)

**Pros**: 3-4 week timeline with 3 contributors
**Cons**: Requires coordination, integration complexity

---

### Option C: Prioritized MVP (Pragmatic)

1. Complete weeks 1-7 (existing plan) → 3 weeks
2. Implement **only critical hooks** (weeks 8-9) → 2 weeks
3. Defer weeks 10-11 to "Phase 1" post-cutover

**Pros**: 5-week complete system, low-risk cutover
**Cons**: Missing enforcement hooks initially

---

## Next Steps

1. **Decision**: Choose Option A, B, or C
2. **Create detailed weekly plans**: Weeks 8-11 (if approved)
3. **Update INDEX.md**: Add new weeks to navigation
4. **Refactor Week 6-7**: Expand scope as described above
5. **Update PROGRESS.md**: Reflect new total (94 tickets)

---

## Document Status

**Status**: ✅ Complete - Ready for approval
**Prepared By**: Orchestrator Agent
**Date**: 2026-01-18
**Approval Required**: User decision on Option A/B/C

---

## References

- **Existing Weeks**: 04-week2-session-archive.md, 05-week2-sharp-edge-memory.md
- **Hooks Location**: `~/.claude/hooks/`
- **Routing Schema**: `~/.claude/routing-schema.json`
- **LLM Guidelines**: `~/.claude/rules/LLM-guidelines.md` (Enforcement Architecture)
