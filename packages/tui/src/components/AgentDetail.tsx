/**
 * AgentDetail - Detail panel showing selected agent information
 * Features:
 * - Displays model, tier, status, duration, token usage
 * - Formatted duration (ms/s/m)
 * - Empty state when no agent selected
 * - Uses theme constants for colors
 */

import React, { useMemo } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";
import type { Agent } from "../store/types.js";

export interface AgentDetailProps {
  /**
   * Whether this component has focus
   */
  focused: boolean;
}

/**
 * Format duration in human-readable format
 * - <1s: display in milliseconds
 * - <1m: display in seconds
 * - >=1m: display in minutes and seconds
 */
function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
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
 * Format token count with thousands separator
 */
function formatTokens(count: number): string {
  return count.toLocaleString();
}

/**
 * Main AgentDetail component
 */
export function AgentDetail({ focused }: AgentDetailProps): JSX.Element {
  const { agents, selectedAgentId } = useStore();

  // Get selected agent
  const selectedAgent = useMemo(() => {
    if (!selectedAgentId || !agents[selectedAgentId]) {
      return null;
    }
    return agents[selectedAgentId];
  }, [agents, selectedAgentId]);

  // Calculate duration
  const duration = useMemo(() => {
    if (!selectedAgent) return null;
    const endTime = selectedAgent.endTime || Date.now();
    return endTime - selectedAgent.startTime;
  }, [selectedAgent]);

  // Empty state
  if (!selectedAgent) {
    return (
      <Box flexDirection="column" paddingX={1}>
        <Box marginBottom={1}>
          <Text bold color={focused ? colors.focused : colors.muted}>
            Agent Detail
          </Text>
        </Box>
        <Text color={colors.muted}>Select an agent to view details</Text>
      </Box>
    );
  }

  const statusColor = getStatusColor(selectedAgent.status);

  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color={focused ? colors.focused : colors.muted}>
          Agent Detail
        </Text>
      </Box>

      {/* Model */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Model: </Text>
        <Text color={colors.primary}>{selectedAgent.model}</Text>
      </Box>

      {/* Tier */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Tier: </Text>
        <Text color={colors.secondary}>{selectedAgent.tier}</Text>
      </Box>

      {/* Status */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Status: </Text>
        <Text color={statusColor} bold>
          {selectedAgent.status}
        </Text>
      </Box>

      {/* Duration */}
      {duration !== null && (
        <Box marginBottom={0}>
          <Text color={colors.muted}>Duration: </Text>
          <Text>{formatDuration(duration)}</Text>
        </Box>
      )}

      {/* Token usage (if available) */}
      {selectedAgent.tokenUsage && (
        <>
          <Box marginBottom={0}>
            <Text color={colors.muted}>Input tokens: </Text>
            <Text>{formatTokens(selectedAgent.tokenUsage.input)}</Text>
          </Box>
          <Box marginBottom={0}>
            <Text color={colors.muted}>Output tokens: </Text>
            <Text>{formatTokens(selectedAgent.tokenUsage.output)}</Text>
          </Box>
          <Box marginBottom={0}>
            <Text color={colors.muted}>Total tokens: </Text>
            <Text bold>
              {formatTokens(
                selectedAgent.tokenUsage.input + selectedAgent.tokenUsage.output
              )}
            </Text>
          </Box>
        </>
      )}

      {/* Description (if available) */}
      {selectedAgent.description && (
        <Box marginTop={1} flexDirection="column">
          <Text color={colors.muted}>Description:</Text>
          <Text>{selectedAgent.description}</Text>
        </Box>
      )}
    </Box>
  );
}
