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
