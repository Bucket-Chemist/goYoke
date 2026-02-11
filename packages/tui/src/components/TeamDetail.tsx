/**
 * TeamDetail component
 * Shows detailed information about selected team
 * Pattern follows AgentDetail.tsx
 */

import React, { useEffect, useRef } from "react";
import { Box, Text } from "ink";
import { readFile } from "fs/promises";
import { join } from "path";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";
import type { TeamConfig } from "../store/types.js";
import {
  formatDuration,
  formatElapsed,
  getMemberStatusIcon,
  getMemberStatusColor,
  getTeamStatusColor,
} from "../utils/teamFormatting.js";

/**
 * Format time from ISO timestamp
 */
function formatTime(timestamp: string | null): string {
  if (!timestamp) return "—";
  const date = new Date(timestamp);
  return date.toLocaleTimeString("en-US", { hour12: false });
}

export function TeamDetail(): JSX.Element {
  const { selectedTeamDir, selectedTeamDetail, setTeamDetail } = useStore();
  const loadSequenceRef = useRef(0);

  // Load team config when selection changes
  useEffect(() => {
    let isMounted = true;
    const currentSeq = ++loadSequenceRef.current;

    const loadConfig = async (): Promise<void> => {
      if (!selectedTeamDir) {
        if (isMounted && currentSeq === loadSequenceRef.current) {
          setTeamDetail(null);
        }
        return;
      }

      const sessionDir = process.env["GOGENT_SESSION_DIR"];
      if (!sessionDir) {
        if (isMounted && currentSeq === loadSequenceRef.current) {
          setTeamDetail(null);
        }
        return;
      }

      try {
        const configPath = join(
          sessionDir,
          "teams",
          selectedTeamDir,
          "config.json"
        );
        const data = await readFile(configPath, "utf-8");
        const config: TeamConfig = JSON.parse(data);

        if (isMounted && currentSeq === loadSequenceRef.current) {
          setTeamDetail(config);
        }
      } catch (error) {
        if (process.env["VERBOSE"]) {
          console.warn(
            `Failed to load team config: ${
              error instanceof Error ? error.message : String(error)
            }`
          );
        }
        if (isMounted && currentSeq === loadSequenceRef.current) {
          setTeamDetail(null);
        }
      }
    };

    void loadConfig();

    return () => {
      isMounted = false;
    };
  }, [selectedTeamDir, setTeamDetail]);

  // Empty state
  if (!selectedTeamDetail) {
    return (
      <Box flexDirection="column" paddingX={1}>
        <Box marginBottom={1}>
          <Text bold color={colors.muted}>
            Team Detail
          </Text>
        </Box>
        <Text color={colors.muted}>Select a team to view details</Text>
      </Box>
    );
  }

  const config = selectedTeamDetail;

  // Calculate budget percentage
  const budgetPct = Math.round(
    ((config.budget_max_usd - config.budget_remaining_usd) /
      config.budget_max_usd) *
      100
  );

  // Count wave status
  const completedWaves = config.waves.filter((w) =>
    w.members.every((m) => m.status === "completed")
  ).length;
  const runningWaves = config.waves.filter((w) =>
    w.members.some((m) => m.status === "running")
  ).length;

  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color={colors.primary}>
          Team: {config.team_name}
        </Text>
      </Box>

      {/* Type */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Type: </Text>
        <Text color={colors.secondary}>{config.workflow_type}</Text>
      </Box>

      {/* Status */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Status: </Text>
        <Text color={getTeamStatusColor(config.status)} bold>
          {config.status.toUpperCase()}
        </Text>
        {config.background_pid && (
          <Text color={colors.muted}> (PID {config.background_pid})</Text>
        )}
      </Box>

      {/* Budget */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Budget: </Text>
        <Text
          color={budgetPct >= 90 ? colors.error : budgetPct >= 70 ? colors.warning : colors.success}
        >
          ${config.budget_remaining_usd.toFixed(2)}
        </Text>
        <Text color={colors.muted}> / ${config.budget_max_usd.toFixed(2)}</Text>
        <Text color={colors.muted}> ({budgetPct}%)</Text>
      </Box>

      {/* Timing */}
      <Box marginBottom={0}>
        <Text color={colors.muted}>Started: </Text>
        <Text>{formatTime(config.started_at)}</Text>
        {config.started_at && (
          <>
            <Text color={colors.muted}> · Elapsed: </Text>
            <Text>{formatElapsed(config.started_at)}</Text>
          </>
        )}
      </Box>

      {/* Waves breakdown */}
      <Box marginTop={1} flexDirection="column">
        {config.waves.map((wave) => {
          const completed = wave.members.filter((m) => m.status === "completed")
            .length;
          const running = wave.members.filter((m) => m.status === "running")
            .length;
          const failed = wave.members.filter((m) => m.status === "failed")
            .length;

          const waveStatus =
            completed === wave.members.length
              ? "COMPLETED"
              : running > 0
                ? "RUNNING"
                : failed > 0
                  ? "FAILED"
                  : "PENDING";

          return (
            <Box key={wave.wave_number} flexDirection="column" marginBottom={1}>
              {/* Wave header */}
              <Text bold color={colors.muted}>
                Wave {wave.wave_number}: {waveStatus} ({completed}/
                {wave.members.length})
              </Text>

              {/* Members */}
              {wave.members.map((member) => {
                const icon = getMemberStatusIcon(member.status);
                const color = getMemberStatusColor(member.status);

                // Format duration
                let duration = "—";
                if (member.started_at) {
                  if (member.completed_at) {
                    const start = new Date(member.started_at).getTime();
                    const end = new Date(member.completed_at).getTime();
                    duration = formatDuration(end - start);
                  } else {
                    duration = formatElapsed(member.started_at);
                  }
                }

                return (
                  <Box key={member.name} paddingLeft={2}>
                    <Text color={color}>{icon} </Text>
                    <Text>{member.name}</Text>
                    <Text color={colors.muted}>
                      {" "}
                      {member.agent} ${member.cost_usd.toFixed(2)} {duration}
                    </Text>
                  </Box>
                );
              })}
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}
