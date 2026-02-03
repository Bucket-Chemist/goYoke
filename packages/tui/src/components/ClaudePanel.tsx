/**
 * ClaudePanel - Main conversation panel
 * Features:
 * - Message viewport with scrolling
 * - Text input with submit handling
 * - Markdown rendering
 * - Streaming state management
 * - Visual distinction between user/assistant messages
 */

import React, { useState } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
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
  const { messages, streaming, addMessage } = useStore();
  const [input, setInput] = useState("");

  // Handle message submission
  const handleSubmit = (): void => {
    const trimmedInput = input.trim();
    if (!trimmedInput || streaming) {
      return;
    }

    // Add user message to store
    addMessage({
      role: "user",
      content: [{ type: "text", text: trimmedInput }],
      partial: false,
    });

    // Clear input
    setInput("");

    // TODO: Trigger Claude API call
    // For now, just add a mock response
    setTimeout(() => {
      addMessage({
        role: "assistant",
        content: [
          {
            type: "text",
            text: `Mock response to: "${trimmedInput}"\n\nThis is a placeholder. Real Claude API integration coming soon.`,
          },
        ],
        partial: false,
      });
    }, 500);
  };

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
