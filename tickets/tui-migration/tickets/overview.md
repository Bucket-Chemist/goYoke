# Implementation Plan: TUI Migration (React/Ink/TypeScript to Go/Bubble Tea)

> **Generated:** 2026-03-16
> **Workflow:** /plan-tickets v1.0
> **Planning cost:** ~$8.50 (Planner $2.50 + Architect $3.50 + Staff-Architect $2.50)
> **Braintrust cost (prior):** $6.61 (Einstein $2.69 + Staff-Architect $2.39 + Beethoven $1.53)
> **Review Status:** APPROVE_WITH_CONDITIONS (High Confidence)

---

## Executive Summary

Migrate the GOgent-Fortress TUI from React/Ink/TypeScript to Go/Bubble Tea using a **two-process topology** (Go TUI + Claude Code CLI). This replaces the original plan's three-process topology (Go TUI + TS MCP sidecar + CLI) which was unanimously rejected by braintrust analysis due to 9 failure modes, permanent Node.js dependency, and three critical architectural issues (unverified MCP transport, unidirectional bridge, split-brain agent state).

The Go TUI serves MCP tools natively via the official Go MCP SDK (v1.2.0+, stdio transport), eliminating the TS sidecar entirely. A prerequisite spike (4-6h) resolves the permission wire format unknown before implementation commits.

**42 tickets across 9 phases. Estimated 10-16 weeks serial, 7-10 weeks parallel.**

---

## Strategic Approach

### Two-Process Architecture

```
Go TUI Process (single binary)
  |-- Bubble Tea event loop (owns terminal stdin/stdout)
  |-- CLI Driver (manages Claude CLI subprocess via pipes)
  |-- IPC Bridge (UDS listener for MCP server communication)
  |
  +--spawns--> Claude Code CLI (--output-format stream-json)
                  |
                  +--spawns--> gofortress-mcp (Go MCP server, stdio transport)
                                  |
                                  +--connects--> TUI via UDS side channel
```

### Key Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Topology | Two-process | Braintrust unanimous. 3 failure modes vs 9. Eliminates Node.js. |
| MCP server | Separate binary (`gofortress-mcp`) | Clear process identity, simpler debugging |
| MCP-to-TUI IPC | Unix domain socket (`gofortress-{pid}.sock`) | Existing codebase pattern, ~0.1ms latency |
| Permission architecture | Option A (control protocol), fallback B | Spike determines feasibility |
| AgentRegistry | Flat map with computed tree cache | O(1) lookups, tree only for View() |
| Bubble Tea version | v1 (1.3.10) | Battle-tested, ecosystem compatible |
| Module path | Existing `github.com/Bucket-Chemist/GOgent-Fortress` | Shares pkg/routing, internal/lifecycle |
| Cost tracker | Go port, display-only budget warning | CLI-side `--max-budget-usd` does blocking |

---

## Implementation Phases

| Phase | Description | Tickets | Dependencies | Estimate |
|-------|-------------|---------|--------------|----------|
| 1 | **Prerequisite Spike** | TUI-001 to TUI-004 | None (all parallel) | 4-6 hours |
| 2 | **Foundation** | TUI-005 to TUI-011 | Spike results | 1-2 weeks |
| 3 | **CLI Driver + NDJSON Parser + Go MCP Server** | TUI-012 to TUI-016 | Phase 1+2 | 2-3 weeks |
| 4 | **Modal System** | TUI-017, TUI-018 | Phase 3 | 1-2 weeks |
| 5 | **Agent Tree + Process Management** | TUI-019 to TUI-021 | Phase 3 | 1-2 weeks |
| 6 | **Rich Features** | TUI-022 to TUI-027 | Phase 3+5 | 1-2 weeks |
| 7 | **Settings, Providers, Teams** | TUI-028 to TUI-032 | Phase 3+6 | 1-2 weeks |
| 8 | **Lifecycle** | TUI-033 to TUI-035 | Phase 3+5 | 1 week |
| 9 | **Integration Testing** | TUI-036 to TUI-042 | All previous | 1-2 weeks |

**Parallelizable:** Phases 5, 6, 7 can run concurrently after Phase 3 completes.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Permission wire format undocumented/incompatible | ~~Medium~~ **RESOLVED** | High | TUI-001 spike: no control_request protocol exists. Architecture uses Option D (hybrid): acceptEdits + MCP side-channel. See spike-results/permission-protocol.md |
| Go MCP SDK fails to serve tools to Claude CLI | ~~Very Low~~ **RESOLVED** | High | TUI-002 spike: v1.2.0 works, full roundtrip confirmed. Key finding: MCP tools need `--allowedTools` (acceptEdits doesn't cover them). See spike-results/go-mcp-poc.md |
| NDJSON event types expand without notice | ~~Medium~~ **MITIGATED** | Medium | TUI-003 spike: 6 top-level types + stream_event catalogued with full schemas. 3 known-but-unobserved types documented. Go type mapping in ndjson-catalog.md §7. Log-and-continue parser handles unknowns. |
| Agent state race (NDJSON vs MCP goroutines) | Medium | Medium | AgentRegistry with RWMutex. Dedup on agentType+description. All mutations via tea.Msg |
| Multi-provider underscoped | High | Medium | Decomposed into 4 tickets (TUI-028 to TUI-031) |
| Cost tracker logic lost | Medium | Medium | Explicit Go port (TUI-024). Session cost from result event |
| CLI update breaks stream-json format | Medium | Medium | Version detection, defensive parsing, optional pointer fields |
| Graceful shutdown orphans | Medium | Medium | Timing budget (10s), SIGTERM→SIGKILL escalation |
| UDS startup race | ~~Low~~ **RESOLVED** | Low | TUI-004 spike: exponential backoff validated (connected on attempt 1). UDS roundtrip 56µs. See spike-results/ipc-protocol.md |
| Bubble Tea v1 EOL | Low | Low | v1→v2 migration is incremental, deferred to post-migration |

---

## Review Summary

**Verdict:** APPROVE_WITH_CONDITIONS
**Reviewer:** Staff Architect Critical Review
**Confidence:** HIGH

**Critical Issues (2) — must fix before implementation:**
- **C-1:** Go version constraint → update from "Go 1.22+" to "Go 1.25+" (matches go.mod)
- **C-2:** MCP SDK version → resolve v1.2.0 (go.mod) vs v1.3.0 (strategy references)

**Major Issues (5) — should fix before assigning tickets:**
- **M-1:** Cobra CLI assumed but absent from go.mod (consider `flag` package)
- **M-2:** Provider/model definitions fabricated — must port from actual `providers.ts`
- **M-3:** AgentRegistry treeCache — Register() must send tea.Msg, not mutate directly
- **M-4:** TUI-032 scope monster — 5 panels in 1 ticket (consider decomposing)
- **M-5:** Glamour not in go.mod — blocks Phase 6

**Minor Issues (8):** teatest import path, hidden dependencies (TUI-012, TUI-025), mock CLI script timing, provider model sources, git/auth fallbacks, process.go documentation gap

**Commendations (7):** Braintrust-informed architecture, explicit concurrency model, requirements traceability, permission decision framework, faithful cost tracker port, risk register with mitigations, multi-provider decomposition

### Conditions (must be addressed)

1. Update Go version constraint from 1.22+ to 1.25+ in all references
2. Resolve MCP SDK version: verify v1.2.0 has StdioTransport OR upgrade go.mod to v1.3.0

### Review Amendments Incorporated into Tickets

The following review findings have been incorporated into the ticket descriptions:
- C-1/C-2: Version constraints corrected in all affected tickets
- M-1: TUI-011 notes Cobra dependency gap with `flag` alternative
- M-2: TUI-028 references actual `packages/tui/src/config/providers.ts` as source of truth
- M-3: TUI-019 explicitly states Register() → tea.Msg → Update() → treeCache pattern
- M-5: TUI-023 notes Glamour must be added to go.mod
- m-2: TUI-012 dependencies updated to include TUI-001 and TUI-003
- m-7: TUI-025 dependency on TUI-010 added
- m-8: Relevant tickets note process.go limitations

---

## Review History

### Review 1: Planning Review (2026-03-16)

**Reviewer:** Staff Architect Critical Review
**Verdict:** APPROVE_WITH_CONDITIONS (High Confidence)
**Scope:** Implementation plan + ticket set (Phases 1–9)
**Outcome:** 2 critical issues, 5 major issues, 8 minor issues. All critical/major findings incorporated into tickets before implementation began. See full critique at `.claude/sessions/20260316-plan-tickets-tui/review-critique.md`.

### Review 2: Post-Phase 7 Code Review (2026-03-23)

**Reviewer:** 4-reviewer panel (backend, standards, Go-TUI specialist, staff architect)
**Verdict:** APPROVE_WITH_CONDITIONS (High Confidence, 8 commendations)
**Scope:** All code delivered through Phase 7 (Phases 1–7 + remediation R-1–R-4)
**Phase 8–9 readiness:** CONDITIONALLY READY

**Bug fixes applied (Wave 1):**

| ID | Fix | Files |
|----|-----|-------|
| FIX-1 | Unified 5 duplicate `truncate` helpers → `util.Truncate`; fixed UTF-8 byte-slicing bug in handoff.go | `internal/tui/util/text.go`, `internal/tui/model/handoff.go` |
| FIX-2 | Fixed `ModalResponseMsg` undefined in app_test.go; removed deprecated aliases (BridgeModalRequestMsg, CLIDriverSender, GetAgentCosts) | `internal/tui/model/app_test.go` |
| FIX-3 | CLI driver scanner made interruptible via inner goroutine + channel pattern (fixes goroutine leak + TestConsumeEvents_ExitsOnShutdownCh) | `internal/tui/cli/driver.go`, `internal/tui/cli/driver_test.go` |
| FIX-4 | Bridge Shutdown uses non-blocking send instead of close(ch) to prevent double-close panic; AgentRegistry.Remove uses fresh slice to prevent aliasing | `internal/tui/bridge/server.go`, `internal/tui/state/cost.go` |
| FIX-5 | UDS SendRequest gets 10-minute read deadline + reconnect-on-error with single retry; magic number 300000 extracted as constant | `internal/tui/bridge/server.go` |
| FIX-6 | CLIReconnectMsg carries sequence number to prevent ghost reconnections after provider switch; filepath.Join replaces fmt.Sprintf for path construction | `internal/tui/model/messages.go`, `internal/tui/model/app.go` |

**Design refactors applied (Wave 2):**

| ID | Refactor | Outcome |
|----|----------|---------|
| DES-2 | Split app.go (1333 lines) into 4 files | app.go (766), interfaces.go (225), layout.go (257), provider_switch.go (135) |
| DES-3 | Moved TaskEntry from taskboard to state/task.go | Eliminated model→taskboard import (only leaky widget abstraction) |
| DES-4 | Unified DisplayMessage/ToolBlock: claude package uses state.* aliases | Removed ~50 lines of conversion boilerplate; added Expanded field to state.ToolBlock |
| DES-6 | Created internal/teamconfig/ shared package for TeamConfig/Wave/Member types | teams package uses aliases; eliminates type duplication |

**Deferred (tracked in existing tickets):**

| ID | Finding | Ticket |
|----|---------|--------|
| DES-1 | Shutdown orchestration: ProcessManager.StartSignalHandler wiring, LIFO ordering, SIGKILL escalation | TUI-034 (Phase 8) |
| DES-5 | CLI message types: moving them doesn't break model→cli import (9 other event types remain) | Not ticketed — low value |
| DES-7 | Integration test harness: cross-layer TestHarness with pipe-injected CLI + real AppModel | TUI-036 (Phase 9) |

**Post-fix test status:** 21/21 packages green, race detector clean (`go test -race ./internal/tui/...`).

---

## Requirements Traceability

### Old Tickets (GOgent-109 to GOgent-121)

| Old Ticket | Title | New Ticket(s) | Status |
|-----------|-------|---------------|--------|
| GOgent-109 | Agent Lifecycle Telemetry | TUI-019, TUI-021 | SUPERSEDED |
| GOgent-110 | CLI Subprocess Management | TUI-013 | SUPERSEDED |
| GOgent-111 | Performance Dashboard Shell | TUI-010, TUI-032 | SUPERSEDED |
| GOgent-112 | Auto-Restart on Panic | TUI-013, TUI-016 | PRESERVED |
| GOgent-113 | File Watchers for Telemetry | TUI-027, TUI-032 | MODIFIED |
| GOgent-114 | Event System Integration | TUI-012 | SUPERSEDED |
| GOgent-115 | Agent Tree Model | TUI-019 | SUPERSEDED |
| GOgent-116 | Tree View Component | TUI-020 | SUPERSEDED |
| GOgent-117 | Agent Detail Sidebar | TUI-020 | SUPERSEDED |
| GOgent-118 | Claude Conversation Panel | TUI-022 | SUPERSEDED |
| GOgent-119 | 70/30 Layout Integration | TUI-010 | SUPERSEDED |
| GOgent-120 | Persistent Banner | TUI-009 | SUPERSEDED |
| GOgent-121 | Session Management | TUI-033, TUI-034 | SUPERSEDED |

### Original Plan Phase Disposition

| Original Phase | Disposition | Rationale |
|---------------|-------------|-----------|
| P0: Scaffolding | MERGED → Phase 2 | Directory structure adapted for two-process |
| P1: Core Shell | PRESERVED → Phase 2 | Topology-independent |
| P2: CLI Driver | PRESERVED → Phase 3 | Event types expanded per v2.0 catalog |
| P3: MCP Sidecar | **ELIMINATED** | Go MCP server replaces entirely. Saves ~12 tickets |
| P4: Modals | PRESERVED → Phase 4 | Bridge mechanism replaced by channels |
| P5: Agent Tree | PRESERVED → Phase 5 | Bridge handler → MCP tool handler |
| P6: Teams | MERGED → Phase 6 | Combined with rich features |
| P7: Settings/Tabs | PRESERVED → Phase 7 | P7-7 decomposed into 4 sub-tickets |
| P8: Persistence | PRESERVED → Phase 8 | Sidecar shutdown step eliminated |
| P9: Testing | PRESERVED → Phase 9 | Bridge/sidecar tests → MCP server tests |

---

## Success Criteria

- [x] Spike passes: permission wire format documented (TUI-001 ✓), Go MCP SDK POC verified (TUI-002 ✓), NDJSON catalog confirmed (TUI-003 ✓), UDS IPC validated at 56µs (TUI-004 ✓) — **Phase 1 complete**
- [ ] Two-process topology works: Go TUI → CLI → MCP tools — no Node.js
- [ ] Feature parity achieved: all 18 features from P9-6 checklist
- [x] Performance targets met: startup 0.31ms/<200ms, modal 0.002ms/<100ms, view 0.82ms/<16ms (all 5 benchmarks pass, TUI-040 ✅)
- [ ] No orphaned processes: graceful shutdown within 10s
- [x] Per-phase smoke tests pass: Phase 1 ✓, Phase 2 ✓, Phase 3 ✓, Phase 4 ✓, Phase 5 ✓, Phase 6 ✓ + integration wiring, Phase 7 ✓ + remediation, Post-Phase 7 review ✓ (21/21 packages green, race-clean), Phase 8 ✓ (lifecycle), Phase 9 partial ✓ (TUI-036 component + TUI-037 CLI integration + TUI-038 MCP integration 81.9% + TUI-039 E2E smoke + TUI-040 benchmarks all 5 targets pass)
- [x] Old ticket requirements traced: all 13 GOgent-109–121 mapped in traceability table below
- [ ] Ink TUI removable: `packages/tui/` deletable after parity
- [x] Race detector clean: `go test -race ./internal/tui/...` passes ✅ (verified after integration wiring)
- [x] Cost tracker functional: session cost, per-agent cost, budget enforcement (TUI-024 ✅, 97% coverage)

---

## Next Steps

~~1. Run `/ticket` to begin implementation with TUI-001 (prerequisite spike)~~ ✅ Phase 1 complete
~~2. Address review conditions (C-1, C-2) before Phase 2 tickets~~ ✅ C-1 Go 1.25+ applied, C-2 MCP SDK v1.2.0 confirmed
~~3. Address major issues (M-1 through M-5) before Phases 2-7~~ ✅ M-1 resolved (stdlib `flag`), M-2–M-5 deferred to respective tickets
~~4. Re-review after Phase 3 if significant design changes emerge from spike~~ ✅ No design changes; two-process topology validated
~~5. Continue with Phase 3: TUI-012 complete (NDJSON types), TUI-013 next (CLI subprocess driver)~~ ✅ Phase 3 complete
~~6. Phase 6 in progress~~ ✅ Phase 6 COMPLETE (TUI-022–027, 6/6 done)
~~6.5. Integration wiring~~ ✅ TUI-027.5: Placeholders replaced, components wired, streaming bug fixed, cost tracker unified
~~7. Phase 7 next: TUI-028 (multi-provider config)~~ ✅ TUI-028 COMPLETE
~~8. Phase 7 continues: TUI-029 (provider switching + message isolation)~~ ✅ TUI-029 COMPLETE
~~9. Phase 7 continues: TUI-030 (provider tab bar UI)~~ ✅ TUI-030 COMPLETE
~~10. Phase 7 continues: TUI-031 (provider session resume)~~ ✅ TUI-031 COMPLETE
~~11. Phase 7 final: TUI-032 (panels)~~ ✅ TUI-032 COMPLETE — **Phase 7 DONE**
~~12. Post-Phase 7 code review + fixes~~ ✅ FIX-1–6 bug fixes + DES-2–6 design refactors complete. 21/21 packages green, race detector clean.
~~13. Phase 8: TUI-033 (session persistence), TUI-034 (graceful shutdown — DES-1 wired), TUI-035 (clipboard/search/history)~~ ✅ Phase 8 COMPLETE
~~14. Phase 9 started: TUI-036 (component unit tests), TUI-037 (CLI driver integration test)~~ ✅ TUI-036 + TUI-037 COMPLETE
~~15. Phase 9 continues: TUI-038 (MCP server integration test)~~ ✅ TUI-038 COMPLETE (10 integration tests, 81.9% mcp coverage, race-clean)
~~16. Phase 9 continues: TUI-039 (E2E smoke test with live CLI)~~ ✅ TUI-039 COMPLETE (6 E2E tests, //go:build e2e tag, CLIDriver-direct harness, ~$0.05/run)
~~17. Phase 9 continues: TUI-040 (performance benchmarks)~~ ✅ TUI-040 COMPLETE (4 benchmark packages, all 5 targets pass: startup 0.31ms/200ms, modal 0.002ms/100ms, NDJSON 195K lines/sec vs 10K, view 0.82ms/16ms, UDS 0.009ms/5ms)
~~18. Phase 9 continues: TUI-041 (unknown event resilience)~~ ✅ TUI-041 COMPLETE (19 tests, 57 subtests, 91.2% cli coverage, race-clean, stress-tested)
19. Phase 9 final: TUI-042 (feature parity checklist — now unblocked by TUI-039)

## Implementation Progress (updated 2026-03-23, Phase 9 in progress)

| Phase | Status | Tickets | Notes |
|-------|--------|---------|-------|
| 1 | ✅ COMPLETE | TUI-001–004 | All 4 spikes done, results in `spike-results/` |
| 2 | ✅ COMPLETE | TUI-005–011 | 7/7 done. 174 tests, avg 95% coverage |
| 3 | ✅ COMPLETE | TUI-012–016 | 5/5 done. CLI driver, NDJSON parser, MCP server, UDS bridge, startup wiring |
| 4 | ✅ COMPLETE | TUI-017–018 | 2/2 done. Modal system + permission flow. 107 modals tests, 88.5% coverage |
| 5 | ✅ COMPLETE | TUI-019–021 | 3/3 done. AgentRegistry, tree/detail views, NDJSON sync. 249 tests across 3 pkgs |
| 6 | ✅ COMPLETE | TUI-022–027 | 6/6 done + integration wiring (TUI-027.5). All components wired into AppModel. 750+ tests |
| 7 | ✅ COMPLETE | TUI-028–032 | 5/5 done + remediation (R-1–R-4). Multi-provider, switching, panels, handoff, debounce. 1108 tests |
| Post-7 Review | ✅ COMPLETE | FIX-1–6, DES-2–6 | 4-reviewer code review (2026-03-23). 6 bug fixes + 4 design refactors. 21/21 packages green, race-clean. Staff Architect: APPROVE_WITH_CONDITIONS (High Confidence). DES-1 → TUI-034; DES-7 → TUI-036. |
| 8 | ✅ COMPLETE | TUI-033–035 | Session persistence (atomic writes, auto-save), graceful shutdown (5-phase LIFO, DES-1 resolved), clipboard/search/history. ~1153 tests, 23 packages |
| 9 | 🔧 IN PROGRESS | TUI-036–042 | TUI-036 ✅, TUI-037 ✅, TUI-038 ✅ (MCP integration 81.9%), TUI-039 ✅ (E2E smoke, 6 tests, //go:build e2e), TUI-040 ✅ (benchmarks, all 5 targets pass), TUI-041 ✅ (resilience, 19 tests/57 subtests, 91.2% coverage). TUI-042 pending |

### Phase 2 Package Tree (delivered)

```
internal/tui/
├── cli/                          # NDJSON events + CLI driver (TUI-012, TUI-013)
│   ├── events.go                 # 9 event structs, ParseCLIEvent()
│   ├── events_test.go            # 47 tests, 98.1% coverage
│   ├── driver.go                 # CLIDriver: subprocess lifecycle, NDJSON streaming
│   └── driver_test.go            # 60 tests, 78.6% coverage, race-free
├── mcp/                          # MCP server tools + IPC protocol (TUI-014)
│   ├── protocol.go               # IPCRequest/Response, payload types
│   ├── tools.go                  # 7 tool handlers + UDSClient
│   ├── tools_test.go             # 42 unit tests
│   ├── tools_coverage_test.go    # TUI-036 gap-filling tests
│   └── server_integration_test.go # TUI-038: 10 integration tests, 81.9% total coverage
├── bridge/                       # TUI-side UDS listener (TUI-015)
│   ├── server.go                 # IPCBridge, modal correlation, fire-and-forget dispatch
│   └── server_test.go            # 10 tests, 79% coverage, race-free
├── components/
│   ├── agents/                   # Agent tree + detail views (TUI-020)
│   │   ├── tree.go               # AgentTreeModel: Unicode box-drawing, scrollable
│   │   ├── tree_test.go          # 23 tests, 90.6% coverage
│   │   ├── detail.go             # AgentDetailModel: display-only, word-wrapped
│   │   └── detail_test.go        # 18 tests
│   ├── claude/                   # Claude conversation panel (TUI-022)
│   │   ├── panel.go              # ClaudePanelModel: viewport + textinput, streaming
│   │   └── panel_test.go         # 44 tests, 82.3% coverage
│   ├── banner/                   # BannerModel (TUI-009)
│   │   ├── banner.go
│   │   └── banner_test.go
│   ├── modals/                   # Modal system (TUI-017, TUI-018)
│   │   ├── types.go              # ModalType, ModalRequest, ModalResponse
│   │   ├── types_test.go
│   │   ├── model.go              # ModalModel: option selection, free-text "Other"
│   │   ├── model_test.go
│   │   ├── queue.go              # ModalQueue: FIFO queue, auto-activate next
│   │   ├── queue_test.go
│   │   ├── permission.go         # PermissionHandler: 6 flow types, multi-step ExitPlan
│   │   └── permission_test.go    # 107 total modals tests, 88.5% coverage
│   ├── statusline/               # StatusLineModel (TUI-009)
│   │   ├── statusline.go
│   │   └── statusline_test.go
│   ├── tabbar/                   # TabBarModel (TUI-009)
│   │   ├── tabbar.go
│   │   └── tabbar_test.go
│   ├── teams/                    # Team orchestration display (TUI-027)
│   │   ├── state.go              # TeamRegistry, TeamConfig/Wave/Member types
│   │   ├── state_test.go         # 17 tests
│   │   ├── list.go               # TeamListModel: polling, navigation, status display
│   │   ├── list_test.go          # 25 tests
│   │   ├── detail.go             # TeamDetailModel: wave-grouped member view
│   │   └── detail_test.go        # 23 tests — 94.0% coverage total
│   └── toast/                    # Toast notifications (TUI-025)
│       ├── toast.go              # ToastModel: auto-expire, max 3, level-colored
│       └── toast_test.go         # 17 tests, 93.9% coverage
├── config/                       # Theme + keybindings (TUI-005, TUI-007)
│   ├── theme.go                  # 7 colors, 10 styles, 6 icons, Theme struct
│   ├── theme_test.go
│   ├── keys.go                   # 24 bindings across 5 groups
│   └── keys_test.go
├── state/                        # Shared state (TUI-019, TUI-024, TUI-028)
│   ├── agent.go                  # AgentRegistry: RWMutex, dedup, DFS tree
│   ├── agent_test.go             # 56 tests
│   ├── cost.go                   # CostTracker: session/agent costs, budget (TUI-024)
│   ├── cost_test.go
│   ├── provider.go               # ProviderState: 4 providers, per-provider isolation (TUI-028)
│   ├── provider_test.go          # 161 tests (incl subtests), 97.5% coverage
│   └── task.go                   # TaskEntry type (DES-3: moved from taskboard to break model→taskboard import)
├── util/                         # Shared utilities (TUI-023)
│   ├── markdown.go               # Cached Glamour renderer, RenderMarkdown()
│   ├── markdown_test.go          # 14 tests, 87.0% coverage
│   ├── text.go                   # util.Truncate: UTF-8-safe truncation (FIX-1: replaces 5 duplicate helpers)
│   └── text_test.go
└── model/                        # Root AppModel + types (TUI-006, TUI-008)
    ├── focus.go                  # FocusTarget, RightPanelMode
    ├── focus_test.go
    ├── app.go                    # AppModel core (DES-2: split 1333→766 lines)
    ├── app_test.go
    ├── interfaces.go             # Widget interfaces: all mockable (DES-2: extracted, 225 lines)
    ├── layout.go                 # Layout compositor + sizing logic (DES-2: extracted, 257 lines)
    ├── provider_switch.go        # Provider switching handlers (DES-2: extracted, 135 lines)
    ├── startup.go                # CLI startup sequence, reconnection logic
    ├── startup_test.go
    ├── handoff.go                # Session handoff serialization
    └── messages.go               # 20+ tea.Msg types (expanded in TUI-016, TUI-018)

cmd/
├── gofortress/main.go            # TUI entry point (TUI-011)
└── gofortress-mcp/main.go        # MCP server stub (TUI-011)

internal/
└── teamconfig/
    └── config.go                 # TeamConfig/Wave/Member shared types (DES-6: moved from teams pkg to break import cycle)
```

### Key Design Decisions (Phase 2)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI arg parsing | stdlib `flag` | Review M-1: Cobra not in go.mod, only 7 flags |
| TabID type | int iota (not string) | Ticket spec said string; app.go already defined int — reused existing |
| model ↔ tabbar cycle | `tabBarWidget` interface | tabbar imports model.TabID; interface decouples reverse direction |
| ContentBlock | flat struct (not interface) | Simpler JSON unmarshaling, acceptable field overlap |
| Version injection | ldflags `-X main.version` | Standard Go pattern |
| Review C-1 | Go 1.25+ | Matches go.mod, applied to all tickets |
| Review C-2 | MCP SDK v1.2.0 | Spike TUI-002 confirmed working, no upgrade needed |

### Key Design Decisions (Phases 3-5)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| NDJSON parsing | Two-pass JSON (discriminator then full) | CLIUnknownEvent for unknown types, forward-compatible |
| CLI Driver event pump | Channel-to-Cmd re-subscription | Mandatory WaitForEvent() after each event; standard Bubbletea pattern |
| MCP server binary | Separate `gofortress-mcp` process | Clear identity, spawned by CLI, connects back to TUI via UDS |
| UDS IPC | json.NewEncoder/Decoder over net.Conn | 56µs roundtrip validated in spike |
| Permission flow | Option D hybrid (acceptEdits + MCP side-channel) | No control_request protocol; MCP tools need --allowedTools |
| PermissionHandler | 6 FlowTypes, multi-step ExitPlan | FlowEnterPlan, FlowExitPlan (2-step), FlowAskUser, FlowConfirm, FlowInput, FlowSelect |
| Modal → bridge response | ResolveModalSimple(requestID, value) | Avoids mcp import in model package |
| AgentRegistry | Flat map + computed tree cache, RWMutex | O(1) lookups, DFS tree only for View(), dedup on agentType+description |
| Agent tree rendering | Unicode box-drawing (├─ / └─) | Status icon colors: Green=Complete, Red=Error, Yellow=Running, Gray=Pending |
| Agent sync from NDJSON | SyncAssistantEvent/SyncUserEvent | Scans ContentBlock for Task tool_use → register; tool_result → complete/error |
| sharedState pointer | Heap-allocated struct on AppModel | Survives tea.NewProgram value-copy; holds cliDriver, bridge, modalQueue |
| ModalResponseMsg | Defined in modals package (not model) | Avoids circular import model → modals → model |
| BridgeModalRequestMsg | Defined in model package | Bridge imports model (not reverse); consistent dependency direction |

---

## Artifact Index

| File | Purpose |
|------|---------|
| `tickets/tui-migration/tickets/overview.md` | This file — executive summary |
| `tickets/tui-migration/tickets/tickets-index.json` | Machine-readable ticket registry |
| `tickets/tui-migration/tickets/TUI-001.md` through `TUI-042.md` | Individual ticket files |
| `.claude/sessions/20260316-plan-tickets-tui/strategy.md` | Strategic plan |
| `.claude/sessions/20260316-plan-tickets-tui/specs.md` | Detailed implementation specs |
| `.claude/sessions/20260316-plan-tickets-tui/review-critique.md` | Staff-architect review |
| `.claude/sessions/20260316-plan-tickets-tui/review-metadata.json` | Review verdict and counts |
| `tickets/tui-migration/braintrust-handoff-v2.md` | Braintrust analysis (foundation) |

---

_Generated by /plan-tickets skill. Review critique: .claude/sessions/20260316-plan-tickets-tui/review-critique.md_
