import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn } from "child_process";
import { getProcessRegistry } from "../../spawn/processRegistry";
import { randomUUID } from "crypto";

// Constants
const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB
const DEFAULT_TIMEOUT = 300000; // 5 minutes
const MAX_NESTING_DEPTH = 10;

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
    // === DEPTH VALIDATION ===
    const depthError = validateNestingDepth();
    if (depthError) {
      return {
        content: [{
          type: "text",
          text: JSON.stringify({
            agentId: null,
            agent: args.agent,
            success: false,
            error: depthError,
            errorCode: "E_MAX_DEPTH_EXCEEDED",
          }, null, 2),
        }],
      };
    }
    // === END DEPTH VALIDATION ===

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
export function buildCliArgs(args: {
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
export function parseCliOutput(stdout: string): {
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
export function getCurrentNestingLevel(): number {
  const level = process.env.GOGENT_NESTING_LEVEL;
  if (!level) return 0;
  const parsed = parseInt(level, 10);
  return isNaN(parsed) ? 0 : parsed;
}

/**
 * Validate nesting depth before spawning.
 * Returns error message if depth exceeded, null if OK.
 */
export function validateNestingDepth(): string | null {
  const currentLevel = getCurrentNestingLevel();

  if (currentLevel >= MAX_NESTING_DEPTH) {
    return `Maximum nesting depth (${MAX_NESTING_DEPTH}) exceeded. ` +
           `Current level: ${currentLevel}. ` +
           `Cannot spawn sub-agent at this depth.`;
  }

  return null;
}
