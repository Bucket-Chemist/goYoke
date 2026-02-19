/**
 * List sessions command
 * Display formatted session list for --list flag
 */

import { Box, Text } from "ink";
import React from "react";
import type { SessionData } from "../store/types.js";
import { colors } from "../config/theme.js";

interface ListSessionsProps {
  sessions: SessionData[];
}

/**
 * Format cost for display
 */
function formatCost(cost: number): string {
  return `$${cost.toFixed(4)}`;
}

/**
 * Format date for display (relative or absolute)
 */
function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 60) {
    return diffMins === 1 ? "1 minute ago" : `${diffMins} minutes ago`;
  } else if (diffHours < 24) {
    return diffHours === 1 ? "1 hour ago" : `${diffHours} hours ago`;
  } else if (diffDays < 7) {
    return diffDays === 1 ? "1 day ago" : `${diffDays} days ago`;
  } else {
    return date.toLocaleDateString();
  }
}

/**
 * Truncate session ID for display
 */
function truncateId(id: string): string {
  return id;
}

/**
 * List sessions component
 * Displays formatted session list with ID, date, cost, name
 */
export function ListSessions({ sessions }: ListSessionsProps): JSX.Element {
  if (sessions.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color={colors.muted}>No sessions found.</Text>
        <Text color={colors.muted} dimColor>
          Start a new session to begin.
        </Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color={colors.primary}>
          Available Sessions
        </Text>
      </Box>

      {/* Column headers */}
      <Box>
        <Box width={40}>
          <Text bold dimColor>
            ID
          </Text>
        </Box>
        <Box width={20}>
          <Text bold dimColor>
            Last Used
          </Text>
        </Box>
        <Box width={12}>
          <Text bold dimColor>
            Cost
          </Text>
        </Box>
        <Box width={10}>
          <Text bold dimColor>
            Tools
          </Text>
        </Box>
        <Box>
          <Text bold dimColor>
            Name
          </Text>
        </Box>
      </Box>

      {/* Session rows */}
      {sessions.map((session) => (
        <Box key={session.id} marginTop={1}>
          <Box width={40}>
            <Text color={colors.secondary}>{truncateId(session.id)}</Text>
          </Box>
          <Box width={20}>
            <Text>{formatDate(session.last_used)}</Text>
          </Box>
          <Box width={12}>
            <Text color={colors.primary}>{formatCost(session.cost)}</Text>
          </Box>
          <Box width={10}>
            <Text>{session.tool_calls}</Text>
          </Box>
          <Box>
            <Text color={session.name ? "white" : colors.muted}>
              {session.name || "(unnamed)"}
            </Text>
          </Box>
        </Box>
      ))}

      {/* Footer */}
      <Box marginTop={1} flexDirection="column">
        <Text dimColor color={colors.muted}>
          Resume: gofortress --resume | gofortress --session {"<id>"}
        </Text>
        <Text dimColor color={colors.muted}>
          {"        "}goclaude --resume   | goclaude --session {"<id>"}
        </Text>
      </Box>
    </Box>
  );
}
