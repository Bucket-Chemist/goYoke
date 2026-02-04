```yaml
---
id: MCP-SPAWN-011
title: Review-Orchestrator Update
description: Update review-orchestrator to use spawn_agent for parallel reviewer spawning.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009]
phase: 2
tags: [orchestrator, review, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-011: Review-Orchestrator Update

## Description

Update the review-orchestrator to use MCP spawn_agent for spawning parallel reviewers (backend, frontend, standards, architect).

**Source**: Staff-Architect Analysis v2 §Part 6

## Task

1. Update review-orchestrator prompt to use spawn_agent
2. Spawn reviewers in parallel
3. Collect all results (handle partial failures)
4. Test full review workflow

## Acceptance Criteria

- [ ] review-orchestrator uses spawn_agent for reviewers
- [ ] Parallel spawning works (all reviewers start simultaneously)
- [ ] Partial failures handled (continue if 1 reviewer fails)
- [ ] All findings collected and synthesized
- [ ] Full /review workflow completes

