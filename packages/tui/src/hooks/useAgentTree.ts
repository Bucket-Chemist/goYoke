/**
 * useAgentTree - Hook for navigating agent tree with keyboard
 * Features:
 * - Flatten tree to navigable list (depth-first order)
 * - Provide selectPrevious() and selectNext() functions
 * - Handle navigation wrapping at boundaries
 * - Memoize flattened list for performance
 */

import { useMemo } from "react";
import { useStore } from "../store/index.js";
import type { Agent } from "../store/types.js";

export interface AgentTreeNavigation {
  /**
   * Flattened list of agent IDs in tree order (DFS)
   */
  agentIds: string[];

  /**
   * Current selection index in flattened list
   */
  currentIndex: number;

  /**
   * Select previous agent in tree (wraps to end)
   */
  selectPrevious: () => void;

  /**
   * Select next agent in tree (wraps to beginning)
   */
  selectNext: () => void;

  /**
   * Total number of agents
   */
  totalCount: number;
}

/**
 * Flatten agent tree to depth-first traversal order
 */
function flattenTree(
  agents: Record<string, Agent>,
  rootId: string | null,
  getChildren: (id: string) => Agent[]
): string[] {
  if (!rootId || !agents[rootId]) {
    return [];
  }

  const result: string[] = [];

  function traverse(agentId: string): void {
    result.push(agentId);
    const children = getChildren(agentId);
    // Sort children by startTime for consistent ordering
    children.sort((a, b) => a.startTime - b.startTime);
    children.forEach((child) => traverse(child.id));
  }

  traverse(rootId);
  return result;
}

/**
 * Hook for agent tree navigation
 */
export function useAgentTree(): AgentTreeNavigation {
  const { agents, rootAgentId, selectedAgentId, selectAgent, getAgentChildren } =
    useStore();

  // Flatten tree to navigable list (memoized)
  const agentIds = useMemo(() => {
    return flattenTree(agents, rootAgentId, getAgentChildren);
  }, [agents, rootAgentId, getAgentChildren]);

  // Find current selection index
  const currentIndex = useMemo(() => {
    if (!selectedAgentId) return -1;
    return agentIds.indexOf(selectedAgentId);
  }, [agentIds, selectedAgentId]);

  // Select previous agent (with wrapping)
  const selectPrevious = (): void => {
    if (agentIds.length === 0) return;

    if (currentIndex <= 0) {
      // Wrap to end
      const targetId = agentIds[agentIds.length - 1];
      if (targetId !== undefined) selectAgent(targetId);
    } else {
      const targetId = agentIds[currentIndex - 1];
      if (targetId !== undefined) selectAgent(targetId);
    }
  };

  // Select next agent (with wrapping)
  const selectNext = (): void => {
    if (agentIds.length === 0) return;

    if (currentIndex === -1 || currentIndex >= agentIds.length - 1) {
      // Wrap to beginning
      const targetId = agentIds[0];
      if (targetId !== undefined) selectAgent(targetId);
    } else {
      const targetId = agentIds[currentIndex + 1];
      if (targetId !== undefined) selectAgent(targetId);
    }
  };

  return {
    agentIds,
    currentIndex,
    selectPrevious,
    selectNext,
    totalCount: agentIds.length,
  };
}
