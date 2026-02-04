```yaml
---
id: MCP-SPAWN-004
title: Environment Validation Pre-flight Checks
description: Implement pre-flight checks for required dependencies before spawn_agent can be used.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-001]
phase: 1
tags: [infrastructure, validation, phase-1]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-004: Environment Validation Pre-flight Checks

## Description

Implement pre-flight validation that checks all required dependencies before spawn_agent can be used. Fail fast with clear error messages instead of failing later with cryptic errors.

**Source**: Staff-Architect Analysis §4.2.1

## Why This Matters

Without pre-flight checks:
- `spawn()` fails silently if `claude` not in PATH
- Temp file creation fails without clear error
- Users get cryptic errors instead of actionable guidance

## Task

1. Create environment validator module
2. Check for claude CLI in PATH
3. Check /tmp writability
4. Check required env vars
5. Integrate with TUI startup

## Files

- `packages/tui/src/spawn/validation.ts` — Validation functions
- `packages/tui/src/spawn/validation.test.ts` — Tests
- `packages/tui/src/index.tsx` — Integration point

## Implementation

### Validation Module (`packages/tui/src/spawn/validation.ts`)

```typescript
import { execSync } from "child_process";
import * as fs from "fs/promises";
import * as path from "path";
import * as os from "os";

export interface ValidationResult {
  ok: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  code: string;
  message: string;
  resolution: string;
}

export interface ValidationWarning {
  code: string;
  message: string;
  impact: string;
}

/**
 * Validates environment for spawn_agent functionality.
 * Call at TUI startup to fail fast with clear errors.
 */
export async function validateSpawnEnvironment(): Promise<ValidationResult> {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  // Check 1: claude CLI in PATH
  try {
    execSync("which claude", { stdio: "pipe" });
  } catch {
    errors.push({
      code: "E_CLAUDE_NOT_FOUND",
      message: "claude CLI not found in PATH",
      resolution:
        "Install Claude Code CLI: npm install -g @anthropic-ai/claude-code",
    });
  }

  // Check 2: /tmp writable
  const tmpTestFile = path.join(os.tmpdir(), `spawn-test-${Date.now()}`);
  try {
    await fs.writeFile(tmpTestFile, "test", "utf-8");
    await fs.unlink(tmpTestFile);
  } catch (err) {
    errors.push({
      code: "E_TMP_NOT_WRITABLE",
      message: `Cannot write to temp directory: ${os.tmpdir()}`,
      resolution:
        "Ensure temp directory exists and is writable, or set TMPDIR env var",
    });
  }

  // Check 3: GOGENT_MCP_SPAWN_ENABLED not explicitly disabled
  if (process.env.GOGENT_MCP_SPAWN_ENABLED === "false") {
    warnings.push({
      code: "W_SPAWN_DISABLED",
      message: "MCP spawn is disabled via GOGENT_MCP_SPAWN_ENABLED=false",
      impact: "spawn_agent tool will not be available",
    });
  }

  // Check 4: XDG_DATA_HOME for telemetry (warning only)
  if (!process.env.XDG_DATA_HOME) {
    warnings.push({
      code: "W_XDG_DATA_HOME_MISSING",
      message: "XDG_DATA_HOME not set",
      impact: "Telemetry will use fallback ~/.local/share",
    });
  }

  // Check 5: Node.js version
  const nodeVersion = process.versions.node;
  const [major] = nodeVersion.split(".").map(Number);
  if (major < 20) {
    errors.push({
      code: "E_NODE_VERSION",
      message: `Node.js ${nodeVersion} is below minimum required version 20`,
      resolution: "Upgrade Node.js to version 20 or higher",
    });
  }

  return {
    ok: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Formats validation result for console output.
 */
export function formatValidationResult(result: ValidationResult): string {
  const lines: string[] = [];

  if (result.ok) {
    lines.push("✅ Environment validation passed");
  } else {
    lines.push("❌ Environment validation failed");
  }

  if (result.errors.length > 0) {
    lines.push("");
    lines.push("Errors:");
    for (const err of result.errors) {
      lines.push(`  [${err.code}] ${err.message}`);
      lines.push(`    → ${err.resolution}`);
    }
  }

  if (result.warnings.length > 0) {
    lines.push("");
    lines.push("Warnings:");
    for (const warn of result.warnings) {
      lines.push(`  [${warn.code}] ${warn.message}`);
      lines.push(`    Impact: ${warn.impact}`);
    }
  }

  return lines.join("\n");
}

/**
 * Validates and throws if critical errors found.
 * Use at startup to prevent running with invalid environment.
 */
export async function assertValidSpawnEnvironment(): Promise<void> {
  const result = await validateSpawnEnvironment();

  if (!result.ok) {
    const formatted = formatValidationResult(result);
    throw new Error(`Spawn environment validation failed:\n${formatted}`);
  }

  // Log warnings but don't fail
  if (result.warnings.length > 0) {
    console.warn(formatValidationResult(result));
  }
}
```

### Tests (`packages/tui/src/spawn/validation.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as child_process from "child_process";
import * as fs from "fs/promises";
import {
  validateSpawnEnvironment,
  formatValidationResult,
  assertValidSpawnEnvironment,
} from "./validation";

// Mock child_process.execSync
vi.mock("child_process", () => ({
  execSync: vi.fn(),
}));

// Mock fs/promises
vi.mock("fs/promises", () => ({
  writeFile: vi.fn(),
  unlink: vi.fn(),
}));

describe("validateSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Default: all checks pass
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should pass when all checks succeed", async () => {
    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("should fail when claude CLI not found", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_CLAUDE_NOT_FOUND" })
    );
  });

  it("should fail when /tmp not writable", async () => {
    vi.mocked(fs.writeFile).mockRejectedValue(new Error("EACCES"));

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_TMP_NOT_WRITABLE" })
    );
  });

  it("should warn when GOGENT_MCP_SPAWN_ENABLED=false", async () => {
    const originalEnv = process.env.GOGENT_MCP_SPAWN_ENABLED;
    process.env.GOGENT_MCP_SPAWN_ENABLED = "false";

    try {
      const result = await validateSpawnEnvironment();

      expect(result.ok).toBe(true); // Warnings don't fail
      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_SPAWN_DISABLED" })
      );
    } finally {
      process.env.GOGENT_MCP_SPAWN_ENABLED = originalEnv;
    }
  });
});

describe("formatValidationResult", () => {
  it("should format success result", () => {
    const result = { ok: true, errors: [], warnings: [] };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("✅ Environment validation passed");
  });

  it("should format errors with resolution", () => {
    const result = {
      ok: false,
      errors: [
        {
          code: "E_TEST",
          message: "Test error",
          resolution: "Fix the test",
        },
      ],
      warnings: [],
    };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("❌ Environment validation failed");
    expect(formatted).toContain("[E_TEST]");
    expect(formatted).toContain("Test error");
    expect(formatted).toContain("Fix the test");
  });
});

describe("assertValidSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should not throw when validation passes", async () => {
    await expect(assertValidSpawnEnvironment()).resolves.not.toThrow();
  });

  it("should throw when validation fails", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    await expect(assertValidSpawnEnvironment()).rejects.toThrow(
      /Spawn environment validation failed/
    );
  });
});
```

## Acceptance Criteria

- [ ] Validation module created with all 5 checks
- [ ] formatValidationResult produces clear, actionable output
- [ ] assertValidSpawnEnvironment throws on critical errors
- [ ] All tests pass: `npm test -- src/spawn/validation.test.ts`
- [ ] Code coverage ≥80%
- [ ] Integrated with TUI startup (call before MCP server registration)

## Test Deliverables

- [ ] Test file created: `packages/tui/src/spawn/validation.test.ts`
- [ ] Number of test functions: 7
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Mocks properly isolate external dependencies

