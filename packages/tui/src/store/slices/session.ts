/**
 * Session slice for Zustand store
 * Matches Go CLI session format for rollback compatibility
 */

import type { StateCreator } from "zustand";
import type { Store, SessionSlice } from "../types.js";

export const createSessionSlice: StateCreator<Store, [], [], SessionSlice> = (
  set,
  get
) => ({
  sessionId: null,
  totalCost: 0,
  tokenCount: {
    input: 0,
    output: 0,
  },
  contextWindow: {
    usedTokens: 0,
    totalCapacity: 200000,
  },
  permissionMode: "default",
  isCompacting: false,
  preferredModel: null,
  activeModel: null,

  updateSession: (data): void => {
    set((state) => ({
      sessionId: data.id ?? state.sessionId,
      totalCost: data.cost ?? state.totalCost,
      // Note: Go format uses tool_calls, we track internally as tokenCount
    }));
  },

  incrementCost: (cost): void => {
    set((state) => ({
      totalCost: state.totalCost + cost,
    }));
  },

  addTokens: (tokens): void => {
    set((state) => ({
      tokenCount: {
        input: state.tokenCount.input + (tokens.input ?? 0),
        output: state.tokenCount.output + (tokens.output ?? 0),
      },
    }));
  },

  updateContextWindow: (usedTokens, totalCapacity): void => {
    set({
      contextWindow: {
        usedTokens,
        totalCapacity,
      },
    });
  },

  setPermissionMode: (mode): void => {
    set({ permissionMode: mode });
  },

  setCompacting: (compacting): void => {
    set({ isCompacting: compacting });
  },

  setPreferredModel: (model): void => {
    set({ preferredModel: model });
  },

  setActiveModel: (model): void => {
    set({ activeModel: model });
  },

  // Computed property: check if currently in plan mode
  isPlanMode: (): boolean => {
    return get().permissionMode === "plan";
  },

  clearSession: (): void => {
    set({
      sessionId: null,
      totalCost: 0,
      tokenCount: {
        input: 0,
        output: 0,
      },
      contextWindow: {
        usedTokens: 0,
        totalCapacity: 200000,
      },
      permissionMode: "default",
      isCompacting: false,
      preferredModel: null,
      activeModel: null,
    });
  },
});
