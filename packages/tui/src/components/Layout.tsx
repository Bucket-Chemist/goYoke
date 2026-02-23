import React, { useState, useEffect } from "react";
import { Box } from "ink";
import { useStore } from "../store/index.js";
import { useKeymap } from "../hooks/useKeymap.js";
import type { KeyBinding } from "../hooks/useKeymap.js";
import { useUnifiedNav } from "../hooks/useAgentTree.js";
import { useUnifiedTree } from "../hooks/useUnifiedTree.js";
import { useTerminalDimensions } from "../hooks/useTerminalDimensions.js";
import { useTeamsPoller } from "../hooks/useTeams.js";
import { useAgentSync } from "../hooks/useAgentSync.js";
import { getSessionManager } from "../session/index.js";
import { createGlobalBindings } from "../config/keybindings.js";
import { Banner } from "./Banner.js";
import { TabBar } from "./TabBar.js";
import { ClaudePanel } from "./ClaudePanel.js";
import { UnifiedTree } from "./UnifiedTree.js";
import { UnifiedDetail } from "./UnifiedDetail.js";
import { DashboardView } from "./DashboardView.js";
import { SettingsView } from "./SettingsView.js";
import { AgentConfigView } from "./AgentConfigView.js";
import { TeamConfigView } from "./TeamConfigView.js";
import { TelemetryView } from "./TelemetryView.js";
import { ModalOverlay } from "./Modal.js";
import type { AskPayload, ModalRequest } from "../store/slices/modal.js";

/** Plan approval modals need a split layout so the conversation remains readable. */
function isPlanApprovalModal(request: ModalRequest): boolean {
  if (request.type !== "ask") return false;
  return (request.payload as AskPayload).header === "Plan";
}
import { StatusLine } from "./StatusLine.js";
import { ToastContainer } from "./Toast.js";
import { TaskBoard } from "./TaskBoard.js";
import { colors, borders } from "../config/theme.js";

// Fixed heights
const BANNER_HEIGHT = 3; // Banner takes 3 rows
const TAB_BAR_HEIGHT = 1; // TabBar takes 1 row

/**
 * Layout component - main 2-panel split (70/30) with focus management and tabbed navigation
 *
 * Features:
 * - Tab navigation: Chat, Agent Config, Team Config, Telemetry
 * - Left panel (70%): Claude conversation (chat tab only)
 * - Right panel (30%): Agent tree (60%) + Agent detail (40%) (chat tab only)
 * - Global keyboard bindings (Tab, Escape, Ctrl+C, Ctrl+L)
 * - Tab shortcuts (Alt+c, Alt+a, Alt+t, Alt+y)
 * - Focus indicated by border color (cyan/gray from theme)
 * - Modal overlay renders when queue is non-empty
 * - Modal captures all input when active
 */
export function Layout(): JSX.Element {
  const { focusedPanel, setFocusedPanel, modalQueue, clearMessages, rightPanelMode, streaming, interruptQuery, clearPendingMessage, activeTab, cycleRightPanel } = useStore();
  const { nodes, selectedNode, selectNode } = useUnifiedTree();
  const { selectPrevious, selectNext } = useUnifiedNav(nodes, selectedNode, selectNode);
  const { rows: terminalHeight, columns: terminalWidth } = useTerminalDimensions();
  const [taskBoardTab, setTaskBoardTab] = useState<"active" | "done">("active");

  // Start team polling unconditionally - this fixes the circular dependency
  // where TeamList only renders when rightPanelMode === "teams", but polling
  // was only happening inside useTeams() which was only called by TeamList
  useTeamsPoller();
  useAgentSync();

  // Responsive layout breakpoints
  const isNarrow = terminalWidth < 100;
  const isVeryNarrow = terminalWidth < 80;
  const leftWidth = isVeryNarrow ? "100%" : isNarrow ? "75%" : "70%";
  const showRightPanel = !isVeryNarrow;

  // Calculate panel width based on responsive layout
  const claudePanelWidth = Math.floor(terminalWidth * (isVeryNarrow ? 1 : isNarrow ? 0.75 : 0.7)) - 4;

  // Auto-select the first active agent when nothing is selected yet.
  // Hysteresis: only fires when selectedNode is null — never overrides a manual selection.
  useEffect(() => {
    if (selectedNode !== null) return;
    if (nodes.length === 0) return;

    const activeNode = nodes.find(
      (n) => n.status === "running" || n.status === "streaming"
    );
    const target = activeNode ?? nodes[0];
    if (target) selectNode(target.id);
  }, [nodes, selectedNode, selectNode]);

  // Global key bindings (only active when no modal is present)
  const globalBindings = createGlobalBindings({
    toggleFocus: () => {
      setFocusedPanel(focusedPanel === "claude" ? "agents" : "claude");
    },
    cycleRightPanel: () => {
      cycleRightPanel();
    },
    cyclePermissionMode: () => {
      const sessionManager = getSessionManager();
      void sessionManager.cyclePermissionMode();
    },
    interruptQuery: () => {
      // If modal is active, cancel it
      if (modalQueue.length > 0 && modalQueue[0]) {
        const modal = modalQueue[0];
        useStore.getState().cancel(modal.id);
      } else if (streaming && interruptQuery) {
        // If streaming, interrupt the query and clear any queued message
        void interruptQuery();
        clearPendingMessage?.();
      }
      // Otherwise do nothing
    },
    forceQuit: () => {
      process.exit(0);
    },
    clearScreen: () => {
      clearMessages();
    },
    toggleTaskBoardTab: () => {
      setTaskBoardTab((prev) => (prev === "active" ? "done" : "active"));
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

      {/* TabBar - FIXED below banner */}
      <Box height={TAB_BAR_HEIGHT} paddingX={1}>
        <TabBar enabled={modalQueue.length === 0} />
      </Box>

      {/* Content area: modal handling.
          Plan approval: split layout so conversation stays readable above the modal.
          All other modals: full-screen replacement (no bleed-through). */}
      {modalQueue.length > 0 && modalQueue[0] ? (
        isPlanApprovalModal(modalQueue[0]) ? (
          <Box flexGrow={1} flexDirection="column">
            {/* Conversation stays visible so the plan is readable */}
            <Box flexGrow={1} overflow="hidden">
              <ClaudePanel focused={false} width={claudePanelWidth} />
            </Box>
            {/* Compact approval strip at the bottom */}
            <ModalOverlay request={modalQueue[0]} compact />
          </Box>
        ) : (
          <ModalOverlay request={modalQueue[0]} />
        )
      ) : (
        <>
          <Box flexDirection="row" flexGrow={1}>
            {activeTab === "chat" && (
              <>
                {/* Left Panel: Claude conversation */}
                <Box width={leftWidth} flexDirection="column">
                  <ClaudePanel focused={focusedPanel === "claude"} width={claudePanelWidth} />
                </Box>

                {/* Right Panel: Conditional rendering based on mode */}
                {showRightPanel && (
                  <Box width={isNarrow ? "25%" : "30%"} flexDirection="column">
                    {rightPanelMode === "agents" && (
                      <>
                        {/* Unified Tree (60% via flexGrow) */}
                        <Box
                          flexGrow={6}
                          borderStyle={borders.panel}
                          borderColor={focusedPanel === "agents" ? colors.focused : colors.unfocused}
                          flexDirection="column"
                          overflow="hidden"
                        >
                          <UnifiedTree focused={focusedPanel === "agents"} nodes={nodes} selectedNode={selectedNode} />
                        </Box>

                        {/* Unified Detail (40% via flexGrow) */}
                        <Box
                          flexGrow={4}
                          borderStyle={borders.panel}
                          borderColor={colors.muted}
                          flexDirection="column"
                          overflow="hidden"
                        >
                          <UnifiedDetail focused={false} selectedNode={selectedNode} />
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
              </>
            )}

            {activeTab === "agent-config" && (
              <Box flexGrow={1}>
                <AgentConfigView />
              </Box>
            )}

            {activeTab === "team-config" && (
              <Box flexGrow={1}>
                <TeamConfigView />
              </Box>
            )}

            {activeTab === "telemetry" && (
              <Box flexGrow={1}>
                <TelemetryView />
              </Box>
            )}
          </Box>

          {/* TaskBoard - compact strip above status line, chat tab only */}
          {activeTab === "chat" && (
            <Box height={8} borderStyle={borders.panel} borderColor={colors.unfocused}>
              <TaskBoard width={terminalWidth - 2} tab={taskBoardTab} />
            </Box>
          )}
        </>
      )}

      {/* Status Line - FIXED at bottom */}
      <StatusLine width={terminalWidth} height={2} />

      {/* Toast notifications */}
      <ToastContainer />
    </Box>
  );
}
