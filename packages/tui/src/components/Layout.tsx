import React from "react";
import { Box, Text, useInput } from "ink";
import { useStore } from "../store/index.js";
import { Banner } from "./Banner.js";
import { ClaudePanel } from "./ClaudePanel.js";
import { ModalOverlay } from "./Modal.js";
import { colors, borders } from "../config/theme.js";

/**
 * Layout component - main 2-panel split (70/30) with focus management
 *
 * Features:
 * - Left panel (70%): Claude conversation (placeholder)
 * - Right panel (30%): Agent tree (60%) + Agent detail (40%)
 * - Tab key switches focus between claude/agents panels
 * - Escape exits when no modal active
 * - Focus indicated by border color (cyan/gray from theme)
 * - Modal overlay renders when queue is non-empty
 */
export function Layout(): JSX.Element {
  const { focusedPanel, setFocusedPanel, modalQueue } = useStore();

  useInput((input, key) => {
    // Only handle input when no modal is active
    if (modalQueue.length === 0) {
      // Tab switches focus between panels
      if (key.tab) {
        setFocusedPanel(focusedPanel === "claude" ? "agents" : "claude");
      }

      // Escape exits when no modal is active
      if (key.escape) {
        process.exit(0);
      }
    }
  });

  return (
    <Box flexDirection="column" height="100%">
      <Banner />
      <Box flexDirection="row" flexGrow={1} position="relative">
        {/* Left Panel: Claude conversation (70%) */}
        <Box width="70%">
          <ClaudePanel focused={focusedPanel === "claude"} />
        </Box>

        {/* Right Panel: Agent tree + detail (30%) */}
        <Box width="30%" flexDirection="column">
          {/* Agent Tree (60% of right panel) */}
          <Box
            height="60%"
            borderStyle={borders.panel}
            borderColor={focusedPanel === "agents" ? colors.focused : colors.unfocused}
            flexDirection="column"
            paddingX={1}
          >
            <Text color={colors.muted}>Agent Tree (placeholder)</Text>
          </Box>

          {/* Agent Detail (40% of right panel) */}
          <Box
            height="40%"
            borderStyle={borders.panel}
            borderColor={colors.muted}
            flexDirection="column"
            paddingX={1}
          >
            <Text color={colors.muted}>Agent Detail (placeholder)</Text>
          </Box>
        </Box>

        {/* Modal overlay (rendered when queue is non-empty) */}
        {modalQueue.length > 0 && modalQueue[0] && <ModalOverlay request={modalQueue[0]} />}
      </Box>
    </Box>
  );
}
