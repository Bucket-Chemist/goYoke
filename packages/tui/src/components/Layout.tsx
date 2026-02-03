import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import { createGlobalBindings } from "../config/keybindings.js";
import { Banner } from "./Banner.js";
import { ClaudePanel } from "./ClaudePanel.js";
import { ModalOverlay } from "./Modal.js";
import { colors, borders } from "../config/theme.js";

/**
 * Layout component - main 2-panel split (70/30) with focus management
 *
 * Features:
 * - Left panel (70%): Claude conversation
 * - Right panel (30%): Agent tree (60%) + Agent detail (40%)
 * - Global keyboard bindings (Tab, Escape, Ctrl+C, Ctrl+L)
 * - Focus indicated by border color (cyan/gray from theme)
 * - Modal overlay renders when queue is non-empty
 * - Modal captures all input when active
 */
export function Layout(): JSX.Element {
  const { focusedPanel, setFocusedPanel, modalQueue, clearMessages } = useStore();

  // Global key bindings (only active when no modal is present)
  const globalBindings = createGlobalBindings({
    toggleFocus: () => {
      setFocusedPanel(focusedPanel === "claude" ? "agents" : "claude");
    },
    handleEscape: () => {
      // If modal is active, cancel it; otherwise exit
      if (modalQueue.length > 0 && modalQueue[0]) {
        const modal = modalQueue[0];
        useStore.getState().cancel(modal.id);
      } else {
        process.exit(0);
      }
    },
    forceQuit: () => {
      process.exit(0);
    },
    clearScreen: () => {
      clearMessages();
    },
  });

  // Only enable global bindings when no modal is active
  useKeymap(globalBindings, modalQueue.length === 0);

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
