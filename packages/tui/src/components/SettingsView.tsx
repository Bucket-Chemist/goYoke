import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";

/**
 * SettingsView component
 * Shows configuration and keybindings
 */
export function SettingsView(): JSX.Element {
  const { permissionMode, activeModel, preferredModel } = useStore();

  return (
    <Box flexDirection="column" paddingX={1} paddingY={0}>
      <Text bold color={colors.primary}>Settings</Text>
      <Box marginTop={1} flexDirection="column">
        <Text><Text color={colors.muted}>Permission: </Text>{permissionMode || "default"}</Text>
        <Text><Text color={colors.muted}>Model:      </Text>{activeModel || preferredModel || "—"}</Text>
        <Text><Text color={colors.muted}>MCP:        </Text>gofortress</Text>
      </Box>
      <Box marginTop={1} flexDirection="column">
        <Text bold color={colors.muted}>Keybindings</Text>
        <Text color={colors.muted}>  Tab     Switch panel</Text>
        <Text color={colors.muted}>  o       Cycle right view</Text>
        <Text color={colors.muted}>  e       Expand/collapse tools</Text>
        <Text color={colors.muted}>  Ctrl+F  Search messages</Text>
        <Text color={colors.muted}>  Ctrl+L  Clear screen</Text>
        <Text color={colors.muted}>  Esc     Exit</Text>
      </Box>
    </Box>
  );
}
