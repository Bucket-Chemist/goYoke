/**
 * useClaudeQuery hook - React hook for Claude Agent SDK query integration
 * Connects Claude SDK query() to Zustand store with event streaming
 */

import { useCallback, useRef, useState } from "react";
import { query } from "@anthropic-ai/claude-agent-sdk";
import type {
  SDKMessage,
  SDKSystemMessage,
  SDKAssistantMessage,
  SDKResultMessage,
} from "@anthropic-ai/claude-agent-sdk";
import { useStore } from "../store/index.js";
import { mcpServer } from "../mcp/server.js";
import type { ClassifiedError } from "../types/events.js";
import type { ContentBlock } from "../store/types.js";

/**
 * Hook return type
 */
interface UseClaudeQueryReturn {
  sendMessage: (message: string) => Promise<void>;
  isStreaming: boolean;
  error: ClassifiedError | null;
}

/**
 * Classify error for better error handling
 */
function classifyError(error: unknown): ClassifiedError {
  const errorMessage =
    error instanceof Error ? error.message : String(error);

  // Network errors
  if (
    errorMessage.includes("ECONNREFUSED") ||
    errorMessage.includes("ENOTFOUND") ||
    errorMessage.includes("network")
  ) {
    return {
      type: "network",
      message: "Network connection failed. Check your internet connection.",
      originalError: error,
      retryable: true,
    };
  }

  // Authentication errors
  if (
    errorMessage.includes("401") ||
    errorMessage.includes("authentication") ||
    errorMessage.includes("API key")
  ) {
    return {
      type: "auth",
      message: "Authentication failed. Check your API key.",
      originalError: error,
      retryable: false,
    };
  }

  // Rate limiting
  if (errorMessage.includes("429") || errorMessage.includes("rate limit")) {
    return {
      type: "rate_limit",
      message: "Rate limit exceeded. Please wait before retrying.",
      originalError: error,
      retryable: true,
    };
  }

  // Invalid request
  if (errorMessage.includes("400") || errorMessage.includes("invalid")) {
    return {
      type: "invalid_request",
      message: "Invalid request. Check your input.",
      originalError: error,
      retryable: false,
    };
  }

  // Server errors
  if (errorMessage.includes("500") || errorMessage.includes("server error")) {
    return {
      type: "server_error",
      message: "Server error. Please try again later.",
      originalError: error,
      retryable: true,
    };
  }

  // Timeout
  if (errorMessage.includes("timeout") || errorMessage.includes("ETIMEDOUT")) {
    return {
      type: "timeout",
      message: "Request timed out. Please try again.",
      originalError: error,
      retryable: true,
    };
  }

  // Unknown error
  return {
    type: "unknown",
    message: errorMessage || "An unknown error occurred",
    originalError: error,
    retryable: false,
  };
}

/**
 * Main hook for Claude query integration
 */
export function useClaudeQuery(): UseClaudeQueryReturn {
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<ClassifiedError | null>(null);

  // Store actions
  const addMessage = useStore((state) => state.addMessage);
  const updateLastMessage = useStore((state) => state.updateLastMessage);
  const updateSession = useStore((state) => state.updateSession);
  const incrementCost = useStore((state) => state.incrementCost);
  const addTokens = useStore((state) => state.addTokens);
  const setStreamingState = useStore((state) => state.setStreaming);

  // Track current assistant message being built
  const currentMessageRef = useRef<ContentBlock[]>([]);

  /**
   * Handle system event - initialize session metadata
   */
  const handleSystemEvent = useCallback(
    (event: SDKSystemMessage) => {
      // Update session with ID
      updateSession({
        id: event.session_id,
      });

      // Initialize agents from system event if present
      // Note: SDK system messages may contain agent metadata in future versions
      // For now, we just initialize the session
    },
    [updateSession]
  );

  /**
   * Handle assistant event - add or update message
   */
  const handleAssistantEvent = useCallback(
    (event: SDKAssistantMessage) => {
      // Convert BetaMessage content to ContentBlock format
      const contentBlocks: ContentBlock[] = event.message.content.map(
        (block) => {
          if (block.type === "text") {
            return {
              type: "text" as const,
              text: block.text,
            };
          } else if (block.type === "tool_use") {
            return {
              type: "tool_use" as const,
              id: block.id,
              name: block.name,
              input: block.input as Record<string, unknown>,
            };
          }
          // Fallback to text block (shouldn't happen with proper types)
          return {
            type: "text" as const,
            text: "",
          };
        }
      );

      // If streaming, update the last message
      if (isStreaming && currentMessageRef.current.length > 0) {
        currentMessageRef.current = contentBlocks;
        updateLastMessage(contentBlocks);
      } else {
        // First chunk - add new message
        currentMessageRef.current = contentBlocks;
        addMessage({
          role: "assistant",
          content: contentBlocks,
          partial: true,
        });
      }
    },
    [isStreaming, addMessage, updateLastMessage]
  );

  /**
   * Handle result event - finalize message and update stats
   */
  const handleResultEvent = useCallback(
    (event: SDKResultMessage) => {
      // Mark message as complete if we have one
      if (currentMessageRef.current.length > 0) {
        updateLastMessage(currentMessageRef.current);
        currentMessageRef.current = [];
      }

      // Update usage statistics (available on both success and error)
      incrementCost(event.total_cost_usd);

      if (event.usage) {
        addTokens({
          input: event.usage.input_tokens,
          output: event.usage.output_tokens,
        });
      }

      // Handle error result
      if (event.subtype !== "success") {
        const errorMessages =
          "errors" in event ? event.errors.join("; ") : "Query failed";
        setError({
          type: "server_error",
          message: errorMessages,
          retryable: true,
        });
      }

      // Stop streaming
      setIsStreaming(false);
      setStreamingState(false);
    },
    [
      updateLastMessage,
      incrementCost,
      addTokens,
      setIsStreaming,
      setStreamingState,
    ]
  );

  /**
   * Send message to Claude and handle streaming events
   */
  const sendMessage = useCallback(
    async (message: string): Promise<void> => {
      try {
        // Reset error state
        setError(null);
        currentMessageRef.current = [];

        // Add user message to store
        addMessage({
          role: "user",
          content: [
            {
              type: "text",
              text: message,
            },
          ],
          partial: false,
        });

        // Start streaming
        setIsStreaming(true);
        setStreamingState(true);

        // Query Claude with MCP server registration and GOgent settings
        const eventStream = query({
          prompt: message,
          options: {
            // Load GOgent-Fortress settings (hooks, CLAUDE.md, etc.)
            settingSources: ['user', 'project', 'local'],
            // SDK expects specific config format - mcpServers array type is more specific than our config
            mcpServers: [mcpServer] as unknown as NonNullable<Parameters<typeof query>[0]["options"]>["mcpServers"],
          },
        });

        // Iterate over events
        for await (const event of eventStream) {
          const sdkMessage = event as SDKMessage;

          // Type-safe event handling
          if (sdkMessage.type === "system" && sdkMessage.subtype === "init") {
            handleSystemEvent(sdkMessage);
          } else if (sdkMessage.type === "assistant") {
            handleAssistantEvent(sdkMessage);
          } else if (sdkMessage.type === "result") {
            handleResultEvent(sdkMessage);
          }
          // Ignore other event types (stream_event, status, etc.)
        }
      } catch (err) {
        // Classify and store error
        const classifiedError = classifyError(err);
        setError(classifiedError);

        // Add error message to chat
        addMessage({
          role: "assistant",
          content: [
            {
              type: "text",
              text: `Error: ${classifiedError.message}`,
            },
          ],
          partial: false,
        });

        // Stop streaming
        setIsStreaming(false);
        setStreamingState(false);

        console.error("Query error:", err);
      }
    },
    [
      addMessage,
      handleSystemEvent,
      handleAssistantEvent,
      handleResultEvent,
      setStreamingState,
    ]
  );

  return {
    sendMessage,
    isStreaming,
    error,
  };
}
