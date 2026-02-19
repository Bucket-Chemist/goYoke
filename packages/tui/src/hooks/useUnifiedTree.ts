/**
 * useUnifiedTree - Derives a flat UnifiedNode[] from agent and team store slices.
 * View-layer projection only — never stored in Zustand.
 */

import { useMemo, useCallback } from "react";
import { useStore } from "../store/index.js";
import type { UnifiedNode, AgentStatus } from "../store/types.js";

export interface UnifiedTreeResult {
  nodes: UnifiedNode[];
  selectedNode: UnifiedNode | null;
  selectNode: (id: string) => void;
}

export function useUnifiedTree(): UnifiedTreeResult {
  const agents = useStore((s) => s.agents);
  const rootAgentId = useStore((s) => s.rootAgentId);
  const getAgentChildren = useStore((s) => s.getAgentChildren);
  const selectedAgentId = useStore((s) => s.selectedAgentId);
  const selectAgent = useStore((s) => s.selectAgent);
  const teams = useStore((s) => s.teams);
  const selectedUnifiedId = useStore((s) => s.selectedUnifiedId);
  const setSelectedUnifiedId = useStore((s) => s.setSelectedUnifiedId);

  const nodes = useMemo(() => {
    const result: UnifiedNode[] = [];

    // 1. SDK agents section: DFS from rootAgentId
    if (rootAgentId && agents[rootAgentId]) {
      const traverseAgents = (agentId: string, depth: number): void => {
        const agent = agents[agentId];
        if (!agent) return;
        result.push({
          kind: "sdk-agent",
          id: `agent:${agent.id}`,
          parentId: agent.parentId ? `agent:${agent.parentId}` : null,
          displayName: agent.description || agent.model,
          model: agent.model,
          status: agent.status,
          startTime: agent.startTime,
          endTime: agent.endTime,
          depth,
          agentRef: agent.id,
          cost: agent.cost,
        });
        const children = getAgentChildren(agentId);
        children.sort((a, b) => a.startTime - b.startTime);
        for (const child of children) {
          traverseAgents(child.id, depth + 1);
        }
      };
      traverseAgents(rootAgentId, 0);
    }

    // 2. Teams section: each TeamSummary becomes a team-root, members become team-member children
    for (const team of teams) {
      const teamStatus: AgentStatus =
        team.status === "completed" ? "complete" :
        team.status === "failed" ? "error" :
        team.status === "running" ? "running" : "queued";

      result.push({
        kind: "team-root",
        id: `team:${team.dir}`,
        parentId: null,
        displayName: team.name,
        model: team.workflowType,
        status: teamStatus,
        startTime: team.startedAt ? new Date(team.startedAt).getTime() : Date.now(),
        endTime: team.completedAt ? new Date(team.completedAt).getTime() : undefined,
        depth: 0,
        teamDir: team.dir,
        cost: team.totalCost,
      });

      for (const member of team.members) {
        const memberStatus: AgentStatus =
          member.status === "completed" ? "complete" :
          member.status === "failed" ? "error" :
          member.status === "running" ? "running" : "queued";

        result.push({
          kind: "team-member",
          id: `member:${team.dir}:${member.name}`,
          parentId: `team:${team.dir}`,
          displayName: member.name,
          model: member.model,
          status: memberStatus,
          startTime: member.startedAt ? new Date(member.startedAt).getTime() : Date.now(),
          endTime: member.completedAt ? new Date(member.completedAt).getTime() : undefined,
          depth: 1,
          teamDir: team.dir,
          waveNumber: member.wave,
          latestActivity: member.latestActivity,
          healthStatus: member.healthStatus,
          cost: member.cost,
        });
      }
    }

    return result;
  }, [agents, rootAgentId, getAgentChildren, teams]);

  const selectedNode = useMemo(() => {
    if (selectedUnifiedId) {
      return nodes.find((n) => n.id === selectedUnifiedId) ?? null;
    }
    if (selectedAgentId) {
      return nodes.find((n) => n.id === `agent:${selectedAgentId}`) ?? null;
    }
    return null;
  }, [nodes, selectedUnifiedId, selectedAgentId]);

  const selectNode = useCallback((id: string) => {
    const node = nodes.find((n) => n.id === id);
    if (!node) return;

    if (node.kind === "sdk-agent" && node.agentRef) {
      selectAgent(node.agentRef);
      setSelectedUnifiedId(null);
    } else {
      selectAgent(null);
      setSelectedUnifiedId(id);
    }
  }, [nodes, selectAgent, setSelectedUnifiedId]);

  return { nodes, selectedNode, selectNode };
}
