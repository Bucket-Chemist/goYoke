/**
 * test_mcp_ping MCP tool
 * Minimal verification tool for MCP-SPAWN-001 gate test.
 * Verifies that MCP tools are accessible from Task()-spawned subagents.
 */

import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";

export const testMcpPingSchema = z.object({
  echo: z.string().optional().describe("Optional string to echo back"),
});

/**
 * Minimal MCP tool for verifying subagent accessibility.
 * Returns PONG with timestamp to prove invocation succeeded.
 */
export const testMcpPingTool = tool(
  "test_mcp_ping",
  "Verify MCP tool accessibility from subagents. Returns PONG with timestamp.",
  testMcpPingSchema.shape,
  async (args) => {
    const timestamp = new Date().toISOString();
    const response = {
      status: "PONG",
      timestamp,
      echo: args.echo || null,
      message: "MCP tool successfully invoked",
      sdkVersion: "0.2.31",
      gate: "MCP-SPAWN-001",
    };

    return {
      content: [
        {
          type: "text" as const,
          text: JSON.stringify(response, null, 2),
        },
      ],
    };
  }
);
