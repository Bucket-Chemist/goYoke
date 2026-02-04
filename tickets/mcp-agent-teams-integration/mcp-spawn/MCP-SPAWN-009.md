```yaml
---
id: MCP-SPAWN-009
title: MCP Server Registration and Integration
description: Register spawn_agent tool with MCP server and integrate with TUI startup.
status: pending
time_estimate: 1h
dependencies: [MCP-SPAWN-008]
phase: 2
tags: [mcp, integration, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-009: MCP Server Registration and Integration

## Description

Register the spawn_agent tool with the TUI's MCP server and integrate with startup validation.

**Source**: Staff-Architect Analysis §4.2.2

## Task

1. Add spawn_agent to MCP server tools
2. Add feature flag check
3. Integrate environment validation
4. Test tool availability

## Files

- `packages/tui/src/mcp/server.ts` — Add tool registration
- `packages/tui/src/index.tsx` — Add startup validation

## Implementation

### MCP Server Update (`packages/tui/src/mcp/server.ts`)

```typescript
import { createSdkMcpServer } from "@anthropic-ai/claude-agent-sdk";
import { spawnAgent } from "./tools/spawnAgent";
import { testMcpPing } from "./tools/testMcpPing";
// ... existing tool imports

/**
 * Check if MCP spawning is enabled via feature flag.
 */
function isSpawnEnabled(): boolean {
  return process.env.GOGENT_MCP_SPAWN_ENABLED !== "false";
}

/**
 * Create MCP server with all tools.
 */
export function createMcpServer() {
  const tools = [
    // Existing tools
    askUser,
    confirmAction,
    requestInput,
    selectOption,
    // Test tool (always available for verification)
    testMcpPing,
  ];

  // Conditionally add spawn tools
  if (isSpawnEnabled()) {
    tools.push(spawnAgent);
  }

  return createSdkMcpServer({
    name: "gofortress",
    version: "1.0.0",
    tools,
  });
}
```

### Startup Integration (`packages/tui/src/index.tsx`)

```typescript
import { assertValidSpawnEnvironment } from "./spawn/validation";

async function main() {
  // Validate spawn environment before starting
  try {
    await assertValidSpawnEnvironment();
  } catch (err) {
    console.error(err.message);
    // Continue with warnings but allow startup
  }

  // ... rest of startup
}
```

## Acceptance Criteria

- [ ] spawn_agent registered in MCP server
- [ ] Feature flag respected (GOGENT_MCP_SPAWN_ENABLED=false disables)
- [ ] Environment validation runs at startup
- [ ] Tool available to subagents (verified with testMcpPing)

