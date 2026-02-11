import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";

/**
 * TeamConfigView component
 * Placeholder for team configuration editing
 */
export function TeamConfigView(): JSX.Element {
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
        Team Configuration
      </Text>
      <Box marginTop={1}>
        <Text color={colors.muted}>
          Placeholder - team config editing coming soon
        </Text>
      </Box>
    </Box>
  );
}
