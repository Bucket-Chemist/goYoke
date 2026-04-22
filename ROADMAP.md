# Roadmap

goYoke is built for Claude Code at launch. The architecture is designed to generalize across providers — the hook system, contract schemas, and agent routing are provider-neutral by design, even though the current runtime targets Claude Code CLI.

This roadmap is ordered by dependency chain and user impact.

---

## v0.5.0 — Initial Public Release (current)

Single-provider (Claude Code), full enforcement stack.

- [x] 78 specialized agents across 4 model tiers
- [x] 11 compiled Go hooks enforcing routing, delegation, and compliance
- [x] Team orchestration with typed stdin/stdout contracts
- [x] Multi-agent workflows: braintrust, review, implement, plan-tickets, cleanup
- [x] Terminal UI with agent visualization, cost tracking, session persistence
- [x] Convention system for Go, Python, TypeScript, React, Rust, R
- [x] ML telemetry capture (routing decisions, agent collaborations, sharp edges)
- [x] Cross-platform: Linux, macOS, Windows
- [x] Distribution: Homebrew, AUR, standalone binary, go install

---

## v0.6.0 — Provider Abstraction + Codex

**Goal:** Onboard users who don't use Claude. OpenAI Codex is the largest AI dev tool audience — supporting it first maximizes reach.

**Why this comes first:** Every other multi-provider feature depends on the abstraction layer. Build it once, validate with Codex, then every subsequent provider is an adapter.

### v0.6.0 — Provider abstraction interface

- [ ] `ProviderAdapter` interface: spawn session, inject context, capture output
- [ ] `ModelCapabilities` declarations: which providers support which tiers
- [ ] Provider-neutral agent definitions (agents declare required capabilities, not model names)
- [ ] Refactor CLIDriver to use adapter interface (Claude adapter = current behavior)
- [ ] MCP tool routing through provider-agnostic spawn layer

### v0.6.1 — OpenAI Codex adapter

- [ ] Codex CLI adapter (`codex-adapter` binary or built-in)
- [ ] Tool-use mapping: goYoke MCP tools → Codex tool format
- [ ] Stdin/stdout contract validation works with Codex agent output
- [ ] Convention injection via Codex's context mechanism

### v0.6.2 — Cross-provider routing

- [ ] Route different agents to different providers in the same session
- [ ] Example: Opus architect on Claude, Sonnet workers on Codex
- [ ] Cost normalization across providers in telemetry
- [ ] Provider preference configuration (per-agent, per-tier, or global)

### v0.6.3 — MCP tools + stdio capture for inter-provider calls

- [ ] Capture stdin/stdout contracts across provider boundaries
- [ ] Validate typed contracts regardless of which provider executed the agent
- [ ] Provider-aware spawn_agent: MCP server selects adapter based on routing config
- [ ] Fallback handling: if preferred provider is unavailable, route to alternative

---

## v0.7.0 — Open Source & Local LLMs

**Goal:** Free-tier option. Run scouts and simple agents locally, reserve cloud providers for complex work.

### v0.7.0 — Local LLM adapter

- [ ] Ollama adapter (most popular local runtime)
- [ ] llama.cpp / vLLM adapter for direct model hosting
- [ ] Capability mapping: which local models handle which tiers (e.g., Qwen-32B for Sonnet-equivalent)
- [ ] Graceful degradation: if local model can't handle a task, escalate to cloud provider

### v0.7.1 — Cloud open source providers

- [ ] Together.ai adapter
- [ ] Fireworks AI adapter
- [ ] Groq adapter (for speed-optimized routing)
- [ ] Provider latency tracking in telemetry

### v0.7.2 — Intelligent provider routing

- [ ] Cost/quality/latency tradeoff configuration
- [ ] Historical telemetry-driven provider selection ("this agent performs best on provider X")
- [ ] Budget-aware routing: shift to cheaper providers as session budget depletes
- [ ] Per-project provider preferences

---

## v0.8.0 — Benchmark Suite & Data-Driven Optimization

**Goal:** Measure everything. With multiple providers, you need data to make routing decisions — not intuition.

### v0.8.0 — Agent benchmark suite

- [ ] SkillsBench integration: standardized evaluation for each agent
- [ ] Per-provider agent quality scoring (same task, different providers, compare output)
- [ ] Regression detection: alert when a provider update degrades agent quality
- [ ] Benchmark history stored alongside telemetry

### v0.8.1 — Team workflow benchmarks

- [ ] Reproducibility scoring: run same team config N times, measure output variance
- [ ] Contract compliance rates: what percentage of agent outputs validate against schema
- [ ] Wave timing analysis: identify bottlenecks in multi-wave workflows
- [ ] Cost efficiency comparison: same workflow, different provider configurations

### v0.8.2 — Hybrid benchmarks: agent categories, team operations, stdio extensions

- [ ] Expand agent categories beyond language-specific: planning, review, analysis, synthesis, orchestration
- [ ] Category-level quality metrics (do all reviewers meet the same bar? do all planners produce valid dependency graphs?)
- [ ] Team configuration benchmarks: same operation type, different team topologies (2-wave vs 3-wave, parallel vs sequential)
- [ ] Stdin/stdout schema extensions for benchmark capture: embed quality signals, timing, and token usage in the contract itself
- [ ] Operation-type scoring: compare `/review` vs `/braintrust` vs `/plan-tickets` across providers and configurations
- [ ] Golden-output comparison: diff agent stdout against known-good reference outputs per category

### v0.8.3 — Telemetry-driven routing optimization

- [ ] Merge benchmark data with routing telemetry for automated provider selection
- [ ] Confidence scores on routing decisions (data-backed, not heuristic)
- [ ] A/B routing: split traffic between providers and compare outcomes
- [ ] Sharp edge correlation: which providers trigger which failure patterns

---

## v0.9.0 — TUI Power User Features

**Goal:** Do everything from the TUI. No more editing JSON files to configure agents or teams.

### v0.9.0 — Provider selection UX + agent/team config editor

- [ ] Provider selector dropdown at session start (Bubbletea component: `internal/tui/components/provider/selector.go`)
- [ ] Live provider status: connected/disconnected, latency, cost-per-1K, model version
- [ ] Per-agent provider indicator in agent panel (which provider ran each agent, with cost)
- [ ] "Best fit" recommendations from v0.8 telemetry displayed alongside provider options
- [ ] User preference override: pin specific agents to specific providers via TUI config
- [ ] Provider configuration panel: add/remove providers, set credentials, test connectivity
- [ ] Browse and edit agent configs in the agents tab (identity, triggers, model, provider preference)
- [ ] Create new agents from templates within the TUI
- [ ] Team config editor: add/remove waves, members, adjust budgets, set per-member provider
- [ ] Stdin/stdout editor with live schema validation
- [ ] Preview team topology changes before execution

### v0.9.1 — Telemetry dashboard

- [ ] Agent performance charts (success rate, cost, duration over time) grouped by provider
- [ ] Workflow comparison views (which skills are most cost-effective on which provider)
- [ ] Team execution timeline visualization (wave parallelism, stalls, failures)
- [ ] Sharp edge trend analysis (recurring patterns across sessions)
- [ ] Provider comparison heatmaps (quality × cost × latency per agent category)
- [ ] "What if" simulator: estimate cost of running current workflow on different provider mix
- [ ] Export to CSV/JSON for external analysis

---

## v1.0.0 — Stable

**Commitment:** Contract schemas, hook interface, provider adapter API, and configuration format are frozen. Breaking changes follow SemVer major bumps.

- [ ] All stdin/stdout schemas at v1.0
- [ ] Provider adapter interface at v1.0
- [ ] Hook event interface at v1.0
- [ ] Agent definition format at v1.0
- [ ] Team config format at v1.0
- [ ] Comprehensive documentation for all public APIs

---

## v1.1.0 — Obsidian Graph-Based Memory

**Goal:** Replace the flat JSONL memory system with a structured temporal knowledge graph that provides semantically rich, graph-queryable memory with sub-100ms retrieval — and gives humans a browsable, editable view via an Obsidian vault.

**Why this is v1.1 (not earlier):** The knowledge graph is a major new subsystem with its own schema, consolidation pipeline, and embedding infrastructure. It depends on a stable hook interface and agent contract system (v1.0). It doesn't change existing APIs — it adds a new memory backend behind a feature flag (`GOYOKE_MEMORY_BACKEND=graph`).

**Full design:** See `tickets/obsidian-cli-knowledge-graph/SCOPE-v4.md` (1000+ lines, braintrust-reviewed).

### v1.1.0 — Core storage layer

Phase 0 (validation) + Phase 1 (deployment) + Phase 2 (storage) from SCOPE-v4.

- [ ] SQLite graph store via `modernc.org/sqlite` (pure Go, no CGO): nodes, edges (bi-temporal), episodes
- [ ] FTS5 full-text search with AFTER sync triggers for BM25 keyword retrieval
- [ ] `chromem-go` vector index for embedding similarity search (<8ms at 10K vectors)
- [ ] `atomic.Pointer[map]` hot cache for core memory blocks (<1ms reads, zero contention)
- [ ] IR-based composite scoring: BM25 (30%) + cosine similarity (50%) + frequency (10%) + PageRank (10%)
- [ ] Reciprocal Rank Fusion for merging BM25 and vector result sets (upgrade path to Convex Combination)
- [ ] Dual connection pools (N readers + 1 writer) with WAL mode
- [ ] `GraphStore`, `VectorIndex`, `VaultSync` Go interfaces for testability and future backend swaps
- [ ] Pre-implementation validation: nomic-embed-text normalization, SQLite cold-start latency, goldmark extension coexistence

### v1.1.1 — Obsidian vault + consolidation pipeline

Phase 3 from SCOPE-v4.

- [ ] Write-only vault export: entities, episodes, procedures, communities as Obsidian-compatible Markdown
- [ ] YAML frontmatter with content hashes for change detection
- [ ] `[[wikilinks]]` as graph edges — browsable in Obsidian Graph View
- [ ] Batch import: detect human edits between sessions via content hash diffing
- [ ] Hybrid goldmark parsing stack for vault import (powerman/goldmark-obsidian + callout extension)
- [ ] Session-end consolidation pipeline: entity extraction (Claude Sonnet) → entity resolution (embedding-first, tiered OR-logic with context-enriched embeddings) → graph algorithms (PageRank, Louvain community detection via gonum) → pre-computed features written back to SQLite
- [ ] Auto-synthesized `MEMORY.md` brief (first 200 lines injected at startup)

### v1.1.2 — Hook integration + JSONL migration

Phase 4 from SCOPE-v4.

- [ ] Extend `goyoke-load-context` with graph memory retrieval (no new binary)
- [ ] Three-phase startup: core blocks (<10ms, atomic cache) → session-relevant retrieval (<50ms, hybrid query) → lazy on-demand tools
- [ ] `GOYOKE_MEMORY_BACKEND` env var: `jsonl` (default, preserves existing behavior) or `graph` (knowledge graph active)
- [ ] Migrate existing JSONL memories: handoffs → episodes, decisions → semantic nodes, sharp edges → procedural nodes
- [ ] JSONL originals preserved as `.jsonl.bak` (instant rollback: set env var back to `jsonl`)
- [ ] Total startup overhead with graph backend: <100ms (72ms budget after baseline)
- [ ] `memory_search` and `memory_get` MCP tools exposed to agents for on-demand retrieval

### Vault structure

```
.goyoke-vault/                  ← Obsidian vault (human-browsable)
├── entities/
│   ├── agents/
│   ├── concepts/
│   ├── decisions/
│   └── errors/
├── episodes/
├── procedures/
├── communities/
└── MEMORY.md                   ← auto-synthesized, loaded at startup

.goyoke/graphdb/                ← Runtime (agent-only, gitignored)
├── graph.db                    ← SQLite
└── index/                      ← chromem-go persistence
```

---

## Contributing to the Roadmap

Features are tracked as tickets in the repo. To propose a new feature or reprioritize, open an issue with:

- **What:** One-sentence description
- **Why:** What problem it solves or what it enables
- **Dependencies:** Which version/feature it depends on
- **Effort estimate:** T-shirt size (S/M/L/XL)

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
