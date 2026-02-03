/**
 * Agents slice for Zustand store
 * Manages agent tree with parent/child relationships
 */

import type { StateCreator } from "zustand";
import type { Store, AgentsSlice, Agent } from "../types.js";

export const createAgentsSlice: StateCreator<Store, [], [], AgentsSlice> = (
  set,
  get
) => ({
  agents: new Map(),
  selectedAgentId: null,
  rootAgentId: null,

  addAgent: (agent): void => {
    set((state) => {
      const newAgent: Agent = {
        ...agent,
        startTime: Date.now(),
      };

      const agents = new Map(state.agents);
      agents.set(newAgent.id, newAgent);

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
      const agent = state.agents.get(id);
      if (!agent) {
        return state;
      }

      const agents = new Map(state.agents);
      agents.set(id, { ...agent, ...data });

      return { agents };
    });
  },

  selectAgent: (id): void => {
    set({ selectedAgentId: id });
  },

  getAgentChildren: (id): Agent[] => {
    const agents = get().agents;
    const children: Agent[] = [];

    agents.forEach((agent) => {
      if (agent.parentId === id) {
        children.push(agent);
      }
    });

    return children;
  },

  clearAgents: (): void => {
    set({
      agents: new Map(),
      selectedAgentId: null,
      rootAgentId: null,
    });
  },
});
