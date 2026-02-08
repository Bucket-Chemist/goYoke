/**
 * Teams slice for Zustand store
 * Manages background team orchestration state
 */

import type { StateCreator } from "zustand";
import type { Store, TeamsSlice } from "../types.js";

export const createTeamsSlice: StateCreator<Store, [], [], TeamsSlice> = (set) => ({
  backgroundTeamCount: 0,
  setBackgroundTeamCount: (count): void => {
    set({ backgroundTeamCount: count });
  },
});
