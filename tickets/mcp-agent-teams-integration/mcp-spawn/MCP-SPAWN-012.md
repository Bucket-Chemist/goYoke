```yaml
---
id: MCP-SPAWN-012
title: Integration Testing and Documentation
description: End-to-end testing of MCP spawning and documentation updates.
status: pending
time_estimate: 4h
dependencies: [MCP-SPAWN-010, MCP-SPAWN-011, MCP-SPAWN-014]
phase: 3
tags: [testing, documentation, phase-3]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-012: Integration Testing and Documentation

## Description

End-to-end testing of the complete MCP spawning system and documentation updates for CLAUDE.md and troubleshooting guide.

**Source**: Staff-Architect Analysis §4.5.2, §4.7.2

## Task

1. Create E2E test for Braintrust workflow
2. Create E2E test for /review workflow
3. Test timeout and error scenarios
4. Update CLAUDE.md with spawning documentation
5. Create troubleshooting guide

## Files

- `packages/tui/tests/e2e/braintrust.test.ts` — E2E test
- `packages/tui/tests/e2e/review.test.ts` — E2E test
- `~/.claude/CLAUDE.md` — Documentation update
- `~/.claude/docs/mcp-spawning-troubleshooting.md` — New guide

## Acceptance Criteria

- [ ] E2E Braintrust test passes (may use mock CLI for CI)
- [ ] E2E Review test passes
- [ ] Timeout scenarios tested
- [ ] Error propagation tested
- [ ] CLAUDE.md updated with spawn_agent documentation
- [ ] Troubleshooting guide created
- [ ] 3 successful real Braintrust runs without intervention
- [ ] Contractor onboarding docs complete
- [ ] Cost tracking integration verified
- [ ] Cost aggregation test passes

## R9: Local Development Setup (Contractor Onboarding)

### Prerequisites

| Requirement | Version | Verification |
|-------------|---------|--------------|
| Node.js | ≥20.x | `node --version` |
| npm | ≥10.x | `npm --version` |
| Claude CLI | latest | `claude --version` |
| Go | ≥1.21 | `go version` |

### Environment Variables

Create `.env.local` in project root:

```bash
# Required for MCP spawning
GOGENT_NESTING_LEVEL=0
GOGENT_MCP_SPAWN_ENABLED=true

# Optional: Test mode
GOGENT_TEST_MODE=false

# Optional: Custom paths
CLAUDE_PROJECT_DIR=/path/to/project
XDG_DATA_HOME=~/.local/share
XDG_RUNTIME_DIR=/run/user/$(id -u)
```

### Quick Start

```bash
# 1. Install dependencies
cd packages/tui && npm install

# 2. Build TUI
npm run build

# 3. Build Go hooks
cd ../.. && make hooks

# 4. Run tests
npm test

# 5. Start TUI in dev mode
npm run dev
```

### Test Data Generation

```bash
# Generate mock agents-index.json for testing
npm run gen:test-agents

# Generate sample spawn scenarios
npm run gen:test-scenarios

# Reset test environment
npm run test:reset
```

### Common Issues

| Issue | Solution |
|-------|----------|
| `spawn_agent tool not found` | Ensure `GOGENT_MCP_SPAWN_ENABLED=true` |
| `agents-index.json not found` | Check `~/.claude/agents/` exists |
| `Permission denied on hooks` | Run `chmod +x cmd/gogent-*/gogent-*` |
| `MCP server not starting` | Check port 3000 not in use |

## R10: Cost Attribution Strategy

### CLI Spawn Cost Rollup

When agents are spawned via CLI (`spawn_agent`), costs must roll up to the parent session.

#### Cost Flow

```
Router Session (Level 0)
├── Mozart spawn via Task() → costs directly attributed
│   └── Einstein spawn via CLI → costs in CLI output JSON
│   └── Staff-Architect spawn via CLI → costs in CLI output JSON
│   └── Beethoven spawn via CLI → costs in CLI output JSON
```

#### Cost Extraction from CLI Output

The CLI returns JSON with cost information:

```json
{
  "result": "...",
  "cost_usd": 0.0523,
  "num_turns": 5,
  "input_tokens": 12500,
  "output_tokens": 3200
}
```

#### Cost Aggregation Implementation

In `spawnAgent.ts`, extract and aggregate costs:

```typescript
// After CLI process completes
const parsed = parseCliOutput(stdout);
if (parsed.cost) {
  // Add to session cost tracker
  getSessionCostTracker().addSpawnCost({
    agentId,
    agentType: args.agent,
    cost: parsed.cost,
    tokens: {
      input: parsed.inputTokens,
      output: parsed.outputTokens,
    },
    turns: parsed.turns,
  });
}
```

#### Cost Verification Test

Add to `packages/tui/tests/e2e/cost-tracking.test.ts`:

```typescript
describe("Cost Attribution", () => {
  it("should aggregate spawn costs to parent session", async () => {
    const tracker = getSessionCostTracker();
    tracker.reset();

    // Spawn child agent via CLI
    await spawnAgent({
      agent: "einstein",
      prompt: "Test prompt",
    });

    const costs = tracker.getSpawnCosts();
    expect(costs.length).toBe(1);
    expect(costs[0].agentType).toBe("einstein");
    expect(costs[0].cost).toBeGreaterThan(0);
  });

  it("should include spawn costs in session total", async () => {
    const tracker = getSessionCostTracker();
    const initialTotal = tracker.getSessionTotal();

    await spawnAgent({ agent: "einstein", prompt: "Test" });

    const finalTotal = tracker.getSessionTotal();
    expect(finalTotal).toBeGreaterThan(initialTotal);
  });
});
```

#### Session Summary

At session end, output cost breakdown:

```
Session Cost Summary:
├── Router direct costs: $0.12
├── Spawn costs:
│   ├── einstein: $0.05
│   ├── staff-architect: $0.04
│   └── beethoven: $0.03
└── Total: $0.24
```

