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
  panelAutoSwitched: false,
  selectedUnifiedId: null,

  setStreaming: (streaming): void => {
    set({ streaming });
  },

  setFocusedPanel: (panel): void => {
    set({ focusedPanel: panel });
  },

  cycleRightPanel: (): void => {
    set((state) => {
      const modes: Array<"agents" | "dashboard" | "settings"> = ["agents", "dashboard", "settings"];
      const current = modes.indexOf(state.rightPanelMode);
      const next = (current + 1) % modes.length;
      return { rightPanelMode: modes[next]!, panelAutoSwitched: false };
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

  setPanelAutoSwitched: (switched): void => {
    set({ panelAutoSwitched: switched });
  },

  setRightPanelMode: (mode): void => {
    set({ rightPanelMode: mode });
  },

  setSelectedUnifiedId: (id): void => {
    set({ selectedUnifiedId: id });
  },
});
