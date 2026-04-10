# Claude CLI Flags Reference — GOgent-Fortress

**Created:** 2026-03-25
**Last Updated:** 2026-03-25
**Context:** Major debugging session uncovered multiple flag-related failures. This document exists to future-proof against claude CLI updates that break spawn_agent or the Go TUI.

---

## Critical Flags for Go TUI

### TUI Driver (`internal/tui/cli/driver.go:buildArgs()`)

The Go TUI launches `claude` as a subprocess in interactive mode (not `-p`):

```
claude --input-format stream-json --output-format stream-json --verbose
       --include-partial-messages --permission-mode acceptEdits
       [--mcp-config /tmp/gofortress-mcp-*.json]
       [--allowedTools mcp__gofortress-interactive__*]
       [--resume <session-id>] [--model <model>]
```

| Flag | Required? | Why | Broke when |
|------|-----------|-----|------------|
| `--output-format stream-json` | Yes | TUI parses continuous NDJSON event stream | — |
| `--verbose` | **Yes (since 2.1.81)** | Required when using `--output-format stream-json` | claude 2.1.81 added this requirement; without it: `Error: When using --print, --output-format=stream-json requires --verbose` |
| `--input-format stream-json` | Yes | TUI sends JSON messages over stdin | — |
| `--include-partial-messages` | Yes | Get streaming updates during generation | — |
| `--permission-mode` | Yes | Default is "default" which asks user; TUI needs `acceptEdits` | — |
| `--mcp-config` | Optional | Points to temp file with gofortress-interactive MCP server | MCP server name mismatch broke tool discovery |
| `--allowedTools` | Optional | Pattern must match MCP server name in config | Was `mcp__gofortress__*` but needed `mcp__gofortress-interactive__*` |

### Flags that DO NOT exist

| Flag | What we tried | What works instead |
|------|---------------|-------------------|
| `--config-dir` | Tried passing to override config directory | Use `CLAUDE_CONFIG_DIR` environment variable |

### Spawner (`internal/tui/mcp/tools.go:buildSpawnArgs()`)

Spawned agents use `-p` (print/pipe mode) for one-shot execution:

```
claude -p --output-format json --permission-mode bypassPermissions
       --model <model> [--allowedTools <tools>] [--max-budget-usd <n>]
```

| Flag | Required? | Why | Broke when |
|------|-----------|-----|------------|
| `-p` | Yes | One-shot print mode | — |
| `--output-format json` | Yes | Single JSON result object | Was `stream-json` which hangs waiting; `json` is simpler for one-shot |
| `--permission-mode bypassPermissions` | **Yes** | `-p` mode has no terminal for permission approval | Without it, Write/Edit operations block forever |
| `--model` | Yes | Explicit model selection for the agent | — |

**DO NOT USE** for spawned agents:
- `--output-format stream-json` — requires `--verbose`, produces NDJSON instead of single JSON
- `--verbose` — not needed with `json` format
- `--input-format stream-json` — only for interactive TUI mode

---

## Environment Variables

| Variable | Set where | Used by | Purpose |
|----------|-----------|---------|---------|
| `CLAUDE_CONFIG_DIR` | `cmd/gofortress/main.go:88` | claude CLI subprocess | Override config directory (e.g. `~/.claude-em`) |
| `GOFORTRESS_SOCKET` | `cmd/gofortress/main.go:315` | gofortress-mcp binary | UDS path for IPC bridge to TUI |
| `GOGENT_NESTING_LEVEL` | `internal/tui/mcp/spawner.go:71` | spawned agents | Prevent infinite nesting |
| `GOGENT_PARENT_AGENT` | `internal/tui/mcp/spawner.go:72` | spawned agents | Tree hierarchy linkage |

---

## MCP Server Naming

| Config key | Binary | Tool prefix | Used by |
|------------|--------|-------------|---------|
| `gofortress-interactive` | `bin/gofortress-mcp` | `mcp__gofortress-interactive__*` | Go TUI (via `--mcp-config`) |

**Critical:** The config key in `writeMCPConfig()` determines the tool prefix. CLAUDE.md tells the LLM to call `mcp__gofortress-interactive__spawn_agent`. If the config key doesn't match, the LLM can't find the tool.

---

## Binary Management

All binaries output to `bin/` (fixed 2026-03-25):

```makefile
build-go-tui:  -o bin/gofortress
build-go-mcp:  -o bin/gofortress-mcp
```

**`findMCPBinary()`** searches: same dir as TUI binary → `bin/` subdir → `../bin/` → PATH.
Since both are now in `bin/`, the first candidate always finds the fresh binary.

Run `make clean-stale` to remove any leftover root binaries from older builds.

---

## Future-Proofing Checklist

When claude CLI updates, check:

- [ ] Does `--output-format stream-json` still work with `--verbose`?
- [ ] Does `--output-format json` still work for `-p` mode?
- [ ] Does `--permission-mode bypassPermissions` still exist?
- [ ] Does `CLAUDE_CONFIG_DIR` env var still work?
- [ ] Are there new required flags for `--input-format stream-json`?
- [ ] Has the MCP server startup behavior changed?
- [ ] Run `claude --help 2>&1 | grep -i 'output-format\|permission\|config\|verbose'`

---

## Bug History (2026-03-25 Session)

| Bug | Root Cause | Fix |
|-----|-----------|-----|
| spawn_agent "is a stub" | MCP server name `gofortress` didn't match CLAUDE.md's `gofortress-interactive` | Changed `writeMCPConfig()` key to `gofortress-interactive` |
| gofortress-EM-go: CLI never called | `--config-dir` is not a valid claude CLI flag | Removed; use `CLAUDE_CONFIG_DIR` env var |
| gofortress-EM-go: blank screen | `--output-format stream-json` requires `--verbose` since 2.1.81 | Added `--verbose` to `buildArgs()` |
| Spawned agents blocked on Write | `-p` mode can't prompt for permissions | Added `--permission-mode bypassPermissions` |
| Spawned agents hang | `--output-format stream-json` hangs in one-shot mode | Changed to `--output-format json` |
| Stale binary picked up | Makefile output TUI to root, MCP to `bin/` | All outputs to `bin/`, added `clean-stale` target |
| Agent tree empty | `handleAgentRegistryMsg()` discarded message data | Added `registry.Register()` calls |
| Agents orphaned in tree | Empty `ParentID` → not linked to root | Default to `rootAgentID` (matches TS TUI) |
| Tab keys (Alt+C/A/T/Y) didn't work | Key handler returned before reaching tab bar | Added tab keys to global switch |
| Text input unfocused on startup | `syncFocusState()` never called at startup | Added to `handleWindowSize()` |
