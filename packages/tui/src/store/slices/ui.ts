/**
 * UI slice for Zustand store
 * Manages UI state (streaming, focus)
 */

import type { StateCreator } from "zustand";
import type { Store, UISlice } from "../types.js";

export const createUISlice: StateCreator<Store, [], [], UISlice> = (set) => ({
  streaming: false,
  focusedPanel: "claude",

  setStreaming: (streaming): void => {
    set({ streaming });
  },

  setFocusedPanel: (panel): void => {
    set({ focusedPanel: panel });
  },
});
