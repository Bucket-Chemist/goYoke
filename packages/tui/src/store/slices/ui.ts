/**
 * UI slice for Zustand store
 * Manages UI state (streaming, focus)
 */

import type { StateCreator } from "zustand";
import type { Store, UISlice } from "../types.js";

export const createUISlice: StateCreator<Store, [], [], UISlice> = (set) => ({
  streaming: false,
  focusedPanel: "claude",
  rightPanelMode: "agents",
  activeTab: "chat",
  interruptQuery: null,
  clearPendingMessage: null,

  setStreaming: (streaming): void => {
    set({ streaming });
  },

  setFocusedPanel: (panel): void => {
    set({ focusedPanel: panel });
  },

  cycleRightPanel: (): void => {
    set((state) => {
      const modes: Array<"agents" | "dashboard" | "settings" | "teams"> = ["agents", "dashboard", "teams", "settings"];
      const current = modes.indexOf(state.rightPanelMode);
      const next = (current + 1) % modes.length;
      return { rightPanelMode: modes[next]! };
    });
  },

  setActiveTab: (tab): void => {
    set({ activeTab: tab });
  },

  setInterruptQuery: (fn): void => {
    set({ interruptQuery: fn });
  },

  setClearPendingMessage: (fn): void => {
    set({ clearPendingMessage: fn });
  },
});
