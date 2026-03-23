# TUI Migration Status

> **Project:** GOgent-Fortress TUI rewrite (React/Ink/TypeScript to Go/Bubble Tea)
> **Tickets:** [[tickets/tui-migration/tickets/overview|overview.md]] (42 tickets, 9 phases)
> **Architecture:** [[docs/ARCHITECTURE|ARCHITECTURE.md]] Section 16
> **Braintrust:** [[tickets/tui-migration/braintrust-handoff-v2|braintrust-handoff-v2.md]]

---

## Current Status (2026-03-23)

**Phase 1: Spikes** — COMPLETE (4/4)
**Phase 2: Foundation** — COMPLETE (7/7)
**Phase 3: CLI Driver + NDJSON + MCP** — COMPLETE (5/5)
**Total: 16/42 tickets complete (38%)**

---

## Phase Completion Log

### Phase 1: Prerequisite Spikes (complete)
- TUI-001: Permission wire format — Option D (hybrid: acceptEdits + MCP side-channel)
- TUI-002: Go MCP SDK POC — v1.2.0 works, full roundtrip confirmed
- TUI-003: NDJSON event catalog — 6 top-level types, log-and-continue parser
- TUI-004: UDS IPC POC — 56us roundtrip, exponential backoff validated

### Phase 2: Foundation (complete)
- TUI-005: Theme system — 7 AdaptiveColor, 10 styles, 6 icons, Theme struct
- TUI-006: Focus management — FocusTarget, RightPanelMode with cycling
- TUI-007: Keybinding registry — 24 bindings, 5 groups, bubbles/key
- TUI-008: Root AppModel — Elm Architecture, 16 message types, key routing
- TUI-009: Chrome components — banner, tabbar, statusline (3 packages)
- TUI-010: Layout compositor — 70/30 split, 3 responsive breakpoints
- TUI-011: CLI entry point — stdlib flag (not Cobra per M-1), version injection

### Phase 3: CLI Driver + NDJSON + MCP (complete)
- TUI-012: NDJSON event types — 9 event structs, ParseCLIEvent, 98.1% coverage
- TUI-013: CLI subprocess driver — CLIDriver with Start/consumeEvents/WaitForEvent/SendMessage/Shutdown, 78.6% coverage, 60 tests, race-free
- TUI-014: Go MCP server — 7 tools (ask_user, confirm_action, request_input, select_option, spawn_agent, team_run, test_mcp_ping), UDS client, IPC protocol types, 74.4% coverage, 30 tests
- TUI-015: TUI-side UDS bridge — IPCBridge with messageSender interface, modal request/response correlation, fire-and-forget agent events, cancellation-safe shutdown, 79% coverage, 10 tests, race-free
- TUI-016: Startup wiring — sharedState pointer pattern, Init→startCLI→WaitForEvent loop, 3-attempt reconnect with exponential backoff, GOFORTRESS_SOCKET env wiring, 86% coverage, 27 new tests

---

## Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/tui/config` | 48 | 100% |
| `internal/tui/model` | 44 | 86.9% |
| `internal/tui/components/banner` | 8 | 100% |
| `internal/tui/components/tabbar` | 12 | 100% |
| `internal/tui/components/statusline` | 15 | 100% |
| `internal/tui/cli` (events) | 47 | 98.1% |
| `internal/tui/cli` (driver) | 60 | 78.6% |
| `internal/tui/mcp` | 30 | 74.4% |
| `internal/tui/bridge` | 10 | 79.0% |
| **Total** | **274** | **avg ~89%** |

---

## Key Design Decisions

### tabBarWidget interface (TUI-010)
The `tabbar` package imports `model.TabID`, so `model` cannot import `tabbar` (circular). Solution: `tabBarWidget` interface in model package, tabbar wired via `SetTabBar()` from cmd/gofortress/main.go.

### stdlib flag over Cobra (TUI-011)
Review M-1 flagged Cobra not in go.mod. With only 7 flags, stdlib `flag` is simpler. No new dependency added.

### ContentBlock as flat struct (TUI-012)
NDJSON content blocks have 4 variants (text, tool_use, tool_result, thinking). Used a single struct with omitempty fields instead of an interface — simpler JSON unmarshaling at the cost of a few unused zero fields per variant.

### ParseCLIEvent two-pass parsing (TUI-012)
First unmarshal discriminator (type/subtype), then unmarshal full struct. Unknown types return CLIUnknownEvent with raw JSON preserved (log-and-continue pattern).

### Go MCP server import alias + jsonschema tag (TUI-014)
Internal package is also named `mcp` — uses import alias `mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"`. Sharp edge: `jsonschema` struct tag takes a bare description string (NOT `description=...`), wrong format panics at AddTool registration time. UDS client uses lazy connect with exponential backoff (100ms base, 5 attempts). spawn_agent and team_run are validated stubs (check configs/paths, return structured responses) — full subprocess management deferred.

### CLI subprocess driver channel-to-Cmd pattern (TUI-013)
`WaitForEvent()` returns a `tea.Cmd` that blocks on `<-eventCh`. After processing each CLI event in Update(), the AppModel must return `d.WaitForEvent()` as a Cmd to maintain the subscription. 1MB scanner buffer for large tool outputs. `consumeEvents` goroutine logs+continues on parse errors (never crashes). Shutdown: SIGTERM → 2s → SIGKILL in goroutine. Tests use `io.Pipe` injection + live `sleep 60` subprocess for signal tests.

---

## Review Condition Resolution

| Condition | Resolution |
|-----------|-----------|
| C-1: Go 1.22+ → 1.25+ | Applied to all ticket descriptions |
| C-2: MCP SDK v1.2.0 vs v1.3.0 | TUI-002 spike confirmed v1.2.0 works |
| M-1: Cobra not in go.mod | Used stdlib `flag` in TUI-011 |

---

## Links

- Spike results: `tickets/tui-migration/spike-results/`
- Ticket index: `tickets/tui-migration/tickets/tickets-index.json`
- Staff architect review: `.claude/sessions/20260316-plan-tickets-tui/review-critique.md`
