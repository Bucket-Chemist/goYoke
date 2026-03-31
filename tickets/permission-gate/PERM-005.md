---
id: PERM-005
title: "Hook Registration in settings.json"
status: pending
dependencies: [PERM-004]
time_estimate: "15min"
phase: 5a
tags: [plan-generated, phase-5a, config, hook-registration]
needs_planning: false
---

# PERM-005: Hook Registration in settings.json

## Description

Register `gogent-permission-gate` as a PreToolUse hook in settings.json with a
`"Bash"` matcher. This must land BEFORE the allowedTools expansion (PERM-006)
to prevent a window where Bash runs ungated.

## Acceptance Criteria

- [ ] PreToolUse hook entry added to settings.json
- [ ] Matcher is `"Bash"` (not `"*"`) — avoids spawning process for every tool call
- [ ] Hook fires when Bash tool is invoked (even though CLI currently denies Bash internally — harmless)
- [ ] Existing hooks (gogent-validate, gogent-sharp-edge) unaffected

## Files

- `~/.claude/settings.json` (or `settings.local.json`) — Add hook entry

## Technical Spec

### Exact settings.json Entry

Add to the existing `PreToolUse` array:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "command": "bin/gogent-permission-gate",
        "matcher": "Bash"
      }
    ]
  }
}
```

### Ordering

- Register AFTER `gogent-validate` in the hooks array
- `gogent-validate` handles Task/Agent validation
- `gogent-permission-gate` handles Bash permission gating
- Both fire for their respective matchers; no conflict

### Safety Note

At this point, the CLI still internally denies Bash (it's not in `--allowedTools`).
The hook fires but the CLI has already decided to deny, so the hook's decision
is moot. This is the safe ordering — hook exists but doesn't matter yet.

PERM-006 flips the switch by adding Bash to `--allowedTools`.

## Review Notes

- M-1: This ticket exists specifically to ensure hook registration happens before allowedTools expansion
- M-2: Exact settings.json entry specified (was missing in v1)
- Matcher is `"Bash"` only, not `"*"` — efficient, matches Bash-only scope

---

_Generated from: Revised Architecture v2, Phase 5a_
