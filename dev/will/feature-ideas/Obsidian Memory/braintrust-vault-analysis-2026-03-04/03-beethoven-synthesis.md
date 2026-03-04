---
tags:
  - braintrust
  - beethoven
  - synthesis
  - vault-architecture
  - obsidian-cli
date: 2026-03-04
agent: beethoven
model: opus
cost_usd: 0.64
status: complete
total_team_cost_usd: 2.67
---

# Braintrust Synthesis: EM-Deconvoluter Vault Architecture + Obsidian CLI Integration

> **Braintrust team:** braintrust-1772589462
> **Total cost:** $2.67 (Einstein: $1.26, Staff-Architect: $0.77, Beethoven: $0.64)
> **Duration:** ~9 minutes | **Budget remaining:** $47.33 / $50.00
> **Convergences:** 6 | **Divergences resolved:** 4 | **Open questions:** 5

---

## Executive Summary

Both analysts converge on a **definitive CLI rejection** (22.8% silent failure rate, 200-500ms latency, desktop dependency, and SCOPE-v3 contradiction each independently disqualify it) and agree that **direct file I/O via goldmark + frontmatter** is the correct interim agent interface.

The vault embodies seven transferable design principles — most critically the **Dual-Interface Pattern** (structured frontmatter + semantic markdown body) and the **Index-as-API** pattern (machine-readable JSON alongside human-readable markdown).

The central tension is the future **source-of-truth topology**: Einstein argues the vault is a canonical serialization format that should remain authoritative forever (graph as derived index), while Staff-Architect endorses SCOPE-v3's inversion where the graph becomes authoritative and the vault becomes projection.

**Resolution:** Separate **authoring authority** (vault, always) from **query authority** (graph, when available) — a dual-authority model where edits flow vault→graph and queries flow graph→caller.

---

## Convergence Points (High Confidence)

### 1. Obsidian CLI Rejection
- **Einstein:** Categorically unsuitable — IPC overhead is inherent, not fixable. 200-500ms and 22.8% failure are symptoms of a fundamental mismatch.
- **Staff-Architect:** Fails all four evaluation criteria. Any single criterion sufficient to reject; all four make it categorical.
- **Synthesis:** Overdetermined rejection. Codify as ADR. Re-evaluate only if CLI achieves <1% failure rate and <10ms latency. Narrow legitimate role: optional fire-and-forget notifications (graph view snapshots), never on critical data path.

### 2. Direct File I/O as Correct Interim Interface
- **Einstein:** <1ms latency, 100% reliability, no external dependencies, works headless. /ticket skill already proves the pattern.
- **Staff-Architect:** Already proven, 0% silent failure rate. Should be codified as committed approach.
- **Synthesis:** Abstract behind VaultReader/VaultWriter interfaces now so SCOPE-v3 swap is mechanical, not architectural.

### 3. YAML Frontmatter as Human-Machine Contract
- **Einstein:** Pareto-optimal encoding for the human-readability vs. machine-parseability tradeoff.
- **Staff-Architect:** Ticket-Compliance-Guide.md cited as exemplary interface specification.
- **Synthesis:** Validated as correct. Monitor as artifact types expand. Structured code blocks in markdown body as escape hatch for nested structures.

### 4. Vault Design Principles are Transferable
- **Einstein:** Seven named principles identified (Dual-Interface, Template-as-Schema, Vault Authority, Lifecycle-Aware, Cross-Reference, Index-as-API, Dependency-as-DAG).
- **Staff-Architect:** Should be extracted as standalone reference document.
- **Synthesis:** Extract as standalone doc with Einstein's naming and Staff-Architect's emphasis on actionable patterns.

### 5. Dual-Index Pattern is Correct with Known Debt
- **Einstein:** Intentional Interface Segregation, not redundancy.
- **Staff-Architect:** Right design, but manual sync is technical debt. Generation script needed.
- **Synthesis:** Make .md files canonical, derive all indices automatically. Eliminates drift, validates vault-authoritative principle.

### 6. Concurrent Access Requires Atomic Operations
- **Einstein:** Operations must be atomic at file level and idempotent.
- **Staff-Architect:** Temp + POSIX rename + flock on JSON indices.
- **Synthesis:** Implement now. Race window <1ms is acceptable. SCOPE-v3 solves by design with transactional graph semantics.

---

## Divergences Resolved

### D-1: Source-of-Truth Topology
| Einstein | Staff-Architect |
|---|---|
| Vault is canonical serialization format, forever authoritative. Graph is derived materialized view. | SCOPE-v3's graph is directionally correct as source of truth, vault as projection. |

**Resolution: Dual-Authority Model**
- **Authoring authority** → vault (always). Humans and agents create/modify as markdown + YAML.
- **Query authority** → graph (when available). Currently JSON indices, eventually SCOPE-v3 SQLite.
- **Flow:** Edits go vault→graph. Queries go graph→caller.
- **Reframe SCOPE-v3** from "vault as projection" to "vault as canonical source with graph as query acceleration layer."

### D-2: Memory Tier Granularity
| Einstein | Staff-Architect |
|---|---|
| Section-level is theoretically correct. ADR.Context is episodic, ADR.Decision is semantic. | Not directly addressed; defers to theoretical analysis. |

**Resolution:** Start artifact-level, design section-level. Include nullable `section` field in graph schema from day one. Trigger migration at ~100 artifacts or when retrieval degrades.

### D-3: Vault Transactions vs Atomic Writes
| Einstein | Staff-Architect |
|---|---|
| Domain transactions (e.g., "complete ticket PREP-002") maintaining vault invariants. | Atomic file writes + advisory locking. Transactions may be premature. |

**Resolution:** Both correct at different scales. For 28 tickets, atomic writes suffice. Name interfaces as domain verbs (complete_ticket, not write_file) to preserve the option for transactions later. Don't build transaction layer until invariant violations actually occur.

### D-4: Agent-Generated Metadata Storage
| Einstein | Staff-Architect |
|---|---|
| Key open question. Recommends graph DB table as controlled exception. | Not addressed. |

**Resolution:** Separate artifact metadata (frontmatter, in vault) from system metadata (embeddings, in graph DB). Filesystem analogy: file content in vault, OS annotations in graph.

---

## Artifact-to-Memory-Tier Mapping

| Vault Artifact | Primary Tier | Section Leakage |
|---|---|---|
| **ADR** | Semantic | Context→episodic, Consequences→procedural |
| **Ticket** | Episodic | Acceptance Criteria→procedural |
| **Experiment** | Episodic | Conclusion/Follow-up→procedural |
| **Work Log** | Episodic | Decisions Made→semantic |
| **Reference** (proposed 5th type) | Semantic | — |

---

## Implementation Phases

### Phase 1: Codify Decisions (1-2 days)
- [ ] Write ADR for CLI rejection with quantitative evidence
- [ ] Publish vault design principles as standalone reference doc
- [ ] Document direct file I/O as committed interim approach

### Phase 2: Harden Current System (3-5 days)
- [ ] Define VaultReader/VaultWriter interfaces, refactor /ticket skill
- [ ] Build generation script (derive tickets-index.json + 00-Board.md from .md frontmatter)
- [ ] Implement atomic file writes (temp + rename) for all agent vault operations
- [ ] Add advisory locking (flock) on tickets-index.json

### Phase 3: Prepare for SCOPE-v3 (when work begins)
- [ ] Graph schema with nullable `section` field for future section-level classification
- [ ] Artifact-to-memory-tier mapping at whole-artifact level
- [ ] VaultReader/VaultWriter implementations swappable to graph-backed without caller changes

---

## What NOT to Do

1. **Obsidian CLI as agent interface** — Fails reliability, performance, availability, and strategic alignment.
2. **Graph-DB-authoritative topology** — Breaks human authoring. Developers can't author in "generated" artifacts.
3. **Bidirectional vault↔graph sync** — Inherently complex. Unidirectional vault→graph eliminates entire categories of bugs.
4. **Full transaction layer at current scale** — 28 tickets don't need it. Atomic writes + flock address actual risks.

---

## Assumptions to Validate

| Assumption | Blocking | Priority | Validation |
|---|---|---|---|
| Obsidian wikilink resolution replicable in Go | Yes | High | Edge-case filename test fixtures |
| YAML parsing consistent between Obsidian and Go | Yes | High | Parse all vault frontmatter, compare with Dataview |
| /ticket file I/O pattern generalizes to ADRs | No | Medium | Prototype /adr skill |
| Concurrent access safe with atomic writes | No | Medium | Test Obsidian file-watching with external writes |
| Git adequate for temporal queries | No | Low | Benchmark git log vs SQLite |

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| SCOPE-v3 delayed indefinitely | Medium | Medium | Generation script remains valuable standalone |
| YAML parsing inconsistencies | Low | High | CI check comparing Go parser vs Obsidian Dataview |
| Concurrent human+agent data loss | Low | Medium | Atomic writes, advisory locking, <1ms race window |
| Four-type taxonomy incomplete | Medium | Low | Extensible string enum in graph schema |
| Wikilink resolution differences | Low | Medium | Edge-case test fixtures, document Obsidian's algorithm |
| Interface design doesn't map to SCOPE-v3 | Low | Medium | Domain operations (read_ticket), not storage operations (read_file) |

---

## Open Questions

1. **Agent metadata boundary** — Which annotations are artifact metadata (vault) vs system metadata (graph)?
2. **Conflict resolution** — What does Obsidian actually do when a file is modified externally with unsaved edits?
3. **SCOPE-v3 reframe** — Should v3 spec be updated from "vault as projection" to "vault as canonical, graph as query layer"?
4. **Reference template** — Fifth artifact type for curated external knowledge?
5. **Vault-to-graph compiler** — Incremental interface, full recompile initially?
