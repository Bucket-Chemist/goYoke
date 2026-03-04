# GOgent-Fortress Embedded Knowledge Graph — Unified Scope (v3)

> **Status:** APPROVED FOR IMPLEMENTATION — All blockers resolved, validation tasks defined
> **Source:** SCOPE-v2 + Braintrust Analysis (Einstein + Staff-Architect + Beethoven synthesis)
> **Date:** 2026-02-19 (v3)
> **Use:** Input to `/plan-tickets` and `/implement`

---

## v3 Changelog

> Braintrust analysis (Einstein theoretical + Staff-Architect practical + Beethoven synthesis) produced 3 corrections to existing deltas and 3 new deltas addressing gaps in both v1 and v2.

| # | Section | Change | Severity | Source |
|---|---------|--------|----------|--------|
| Δ6-CORRECTED | §5, §10 | **goldmark-obsidian coverage corrected: 7/10, not 10/10.** Callouts, highlights, and comments are listed as "Not Yet Implemented" in upstream. VojtaStruhar/goldmark-obsidian-callout **retained** as companion dependency. Hybrid extension stack adopted. | Major | Einstein + Staff-Architect (convergence) |
| Δ11-QUALIFIED | §5 | **viterin/vek claim qualified.** Changed from "extends chromem-go viability by 2-4x" to "up to 4x kernel speedup; end-to-end improvement requires benchmarking". SIMD accelerates distance kernel but memory bandwidth and allocation overhead reduce total gain. | Minor | Einstein (divergence resolution) |
| Δ13 | §6.1, §16 | **FTS5 external content sync gap.** Current schema uses `content=nodes` but has NO sync triggers — new nodes unsearchable, updates stale, deletes produce phantoms. **Design decision required:** external content with AFTER triggers OR contentless FTS5 with application sync. Sharp edge added. | Critical | Einstein (new finding) |
| Δ14 | §9 | **Entity embedding input format specified.** Context-enriched embeddings (`type::name::description`) adopted as primary with name-only fallback. Prevents homonym false-merges (e.g., "Einstein" agent vs physicist). | Major | Einstein (new finding) |
| Δ15 | §9 | **Tier 4 default added for borderline entity resolution.** Scores in range (cosine 0.60-0.89, Jaro-Winkler <0.85) now have explicit disposition: DISTINCT + flag for periodic review. Prevents unbounded duplicate accumulation. | Minor | Staff-Architect (new finding) |
| Δ16 | §14 | **Blocking validation tasks formalized.** Two HIGH-BLOCKING validations must pass before Phase 2: nomic-embed-text L2 normalization, hybrid goldmark extension coexistence. | Major | Beethoven (synthesis) |
| Δ17 | §1, §2, §10 | **Dual-authority model for vault/graph topology.** Braintrust vault-architecture analysis (2026-03-04) resolved the source-of-truth tension: `.gogent-vault/` (system memory) remains graph-authoritative (consolidation pipeline writes vault as export). Per-project dev vaults (EM-Deconvoluter style) are vault-authoritative (human/agent edits canonical, graph is derived query layer). `VaultSync` interface unchanged — direction is always graph→vault for system memory and vault→graph for dev vaults. See `Obsidian Memory/braintrust-vault-analysis-2026-03-04/`. | Major | Braintrust synthesis (2026-03-04) |

### Cumulative Delta Summary (v1 → v3)

| Delta | Status | Confidence |
|-------|--------|------------|
| Δ1 (coder/hnsw CC0-1.0) | ✅ Verified correct | High |
| Δ2 (chromem-go MPL-2.0) | ✅ Verified correct | High |
| Δ3 (coder/hnsw as fallback) | ✅ Verified correct | High |
| Δ4 (fogfish/hnsw option) | ⚠️ Correct but weakly grounded | Medium |
| Δ5 (atomic.Pointer cache) | ✅ Verified correct | High |
| Δ6 (goldmark-obsidian) | ⚠️ **Corrected in v3** — 7/10 coverage | High |
| Δ7 (goldmark parse-only) | ✅ Verified correct | High |
| Δ8 (embedding-first resolution) | ✅ Verified correct | High |
| Δ9 (RRF → CC upgrade path) | ✅ Verified correct | High |
| Δ10 (sharp edges update) | ✅ Verified correct | High |
| Δ11 (viterin/vek SIMD) | ⚠️ **Qualified in v3** — requires benchmarking | Medium |
| Δ12 (Appendix A update) | ✅ Verified correct | High |
| Δ13 (FTS5 sync) | 🆕 **New in v3** | High |
| Δ14 (embedding format) | 🆕 **New in v3** | High |
| Δ15 (Tier 4 default) | 🆕 **New in v3** | High |
| Δ16 (validation tasks) | 🆕 **New in v3** | High |
| Δ17 (dual-authority model) | 🆕 **New in v3.1** | High |

---

## 1. Vision

Replace the current flat JSONL + handoff-document memory system with a **structured temporal knowledge graph** that provides semantically rich, graph-queryable memory while maintaining sub-100ms retrieval. The system must:

- Be a single zero-dependency Go binary (no CGO, no external services)
- Extend `memory-archivist` rather than replacing it
- Give humans a browsable, editable view via Obsidian vault compatibility
- Integrate into the existing hook chain without exceeding the latency budget

The **Obsidian-compatible Markdown vault** is the genuine architectural differentiator — no existing agent framework offers a human-inspectable, human-editable knowledge graph. This is worth preserving at all costs.

---

## 1a. Blocking Assumptions — VALIDATED (2026-02-19)

> Both HIGH-BLOCKING open questions from Section 14 answered before planning. Safe to proceed with all 4 phases.

### Latency Budget (was: blocking Phase 4 design)

Measured p50 execution times for all three SessionStart-chain hooks:

| Hook | p50 | Notes |
|------|-----|-------|
| `gogent-load-context` | **28ms** | Integration point for memory retrieval |
| `gogent-validate` | **6ms** | Per-Task call, not session startup |
| `gogent-sharp-edge` | **10ms** | Per-tool, not session startup |

**Available for memory retrieval inside gogent-load-context: 72ms** (100ms budget − 28ms baseline).
Target was <50ms. **Budget is fine — synchronous retrieval is viable. No async injection needed.**

### Entity Extraction Rate (was: blocking Phase 3 design)

Estimated from 417 sessions of handoff history + MEMORY.md content patterns:
- Raw entities per session: ~15-30 (decisions, files, agents, concepts)
- After deduplication/entity resolution: **~5-15 new unique entities/session**
- At 15/session → 10K node ceiling reached in ~667 sessions (~1.5-2 years at current pace)

**Well below Einstein's 100/session threshold. chromem-go 100K cap is not a near-term constraint.**
Auto-extraction is viable. Precision validation (≥0.8) still required before Phase 3 goes live.

---

## 2. Architecture: Simplified Two-Layer Design

> **Decision source:** Beethoven synthesis of Einstein (theoretical) + Staff-Architect (practical)
> **Verdict:** APPROVED FOR IMPLEMENTATION — All blockers resolved

### 2.1 What We Are Building

```
┌─────────────────────────────────────────────────────────────────────┐
│                    RUNTIME PATH                                      │
│                                                                      │
│  atomic.Pointer  ←──  Core blocks (~10 entries)                     │  [Δ5]
│  [map]cache            <1ms reads, zero contention                   │
│       ↑                                                              │
│  SQLite (truth)   ←──  Full graph, FTS5, bi-temporal                │
│       ↑                  1-50ms reads                                │
│       ↑                  FTS5 synced via AFTER triggers              │  [Δ13]
│  chromem-go       ←──  Vector similarity search                     │
│                          ~8ms@10K, ~80ms@100K (768d)                │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│               CONSOLIDATION PATH (offline)                           │
│                                                                      │
│  [Session end] → Entity extraction (Claude Sonnet)                  │
│  → Entity resolution (embedding-first OR-logic)                     │  [Δ8]
│      └─ Context-enriched embeddings (type::name::description)       │  [Δ14]
│      └─ Tier 4 default: DISTINCT + flag for review                  │  [Δ15]
│  → Graph algorithm batch (PageRank, Louvain, labels)                │
│  → Pre-computed features written to SQLite columns                  │
│  → Markdown export → .gogent-vault/                                 │
│    (direct string construction, not goldmark render)                │  [Δ7]
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│               SESSION STARTUP PATH                                   │
│                                                                      │
│  Load core blocks → atomic.Pointer[map]                             │  [Δ5]
│  Scan vault for human edits → import changed files                  │
│  Hybrid query → top-k context injection                             │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 What We Are NOT Building (v1)

| Dropped from original design | Why | Upgrade path |
|------------------------------|-----|--------------|
| **bbolt cache layer** | 5ms→0.5ms improvement is imperceptible; atomic.Pointer[map] provides equivalent benefit without a second consistency boundary (27-125 combined failure modes across 3 engines) | Add if profiling reveals SQLite reads are the bottleneck |
| **FSRS/Ebbinghaus scoring** | Agents don't forget — they lose context window space. Spaced repetition optimizes for human biological retention, not information retrieval precision | Already replaced by IR scoring below |
| **Bidirectional real-time vault sync** | Solves a rare use case (human edits during active session) at maximum complexity (fsnotify + debouncing + conflict resolution + feedback loop prevention) | v2 enhancement if users demand it; sharp edges documentation already covers pitfalls |
| **Runtime graph traversal** | PageRank/Louvain run in 50-500ms — violates latency budget. Flat retrieval handles 90%+ of agent query patterns | Pre-computed features from consolidation serve the same need |

---

## 3. Deployment Blockers — ALL RESOLVED

> **v3 status:** All blockers from v1/v2 have been resolved. Implementation may proceed.

### ~~BLOCKER C-1: AGPL License Contradiction~~ — RESOLVED (v2)

> **[Δ1]** `github.com/coder/hnsw` is licensed CC0-1.0 (public domain dedication), not AGPL-3.0. Verified against upstream LICENSE file. Now approved as fallback dependency.

### ~~BLOCKER M-1: No agents-index.json Entry~~ — RESOLUTION DEFINED

**Problem:** `go-db-architect` has no entry in `agents-index.json`.

**Resolution:** Create full agents-index.json entry:
```json
{
  "id": "go-db-architect",
  "name": "Go DB Architect",
  "model": "sonnet",
  "thinking": true,
  "thinking_budget": 14000,
  "tier": 2,
  "category": "language",
  "triggers": ["graph schema", "knowledge graph", "memory graph", "memory subsystem schema",
                "temporal graph", "db architect", "graph store", "consolidation pipeline",
                "entity extraction", "vector index", "embedding store"],
  "auto_activate": {
    "paths": ["internal/memory/**", "internal/graphstore/**", "internal/vectorindex/**",
              "internal/vault/**", "internal/consolidation/**"]
  },
  "context_requirements": {
    "conventions": {
      "base": ["db-conventions.md"]
    }
  },
  "sharp_edges_count": 20
}
```
Also add `go-db-architect` to `impl-manager.can_spawn` and `orchestrator.can_spawn`.

**Effort:** 30-60 minutes including validation testing.
**Owner:** Phase 1 implementer.

### ~~BLOCKER M-3: Wrong File Location~~ — RESOLUTION DEFINED

**Problem:** Agent definition lives in `tickets/obsidian-cli-knowledge-graph/`.

**Resolution:**
```bash
mkdir -p ~/.claude/agents/go-db-architect/
mv tickets/obsidian-cli-knowledge-graph/go-db-architect.md ~/.claude/agents/go-db-architect/go-db-architect.md
mv tickets/obsidian-cli-knowledge-graph/go-db-architect-sharp-edges.yaml ~/.claude/agents/go-db-architect/sharp-edges.yaml
```

**Effort:** 5 minutes.
**Owner:** Phase 1 implementer.

---

## 4. Agent Definition Refactoring (Pre-Implementation)

> This must be done before registering in agents-index.json

**Problem:** The current `go-db-architect.md` is 882 lines — 2.4x the peer average (go-pro: 374, go-tui: 471). It contains complete function implementations that inflate every invocation's context budget unnecessarily.

**Resolution:** Split into two files following the go-tui pattern:

**`~/.claude/agents/go-db-architect/go-db-architect.md` (~350 lines)**
- Role description and system constraints
- Approved dependencies (with license annotations — corrected per Δ1, Δ2, Δ6-CORRECTED)
- Focus area rules as CORRECT/WRONG patterns (not full implementations)
- Interface contracts section (GraphStore, VectorIndex, VaultSync interfaces)
- Testing strategy overview
- Output requirements and parallelization model

**`~/.claude/conventions/db-conventions.md` (~350 lines)**
- Full SQL schema with all DDL including FTS5 sync triggers [Δ13]
- Complete function implementations (decayScore, compositeScore, RRF, RunMigrations)
- Algorithm details (entity resolution with context-enriched embeddings [Δ14], label propagation)
- Template strings and query patterns
- Vault file structure and hybrid goldmark extension stack [Δ6-CORRECTED]

**Wire via** `context_requirements.conventions.base: ["db-conventions.md"]` in agents-index.json.

---

## 5. Technology Stack

> **[Δ1, Δ2, Δ4, Δ5, Δ6-CORRECTED, Δ11-QUALIFIED]** Corrected license column. Added fallback HNSW options. Replaced sync.Map. Hybrid goldmark extension stack. Qualified SIMD claim.

| Component | Library | License | Purpose |
|-----------|---------|---------|---------|
| **SQLite** | `modernc.org/sqlite` | BSD-3 + Public Domain | Graph store, FTS5 (compiled in by default), WAL mode, bi-temporal schema. Embeds SQLite 3.51.2. |
| **Vector search** | `github.com/philippgille/chromem-go` | **MPL-2.0** [Δ2] | In-process embedding store, exhaustive NN, <100K vectors. Zero deps. ~8ms@10K, ~80ms@100K (768d). |
| **In-process cache** | `atomic.Pointer[map]` (stdlib) [Δ5] | BSD | Core blocks hot path, copy-on-write, zero read contention |
| **Graph algorithms** | `gonum.org/v1/gonum/graph` | BSD-3 | PageRank, community detection, BFS (consolidation only) |
| **Entity similarity** | `github.com/hbollon/go-edlib` | MIT | Jaro-Winkler, Levenshtein, Sørensen-Dice for entity resolution (secondary signal). Uses float32. |
| **Embeddings** | Ollama `nomic-embed-text` | — | 768-dim, 8K context, local. **L2 normalization must be validated before Phase 2.** |
| **HNSW fallback** | `github.com/coder/hnsw` [Δ1, Δ3] | **CC0-1.0** (public domain) | Pure Go HNSW with generics. Fallback when chromem-go exhaustive search exceeds latency budget (>100K vectors). |
| **HNSW fallback (alt)** | `github.com/fogfish/hnsw` [Δ4] | MIT | Pure Go HNSW with generics + pluggable Surface[Vector] interface. Pairs with `kshard/vector` for SIMD-accelerated distance. |
| **SIMD acceleration** | `github.com/viterin/vek` [Δ11-QUALIFIED] | MIT | SIMD dot products for brute-force distance computation. **Up to 4x kernel speedup; end-to-end improvement requires benchmarking before relying on for scaling projections.** |
| **NOT included** | `go.etcd.io/bbolt` | MIT | Removed — atomic.Pointer[map] replaces it |

### Markdown Parsing: Hybrid Extension Stack [Δ6-CORRECTED]

> **Critical correction from Braintrust analysis:** powerman/goldmark-obsidian covers 7/10 required features, NOT 10/10. Callouts, highlights, and comments are listed as "Not Yet Implemented" in upstream (as of Feb 2026). The v2 consolidation was a **net regression** from v1 which included working callout support.

| Extension | License | Features | Status |
|-----------|---------|----------|--------|
| `github.com/powerman/goldmark-obsidian` | MIT | Wikilinks (aliases, fragments, embeds), hashtags (ObsidianVariant), block IDs, YAML frontmatter, footnotes (reference-style only), LaTeX, Mermaid | ✅ Primary |
| `github.com/VojtaStruhar/goldmark-obsidian-callout` | MIT | `> [!note]` callout blocks (all Obsidian callout types, collapsible, nested) | ✅ **Retained** — required for callout parsing |
| Custom parser (~30 lines) | — | `==highlights==` | 🔨 Build if needed |
| Custom parser (~30 lines) | — | `%%comments%%` | 🔨 Build if needed |

**Validation task (blocking Phase 3):** Register both extensions in goldmark pipeline and parse 50 representative vault files containing callouts + wikilinks + hashtags. Verify no AST node conflicts. goldmark's extension model is designed for composability — conflict is unlikely but must be verified.

> **[Δ2] MPL-2.0 compliance note:** chromem-go's MPL-2.0 license permits use in proprietary binaries without modification. If GOgent-Fortress modifies chromem-go source files, those modifications must be disclosed under MPL-2.0. Consuming it as an unmodified dependency via `go get` requires no disclosure. Track this in the project's LICENSE-THIRD-PARTY file.

**SQLite PRAGMA configuration:**
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;           -- reduced from 10000; 5s is production standard
PRAGMA cache_size = -32000;           -- 32MB
PRAGMA temp_store = MEMORY;
PRAGMA foreign_keys = ON;
PRAGMA mmap_size = 268435456;         -- 256MB
PRAGMA wal_autocheckpoint = 1000;
```

---

## 6. Schema Design

### 6.1 Core Tables

```sql
-- Nodes: entities in the knowledge graph
CREATE TABLE nodes (
    id          INTEGER PRIMARY KEY,
    type        TEXT NOT NULL,          -- 'agent', 'concept', 'decision', 'error', 'convention'
    name        TEXT NOT NULL,
    properties  TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(properties)),
    pagerank    REAL NOT NULL DEFAULT 0.0,   -- pre-computed at consolidation
    community   INTEGER,                     -- pre-computed at consolidation
    embedding   BLOB,                        -- stored alongside for retrieval
    model_ver   TEXT NOT NULL DEFAULT '',    -- embedding model version tag
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now'))
) STRICT;

-- Edges: bi-temporal relationships (Graphiti-inspired, independent reimplementation)
CREATE TABLE edges (
    id              INTEGER PRIMARY KEY,
    source_id       INTEGER NOT NULL REFERENCES nodes(id),
    target_id       INTEGER NOT NULL REFERENCES nodes(id),
    relation_type   TEXT NOT NULL,
    properties      TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(properties)),
    weight          REAL NOT NULL DEFAULT 1.0,
    -- Bi-temporal columns (4 extra columns, low cost, expensive to retrofit)
    valid_from      TEXT NOT NULL,
    valid_to        TEXT NOT NULL DEFAULT '9999-12-31T23:59:59.999',
    tx_from         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    tx_to           TEXT NOT NULL DEFAULT '9999-12-31T23:59:59.999',
    confidence      REAL NOT NULL DEFAULT 1.0,
    source_type     TEXT NOT NULL DEFAULT 'agent'  -- 'agent', 'human', 'extracted'
) STRICT;

-- Episodes: raw session records (non-lossy, append-only)
CREATE TABLE episodes (
    id          INTEGER PRIMARY KEY,
    session_id  TEXT NOT NULL,
    content     TEXT NOT NULL,
    summary     TEXT,
    occurred_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    tx_from     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now'))
) STRICT;

-- FTS5 index for BM25 keyword search
-- [Δ13] Using external content table WITH sync triggers
CREATE VIRTUAL TABLE nodes_fts USING fts5(
    name, properties,
    content=nodes, content_rowid=id,
    tokenize='porter unicode61'
);

-- [Δ13] FTS5 sync triggers — CRITICAL for external content tables
-- Without these, FTS5 index silently desyncs from nodes table
CREATE TRIGGER nodes_fts_insert AFTER INSERT ON nodes BEGIN
    INSERT INTO nodes_fts(rowid, name, properties)
    VALUES (new.id, new.name, new.properties);
END;

CREATE TRIGGER nodes_fts_update AFTER UPDATE ON nodes BEGIN
    UPDATE nodes_fts SET name = new.name, properties = new.properties
    WHERE rowid = old.id;
END;

CREATE TRIGGER nodes_fts_delete AFTER DELETE ON nodes BEGIN
    DELETE FROM nodes_fts WHERE rowid = old.id;
END;
```

### 6.2 Critical Indexes
```sql
-- Graph traversal (always filter by start node + edge type)
CREATE INDEX idx_edges_source ON edges(source_id, relation_type);
CREATE INDEX idx_edges_target ON edges(target_id, relation_type);
-- Current-truth queries (bi-temporal "latest" filter)
CREATE INDEX idx_edges_current ON edges(source_id, relation_type, tx_to, valid_to);
-- Episode retrieval by session
CREATE INDEX idx_episodes_session ON episodes(session_id, occurred_at);
-- Node lookup by type (for core block loading)
CREATE INDEX idx_nodes_type ON nodes(type);
```

### 6.3 Bi-Temporal Note
The 4 bi-temporal columns on `edges` are **retained but no query utilities built until concrete examples emerge** (Beethoven resolution of Einstein/Staff-Architect divergence). Cost of adding 4 columns: minutes. Cost of retrofitting bi-temporal later: days. After 3 months of operation, if no bi-temporal query has been articulated, columns are inert schema — ignore and proceed.

### 6.4 FTS5 Design Decision [Δ13]

> **Braintrust finding:** v1 and v2 both specified `content=nodes` (external content table) but omitted sync triggers. Per SQLite documentation, external content tables do NOT auto-sync — this would cause silent retrieval corruption.

**Decision:** Use external content table WITH AFTER triggers (the SQLite-canonical approach).

**Rationale:**
- External content saves storage (no content duplication)
- AFTER triggers ensure sync at the database layer (not application layer)
- Easier to test correctness (triggers fire automatically, application sync requires test coverage)
- SQLite-canonical pattern with extensive documentation

**Alternative considered:** Contentless FTS5 (`content=''`) with application-level sync in CRUD functions. Simpler to understand for teams with limited SQLite FTS5 experience but requires explicit sync calls in every Insert/Update/Delete path.

---

## 7. Retrieval Scoring: IR-Based (Replacing FSRS/Ebbinghaus)

> **Decision:** Einstein's reframing adopted. The math changes; cognitive metaphors persist only at the Obsidian UI layer (e.g., "memory strength" as a label).

**Composite relevance score:**
```
Score = α × BM25(query, memory_name_properties)
      + β × cosine_sim(query_embedding, memory_embedding)
      + γ × log(1 + access_count) / log(1 + age_days)   -- frequency boost
      + δ × pre_computed_pagerank                         -- graph centrality
```

**Initial weights (tune empirically):** α=0.3, β=0.5, γ=0.1, δ=0.1

**Time decay:** Simple logarithmic decay — tunable to empirical retrieval utility data. No FSRS power-law, no Ebbinghaus exponential. Both are theoretically inapplicable to agents (agents don't forget; they lose context window space — a capacity constraint, not biological decay).

**Reciprocal Rank Fusion (k=60)** for merging BM25 and vector search result sets before final ranking.

> **[Δ9] Upgrade path: Convex Combination (CC).** Bruch et al. (ACM TOIS 2022, "An Analysis of Fusion Functions for Hybrid Retrieval") shows CC outperforms RRF — more sample-efficient, more robust to domain shift, and theoretically sounder because RRF discards score distribution information. CC formula: `score(d) = α·norm(score_vector) + (1-α)·norm(score_bm25)` with α≈0.7 as starting point. Requires min-max normalisation of BM25 scores (cosine similarity is already bounded). **Default remains RRF k=60 for v1** — graduate to CC once evaluation data exists to tune α. No code change needed; the scoring interface should accept pluggable fusion strategies from the start.

---

## 8. Three-Phase Startup Loading

**Phase 1 — Core blocks (<10ms, from atomic.Pointer[map]):** [Δ5]
Always injected into system prompt. Covers: agent persona/role, user profile, active project context, critical procedural rules. Budget: ~10-15% context window (400-600 tokens). On startup: load all `type='core_block'` nodes from SQLite → populate map, store via `atomic.Pointer.Store()`. On write: update SQLite first → build new map copy → `atomic.Pointer.Store()` atomically. Reads: `cache.Load()` — zero contention, no locks.

**Constructor must initialize atomic.Pointer with empty map** to prevent nil dereference before first SQLite load completes.

**Phase 2 — Session-relevant retrieval (<50ms, from SQLite + chromem-go):**
Query using hook event metadata (opened file, command run, error message) as context. Hybrid retrieval: BM25 (30% weight) + embedding cosine similarity (70% weight) via FTS5 + chromem-go. Return top-k results. Budget: ~10-20% context window.

**Phase 3 — Lazy on-demand (<100ms per query):**
Expose `memory_search` and `memory_get` tools to the agent. Model-driven retrieval decisions following Claude Code's production pattern.

**Token budget allocation (200K context window):**
- System instructions: ~15% (30K)
- Core memory blocks: ~10% (20K)
- Retrieved memories: ~15% (30K)
- Active conversation: ~60% (120K)

---

## 9. Consolidation Pipeline (Offline, Session-End)

> All graph algorithm work happens here. Runtime path has zero graph computation.

```
[Session end trigger]
    ↓
[Pre-compaction flush (optional)]
    Inject: "Session ending. Extract durable insights to knowledge graph."
    Note: OpenClaw data suggests ~60% NO_REPLY rate — validate before relying on this
    ↓
[Entity extraction (Claude Sonnet)]
    Structured output: entity nodes + relationship triples
    Validate precision on 50 sessions before committing to auto-extraction
    Threshold: reject extractions below 0.8 confidence
    ↓
[Entity resolution — embedding-first OR-logic]                              [Δ8, Δ14, Δ15]

    EMBEDDING INPUT FORMAT [Δ14]:
        Primary: context-enriched string "{type}::{name}::{description}"
                 e.g. "agent::Einstein::Theoretical analysis agent for Braintrust"
        Fallback: name-only (when description unavailable)
        Rationale: Prevents homonym false-merges (Einstein agent vs Einstein physicist)

    Tier 1 (semantic match):  cosine_sim(embed_a, embed_b) ≥ 0.90 → MERGE
        Handles abbreviations, synonyms, rephrased concepts
        (e.g. "MIT" ↔ "Massachusetts Institute of Technology")

    Tier 2 (fuzzy + semantic): jaro_winkler(a, b) ≥ 0.85
                            AND cosine_sim(embed_a, embed_b) ≥ 0.70 → MERGE
        Handles typos and minor variations

    Tier 3 (auto-reject):    both signals < 0.60 → DISTINCT ENTITIES

    Tier 4 (borderline) [Δ15]: (cosine 0.60-0.89) AND (jaro_winkler < 0.85)
                            → DISTINCT + flag for periodic review
        Prevents unbounded duplicate accumulation
        Flag stored in node.properties as "resolution_review_needed": true
        Periodic deduplication pass reviews flagged pairs

    Alias dictionary: deterministic backstop for known abbreviations/project names
    ↓
[Graph algorithms (gonum, consolidation-only)]
    PageRank → nodes.pagerank
    Louvain community detection → nodes.community
    Label propagation (custom ~30 lines) → community refinement
    Write pre-computed features back to SQLite
    ↓
[Markdown export → .gogent-vault/]
    Write entity files via direct string construction (not goldmark render)  [Δ7]
    YAML frontmatter + wikilinks built as text templates
    Store content hash in frontmatter for change detection
    ↓
[Done — next session startup scans vault for human edits]
```

### 9.1 Entity Resolution Threshold Calibration

> **Einstein recommendation:** After first 20 consolidation passes, audit all auto-merges at Tier 1 (0.90 threshold). If false positives appear, raise threshold to 0.92-0.95. Context-enriched embeddings [Δ14] may make 0.90 sufficiently discriminative — empirical validation required.

---

## 10. Obsidian Vault: Write-Only Export + Batch Import

> **Decision:** v1 is write-only from agent. No fsnotify, no real-time sync. Eliminates entire sync subsystem.

**Agent writes (on consolidation):**
- Entity nodes → `entities/{type}/{name}.md`
- Episode summaries → `episodes/YYYY-MM-DD.md`
- Procedural rules → `procedures/*.md`
- Community summaries → `communities/*.md`
- Auto-synthesized brief → `MEMORY.md` (first 200 lines injected at startup)
- Content hash in YAML frontmatter for change detection

**Human reads (always available in Obsidian):**
- Browse entities, graph visualization (Obsidian Graph View), Dataview queries
- Edit any Markdown file between sessions

**Human edits detected (on next session startup):**
- Scan vault directory, compare content hashes against last-known SQLite state
- Import changed files as `source_type='human'` entities
- No conflict resolution needed (human edits are always "later" than last export)

**Vault structure:**
```
.gogent-vault/
├── entities/
│   ├── agents/
│   ├── concepts/
│   ├── decisions/
│   └── errors/
├── episodes/
├── procedures/
├── communities/
├── MEMORY.md               ← auto-synthesized brief, loaded at startup
└── .gogent/
    ├── graph.db            ← SQLite (agent-only, gitignored)
    └── index/              ← chromem-go persistence (agent-only)
```

### Markdown Strategy [Δ6-CORRECTED, Δ7]

> **Critical: goldmark cannot round-trip Obsidian markdown.** goldmark is a Markdown→HTML converter. Its AST discards whitespace, formatting details, and Obsidian-specific syntax during rendering. A wikilink `[[Foo]]` becomes `<a href="Foo.html">Foo</a>` — not `[[Foo]]`.

**For parsing (vault import / human edit detection):**
Use **hybrid extension stack** [Δ6-CORRECTED]:
1. `powerman/goldmark-obsidian` — wikilinks (including aliases and fragments), `![[embeds]]`, `#hashtags` (ObsidianVariant), block IDs, YAML frontmatter, footnotes (reference-style only), LaTeX, Mermaid
2. `VojtaStruhar/goldmark-obsidian-callout` — `> [!note]` callouts (all Obsidian callout types, collapsible + nested)
3. Custom parsers if needed for `==highlights==` and `%%comments%%`

**For writing (vault export from consolidation):**
Construct Markdown directly as text strings via Go `text/template` or `fmt.Sprintf`. Do not parse→transform→render through goldmark's AST. This preserves Obsidian-specific syntax with perfect fidelity.

**Known Obsidian syntax gaps (no goldmark extension covers these):**
- Document aliases defined in YAML `aliases:` property (Obsidian-specific)
- Dataview plugin inline fields (`key:: value`)
- Tags defined in YAML frontmatter `tags:` (vs inline `#tags`)
- Footnotes: powerman supports reference-style only, not inline footnotes

**v2 (future, when user demand warrants):** Bidirectional real-time sync via fsnotify. Sharp edges documentation already covers all pitfalls (duplicate events, no recursive watch, write feedback loop, debouncing patterns) — retain as v2 reference.

---

## 11. Hook Integration

**Integration point:** Extend `gogent-load-context` (SessionStart hook) to query knowledge graph instead of (or in addition to) JSONL files.

**Decision points for Phase 4:**
- Extend existing hook binary vs. create new hook
- JSONL dual-read period: how long to maintain backward compatibility
- Migration trigger: one-time JSONL→SQLite import vs. gradual dual-write

**Latency budget allocation:**
- Hook baseline: 28ms
- Available for memory retrieval: 72ms
- Target retrieval latency: <50ms (leaves 22ms headroom)

---

## 12. Memory-Archivist Relationship

**Relationship question (resolved):** `go-db-architect` handles the new graph-based semantic layer. `memory-archivist` continues to handle session handoff generation during transition.

**Working model:**
- During transition: dual-write to both systems
- Post-migration: memory-archivist writes episodes to SQLite instead of JSONL
- `go-db-architect` is NOT in memory-archivist's `can_spawn` list (different trigger patterns, not a dependency relationship)

**Migration path:**
- Session scope JSONL → episodes table
- Project scope JSONL → semantic entity nodes
- Global scope JSONL → procedural memory nodes
- ML scope JSONL → telemetry subgraph (new fourth category)

---

## 13. Implementation Phases

### Phase 0 — Pre-Implementation Validation (~1 day) [Δ16]

> **BLOCKING:** These validations must pass before Phase 2 implementation begins.

**Validation Task 1: nomic-embed-text L2 Normalization**
- Method: Embed 10 test strings via Ollama nomic-embed-text, compute L2 norms
- Success: All norms within [0.99, 1.01]
- Failure action: Add pre-normalization at storage time (~0.1ms per vector), recalibrate all cosine similarity thresholds
- Owner: Phase 2 implementer
- Effort: ~30 minutes

**Validation Task 2: Hybrid goldmark Extension Coexistence**
- Method: Register powerman/goldmark-obsidian + VojtaStruhar callout extension in goldmark pipeline, parse 50 vault files containing callouts + wikilinks + hashtags
- Success: No AST node conflicts, all features parsed correctly
- Failure action: Investigate conflict, potentially fork or patch one extension
- Owner: Phase 3 implementer
- Effort: ~2 hours

**Validation Task 3: viterin/vek End-to-End Benchmark** (non-blocking)
- Method: Benchmark chromem-go with and without viterin/vek at 768d, 10K and 100K vectors
- Measure: End-to-end query latency, not just kernel speedup
- Output: Qualified performance claim with empirical data
- Owner: Phase 2 implementer
- Effort: ~2 hours

### Phase 1 — Deployment Setup (~2 hours)

**Must complete before any implementation code is written.**

1. Resolve M-3: Move files to `~/.claude/agents/go-db-architect/`
2. Refactor agent definition: split into go-db-architect.md (~350 lines) + db-conventions.md (~350 lines)
3. Update with v3 corrections:
   - Hybrid goldmark extension stack [Δ6-CORRECTED]
   - FTS5 sync triggers [Δ13]
   - Context-enriched entity embeddings [Δ14]
   - Tier 4 borderline disposition [Δ15]
   - Qualified SIMD claim [Δ11-QUALIFIED]
4. Resolve M-1: Create agents-index.json entry + add to can_spawn lists
5. Validate routing: `gogent-validate routes to go-db-architect for "graph schema" trigger`

**Success criteria:**
- [ ] `go-db-architect` in agents-index.json with all required fields
- [ ] Identity injection loads from `~/.claude/agents/go-db-architect/go-db-architect.md`
- [ ] Agent identity ≤400 lines; db-conventions.md ≤400 lines
- [ ] License annotations correct (chromem-go: MPL-2.0, coder/hnsw: CC0-1.0)
- [ ] gogent-validate routes correctly for "graph schema", "knowledge graph" triggers
- [ ] db-conventions.md includes FTS5 sync triggers

### Phase 2 — Core Storage Layer (~2 weeks)

SQLite schema with FTS5 sync triggers, atomic.Pointer[map] cache, IR-based scoring, basic CRUD + search.

**Prerequisites:**
- [ ] Phase 0 Validation Task 1 (nomic-embed-text normalization) PASSED
- [ ] Phase 1 complete

**Decision points before starting:**
- Validate chromem-go with agent memory content types (code, errors, decisions) — embed 100 representative memories, run 20 queries
- Choose BM25 implementation: SQLite FTS5 rank function vs. Go-side scoring
- Set initial α/β/γ/δ weights (recommend 0.3/0.5/0.1/0.1 as starting point)
- Choose chromem-go persistence mode: `ExportToFile` (single file) vs per-document (avoid per-document at scale)

**Parallelization layers (from go-db-architect's Layer 0-4 model):**
- Layer 0: Types and constants (`internal/memory/types.go`)
- Layer 1: Interfaces (`GraphStore`, `VectorIndex`, `VaultSync`)
- Layer 2: SQLite implementation (schema + CRUD + FTS5 with sync triggers [Δ13])
- Layer 3: chromem-go vector index + atomic.Pointer[map] cache
- Layer 4: Composite scoring engine (RRF v1, pluggable for CC upgrade)

**Success criteria:**
- [ ] SQLite schema created: nodes, edges (bi-temporal), episodes, nodes_fts with sync triggers
- [ ] FTS5 sync triggers fire correctly on INSERT/UPDATE/DELETE
- [ ] atomic.Pointer[map] cache loads core blocks in <5ms
- [ ] atomic.Pointer initialized with empty map (not nil)
- [ ] End-to-end retrieval (query → ranked results) in <50ms at 1K nodes
- [ ] Dual connection pools (readDB N readers + writeDB 1 writer)
- [ ] All sharp edges from yaml validated with test coverage

### Phase 3 — Obsidian Vault + Consolidation (~2 weeks)

Write-only vault export (string construction), batch import (hybrid goldmark parsing), consolidation pipeline with context-enriched entity resolution.

**Prerequisites:**
- [ ] Phase 0 Validation Task 2 (hybrid goldmark coexistence) PASSED
- [ ] Phase 2 complete

**Decision points before starting:**
- Validate vault *write* fidelity with text/template approach (round-trip 50 files through Obsidian) [Δ7]
- Consolidation trigger: session-end only vs. periodic (recommend session-end for v1)
- Entity extraction validation: test on 10 real sessions, measure precision (must be ≥0.8)

**Success criteria:**
- [ ] Vault export produces valid Obsidian-compatible Markdown with wikilinks + frontmatter
- [ ] Vault files written via string construction, not goldmark rendering [Δ7]
- [ ] Hybrid goldmark stack parses wikilinks + hashtags + callouts correctly [Δ6-CORRECTED]
- [ ] Batch import detects human edits via content hash comparison
- [ ] Consolidation completes in <30s for 1K nodes
- [ ] Pre-computed PageRank and community scores accessible as column reads at runtime
- [ ] Entity resolution uses context-enriched embeddings [Δ14]
- [ ] Entity resolution applies Tier 4 default for borderline scores [Δ15]
- [ ] Flagged borderline pairs stored in node.properties

### Phase 4 — Hook Integration + Migration (~1.5 weeks)

Wire into hook chain, migrate existing JSONL memories.

**Prerequisites:**
- [ ] Phase 3 complete

**Decision points before starting:**
- Migration strategy: one-time import vs. gradual dual-write

**Success criteria:**
- [ ] gogent-load-context retrieves memories from knowledge graph within latency budget
- [ ] Existing JSONL memories imported to knowledge graph
- [ ] No regression in existing memory-archivist functionality
- [ ] Session handoff works with new storage backend
- [ ] Total startup overhead (Phase 1+2 load) ≤100ms

---

## 14. Open Questions (Validate During / Before Implementation)

| Question | Importance | When to Answer | Method | Status |
|----------|-----------|----------------|--------|--------|
| Does nomic-embed-text return L2-normalized vectors? | **BLOCKING** | Phase 0 | Embed 10 test strings, compute L2 norms, verify within [0.99, 1.01] | ⏳ Pending |
| Do powerman/goldmark-obsidian and VojtaStruhar callout coexist without AST conflicts? | **BLOCKING** | Phase 0 | Register both, parse 50 vault files | ⏳ Pending |
| What is the actual entity extraction rate per session? | HIGH | Before Phase 3 | Run extraction pipeline on 10 real sessions; project growth rate | ⏳ Pending |
| What % of useful memory retrievals require graph traversal vs. flat search? | HIGH | Before Phase 2 | Classify 20 representative agent queries as flat-retrieval vs. traversal-needed | ⏳ Pending |
| Should the 0.90 cosine threshold be raised to 0.92-0.95? | MEDIUM | After 20 consolidation passes | Audit all auto-merges; raise if false positives appear | ⏳ Deferred |
| Does the pre-compaction flush produce useful memories or noise? | MEDIUM | Phase 3 experiment | Implement as standalone test across 20 sessions; measure NO_REPLY rate | ⏳ Pending |
| What concrete bi-temporal queries would agents actually execute? | MEDIUM | Before Phase 3 query utilities | Articulate 3 scenarios; if none found, defer query utilities | ⏳ Pending |
| Is there a timeline for powerman/goldmark-obsidian to implement callouts? | LOW | Before Phase 3 | Check GitHub issues/milestones | ⏳ Pending |

---

## 15. Risk Register

| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|--------|------------|-------|
| **FTS5 index silently desyncs from nodes table** [Δ13] | ~~High~~ **Eliminated** | High | **Mitigated by design:** AFTER triggers ensure sync at database layer. Test trigger firing in Phase 2 test suite. | Phase 2 |
| **Entity resolution false merges from homonym collision** | Medium | High | Context-enriched embeddings [Δ14] prevent homonym collision. Monitor false-merge rate during first 20 consolidation runs. Raise Tier 1 threshold if needed. | Phase 3 |
| **nomic-embed-text vectors not L2-normalized** | Medium | High | **Blocking validation in Phase 0.** If not normalized, add pre-normalization at storage time. | Phase 0 |
| **Borderline entity pairs accumulate as duplicates** | Medium | Low | Tier 4 default [Δ15] flags for review. Periodic deduplication pass. | Phase 3+ |
| **goldmark extension pipeline conflict** | Low | Medium | **Blocking validation in Phase 0.** goldmark's extension model is designed for composability. | Phase 0 |
| **atomic.Pointer nil dereference at startup** | Low | High | Constructor must initialize with empty map. Add test case that reads before any Store() call. | Phase 2 |
| Entity extraction produces noisy graphs | Medium | High | Validate precision ≥0.8 on 50 sessions before committing to auto-extraction; add human-review gate if needed | Phase 3 |
| chromem-go degrades beyond 50K vectors before anticipated | Medium | Medium | Benchmark at 100K vectors (768d) before committing; if fails, swap to `coder/hnsw` (CC0) or `fogfish/hnsw` (MIT) behind interface | Phase 2 |
| Hook chain latency budget leaves insufficient room for retrieval | Medium | High | Profile first; if >50ms consumed, design async injection after first response | Phase 4 |
| Trigger collision: 'memory system' routes to go-pro instead of go-db-architect | Medium | Medium | auto_activate paths take priority; verify after first 10 invocations; prefix ambiguous triggers | Phase 1 |
| chromem-go single-maintainer risk — project goes unmaintained | Medium | Medium | Interface abstraction allows swap to fogfish/hnsw or coder/hnsw. Monitor repo activity quarterly. | Ongoing |
| chromem-go per-document persistence creates 100K files on disk | Medium | Low | Use `ExportToFile` for single-file persistence instead of per-document mode | Phase 2 |
| MPL-2.0 compliance missed if chromem-go source is modified | Low | Medium | Track in LICENSE-THIRD-PARTY; code review gate for any chromem-go source modifications | Ongoing |

---

## 16. Sharp Edges (v1 Active)

> **v3 update:** Added 3 new sharp edges from Braintrust analysis. Total: ~20 active edges.

**SQLite:**
- `deferred-tx-upgrade`: Deferred TX upgrading to write gets instant SQLITE_BUSY regardless of busy_timeout. Fix: `_txlock=immediate` in DSN or explicit `BEGIN IMMEDIATE`
- `unclosed-rows-wal-growth`: Unclosed `rows` objects prevent WAL checkpointing → unbounded WAL growth. Always `defer rows.Close()`, check `rows.Err()` after iteration
- `modernc-maxopenconns`: `SetMaxOpenConns(>0)` required for concurrent access; without it, deadlock
- `recursive-cte-dense-graph`: Recursive CTEs cause exponential blowup in dense graphs (>10K edges). Use gonum graph algorithms for traversal instead
- `sqlite-vec-modernc-incompatible`: sqlite-vec will NOT work with modernc.org/sqlite; this is why chromem-go is separate
- `fts5-external-content-no-autosync` [Δ13]: FTS5 external content tables (`content=tablename`) do NOT auto-sync. INSERT/UPDATE/DELETE on the content table does not update the FTS5 index. **Requires AFTER triggers or application-level sync.** v3 specifies AFTER triggers.

**Vector search:**
- `embedding-model-mismatch`: Vectors from different models exist in incompatible spaces. Always store `model_version` with every embedding; never mix models in similarity queries
- `chromem-go-100k-cap`: chromem-go uses exhaustive search; beyond 100K vectors (768d), latency degrades to ~80ms+. Benchmark at 100K before committing as sole solution
- `chromem-go-no-update`: chromem-go has no document update operation. Must delete-then-re-add. Wrap in helper function.
- `chromem-mpl2-compliance` [Δ2]: chromem-go is MPL-2.0. Modifications to its source files must be disclosed. Unmodified dependency use requires no disclosure. Track in LICENSE-THIRD-PARTY.

**Vault / Markdown:**
- `yaml-frontmatter-special-chars`: Unquoted colons in YAML values parse as nested mappings. `yes/no/true/false` strings parsed as booleans. Always quote values with special characters
- `goldmark-no-roundtrip` [Δ7]: goldmark is a Markdown→HTML converter. Its AST discards formatting. **Never use goldmark to write vault files.** Use string construction for output, goldmark for parsing only.
- `goldmark-obsidian-incomplete` [Δ6-CORRECTED]: powerman/goldmark-obsidian covers 7/10 features (as of Feb 2026). Callouts, highlights, and comments are "Not Yet Implemented". **Requires hybrid extension stack with VojtaStruhar callout library.**
- `goldmark-footnotes-reference-only`: powerman/goldmark-obsidian supports reference-style footnotes only, not inline footnotes `^[like this]`.

**Entity extraction:**
- `extraction-noise-accumulation`: Low-confidence entity extraction accumulates noise faster than signal; degrades retrieval. Reject extractions below 0.8 confidence threshold
- `entity-resolution-threshold`: Embedding cosine similarity is the primary signal (handles abbreviations). Jaro-Winkler is secondary. Use tiered OR-logic (§9), not a single blended threshold. [Δ8]
- `entity-resolution-homonyms` [Δ14]: Name-only embeddings produce false merges for homonyms (e.g., "Einstein" agent vs physicist). **Use context-enriched embeddings (type::name::description).**

**chromem-go / embeddings:**
- `ollama-not-normalized`: Not all Ollama models return L2-normalized vectors. Verify `nomic-embed-text` normalization or pre-normalize at storage time (reduces cosine sim to dot product). **Blocking validation task in Phase 0.**

**IR scoring:**
- `bm25-feedback-loop`: Access-frequency boosting creates feedback loops (popular memories get retrieved more → become more popular). Bootstrap new memories with `access_count=1` minimum to prevent cold-start disadvantage
- `scoring-weight-sensitivity`: α/β/γ/δ weights dramatically affect retrieval quality. Start with 0.3/0.5/0.1/0.1 and tune empirically using actual retrieval utility measurements

**Cache:**
- `atomic-pointer-nil-init` [Δ5]: `atomic.Pointer[map]` starts as nil. Always check `cache.Load() != nil` before accessing, or initialise with empty map at construction time. **Constructor must initialize with empty map.**

---

## 17. Integration with GOgent-Fortress Routing

**Trigger patterns for go-db-architect (unambiguous, no collision):**
- "graph schema", "knowledge graph", "temporal graph" — unambiguous
- "memory graph system", "memory subsystem schema" — prefixed to avoid memory-archivist collision
- "graph store", "consolidation pipeline", "entity extraction", "vector index", "embedding store"

**Auto-activate paths (resolve trigger ambiguity via path priority):**
```
internal/memory/**
internal/graphstore/**
internal/vectorindex/**
internal/vault/**
internal/consolidation/**
```

**Trigger collision monitoring:** Check routing logs after first 10 invocations. Watch for `go-pro` handling requests that should route to `go-db-architect`.

---

## 18. Connection Pool Architecture

```go
// Write pool: 1 connection, serializes all writes (SQLite WAL requirement)
writeDB, _ := sql.Open("sqlite", dsn + "?_txlock=immediate")
writeDB.SetMaxOpenConns(1)
writeDB.SetMaxIdleConns(1)

// Read pool: N connections for concurrent reads
readDB, _ := sql.Open("sqlite", dsn + "?mode=ro")
readDB.SetMaxOpenConns(runtime.GOMAXPROCS(0))
readDB.SetMaxIdleConns(runtime.GOMAXPROCS(0))
```

This mirrors SQLite's fundamental concurrency model: **one writer, many concurrent readers**.

---

## 19. Package Structure

```
internal/
├── memory/
│   ├── types.go              -- Node, Edge, Episode, CoreBlock types
│   ├── store.go              -- GraphStore interface
│   ├── sqlite.go             -- SQLite implementation
│   ├── cache.go              -- atomic.Pointer[map] core block cache  [Δ5]
│   └── scoring.go            -- IR-based composite scoring (RRF v1, CC-ready)  [Δ9]
├── graphstore/
│   ├── schema.go             -- Migration runner, DDL, FTS5 sync triggers  [Δ13]
│   ├── crud.go               -- Node/edge CRUD
│   └── query.go              -- FTS5 + vector hybrid retrieval
├── vectorindex/
│   ├── index.go              -- VectorIndex interface
│   ├── chromem.go            -- chromem-go implementation
│   └── hnsw.go               -- coder/hnsw or fogfish/hnsw fallback  [Δ3]
├── vault/
│   ├── export.go             -- SQLite → Markdown (string construction)  [Δ7]
│   ├── import.go             -- Batch import: content hash diffing
│   ├── parse.go              -- Hybrid goldmark parsing  [Δ6-CORRECTED]
│   └── templates.go          -- text/template vault file templates  [Δ7]
├── consolidation/
│   ├── pipeline.go           -- Session-end consolidation orchestration
│   ├── extract.go            -- Entity extraction via Claude Sonnet
│   ├── resolve.go            -- Entity resolution (context-enriched, embedding-first)  [Δ8, Δ14, Δ15]
│   └── algorithms.go         -- PageRank, Louvain, label propagation via gonum
└── hooks/
    └── loader.go             -- gogent-load-context integration
```

---

## 20. Interface Contracts

### 20.1 GraphStore Interface

```go
type GraphStore interface {
    // Node operations
    CreateNode(ctx context.Context, node *Node) (int64, error)
    GetNode(ctx context.Context, id int64) (*Node, error)
    UpdateNode(ctx context.Context, node *Node) error
    DeleteNode(ctx context.Context, id int64) error

    // Edge operations (bi-temporal aware)
    CreateEdge(ctx context.Context, edge *Edge) (int64, error)
    GetCurrentEdges(ctx context.Context, nodeID int64, relationType string) ([]*Edge, error)
    InvalidateEdge(ctx context.Context, id int64, validTo time.Time) error

    // Episode operations (append-only)
    AppendEpisode(ctx context.Context, episode *Episode) (int64, error)
    GetEpisodesBySession(ctx context.Context, sessionID string) ([]*Episode, error)

    // Search operations
    SearchFTS(ctx context.Context, query string, limit int) ([]*Node, error)
    SearchHybrid(ctx context.Context, query string, embedding []float32, weights HybridWeights, limit int) ([]*ScoredNode, error)

    // Core block operations
    GetCoreBlocks(ctx context.Context) ([]*Node, error)

    // Consolidation operations
    UpdatePrecomputedFeatures(ctx context.Context, nodeID int64, pagerank float64, community int) error
    GetNodesForConsolidation(ctx context.Context, since time.Time) ([]*Node, error)
}
```

### 20.2 VectorIndex Interface

```go
type VectorIndex interface {
    // Core operations
    Add(ctx context.Context, id string, embedding []float32, metadata map[string]any) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query []float32, k int) ([]SearchResult, error)

    // Persistence
    Persist(ctx context.Context, path string) error
    Load(ctx context.Context, path string) error

    // Metrics
    Count() int
    Dimensions() int
}

type SearchResult struct {
    ID         string
    Distance   float32  // Lower is more similar for L2; higher for cosine
    Metadata   map[string]any
}
```

### 20.3 VaultSync Interface

```go
type VaultSync interface {
    // Export (agent → vault)
    ExportNode(ctx context.Context, node *Node) error
    ExportEpisodeSummary(ctx context.Context, date time.Time, episodes []*Episode) error
    ExportMemoryBrief(ctx context.Context, topNodes []*Node) error

    // Import (vault → agent, on startup)
    DetectChanges(ctx context.Context) ([]ChangedFile, error)
    ImportChanges(ctx context.Context, changes []ChangedFile) error

    // Utilities
    ComputeContentHash(content []byte) string
}

type ChangedFile struct {
    Path        string
    ContentHash string
    ChangeType  string  // "modified", "created", "deleted"
}
```

---

## Appendix A: Decisions Made (Do Not Revisit Without Evidence)

| Decision | Rationale | Evidence required to change |
|----------|-----------|---------------------------|
| Drop bbolt | 5ms→0.5ms imperceptible; atomic.Pointer[map] equivalent; eliminates consistency boundary [Δ5] | Profiling showing SQLite reads are actual bottleneck in hook path |
| Write-only vault v1 | Bidirectional sync complexity > value for rare use case | Multiple users reporting frustration with between-session edit model |
| Consolidation-only graph algorithms | 50-500ms runtime graph compute violates latency budget | New agent query patterns that demonstrably require runtime traversal |
| IR scoring over FSRS | Agents don't forget; retrieval doesn't strengthen traces; no spacing effect | Empirical data showing FSRS predictions correlate with actual useful memory retrieval |
| Keep bi-temporal columns | Retrofitting is expensive; 4 columns are cheap | N/A — columns are already in schema |
| coder/hnsw as approved fallback | CC0-1.0 license verified [Δ1]. chromem-go remains primary (exhaustive search sufficient for ≤100K). Fallback to coder/hnsw or fogfish/hnsw behind VectorIndex interface at >100K. | chromem-go benchmark succeeds at 100K AND no latency improvement needed |
| goldmark for parsing only, string construction for writing [Δ7] | goldmark AST discards Obsidian formatting; round-trip is lossy. Information-theoretically proven (Einstein). | goldmark adds a Markdown renderer that preserves Obsidian syntax perfectly |
| Hybrid goldmark extension stack [Δ6-CORRECTED] | powerman covers 7/10 features. VojtaStruhar required for callouts. Consolidation was a regression from v1. | powerman releases update implementing callouts, highlights, and comments |
| Embedding-first entity resolution [Δ8] | String metrics score near zero on abbreviations/synonyms; embeddings capture semantic equivalence | Empirical data showing Jaro-Winkler-first produces fewer false negatives than embedding-first |
| Context-enriched entity embeddings [Δ14] | Name-only embeddings produce homonym collisions. type::name::description provides discrimination. | Empirical data showing name-only produces acceptable false-merge rate |
| Tier 4 borderline disposition [Δ15] | Prevents unbounded duplicate accumulation from scores in ambiguous range. | Alternative deduplication strategy that handles borderline cases better |
| RRF k=60 for v1, CC as upgrade [Δ9] | RRF is battle-tested and requires no score normalisation; CC upgrade when eval data exists | CC provides measurably better retrieval quality on first evaluation run |
| FTS5 external content with triggers [Δ13] | SQLite-canonical approach. Ensures sync at database layer. Easier to test than application-level sync. | Trigger-based sync proves unreliable or creates performance issues |

---

## Appendix B: Glossary

| Term | Definition |
|------|-----------|
| **BM25** | Okapi BM25: probabilistic relevance scoring function for full-text search (successor to TF-IDF) |
| **RRF** | Reciprocal Rank Fusion: merges ranked lists from different retrieval methods using `1/(k + rank)` formula |
| **CC** | Convex Combination: weighted linear blend of normalised scores from different retrievers. More expressive than RRF. |
| **FTS5** | Full-Text Search version 5: SQLite's built-in full-text search extension with BM25 ranking |
| **WAL** | Write-Ahead Logging: SQLite journal mode enabling concurrent reads during writes |
| **CTE** | Common Table Expression: SQL `WITH` clause enabling recursive graph traversal in SQL |
| **Bi-temporal** | Two time dimensions: valid_time (when fact is true in reality) + tx_time (when system recorded it) |
| **PageRank** | Graph centrality algorithm: nodes linked from many nodes score higher |
| **Louvain** | Community detection algorithm: groups nodes by edge density |
| **HNSW** | Hierarchical Navigable Small World: approximate nearest neighbor (ANN) graph structure for fast vector search |
| **Jaro-Winkler** | String similarity metric optimized for short strings and names (0-1 scale) |
| **chromem-go** | Pure Go in-process embedding store with exhaustive nearest-neighbor search (MPL-2.0) |
| **nomic-embed-text** | Ollama embedding model: 768 dimensions, 8K context, local operation |
| **MPL-2.0** | Mozilla Public License 2.0: weak copyleft — modified source files must be disclosed, but unmodified dependency use in proprietary binaries is permitted |
| **CC0-1.0** | Creative Commons Zero: public domain dedication with no restrictions |
| **atomic.Pointer** | Go stdlib type providing lock-free atomic load/store of typed pointers. Used with copy-on-write for read-heavy caches. |
| **Context-enriched embedding** | Embedding generated from `type::name::description` string rather than bare name. Improves discrimination for homonyms. [Δ14] |
| **External content table** | FTS5 table that references content from another table rather than storing its own copy. Requires sync triggers. [Δ13] |

---

## Appendix C: Braintrust Analysis Reference

This scope document incorporates findings from Braintrust analysis session `braintrust-1739980800`:

- **Einstein (theoretical):** Verified 9/12 deltas, identified 3 missing corrections (FTS5 sync, embedding format, Δ6 regression), proposed novel approaches (hybrid extension stack, context-enriched embeddings, contentless FTS5)
- **Staff-Architect (practical):** Independent license verification, 7-layer review with 1 major + 4 minor issues, concrete sign-off conditions
- **Beethoven (synthesis):** Resolved 4 divergences, produced 7 convergence points, no unresolved tensions

Full analysis available at: `tickets/obsidian-cli-knowledge-graph/braintrust-scope-v2-analysis.json`

---

## Appendix D: Implementation Checklist

### Phase 0 Checklist (Pre-Implementation Validation)
- [ ] nomic-embed-text L2 normalization verified
- [ ] Hybrid goldmark extension coexistence verified
- [ ] viterin/vek end-to-end benchmark completed (non-blocking)

### Phase 1 Checklist (Deployment Setup)
- [ ] Agent files moved to `~/.claude/agents/go-db-architect/`
- [ ] Agent definition split: identity ≤400 lines, conventions ≤400 lines
- [ ] agents-index.json entry created
- [ ] Routing validation passed
- [ ] db-conventions.md includes FTS5 sync triggers

### Phase 2 Checklist (Core Storage Layer)
- [ ] SQLite schema with FTS5 sync triggers deployed
- [ ] FTS5 trigger tests pass
- [ ] atomic.Pointer cache initialized with empty map
- [ ] Cache loads core blocks in <5ms
- [ ] Dual connection pools configured
- [ ] Hybrid retrieval in <50ms at 1K nodes
- [ ] Sharp edge tests complete

### Phase 3 Checklist (Vault + Consolidation)
- [ ] Vault export via string construction
- [ ] Hybrid goldmark parsing for import
- [ ] Content hash change detection
- [ ] Context-enriched entity embeddings [Δ14]
- [ ] Tiered entity resolution with Tier 4 [Δ15]
- [ ] Consolidation <30s at 1K nodes
- [ ] PageRank/community pre-computation

### Phase 4 Checklist (Hook Integration)
- [ ] gogent-load-context integration
- [ ] JSONL migration complete
- [ ] Total startup overhead ≤100ms
- [ ] No memory-archivist regression
