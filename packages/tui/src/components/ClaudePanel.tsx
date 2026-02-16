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

import React, { useState, useRef, useEffect, useCallback } from "react";
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
    messages,
    streaming,
    addToHistory,
    navigateHistory,
    resetHistoryIndex,
    modalQueue,
    isPlanMode,
    setClearPendingMessage,
  } = useStore();
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
    useStore.getState().addMessage({
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

  // Handle /model command
  const handleModelCommand = async (arg: string): Promise<void> => {
    if (arg) {
      // Direct model set: /model haiku
      // Use short aliases - SDK prefers these and resolves to latest version
      const MODEL_ALIASES: Record<string, string> = {
        "haiku": "haiku",
        "sonnet": "sonnet",
        "opus": "opus",
        "gemini": "gemini-pro",
        "flash": "gemini-flash",
      };
      const modelId = MODEL_ALIASES[arg.toLowerCase()] || arg;

      void logger.debug("Setting model", { modelId });

      // Try setModel first (works if query active AND in streaming input mode)
      const success = await setModel(modelId);
      if (success) {
        useStore.getState().setActiveModel(modelId);
        addSystemMessage(`Model switched to: ${modelId}`);
      } else {
        // No active query - store preference for next message
        void logger.debug("No active query, storing preference", { modelId });
        useStore.getState().setPreferredModel(modelId);
        useStore.getState().setActiveModel(modelId);
        addSystemMessage(`Model set to: ${modelId}. Will apply on next message.`);
      }
    } else {
      // Show model selector modal - use short aliases
      const result = await useStore.getState().enqueue({
        type: "select",
        payload: {
          message: "Select a model:",
          options: [
            {
              label: "Haiku (fast, cheap)",
              value: "haiku",
            },
            {
              label: "Sonnet (balanced)",
              value: "sonnet",
            },
            {
              label: "Opus (powerful)",
              value: "opus",
            },
            {
              label: "Gemini 3 Pro (Powerful)",
              value: "gemini-pro",
            },
            {
              label: "Gemini 3 Flash (Fast)",
              value: "gemini-flash",
            },
          ],
        },
      });

      if (result.type === "select" && result.selected) {
        // Try setModel first (works if query active)
        const success = await setModel(result.selected);
        if (success) {
          useStore.getState().setActiveModel(result.selected);
          addSystemMessage(`Model switched to: ${result.selected}`);
        } else {
          // No active query - store preference for next message
          useStore.getState().setPreferredModel(result.selected);
          useStore.getState().setActiveModel(result.selected);
          addSystemMessage(`Model set to: ${result.selected}. Will apply on next message.`);
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
          setInput("");
          return;

        case "help":
          addSystemMessage(
            "Available commands:\n" +
              "  /model [haiku|sonnet|opus] - Switch model\n" +
              "  /clear - Clear message history\n" +
              "  /help - Show this help"
          );
          setInput("");
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

  // Ctrl+E toggles tool blocks on/off (0↔1), Ctrl+Shift+E cycles levels (0→1→2→0)
  const toolToggleBindings: KeyBinding[] = [
    {
      key: "e",
      ctrl: true,
      shift: true,
      action: () => setExpansionLevel((prev) => (prev + 1) % 3),
      description: "Cycle expansion level (collapsed → expanded → full)",
    },
    {
      key: "e",
      ctrl: true,
      action: () => setExpansionLevel((prev) => prev > 0 ? 0 : 1),
      description: "Toggle tool expansion on/off",
    },
  ];

  // Enable panel bindings only when focused and no modal is active
  // Remove !streaming condition to allow input during streaming (TC-015a)
  useKeymap(panelBindings, focused && modalQueue.length === 0);
  // Allow Ctrl+E tool expansion even when modal is active — user needs to
  // expand tool blocks to read plan content while ExitPlanMode modal is showing.
  // No conflict: modal uses arrows/Enter/Escape, not Ctrl+E.
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
      <Box marginBottom={1}>
        <Text bold color={focused ? colors.focused : colors.muted}>
          Claude Conversation
        </Text>
        {isPlan && (
          <Text bold color="yellow"> [PLAN MODE]</Text>
        )}
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
