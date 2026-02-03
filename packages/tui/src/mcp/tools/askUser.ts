/**
 * ask_user MCP tool
 * Asks the user a question with optional predefined options
 */

import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useStore } from "../../store/index.js";
import type { ModalResponse } from "../../store/slices/modal.js";

export const askUserTool = tool(
  "ask_user",
  "Ask the user a question with optional predefined options",
  {
    message: z.string().describe("The question to ask"),
    options: z.array(z.string()).optional().describe("Predefined answer options"),
    default: z.string().optional().describe("Default value if timeout"),
  },
  async (args) => {
    const response = (await useStore.getState().enqueue({
      type: "ask",
      payload: {
        message: args.message,
        options: args.options,
        defaultValue: args.default,
      },
    })) as Extract<ModalResponse, { type: "ask" }>;

    return {
      content: [
        {
          type: "text" as const,
          text: response.value,
        },
      ],
    };
  }
);
