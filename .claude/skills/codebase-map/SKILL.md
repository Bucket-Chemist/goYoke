---
name: codebase-map
description: Map codebase structure to JSON + ARCHITECTURE.md with Mermaid diagrams
---

# /codebase-map

Run the codebase mapping binary. All stages handled by `goyoke-codebase-extract`.

## Prerequisites

- `goyoke-codebase-extract` binary (run `make install-codebase-extract`)
- `ctags` for multi-language support: `pacman -S ctags` (optional — Go works without it)
- `ANTHROPIC_API_KEY` env var for LLM enrichment (optional — deterministic extraction works without it)

## Languages Supported

| Language | Engine | Call Graph |
|---|---|---|
| Go | go/ast + go/types (native, compiler-grade) | Yes (cross-module) |
| Rust | ctags | No |
| TypeScript | ctags | No |
| Python | ctags | No |
| R | ctags | No |

## Usage

```bash
# Deterministic extraction + graph + Mermaid diagrams (no LLM, free)
goyoke-codebase-extract --path=. --render

# Full pipeline with LLM enrichment + ARCHITECTURE.md (requires ANTHROPIC_API_KEY)
goyoke-codebase-extract --path=. --enrich --narrate --render

# With cost cap
goyoke-codebase-extract --path=. --enrich --narrate --render --budget=2.00

# Incremental (only changed files since last map)
goyoke-codebase-extract --path=. --incremental --render

# Fast mode (skip call graph)
goyoke-codebase-extract --path=. --skip-call-graph --render
```

## Output

| Path | Content | Committed? |
|---|---|---|
| `.claude/codebase-map/extract/` | Per-module extraction JSON | No (runtime) |
| `.claude/codebase-map/enriched/` | Per-module enriched JSON | No (runtime) |
| `.claude/codebase-map/graph.json` | Cross-module topology graph | No (runtime) |
| `.claude/codebase-map/manifest.json` | Extraction state + timestamps | No (runtime) |
| `docs/codebase-architecture/{repo}/ARCHITECTURE.md` | Architecture document | Yes |
| `docs/codebase-architecture/{repo}/diagrams/*.mmd` | Mermaid diagrams | Yes |

## Incremental Mode

Re-extract only files that changed since the last map.

```bash
# Full extraction first (creates manifest)
goyoke-codebase-extract --path=. --render

# Subsequent runs: only changed files
goyoke-codebase-extract --path=. --incremental --render
```

Change detection: git diff (preferred) → mtime fallback.
Exits 0 with "no changes detected" when nothing changed.

## Context Injection

When `GOYOKE_CODEBASE_MAP_INJECT=1` is set, spawned implementation agents automatically receive relevant module context from graph.json in their prompts. Off by default (experimental).
