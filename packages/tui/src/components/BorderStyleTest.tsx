import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";

/**
 * Border style test component
 * Tests all available Ink border styles: single, double, round, bold
 * Verifies no rendering artifacts occur
 */
export function BorderStyleTest(): JSX.Element {
  const borderStyles = ["single", "double", "round", "bold"] as const;

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color={colors.primary}>
        Border Style Test
      </Text>

      <Box marginTop={1} flexDirection="column">
        {borderStyles.map((style) => (
          <Box key={style} marginTop={1}>
            <Box
              borderStyle={style}
              borderColor={colors.focused}
              paddingX={2}
              paddingY={1}
              width={40}
            >
              <Text>
                <Text bold color={colors.secondary}>
                  {style}:
                </Text>{" "}
                <Text color={colors.muted}>Border style test</Text>
              </Text>
            </Box>
          </Box>
        ))}
      </Box>

      <Box marginTop={1}>
        <Text dimColor color={colors.muted}>
          Check for rendering artifacts at corners and edges
        </Text>
      </Box>
    </Box>
  );
}
