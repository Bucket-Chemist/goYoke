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
**Phase 6: Rich Features** — COMPLETE (6/6 + integration wiring)
**Phase 7: Multi-Provider** — COMPLETE (5/5)
**Phase 8: Lifecycle** — COMPLETE (3/3)
**Phase 9: Testing** — IN PROGRESS (2/7, TUI-038 next)
**Total: 37/42 tickets complete (88%)**

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

### Phase 6: Rich Features (complete)
- TUI-022: Claude conversation panel — ClaudePanelModel with viewport + textinput, streaming content blocks, message grouping, markdown rendering via Glamour, 83.3% coverage, 46 tests
- TUI-023: Markdown rendering via Glamour — Cached Glamour renderer with sync.Mutex, RenderMarkdown() helper in util package, auto-added to go.mod (resolved M-5), 87.0% coverage, 15 tests
- TUI-024: Cost tracker — Go port from TS, SessionCost/PerAgentCosts/BudgetUSD with RWMutex, recomputeOverBudget invariant, 97.0% state coverage, 92 state tests (includes agent + cost)
- TUI-025: Toast notification system — ToastModel with auto-expire timers, max 3 concurrent, level-colored (Info/Warning/Error/Success), disconnect visual feedback resolved, 94.2% coverage, 19 tests
- TUI-026: Status line data wiring — Real-time cost display, active model, provider info, session duration in statusline, integrated with CostTracker, 87.8% coverage, 29 tests
- TUI-027: Team polling and team list — TeamRegistry with TeamConfig/Wave/Member types, TeamListModel with filesystem polling, TeamDetailModel with wave-grouped member view, 94.1% coverage, 62 tests
- TUI-027.5: Integration wiring — All components wired into AppModel, placeholders replaced, streaming pointer bug fixed in claude/panel.go, cost tracker unified, race detector clean

### Phase 7: Multi-Provider (in progress)
- TUI-028: Multi-provider config types ✅ — ProviderID (4 constants), ProviderConfig/ModelConfig/DisplayMessage types, ProviderState with RWMutex and per-provider isolation (messages, sessionIDs, models, projectDirs), SwitchProvider preserves state, deep-copy returns, all models ported from TS source (NOT architect specs per review M-2), 97.5% coverage, 161 tests (incl. 3 concurrent stress tests), race-free
- TUI-029: Provider switching logic + message isolation ✅ — Shift+Tab cycles 4 providers, per-provider message save/restore via SaveMessages/RestoreMessages on ClaudePanel, CLIDriver replaced (new instance per provider, single-use pattern), CLIDriverOpts extended with AdapterPath+EnvVars for non-Anthropic, ProviderSwitchMsg→handleProviderSwitch flow (save→cycle→restore→shutdown→start), streaming blocks switch, SetActiveMessages bulk setter, providerState wired into sharedState, 86.5% model coverage, 84.7% claude coverage, 81.7% cli coverage, 953 total tests, race-free
- TUI-030: Provider tab bar UI ✅ — New package `internal/tui/components/providers/`, ProviderTabBarModel (display-only, no key handling), horizontal tab strip with StyleHighlight/StyleSubtle, hidden when ≤1 provider, providerTabBarWidget interface in model, wired into renderLayout between tabBar and mainArea, computeLayout subtracts Height(), handleProviderSwitch calls SetActive(), 90.6% coverage, 16 tests
- TUI-031: Provider session resume ✅ — GetActiveSessionID/GetActiveProjectDir read methods, ExportSessionIDs/ImportSessionIDs for persistence (additive, never overwrites), ExportModels/ImportModels with model validation, handleProviderSwitch now passes SessionID+ProjectDir to new CLIDriver opts, SystemInitEvent persists session ID to ProviderState, 97.8% state coverage, 85.4% model coverage, 999 total tests, race-free
- TUI-032: Dashboard, Settings, Telemetry, PlanPreview, TaskBoard panels ✅ — 5 new packages (dashboard 100%, settings 94.4%, telemetry 91.9%, planpreview 97.4%, taskboard 97.1%), RPMPlanPreview added to RightPanelMode enum, 5 widget interfaces + sharedState fields in model, renderRightPanel expanded with all modes, TaskBoard overlay with Alt+B toggle (visible between mainArea and toasts, height subtracted from computeLayout), telemetry loads from JSONL via tea.Cmd (max 50 entries), planpreview renders via Glamour in SetContent, 79.2% model coverage, 1069 total tests, 21 packages, race-free
- **Phase 7 COMPLETE** — all 5 multi-provider tickets done

### Phase 8: Lifecycle (complete)
- TUI-033: Session persistence (load/save) ✅ — New `internal/tui/session/` package (persistence.go + history.go), SessionData struct with provider maps, Store with atomic writes (temp+rename), NewSessionID (YYYYMMDD.UUID), LoadSession/SaveSession/SetupSessionDir, per-provider conversation history (LoadConversationHistory/SaveConversationHistory), auto-save debounced 5s via SessionAutoSaveMsg+seq counter, save-on-shutdown via ForceQuit, session resume via --session-id flag in main.go, ExportAllMessages+GetMessages added to ProviderState, 85.6% coverage, 34 tests, race-free
- TUI-034: Graceful shutdown with timing budget ✅ — New `internal/tui/lifecycle/` package, ShutdownManager with 10s total budget (5 phases: save→interrupt CLI→shutdown CLI→close bridge→wait hooks), Shutdownable/BridgeShutdownable interfaces for testability, defer-based LIFO shutdown removed from main.go, replaced with explicit sequenced shutdown (driver BEFORE bridge per DES-1), ProcessManager.StartSignalHandler wired for SIGINT/SIGTERM, double-Ctrl+C (shutdownInProgress flag→immediate tea.Quit), ShutdownCompleteMsg message type, SaveSessionPublic() on AppModel, 80.0% coverage, 11 tests, race-free. Integration tests (orphan verification) deferred to TUI-039.
- TUI-035: Clipboard, search, and input history ✅ — 3 new files (util/clipboard.go, claude/history.go, claude/search.go + tests), CopyToClipboard via atotto/clipboard (indirect dep), InputHistory with JSON persistence (atomic write, max 500, dedup consecutive, resilient load), SearchModel with case-insensitive substring matching + Ctrl+N/P navigation + wraparound + '/' trigger + Esc dismiss, 4 new keybindings (/, ctrl+n, ctrl+p, ctrl+y), SearchQueryChangedMsg for real-time re-search on typing, scrollToSearchResult viewport integration, search-mode guard in handleKey, 78.8% claude coverage, 90.3% util coverage, ~100 claude tests, race-free
- **Phase 8 COMPLETE** — all 3 lifecycle tickets done

### Phase 9: Testing (in progress)
- TUI-036: Component unit tests ✅ — testdata/ fixtures (5 NDJSON files + helpers.go), 22 new model tests (SessionAutoSaveMsg, ShutdownCompleteMsg, double-Ctrl+C, toast nil safety, renderRightPanel modes, modal overlay), MCP coverage tests (send/notify/spawn_agent/select_option/request_input), bridge coverage tests. Coverage improvements: model 75→89%, bridge 75→84%, claude 79→91%, mcp 70→79%. All 23 packages pass with race detector. TestHarness (DES-7) deferred — widget interface mocking pattern validated.
- TUI-037: CLI driver integration test ✅ — mock-claude.sh in testdata/ (env-controlled NDJSON emitter), 5 integration tests (NormalFlow, CrashRecovery, Interrupt, Shutdown, UnknownEvent), AdapterPath injection for mock, waitForMsg helper with 5s deadline, testing.Short() guards, CLI coverage 81→91%, all pass <1s, race-free
- TUI-038: MCP server integration test — PENDING (depends TUI-014, TUI-015)
- TUI-039: E2E smoke test with live CLI — PENDING (depends TUI-016, TUI-018)
- TUI-040: Performance benchmarks — PENDING (depends TUI-016, TUI-017, TUI-020)
- TUI-041: Unknown event resilience test — PENDING (depends TUI-012, TUI-013)
- TUI-042: Feature parity checklist verification — PENDING (depends TUI-039)

### Phase 7 Remediation (post-completion)
- R-1: main.go wiring ✅ — All 7 Phase 7 widgets instantiated (provider tab bar, dashboard, settings, telemetry, planpreview, taskboard). ProviderState() getter on AppModel. Settings initialized with CLI opts.
- R-2: Provider switch debounce ✅ — 300ms tea.Tick with seq counter. Rapid Shift+Tab → only latest fires. Stale ProviderSwitchExecuteMsg discarded via seq mismatch. ProviderSwitchMsg kept for programmatic (non-debounced) switching.
- R-3: Handoff generation ✅ — New `handoff.go`: buildHandoffSummary scans last 10 msgs, extracts last user request + assistant response + counts. Injected as system message to NEW provider. Returns "" if <2 messages. 16 table-driven tests.
- R-4: ToolBlock persistence ✅ — state.ToolBlock type (Name/Input/Output, no Expanded). state.DisplayMessage extended with ToolBlocks field. SaveMessages converts claude→state, RestoreMessages converts state→claude with Expanded=false. copyMessages deep-copies ToolBlocks. 8 new tests.
- **Post-remediation:** model 81.7%, state 97.9%, claude 86.8%. 1108 total tests. All gaps closed except R-5 (deferred: in-session model switch — CLI protocol limitation).

---

## Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/tui/config` | 100 | 100.0% |
| `internal/tui/model` | ~194 | 88.8% |
| `internal/tui/components/banner` | 8 | 100.0% |
| `internal/tui/components/tabbar` | 12 | 100.0% |
| `internal/tui/components/statusline` | 29 | 87.8% |
| `internal/tui/components/modals` | 107 | 88.5% |
| `internal/tui/components/agents` | 45 | 90.6% |
| `internal/tui/components/claude` | ~100 | 91.0% |
| `internal/tui/components/toast` | 19 | 94.2% |
| `internal/tui/components/teams` | 62 | 94.1% |
| `internal/tui/state` | 208 | 97.9% |
| `internal/tui/util` | ~18 | 90.3% |
| `internal/tui/cli` | ~160 | 91.2% |
| `internal/tui/mcp` | ~44 | 78.6% |
| `internal/tui/components/providers` | 16 | 90.6% |
| `internal/tui/components/dashboard` | 10 | 100.0% |
| `internal/tui/components/settings` | 10 | 94.4% |
| `internal/tui/components/telemetry` | 12 | 91.9% |
| `internal/tui/components/planpreview` | 10 | 97.4% |
| `internal/tui/components/taskboard` | 12 | 97.1% |
| `internal/tui/bridge` | ~20 | 83.5% |
| `internal/tui/session` | 34 | 85.6% |
| `internal/tui/lifecycle` | 11 | 80.0% |
| **Total** | **~1250+** | **avg ~91%** |

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

### Glamour cached renderer (TUI-023)
`RenderMarkdown(content string)` in `internal/tui/util/markdown.go` wraps Glamour with a `sync.Mutex`-protected singleton renderer. Glamour added to go.mod (resolved review M-5). Uses DarkStyle with 80-char word wrap. Renderer created lazily on first call. 87% coverage, 15 tests.

### CostTracker Go port (TUI-024)
`CostTracker` in `internal/tui/state/cost.go` mirrors TS SessionStore cost fields. SessionCost, PerAgentCosts map, optional BudgetUSD (*float64, nil = no budget). RWMutex-protected. `recomputeOverBudget()` called after every mutation as invariant maintenance. Display formatting via `FormatCost()` utility. 97% coverage.

### Toast auto-expire pattern (TUI-025)
`ToastModel` manages a FIFO toast queue with max 3 concurrent. Each toast gets a `tea.Tick` Cmd on creation. `ExpireToastMsg{ID}` triggers removal after duration (3s info, 5s warning, 8s error). Level-colored via theme (green success, yellow warning, red error). Disconnect notifications route here from CLI driver reconnection failures.

### Provider switching flow (TUI-029)
Provider switching is triggered by Shift+Tab → `ProviderSwitchMsg` → `handleProviderSwitch()`. The flow: (1) save current conversation via `claudePanel.SaveMessages()` → `providerState.SetActiveMessages()`, (2) save session ID, (3) cycle to next provider via `AllProviders()` + modular index, (4) restore messages via `providerState.GetActiveMessages()` → `claudePanel.RestoreMessages()`, (5) shutdown old CLIDriver, (6) create new CLIDriver with provider-specific opts (model, adapter path, env vars), (7) start new driver. CLIDriver is single-use (Start once only), so a new instance is created per switch. Streaming blocks the switch (checked via `claudePanel.IsStreaming()`). `claude.DisplayMessage` → `state.DisplayMessage` conversion drops ToolBlocks (transient rendering state — same as TS source). `CLIDriverOpts` extended with `AdapterPath` and `EnvVars` for non-Anthropic providers — Start() uses AdapterPath as binary name when non-empty, merges EnvVars into `cmd.Env`.

### ProviderState thread-safe container (TUI-028)
`ProviderState` in `internal/tui/state/provider.go` follows the same RWMutex + deep-copy pattern as AgentRegistry. Static configs are immutable after `NewProviderState()` — only mutable state (messages, sessionIDs, models, projectDirs) is written per-provider. `SwitchProvider(id)` simply changes the `active` field without touching any per-provider data. `copyConfig()` deep-copies both `Models []ModelConfig` and `EnvVars map[string]string`. Sentinel errors `ErrProviderNotFound` and `ErrModelNotFound` support `errors.Is()` wrapping. Model/provider data faithfully matches TS `providers.ts`, NOT architect specs (per review M-2).

### TeamRegistry filesystem polling (TUI-027)
`TeamListModel` polls `$SESSION_DIR/teams/` every 2 seconds via `tea.Tick`. Reads `config.json` + `stdout_*.json` files per team directory. `TeamRegistry` tracks team state in memory with `LoadFromDir()` refresh. Wave-grouped member view in `TeamDetailModel` shows member status, duration, cost.

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

### Graceful shutdown with ShutdownManager (TUI-034)
New `internal/tui/lifecycle/` package provides `ShutdownManager` with configurable timing budgets (10s total, 2s CLI, 500ms hooks). Replaces defer-based LIFO shutdown (which had **wrong ordering**: bridge before driver) with explicit 5-phase sequence: save session → interrupt CLI (SIGINT) → shutdown CLI (SIGTERM→SIGKILL) → close bridge → wait for hooks. `Shutdownable` and `BridgeShutdownable` interfaces decouple from concrete types for testability. `ProcessManager.StartSignalHandler()` wired into main.go for OS-level SIGINT/SIGTERM handling. Double-Ctrl+C pattern: first press sets `shutdownInProgress=true` and runs sequenced shutdown via tea.Cmd; second press immediately calls `tea.Quit`. `ShutdownCompleteMsg` message drives the Quit after graceful completion. `shutdownFunc func() error` stored in sharedState (closure pattern avoids importing lifecycle from model). Post-p.Run() fallback shutdown call ensures cleanup even for menu-based exits.

### Session persistence Store pattern (TUI-033)
New `internal/tui/session/` package provides `Store` struct with configurable `baseDir` (default `~/.claude/sessions/`). All writes use atomic temp-file-then-rename to prevent corruption on crash. `SessionData` struct holds provider session IDs, model selections, active provider, cost, and tool call count — all serialized to `{baseDir}/{id}/session.json`. Per-provider conversation histories at `{baseDir}/{id}/history-{provider}.json`. Empty histories are removed (no accumulation). `SetupSessionDir` creates session dir + `current-session` marker + `.claude/tmp` symlink. Auto-save debounced with 5s cooldown via `SessionAutoSaveMsg` + seq counter (same pattern as provider switch debounce). `ExportAllMessages()` added to `ProviderState` for bulk history export on save. Session resume via `--session-id` flag restores provider state (ImportSessionIDs, ImportModels, SwitchProvider).

---

## Architectural Risks for Phase 7+

- ~~No visual feedback when CLI disconnects~~ ✅ RESOLVED: TUI-025 toast notifications provide disconnect feedback
- UDSClient serializes requests (one at a time) — acceptable for current tools but limits parallelism
- model package coverage at 84.5% — TUI-036 should push above 90%
- ~~DiffEntry rendering not yet implemented~~ ✅ RESOLVED: TUI-022 Claude panel renders inline diffs
- Multi-provider adapter paths will reference TS scripts initially — Go-native adapters deferred to post-migration

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
