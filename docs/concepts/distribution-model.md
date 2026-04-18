---
title: Distribution Model
type: concept
created: 2026-04-18
tags: [distribution, multicall, planned]
related: [hook-system, agent-spawning]
status: planned
---

# Distribution Model

> **Status:** Planned (39 tickets, 3 complete). See [[ARCHITECTURE#18.3 Distribution (39 tickets)]].

The distribution system packages goYoke's ~12 hook binaries into a single multicall binary, adds lifecycle commands (`init`, `upgrade`, `doctor`), and enables community agent sharing.

## Multicall Binary

Each `cmd/goyoke-*/main.go` exports a `HookMain()` function. The multicall dispatcher (`cmd/goyoke/`) routes based on `os.Args[0]` or subcommand name. Symlinks or `goyoke run <hook>` invoke individual hooks.

## Embedded FS

`goyoke init` bootstraps a project by extracting embedded copies of `.claude/` config files (agents, conventions, rules, skills, schemas). This is why all other features must land before DIST-009.

## Override System

Users can override any embedded config file by placing a copy in `.claude/overrides/`. `goyoke upgrade` preserves overrides while updating defaults.

## Community Agents

A manifest schema, catalog, and install/remove workflow allow sharing agent definitions. The TUI gets a panel for browsing and installing agents from GitHub.

## See Also

- [[ARCHITECTURE#18. Planned Features]] — All planned features
- `tickets/distribution/tickets-index.json` — Full ticket list
