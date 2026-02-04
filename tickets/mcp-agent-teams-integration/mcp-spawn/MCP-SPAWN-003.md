```yaml
---
id: MCP-SPAWN-003
title: Mock CLI Infrastructure
description: Create mock Claude CLI for unit testing spawn_agent without consuming API credits.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-002]
phase: 0
tags: [testing, infrastructure, phase-0]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-003: Mock CLI Infrastructure

## Description

Create a mock Claude CLI that can be used in unit tests to verify spawn_agent behavior without consuming API credits. This is essential for testing timeout handling, error scenarios, and output parsing.

**Source**: Staff-Architect Analysis §4.5.1

## Why This Matters

Without mock CLI infrastructure:
- Every test consumes API credits
- Cannot test timeout scenarios (would need real slow agents)
- Cannot test error scenarios reliably
- CI/CD pipeline cannot run tests

## Task

1. Create mock CLI script generator
2. Create vitest integration helpers
3. Create test scenarios (success, timeout, error)
4. Verify mock works with spawn()

## Files

- `packages/tui/tests/mocks/mockClaude.ts` — Mock CLI generator
- `packages/tui/tests/mocks/mockScenarios.ts` — Predefined scenarios
- `packages/tui/tests/mocks/spawnHelper.ts` — Vitest integration
- `packages/tui/tests/mocks/mockClaude.test.ts` — Self-tests for mock

## Implementation

### Mock CLI Generator (`packages/tui/tests/mocks/mockClaude.ts`)

```typescript
import * as fs from "fs/promises";
import * as path from "path";
import * as os from "os";
import { randomUUID } from "crypto";

export type MockBehavior =
  | "success"
  | "success_slow"
  | "error_max_turns"
  | "error_rate_limit"
  | "timeout"
  | "invalid_json"
  | "partial_output";

export interface MockOptions {
  behavior: MockBehavior;
  delay?: number; // milliseconds
  output?: string; // custom output
  cost?: number;
  tokens?: { input: number; output: number };
}

const MOCK_SCRIPTS: Record<MockBehavior, (opts: MockOptions) => string> = {
  success: (opts) => `#!/bin/bash
# Mock Claude CLI - Success
sleep ${(opts.delay || 100) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "success",
  "cost_usd": ${opts.cost || 0.001},
  "total_cost_usd": ${opts.cost || 0.001},
  "duration_ms": ${opts.delay || 100},
  "num_turns": 1,
  "result": "${opts.output || "Mock agent completed successfully"}",
  "session_id": "mock-session-${randomUUID()}"
}
MOCK_EOF
`,

  success_slow: (opts) => `#!/bin/bash
# Mock Claude CLI - Slow Success
sleep ${(opts.delay || 5000) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "success",
  "cost_usd": ${opts.cost || 0.01},
  "total_cost_usd": ${opts.cost || 0.01},
  "duration_ms": ${opts.delay || 5000},
  "num_turns": 5,
  "result": "Slow mock agent completed"
}
MOCK_EOF
`,

  error_max_turns: (opts) => `#!/bin/bash
# Mock Claude CLI - Max Turns Error
sleep ${(opts.delay || 100) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "error_max_turns",
  "cost_usd": ${opts.cost || 0.05},
  "total_cost_usd": ${opts.cost || 0.05},
  "duration_ms": ${opts.delay || 100},
  "num_turns": 30,
  "result": null
}
MOCK_EOF
exit 1
`,

  error_rate_limit: (opts) => `#!/bin/bash
# Mock Claude CLI - Rate Limit Error
sleep ${(opts.delay || 50) / 1000}
echo '{"error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}' >&2
exit 1
`,

  timeout: (opts) => `#!/bin/bash
# Mock Claude CLI - Timeout (hangs forever)
sleep 3600
`,

  invalid_json: (opts) => `#!/bin/bash
# Mock Claude CLI - Invalid JSON
sleep ${(opts.delay || 100) / 1000}
echo "This is not valid JSON output {{{{"
`,

  partial_output: (opts) => `#!/bin/bash
# Mock Claude CLI - Partial Output (simulates crash)
sleep ${(opts.delay || 100) / 1000}
echo '{"type": "result", "subtype":'
# Script exits mid-output
`,
};

/**
 * Creates a temporary mock Claude CLI script.
 * Returns the path to the executable script.
 */
export async function createMockClaude(
  options: MockOptions
): Promise<string> {
  const scriptContent = MOCK_SCRIPTS[options.behavior](options);
  const tempDir = os.tmpdir();
  const scriptPath = path.join(
    tempDir,
    `mock-claude-${options.behavior}-${randomUUID()}.sh`
  );

  await fs.writeFile(scriptPath, scriptContent, { mode: 0o755 });

  return scriptPath;
}

/**
 * Cleans up a mock script after use.
 */
export async function cleanupMockClaude(scriptPath: string): Promise<void> {
  try {
    await fs.unlink(scriptPath);
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Creates mock and returns cleanup function.
 * Use with try/finally or vitest afterEach.
 */
export async function withMockClaude(
  options: MockOptions
): Promise<{ path: string; cleanup: () => Promise<void> }> {
  const scriptPath = await createMockClaude(options);
  return {
    path: scriptPath,
    cleanup: () => cleanupMockClaude(scriptPath),
  };
}
```

### Vitest Integration (`packages/tui/tests/mocks/spawnHelper.ts`)

```typescript
import { spawn, ChildProcess } from "child_process";
import { createMockClaude, cleanupMockClaude, MockOptions } from "./mockClaude";

export interface SpawnResult {
  stdout: string;
  stderr: string;
  exitCode: number | null;
  killed: boolean;
  duration: number;
}

/**
 * Spawns the mock CLI and collects output.
 * Use for testing spawn_agent behavior.
 */
export async function spawnMockClaude(
  options: MockOptions,
  stdinContent?: string,
  timeout?: number
): Promise<SpawnResult> {
  const mockPath = await createMockClaude(options);
  const startTime = Date.now();

  return new Promise(async (resolve) => {
    let stdout = "";
    let stderr = "";
    let killed = false;

    const proc = spawn(mockPath, [], {
      stdio: ["pipe", "pipe", "pipe"],
    });

    // Write stdin if provided
    if (stdinContent) {
      proc.stdin.write(stdinContent);
      proc.stdin.end();
    }

    proc.stdout.on("data", (data) => {
      stdout += data.toString();
    });

    proc.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    // Timeout handling
    let timer: NodeJS.Timeout | null = null;
    if (timeout) {
      timer = setTimeout(() => {
        killed = true;
        proc.kill("SIGTERM");
        // Escalate to SIGKILL after 1s
        setTimeout(() => {
          if (!proc.killed) {
            proc.kill("SIGKILL");
          }
        }, 1000);
      }, timeout);
    }

    proc.on("close", async (code) => {
      if (timer) clearTimeout(timer);
      await cleanupMockClaude(mockPath);

      resolve({
        stdout,
        stderr,
        exitCode: code,
        killed,
        duration: Date.now() - startTime,
      });
    });
  });
}
```

### Self-Tests (`packages/tui/tests/mocks/mockClaude.test.ts`)

```typescript
import { describe, it, expect } from "vitest";
import { spawnMockClaude } from "./spawnHelper";

describe("Mock Claude CLI", () => {
  describe("success behavior", () => {
    it("should return valid JSON with success result", async () => {
      const result = await spawnMockClaude({ behavior: "success" });

      expect(result.exitCode).toBe(0);
      expect(result.killed).toBe(false);

      const output = JSON.parse(result.stdout);
      expect(output.type).toBe("result");
      expect(output.subtype).toBe("success");
      expect(output.cost_usd).toBeGreaterThan(0);
    });

    it("should accept custom output", async () => {
      const result = await spawnMockClaude({
        behavior: "success",
        output: "Custom test output",
      });

      const output = JSON.parse(result.stdout);
      expect(output.result).toBe("Custom test output");
    });
  });

  describe("error behaviors", () => {
    it("should return max_turns error with exit code 1", async () => {
      const result = await spawnMockClaude({ behavior: "error_max_turns" });

      expect(result.exitCode).toBe(1);
      const output = JSON.parse(result.stdout);
      expect(output.subtype).toBe("error_max_turns");
    });

    it("should return rate_limit error on stderr", async () => {
      const result = await spawnMockClaude({ behavior: "error_rate_limit" });

      expect(result.exitCode).toBe(1);
      expect(result.stderr).toContain("rate_limit");
    });
  });

  describe("timeout handling", () => {
    it("should kill hanging process after timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        undefined,
        200 // 200ms timeout
      );

      expect(result.killed).toBe(true);
      expect(result.duration).toBeLessThan(1000);
    });
  });

  describe("invalid output handling", () => {
    it("should return invalid JSON for parsing tests", async () => {
      const result = await spawnMockClaude({ behavior: "invalid_json" });

      expect(result.exitCode).toBe(0);
      expect(() => JSON.parse(result.stdout)).toThrow();
    });

    it("should return partial output for crash simulation", async () => {
      const result = await spawnMockClaude({ behavior: "partial_output" });

      expect(result.stdout).toContain('{"type":');
      expect(() => JSON.parse(result.stdout)).toThrow();
    });
  });

  describe("stdin handling", () => {
    it("should accept stdin content", async () => {
      // Success mock ignores stdin but accepts it
      const result = await spawnMockClaude(
        { behavior: "success" },
        "Test prompt content"
      );

      expect(result.exitCode).toBe(0);
    });
  });
});
```

## Acceptance Criteria

- [ ] Mock CLI generator creates executable scripts
- [ ] All 7 behaviors implemented (success, success_slow, error_max_turns, error_rate_limit, timeout, invalid_json, partial_output)
- [ ] Vitest integration helper works with spawn()
- [ ] Self-tests pass: `npm test -- tests/mocks/mockClaude.test.ts`
- [ ] Timeout test completes in <2s (not actually waiting for full timeout)
- [ ] Cleanup removes temporary scripts
- [ ] Code coverage ≥80% on mock infrastructure

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/mocks/mockClaude.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing: `npm test -- tests/mocks/`
- [ ] Coverage ≥80%

