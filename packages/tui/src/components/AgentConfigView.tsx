import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";

/**
 * AgentConfigView component
 * Placeholder for agent configuration editing
 */
export function AgentConfigView(): JSX.Element {
  return (
    <Box
      flexDirection="column"
      paddingX={1}
      paddingY={0}
      borderStyle="single"
      borderColor={colors.muted}
      padding={1}
    >
      <Text bold color={colors.primary}>
        Agent Configuration
      </Text>
      <Box marginTop={1}>
        <Text color={colors.muted}>
          Placeholder - agent config editing coming soon
        </Text>
      </Box>
    </Box>
  );
}
