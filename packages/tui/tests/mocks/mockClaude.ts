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
