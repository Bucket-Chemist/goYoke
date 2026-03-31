---
id: PERM-006
title: "Expand --allowedTools to include Bash"
status: pending
dependencies: [PERM-005]
time_estimate: "30min"
phase: 5b
tags: [plan-generated, phase-5b, driver, permission-model]
needs_planning: false
---

# PERM-006: Expand --allowedTools to include Bash

## Description

Expand the `--allowedTools` flag in `driver.go` to include all built-in tools
(including Bash). This tells the CLI to never internally deny any tool — the
`gogent-permission-gate` hook (registered in PERM-005) becomes the sole
gatekeeper for Bash.

**CRITICAL DEPENDENCY:** PERM-005 must be complete and active before this
ticket lands. Without the hook, Bash commands execute with zero gating.

## Acceptance Criteria

- [ ] `--allowedTools` in `driver.go:295` expanded to include Bash and all built-in tools
- [ ] `--disallowedTools Agent` flag remains unchanged
- [ ] `--permission-mode acceptEdits` remains unchanged
- [ ] Bash commands now trigger the permission gate hook (not CLI internal denial)
- [ ] Verify: comma-separated format works (A-4); if not, use multiple `--allowedTools` flags

## Files

- `internal/tui/cli/driver.go:295` — Expand `--allowedTools`

## Technical Spec

### Current (driver.go:295)

```go
args = append(args, "--allowedTools", "mcp__gofortress-interactive__*")
```

### New

```go
args = append(args, "--allowedTools",
    "Bash,Read,Write,Edit,Glob,Grep,WebSearch,WebFetch,NotebookEdit,"+
    "TodoWrite,EnterPlanMode,ExitPlanMode,Skill,ToolSearch,AskUserQuestion,"+
    "mcp__gofortress-interactive__*")
```

### Fallback (if comma-separated doesn't work)

```go
builtinTools := []string{
    "Bash", "Read", "Write", "Edit", "Glob", "Grep",
    "WebSearch", "WebFetch", "NotebookEdit", "TodoWrite",
    "EnterPlanMode", "ExitPlanMode", "Skill", "ToolSearch",
    "AskUserQuestion",
}
for _, t := range builtinTools {
    args = append(args, "--allowedTools", t)
}
args = append(args, "--allowedTools", "mcp__gofortress-interactive__*")
```

### Rollback

Revert this single line change. Bash reverts to CLI internal denial.

## Review Notes

- M-1: This is Phase 5b, deliberately after PERM-005 (Phase 5a)
- A-4: Test comma-separated format — if rejected, use multiple flags
- A-6: Verified — acceptEdits does not hard-deny Bash

---

_Generated from: Revised Architecture v2, Phase 5b_
