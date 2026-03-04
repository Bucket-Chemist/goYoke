---
tags:
  - braintrust
  - staff-architect
  - vault-architecture
  - obsidian-cli
date: 2026-03-04
agent: staff-architect-critical-review
model: opus
cost_usd: 0.77
status: complete
verdict: APPROVE_WITH_CONDITIONS
---

# Staff-Architect: Practical Review — EM-Deconvoluter Vault + Obsidian CLI

> **Braintrust team:** braintrust-1772589462
> **Cost:** $0.77 | **Model:** Opus | **Verdict:** APPROVE_WITH_CONDITIONS

---

## Executive Assessment

**Confidence:** High | **Critical:** 2 | **Major:** 3 | **Minor:** 3

The vault design principles analysis (Part A) is sound and the patterns are genuinely generalizable. However, the Obsidian CLI integration proposal (Part B) should be **definitively rejected**: a 22.8% silent failure rate, 3-7x latency budget overrun, desktop process dependency, and direct architectural contradiction with SCOPE-v3's vault-as-projection model make CLI adoption a negative-value investment.

The correct path — direct file I/O via goldmark + frontmatter — is already proven and should be codified as the committed approach.

---

## Critical Issues

### C-1: CLI Integration Architecturally Contradicts SCOPE-v3 Direction
**Severity:** Critical | **Layer:** Architecture Smells

SCOPE-v3 commits to a three-layer architecture where the vault is a human-readable PROJECTION of a SQLite+bbolt temporal knowledge graph. Investing in CLI as an agent interface means investing in the **wrong abstraction boundary**. When SCOPE-v3 ships, CLI integration work becomes dead code.

**Recommendation:** Reject CLI entirely. Design VaultReader/VaultWriter interfaces that map cleanly to the future graph API.

### C-2: 22.8% Silent Failure Rate is Categorically Blocking
**Severity:** Critical | **Layer:** Failure Modes

Silent failures mean no error signal — the agent proceeds with corrupted state. With 28 tickets across 3 sprints, statistical expectation is ~6 silent failures per full sprint processing pass. Direct file I/O has effectively 0% silent failure rate.

**Recommendation:** No amount of retry logic makes 22.8% silent failures acceptable for state-mutating operations.

---

## Major Issues

### M-1: Obsidian Desktop Process Dependency
**Severity:** Major | **Layer:** Dependencies

Every vault-touching operation must handle two code paths: CLI-available and CLI-unavailable. This doubles test surface area. The fallback path IS the direct file I/O approach — meaning you build and maintain both systems.

**Recommendation:** One code path (direct file I/O) that works unconditionally. CLI-specific features as optional fire-and-forget notifications only.

### M-2: Race Condition Between Agent and Human Edits
**Severity:** Major | **Layer:** Failure Modes

The dual-index pattern means a single logical operation (update ticket status) requires two file writes. If either is interrupted or overwritten, the index drifts.

**Recommendation:** Atomic file writes (temp + rename), advisory locking (flock) on JSON index. Accept narrow race window (<1ms) for .md files.

### M-3: CLI Provides Zero Capabilities Beyond Direct File I/O
**Severity:** Major | **Layer:** Cost-Benefit

The CLI's 100+ commands (create notes, update frontmatter, search, manage links, tag operations) — every one is achievable with direct file I/O. The /ticket skill already implements all of these at sub-millisecond latency.

**Recommendation:** Codify this finding as an ADR so the question doesn't resurface.

---

## Minor Issues

### m-1: Dual-Index Drift
The dual-index pattern (JSON + MD) is technical debt. Build a generation script that derives tickets-index.json and 00-Board.md from .md frontmatter. **Effort:** 3-4 hours.

### m-2: Vault Design Principles Should Be Independent Deliverable
If bundled with CLI rejection, the valuable pattern extraction may be lost. Extract as standalone reference document. **Effort:** 2 hours.

### m-3: Cold-Start Latency Not Measured
200-500ms may not account for IPC connection establishment. Cold-start could be 1-2 seconds.

---

## Commendations

1. **Agent-compatible vault patterns** — machine-readable JSON indices alongside markdown, YAML frontmatter contracts, template standardization. This is a transferable reference architecture.
2. **Evidence-based CLI risk quantification** — 22.8% failure rate from actual testing, not hand-waving.
3. **Ticket-Compliance-Guide.md** — exemplary interface specification defining exact contract between /ticket skill and vault.
4. **Dual-index design** — right choice for current stage, avoids database complexity while giving agents O(1) lookup.
5. **spec-vault-mapping.md** — explicitly establishes vault authority, prevents external systems dictating internal structure.

---

## Failure Mode Analysis

| Scenario | Probability | Impact | Mitigation |
|---|---|---|---|
| CLI: Obsidian not running when agent executes | High | High | Don't adopt CLI — single code path always works |
| CLI: Silent failure corrupts ticket status | High | High | Don't adopt CLI |
| File I/O: Human edits while agent updates JSON | Low | Medium | Atomic writes + flock |
| File I/O: Vault exceeds 1000+ tickets | Low | Low | SCOPE-v3 should be in place by then |
| SCOPE-v3 delayed: dual-index drift accumulates | Medium | Medium | Generation script eliminates drift |

---

## Prioritized Recommendations

### High Priority
1. **Reject CLI, codify as ADR** — 1 hour. Reference this review as evidence.
2. **Define VaultReader/VaultWriter interfaces** — 2-3 hours. Abstracts /ticket skill file I/O for SCOPE-v3 migration.

### Medium Priority
3. **Generation script** — 3-4 hours. Derives JSON + Board from .md frontmatter. Eliminates manual sync.
4. **Atomic file writes** — 1 hour. Write-to-temp + rename for all agent vault operations.
5. **Extract vault design principles** — 2 hours. Standalone reference doc for reuse.

### Low Priority
6. **Advisory locking on tickets-index.json** — 30 minutes. Defense-in-depth.
7. **Wikilink validation** — 1 hour. Verify [[links]] resolve to actual files.

---

## Assumption Register

| ID | Assumption | Verified | Risk if False |
|---|---|---|---|
| A-1 | 22.8% failure rate measured accurately | No | Even 5% is unacceptable for state-mutating ops |
| A-2 | 200-500ms is per-call, not amortizable | No | Batching doesn't help — typical ops are single-field |
| A-3 | SCOPE-v3 is committed direction | No | If abandoned, vault becomes permanent → invest more in file I/O |
| A-4 | Hook latency budget ~72ms is hard constraint | Yes | CLI exceeds budget by 2.8-7x |
| A-5 | Direct file I/O is proven baseline | Yes | /ticket skill + 28 tickets + 6 ADRs all work |
| A-6 | Simultaneous human+agent access required | Yes | Vault's value depends on dual access |

---

## Sign-Off Conditions

1. CLI rejection codified as ADR before further evaluation cycles
2. Direct file I/O documented as committed interim approach
3. Vault design principles extracted as standalone deliverable

### Post-Approval Monitoring
- Watch for dual-index drift as Sprint 2 tickets are created
- Monitor for requests to re-evaluate CLI — point to ADR. Re-evaluate only if: <1% failure rate AND <10ms latency AND headless support
- Track SCOPE-v3 timeline — if delayed >6 months, invest more in file I/O layer
