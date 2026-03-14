/**
 * ClaudePanel - Main conversation panel
 * Features:
 * - Message viewport with scrolling
 * - Text input with submit handling and history navigation
 * - Markdown rendering
 * - Streaming state management
 * - Visual distinction between user/assistant messages
 * - Up/Down arrow keys for input history (TUI-005 integration)
 */

import React, { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { Box, Text, measureElement } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import type { KeyBinding } from "../hooks/useKeymap.js";
import { createClaudePanelBindings } from "../config/keybindings.js";
import { useClaudeQuery } from "../hooks/useClaudeQuery.js";
import { ScrollView } from "./primitives/ScrollView.js";
import { TextInput } from "./primitives/TextInput.js";
import { Spinner } from "./primitives/Spinner.js";
import { MessageRenderer } from "./MessageRenderer.js";
import { colors, borders } from "../config/theme.js";
import { logger } from "../utils/logger.js";
import { filterCommands } from "../utils/slashCommands.js";
import { SlashCommandMenu } from "./SlashCommandMenu.js";
import { PROVIDERS } from "../config/providers.js";
import { ProviderTabs } from "./ProviderTabs.js";
import { initiateShutdown } from "../lifecycle/shutdown.js";

export interface ClaudePanelProps {
  /**
   * Whether this panel has focus
   */
  focused: boolean;
  /**
   * Available character columns for content (for width constraints)
   */
  width?: number;
}

/**
 * Main conversation panel with messages and input
 */
export function ClaudePanel({ focused, width }: ClaudePanelProps): JSX.Element {
  const {
    streaming,
    addToHistory,
    navigateHistory,
    resetHistoryIndex,
    modalQueue,
    isPlanMode,
    setClearPendingMessage,
    getActiveMessages,
    activeProvider,
  } = useStore();
  // Use active provider's messages instead of global messages array
  const messages = getActiveMessages();

  // Collect tool_use ids for agent-spawning calls so MessageRenderer can suppress their verbose content.
  // "Agent" = Claude Agent SDK tool name, "spawn_agent" = MCP spawn tool
  const taskToolUseIds = useMemo(() => {
    const ids = new Set<string>();
    for (const msg of messages) {
      for (const block of msg.content) {
        if (block.type === "tool_use" && (block.name === "Agent" || block.name === "Task" || block.name === "spawn_agent") && block.id) {
          ids.add(block.id);
        }
      }
    }
    return ids;
  }, [messages]);

  const [input, setInput] = useState("");
  // Expansion level: 0=collapsed, 1=expanded (truncated), 2=full (no truncation)
  const [expansionLevel, setExpansionLevel] = useState(0);
  const currentInputRef = useRef(""); // Store current input when navigating history
  const isPlan = isPlanMode(); // Compute plan mode state

  // Slash command autocomplete state
  const [slashMenuIndex, setSlashMenuIndex] = useState(0);
  const slashQuery = input.startsWith("/") && !input.includes(" ") ? input.slice(1) : null;
  const slashMatches = slashQuery !== null ? filterCommands(slashQuery) : [];
  const showSlashMenu = slashQuery !== null && !streaming;

  // Reset slash menu selection when filter changes
  useEffect(() => {
    setSlashMenuIndex(0);
  }, [slashQuery]);

  // Input buffer state (TC-015a)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null);
  const pendingMessageRef = useRef<string | null>(null);

  // Keep ref in sync with state to avoid stale closures
  useEffect(() => {
    pendingMessageRef.current = pendingMessage;
  }, [pendingMessage]);

  // Drain queue callback - uses ref to avoid stale closure
  const handleStreamingComplete = useCallback(() => {
    const pending = pendingMessageRef.current;
    if (pending) {
      setPendingMessage(null);
      // Use ref to get the latest sendMessage without circular dependency
      // Small delay to let React render the completed response
      setTimeout(() => {
        void sendMessageRef.current?.(pending);
      }, 100);
    }
  }, []);

  // Initialize useClaudeQuery with drain callback
  const { sendMessage: sendMessageOriginal, setModel, error } = useClaudeQuery({
    onStreamingComplete: handleStreamingComplete,
  });

  // Store sendMessage in ref for drain callback access
  const sendMessageRef = useRef<(msg: string) => Promise<void>>();
  sendMessageRef.current = sendMessageOriginal;

  // Wrap sendMessage for consistency
  const sendMessage = useCallback(async (msg: string) => {
    return sendMessageOriginal(msg);
  }, [sendMessageOriginal]);

  // Measure ScrollView height dynamically
  const scrollContainerRef = useRef<any>(null);
  const [scrollHeight, setScrollHeight] = useState(10);

  useEffect(() => {
    if (scrollContainerRef.current) {
      try {
        const measured = measureElement(scrollContainerRef.current);
        if (measured.height > 0) {
          setScrollHeight(measured.height);
        }
      } catch {
        // measureElement may fail in test environments
      }
    }
  });

  // Helper to add system messages
  const addSystemMessage = useCallback((text: string): void => {
    const store = useStore.getState();
    store.addProviderMessage(store.activeProvider, {
      role: "system",
      content: [{ type: "text", text }],
      partial: false,
    });
  }, []);

  // Helper to enqueue a message when streaming
  const enqueueMessage = useCallback((message: string) => {
    setPendingMessage(message);
    addSystemMessage("Message queued — will send when current response completes.");
  }, [addSystemMessage]);

  // Helper to clear pending message (for interrupt/cancel)
  const clearPendingMessage = useCallback(() => {
    if (pendingMessageRef.current) {
      setPendingMessage(null);
      addSystemMessage("Queued message cancelled.");
    }
  }, [addSystemMessage]);

  // Register clear function in store (like interruptQuery pattern)
  useEffect(() => {
    setClearPendingMessage(clearPendingMessage);
    return () => setClearPendingMessage(null);
  }, [clearPendingMessage, setClearPendingMessage]);

  // Handle /model command (provider-aware)
  const handleModelCommand = async (arg: string): Promise<void> => {
    const store = useStore.getState();
    const activeProvider = store.activeProvider;
    const providerConfig = PROVIDERS[activeProvider];

    if (!providerConfig) {
      addSystemMessage(`Error: Unknown provider "${activeProvider}"`);
      return;
    }

    if (arg) {
      // Direct model set: /model haiku
      const modelId = arg.trim();

      // Validate model exists in current provider
      const validModel = providerConfig.models.find((m) => m.id === modelId);
      if (!validModel) {
        const validIds = providerConfig.models.map((m) => m.id).join(", ");
        addSystemMessage(
          `Error: Model "${modelId}" not found for ${providerConfig.name}.\n` +
          `Valid models: ${validIds}`
        );
        return;
      }

      void logger.debug("Setting model", { modelId, provider: activeProvider });

      // Try setModel first (works if query active AND in streaming input mode)
      const success = await setModel(modelId);
      if (success) {
        store.setProviderModel(activeProvider, modelId);
        addSystemMessage(`Model switched to: ${validModel.displayName} for ${providerConfig.name}`);
      } else {
        // No active query - store preference for next message
        void logger.debug("No active query, storing preference", { modelId });
        store.setProviderModel(activeProvider, modelId);
        addSystemMessage(
          `Model set to: ${validModel.displayName} for ${providerConfig.name}. ` +
          `Will apply on next message.`
        );
      }
    } else {
      // Show model selector modal with provider-specific models
      const result = await store.enqueue({
        type: "select",
        payload: {
          message: `Select a model for ${providerConfig.name}:`,
          options: providerConfig.models.map((model) => ({
            label: `${model.displayName} - ${model.description}`,
            value: model.id,
          })),
        },
      });

      if (result.type === "select" && result.selected) {
        const selectedModel = providerConfig.models.find((m) => m.id === result.selected);
        if (!selectedModel) {
          addSystemMessage(`Error: Model "${result.selected}" not found`);
          return;
        }

        // Try setModel first (works if query active)
        const success = await setModel(result.selected);
        if (success) {
          store.setProviderModel(activeProvider, result.selected);
          addSystemMessage(`Model switched to: ${selectedModel.displayName} for ${providerConfig.name}`);
        } else {
          // No active query - store preference for next message
          store.setProviderModel(activeProvider, result.selected);
          addSystemMessage(
            `Model set to: ${selectedModel.displayName} for ${providerConfig.name}. ` +
            `Will apply on next message.`
          );
        }
      }
    }
  };

  // Handle message submission
  const handleSubmit = (): void => {
    const trimmedInput = input.trim();
    if (!trimmedInput) {
      return;
    }

    // If streaming, enqueue the message instead of rejecting it
    if (streaming) {
      enqueueMessage(trimmedInput);
      setInput("");
      return;
    }

    // Check for known slash commands (unknown commands pass through to Claude)
    if (trimmedInput.startsWith("/")) {
      const [command, ...args] = trimmedInput.slice(1).split(" ");

      // Guard against empty command (just "/" typed)
      if (!command) {
        setInput("");
        return;
      }

      // Handle known commands - these return early
      switch (command.toLowerCase()) {
        case "model":
          void handleModelCommand(args.join(" "));
          setInput("");
          return;

        case "clear":
          useStore.getState().clearMessages();
          // Fall through to send /clear to CLI so session resets
          break;

        case "help":
          addSystemMessage(
            "Available commands:\n" +
              "  /model [haiku|sonnet|opus] - Switch model\n" +
              "  /clear - Clear message history\n" +
              "  /exit - Exit gracefully (saves session)\n" +
              "  /help - Show this help"
          );
          setInput("");
          return;

        case "exit":
        case "quit":
          setInput("");
          void initiateShutdown("/exit");
          return;

        default:
          // Unknown command - fall through to normal submission
          break;
      }
      // Execution continues here for unknown slash commands
    }

    // Normal message submission (reached by regular messages and unknown slash commands)
    // Add to input history (TUI-005)
    addToHistory(trimmedInput);

    // Clear input and reset history navigation
    setInput("");
    currentInputRef.current = "";
    resetHistoryIndex();

    // Send to Claude API
    void sendMessage(trimmedInput);
  };

  // Navigate to previous input in history (up arrow)
  const handleHistoryPrev = (): void => {
    // Save current input if we're starting navigation
    const historyIndex = useStore.getState().inputHistoryIndex;
    if (historyIndex === -1) {
      currentInputRef.current = input;
    }

    const historyItem = navigateHistory("up");
    if (historyItem !== null) {
      setInput(historyItem);
    }
  };

  // Navigate to next input in history (down arrow)
  const handleHistoryNext = (): void => {
    const historyItem = navigateHistory("down");
    if (historyItem !== null) {
      setInput(historyItem);
    } else {
      // Reached the end, restore current input
      setInput(currentInputRef.current);
      resetHistoryIndex();
    }
  };

  // Slash menu handlers
  const handleSlashMenuUp = (): void => {
    setSlashMenuIndex((prev) => (prev > 0 ? prev - 1 : slashMatches.length - 1));
  };
  const handleSlashMenuDown = (): void => {
    setSlashMenuIndex((prev) => (prev < slashMatches.length - 1 ? prev + 1 : 0));
  };
  const handleSlashMenuSelect = (): void => {
    const selected = slashMatches[slashMenuIndex];
    if (selected) {
      setInput(`/${selected.name} `);
    }
  };
  const handleSlashMenuDismiss = (): void => {
    setInput("");
  };

  // Panel-specific key bindings — slash menu overrides history nav when visible
  const panelBindings: KeyBinding[] = showSlashMenu
    ? [
        { key: "return", action: handleSlashMenuSelect, description: "Select command" },
        { key: "tab", action: handleSlashMenuSelect, description: "Complete command" },
        { key: "up", action: handleSlashMenuUp, description: "Previous command" },
        { key: "down", action: handleSlashMenuDown, description: "Next command" },
        { key: "escape", action: handleSlashMenuDismiss, description: "Dismiss menu" },
      ]
    : createClaudePanelBindings({
        submitMessage: handleSubmit,
        historyPrev: handleHistoryPrev,
        historyNext: handleHistoryNext,
      });

  // Alt+E / Alt+Shift+E for tool expansion — NOT Ctrl+E / Ctrl+Shift+E.
  // Ctrl+E is eaten by IBus Unicode input (Linux, system-wide) and Zellij scroll mode.
  // Ctrl+Shift+E is eaten by IBus Unicode codepoint entry unconditionally on GNOME.
  // Alt (Meta) keys pass through both Zellij panes and IBus without interception.
  // Terminal sends Alt+Shift+E as Meta + capital "E" (ESC+E sequence).
  const toolToggleBindings: KeyBinding[] = [
    {
      key: "E", // Alt+Shift+E
      meta: true,
      action: () => setExpansionLevel((prev) => (prev + 1) % 3),
      description: "Cycle expansion level (collapsed → expanded → full)",
    },
    {
      key: "e", // Alt+E
      meta: true,
      action: () => setExpansionLevel((prev) => prev > 0 ? 0 : 1),
      description: "Toggle tool expansion on/off",
    },
  ];

  // Enable panel bindings only when focused and no modal is active
  // Remove !streaming condition to allow input during streaming (TC-015a)
  useKeymap(panelBindings, focused && modalQueue.length === 0);
  // Allow Alt+E tool expansion even when modal is active — user needs to
  // expand tool blocks to read plan content while ExitPlanMode modal is showing.
  // No conflict: modal uses arrows/Enter/Escape, not Alt+E.
  useKeymap(toolToggleBindings, focused);

  return (
    <Box
      flexDirection="column"
      borderStyle={borders.panel}
      borderColor={focused ? colors.focused : colors.unfocused}
      paddingX={1}
      height="100%"
    >
      {/* Header */}
      <Box marginBottom={1} flexDirection="column">
        <Box marginBottom={0}>
          <Text bold color={focused ? colors.focused : colors.muted}>
            {PROVIDERS[activeProvider]?.name ?? "Claude"} Conversation
          </Text>
          {isPlan && (
            <Text bold color="yellow"> [PLAN MODE]</Text>
          )}
        </Box>
        <ProviderTabs enabled={focused && modalQueue.length === 0} />
      </Box>

      {/* Plan mode info banner */}
      {isPlan && (
        <Box marginBottom={1} borderStyle="round" borderColor="yellow" paddingX={1}>
          <Text color="yellow">
            📋 Claude is planning. Review the plan before approving.
          </Text>
        </Box>
      )}

      {/* Message viewport - uses measured height */}
      <Box flexGrow={1} overflow="hidden" ref={scrollContainerRef}>
        <ScrollView
          height={scrollHeight}
          width={width}
          focused={focused}
          disableArrowKeys={true}
          autoScroll={true}
          forceScrollToBottom={modalQueue.length}
        >
          {messages.length === 0 ? (
            <Text color={colors.muted} italic>
              No messages yet. Start a conversation!
            </Text>
          ) : (
            messages.map((msg) => (
              <MessageRenderer
                key={msg.id}
                message={msg}
                maxWidth={width ? width - 4 : undefined}
                expansionLevel={expansionLevel}
                taskToolUseIds={taskToolUseIds}
              />
            ))
          )}
        </ScrollView>
      </Box>

      {/* Slash command autocomplete menu */}
      {showSlashMenu && (
        <SlashCommandMenu
          commands={slashMatches}
          selectedIndex={slashMenuIndex}
        />
      )}

      {/* Input area */}
      <Box flexDirection="column" marginTop={1}>
        <TextInput
          value={input}
          onChange={setInput}
          placeholder={streaming ? "Type to queue message..." : "Type a message..."}
          focused={focused}
        />

        {/* Queued message indicator */}
        {pendingMessage && (
          <Box marginTop={1}>
            <Text dimColor>
              Queued: {pendingMessage.slice(0, 60)}
              {pendingMessage.length > 60 ? "..." : ""}
            </Text>
          </Box>
        )}

        {/* Streaming indicator */}
        {streaming && (
          <Box marginTop={1}>
            <Spinner type="dots" />
            <Text color={colors.muted}> Claude is thinking...</Text>
          </Box>
        )}
      </Box>
    </Box>
  );
}
