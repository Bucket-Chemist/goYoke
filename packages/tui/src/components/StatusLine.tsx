import React, { useState, useEffect, useRef, useMemo } from "react";
import { Box, Text } from "ink";
import { execSync } from "child_process";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";
import { PROVIDERS } from "../config/providers.js";

interface StatusLineProps {
  width: number;
  height?: 1 | 2;
}

/**
 * Truncates an email address for compact display.
 * Format: first 5 chars of local part + "...@" + domain
 * e.g. "will.klare.nl@gmail.com" → "will....@gmail.com"
 */
export function truncateEmail(email: string): string {
  const atIdx = email.indexOf("@");
  if (atIdx === -1) return email;
  const local = email.substring(0, atIdx);
  const domain = email.substring(atIdx + 1);
  const prefix = local.length <= 5 ? local : local.substring(0, 5) + "...";
  return `${prefix}@${domain}`;
}

interface AuthInfo {
  authMethod: string | null;
  email: string | null;
}

/**
 * Auth info hook with caching
 * Polls `claude auth status --json` every cacheTtlMs to avoid expensive
 * process spawns on every render. Auth state changes very rarely.
 */
function useAuthInfo(cacheTtlMs = 30000): AuthInfo {
  const [info, setInfo] = useState<AuthInfo>({
    authMethod: null,
    email: null,
  });
  const lastFetch = useRef(0);

  useEffect(() => {
    const fetch = () => {
      if (Date.now() - lastFetch.current < cacheTtlMs) return;
      lastFetch.current = Date.now();
      try {
        const raw = execSync("claude auth status --json", {
          encoding: "utf8",
          stdio: ["pipe", "pipe", "ignore"],
        }).trim();
        const parsed = JSON.parse(raw) as {
          loggedIn?: boolean;
          authMethod?: string;
          email?: string;
        };
        if (parsed.loggedIn) {
          setInfo({
            authMethod: parsed.authMethod ?? null,
            email: parsed.email ?? null,
          });
        }
      } catch {
        /* claude CLI unavailable or not logged in — leave defaults */
      }
    };
    fetch();
    const interval = setInterval(fetch, cacheTtlMs);
    return () => clearInterval(interval);
  }, [cacheTtlMs]);

  return info;
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
    totalCost,
    contextWindow,
    streaming,
    agents,
    permissionMode,
  } = useStore();

  // Use selector to subscribe to the actual providerModels data, avoiding
  // the getter-on-state-object issue where Object.assign in Zustand's setState
  // evaluates getters once and stores static values, breaking reactivity.
  const modelName = useStore((state) => {
    const provider = state.activeProvider;
    const modelId = state.getActiveModel();
    if (!modelId) return "unknown";
    const def = PROVIDERS[provider]?.models.find((m) => m.id === modelId);
    return def?.displayName ?? modelId.substring(0, 12);
  });

  const gitInfo = useGitInfo();
  const authInfo = useAuthInfo();
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

  // Responsive ContextBar width: scales with terminal width (10–30 chars)
  const contextBarWidth = Math.max(10, Math.min(30, Math.floor(width * 0.12)));

  return (
    <Box flexDirection="column" width={width}>
      {/* Separator line */}
      <Text color={colors.muted} dimColor>
        {"─".repeat(width)}
      </Text>

      {/* Line 1: LEFT = model + permission + project + git | RIGHT = auth */}
      <Box width={width} justifyContent="space-between">
        {/* Left group */}
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

        {/* Right group: auth info */}
        {authInfo.authMethod && (
          <Box>
            <Text color={colors.muted}>
              {authInfo.authMethod}
              {authInfo.email && ` · ${truncateEmail(authInfo.email)}`}
            </Text>
          </Box>
        )}
      </Box>

      {/* Line 2: LEFT = context bar + cost | RIGHT = duration + agents + teams */}
      {effectiveHeight >= 2 && (
        <Box width={width} justifyContent="space-between">
          {/* Left group */}
          <Box>
            <ContextBar percentage={contextPct} width={contextBarWidth} />
            <Text color={colors.muted}> | </Text>
            <Text color={colors.warning}>${cost}</Text>
          </Box>

          {/* Right group: duration + agents + optional teams */}
          <Box>
            <Text color={colors.muted}>
              ⏱️ {minutes}m {String(seconds).padStart(2, "0")}s
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
        </Box>
      )}
    </Box>
  );
}
