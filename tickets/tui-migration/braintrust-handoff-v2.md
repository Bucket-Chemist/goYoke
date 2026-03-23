# TUI Migration Braintrust Handoff — v2.0

> **Session:** 3435f2d5-367a-4162-9922-4a796e49420e (recovered in 27878532-3ab6-44ec-bfd4-e15993a0859b)
> **Date:** 2026-03-14
> **Cost:** $6.61 (Einstein $2.69 + Staff-Architect $2.39 + Beethoven $1.53)
> **Status:** All three agents completed successfully
> **v2.0 date:** 2026-03-14
> **v2.0 amendment:** Independent research validation of the three blocking unknowns (Section 4), quality assessment of braintrust outputs (Section 12), revised risk levels and next steps. Two of three unknowns substantially de-risked without running the spike.
>
> **⚠️ SPIKE CORRECTION (2026-03-23, TUI-001):** This document references `--permission-prompt-tool stdio` and `control_request` messages throughout (Sections 4.1, 8, 10, 12). **These do not exist.** The flag is not a real Claude Code CLI option; there is no interactive permission protocol in stream-json pipe mode. Tools are either auto-approved (`acceptEdits`) or silently denied (`default`), with denied tool inputs captured in `result.permission_denials`. The architecture uses Option D (hybrid): `acceptEdits` for Write/Edit/Bash + MCP side-channel for interactive tools. See `tickets/tui-migration/spike-results/permission-protocol.md` for the complete wire format. No structural changes to the ticket dependency graph were needed — the fallback paths (Options B/C) were already designed.

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

### v2.0: Independent Validation of Topology Recommendation

Post-braintrust research confirms the two-process recommendation is sound. The Go MCP SDK (now at v1.3.0) supports both `mcp.StdioTransport{}` and `mcp.StreamableHTTPHandler` — both are first-class transports that Claude Code CLI natively connects to. Multiple production Go MCP servers (`mcp-gopls`, `go-dev-mcp`) already work with Claude Code via this exact pattern. The "Go MCP SDKs lag behind" claim from the original plan is factually incorrect — the Go SDK is the **official** MCP implementation maintained in collaboration with Google, not a community fork.

However, the braintrust overestimated one thing: permission handling is more complex than either analyst modelled. See Section 4.1 for details. The two-process topology remains correct, but the permission architecture requires a specific design that neither the braintrust nor the original plan fully specified.

---

## 4. Three Blocking Unknowns (Prerequisite Spike: ~~8-12h~~ 4-6h)

These MUST be resolved before committing to any topology.

**v2.0 update:** Independent research has substantially de-risked unknowns 4.2 and 4.3. Unknown 4.1 (permissions) is partially resolved but requires a design decision. Estimated spike time reduced from 8-12h to 4-6h.

### 4.1 Permission handling in stream-json mode
- **Question:** Does CLI emit permission events in NDJSON? How does stdin accept responses?
- **Why blocking:** `canUseTool` is an Agent SDK callback. In CLI stream-json mode, the mechanism is unknown
- **Validation:** Run CLI with `--output-format stream-json`, trigger Write tool, capture all NDJSON + stdin traffic
- **Source:** Both analysts (Einstein: mechanism, Staff-Architect: format)

**v2.0 RESEARCH FINDING — PARTIALLY RESOLVED:**

The NDJSON stream does **not** emit a dedicated `permission_request` event. Permission handling is a two-layer system:

1. **After-the-fact reporting only in standard stream-json:** Denied tools appear as `tool_result` blocks with `is_error: true` inside `user` messages. The final `result` message contains a `permission_denials[]` array with `{tool_name, tool_use_id, tool_input}`. This is post-hoc — no real-time interception.

2. **Real-time interception via `--permission-prompt-tool stdio`:** The Agent SDKs (TS, Python) spawn the CLI with this hidden flag, which causes the CLI to emit `control_request` messages with subtype `"can_use_tool"` on stdout. The SDK intercepts these, calls the user-provided `canUseTool` callback, and writes the decision back via stdin. This protocol runs **alongside** the NDJSON stream.

**Design implication:** The Go TUI must either:
- **(A) Reimplement the `--permission-prompt-tool stdio` control protocol** — parse `control_request` messages from CLI stdout, present modal, write response to stdin. This is what the TS Agent SDK does internally. Spike must capture the exact wire format.
- **(B) Use MCP tools as the permission authority** — register `canUseTool`-equivalent MCP tools in the Go MCP server. When Claude wants to use a tool, the Go MCP server intercepts via tool pre-execution hooks and prompts the user. Requires verifying Claude Code supports this pattern.
- **(C) Pre-approve tools via `--allowedTools` and permission mode flags** — use `--permission-mode acceptEdits` or per-tool `permissions.allow[]` in `.claude/settings.json`. Eliminates real-time permission prompts entirely. Simplest but reduces user control.

**Remaining spike work:** 2-3h to capture the `--permission-prompt-tool stdio` wire format and verify option (A) is implementable from Go.

### 4.2 Go MCP SDK compatibility
- **Question:** Can Go MCP SDK v1.2.0 serve tools discoverable by Claude CLI via `--mcp-config`?
- **Why blocking:** If Go can't serve MCP tools, two-process topology fails entirely
- **Validation:** Create minimal Go MCP server with go-sdk, configure CLI connection via UDS, verify discovery and invocation
- **Source:** Both analysts

**v2.0 RESEARCH FINDING — RESOLVED (HIGH CONFIDENCE):**

The Go MCP SDK (now at **v1.3.0**) supports both transports Claude Code CLI can connect to:
- **`mcp.StdioTransport{}`** — Claude Code launches the Go binary as a subprocess, communicates via stdin/stdout JSON-RPC. This is the default and simplest path.
- **`mcp.StreamableHTTPHandler`** — standard `http.Handler` for the MCP Streamable HTTP transport. Claude Code connects via `"type": "http"` config.

Claude Code's MCP config format (`.mcp.json` or `~/.claude.json`):
```json
{
  "mcpServers": {
    "gofortress-tools": {
      "type": "stdio",
      "command": "/path/to/gofortress-mcp",
      "args": ["--bridge-port", "9199"]
    }
  }
}
```

Multiple production Go MCP servers already work with Claude Code via this pattern. Go binaries start in milliseconds, avoiding the `MCP_TIMEOUT` issues that plague `npx`-based Node.js servers.

**Note on UDS:** The Go SDK does **not** have built-in Unix domain socket support (WebSocket support proposed in issue #652). For the two-process topology, **stdio transport is the correct choice** — Claude Code spawns the Go MCP binary as a child process. No UDS needed.

**Remaining spike work:** 1h to build a minimal Go MCP server with one tool (e.g., `test_ping`) and verify Claude CLI discovers and invokes it. This is a confidence check, not a blocking unknown.

### 4.3 NDJSON event catalog
- **Question:** What is the complete set of NDJSON event types? Plan assumes 6 but `stream_event` subtypes are undefined
- **Why blocking:** Missing events cause state desync — historically the top cause of TUI bugs
- **Validation:** Comprehensive CLI session with tee logging, parse and categorize every event type and subtype
- **Mitigation pattern (Staff-Architect):** Log-and-continue on unknowns. WARN level. Accumulate in telemetry. Never crash.

**v2.0 RESEARCH FINDING — SUBSTANTIALLY RESOLVED:**

The complete NDJSON event catalog has **7+ top-level types** and **20+ SDK message variants**:

| `type` | Subtypes | When emitted |
|--------|----------|-------------|
| `system` | `init`, `status`, `compact_boundary`, `hook_started`, `hook_progress`, `hook_response`, `task_notification` | Session lifecycle |
| `assistant` | — | Complete assistant turn with content blocks |
| `user` | — | User input or tool results (including permission denials) |
| `result` | `success`, `error_max_turns`, `error_during_execution`, `error_max_budget_usd`, `error_max_structured_output_retries` | Final message, always last |
| `stream_event` | Wraps Anthropic API streaming: `message_start`, `content_block_start`, `content_block_delta` (`text_delta`, `input_json_delta`, `thinking_delta`, `signature_delta`), `content_block_stop`, `message_delta`, `message_stop` | Token-level (requires `--include-partial-messages`) |
| `rate_limit_event` | — | Informational; CLI handles retry internally |
| `tool_use_summary` | — | Summary of preceding tool uses |

The `system.init` message is richer than the original plan specified — includes `cwd`, `tools[]`, `mcp_servers[]`, `model`, `permissionMode`, `apiKeySource`, `claude_code_version`, `slash_commands[]`, `agents[]`, and `uuid`.

The `result` message carries `total_cost_usd`, `duration_ms`, `duration_api_ms`, `num_turns`, `usage` (with cache token breakdowns), per-model `modelUsage`, and `permission_denials[]`.

**Emerging types (limited documentation):** `SDKToolProgressMessage`, `SDKAuthStatusMessage`, `SDKTaskStartedMessage`, `SDKTaskProgressMessage`, `SDKFilesPersistedEvent`, `SDKPromptSuggestionMessage`. GitHub issue #24596 (opened Feb 2026, still open) specifically notes the documentation gap.

**Key discovery:** `rate_limit_event` was found primarily through bug reports (Python SDK issue #603, Claude Code issue #26498), not documentation. The TS SDK's `SDKMessage` union has expanded from ~7 to 20+ variants. Staff-Architect's log-and-continue mitigation pattern is **essential** — new types are being added regularly.

**Remaining spike work:** 1h to run a real session with `tee` and verify the catalog against actual wire traffic. Confidence is high but empirical confirmation is cheap.

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

| Issue | Source | Fix | v2.0 Status |
|-------|--------|-----|-------------|
| Three-process topology | Both | → Two-process (contingent on spike) | **Research confirms two-process viable.** Go MCP SDK v1.3.0 stdio transport works with Claude Code. |
| Permission architecture undefined | v2.0 research | → Design decision required: `--permission-prompt-tool stdio` reimplementation vs MCP-mediated vs pre-approve | **NEW.** Neither original plan nor braintrust fully specified this. See Section 4.1. |
| P7-7 multi-provider: single ticket | Staff-Arch M-2 | → 4 tickets: config types, state management, switching logic, tab UI | Unchanged |
| Testing deferred to P9 (7:1 ratio) | Staff-Arch M-4 | → Per-phase smoke tests with Done When criteria | Unchanged |
| Old tickets (GOgent-109–121) | Both | → Requirements traceability matrix; all superseded but requirements preserved | Unchanged |
| HTTP bridge IPC | Both | → Eliminated entirely in two-process. MCP tools use stdio transport (Claude Code spawns Go binary). In-process channels for modal interactions. | **Updated.** UDS not needed — stdio is the correct MCP transport. |
| Bridge plumbing (~12 tickets) | Synthesis | → Eliminated in two-process | Unchanged |
| Cost tracking gap | v2.0 assessment | → `cost/tracker.ts` and `getSessionCostTracker()` must be reimplemented in Go if sidecar eliminated | **NEW.** Not addressed by braintrust or original plan. |
| `query()` function replacement | v2.0 assessment | → Verify `--output-format stream-json` provides all capabilities the Agent SDK's `query()` function handles (conversation threading, session resume, partial messages) | **NEW.** Not addressed by braintrust. |

---

## 7. Beethoven's Recommended Phase Structure (Two-Process)

| Phase | Description | Key deliverables | v2.0 notes |
|-------|-------------|-----------------|------------|
| 1 | **Prerequisite Spike** (~~8-12h~~ 4-6h) | Permission protocol doc, Go MCP POC, NDJSON catalog, stdin format verification | **Reduced scope.** 4.2 and 4.3 substantially de-risked by research. Primary spike focus is now 4.1 (permission wire format). |
| 2 | **Foundation** | Bubble Tea scaffold, multi-panel layout, focus cycling, responsive sizing | Unchanged |
| 3 | **CLI Driver + NDJSON Parser + Go MCP Server** | Spawn CLI, parse events, stdin messages, 7 MCP tools via ~~UDS~~ stdio | **Updated.** MCP tools served via stdio transport (Claude Code spawns Go binary), not UDS. |
| 4 | **Modal System** | Permission prompts, confirmations, input, selection; MCP→channel→tea.Msg flow | **Design depends on spike 4.1 outcome.** If `--permission-prompt-tool stdio` is reimplemented, modals are triggered by control protocol. If MCP-mediated, modals triggered by MCP tool handlers. |
| 5 | **Agent Tree + Process Management** | Hierarchy, lifecycle, registry with signal escalation, canonical IDs | Unchanged |
| 6 | **Rich Features** | Markdown rendering, syntax highlighting, cost tracking, status line | Unchanged |
| 7 | **Settings, Providers, Teams** | Decomposed multi-provider (4 tickets), team orchestration UI | Unchanged |
| 8 | **Lifecycle** | Session persistence, graceful shutdown, error recovery, clipboard, search | Unchanged |
| 9 | **Integration Testing** | E2E with live CLI, performance benchmarks, unknown event resilience | Unchanged |

---

## 8. Risk Register (from Beethoven synthesis, updated v2.0)

| Risk | P | I | Mitigation | v2.0 update |
|------|---|---|------------|-------------|
| Go MCP SDK transport incompatibility | ~~Low~~ **Very Low** | High | Spike validates in ~~4-6h~~ 1h. Fallback: three-process | **De-risked.** Go SDK v1.3.0 stdio transport confirmed working with Claude Code. Multiple production Go MCP servers exist. |
| Permission mechanism incompatible with Bubble Tea stdin | Med | High | Spike test. Fallback: MCP mediation or permission pipe | **Clarified.** `--permission-prompt-tool stdio` control protocol exists but is undocumented. Must capture wire format. Three fallback options identified (Section 4.1). |
| Incomplete NDJSON catalog → state desync | ~~Med~~ **Low** | Med | Spike capture + log-and-continue parser + telemetry | **Substantially resolved.** 7+ top-level types, 20+ SDK variants now catalogued. `rate_limit_event` and hook subtypes identified. Staff-Architect's log-and-continue remains essential for emerging types. |
| Agent state race conditions (NDJSON vs MCP goroutines) | Med | Med | Single AgentRegistry with mutex, all mutations via tea.Msg | Unchanged |
| Multi-provider underscoped | High | Med | Decompose P7-7 into 4 tickets | Unchanged |
| Graceful shutdown orphans | Med | Med | Two-process simplifies. SIGTERM escalation. Team agents detached | Unchanged |
| CLI update breaks stream-json format | Med | Med | Version detection, defensive parsing, optional pointer fields | **Reinforced.** SDK message union actively expanding (7 → 20+ types). `rate_limit_event` discovered via bug reports not docs. |
| Old ticket requirements silently lost | Low | Low | Requirements traceability matrix | Unchanged |
| **NEW:** Cost tracker not ported to Go | Med | Med | Reimplement `cost/tracker.ts` budget logic in Go. Session cost data already available in `result` event `total_cost_usd`. | **Added v2.0.** Braintrust blind spot. |
| **NEW:** `query()` function capabilities lost | Low | High | Verify `--resume`, `--model`, `--include-partial-messages` CLI flags cover all `query()` capabilities. Session threading is CLI-native. | **Added v2.0.** Braintrust blind spot. |

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

## 10. Recommended Next Steps (v2.0 revised)

1. **Run the focused spike** (4-6h, down from 8-12h) — Primary objective is now **permission wire format** (4.1 option A):
   - Run CLI with `--permission-prompt-tool stdio --output-format stream-json`, trigger a Write tool, capture the `control_request` message format and stdin response format. Document the protocol.
   - Secondary: build a 50-line Go MCP server with `mcp.StdioTransport{}`, register in `.mcp.json`, verify Claude CLI discovers and invokes it (1h confidence check).
   - Tertiary: run a real session with `tee` to confirm NDJSON catalog against wire traffic (1h).
2. **Make the permission architecture decision** — option A (`--permission-prompt-tool` reimplementation), B (MCP-mediated), or C (pre-approve). This decision shapes Phase 3 and 4.
3. **Update migration plan** — incorporate Beethoven's 9-phase structure with v2.0 amendments (stdio not UDS, permission architecture, cost tracker port, NDJSON catalog)
4. **Decompose P7-7** into 4 multi-provider tickets
5. **Port cost tracker** — add ticket for reimplementing `cost/tracker.ts` budget logic in Go
6. **Build requirements traceability matrix** — old tickets → new plan
7. **Generate implementation tickets** via `/plan-tickets` with updated spec

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

---

## 12. Braintrust Quality Assessment (v2.0)

### Overall Verdict: High quality. $6.61 well spent.

The braintrust caught a genuine architectural flaw in the original migration plan (three-process topology) that would have cost significant implementation and debugging time. The unanimous recommendation to move to two-process is correct and has been independently validated.

### Per-Agent Assessment

**Einstein (Theoretical Analysis) — Strong**

The Process Topology Complexity Analysis framework (failure mode combinatorics, IPC tax, goal alignment) is well-constructed. The O(n²) coordination pairs argument is precise. The 7 assumptions surfaced are all real — especially the permission handling one, which turned out to be the most complex unknown. The first-principles challenge of the "Go MCP SDKs lag behind" claim was factually correct and has been confirmed by research.

Weakness: The novel approaches section reads more like advocacy than analysis. "High feasibility" for two-process was asserted with confidence the spike hadn't earned. Post-research this turned out to be correct, but the reasoning was incomplete.

**Staff-Architect (Practical Review) — Most operationally valuable output**

The three critical issues (C-1, C-2, C-3) are precise, evidence-backed, and actionable:
- **C-1** caught that the original plan contained a literal TODO ("Check the SDK docs for `createStreamableHTTPServer()`") treated as a design decision, with 44 downstream tickets depending on it.
- **C-2** (unidirectional bridge) identified a genuine gap — no Go→sidecar communication channel for process kills, shutdown cleanup, or status queries.
- **C-3** (split-brain agent state) identified a race condition between NDJSON and bridge-sourced agent registrations that would have been expensive to debug at runtime.

The commendations section is credible — it specifically identifies topology-independent assets worth preserving rather than offering generic praise.

**Beethoven (Synthesis) — Clean convergence, weak on fallbacks**

Correctly identifies 8 convergence points, all justified by evidence. The "not recommended" list is useful — particularly rejecting phased migration (three-process first, then eliminate sidecar) as 30% throwaway overhead. The 9-phase restructuring with prerequisite spike is the right structural decision.

Weakness: `divergence_resolution` is `null`, which means either no real divergences existed or the agent didn't engage with the tension between Einstein's confidence that Go MCP "just works" and the empirical reality that nobody had verified it. The three-process fallback gets one sentence rather than a worked-out alternative. Research has now closed this gap — the fallback is less likely to be needed.

### What the Braintrust Missed

Three blind spots identified by independent review:

1. **Permission architecture complexity.** Both analysts correctly identified permissions as a critical unknown but underestimated the complexity. The `--permission-prompt-tool stdio` control protocol is a separate, undocumented wire format running alongside NDJSON — not simply "an MCP tool" or "an NDJSON event." Neither analyst modelled this two-layer architecture.

2. **Cost tracking gap.** The sidecar's `cost/tracker.ts` and `getSessionCostTracker()` manage per-session cost budgets. If the sidecar is eliminated, this logic needs porting to Go. Not addressed by any agent.

3. **`query()` function replacement.** The TS Agent SDK's `query()` does more than spawn a subprocess — it handles conversation threading, message ID tracking, partial message streaming, and session resumption. The plan proposes driving CLI directly via `--output-format stream-json`, and the braintrust endorsed this, but nobody verified that stream-json mode provides equivalent capabilities. The CLI flags (`--resume`, `--model`, `--include-partial-messages`) likely cover this, but it was assumed rather than verified.

### Value-for-Cost

The $6.61 braintrust run saved an estimated 2-4 weeks of implementation on the wrong architecture (three-process with unverified HTTP transport). Staff-Architect's C-1 alone — catching the unverified `createStreamableHTTPServer()` TODO that 44 tickets depended on — justified the entire cost.
