/**
 * Session slice for Zustand store
 * Matches Go CLI session format for rollback compatibility
 */

import type { StateCreator } from "zustand";
import type { Store, SessionSlice, SessionData } from "../types.js";

export const createSessionSlice: StateCreator<Store, [], [], SessionSlice> = (
  set
) => ({
  sessionId: null,
  totalCost: 0,
  tokenCount: {
    input: 0,
    output: 0,
  },

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

  clearSession: (): void => {
    set({
      sessionId: null,
      totalCost: 0,
      tokenCount: {
        input: 0,
        output: 0,
      },
    });
  },
});
