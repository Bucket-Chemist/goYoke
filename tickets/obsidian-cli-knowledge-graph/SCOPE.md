# GOgent-Fortress Embedded Knowledge Graph — Unified Scope

> **Status:** CONDITIONAL GO — 3 deployment blockers must be resolved before implementation begins
> **Source:** Synthesized from system_design.md + embedded_kg.md + Braintrust analysis (Einstein + Staff-Architect + Beethoven)
> **Date:** 2026-02-19
> **Use:** Input to `/plan-tickets`

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
> **Verdict:** Both tracks CONDITIONAL GO — proceed only after resolving three blockers below

### 2.1 What We Are Building

```
┌─────────────────────────────────────────────────────────┐
│                    RUNTIME PATH                         │
│                                                         │
│  sync.Map cache   ←──  Core blocks (~10 entries)       │
│       ↑                  <1ms reads, volatile           │
│  SQLite (truth)   ←──  Full graph, FTS5, bi-temporal   │
│       ↑                  1-50ms reads                   │
│  chromem-go       ←──  Vector similarity search        │
│                          ~40ms for 100K vectors         │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│               CONSOLIDATION PATH (offline)              │
│                                                         │
│  [Session end] → Entity extraction (Claude Sonnet)     │
│  → Entity resolution (Jaro-Winkler + embedding sim)    │
│  → Graph algorithm batch (PageRank, Louvain, labels)   │
│  → Pre-computed features written to SQLite columns     │
│  → Markdown export → .gogent-vault/                    │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│               SESSION STARTUP PATH                      │
│                                                         │
│  Load core blocks → sync.Map                           │
│  Scan vault for human edits → import changed files     │
│  Hybrid query → top-k context injection                │
└─────────────────────────────────────────────────────────┘
```

### 2.2 What We Are NOT Building (v1)

| Dropped from original design | Why | Upgrade path |
|------------------------------|-----|--------------|
| **bbolt cache layer** | 5ms→0.5ms improvement is imperceptible; sync.Map provides equivalent benefit without a second consistency boundary (27-125 combined failure modes across 3 engines) | Add if profiling reveals SQLite reads are the bottleneck |
| **FSRS/Ebbinghaus scoring** | Agents don't forget — they lose context window space. Spaced repetition optimizes for human biological retention, not information retrieval precision | Already replaced by IR scoring below |
| **Bidirectional real-time vault sync** | Solves a rare use case (human edits during active session) at maximum complexity (fsnotify + debouncing + conflict resolution + feedback loop prevention) | v2 enhancement if users demand it; sharp edges documentation already covers pitfalls |
| **Runtime graph traversal** | PageRank/Louvain run in 50-500ms — violates latency budget. Flat retrieval handles 90%+ of agent query patterns | Pre-computed features from consolidation serve the same need |
| **coder/hnsw (AGPL-3)** | Distribution poison pill; violates agent's own LICENSE RULE | Custom ~200-line HNSW using gonum if chromem-go cap proves insufficient |

---

## 3. Deployment Blockers (Must Resolve Before Implementation)

> These are structural prerequisites. No code should be written until all three are resolved.

### BLOCKER C-1: AGPL License Contradiction (Critical)
**Problem:** `github.com/coder/hnsw` is listed as an approved dependency but is AGPL-3 only. The agent's own LICENSE RULE prohibits GPL/SSPL dependencies. AGPL-3 is *more* restrictive than GPL-3 — using it in a desktop binary may require open-sourcing the entire binary.

**Resolution:** Remove `coder/hnsw` from approved dependencies. Cap vector search at `chromem-go`'s 100K vector limit (MIT dual-licensed). This is sufficient for the 10K-node near-term scale both analysts agree on.

**Effort:** 30 minutes. Decision only — no code change needed pre-implementation.

### BLOCKER M-1: No agents-index.json Entry (Major)
**Problem:** `go-db-architect` has no entry in `agents-index.json`. Without it, the agent cannot be routed to, auto-activated, spawned by orchestrators/impl-manager, or used in `/implement`. The entire definition is inert.

**Resolution:** Create full agents-index.json entry following go-pro template:
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
  "sharp_edges_count": 17
}
```
Also add `go-db-architect` to `impl-manager.can_spawn` and `orchestrator.can_spawn`.

**Effort:** 30-60 minutes including validation testing.

### BLOCKER M-3: Wrong File Location (Major)
**Problem:** Agent definition lives in `tickets/obsidian-cli-knowledge-graph/` — a ticket working directory. Identity injection reads from `~/.claude/agents/{id}/{id}.md`. Current location is invisible to runtime.

**Resolution:**
```bash
mkdir -p ~/.claude/agents/go-db-architect/
mv tickets/obsidian-cli-knowledge-graph/go-db-architect.md ~/.claude/agents/go-db-architect/go-db-architect.md
mv tickets/obsidian-cli-knowledge-graph/go-db-architect-sharp-edges.yaml ~/.claude/agents/go-db-architect/sharp-edges.yaml
```

**Effort:** 5 minutes.

---

## 4. Agent Definition Refactoring (Pre-Implementation)

> This must be done before registering in agents-index.json (Major issue M-2 from Staff-Architect)

**Problem:** The current `go-db-architect.md` is 882 lines — 2.4x the peer average (go-pro: 374, go-tui: 471). It contains complete function implementations that inflate every invocation's context budget unnecessarily.

**Resolution:** Split into two files following the go-tui pattern:

**`~/.claude/agents/go-db-architect/go-db-architect.md` (~350 lines)**
- Role description and system constraints
- Approved dependencies (with license annotations, minus coder/hnsw)
- Focus area rules as CORRECT/WRONG patterns (not full implementations)
- Interface contracts section (GraphStore, VectorIndex, VaultSync interfaces)
- Testing strategy overview
- Output requirements and parallelization model

**`~/.claude/conventions/db-conventions.md` (~300 lines)**
- Full SQL schema with all DDL
- Complete function implementations (decayScore, compositeScore, RRF, RunMigrations)
- Algorithm details (entity resolution, label propagation)
- Template strings and query patterns
- Vault file structure and goldmark extension stack

**Wire via** `context_requirements.conventions.base: ["db-conventions.md"]` in agents-index.json.

**After Einstein's simplifications, the delta shrinks:** bbolt patterns (~80 lines), runtime traversal code (~60 lines), and bidirectional sync patterns (~50 lines) are no longer needed → convention file targets ~300 lines, not 400.

---

## 5. Technology Stack

| Component | Library | License | Purpose |
|-----------|---------|---------|---------|
| **SQLite** | `modernc.org/sqlite` | BSD + Public Domain | Graph store, FTS5, WAL mode, bi-temporal schema |
| **Vector search** | `github.com/philippgille/chromem-go` | MIT | In-process embedding store, exhaustive NN, <100K vectors |
| **In-process cache** | `sync.Map` (stdlib) | BSD | Core blocks hot path, volatile |
| **Graph algorithms** | `gonum.org/v1/gonum/graph` | BSD | PageRank, community detection, BFS (consolidation only) |
| **Markdown parser** | `github.com/yuin/goldmark` | MIT | Obsidian vault read/write |
| **Wikilinks** | `go.abhg.dev/goldmark/wikilink` | BSD | `[[link]]` syntax |
| **Frontmatter** | `go.abhg.dev/goldmark/frontmatter` | MIT | YAML frontmatter |
| **Hashtags** | `go.abhg.dev/goldmark/hashtag` | MIT | Obsidian hashtag variant |
| **Entity similarity** | `github.com/hbollon/go-edlib` | MIT | Jaro-Winkler for entity resolution |
| **Embeddings** | Ollama `nomic-embed-text` | — | 768-dim, 8K context, local |
| **NOT included** | `github.com/coder/hnsw` | AGPL-3 🚫 | License violation — cap at chromem-go |
| **NOT included** | `go.etcd.io/bbolt` | MIT | Removed — sync.Map replaces it |

**SQLite PRAGMA configuration:**
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 10000;
PRAGMA cache_size = -32000;        -- 32MB
PRAGMA temp_store = MEMORY;
PRAGMA foreign_keys = ON;
PRAGMA mmap_size = 268435456;      -- 256MB
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
CREATE VIRTUAL TABLE nodes_fts USING fts5(
    name, properties,
    content=nodes, content_rowid=id,
    tokenize='porter unicode61'
);
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
```

### 6.3 Bi-Temporal Note
The 4 bi-temporal columns on `edges` are **retained but no query utilities built until concrete examples emerge** (Beethoven resolution of Einstein/Staff-Architect divergence). Cost of adding 4 columns: minutes. Cost of retrofitting bi-temporal later: days. After 3 months of operation, if no bi-temporal query has been articulated, columns are inert schema — ignore and proceed.

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

---

## 8. Three-Phase Startup Loading

**Phase 1 — Core blocks (<10ms, from sync.Map):**
Always injected into system prompt. Covers: agent persona/role, user profile, active project context, critical procedural rules. Budget: ~10-15% context window (400-600 tokens). On startup: load all `type='core_block'` nodes from SQLite → populate sync.Map. On write: update SQLite first → update sync.Map atomically.

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
[Entity resolution]
    match_score = α × jaro_winkler(a, b) + (1-α) × embedding_cosine_sim(a, b)
    Threshold: 0.85 for merge
    ↓
[Graph algorithms (gonum, consolidation-only)]
    PageRank → nodes.pagerank
    Louvain community detection → nodes.community
    Label propagation (custom ~30 lines) → community refinement
    Write pre-computed features back to SQLite
    ↓
[Markdown export → .gogent-vault/]
    Write entity files with YAML frontmatter + wikilinks
    Store content hash in frontmatter for change detection
    ↓
[Done — next session startup scans vault for human edits]
```

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

**goldmark extension stack for vault read/write:**
- `go.abhg.dev/goldmark/wikilink` — `[[links]]`, `[[link|display]]`, `![[embeds]]`
- `go.abhg.dev/goldmark/frontmatter` — YAML frontmatter in AST pipeline
- `go.abhg.dev/goldmark/hashtag` — ObsidianVariant mode
- `github.com/VojtaStruhar/goldmark-obsidian-callout` — all callout types
- Custom parsers (~50-100 lines each): highlights (`==text==`), comments (`%%...%%`)

**v2 (future, when user demand warrants):** Bidirectional real-time sync via fsnotify. Sharp edges documentation already covers all pitfalls (duplicate events, no recursive watch, write feedback loop, debouncing patterns) — retain as v2 reference.

---

## 11. Hook Integration

**Integration point:** Extend `gogent-load-context` (SessionStart hook) to query knowledge graph instead of (or in addition to) JSONL files.

**Decision points for Phase 4:**
- Extend existing hook binary vs. create new hook
- JSONL dual-read period: how long to maintain backward compatibility
- Migration trigger: one-time JSONL→SQLite import vs. gradual dual-write

**Latency budget validation (blocking assumption):**
Must profile `gogent-load-context`, `gogent-validate`, and `gogent-sharp-edge` p50/p95 before implementation. If existing hooks consume >50ms, only 50ms remains for memory retrieval — may require async injection after first agent response.

---

## 12. Memory-Archivist Relationship

**Relationship question (open):** Does `go-db-architect` replace `memory-archivist`'s storage backend (dependency) or serve completely different trigger patterns (peer)?

**Working assumption:** `memory-archivist` continues to handle session handoff generation (JSONL + last-handoff.md). `go-db-architect` handles the new graph-based semantic layer. During transition: dual-write to both systems. Post-migration: memory-archivist writes episodes to SQLite instead of JSONL.

**Migration path:**
- Session scope JSONL → episodes table
- Project scope JSONL → semantic entity nodes
- Global scope JSONL → procedural memory nodes
- ML scope JSONL → telemetry subgraph (new fourth category)

---

## 13. Implementation Phases

### Phase 1 — Deployment Setup (Prerequisite, ~2 hours)
**Must complete before any implementation code is written.**

1. Resolve C-1: Remove `coder/hnsw` from approved dependencies
2. Resolve M-3: Move files to `~/.claude/agents/go-db-architect/`
3. Refactor agent definition: split into go-db-architect.md (~350 lines) + db-conventions.md (~300 lines)
4. Update scoring section in agent definition: replace FSRS/Ebbinghaus with IR-based approach
5. Resolve M-1: Create agents-index.json entry + add to can_spawn lists
6. Validate routing: `gogent-validate routes to go-db-architect for "graph schema" trigger`

**Success criteria:**
- [ ] `go-db-architect` in agents-index.json with all required fields
- [ ] Identity injection loads from `~/.claude/agents/go-db-architect/go-db-architect.md`
- [ ] Agent identity ≤400 lines; db-conventions.md ≤350 lines
- [ ] No AGPL dependencies in approved list
- [ ] gogent-validate routes correctly for "graph schema", "knowledge graph" triggers

### Phase 2 — Core Storage Layer (~2 weeks)
SQLite schema, sync.Map cache, IR-based scoring, basic CRUD + search.

**Decision points before starting:**
- Validate chromem-go with agent memory content types (code, errors, decisions) — embed 100 representative memories, run 20 queries
- Choose BM25 implementation: SQLite FTS5 rank function vs. Go-side scoring
- Set initial α/β/γ/δ weights (recommend 0.3/0.5/0.1/0.1 as starting point)

**Parallelization layers (from go-db-architect's Layer 0-4 model):**
- Layer 0: Types and constants (`internal/memory/types.go`)
- Layer 1: Interfaces (`GraphStore`, `VectorIndex`, `VaultSync`)
- Layer 2: SQLite implementation (schema + CRUD + FTS5)
- Layer 3: chromem-go vector index + sync.Map cache
- Layer 4: Composite scoring engine

**Success criteria:**
- [ ] SQLite schema created: nodes, edges (bi-temporal), episodes, nodes_fts
- [ ] sync.Map cache loads core blocks in <5ms
- [ ] End-to-end retrieval (query → ranked results) in <50ms at 1K nodes
- [ ] Dual connection pools (readDB N readers + writeDB 1 writer)
- [ ] All sharp edges from yaml validated with test coverage

### Phase 3 — Obsidian Vault + Consolidation (~2 weeks)
Write-only vault export, batch import on startup, consolidation pipeline.

**Decision points before starting:**
- Validate goldmark + abhinav extension stack round-trip fidelity (test vault with 50 files)
- Consolidation trigger: session-end only vs. periodic (recommend session-end for v1)
- Entity extraction validation: test on 10 real sessions, measure precision (must be ≥0.8)

**Success criteria:**
- [ ] Vault export produces valid Obsidian-compatible Markdown with wikilinks + frontmatter
- [ ] Batch import detects human edits via content hash comparison
- [ ] Consolidation completes in <30s for 1K nodes
- [ ] Pre-computed PageRank and community scores accessible as column reads at runtime
- [ ] goldmark round-trip: zero data loss on test vault

### Phase 4 — Hook Integration + Migration (~1.5 weeks)
Wire into hook chain, migrate existing JSONL memories.

**Decision points before starting:**
- Profile existing hook chain p50/p95 to determine actual retrieval budget
- If >50ms consumed: design async injection path
- Migration strategy: one-time import vs. gradual dual-write

**Success criteria:**
- [ ] gogent-load-context retrieves memories from knowledge graph within latency budget
- [ ] Existing JSONL memories imported to knowledge graph
- [ ] No regression in existing memory-archivist functionality
- [ ] Session handoff works with new storage backend
- [ ] Total startup overhead (Phase 1+2 load) ≤100ms

---

## 14. Open Questions (Validate During / Before Implementation)

| Question | Importance | When to Answer | Method |
|----------|-----------|----------------|--------|
| What is the actual entity extraction rate per session? Does it support the 10K-node scale assumption? | HIGH (BLOCKING) | Before Phase 3 | Run extraction pipeline on 10 real sessions; project growth rate |
| How much of the hook chain's 100ms budget remains for memory retrieval? | HIGH (BLOCKING) | Before Phase 4 | Profile gogent-load-context + gogent-validate + gogent-sharp-edge p50/p95 |
| What % of useful memory retrievals require graph traversal vs. flat search? | HIGH | Before Phase 2 | Classify 20 representative agent queries as flat-retrieval vs. traversal-needed |
| Does the pre-compaction flush produce useful memories or noise? | MEDIUM | Phase 3 experiment | Implement as standalone test across 20 sessions; measure NO_REPLY rate |
| What concrete bi-temporal queries would agents actually execute? | MEDIUM | Before Phase 3 query utilities | Articulate 3 scenarios; if none found, defer query utilities |
| Should go-db-architect be in memory-archivist's `can_spawn` list? | MEDIUM | Phase 4 | Map the workflow dependency structure |

---

## 15. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Entity extraction produces noisy graphs | Medium | High | Validate precision ≥0.8 on 50 sessions before committing to auto-extraction; add human-review gate if needed |
| chromem-go degrades beyond 50K vectors before anticipated | Medium | Medium | Benchmark at 100K vectors before committing; if fails, custom ~200-line HNSW using gonum |
| Hook chain latency budget leaves insufficient room for retrieval | Medium | High | Profile first; if >50ms consumed, design async injection after first response |
| Trigger collision: 'memory system' routes to go-pro instead of go-db-architect | Medium | Medium | auto_activate paths take priority; verify after first 10 invocations; prefix ambiguous triggers |
| goldmark round-trip is lossy for Obsidian-specific syntax | Medium | Medium | Test vault with 50 representative files before Phase 3; any data loss is a user trust issue |
| Simplified architecture proves insufficient, requires retrofitting | Low | Medium | Design storage interfaces accommodating future bbolt/bidirectional sync without breaking changes |

---

## 16. Sharp Edges (v1 Active, v2 Deferred)

> **After Einstein's simplifications:** ~15-17 active edges (from 25 original). bbolt edges (~5) and bidirectional vault sync edges (~3) deferred to v2 documentation.

**Active v1 sharp edges:**

**SQLite:**
- `deferred-tx-upgrade`: Deferred TX upgrading to write gets instant SQLITE_BUSY regardless of busy_timeout. Fix: `_txlock=immediate` in DSN or explicit `BEGIN IMMEDIATE`
- `unclosed-rows-wal-growth`: Unclosed `rows` objects prevent WAL checkpointing → unbounded WAL growth. Always `defer rows.Close()`, check `rows.Err()` after iteration
- `modernc-maxopenconns`: `SetMaxOpenConns(>0)` required for concurrent access; without it, deadlock
- `recursive-cte-dense-graph`: Recursive CTEs cause exponential blowup in dense graphs (>10K edges). Use gonum graph algorithms for traversal instead
- `sqlite-vec-modernc-incompatible`: sqlite-vec will NOT work with modernc.org/sqlite; this is why chromem-go is separate

**Vector search:**
- `embedding-model-mismatch`: Vectors from different models exist in incompatible spaces. Always store `model_version` with every embedding; never mix models in similarity queries
- `chromem-go-100k-cap`: chromem-go uses exhaustive search; beyond 100K vectors, latency degrades. Benchmark at 100K before committing as sole solution

**Vault / Markdown:**
- `yaml-frontmatter-special-chars`: Unquoted colons in YAML values parse as nested mappings. `yes/no/true/false` strings parsed as booleans. Always quote values with special characters
- `goldmark-obsidian-highlights`: `==text==` (highlights) and `%%...%%` (comments) require custom parsers; no existing library covers these
- `goldmark-round-trip-fidelity`: Complex Obsidian syntax (callouts, complex wikilinks) may have lossy round-trips. Validate with test vault before production use

**Entity extraction:**
- `extraction-noise-accumulation`: Low-confidence entity extraction accumulates noise faster than signal; degrades retrieval. Reject extractions below 0.8 confidence threshold
- `entity-resolution-threshold`: Jaro-Winkler threshold must be tuned per domain. 0.80-0.90 for names. Too low → false merges; too high → duplicates persist

**chromem-go / embeddings:**
- `ollama-not-normalized`: Not all Ollama models return L2-normalized vectors. Verify `nomic-embed-text` normalization or pre-normalize at storage time (reduces cosine sim to dot product)

**IR scoring:**
- `bm25-feedback-loop`: Access-frequency boosting creates feedback loops (popular memories get retrieved more → become more popular). Bootstrap new memories with `access_count=1` minimum to prevent cold-start disadvantage
- `scoring-weight-sensitivity`: α/β/γ/δ weights dramatically affect retrieval quality. Start with 0.3/0.5/0.1/0.1 and tune empirically using actual retrieval utility measurements

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
│   ├── cache.go              -- sync.Map core block cache
│   └── scoring.go            -- IR-based composite scoring
├── graphstore/
│   ├── schema.go             -- Migration runner, DDL
│   ├── crud.go               -- Node/edge CRUD
│   └── query.go              -- FTS5 + vector hybrid retrieval
├── vectorindex/
│   ├── index.go              -- VectorIndex interface
│   └── chromem.go            -- chromem-go implementation
├── vault/
│   ├── export.go             -- SQLite → Markdown write-only export
│   ├── import.go             -- Batch import: content hash diffing
│   └── markdown.go           -- goldmark + extension stack
├── consolidation/
│   ├── pipeline.go           -- Session-end consolidation orchestration
│   ├── extract.go            -- Entity extraction via Claude Sonnet
│   ├── resolve.go            -- Entity resolution (Jaro-Winkler + embedding)
│   └── algorithms.go         -- PageRank, Louvain, label propagation via gonum
└── hooks/
    └── loader.go             -- gogent-load-context integration
```

---

## Appendix A: Decisions Made (Do Not Revisit Without Evidence)

| Decision | Rationale | Evidence required to change |
|----------|-----------|---------------------------|
| Drop bbolt | 5ms→0.5ms imperceptible; sync.Map equivalent; eliminates consistency boundary | Profiling showing SQLite reads are actual bottleneck in hook path |
| Write-only vault v1 | Bidirectional sync complexity > value for rare use case | Multiple users reporting frustration with between-session edit model |
| Consolidation-only graph algorithms | 50-500ms runtime graph compute violates latency budget | New agent query patterns that demonstrably require runtime traversal |
| IR scoring over FSRS | Agents don't forget; retrieval doesn't strengthen traces; no spacing effect | Empirical data showing FSRS predictions correlate with actual useful memory retrieval |
| Keep bi-temporal columns | Retrofitting is expensive; 4 columns are cheap | N/A — columns are already in schema |
| Cap at chromem-go, no coder/hnsw | AGPL-3 license violation; 10K-node realistic scale well within 100K cap | chromem-go benchmark fails at 100K AND MIT alternative found |

---

## Appendix B: Glossary

| Term | Definition |
|------|-----------|
| **BM25** | Okapi BM25: probabilistic relevance scoring function for full-text search (successor to TF-IDF) |
| **RRF** | Reciprocal Rank Fusion: merges ranked lists from different retrieval methods using `1/(k + rank)` formula |
| **FTS5** | Full-Text Search version 5: SQLite's built-in full-text search extension with BM25 ranking |
| **WAL** | Write-Ahead Logging: SQLite journal mode enabling concurrent reads during writes |
| **CTE** | Common Table Expression: SQL `WITH` clause enabling recursive graph traversal in SQL |
| **Bi-temporal** | Two time dimensions: valid_time (when fact is true in reality) + tx_time (when system recorded it) |
| **PageRank** | Graph centrality algorithm: nodes linked from many nodes score higher |
| **Louvain** | Community detection algorithm: groups nodes by edge density |
| **HNSW** | Hierarchical Navigable Small World: approximate nearest neighbor (ANN) graph structure for fast vector search |
| **Jaro-Winkler** | String similarity metric optimized for short strings and names (0-1 scale) |
| **chromem-go** | Pure Go in-process embedding store with exhaustive nearest-neighbor search |
| **nomic-embed-text** | Ollama embedding model: 768 dimensions, 8K context, local operation |
