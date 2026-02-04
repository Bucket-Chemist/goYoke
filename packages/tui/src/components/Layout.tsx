import React from "react";
import { Box, useStdout } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap, KeyBinding } from "../hooks/useKeymap.js";
import { useAgentTree } from "../hooks/useAgentTree.js";
import { createGlobalBindings } from "../config/keybindings.js";
import { Banner } from "./Banner.js";
import { ClaudePanel } from "./ClaudePanel.js";
import { AgentTree } from "./AgentTree.js";
import { AgentDetail } from "./AgentDetail.js";
import { ModalOverlay } from "./Modal.js";
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
  const { focusedPanel, setFocusedPanel, modalQueue, clearMessages } = useStore();
  const { selectPrevious, selectNext } = useAgentTree();
  const { stdout } = useStdout();

  // Calculate available height for content area
  const terminalHeight = stdout?.rows ?? 24;
  const contentHeight = terminalHeight - BANNER_HEIGHT;

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

  // Calculate right panel heights (fixed row counts)
  const agentTreeHeight = Math.floor(contentHeight * 0.6);
  const agentDetailHeight = contentHeight - agentTreeHeight;

  return (
    <Box flexDirection="column" height={terminalHeight}>
      {/* Banner - FIXED at top */}
      <Box height={BANNER_HEIGHT}>
        <Banner />
      </Box>

      {/* Content area - FIXED height */}
      <Box flexDirection="row" height={contentHeight}>
        {/* Left Panel: Claude conversation (70%) */}
        <Box width="70%" height={contentHeight}>
          <ClaudePanel focused={focusedPanel === "claude"} maxHeight={contentHeight - 2} />
        </Box>

        {/* Right Panel: Agent tree + detail (30%) */}
        <Box width="30%" flexDirection="column" height={contentHeight}>
          {/* Agent Tree (60% of right panel) */}
          <Box
            height={agentTreeHeight}
            borderStyle={borders.panel}
            borderColor={focusedPanel === "agents" ? colors.focused : colors.unfocused}
            flexDirection="column"
          >
            <AgentTree focused={focusedPanel === "agents"} />
          </Box>

          {/* Agent Detail (40% of right panel) */}
          <Box
            height={agentDetailHeight}
            borderStyle={borders.panel}
            borderColor={colors.muted}
            flexDirection="column"
          >
            <AgentDetail focused={false} />
          </Box>
        </Box>
      </Box>

      {/* Modal overlay (rendered when queue is non-empty) */}
      {modalQueue.length > 0 && modalQueue[0] && <ModalOverlay request={modalQueue[0]} />}
    </Box>
  );
}
