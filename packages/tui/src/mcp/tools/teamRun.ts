import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn } from "child_process";
import { readFileSync } from "fs";

/**
 * team_run MCP tool - launches gogent-team-run as a detached background process.
 *
 * This is the TUI equivalent of `Bash("gogent-team-run $team_dir")`.
 * CLI users dispatch via Bash; TUI-spawned subagents (who lack Bash) use this.
 */
export const teamRun = tool(
  "team_run",
  `Launch a gogent-team-run background process for a team configuration directory.

Use this when you need to start a team execution (braintrust, review, implementation)
from a TUI context where Bash is not available.

The team directory must contain a valid config.json created by an orchestrator (e.g., Mozart).

Example:
  team_run({
    team_dir: "/home/user/.claude/sessions/20260210/teams/1707580800.braintrust"
  })`,
  {
    team_dir: z
      .string()
      .describe(
        "Absolute path to team config directory containing config.json"
      ),
    wait_for_start: z
      .boolean()
      .optional()
      .describe("Poll for daemon PID in config.json (default: true)"),
    timeout_ms: z
      .number()
      .optional()
      .describe("Startup verification timeout in ms (default: 5000)"),
  },
  async (args): Promise<{ content: Array<{ type: "text"; text: string }> }> => {
    const teamDir = args.team_dir;
    const waitForStart = args.wait_for_start ?? true;
    const timeoutMs = args.timeout_ms ?? 5000;

    // Validate team_dir exists and contains config.json
    try {
      readFileSync(`${teamDir}/config.json`, "utf-8");
    } catch {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                success: false,
                team_dir: teamDir,
                error: `config.json not found in ${teamDir}`,
              },
              null,
              2
            ),
          },
        ],
      };
    }

    // Spawn gogent-team-run as detached background process
    try {
      const proc = spawn("gogent-team-run", [teamDir], {
        detached: true,
        stdio: "ignore",
      });

      // Handle spawn errors (e.g., binary not found, permission denied)
      proc.on("error", (err) => {
        console.error(`[team_run] Failed to spawn gogent-team-run: ${err.message}`);
      });

      proc.unref(); // Allow parent to exit independently

      // If not waiting for start, return immediately
      if (!waitForStart) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(
                {
                  success: true,
                  team_dir: teamDir,
                  background_pid: null,
                  monitor: "/team-status",
                  result: "/team-result",
                  cancel: "/team-cancel",
                },
                null,
                2
              ),
            },
          ],
        };
      }

      // Poll config.json for background_pid with exponential backoff
      const startTime = Date.now();
      let backgroundPid: number | null = null;
      let delay = 100; // Start at 100ms, increase to max 500ms

      while (Date.now() - startTime < timeoutMs) {
        await new Promise((resolve) => setTimeout(resolve, delay));
        delay = Math.min(delay * 2, 500); // Exponential backoff, cap at 500ms

        try {
          const configData = readFileSync(`${teamDir}/config.json`, "utf-8");
          const config = JSON.parse(configData);
          if (config.background_pid && config.background_pid !== null) {
            backgroundPid = config.background_pid;
            break;
          }
        } catch {
          // Config may be mid-write, retry
        }
      }

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                success: true,
                team_dir: teamDir,
                background_pid: backgroundPid,
                monitor: "/team-status",
                result: "/team-result",
                cancel: "/team-cancel",
              },
              null,
              2
            ),
          },
        ],
      };
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : String(err);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                success: false,
                team_dir: teamDir,
                error: `Failed to spawn gogent-team-run: ${errorMessage}`,
              },
              null,
              2
            ),
          },
        ],
      };
    }
  }
);
