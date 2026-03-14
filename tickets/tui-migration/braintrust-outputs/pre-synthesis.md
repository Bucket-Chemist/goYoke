# Pre-Synthesis Input for Beethoven

This document contains extracted insights from Einstein (theoretical analysis) and Staff-Architect (critical review) for Beethoven to synthesize.

---

## Einstein: Theoretical Analysis

### Executive Summary

The three-process topology (Go TUI + TS MCP sidecar + Claude Code CLI) is architecturally over-engineered and self-contradictory: the entire motivation for migrating to Go is single-binary distribution and eliminating Node.js, yet the plan retains a Node.js sidecar as a permanent dependency. The Go MCP SDK (modelcontextprotocol/go-sdk v1.2.0, already in go.mod) can fully replace `createSdkMcpServer` since MCP is a language-agnostic protocol standard. A two-process topology (Go TUI with native MCP + Claude Code CLI) eliminates the HTTP bridge, removes N² process coordination failure modes, and achieves the stated migration goal. The Zustand-to-Elm state translation is low-risk and natural. However, the migration plan critically underspecifies the permission handling mechanism in `--output-format stream-json` mode — the current `canUseTool` callback is an Agent SDK concept that doesn't exist in direct CLI mode, and the plan silently assumes MCP tools will replace it without verifying how Claude CLI actually handles permission prompts in stream-json mode.

### Root Cause Analysis

- **Historical path dependency: TS sidecar exists because the TUI was originally TypeScript, not because TypeScript is architecturally required for MCP** (confidence: high)
  - Evidence: The MCP server uses `createSdkMcpServer` from `@anthropic-ai/claude-agent-sdk` — a convenience wrapper around standard MCP protocol. The Go MCP SDK (modelcontextprotocol/go-sdk v1.2.0) implements the same protocol standard and is already in go.mod. The sidecar's most complex tool (spawnAgent.ts) spawns Go CLI processes and does process management, which `internal/lifecycle/process.go` and `gogent-team-run` already handle natively in Go.
  - Scope: packages/tui/src/mcp/, packages/tui/src/spawn/, packages/tui/src/session/SessionManager.ts

- **Unverified assumption about permission handling mechanism in --output-format stream-json mode** (confidence: high)
  - Evidence: The current TUI uses `canUseTool` — an Agent SDK `query()` callback (SessionManager.ts:407,977-1096). The migration plan claims this moves to MCP tools calling bridge API (P4-5), but `canUseTool` is NOT an MCP tool; it's a host callback. With direct CLI subprocess (no `query()`), the permission mechanism is fundamentally different. The plan does not verify or document how Claude CLI handles permission prompts in stream-json mode — whether via NDJSON events requiring stdin responses, `--permission-mode` flags, or some other mechanism.
  - Scope: internal/cli/driver.go (proposed), internal/bridge/ (proposed), packages/tui/src/session/SessionManager.ts

- **HTTP bridge introduces unnecessary latency and failure modes for localhost IPC** (confidence: high)
  - Evidence: The bridge API (localhost:9199) is HTTP over TCP for communication between two processes on the same machine. Every modal interaction (permission prompt, agent registration, toast) traverses: TS sidecar → HTTP serialize → TCP → HTTP deserialize → Go handler → tea.Msg → user response → HTTP serialize → TCP → TS sidecar. This adds ~1-5ms per call plus two TCP connection establishments. Port conflicts (:9198, :9199) create startup race conditions. Unix domain sockets or in-process function calls would eliminate both latency and port conflicts.
  - Scope: internal/bridge/ (proposed), sidecar/src/bridge.ts (proposed)

- **Three-process topology creates N² failure mode combinations** (confidence: high)
  - Evidence: With three processes (Go TUI, TS sidecar, Claude CLI), there are 6 two-process failure combinations and 3 single-process failure scenarios to handle. The plan addresses only two: 'If sidecar dies, attempt restart' and 'NDJSON scanner EOF on CLI death.' Missing: sidecar dies mid-modal (HTTP handler blocks forever on dead connection), Go TUI crashes (orphans both sidecar and CLI), Claude CLI dies during sidecar restart (stale MCP config), simultaneous sidecar+CLI failure (irrecoverable state). Each requires distinct recovery logic.
  - Scope: internal/sidecar/launcher.go (proposed), internal/bridge/server.go (proposed), internal/cli/driver.go (proposed)

### Conceptual Frameworks

**Process Topology Complexity Analysis (Distributed Systems First Principles)**
This framework analyzes multi-process architectures through three lenses: (1) Failure Mode Combinatorics — each additional process creates multiplicative, not additive, failure scenarios; (2) IPC Tax — every cross-process boundary adds serialization, deserialization, and transport overhead that compounds under load; (3) Goal Alignment — architecture must serve the stated migration objectives, not perpetuate historical accidents. Applied to the three-process topology: the TS sidecar adds 4 unique failure modes, ~2-5ms per modal interaction, and directly contradicts the migration's goal of eliminating Node.js dependency.
Key insights:
  - Every process boundary is a serialization boundary, a failure boundary, and a deployment boundary. Minimizing process count minimizes all three.
  - The TS sidecar's value proposition ('tracks Claude Code CLI updates') assumes MCP protocol instability, but MCP is a standardized protocol with an official Go SDK. The instability assumption is unsubstantiated.
  - The most complex sidecar functionality (spawnAgent) is a TypeScript reimplementation of existing Go code (gogent-team-run, internal/lifecycle/process.go). Maintaining parallel implementations in two languages doubles maintenance cost.
  - The migration plan's HTTP bridge pattern (localhost HTTP between co-located processes) is an enterprise integration pattern designed for cross-machine communication, applied to a scenario where in-process function calls would suffice.
  - Process lifecycle coordination complexity grows as O(n²) with process count. Two processes have 1 coordination pair; three processes have 3 coordination pairs. Each pair needs startup ordering, health checking, failure detection, and graceful shutdown logic.


### First Principles Analysis

- **createSdkMcpServer tightly tracks Claude Code CLI updates — Go MCP SDKs lag behind** (validity: questionable)
  - Evidence: MCP is a protocol standard (modelcontextprotocol.io). The Go SDK (modelcontextprotocol/go-sdk v1.2.0, already in go.mod) is the OFFICIAL Go implementation from the MCP project — not a community fork. It tracks the same protocol specification as the TypeScript SDK. The `createSdkMcpServer` function provides convenience (Zod schema definitions, tool registration), not unique protocol capabilities. Go can register tool handlers with JSON schemas natively. The claim of SDK lag is unsubstantiated — both SDKs implement the same protocol version. The real risk would be Anthropic adding non-standard protocol extensions, which would violate MCP specification.
  - If wrong: If the Go MCP SDK does fully support Claude Code's MCP requirements, the entire rationale for the TS sidecar collapses. The three-process topology becomes unnecessary complexity.

- **HTTP bridge is necessary because Bubble Tea owns stdin/stdout** (validity: valid)
  - Evidence: Bubble Tea does own the terminal's stdin/stdout. However, this only means the MCP server cannot use STDIO transport on the terminal's file descriptors. The MCP server CAN use: (1) TCP on a Unix domain socket, (2) HTTP transport on localhost, or (3) named pipes. All of these work from within the Go process — no separate TS sidecar needed. The migration plan correctly identifies the stdin/stdout constraint but incorrectly concludes it requires a separate process.
  - If wrong: If the HTTP bridge is unnecessary, the architecture simplifies from three processes to two, and from 9 bridge API endpoints to zero.

- **Permission handling (canUseTool) will work via MCP tools in --output-format stream-json mode** (validity: questionable)
  - Evidence: canUseTool is an Agent SDK query() callback (SessionManager.ts:407). It does NOT exist in Claude CLI's --output-format stream-json interface. The migration plan (P4-5) says permissions move to the TS sidecar's MCP tools, but doesn't explain HOW Claude CLI requests permission in stream-json mode. Does it emit a NDJSON event and wait for stdin JSON response? Does --permission-mode handle it entirely? Does the CLI call an MCP tool? This is undocumented and unverified. The plan's risk register (Section 17) mentions 'stdin message format' uncertainty but doesn't identify this as the critical path it is.
  - If wrong: If Claude CLI's permission mechanism in stream-json mode is fundamentally different from what the plan assumes, Phase 4 (modals) needs complete redesign. This could cascade to Phases 5-9.

- **Bubble Tea v1 should be used over v2** (validity: valid)
  - Evidence: Bubble Tea v2 shipped Feb 2026 (~6 weeks ago). The tea.View struct API replaces string-based View(). While v2 is architecturally cleaner, the ecosystem (Bubbles v0.20.0, Lipgloss v1.1.0 in go.mod) may not be fully v2-compatible yet. The project already has bubbletea v1.3.10. Starting on v1 is pragmatic — migration to v2 can happen later when the ecosystem stabilizes. However, the plan should explicitly create a v2 migration ticket to avoid accumulating v1 technical debt.
  - If wrong: If v2 is stable enough, starting on v1 means a second migration later. But v1→v2 migration is well-documented and incremental (change View() signature, update Bubbles imports).

- **NDJSON protocol has approximately 6 well-defined event types** (validity: questionable)
  - Evidence: The plan identifies: system.init, system.status, system.compact_boundary, assistant, user (tool_results), result. But the CLIEvent struct also mentions stream_event as a type. The plan doesn't elaborate on stream_event, which likely corresponds to partial/streaming content blocks (SSE-style token streaming). Additionally, undocumented events may exist for: content_block_start, content_block_delta, content_block_stop (standard Anthropic streaming events). The plan's risk register acknowledges this ('Comprehensive logging of unknown event types') but treats it as low risk when it could cause state desync.
  - If wrong: Missing event types cause silent state desync — messages appear garbled, tool results are lost, or streaming indicators never clear. This is the #1 cause of TUI bugs in production.

- **[Constraint]** Bubble Tea's event loop (Init/Update/View) is single-threaded by design. All state mutations must flow through Update() via tea.Msg. External goroutines (bridge handlers, CLI parser) must use Program.Send() — never direct state mutation.

- **[Constraint]** Claude Code CLI's --output-format stream-json is the stable external interface. The NDJSON event schema IS the API contract. Any Go TUI must be resilient to unknown/new event types.

- **[Constraint]** MCP tool invocations from Claude CLI are synchronous from Claude's perspective — the tool call blocks until a response is returned. Modal interactions must complete before the MCP tool handler returns, which means blocking goroutines on channels.

- **[Constraint]** Process tree relationships must be maintained: Go TUI is parent of both Claude CLI and MCP server. If Go TUI dies, children must be cleaned up (SIGTERM/SIGKILL escalation). This is simpler with one child (two-process) than two children (three-process).

- **[Constraint]** The migration must preserve all hook binary functionality (gogent-validate, gogent-sharp-edge, etc.). These hooks run in response to Claude Code CLI events, independent of the TUI process.

### Novel Approaches

1. **Pure Go Two-Process Topology: Eliminate TS sidecar entirely, implement MCP server natively in Go**
The stated migration goal is single-binary Go distribution. Keeping a Node.js sidecar fundamentally contradicts this. The Go MCP SDK (v1.2.0, already in go.mod) provides the same MCP protocol capabilities. The sidecar's most complex tool (spawnAgent) reimplements existing Go code (gogent-team-run, internal/lifecycle/process.go). A native Go MCP server eliminates the HTTP bridge, removes 4 failure modes, and achieves true single-binary distribution. MCP tool handlers become Go functions that inject tea.Msg via Program.Send() — exactly what the HTTP bridge handlers do, minus the HTTP.
Feasibility: high
Pros: True single-binary distribution (no Node.js runtime dependency), Eliminates HTTP bridge (9 endpoints, TCP overhead, port conflicts), Reduces failure modes from 9 to 3 (two-process vs three-process), Faster startup (no Node.js cold start, estimated 500ms-1s savings), spawnAgent reuses existing Go code instead of maintaining TS parallel implementation, Modal interactions become in-process function calls (~0ms vs ~2-5ms HTTP), Simpler shutdown: one child process to clean up instead of two
Cons: Must reimplement 7 MCP tool handlers in Go (estimated ~400 lines total), If Anthropic adds non-standard MCP extensions to their TS SDK, Go SDK may not have them, Loses createSdkMcpServer convenience (Zod schemas → JSON Schema definitions), Relationship validation (relationshipValidation.ts, ~180 lines) must be ported to Go
Risks: Go MCP SDK may not support all MCP transport modes Claude CLI expects (verify before committing), If Claude CLI requires TS-specific MCP SDK features (unknown at time of analysis), this approach fails, Higher initial implementation effort for MCP server (~2 weeks vs ~1 week for sidecar extraction)

2. **Phased Migration: Three-Process (Phase A) → Two-Process (Phase B)**
Derisks the migration by first proving the Go TUI works with a thin TS sidecar (lower implementation risk), then eliminating the sidecar once the Go TUI is stable. This separates the UI migration risk from the MCP migration risk.
Feasibility: high
Pros: Lower risk per phase — each phase has a clear success criterion, Phase A validates the NDJSON parser, Bubble Tea UI, and modal system independently of MCP, Phase B can be deferred if the TS sidecar proves stable enough, Team can work on Phase A without Go MCP expertise
Cons: Total scope is ~30% larger (implement sidecar bridge, then replace it), Temporary HTTP bridge code is throwaway work, Maintains two IPC contracts during transition (bridge API + eventual Go MCP), Delays the single-binary goal to Phase B completion
Risks: Phase B may never happen ('temporary' becomes permanent), Bridge API design decisions leak into Go TUI architecture, creating tech debt, Team may resist Phase B after investing in Phase A bridge code

3. **Hybrid: Go MCP with TS canUseTool Adapter**
Implements most MCP tools natively in Go but keeps a minimal TS adapter ONLY for the canUseTool permission flow, since this is the least-understood interface between Claude CLI and the host. Once the permission mechanism in stream-json mode is verified and documented, the TS adapter is removed.
Feasibility: medium
Pros: Addresses the permission handling uncertainty without blocking the entire migration, Most MCP tools (5 of 7) move to Go immediately, TS adapter is tiny (~50 lines) compared to full sidecar (~800 lines), Adapter can be eliminated once stream-json permission mechanism is verified
Cons: Still requires Node.js for the adapter (smaller footprint than full sidecar), Two MCP servers in parallel (Go primary + TS adapter) adds configuration complexity, Temporary architecture that needs explicit retirement plan
Risks: Adapter may mask underlying permission protocol issues that surface later, Configuration complexity of dual MCP servers may cause tool registration conflicts

### Theoretical Tradeoffs

**Process Count (Topology Simplicity vs SDK Convenience)**
  - Option A: Three-process topology: Retains createSdkMcpServer convenience, uses familiar TS tooling for MCP, but adds Node.js dependency, HTTP bridge, and 6 additional failure modes
  - Option B: Two-process topology: Eliminates Node.js dependency, removes HTTP bridge, reduces failure modes by 67%, but requires reimplementing 7 MCP tools in Go (~400 lines) and verifying Go MCP SDK compatibility
  - Recommendation: Two-process. The entire migration motivation is single-binary Go distribution. The convenience savings of createSdkMcpServer (~200 lines of Zod schema definitions) do not justify a permanent Node.js runtime dependency, 9 HTTP bridge endpoints, and 6 additional failure modes. The Go MCP SDK is the official implementation — it's not a risk, it's the intended usage.

**IPC Mechanism (HTTP vs Unix Domain Socket vs In-Process)**
  - Option A: HTTP over TCP localhost (current plan): Standard, debuggable with curl, but adds TCP overhead, port allocation, and connection management
  - Option B: Unix domain socket or in-process function calls: Lower latency, no port conflicts, but less debuggable externally
  - Recommendation: Unix domain socket for MCP (Claude CLI → Go MCP server), in-process for modal interactions (MCP handler → Bubble Tea). The MCP protocol requires a socket/transport between Claude CLI and the MCP server. Unix domain sockets avoid port conflicts and are ~10x faster than TCP. Modal interactions within the Go process should be direct channel operations, not HTTP.

**Permission Handling (MCP-Mediated vs NDJSON-Native)**
  - Option A: Permission prompts handled via MCP tools (sidecar calls bridge API): Proven pattern from current Agent SDK, but depends on unverified assumption about Claude CLI stream-json behavior
  - Option B: Permission prompts handled via NDJSON event/stdin protocol: Native to Claude CLI, no MCP detour, but protocol is undocumented
  - Recommendation: This MUST be empirically verified before committing to either approach. Run Claude CLI with --output-format stream-json in interactive mode, trigger a tool requiring permission, and observe the NDJSON output. The answer determines the entire Phase 4 design. Do not design the modal system until this is verified.

**Bubble Tea Version (v1 Battle-Tested vs v2 Modern)**
  - Option A: v1 (current in go.mod, v1.3.10): String-based View(), battle-tested, full Bubbles/Lipgloss compatibility, but will need migration to v2 eventually
  - Option B: v2 (shipped Feb 2026): tea.View struct API, cleaner architecture, but ecosystem may not be fully compatible yet
  - Recommendation: Start with v1. The ecosystem (Bubbles v0.20.0, Lipgloss v1.1.0) is designed for v1. Add a P10 ticket for v2 migration when the ecosystem catches up. This avoids fighting two battles at once (TUI rewrite + v2 adoption).

**State Model (Flat AppModel vs Nested Component Models)**
  - Option A: Single flat AppModel struct with all fields: Simple, matches Elm architecture purity, but struct grows large (30+ fields from 9 Zustand slices)
  - Option B: Root AppModel with nested child model structs: Better encapsulation, each component manages its own state, matches Bubble Tea component composition pattern
  - Recommendation: Nested child models (as proposed in the migration plan). The plan correctly has AppModel containing ClaudePanelModel, AgentTreeModel, DashboardModel, etc. This matches go-bubbletea.md conventions and keeps each model's Update() focused. The 9 Zustand slices map naturally to ~8 child models. No semantic loss in translation.

### Assumptions Surfaced

- **Claude CLI's --output-format stream-json emits NDJSON events that include permission request events, and accepts permission responses via stdin JSON** (source: Implicit in migration plan Phase 2 (CLI driver) and Phase 4 (modal system). The plan assumes Go TUI can intercept permission prompts from the NDJSON stream, but this mechanism is never explicitly verified.)
  - Risk if false: If Claude CLI handles permissions internally (via terminal prompts) rather than through NDJSON events, the entire modal/permission system design fails. The Go TUI would need to either: (a) pipe through terminal I/O for permissions (conflicts with Bubble Tea owning stdin), or (b) use MCP tools for permission mediation (requires MCP server to be the permission authority).
  - Validation: Run: `claude --output-format stream-json` interactively. Trigger a tool that requires permission (e.g., Write to a file in default permission mode). Capture ALL NDJSON output and stdin traffic. Document the permission request/response protocol.

- **Go MCP SDK (modelcontextprotocol/go-sdk v1.2.0) supports all MCP features that Claude Code CLI uses** (source: Migration plan rationale table (Section 1): 'Go MCP SDKs lag behind'. Implicitly challenged by go.mod already containing go-sdk v1.2.0.)
  - Risk if false: If go-sdk doesn't support a required MCP feature (e.g., specific transport mode, tool schema format, or streaming protocol), the Go-native MCP server approach fails. Would need to either update go-sdk or keep TS sidecar.
  - Validation: Enumerate MCP features used by current TS MCP server (tool registration, in-process transport, tool result streaming). Verify each is supported in go-sdk v1.2.0 by checking go-sdk documentation and examples. Create a proof-of-concept Go MCP server with one tool and verify Claude CLI can connect and invoke it.

- **The NDJSON stream_event type corresponds to partial/streaming content and can be safely handled as incremental message updates** (source: Migration plan CLIEvent struct (P2-1) lists stream_event as a type but provides no handler or struct definition for it.)
  - Risk if false: If stream_event has complex semantics (e.g., SSE-style content_block_start/delta/stop with tool streaming), the parser (P2-3) will miss partial content blocks, causing the conversation display to show incomplete or garbled output.
  - Validation: Capture a real Claude CLI session with --output-format stream-json. Use `tee` to save all NDJSON lines. Count and categorize ALL event types, especially stream_event subtypes. Compare against the 6 types identified in the plan.

- **TS sidecar startup latency (500ms-1s) is acceptable and can be mitigated with esbuild bundling** (source: Migration plan Risk Register (Section 17).)
  - Risk if false: If the TUI feels sluggish at startup because it waits for Node.js to initialize, user experience suffers. This is a permanent tax in three-process topology but eliminated entirely in two-process.
  - Validation: Benchmark: time `node sidecar/dist/index.js` cold start vs time `./gofortress` native Go start. If >300ms difference, user perception of startup lag is real.

- **spawnAgent's process management (PID tracking, SIGTERM/SIGKILL escalation, output buffering) can remain in TS sidecar** (source: Migration plan P3-3: 'spawnAgent's process spawning (exec.spawn) stays in the TS sidecar'.)
  - Risk if false: Keeping process management in TS while the TUI is in Go creates a split-brain problem: Go TUI tracks agent state (for rendering), but TS sidecar tracks process state (PIDs, exit codes). If they disagree (e.g., Go thinks agent is running but TS has already killed it), the agent tree shows stale information.
  - Validation: Map all state that must be consistent between Go TUI and TS sidecar. Identify which process is the source of truth for each state field. If more than 3 fields require cross-process synchronization, the split is architecturally unsound.

- **Claude CLI stdin accepts line-delimited JSON for user messages in --output-format stream-json mode** (source: Migration plan P2-6: 'Write user messages to CLI stdin as JSON'.)
  - Risk if false: If Claude CLI expects plain text on stdin (not JSON), the message delivery mechanism needs to change. If it expects a specific JSON schema, that schema must be documented.
  - Validation: Run Claude CLI with --output-format stream-json. Attempt to send a JSON-formatted message via stdin. Verify it's processed correctly. Try plain text as well. Document the accepted format.

- **All 13 old TUI tickets (GOgent-109 to GOgent-121) are fully subsumed by the 56 new tickets** (source: Problem brief: 'reconcile 13 old tickets with 56 new tickets'.)
  - Risk if false: If old tickets contain requirements or design decisions not captured in the new plan, features may be lost. Specifically: TUI-CLI-01a (auto-restart logic) is not explicitly present in the new plan. TUI-TELEM-01 (file watchers for telemetry) uses chokidar; the new plan replaces with polling but may lose real-time telemetry updates.
  - Validation: Create a requirements traceability matrix: each old ticket requirement mapped to the specific new ticket that covers it. Flag any unmapped requirements.

### Open Questions

- **How does Claude CLI handle permission prompts in --output-format stream-json mode? Is it via NDJSON events, stdin protocol, MCP tools, or --permission-mode flag?** (importance: high)
  - Investigation: Run Claude CLI with --output-format stream-json in default permission mode. Trigger a Write tool. Capture all NDJSON output. Check if a permission event is emitted. Check if stdin accepts a permission response. This is the single most important empirical question for the migration — it determines the entire Phase 4 design.

- **Does the Go MCP SDK (go-sdk v1.2.0) support the same transport modes that Claude Code CLI expects? Specifically: can it serve MCP over Unix domain sockets or localhost HTTP with the exact handshake Claude CLI uses?** (importance: high)
  - Investigation: Create a minimal Go MCP server using go-sdk with one dummy tool. Configure Claude CLI to connect to it via --mcp-config. Verify tool discovery, invocation, and result return work correctly. If this proof-of-concept works, the entire two-process topology is validated.

- **What is the complete set of NDJSON event types emitted by --output-format stream-json? Are there undocumented events beyond the 6 identified in the plan?** (importance: high)
  - Investigation: Run a comprehensive Claude session (multi-turn, tool use, agent spawning, context compaction, model switching) with --output-format stream-json, logging all output. Parse and categorize every event type and subtype. Create a reference schema.

- **Is Bubble Tea v2's ecosystem (Bubbles, Lipgloss) production-ready as of March 2026? Would starting on v2 save a later migration?** (importance: medium)
  - Investigation: Check Bubbles and Lipgloss changelogs/releases for v2 compatibility statements. Try importing bubbletea v2 with current Bubbles v0.20.0 and Lipgloss v1.1.0. If compilation succeeds with all needed components, v2 is viable.

- **What happens to the SessionManager's AsyncGenerator/MessageCoordinator pattern in the new architecture? Is there an equivalent coordination mechanism needed for the Go CLI driver?** (importance: medium)
  - Investigation: The SessionManager uses an AsyncGenerator to keep the CLI process alive between messages (yielding user messages, waiting for response promises). The Go CLI driver uses stdin/stdout pipes directly. Verify that writing to stdin and reading from stdout provides equivalent coordination without an explicit state machine — i.e., that Claude CLI processes messages sequentially and doesn't require explicit flow control.

### Handoff Notes

Key points for Beethoven to focus on during synthesis:

1. **TOPOLOGY IS THE CENTRAL DECISION**: The theoretical analysis strongly favors two-process over three-process topology. Staff-Architect's practical review will likely have implementation-level concerns (Go MCP SDK maturity, effort estimates). Beethoven should weigh theoretical simplicity against practical risk to produce a clear recommendation with contingency.

2. **PERMISSION HANDLING IS THE CRITICAL UNKNOWN**: Both Einstein and Staff-Architect should flag the canUseTool → stream-json permission mechanism as the #1 risk. This must be empirically verified BEFORE finalizing the architecture. Beethoven should synthesize this into a mandatory prerequisite task.

3. **OLD TICKET RECONCILIATION**: The 13 old tickets (GOgent-109–121) are architecturally obsolete — they augmented the Ink TUI rather than replacing it. However, specific requirements (auto-restart, real-time telemetry, chain depth visibility) should be traced into the new plan. Beethoven should flag any gaps.

4. **NDJSON COMPLETENESS**: The plan identifies 6 event types but the actual protocol may have more (especially stream_event subtypes). This is empirically verifiable and should be a Phase 0 prerequisite, not a Phase 2 discovery.

5. **POINTS OF LIKELY DISAGREEMENT**: Staff-Architect may favor the phased approach (three-process → two-process) for risk reduction. Einstein's position is that the phased approach costs ~30% more total effort and risks the TS sidecar becoming permanent. Beethoven should present both perspectives with clear decision criteria.

6. **ZUSTAND → ELM IS LOW RISK**: Both analyses should agree that the state model translation is natural and low-risk. This is not where the architectural danger lies. The danger is in process coordination and IPC, not in data model translation.

---

## Staff-Architect: Critical Review

### Executive Assessment

**Verdict:** REVISE (confidence: high)
**Summary:** The migration plan is thorough in UI component mapping (56 tickets across 10 phases) and correctly identifies the Bubble Tea Elm architecture as superior to Ink/React for terminal UIs. However, the plan's centerpiece — the three-process topology with HTTP bridge — rests on three unverified critical assumptions: (1) that createSdkMcpServer supports HTTP/SSE transport for out-of-process hosting, (2) that unidirectional bridge communication (sidecar→Go only) is sufficient, and (3) that dual-source agent state (NDJSON + bridge) can be reconciled without races. The topology should be challenged: a two-process design (Go TUI + Claude CLI with native Go MCP server) eliminates the sidecar entirely and the Go MCP SDK v1.2.0 is already in go.mod. Recommend revising the architecture decision before cutting implementation tickets.
**Issue Counts:** Critical=3, Major=5, Minor=4

### Critical Issues

**C-1: createSdkMcpServer HTTP/SSE transport unverified — blocks entire sidecar architecture** (layer: assumptions)
The plan's P3-4 ticket explicitly states 'Check the SDK docs for createStreamableHTTPServer() or equivalent' — meaning the plan author has NOT verified that the Anthropic Agent SDK's createSdkMcpServer() can be exposed as an HTTP/SSE server for out-of-process consumption by Claude Code CLI. Currently, createSdkMcpServer() is used exclusively in-process: the MCP server object is passed directly to query() as a JavaScript object reference (SessionManager.ts:403-405). The sidecar architecture requires this to work over HTTP, discovered via --mcp-config. If this transport doesn't exist or requires substantial SDK changes, the entire three-process topology collapses.
  - Evidence: Plan P3-4 line: 'The createSdkMcpServer needs to be configured for HTTP transport rather than in-process. Check the SDK docs for createStreamableHTTPServer() or equivalent.' — This is a TODO, not a verified design decision. Scout verification found zero usage of createStreamableHTTPServer or any HTTP transport configuration in the codebase. The Agent SDK changelog mentions 'Enabled SSE MCP servers on native build' but this refers to Claude Code's ability to connect to SSE servers, not to createSdkMcpServer's ability to expose one.
  - Impact: If createSdkMcpServer cannot serve over HTTP/SSE, the entire Phase 3 (6 tickets) is blocked, which cascades to Phases 4-9 (44 tickets). The plan becomes architecturally infeasible without a fundamental redesign of the MCP transport layer.
  - Recommendation: BEFORE cutting any implementation tickets: (1) Write a 50-line proof-of-concept that creates an MCP server with createSdkMcpServer(), wraps it in an HTTP/SSE transport, and verifies Claude CLI can discover it via --mcp-config. (2) If this fails, evaluate the two-process topology (see M-1). Budget: 2-4 hours for POC.

**C-2: No Go TUI → Sidecar communication channel — unidirectional bridge only** (layer: failure_modes)
The bridge API (Section 2, 'Bridge API Endpoints') defines 9 endpoints, ALL of which are sidecar→Go TUI direction (POST /modal/*, POST /agent/*, GET /health). There is NO defined mechanism for the Go TUI to communicate back to the TS sidecar. This creates critical gaps: (1) How does Go TUI tell sidecar to kill a specific spawned agent? (2) How does Go TUI trigger sidecar processRegistry.cleanupAll() during shutdown? (3) How does Go TUI request agent status from the sidecar's ProcessRegistry? The plan says 'Shutdown MCP sidecar' in P8-3 but only via SIGTERM to the Node process — this doesn't cleanly terminate spawned agents with SIGTERM→SIGKILL escalation.
  - Evidence: Bridge API table (plan lines 123-133) lists only: POST /modal/ask, /modal/confirm, /modal/input, /modal/select, POST /agent/register, /agent/update, /agent/activity, POST /toast, GET /health. All are called BY the sidecar. No reverse endpoints exist. ProcessRegistry.cleanupAll() (processRegistry.ts:116-146) requires in-process access. Shutdown handler (shutdown.ts:37-88) calls cleanupAll() directly.
  - Impact: Graceful shutdown will leave orphaned Claude CLI processes spawned by the sidecar. Agent kill operations from TUI will be impossible. Any UI action that requires sidecar-side execution (process management, cost tracking queries) cannot be implemented.
  - Recommendation: Define reverse bridge endpoints on the sidecar (Go→TS direction): (1) POST /sidecar/kill-agent {id} (2) POST /sidecar/shutdown (3) GET /sidecar/agent-status {id}. Alternatively, use a bidirectional protocol (WebSocket or Unix socket pair) instead of two unidirectional HTTP servers.

**C-3: Split-brain agent state — dual sources with no synchronization** (layer: architecture_smells)
Agent data arrives in the Go TUI from two independent, unsynchronized sources: (1) NDJSON stream parsing extracts Task() tool_use blocks from Claude CLI stdout (plan P5-2), and (2) Bridge API receives POST /agent/register from the sidecar when spawn_agent fires (plan P5-6). These create the SAME conceptual agent in the tree but via different mechanisms, different timing, and potentially different data. The plan provides no deduplication, ordering, or reconciliation strategy. A Task()-spawned agent appears first via NDJSON (when the tool_use block is emitted) and may or may not also appear via bridge (if the sidecar's spawn_agent calls bridgeAgentRegister). An MCP-spawned agent appears only via bridge (no NDJSON tool_use for MCP tools). But there's no ID coordination between the two paths.
  - Evidence: Plan P5-2: 'In Go, we extract this from the NDJSON stream directly' — handles Task() tool_use blocks. Plan P5-6: 'Handle POST /agent/register from sidecar' — handles spawn_agent notifications. In the current TS architecture, both paths write to the same Zustand store atomically (useAgentSync.ts:38 and spawnAgent.ts:235). In the Go architecture, NDJSON parsing runs in the CLI driver goroutine while bridge API runs in HTTP handler goroutines — no shared mutex or dedup logic is specified.
  - Impact: Race conditions will cause: (1) duplicate agents in tree (same agent registered twice with different IDs), (2) missing status updates (completion arrives on wrong path), (3) incorrect parent-child relationships (NDJSON path uses tool_use_id as agent ID while bridge path uses UUID). This will manifest as ghost agents, stuck 'running' status, and tree rendering glitches.
  - Recommendation: Define a single AgentRegistry with a mutex and a canonical ID strategy: (1) Task()-spawned agents use tool_use_id as canonical ID (matches current TS behavior in useAgentSync.ts:43). (2) MCP-spawned agents use the UUID from spawn_agent. (3) Add an 'agentType' + 'description' dedup key to catch double-registration. (4) All mutations go through the same AppModel.Update() via tea.Msg — never directly from goroutines.

### Major Issues

**M-1: Two-process topology not evaluated — Go MCP SDK v1.2.0 already available** (layer: cost_benefit)
The plan's primary rationale for keeping the TS sidecar is: 'createSdkMcpServer is an Anthropic primitive that tightly tracks Claude Code CLI updates. Go MCP SDKs lag behind.' (Section 1, 'Why this topology'). However, github.com/modelcontextprotocol/go-sdk v1.2.0 is already in go.mod. A two-process topology (Go TUI with native Go MCP server + Claude Code CLI) would eliminate: the sidecar process, the bridge API, both hardcoded ports, the TS bundling requirement, the Node.js runtime dependency, the startup latency, and all three critical issues above. The 7 MCP tools (askUser, confirmAction, requestInput, selectOption, spawnAgent, teamRun, testMcpPing) are each 40-160 lines of TS that could be reimplemented in Go.
  - Evidence: go.mod line 12: github.com/modelcontextprotocol/go-sdk v1.2.0. Scout verification: zero direct usage in Go codebase (dependency installed but unused). Total MCP tool code: askUser.ts (41 lines), confirmAction.ts (~40 lines), requestInput.ts (~40 lines), selectOption.ts (~40 lines), spawnAgent.ts (512 lines — but most is process management already in Go via internal/lifecycle/process.go), teamRun.ts (165 lines — trivial wrapper around gogent-team-run binary already in Go), testMcpPing.ts (~20 lines).
  - Impact: If the two-process topology is viable, the plan is over-engineered by ~12 tickets (P3-1 through P3-6, bridge-related portions of P4-4/P4-5, P5-6, sidecar startup in P0). This represents ~25% of implementation effort spent on inter-process plumbing that could be eliminated.
  - Recommendation: Add a Phase -1 spike: Implement the ask_user MCP tool in Go using go-sdk v1.2.0, verify Claude CLI discovers and calls it via --mcp-config. If successful, redesign as two-process topology. If Go SDK lacks critical features (e.g., tool() schema generation, MCP protocol compliance), document the gaps and proceed with three-process. Budget: 4-8 hours for spike.

**M-2: Multi-provider support grossly underscoped — single ticket for 4-provider system** (layer: contractor_readiness)
P7-7 ('Provider tabs and switching') is a single ticket covering: provider tab strip UI, Shift+Tab cycling, per-provider message history, per-provider session IDs, per-provider models, and per-provider project dirs. The actual TS implementation spans: providers.ts (130 lines with 4 full provider definitions including adapter paths), session.ts (291 lines with ~60% being per-provider state management), and SessionManager.ts (using providerSessionIds, providerModels, providerProjectDirs in query() calls). This is at minimum 3-4 tickets: (1) provider config types, (2) per-provider state management, (3) provider switching + message isolation, (4) provider tab UI.
  - Evidence: providers.ts defines PROVIDERS record with Anthropic (3 models), Google (2 models, adapter path), OpenAI (3 models, adapter path + env vars), Local/Ollama (3 models, adapter path) — total 11 models across 4 providers. session.ts has 5 per-provider state maps: providerMessages, providerSessionIds, providerModels, providerProjectDirs, plus 9 per-provider actions. Plan P7-7 is a single line: 'Provider tab strip (Anthropic | Google | OpenAI | Local) with Shift+Tab cycling. Per-provider message history, session ID, and model.'
  - Impact: A contractor receiving P7-7 as a single ticket would underestimate scope by 3-4x. The per-provider state management alone is more complex than several complete phases (P0, P1-3). Risk of incomplete implementation leaving multi-provider broken.
  - Recommendation: Split P7-7 into at minimum: (1) P7-7a: ProviderState struct + config types (Go translation of providers.ts + session.ts per-provider fields), (2) P7-7b: Provider switching logic + message isolation in AppModel.Update(), (3) P7-7c: Provider tab bar UI component, (4) P7-7d: Session resume with per-provider session IDs. Add acceptance criteria for each.

**M-3: Hardcoded ports 9198/9199 with no negotiation — multiple instances impossible** (layer: failure_modes)
The plan hardcodes MCP sidecar to port 9198 and bridge API to port 9199 (Section 2). No port negotiation, discovery, or fallback is specified. Running two TUI instances simultaneously (e.g., two terminal windows) will fail with EADDRINUSE. This also prevents running the TUI alongside other services that may use these ports.
  - Evidence: Plan Section 2: 'Spawns TS MCP sidecar on :9198', 'Starts Bridge API on :9199'. P3-4: 'httpServer.listen(parseInt(process.env.MCP_PORT || "9198"), "127.0.0.1")'. P3-5: 'fmt.Sprintf("MCP_PORT=%d", port)'. Environment variable fallback exists but no dynamic port allocation.
  - Impact: Users cannot run multiple TUI sessions. CI/test environments with parallel test runs will collide. Port 9198/9199 may conflict with other localhost services.
  - Recommendation: Use dynamic port allocation: bind to port 0, read the assigned port from the listener, pass it to child processes via environment variables. Alternative: use Unix domain sockets (e.g., $XDG_RUNTIME_DIR/gofortress-{pid}.sock) — this eliminates port collision entirely and is faster than TCP. The existing internal/lifecycle/process.go already uses this pattern for socket management.

**M-4: Testing ratio 7:1 — no per-phase test milestones** (layer: testing)
The plan has 49 implementation tickets (P0-P8) and 7 test tickets (P9), all concentrated in the final phase. No implementation ticket includes test requirements. Red flag: the plan's own Section 17 recommends 'Comprehensive logging of unknown event types. Fuzzy-test with real Claude sessions' but no ticket implements fuzzy testing.
  - Evidence: Plan 'Ticket Summary by Phase' table: P0(3), P1(7), P2(6), P3(6), P4(5), P5(6), P6(4), P7(7), P8(5) = 49 implementation. P9(7) = all testing. Testing is deferred to end. Compare: old tickets (GOgent-109–121) each specify 'tests_required: true' with dedicated test files.
  - Impact: Bugs accumulate across 8 phases without detection. Integration issues between phases (especially P2↔P3 bridge, P4↔P3 modal round-trip) will only surface in P9 when rework cost is highest. Contractor estimates will undercount by not including per-ticket testing.
  - Recommendation: Add test requirements to each implementation ticket. At minimum: (1) P0-3 includes smoke test verifying empty Bubble Tea app renders, (2) P2-3 includes unit test with mock NDJSON input, (3) P3-6 includes integration test for bridge round-trip, (4) P4-4 includes modal queue test with concurrent requests. Add a 'Definition of Done' section requiring tests for each ticket.

**M-5: Claude CLI stdin message format for stream-json mode unverified** (layer: assumptions)
P2-6 proposes sending user messages to Claude CLI stdin as JSON: '{"type": "user", "text": text}'. The plan itself acknowledges uncertainty: 'Verify: may need to send structured JSON or use --input-format flag. Fallback: pipe user text directly.' (Section 17, Risk Register). This is a critical data path — if the format is wrong, users cannot send messages.
  - Evidence: Plan P2-6: 'msg := map[string]string{"type": "user", "text": text}'. Risk Register item: 'stdin message format — Claude CLI stdin format for --output-format stream-json may not accept raw text'. Current spawnAgent.ts uses '-p' flag (pipe mode) which accepts raw text on stdin (line 429: cliArgs = ["-p", "--output-format", "json"]). The interactive TUI uses query() from Agent SDK which handles stdin internally. No codebase evidence of JSON-formatted stdin to Claude CLI.
  - Impact: If the stdin format is wrong, Phase 2 (CLI driver) is blocked. The entire Go TUI cannot send user messages. This is on the critical path — everything after P2 depends on working bidirectional CLI communication.
  - Recommendation: Verify before implementation: (1) Run 'claude --output-format stream-json --verbose' interactively and observe stdin handling. (2) Check Claude Code CLI source/docs for --input-format flag. (3) Test: echo '{"type":"user","text":"hello"}' | claude --output-format stream-json. Add results to plan as verified assumption. Budget: 1 hour.

### Minor Issues

**m-1: Old ticket reconciliation incomplete — 33 new tickets have no coverage** (layer: dependencies)
13 old tickets (GOgent-109 to GOgent-121) cover primarily agent tree, CLI subprocess, layout, and session management. The new plan's 56 tickets include 33 tickets with no old-ticket equivalent: P0 scaffolding (3), P3 sidecar+bridge (6), P4 modals (5), P6-3/P6-4 team UI (2), P7-1 through P7-7 settings/tabs (7), P8-3 through P8-5 lifecycle (3), P9 testing (7). Reconciliation guidance is missing — should old tickets be superseded, merged, or kept as sub-tickets?
  - Evidence: Old ticket mapping: GOgent-110→P2-2 (strong), GOgent-114→P2-1 (strong), GOgent-115→P5-1 (strong), GOgent-116→P5-4 (strong), GOgent-117→P5-5 (strong), GOgent-118→P1-5+P2 (partial), GOgent-119→P1-5 (strong), GOgent-120→P1-3 (strong), GOgent-121→P8-1/P8-2 (moderate). GOgent-109, GOgent-111, GOgent-112, GOgent-113 have weaker mappings to new plan.
  - Impact: Without reconciliation, parallel ticket tracking systems create confusion. Developers may reference either old or new ticket IDs. Priority/dependency conflicts between old (week-based) and new (phase-based) ordering.
  - Recommendation: Add a reconciliation appendix: (1) Map each old ticket to its new equivalent(s) with status (superseded/merged/kept). (2) Close old tickets that are fully covered by new plan. (3) Keep old tickets that cover areas the new plan omits (GOgent-113 file watchers differ from new plan's polling approach).

**m-2: Graceful shutdown ordering underspecified for three-process topology** (layer: failure_modes)
P8-3 lists shutdown steps (save session, interrupt CLI, shutdown CLI, shutdown sidecar, wait for hooks, exit) but doesn't specify: (1) timeout per step, (2) what happens if CLI is mid-response when shutdown starts, (3) how sidecar cleans up its own spawned agents before exiting, (4) whether background team agents (detached processes from teamRun) should be waited for or orphaned.
  - Evidence: Plan P8-3 lists 6 steps without timeouts. Current shutdown.ts has a 500ms wait for Go hooks (line 79) and delegates to registered handlers (line 63). ProcessRegistry.cleanupAll() uses gracePeriod (5s) and forceKillDelay (1s) from processRegistry.ts:30-31. These timing constants are not carried into the plan.
  - Impact: Shutdown may hang indefinitely waiting for stuck processes, or exit too quickly leaving orphaned processes. Background team agents may be killed mid-execution.
  - Recommendation: Add timing constraints: (1) CLI interrupt: 2s timeout before SIGKILL. (2) Sidecar shutdown: 5s for graceful agent cleanup. (3) Total shutdown budget: 10s max. (4) Background team agents: list as 'running' in status but don't wait (they're detached by design).

**m-3: esbuild single-file bundling not validated with full sidecar dependency tree** (layer: assumptions)
Section 17 recommends 'Bundle the TS sidecar as a single file. Use the existing esbuild config to produce one .js file.' The sidecar would include: @anthropic-ai/claude-agent-sdk (which imports node:crypto, node:child_process, etc.), zod, nanoid, async-mutex, and all spawn/validation/context injection code. Native Node.js modules (child_process, fs, crypto) cannot be bundled — esbuild handles them as externals, but the resulting bundle still requires a Node.js runtime.
  - Evidence: Plan Section 18 lists sidecar dependencies. spawnAgent.ts imports: child_process (spawn), crypto (randomUUID), fs/promises (readFile). contextInjector.ts imports: fs/promises, path, os. These are Node.js built-ins that esbuild marks as external by default.
  - Impact: The bundled sidecar still requires Node.js installed. The plan's goal of 'Go binary is the primary artifact' is partially defeated. Users still need Node.js runtime. go:embed of the .js bundle works for distribution but doesn't eliminate the runtime dependency.
  - Recommendation: Acknowledge Node.js runtime requirement explicitly in the plan. Consider bun as alternative runtime (faster startup, single binary distribution). Add a P0 spike ticket: 'Validate esbuild bundle of sidecar with all dependencies, measure bundle size and startup time.'

**m-4: No per-phase validation milestones or demo checkpoints** (layer: contractor_readiness)
The plan jumps from implementation tickets directly to Phase 9 (integration testing). No intermediate milestones define what 'Phase N complete' looks like or how to demo progress. This makes it hard to detect drift, validate partial implementations, or demonstrate progress to stakeholders.
  - Evidence: Each phase has a 'Goal' statement but no 'Done When' criteria. Phase 1 goal: 'Multi-panel layout with focus cycling, responsive sizing, and themed borders. No data — just the chrome.' — but no acceptance test or screenshot comparison is required.
  - Impact: Phases may be marked 'complete' with subtle bugs or missing features. Cascading assumptions between phases go unvalidated until end-to-end testing in P9.
  - Recommendation: Add a validation milestone to each phase: P0 'make run shows bordered text', P1 'screenshot matches Layout.tsx at 120x40 terminal', P2 'mock CLI script produces correct messages', P3 'sidecar health check passes within 3s', P4 'modal round-trip completes in <100ms', etc.

### Commendations

- Comprehensive state translation mapping (Section 15): Every Zustand slice field is mapped to a specific Go struct location with explicit notes. This is unusually thorough and will save significant implementation time.
- Correct architecture pattern for Bubble Tea: Value receivers, centralized message types in messages.go, tea.Batch for Init(), channel-based modal resolution — all follow established Bubble Tea best practices and avoid common pitfalls.
- Complete keybinding inventory (Section 16): Every keyboard shortcut is documented with current source, action, and Go binding name. This prevents the common 'forgot a keybinding' regression.
- Accurate identification of the modal system as the critical UX path: P4 correctly identifies the canUseTool permission flow as the most complex modal interaction and documents all 5 variants (EnterPlanMode, ExitPlanMode, AskUserQuestion, standard tool, acceptEdits).
- Strong design recommendations (Section 17): The 9 design recommendations are practical, experience-derived, and correct — especially #6 (auto-scroll with manual override), #5 (tea.Program.Send() concurrency safety), and #9 (gate CLI on sidecar readiness).
- Honest risk register: The plan openly acknowledges its own uncertainties (stdin format, NDJSON edge cases, sidecar latency) rather than hand-waving them. This intellectual honesty makes the plan trustworthy.

### Failure Mode Analysis

**TS MCP sidecar crashes during active Claude CLI session with pending modal (permission prompt)** (probability: medium, impact: high)
  - Detection: Bridge API POST /modal/* times out after 60s. Claude CLI tool_use hangs waiting for MCP response. User sees frozen TUI.
  - Mitigation: Plan mentions 'attempt restart' but doesn't specify: (1) re-queue pending modal request after sidecar restart, (2) timeout and auto-deny if sidecar doesn't recover within 10s, (3) surface error to user in status line. Recommend: detect sidecar death via health check failure, show 'MCP sidecar restarting...' toast, auto-deny pending permissions with error message.

**Claude CLI emits unknown NDJSON event type not covered by plan's struct definitions** (probability: medium, impact: low)
  - Detection: json.Unmarshal succeeds on CLIEvent discriminator but type switch falls through to default case. Event silently dropped.
  - Mitigation: Plan Section 17 correctly recommends logging unknown types. Implementation should: (1) log unknown event type + raw JSON at WARN level, (2) continue processing (don't crash), (3) accumulate unknown types in telemetry for discovery.

**Go TUI and sidecar start on ports 9198/9199 but another TUI instance is already running** (probability: high, impact: medium)
  - Detection: net.Listen returns EADDRINUSE. Sidecar process exits with error. Go TUI logs error but may continue without MCP.
  - Mitigation: Use dynamic port allocation (port 0) or Unix domain sockets. Pass allocated port via environment variable to child processes. This is a common and solved problem — see internal/lifecycle/process.go socket pattern.

**Modal permission prompt adds >500ms latency due to HTTP round-trips (CLI→sidecar→bridge→TUI→user→TUI→bridge→sidecar→CLI)** (probability: medium, impact: medium)
  - Detection: Users perceive sluggish permission prompts. Each tool_use requiring permission adds 2 HTTP round-trips vs 0 in current in-process architecture.
  - Mitigation: Measure round-trip latency in P3 integration test. If >200ms, consider: (1) Unix socket instead of TCP (eliminates TCP overhead), (2) persistent HTTP/2 connections to avoid connection setup per request, (3) batch sequential modal calls into single bridge request for multi-step flows (canUseTool).

**Graceful shutdown with 3 processes: Go TUI sends SIGTERM to sidecar, but sidecar has 5 spawned agents still running** (probability: high, impact: high)
  - Detection: Orphaned Claude CLI processes left running after TUI exit. Visible via `ps aux | grep claude`.
  - Mitigation: Sidecar must handle SIGTERM by calling processRegistry.cleanupAll() (SIGTERM→SIGKILL escalation for all spawned agents) before exiting. Go TUI should wait up to 10s for sidecar to confirm cleanup. Add POST /sidecar/shutdown endpoint that triggers cleanup and returns when complete.

**Plan Phase 3 implementation discovers createSdkMcpServer cannot be exposed via HTTP — fundamental architecture block** (probability: medium, impact: high)
  - Detection: P3-4 implementation fails to create working HTTP MCP server. Claude CLI cannot discover sidecar tools.
  - Mitigation: This is why C-1 is critical: validate BEFORE starting implementation. If HTTP transport is impossible, pivot to: (1) two-process topology with Go MCP SDK, or (2) stdio transport where sidecar communicates with Claude CLI via stdin/stdout pipes (but this conflicts with Bubble Tea's stdin ownership — would require Go TUI to proxy stdin).

### Recommendations

**[HIGH]** Conduct MCP transport proof-of-concept before any implementation work (addresses C-1)
  - Rationale: The entire three-process topology depends on createSdkMcpServer supporting HTTP/SSE transport. If this fails, 44 tickets need redesign. A 50-line POC costs 2-4 hours vs. weeks of wasted implementation.
  - Effort: 2-4 hours for POC, 4-8 hours if Go MCP SDK spike also needed (M-1)

**[HIGH]** Define bidirectional bridge API with Go→sidecar endpoints (addresses C-2)
  - Rationale: Current plan only has sidecar→Go communication. Process kill, shutdown cleanup, and status queries all require reverse communication. Add at minimum: POST /sidecar/kill-agent, POST /sidecar/shutdown, GET /sidecar/agents.
  - Effort: 2 hours to design API contract, 4 hours to implement

**[HIGH]** Design canonical AgentRegistry with dedup and mutex for dual-source agent state (addresses C-3)
  - Rationale: NDJSON and bridge API both feed agent data into the same tree. Without dedup, races produce ghost agents and incorrect status. All mutations must go through tea.Msg → Update() pipeline.
  - Effort: 4 hours to design, implementation folded into P5-1

**[HIGH]** Verify Claude CLI stdin format for stream-json mode (addresses M-5)
  - Rationale: Critical path dependency. 1 hour of manual testing prevents potential Phase 2 block.
  - Effort: 1 hour

**[HIGH]** Split P7-7 into 3-4 sub-tickets for multi-provider support (addresses M-2)
  - Rationale: Current single ticket covers 4 providers, 11 models, per-provider state management (~180 lines of state code), and provider switching UI. Grossly underscoped for contractor readiness.
  - Effort: 1 hour to decompose tickets

**[MEDIUM]** Evaluate two-process topology via Go MCP SDK spike (addresses M-1)
  - Rationale: github.com/modelcontextprotocol/go-sdk v1.2.0 is already in go.mod but unused. Reimplementing 7 MCP tools in Go eliminates sidecar, bridge, 2 ports, and Node.js dependency. The total MCP tool code (excluding spawnAgent process management which already exists in Go) is ~300 lines of TS.
  - Effort: 4-8 hours for spike

**[MEDIUM]** Use Unix domain sockets instead of TCP for IPC (addresses M-3)
  - Rationale: Eliminates port collision entirely. Lower latency than TCP. Existing internal/lifecycle/process.go already uses the gofortress-{pid}.sock pattern. Consistent with existing codebase conventions.
  - Effort: Minimal if done during initial P3 implementation

**[MEDIUM]** Add per-phase test requirements and validation milestones (addresses M-4, m-4)
  - Rationale: Current 7:1 implementation:test ratio with all testing deferred to P9 is a red flag. Each phase should have a 'Done When' criteria with at minimum a smoke test.
  - Effort: 2 hours to add test requirements to each phase's tickets

**[LOW]** Create ticket reconciliation appendix mapping old GOgent-109–121 to new plan tickets (addresses m-1)
  - Rationale: Prevents confusion from parallel ticket numbering systems. Low effort, high clarity.
  - Effort: 1 hour

**[LOW]** Specify shutdown timing constraints for three-process topology (addresses m-2)
  - Rationale: Current plan lists shutdown steps without timeouts. Add: CLI interrupt 2s, sidecar cleanup 5s, total budget 10s.
  - Effort: 30 minutes to add to P8-3 ticket

**[LOW]** Validate esbuild bundle of sidecar dependencies (addresses m-3)
  - Rationale: Confirm single-file bundling works with native Node.js module externals. Acknowledge Node.js runtime requirement in plan.
  - Effort: 1 hour

### Sign-Off Conditions

- C-1 RESOLVED: MCP HTTP/SSE transport POC demonstrates working Claude CLI ↔ sidecar communication, OR architecture pivots to two-process topology with Go MCP SDK
- C-2 RESOLVED: Bidirectional bridge API designed with Go→sidecar endpoints for kill, shutdown, and status query
- C-3 RESOLVED: AgentRegistry design specifies canonical ID strategy, dedup mechanism, and mutex for concurrent NDJSON + bridge API writes
- M-5 RESOLVED: Claude CLI stdin format verified with manual test, results documented in plan

### Handoff Notes

Beethoven should focus on synthesizing the topology decision: Einstein's theoretical analysis likely explores the three-process vs two-process tradeoff from first principles, while this review provides empirical evidence from the codebase (Go MCP SDK present but unused, createSdkMcpServer in-process only, total MCP tool code ~300 lines excluding process management already in Go). The key tension is: the plan's sidecar rationale ('Go MCP SDKs lag behind') vs the empirical fact that go-sdk v1.2.0 exists and the tools are small. The MCP transport POC (C-1) is the gate that resolves this tension — if it passes, three-process is viable; if it fails, two-process is forced. Beethoven should present both paths with clear decision criteria rather than picking a side prematurely. Also note: the plan's UI/component mapping (Sections 3, 14, 15, 16) is excellent and should be preserved regardless of topology decision — these are topology-independent.

---

**End of Pre-Synthesis Document**
