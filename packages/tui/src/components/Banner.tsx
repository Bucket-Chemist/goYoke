import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors, borders } from "../config/theme.js";

/**
 * Banner component - displays branding, session ID, and keybind hints
 * Positioned at top of Layout, uses theme constants
 * Model, cost, and streaming status are now shown in StatusLine
 */
export function Banner(): JSX.Element {
  const { sessionId } = useStore();

  return (
    <Box
      borderStyle={borders.banner}
      borderColor={colors.primary}
      paddingX={1}
      width="100%"
      justifyContent="space-between"
    >
      <Text bold color={colors.primary}>
        GOfortress
      </Text>
      <Text color={colors.muted}>
        Session: {sessionId ? sessionId.slice(0, 8) : "—"}
      </Text>
      <Text color={colors.muted} dimColor>
        Tab: panels · Esc: exit · /help
      </Text>
    </Box>
  );
}
