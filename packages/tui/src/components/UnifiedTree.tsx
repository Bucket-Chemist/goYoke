/**
 * UnifiedTree - Single tree view showing SDK agents and background teams.
 *
 * Visual structure:
 *   Agents (N)
 *   ● Router: running
 *     ├─ ✓ codebase-search: complete
 *     └─ ● go-pro: running
 *
 *   Teams (N)
 *   ▶ braintrust [running] $1.48
 *     ├─ ● einstein: running  "Analyzing root cause..."
 *     └─ ◐ beethoven: pending
 *
 * Nodes arrive pre-ordered and pre-leveled from useUnifiedTree hook.
 * The `depth` field drives indentation; parentId drives branch/leaf selection.
 */

import React, { useMemo } from "react";
import { Box, Text } from "ink";
import { colors, icons } from "../config/theme.js";
import type { UnifiedNode, AgentStatus } from "../store/types.js";

export interface UnifiedTreeProps {
  focused: boolean;
  nodes: UnifiedNode[];
  selectedNode: UnifiedNode | null;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getStatusIcon(status: AgentStatus): string {
  switch (status) {
    case "spawning":
      return icons.agentSpawning;
    case "running":
    case "streaming":
      return icons.agentRunning;
    case "complete":
      return icons.agentComplete;
    case "error":
    case "timeout":
      return icons.agentError;
    case "queued":
      return "◐";
    default:
      return "?";
  }
}

function getStatusColor(status: AgentStatus): string {
  switch (status) {
    case "spawning":
    case "queued":
      return colors.agentSpawning;
    case "running":
    case "streaming":
      return colors.agentRunning;
    case "complete":
      return colors.agentComplete;
    case "error":
    case "timeout":
      return colors.agentError;
    default:
      return colors.muted;
  }
}

/** Returns true if this node is the last child of its parent in the flat list. */
function isLastChild(nodes: UnifiedNode[], index: number): boolean {
  const node = nodes[index];
  if (!node || node.parentId === null) return false;

  // Scan forward: if we hit another node with the same parentId before hitting
  // a node at depth <= current depth, this is NOT the last child.
  for (let i = index + 1; i < nodes.length; i++) {
    const next = nodes[i];
    if (!next) break;
    if (next.depth <= node.depth) break; // left the sibling group
    if (next.parentId === node.parentId) return false;
  }
  return true;
}

function truncate(str: string, maxLen: number): string {
  return str.length > maxLen ? str.slice(0, maxLen - 1) + "…" : str;
}

// ---------------------------------------------------------------------------
// Row renderers
// ---------------------------------------------------------------------------

interface RowProps {
  node: UnifiedNode;
  isSelected: boolean;
  isLast: boolean;
}

function SdkAgentRow({ node, isSelected, isLast }: RowProps): JSX.Element {
  const statusColor = getStatusColor(node.status);
  const statusIcon = getStatusIcon(node.status);

  const indent = node.depth > 0 ? "  ".repeat(node.depth - 1) : "";
  const branch = node.depth > 0 ? (isLast ? icons.treeLeaf : icons.treeBranch) + " " : "";
  const prefix = `${indent}${branch}`;

  const line = `${prefix}${statusIcon} ${node.displayName}: ${node.status}`;

  const isActive = node.status === "running" || node.status === "streaming";
  const activity = node.activity;
  const activityIndent = " ".repeat(prefix.length + 2);

  let toolLine: JSX.Element | null = null;
  let textLine: JSX.Element | null = null;

  if (isActive && !isSelected && activity) {
    const tool = activity.currentTool;
    const result = activity.toolResult;

    if (tool) {
      if (result?.status === "failed") {
        const errMsg = truncate(result.error ?? "failed", 40);
        toolLine = (
          <Text color={colors.agentError}>
            {activityIndent}✗ {tool.name}: {errMsg}
          </Text>
        );
      } else {
        const pending = result?.status === "pending";
        const toolPrefix = pending ? "⏳" : "▸";
        const target = truncate(tool.target, 40);
        toolLine = (
          <Text color={colors.accent} dimColor>
            {activityIndent}{toolPrefix} {tool.name} → {target}
          </Text>
        );
      }
    }

    if (activity.lastText) {
      const snippet = truncate(activity.lastText, 40);
      textLine = (
        <Text color={colors.muted} dimColor>
          {activityIndent}&quot;{snippet}&quot;
        </Text>
      );
    }
  }

  return (
    <Box flexDirection="column">
      <Text
        color={isSelected ? undefined : statusColor}
        inverse={isSelected}
        bold={isSelected}
      >
        {line}
      </Text>
      {toolLine}
      {textLine}
    </Box>
  );
}

function TeamRootRow({ node, isSelected }: Omit<RowProps, "isLast">): JSX.Element {
  const statusColor = getStatusColor(node.status);
  const cost = node.cost !== undefined ? ` $${node.cost.toFixed(2)}` : "";

  return (
    <Box>
      <Text
        color={isSelected ? undefined : statusColor}
        inverse={isSelected}
        bold={isSelected}
      >
        {icons.teamRoot} {node.displayName} [{node.status}]{cost}
      </Text>
    </Box>
  );
}

function TeamMemberRow({ node, isSelected, isLast }: RowProps): JSX.Element {
  const statusColor = getStatusColor(node.status);
  const statusIcon = getStatusIcon(node.status);
  const branch = isLast ? icons.treeLeaf : icons.treeBranch;

  const isActive = node.status === "running" || node.status === "streaming";
  const activity = node.activity;
  // 4 spaces: aligns under member name after "  " indent + branch chars
  const activityIndent = "    ";

  let toolLine: JSX.Element | null = null;
  let textLine: JSX.Element | null = null;

  if (isActive && !isSelected && activity) {
    const tool = activity.currentTool;
    const result = activity.toolResult;

    if (tool) {
      if (result?.status === "failed") {
        const errMsg = truncate(result.error ?? "failed", 40);
        toolLine = (
          <Text color={colors.agentError}>
            {activityIndent}✗ {tool.name}: {errMsg}
          </Text>
        );
      } else {
        const pending = result?.status === "pending";
        const toolPrefix = pending ? "⏳" : "▸";
        const target = truncate(tool.target, 40);
        toolLine = (
          <Text color={colors.accent} dimColor>
            {activityIndent}{toolPrefix} {tool.name} → {target}
          </Text>
        );
      }
    }

    if (activity.lastText) {
      const snippet = truncate(activity.lastText, 40);
      textLine = (
        <Text color={colors.muted} dimColor>
          {activityIndent}&quot;{snippet}&quot;
        </Text>
      );
    }
  }

  return (
    <Box flexDirection="column">
      <Text
        color={isSelected ? undefined : statusColor}
        inverse={isSelected}
        bold={isSelected}
      >
        {"  "}{branch} {statusIcon} {node.displayName}: {node.status}
      </Text>
      {toolLine}
      {textLine}
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Section header counters
// ---------------------------------------------------------------------------

interface SectionCounts {
  sdkAgents: number;
  teamRoots: number;
}

function countSections(nodes: UnifiedNode[]): SectionCounts {
  let sdkAgents = 0;
  let teamRoots = 0;
  for (const node of nodes) {
    if (node.kind === "sdk-agent") sdkAgents++;
    else if (node.kind === "team-root") teamRoots++;
  }
  return { sdkAgents, teamRoots };
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function UnifiedTree({ focused, nodes, selectedNode }: UnifiedTreeProps): JSX.Element {
  const { sdkAgents, teamRoots } = useMemo(() => countSections(nodes), [nodes]);

  // Empty state
  if (nodes.length === 0) {
    return (
      <Box flexDirection="column" paddingX={1}>
        <Text color={colors.muted}>No agents or teams</Text>
      </Box>
    );
  }

  const headerColor = focused ? colors.focused : colors.muted;

  // Track whether we've emitted each section header yet
  let sdkHeaderEmitted = false;
  let teamHeaderEmitted = false;

  const rows: JSX.Element[] = [];

  nodes.forEach((node, index) => {
    const isSelected = selectedNode !== null && selectedNode.id === node.id;
    const isLast = isLastChild(nodes, index);

    // Emit section headers on first occurrence of each kind
    if (node.kind === "sdk-agent" && !sdkHeaderEmitted) {
      sdkHeaderEmitted = true;
      rows.push(
        <Box key="header-agents" marginBottom={0}>
          <Text bold color={headerColor}>
            Agents ({sdkAgents})
          </Text>
        </Box>
      );
    }

    if ((node.kind === "team-root" || node.kind === "team-member") && !teamHeaderEmitted) {
      teamHeaderEmitted = true;
      rows.push(
        <Box key="header-teams" marginTop={sdkHeaderEmitted ? 1 : 0} marginBottom={0}>
          <Text bold color={headerColor}>
            Teams ({teamRoots})
          </Text>
        </Box>
      );
    }

    // Emit the node row
    switch (node.kind) {
      case "sdk-agent":
        rows.push(
          <SdkAgentRow key={node.id} node={node} isSelected={isSelected} isLast={isLast} />
        );
        break;
      case "team-root":
        rows.push(
          <TeamRootRow key={node.id} node={node} isSelected={isSelected} />
        );
        break;
      case "team-member":
        rows.push(
          <TeamMemberRow key={node.id} node={node} isSelected={isSelected} isLast={isLast} />
        );
        break;
    }
  });

  return (
    <Box flexDirection="column" paddingX={1}>
      {rows}
    </Box>
  );
}
