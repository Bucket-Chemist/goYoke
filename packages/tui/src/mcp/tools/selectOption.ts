/**
 * select_option MCP tool
 * Let the user select from a list of options
 */

import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useStore } from "../../store/index.js";
import type { ModalResponse } from "../../store/slices/modal.js";

export const selectOptionSchema = z.object({
  message: z.string().min(1, "Message must not be empty").describe("The message to display"),
  options: z
    .array(
      z.object({
        label: z.string().min(1, "Option label must not be empty").describe("Display label for the option"),
        value: z.string().describe("Value to return if selected"),
      })
    )
    .min(1, "Options array must contain at least one option")
    .describe("List of options to choose from"),
});

export const selectOptionTool = tool(
  "select_option",
  "Let the user select from a list of options",
  selectOptionSchema.shape,
  async (args) => {
    const response = (await useStore.getState().enqueue({
      type: "select",
      payload: {
        message: args.message,
        options: args.options,
      },
    })) as Extract<ModalResponse, { type: "select" }>;

    return {
      content: [
        {
          type: "text" as const,
          text: JSON.stringify({
            selected: response.selected,
            index: response.index,
          }),
        },
      ],
    };
  }
);
