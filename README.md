# goYoke

**Programmatic enforcement for Claude Code agentic workflows.**

goYoke is a Go-based orchestration framework that wraps Claude Code with runtime hooks, tiered agent routing, and a terminal UI. Enforcement happens in compiled Go binaries that intercept Claude Code events — not in prompt instructions.

## Author

Created and maintained by [Dokter Smol](https://github.com/Bucket-Chemist)  

---

## What It Does

- **Hook enforcement** — Go binaries run at SessionStart, PreToolUse, PostToolUse, SubagentStop, and SessionEnd to validate, track, and gate Claude Code behavior
- **Agent routing** — 70+ agent definitions with tiered model selection (Haiku/Sonnet/Opus), automatic convention loading, and delegation validation
- **Terminal UI** — Bubbletea-based TUI that wraps Claude Code CLI with agent visualization, team orchestration, cost tracking, and session persistence
- **ML telemetry** — Append-only logging of routing decisions and agent collaborations for optimization analysis

## Architecture

Two-process topology: the Go TUI owns the terminal and spawns Claude Code CLI as a subprocess. An MCP server (also Go) provides agent spawning and interactive tools over a Unix domain socket side channel.

```
Go TUI (goyoke)
  |-- Bubbletea event loop
  |-- CLI Driver (manages Claude subprocess)
  |-- IPC Bridge (UDS)
  |
  +--spawns--> claude --output-format stream-json
                  |
                  +--spawns--> goyoke-mcp (Go MCP server)
                                  |
                                  +--connects--> TUI via UDS
```

24 Go binaries: 1 TUI, 1 MCP server, 11 hook/enforcement binaries, 11 utilities.

## Requirements

- Go 1.25+
- Claude Code CLI installed and authenticated
- Linux or macOS

## Build

```bash
make build      # Build TUI + all binaries
make install    # Install to ~/.local/bin
./bin/goyoke    # Launch TUI
```

## Project Structure

```
cmd/           24 binary packages (TUI, MCP server, hooks, utilities)
internal/tui/  Bubbletea UI components (23 packages)
pkg/           Shared libraries (routing, session, memory, telemetry, config)
defaults/      Embedded config for distributable builds (go:embed)
.claude/       Agent definitions, conventions, rules, schemas, skills
```

## Status

Active development. Core hook system, TUI, and agent routing are production-ready.

## License

Copyright 2025-2026 William Klare. All rights reserved.

Proprietary software. No license is granted to use, copy, modify, or distribute without explicit written permission from the author.
