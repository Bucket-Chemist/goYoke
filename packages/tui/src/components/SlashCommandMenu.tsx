/**
 * SlashCommandMenu — autocomplete dropdown for slash commands
 * Appears above the input when user types "/"
 * Up/Down to navigate, Enter/Tab to complete, Escape to dismiss
 */

import React from "react";
import { Box, Text } from "ink";
import { colors, borders } from "../config/theme.js";
import type { SlashCommand } from "../utils/slashCommands.js";

interface SlashCommandMenuProps {
  commands: SlashCommand[];
  selectedIndex: number;
  maxVisible?: number;
}

const SOURCE_TAGS: Record<string, string> = {
  builtin: "tui",
  native: "cli",
  skill: "skill",
};

export function SlashCommandMenu({
  commands,
  selectedIndex,
  maxVisible = 10,
}: SlashCommandMenuProps): JSX.Element {
  if (commands.length === 0) {
    return (
      <Box borderStyle={borders.input} borderColor={colors.muted} paddingX={1}>
        <Text color={colors.muted} italic>No matching commands</Text>
      </Box>
    );
  }

  // Windowed view: keep selected item visible
  const total = commands.length;
  const visible = Math.min(maxVisible, total);
  let startIdx = 0;
  if (selectedIndex >= visible) {
    startIdx = Math.min(selectedIndex - visible + 1, total - visible);
  }
  const visibleCommands = commands.slice(startIdx, startIdx + visible);

  return (
    <Box
      flexDirection="column"
      borderStyle={borders.input}
      borderColor={colors.focused}
      paddingX={1}
    >
      {startIdx > 0 && (
        <Text color={colors.muted} dimColor>  ↑ {startIdx} more</Text>
      )}
      {visibleCommands.map((cmd, i) => {
        const actualIndex = startIdx + i;
        const isSelected = actualIndex === selectedIndex;
        const tag = SOURCE_TAGS[cmd.source] || cmd.source;

        return (
          <Box key={cmd.name} flexDirection="row">
            <Text color={isSelected ? colors.primary : colors.muted} bold={isSelected}>
              {isSelected ? "▶ " : "  "}
              /{cmd.name}
            </Text>
            <Text color={colors.muted} dimColor> [{tag}] </Text>
            <Text color={colors.muted} dimColor wrap="wrap">
              {cmd.description}
            </Text>
          </Box>
        );
      })}
      {startIdx + visible < total && (
        <Text color={colors.muted} dimColor>  ↓ {total - startIdx - visible} more</Text>
      )}
      <Text dimColor>↑↓ Navigate • Enter/Tab Complete • Esc Dismiss</Text>
    </Box>
  );
}
