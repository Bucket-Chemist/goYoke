/**
 * Zustand store - combines all slices with devtools middleware
 * Central state management for TUI application
 */

import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import type { Store } from "./types.js";
import { createMessagesSlice } from "./slices/messages.js";
import { createAgentsSlice } from "./slices/agents.js";
import { createSessionSlice } from "./slices/session.js";
import { createUISlice } from "./slices/ui.js";
import { createInputSlice } from "./slices/input.js";

/**
 * Combined Zustand store with all slices
 * Persistence: Only messages, agents, and session slices are persisted
 * UI and input slices are ephemeral (reset on app restart)
 */
export const useStore = create<Store>()(
  devtools(
    persist(
      (...a) => ({
        ...createMessagesSlice(...a),
        ...createAgentsSlice(...a),
        ...createSessionSlice(...a),
        ...createUISlice(...a),
        ...createInputSlice(...a),
      }),
      {
        name: "tui-store",
        // Only persist messages, agents, and session
        // UI state and input history are ephemeral
        partialize: (state) => ({
          messages: state.messages,
          agents: state.agents,
          selectedAgentId: state.selectedAgentId,
          rootAgentId: state.rootAgentId,
          sessionId: state.sessionId,
          totalCost: state.totalCost,
          tokenCount: state.tokenCount,
          // Exclude: streaming, focusedPanel, inputHistory, inputHistoryIndex
        }),
      }
    ),
    { name: "TUI Store" }
  )
);

// Export types for convenience
export type { Store } from "./types.js";
export * from "./types.js";
