/**
 * request_input MCP tool
 * Request text input from the user
 */

import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useStore } from "../../store/index.js";
import type { ModalResponse } from "../../store/slices/modal.js";

export const requestInputTool = tool(
  "request_input",
  "Request text input from the user",
  {
    prompt: z.string().describe("The prompt message to show"),
    placeholder: z.string().optional().describe("Placeholder text for the input field"),
  },
  async (args) => {
    const response = (await useStore.getState().enqueue({
      type: "input",
      payload: {
        prompt: args.prompt,
        placeholder: args.placeholder,
      },
    })) as Extract<ModalResponse, { type: "input" }>;

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
