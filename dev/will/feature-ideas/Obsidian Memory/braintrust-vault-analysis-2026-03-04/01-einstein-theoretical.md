---
tags:
  - braintrust
  - einstein
  - vault-architecture
  - obsidian-cli
date: 2026-03-04
agent: einstein
model: opus
cost_usd: 1.26
status: complete
---

# Einstein: Theoretical Analysis — EM-Deconvoluter Vault + Obsidian CLI

> **Braintrust team:** braintrust-1772589462
> **Cost:** $1.26 | **Model:** Opus

---

## Executive Summary

The EM-Deconvoluter dev vault embodies seven transferable design principles that make it simultaneously effective for human navigation and machine parsing, with the **Dual-Interface Pattern** (structured frontmatter + semantic markdown body, machine-readable JSON indices alongside human-readable markdown) being the most architecturally significant.

The correct source-of-truth topology is **vault-authoritative with graph-DB as derived index** — the vault is a canonical serialization format that happens to be human-readable, not a UI projection of structured data.

The four artifact types map to cognitive memory tiers with systematic leakage at section boundaries, suggesting that memory tier is a property of artifact SECTIONS, not whole artifacts.

The Obsidian CLI is categorically unsuitable as the primary agent interface (latency, reliability) but has a narrow legitimate role for Obsidian-specific operations that cannot be replicated by file I/O.

---

## Conceptual Framework

**Framework:** Boundary Object Theory (Star & Griesemer, 1989) + Interface Segregation Principle

The vault functions as a **boundary object** — an artifact that sits at the intersection of multiple communities of practice (human developers, AI agents, the planned knowledge graph system) and is plastic enough to adapt to local needs while maintaining a common identity.

- **YAML frontmatter** = shared identity (structured, parseable by all consumers)
- **Markdown body** = local adaptation (human-readable narrative, Obsidian-rendered callouts and mermaid diagrams)
- **JSON indices** = further adaptation for a third consumer (agent query operations)

### Key Insights

1. The vault's effectiveness comes from being simultaneously interpretable by multiple consumers through layered encoding: YAML (structured) → markdown sections (semi-structured) → prose (unstructured). Each layer adds richness at the cost of parseability.

2. The dual-index pattern (tickets-index.json alongside .md files) is **not redundancy** — it's intentional Interface Segregation. The .md files serve human authoring; the .json serves machine querying. The invariant is that they must agree on primary keys and status values.

3. The wikilink graph and the planned SQLite graph serve different cognitive functions: wikilinks enable **associative navigation** (serendipity, browsing), while typed graph edges enable **relational querying** (find all tickets blocked by this ADR). Both are valuable; neither subsumes the other.

---

## Seven Transferable Design Principles

### 1. Dual-Interface Pattern
Structured YAML frontmatter + semantic markdown body. Machine-readable JSON indices alongside human-readable markdown. Different consumers see the interface they need without being burdened by the others.

### 2. Template-as-Schema
Templates enforce structure across artifact types. The four templates (ADR, Ticket, Experiment, Work Log) define the "schema" for each knowledge artifact — which frontmatter fields are required, which sections are mandatory, what lifecycle states exist.

### 3. Vault Authority
The vault is canonical, everything else is derived. Demonstrated by `spec-vault-mapping.md` which explicitly establishes "vault numbering is authoritative." Any derived index must be regenerable from vault files alone.

### 4. Lifecycle-Aware Artifacts
Status fields with defined transitions. Tickets: `pending → in_progress → completed → blocked`. ADRs: `proposed → accepted → superseded → deprecated`. Lifecycle state in frontmatter enables machine-parseable progress tracking.

### 5. Cross-Reference via Wikilinks
`[[links]]` as the relationship layer. Within this vault, wikilinks encode: dependency relationships, rationale links, lifecycle links, temporal links, and navigation links. Section context provides implicit typing.

### 6. Index-as-API
Machine-readable JSON alongside human-readable markdown. `tickets-index.json` provides O(1) lookup on ticket status and dependencies without parsing 28+ markdown files.

### 7. Dependency-as-DAG
Mermaid graphs encoding explicit dependency relationships. Both visual (in Board markdown) and machine-readable (in frontmatter `dependencies` arrays).

---

## Root Cause Analysis

### 1. Source-of-Truth Misframing
**Cause:** The source-of-truth question is misframed as a binary choice between vault and graph DB because the problem conflates two distinct functions: authoring surface vs. query engine.

**Resolution:** The vault serves as an **authoring surface** where humans and agents create/modify artifacts. The graph DB serves as a **query engine** for temporal, relational, and cross-artifact queries. These are complementary functions, not competing authorities.

### 2. Obsidian CLI Category Error
**Cause:** Treating the CLI as an alternative to direct file I/O rather than recognizing it operates at a fundamentally different abstraction level.

Direct file I/O operates on the vault's **serialization format** (markdown + YAML frontmatter). The Obsidian CLI operates on Obsidian's **runtime model** (graph view, plugin state, rendered content). The 22.8% failure rate and 200-500ms latency are not fixable engineering problems — they're symptoms of IPC overhead inherent to communicating with a GUI application's runtime.

### 3. Memory Tier Leakage
**Cause:** The artifact-to-memory-tier mapping appears clean at the artifact level but leaks systematically at the section level.

**Evidence:** An ADR's `## Context` section is episodic; its `## Decision` section is semantic; its `## Consequences` section is procedural. The Experiment template's Conclusion section transforms episodic knowledge into semantic knowledge.

---

## Theoretical Tradeoffs

### Source of Truth: Vault vs Graph DB
- **Vault-authoritative** (recommended): .md files are canonical, graph DB is a derived materialized view. Regenerable from vault alone.
- **Graph-DB-authoritative**: SQLite is canonical, vault files are human-readable projections. Breaks human authoring workflow.

### Agent Interface: CLI vs File I/O
- **Direct file I/O** (recommended): <1ms latency, 100% reliability, no dependency on running Obsidian, works headless.
- **CLI as optional supplement only**: For Obsidian-runtime-specific operations (graph view snapshots).

### Memory Tier Granularity
- **Whole-artifact** (start here): Simple, predictable, easy to implement.
- **Section-level** (grow into): Precise, eliminates tier leakage. Design schema for it from day one.

### Index Strategy
- **Multiple specialized indices** (recommended): tickets-index.json, adr-index.json, etc. Type-specific fields for each. Interface Segregation applied to indices.

---

## Novel Approaches Proposed

### 1. Vault as Canonical Serialization Format
Instead of asking "is the vault a UI concern or a data concern?", recognize it as a canonical serialization format — analogous to how protobuf `.proto` files are both human-readable definitions AND the source from which code is generated. The graph DB is compiled output. Obsidian rendering is a view.

### 2. Section-Granular Memory Tier Tagging
Instead of classifying entire artifacts, classify at the markdown section level. ADR.Context→episodic, ADR.Decision→semantic, ADR.Consequences→procedural. Templates already define canonical section structures, making section parsing deterministic.

### 3. Agent Operations as Vault Transactions
Instead of CRUD, model agent operations as domain transactions that maintain vault invariants. "Complete ticket PREP-002" atomically updates frontmatter + index + board + validates acceptance criteria.

### 4. Dual Graph Layers
The wikilink graph (emergent, associative) and the SQLite graph (designed, typed) serve different cognitive functions and should coexist. Wikilinks without corresponding typed edges are candidates for relationship classification.

---

## First Principles: Fundamental Constraints

1. **Human readability and machine parseability are in fundamental tension.** YAML frontmatter + markdown body is the Pareto-optimal encoding.
2. **Artifact identity IS filename identity.** Wikilink resolution depends on filename uniqueness.
3. **Agent operations must be atomic at file level and idempotent.** Multi-file transactions require explicit coordination.
4. **The vault must remain functional without agents or graph DB.** Wikilinks, not database queries, are the primary navigation.
5. **Any derived index must be regenerable from vault files alone.** The "canonical form" constraint.

---

## Assumptions Surfaced

| Assumption | Risk if False | Validation Method |
|---|---|---|
| Obsidian wikilink resolution algorithm is stable/replicable in Go | Graph DB resolves links differently | Test with edge-case filenames |
| YAML frontmatter parsing is consistent between Obsidian and Go | Silent data mismatches | Parse all vault frontmatter with Go, compare with Dataview |
| /ticket skill's file I/O pattern scales to all artifact types | ADR supersedes chains break the pattern | Prototype /adr skill |
| Git provides adequate temporal history | Field-level queries too slow | Benchmark git log vs SQLite |
| Concurrent human+agent access won't cause conflicts | Data loss from last-writer-wins | Test Obsidian's file-watching behavior |
| Four-template taxonomy is stable | Graph schema migration needed | Check if References directory needs a 5th type |

---

## Open Questions

1. **Where to store agent-generated metadata** (embeddings, confidence scores) if the vault is canonical? → Likely graph DB table (controlled exception to "graph is derived" principle)
2. **Conflict resolution strategy** for simultaneous human+agent edits? → Test Obsidian's file-watching behavior empirically
3. **Vault-to-graph compiler**: incremental or full recompilation? → Design incremental interface, implement full recompile initially (<100ms at 45 files)
4. **Reference template needed?** → Check if References directory documents follow a pattern
5. **Mermaid graph derivation**: Should 00-Board.md's mermaid graph be generated from frontmatter dependency arrays?
