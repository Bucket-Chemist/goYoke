---
id: PERM-003
title: "TUI Modal Integration: Permission flow for CLI tool requests"
status: pending
dependencies: [PERM-002]
time_estimate: "1.5h"
phase: 3
tags: [plan-generated, phase-3, tui, modal, bubbletea]
needs_planning: false
---

# PERM-003: TUI Modal Integration — Permission flow for CLI tool requests

## Description

Wire the `CLIPermissionRequestMsg` from the bridge into the TUI's existing
Permission modal system. Add a new `FlowToolPermission` flow type, handle the
message in the Update switch, and route the modal response back to the bridge
via `ResolvePermGate`.

## Acceptance Criteria

- [ ] `CLIPermissionRequestMsg` type added to `model/messages.go`
- [ ] `FlowToolPermission` constant added to `modals/permission.go`
- [ ] `case CLIPermissionRequestMsg:` in `app.go` Update switch
- [ ] `handleCLIPermissionRequest()` in `ui_event_handlers.go` — enqueues Permission modal
- [ ] Modal shows: "Tool Permission Required", tool name, rendered tool input, Allow/Deny/Allow for Session
- [ ] `handleModalResponse` for `FlowToolPermission` calls `bridge.ResolvePermGate(requestID, decision)`
- [ ] Modal auto-cancels after `TimeoutMS` (driven by `ModalRequest.TimeoutMS`)

## Files

- `internal/tui/model/messages.go` — Add `CLIPermissionRequestMsg`
- `internal/tui/model/app.go` — Add case in Update switch (~line 364)
- `internal/tui/model/ui_event_handlers.go` — Add `handleCLIPermissionRequest()`
- `internal/tui/components/modals/permission.go` — Add `FlowToolPermission`, build flow/request

## Technical Spec

### Message Type

```go
type CLIPermissionRequestMsg struct {
    RequestID string
    ToolName  string
    ToolInput json.RawMessage
    TimeoutMS int
}
```

### Modal Content

- **Header:** "Tool Permission Required"
- **Message:** Tool name + rendered input (for Bash: the command string)
- **Options:** `["Allow", "Deny", "Allow for Session"]`
- Uses existing `Permission` ModalType (orange warning border, Allow/Deny labels)

### Response Routing

`ModalResponseMsg` → `handleModalResponse` → for `FlowToolPermission`:
1. Map button index to decision: 0="allow", 1="deny", 2="allow_session"
2. Call `bridge.ResolvePermGate(requestID, decision)`

## Context

- Reuses the existing Permission ModalType and PermissionHandler pattern
- ModalQueue handles sequential presentation (only one modal at a time)
- The bridge blocks until ResolvePermGate is called — the modal response unblocks it

---

_Generated from: Revised Architecture v2, Phase 3_
