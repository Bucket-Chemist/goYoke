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

### v0.9.0 — Agent & team config editor

- [ ] Browse and edit agent configs in the agents tab (identity, triggers, model, tier)
- [ ] Create new agents from templates within the TUI
- [ ] Team config editor: add/remove waves, members, adjust budgets
- [ ] Stdin/stdout editor with live schema validation
- [ ] Preview team topology changes before execution

### v0.9.1 — Telemetry dashboard

- [ ] Agent performance charts (success rate, cost, duration over time)
- [ ] Workflow comparison views (which skills are most cost-effective)
- [ ] Team execution timeline visualization (wave parallelism, stalls, failures)
- [ ] Sharp edge trend analysis (recurring patterns across sessions)
- [ ] Provider comparison heatmaps (quality × cost × latency)
- [ ] Export to CSV/JSON for external analysis

### v0.9.2 — Memory persistence

- [ ] Persistent memory backend (decisions, sharp edges, learnings across sessions)
- [ ] SessionStart hook injects relevant memories from previous sessions
- [ ] Automatic learning capture from agent results
- [ ] Memory deduplication and decay (stale memories surfaced less frequently)

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

## Contributing to the Roadmap

Features are tracked as tickets in the repo. To propose a new feature or reprioritize, open an issue with:

- **What:** One-sentence description
- **Why:** What problem it solves or what it enables
- **Dependencies:** Which version/feature it depends on
- **Effort estimate:** T-shirt size (S/M/L/XL)

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
