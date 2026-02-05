# Braintrust Analysis: Agent Monitoring Architecture

> **Problem:** Spawned agents don't appear in TUI monitoring panels despite processes running
> **Generated:** 2026-02-05
> **Analysts:** Einstein (Theoretical), Staff-Architect (Practical)
> **Synthesizer:** Beethoven

---

## Executive Summary

The TUI's agent monitoring failure stems from a **missing store registration call**—`spawnAgent.ts` creates CLI processes and registers them with `ProcessRegistry`, but never calls `store.addAgent()` to create the Zustand entity that drives UI rendering. Both analysts independently identified this gap: Einstein traced it to a "violated single-source-of-truth principle," while Staff-Architect mapped the exact code locations (`spawnAgent.ts:81-83` gets the store but never uses it for creation). The fix is straightforward: expand `storeAdapter.ts` to expose `addAgent()`/`updateAgent()`, then wire the spawn lifecycle events to store updates. Both analysts recommend **store-first** ordering (create entity BEFORE spawning process) to guarantee UI consistency. Implementation risk is low—this is additive code with no backward compatibility concerns since the feature is currently non-functional.

---

## Root Cause

### Unified Understanding

**Technical Gap:** `spawnAgent.ts` performs three operations, but only two are visible:

| Operation | Executed | Visible to UI |
|-----------|----------|---------------|
| Generate agent ID | ✓ | ✗ |
| Register process with ProcessRegistry | ✓ | ✗ |
| Validate parent-child relationship via storeAdapter | ✓ | ✗ |
| **Create agent entity in Zustand** | **✗** | **N/A** |

**Why It Happened:** The `storeAdapter.ts` module was designed *specifically* for relationship validation (checking if parents exist, recording parent-child links). Its interface exposes `get()`, `addChild()`, `removeChild()`—but not `addAgent()`. This was intentional minimalism for the validation use case, but it created an **accidental architectural isolation** where the spawn layer and UI layer never connect.

**The Tell:** `ProcessRegistry` emits `registered` and `unregistered` events (lines 37-59 in `processRegistry.ts`), but nothing subscribes to them. This infrastructure exists but is unused—suggesting the integration was planned but never completed.

---

## Recommended Architecture

### Synthesis: Store-First Spawn with Lifecycle Events

Both analysts converge on **store-first** ordering and **event-driven updates**. The synthesized recommendation:

```
┌─────────────────────────────────────────────────────────────────┐
│                     SPAWN LIFECYCLE                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. PRE-SPAWN: store.addAgent({ status: "spawning", ... })     │
│     └── UI immediately shows agent with "spawning" indicator    │
│                                                                 │
│  2. SPAWN: proc = spawn("claude", ...)                          │
│            registry.register(agentId, proc)                     │
│     └── Process running, store entity already exists            │
│                                                                 │
│  3. LIFECYCLE: Wire process events to store                     │
│     proc.stdout.on("data")  → updateAgent(streamBuffer)         │
│     proc.on("close", 0)     → updateAgent(status: "complete")   │
│     proc.on("close", !0)    → updateAgent(status: "error")      │
│     timeout fires           → updateAgent(status: "timeout")    │
│                                                                 │
│  4. RENDER: AgentTree/AgentDetail read from Zustand             │
│     └── Automatic reactivity via useAgentStore()                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Why Store-First Over Process-First

| Dimension | Store-First | Process-First (current) |
|-----------|-------------|------------------------|
| **UI Consistency** | Agent appears immediately | Agent invisible until process emits |
| **Spawn Failure Handling** | Shows error state naturally | No record of failed spawn attempt |
| **Debugging** | Store is SSOT, easy to inspect | Must correlate ProcessRegistry + store |
| **Race Conditions** | Eliminated by ordering | Possible gap between spawn and registration |

**HIGH CONFIDENCE:** Both analysts agree on store-first ordering.

---

## Implementation Plan

### Phase 1: Bridge the Gap (Immediate Fix)
**Confidence: HIGH** (Both analysts agree)

**Objective:** Get spawned agents appearing in TUI with basic status tracking.

| Step | File | Change | Risk |
|------|------|--------|------|
| 1.1 | `storeAdapter.ts` | Add `addAgent()` and `updateAgent()` to interface | None |
| 1.2 | `relationshipValidation.ts` | Extend `AgentsStore` interface type | None |
| 1.3 | `spawnAgent.ts:~166` | Call `store.addAgent()` BEFORE `spawn()` call | Low |
| 1.4 | `spawnAgent.ts:~200` | Wire `proc.on("close")` to `store.updateAgent()` | Low |

**Expected Outcome:** Agents appear in AgentTree immediately on spawn, with correct final status.

**NOT included in Phase 1:** Streaming buffer updates (see Phase 2).

---

### Phase 2: Streaming Observability (Enhanced)
**Confidence: MEDIUM** (Einstein recommends; Staff-Architect notes throttling concern)

**Objective:** Real-time activity stream in AgentDetail panel.

| Step | File | Change | Risk |
|------|------|--------|------|
| 2.1 | `spawnAgent.ts` | Wire `proc.stdout.on("data")` to buffer updates | Medium |
| 2.2 | `spawnAgent.ts` | Implement 100ms throttle on buffer updates | None |
| 2.3 | `agents.ts` | Add `appendToStreamBuffer()` action | None |

**Throttling Rationale (Staff-Architect concern):** High-frequency stdout can trigger 100+ updates/sec. Cap at 10 updates/sec via debounce. Use `appendToStreamBuffer()` that batches chunks between throttle windows.

**REQUIRES JUDGMENT:** Exact throttle interval (100ms vs 200ms) depends on UX testing.

---

### Phase 3: Lifecycle Hardening (Production-Ready)
**Confidence: MEDIUM** (Staff-Architect identified risks; Einstein's architecture supports)

**Objective:** Handle edge cases robustly.

| Step | Issue | Solution |
|------|-------|----------|
| 3.1 | Memory leak: orphaned agents | Wire `registry.on("unregistered")` → `updateAgent(status: "complete")` as fallback |
| 3.2 | Stale spawning status | Add 30s timeout watchdog that transitions stuck "spawning" to "error" |
| 3.3 | Persistence bloat | Exclude agents with `spawnMethod: "mcp-cli"` from `partialize` config |

**Persistence Decision (Staff-Architect recommendation):** Do NOT persist MCP-spawned agents initially. They are ephemeral by nature, and persistence creates unbounded growth. If needed later, add TTL-based cleanup.

---

## Risk Mitigation

### Unified Risk Assessment

| Risk | Likelihood | Impact | Mitigation | Owner |
|------|------------|--------|------------|-------|
| **Race condition: spawn vs store** | Medium | Medium | Store-first ordering (Phase 1.3) | Implementation |
| **Memory leak: orphaned agents** | High | Medium | Fallback cleanup via registry events (Phase 3.1) | Implementation |
| **High-frequency UI thrashing** | Medium | Low | Throttle buffer updates (Phase 2.2) | Implementation |
| **Persistence bloat** | Medium | Medium | Exclude from partialize (Phase 3.3) | Implementation |
| **Status stuck at "spawning"** | Low | Medium | Timeout watchdog (Phase 3.2) | Implementation |

### What Won't Break

- **Existing spawn validation:** Relationship validation is orthogonal to entity creation
- **ProcessRegistry functionality:** New store calls are additive
- **TUI rendering:** Zustand reactivity is unchanged, just more data flowing through

---

## Tension Resolution

### Divergence: Unified Spawn Manager vs Minimal Fix

**Einstein (Option C):** Recommends new `SpawnManager` class that encapsulates both ProcessRegistry and Zustand updates. "Best long-term architecture. Pays for itself quickly in maintainability."

**Staff-Architect:** Recommends expanding existing `storeAdapter.ts` without new abstraction. "No backward compatibility concerns (feature currently broken)."

**Resolution:** **Adopt Staff-Architect's approach for Phase 1, with Einstein's architecture informing Phase 2-3.**

**Justification:**
- The feature is currently **completely broken**—any fix improves the situation
- Minimal fix lands value faster with lower risk
- `storeAdapter.ts` is already the designated bridge module
- If Phase 2-3 reveals complexity, refactor to `SpawnManager` then (YAGNI)

**Confidence: MEDIUM** — Reasonable engineers could disagree. Revisit after Phase 1 ships.

---

### Agreement: Store-First Ordering

**HIGH CONFIDENCE:** Both analysts independently arrived at store-first ordering with the same rationale (UI consistency, spawn failure visibility, SSOT). No resolution needed.

---

### Agreement: No Persistence for Spawned Agents

**HIGH CONFIDENCE:** Both analysts agree spawned agents should NOT persist across TUI restarts:
- Einstein: "Process death = data loss" is acceptable when Zustand is SSOT
- Staff-Architect: "Do NOT persist spawned agents initially (exclude `spawnMethod: 'mcp-cli'` from partialize)"

---

## Open Questions

The following require user decision before implementation:

1. **Status Granularity:** Should we distinguish `running` from `streaming`, or collapse to simpler states (`spawning`, `active`, `done`, `error`)?
   - Einstein suggests explicit 7-state FSM
   - Staff-Architect implicitly suggests simpler model
   - **Recommendation:** Start with 4 states, expand if UX demands it

2. **Cost Display:** When should cost information appear in AgentDetail?
   - On completion only (current behavior in cost tracker)
   - Estimated during streaming (requires token counting)
   - **Recommendation:** Completion only for Phase 1

3. **AgentTree Sorting:** How should multiple spawned agents be ordered?
   - By spawn time (newest first or oldest first)
   - By status (running first, then completed)
   - **Recommendation:** Spawn time, oldest first (chronological reading order)

---

## Appendix A: Einstein's Full Analysis

Full theoretical analysis available at:
`/tmp/claude-1000/-home-doktersmol-Documents-GOgent-Fortress-packages-tui/f536fdda-ff17-4c44-8c31-20e22bcdee01/scratchpad/einstein-analysis-2026-02-05-agent-monitoring.md`

**Key Contributions:**
- Root cause framing as "violated SSOT principle"
- Comparative analysis of 4 implementation options (A-D)
- Bubbletea pattern reference as architectural model
- Data flow diagrams and state transition tables

---

## Appendix B: Staff-Architect's Practical Review

**Key Contributions:**
- Precise code location mapping (line numbers for all integration points)
- Risk matrix with likelihood/impact assessment
- Explicit "do NOT persist" recommendation
- Phased rollout approach without feature flags

**Critical Implementation Points Identified:**
1. `storeAdapter.ts` interface expansion
2. `spawnAgent.ts:~166` for pre-spawn store call
3. `spawnAgent.ts` close/error handlers for status transitions
4. `partialize` exclusion for MCP-spawned agents

---

## Metadata

```yaml
braintrust_analysis_id: ba-2026-02-05-agent-monitoring
problem_brief_id: pb-2026-02-05-agent-monitoring
einstein_analysis_timestamp: 2026-02-05T21:45:00Z
synthesis_timestamp: 2026-02-05T22:15:00Z
confidence_summary:
  high: 3  # Store-first, no persistence, root cause
  medium: 2  # SpawnManager deferral, streaming throttle
  requires_judgment: 3  # Status granularity, cost display, sorting
implementation_phases: 3
estimated_phase1_effort: 20-30 lines changed
breaking_changes: none
```

---

*Braintrust synthesis complete. Ready for implementation.*
