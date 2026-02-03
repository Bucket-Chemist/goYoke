import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import { colors } from "./config/theme.js";
import { LayoutSpike } from "./components/LayoutSpike.js";
import { ResponsiveLayout } from "./components/ResponsiveLayout.js";
import { BorderStyleTest } from "./components/BorderStyleTest.js";

type DemoMode = "hello" | "layout" | "responsive" | "borders";

/**
 * Root component for GOfortress TUI
 * Entry point for the application UI structure
 * Includes spike testing modes for layout validation
 */
export function App(): JSX.Element {
  const [mode, setMode] = useState<DemoMode>("hello");

  useInput((input, key) => {
    // Cycle through demo modes with number keys
    if (input === "1") setMode("hello");
    if (input === "2") setMode("layout");
    if (input === "3") setMode("responsive");
    if (input === "4") setMode("borders");

    // Exit handled by Ink's default Ctrl+C
  });

  return (
    <Box flexDirection="column" width="100%" height="100%">
      {/* Header */}
      <Box borderStyle="round" borderColor={colors.primary} paddingX={2}>
        <Text bold color={colors.primary}>
          GOfortress TUI - Ink Layout Spike
        </Text>
      </Box>

      {/* Mode selector */}
      <Box paddingX={2} paddingY={1}>
        <Text dimColor color={colors.muted}>
          Press: [1] Hello | [2] Layout | [3] Responsive | [4] Borders | [Ctrl+C] Exit
        </Text>
      </Box>

      {/* Content area */}
      <Box flexGrow={1}>
        {mode === "hello" && (
          <Box flexDirection="column" padding={1}>
            <Text bold color={colors.primary}>
              GOfortress TUI
            </Text>
            <Text color={colors.muted}>Hello from Ink!</Text>
            <Box marginTop={1}>
              <Text color={colors.secondary}>
                Use number keys to test different spike components
              </Text>
            </Box>
          </Box>
        )}

        {mode === "layout" && <LayoutSpike />}
        {mode === "responsive" && <ResponsiveLayout />}
        {mode === "borders" && <BorderStyleTest />}
      </Box>
    </Box>
  );
}
