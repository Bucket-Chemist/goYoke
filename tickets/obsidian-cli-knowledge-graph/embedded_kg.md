# Building an embedded temporal knowledge graph in Go

**A pure Go stack combining SQLite, bbolt, and Obsidian-compatible Markdown can deliver a production-quality temporal knowledge graph memory system as a single zero-dependency binary.** The critical architectural decisions center on SQLite driver choice (modernc.org/sqlite for pure Go vs. ncruces/go-sqlite3 for sqlite-vec WASM support), dual connection pools for concurrent reads with serialized writes, and goldmark with abhinav's extension suite for Obsidian interop. This report synthesizes research across eight domains — from recursive CTE graph traversal to Ebbinghaus decay curves — to inform a complete subagent definition for the GOgent-Fortress framework.

---

## SQLite driver choice shapes every downstream decision

The first fork in the road is the SQLite driver. Three viable options exist for Go, each with distinct tradeoffs:

**modernc.org/sqlite** is the primary recommendation for a no-CGO single binary. It transpiles SQLite's C source to Go via ccgo/v4, currently wrapping SQLite 3.51.2. Benchmarks from cvilsmeier/go-sqlite-bench show it runs at roughly **75% of mattn/go-sqlite3's speed** for most operations, but critically, it is **faster than mattn for concurrent reads** (870ms vs 1149ms at N=2, scaling to 2139ms vs 2830ms at N=8). The CGO call overhead in mattn compounds under concurrency, while modernc avoids it entirely. FTS5, JSON1, RTree, and Session extensions are compiled in by default. The `vtab` package exposes a pure Go API for custom virtual table modules — essential for implementing vector search without CGO. Compilation is dramatically faster than mattn since no C toolchain is required.

**ncruces/go-sqlite3** is the alternative when sqlite-vec integration is non-negotiable. It runs SQLite compiled to WASM via wazero (pure Go WASM runtime), requiring no CGO. The sqlite-vec Go bindings explicitly support ncruces via `github.com/asg017/sqlite-vec-go-bindings/ncruces`. The tradeoff is higher memory overhead from WASM sandboxing and a less conventional architecture.

**The sqlite-vec incompatibility with modernc is the single most important constraint.** The sqlite-vec README states explicitly: "this will NOT work with modernc.org/sqlite." If vector search must live inside SQLite, you must use ncruces or mattn. If vector search can live in a Go-native library (recommended — see vector search section), modernc.org/sqlite is the better foundation.

The recommended PRAGMA configuration for either driver:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 10000;
PRAGMA cache_size = -32000;        -- 32MB
PRAGMA temp_store = MEMORY;
PRAGMA foreign_keys = ON;
PRAGMA mmap_size = 268435456;      -- 256MB
PRAGMA wal_autocheckpoint = 1000;
PRAGMA journal_size_limit = 67108864;  -- 64MB
```

**A critical operational finding**: modernc.org/sqlite **requires** `SetMaxOpenConns(>0)` for concurrency to work correctly. Without it, concurrent access may deadlock. The recommended pattern is dual connection pools — a write pool with `MaxOpenConns(1)` serializing all writes, and a read pool with `MaxOpenConns(runtime.GOMAXPROCS(0))` for concurrent reads. This mirrors SQLite's fundamental concurrency model: **one writer, many readers**.

---

## Property graph schema with bi-temporal versioning

The schema design combines the proven pattern from dpapathanasiou/simple-graph (1.5k stars) with bi-temporal modeling. The core tables:

```sql
CREATE TABLE nodes (
    id          INTEGER PRIMARY KEY,
    type        TEXT NOT NULL,
    name        TEXT NOT NULL,
    properties  TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(properties)),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now'))
) STRICT;

CREATE TABLE edges (
    id              INTEGER PRIMARY KEY,
    source_id       INTEGER NOT NULL REFERENCES nodes(id),
    target_id       INTEGER NOT NULL REFERENCES nodes(id),
    relation_type   TEXT NOT NULL,
    properties      TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(properties)),
    weight          REAL NOT NULL DEFAULT 1.0,
    valid_from      TEXT NOT NULL,
    valid_to        TEXT NOT NULL DEFAULT '9999-12-31T23:59:59.999',
    tx_from         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    tx_to           TEXT NOT NULL DEFAULT '9999-12-31T23:59:59.999',
    confidence      REAL NOT NULL DEFAULT 1.0,
    source_type     TEXT NOT NULL DEFAULT 'manual'
) STRICT;
```

**Composite indexes on `edges(source_id, relation_type)` and `edges(target_id, relation_type)` are essential** — graph traversals almost always filter by starting node plus edge type. For temporal queries, add `CREATE INDEX idx_edges_current ON edges(source_id, relation_type, tx_to, valid_to)` to efficiently find "current truth." The bi-temporal approach uses **valid time** (when the fact is true in reality) and **transaction time** (when the system recorded it). Querying "what was true at time T as we knew it at time S" requires filtering on both dimensions simultaneously. An append-only approach (inspired by Datomic) never UPDATEs or DELETEs facts — instead, expired facts get their `tx_to` set, and new versions are inserted.

JSON1 extension usage enables flexible property queries: `json_extract(properties, '$.name')` for lookups, `json_each(properties, '$.tags')` for array containment, and generated columns (`GENERATED ALWAYS AS (json_extract(...)) STORED`) with indexes for frequently-queried properties.

---

## Recursive CTEs and FTS5 power graph traversal and search

**Recursive CTEs** handle BFS and DFS graph traversal directly in SQL. The key differentiation is `ORDER BY depth ASC` for BFS versus `ORDER BY depth DESC` for DFS in the recursive term. Cycle prevention uses the `instr(',' || path || ',', ',' || target_id || ',') = 0` pattern to track visited nodes in a comma-separated path string. An N-hop neighbor query with depth limit looks like:

```sql
WITH RECURSIVE neighbors(node_id, depth, path) AS (
    SELECT :start_id, 0, CAST(:start_id AS TEXT)
    UNION ALL
    SELECT e.target_id, n.depth + 1, n.path || ',' || e.target_id
    FROM edges e JOIN neighbors n ON e.source_id = n.node_id
    WHERE n.depth < :max_hops
      AND e.tx_to = '9999-12-31T23:59:59.999'
      AND instr(',' || n.path || ',', ',' || e.target_id || ',') = 0
)
SELECT DISTINCT node_id, MIN(depth) FROM neighbors GROUP BY node_id;
```

**Important limitations**: SQLite's default recursion depth is 1,000 rows. The path-tracking approach prevents cycles but doesn't prevent visiting the same node via different paths, causing exponential blowup in dense graphs. For graphs exceeding **~10K edges, perform traversal in Go code** rather than SQL. True Dijkstra's (weighted shortest path) cannot be efficiently implemented in a single recursive CTE because SQLite's recursive CTEs are append-only.

**FTS5** with BM25 ranking provides keyword search. Use an external content table pointing to the nodes table, with triggers to keep the FTS index synchronized. The tokenizer `porter unicode61` enables stemming with Unicode support. Column weights are tunable via `bm25(10.0, 1.0)` arguments (weight name 10x over description). BM25 returns negative scores where lower means better match. Both modernc.org/sqlite and ncruces include FTS5 by default with no build tags required.

---

## bbolt as a hot cache demands careful transaction discipline

bbolt (go.etcd.io/bbolt, ~9.2k stars) provides an ideal hot cache layer: pure Go, ACID transactions, memory-mapped reads, and zero-copy `Get()` operations. The recommended architecture is **write-through caching with generation-based invalidation**: write to SQLite first (source of truth), then update bbolt, with a monotonic generation counter for bulk invalidation.

**Flat buckets with composite keys outperform nested buckets** for a cache. Store nodes in a `"nodes"` bucket keyed by big-endian uint64 IDs, edges in `"edges"` with composite `srcID+dstID` keys. Use `encoding/json` initially for serialization — it's zero-dependency and human-readable for debugging. If profiling reveals bottlenecks, `github.com/vmihailenco/msgpack/v5` delivers ~30-40% smaller output at 2-3x the speed without requiring code generation.

The most dangerous bbolt pitfall is **byte slice lifetime**: values returned by `Bucket.Get()` point directly into mmap'd memory and are only valid during the transaction. Using them after the transaction closes causes `unexpected fault address` panics. The Lightning Network's lnd project hit this exact bug (PR #6547). Always `copy(result, v)` within the transaction.

Four rules prevent bbolt deadlocks: never nest transactions (read inside write or vice versa in the same goroutine), keep transactions short, set `InitialMmapSize` large enough (256MB+) to avoid frequent remapping, and copy data out of read transactions before performing writes. bbolt files **never shrink** — deleted data leaves free pages but the file stays large. Periodic compaction (copying to a new database) is the only remedy.

---

## Vector search works best outside SQLite for a pure Go stack

Since sqlite-vec is incompatible with modernc.org/sqlite, the recommended approach separates vector search from SQLite:

**chromem-go** (`github.com/philippgille/chromem-go`) is the strongest candidate: pure Go, zero dependencies, in-memory with optional persistence, and built-in embedding functions for Ollama, OpenAI, and Cohere. Benchmarks on an Intel i5 show **100 docs in 90µs, 1K in 520µs, 10K in ~5.3ms, and 100K in ~40ms** — viable for a desktop knowledge graph. It uses exhaustive nearest-neighbor search (no ANN indexing), which is perfectly adequate below 100K documents.

**coder/hnsw** (`github.com/coder/hnsw`) provides O(log n) search via a pure Go HNSW implementation if ANN acceleration is needed at scale. It offers binary encoding for fast serialization (~1.2 GB/s export) and a simple API. **sqvect** (`github.com/liliang-cn/sqvect`) is worth investigating as a combined solution — pure Go, single SQLite file, HNSW vector search plus FTS5 plus graph relationships in one package.

For cosine similarity, **pre-normalize vectors at storage time** (Ollama returns L2-normalized vectors by default), reducing cosine similarity to a dot product. The SIMD-accelerated library **viterin/vek** delivers an average **10x speedup** for float32 operations on x86 with AVX2/FMA, falling back to pure Go on unsupported platforms. For embedding generation, Ollama with `nomic-embed-text` (768 dimensions, 8K context) provides the best balance of accuracy, speed, and local operation. Store embeddings in SQLite as BLOBs alongside content hashes, with a `model_version` column to handle embedding model migrations.

---

## Goldmark with abhinav's extensions handles Obsidian interop

**goldmark** (~4.4k stars, used by Hugo) is the clear choice for Markdown parsing: CommonMark 0.31 compliant, pure Go, performance on par with C's cmark implementation (4.20ms/op with best-in-class memory at 2.56MB/op and 13,435 allocations). The Obsidian-compatible extension stack:

- **Wikilinks**: `go.abhg.dev/goldmark/wikilink` — parses `[[links]]`, `[[link|display]]`, `![[embeds]]`, `[[Page#Section]]`
- **Hashtags**: `go.abhg.dev/goldmark/hashtag` with `ObsidianVariant` mode
- **Frontmatter**: `go.abhg.dev/goldmark/frontmatter` — YAML and TOML, integrated into goldmark's AST pipeline
- **Callouts**: `github.com/VojtaStruhar/goldmark-obsidian-callout` — all Obsidian callout types
- **GFM built-ins**: Tables, strikethrough, task lists, linkify

No single library covers all Obsidian Flavored Markdown. **Highlights (`==text==`) and comments (`%%...%%`) require custom parsers** (~50-100 lines each). The `powerman/goldmark-obsidian` combined package exists but has only 6 stars and TODO items for highlights and callouts.

For **bidirectional sync**, fsnotify provides cross-platform file watching but requires manual recursive directory walking and event debouncing at 100-200ms to handle the 2-5 duplicate events per file save. The feedback loop prevention pattern: maintain a `pendingWrites` map, mark files before writing, and ignore fsnotify events for those files within the debounce window. Use **content hashing (SHA-256)** rather than timestamps for change detection — it's more reliable across filesystems. Atomic file writes via `github.com/natefinch/atomic` prevent partial-read scenarios.

---

## Graph algorithms via gonum with memory consolidation scoring

**gonum.org/v1/gonum/graph** is the standard Go graph algorithm library, providing:

- **PageRank** and PageRankSparse via `graph/network` — supports edge-weighted variants
- **Connected Components** and Tarjan SCC via `graph/topo`
- **Louvain community detection** via `graph/community` — randomized, supports resolution parameter
- **BFS/DFS with depth limits** via `graph/traverse` — `BreadthFirst.Walk` passes depth to the `until` callback natively
- **Shortest paths**: A*, Dijkstra, Bellman-Ford, Floyd-Warshall via `graph/path`

**Label propagation is NOT in gonum** and requires a custom implementation (~30 lines: initialize each node with its own label, iteratively adopt the most frequent neighbor label). The recommended workflow: load the relevant subgraph from SQLite into a gonum `simple.DirectedGraph`, run algorithms in memory, then write results back. Performance for 100K nodes: PageRank ~50-200ms, connected components ~10-50ms, Louvain ~100-500ms.

For **memory consolidation**, the FSRS algorithm (now default in Anki 23.10+) uses power-law decay rather than exponential: **R(t, S) = (1 + t/(9·S))^(-1)**, which provides a longer tail than Ebbinghaus's exponential. Apply configurable decay rates per relationship type — factual relationships (half-life ~2 years, λ=0.001), professional relationships (~4.5 months, λ=0.005), contextual mentions (~2 weeks, λ=0.05). The composite relevance score combines `w_r × recency + w_f × frequency + w_i × pagerank + w_c × connectivity`.

**Entity extraction** works best with structured output / function calling, using a multi-stage pipeline: entities first, then candidate relations, then normalization. Schema-constraining the allowed node and relationship types significantly improves consistency. **Entity resolution** combines string similarity (Jaro-Winkler at 0.80-0.90 threshold via `github.com/hbollon/go-edlib`) with embedding similarity for a hybrid `match_score = α × string_sim + (1-α) × embedding_sim`. Track provenance per fact (`source_type`, `source_id`, `extraction_method`, `confidence`) and resolve temporal conflicts by recency for single-valued relationships, union for multi-valued ones.

---

## The sharp edges that will bite hardest

**SQLite's most dangerous pitfall**: a deferred transaction upgrading to write gets **instant SQLITE_BUSY regardless of busy_timeout**. The fix is `_txlock=immediate` in the DSN or using `BEGIN IMMEDIATE` explicitly. Without this, write transactions inside read transactions fail silently and unpredictably.

**Unclosed `rows` objects prevent WAL checkpointing**, causing the WAL file to grow indefinitely. Always use `QueryRow` for single-row results and `defer rows.Close()` for multi-row queries. Check `rows.Err()` after iteration. A single leaked rows object in a long-running process can consume gigabytes of disk.

**bbolt's byte-slice-after-transaction bug** causes `unexpected fault address` panics that are difficult to reproduce because they depend on when the OS remaps memory. Always copy bytes within the transaction closure.

**fsnotify fires 2-5 events per file save** and does not support recursive directory watching. Every editor has different save patterns — Vim writes a temp file then renames, VS Code may do multiple writes. Debouncing at 100-200ms with per-file timers is mandatory. On macOS, Spotlight generates spurious CHMOD events that should always be filtered.

**YAML frontmatter breaks on unquoted colons** (`title: The Best Food: Pizza` parses as a nested mapping). Boolean strings `yes`, `no`, `true`, `false` are interpreted as YAML booleans, not strings. Always quote values containing special characters.

**Vector dimension mismatches across embedding models** are irrecoverable — vectors from different models exist in incompatible spaces. Store `model_version` alongside every embedding and never mix models in similarity queries. When changing models, re-embed everything (for <100K entities, this takes minutes locally with Ollama).

**bbolt files never shrink** and long-running read transactions prevent page reclamation, causing unbounded file growth. On Linux kernels 5.10-5.16 with ext4 fast_commit, bbolt can suffer data corruption (fixed in 5.10.94+).

---

## Conclusion

The optimal pure Go stack for this system uses **modernc.org/sqlite** for the graph store (FTS5 + JSON1 + WAL mode + bi-temporal schema), **chromem-go** for vector search (separated from SQLite to avoid the sqlite-vec CGO dependency), **bbolt** for hot caching with write-through invalidation, **goldmark** with abhinav's extension suite for Obsidian Markdown interop, and **gonum/graph** for PageRank, community detection, and traversal algorithms. The choice to keep vector search outside SQLite is the key architectural insight — it unlocks the pure Go modernc driver while chromem-go's exhaustive search performs adequately up to 100K documents.

Three non-obvious findings deserve emphasis: modernc.org/sqlite **outperforms** CGO-based mattn for concurrent reads (the primary access pattern for a knowledge graph), FSRS's power-law decay is materially better than Ebbinghaus's exponential for long-term knowledge retention scoring, and the dual connection pool pattern (1 writer, N readers) is not optional for SQLite in Go — it's the only way to achieve both correctness and performance under concurrency. The subagent definition should encode these findings as hard constraints, with the sharp-edges.yaml capturing the byte-slice-lifetime, deferred-transaction-upgrade, and unclosed-rows pitfalls as explicit warnings with both wrong and correct code patterns.