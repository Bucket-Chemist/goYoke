import React from "react";
import { Box } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import type { KeyBinding } from "../hooks/useKeymap.js";
import { useAgentTree } from "../hooks/useAgentTree.js";
import { useTerminalDimensions } from "../hooks/useTerminalDimensions.js";
import { createGlobalBindings } from "../config/keybindings.js";
import { Banner } from "./Banner.js";
import { ClaudePanel } from "./ClaudePanel.js";
import { AgentTree } from "./AgentTree.js";
import { AgentDetail } from "./AgentDetail.js";
import { DashboardView } from "./DashboardView.js";
import { SettingsView } from "./SettingsView.js";
import { ModalOverlay } from "./Modal.js";
import { StatusLine } from "./StatusLine.js";
import { ToastContainer } from "./Toast.js";
import { colors, borders } from "../config/theme.js";

// Fixed heights
const BANNER_HEIGHT = 3; // Banner takes 3 rows

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
  const { focusedPanel, setFocusedPanel, modalQueue, clearMessages, rightPanelMode } = useStore();
  const { selectPrevious, selectNext } = useAgentTree();
  const { rows: terminalHeight, columns: terminalWidth } = useTerminalDimensions();

  // Responsive layout breakpoints
  const isNarrow = terminalWidth < 100;
  const isVeryNarrow = terminalWidth < 80;
  const leftWidth = isVeryNarrow ? "100%" : isNarrow ? "75%" : "70%";
  const showRightPanel = !isVeryNarrow;

  // Calculate panel width based on responsive layout
  const claudePanelWidth = Math.floor(terminalWidth * (isVeryNarrow ? 1 : isNarrow ? 0.75 : 0.7)) - 4;

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

  // Agent tree navigation bindings (only when agents panel focused)
  const agentBindings: KeyBinding[] = [
    { key: "up", action: selectPrevious, description: "Select previous agent" },
    { key: "down", action: selectNext, description: "Select next agent" },
  ];

  // Only enable global bindings when no modal is active
  useKeymap(globalBindings, modalQueue.length === 0);

  // Only enable agent navigation when agents panel focused and no modal
  useKeymap(agentBindings, focusedPanel === "agents" && modalQueue.length === 0);

  return (
    <Box flexDirection="column" height={terminalHeight}>
      {/* Banner - FIXED at top */}
      <Box height={BANNER_HEIGHT}>
        <Banner />
      </Box>

      {/* Content area - FILLS remaining space */}
      <Box flexDirection="row" flexGrow={1}>
        {/* Left Panel: Claude conversation */}
        <Box width={leftWidth}>
          <ClaudePanel focused={focusedPanel === "claude"} width={claudePanelWidth} />
        </Box>

        {/* Right Panel: Conditional rendering based on mode */}
        {showRightPanel && (
          <Box width={isNarrow ? "25%" : "30%"} flexDirection="column">
            {rightPanelMode === "agents" && (
              <>
                {/* Agent Tree (60% via flexGrow) */}
                <Box
                  flexGrow={6}
                  borderStyle={borders.panel}
                  borderColor={focusedPanel === "agents" ? colors.focused : colors.unfocused}
                  flexDirection="column"
                  overflow="hidden"
                >
                  <AgentTree focused={focusedPanel === "agents"} />
                </Box>

                {/* Agent Detail (40% via flexGrow) */}
                <Box
                  flexGrow={4}
                  borderStyle={borders.panel}
                  borderColor={colors.muted}
                  flexDirection="column"
                  overflow="hidden"
                >
                  <AgentDetail focused={false} />
                </Box>
              </>
            )}
            {rightPanelMode === "dashboard" && (
              <Box
                flexGrow={1}
                borderStyle={borders.panel}
                borderColor={colors.muted}
                flexDirection="column"
                overflow="hidden"
              >
                <DashboardView />
              </Box>
            )}
            {rightPanelMode === "settings" && (
              <Box
                flexGrow={1}
                borderStyle={borders.panel}
                borderColor={colors.muted}
                flexDirection="column"
                overflow="hidden"
              >
                <SettingsView />
              </Box>
            )}
          </Box>
        )}
      </Box>

      {/* Status Line - FIXED at bottom */}
      <StatusLine width={terminalWidth} height={2} />

      {/* Toast notifications */}
      <ToastContainer />

      {/* Modal overlay (rendered when queue is non-empty) */}
      {modalQueue.length > 0 && modalQueue[0] && <ModalOverlay request={modalQueue[0]} />}
    </Box>
  );
}
