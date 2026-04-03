# GOgent-Fortress Embedded Knowledge Graph — Unified Scope (v4)

> **Status:** APPROVED FOR IMPLEMENTATION — All blockers resolved, validation tasks defined
> **Source:** SCOPE-v3 + Braintrust Analysis (2026-03-31: Einstein + Staff-Architect + Beethoven synthesis)
> **Date:** 2026-03-31 (v4)
> **Use:** Input to `/plan-tickets` and `/implement`

---

## v4 Changelog

> Braintrust analysis (2026-03-31) produced 1 critical, 4 major, and 5 minor findings. The `.claude/` → `.gogent/` runtime I/O migration (commit 6bdefe79, 2026-03-31, 23 tickets) post-dates v3 (2026-02-19) by 40 days and is the primary invalidation event.

| # | Section | Change | Severity | Source |
|---|---------|--------|----------|--------|
| Δ18 | §3, §10, §19 | **agents-index.json entry expanded to v2.6.0 schema.** Added parallelization_template, path, tools, cli_flags, spawned_by, can_spawn, description, inputs, outputs. Original entry had 10 fields; v2.6.0 requires 20+. | Critical | Einstein + Staff-Architect (convergence) |
| Δ19 | §10 | **Vault DB path renamed.** `.gogent-vault/.gogent/` → `.gogent/graphdb/`. Eliminates naming collision with runtime `.gogent/` directory created by MIG migration. DB files are runtime artifacts and belong under `.gogent/`. Kept directly under `.gogent/` (not `.gogent/memory/`) to avoid confusion with the JSONL-based `.gogent/memory/` directory during the transition period. | Major | Einstein + Staff-Architect (convergence) |
| Δ20 | §11, §19 | **Hook integration architecture specified.** Extend `cmd/gogent-load-context/` to import `internal/hooks/loader.go`. No new binary — single integration point, no binary proliferation. `settings.json` registration unchanged. | Major | Staff-Architect (M-1), reconciled with Option B defaults |
| Δ21 | §3 | **Bidirectional spawn relationships added.** `spawned_by: ['router', 'orchestrator', 'impl-manager']` in go-db-architect entry. Added to `impl-manager.can_spawn` and `orchestrator.can_spawn`. | Major | Staff-Architect (M-3) |
| Δ22 | §3, §4, §12 | **Path audit applied.** Agent mv command uses repo-relative paths. JSONL migration references updated to `.gogent/memory/`. All `.claude/` runtime I/O paths corrected to `.gogent/` where appropriate. Config paths (`.claude/agents/`, `.claude/conventions/`) unchanged. | Major | Einstein + Staff-Architect (convergence) |
| Δ23 | §13 | **Phase 0 expanded.** Added Validation Task 4: SQLite cold-start latency benchmark. Hook latency measurements are 40 days old and pre-migration. | Minor | Einstein + Staff-Architect (convergence) |
| Δ24 | §13 | **go.mod dependency commands added.** Explicit `go get` commands in Phase 2 prerequisites. | Minor | Staff-Architect (m-1) |
| Δ25 | §19 | **Test file structure added.** `_test.go` counterparts specified in package layout. | Minor | Staff-Architect (m-2) |
| Δ26 | §13 | **Phase 4 rollback strategy added.** `GOGENT_MEMORY_BACKEND` env var (`jsonl` default, `graph` opt-in) checked by `gogent-load-context`. Allows instant rollback without binary swap or settings.json changes. JSONL originals preserved as `.jsonl.bak`. | Minor | Staff-Architect (m-5), reconciled with Option B defaults |
| Δ27 | §15 | **New risks added.** Hook interference from new binaries, MCP convention loading, incomplete migration paths. | Minor | Einstein (assumptions surfaced) |

### Cumulative Delta Summary (v1 → v4)

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
| Δ13 (FTS5 sync) | ✅ New in v3 | High |
| Δ14 (embedding format) | ✅ New in v3 | High |
| Δ15 (Tier 4 default) | ✅ New in v3 | High |
| Δ16 (validation tasks) | ✅ New in v3, **expanded in v4** (Δ23) | High |
| Δ17 (dual-authority model) | ✅ New in v3.1 | High |
| Δ18 (agents-index v2.6.0) | 🆕 **New in v4** | High |
| Δ19 (vault DB path fix) | 🆕 **New in v4** | High |
| Δ20 (hook integration arch) | 🆕 **New in v4** | High |
| Δ21 (spawn relationships) | 🆕 **New in v4** | High |
| Δ22 (path audit) | 🆕 **New in v4** | High |
| Δ23 (SQLite cold-start bench) | 🆕 **New in v4** | High |
| Δ24 (go.mod deps) | 🆕 **New in v4** | High |
| Δ25 (test file structure) | 🆕 **New in v4** | High |
| Δ26 (Phase 4 rollback) | 🆕 **New in v4** | High |
| Δ27 (new risks) | 🆕 **New in v4** | High |

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

> **[Δ23] v4 note:** These measurements are from 2026-02-19, pre-migration. The `.gogent/` migration (2026-03-31) touched 23 files. **Re-verification required in Phase 0** (Validation Task 4).

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
│  (triggered by gogent-load-context, graph backend)                  │  [Δ20]
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

> **v4 status:** All blockers from v1/v2/v3 have been resolved. Implementation may proceed.

### ~~BLOCKER C-1: AGPL License Contradiction~~ — RESOLVED (v2)

> **[Δ1]** `github.com/coder/hnsw` is licensed CC0-1.0 (public domain dedication), not AGPL-3.0. Verified against upstream LICENSE file. Now approved as fallback dependency.

### ~~BLOCKER M-1: No agents-index.json Entry~~ — RESOLUTION DEFINED [Δ18]

**Problem:** `go-db-architect` has no entry in `agents-index.json`.

**Resolution:** Create full agents-index.json entry per v2.6.0 schema. [Δ18] The v3 proposed entry was missing 8+ required fields (parallelization_template, path, tools, cli_flags, spawned_by, can_spawn, description, inputs, outputs).

```json
{
  "id": "go-db-architect",
  "parallelization_template": "B",
  "name": "Go DB Architect",
  "model": "sonnet",
  "thinking": true,
  "thinking_budget": 14000,
  "tier": 2,
  "category": "language",
  "path": "go-db-architect",
  "triggers": ["graph schema", "knowledge graph", "memory graph", "memory subsystem schema",
                "temporal graph", "db architect", "graph store", "consolidation pipeline",
                "entity extraction", "vector index", "embedding store"],
  "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
  "cli_flags": {
    "allowed_tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
    "additional_flags": ["--permission-mode", "delegate"]
  },
  "auto_activate": {
    "paths": ["internal/memory/**", "internal/graphstore/**", "internal/vectorindex/**",
              "internal/vault/**", "internal/consolidation/**"]
  },
  "inputs": [
    ".gogent/graphdb/graph.db",
    ".gogent-vault/entities/"
  ],
  "outputs": [
    ".gogent/graphdb/graph.db",
    ".gogent-vault/"
  ],
  "spawned_by": ["router", "orchestrator", "impl-manager"],
  "can_spawn": [],
  "description": "Specialist for the embedded knowledge graph subsystem. Handles SQLite graph store, chromem-go vector index, Obsidian vault sync, and consolidation pipeline.",
  "context_requirements": {
    "rules": ["agent-guidelines.md"],
    "conventions": {
      "base": ["go.md", "db-conventions.md"]
    }
  },
  "sharp_edges_count": 20
}
```

Also add `go-db-architect` to `impl-manager.can_spawn` and `orchestrator.can_spawn`. [Δ21]

**Effort:** 60-90 minutes including validation testing. (v3 estimated 30-60 min; corrected upward per Braintrust finding.)
**Owner:** Phase 1 implementer.

### ~~BLOCKER M-3: Wrong File Location~~ — RESOLUTION DEFINED [Δ22]

**Problem:** Agent definition lives in `tickets/obsidian-cli-knowledge-graph/`.

**Resolution:** [Δ22] Use repo-relative paths (not absolute `~/.claude/` paths that depend on symlink):
```bash
mkdir -p .claude/agents/go-db-architect/
mv tickets/obsidian-cli-knowledge-graph/go-db-architect.md .claude/agents/go-db-architect/go-db-architect.md
mv tickets/obsidian-cli-knowledge-graph/go-db-architect-sharp-edges.yaml .claude/agents/go-db-architect/sharp-edges.yaml
```

**Effort:** 5 minutes.
**Owner:** Phase 1 implementer.

---

## 4. Agent Definition Refactoring (Pre-Implementation)

> This must be done before registering in agents-index.json

**Problem:** The current `go-db-architect.md` is 882 lines — 2.4x the peer average (go-pro: 374, go-tui: 471). It contains complete function implementations that inflate every invocation's context budget unnecessarily.

**Resolution:** Split into two files following the go-tui pattern:

**`.claude/agents/go-db-architect/go-db-architect.md` (~350 lines)**
- Role description and system constraints
- Approved dependencies (with license annotations — corrected per Δ1, Δ2, Δ6-CORRECTED)
- Focus area rules as CORRECT/WRONG patterns (not full implementations)
- Interface contracts section (GraphStore, VectorIndex, VaultSync interfaces)
- Testing strategy overview
- Output requirements and parallelization model

**`.claude/conventions/db-conventions.md` (~350 lines)**
- Full SQL schema with all DDL including FTS5 sync triggers [Δ13]
- Complete function implementations (decayScore, compositeScore, RRF, RunMigrations)
- Algorithm details (entity resolution with context-enriched embeddings [Δ14], label propagation)
- Template strings and query patterns
- Vault file structure and hybrid goldmark extension stack [Δ6-CORRECTED]

**Wire via** `context_requirements.conventions.base: ["go.md", "db-conventions.md"]` in agents-index.json. [Δ18]

> **[Δ22] v4 note:** The go-db-architect.md agent spec itself needs updating for the `.gogent/` migration — file paths within it may reference `.claude/memory/` instead of `.gogent/memory/`.

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

> **Critical correction from Braintrust analysis (v3):** powerman/goldmark-obsidian covers 7/10 required features, NOT 10/10. Callouts, highlights, and comments are listed as "Not Yet Implemented" in upstream (as of Feb 2026).

| Extension | License | Features | Status |
|-----------|---------|----------|--------|
| `github.com/powerman/goldmark-obsidian` | MIT | Wikilinks (aliases, fragments, embeds), hashtags (ObsidianVariant), block IDs, YAML frontmatter, footnotes (reference-style only), LaTeX, Mermaid | ✅ Primary |
| `github.com/VojtaStruhar/goldmark-obsidian-callout` | MIT | `> [!note]` callout blocks (all Obsidian callout types, collapsible, nested) | ✅ **Retained** — required for callout parsing |
| Custom parser (~30 lines) | — | `==highlights==` | 🔨 Build if needed |
| Custom parser (~30 lines) | — | `%%comments%%` | 🔨 Build if needed |

**Validation task (blocking Phase 3):** Register both extensions in goldmark pipeline and parse 50 representative vault files containing callouts + wikilinks + hashtags. Verify no AST node conflicts.

> **[Δ2] MPL-2.0 compliance note:** chromem-go's MPL-2.0 license permits use in proprietary binaries without modification. If GOgent-Fortress modifies chromem-go source files, those modifications must be disclosed under MPL-2.0. Track in LICENSE-THIRD-PARTY file.

**SQLite PRAGMA configuration:**
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
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
CREATE INDEX idx_edges_source ON edges(source_id, relation_type);
CREATE INDEX idx_edges_target ON edges(target_id, relation_type);
CREATE INDEX idx_edges_current ON edges(source_id, relation_type, tx_to, valid_to);
CREATE INDEX idx_episodes_session ON episodes(session_id, occurred_at);
CREATE INDEX idx_nodes_type ON nodes(type);
```

### 6.3 Bi-Temporal Note
The 4 bi-temporal columns on `edges` are **retained but no query utilities built until concrete examples emerge**. Cost of adding 4 columns: minutes. Cost of retrofitting bi-temporal later: days. After 3 months of operation, if no bi-temporal query has been articulated, columns are inert schema — ignore and proceed.

### 6.4 FTS5 Design Decision [Δ13]

> **Braintrust finding (v3):** v1 and v2 both specified `content=nodes` (external content table) but omitted sync triggers.

**Decision:** Use external content table WITH AFTER triggers (the SQLite-canonical approach).

**Rationale:**
- External content saves storage (no content duplication)
- AFTER triggers ensure sync at the database layer (not application layer)
- Easier to test correctness (triggers fire automatically)
- SQLite-canonical pattern with extensive documentation

---

## 7. Retrieval Scoring: IR-Based (Replacing FSRS/Ebbinghaus)

**Composite relevance score:**
```
Score = α × BM25(query, memory_name_properties)
      + β × cosine_sim(query_embedding, memory_embedding)
      + γ × log(1 + access_count) / log(1 + age_days)   -- frequency boost
      + δ × pre_computed_pagerank                         -- graph centrality
```

**Initial weights (tune empirically):** α=0.3, β=0.5, γ=0.1, δ=0.1

**Time decay:** Simple logarithmic decay — tunable to empirical retrieval utility data.

**Reciprocal Rank Fusion (k=60)** for merging BM25 and vector search result sets before final ranking.

> **[Δ9] Upgrade path: Convex Combination (CC).** CC outperforms RRF (Bruch et al., ACM TOIS 2022). CC formula: `score(d) = α·norm(score_vector) + (1-α)·norm(score_bm25)` with α≈0.7. **Default remains RRF k=60 for v1** — graduate to CC once evaluation data exists. The scoring interface should accept pluggable fusion strategies from the start.

---

## 8. Three-Phase Startup Loading

**Phase 1 — Core blocks (<10ms, from atomic.Pointer[map]):** [Δ5]
Always injected into system prompt. Budget: ~10-15% context window (400-600 tokens). On startup: load all `type='core_block'` nodes from SQLite → populate map, store via `atomic.Pointer.Store()`. Reads: `cache.Load()` — zero contention, no locks.

**Constructor must initialize atomic.Pointer with empty map** to prevent nil dereference before first SQLite load completes.

**Phase 2 — Session-relevant retrieval (<50ms, from SQLite + chromem-go):**
Query using hook event metadata as context. Hybrid retrieval: BM25 (30%) + embedding cosine similarity (70%) via FTS5 + chromem-go. Return top-k results. Budget: ~10-20% context window.

**Phase 3 — Lazy on-demand (<100ms per query):**
Expose `memory_search` and `memory_get` tools to the agent.

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

    Tier 1 (semantic match):  cosine_sim(embed_a, embed_b) ≥ 0.90 → MERGE
    Tier 2 (fuzzy + semantic): jaro_winkler(a, b) ≥ 0.85
                            AND cosine_sim(embed_a, embed_b) ≥ 0.70 → MERGE
    Tier 3 (auto-reject):    both signals < 0.60 → DISTINCT ENTITIES
    Tier 4 (borderline) [Δ15]: (cosine 0.60-0.89) AND (jaro_winkler < 0.85)
                            → DISTINCT + flag for periodic review

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

> **Einstein recommendation:** After first 20 consolidation passes, audit all auto-merges at Tier 1 (0.90 threshold). If false positives appear, raise threshold to 0.92-0.95.

---

## 10. Obsidian Vault: Write-Only Export + Batch Import

> **Decision:** v1 is write-only from agent. No fsnotify, no real-time sync.

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

**Vault structure:** [Δ19]
```
.gogent-vault/                         ← Obsidian vault root (human-browsable)
├── entities/
│   ├── agents/
│   ├── concepts/
│   ├── decisions/
│   └── errors/
├── episodes/
├── procedures/
├── communities/
└── MEMORY.md                          ← auto-synthesized brief, loaded at startup

.gogent/graphdb/                       ← Runtime data (agent-only, gitignored) [Δ19]
├── graph.db                           ← SQLite database
└── index/                             ← chromem-go persistence
```

> **[Δ19] v4 change:** Database files moved from `.gogent-vault/.gogent/` to `.gogent/graphdb/`. The `.gogent/` migration (commit 6bdefe79) established `.gogent/` as the runtime I/O directory. Nesting `.gogent/` inside `.gogent-vault/` created a naming collision. Database files are runtime artifacts and belong under `.gogent/`, not in the Obsidian vault. Placed at `.gogent/graphdb/` (not `.gogent/memory/graphdb/`) to avoid confusion with the JSONL-based `.gogent/memory/` directory during the transition period.

### Markdown Strategy [Δ6-CORRECTED, Δ7]

> **Critical: goldmark cannot round-trip Obsidian markdown.**

**For parsing (vault import / human edit detection):**
Use **hybrid extension stack** [Δ6-CORRECTED]:
1. `powerman/goldmark-obsidian` — wikilinks, embeds, hashtags, block IDs, YAML frontmatter, footnotes, LaTeX, Mermaid
2. `VojtaStruhar/goldmark-obsidian-callout` — callouts
3. Custom parsers if needed for `==highlights==` and `%%comments%%`

**For writing (vault export from consolidation):**
Construct Markdown directly as text strings via Go `text/template` or `fmt.Sprintf`. Do not parse→transform→render through goldmark's AST.

**Known Obsidian syntax gaps (no goldmark extension covers these):**
- Document aliases defined in YAML `aliases:` property
- Dataview plugin inline fields (`key:: value`)
- Tags defined in YAML frontmatter `tags:` (vs inline `#tags`)
- Footnotes: powerman supports reference-style only, not inline footnotes

---

## 11. Hook Integration [Δ20]

> **[Δ20] v4 change:** Hook integration architecture now fully specified. v3 left this as a Phase 4 decision point.

**Integration approach:** Extend `cmd/gogent-load-context/` to import `internal/hooks/loader.go`. No new binary.

**Why extend gogent-load-context (not a new binary):**
- Avoids binary proliferation — the hook chain already has 5 binaries
- `gogent-load-context` already has the session context (conventions, git state) needed for memory retrieval queries
- Single integration point: memory retrieval runs after convention loading within the same process, eliminating IPC overhead
- Rollback via `GOGENT_MEMORY_BACKEND` env var (default: `jsonl`) — when set to `jsonl`, the graph codepath is skipped entirely, preserving the original 28ms baseline [Δ26]
- The 72ms available budget (100ms − 28ms baseline) is sufficient for SQLite + chromem-go retrieval (<50ms target)

**Hook registration in `settings.json` (UNCHANGED):**
```json
{
  "hooks": {
    "SessionStart": [
      {
        "command": "bin/gogent-load-context",
        "timeout": 100
      }
    ]
  }
}
```

> **Note:** Timeout increased from 50ms to 100ms to accommodate the memory retrieval phase. When `GOGENT_MEMORY_BACKEND=jsonl` (default), the hook exits at ~28ms as before.

**Memory retrieval phase (inside gogent-load-context):**
1. Check `GOGENT_MEMORY_BACKEND` env var — if `jsonl` (default), skip graph path entirely [Δ26]
2. Open SQLite at `.gogent/graphdb/graph.db` (read-only)
3. Load core blocks → inject as `additionalContext`
4. Query session-relevant memories using session context already available in-process
5. Return top-k results as `additionalContext`
6. Total execution budget: <50ms for graph retrieval (within remaining 72ms after baseline work)

**Latency budget allocation (revised):**
- `gogent-load-context` baseline (conventions, git context): 28ms
- Memory retrieval phase (SQLite + chromem-go): <50ms target
- Total SessionStart: <80ms typical, <100ms worst case
- Headroom: sufficient

> **[Δ23] Note:** These latency targets must be re-verified in Phase 0 (Validation Task 4) against the post-migration codebase.

---

## 12. Memory-Archivist Relationship

**Working model:**
- During transition: dual-write to both systems
- Post-migration: memory-archivist writes episodes to SQLite instead of JSONL
- `go-db-architect` is NOT in memory-archivist's `can_spawn` list

**Migration path:** [Δ22]
- Session scope JSONL (`.gogent/memory/handoffs.jsonl`) → episodes table
- Project scope JSONL (`.gogent/memory/decisions/`) → semantic entity nodes
- Global scope JSONL (`.gogent/memory/sharp-edges/`) → procedural memory nodes
- ML scope JSONL (`$XDG_DATA_HOME/gogent/`) → telemetry subgraph (new fourth category)

> **[Δ22] v4 change:** JSONL paths updated from `.claude/memory/` to `.gogent/memory/` per the runtime I/O migration. The `pkg/session/handoff.go` HandoffPath already uses `.gogent/memory/handoffs.jsonl`.

---

## 13. Implementation Phases

### Phase 0 — Pre-Implementation Validation (~1 day) [Δ16, Δ23]

> **BLOCKING:** These validations must pass before Phase 2 implementation begins.

**Validation Task 1: nomic-embed-text L2 Normalization**
- Method: Embed 10 test strings via Ollama nomic-embed-text, compute L2 norms
- Success: All norms within [0.99, 1.01]
- Failure action: Add pre-normalization at storage time (~0.1ms per vector)
- Owner: Phase 2 implementer
- Effort: ~30 minutes

**Validation Task 2: Hybrid goldmark Extension Coexistence**
- Method: Register powerman/goldmark-obsidian + VojtaStruhar callout extension in goldmark pipeline, parse 50 vault files
- Success: No AST node conflicts, all features parsed correctly
- Failure action: Investigate conflict, potentially fork or patch one extension
- Owner: Phase 3 implementer
- Effort: ~2 hours

**Validation Task 3: viterin/vek End-to-End Benchmark** (non-blocking)
- Method: Benchmark chromem-go with and without viterin/vek at 768d, 10K and 100K vectors
- Measure: End-to-end query latency, not just kernel speedup
- Owner: Phase 2 implementer
- Effort: ~2 hours

**Validation Task 4: SQLite Cold-Start Latency** [Δ23] (BLOCKING)
- Method: Open `.gogent/graphdb/graph.db` (empty → 1K nodes), execute first FTS5 query, measure total latency from `sql.Open()` to result
- Success: Cold-start query < 50ms on post-migration codebase
- Failure action: If > 50ms, design async injection or pre-warm strategy
- Owner: Phase 2 implementer
- Effort: ~30 minutes

**Validation Task 5: Hook Chain Interference Audit** [Δ27] (non-blocking)
- Method: Read `settings.json` to identify all active hooks. Cross-reference with go-db-architect's expected operations (SQLite writes, consolidation subprocess)
- Assess: Will `gogent-permission-gate` block SQLite file access? Will `gogent-orchestrator-guard` interfere with consolidation?
- Owner: Phase 1 implementer
- Effort: ~1 hour

### Phase 1 — Deployment Setup (~3 hours) [Δ18, Δ22, Δ24]

**Must complete before any implementation code is written.**

1. Resolve M-3: Move files to `.claude/agents/go-db-architect/` (repo-relative) [Δ22]
2. Refactor agent definition: split into go-db-architect.md (~350 lines) + db-conventions.md (~350 lines)
3. Update with v3/v4 corrections:
   - Hybrid goldmark extension stack [Δ6-CORRECTED]
   - FTS5 sync triggers [Δ13]
   - Context-enriched entity embeddings [Δ14]
   - Tier 4 borderline disposition [Δ15]
   - Qualified SIMD claim [Δ11-QUALIFIED]
   - Vault DB path `.gogent/graphdb/` [Δ19]
   - Hook integration via `gogent-load-context` extension [Δ20]
4. Resolve M-1: Create agents-index.json entry (full v2.6.0 schema) [Δ18] + add to can_spawn lists [Δ21]
5. Validate routing: `gogent-validate routes to go-db-architect for "graph schema" trigger`

**Success criteria:**
- [ ] `go-db-architect` in agents-index.json with all v2.6.0 required fields [Δ18]
- [ ] Identity injection loads from `.claude/agents/go-db-architect/go-db-architect.md`
- [ ] Agent identity ≤400 lines; db-conventions.md ≤400 lines
- [ ] License annotations correct (chromem-go: MPL-2.0, coder/hnsw: CC0-1.0)
- [ ] `gogent-validate` routes correctly for "graph schema", "knowledge graph" triggers
- [ ] db-conventions.md includes FTS5 sync triggers
- [ ] `spawned_by` includes router, orchestrator, impl-manager [Δ21]
- [ ] Hook interference audit completed (Validation Task 5) [Δ27]

### Phase 2 — Core Storage Layer (~2 weeks) [Δ24]

SQLite schema with FTS5 sync triggers, atomic.Pointer[map] cache, IR-based scoring, basic CRUD + search.

**Prerequisites:**
- [ ] Phase 0 Validation Task 1 (nomic-embed-text normalization) PASSED
- [ ] Phase 0 Validation Task 4 (SQLite cold-start latency) PASSED [Δ23]
- [ ] Phase 1 complete
- [ ] Dependencies added to go.mod [Δ24]:
  ```bash
  go get modernc.org/sqlite
  go get github.com/philippgille/chromem-go
  go get github.com/hbollon/go-edlib
  go get gonum.org/v1/gonum
  go get github.com/coder/hnsw
  go get github.com/viterin/vek
  ```

**Decision points before starting:**
- Validate chromem-go with agent memory content types — embed 100 representative memories, run 20 queries
- Choose BM25 implementation: SQLite FTS5 rank function vs. Go-side scoring
- Set initial α/β/γ/δ weights (recommend 0.3/0.5/0.1/0.1)
- Choose chromem-go persistence mode: `ExportToFile` (single file, recommended) vs per-document

**Parallelization layers:**
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
- [ ] Database stored at `.gogent/graphdb/graph.db` [Δ19]

### Phase 3 — Obsidian Vault + Consolidation (~2 weeks)

Write-only vault export (string construction), batch import (hybrid goldmark parsing), consolidation pipeline with context-enriched entity resolution.

**Prerequisites:**
- [ ] Phase 0 Validation Task 2 (hybrid goldmark coexistence) PASSED
- [ ] Phase 2 complete
- [ ] Goldmark dependencies added to go.mod [Δ24]:
  ```bash
  go get github.com/powerman/goldmark-obsidian
  go get github.com/VojtaStruhar/goldmark-obsidian-callout
  ```

**Decision points before starting:**
- Validate vault *write* fidelity with text/template approach [Δ7]
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

### Phase 4 — Hook Integration + Migration (~1.5 weeks) [Δ20, Δ26]

Extend `gogent-load-context` with graph backend, migrate existing JSONL memories.

**Prerequisites:**
- [ ] Phase 3 complete

**Decision points before starting:**
- Migration strategy: one-time import vs. gradual dual-write

**Rollback strategy:** [Δ26]
1. Preserve original JSONL files as `.jsonl.bak` during migration (never delete originals)
2. `GOGENT_MEMORY_BACKEND` env var controls backend selection:
   - `jsonl` (default): original JSONL codepath, graph code skipped entirely
   - `graph`: knowledge graph retrieval active
3. To rollback: `export GOGENT_MEMORY_BACKEND=jsonl` — no settings.json changes, no binary swaps
4. Document rollback procedure in `.claude/docs/memory-rollback.md`

**Success criteria:**
- [ ] `gogent-load-context` extended with graph retrieval codepath [Δ20]
- [ ] `GOGENT_MEMORY_BACKEND=graph` retrieves memories from knowledge graph within latency budget
- [ ] `GOGENT_MEMORY_BACKEND=jsonl` (default) preserves original behavior, ≤28ms baseline
- [ ] Existing JSONL memories imported to knowledge graph
- [ ] JSONL originals preserved as `.jsonl.bak` [Δ26]
- [ ] No regression in existing memory-archivist functionality
- [ ] Session handoff works with new storage backend
- [ ] Total startup overhead with `GOGENT_MEMORY_BACKEND=graph` ≤100ms
- [ ] Rollback to JSONL tested and documented [Δ26]

---

## 14. Open Questions (Validate During / Before Implementation)

| Question | Importance | When to Answer | Method | Status |
|----------|-----------|----------------|--------|--------|
| Does nomic-embed-text return L2-normalized vectors? | **BLOCKING** | Phase 0 | Embed 10 test strings, compute L2 norms | ⏳ Pending |
| Do powerman/goldmark-obsidian and VojtaStruhar callout coexist? | **BLOCKING** | Phase 0 | Register both, parse 50 vault files | ⏳ Pending |
| SQLite cold-start latency on post-migration codebase? [Δ23] | **BLOCKING** | Phase 0 | Open DB, execute FTS5 query, measure | ⏳ Pending |
| What is the actual entity extraction rate per session? | HIGH | Before Phase 3 | Run extraction on 10 real sessions | ⏳ Pending |
| What % of useful memory retrievals require graph traversal? | HIGH | Before Phase 2 | Classify 20 representative queries | ⏳ Pending |
| Should the 0.90 cosine threshold be raised to 0.92-0.95? | MEDIUM | After 20 consolidation passes | Audit all auto-merges | ⏳ Deferred |
| Does pre-compaction flush produce useful memories or noise? | MEDIUM | Phase 3 experiment | Test across 20 sessions | ⏳ Pending |
| What concrete bi-temporal queries would agents execute? | MEDIUM | Before Phase 3 | Articulate 3 scenarios | ⏳ Pending |
| Is there a timeline for powerman callout support? | LOW | Before Phase 3 | Check GitHub issues | ⏳ Pending |
| Do new hooks (gogent-permission-gate, etc.) interfere? [Δ27] | MEDIUM | Phase 0 | Audit settings.json, cross-reference ops | ⏳ Pending |

---

## 15. Risk Register

| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|--------|------------|-------|
| **FTS5 index silently desyncs** [Δ13] | ~~High~~ **Eliminated** | High | AFTER triggers ensure sync. Test in Phase 2. | Phase 2 |
| **Entity resolution false merges** | Medium | High | Context-enriched embeddings [Δ14]. Monitor first 20 consolidation runs. | Phase 3 |
| **nomic-embed-text not L2-normalized** | Medium | High | **Blocking Phase 0 validation.** Pre-normalize if needed. | Phase 0 |
| **Borderline entity pairs accumulate** | Medium | Low | Tier 4 default [Δ15] flags for review. | Phase 3+ |
| **goldmark extension conflict** | Low | Medium | **Blocking Phase 0 validation.** | Phase 0 |
| **atomic.Pointer nil dereference** | Low | High | Constructor initializes with empty map. Test case. | Phase 2 |
| Entity extraction produces noisy graphs | Medium | High | Validate precision ≥0.8 on 50 sessions; human-review gate if needed | Phase 3 |
| chromem-go degrades beyond 50K vectors | Medium | Medium | Benchmark at 100K; swap to coder/hnsw behind interface | Phase 2 |
| Hook chain latency budget insufficient | Medium | High | Profile first; graph backend skipped when `GOGENT_MEMORY_BACKEND=jsonl` [Δ20, Δ26] | Phase 4 |
| Trigger collision: routes to go-pro | Medium | Medium | auto_activate paths take priority; verify after first 10 invocations | Phase 1 |
| chromem-go single-maintainer risk | Medium | Medium | Interface abstraction allows swap. Monitor quarterly. | Ongoing |
| chromem-go per-document persistence | Medium | Low | Use `ExportToFile` for single-file persistence | Phase 2 |
| MPL-2.0 compliance if chromem-go modified | Low | Medium | Track in LICENSE-THIRD-PARTY | Ongoing |
| **SQLite cold-start exceeds 50ms** [Δ23] | Medium | High | Phase 0 validation. If fails, design async injection. | Phase 0 |
| **New hooks block go-db-architect ops** [Δ27] | Low | Medium | Audit settings.json in Phase 0 (Validation Task 5). Whitelist paths if needed. | Phase 0 |
| **MCP spawn doesn't load db-conventions.md** [Δ27] | Low | Medium | Verify buildFullAgentContext() loads conventions via agents-index.json context_requirements automatically. | Phase 1 |
| **Incomplete .gogent/ migration in agents-index.json** [Δ22] | Medium | Low | Audit agents-index.json for stale .claude/ runtime I/O paths. Fix memory-archivist inputs/outputs if needed. | Phase 1 |

---

## 16. Sharp Edges (v1 Active)

> **v4 update:** Unchanged from v3. Total: ~20 active edges.

**SQLite:**
- `deferred-tx-upgrade`: Deferred TX upgrading to write gets instant SQLITE_BUSY. Fix: `_txlock=immediate` in DSN
- `unclosed-rows-wal-growth`: Unclosed `rows` objects prevent WAL checkpointing. Always `defer rows.Close()`
- `modernc-maxopenconns`: `SetMaxOpenConns(>0)` required for concurrent access
- `recursive-cte-dense-graph`: Recursive CTEs cause exponential blowup. Use gonum instead
- `sqlite-vec-modernc-incompatible`: sqlite-vec will NOT work with modernc.org/sqlite
- `fts5-external-content-no-autosync` [Δ13]: FTS5 external content tables do NOT auto-sync. Requires AFTER triggers.

**Vector search:**
- `embedding-model-mismatch`: Always store `model_version` with every embedding
- `chromem-go-100k-cap`: Beyond 100K vectors, latency degrades to ~80ms+
- `chromem-go-no-update`: No document update operation. Must delete-then-re-add.
- `chromem-mpl2-compliance` [Δ2]: Modifications must be disclosed under MPL-2.0

**Vault / Markdown:**
- `yaml-frontmatter-special-chars`: Always quote values with special characters
- `goldmark-no-roundtrip` [Δ7]: Never use goldmark to write vault files
- `goldmark-obsidian-incomplete` [Δ6-CORRECTED]: 7/10 features. Requires hybrid stack.
- `goldmark-footnotes-reference-only`: Reference-style footnotes only

**Entity extraction:**
- `extraction-noise-accumulation`: Reject below 0.8 confidence
- `entity-resolution-threshold`: Use tiered OR-logic (§9), not single threshold [Δ8]
- `entity-resolution-homonyms` [Δ14]: Use context-enriched embeddings

**chromem-go / embeddings:**
- `ollama-not-normalized`: Verify nomic-embed-text normalization. **Blocking Phase 0.**

**IR scoring:**
- `bm25-feedback-loop`: Bootstrap new memories with `access_count=1`
- `scoring-weight-sensitivity`: Start with 0.3/0.5/0.1/0.1, tune empirically

**Cache:**
- `atomic-pointer-nil-init` [Δ5]: Constructor must initialize with empty map

---

## 17. Integration with GOgent-Fortress Routing

**Trigger patterns for go-db-architect (unambiguous, no collision):**
- "graph schema", "knowledge graph", "temporal graph" — unambiguous
- "memory graph system", "memory subsystem schema" — prefixed to avoid memory-archivist collision
- "graph store", "consolidation pipeline", "entity extraction", "vector index", "embedding store"

**Auto-activate paths:**
```
internal/memory/**
internal/graphstore/**
internal/vectorindex/**
internal/vault/**
internal/consolidation/**
```

**Trigger collision monitoring:** Check routing logs after first 10 invocations.

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

> **[Δ19] DSN path:** `.gogent/graphdb/graph.db` (was `.gogent-vault/.gogent/graph.db`)

---

## 19. Package Structure [Δ25]

```
internal/
├── memory/
│   ├── types.go              -- Node, Edge, Episode, CoreBlock types
│   ├── types_test.go         -- [Δ25]
│   ├── store.go              -- GraphStore interface
│   ├── sqlite.go             -- SQLite implementation
│   ├── sqlite_test.go        -- [Δ25]
│   ├── cache.go              -- atomic.Pointer[map] core block cache  [Δ5]
│   ├── cache_test.go         -- [Δ25]
│   └── scoring.go            -- IR-based composite scoring (RRF v1, CC-ready)  [Δ9]
│   └── scoring_test.go       -- [Δ25]
├── graphstore/
│   ├── schema.go             -- Migration runner, DDL, FTS5 sync triggers  [Δ13]
│   ├── schema_test.go        -- [Δ25]
│   ├── crud.go               -- Node/edge CRUD
│   ├── crud_test.go          -- [Δ25]
│   └── query.go              -- FTS5 + vector hybrid retrieval
│   └── query_test.go         -- [Δ25]
├── vectorindex/
│   ├── index.go              -- VectorIndex interface
│   ├── chromem.go            -- chromem-go implementation
│   ├── chromem_test.go       -- [Δ25]
│   └── hnsw.go               -- coder/hnsw or fogfish/hnsw fallback  [Δ3]
├── vault/
│   ├── export.go             -- SQLite → Markdown (string construction)  [Δ7]
│   ├── export_test.go        -- [Δ25]
│   ├── import.go             -- Batch import: content hash diffing
│   ├── parse.go              -- Hybrid goldmark parsing  [Δ6-CORRECTED]
│   └── templates.go          -- text/template vault file templates  [Δ7]
├── consolidation/
│   ├── pipeline.go           -- Session-end consolidation orchestration
│   ├── extract.go            -- Entity extraction via Claude Sonnet
│   ├── resolve.go            -- Entity resolution (context-enriched, embedding-first)  [Δ8, Δ14, Δ15]
│   ├── resolve_test.go       -- [Δ25]
│   └── algorithms.go         -- PageRank, Louvain, label propagation via gonum
└── hooks/                     -- [Δ20]
    ├── loader.go              -- Graph-backed memory retrieval (imported by gogent-load-context)
    └── loader_test.go         -- [Δ25]
```

---

## 20. Interface Contracts

### 20.1 GraphStore Interface

```go
type GraphStore interface {
    CreateNode(ctx context.Context, node *Node) (int64, error)
    GetNode(ctx context.Context, id int64) (*Node, error)
    UpdateNode(ctx context.Context, node *Node) error
    DeleteNode(ctx context.Context, id int64) error

    CreateEdge(ctx context.Context, edge *Edge) (int64, error)
    GetCurrentEdges(ctx context.Context, nodeID int64, relationType string) ([]*Edge, error)
    InvalidateEdge(ctx context.Context, id int64, validTo time.Time) error

    AppendEpisode(ctx context.Context, episode *Episode) (int64, error)
    GetEpisodesBySession(ctx context.Context, sessionID string) ([]*Episode, error)

    SearchFTS(ctx context.Context, query string, limit int) ([]*Node, error)
    SearchHybrid(ctx context.Context, query string, embedding []float32, weights HybridWeights, limit int) ([]*ScoredNode, error)

    GetCoreBlocks(ctx context.Context) ([]*Node, error)

    UpdatePrecomputedFeatures(ctx context.Context, nodeID int64, pagerank float64, community int) error
    GetNodesForConsolidation(ctx context.Context, since time.Time) ([]*Node, error)
}
```

### 20.2 VectorIndex Interface

```go
type VectorIndex interface {
    Add(ctx context.Context, id string, embedding []float32, metadata map[string]any) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query []float32, k int) ([]SearchResult, error)

    Persist(ctx context.Context, path string) error
    Load(ctx context.Context, path string) error

    Count() int
    Dimensions() int
}

type SearchResult struct {
    ID         string
    Distance   float32
    Metadata   map[string]any
}
```

### 20.3 VaultSync Interface

```go
type VaultSync interface {
    ExportNode(ctx context.Context, node *Node) error
    ExportEpisodeSummary(ctx context.Context, date time.Time, episodes []*Episode) error
    ExportMemoryBrief(ctx context.Context, topNodes []*Node) error

    DetectChanges(ctx context.Context) ([]ChangedFile, error)
    ImportChanges(ctx context.Context, changes []ChangedFile) error

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
| Drop bbolt | 5ms→0.5ms imperceptible; atomic.Pointer[map] equivalent [Δ5] | Profiling showing SQLite reads are actual bottleneck |
| Write-only vault v1 | Bidirectional sync complexity > value for rare use case | Multiple users reporting frustration |
| Consolidation-only graph algorithms | 50-500ms runtime violates latency budget | New query patterns requiring runtime traversal |
| IR scoring over FSRS | Agents don't forget; no spacing effect | FSRS predictions correlate with actual retrieval utility |
| Keep bi-temporal columns | Retrofitting expensive; 4 columns cheap | N/A |
| coder/hnsw as approved fallback | CC0-1.0 verified [Δ1]. Behind VectorIndex interface. | chromem-go benchmark succeeds at 100K |
| goldmark for parsing only [Δ7] | AST discards Obsidian formatting. Proven. | goldmark adds perfect Obsidian Markdown renderer |
| Hybrid goldmark stack [Δ6-CORRECTED] | powerman 7/10 features. VojtaStruhar for callouts. | powerman implements callouts natively |
| Embedding-first entity resolution [Δ8] | String metrics fail on abbreviations/synonyms | Jaro-Winkler-first produces fewer false negatives |
| Context-enriched embeddings [Δ14] | Name-only causes homonym collisions | Name-only produces acceptable false-merge rate |
| Tier 4 borderline disposition [Δ15] | Prevents unbounded duplicate accumulation | Better deduplication strategy found |
| RRF k=60 for v1, CC upgrade [Δ9] | RRF battle-tested, no normalization needed | CC measurably better on first evaluation |
| Vault DB at `.gogent/graphdb/` [Δ19] | DB files are runtime artifacts; `.gogent/` is runtime I/O dir post-migration. Separate from `.gogent/memory/` to avoid JSONL transition confusion. | N/A |
| Extend gogent-load-context for memory loading [Δ20] | Avoids binary proliferation; single integration point; session context already in-process; rollback via env var | Separate binary provides measurably better fault isolation |
| `GOGENT_MEMORY_BACKEND` env var rollback [Δ26] | Instant rollback without settings.json changes or binary swaps; `jsonl` default preserves existing behavior | N/A |
