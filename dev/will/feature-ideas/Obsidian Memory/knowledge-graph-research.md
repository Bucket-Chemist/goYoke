# Designing a novel memory system for GOgent-Fortress

**GOgent-Fortress should build a proprietary, Go-native temporal knowledge graph on SQLite + bbolt with Obsidian-compatible Markdown as the human interface layer, independently reimplementing Graphiti's proven architectural patterns while avoiding all GPL and EULA risks.** This hybrid approach achieves sub-100ms retrieval, full IP ownership, zero external service dependencies, and a dual-purpose vault that serves both agents and humans. The key insight from this research is that the most effective agent memory systems (Claude Code, OpenClaw, Letta) all converge on a three-part pattern: typed memory stores (semantic/episodic/procedural), hybrid retrieval (vector + keyword + graph), and asynchronous consolidation — all of which can be built entirely in Go with permissively licensed components.

---

## The landscape of agent memory architectures has converged on clear winners

Modern agent memory systems cluster around a surprisingly consistent architecture. **Claude Code** uses hierarchical CLAUDE.md files loaded eagerly at startup with lazy subdirectory loading, auto-synthesized MEMORY.md (first 200 lines only), and on-demand semantic + keyword retrieval via internal tools. **OpenClaw** (145K+ GitHub stars) implements a Markdown-first approach with daily logs as STM (`memory/YYYY-MM-DD.md`, today + yesterday loaded at startup), curated MEMORY.md as LTM, and a novel pre-compaction memory flush — a silent agentic turn triggered when context approaches its limit, giving the agent one chance to persist durable insights before truncation. **Letta/MemGPT** pioneered self-editing core memory blocks always in context, with archival vector storage searched on demand, plus asynchronous "sleep-time agents" that consolidate memories during idle periods.

The pattern that emerges across all production systems is a **three-tier memory hierarchy** that maps directly to cognitive science:

- **Semantic memory** (facts, entities, relationships) stored in knowledge graphs or structured stores, updated by merging/overwriting, retrieved via graph traversal + similarity search
- **Episodic memory** (specific events, conversations, decisions) stored as timestamped logs, append-only, retrieved via temporal queries + semantic matching
- **Procedural memory** (rules, conventions, learned behaviors) stored as structured text files, updated via reflection, loaded eagerly at startup

GOgent-Fortress's existing four-tier JSONL model (session/project/global/ML) maps partially to this: session scope ≈ episodic STM, project scope ≈ semantic memory, global scope ≈ procedural memory, ML scope ≈ telemetry (a distinct fourth category). The gap is the absence of a graph-based semantic layer and a consolidation pipeline.

---

## Graphiti's architecture provides the blueprint, not the dependency

Graphiti implements a **three-subgraph hierarchical knowledge graph** G = (N, E, φ): an Episode Subgraph storing raw inputs as non-lossy records, a Semantic Entity Subgraph of extracted entities with resolved duplicates and typed relationships, and a Community Subgraph of clustered entity groups with LLM-generated summaries. Its most powerful innovation is a **bi-temporal model** with four timestamps on every edge: `t_valid`/`t_invalid` (when a fact was/stopped being true in reality) and `t'_created`/`t'_expired` (when the system recorded/invalidated it). This enables point-in-time queries and full audit trails without ever deleting data.

The ingestion pipeline processes each episode through entity extraction → entity resolution (embedding similarity + full-text + LLM deduplication) → fact extraction as relationship triples → edge deduplication → temporal extraction → contradiction detection → community updates via label propagation. **Every step except community detection requires LLM calls**, making ingestion expensive. One independent benchmark found ~1,028 LLM calls per conversational case and ~1.17M tokens per case on the MemBench dataset. However, retrieval requires **zero LLM calls** — it uses pre-computed embeddings, BM25 indices, and BFS graph traversal, achieving production P95 latency of ~300ms at Zep's scale and under 100ms in OSS deployments.

Graphiti has moved beyond Neo4j-only. Its `GraphDriver` abstract base class now supports **four backends**: Neo4j, FalkorDB (Redis-based, now the default for MCP), Kuzu (embedded), and AWS Neptune. The abstraction uses Cypher queries throughout, but each driver has its own `SearchInterface` and `GraphOperationsInterface`. The core dependency is on Cypher-compatible property graph operations with fulltext and vector index support.

**For GOgent-Fortress, the recommendation is clear: reimplement Graphiti's architectural concepts in Go, not use it as a dependency.** The concepts (bi-temporal edges, episodic/semantic/community subgraphs, hybrid BFS+BM25+vector search, label propagation for communities) are well-documented computer science techniques, many predating Graphiti. The Apache 2.0 license protects Graphiti's *code expression*, not these ideas. Reimplementation in Go creates zero license obligations and gives full IP ownership over the novel implementation.

---

## SQLite is the optimal embedded graph engine for this use case

After evaluating six embedded storage options for Go, **SQLite via modernc.org/sqlite (pure Go, CGo-free) emerges as the strongest foundation**, with bbolt as a complementary fast-path cache.

| Engine | Read (1-3 hops) | Write | RAM (10K nodes) | Startup | FTS | Pure Go |
|--------|-----------------|-------|-----------------|---------|-----|---------|
| **SQLite + CTE** | **1–50ms** | 1–10ms | 5–20MB | <50ms | **FTS5 built-in** | ✅ modernc.org |
| bbolt | <1–5ms (cached) | 10–50ms | 10–30MB | **<50ms** | ❌ | ✅ |
| BadgerDB | <1–5ms | <1–5ms | 50–150MB | 100–500ms | ❌ | ✅ |
| CayleyDB | 5–65ms | 30–100ms | 30–50MB | ~100ms | ❌ | ✅ |
| EliasDB | ~1–20ms | ~5–30ms | 20–40MB | ~100ms | Partial | ✅ |

SQLite's killer advantages for agent memory are **FTS5** (full-text search for BM25 keyword retrieval), **JSON1** extension for flexible property storage, **recursive CTEs** for graph traversal, **WAL mode** for concurrent reads, and a mature ecosystem with EXPLAIN QUERY PLAN for debugging. A schema like `nodes(id, type, data JSON, embedding BLOB)` + `edges(from_id, to_id, type, fact TEXT, t_valid, t_invalid, t_created, t_expired, data JSON)` with appropriate indices achieves sub-10ms for 1-3 hop traversals on 10K-node graphs.

The modernc.org/sqlite driver adds ~15-20MB to binary size and runs 10-50% slower than CGo mattn/go-sqlite3, but enables cross-compilation to any Go target without a C compiler. For GOgent-Fortress's local-first deployment model, this tradeoff is worth it.

**bbolt complements SQLite** as a hot-path cache for the most frequently accessed memory blocks. Its mmap-based reads approach microsecond latency when cached, its memory footprint is minimal (10-30MB), and startup is under 50ms. Use bbolt for the "core memory blocks" always loaded into context (procedural rules, user profile, project context) and SQLite for the full semantic/episodic graph that requires search capabilities.

BadgerDB, while offering superior write throughput, demands 50-150MB baseline RAM and 100-500ms startup — unacceptable for an agent hook binary that must initialize in under 100ms. CayleyDB's development has effectively stalled (sporadic commits since 2020, not yet v1.0), making it unsuitable for production dependency.

---

## The Obsidian vault as a dual-purpose human interface layer

Obsidian's value for GOgent-Fortress is **not as a storage backend but as a human interface to agent-managed knowledge**. The official Obsidian CLI (released February 10, 2026) supports 100+ commands across files, search, tags, properties, links, templates, and sync — but it requires the Obsidian desktop app to be running via IPC, has a **22.8% silent failure rate** in independent testing (13 of 57 scenarios), and adds 200-500ms overhead per invocation. This disqualifies it as a performance-critical backend.

The correct architecture is **direct file I/O on an Obsidian-compatible vault**. Since vaults are plain Markdown files with YAML frontmatter and `[[wikilinks]]` in a folder, a Go program can read/write them directly in under 1ms per file. Excellent Go libraries exist for this: **goldmark** (CommonMark parser matching C reference performance), **goldmark-wikilink** (`[[...]]` syntax), **goldmark-obsidian** (full Obsidian Flavored Markdown), and **adrg/frontmatter** (YAML frontmatter parsing). Build a custom in-memory index at startup using `fsnotify` for real-time change detection.

The dual-purpose vault works like this: GOgent-Fortress writes memory nodes as Markdown files with structured YAML frontmatter (entity type, timestamps, embeddings reference, relationship links) and `[[wikilinks]]` as edges. Humans open the same vault in Obsidian to browse the knowledge graph visually, search with Dataview queries, and manually edit memories. The agent's writes appear instantly in Obsidian via file watching; human edits are picked up by GOgent-Fortress's fsnotify watcher.

**Practical vault structure:**
```
.gogent-vault/
├── entities/           # Semantic memory nodes
│   ├── people/
│   ├── projects/
│   └── concepts/
├── episodes/           # Episodic memory (daily logs)
│   └── 2026-02-17.md
├── procedures/         # Procedural memory (rules, conventions)
│   ├── project-conventions.md
│   └── user-preferences.md
├── communities/        # Community summaries
├── MEMORY.md           # Auto-synthesized memory brief (loaded at startup)
└── .gogent/
    ├── graph.db        # SQLite graph database (agent-only)
    ├── cache.db        # bbolt hot cache (agent-only)
    └── index/          # Search indices
```

Obsidian sees everything above `.gogent/` as normal vault content. The `.gogent/` directory is gitignored and holds the performance-critical databases that the agent uses internally.

---

## A startup memory loading strategy that eliminates discovery tool calls

The "first action on startup" pattern is the single highest-ROI optimization for reducing token spend. Based on Claude Code's production system and OpenClaw's architecture, GOgent-Fortress should implement a **three-phase startup sequence**:

**Phase 1 — Instant load (<10ms, from bbolt cache):** Core memory blocks always injected into the system prompt. These are agent-editable blocks covering: agent persona/role, user profile (preferences, constraints), active project context (current goals, blockers), and critical procedural rules. Total budget: **10-15% of context window** (~400-600 tokens). These blocks are stored in bbolt for microsecond reads and synced to the Markdown vault asynchronously.

**Phase 2 — Session-relevant retrieval (<50ms, from SQLite):** Query the temporal knowledge graph for memories relevant to the current session type. Use the hook event metadata (which file was opened, what command was run, what error occurred) as the query. Hybrid retrieval: BM25 keyword search (30% weight) + embedding cosine similarity (70% weight) via SQLite FTS5 + sqlite-vec extension. Return top-k results. Budget: **10-20% of context window**.

**Phase 3 — Lazy on-demand (<100ms per query):** Expose `memory_search` and `memory_get` tools to the agent for retrieving deeper context when needed. The agent decides when to call these based on its judgment, following Claude Cowork's pattern of model-driven retrieval decisions rather than automatic injection.

**Token budget allocation for a 200K context window:**
- System instructions: ~15% (30K tokens)
- Core memory blocks: ~10% (20K tokens)
- Retrieved memories: ~15% (30K tokens)
- **Active conversation: ~60% (120K tokens)**

This mirrors OpenClaw's approach of loading today + yesterday's daily logs plus MEMORY.md at startup, but extends it with graph-based relevance retrieval. The key insight from Mem0's production data: this approach achieves **90% token cost savings** compared to full-context loading while maintaining retrieval accuracy.

---

## Memory consolidation bridges the STM-LTM gap

The most critical architectural decision is how memories promote from short-term to long-term storage. Three proven consolidation strategies should be combined:

**Pre-compaction flush (adapted from OpenClaw):** When the session context approaches the compaction threshold (`contextWindow - reserveTokens - softThreshold`), inject a silent system turn: *"Session nearing compaction. Extract and persist any durable insights to the knowledge graph."* The agent writes semantic entities, updated relationships, and episode summaries to the graph. This costs one extra LLM turn but preserves information that would otherwise be lost to context truncation. OpenClaw reports this is usually a no-op (`NO_REPLY`) when no new knowledge exists.

**Sleep-time consolidation (adapted from Letta):** Between sessions, run an asynchronous consolidation pass. This is where the expensive LLM work happens — entity extraction, deduplication, relationship building, community summary generation — but latency doesn't matter because no user is waiting. Use a stronger model (Claude Sonnet) for consolidation quality while using a faster model (Claude Haiku) for real-time interactions. This directly mirrors hippocampal replay during sleep: memories formed during active interaction are replayed, consolidated, and integrated into the long-term knowledge structure.

**Decay-weighted scoring (adapted from MemoryBank):** Every memory gets a relevance score: `score = α × recency_decay(t) + β × frequency(access_count) + γ × importance(llm_scored) + δ × similarity(query)`. Recency follows Ebbinghaus exponential decay: `R = e^(-t/S)` where S = memory strength (reinforced by each retrieval). Memories below a threshold are deprioritized in retrieval but never deleted — following Graphiti's principle that edges are invalidated, not removed. This naturally handles the "forgetting" problem: stale, unreinforced memories fade from active retrieval while remaining available for historical queries.

The consolidation pipeline for each session:

```
[Active Session] → raw conversation stored as episode
       ↓ (pre-compaction flush if context limit approached)
[Agent extracts] → key entities, facts, decisions, rules
       ↓ (session end)
[Sleep-time agent] → entity resolution, dedup, temporal tagging,
                     community update, procedural rule extraction
       ↓
[Long-term graph] → bi-temporal knowledge graph in SQLite
       ↓ (async)
[Markdown export] → updated vault files for human inspection
```

---

## Licensing analysis reveals a clear safe path

The licensing landscape creates sharp constraints that eliminate certain options entirely and strongly favor others.

**Apache 2.0 (Graphiti, BadgerDB)** is unambiguously safe. It permits proprietary derivative works, requires only license inclusion and NOTICE file preservation, and grants an explicit patent license. Reimplementing Graphiti's *concepts* (not code) in Go creates zero obligations — copyright protects expression, not ideas. The bi-temporal model, episodic/semantic separation, hybrid search, and community detection are all established computer science techniques documented in academic literature.

**Neo4j Community Edition runs GPLv3** — a strong copyleft license that would require open-sourcing GOgent-Fortress if Neo4j CE is embedded or distributed with it. The Neo4j Go *driver* is Apache 2.0 (safe), but the database server itself is GPL. Neo4j has demonstrated willingness to litigate aggressively (PureThink ordered to pay $597K in July 2024). **Neo4j should be offered only as an optional external service, never embedded.** For the recommended architecture, Neo4j is unnecessary.

**Obsidian's Terms of Service** contain restrictions on using "the Services or Software to provide a service for others" and on creating "derivative works based on or otherwise modify the Services or Software." However, **reading and writing standard Markdown files in a folder is not using Obsidian's software.** The file format (Markdown + YAML frontmatter + wikilinks) is not proprietary. The safest approach: GOgent-Fortress reads/writes Obsidian-compatible files directly via Go libraries. Market it as "compatible with Obsidian vault format" (nominative fair use) without implying endorsement. Obsidian became free for all commercial use in February 2025, but this applies to *using Obsidian the app*, not to third-party integrations.

**FalkorDB uses SSPL** (Server Side Public License) — not OSI-approved, restricts offering as a service. Avoid as a dependency for a proprietary framework.

**Risk summary for the recommended stack:**

| Component | License | Risk |
|-----------|---------|------|
| SQLite (modernc.org) | Public domain (SQLite) + BSD (Go wrapper) | **None** |
| bbolt | MIT | **None** |
| goldmark + extensions | MIT | **None** |
| BadgerDB (if used) | Apache 2.0 | **Very low** — attribution only |
| Obsidian file format | N/A (not copyrightable) | **None** |
| Graphiti patterns | N/A (independent reimplementation) | **None** if documented |

---

## Recommended technology stack and architecture

The final recommendation is a **three-layer architecture** that maximizes IP ownership, achieves sub-100ms performance, and provides the dual-purpose Obsidian experience:

**Layer 1 — Proprietary Temporal Knowledge Graph (Go, SQLite + bbolt)**
The core innovation. Independently implement a bi-temporal knowledge graph in Go with: entity nodes (typed, with embeddings and summaries), relationship edges (with four temporal timestamps), episode records (raw inputs, non-lossy), and community clusters (via label propagation). SQLite stores the full graph with FTS5 indices and sqlite-vec for vector similarity. bbolt caches the hot-path core memory blocks. All graph operations go through a proprietary `GraphStore` interface that can be swapped to Neo4j/FalkorDB via adapters if users want external graph databases.

**Layer 2 — Memory Orchestration Engine (Go, proprietary)**
Implements the three-phase startup loading, pre-compaction flush, sleep-time consolidation, decay-weighted scoring, and hybrid retrieval (BM25 + vector + BFS traversal with RRF reranking). Manages the STM → LTM promotion pipeline. Exposes `memory_search` and `memory_get` tools to agents. Handles token budget management and hierarchical summarization. This is where the framework's core IP lives.

**Layer 3 — Obsidian-Compatible Vault Interface (Go, goldmark + frontmatter)**
Async bidirectional sync between the SQLite graph and Markdown files. Writes entity nodes as `entities/{type}/{name}.md` with YAML frontmatter containing all structured metadata. Wikilinks encode relationships. Frontmatter encodes temporal data. fsnotify watches for human edits and propagates changes back to the graph. Humans get a fully browsable, searchable knowledge base in Obsidian with graph visualization, Dataview queries, and direct editing capabilities.

**What to build first:** Start with the SQLite graph schema and the three-phase startup loader — these deliver immediate value by replacing the current JSONL + handoff-document system with semantically rich, graph-queryable memory. Add the consolidation pipeline second. Add the Obsidian vault sync third. The existing JSONL scopes can serve as the migration source: import session scope as episodes, project scope as semantic entities, global scope as procedural rules, and ML scope as a new telemetry subgraph.

**Estimated development complexity:** The SQLite graph layer (schema, CRUD, search) is ~2 weeks. The startup loader + hook integration is ~1 week. The consolidation pipeline is ~2 weeks. The Obsidian sync layer is ~1 week. Total: **6-8 weeks** for a working prototype, compared to months of integration pain if adopting Graphiti as a Python dependency in a Go framework. The result is a fully proprietary, embeddable, zero-external-dependency memory system that outperforms the current JSONL approach by orders of magnitude in semantic richness while maintaining the sub-100ms retrieval requirement.

## Conclusion

The agent memory space has matured enough in 2024-2025 that clear architectural patterns have emerged from production systems. The convergence of Claude Code, OpenClaw, Letta, and Graphiti on typed memory stores + hybrid retrieval + async consolidation validates this as the winning architecture. GOgent-Fortress's unique advantage is implementing this entirely in Go as an embedded system — no Python runtime, no external databases, no network calls for memory operations. The bi-temporal knowledge graph pattern from Graphiti is the most architecturally sophisticated approach available, and reimplementing it in Go with SQLite + bbolt delivers better performance (sub-10ms cached reads) than Graphiti's own Neo4j-backed implementation (300ms P95 in production). The Obsidian-compatible Markdown layer transforms what would be an opaque agent database into a human-inspectable, human-editable knowledge graph — a genuine differentiator that no existing agent framework offers. The licensing path is clean: public domain SQLite, MIT bbolt, independently developed graph algorithms, and Obsidian interoperability through standard file formats rather than software dependencies.