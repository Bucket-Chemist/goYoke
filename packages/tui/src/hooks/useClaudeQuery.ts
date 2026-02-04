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
  SDKUserMessage,
} from "@anthropic-ai/claude-agent-sdk";
import { useStore } from "../store/index.js";
import { mcpServer } from "../mcp/server.js";
import type { ClassifiedError } from "../types/events.js";
import type { ContentBlock } from "../store/types.js";
import fs from "fs";

/**
 * Debug configuration
 */
const DEBUG_EVENTS = false; // Set to true for debugging
const DEBUG_FILE = "/tmp/tui-events.jsonl";

/**
 * Log SDK events for debugging
 */
const logEvent = (event: unknown) => {
  if (!DEBUG_EVENTS) return;

  const summary = {
    timestamp: new Date().toISOString(),
    type: (event as any)?.type,
    subtype: (event as any)?.subtype,
    keys: Object.keys(event as object),
    preview: JSON.stringify(event).slice(0, 500),
  };

  console.log("[SDK Event]", summary.type, summary.subtype || "");

  // Append to file (async, fire-and-forget)
  try {
    fs.appendFileSync(DEBUG_FILE, JSON.stringify(summary) + "\n");
  } catch {
    // Ignore errors - debug logging should never crash the app
  }
};

/**
 * Hook return type
 */
interface UseClaudeQueryReturn {
  sendMessage: (message: string) => Promise<void>;
  setModel: (modelId: string) => Promise<boolean>;
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
 * SDK built-in tools that require streamInput handling
 */
const SDK_BUILTIN_TOOLS = [
  'AskUserQuestion',
  'RequestInput',
  'ConfirmAction',
  'EnterPlanMode',
  'ExitPlanMode',
] as const;

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
  const setPermissionMode = useStore((state) => state.setPermissionMode);
  const setCompacting = useStore((state) => state.setCompacting);

  // Track current assistant message being built
  const currentMessageRef = useRef<ContentBlock[]>([]);

  // Track active query for setModel calls and streamInput
  const eventStreamRef = useRef<Awaited<ReturnType<typeof query>> | null>(null);

  /**
   * Handle system event - initialize session metadata
   */
  const handleSystemEvent = useCallback(
    (event: SDKSystemMessage) => {
      // Extract model from init message (SDK provides this authoritatively)
      const initEvent = event as SDKSystemMessage & { model?: string };
      if (initEvent.model) {
        fs.appendFileSync("/tmp/tui-model-debug.log",
          `[${new Date().toISOString()}] SDK Init returned model: ${initEvent.model}\n`);
        useStore.getState().setActiveModel(initEvent.model);
      }

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
   * Handle status event - track permission mode and compacting state
   */
  const handleStatusEvent = useCallback(
    (event: SDKMessage) => {
      // Type-narrow to status event structure
      const statusEvent = event as {
        type: "system";
        subtype: "status";
        status?: "compacting" | null;
        permissionMode?: string;
      };

      // Update compacting state if present
      if (statusEvent.status !== undefined) {
        setCompacting(statusEvent.status === "compacting");
      }

      // Update permission mode if present (triggers plan mode detection)
      if (statusEvent.permissionMode) {
        setPermissionMode(statusEvent.permissionMode);

        // Debug logging
        console.log(
          "[Status] Permission mode:",
          statusEvent.permissionMode,
          "→ Plan mode:",
          statusEvent.permissionMode === "plan"
        );
      }
    },
    [setPermissionMode, setCompacting]
  );

  /**
   * Handle SDK built-in tool_use that requires streamInput response
   */
  const handleBuiltinToolUse = useCallback(
    async (block: { id: string; name: string; input: unknown }) => {
      const sessionId = useStore.getState().sessionId;

      if (!sessionId) {
        console.error('[handleBuiltinToolUse] No session ID available');
        return;
      }

      try {
        // Handle different built-in tool types
        let toolResult: string;

        if (block.name === 'AskUserQuestion') {
          // AskUserQuestion structure from SDK
          const input = block.input as {
            questions?: Array<{
              question: string;
              header?: string;
              options?: Array<{ label: string; value?: string }>;
            }>;
          };

          const question = input.questions?.[0];
          if (!question) {
            toolResult = 'Error: No question provided';
          } else {
            // Preserve SDK option structure {label, value}
            const options = question.options?.map(o => ({
              label: o.label,
              value: o.value || o.label, // Fallback to label if value missing
            })) || [];

            // Show modal and wait for user response
            const modalResult = await useStore.getState().enqueue({
              type: 'ask',
              payload: {
                message: question.question,
                options,
              },
              timeout: 120000, // 2 minutes for user questions
            });

            if (modalResult.type === 'ask') {
              toolResult = modalResult.value;
            } else {
              toolResult = 'Question cancelled by user';
            }
          }
        } else if (block.name === 'RequestInput') {
          // Generic input request
          const input = block.input as { prompt?: string; placeholder?: string };

          const modalResult = await useStore.getState().enqueue({
            type: 'input',
            payload: {
              prompt: input.prompt || 'Enter input',
              placeholder: input.placeholder,
            },
            timeout: 120000,
          });

          if (modalResult.type === 'input') {
            toolResult = modalResult.value;
          } else {
            toolResult = 'Input cancelled by user';
          }
        } else if (block.name === 'ConfirmAction') {
          // Confirmation prompt
          const input = block.input as { action?: string };

          const modalResult = await useStore.getState().enqueue({
            type: 'confirm',
            payload: {
              action: input.action || 'Confirm this action?',
            },
            timeout: 60000,
          });

          if (modalResult.type === 'confirm') {
            toolResult = modalResult.confirmed ? 'confirmed' : 'denied';
          } else {
            toolResult = 'cancelled';
          }
        } else {
          // Other built-in tools (EnterPlanMode, ExitPlanMode)
          // These are typically handled automatically by SDK
          console.warn(`[handleBuiltinToolUse] Unhandled built-in tool: ${block.name}`);
          return;
        }

        // Send tool result back to Claude via streamInput
        if (eventStreamRef.current) {
          await eventStreamRef.current.streamInput(
            (async function* () {
              yield {
                type: 'user' as const,
                message: {
                  role: 'user' as const,
                  content: [
                    {
                      type: 'tool_result' as const,
                      tool_use_id: block.id,
                      content: toolResult,
                    },
                  ],
                },
                parent_tool_use_id: block.id,
                tool_use_result: toolResult,
                session_id: sessionId,
              };
            })()
          );

          console.log(`[handleBuiltinToolUse] Sent result for ${block.name}:`, toolResult);
        }
      } catch (err) {
        console.error('[handleBuiltinToolUse] Error:', err);

        // Send error response back to Claude
        if (eventStreamRef.current) {
          await eventStreamRef.current.streamInput(
            (async function* () {
              yield {
                type: 'user' as const,
                message: {
                  role: 'user' as const,
                  content: [
                    {
                      type: 'tool_result' as const,
                      tool_use_id: block.id,
                      content: `Error: ${err instanceof Error ? err.message : String(err)}`,
                      is_error: true,
                    },
                  ],
                },
                parent_tool_use_id: block.id,
                session_id: useStore.getState().sessionId || '',
              };
            })()
          );
        }
      }
    },
    []
  );

  /**
   * Handle assistant event - add or update message
   */
  const handleAssistantEvent = useCallback(
    async (event: SDKAssistantMessage) => {
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

      // Check for SDK built-in tools that need streamInput handling
      const builtinTools = event.message.content.filter(
        (block) => block.type === 'tool_use' && SDK_BUILTIN_TOOLS.includes(block.name as any)
      );

      // Handle built-in tools asynchronously (don't block message display)
      if (builtinTools.length > 0) {
        // Process each built-in tool
        for (const block of builtinTools) {
          if (block.type === 'tool_use') {
            // Handle in background - don't await here to avoid blocking message rendering
            handleBuiltinToolUse({
              id: block.id,
              name: block.name,
              input: block.input,
            }).catch((err) => {
              console.error('[handleAssistantEvent] Built-in tool error:', err);
            });
          }
        }
      }

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
    [isStreaming, addMessage, updateLastMessage, handleBuiltinToolUse]
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
      // GUARD: Prevent concurrent queries (fixes multiple session spawning bug)
      if (isStreaming) {
        console.warn('[sendMessage] Query already in progress, ignoring duplicate call');
        return;
      }

      try {
        // Reset error state
        setError(null);
        currentMessageRef.current = [];

        // Get preferred model if set (for pre-query model selection)
        const preferredModel = useStore.getState().preferredModel;

        // DEBUG: Log to file (Ink captures stdout)
        fs.appendFileSync("/tmp/tui-model-debug.log",
          `[${new Date().toISOString()}] preferredModel from store: ${preferredModel}\n`);

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

        // DEBUG: Log query options to file
        const queryModel = preferredModel || undefined;
        fs.appendFileSync("/tmp/tui-model-debug.log",
          `[${new Date().toISOString()}] Calling query() with model: ${queryModel}\n`);

        // Query Claude with MCP server registration and GOgent settings
        const eventStream = query({
          prompt: message,
          options: {
            // Apply preferred model if set (allows /model before first message)
            model: queryModel,
            // Load GOgent-Fortress settings (hooks, CLAUDE.md, etc.)
            settingSources: ['user', 'project', 'local'],
            // SDK expects specific config format - mcpServers array type is more specific than our config
            mcpServers: [mcpServer] as unknown as NonNullable<Parameters<typeof query>[0]["options"]>["mcpServers"],
            // Permission callback - prompts user before tool execution
            canUseTool: async (
              toolName: string,
              input: Record<string, unknown>,
              options: {
                signal: AbortSignal;
                suggestions?: unknown[];
                blockedPath?: string;
                decisionReason?: string;
                toolUseID: string;
                agentID?: string;
              }
            ) => {
              // Define destructive tools that need special UI treatment
              const destructiveTools = ['Bash', 'Write', 'Edit', 'MultiEdit', 'NotebookEdit'];
              const isDestructive = destructiveTools.includes(toolName);

              // Create input preview (truncate if too long)
              const inputPreview = JSON.stringify(input, null, 2);
              const truncatedPreview = inputPreview.length > 200
                ? inputPreview.slice(0, 200) + '...'
                : inputPreview;

              // Build action description
              const actionDescription = options.blockedPath
                ? `Allow ${toolName} to access "${options.blockedPath}"?`
                : `Allow Claude to use ${toolName}?`;

              // Enqueue permission modal
              const result = await useStore.getState().enqueue({
                type: 'confirm',
                payload: {
                  action: `${actionDescription}\n\nInput:\n${truncatedPreview}`,
                  destructive: isDestructive,
                },
                timeout: 60000, // 60 second timeout
              });

              // Handle abort signal (if request was cancelled)
              if (options.signal.aborted) {
                return {
                  behavior: 'deny' as const,
                  message: 'Request aborted',
                  toolUseID: options.toolUseID,
                };
              }

              // Return SDK-expected format
              // NOTE: SDK Zod schema requires updatedInput even though TS type marks it optional
              if (result.type === 'confirm' && result.confirmed) {
                return {
                  behavior: 'allow' as const,
                  updatedInput: input,  // Pass through original input
                  toolUseID: options.toolUseID,
                };
              } else {
                return {
                  behavior: 'deny' as const,
                  message: 'User denied permission',
                  toolUseID: options.toolUseID,
                };
              }
            },
          },
        });

        // Store reference for setModel calls
        eventStreamRef.current = eventStream;

        // NOTE: Don't clear preferredModel - it should persist for all
        // subsequent messages until user explicitly changes it via /model

        // Iterate over events
        for await (const event of eventStream) {
          // DEBUG: Log all SDK events
          logEvent(event);

          const sdkMessage = event as SDKMessage;

          // Type-safe event handling
          if (sdkMessage.type === "system") {
            if (sdkMessage.subtype === "init") {
              handleSystemEvent(sdkMessage);
            } else if (sdkMessage.subtype === "status") {
              handleStatusEvent(sdkMessage);
            }
          } else if (sdkMessage.type === "assistant") {
            handleAssistantEvent(sdkMessage);
          } else if (sdkMessage.type === "result") {
            handleResultEvent(sdkMessage);
          }
          // Other event types logged but not processed
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
      isStreaming, // CRITICAL: Required for re-entry guard
      addMessage,
      handleSystemEvent,
      handleStatusEvent,
      handleAssistantEvent,
      handleResultEvent,
      setStreamingState,
    ]
  );

  /**
   * Switch the model for the active query
   * Returns true on success, false if no active query or on error
   */
  const setModel = useCallback(async (modelId: string): Promise<boolean> => {
    if (!eventStreamRef.current) {
      console.warn('[setModel] No active query. Send a message first.');
      return false;
    }

    try {
      await eventStreamRef.current.setModel(modelId);
      console.log(`[setModel] Successfully switched to ${modelId}`);
      return true;
    } catch (err) {
      console.error('[setModel] Failed:', err);
      return false;
    }
  }, []);

  return {
    sendMessage,
    setModel,
    isStreaming,
    error,
  };
}
