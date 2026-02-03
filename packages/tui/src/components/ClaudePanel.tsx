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

import React, { useState, useRef, useEffect } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import { createClaudePanelBindings } from "../config/keybindings.js";
import { Viewport } from "./primitives/Viewport.js";
import { TextInput } from "./primitives/TextInput.js";
import { Spinner } from "./primitives/Spinner.js";
import { renderMarkdown } from "../utils/markdown.js";
import { colors, borders } from "../config/theme.js";
import type { Message, ContentBlock } from "../store/types.js";

export interface ClaudePanelProps {
  /**
   * Whether this panel has focus
   */
  focused: boolean;
}

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

      {/* Text content with markdown rendering */}
      {textContent && (
        <Box flexDirection="column" paddingLeft={2}>
          <Text>{renderMarkdown(textContent)}</Text>
        </Box>
      )}

      {/* Tool blocks (simplified display) */}
      {toolBlocks.map((block, idx) => (
        <Box key={idx} paddingLeft={2}>
          {block.type === "tool_use" && (
            <Text color={colors.accent} dimColor>
              [Tool: {block.name}]
            </Text>
          )}
          {block.type === "tool_result" && (
            <Text color={colors.muted} dimColor>
              [Tool result]
            </Text>
          )}
        </Box>
      ))}
    </Box>
  );
}

/**
 * Main conversation panel with messages and input
 */
export function ClaudePanel({ focused }: ClaudePanelProps): JSX.Element {
  const {
    messages,
    streaming,
    addMessage,
    addToHistory,
    navigateHistory,
    resetHistoryIndex,
    modalQueue,
  } = useStore();
  const [input, setInput] = useState("");
  const [pendingMessage, setPendingMessage] = useState<string | null>(null);
  const currentInputRef = useRef(""); // Store current input when navigating history

  // Handle mock API response with cleanup
  useEffect(() => {
    if (!pendingMessage) return;

    const timerId = setTimeout(() => {
      addMessage({
        role: "assistant",
        content: [
          {
            type: "text",
            text: `Mock response to: "${pendingMessage}"\n\nThis is a placeholder. Real Claude API integration coming soon.`,
          },
        ],
        partial: false,
      });
      setPendingMessage(null);
    }, 500);

    return () => clearTimeout(timerId);
  }, [pendingMessage, addMessage]);

  // Handle message submission
  const handleSubmit = (): void => {
    const trimmedInput = input.trim();
    if (!trimmedInput || streaming) {
      return;
    }

    // Add to input history (TUI-005)
    addToHistory(trimmedInput);

    // Add user message to store
    addMessage({
      role: "user",
      content: [{ type: "text", text: trimmedInput }],
      partial: false,
    });

    // Clear input and reset history navigation
    setInput("");
    currentInputRef.current = "";
    resetHistoryIndex();

    // TODO: Trigger Claude API call
    // For now, trigger mock response via state
    setPendingMessage(trimmedInput);
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
      </Box>

      {/* Message viewport */}
      <Viewport
        items={messages}
        renderItem={renderMessage}
        height={20}
        focused={focused && !streaming}
        autoScroll={true}
      />

      {/* Input area */}
      <Box flexDirection="column" marginTop={1}>
        <TextInput
          value={input}
          onChange={setInput}
          onSubmit={handleSubmit}
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
