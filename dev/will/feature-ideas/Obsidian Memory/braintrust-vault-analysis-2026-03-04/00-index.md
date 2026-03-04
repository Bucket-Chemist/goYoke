---
tags:
  - braintrust
  - index
  - vault-architecture
date: 2026-03-04
team_id: braintrust-1772589462
total_cost_usd: 2.67
status: complete
---

# Braintrust Analysis: Vault Architecture + Obsidian CLI

> **Date:** 2026-03-04 | **Team:** braintrust-1772589462 | **Cost:** $2.67

## Problem

Critical evaluation of the EM-Deconvoluter dev vault structure and scoping Obsidian CLI integration for agent operations — analyzed from both theoretical and practical perspectives.

## Verdict (Updated with Empirical Data)

- **CLI hot path:** Rejected (515ms mean latency = 7x hook budget, mutation surprises)
- **CLI cold path:** Adopted for vault maintenance (backlinks, unresolved, orphans — unique graph queries)
- **Direct file I/O:** Primary agent interface for all reads/writes
- **Source of truth:** Dual-authority model (authoring in vault, queries in graph)
- **Three-tier CLI strategy:** Never (mutations) / Maintenance (graph queries) / Optional (search, tags)

## Documents

| # | Document | Agent | Cost |
|---|----------|-------|------|
| 0 | [[00-problem-brief]] | Mozart | — |
| 1 | [[01-einstein-theoretical]] | Einstein (Opus) | $1.26 |
| 2 | [[02-staff-architect-practical]] | Staff-Architect (Opus) | $0.77 |
| 3 | [[03-beethoven-synthesis]] | Beethoven (Opus) | $0.64 |
| 4 | [[04-empirical-cli-test-results]] | Router (empirical) | — |

## Key Takeaways

1. **Seven transferable vault design principles** identified (Dual-Interface, Template-as-Schema, Vault Authority, Lifecycle-Aware, Cross-Reference, Index-as-API, Dependency-as-DAG)
2. **CLI nuanced** — not categorically rejected. Hot path rejected (515ms, mutation surprises), cold path adopted (backlinks, unresolved, orphans are unique capabilities)
3. **Dual-authority model** resolves source-of-truth tension: authoring in vault (always), queries in graph (when available)
4. **Memory tier mapping leaks at section boundaries** — design schema for section-level from day one
5. **VaultReader/VaultWriter interfaces** are cheap now (~3h) and convert SCOPE-v3 migration from rewrite to swap
6. **CLI reliability is ~91%** (not 77.2% as SCOPE-v3 claimed) — but mutations have critical surprises (create deduplication, move index lag)
7. **Three critical CLI-only capabilities:** backlinks, unresolved links, orphan detection — hard to replicate without full vault graph construction

## Related

- [[knowledge-graph-research]] — SQLite-based memory system research
- [[cli-commands]] — Obsidian CLI command reference
- [[Obsidian as a RAG]] — Earlier RAG exploration
