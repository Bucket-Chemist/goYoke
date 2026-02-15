/**
 * MessageRenderer - Renders individual messages with markdown and collapsible tool blocks
 * Features:
 * - Role-based styling (user/assistant/system)
 * - Markdown rendering for assistant messages
 * - Collapsible tool blocks (single-line summaries)
 * - Tool result indicators (✓/✗)
 */

import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";
import { renderMarkdown } from "../utils/markdown.js";
import { sanitizeAnsi } from "../utils/ansi.js";
import type { Message, ContentBlock } from "../store/types.js";

export interface MessageRendererProps {
  message: Message;
  maxWidth?: number;
  allExpanded?: boolean;
}

/**
 * Render a single message with role-based styling and markdown support
 */
export function MessageRenderer({ message, maxWidth, allExpanded }: MessageRendererProps): JSX.Element {
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

  // Render text content (with markdown for assistant messages)
  const renderedText = message.role === "assistant" && textContent
    ? (() => {
        try {
          return renderMarkdown(textContent);
        } catch {
          return textContent;
        }
      })()
    : textContent;

  // Extract tool blocks
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

      {/* Text content - render markdown for assistant, plain for others */}
      {renderedText && (
        <Box flexDirection="column" paddingLeft={2}>
          {message.role === "assistant" ? (
            // Markdown already formatted with ANSI codes, render as single text
            <Text wrap="wrap">{renderedText}</Text>
          ) : (
            // Plain text for user/system - split lines for proper wrapping
            renderedText.split('\n').map((line, idx) => (
              <Text key={`${message.id}-line-${idx}`} wrap="wrap">{line || ' '}</Text>
            ))
          )}
        </Box>
      )}

      {/* Tool blocks - collapsed single-line summaries */}
      {toolBlocks.map((block) => {
        // Use stable ID for tool blocks
        const blockId = block.type === "tool_use"
          ? block.id
          : block.type === "tool_result"
            ? block.tool_use_id
            : `unknown-${Math.random()}`;

        const isToolUse = block.type === "tool_use";

        if (isToolUse) {
          // --- TodoWrite: dedicated task list rendering ---
          if (block.name === 'TodoWrite' && block.input) {
            const todos = (block.input as Record<string, unknown>)['todos'] as
              Array<{ content: string; status: string; activeForm?: string }> | undefined;

            if (todos && Array.isArray(todos)) {
              return (
                <Box key={blockId} paddingLeft={2} flexDirection="column">
                  {todos.map((todo, idx) => {
                    const icon = todo.status === 'completed' ? '\u2713'
                      : todo.status === 'in_progress' ? '\u25B6'
                      : '\u25CB';
                    const statusColor = todo.status === 'completed' ? colors.success
                      : todo.status === 'in_progress' ? colors.warning
                      : colors.muted;

                    return (
                      <Box key={`${blockId}-todo-${idx}`}>
                        <Text color={statusColor}>{icon} </Text>
                        <Text
                          color={todo.status === 'completed' ? colors.muted : colors.assistantMessage}
                          dimColor={todo.status === 'completed'}
                          strikethrough={todo.status === 'completed'}
                        >
                          {todo.content}
                        </Text>
                        {todo.status === 'in_progress' && todo.activeForm && (
                          <Text color={colors.warning} dimColor> ({todo.activeForm})</Text>
                        )}
                      </Box>
                    );
                  })}
                </Box>
              );
            }
          }

          const isExpanded = allExpanded ?? false;

          if (!isExpanded) {
            // Collapsed view: Single-line summary
            const inputSummary = block.input
              ? Object.entries(block.input)
                  .slice(0, 2) // Show max 2 params
                  .map(([k, v]) => {
                    const val = typeof v === 'string' ? sanitizeAnsi(v) : JSON.stringify(v);
                    const maxParamWidth = maxWidth ? Math.floor(maxWidth / 2) : 40;
                    const truncated = val.length > maxParamWidth
                      ? val.slice(0, maxParamWidth - 3) + '...'
                      : val;
                    return `${k}: ${truncated}`;
                  })
                  .join('  ')
              : '';

            return (
              <Box key={blockId} paddingLeft={2}>
                <Text color={colors.accent} dimColor>
                  ▸ [{block.name}] {inputSummary}
                </Text>
              </Box>
            );
          }

          // Expanded view: Show all parameters with values
          return (
            <Box key={blockId} paddingLeft={2} flexDirection="column">
              <Text color={colors.accent} bold>▾ [{block.name}]</Text>
              {block.input && Object.entries(block.input).map(([key, value]) => {
                const displayValue = typeof value === 'string'
                  ? sanitizeAnsi(value)
                  : JSON.stringify(value, null, 2);
                const maxLen = maxWidth ? maxWidth - 8 : 120;
                const truncated = displayValue.length > maxLen
                  ? displayValue.slice(0, maxLen - 3) + '...'
                  : displayValue;
                return (
                  <Box key={key} paddingLeft={2}>
                    <Text color={colors.muted}>{key}: </Text>
                    <Text color={colors.assistantMessage} wrap="wrap">{truncated}</Text>
                  </Box>
                );
              })}
            </Box>
          );
        }

        // tool_result
        const isExpanded = allExpanded ?? false;

        if (!isExpanded) {
          // Collapsed view: minimal indicator
          return (
            <Box key={blockId} paddingLeft={2}>
              <Text color={colors.muted} dimColor>
                ▸ [result] {block.is_error ? '✗' : '✓'}
              </Text>
            </Box>
          );
        }

        // Expanded view: show content preview
        const content = sanitizeAnsi(typeof block.content === 'string' ? block.content : '');
        const lines = content.split('\n').slice(0, 5);
        const totalLines = content.split('\n').length;

        return (
          <Box key={blockId} paddingLeft={2} flexDirection="column">
            <Text color={colors.muted}>▾ [result] {block.is_error ? '✗' : '✓'}</Text>
            {lines.map((line, i) => (
              <Box key={`${blockId}-line-${i}`} paddingLeft={2}>
                <Text color={colors.assistantMessage} dimColor wrap="wrap">{line || ' '}</Text>
              </Box>
            ))}
            {totalLines > 5 && (
              <Box paddingLeft={2}>
                <Text color={colors.muted} dimColor italic>... ({totalLines - 5} more lines)</Text>
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
}
