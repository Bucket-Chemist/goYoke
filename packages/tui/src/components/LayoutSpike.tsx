import React from "react";
import { Box, Text } from "ink";
import { colors, borders } from "../config/theme.js";

/**
 * Layout spike component testing 2-panel split layout
 * Tests flex layout with percentage-based widths and nested panels
 */
export function LayoutSpike(): JSX.Element {
  return (
    <Box flexDirection="row" width="100%" height="100%">
      {/* Left panel - 70% width */}
      <Box
        width="70%"
        borderStyle={borders.panel}
        borderColor={colors.focused}
        flexDirection="column"
        padding={1}
      >
        <Text bold color={colors.primary}>
          Left Panel (70%)
        </Text>
        <Text color={colors.muted}>Main content area</Text>
        <Text>Testing percentage-based width allocation</Text>
      </Box>

      {/* Right panel - 30% width with nested split */}
      <Box width="30%" flexDirection="column">
        {/* Top right - 60% height */}
        <Box
          height="60%"
          borderStyle={borders.panel}
          borderColor={colors.secondary}
          flexDirection="column"
          padding={1}
        >
          <Text bold color={colors.secondary}>
            Top Right (60%)
          </Text>
          <Text color={colors.muted}>Nested panel test</Text>
        </Box>

        {/* Bottom right - 40% height */}
        <Box
          height="40%"
          borderStyle={borders.panel}
          borderColor={colors.accent}
          flexDirection="column"
          padding={1}
        >
          <Text bold color={colors.accent}>
            Bottom Right (40%)
          </Text>
          <Text color={colors.muted}>Second nested panel</Text>
        </Box>
      </Box>
    </Box>
  );
}
