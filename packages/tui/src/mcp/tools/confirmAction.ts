/**
 * confirm_action MCP tool
 * Request user confirmation for an action
 */

import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useStore } from "../../store/index.js";
import type { ModalResponse } from "../../store/slices/modal.js";

export const confirmActionSchema = z.object({
  action: z.string().min(1, "Action description must not be empty").describe("Description of the action to confirm"),
  destructive: z.boolean().optional().describe("Whether action is destructive"),
});

export const confirmActionTool = tool(
  "confirm_action",
  "Request user confirmation for an action",
  confirmActionSchema.shape,
  async (args) => {
    const response = (await useStore.getState().enqueue({
      type: "confirm",
      payload: {
        action: args.action,
        destructive: args.destructive ?? false,
      },
    })) as Extract<ModalResponse, { type: "confirm" }>;

    return {
      content: [
        {
          type: "text" as const,
          text: JSON.stringify({
            confirmed: response.confirmed,
            cancelled: response.cancelled,
          }),
        },
      ],
    };
  }
);
