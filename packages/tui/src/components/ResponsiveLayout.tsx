import React from "react";
import { Box, Text, useStdout } from "ink";
import { colors } from "../config/theme.js";

/**
 * Responsive layout component testing terminal resize handling
 * Displays current terminal dimensions and adjusts layout accordingly
 */
export function ResponsiveLayout(): JSX.Element {
  const { stdout } = useStdout();
  const cols = stdout?.columns ?? 80;
  const rows = stdout?.rows ?? 24;

  // Determine layout mode based on terminal size
  const isNarrow = cols < 100;
  const isShort = rows < 20;

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color={colors.primary}>
        Responsive Layout Test
      </Text>

      <Box marginTop={1}>
        <Text color={colors.secondary}>Terminal Size: </Text>
        <Text bold color={colors.focused}>
          {cols}x{rows}
        </Text>
      </Box>

      <Box marginTop={1} flexDirection="column">
        <Text color={colors.muted}>Layout Mode:</Text>
        {isNarrow && (
          <Text color={colors.warning}>• Narrow mode (&lt; 100 cols)</Text>
        )}
        {isShort && (
          <Text color={colors.warning}>• Short mode (&lt; 20 rows)</Text>
        )}
        {!isNarrow && !isShort && (
          <Text color={colors.success}>• Standard mode</Text>
        )}
      </Box>

      <Box marginTop={1} flexDirection="column">
        <Text color={colors.muted}>
          Resize your terminal to see this update
        </Text>
        <Text dimColor color={colors.muted}>
          (Press Ctrl+C to exit)
        </Text>
      </Box>
    </Box>
  );
}
