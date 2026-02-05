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

import React, { useState, useRef } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import { createClaudePanelBindings } from "../config/keybindings.js";
import { useClaudeQuery } from "../hooks/useClaudeQuery.js";
import { Viewport } from "./primitives/Viewport.js";
import { TextInput } from "./primitives/TextInput.js";
import { Spinner } from "./primitives/Spinner.js";
// Removed: import { renderMarkdown } - causes ANSI conflicts with Ink
import { colors, borders } from "../config/theme.js";
import type { Message, ContentBlock } from "../store/types.js";

export interface ClaudePanelProps {
  /**
   * Whether this panel has focus
   */
  focused: boolean;
  /**
   * Maximum height for the message viewport in rows
   */
  maxHeight?: number;
}

// Named constants for height calculations
// Prevents magic numbers and makes layout calculations explicit
const HEADER_ROWS = 2;
const INPUT_ROWS = 4;
const VIEWPORT_CHROME = 2;

/**
 * Render a single message with role-based styling
 */
function MessageItem({ message }: { message: Message }): JSX.Element {
  // Determine message color based on role
  const roleColor =
    message.role === "user"
      ? colors.userMessage
      : message.role === "assistant"
        ? colors.assistantMessage
        : colors.systemMessage;

  // Extract text content from content blocks
  const textContent = message.content
    .filter((block): block is Extract<ContentBlock, { type: "text" }> => block.type === "text")
    .map((block) => block.text)
    .join("\n");

  // Render tool use/results (simplified for now)
  const toolBlocks = message.content.filter(
    (block) => block.type === "tool_use" || block.type === "tool_result"
  );

  return (
    <Box flexDirection="column" marginY={0} paddingBottom={1}>
      {/* Role header */}
      <Box>
        <Text bold color={roleColor}>
          {message.role === "user" ? "You" : message.role === "assistant" ? "Claude" : "System"}
        </Text>
        {message.partial && (
          <Text color={colors.muted}> (streaming...)</Text>
        )}
      </Box>

      {/* Text content - render each line separately for proper Ink handling */}
      {textContent && (
        <Box flexDirection="column" paddingLeft={2}>
          {textContent.split('\n').map((line, idx) => (
            <Text key={`${message.id}-line-${idx}`}>{line || ' '}</Text>
          ))}
        </Box>
      )}

      {/* Tool blocks (simplified display) */}
      {toolBlocks.map((block) => {
        // Use stable ID for tool blocks
        const blockId = block.type === "tool_use"
          ? block.id
          : block.type === "tool_result"
            ? block.tool_use_id
            : `unknown-${Math.random()}`;

        return (
          <Box key={blockId} paddingLeft={2} flexDirection="column">
            {block.type === "tool_use" && (
              <>
                <Text color={colors.accent} dimColor bold>
                  [Tool: {block.name}]
                </Text>
                {/* Render tool inputs */}
                {block.input && Object.entries(block.input).map(([key, value]) => (
                  <Box key={key} paddingLeft={2}>
                    <Text color={colors.muted} dimColor>
                      {key}:{" "}
                    </Text>
                    <Text color={colors.assistantMessage}>
                      {typeof value === 'string' ? value : JSON.stringify(value)}
                    </Text>
                  </Box>
                ))}
              </>
            )}
            {block.type === "tool_result" && (
              <Box flexDirection="column">
                <Text color={colors.muted} dimColor>
                  [Tool result]
                </Text>
                {/* Optional: Render truncated result preview if needed */}
                {/* <Text color={colors.dim} numberOfLines={2}>{block.content}</Text> */}
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
}

/**
 * Main conversation panel with messages and input
 */
export function ClaudePanel({ focused, maxHeight = 20 }: ClaudePanelProps): JSX.Element {
  const {
    messages,
    streaming,
    addToHistory,
    navigateHistory,
    resetHistoryIndex,
    modalQueue,
    isPlanMode,
  } = useStore();
  const { sendMessage, setModel, error } = useClaudeQuery();
  const [input, setInput] = useState("");
  const currentInputRef = useRef(""); // Store current input when navigating history
  const isPlan = isPlanMode(); // Compute plan mode state

  // Helper to add system messages
  const addSystemMessage = (text: string): void => {
    useStore.getState().addMessage({
      role: "system",
      content: [{ type: "text", text }],
      partial: false,
    });
  };

  // Handle /model command
  const handleModelCommand = async (arg: string): Promise<void> => {
    if (arg) {
      // Direct model set: /model haiku
      // Use short aliases - SDK prefers these and resolves to latest version
      const MODEL_ALIASES: Record<string, string> = {
        "haiku": "haiku",
        "sonnet": "sonnet",
        "opus": "opus",
      };
      const modelId = MODEL_ALIASES[arg.toLowerCase()] || arg;

      console.log("[/model] Setting model to:", modelId);

      // Try setModel first (works if query active AND in streaming input mode)
      const success = await setModel(modelId);
      if (success) {
        addSystemMessage(`Model switched to: ${modelId}`);
      } else {
        // No active query - store preference for next message
        console.log("[/model] No active query, storing preference:", modelId);
        useStore.getState().setPreferredModel(modelId);
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
          ],
        },
      });

      if (result.type === "select" && result.selected) {
        // Try setModel first (works if query active)
        const success = await setModel(result.selected);
        if (success) {
          addSystemMessage(`Model switched to: ${result.selected}`);
        } else {
          // No active query - store preference for next message
          useStore.getState().setPreferredModel(result.selected);
          addSystemMessage(`Model set to: ${result.selected}. Will apply on next message.`);
        }
      }
    }
  };

  // Handle message submission
  const handleSubmit = (): void => {
    const trimmedInput = input.trim();
    if (!trimmedInput || streaming) {
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

  // Panel-specific key bindings (only active when focused and no modal)
  const panelBindings = createClaudePanelBindings({
    submitMessage: handleSubmit,
    historyPrev: handleHistoryPrev,
    historyNext: handleHistoryNext,
  });

  // Enable panel bindings only when focused and no modal is active
  useKeymap(panelBindings, focused && modalQueue.length === 0);

  // Render message item
  const renderMessage = (message: Message, _index: number): React.ReactNode => {
    return <MessageItem message={message} />;
  };

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

      {/* Message viewport - constrained to available space */}
      <Box height={maxHeight - HEADER_ROWS - INPUT_ROWS} overflow="hidden">
        <Viewport
          items={messages}
          renderItem={renderMessage}
          height={Math.max(5, maxHeight - HEADER_ROWS - INPUT_ROWS - VIEWPORT_CHROME)}
          focused={focused && !streaming}
          autoScroll={true}
        />
      </Box>

      {/* Input area */}
      <Box flexDirection="column" marginTop={1}>
        <TextInput
          value={input}
          onChange={setInput}
          placeholder={streaming ? "Waiting for response..." : "Type a message..."}
          disabled={streaming}
          focused={focused}
        />

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
