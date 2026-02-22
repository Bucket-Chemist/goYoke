/**
 * Agents slice for Zustand store
 * Manages agent tree with parent/child relationships
 */

import type { StateCreator } from "zustand";
import type { Store, AgentsSlice, Agent, AgentActivity } from "../types.js";

export const createAgentsSlice: StateCreator<Store, [], [], AgentsSlice> = (
  set,
  get
) => ({
  agents: {},
  selectedAgentId: null,
  rootAgentId: null,

  addAgent: (agent): void => {
    set((state) => {
      const newAgent: Agent = {
        ...agent,
        startTime: Date.now(),
      };

      const agents = { ...state.agents };
      agents[newAgent.id] = newAgent;

      // Track root agent (first agent with no parent)
      const rootAgentId =
        state.rootAgentId || (agent.parentId === null ? newAgent.id : null);

      return {
        agents,
        rootAgentId,
      };
    });
  },

  updateAgent: (id, data): void => {
    set((state) => {
      const agent = state.agents[id];
      if (!agent) {
        return state;
      }

      const agents = { ...state.agents };
      agents[id] = { ...agent, ...data };

      return { agents };
    });
  },

  updateAgentActivity: (id, activity): void => {
    set((state) => {
      const agent = state.agents[id];
      if (!agent) return state;
      return {
        agents: { ...state.agents, [id]: { ...agent, activity } },
      };
    });
  },

  selectAgent: (id): void => {
    set({ selectedAgentId: id });
  },

  getAgentChildren: (id): Agent[] => {
    const agents = get().agents;

    // Null safety: return empty array if parent doesn't exist
    if (!(id in agents)) {
      return [];
    }

    const children: Agent[] = [];

    Object.values(agents).forEach((agent) => {
      if (agent.parentId === id) {
        children.push(agent);
      }
    });

    return children;
  },

  clearAgents: (): void => {
    set({
      agents: {},
      selectedAgentId: null,
      rootAgentId: null,
    });
  },
});
