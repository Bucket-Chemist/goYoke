import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors, borders } from "../config/theme.js";

/**
 * Model ID to display name mapping (matches native Claude CLI format)
 */
const MODEL_DISPLAY_NAMES: Record<string, string> = {
  "claude-opus-4-5-20251101": "Opus 4.5",
  "claude-sonnet-4-5-20250929": "Sonnet 4.5",
  "claude-haiku-4-5-20251001": "Haiku 4.5",
  // Aliases resolve to full IDs, but handle them too
  "opus": "Opus",
  "sonnet": "Sonnet",
  "haiku": "Haiku",
};

/**
 * Get friendly display name for model ID
 */
function getModelDisplayName(modelId: string | null): string {
  if (!modelId) return "—";
  return MODEL_DISPLAY_NAMES[modelId] ?? modelId.replace("claude-", "");
}

/**
 * Banner component - displays session info, cost, and status
 * Positioned at top of Layout, uses theme constants
 */
export function Banner(): JSX.Element {
  const { sessionId, totalCost, streaming, activeModel } = useStore();

  const status = streaming ? "Streaming..." : "Ready";
  const costDisplay = `$${totalCost.toFixed(4)}`;
  const modelDisplay = getModelDisplayName(activeModel);

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
      <Text color={colors.accent}>{modelDisplay}</Text>
      <Text>Session: {sessionId ? sessionId.slice(0, 8) : "None"}</Text>
      <Text>Cost: {costDisplay}</Text>
      <Text color={streaming ? colors.warning : colors.success}>{status}</Text>
    </Box>
  );
}
