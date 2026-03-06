/**
 * UI slice for Zustand store
 * Manages UI state (streaming, focus)
 */

import type { StateCreator } from "zustand";
import type { Store, UISlice } from "../types.js";
import { generateHandoff } from "../../utils/handoffGenerator.js";

// Debounce timer for provider switching
let switchDebounceTimer: NodeJS.Timeout | null = null;

export const createUISlice: StateCreator<Store, [], [], UISlice> = (set, get) => ({
  streaming: false,
  focusedPanel: "claude",
  rightPanelMode: "agents",
  activeTab: "chat",
  activeProvider: "anthropic",
  interruptQuery: null,
  clearPendingMessage: null,
  panelAutoSwitched: false,
  selectedUnifiedId: null,
  planPreviewContent: null,
  planPreviewPath: null,
  currentPlanFile: null,
  previousRightPanelMode: "agents" as const,

  setStreaming: (streaming): void => {
    set({ streaming });
  },

  setActiveProvider: (provider): void => {
    // Clear any pending debounce timer
    if (switchDebounceTimer) {
      clearTimeout(switchDebounceTimer);
      switchDebounceTimer = null;
    }

    // Debounce rapid switches (500ms)
    switchDebounceTimer = setTimeout(() => {
      const state = get();
      const fromProvider = state.activeProvider;
      const oldMessages = state.providerMessages[fromProvider] ?? [];

      // Switch provider immediately (non-blocking)
      set({ activeProvider: provider });

      // Skip handoff if:
      // - Switching to same provider (no-op)
      // - Less than 3 messages in conversation
      // - Handoff disabled in settings (if implemented)
      if (fromProvider === provider || oldMessages.length < 3) {
        return;
      }

      // Generate handoff asynchronously (fire-and-forget)
      void generateHandoff(oldMessages, fromProvider, provider).then(
        (summary) => {
          if (summary) {
            // Inject handoff as system message on new provider
            get().injectHandoffMessage(provider, summary, fromProvider);
          }
        }
      );
    }, 500);
  },

  setFocusedPanel: (panel): void => {
    set({ focusedPanel: panel });
  },

  cycleRightPanel: (): void => {
    set((state) => {
      const modes: Array<"agents" | "dashboard" | "settings"> = ["agents", "dashboard", "settings"];
      const current = modes.indexOf(state.rightPanelMode as typeof modes[number]);
      const next = (current + 1) % modes.length;
      // Clean up plan preview state if escaping planPreview mode
      const cleanup = state.rightPanelMode === "planPreview"
        ? { planPreviewContent: null, planPreviewPath: null, currentPlanFile: null }
        : {};
      return { rightPanelMode: modes[next]!, panelAutoSwitched: false, ...cleanup };
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

  setPlanPreview: (content, path): void => {
    set({ planPreviewContent: content, planPreviewPath: path });
  },

  setCurrentPlanFile: (path): void => {
    set({ currentPlanFile: path });
  },

  setPreviousRightPanelMode: (mode): void => {
    set({ previousRightPanelMode: mode });
  },
});
