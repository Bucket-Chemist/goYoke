---
id: PERM-002
title: "Bridge Extension: handlePermGate + dispatch case"
status: pending
dependencies: [PERM-001]
time_estimate: "1h"
phase: 2
tags: [plan-generated, phase-2, bridge, ipc]
needs_planning: false
---

# PERM-002: Bridge Extension — handlePermGate + dispatch case

## Description

Extend the IPC bridge (`bridge/server.go`) to handle `permission_gate_request`
messages from the hook binary. Add a new dispatch case, a handler that blocks
on a response channel, and a `ResolvePermGate` method for the TUI to call when
the user responds to the modal.

## Acceptance Criteria

- [ ] `dispatch()` has `case mcp.TypePermGateRequest:` routing to `handlePermGate`
- [ ] `pendingPermGates map[string]chan mcp.PermGateResponsePayload` field added to `IPCBridge`
- [ ] `handlePermGate(req, enc)` sends `CLIPermissionRequestMsg` into Bubbletea event loop, then blocks on channel
- [ ] `ResolvePermGate(requestID, decision)` sends to the pending channel
- [ ] Unit test: mock sender → handlePermGate blocks → ResolvePermGate unblocks → response sent
- [ ] Timeout: if channel not resolved within `timeout_ms`, auto-deny and clean up

## Files

- `internal/tui/bridge/server.go` — Add dispatch case, handler, resolver, pending map

## Technical Spec

### dispatch() addition

```go
case mcp.TypePermGateRequest:
    b.handlePermGate(req, enc)
```

### Handler flow

```go
func (b *IPCBridge) handlePermGate(req mcp.IPCRequest, enc *json.Encoder) {
    var pgReq mcp.PermGateRequest
    // unmarshal req into pgReq

    ch := make(chan mcp.PermGateResponsePayload, 1)
    b.mu.Lock()
    b.pendingPermGates[pgReq.ID] = ch
    b.mu.Unlock()

    // Inject CLIPermissionRequestMsg into Bubbletea
    b.program.Send(model.CLIPermissionRequestMsg{
        RequestID: pgReq.ID,
        ToolName:  pgReq.Payload.ToolName,
        ToolInput: pgReq.Payload.ToolInput,
        TimeoutMS: pgReq.Payload.TimeoutMS,
    })

    // Block until response or timeout
    timeout := time.Duration(pgReq.Payload.TimeoutMS) * time.Millisecond
    select {
    case resp := <-ch:
        enc.Encode(mcp.PermGateResponse{...resp})
    case <-time.After(timeout):
        enc.Encode(mcp.PermGateResponse{...deny with timeout reason})
    }

    // Cleanup
    b.mu.Lock()
    delete(b.pendingPermGates, pgReq.ID)
    b.mu.Unlock()
}
```

### Resolver

```go
func (b *IPCBridge) ResolvePermGate(requestID string, decision string) {
    b.mu.Lock()
    ch, ok := b.pendingPermGates[requestID]
    b.mu.Unlock()
    if ok {
        ch <- mcp.PermGateResponsePayload{Decision: decision}
    }
}
```

## Review Notes

- M-3 from review: dispatch case was missing in v1 plan — explicitly included here
- `pendingPermGates` is separate from `pendingModals` to avoid cross-contamination (Decision Log)
- Bridge goroutine-per-connection model handles the blocking wait cleanly (verified in review)

---

_Generated from: Revised Architecture v2, Phase 2_
