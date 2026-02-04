```yaml
---
id: MCP-SPAWN-010
title: Mozart Orchestrator Update
description: Update Mozart to use spawn_agent for spawning Einstein and Staff-Architect.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009, MCP-SPAWN-013]
phase: 2
tags: [orchestrator, braintrust, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-010: Mozart Orchestrator Update

## Description

Update the Mozart orchestrator (Braintrust skill) to use MCP spawn_agent for spawning Einstein and Staff-Architect instead of attempting Task().

**Source**: Einstein Analysis §3.6.1

## Task

1. Update Mozart prompt to use spawn_agent
2. Add parallel spawning for Einstein + Staff-Architect
3. Update error handling for spawn failures
4. Test full Braintrust workflow

## Files

- `~/.claude/skills/braintrust/SKILL.md` — Update Mozart instructions

## Implementation

### Updated Mozart Instructions

Mozart should be instructed to use spawn_agent like this:

```
When spawning Einstein and Staff-Architect, use the MCP spawn_agent tool:

// Spawn Einstein
mcp__gofortress__spawn_agent({
  agent: "einstein",
  description: "Theoretical analysis for Braintrust",
  prompt: `AGENT: einstein

BRAINTRUST WORKFLOW - THEORETICAL ANALYSIS

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000  // 10 minutes for complex analysis
})

// Spawn Staff-Architect (can be parallel with Einstein)
mcp__gofortress__spawn_agent({
  agent: "staff-architect-critical-review",
  description: "Practical review for Braintrust",
  prompt: `AGENT: staff-architect-critical-review

BRAINTRUST WORKFLOW - PRACTICAL REVIEW

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000
})
```

## Acceptance Criteria

- [ ] Mozart uses spawn_agent instead of Task() for Level 2 spawning
- [ ] Einstein spawns successfully via MCP
- [ ] Staff-Architect spawns successfully via MCP
- [ ] Both outputs collected correctly
- [ ] Beethoven synthesis works with collected outputs
- [ ] Full Braintrust workflow completes end-to-end

