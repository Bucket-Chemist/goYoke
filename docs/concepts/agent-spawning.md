---
title: Agent Spawning
type: concept
created: 2026-04-18
tags: [agents, mcp, spawning]
related: [ipc-bridge, routing-tiers]
---

# Agent Spawning

Agents are spawned via the `mcp__goyoke-interactive__spawn_agent` MCP tool, not via Claude Code's built-in `Agent`/`Task` tool. This ensures full context injection (identity, conventions, rules).

## Why MCP, Not Agent/Task

The built-in `Agent` tool fires no PreToolUse hooks, so no conventions, rules, or agent identity would be injected. The MCP `spawn_agent` calls `routing.BuildFullAgentContext()` to inject full context before spawning `claude -p`.

## Context Injection

`pkg/routing/identity_loader.go:BuildFullAgentContext()` assembles:
- Agent identity (from `.claude/agents/<name>/<name>.md`)
- Language conventions (from `.claude/conventions/*.md`)
- Rules (from `.claude/rules/*.md`)
- Sharp edges (from `.claude/agents/<name>/sharp-edges.yaml`)

## Validation

`internal/tui/mcp/validator.go` performs bidirectional checks:
1. Does the child's `spawned_by` include the caller? 
2. Does the caller's `can_spawn` include the child?

## Nesting

Max 10 levels via `GOYOKE_NESTING_LEVEL` env var. Each spawn increments the counter.

## See Also

- [[ARCHITECTURE#Agent Spawning Architecture]] — Full spawning architecture
- [[mcp-spawning-troubleshooting]] — Debugging spawn failures
- [[ipc-bridge]] — How MCP server communicates with TUI
