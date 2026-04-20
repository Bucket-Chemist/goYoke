---
title: IPC Bridge
type: concept
created: 2026-04-18
tags: [ipc, uds, mcp, tui]
related: [tui-architecture, agent-spawning]
---

# IPC Bridge

The TUI and MCP server communicate via a Unix domain socket (UDS) side channel. This enables the MCP server (running inside the Claude Code subprocess tree) to trigger UI interactions in the TUI process.

## Architecture

```
goyoke-mcp (MCP server, stdio transport to Claude CLI)
    |
    +-- UDSClient (connects to GOYOKE_SOCKET env var)
    |
    v
IPCBridge (internal/tui/bridge/server.go)
    |-- listens on $XDG_RUNTIME_DIR/goyoke-{pid}.sock
    |-- injects tea.Msg into Bubbletea event loop via program.Send()
    v
AppModel.Update() → UI renders
```

## Protocol

Newline-delimited JSON (NDJSON) over persistent UDS connection.

**MCP → TUI:** `modal_request`, `agent_register`, `agent_update`, `agent_activity`, `permission_gate_request`, `team_update`, `toast`

**TUI → MCP:** `modal_response`, `permission_gate_response`

## Key Files

- `internal/tui/bridge/server.go` — IPCBridge UDS listener
- `internal/tui/mcp/protocol.go` — IPC message types and constants
- `internal/tui/mcp/tools.go` — UDSClient that connects to the bridge

## See Also

- [[tui-architecture]] — Two-process topology overview
- [[ARCHITECTURE#16. TUI Architecture (Go/Bubbletea — Current)]] — Full TUI section
