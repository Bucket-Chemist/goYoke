import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn } from "child_process";
import { getProcessRegistry } from "../../spawn/processRegistry.js";
import { randomUUID } from "crypto";
import {
  validateAndRegisterSpawn,
  formatValidationResult,
  cleanupParentMutex,
} from "../../spawn/relationshipValidation.js";
import { getAgentsStore } from "../../spawn/storeAdapter.js";
import { getAgentConfig } from "../../spawn/agentConfig.js";
import { buildFullAgentContext } from "../../spawn/contextInjector.js";
import { logger, getSessionId } from "../../utils/logger.js";
import { useStore } from "../../store/index.js";
import { getSessionCostTracker } from "../../cost/tracker.js";

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
    caller_type: z.string().optional().describe("Self-identification of calling agent type (for Task-spawned agents like Mozart)"),
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
    const store = getAgentsStore();
    const timeout = args.timeout ?? DEFAULT_TIMEOUT;
    const startTime = Date.now();

    // Get parent info from store
    const parentId = process.env["GOGENT_PARENT_AGENT"] || null;
    const parentAgent = parentId ? store.get(parentId) : null;
    const parentTypeFromStore = parentAgent?.agentType;

    // Determine effective parent type:
    // 1. Store lookup (for spawn_agent-spawned parents)
    // 2. caller_type parameter (for Task-spawned parents like Mozart)
    // 3. Default to null (will become "router" in validation)
    const effectiveParentType = parentTypeFromStore || args.caller_type || null;

    // === RELATIONSHIP VALIDATION ===
    const validation = await validateAndRegisterSpawn(
      parentId,
      effectiveParentType,
      args.agent,
      agentId,
      store,
      !parentTypeFromStore && !!args.caller_type // Flag: using claimed type, needs bidirectional check
    );

    const sessionId = getSessionId();

    // Log validation result to file and console
    if (!validation.valid || validation.warnings.length > 0) {
      const formattedResult = formatValidationResult(validation);
      await logger.info(
        "Spawn validation result",
        {
          parentId,
          parentType: effectiveParentType,
          childAgent: args.agent,
          childId: agentId,
          valid: validation.valid,
          errors: validation.errors,
          warnings: validation.warnings,
          formattedResult,
        },
        sessionId
      );
    }

    // Block on validation errors
    if (!validation.valid) {
      await logger.error(
        "Spawn validation failed - blocking spawn",
        {
          parentId,
          parentType: effectiveParentType,
          childAgent: args.agent,
          childId: agentId,
          errors: validation.errors,
          warnings: validation.warnings,
        },
        sessionId
      );

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                agentId: null,
                agent: args.agent,
                success: false,
                error: `Spawn validation failed: ${validation.errors
                  .map((e) => e.message)
                  .join("; ")}`,
                validationErrors: validation.errors,
                validationWarnings: validation.warnings,
              },
              null,
              2
            ),
          },
        ],
      };
    }
    // === END VALIDATION ===

    // Build CLI arguments
    const cliArgs = buildCliArgs({ ...args, agent: args.agent });

    // Resolve effort level from agent config
    const agentConfig = getAgentConfig(args.agent);
    const effortLevel = agentConfig?.effortLevel;

    // Inject agent identity + rules + conventions into the prompt.
    // Mirrors what gogent-validate does for Task() and team-run does for batch spawns.
    const augmentedPrompt = await buildFullAgentContext(args.agent, agentConfig, args.prompt);

    return new Promise((resolve) => {
      // Build env with optional effort level override
      const spawnEnv: Record<string, string> = {
        ...process.env as Record<string, string>,
        GOGENT_NESTING_LEVEL: String(getCurrentNestingLevel() + 1),
        GOGENT_PARENT_AGENT: agentId,
        GOGENT_SPAWN_METHOD: "mcp-cli",
      };
      if (effortLevel) {
        spawnEnv["CLAUDE_CODE_EFFORT_LEVEL"] = effortLevel;
      }

      // Spawn CLI process (NO shell: true)
      const proc = spawn("claude", cliArgs, {
        stdio: ["pipe", "pipe", "pipe"],
        env: spawnEnv,
      });

      // Register with process registry
      registry.register(agentId, proc, args.agent);

      // Register agent in Zustand store for the agents panel
      const zustand = useStore.getState();
      const tierMap: Record<string, "haiku" | "sonnet" | "opus"> = {
        haiku: "haiku", sonnet: "sonnet", opus: "opus",
      };
      const resolvedModel = args.model || agentConfig?.model || "sonnet";

      // Ensure a synthetic "Router" root exists so the tree has a parent
      let effectiveParent = parentId;
      if (!zustand.rootAgentId) {
        console.warn("[spawnAgent] Root agent missing at first spawn — creating fallback");
        const routerId = "router-root";
        const activeModel = zustand.getActiveModel?.() ?? "opus";
        const fallbackTier: "haiku" | "sonnet" | "opus" =
          activeModel.includes("haiku") ? "haiku"
          : activeModel.includes("sonnet") ? "sonnet"
          : "opus";
        zustand.addAgent({
          id: routerId,
          parentId: null,
          model: activeModel,
          tier: fallbackTier,
          status: "running",
          description: "Router",
          agentType: "router",
          spawnMethod: "task",
        });
        effectiveParent = routerId;
      } else if (!effectiveParent) {
        effectiveParent = zustand.rootAgentId;
      }

      zustand.addAgent({
        id: agentId,
        parentId: effectiveParent,
        model: resolvedModel,
        tier: tierMap[resolvedModel] || "sonnet",
        status: "running",
        description: args.description,
        agentType: args.agent,
        spawnMethod: "mcp-cli",
        pid: proc.pid ?? undefined,
        childIds: [],
        activity: {
          lastText: args.description,
          currentTool: null,
          toolResult: { status: "pending" },
        },
      });

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

      // Send augmented prompt via stdin (identity + rules + conventions prepended)
      proc.stdin.write(augmentedPrompt);
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

        // Update store on timeout
        const s = useStore.getState();
        s.updateAgent(agentId, { status: "timeout", endTime: Date.now(), error: result.error });
        s.updateAgentActivity(agentId, {
          lastText: args.description,
          currentTool: null,
          toolResult: { status: "failed", error: `Timed out after ${timeout}ms` },
        });

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      }, timeout);

      // Process completion
      proc.on("close", (code, signal) => {
        clearTimeout(timer);

        // Cleanup mutex for this parent to prevent memory leak
        if (parentId) {
          cleanupParentMutex(parentId);
        }

        const duration = Date.now() - startTime;
        const parsed = parseCliOutput(stdout);

        // === COST TRACKING ===
        // Extract cost from CLI output and add to session tracker
        if (parsed.cost && parsed.cost > 0) {
          const tracker = getSessionCostTracker();
          tracker.addSpawnCost({
            agentId,
            agentType: args.agent,
            cost: parsed.cost,
            tokens: {
              input: parsed.inputTokens || 0,
              output: parsed.outputTokens || 0,
            },
            turns: parsed.turns || 0,
          });
        }
        // === END COST TRACKING ===

        const success = code === 0 && !signal;
        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success,
          output: parsed.result || stdout,
          error: code !== 0 ? stderr || `Exit code ${code}` : undefined,
          cost: parsed.cost,
          turns: parsed.turns,
          duration,
          truncated,
        };

        // Update store on completion
        const s = useStore.getState();
        s.updateAgent(agentId, {
          status: success ? "complete" : "error",
          endTime: Date.now(),
          cost: parsed.cost,
          turns: parsed.turns,
          toolCalls: parsed.turns,
          output: (parsed.result || stdout).slice(0, 500),
          error: result.error,
          tokenUsage: parsed.inputTokens || parsed.outputTokens
            ? { input: parsed.inputTokens || 0, output: parsed.outputTokens || 0 }
            : undefined,
        });
        s.updateAgentActivity(agentId, {
          lastText: success
            ? (parsed.result || "Complete").slice(0, 200)
            : result.error || "Failed",
          currentTool: null,
          toolResult: {
            status: success ? "success" : "failed",
            error: success ? undefined : result.error,
          },
        });

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      });

      proc.on("error", (err) => {
        clearTimeout(timer);

        // On error, remove child from parent (spawn failed after validation)
        if (parentId) {
          store.removeChild(parentId, agentId);
          cleanupParentMutex(parentId);
        }

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: false,
          error: `Spawn error: ${err.message}`,
          duration: Date.now() - startTime,
        };

        // Update store on spawn error
        const s = useStore.getState();
        s.updateAgent(agentId, { status: "error", endTime: Date.now(), error: result.error });
        s.updateAgentActivity(agentId, {
          lastText: result.error ?? null,
          currentTool: null,
          toolResult: { status: "failed", error: result.error },
        });

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
  agent?: string;
  model?: string;
  allowedTools?: string[];
  maxBudget?: number;
}): string[] {
  const cliArgs = ["-p", "--output-format", "json"];

  if (args.model) {
    // Inherit 1M context from root session — if the root model uses [1m],
    // propagate it to subagents so they don't fall back to 200K.
    // Only Opus and Sonnet support 1M context; Haiku does not.
    let model = args.model;
    const rootModel = useStore.getState().getActiveModel() ?? "";
    if (rootModel.includes("[1m]") && !model.includes("[1m]") && !model.includes("haiku")) {
      model = `${model}[1m]`;
    }
    cliArgs.push("--model", model);
  }

  // NOTE: --permission-mode is intentionally NOT passed. "delegate" is not a valid
  // CLI mode (valid: default, acceptEdits, plan, dontAsk, bypassPermissions, auto).
  // The spawned subprocess inherits the default mode from settings.json.
  // Tool access is controlled via --allowedTools instead.
  // This matches the Go spawner (gofortress-mcp-standalone/spawner.go:buildSpawnArgs).

  // Resolve allowed tools: caller > config > fallback
  let resolvedTools: string[] | undefined = args.allowedTools;

  if ((!resolvedTools || resolvedTools.length === 0) && args.agent) {
    const config = getAgentConfig(args.agent);
    if (config?.cli_flags?.allowed_tools && config.cli_flags.allowed_tools.length > 0) {
      resolvedTools = config.cli_flags.allowed_tools;
    }
  }

  // Conservative fallback if nothing resolved
  if (!resolvedTools || resolvedTools.length === 0) {
    resolvedTools = ["Read", "Glob", "Grep"];
  }

  cliArgs.push("--allowedTools", resolvedTools.join(","));

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
  inputTokens?: number;
  outputTokens?: number;
} {
  try {
    const json = JSON.parse(stdout.trim());
    return {
      result: json.result || json.output,
      cost: json.cost_usd || json.total_cost_usd,
      turns: json.num_turns,
      inputTokens: json.input_tokens,
      outputTokens: json.output_tokens,
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
  const level = process.env['GOGENT_NESTING_LEVEL'];
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
