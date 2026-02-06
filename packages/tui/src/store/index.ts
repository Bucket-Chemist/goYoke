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
import { createModalSlice } from "./slices/modal.js";
import { createTelemetrySlice } from "./slices/telemetry.js";
import { createToastSlice } from "./slices/toasts.js";

/**
 * Combined Zustand store with all slices
 * Persistence: Only messages, agents, and session slices are persisted
 * UI and input slices are ephemeral (reset on app restart)
 */

// Core store configuration (shared between dev and prod)
const storeConfig = (...a: Parameters<typeof createMessagesSlice>) => ({
  ...createMessagesSlice(...a),
  ...createAgentsSlice(...a),
  ...createSessionSlice(...a),
  ...createUISlice(...a),
  ...createInputSlice(...a),
  ...createModalSlice(...a),
  ...createTelemetrySlice(...a),
  ...createToastSlice(...a),
});

const persistConfig = {
  name: "tui-store",
  // Only persist messages, agents, and session
  // UI state, input history, and telemetry are ephemeral
  partialize: (state: Store) => ({
    messages: state.messages,
    agents: state.agents,
    selectedAgentId: state.selectedAgentId,
    rootAgentId: state.rootAgentId,
    sessionId: state.sessionId,
    totalCost: state.totalCost,
    tokenCount: state.tokenCount,
    // Exclude: streaming, focusedPanel, inputHistory, inputHistoryIndex, modalQueue
    // Exclude telemetry: routingDecisions, lastHandoff, sharpEdges, totalRoutingCost
  }),
};

// Conditionally apply devtools middleware only in development
// Type assertion needed due to conditional middleware application
// The middleware chain returns a compatible store creator despite TypeScript's inability to infer it
export const useStore = create<Store>()(
  (process.env["NODE_ENV"] === "development"
    ? devtools(persist(storeConfig, persistConfig), { name: "TUI Store" })
    : persist(storeConfig, persistConfig)) as unknown as typeof storeConfig
);

// Export types for convenience
export type { Store } from "./types.js";
export * from "./types.js";
