---
title: Session Lifecycle
type: concept
created: 2026-04-18
tags: [session, hooks, handoff, archive]
related: [hook-system, ml-telemetry]
---

# Session Lifecycle

Every Claude Code session follows a structured lifecycle managed by hooks. Sessions produce artifacts that persist across conversations.

## Phases

### 1. Start (SessionStart → `goyoke-load-context`)
- Detects project language from file patterns
- Loads conventions from `.claude/conventions/`
- Restores handoff from previous session (`memory/last-handoff.md`)
- Injects git context (branch, status, recent commits)

### 2. During Session
- **Every tool:** ML telemetry logged by `goyoke-sharp-edge`
- **Every 10 tools:** Routing compliance reminder injected
- **On failures:** Sharp edge tracking (3+ consecutive → execution blocked)
- **PreToolUse:** Validation by `goyoke-validate`, `goyoke-skill-guard`

### 3. End (SessionEnd → `goyoke-archive`)
- Handoff generated to `memory/handoffs.jsonl`
- Human-readable summary to `memory/last-handoff.md`
- Session metrics captured

## Session Directory

Each session gets a unique directory: `.goyoke/sessions/{uuid}/`. The symlink `.goyoke/tmp/` always points to the current session directory.

## See Also

- [[hook-system]] — Hook infrastructure
- [[ml-telemetry]] — Telemetry captured during sessions
- [[ARCHITECTURE#8. Handoff Schema v1.3]] — Handoff format
