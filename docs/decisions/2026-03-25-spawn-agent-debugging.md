# Decision Log: spawn_agent End-to-End Debugging

**Date:** 2026-03-25
**Session:** ~3 hours intensive debugging + feature work
**Trigger:** spawn_agent returning "stub" error in Go TUI; cascading failures across MCP, CLI flags, agent tree, and UI
**Outcome:** Full pipeline working — spawn_agent → agent tree → detail panel → status line

---

## Context

The Go TUI replaced the TypeScript TUI as the default frontend. The `spawn_agent` MCP tool was fully implemented in code but had never worked end-to-end due to a cascade of naming mismatches, invalid CLI flags, missing permission modes, and broken UI wiring.

## Decisions Made

### D-1: MCP Server Name — `gofortress-interactive`

**Decision:** The Go TUI's MCP server config key is `gofortress-interactive` (matching the TS TUI's in-process server name).

**Rationale:** CLAUDE.md instructs the LLM to call `mcp__gofortress-interactive__spawn_agent`. The config key determines the tool prefix. Using any other name causes tool-not-found.

**Files:** `cmd/gofortress/main.go:453`, `internal/tui/cli/driver.go:287`

### D-2: No `--config-dir` CLI flag

**Decision:** Config directory override uses `CLAUDE_CONFIG_DIR` environment variable only, not a CLI flag.

**Rationale:** `claude --config-dir` returns `error: unknown option`. The env var is the only supported mechanism.

**Files:** `cmd/gofortress/main.go:88`, `internal/tui/cli/driver.go:267-270`

### D-3: `--verbose` mandatory for stream-json

**Decision:** Always include `--verbose` in the TUI driver's args.

**Rationale:** claude CLI 2.1.81 added a requirement that `--output-format stream-json` needs `--verbose`. Without it, the process dies silently.

**Files:** `internal/tui/cli/driver.go:262-265`

### D-4: `--output-format json` for spawned agents

**Decision:** Use `json` (not `stream-json`) for one-shot `-p` spawns.

**Rationale:** `stream-json` requires `--verbose` and produces NDJSON which is harder to parse for one-shot results. The TS TUI also uses `json`. Simpler, more reliable, matches reference implementation.

**Files:** `internal/tui/mcp/tools.go:872`, `cmd/gofortress-mcp-standalone/spawner.go:76`

### D-5: `--permission-mode bypassPermissions` for spawned agents

**Decision:** All spawned agents run with `bypassPermissions`.

**Rationale:** `-p` mode has no interactive terminal. Without explicit permission mode, Write/Edit operations block forever waiting for user approval that can never arrive.

**Files:** `internal/tui/mcp/tools.go:872`, `cmd/gofortress-mcp-standalone/spawner.go:76`

### D-6: All binaries to `bin/`

**Decision:** Makefile outputs both `gofortress` and `gofortress-mcp` to `bin/`. No binaries at project root.

**Rationale:** `findMCPBinary()` searches same-dir-as-TUI first. When TUI was at root and MCP was in `bin/`, stale root MCP binaries were found first. Unified output prevents this.

**Files:** `Makefile:108-109,114`, `~/.local/bin/zellij-gofortress-tui-go:28`

### D-7: Default empty ParentID to root agent

**Decision:** When `AgentRegisteredMsg` arrives with empty `ParentID`, default it to the registry's `RootID()`.

**Rationale:** Without a parent, agents are orphaned — counted by `Count()` but invisible in `Tree()` (DFS from root never reaches them). The TS TUI does the same (`spawnAgent.ts:232`).

**Guard:** Don't self-reference (agent can't be its own parent).

**Files:** `internal/tui/model/ui_event_handlers.go:194-197`

### D-8: Status line matches TS TUI layout

**Decision:** Two-row layout with left/right split:
- Row 1: `[model] [perm] 📁 project | 🌿 branch` ←→ `auth`
- Row 2: `ctx:[bar] % | $cost` ←→ `⏱ elapsed | ↻ streaming`

**Rationale:** User preference — the TS TUI's status line is cleaner and more information-dense. Badges `[model]` are easier to scan than `model: value` labels.

**Files:** `internal/tui/components/statusline/statusline.go:View()`

### D-9: Collapsible sections in agent detail panel

**Decision:** 5 sections: Overview (expanded), Context (collapsed), Prompt (collapsed), Activity (expanded), Error (expanded when visible).

**Rationale:** Users need visibility into what conventions were loaded, what prompt was injected, and what the agent is doing. Collapsible sections keep the panel manageable. Follows the `settingstree.go` pattern already in the codebase.

**Files:** `internal/tui/components/agents/detail.go`

### D-10: Tab keys in global handler

**Decision:** Alt+C/A/T/Y handled in the global key switch, not delegated to tab bar at end of Update().

**Rationale:** The tab bar's `HandleMsg()` was only reached for messages that fell through ALL handlers. Since `handleKey()` always returns, the tab bar never got key events. Moving to global switch ensures they work regardless of focus.

**Files:** `internal/tui/model/key_handlers.go:167-178`

---

## Architecture Notes

### spawn_agent Data Flow (final working state)

```
User types prompt in Go TUI
  → TUI sends via stream-json stdin to claude CLI
    → Claude CLI (router) decides to call mcp__gofortress-interactive__spawn_agent
      → Claude CLI forwards to gofortress-mcp binary (stdio MCP server)
        → handleSpawnAgent() validates relationship, builds prompt
        → Sends agent_register IPC via UDS to TUI bridge
          → Bridge sends AgentRegisteredMsg to Bubbletea event loop
            → handleAgentRegistryMsg() calls registry.Register()
              → buildTree() DFS includes new agent (linked to root)
                → Agent appears in sidebar tree
        → runSubprocess() spawns: claude -p --output-format json --permission-mode bypassPermissions
          → Agent runs, produces JSON result
        → Sends agent_update IPC (complete/error)
        → Returns SpawnAgentOutput to MCP protocol
      → Claude CLI receives tool result
    → Claude CLI sends assistant response via stream-json
  → TUI renders response in chat panel
```

### Staff Architect Review Summary

Conducted by Staff Architect Critical Review agent (opus tier). Verdict: CONCERNS.
- **C-1 (fixed):** Agent tree handler discarded message data
- **C-2 (fixed):** Binary output paths inconsistent
- **M-1 (fixed):** Standalone spawner diverged on output format
- **M-3 (fixed):** Interactive spawner lacked relationship validation
- **m-1 (fixed):** Context requirements not passed through
- **m-3 (fixed):** ParentID not set in register payload
- **M-2 (deferred):** Duplicated spawner code — extract to pkg/spawn
- **M-4 (deferred):** No E2E integration test with mock claude binary

Full review: `review-critique.md`

---

## Links

- [[claude-cli-flags-reference]] — Future-proofing reference for CLI flags
- [[review-critique]] — Staff architect critical review document
- [[TUI Migration Status]] — Overall migration status
- [[ARCHITECTURE]] — System architecture (Section 16)
