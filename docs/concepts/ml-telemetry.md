---
title: ML Telemetry
type: concept
created: 2026-04-18
tags: [telemetry, ml, routing-decisions]
related: [hook-system, session-lifecycle, routing-tiers]
---

# ML Telemetry

goYoke captures structured telemetry for every routing decision, agent collaboration, and sharp edge. This data feeds future ML models for automated tier selection.

## Data Streams

| Stream | File | Captured By |
|--------|------|-------------|
| Routing decisions | `$XDG_DATA_HOME/goyoke/routing-decisions.jsonl` | `goyoke-sharp-edge` |
| Decision outcomes | `$XDG_DATA_HOME/goyoke/routing-decision-updates.jsonl` | `goyoke-agent-endstate` |
| Agent collaborations | `$XDG_DATA_HOME/goyoke/agent-collaborations.jsonl` | `goyoke-agent-endstate` |

## Sharp Edges

When 3+ consecutive failures occur on the same file/function, `goyoke-sharp-edge` captures a sharp edge record. These identify recurring problem patterns for future avoidance.

## Export Tools

```bash
goyoke-ml-export routing-decisions --output=decisions.jsonl
goyoke-ml-export stats
```

## Key Packages

- `pkg/telemetry/` — Telemetry types and writers
- `pkg/routing/` — Decision capture in hook logic

## See Also

- [[ARCHITECTURE#4. ML Telemetry System]] — Full telemetry architecture
- [[session-lifecycle]] — When telemetry is captured
