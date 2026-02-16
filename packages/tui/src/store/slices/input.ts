/**
 * Input history slice for Zustand store
 * Persisted to ~/.claude/input-history.json for cross-session recall
 * GAP-2 resolution: shell-like input recall
 */

import { readFileSync, writeFileSync, mkdirSync } from "fs";
import { join } from "path";
import { homedir } from "os";
import type { StateCreator } from "zustand";
import type { Store, InputSlice } from "../types.js";

const MAX_HISTORY_SIZE = 100;
const HISTORY_FILE = join(homedir(), ".claude", "input-history.json");

/** Load history from disk (best-effort, returns [] on failure) */
function loadHistory(): string[] {
  try {
    const data = readFileSync(HISTORY_FILE, "utf-8");
    const parsed = JSON.parse(data);
    if (Array.isArray(parsed)) {
      return parsed.filter((item): item is string => typeof item === "string").slice(0, MAX_HISTORY_SIZE);
    }
  } catch {
    // File doesn't exist or is corrupt — start fresh
  }
  return [];
}

/** Save history to disk (best-effort, fire-and-forget) */
function saveHistory(history: string[]): void {
  try {
    mkdirSync(join(homedir(), ".claude"), { recursive: true });
    writeFileSync(HISTORY_FILE, JSON.stringify(history), "utf-8");
  } catch {
    // Best-effort — don't crash on write failure
  }
}

export const createInputSlice: StateCreator<Store, [], [], InputSlice> = (
  set,
  get
) => ({
  inputHistory: loadHistory(),
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

      // Persist to disk
      saveHistory(newHistory);

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
