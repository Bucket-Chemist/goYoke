/**
 * TeamList component
 * Right panel showing all teams with summary stats
 * Pattern follows DashboardView.tsx
 */

import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { useTeams } from "../hooks/useTeams.js";
import { colors } from "../config/theme.js";
import {
  formatElapsed,
  getTeamStatusDisplay,
} from "../utils/teamFormatting.js";

export function TeamList(): JSX.Element {
  const teams = useTeams();
  const { selectedTeamDir, selectTeam } = useStore();

  // Empty state
  if (teams.length === 0) {
    return (
      <Box flexDirection="column" paddingX={1} paddingY={0}>
        <Text bold color={colors.primary}>
          Teams (0)
        </Text>
        <Box marginTop={1}>
          <Text color={colors.muted}>No background teams running</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={1} paddingY={0}>
      {/* Header */}
      <Text bold color={colors.primary}>
        Teams ({teams.length})
      </Text>

      {/* Team list */}
      <Box marginTop={1} flexDirection="column">
        {teams.map((team) => {
          const { icon, color } = getTeamStatusDisplay(team.status, team.alive);
          const isSelected = selectedTeamDir === team.dir;

          // Format wave progress
          const waveProgress =
            team.currentWave > 0
              ? `wave ${team.currentWave}/${team.waveCount}`
              : `${team.waveCount} waves`;

          // Format cost
          const cost = `$${team.totalCost.toFixed(2)}`;

          // Format duration
          const duration = formatElapsed(team.startedAt);

          return (
            <Box key={team.dir} flexDirection="column" marginBottom={1}>
              {/* Team name row */}
              <Box>
                <Text color={color} bold={isSelected}>
                  {icon}{" "}
                </Text>
                <Text
                  color={isSelected ? colors.focused : colors.primary}
                  bold={isSelected}
                >
                  {team.name}
                </Text>
                <Text color={colors.muted}> [{team.status}]</Text>
              </Box>

              {/* Stats row */}
              <Box paddingLeft={2}>
                <Text color={colors.muted}>
                  {team.workflowType} · {waveProgress} ·{" "}
                  {team.completedMembers}/{team.memberCount} workers · {cost} ·{" "}
                  {duration}
                </Text>
              </Box>
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}
