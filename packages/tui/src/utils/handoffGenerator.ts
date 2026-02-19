/**
 * Handoff generation utility for provider switching
 *
 * Uses SDK query() to generate context summaries when switching between providers.
 * Async, non-blocking - provider switch happens immediately, handoff injected when ready.
 */

import { query, type SDKUserMessage } from "@anthropic-ai/claude-agent-sdk";
import type { MessageParam } from "@anthropic-ai/sdk/resources/messages";
import type { Message, ProviderId } from "../store/types.js";
import { logger } from "./logger.js";

/**
 * Build handoff prompt from recent messages
 */
export function buildHandoffPrompt(
  messages: Message[],
  fromProvider: ProviderId,
  toProvider: ProviderId
): string {
  // Format last 10 messages for context
  const recentMessages = messages.slice(-10);
  const messagesFormatted = recentMessages
    .map((msg) => {
      const textContent = msg.content
        .filter((block) => block.type === "text")
        .map((block) => (block as { text: string }).text)
        .join("\n");
      return `**${msg.role.toUpperCase()}**: ${textContent}`;
    })
    .join("\n\n");

  return `You are helping summarize a conversation for context transfer between AI providers.

The user was conversing with ${fromProvider} and is now switching to ${toProvider}.

Conversation history (last ${recentMessages.length} messages):
${messagesFormatted}

Generate a concise handoff summary (max 500 tokens) covering:
1. Key topics discussed
2. Active tasks or decisions
3. Files mentioned
4. Technical context (errors, code snippets)
5. What the user likely wants next

Format as markdown. Be concise.`;
}

/**
 * Generate handoff summary using SDK query()
 *
 * This is a fire-and-forget async operation. Returns null on error.
 */
export async function generateHandoff(
  messages: Message[],
  fromProvider: ProviderId,
  toProvider: ProviderId
): Promise<string | null> {
  try {
    const promptText = buildHandoffPrompt(messages, fromProvider, toProvider);

    // Create async generator for single message
    async function* messageGenerator(): AsyncIterableIterator<SDKUserMessage> {
      const userMessage: SDKUserMessage = {
        type: "user" as const,
        message: {
          role: "user" as const,
          content: [
            {
              type: "text" as const,
              text: promptText,
            },
          ],
        } as MessageParam,
        parent_tool_use_id: null,
        session_id: "",
      };

      yield userMessage;
    }

    // Use SDK query() with haiku model
    const eventStream = await query({
      prompt: messageGenerator(),
      options: {
        model: "claude-3-5-haiku-20241022",
        resume: undefined, // Ephemeral session
        includePartialMessages: false,
      },
    });

    // Collect response text
    let summary = "";

    for await (const event of eventStream) {
      if (event.type === "assistant" && event.message.content) {
        for (const block of event.message.content) {
          if (block.type === "text") {
            summary += block.text;
          }
        }
      }
    }

    return extractSummary(summary);
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : String(err);
    void logger.error("Handoff generation failed", { error: errorMessage });
    return null;
  }
}

/**
 * Extract clean summary from SDK response
 *
 * SDK may include metadata or formatting - extract the core content.
 */
export function extractSummary(result: string): string {
  // Trim whitespace
  let summary = result.trim();

  // If empty, return fallback
  if (!summary) {
    return "Context transferred from previous provider.";
  }

  // Limit to reasonable length (500 tokens ~= 2000 chars)
  if (summary.length > 2000) {
    summary = summary.slice(0, 2000) + "...";
  }

  return summary;
}
