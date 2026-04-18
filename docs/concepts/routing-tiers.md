---
title: Routing Tiers
type: concept
created: 2026-04-18
tags: [routing, tiers, cost-optimization]
related: [hook-system, agent-spawning]
---

# Routing Tiers

goYoke enforces a tiered model routing strategy to minimize cost while maintaining quality. The router (Opus) classifies requests and delegates to cheaper models for execution.

## Tier Hierarchy

| Tier | Model | Cost/1K tokens | Use For |
|------|-------|----------------|---------|
| Haiku | claude-haiku | ~$0.0005 | File search, grep, counting, boilerplate |
| Haiku+Thinking | claude-haiku | ~$0.001 | Scaffolding, docs, style-only review |
| Sonnet | claude-sonnet | ~$0.009 | Implementation, refactoring, debugging |
| Opus | claude-opus | ~$0.045 | Architecture, synthesis, planning |

## Enforcement

- `routing-schema.json` defines allowed tiers and thresholds
- `goyoke-validate` blocks tier violations (e.g., Task(opus) without allowlist)
- `goyoke-sharp-edge` injects routing reminders every 10 tools
- `goyoke-load-context` sets initial tier context at session start

## Cost Savings

Aggressive tiered routing saves ~70% on exploration workflows. A $0.02 Haiku scout prevents $0.50 Opus mis-routing.

## See Also

- [[ARCHITECTURE#7. Cost Tracking & Tier Pricing]] — Full pricing model
- [[agent-spawning]] — How agents are dispatched at each tier
