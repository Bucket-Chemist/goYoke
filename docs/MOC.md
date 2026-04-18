---
title: goYoke — Map of Content
type: moc
created: 2026-04-18
updated: 2026-04-18
tags: [moc, index, goYoke]
---

# goYoke — Map of Content

Entry point for navigating goYoke documentation. All paths are relative to the project root.

---

## Architecture

- [[ARCHITECTURE]] — Systems architecture v2.0 (18 sections, 1565 lines)
- [[systems-architecture-overview]] — High-level overview diagram

## Concepts

- [[concepts/hook-system]] — Hook lifecycle, event types, binary registration
- [[concepts/routing-tiers]] — Haiku / Sonnet / Opus tier selection and cost model
- [[concepts/agent-spawning]] — MCP spawn_agent, BuildFullAgentContext, validation
- [[concepts/ipc-bridge]] — UDS side-channel between TUI and MCP server
- [[concepts/tui-architecture]] — Bubbletea two-process topology, sharedState
- [[concepts/session-lifecycle]] — SessionStart → tool tracking → SessionEnd archive
- [[concepts/ml-telemetry]] — Routing decisions, agent collaborations, sharp edges
- [[concepts/distribution-model]] — Multicall binary, embedded FS, community agents

## Configuration

- [[hook-configuration]] — Hook setup and registration guide
- [[agent-config-quick-reference]] — Agent definition format and fields
- [[claude-cli-flags-reference]] — Claude Code CLI flags reference
- [[INSTALL-GUIDE]] — Installation guide
- [[INSTALL-NEW-AGENT-GUIDE]] — Adding new agents

## Operations

- [[mcp-spawning-troubleshooting]] — MCP spawn_agent debugging
- [[PERMISSION_HANDLING]] — Permission system overview
- [[PERMISSION-SOLUTION-SUMMARY]] — Permission resolution patterns
- [[QUICK-REFERENCE-PERMISSIONS]] — Quick permission reference
- [[task-tools-reference]] — Task tool usage patterns

## TUI

- [[tui/index]] — TUI documentation index (if exists)
- [[SESSION-20260411-UX-OVERHAUL]] — UX overhaul session notes

## Teams & Workflows

- [[TEAM-RUN-FRAMEWORK]] — Team orchestration framework
- [[TICKET-SKILL-SETUP]] — Ticket skill configuration

## Planning & Decisions

- [[decisions/index]] — Decision records index (if exists)
- [[ticket-analysis-prompt]] — Ticket analysis methodology
- [[ASSUMPTION-VALIDATION]] — Assumption tracking
- [[ml-telemetry-gap-analysis]] — ML telemetry gap analysis

## Planned Features

- [[ARCHITECTURE#18. Planned Features]] — Synthesis extension, codebase map, distribution
- `tickets/MASTER-INDEX.md` — Master ticket index (76 tickets, 4 features)

---

*Navigate concept notes for atomic explanations. Navigate ARCHITECTURE.md for the full technical reference.*
