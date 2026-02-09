/**
 * useClaudeQuery hook - React hook for Claude Agent SDK query integration
 * Connects Claude SDK query() to Zustand store with event streaming
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { query } from "@anthropic-ai/claude-agent-sdk";
import type {
  SDKMessage,
  SDKSystemMessage,
  SDKAssistantMessage,
  SDKResultMessage,
  SDKUserMessage,
} from "@anthropic-ai/claude-agent-sdk";
import { join } from "path";
import { homedir } from "os";
import { useStore } from "../store/index.js";
import { mcpServer } from "../mcp/server.js";
import type { ClassifiedError } from "../types/events.js";
import type { ContentBlock } from "../store/types.js";
import { logger } from "../utils/logger.js";

/**
 * Debug configuration
 */
const DEBUG_EVENTS = false; // Set to true for debugging

/**
 * Log SDK events for debugging
 */
const logEvent = (event: unknown) => {
  if (!DEBUG_EVENTS) return;

  const summary = {
    type: (event as any)?.type,
    subtype: (event as any)?.subtype,
    keys: Object.keys(event as object),
    preview: JSON.stringify(event).slice(0, 500),
  };

  // Log to file only - never stdout/stderr
  void logger.debug("[SDK Event]", summary);
};

/**
 * Hook options
 */
interface UseClaudeQueryOptions {
  onStreamingComplete?: () => void;
}

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
export function useClaudeQuery(options?: UseClaudeQueryOptions): UseClaudeQueryReturn {
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<ClassifiedError | null>(null);

  // Store the callback ref to avoid stale closures
  const onStreamingCompleteRef = useRef(options?.onStreamingComplete);

  // Keep callback ref in sync with latest version
  useEffect(() => {
    onStreamingCompleteRef.current = options?.onStreamingComplete;
  }, [options?.onStreamingComplete]);

  // Sync streamingRef with isStreaming state to avoid stale closures
  useEffect(() => {
    streamingRef.current = isStreaming;
  }, [isStreaming]);

  // Store actions
  const addMessage = useStore((state) => state.addMessage);
  const updateLastMessage = useStore((state) => state.updateLastMessage);
  const updateSession = useStore((state) => state.updateSession);
  const incrementCost = useStore((state) => state.incrementCost);
  const addTokens = useStore((state) => state.addTokens);
  const updateContextWindow = useStore((state) => state.updateContextWindow);
  const setStreamingState = useStore((state) => state.setStreaming);
  const setPermissionMode = useStore((state) => state.setPermissionMode);
  const setCompacting = useStore((state) => state.setCompacting);
  const setInterruptQuery = useStore((state) => state.setInterruptQuery);

  // Track current assistant message being built
  const currentMessageRef = useRef<ContentBlock[]>([]);

  // Track current assistant message ID to distinguish streaming updates from new messages
  const currentMessageIdRef = useRef<string | null>(null);

  // Track streaming state with ref to avoid stale closures
  const streamingRef = useRef(false);

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
        void logger.debug("SDK Init returned model", { model: initEvent.model });
        useStore.getState().setActiveModel(initEvent.model);
      }

      // Update session with ID and set session dir for child processes/team polling
      updateSession({
        id: event.session_id,
      });

      // Set GOGENT_SESSION_DIR so team polling and child processes can find the session
      if (event.session_id && !process.env["GOGENT_SESSION_DIR"]) {
        const home = process.env["HOME"] || homedir();
        const sessionDirPath = join(home, ".claude", "sessions", event.session_id);
        process.env["GOGENT_SESSION_DIR"] = sessionDirPath;

        // Write current-session marker + setup tmp symlink (best-effort)
        const projectRoot = process.env["GOGENT_PROJECT_DIR"] || process.cwd();
        void (async () => {
          try {
            const { writeFile, mkdir, unlink, symlink, lstat } = await import("fs/promises");
            await mkdir(sessionDirPath, { recursive: true });
            await writeFile(join(projectRoot, ".claude", "current-session"), sessionDirPath + "\n");
            const tmpPath = join(projectRoot, ".claude", "tmp");
            try {
              const stat = await lstat(tmpPath);
              if (stat.isSymbolicLink()) { await unlink(tmpPath); }
              else { return; } // Real directory — skip
            } catch { /* doesn't exist */ }
            await symlink(sessionDirPath, tmpPath);
          } catch { /* best-effort */ }
        })();
      }
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
        void logger.debug("Permission mode changed", {
          permissionMode: statusEvent.permissionMode,
          planMode: statusEvent.permissionMode === "plan",
        });
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
        void logger.error("No session ID available for built-in tool", { blockName: block.name });
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
          void logger.warn("Unhandled built-in tool", { toolName: block.name });
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

          void logger.debug("Sent built-in tool result", {
            toolName: block.name,
            result: toolResult,
          });
        }
      } catch (err) {
        void logger.error("Built-in tool error", {
          toolName: block.name,
          error: err instanceof Error ? err.message : String(err),
        });

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
              void logger.error("Built-in tool error in background", {
                error: err instanceof Error ? err.message : String(err),
              });
            });
          }
        }
      }

      // Use message ID to distinguish streaming updates from new messages.
      // The SDK sends multiple assistant events with the SAME message.id for
      // streaming updates to one message, but a DIFFERENT message.id when
      // Claude starts a new response (e.g., after a tool call completes).
      const messageId = event.message.id;

      if (messageId === currentMessageIdRef.current) {
        // Same message ID - streaming update to current message
        currentMessageRef.current = contentBlocks;
        updateLastMessage(contentBlocks);
      } else {
        // Different message ID - new assistant message (first chunk or new turn after tool call)
        // Finalize previous message if any
        if (currentMessageRef.current.length > 0) {
          updateLastMessage(currentMessageRef.current);
        }
        currentMessageIdRef.current = messageId;
        currentMessageRef.current = contentBlocks;
        addMessage({
          role: "assistant",
          content: contentBlocks,
          partial: true,
        });
      }
    },
    [addMessage, updateLastMessage, handleBuiltinToolUse]
  );

  /**
   * Handle user event - tool results from CLI tool execution
   * These appear between assistant messages and provide the audit trail
   */
  const handleUserEvent = useCallback(
    (event: SDKUserMessage) => {
      // Finalize the current assistant message before adding tool results
      if (currentMessageRef.current.length > 0) {
        updateLastMessage(currentMessageRef.current);
        currentMessageRef.current = [];
        currentMessageIdRef.current = null;
      }

      // Convert user message content to ContentBlocks
      // SDK content can be string or ContentBlockParam[]
      const rawContent = event.message.content;
      const contentBlocks: ContentBlock[] = typeof rawContent === "string"
        ? [{ type: "text" as const, text: rawContent }]
        : rawContent.map(
          (block: any) => {
            if (block.type === "tool_result") {
              return {
                type: "tool_result" as const,
                tool_use_id: block.tool_use_id ?? "",
                content: typeof block.content === "string"
                  ? block.content
                  : Array.isArray(block.content)
                    ? block.content.map((c: any) => c.text || JSON.stringify(c)).join("\n")
                    : JSON.stringify(block.content ?? ""),
                is_error: block.is_error || false,
              };
            }
            if (block.type === "text") {
              return { type: "text" as const, text: block.text };
            }
            // Fallback
            return { type: "text" as const, text: JSON.stringify(block) };
          }
        );

      // Add as a system message (tool results aren't really "user" messages in the UI)
      addMessage({
        role: "system",
        content: contentBlocks,
        partial: false,
      });
    },
    [addMessage, updateLastMessage]
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
        currentMessageIdRef.current = null;
      }

      // Update usage statistics (available on both success and error)
      incrementCost(event.total_cost_usd);

      if (event.usage) {
        addTokens({
          input: event.usage.input_tokens,
          output: event.usage.output_tokens,
        });
      }

      // Extract context window usage from modelUsage
      if (event.modelUsage) {
        const models = Object.values(event.modelUsage);
        if (models.length > 0) {
          // Sum across all models (usually just one)
          const totalUsed = models.reduce(
            (sum, m) =>
              sum + m.inputTokens + m.cacheCreationInputTokens + m.cacheReadInputTokens,
            0
          );
          const capacity = models[0]?.contextWindow || 200000;
          updateContextWindow(totalUsed, capacity);
        }
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

      // Stop streaming - update both state and ref synchronously
      setIsStreaming(false);
      streamingRef.current = false;
      setStreamingState(false);

      // Clear interrupt function
      setInterruptQuery(null);

      // Notify that streaming is complete (for queue drain)
      onStreamingCompleteRef.current?.();
    },
    [
      updateLastMessage,
      incrementCost,
      addTokens,
      updateContextWindow,
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
      // Use streamingRef to avoid stale closure in concurrent calls
      if (streamingRef.current) {
        void logger.warn("Query already in progress, ignoring duplicate call");
        return;
      }

      try {
        // Reset error state
        setError(null);
        currentMessageRef.current = [];
        currentMessageIdRef.current = null;

        // Get preferred model if set (for pre-query model selection)
        const preferredModel = useStore.getState().preferredModel;

        void logger.debug("preferredModel from store", { preferredModel });

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

        // Start streaming - update both state and ref synchronously
        setIsStreaming(true);
        streamingRef.current = true;
        setStreamingState(true);

        // DEBUG: Log query options to file
        const queryModel = preferredModel || undefined;
        const existingSessionId = useStore.getState().sessionId;
        void logger.debug("Calling query()", {
          model: queryModel,
          resume: existingSessionId,
        });

        // Query Claude with MCP server registration and GOgent settings
        const eventStream = query({
          prompt: message,
          options: {
            // CRITICAL: Resume existing session to maintain conversation context
            // Without this, each message creates a new session and loses history
            resume: existingSessionId || undefined,
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

        // Register interrupt function in store
        setInterruptQuery(eventStream.interrupt.bind(eventStream));

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
          } else if (sdkMessage.type === "user") {
            handleUserEvent(sdkMessage as SDKUserMessage);
          } else if (sdkMessage.type === "result") {
            handleResultEvent(sdkMessage);
          }
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

        // Stop streaming - update both state and ref synchronously
        setIsStreaming(false);
        streamingRef.current = false;
        setStreamingState(false);

        // Clear interrupt function
        setInterruptQuery(null);

        // Notify that streaming is complete even on error (for queue drain)
        onStreamingCompleteRef.current?.();

        void logger.error("Query error", {
          error: err instanceof Error ? err.message : String(err),
        });
      }
    },
    [
      // Note: streamingRef used instead of isStreaming in callback body to avoid stale closures
      addMessage,
      handleSystemEvent,
      handleStatusEvent,
      handleAssistantEvent,
      handleUserEvent,
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
      void logger.warn("No active query for setModel", { modelId });
      return false;
    }

    try {
      await eventStreamRef.current.setModel(modelId);
      void logger.info("Successfully switched model", { modelId });
      return true;
    } catch (err) {
      void logger.error("setModel failed", {
        modelId,
        error: err instanceof Error ? err.message : String(err),
      });
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
