import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";

/**
 * TelemetryView component
 * Placeholder for telemetry analytics dashboard
 */
export function TelemetryView(): JSX.Element {
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
        Telemetry Analytics
      </Text>
      <Box marginTop={1}>
        <Text color={colors.muted}>
          Placeholder - telemetry dashboard coming soon
        </Text>
      </Box>
    </Box>
  );
}
