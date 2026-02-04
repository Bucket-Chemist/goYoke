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
- [ ] All tests pass: `npm test -- src/mcp/server.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/src/mcp/server.test.ts`
- [ ] Number of test functions: 6
- [ ] All tests passing
- [ ] Coverage ≥80%

### Required Test Cases (`packages/tui/src/mcp/server.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

describe("MCP Server Registration", () => {
  describe("isSpawnEnabled", () => {
    it("should return true when GOGENT_MCP_SPAWN_ENABLED is not set", () => {
      delete process.env.GOGENT_MCP_SPAWN_ENABLED;
      const { isSpawnEnabled } = require("./server");
      expect(isSpawnEnabled()).toBe(true);
    });

    it("should return true when GOGENT_MCP_SPAWN_ENABLED is 'true'", () => {
      process.env.GOGENT_MCP_SPAWN_ENABLED = "true";
      const { isSpawnEnabled } = require("./server");
      expect(isSpawnEnabled()).toBe(true);
    });

    it("should return false when GOGENT_MCP_SPAWN_ENABLED is 'false'", () => {
      process.env.GOGENT_MCP_SPAWN_ENABLED = "false";
      const { isSpawnEnabled } = require("./server");
      expect(isSpawnEnabled()).toBe(false);
    });
  });

  describe("createMcpServer", () => {
    it("should include spawn_agent when spawn is enabled", () => {
      delete process.env.GOGENT_MCP_SPAWN_ENABLED;
      const server = createMcpServer();
      const toolNames = server.tools.map(t => t.name);
      expect(toolNames).toContain("spawn_agent");
    });

    it("should exclude spawn_agent when spawn is disabled", () => {
      process.env.GOGENT_MCP_SPAWN_ENABLED = "false";
      const server = createMcpServer();
      const toolNames = server.tools.map(t => t.name);
      expect(toolNames).not.toContain("spawn_agent");
    });

    it("should always include test_mcp_ping tool", () => {
      const server = createMcpServer();
      const toolNames = server.tools.map(t => t.name);
      expect(toolNames).toContain("test_mcp_ping");
    });
  });
});
```

