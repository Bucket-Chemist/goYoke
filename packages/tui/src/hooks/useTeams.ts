/**
 * useTeams hook
 * Polls the session teams directory to track all teams with full metadata
 * Replaces useTeamCount with comprehensive team tracking
 */

import { useEffect, useRef } from "react";
import { readdir, readFile, open, stat } from "fs/promises";
import { join } from "path";
import { useStore } from "../store/index.js";
import type { TeamSummary, TeamMemberRow, AgentActivity } from "../store/types.js";
import { activityFromNdjsonChunk } from "../utils/agentActivity.js";

interface TeamConfigJSON {
  team_name: string;
  workflow_type: string;
  status: string;
  background_pid: number | null;
  budget_max_usd: number;
  budget_remaining_usd: number;
  started_at: string | null;
  completed_at: string | null;
  waves: Array<{
    wave_number: number;
    members: Array<{
      name: string;
      agent: string;
      model: string;
      status: string;
      cost_usd: number;
      started_at: string | null;
      completed_at: string | null;
    }>;
  }>;
}

/**
 * Check if a PID is alive
 * Uses process.kill(pid, 0) which throws ESRCH if process doesn't exist
 *
 * Note: PID-based liveness check has inherent race condition with PID reuse.
 * Acceptable for dashboard display — worst case is a brief false positive.
 */
function isPidAlive(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ESRCH") {
      return false;
    }
    // Permission errors (EPERM) mean process exists but we can't signal it
    return true;
  }
}

interface StreamActivityResult {
  text: string | null;
  activity: AgentActivity | null;
}

/**
 * Read the last 4KB of a stream_{agentName}.ndjson file and extract
 * the full AgentActivity (text, currentTool, toolResult).
 * Returns null fields on any error (missing file, parse failure, etc.).
 */
async function readStreamActivity(
  teamDir: string,
  agentName: string
): Promise<StreamActivityResult> {
  const nullResult: StreamActivityResult = { text: null, activity: null };

  try {
    const filePath = join(teamDir, `stream_${agentName}.ndjson`);

    // Get file size first
    const fileStat = await stat(filePath);
    const fileSize = fileStat.size;

    if (fileSize === 0) {
      return nullResult;
    }

    // Read last 4KB
    const chunkSize = 4096;
    const readSize = Math.min(chunkSize, fileSize);
    const position = fileSize - readSize;

    const buffer = Buffer.alloc(readSize);
    const fh = await open(filePath, "r");
    try {
      await fh.read(buffer, 0, readSize, position);
    } finally {
      await fh.close();
    }

    const chunk = buffer.toString("utf-8");

    // Split into lines and skip first (potentially truncated)
    const lines = chunk.split("\n");
    const candidateLines = position === 0 ? lines : lines.slice(1);

    // Collect valid non-empty trimmed lines for forward-parse
    const validLines = candidateLines.filter(
      (line): line is string => typeof line === "string" && line.trim().length > 0
    );

    const activity = activityFromNdjsonChunk(validLines);

    return {
      text: activity?.lastText?.slice(0, 100) ?? null,
      activity,
    };
  } catch {
    return nullResult;
  }
}

/**
 * Attach stream activity to running team members.
 * Mutates summary.members in place.
 */
async function attachStreamActivity(
  summary: TeamSummary,
  teamsBasePath: string
): Promise<void> {
  const teamPath = join(teamsBasePath, summary.dir);
  for (const member of summary.members) {
    if (member.status === "running") {
      const result = await readStreamActivity(teamPath, member.name);
      member.latestActivity = result.text?.slice(0, 100) ?? undefined;
      member.activity = result.activity ?? undefined;
    }
  }
}

/**
 * Parse a team config.json and generate summary
 */
function parseTeamSummary(
  dirName: string,
  config: TeamConfigJSON
): TeamSummary {
  // Calculate current wave (highest wave_number with running/completed members)
  let currentWave = 0;
  for (const wave of config.waves) {
    const hasActivity = wave.members.some(
      (m) => m.status === "running" || m.status === "completed" || m.status === "failed"
    );
    if (hasActivity && wave.wave_number > currentWave) {
      currentWave = wave.wave_number;
    }
  }

  // Calculate totals across all waves
  let totalCost = 0;
  let memberCount = 0;
  let completedMembers = 0;
  let failedMembers = 0;

  for (const wave of config.waves) {
    memberCount += wave.members.length;
    for (const member of wave.members) {
      totalCost += member.cost_usd;
      if (member.status === "completed") completedMembers++;
      if (member.status === "failed") failedMembers++;
    }
  }

  // Build members array from waves
  const members: TeamMemberRow[] = [];
  for (const wave of config.waves) {
    for (const member of wave.members) {
      members.push({
        name: member.name,
        agent: member.agent,
        model: member.model,
        status: member.status,
        wave: wave.wave_number,
        cost: member.cost_usd,
        startedAt: member.started_at,
        completedAt: member.completed_at,
      });
    }
  }

  // Check PID liveness
  const alive =
    config.background_pid !== null &&
    isPidAlive(config.background_pid);

  return {
    dir: dirName,
    name: config.team_name,
    workflowType: config.workflow_type,
    status: config.status as TeamSummary["status"],
    backgroundPid: config.background_pid,
    alive,
    budgetMax: config.budget_max_usd,
    budgetRemaining: config.budget_remaining_usd,
    startedAt: config.started_at,
    completedAt: config.completed_at,
    totalCost,
    waveCount: config.waves.length,
    currentWave,
    memberCount,
    completedMembers,
    failedMembers,
    members,
  };
}

/**
 * Enumerate teams directory and generate summaries
 * Returns empty array on any error (missing directory, invalid JSON, etc.)
 */
async function getTeamSummaries(sessionDir: string): Promise<TeamSummary[]> {
  try {
    const teamsDir = join(sessionDir, "teams");
    const entries = await readdir(teamsDir, { withFileTypes: true });

    const summaries: TeamSummary[] = [];

    for (const entry of entries) {
      if (!entry.isDirectory()) continue;

      const configPath = join(teamsDir, entry.name, "config.json");

      try {
        const configData = await readFile(configPath, "utf-8");
        const config: TeamConfigJSON = JSON.parse(configData);

        const summary = parseTeamSummary(entry.name, config);
        summaries.push(summary);
      } catch (error) {
        if (process.env["VERBOSE"]) {
          console.warn(
            `Skipping team ${entry.name}: ${
              error instanceof Error ? error.message : String(error)
            }`
          );
        }
        continue;
      }
    }

    // Sort by directory name descending (most recent first - dir names are timestamps)
    summaries.sort((a, b) => b.dir.localeCompare(a.dir));

    // Attach stream activity for running teams
    await Promise.all(
      summaries.filter((s) => s.status === "running").map((s) => attachStreamActivity(s, teamsDir))
    );

    return summaries;
  } catch (error) {
    // Missing teams directory is expected on first run
    // Log other errors for debugging
    if (process.env["VERBOSE"]) {
      console.warn(
        `[useTeams] Failed to read teams directory: ${
          error instanceof Error ? error.message : String(error)
        }`
      );
    }
    return [];
  }
}

/**
 * Polling-only hook - runs team polling and auto-switch logic
 * This hook should be called unconditionally in Layout.tsx to ensure polling
 * runs regardless of which right panel mode is active.
 *
 * Polling interval:
 * - 5s when any team is running
 * - 30s when all teams are completed/failed
 */
export function useTeamsPoller(): void {
  const { setTeams } = useStore();
  const isMountedRef = useRef(true);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const pollingRef = useRef(false);
  const currentIntervalRate = useRef(0);

  useEffect(() => {
    isMountedRef.current = true;

    const poll = async (): Promise<number> => {
      // Guard against concurrent poll() executions
      if (pollingRef.current) {
        return currentIntervalRate.current;
      }

      pollingRef.current = true;

      try {
        const sessionDir = process.env["GOGENT_SESSION_DIR"];

        if (!sessionDir) {
          if (isMountedRef.current) setTeams([]);
          return 30000; // Default to slow polling
        }

        const summaries = await getTeamSummaries(sessionDir);
        if (isMountedRef.current) {
          setTeams(summaries);

          // Auto-switch panel tracking when teams become active
          const hasRunning = summaries.some((t) => t.alive);
          if (hasRunning) {
            const state = useStore.getState();
            if (!state.panelAutoSwitched) {
              state.setPanelAutoSwitched(true);
            }
          }

          // Adaptive polling: 5s if any running, 30s otherwise
          return hasRunning ? 5000 : 30000;
        }

        return 30000;
      } finally {
        pollingRef.current = false;
      }
    };

    const setupInterval = (rate: number): void => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }

      intervalRef.current = setInterval(() => {
        void poll().then((nextRate) => {
          // Only recreate interval if rate changed
          if (nextRate !== currentIntervalRate.current) {
            currentIntervalRate.current = nextRate;
            setupInterval(nextRate);
          }
        }).catch(() => {
          // Filesystem errors during polling are non-fatal — next interval will retry
        });
      }, rate);

      currentIntervalRate.current = rate;
    };

    // Poll immediately on mount and setup interval
    void poll().then((initialRate) => {
      if (isMountedRef.current) {
        setupInterval(initialRate);
      }
    }).catch(() => {
      // Initial poll failure is non-fatal — interval will retry
    });

    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [setTeams]);
}

/**
 * Data-only hook - returns current teams from store
 * Components that need team data should call this
 */
export function useTeams(): TeamSummary[] {
  return useStore((state) => state.teams);
}

/**
 * Backward compatible useTeamCount wrapper
 * Returns count of alive teams (background_pid is alive)
 */
export function useTeamCount(): number {
  const teams = useStore((state) => state.teams);
  return teams.filter((t) => t.alive).length;
}
