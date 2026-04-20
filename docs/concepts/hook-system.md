---
title: Hook System
type: concept
created: 2026-04-18
tags: [hooks, enforcement, claude-code]
related: [routing-tiers, session-lifecycle]
---

# Hook System

goYoke hooks are Go binaries that intercept Claude Code events. They are the enforcement layer — routing rules, validation, telemetry, and archival all happen through hooks.

## Event Types

| Event | When | Example Hooks |
|-------|------|---------------|
| SessionStart | Session begins, resumes, or compacts | `goyoke-load-context` |
| PreToolUse | Before any tool executes | `goyoke-validate`, `goyoke-skill-guard`, `goyoke-direct-impl-check` |
| PostToolUse | After any tool executes | `goyoke-sharp-edge` |
| SubagentStop | When a spawned agent completes | `goyoke-agent-endstate`, `goyoke-orchestrator-guard` |
| SessionEnd | Session closes | `goyoke-archive` |
| ConfigChange | Settings modified | `goyoke-config-guard` |

## How Hooks Work

1. Claude Code fires an event with structured JSON on stdin
2. The hook binary parses the event (`pkg/routing/stdin.go`)
3. Hook logic runs (validate, log, transform)
4. Hook returns JSON on stdout (allow, block, or inject `additionalContext`)

## Registration

Hooks are registered in `.claude/settings.json` under the `hooks` array. Each entry specifies `type` (event), `event` (lifecycle stage), `matcher` (regex for tool name filtering), and `command` (path to Go binary).

## Key Packages

- `pkg/routing/` — Shared hook infrastructure (stdin parsing, response builders, schema loading)
- `pkg/session/` — Session state, handoff, artifacts
- `cmd/goyoke-*/` — Individual hook binaries

## See Also

- [[ARCHITECTURE#2. Hook Event Flow]] — Detailed event flow diagrams
- [[hook-configuration]] — Configuration guide
- [[routing-tiers]] — How hooks enforce tier selection
