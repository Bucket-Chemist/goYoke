# goYoke

**Programmatic enforcement for Claude Code agentic workflows.**

goYoke wraps Claude Code with compiled Go hooks, tiered agent routing, and a terminal UI. Enforcement happens in binaries that intercept Claude Code events at runtime — not in prompt instructions that degrade over long conversations.

## Author

Created and maintained by [Dokter Smol](https://github.com/Bucket-Chemist)  

---

## Installation

**Prerequisites:** [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated.

### Go install (recommended)

```bash
go install github.com/Bucket-Chemist/goYoke/cmd/...@latest
goyoke
```

### Standalone binary

Download the archive for your platform from [Releases](../../releases), extract to a directory on your `PATH`:

```bash
# Linux (amd64)
curl -L https://github.com/Bucket-Chemist/goYoke/releases/latest/download/goYoke_*_linux_amd64.tar.gz | tar xz -C ~/.local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/Bucket-Chemist/goYoke/releases/latest/download/goYoke_*_darwin_arm64.tar.gz | tar xz -C /usr/local/bin/

# Windows (PowerShell) — extract to a directory on your PATH
Invoke-WebRequest -Uri https://github.com/Bucket-Chemist/goYoke/releases/latest/download/goYoke_*_windows_amd64.zip -OutFile goYoke.zip
Expand-Archive goYoke.zip -DestinationPath "$env:LOCALAPPDATA\goYoke"
# Add $env:LOCALAPPDATA\goYoke to your PATH
```

### Build from source

```bash
git clone https://github.com/Bucket-Chemist/goYoke.git
cd goYoke
make dist       # Generate embedded defaults + build all binaries
make install    # Install to ~/.local/bin
```

---

## What It Does

- **Hook enforcement** — 11 Go binaries intercept Claude Code events (SessionStart, PreToolUse, PostToolUse, SubagentStop, SessionEnd) to validate, track, and gate behavior
- **Agent routing** — 46 agent definitions with tiered model selection (Haiku → Sonnet → Opus), automatic convention loading, and delegation validation
- **Multi-agent orchestration** — Team-based workflows with parallel wave execution, background task collection, and cost attribution
- **Terminal UI** — Bubbletea TUI wrapping Claude Code CLI with agent visualization, team progress, cost tracking, and session persistence
- **Convention system** — Language-specific coding conventions auto-loaded based on file context (Go, Python, R, Rust, TypeScript, React)
- **ML telemetry** — Append-only logging of routing decisions, agent collaborations, and sharp edge patterns for optimization

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Go TUI (goyoke)                       │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────────┐  │
│  │Bubbletea │  │ CLI Driver │  │     IPC Bridge       │  │
│  │event loop│  │ (pipes)    │  │ (UDS: goyoke-{pid})  │  │
│  └──────────┘  └─────┬─────┘  └──────────┬───────────┘  │
└────────────────────── │ ─────────────────── │ ───────────┘
                        │                     │
              spawns    │                     │ NDJSON
                        ▼                     │
          ┌──────────────────────┐            │
          │   Claude Code CLI    │            │
          │  --output-format     │            │
          │   stream-json        │            │
          └──────────┬───────────┘            │
                     │                        │
           spawns    │ MCP stdio              │
                     ▼                        │
          ┌──────────────────────┐            │
          │  goyoke-mcp          │ ◄──────────┘
          │  (MCP server)        │  UDS side channel
          │                      │
          │  spawn_agent         │──► claude -p (subagent)
          │  team_run            │──► goyoke-team-run (background)
          │  ask_user            │──► TUI modal (via UDS)
          └──────────────────────┘
                     │
          hooks fire on every Claude Code event
                     │
    ┌────────────────┼────────────────────────┐
    ▼                ▼                        ▼
┌────────┐  ┌──────────────┐  ┌──────────────────────┐
│load-   │  │  validate    │  │  sharp-edge          │
│context │  │  skill-guard │  │  agent-endstate      │
│        │  │  direct-impl │  │  orchestrator-guard  │
│Session │  │  permission  │  │  archive             │
│Start   │  │  Pre/Post    │  │  config-guard        │
│        │  │  ToolUse     │  │  instructions-audit  │
└────────┘  └──────────────┘  └──────────────────────┘
```

### Two-Process Topology

The TUI owns the terminal. It spawns Claude Code CLI as a subprocess, communicating via stream-JSON pipes. Claude Code spawns the MCP server (`goyoke-mcp`), which connects back to the TUI via a Unix domain socket for interactive tools (modals, confirmations, agent status updates).

### Layered Config Resolution

```
User disk (~/.claude/)          Embedded defaults (go:embed)
        │                                │
        └──────────┬─────────────────────┘
                   ▼
            pkg/resolve.Resolver
            ├── ReadFile()    → first-found (user wins)
            ├── ReadDir()     → union (both layers)
            ├── ReadFileAll() → merge (agents-index.json)
            └── HasFile()     → any layer
```

Zero-install: `go install` produces working binaries with embedded agent definitions, conventions, rules, and schemas. User files in `~/.claude/` override or extend embedded defaults.

---

## 25 Binaries

| Category | Count | Binaries | Embedded Config |
|----------|-------|----------|-----------------|
| **TUI** | 1 | `goyoke` | Yes |
| **MCP Server** | 1 | `goyoke-mcp` | Yes |
| **Orchestration** | 1 | `goyoke-team-run` | Yes |
| **Hooks** | 11 | `goyoke-load-context`, `goyoke-validate`, `goyoke-skill-guard`, `goyoke-sharp-edge`, `goyoke-direct-impl-check`, `goyoke-permission-gate`, `goyoke-agent-endstate`, `goyoke-orchestrator-guard`, `goyoke-archive`, `goyoke-config-guard`, `goyoke-instructions-audit` | 2 Yes, 9 No |
| **Utilities** | 11 | `goyoke-aggregate`, `goyoke-scout`, `goyoke-codebase-extract`, `goyoke-ml-export`, `goyoke-plan-impl`, `goyoke-capture-intent`, `goyoke-team-prepare-synthesis`, `goyoke-update-review-outcome`, `goyoke-validate-schemas`, `goyoke-version`, `goyoke-log-review` | No |

Hook binaries without embedded config degrade gracefully — they log warnings but continue operating on whatever local state is available.

---

## Agent System

46 agents organized into 4 tiers, selected automatically based on task complexity:

| Tier | Model | Cost | Use Case | Example Agents |
|------|-------|------|----------|----------------|
| **1** | Haiku | $0.0005/1K | File search, scope assessment | `codebase-search`, `haiku-scout` |
| **1.5** | Haiku+Thinking | $0.001/1K | Scaffolding, docs, review | `scaffolder`, `tech-docs-writer`, `code-reviewer` |
| **2** | Sonnet | $0.009/1K | Implementation, refactoring | `go-pro`, `python-pro`, `rust-pro`, `react-pro` |
| **3** | Opus | $0.045/1K | Architecture, deep analysis | `architect`, `planner`, `einstein`, `beethoven` |

### Multi-Agent Workflows

| Workflow | Agents | Purpose |
|----------|--------|---------|
| `/braintrust` | Mozart → Einstein + Staff-Architect → Beethoven | Multi-perspective deep analysis |
| `/implement` | Architect → team-run (parallel workers) | Plan and implement features |
| `/review` | 4 parallel reviewers → synthesis | Multi-domain code review |
| `/plan-tickets` | Scout → Planner → Architect → Review → Tickets | Full planning pipeline |
| `/ticket` | Select → Validate → Plan → Implement → Verify | Ticket-driven development |

---

## Hook System

Hooks are compiled Go binaries registered in Claude Code's hook config. They fire deterministically on every event — no prompt-based enforcement that degrades over context.

| Event | Hook | What It Enforces |
|-------|------|-----------------|
| SessionStart | `goyoke-load-context` | Loads routing schema, conventions, git context, session handoff |
| PreToolUse | `goyoke-validate` | Blocks unauthorized Task(opus), validates subagent_type |
| PreToolUse | `goyoke-skill-guard` | Tool allowlist enforcement during active skills |
| PreToolUse | `goyoke-direct-impl-check` | Warns when router writes code instead of delegating |
| PreToolUse | `goyoke-permission-gate` | Gates Bash commands against permission rules |
| PostToolUse | `goyoke-sharp-edge` | Tool counting, routing reminders, failure tracking, ML telemetry |
| SubagentStop | `goyoke-agent-endstate` | Records decision outcomes, logs collaborations |
| SubagentStop | `goyoke-orchestrator-guard` | Blocks completion when background tasks uncollected |
| SessionEnd | `goyoke-archive` | Generates handoff summary, archives metrics |
| ConfigChange | `goyoke-config-guard` | Validates config changes against schema |

---

## Convention System

Language-specific coding conventions auto-loaded based on file context:

| Language | Conventions | Specialized |
|----------|-------------|-------------|
| Go | `go.md` | `go-cobra.md` (CLI), `go-bubbletea.md` (TUI) |
| Python | `python.md` | Domain-specific extensions |
| R | `R.md` | `R-shiny.md`, `R-golem.md` |
| Rust | `rust.md` | — |
| TypeScript | `typescript.md` | `react.md` |

---

## Project Structure

```
cmd/                    25 binary entry points
internal/tui/           Bubbletea UI (23 component packages)
internal/hooks/         Hook library implementations
internal/codemap/       Codebase extraction engine
pkg/resolve/            Layered config resolution (ReadFile/ReadDir/ReadFileAll)
pkg/routing/            Agent index, schema, conventions, identity loading
pkg/session/            Session lifecycle, handoffs, context response
pkg/config/             Tool counter, guard files, XDG paths
pkg/memory/             Sharp edge pattern matching
pkg/telemetry/          ML telemetry logging
pkg/enforcement/        Documentation theater detection
defaults/               Embedded config for zero-install (go:embed)
scripts/                Build, test, and distribution scripts
test/                   Integration tests, simulation harness, regression
```

---

## Requirements

- **Go 1.25+** (for building from source)
- **Claude Code CLI** installed and authenticated
- **Linux or macOS** (Windows: builds and runs, TUI requires Windows Terminal)

---

## Development

```bash
make build          # Build TUI + all hook binaries
make test           # Run full test suite
make defaults       # Generate embedded config from source
make dist           # defaults + build all binaries
make test-defaults  # 18-point smoke test on generated defaults
make test-zero-install  # E2E zero-install integration test
make check-size     # Report embedded binary sizes
```

---

## License

[MIT](LICENSE) - Copyright (c) 2025-2026 William Klare
