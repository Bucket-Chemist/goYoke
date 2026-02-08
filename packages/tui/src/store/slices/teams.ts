/**
 * Teams slice for Zustand store
 * Manages background team orchestration state
 */

import type { StateCreator } from "zustand";
import type { Store, TeamsSlice, TeamSummary, TeamConfig } from "../types.js";

export const createTeamsSlice: StateCreator<Store, [], [], TeamsSlice> = (set) => ({
  teams: [],
  selectedTeamDir: null,
  selectedTeamDetail: null,
  /** @deprecated Use teams.filter(t => t.alive).length instead */
  backgroundTeamCount: 0,

  setTeams: (teams: TeamSummary[]): void => {
    set({
      teams,
      // Update derived count for backward compatibility
      backgroundTeamCount: teams.filter((t) => t.alive).length,
    });
  },

  selectTeam: (dir: string | null): void => {
    set({ selectedTeamDir: dir });
  },

  setTeamDetail: (config: TeamConfig | null): void => {
    set({ selectedTeamDetail: config });
  },

  /** @deprecated Use teams.filter(t => t.alive).length instead */
  setBackgroundTeamCount: (count: number): void => {
    set({ backgroundTeamCount: count });
  },
});
