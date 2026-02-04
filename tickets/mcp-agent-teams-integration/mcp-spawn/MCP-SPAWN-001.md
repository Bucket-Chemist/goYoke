```yaml
---
id: MCP-SPAWN-001
title: MCP Tool Availability Verification (GATE)
description: Verify that MCP tools registered in TUI are accessible from Task()-spawned subagents. This is a CRITICAL GATE - failure invalidates the entire architecture.
status: pending
time_estimate: 2h
dependencies: []
phase: 0
tags: [gate, critical, verification, phase-0]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
gate_decision: null
---
```

# MCP-SPAWN-001: MCP Tool Availability Verification (GATE)

## Description

Verify that MCP tools registered in the TUI's MCP server are accessible from Task()-spawned subagents. This is the **most critical assumption** in the entire architecture and has **zero empirical evidence**.

**Source**: Einstein Analysis §3.6.4, Staff-Architect Analysis §4.1.1

## Why This Matters

The entire MCP-based agent spawning architecture depends on this assumption:
- If MCP tools ARE available to subagents → Architecture is valid, proceed
- If MCP tools are NOT available → Architecture is INVALID, must use flat coordination fallback

## Task

1. Create a minimal test MCP tool (`test_mcp_ping`)
2. Register it in the TUI's MCP server
3. Spawn a subagent via Task()
4. Have the subagent attempt to invoke `mcp__gofortress__test_mcp_ping`
5. Document the exact result

## Files

- `packages/tui/src/mcp/tools/testMcpPing.ts` — Test tool implementation
- `packages/tui/src/mcp/server.ts` — Register test tool
- `.claude/tmp/mcp-verification-result.json` — Results documentation

## Implementation

### Test Tool (`packages/tui/src/mcp/tools/testMcpPing.ts`)

```typescript
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";

/**
 * Minimal MCP tool for verifying subagent accessibility.
 * Returns PONG with timestamp to prove invocation succeeded.
 */
export const testMcpPing = tool(
  "test_mcp_ping",
  "Verify MCP tool accessibility from subagents. Returns PONG with timestamp.",
  {
    echo: z.string().optional().describe("Optional string to echo back"),
  },
  async (args) => {
    const timestamp = new Date().toISOString();
    const response = {
      status: "PONG",
      timestamp,
      echo: args.echo || null,
      message: "MCP tool successfully invoked",
    };

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(response, null, 2),
        },
      ],
    };
  }
);
```

### Register in Server (`packages/tui/src/mcp/server.ts`)

```typescript
// Add import
import { testMcpPing } from "./tools/testMcpPing";

// Add to tools array in createSdkMcpServer call
tools: [
  // ... existing tools
  testMcpPing,
],
```

### Verification Script

Run from router level (not as subagent):

```typescript
// Manual test: Spawn subagent and have it try MCP tool
Task({
  description: "MCP availability verification",
  subagent_type: "Explore",
  model: "haiku",
  prompt: `TASK: Verify MCP tool accessibility

1. Attempt to invoke the tool: mcp__gofortress__test_mcp_ping
2. Pass echo parameter: "verification-test"
3. Report the EXACT result:
   - If successful: Copy the full JSON response
   - If failed: Copy the exact error message

DO NOT fabricate results. Report exactly what happens.`
});
```

### M2 Enhancement: Automated Phase 0 Gate Test

**Problem:** The current verification requires manual execution. Need automated test for CI.

**Solution:** Add automated test that can verify MCP tool availability programmatically.

#### Automated Test (`packages/tui/tests/e2e/mcpAvailability.test.ts`)

```typescript
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { spawn, ChildProcess } from "child_process";

describe("MCP Tool Availability Gate", () => {
  let tuiProcess: ChildProcess;
  let mcpServerReady: boolean = false;

  beforeAll(async () => {
    // Start TUI in test mode
    tuiProcess = spawn("npm", ["run", "start:test"], {
      cwd: "packages/tui",
      env: { ...process.env, GOGENT_TEST_MODE: "true" },
    });

    // Wait for MCP server to be ready
    await waitForMcpServer(tuiProcess);
    mcpServerReady = true;
  }, 30000);

  afterAll(() => {
    if (tuiProcess) {
      tuiProcess.kill("SIGTERM");
    }
  });

  it("should make MCP tools available to spawned subagents", async () => {
    expect(mcpServerReady).toBe(true);

    // Spawn a minimal subagent that attempts to invoke MCP tool
    const result = await spawnTestSubagent({
      prompt: "Invoke mcp__gofortress__test_mcp_ping with echo='gate-test'",
      timeout: 30000,
    });

    // Verify the subagent could access the MCP tool
    expect(result.success).toBe(true);
    expect(result.output).toContain("PONG");
    expect(result.output).toContain("gate-test");
  });

  it("should return tool not found for non-existent MCP tools", async () => {
    const result = await spawnTestSubagent({
      prompt: "Invoke mcp__gofortress__nonexistent_tool",
      timeout: 10000,
    });

    // Should fail gracefully with clear error
    expect(result.success).toBe(false);
    expect(result.error).toContain("tool not found");
  });
});

async function waitForMcpServer(proc: ChildProcess): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("MCP server startup timeout")), 20000);

    proc.stdout?.on("data", (data: Buffer) => {
      if (data.toString().includes("MCP server ready")) {
        clearTimeout(timeout);
        resolve();
      }
    });

    proc.on("error", (err) => {
      clearTimeout(timeout);
      reject(err);
    });
  });
}

async function spawnTestSubagent(opts: { prompt: string; timeout: number }): Promise<{
  success: boolean;
  output?: string;
  error?: string;
}> {
  // Implementation uses mock CLI infrastructure from MCP-SPAWN-003
  // Returns parsed result from subagent execution
  throw new Error("Implement with MCP-SPAWN-003 mock infrastructure");
}
```

#### CI Integration

Add to `.github/workflows/ci.yml`:

```yaml
- name: Run MCP Gate Test
  run: npm run test:e2e -- mcpAvailability.test.ts
  timeout-minutes: 5
```

#### Gate Decision Automation

The test outputs to `.claude/tmp/mcp-verification-result.json`:

```json
{
  "timestamp": "2026-02-05T10:30:00Z",
  "gate": "MCP-SPAWN-001",
  "result": "PASS" | "FAIL",
  "details": {
    "toolAccessible": true,
    "responseTime": 150,
    "pongReceived": true
  }
}
```

## Acceptance Criteria

- [ ] `testMcpPing` tool created and compiles without errors
- [ ] Tool registered in MCP server successfully
- [ ] TUI starts without errors with new tool
- [ ] Verification test executed from router level
- [ ] Result documented in `.claude/tmp/mcp-verification-result.json`
- [ ] Gate decision recorded: PROCEED or HALT
- [ ] Automated e2e test created for MCP availability
- [ ] Test can run in CI without manual intervention
- [ ] Gate result automatically written to JSON file

## Gate Decision Matrix

| Result | Gate Decision | Next Action |
|--------|---------------|-------------|
| PONG received with timestamp | **PROCEED** | Continue to MCP-SPAWN-002 |
| Tool not found error | **HALT** | Implement flat coordination fallback |
| Permission denied error | **INVESTIGATE** | May be configurable |
| Other error | **INVESTIGATE** | Document and analyze |

## Test Deliverables

- [ ] Test tool created: `packages/tui/src/mcp/tools/testMcpPing.ts`
- [ ] Tool registered in server
- [ ] Manual verification executed
- [ ] Results documented with exact output
- [ ] Gate decision recorded
- [ ] Test file: `packages/tui/tests/e2e/mcpAvailability.test.ts`
- [ ] Number of test functions: 2
- [ ] CI workflow updated

## Rollback

If this ticket reveals MCP tools are NOT accessible:
1. Document the finding in `.claude/tmp/mcp-verification-result.json`
2. Update architecture to use "flat coordination" model
3. Close all subsequent MCP-SPAWN tickets as "won't fix"
4. Create new ticket series for flat coordination implementation

