```yaml
---
id: MCP-SPAWN-008
title: spawn_agent MCP Tool Implementation
description: Implement the core spawn_agent MCP tool that spawns Claude CLI processes with proper process management.
status: pending
time_estimate: 4h
dependencies: [MCP-SPAWN-003, MCP-SPAWN-004, MCP-SPAWN-005, MCP-SPAWN-006]
phase: 1
tags: [mcp, spawn, core, phase-1]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-008: spawn_agent MCP Tool Implementation

## Description

Implement the core `spawn_agent` MCP tool that spawns Claude CLI processes. Uses stdin piping (not shell: true), integrates with process registry, respects buffer limits, and handles timeouts.

**Source**: Staff-Architect Analysis §4.1.2, §4.3.3, §4.6.1, Einstein Analysis §3.5

## Why This Matters

This is the core mechanism for Level 1+ agent spawning. All orchestrators will use this tool to spawn specialist agents.

## Task

1. Implement spawn_agent tool with correct CLI invocation
2. Integrate with process registry
3. Add buffer limits for output
4. Handle timeout with SIGTERM → SIGKILL
5. Parse JSON output correctly

## Files

- `packages/tui/src/mcp/tools/spawnAgent.ts` — Main implementation
- `packages/tui/src/mcp/tools/spawnAgent.test.ts` — Tests
- `packages/tui/src/mcp/server.ts` — Registration

## Implementation

### spawn_agent Tool (`packages/tui/src/mcp/tools/spawnAgent.ts`)

```typescript
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn } from "child_process";
import { getProcessRegistry } from "../spawn/processRegistry";
import { randomUUID } from "crypto";

// Constants
const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB
const DEFAULT_TIMEOUT = 300000; // 5 minutes

/**
 * Result from a spawn_agent invocation
 */
export interface SpawnResult {
  agentId: string;
  agent: string;
  success: boolean;
  output?: string;
  error?: string;
  cost?: number;
  turns?: number;
  duration?: number;
  truncated?: boolean;
}

/**
 * spawn_agent MCP tool - spawns Claude CLI processes for Level 1+ agent spawning.
 */
export const spawnAgent = tool(
  "spawn_agent",
  `Spawn a Claude Code subagent with full tool access via CLI.
  
Use this tool when you need to spawn a sub-subagent (Level 2+).
The spawned agent runs as an independent CLI process with full tool access.

Example:
  spawn_agent({
    agent: "einstein",
    description: "Theoretical analysis",
    prompt: "AGENT: einstein\\n\\nAnalyze the problem...",
    model: "opus"
  })`,
  {
    agent: z.string().describe("Agent type from agents-index.json (e.g., 'einstein', 'backend-reviewer')"),
    description: z.string().describe("Brief description for logging"),
    prompt: z.string().describe("Full prompt to send to the agent"),
    model: z.enum(["haiku", "sonnet", "opus"]).optional().describe("Model to use (default: from agent config)"),
    timeout: z.number().optional().describe("Timeout in ms (default: 300000)"),
    allowedTools: z.array(z.string()).optional().describe("Restrict available tools"),
    maxBudget: z.number().optional().describe("Max budget in USD"),
  },
  async (args): Promise<{ content: Array<{ type: "text"; text: string }> }> => {
    const agentId = randomUUID();
    const registry = getProcessRegistry();
    const timeout = args.timeout ?? DEFAULT_TIMEOUT;
    const startTime = Date.now();

    // Build CLI arguments
    const cliArgs = buildCliArgs(args);

    return new Promise((resolve) => {
      // Spawn CLI process (NO shell: true)
      const proc = spawn("claude", cliArgs, {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GOGENT_NESTING_LEVEL: String(getCurrentNestingLevel() + 1),
          GOGENT_PARENT_AGENT: agentId,
          GOGENT_SPAWN_METHOD: "mcp-cli",
        },
      });

      // Register with process registry
      registry.register(agentId, proc, args.agent);

      // Output collection with buffer limit
      let stdout = "";
      let stderr = "";
      let truncated = false;

      proc.stdout.on("data", (chunk: Buffer) => {
        if (!truncated && stdout.length < MAX_BUFFER_SIZE) {
          stdout += chunk.toString();
          if (stdout.length >= MAX_BUFFER_SIZE) {
            truncated = true;
            stdout += "\n[OUTPUT TRUNCATED - exceeded 10MB limit]";
          }
        }
      });

      proc.stderr.on("data", (chunk: Buffer) => {
        // Stderr is typically small, but limit anyway
        if (stderr.length < 1024 * 1024) {
          stderr += chunk.toString();
        }
      });

      // Send prompt via stdin
      proc.stdin.write(args.prompt);
      proc.stdin.end();

      // Timeout handling
      const timer = setTimeout(() => {
        // SIGTERM first
        proc.kill("SIGTERM");

        // SIGKILL after 5s if still running
        setTimeout(() => {
          if (!proc.killed) {
            proc.kill("SIGKILL");
          }
        }, 5000);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: false,
          error: `Agent timed out after ${timeout}ms`,
          duration: Date.now() - startTime,
          truncated,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      }, timeout);

      // Process completion
      proc.on("close", (code, signal) => {
        clearTimeout(timer);

        const duration = Date.now() - startTime;
        const parsed = parseCliOutput(stdout);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: code === 0 && !signal,
          output: parsed.result || stdout,
          error: code !== 0 ? stderr || `Exit code ${code}` : undefined,
          cost: parsed.cost,
          turns: parsed.turns,
          duration,
          truncated,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      });

      proc.on("error", (err) => {
        clearTimeout(timer);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: false,
          error: `Spawn error: ${err.message}`,
          duration: Date.now() - startTime,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      });
    });
  }
);

/**
 * Build CLI arguments for claude command.
 */
function buildCliArgs(args: {
  model?: string;
  allowedTools?: string[];
  maxBudget?: number;
}): string[] {
  const cliArgs = ["-p", "--output-format", "json"];

  if (args.model) {
    cliArgs.push("--model", args.model);
  }

  // Use delegate mode instead of dangerously-skip-permissions
  cliArgs.push("--permission-mode", "delegate");

  if (args.allowedTools && args.allowedTools.length > 0) {
    cliArgs.push("--allowedTools", args.allowedTools.join(","));
  }

  if (args.maxBudget) {
    cliArgs.push("--max-budget-usd", String(args.maxBudget));
  }

  return cliArgs;
}

/**
 * Parse JSON output from claude CLI.
 */
function parseCliOutput(stdout: string): {
  result?: string;
  cost?: number;
  turns?: number;
} {
  try {
    const json = JSON.parse(stdout.trim());
    return {
      result: json.result || json.output,
      cost: json.cost_usd || json.total_cost_usd,
      turns: json.num_turns,
    };
  } catch {
    // Not valid JSON, return raw output
    return { result: stdout };
  }
}

/**
 * Get current nesting level from environment.
 */
function getCurrentNestingLevel(): number {
  const level = process.env.GOGENT_NESTING_LEVEL;
  if (!level) return 0;
  const parsed = parseInt(level, 10);
  return isNaN(parsed) ? 0 : parsed;
}
```

### Tests (`packages/tui/src/mcp/tools/spawnAgent.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawnMockClaude } from "../../tests/mocks/spawnHelper";
import { resetProcessRegistry, getProcessRegistry } from "../spawn/processRegistry";

// Note: Full tests require mock CLI infrastructure from MCP-SPAWN-003

describe("spawn_agent tool", () => {
  beforeEach(() => {
    resetProcessRegistry();
  });

  afterEach(() => {
    resetProcessRegistry();
  });

  describe("buildCliArgs", () => {
    it("should include -p and --output-format json", () => {
      // Import the function for testing
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({});
      
      expect(args).toContain("-p");
      expect(args).toContain("--output-format");
      expect(args).toContain("json");
    });

    it("should include model when specified", () => {
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({ model: "opus" });
      
      expect(args).toContain("--model");
      expect(args).toContain("opus");
    });

    it("should include allowedTools when specified", () => {
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({ allowedTools: ["Read", "Glob", "Grep"] });
      
      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });
  });

  describe("parseCliOutput", () => {
    it("should parse valid JSON output", () => {
      const { parseCliOutput } = require("./spawnAgent");
      
      const output = JSON.stringify({
        result: "Analysis complete",
        cost_usd: 0.05,
        num_turns: 3,
      });
      
      const parsed = parseCliOutput(output);
      
      expect(parsed.result).toBe("Analysis complete");
      expect(parsed.cost).toBe(0.05);
      expect(parsed.turns).toBe(3);
    });

    it("should return raw output for invalid JSON", () => {
      const { parseCliOutput } = require("./spawnAgent");
      
      const output = "This is not JSON";
      const parsed = parseCliOutput(output);
      
      expect(parsed.result).toBe("This is not JSON");
    });
  });

  describe("getCurrentNestingLevel", () => {
    it("should return 0 when not set", () => {
      const originalEnv = process.env.GOGENT_NESTING_LEVEL;
      delete process.env.GOGENT_NESTING_LEVEL;
      
      const { getCurrentNestingLevel } = require("./spawnAgent");
      expect(getCurrentNestingLevel()).toBe(0);
      
      process.env.GOGENT_NESTING_LEVEL = originalEnv;
    });

    it("should return parsed level when set", () => {
      const originalEnv = process.env.GOGENT_NESTING_LEVEL;
      process.env.GOGENT_NESTING_LEVEL = "2";
      
      // Need to re-import to pick up new env
      vi.resetModules();
      const { getCurrentNestingLevel } = require("./spawnAgent");
      expect(getCurrentNestingLevel()).toBe(2);
      
      process.env.GOGENT_NESTING_LEVEL = originalEnv;
    });
  });

  // Integration tests with mock CLI
  describe("integration with mock CLI", () => {
    it("should handle successful spawn", async () => {
      const result = await spawnMockClaude(
        { behavior: "success", output: "Test output" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toContain("success");
    });

    it("should handle timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        "Test prompt",
        100 // 100ms timeout
      );

      expect(result.killed).toBe(true);
    });

    it("should handle error response", async () => {
      const result = await spawnMockClaude(
        { behavior: "error_max_turns" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(1);
    });
  });
});
```

## Acceptance Criteria

- [ ] spawn_agent tool compiles and type-checks
- [ ] Uses stdin piping (NOT shell: true)
- [ ] Integrates with process registry
- [ ] Buffer limited to 10MB with truncation indicator
- [ ] Timeout handled with SIGTERM → SIGKILL escalation
- [ ] JSON output parsed correctly
- [ ] Returns structured SpawnResult
- [ ] Nesting level incremented for child processes
- [ ] All tests pass with mock CLI
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/src/mcp/tools/spawnAgent.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: invoke from subagent, verify CLI spawns

