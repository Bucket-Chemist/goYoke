/**
 * Input history slice for Zustand store
 * Ephemeral state (not persisted) for up/down arrow navigation
 * GAP-2 resolution: shell-like input recall
 */

import type { StateCreator } from "zustand";
import type { Store, InputSlice } from "../types.js";

const MAX_HISTORY_SIZE = 100;

export const createInputSlice: StateCreator<Store, [], [], InputSlice> = (
  set,
  get
) => ({
  inputHistory: [],
  inputHistoryIndex: -1,

  addToHistory: (input: string): void => {
    if (!input.trim()) {
      return; // Don't store empty inputs
    }

    set((state) => {
      const history = state.inputHistory;

      // Deduplicate: remove if already exists
      const filtered = history.filter((item) => item !== input);

      // Add to front, limit to MAX_HISTORY_SIZE
      const newHistory = [input, ...filtered].slice(0, MAX_HISTORY_SIZE);

      return {
        inputHistory: newHistory,
        inputHistoryIndex: -1, // Reset navigation
      };
    });
  },

  navigateHistory: (direction: "up" | "down"): string | null => {
    const state = get();
    const { inputHistory, inputHistoryIndex } = state;

    if (inputHistory.length === 0) {
      return null;
    }

    let newIndex = inputHistoryIndex;

    if (direction === "up") {
      // Navigate backward in time (newer to older)
      newIndex = Math.min(inputHistoryIndex + 1, inputHistory.length - 1);
    } else {
      // Navigate forward in time (older to newer)
      newIndex = inputHistoryIndex - 1;
    }

    // Update index
    set({ inputHistoryIndex: newIndex });

    // Return the history item, or null if navigated past the beginning
    return newIndex >= 0 ? inputHistory[newIndex] ?? null : null;
  },

  resetHistoryIndex: (): void => {
    set({ inputHistoryIndex: -1 });
  },

  clearHistory: (): void => {
    set({
      inputHistory: [],
      inputHistoryIndex: -1,
    });
  },
});
