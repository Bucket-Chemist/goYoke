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
**Phase 4: Modal System** — COMPLETE (2/2)
**Phase 5: Agent Tree** — COMPLETE (3/3)
**Total: 21/42 tickets complete (50%)**

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

### Phase 4: Modal System (complete)
- TUI-017: Modal model types and queue — 5 ModalTypes (Ask/Confirm/Input/Select/Permission), ModalRequest/ModalResponse with JSON, ModalModel (tea.Model) with keyboard nav + centered double-border overlay via lipgloss.Place, ModalQueue (sequential FIFO, no concurrent modals), ModalResponseMsg in modals package (avoids circular import), free-text "Other" mode for Ask, textinput.Model for Input type, 89.2% coverage, 72 tests
- TUI-018: Permission flow — PermissionHandler orchestrates multi-step flows (EnterPlan/ExitPlan/AskUser/Confirm/Input/Select), bridgeWidget extended with ResolveModalSimple (avoids mcp import), full MCP→UDS→TUI→UDS→MCP roundtrip, ExitPlan 2-step flow with feedback JSON, post-hoc diff extraction via extractDiffs()+DiffEntry from tool_use_result.structuredPatch, modals.ModalQueue+permHandler in sharedState (survives Bubbletea value-copy), 88.5% modals coverage, 107 modals tests, 75.6% model coverage, race-free

### Phase 5: Agent Tree (complete)
- TUI-019: AgentRegistry — RWMutex-protected store with Agent/AgentStatus/AgentActivity/AgentTreeNode/AgentStats types, Register with dedup (agentType+description key), Update with status transition validation (revert on invalid), DFS Tree() with depth/IsLast, Get() returns copies (no internal state exposure), Review M-3 compliant (InvalidateTreeCache only from Update goroutine), 96.1% coverage, 56 tests, race-free
- TUI-020: Agent tree view + detail — AgentTreeModel (scrollable, Up/Down/j/k/Enter nav, focus-gated), Unicode box-drawing connectors (├─/└─/│), status icons+colors, AgentSelectedMsg on cursor change, AgentDetailModel (display-only with status/model/tier/duration/cost/tokens/activity/error), comma-formatted tokens, word-wrapped error output, 90.6% coverage, 41 tests
- TUI-021: Agent sync from NDJSON — SyncAssistantEvent scans tool_use blocks (Task→Register, non-Task+parent→SetActivity), SyncUserEvent matches tool_result→Complete/Error, ParseTaskInput extracts agent metadata from JSON, ExtractToolActivity dispatches per tool (Read/Write/Bash/Grep→target), normaliseAgentType kebab-case, modelToTier inference, orphaned IDs silently ignored, 84.1% cli coverage, 152 cli tests

---

## Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/tui/config` | 48 | 100% |
| `internal/tui/model` | 98 | 75.6% |
| `internal/tui/components/banner` | 8 | 100% |
| `internal/tui/components/tabbar` | 12 | 100% |
| `internal/tui/components/statusline` | 15 | 100% |
| `internal/tui/components/modals` | 107 | 88.5% |
| `internal/tui/state` | 56 | 96.1% |
| `internal/tui/components/agents` | 41 | 90.6% |
| `internal/tui/cli` (combined) | 152 | 84.1% |
| `internal/tui/mcp` | 30 | 74.4% |
| `internal/tui/bridge` | 10 | 76.2% |
| **Total** | **557** | **avg ~88%** |

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

### ModalResponseMsg in modals package (TUI-017)
`ModalResponseMsg` is defined in `internal/tui/components/modals` (not `model`) to avoid a circular import: `model` → `modals` → `model`. AppModel.Update type-switches on `modals.ModalResponseMsg` to advance the queue and deliver bridge responses. The `ResponseCh chan ModalResponse` channel path works in parallel for bridge goroutines that block-wait on a response (non-blocking send prevents deadlock if caller isn't listening).

### Modal queue sequential gate (TUI-017)
ModalQueue guarantees exactly one modal at a time. Push enqueues; Activate pops front and creates ModalModel; Resolve closes active modal, delivers to ResponseCh, and auto-activates the next queued item. Two simultaneous permission requests from the bridge are safely serialised.

### ResolveModalSimple bridge interface (TUI-018)
bridgeWidget extended with `ResolveModalSimple(requestID, value string)` instead of importing mcp.ModalResponsePayload. The real `IPCBridge.ResolveModalSimple` is a one-line wrapper: `b.ResolveModal(requestID, mcp.ModalResponsePayload{Value: value})`. This breaks the import cycle: model → mcp is avoided.

### PermissionHandler multi-step flow (TUI-018)
PermissionHandler sits between AppModel and ModalQueue. It classifies bridge requests by heuristic (option content → FlowType), manages multi-step state (ExitPlan: step 0 = Select, step 1 = Input for feedback), and combines step responses into a single PermissionResult. The `rootRequestID()` function strips `:step<N>` suffixes so step responses route to their parent flow. ExitPlan result is JSON: `{"decision":"approve|changes|reject","feedback":"..."}`.

### Post-hoc diff extraction (TUI-018)
`extractDiffs()` on AppModel inspects `cli.UserEvent.ToolUseResult` for `structuredPatch` fields. Two-path unmarshal: try single object, fallback to array. DiffEntry accumulates in `m.diffs []DiffEntry`. TUI-022 (Claude panel) will render these inline.

### sharedState pattern extended (TUI-018)
`modalQueue *modals.ModalQueue` and `permHandler *modals.PermissionHandler` added to sharedState. Both are pointer-based to survive Bubbletea's value-copy of AppModel. This is the same pattern used for cliDriver and bridge.

### AgentRegistry Review M-3 compliance (TUI-019)
Register() modifies the agents map under Lock but does NOT call InvalidateTreeCache(). The caller must send AgentRegisteredMsg via program.Send(), and the Bubbletea Update() handler calls InvalidateTreeCache(). This maintains the single-threaded Update/View invariant — the IPC bridge goroutine (which calls Register()) never touches treeCache directly.

### AgentRegistry copy isolation (TUI-019)
Get() and Tree() return deep copies of Agent structs. Tree() copies each AgentTreeNode so concurrent readers cannot observe stale mutations. This prevents the data race between bridge goroutines (which call Register/Update) and View() (which reads treeCache on the main goroutine).

### Status transition validation with revert (TUI-019)
Update() captures status before applying fn, then checks if the transition is valid. If invalid (e.g., Complete→Running), the status is reverted to the pre-fn value and ErrInvalidTransition is returned. Valid transitions: Pending→{Running,Killed}, Running→{Complete,Error,Killed}. Complete/Error/Killed are terminal.

---

## Architectural Risks for Phase 5+

- No visual feedback when CLI disconnects (silent after 3 retries) — TUI-025 should fix
- UDSClient serializes requests (one at a time) — acceptable for current tools but limits parallelism
- model package coverage dropped to 75.6% (from 86%) due to new untested integration paths — TUI-036 should address
- DiffEntry rendering not yet implemented — TUI-022 (Claude panel) will consume `m.diffs`

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
