---
id: PERM-001
title: "IPC Protocol Extension: permission_gate types"
status: pending
dependencies: []
time_estimate: "30min"
phase: 1
tags: [plan-generated, phase-1, ipc, types]
needs_planning: false
---

# PERM-001: IPC Protocol Extension — permission_gate types

## Description

Add IPC message types for the permission gate protocol. These types define the
contract between the `gogent-permission-gate` hook binary and the TUI's IPC
bridge.

## Acceptance Criteria

- [ ] `TypePermGateRequest` and `TypePermGateResponse` constants added to `internal/tui/mcp/tools.go`
- [ ] `PermGateRequest` and `PermGateResponse` structs defined in new file `internal/tui/mcp/perm_gate.go`
- [ ] Round-trip serialization test passes (marshal → unmarshal → compare)
- [ ] `decision` field supports values: `"allow"`, `"deny"`, `"allow_session"`

## Files

- `internal/tui/mcp/tools.go` — Add type constants
- `internal/tui/mcp/perm_gate.go` — **NEW** — Request/response types

## Technical Spec

### Type Constants (tools.go)

```go
const TypePermGateRequest  = "permission_gate_request"
const TypePermGateResponse = "permission_gate_response"
```

### Request Type (perm_gate.go)

```go
type PermGateRequestPayload struct {
    ToolName  string          `json:"tool_name"`
    ToolInput json.RawMessage `json:"tool_input"`
    SessionID string          `json:"session_id"`
    TimeoutMS int             `json:"timeout_ms"`
}

type PermGateRequest struct {
    Type    string                  `json:"type"`
    ID      string                  `json:"id"`
    Payload PermGateRequestPayload  `json:"payload"`
}
```

### Response Type (perm_gate.go)

```go
type PermGateResponse struct {
    Type    string                   `json:"type"`
    ID      string                   `json:"id"`
    Payload PermGateResponsePayload  `json:"payload"`
}

type PermGateResponsePayload struct {
    Decision string `json:"decision"` // "allow", "deny", "allow_session"
}
```

## Context

These types are consumed by:
- PERM-002 (bridge dispatch and handling)
- PERM-004 (hook binary UDS client)

---

_Generated from: Revised Architecture v2, Phase 1_
