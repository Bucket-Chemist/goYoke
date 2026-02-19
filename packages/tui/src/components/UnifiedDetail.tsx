/**
 * UnifiedDetail - Polymorphic detail panel for UnifiedTree nodes.
 * Renders different detail views based on the selected node's `kind`:
 *   - "sdk-agent"   → Agent detail (model, tier, status, tokens, cost)
 *   - "team-root"   → Team summary (budget, progress, waves)
 *   - "team-member" → Member detail (wave, status, cost, activity)
 */

import React, { useMemo } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";
import type { UnifiedNode, AgentStatus } from "../store/types.js";
import {
  formatDuration,
  formatElapsed,
  getTeamStatusColor,
} from "../utils/teamFormatting.js";

export interface UnifiedDetailProps {
  focused: boolean;
  selectedNode: UnifiedNode | null;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getStatusColor(status: AgentStatus): string {
  switch (status) {
    case "spawning":
      return colors.agentSpawning;
    case "running":
    case "streaming":
      return colors.agentRunning;
    case "complete":
      return colors.agentComplete;
    case "error":
      return colors.agentError;
    default:
      return colors.muted;
  }
}

function formatTokens(count: number): string {
  return count.toLocaleString();
}

function calcDuration(startTime: number, endTime?: number): string {
  const ms = (endTime ?? Date.now()) - startTime;
  return formatDuration(ms);
}

// ---------------------------------------------------------------------------
// Sub-views
// ---------------------------------------------------------------------------

interface SdkAgentDetailProps {
  agentRef: string;
}

function SdkAgentDetail({ agentRef }: SdkAgentDetailProps): JSX.Element {
  const agent = useStore((s) => s.agents[agentRef] ?? null);

  if (!agent) {
    return (
      <Box paddingX={1}>
        <Text color={colors.muted}>Agent not found: {agentRef}</Text>
      </Box>
    );
  }

  const statusColor = getStatusColor(agent.status);
  const duration = calcDuration(agent.startTime, agent.endTime);

  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Model */}
      <Box>
        <Text color={colors.muted}>Model: </Text>
        <Text color={colors.primary}>{agent.model}</Text>
      </Box>

      {/* Tier */}
      <Box>
        <Text color={colors.muted}>Tier: </Text>
        <Text color={colors.secondary}>{agent.tier}</Text>
      </Box>

      {/* Status */}
      <Box>
        <Text color={colors.muted}>Status: </Text>
        <Text color={statusColor} bold>
          {agent.status}
        </Text>
      </Box>

      {/* Duration */}
      <Box>
        <Text color={colors.muted}>Duration: </Text>
        <Text>{duration}</Text>
      </Box>

      {/* Token usage */}
      {agent.tokenUsage !== undefined && (
        <>
          <Box>
            <Text color={colors.muted}>Input tokens: </Text>
            <Text>{formatTokens(agent.tokenUsage.input)}</Text>
          </Box>
          <Box>
            <Text color={colors.muted}>Output tokens: </Text>
            <Text>{formatTokens(agent.tokenUsage.output)}</Text>
          </Box>
          <Box>
            <Text color={colors.muted}>Total tokens: </Text>
            <Text bold>
              {formatTokens(agent.tokenUsage.input + agent.tokenUsage.output)}
            </Text>
          </Box>
        </>
      )}

      {/* Description */}
      {agent.description !== undefined && (
        <Box marginTop={1} flexDirection="column">
          <Text color={colors.muted}>Description:</Text>
          <Text>{agent.description}</Text>
        </Box>
      )}

      {/* Cost */}
      {agent.cost !== undefined && (
        <Box>
          <Text color={colors.muted}>Cost: </Text>
          <Text>${agent.cost.toFixed(2)}</Text>
        </Box>
      )}
    </Box>
  );
}

interface TeamRootDetailProps {
  teamDir: string;
  displayName: string;
}

function TeamRootDetail({
  teamDir,
  displayName,
}: TeamRootDetailProps): JSX.Element {
  const team = useStore((s) => s.teams.find((t) => t.dir === teamDir) ?? null);

  if (!team) {
    return (
      <Box paddingX={1}>
        <Text color={colors.muted}>Team not found: {displayName}</Text>
      </Box>
    );
  }

  const budgetUsed = team.budgetMax - team.budgetRemaining;
  const budgetPct = team.budgetMax > 0
    ? Math.round((budgetUsed / team.budgetMax) * 100)
    : 0;
  const budgetColor =
    budgetPct >= 90 ? colors.error : budgetPct >= 70 ? colors.warning : colors.success;

  const statusColor = getTeamStatusColor(team.status);

  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Type */}
      <Box>
        <Text color={colors.muted}>Type: </Text>
        <Text color={colors.secondary}>{team.workflowType}</Text>
      </Box>

      {/* Status */}
      <Box>
        <Text color={colors.muted}>Status: </Text>
        <Text color={statusColor} bold>
          {team.status.toUpperCase()}
        </Text>
        {team.backgroundPid !== null && (
          <Text color={colors.muted}> (PID {team.backgroundPid})</Text>
        )}
      </Box>

      {/* Budget */}
      <Box>
        <Text color={colors.muted}>Budget: </Text>
        <Text color={budgetColor}>${team.budgetRemaining.toFixed(2)}</Text>
        <Text color={colors.muted}> / ${team.budgetMax.toFixed(2)}</Text>
      </Box>

      {/* Elapsed */}
      <Box>
        <Text color={colors.muted}>Elapsed: </Text>
        <Text>{formatElapsed(team.startedAt)}</Text>
      </Box>

      {/* Progress */}
      <Box>
        <Text color={colors.muted}>Progress: </Text>
        <Text>
          {team.completedMembers}/{team.memberCount} workers, wave{" "}
          {team.currentWave}/{team.waveCount}
        </Text>
      </Box>

      {/* Cost */}
      <Box>
        <Text color={colors.muted}>Cost: </Text>
        <Text>${team.totalCost.toFixed(2)}</Text>
      </Box>
    </Box>
  );
}

interface TeamMemberDetailProps {
  node: UnifiedNode;
}

function TeamMemberDetail({ node }: TeamMemberDetailProps): JSX.Element {
  const statusColor = getStatusColor(node.status);
  const duration = calcDuration(node.startTime, node.endTime);

  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Agent / model */}
      <Box>
        <Text color={colors.muted}>Agent: </Text>
        <Text color={colors.primary}>{node.model}</Text>
      </Box>

      {/* Wave */}
      {node.waveNumber !== undefined && (
        <Box>
          <Text color={colors.muted}>Wave: </Text>
          <Text>{node.waveNumber}</Text>
        </Box>
      )}

      {/* Status */}
      <Box>
        <Text color={colors.muted}>Status: </Text>
        <Text color={statusColor} bold>
          {node.status}
        </Text>
      </Box>

      {/* Cost */}
      <Box>
        <Text color={colors.muted}>Cost: </Text>
        <Text>${(node.cost ?? 0).toFixed(2)}</Text>
      </Box>

      {/* Duration */}
      <Box>
        <Text color={colors.muted}>Duration: </Text>
        <Text>{duration}</Text>
      </Box>

      {/* Latest activity (full text, not truncated) */}
      {node.latestActivity !== undefined && node.latestActivity.length > 0 && (
        <Box marginTop={1} flexDirection="column">
          <Text color={colors.muted}>Latest Activity:</Text>
          <Box
            borderStyle="single"
            borderColor={colors.muted}
            paddingX={1}
            marginTop={0}
          >
            <Text dimColor>{node.latestActivity}</Text>
          </Box>
        </Box>
      )}
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function UnifiedDetail({
  focused,
  selectedNode,
}: UnifiedDetailProps): JSX.Element {
  const headerColor = focused ? colors.focused : colors.muted;

  // Derive header label based on node kind
  const headerLabel = useMemo((): string => {
    if (!selectedNode) return "Detail";
    switch (selectedNode.kind) {
      case "sdk-agent":
        return "Agent Detail";
      case "team-root":
        return `Team: ${selectedNode.displayName}`;
      case "team-member":
        return `Member: ${selectedNode.displayName}`;
    }
  }, [selectedNode]);

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1} paddingX={1}>
        <Text bold color={headerColor}>
          {headerLabel}
        </Text>
      </Box>

      {/* Body */}
      {selectedNode === null && (
        <Box paddingX={1}>
          <Text color={colors.muted}>Select an item to view details</Text>
        </Box>
      )}

      {selectedNode?.kind === "sdk-agent" && selectedNode.agentRef !== undefined && (
        <SdkAgentDetail agentRef={selectedNode.agentRef} />
      )}

      {selectedNode?.kind === "team-root" && selectedNode.teamDir !== undefined && (
        <TeamRootDetail
          teamDir={selectedNode.teamDir}
          displayName={selectedNode.displayName}
        />
      )}

      {selectedNode?.kind === "team-member" && (
        <TeamMemberDetail node={selectedNode} />
      )}
    </Box>
  );
}
