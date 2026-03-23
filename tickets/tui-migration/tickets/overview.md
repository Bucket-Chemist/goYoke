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
| UDS startup race | Low | Low | MCP server retries with exponential backoff (5 attempts) |
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

- [x] Spike passes: permission wire format documented (TUI-001 ✓), Go MCP SDK POC verified (TUI-002 ✓), NDJSON catalog confirmed (TUI-003 ✓)
- [ ] Two-process topology works: Go TUI → CLI → MCP tools — no Node.js
- [ ] Feature parity achieved: all 18 features from P9-6 checklist
- [ ] Performance targets met: startup <200ms, modal <100ms, no frame drops
- [ ] No orphaned processes: graceful shutdown within 10s
- [ ] Per-phase smoke tests pass: each phase verified before proceeding
- [ ] Old ticket requirements traced: all 13 GOgent-109–121 mapped
- [ ] Ink TUI removable: `packages/tui/` deletable after parity
- [ ] Race detector clean: `go test -race ./internal/tui/...` passes
- [ ] Cost tracker functional: session cost, per-agent cost, budget enforcement

---

## Next Steps

1. Run `/ticket` to begin implementation with TUI-001 (prerequisite spike)
2. Address review conditions (C-1, C-2) before Phase 2 tickets
3. Address major issues (M-1 through M-5) before Phases 2-7
4. Re-review after Phase 3 if significant design changes emerge from spike

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
