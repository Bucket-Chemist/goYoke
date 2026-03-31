---
id: PERM-004
title: "Hook Binary: gogent-permission-gate"
status: pending
dependencies: [PERM-001]
time_estimate: "2h"
phase: 4
tags: [plan-generated, phase-4, hook, go-binary]
needs_planning: false
---

# PERM-004: Hook Binary — gogent-permission-gate

## Description

Implement the PreToolUse hook binary that gates Bash commands through the TUI
permission modal. The binary reads the tool event from stdin, classifies it
against the permission policy, and either auto-allows or contacts the TUI via
UDS for user approval.

## Acceptance Criteria

- [ ] `cmd/gogent-permission-gate/main.go` — Entry point: read stdin, classify, gate
- [ ] `cmd/gogent-permission-gate/policy.go` — Permission policy loading and classification
- [ ] `cmd/gogent-permission-gate/cache.go` — Session permission cache (file-based)
- [ ] `cmd/gogent-permission-gate/uds.go` — UDS client for bridge communication
- [ ] `cmd/gogent-permission-gate/main_test.go` — Tests
- [ ] Binary builds to `bin/gogent-permission-gate`
- [ ] Auto-allow for all tools except Bash (default: auto_allow)
- [ ] For Bash: check session cache first, then contact UDS bridge
- [ ] Timeout handling: configurable via `GOGENT_PERM_TIMEOUT` (default 30s)
- [ ] On UDS failure: auto-deny with descriptive reason
- [ ] On timeout: auto-deny with "Permission request timed out"

## Files

- `cmd/gogent-permission-gate/main.go` — **NEW**
- `cmd/gogent-permission-gate/policy.go` — **NEW**
- `cmd/gogent-permission-gate/cache.go` — **NEW**
- `cmd/gogent-permission-gate/uds.go` — **NEW**
- `cmd/gogent-permission-gate/main_test.go` — **NEW**
- `Makefile` or build script — Add build target

## Technical Spec

### Permission Policy

```json
{
  "auto_allow": [
    "Read", "Glob", "Grep", "TodoWrite", "EnterPlanMode",
    "ExitPlanMode", "WebSearch", "WebFetch", "ToolSearch",
    "AskUserQuestion", "Skill", "Write", "Edit", "NotebookEdit"
  ],
  "needs_approval": ["Bash"],
  "skip": ["Task", "Agent"],
  "default": "auto_allow"
}
```

Note: With `"Bash"` matcher in settings.json (PERM-005), this hook only fires
for Bash anyway. The policy is a safety net for if the matcher is broadened.

### Main Flow

```
1. Read tool event JSON from stdin
2. Parse tool_name
3. Check policy classification:
   - auto_allow or skip → print "{}" to stdout, exit 0
   - needs_approval → continue
4. Check session cache ($XDG_RUNTIME_DIR/gofortress-perm-cache-{session_id}.json)
   - Cached allow → print "{}" to stdout, exit 0
5. Connect to GOFORTRESS_SOCKET (UDS)
6. Send permission_gate_request
7. Block until response (with timeout)
8. Response = allow → print "{}" to stdout
   Response = deny → print {"decision":"block","reason":"User denied: Bash"}
   Response = allow_session → write to cache, print "{}"
```

### Stdin Format (from CLI)

```json
{
  "tool_name": "Bash",
  "tool_input": {"command": "ls -la /tmp", "description": "List tmp"},
  "session_id": "session-uuid"
}
```

### Stdout Format (to CLI)

Allow: `{}`
Block: `{"decision":"block","reason":"User denied: Bash command: ls -la /tmp"}`

### Error Handling

| Error | Behavior |
|-------|----------|
| GOFORTRESS_SOCKET not set | Auto-deny: "No TUI bridge available" |
| UDS connect failure | Auto-deny: "UDS connection failed: <error>" |
| UDS connect refused (stale socket) | Auto-deny: "Bridge not reachable" |
| Timeout waiting for response | Auto-deny: "Permission request timed out" |
| Malformed stdin | Auto-deny: "Failed to parse tool event" |

## Tests

1. Stdin parsing: valid JSON → correct tool_name extraction
2. Policy classification: Bash → needs_approval, Write → auto_allow, Unknown → auto_allow
3. Session cache: write "allow_session" → subsequent check returns cached allow
4. **Negative:** Invalid GOFORTRESS_SOCKET → auto-deny with descriptive error (m-2)
5. **Negative:** Stale socket (connect refused) → auto-deny
6. Timeout: mock slow UDS → auto-deny after timeout

## Review Notes

- C-1: Only Bash in needs_approval (Write/Edit/NotebookEdit moved to auto_allow)
- m-3: Default is auto_allow — unknown tools pass through
- m-2: Negative tests for bad socket included

---

_Generated from: Revised Architecture v2, Phase 4_
