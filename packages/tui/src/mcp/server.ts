/**
 * MCP server with interactive tools
 * Bridges Claude API calls to TUI modal queue system
 */

import { createSdkMcpServer } from "@anthropic-ai/claude-agent-sdk";
import { askUserTool } from "./tools/askUser.js";
import { confirmActionTool } from "./tools/confirmAction.js";
import { requestInputTool } from "./tools/requestInput.js";
import { selectOptionTool } from "./tools/selectOption.js";

/**
 * In-process MCP server instance
 * Provides interactive tools that integrate with the TUI modal queue
 */
export const mcpServer = createSdkMcpServer({
  name: "gofortress-interactive",
  version: "1.0.0",
  tools: [askUserTool, confirmActionTool, requestInputTool, selectOptionTool],
});
