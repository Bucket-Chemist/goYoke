import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors, borders } from "../config/theme.js";

/**
 * Banner component - displays session info, cost, and status
 * Positioned at top of Layout, uses theme constants
 */
export function Banner(): JSX.Element {
  const { sessionId, totalCost, streaming } = useStore();

  const status = streaming ? "Streaming..." : "Ready";
  const costDisplay = `$${totalCost.toFixed(4)}`;

  return (
    <Box
      borderStyle={borders.banner}
      borderColor={colors.primary}
      paddingX={1}
      justifyContent="space-between"
    >
      <Text bold color={colors.primary}>
        GOfortress
      </Text>
      <Text>Session: {sessionId ?? "None"}</Text>
      <Text>Cost: {costDisplay}</Text>
      <Text color={streaming ? colors.warning : colors.success}>{status}</Text>
    </Box>
  );
}
