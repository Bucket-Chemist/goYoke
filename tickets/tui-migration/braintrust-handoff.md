# TUI Migration Braintrust Handoff

> **Session:** 3435f2d5-367a-4162-9922-4a796e49420e (recovered in 27878532-3ab6-44ec-bfd4-e15993a0859b)
> **Date:** 2026-03-14
> **Cost:** $6.61 (Einstein $2.69 + Staff-Architect $2.39 + Beethoven $1.53)
> **Status:** All three agents completed successfully

---

## 1. Raw Output Files (READ THESE FOR FULL FIDELITY)

All outputs are structured JSON conforming to the braintrust stdin/stdout schemas.

| File | Agent | Size | What it contains |
|------|-------|------|------------------|
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_einstein.json` | Einstein | 34KB | Theoretical analysis: topology critique, 3 novel approaches, 7 assumptions surfaced, first principles analysis, tradeoff matrix |
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_staff-arch.json` | Staff-Architect | 42KB | Practical review: 7-layer analysis, 3 critical + 5 major issues, assumption register, dependency analysis, failure modes, ticket reconciliation |
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_beethoven.json` | Beethoven | 27KB | Unified synthesis: 8 convergence points, 5 divergence resolutions, primary recommendation (two-process), 9-phase implementation, 8 risks, 7 assumptions to validate |
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/pre-synthesis.md` | gogent-team-prepare-synthesis | 60KB | Merged Wave 1 outputs fed to Beethoven |
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/problem-brief.md` | Mozart | 12KB | Problem brief with scope, analysis axes, constraints, anti-scope |
| `~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/config.json` | gogent-team-run | 3KB | Final team state with costs, PIDs, health |

### Schema references (for parsing stdout JSON)

| Schema | Path |
|--------|------|
| Einstein stdout | `~/.claude/schemas/teams/stdin-stdout/braintrust-einstein.json` |
| Staff-Architect stdout | `~/.claude/schemas/teams/stdin-stdout/braintrust-staff-architect.json` |
| Beethoven stdout | `~/.claude/schemas/teams/stdin-stdout/braintrust-beethoven.json` |

---

## 2. Original Plan Under Review

**File:** `tickets/tui-migration/GOgent-Fortress-TUI-Migration-Plan.md` (67KB, 1466 lines, 56 tickets, 10 phases)

**What it proposes:** React/Ink/TypeScript → Go/Bubble Tea TUI migration using three-process topology (Go TUI + TS MCP sidecar + Claude Code CLI) with HTTP bridge IPC.

---

## 3. Core Finding: Topology Change

### UNANIMOUS: Three-process → Two-process

Both Einstein and Staff-Architect independently concluded the three-process topology should be replaced with two-process (Go TUI with native Go MCP server + Claude CLI).

**Einstein's reasoning (theoretical):**
- Process Topology Complexity Analysis: 9 failure modes (three-process) vs 3 (two-process)
- IPC tax: 2-5ms per modal (HTTP bridge) vs ~0ms (in-process channels)
- Goal contradiction: migration aims for single-binary Go, but plan retains permanent Node.js sidecar
- Root cause: sidecar exists due to historical path dependency, not architectural necessity
- Go MCP SDK v1.2.0 is official implementation, already in go.mod

**Staff-Architect's reasoning (practical):**
- C-1 (CRITICAL): `createSdkMcpServer` HTTP transport is an unverified TODO — 44 tickets depend on it
- C-2 (CRITICAL): Bridge is unidirectional (sidecar→Go only). No Go→sidecar channel exists. Missing for permission responses
- C-3 (CRITICAL): Split-brain agent state — dual registration sources (NDJSON + MCP) with no dedup key
- Total MCP tool code: ~300 lines TS (excluding process management already in Go)
- Verdict: REVISE

### Fallback if spike fails

If Go MCP SDK can't serve tools to Claude CLI → three-process with Staff-Architect's fixes:
- Bidirectional bridge (not unidirectional)
- Single AgentRegistry with canonical IDs
- UDS instead of HTTP (existing `gofortress-{pid}.sock` pattern)

---

## 4. Three Blocking Unknowns (Prerequisite Spike: 8-12h)

These MUST be resolved before committing to any topology:

### 4.1 Permission handling in stream-json mode
- **Question:** Does CLI emit permission events in NDJSON? How does stdin accept responses?
- **Why blocking:** `canUseTool` is an Agent SDK callback. In CLI stream-json mode, the mechanism is unknown
- **Validation:** Run CLI with `--output-format stream-json`, trigger Write tool, capture all NDJSON + stdin traffic
- **Source:** Both analysts (Einstein: mechanism, Staff-Architect: format)

### 4.2 Go MCP SDK compatibility
- **Question:** Can Go MCP SDK v1.2.0 serve tools discoverable by Claude CLI via `--mcp-config`?
- **Why blocking:** If Go can't serve MCP tools, two-process topology fails entirely
- **Validation:** Create minimal Go MCP server with go-sdk, configure CLI connection via UDS, verify discovery and invocation
- **Source:** Both analysts

### 4.3 NDJSON event catalog
- **Question:** What is the complete set of NDJSON event types? Plan assumes 6 but `stream_event` subtypes are undefined
- **Why blocking:** Missing events cause state desync — historically the top cause of TUI bugs
- **Validation:** Comprehensive CLI session with tee logging, parse and categorize every event type and subtype
- **Mitigation pattern (Staff-Architect):** Log-and-continue on unknowns. WARN level. Accumulate in telemetry. Never crash.

---

## 5. What to Preserve From Original Plan

These sections are topology-independent and both analysts praised them:

- **State translation mapping** (9 Zustand slices → ~8 Bubble Tea child models) — "unusually thorough"
- **UI component mapping** — UnifiedTree, UnifiedDetail, Modal system identification
- **Keybinding inventory** — comprehensive
- **Bubble Tea v1 architecture patterns** — correct decision over v2
- **go-bubbletea.md conventions** — must-follow

---

## 6. What Must Change

| Issue | Source | Fix |
|-------|--------|-----|
| Three-process topology | Both | → Two-process (contingent on spike) |
| P7-7 multi-provider: single ticket | Staff-Arch M-2 | → 4 tickets: config types, state management, switching logic, tab UI |
| Testing deferred to P9 (7:1 ratio) | Staff-Arch M-4 | → Per-phase smoke tests with Done When criteria |
| Old tickets (GOgent-109–121) | Both | → Requirements traceability matrix; all superseded but requirements preserved |
| HTTP bridge IPC | Both | → UDS for external IPC, in-process channels for internal |
| Bridge plumbing (~12 tickets) | Synthesis | → Eliminated in two-process |

---

## 7. Beethoven's Recommended Phase Structure (Two-Process)

| Phase | Description | Key deliverables |
|-------|-------------|-----------------|
| 1 | **Prerequisite Spike** (8-12h) | Permission protocol doc, Go MCP POC, NDJSON catalog, stdin format verification |
| 2 | **Foundation** | Bubble Tea scaffold, multi-panel layout, focus cycling, responsive sizing |
| 3 | **CLI Driver + NDJSON Parser + Go MCP Server** | Spawn CLI, parse events, stdin messages, 7 MCP tools via UDS |
| 4 | **Modal System** | Permission prompts, confirmations, input, selection; MCP→channel→tea.Msg flow |
| 5 | **Agent Tree + Process Management** | Hierarchy, lifecycle, registry with signal escalation, canonical IDs |
| 6 | **Rich Features** | Markdown rendering, syntax highlighting, cost tracking, status line |
| 7 | **Settings, Providers, Teams** | Decomposed multi-provider (4 tickets), team orchestration UI |
| 8 | **Lifecycle** | Session persistence, graceful shutdown, error recovery, clipboard, search |
| 9 | **Integration Testing** | E2E with live CLI, performance benchmarks, unknown event resilience |

---

## 8. Risk Register (from Beethoven synthesis)

| Risk | P | I | Mitigation |
|------|---|---|------------|
| Go MCP SDK transport incompatibility | Low | High | Spike validates in 4-6h. Fallback: three-process |
| Permission mechanism incompatible with Bubble Tea stdin | Med | High | Spike test. Fallback: MCP mediation or permission pipe |
| Incomplete NDJSON catalog → state desync | Med | Med | Spike capture + log-and-continue parser + telemetry |
| Agent state race conditions (NDJSON vs MCP goroutines) | Med | Med | Single AgentRegistry with mutex, all mutations via tea.Msg |
| Multi-provider underscoped | High | Med | Decompose P7-7 into 4 tickets |
| Graceful shutdown orphans | Med | Med | Two-process simplifies. SIGTERM escalation. Team agents detached |
| CLI update breaks stream-json format | Med | Med | Version detection, defensive parsing, optional pointer fields |
| Old ticket requirements silently lost | Low | Low | Requirements traceability matrix |

---

## 9. Related Artifacts

### TeamDashboard spec (produced during this session)

**File:** `tickets/tui-agent-upgrade/spec.md`

Spec for upgrading the TUI agent detail panel to show health monitoring dashboard (health_status, stall_count, stream sizes, budget bar, wave-grouped view). Inspired by the health monitor used to track this braintrust run. ~260 lines, 1 new file, 5 modified.

### Previous TUI tickets (for reconciliation)

**Files:**
- `dev/will/migration_plan/tickets/TUI/tui-tickets-json-entries.json` — GOgent-109 to GOgent-121
- `dev/will/migration_plan/tickets/TUI/README.md` — Overview
- `dev/will/migration_plan/tickets/TUI/CONSTRUCTION-SUMMARY.md` — Week/priority mapping

All 13 are architecturally superseded by the two-process recommendation but requirements must be traced.

---

## 10. Recommended Next Steps

1. **Run the spike** (8-12h) — resolves all three blocking unknowns
2. **Based on spike results:** finalize topology decision
3. **Update migration plan** — incorporate Beethoven's 9-phase structure
4. **Decompose P7-7** into 4 multi-provider tickets
5. **Build requirements traceability matrix** — old tickets → new plan
6. **Generate implementation tickets** via `/plan-tickets` with updated spec

---

## 11. How to Resume

```
# Read the full synthesis
cat ~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_beethoven.json | python3 -m json.tool

# Read Einstein's theoretical analysis
cat ~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_einstein.json | python3 -m json.tool

# Read Staff-Architect's practical review
cat ~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/stdout_staff-arch.json | python3 -m json.tool

# Read the problem brief
cat ~/.claude/sessions/3435f2d5-367a-4162-9922-4a796e49420e/teams/1773453175.braintrust/problem-brief.md
```

To hand off to a new agent, point it at this file:
```
Read tickets/tui-migration/braintrust-handoff.md then read the three stdout JSON files referenced in Section 1.
```
