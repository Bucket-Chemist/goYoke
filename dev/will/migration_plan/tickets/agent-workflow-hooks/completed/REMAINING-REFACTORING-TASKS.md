# Remaining Ticket Refactoring Tasks

**Status**: 6/13 complete, 7 remaining
**Last Updated**: 2026-01-24 14:30

---

## Task Checklist

### Task 1: GOgent-065 - Endstate Logging XDG Paths
- [ ] Read current GOgent-065.md
- [ ] Replace hardcoded `/tmp/claude-agent-endstates.jsonl` with `config.GetGOgentDir() + "/agent-endstates.jsonl"`
- [ ] Add HandoffArtifacts integration note
- [ ] Update tests to use `t.TempDir()` instead of `/tmp/`
- [ ] Update acceptance criteria (XDG compliance, handoff integration)
- [ ] Save updated file

**Key Change**: Line 54 - change path from hardcoded `/tmp/` to XDG-compliant

---

### Task 2: GOgent-066 - Integration Tests Schema Update
- [ ] Read current GOgent-066.md
- [ ] Update test JSON from speculated schema to ACTUAL schema:
  - Remove: `"type": "stop"`, `"agent_id"`, `"tier"`, etc.
  - Add: `"session_id"`, `"transcript_path"`, `"stop_hook_active"`
- [ ] Add mock transcript file creation in tests
- [ ] Use `t.TempDir()` for test isolation
- [ ] Add simulation harness test examples
- [ ] Save updated file

**Key Change**: Lines 37-45 - test JSON must use actual SubagentStop schema

---

### Task 3: GOgent-067 - CLI Transcript Parsing
- [ ] Read current GOgent-067.md
- [ ] Add transcript parsing step after event parsing (lines 44-61)
- [ ] Handle parsing failures gracefully (warnings, not errors)
- [ ] Add Makefile target: `build-agent-endstate`
- [ ] Update acceptance criteria (transcript parsing, graceful degradation)
- [ ] Save updated file

**Key Addition**: Between event parsing and response generation, add metadata extraction

---

### Task 4: GOgent-068 - Counter Location Change
- [ ] Read current GOgent-068.md
- [ ] Change location from `pkg/observability/counter.go` to `pkg/config/paths.go`
- [ ] Remove ToolCounter struct (use functions directly)
- [ ] Add only: `ShouldRemind()`, `ShouldFlush()`, `GetToolCountAndIncrement()`
- [ ] Reference existing `IncrementToolCount()` pattern (syscall.Flock)
- [ ] Update tests location to `pkg/config/paths_test.go`
- [ ] Update acceptance criteria (extend existing, NOT new package)
- [ ] Save updated file

**Key Change**: DO NOT create new package, extend existing pkg/config/paths.go

---

### Task 5: GOgent-069 - Flush Logic Location Change
- [ ] Read current GOgent-069.md
- [ ] Change location from `pkg/observability/` to `pkg/session/`
- [ ] Check if `CheckPendingLearnings` already exists (may reuse)
- [ ] Add environment variable configuration (GOGENT_FLUSH_THRESHOLD)
- [ ] Update acceptance criteria (reuse existing, env config)
- [ ] Save updated file

**Key Change**: Use pkg/session/, leverage existing CheckPendingLearnings if present

---

### Task 6: GOgent-071 - Test Isolation Updates
- [ ] Read current GOgent-071.md
- [ ] Replace `os.Remove(COUNTER_FILE)` with `t.TempDir()` pattern (lines 36-38)
- [ ] Add simulation harness integration tests
- [ ] Follow existing test patterns in `pkg/config/paths_test.go`
- [ ] Update acceptance criteria (t.TempDir, no global state)
- [ ] Save updated file

**Key Change**: Lines 36-38 - use t.TempDir() for test isolation

---

### Task 7: GOgent-072 - Merge into Sharp-Edge
- [ ] Read current GOgent-072.md
- [ ] Change title from "Build gogent-attention-gate CLI" to "Merge Attention-Gate into gogent-sharp-edge"
- [ ] Change from new CLI to extending `cmd/gogent-sharp-edge/main.go`
- [ ] Use `pkg/routing.ParsePostToolEvent()` (existing, not new parser)
- [ ] Fix environment variable priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > CWD
- [ ] Remove build script (no new CLI being created)
- [ ] Update acceptance criteria (merged, single PostToolUse handler)
- [ ] Save updated file

**Key Change**: DO NOT create new CLI - merge logic into existing gogent-sharp-edge

---

## After All Tasks Complete

### Final Step: Update tickets-index.json
- [ ] Add entry for GOgent-063a (completed)
- [ ] Update GOgent-063 dependencies: add "GOgent-063a"
- [ ] Remove GOgent-070 entry (deleted)
- [ ] Update GOgent-072 title and dependencies (remove 070)
- [ ] Add entries for GOgent-073 and GOgent-074
- [ ] Validate JSON structure

---

## Quick Reference: What Each Ticket Needs

| Ticket | Main Change | Critical Detail |
|--------|-------------|-----------------|
| 065 | XDG paths | Replace `/tmp/` with `config.GetGOgentDir()` |
| 066 | Test schema | Use ACTUAL SubagentStop schema in JSON |
| 067 | Transcript parsing | Add metadata extraction step |
| 068 | Extend pkg/config | DO NOT create pkg/observability |
| 069 | Extend pkg/session | DO NOT create pkg/observability |
| 071 | Test isolation | Use `t.TempDir()` not global files |
| 072 | Merge CLI | Extend sharp-edge, NO new CLI |

---

## Verification Checklist (After Each Ticket)

- [ ] YAML frontmatter is valid (no syntax errors)
- [ ] Dependencies are correct
- [ ] Time estimates updated if changed
- [ ] Acceptance criteria count matches actual criteria
- [ ] BEFORE → AFTER changes match REFACTORING-MAP.md Section 2

---

**Location of Detailed Specs**: See REFACTORING-MAP.md Section 2 for complete BEFORE/AFTER code snippets for each ticket.
