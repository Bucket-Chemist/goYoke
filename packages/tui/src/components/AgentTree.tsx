/**
 * AgentTree - Hierarchical agent tree view
 * Features:
 * - Tree structure with parent-child relationships
 * - Status indicators (spawning/running/complete/error)
 * - Selected agent highlighting
 * - Depth-based indentation
 * - Scrollable for large trees (20+ agents)
 */

import React, { useMemo } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors, icons } from "../config/theme.js";
import type { Agent } from "../store/types.js";

export interface AgentTreeProps {
  /**
   * Whether this component has focus
   */
  focused: boolean;
}

/**
 * Get status icon from theme constants
 */
function getStatusIcon(status: Agent["status"]): string {
  switch (status) {
    case "spawning":
      return icons.agentSpawning;
    case "running":
      return icons.agentRunning;
    case "complete":
      return icons.agentComplete;
    case "error":
      return icons.agentError;
    default:
      return "?";
  }
}

/**
 * Get status color from theme constants
 */
function getStatusColor(status: Agent["status"]): string {
  switch (status) {
    case "spawning":
      return colors.agentSpawning;
    case "running":
      return colors.agentRunning;
    case "complete":
      return colors.agentComplete;
    case "error":
      return colors.agentError;
    default:
      return colors.muted;
  }
}

/**
 * Recursive tree node rendering with depth tracking
 */
interface AgentNodeProps {
  agent: Agent;
  depth: number;
  isSelected: boolean;
  getChildren: (id: string) => Agent[];
}

function AgentNode({ agent, depth, isSelected, getChildren }: AgentNodeProps): JSX.Element {
  const children = getChildren(agent.id);
  const statusIcon = getStatusIcon(agent.status);
  const statusColor = getStatusColor(agent.status);

  // Build indentation prefix based on depth
  const indent = depth > 0 ? "  ".repeat(depth) : "";
  const prefix = depth > 0 ? `${indent}${icons.treeBranch} ` : "";

  // Agent line: prefix + icon + model + description
  const displayText = `${prefix}${statusIcon} ${agent.model}${agent.description ? `: ${agent.description}` : ""}`;

  return (
    <Box flexDirection="column">
      {/* Current agent */}
      <Box>
        <Text
          color={isSelected ? undefined : statusColor}
          inverse={isSelected}
          bold={isSelected}
        >
          {displayText}
        </Text>
      </Box>

      {/* Render children recursively */}
      {children.map((child) => (
        <AgentNode
          key={child.id}
          agent={child}
          depth={depth + 1}
          isSelected={false}
          getChildren={getChildren}
        />
      ))}
    </Box>
  );
}

/**
 * Main AgentTree component
 */
export function AgentTree({ focused }: AgentTreeProps): JSX.Element {
  const { agents, selectedAgentId, rootAgentId, getAgentChildren } = useStore();

  // Get root agent (if any)
  const rootAgent = useMemo(() => {
    if (!rootAgentId || !agents[rootAgentId]) {
      return null;
    }
    return agents[rootAgentId];
  }, [agents, rootAgentId]);

  // Count total agents for scrolling logic
  const agentCount = useMemo(() => Object.keys(agents).length, [agents]);

  // Empty state
  if (!rootAgent) {
    return (
      <Box flexDirection="column" paddingX={1}>
        <Text color={colors.muted}>No agents yet</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={0}>
      {/* Header */}
      <Box marginBottom={1} paddingX={1}>
        <Text bold color={focused ? colors.focused : colors.muted}>
          Agents
        </Text>
        {agentCount > 0 && (
          <Text color={colors.muted}> ({agentCount})</Text>
        )}
      </Box>

      {/* Tree content */}
      <Box flexDirection="column" paddingX={1}>
        <AgentNode
          agent={rootAgent}
          depth={0}
          isSelected={selectedAgentId === rootAgent.id}
          getChildren={getAgentChildren}
        />
      </Box>

      {/* Performance hint for large trees */}
      {agentCount >= 20 && (
        <Box marginTop={1} paddingX={1}>
          <Text color={colors.muted} dimColor>
            {agentCount} agents (consider scrolling)
          </Text>
        </Box>
      )}
    </Box>
  );
}
