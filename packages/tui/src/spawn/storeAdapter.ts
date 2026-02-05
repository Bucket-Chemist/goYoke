import { useStore } from "../store/index.js";
import type { AgentsStore } from "./relationshipValidation.js";

/**
 * Adapter to make Zustand store compatible with AgentsStore interface
 */
export function getAgentsStore(): AgentsStore {
  return {
    get: (id: string) => {
      const state = useStore.getState();
      return state.agents[id];
    },
    addChild: (parentId: string, childId: string) => {
      const state = useStore.getState();
      const parent = state.agents[parentId];
      if (parent) {
        const childIds = parent.childIds || [];
        if (!childIds.includes(childId)) {
          state.updateAgent(parentId, {
            childIds: [...childIds, childId],
          });
        }
      }
    },
    removeChild: (parentId: string, childId: string) => {
      const state = useStore.getState();
      const parent = state.agents[parentId];
      if (parent && parent.childIds) {
        state.updateAgent(parentId, {
          childIds: parent.childIds.filter(id => id !== childId),
        });
      }
    },
  };
}
