---
title: TUI Architecture
type: concept
created: 2026-04-18
tags: [tui, bubbletea, go]
related: [ipc-bridge, agent-spawning, session-lifecycle]
---

# TUI Architecture

The goYoke TUI is a pure Go application built with Charmbracelet Bubbletea v1.3.10, using the Elm Architecture (Model-Update-View). It owns the terminal and manages Claude Code as a subprocess.

## Two-Process Topology

```
Go TUI Process (single binary: cmd/goyoke/main.go)
  |-- Bubbletea event loop (owns terminal stdin/stdout)
  |-- CLIDriver (manages Claude CLI subprocess via pipes)
  |-- IPCBridge (UDS listener for MCP server communication)
  |
  +--spawns--> Claude Code CLI (--output-format stream-json)
                  |
                  +--spawns--> goyoke-mcp (Go MCP server, stdio transport)
                                  |
                                  +--connects--> TUI via UDS side channel
```

## SharedState

Components access shared state via `*state.SharedState` pointer. Bubbletea uses value receivers, so the pointer pattern ensures state survives `tea.Program` copies. Defined in `internal/tui/state/provider.go`.

## Component Structure

Components live in `internal/tui/components/` and implement `tea.Model` (Init, Update, View). 29 packages total, including settingstree, slashcmd, search, hintbar, breadcrumb, skeleton.

## Key Files

- `cmd/goyoke/main.go` — Entry point
- `internal/tui/model/app.go` — Root AppModel
- `internal/tui/cli/driver.go` — CLIDriver subprocess management
- `internal/tui/state/provider.go` — SharedState

## History

70 tickets across 10 phases (TUI-001 to TUI-070). See [[ARCHITECTURE#15. TUI History]] for timeline.

## See Also

- [[ipc-bridge]] — UDS communication with MCP server
- [[ARCHITECTURE#16. TUI Architecture (Go/Bubbletea — Current)]] — Full reference
