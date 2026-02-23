/**
 * SessionManager - Core singleton managing async message flow between user and Claude SDK
 *
 * Implements state machine for session lifecycle and coordinates message queueing with
 * AsyncGenerator pattern to keep CLI process alive between messages.
 *
 * State transitions:
 * UNINITIALIZED -> CONNECTING -> READY <-> STREAMING
 *                      ↓              ↓
 *                   ERROR ---------> DEAD
 *
 * Based on test-async-iterable.ts coordination pattern.
 */

import { query, type SDKUserMessage, type SDKMessage, type SDKSystemMessage, type SDKAssistantMessage, type SDKResultMessage } from "@anthropic-ai/claude-agent-sdk";
import type { MessageParam, ContentBlockParam } from "@anthropic-ai/sdk/resources/messages";
import { nanoid } from "nanoid";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { homedir } from "os";
import {
  SessionState,
  type SessionManagerConfig,
  type QueuedMessage,
  type MessageCoordinator,
  type SessionManagerEvents,
} from "./types.js";
import { useStore } from "../store/index.js";
import { mcpServer } from "../mcp/server.js";
import { type ContentBlock, type ProviderId } from "../store/types.js";
import { logger } from "../utils/logger.js";
import { type ClassifiedError, type ErrorType } from "../types/events.js";
import { onShutdown } from "../lifecycle/shutdown.js";
import { PROVIDERS } from "../config/providers.js";

/**
 * SDK tools handled internally by CLI subprocess.
 * These are processed via canUseTool (permission) — NOT streamInput.
 */
const SDK_INTERNAL_TOOLS = [
  'EnterPlanMode',
  'ExitPlanMode',
] as const;

/**
 * Default configuration values
 */
const DEFAULT_CONFIG: SessionManagerConfig = {
  maxQueueSize: 10,
  reconnectDelayMs: 1000,
  maxReconnectAttempts: 3,
};

/**
 * Valid state transitions map
 * Each state lists the states it can transition to
 */
const VALID_TRANSITIONS: Record<SessionState, SessionState[]> = {
  [SessionState.UNINITIALIZED]: [SessionState.CONNECTING],
  [SessionState.CONNECTING]: [SessionState.READY, SessionState.ERROR],
  [SessionState.READY]: [SessionState.STREAMING, SessionState.ERROR, SessionState.DEAD],
  [SessionState.STREAMING]: [SessionState.READY, SessionState.ERROR, SessionState.DEAD],
  [SessionState.ERROR]: [SessionState.CONNECTING, SessionState.DEAD],
  [SessionState.DEAD]: [], // Terminal state - no transitions allowed
};

/**
 * SessionManager class - manages async message flow between user input and Claude SDK
 *
 * This is a singleton class. Use getSessionManager() to access the instance.
 *
 * @example
 * ```ts
 * const manager = getSessionManager();
 * await manager.connect();
 * await manager.enqueue("Hello Claude");
 * ```
 */
class SessionManager {
  /** Current session state */
  private state: SessionState = SessionState.UNINITIALIZED;

  /** Active session ID from SDK */
  private sessionId: string | null = null;

  /** Bounded FIFO message queue */
  private messageQueue: QueuedMessage[] = [];

  /** Current message being processed */
  private activeCoordinator: MessageCoordinator | null = null;

  /** Active query event stream with interrupt/setModel methods */
  private eventStream: Awaited<ReturnType<typeof query>> | null = null;

  /** Number of reconnection attempts made */
  private reconnectAttempts: number = 0;

  /** Configuration */
  private readonly config: SessionManagerConfig;

  /** Event callbacks for state synchronization */
  private events: SessionManagerEvents | null = null;

  /** Promise resolvers for init event */
  private initResolve: (() => void) | null = null;
  private initReject: ((error: Error) => void) | null = null;

  /** Track current assistant message being built */
  private currentMessageRef: ContentBlock[] = [];

  /** Track current assistant message ID to distinguish streaming updates from new messages */
  private currentMessageIdRef: string | null = null;

  /** Promise resolver for queue notification - replaces polling loop */
  private queueNotifier: (() => void) | null = null;

  /**
   * Private constructor - use getSessionManager() instead
   * @param config - Optional configuration overrides
   */
  constructor(config?: Partial<SessionManagerConfig>) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Register event callbacks for state synchronization
   * @param events - Event callback handlers
   */
  setEvents(events: SessionManagerEvents): void {
    this.events = events;
  }

  /**
   * Resolve adapter configuration for a provider and model
   * @param provider - Provider ID
   * @param model - Model identifier
   * @returns Adapter configuration with executable path and env vars
   */
  private resolveAdapter(provider: ProviderId, model: string): {
    executable?: string;
    env?: Record<string, string>;
  } {
    const providerConfig = PROVIDERS[provider];

    // Anthropic uses native SDK - no adapter needed
    if (provider === "anthropic") {
      return {};
    }

    // Non-Anthropic providers need adapters
    return {
      executable: providerConfig.adapterPath,
      env: {
        ...process.env,
        ...(providerConfig.envVars || {}),
        MODEL: model, // Generic env var for all adapters
      },
    };
  }

  /**
   * Transition to a new state with validation
   * @param newState - Target state
   * @throws Error if transition is invalid
   */
  private transitionTo(newState: SessionState): void {
    const validTargets = VALID_TRANSITIONS[this.state];

    if (!validTargets.includes(newState)) {
      throw new Error(
        `Invalid state transition: ${this.state} -> ${newState}`
      );
    }

    const oldState = this.state;
    this.state = newState;

    // Notify listeners
    this.events?.onStateChange(newState);

    // Debug logging (would use logger in production)
    if (typeof process !== "undefined" && process.env["DEBUG_SESSION"]) {
      console.error(`[SessionManager] ${oldState} -> ${newState}`);
    }
  }

  /**
   * Create async generator that yields messages from queue
   *
   * This generator keeps the CLI process alive between messages by:
   * 1. Yielding user message to SDK
   * 2. Awaiting response via MessageCoordinator.responsePromise
   * 3. Repeating until queue is empty
   *
   * Based on createMessageGenerator pattern from test-async-iterable.ts
   */
  private async *createMessageGenerator(): AsyncGenerator<
    SDKUserMessage,
    void,
    undefined
  > {
    let isFirstMessage = true;

    while (this.state !== SessionState.DEAD) {
      // Wait for a message in queue.
      // For subsequent messages, also wait for READY state.
      // The FIRST message must yield immediately from CONNECTING —
      // the SDK does not start the CLI (or emit system.init) until
      // the generator yields its first message.
      while (
        this.messageQueue.length === 0 ||
        (!isFirstMessage && this.state === SessionState.CONNECTING)
      ) {
        await new Promise<void>((resolve) => {
          this.queueNotifier = resolve;
        });

        // Exit if session died while waiting
        if ((this.state as SessionState) === SessionState.DEAD) {
          return;
        }
      }

      // Get next message (should exist due to await above)
      const queuedMessage = this.messageQueue.shift();
      if (queuedMessage) {
        // Clear notifier only after successfully getting a message
        this.queueNotifier = null;
      }
      if (!queuedMessage) {
        continue; // Race condition - queue was drained
      }

      // Create coordinator for this message
      const coordinator: MessageCoordinator = {
        text: queuedMessage.text,
        yieldedAt: 0,
        responsePromise: Promise.resolve(), // Placeholder
        resolveResponse: () => {},
        rejectResponse: () => {},
        queuedMessage, // Store reference to resolve the queued promise
      };

      // Initialize promise
      coordinator.responsePromise = new Promise<void>((resolve, reject) => {
        coordinator.resolveResponse = resolve;
        coordinator.rejectResponse = reject;
      });

      // Set as active coordinator
      this.activeCoordinator = coordinator;

      // Only transition to STREAMING for non-first messages.
      // The first message yields during CONNECTING state to bootstrap the CLI.
      // system.init will transition CONNECTING→READY while the first response streams.
      if (!isFirstMessage) {
        this.transitionTo(SessionState.STREAMING);
      }
      isFirstMessage = false;

      // Create SDK message structure
      const userMessage: SDKUserMessage = {
        type: "user" as const,
        message: {
          role: "user" as const,
          content: [
            {
              type: "text" as const,
              text: coordinator.text,
            },
          ],
        } as MessageParam,
        parent_tool_use_id: null,
        session_id: this.sessionId || "",
      };

      // Record yield timestamp for latency tracking
      coordinator.yieldedAt = performance.now();

      // Yield to SDK
      yield userMessage;

      // Wait for response before yielding next message
      try {
        await coordinator.responsePromise;

        // Resolve the queued message promise
        queuedMessage.resolve(true);

        // Transition back to READY only if we're in STREAMING.
        // After the first message, state is already READY (from system.init).
        if (this.state === SessionState.STREAMING) {
          this.transitionTo(SessionState.READY);
        }
      } catch (error) {
        // Reject the queued message promise
        queuedMessage.reject(
          error instanceof Error ? error : new Error(String(error))
        );

        // Transition to ERROR (valid from both CONNECTING and STREAMING)
        this.transitionTo(SessionState.ERROR);

        // Notify error listeners
        this.events?.onError({
          type: "server_error",
          message:
            error instanceof Error ? error.message : "Unknown error",
          retryable: true,
          originalError: error,
        });
      } finally {
        // Clear active coordinator
        this.activeCoordinator = null;
      }
    }
  }

  /**
   * Process the message queue
   * Called after enqueue() to start processing if idle
   *
   * This method is a no-op if:
   * - State is not READY (already processing or not connected)
   * - Queue is empty
   * - Active coordinator exists (message in flight)
   */
  private processQueue(): void {
    // Only process if READY and queue has messages
    if (this.state !== SessionState.READY || this.messageQueue.length === 0) {
      return;
    }

    // Only process if no active coordinator
    if (this.activeCoordinator !== null) {
      return;
    }

    // The generator will automatically pick up the next message
    // when it loops back to check the queue
  }

  /**
   * Connect to Claude SDK and initialize session
   *
   * @throws Error if already connected or in invalid state
   */
  async connect(): Promise<void> {
    // Guard: only connect from UNINITIALIZED or ERROR
    if (this.state !== SessionState.UNINITIALIZED && this.state !== SessionState.ERROR) {
      throw new Error(`Cannot connect from state ${this.state}`);
    }

    this.transitionTo(SessionState.CONNECTING);

    // Get existing session ID for resume (from Zustand store)
    const store = useStore.getState();
    const activeProvider = store.activeProvider;
    const existingSessionId = store.providerSessionIds[activeProvider];
    const preferredModel = store.providerModels[activeProvider];

    // Create Promise that resolves when system.init event received
    const initPromise = new Promise<void>((resolve, reject) => {
      this.initResolve = resolve;
      this.initReject = reject;
    });

    try {
      // Resolve adapter for provider
      const { executable, env } = this.resolveAdapter(activeProvider, preferredModel || "");

      if (executable) {
        void logger.info("Using provider adapter", {
          provider: activeProvider,
          adapterPath: executable,
          model: preferredModel,
        });
      }

      // Start query with AsyncIterable prompt
      this.eventStream = query({
        prompt: this.createMessageGenerator(),
        options: {
          // Point to custom executable if needed (e.g. provider adapter)
          pathToClaudeCodeExecutable: executable,

          // Pass adapter env vars
          env,

          // Resume existing session to maintain conversation context
          resume: existingSessionId || undefined,
          // Apply preferred model if set. If no preference, omit to let
          // SDK use the model from user's Claude Code settings.
          model: preferredModel || undefined,
          // Load GOgent-Fortress settings (hooks, CLAUDE.md, etc.)
          settingSources: ["user", "project", "local"],
          // Register MCP server - SDK expects Record<string, McpServerConfig>
          mcpServers: {
            [mcpServer.name]: mcpServer,
          },
          // Permission callback - prompts user before tool execution
          canUseTool: this.handleCanUseTool.bind(this),
          // Enable partial message streaming for better UX
          includePartialMessages: true,
        },
      });

      // Start consuming events in background (don't await)
      this.consumeEvents().catch((error) => {
        this.handleConnectionError(error);
      });

      // Wait for init event
      await initPromise;

      this.reconnectAttempts = 0;
    } catch (error) {
      this.handleConnectionError(error);
      throw error;
    }
  }

  /**
   * Handle connection error - transition to ERROR and notify listeners
   */
  private handleConnectionError(error: unknown): void {
    this.transitionTo(SessionState.ERROR);
    this.events?.onError(this.classifyError(error));
    void this.attemptReconnect();
  }

  /**
   * Attempt reconnection after error with backoff
   */
  private async attemptReconnect(): Promise<void> {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.transitionTo(SessionState.DEAD);
      return;
    }

    this.reconnectAttempts++;

    // Wait before attempting
    await new Promise((resolve) =>
      setTimeout(resolve, this.config.reconnectDelayMs)
    );

    try {
      await this.connect();
    } catch {
      // Error already handled by connect()
    }
  }

  /**
   * Classify error for better error handling
   */
  private classifyError(error: unknown): ClassifiedError {
    const errorMessage =
      error instanceof Error ? error.message : String(error);

    // Network errors
    if (
      errorMessage.includes("ECONNREFUSED") ||
      errorMessage.includes("ENOTFOUND") ||
      errorMessage.includes("network")
    ) {
      return {
        type: "network" as ErrorType,
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
        type: "auth" as ErrorType,
        message: "Authentication failed. Check your API key.",
        originalError: error,
        retryable: false,
      };
    }

    // Rate limiting
    if (errorMessage.includes("429") || errorMessage.includes("rate limit")) {
      return {
        type: "rate_limit" as ErrorType,
        message: "Rate limit exceeded. Please wait before retrying.",
        originalError: error,
        retryable: true,
      };
    }

    // Server errors
    if (errorMessage.includes("500") || errorMessage.includes("server error")) {
      return {
        type: "server_error" as ErrorType,
        message: "Server error. Please try again later.",
        originalError: error,
        retryable: true,
      };
    }

    // Invalid request
    if (errorMessage.includes("400") || errorMessage.includes("invalid")) {
      return {
        type: "invalid_request" as ErrorType,
        message: "Invalid request. Check your input.",
        originalError: error,
        retryable: false,
      };
    }

    // Timeout
    if (errorMessage.includes("timeout") || errorMessage.includes("ETIMEDOUT")) {
      return {
        type: "timeout" as ErrorType,
        message: "Request timed out. Please try again.",
        originalError: error,
        retryable: true,
      };
    }

    // Unknown error
    return {
      type: "unknown" as ErrorType,
      message: errorMessage || "An unknown error occurred",
      originalError: error,
      retryable: false,
    };
  }

  /**
   * Consume events from query stream
   */
  private async consumeEvents(): Promise<void> {
    if (!this.eventStream) return;

    const store = useStore.getState();

    try {
      for await (const event of this.eventStream) {
        const sdkMessage = event as SDKMessage;

        switch (sdkMessage.type) {
          case "system":
            if (sdkMessage.subtype === "init") {
              this.handleSystemEvent(sdkMessage);
            } else if (sdkMessage.subtype === "status") {
              this.handleStatusEvent(sdkMessage);
            } else if (sdkMessage.subtype === "compact_boundary") {
              this.handleCompactBoundaryEvent(sdkMessage);
            }
            break;

          case "assistant":
            await this.handleAssistantEvent(sdkMessage);
            break;

          case "user":
            this.handleUserEvent(sdkMessage as SDKUserMessage);
            break;

          case "result":
            this.handleResultEvent(sdkMessage);
            break;
        }
      }

      // Health check: if iterator completes without error and state is not DEAD,
      // the SDK subprocess crashed silently (SIGTERM, OOM, etc.)
      if (this.state !== SessionState.DEAD) {
        void logger.error("SDK subprocess iterator completed unexpectedly", {
          state: this.state,
          activeCoordinator: !!this.activeCoordinator,
        });

        // Reject orphaned message if exists
        if (this.activeCoordinator) {
          this.activeCoordinator.rejectResponse(
            new Error("SDK subprocess terminated unexpectedly")
          );
          this.activeCoordinator.queuedMessage?.reject(
            new Error("SDK subprocess terminated unexpectedly")
          );
          this.activeCoordinator = null;
        }

        // Transition to ERROR and attempt reconnect
        this.transitionTo(SessionState.ERROR);
        this.events?.onError({
          type: "server_error",
          message: "SDK subprocess terminated unexpectedly",
          retryable: true,
        });
        this.events?.onStreamingComplete();
        void this.attemptReconnect();
      }
    } catch (error) {
      this.transitionTo(SessionState.ERROR);
      this.events?.onError(this.classifyError(error));
      this.events?.onStreamingComplete();
      void this.attemptReconnect();
    }
  }

  /**
   * Handle system event - initialize session metadata
   */
  private handleSystemEvent(event: SDKSystemMessage): void {
    // Guard: only process init during CONNECTING state.
    // SDK emits a duplicate system.init for every generator yield —
    // subsequent inits must not corrupt the STREAMING state.
    if (this.state !== SessionState.CONNECTING) {
      this.sessionId = event.session_id;
      this.events?.onSessionId(event.session_id);
      return;
    }

    // Extract model from init message
    const initEvent = event as SDKSystemMessage & { model?: string };
    const store = useStore.getState();
    const activeProvider = store.activeProvider;

    if (initEvent.model) {
      void logger.debug("SDK Init returned model", { model: initEvent.model });
      store.setProviderModel(activeProvider, initEvent.model);
    }

    // Update session with ID and set session dir for child processes
    this.sessionId = event.session_id;
    this.events?.onSessionId(event.session_id);
    store.setProviderSessionId(activeProvider, event.session_id);
    store.updateSession({ id: event.session_id });

    // Eagerly register the root "Router" agent so the agent panel shows
    // immediately on session start, before the first Task() delegation.
    if (!store.rootAgentId) {
      const realModel = initEvent.model ?? "claude-sonnet-4-5";
      const tier = realModel.includes("haiku")
        ? "haiku"
        : realModel.includes("sonnet")
          ? "sonnet"
          : "opus";
      store.addAgent({
        id: "router-root",
        parentId: null,
        model: realModel,
        tier,
        status: "running",
        description: "Router",
        agentType: "router",
        spawnMethod: "task",
      });
    }

    // Set GOGENT_SESSION_DIR for team polling and child processes
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
            if (stat.isSymbolicLink()) {
              await unlink(tmpPath);
            } else {
              return; // Real directory — skip
            }
          } catch {
            /* doesn't exist */
          }
          await symlink(sessionDirPath, tmpPath);
        } catch {
          /* best-effort */
        }
      })();
    }

    // Transition to READY and resolve init promise
    this.transitionTo(SessionState.READY);
    this.initResolve?.();

    // Notify generator that READY state is reached (unblocks message processing)
    if (this.queueNotifier) {
      this.queueNotifier();
      this.queueNotifier = null;
    }
  }

  /**
   * Handle status event - track permission mode and compacting state
   */
  private handleStatusEvent(event: SDKMessage): void {
    const statusEvent = event as {
      type: "system";
      subtype: "status";
      status?: "compacting" | null;
      permissionMode?: string;
    };

    // Update compacting state if present
    if (statusEvent.status !== undefined) {
      useStore.getState().setCompacting(statusEvent.status === "compacting");
    }

    // Update permission mode if present
    if (statusEvent.permissionMode) {
      useStore.getState().setPermissionMode(statusEvent.permissionMode);

      void logger.debug("Permission mode changed", {
        permissionMode: statusEvent.permissionMode,
        planMode: statusEvent.permissionMode === "plan",
      });
    }
  }

  /**
   * Handle compact_boundary event - context was compacted by CLI
   */
  private handleCompactBoundaryEvent(event: SDKMessage): void {
    const compactEvent = event as {
      type: "system";
      subtype: "compact_boundary";
      compact_metadata: {
        trigger: "manual" | "auto";
        pre_tokens: number;
      };
    };

    const { trigger, pre_tokens } = compactEvent.compact_metadata;
    const preTokensK = Math.round(pre_tokens / 1000);

    void logger.info("Context compacted", { trigger, pre_tokens });

    // Show toast notification
    const store = useStore.getState();
    store.addToast(`Context compacted (was ${preTokensK}K tokens)`, "info");

    // Update compacting state
    store.setCompacting(false);
  }

  /**
   * Handle assistant event - add or update message
   */
  private async handleAssistantEvent(event: SDKAssistantMessage): Promise<void> {
    const store = useStore.getState();

    // Convert BetaMessage content to ContentBlock format
    const contentBlocks: ContentBlock[] = event.message.content.map((block) => {
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
      // Fallback to text block
      return {
        type: "text" as const,
        text: "",
      };
    });

    // Log SDK internal tools for diagnostics (no action needed — CLI handles them)
    const internalTools = event.message.content.filter(
      (block) => block.type === 'tool_use' && SDK_INTERNAL_TOOLS.includes(block.name as (typeof SDK_INTERNAL_TOOLS)[number])
    );
    if (internalTools.length > 0) {
      for (const block of internalTools) {
        if (block.type === 'tool_use') {
          void logger.debug("[SDK Internal Tool]", {
            name: block.name,
            input: JSON.stringify(block.input).slice(0, 200),
          });
        }
      }
    }

    // Update context window usage from per-message token counts
    // BetaMessage.usage reflects the actual context fill for this API call
    if (event.message.usage) {
      const usage = event.message.usage;
      const usedTokens =
        usage.input_tokens +
        (usage.cache_creation_input_tokens ?? 0) +
        (usage.cache_read_input_tokens ?? 0);
      const { contextWindow } = store;
      store.updateContextWindow(usedTokens, contextWindow.totalCapacity);
    }

    // Use message ID to distinguish streaming updates from new messages
    const messageId = event.message.id;

    // Get active provider for message storage
    const activeProvider = store.activeProvider;

    // Extract sub-agent tag (null and undefined both mean root message)
    const subagentToolUseId = event.parent_tool_use_id || undefined;

    if (messageId === this.currentMessageIdRef) {
      // Same message ID - streaming update to current message
      this.currentMessageRef = contentBlocks;
      store.updateLastProviderMessage(activeProvider, contentBlocks);
    } else {
      // Different message ID - new assistant message
      if (this.currentMessageRef.length > 0) {
        store.updateLastProviderMessage(activeProvider, this.currentMessageRef);
      }
      this.currentMessageIdRef = messageId;
      this.currentMessageRef = contentBlocks;
      store.addProviderMessage(activeProvider, {
        role: "assistant",
        content: contentBlocks,
        partial: true,
        subagentToolUseId,
      });
    }
  }

  /**
   * Handle user event - tool results from CLI tool execution
   */
  private handleUserEvent(event: SDKUserMessage): void {
    const store = useStore.getState();
    const activeProvider = store.activeProvider;

    // Finalize the current assistant message before adding tool results
    if (this.currentMessageRef.length > 0) {
      store.updateLastProviderMessage(activeProvider, this.currentMessageRef);
      this.currentMessageRef = [];
      this.currentMessageIdRef = null;
    }

    // Convert user message content to ContentBlocks
    const rawContent = event.message.content;
    const contentBlocks: ContentBlock[] =
      typeof rawContent === "string"
        ? [{ type: "text" as const, text: rawContent }]
        : rawContent.map((block: ContentBlockParam) => {
            if (block.type === "tool_result") {
              return {
                type: "tool_result" as const,
                tool_use_id: block.tool_use_id ?? "",
                content:
                  typeof block.content === "string"
                    ? block.content
                    : Array.isArray(block.content)
                      ? block.content
                          .map((c: ContentBlockParam) =>
                            c.type === "text" ? c.text : JSON.stringify(c)
                          )
                          .join("\n")
                      : JSON.stringify(block.content ?? ""),
                is_error: block.is_error || false,
              };
            }
            if (block.type === "text") {
              return { type: "text" as const, text: block.text };
            }
            // Fallback
            return { type: "text" as const, text: JSON.stringify(block) };
          });

    // Extract sub-agent tag (null and undefined both mean root message)
    const subagentToolUseId = event.parent_tool_use_id || undefined;

    // Add as a system message
    store.addProviderMessage(activeProvider, {
      role: "system",
      content: contentBlocks,
      partial: false,
      subagentToolUseId,
    });
  }

  /**
   * Handle result event - finalize message and update stats
   */
  private handleResultEvent(event: SDKResultMessage): void {
    const store = useStore.getState();
    const activeProvider = store.activeProvider;

    // Mark message as complete if we have one
    if (this.currentMessageRef.length > 0) {
      store.updateLastProviderMessage(activeProvider, this.currentMessageRef);
      this.currentMessageRef = [];
      this.currentMessageIdRef = null;
    }

    // Update usage statistics
    store.incrementCost(event.total_cost_usd);

    if (event.usage) {
      store.addTokens({
        input: event.usage.input_tokens,
        output: event.usage.output_tokens,
      });
    }

    // Update context window capacity from modelUsage (token counts are
    // cumulative here, so we only extract capacity — actual usage is
    // tracked per-message in handleAssistantEvent)
    if (event.modelUsage) {
      const models = Object.values(event.modelUsage);
      if (models.length > 0) {
        const capacity = models[0]?.contextWindow || 200000;
        const { contextWindow } = store;
        store.updateContextWindow(contextWindow.usedTokens, capacity);
      }
    }

    // Handle error result
    if (event.subtype !== "success") {
      const eventWithErrors = event as SDKResultMessage & { errors?: string[] };
      const errorMessages = eventWithErrors.errors
        ? eventWithErrors.errors.join("; ")
        : "Query failed";
      this.events?.onError({
        type: "server_error" as ErrorType,
        message: errorMessages,
        retryable: true,
      });
    }

    // Notify streaming complete (for queue drain)
    this.events?.onStreamingComplete();

    // Stop streaming
    store.setStreaming(false);
    store.setInterruptQuery(null);

    // Resolve active coordinator response promise
    if (this.activeCoordinator) {
      this.activeCoordinator.resolveResponse();
    }

    // NOTE: Do NOT transition to READY here. The generator owns the READY transition
    // via its await coordinator.responsePromise → transitionTo(READY) flow.
    // Transitioning here would cause a READY→READY invalid transition error.
  }

  /**
   * Handle canUseTool callback for permission requests
   */
  private async handleCanUseTool(
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
  ): Promise<
    | { behavior: "allow"; updatedInput?: Record<string, unknown>; toolUseID?: string }
    | { behavior: "deny"; message: string; toolUseID?: string }
  > {
    // --- acceptEdits mode: auto-approve execution tools ---
    const currentMode = useStore.getState().permissionMode;
    if (currentMode === 'acceptEdits') {
      // Interactive tools always require manual approval
      const interactiveTools = new Set([
        'AskUserQuestion',
        'EnterPlanMode',
        'ExitPlanMode',
      ]);

      if (!interactiveTools.has(toolName)) {
        void logger.debug("[canUseTool] Auto-approved in acceptEdits mode", { toolName });
        return {
          behavior: 'allow' as const,
          updatedInput: input,
          toolUseID: options.toolUseID,
        };
      }
      // Interactive tools fall through to normal modal flow below
    }

    // --- Plan mode tools: custom approval UI ---
    if (toolName === 'EnterPlanMode') {
      void logger.debug("[canUseTool] EnterPlanMode intercepted", { input });

      let enterPlanResult;
      try {
        enterPlanResult = await useStore.getState().enqueue({
          type: 'confirm',
          payload: {
            action: 'Claude wants to enter plan mode.\n\nIn plan mode, Claude will explore and design an approach before making changes.\nYou will review the plan before any implementation begins.',
          },
          timeout: 60000,
        });
      } catch {
        return { behavior: 'deny' as const, message: 'User cancelled plan mode prompt', toolUseID: options.toolUseID };
      }

      if (enterPlanResult.type === 'confirm' && enterPlanResult.confirmed) {
        return { behavior: 'allow' as const, updatedInput: input, toolUseID: options.toolUseID };
      }
      return { behavior: 'deny' as const, message: 'User declined plan mode', toolUseID: options.toolUseID };
    }

    if (toolName === 'ExitPlanMode') {
      void logger.debug("[canUseTool] ExitPlanMode intercepted", { input });

      const allowedPrompts = input['allowedPrompts'] as Array<{ tool: string; prompt: string }> | undefined;
      const promptSummary = allowedPrompts?.length
        ? '\n\nPermissions requested:\n' + allowedPrompts.map(p => `  - ${p.tool}: ${p.prompt}`).join('\n')
        : '';

      // Rich ask modal: Approve / Request changes / Reject (+ auto "Other" for free text)
      let exitPlanResult;
      try {
        exitPlanResult = await useStore.getState().enqueue({
          type: 'ask',
          payload: {
            message: `Approve plan and begin implementation?${promptSummary}`,
            header: 'Plan',
            options: [
              { label: 'Approve', value: 'approve', description: 'Begin implementation as planned' },
              { label: 'Request changes', value: 'changes', description: 'Send feedback — Claude will revise the plan' },
              { label: 'Reject', value: 'reject', description: 'Block with a reason' },
            ],
          },
          timeout: 120000,
        });
      } catch {
        return { behavior: 'deny' as const, message: 'User cancelled plan approval', toolUseID: options.toolUseID };
      }

      if (exitPlanResult.type === 'ask') {
        if (exitPlanResult.value === 'Approve') {
          return { behavior: 'allow' as const, updatedInput: input, toolUseID: options.toolUseID };
        }

        if (exitPlanResult.value === 'Request changes' || exitPlanResult.value === 'Reject') {
          // Follow-up: collect feedback/reason via input modal
          const promptText = exitPlanResult.value === 'Request changes'
            ? 'What changes would you like?'
            : 'Why are you rejecting the plan?';

          let feedbackResult;
          try {
            feedbackResult = await useStore.getState().enqueue({
              type: 'input',
              payload: {
                prompt: promptText,
                placeholder: 'Type your feedback...',
              },
              timeout: 120000,
            });
          } catch {
            return { behavior: 'deny' as const, message: 'User cancelled feedback', toolUseID: options.toolUseID };
          }

          if (feedbackResult.type === 'input' && feedbackResult.value.trim()) {
            return { behavior: 'deny' as const, message: feedbackResult.value, toolUseID: options.toolUseID };
          }
          return { behavior: 'deny' as const, message: 'User rejected the plan', toolUseID: options.toolUseID };
        }

        // "Other" (free text from AskModal) — treat as feedback to Claude
        if (exitPlanResult.value && exitPlanResult.value !== 'Other') {
          return { behavior: 'deny' as const, message: exitPlanResult.value, toolUseID: options.toolUseID };
        }
      }

      return { behavior: 'deny' as const, message: 'User rejected the plan', toolUseID: options.toolUseID };
    }

    // --- AskUserQuestion: collect user answers ---
    if (toolName === 'AskUserQuestion') {
      const questions = input['questions'] as Array<{
        question: string;
        header?: string;
        options?: Array<{ label: string; description?: string }>;
        multiSelect?: boolean;
      }> | undefined;

      if (!questions || questions.length === 0) {
        return { behavior: 'deny' as const, message: 'No questions provided', toolUseID: options.toolUseID };
      }

      // Process each question sequentially (1-4 questions per SDK contract)
      const answers: Record<string, string> = {};

      for (const q of questions) {
        const modalOptions = q.options?.map(o => ({
          label: o.label,
          value: o.label,  // SDK contract: answers use labels, not custom values
          description: o.description,
        })) || [];

        let modalResult;
        try {
          modalResult = await useStore.getState().enqueue({
            type: 'ask',
            payload: {
              message: q.question,
              header: q.header,
              options: modalOptions,
              multiSelect: q.multiSelect,
            },
            timeout: 120000,
          });
        } catch {
          // Modal was cancelled (Escape) or timed out
          return { behavior: 'deny' as const, message: 'User cancelled question', toolUseID: options.toolUseID };
        }

        if (modalResult.type === 'ask') {
          answers[q.question] = modalResult.value;
        } else {
          // Unexpected result type — deny
          return { behavior: 'deny' as const, message: 'User cancelled question', toolUseID: options.toolUseID };
        }
      }

      // Return SDK-expected format: pass back questions + answers via updatedInput
      return {
        behavior: 'allow' as const,
        updatedInput: {
          questions,
          answers,
        },
        toolUseID: options.toolUseID,
      };
    }

    // --- Standard tool permission flow ---
    // Create input preview (truncate if too long)
    const inputPreview = JSON.stringify(input, null, 2);
    const truncatedPreview =
      inputPreview.length > 200
        ? inputPreview.slice(0, 200) + "..."
        : inputPreview;

    // Build action description with agent ID
    const actor = options.agentID ? `[${options.agentID}]` : 'Claude';
    const actionDescription = options.blockedPath
      ? `Allow ${actor} to access "${options.blockedPath}" via ${toolName}?`
      : `Allow ${actor} to use ${toolName}?`;

    // Add decision reason if present
    const reasonText = options.decisionReason
      ? `\nReason: ${options.decisionReason}`
      : '';

    // Rich ask modal: Allow / Deny (with reason) / Other (free text)
    // Allow is first option → Enter key approves immediately
    let result;
    try {
      result = await useStore.getState().enqueue({
        type: "ask",
        payload: {
          message: `${actionDescription}${reasonText}\n\nInput:\n${truncatedPreview}`,
          header: toolName.slice(0, 12),
          options: [
            { label: 'Allow', value: 'allow', description: 'Execute as requested' },
            { label: 'Deny', value: 'deny', description: 'Block and tell Claude why' },
          ],
        },
        timeout: 60000,
      });
    } catch {
      // Modal was cancelled (Escape) or timed out — treat as denial
      return {
        behavior: "deny" as const,
        message: "User cancelled permission prompt",
        toolUseID: options.toolUseID,
      };
    }

    // Handle abort signal
    if (options.signal.aborted) {
      return {
        behavior: "deny" as const,
        message: "Request aborted",
        toolUseID: options.toolUseID,
      };
    }

    // Process response
    if (result.type === "ask") {
      if (result.value === 'Allow') {
        return {
          behavior: "allow" as const,
          updatedInput: input,
          toolUseID: options.toolUseID,
        };
      }

      if (result.value === 'Deny') {
        // Follow-up: collect reason via input modal
        let reasonResult;
        try {
          reasonResult = await useStore.getState().enqueue({
            type: 'input',
            payload: {
              prompt: `Why are you denying ${toolName}?`,
              placeholder: 'Enter reason (or leave empty)...',
            },
            timeout: 60000,
          });
        } catch {
          return { behavior: "deny" as const, message: "User denied permission", toolUseID: options.toolUseID };
        }

        const reason = reasonResult.type === 'input' && reasonResult.value.trim()
          ? reasonResult.value
          : 'User denied permission';
        return { behavior: "deny" as const, message: reason, toolUseID: options.toolUseID };
      }

      // "Other" (free text from AskModal) — treat as deny with the typed message
      if (result.value && result.value !== 'Other') {
        return { behavior: "deny" as const, message: result.value, toolUseID: options.toolUseID };
      }
    }

    return {
      behavior: "deny" as const,
      message: "User denied permission",
      toolUseID: options.toolUseID,
    };
  }

  /**
   * Enqueue a message for delivery to Claude
   *
   * @param message - Message text to send
   * @returns Promise that resolves when message is delivered (true) or rejected (false)
   */
  async enqueue(message: string): Promise<boolean> {
    // If state is DEAD: return false
    if (this.state === SessionState.DEAD) {
      return false;
    }

    // Check queue capacity
    if (this.messageQueue.length >= this.config.maxQueueSize) {
      return false;
    }

    // Create queued message with promise
    // IMPORTANT: Push to queue BEFORE connect() to break the deadlock.
    // The generator needs a message in the queue when the SDK starts,
    // otherwise it blocks waiting for a message while connect() blocks
    // waiting for system.init — circular dependency.
    const messagePromise = new Promise<boolean>((resolve, reject) => {
      const queuedMessage: QueuedMessage = {
        id: nanoid(),
        text: message,
        enqueuedAt: Date.now(),
        resolve,
        reject,
      };

      // Add to queue FIRST
      this.messageQueue.push(queuedMessage);

      // Wake up the generator if it's waiting
      if (this.queueNotifier) {
        this.queueNotifier();
        this.queueNotifier = null;
      }
    });

    // Auto-connect if needed (generator will find the message but wait for READY)
    if (this.state === SessionState.ERROR) {
      try {
        await this.connect();
      } catch {
        // Remove failed message from queue
        const failed = this.messageQueue.shift();
        failed?.reject(new Error("Connection failed"));
        return false;
      }
    }

    if (this.state === SessionState.UNINITIALIZED) {
      try {
        await this.connect();
      } catch {
        // Remove failed message from queue
        const failed = this.messageQueue.shift();
        failed?.reject(new Error("Connection failed"));
        return false;
      }
    }

    return messagePromise;
  }

  /**
   * Interrupt the current streaming response
   */
  async interrupt(): Promise<void> {
    // If eventStream exists and has interrupt method
    if (this.eventStream && typeof this.eventStream.interrupt === "function") {
      try {
        await this.eventStream.interrupt();
      } catch (error) {
        // Log but don't throw - interrupt is best-effort
        console.warn("Interrupt failed:", error);
      }
    }

    // Reject active coordinator if exists
    if (this.activeCoordinator) {
      this.activeCoordinator.rejectResponse(new Error("Interrupted by user"));
      this.activeCoordinator.queuedMessage?.resolve(false);
      this.activeCoordinator = null;
    }

    // Transition back to READY if was STREAMING
    if (this.state === SessionState.STREAMING) {
      this.transitionTo(SessionState.READY);
    }
  }

  /**
   * Switch the model for the active query
   *
   * @param modelId - Model identifier (e.g., "haiku", "sonnet", "opus")
   * @returns true on success, false if no active stream
   */
  async setModel(modelId: string): Promise<boolean> {
    if (!this.eventStream) {
      return false;
    }

    try {
      await this.eventStream.setModel(modelId);
      return true;
    } catch (error) {
      // Log error but don't transition to ERROR state
      return false;
    }
  }

  /**
   * Change the permission mode for the active session
   * Available modes: default, acceptEdits, plan
   *
   * @param mode - Permission mode to set
   * @returns true on success, false if no active stream
   */
  async setPermissionMode(mode: 'default' | 'acceptEdits' | 'plan'): Promise<boolean> {
    if (!this.eventStream) {
      void logger.warn("Cannot set permission mode: no active session");
      return false;
    }

    try {
      await this.eventStream.setPermissionMode(mode);
      useStore.getState().setPermissionMode(mode);
      void logger.info("Permission mode changed", { mode });
      return true;
    } catch (err) {
      void logger.error("Failed to set permission mode", {
        error: err instanceof Error ? err.message : String(err),
      });
      return false;
    }
  }

  /**
   * Cycle to the next permission mode: default → acceptEdits → plan → default
   *
   * @returns true on success, false if no active stream
   */
  async cyclePermissionMode(): Promise<boolean> {
    const current = useStore.getState().permissionMode;
    const modes: Array<'default' | 'acceptEdits' | 'plan'> = ['default', 'acceptEdits', 'plan'];
    let currentIndex = modes.indexOf(current as 'default' | 'acceptEdits' | 'plan');
    if (currentIndex === -1) {
      currentIndex = 0; // Default to 'default' if unknown mode
    }
    const nextMode = modes[(currentIndex + 1) % modes.length]!; // Non-null: length is 3, modulo ensures valid index
    return this.setPermissionMode(nextMode);
  }

  /**
   * Shutdown the session and clean up resources
   */
  async shutdown(): Promise<void> {
    // Transition to DEAD (always valid from any state except DEAD itself)
    if (this.state !== SessionState.DEAD) {
      this.transitionTo(SessionState.DEAD);
    }

    // Reject active coordinator if exists
    if (this.activeCoordinator) {
      this.activeCoordinator.rejectResponse(new Error("Session shutdown"));
      this.activeCoordinator.queuedMessage?.reject(new Error("Session shutdown"));
      this.activeCoordinator = null;
    }

    // Interrupt if streaming
    if (this.eventStream) {
      try {
        await this.eventStream.interrupt?.();
      } catch {
        // Ignore interrupt errors during shutdown
      }
    }

    // Reject all queued messages
    for (const msg of this.messageQueue) {
      msg.reject(new Error("Session shutdown"));
    }
    this.messageQueue = [];

    // Wake up any waiting generator so it can exit
    if (this.queueNotifier) {
      this.queueNotifier();
      this.queueNotifier = null;
    }

    // Clear state
    this.eventStream = null;
    this.activeCoordinator = null;
    this.sessionId = null;
    this.initResolve = null;
    this.initReject = null;
  }

  /**
   * Get current session state
   */
  getState(): SessionState {
    return this.state;
  }

  /**
   * Get current session ID
   */
  getSessionId(): string | null {
    return this.sessionId;
  }

  /**
   * Attempt to reconnect after error
   *
   * @throws Error if max reconnect attempts exceeded
   */
  async reconnect(): Promise<void> {
    if (this.state !== SessionState.ERROR) {
      throw new Error(`Cannot reconnect from state: ${this.state}`);
    }

    // Check reconnect limit
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.transitionTo(SessionState.DEAD);
      throw new Error("Max reconnect attempts exceeded");
    }

    this.reconnectAttempts++;

    // Wait before attempting
    await new Promise((resolve) =>
      setTimeout(resolve, this.config.reconnectDelayMs)
    );

    // Transition to CONNECTING
    this.transitionTo(SessionState.CONNECTING);

    // Attempt connection
    try {
      await this.connect();
    } catch (error) {
      this.transitionTo(SessionState.ERROR);
      throw error;
    }
  }
}

/**
 * Singleton instance
 */
let instance: SessionManager | null = null;

/**
 * Track if shutdown handler has been registered
 */
let shutdownRegistered = false;

/**
 * Get the SessionManager singleton instance
 *
 * @param config - Optional configuration (only used on first call)
 * @returns SessionManager instance
 */
export function getSessionManager(
  config?: Partial<SessionManagerConfig>
): SessionManager {
  if (!instance) {
    instance = new SessionManager(config);

    // Register shutdown handler once
    if (!shutdownRegistered) {
      onShutdown(async () => {
        await instance?.shutdown();
      });
      shutdownRegistered = true;
    }
  }
  return instance;
}

/**
 * Reset the SessionManager singleton
 * Only use this for testing - in production, use shutdown() instead
 */
export function resetSessionManager(): void {
  instance = null;
}
