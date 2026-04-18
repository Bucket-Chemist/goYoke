# MCP-Based Agent Spawning Discovery

**Date**: 2026-02-04
**Status**: VALIDATED - Ready for implementation
**Impact**: Unlocks orchestrator pattern that was previously blocked

---

## Executive Summary

We discovered that while Claude Code restricts the `Task` tool from subagents (preventing sub-subagent spawning), **MCP tools ARE available to subagents**. This creates a viable workaround: implement agent spawning as an MCP tool in the TUI's in-process MCP server.

---

## The Problem (Before)

```
Router (Level 0)
  └── Task() → Mozart (Level 1)
                  └── Task() → Einstein (Level 2)  ❌ BLOCKED
                                                    "Task tool not available"
```

Subagents don't have the `Task` tool. This was confirmed by:
1. Mozart explicitly stating "I do not have access to the Task tool"
2. A test subagent listing 16 tools - Task not among them
3. Documentation showing Explore/Plan subagent_types exclude Task

---

## The Discovery

MCP tools ARE available to subagents:

| Capability | Router | Subagent |
|------------|--------|----------|
| Task tool | ✅ | ❌ |
| MCP tools | ✅ | ✅ |
| mcp__ide__getDiagnostics | ✅ Works | ✅ Works |
| mcp__ide__executeCode | ✅ Works | ✅ Works |

**Test Proof**: A subagent successfully called `mcp__ide__getDiagnostics()` and received diagnostic data for 4 workspace files.

---

## The Solution

### Architecture

```
Mozart (subagent, no Task tool)
    │
    │ calls: mcp__goyoke__spawn_agent({
    │   agent: "einstein",
    │   prompt: "Analyze the problem...",
    │   model: "opus"
    │ })
    │
    ▼
┌─────────────────────────────────────────────┐
│  goyoke-tui (Node.js process)           │
│                                             │
│  MCP Handler: spawn_agent                   │
│    → Receives spawn request                 │
│    → Calls SDK query() with new prompt      │
│    → Collects streaming response            │
│    → Returns result to Mozart               │
└─────────────────────────────────────────────┘
    │
    ▼
Einstein's output returned to Mozart
```

### Key Insight

MCP tools bypass Claude Code's subprocess tool restrictions because:
1. MCP tool invocations are sent to YOUR MCP server (the TUI)
2. Your TUI runs in Node.js with full SDK access
3. Your handler can call `query()` to spawn new Claude conversations
4. Results return through the MCP tool response mechanism

---

## Implementation Plan

### 1. Add spawn_agent MCP Tool

```typescript
// packages/tui/src/mcp/tools/spawnAgent.ts
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { query } from "@anthropic-ai/claude-agent-sdk";

export const spawnAgentTool = tool(
  "spawn_agent",
  "Spawn a subagent to perform a task. Used by orchestrators to delegate work.",
  {
    agent: z.string().describe("Agent ID from agents-index.json"),
    description: z.string().describe("Brief task description"),
    prompt: z.string().describe("Full prompt including AGENT: header"),
    model: z.enum(["haiku", "sonnet", "opus"]).default("haiku"),
    background: z.boolean().default(false).describe("Run in background"),
  },
  async (args) => {
    console.log(`[MCP] Spawning agent: ${args.agent} (${args.model})`);

    // Load agent config for validation (optional)
    // const agentConfig = await loadAgentConfig(args.agent);

    // Spawn via SDK
    const messages = [];
    for await (const event of query({
      prompt: args.prompt,
      options: { model: args.model }
    })) {
      if (event.type === "content_block_delta" && event.delta.type === "text_delta") {
        messages.push(event.delta.text);
      }
    }

    const output = messages.join("");

    return {
      content: [{
        type: "text",
        text: JSON.stringify({
          agent: args.agent,
          model: args.model,
          success: true,
          output: output
        })
      }]
    };
  }
);
```

### 2. Register Tool in MCP Server

```typescript
// packages/tui/src/mcp/server.ts
import { spawnAgentTool } from "./tools/spawnAgent";

export const mcpServer = createSdkMcpServer({
  name: "goyoke-interactive",
  version: "1.0.0",
  tools: [
    askUserTool,
    confirmActionTool,
    requestInputTool,
    selectOptionTool,
    spawnAgentTool,  // NEW
  ]
});
```

### 3. Update Orchestrator Prompts

Instead of:
```javascript
Task({
  description: "Einstein analysis",
  subagent_type: "Plan",
  model: "opus",
  prompt: "AGENT: einstein\n..."
})
```

Use:
```javascript
mcp__goyoke__spawn_agent({
  agent: "einstein",
  description: "Einstein analysis",
  model: "opus",
  prompt: "AGENT: einstein\n..."
})
```

### 4. Update CLAUDE.md Documentation

Add to orchestrator section:
```markdown
## Orchestrator Spawning (MCP Method)

Orchestrator agents spawn subagents via MCP, not Task tool:

\`\`\`javascript
mcp__goyoke__spawn_agent({
  agent: "backend-reviewer",
  model: "haiku",
  prompt: "AGENT: backend-reviewer\n\nTASK: Review security..."
})
\`\`\`

This works because MCP tools are available to all agents, while Task is router-only.
```

---

## Advantages of MCP Spawning

| Aspect | Task Tool | MCP Spawning |
|--------|-----------|--------------|
| Available to subagents | ❌ No | ✅ Yes |
| Centralized control | ❌ Claude Code | ✅ Your TUI |
| Custom validation | ❌ Limited | ✅ Full control |
| Logging/telemetry | Via hooks | In-process |
| Cost tracking | Via hooks | In-process |
| Parallel spawning | ✅ Yes | ✅ Yes (Promise.all) |

---

## Limitations & Considerations

1. **Requires TUI running**: MCP spawning only works when your TUI is the interface
2. **Not available in raw CLI**: If using `claude` directly, falls back to flat pattern
3. **Adds TUI dependency**: Orchestrator pattern now requires TUI infrastructure
4. **Latency**: MCP round-trip adds ~50-100ms overhead

---

## Testing Checklist

Before deploying:
- [ ] spawn_agent tool registered in MCP server
- [ ] Test single agent spawn from router
- [ ] Test single agent spawn from subagent (Mozart → Einstein)
- [ ] Test parallel spawns (Mozart → [Einstein, Staff-Architect])
- [ ] Test nested spawns (Mozart → Einstein → Haiku-scout)
- [ ] Verify telemetry captures MCP-spawned agents
- [ ] Verify cost tracking works
- [ ] Test timeout handling for long-running agents
- [ ] Test error propagation when spawned agent fails

---

## Migration Path

### Phase 1: Implement & Test (TUI work)
- Add spawn_agent MCP tool
- Test with simple spawning scenarios

### Phase 2: Update Orchestrators
- Update Mozart to use MCP spawning
- Update review-orchestrator
- Update impl-manager

### Phase 3: Documentation
- Update CLAUDE.md with MCP spawning pattern
- Update agent definitions
- Add to sharp edges (limitation of non-TUI mode)

---

## Conclusion

MCP-based spawning is a viable workaround for Claude Code's Task tool restriction. It requires the TUI infrastructure but enables the full orchestrator pattern we designed.

**Next Step**: Implement `spawn_agent` MCP tool in the TypeScript TUI (Phase 4 of TUI-TS-INK-CONVERSION-PLAN.md).

---

## Metadata

```yaml
discovery_id: mcp-spawning-2026-02-04
validated: true
test_method: "Subagent successfully called mcp__ide__getDiagnostics"
implementation_location: "packages/tui/src/mcp/tools/spawnAgent.ts"
blocks: "TUI Phase 4 (MCP Integration)"
enables: ["braintrust workflow", "review-orchestrator", "impl-manager"]
```
