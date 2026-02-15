import React, { useState, useEffect, useRef, useMemo } from "react";
import { Box, Text } from "ink";
import { execSync } from "child_process";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";

interface StatusLineProps {
  width: number;
  height?: 1 | 2;
}

/**
 * Git info hook with caching
 * Polls git status every cacheTtlMs to avoid expensive syscalls on every render
 */
function useGitInfo(cacheTtlMs = 5000): {
  branch: string | null;
  staged: number;
  modified: number;
} {
  const [info, setInfo] = useState({
    branch: null as string | null,
    staged: 0,
    modified: 0,
  });
  const lastFetch = useRef(0);

  useEffect(() => {
    const fetch = () => {
      if (Date.now() - lastFetch.current < cacheTtlMs) return;
      lastFetch.current = Date.now();
      try {
        const branch = execSync("git branch --show-current", {
          encoding: "utf8",
          stdio: ["pipe", "pipe", "ignore"],
        }).trim();
        const stagedOutput = execSync("git diff --cached --numstat", {
          encoding: "utf8",
          stdio: ["pipe", "pipe", "ignore"],
        }).trim();
        const modifiedOutput = execSync("git diff --numstat", {
          encoding: "utf8",
          stdio: ["pipe", "pipe", "ignore"],
        }).trim();
        setInfo({
          branch: branch || null,
          staged: stagedOutput ? stagedOutput.split("\n").length : 0,
          modified: modifiedOutput ? modifiedOutput.split("\n").length : 0,
        });
      } catch {
        /* not in git repo, leave defaults */
      }
    };
    fetch();
    const interval = setInterval(fetch, cacheTtlMs);
    return () => clearInterval(interval);
  }, [cacheTtlMs]);

  return info;
}

/**
 * Context progress bar component
 * Shows token usage as a visual bar with color coding
 */
function ContextBar({
  percentage,
  width = 10,
}: {
  percentage: number;
  width?: number;
}): JSX.Element {
  const filled = Math.round((percentage * width) / 100);
  const empty = width - filled;
  const color =
    percentage >= 90 ? colors.error : percentage >= 70 ? colors.warning : colors.success;

  return (
    <Text>
      <Text color={color}>{"▓".repeat(filled)}</Text>
      <Text color={colors.muted}>{"░".repeat(empty)}</Text>
      <Text color={colors.muted}> {Math.round(percentage)}%</Text>
    </Text>
  );
}

/**
 * Streaming spinner component
 * Braille animation when streaming is active
 */
function StreamingSpinner(): JSX.Element {
  const BRAILLE = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setFrame((f) => (f + 1) % BRAILLE.length);
    }, 80);
    return () => clearInterval(interval);
  }, []);

  return <Text color={colors.primary}>{BRAILLE[frame]}</Text>;
}

/**
 * StatusLine component
 * 2-line status bar showing model, git, context, cost, duration, and agent count
 */
export function StatusLine({ width, height = 2 }: StatusLineProps): JSX.Element {
  const {
    activeModel,
    preferredModel,
    totalCost,
    contextWindow,
    streaming,
    agents,
    permissionMode,
  } = useStore();

  const gitInfo = useGitInfo();
  const teams = useStore((state) => state.teams);
  const startTime = useRef(Date.now());
  const [tick, setTick] = useState(0);

  // Force re-render every second for duration display
  useEffect(() => {
    const interval = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  // Detect ASCII mode
  const useAscii = process.env["TERM"] === "dumb" || process.env["GOGENT_ASCII"] === "1";

  // Determine model display name
  const modelName = useMemo(() => {
    const model = activeModel || preferredModel || "unknown";
    if (model.includes("opus")) return "Opus";
    if (model.includes("sonnet")) return "Sonnet";
    if (model.includes("haiku")) return "Haiku";
    return model.substring(0, 10); // Fallback: truncate
  }, [activeModel, preferredModel]);

  // Calculate context percentage based on actual context window usage
  const contextPct = useMemo(() => {
    if (contextWindow.totalCapacity === 0) return 0;
    return Math.min(
      100,
      Math.round((contextWindow.usedTokens / contextWindow.totalCapacity) * 100)
    );
  }, [contextWindow]);

  // Format cost
  const cost = useMemo(() => totalCost.toFixed(2), [totalCost]);

  // Calculate session duration
  const { minutes, seconds } = useMemo(() => {
    const elapsed = Math.floor((Date.now() - startTime.current) / 1000);
    return {
      minutes: Math.floor(elapsed / 60),
      seconds: elapsed % 60,
    };
  }, [tick]);

  // Count agents by status
  const agentCounts = useMemo(() => {
    const values = Object.values(agents);
    return {
      running: values.filter(
        (a) => a.status === "running" || a.status === "streaming"
      ).length,
      queued: values.filter(
        (a) => a.status === "queued" || a.status === "spawning"
      ).length,
      complete: values.filter((a) => a.status === "complete").length,
    };
  }, [agents]);

  // Memoize team stats - runs only when teams array changes, not on every tick
  const teamStats = useMemo(() => {
    const aliveTeams = teams.filter((t) => t.alive);
    const aliveCount = aliveTeams.length;
    const furthest =
      aliveTeams.length > 0
        ? aliveTeams.reduce((max, t) =>
            t.currentWave > max.currentWave ? t : max
          )
        : null;
    const totalSpend = teams.reduce((sum, t) => sum + t.totalCost, 0);
    return { aliveCount, furthest, totalSpend };
  }, [teams]);

  // Responsive layout
  const effectiveHeight = width < 100 ? 1 : height;
  const showGit = width >= 80;
  const projectName = "GOgent-Fortress";

  return (
    <Box flexDirection="column" width={width}>
      {/* Separator line */}
      <Text color={colors.muted} dimColor>
        {"─".repeat(width)}
      </Text>

      {/* Line 1: Model, project, git, permission mode */}
      <Box>
        <Text bold color={colors.primary}>
          [{modelName}]
        </Text>
        {permissionMode !== 'default' && (
          <Text bold color={permissionMode === 'plan' ? colors.secondary : colors.warning}>
            {" "}[{permissionMode === 'acceptEdits' ? 'Auto-Edit' : 'Plan'}]
          </Text>
        )}
        <Text color={colors.muted}> 📁 {projectName}</Text>
        {showGit && gitInfo.branch && (
          <Text color={colors.muted}>
            {" "}
            | 🌿 {gitInfo.branch}
            {gitInfo.staged > 0 && (
              <Text color={colors.success}> +{gitInfo.staged}</Text>
            )}
            {gitInfo.modified > 0 && (
              <Text color={colors.warning}> ~{gitInfo.modified}</Text>
            )}
          </Text>
        )}
      </Box>

      {/* Line 2: Context bar, cost, duration, agents (only if height >= 2) */}
      {effectiveHeight >= 2 && (
        <Box>
          <ContextBar percentage={contextPct} />
          <Text color={colors.muted}> | </Text>
          <Text color={colors.warning}>${cost}</Text>
          <Text color={colors.muted}>
            {" "}
            | ⏱️ {minutes}m {String(seconds).padStart(2, "0")}s
          </Text>
          <Text color={colors.muted}> | </Text>
          {streaming ? (
            <>
              <StreamingSpinner />
              <Text color={colors.muted}> streaming</Text>
            </>
          ) : (
            <Text color={colors.muted}>
              🤖 {agentCounts.running} running
              {agentCounts.queued > 0 && (
                <Text color={colors.warning}> ({agentCounts.queued} queued)</Text>
              )}
            </Text>
          )}
          {teams.length > 0 && (
            <>
              <Text color={colors.muted}> | </Text>
              <Text color={colors.success}>
                {useAscii ? "[BG]" : "🏗️"} {teamStats.aliveCount} team{teamStats.aliveCount !== 1 ? "s" : ""}
                {teamStats.furthest && teamStats.furthest.currentWave > 0 && (
                  <>
                    {" · "}wave {teamStats.furthest.currentWave}/{teamStats.furthest.waveCount}
                  </>
                )}
                {teamStats.totalSpend > 0 && (
                  <> · ${teamStats.totalSpend.toFixed(2)}</>
                )}
              </Text>
            </>
          )}
        </Box>
      )}
    </Box>
  );
}
