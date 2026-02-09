/**
 * MCP server with interactive tools
 * Bridges Claude API calls to TUI modal queue system
 */

import { createSdkMcpServer } from "@anthropic-ai/claude-agent-sdk";
import { askUserTool } from "./tools/askUser.js";
import { confirmActionTool } from "./tools/confirmAction.js";
import { requestInputTool } from "./tools/requestInput.js";
import { selectOptionTool } from "./tools/selectOption.js";
import { testMcpPingTool } from "./tools/testMcpPing.js";
import { spawnAgent } from "./tools/spawnAgent.js";

/**
 * Check if MCP spawning is enabled via feature flag.
 * Defaults to enabled unless explicitly set to "false".
 */
export function isSpawnEnabled(): boolean {
  return process.env['GOGENT_MCP_SPAWN_ENABLED'] !== "false";
}

/**
 * Get all tools that should be registered based on feature flags.
 * Exported for testing.
 */
export function getServerTools() {
  const tools = [
    // Interactive tools
    askUserTool,
    confirmActionTool,
    requestInputTool,
    selectOptionTool,
    // Test tool (always available for verification)
    testMcpPingTool,
  ];

  // Conditionally add spawn tools
  if (isSpawnEnabled()) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- tool schemas vary
    tools.push(spawnAgent as any);
  }

  return tools;
}

/**
 * Create MCP server with all tools.
 * Conditionally includes spawn tools based on feature flag.
 */
export function createMcpServer() {
  const tools = getServerTools();

  return createSdkMcpServer({
    name: "gofortress-interactive",
    version: "1.0.0",
    tools,
  });
}

/**
 * In-process MCP server instance
 * Provides interactive tools that integrate with the TUI modal queue
 */
export const mcpServer = createMcpServer();
