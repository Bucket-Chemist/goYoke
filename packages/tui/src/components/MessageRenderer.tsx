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
  /** Expansion level: 0=collapsed, 1=expanded (truncated), 2=full (no truncation) */
  expansionLevel?: number;
  /** @deprecated Use expansionLevel instead */
  allExpanded?: boolean;
  /** Set of tool_use ids that correspond to Task() calls — rendered as collapsed single-line indicators */
  taskToolUseIds?: Set<string>;
}

/**
 * Render a single message with role-based styling and markdown support
 */
export function MessageRenderer({ message, maxWidth, expansionLevel, allExpanded, taskToolUseIds }: MessageRendererProps): JSX.Element {
  // Support both new expansionLevel and deprecated allExpanded
  const level = expansionLevel ?? (allExpanded ? 1 : 0);
  // Determine message color based on role
  const roleColor =
    message.role === "user"
      ? colors.userMessage
      : message.role === "assistant"
        ? colors.assistantMessage
        : colors.systemMessage;

  // Check if this message is a task/spawn result — if so, suppress its text content entirely.
  // Tool results from handleUserEvent arrive as system messages with both text and tool_result blocks.
  const isTaskResultMessage = taskToolUseIds && taskToolUseIds.size > 0 && message.content.some(
    (block) => block.type === "tool_result" && taskToolUseIds.has(block.tool_use_id)
  );

  // Detect if this assistant message contains a Task/spawn_agent delegation call.
  // When it does, verbose prompt text in the same message should be suppressed.
  const isTaskDelegationMessage = message.role === "assistant" && message.content.some(
    (block) => block.type === "tool_use" && (block.name === "Task" || block.name === "spawn_agent")
  );

  // Extract text content from content blocks (suppressed for task result messages).
  // For task delegation messages: suppress verbose delegation prompt text blocks at default
  // expansion level (level 0). Short routing commentary (< 200 chars, no AGENT: prefix) is kept.
  const textContent = isTaskResultMessage
    ? ""
    : message.content
        .filter((block): block is Extract<ContentBlock, { type: "text" }> => block.type === "text")
        .filter((block) => {
          if (!isTaskDelegationMessage || level >= 1) return true;
          // Suppress blocks that look like delegation prompt templates:
          //   - Contain an "AGENT: " line (the standard prompt template header)
          //   - OR are longer than 200 chars (verbose prompt content)
          const isVerbosePrompt = /^AGENT:\s/m.test(block.text) || block.text.length > 200;
          return !isVerbosePrompt;
        })
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
          // Agent-spawning calls are always rendered as a collapsed single-line indicator,
          // regardless of expansionLevel — they contain verbose agent prompts/params.
          // "Task" = Claude Code CLI, "spawn_agent" = Agent SDK (TUI)
          if (block.name === "Task" || block.name === "spawn_agent") {
            const desc = typeof block.input?.["description"] === "string"
              ? block.input["description"] : "agent";
            const model = typeof block.input?.["model"] === "string"
              ? block.input["model"] : "";
            const agent = typeof block.input?.["agent"] === "string"
              ? block.input["agent"] : "";
            const label = agent || desc;
            return (
              <Box key={blockId} paddingLeft={2}>
                <Text color={colors.accent} dimColor>
                  ◐ [{agent ? agent : "Task"}{model ? ` → ${model}` : ""}] {label.length > 50 ? label.slice(0, 47) + "..." : label}
                </Text>
              </Box>
            );
          }

          if (level === 0) {
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

          // Level 1: expanded with truncation. Level 2: full, no truncation.
          const isFull = level >= 2;
          return (
            <Box key={blockId} paddingLeft={2} flexDirection="column">
              <Text color={colors.accent} bold>▾ [{block.name}]{isFull ? ' [FULL]' : ''}</Text>
              {block.input && Object.entries(block.input).map(([key, value]) => {
                const displayValue = typeof value === 'string'
                  ? sanitizeAnsi(value)
                  : JSON.stringify(value, null, 2);
                if (isFull) {
                  // Level 2: show everything, split into lines for wrapping
                  const lines = displayValue.split('\n');
                  return (
                    <Box key={key} paddingLeft={2} flexDirection="column">
                      <Text color={colors.muted}>{key}:</Text>
                      {lines.map((line, i) => (
                        <Box key={`${key}-${i}`} paddingLeft={2}>
                          <Text color={colors.assistantMessage} wrap="wrap">{line || ' '}</Text>
                        </Box>
                      ))}
                    </Box>
                  );
                }
                // Level 1: truncated
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
        // If this result corresponds to a Task() call, collapse it to a single-line indicator.
        if (taskToolUseIds?.has(block.tool_use_id)) {
          return (
            <Box key={blockId} paddingLeft={2}>
              <Text color={block.is_error ? colors.error : colors.muted} dimColor>
                {block.is_error ? "✗" : "✓"} [Task result]
              </Text>
            </Box>
          );
        }

        if (level === 0) {
          // Collapsed view: minimal indicator
          return (
            <Box key={blockId} paddingLeft={2}>
              <Text color={colors.muted} dimColor>
                ▸ [result] {block.is_error ? '✗' : '✓'}
              </Text>
            </Box>
          );
        }

        // Level 1+: show content
        const content = sanitizeAnsi(typeof block.content === 'string' ? block.content : '');
        const allLines = content.split('\n');
        const isFull = level >= 2;
        const displayLines = isFull ? allLines : allLines.slice(0, 5);
        const hiddenCount = allLines.length - displayLines.length;

        return (
          <Box key={blockId} paddingLeft={2} flexDirection="column">
            <Text color={colors.muted}>▾ [result] {block.is_error ? '✗' : '✓'}{isFull ? ' [FULL]' : ''}</Text>
            {displayLines.map((line, i) => (
              <Box key={`${blockId}-line-${i}`} paddingLeft={2}>
                <Text color={colors.assistantMessage} dimColor wrap="wrap">{line || ' '}</Text>
              </Box>
            ))}
            {hiddenCount > 0 && (
              <Box paddingLeft={2}>
                <Text color={colors.muted} dimColor italic>... ({hiddenCount} more lines — Ctrl+Shift+E for full)</Text>
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
}
