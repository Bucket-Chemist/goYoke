/**
 * useTeamCount hook
 * Polls the session teams directory to count running background teams
 * Updates store.backgroundTeamCount every pollIntervalMs
 */

import { useEffect } from "react";
import { readdir, readFile } from "fs/promises";
import { join } from "path";
import { useStore } from "../store/index.js";

interface TeamConfig {
  background_pid: number | null;
  [key: string]: unknown;
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

/**
 * Count running teams in the session directory
 * Returns 0 on any error (missing directory, invalid JSON, etc.)
 */
async function countRunningTeams(sessionDir: string): Promise<number> {
  try {
    const teamsDir = join(sessionDir, "teams");
    const entries = await readdir(teamsDir, { withFileTypes: true });

    let count = 0;

    for (const entry of entries) {
      if (!entry.isDirectory()) continue;

      const configPath = join(teamsDir, entry.name, "config.json");

      try {
        const configData = await readFile(configPath, "utf-8");
        const raw: unknown = JSON.parse(configData);
        if (!raw || typeof raw !== "object" || !("background_pid" in raw)) {
          continue;
        }
        const config = raw as TeamConfig;

        if (
          config.background_pid !== null &&
          typeof config.background_pid === "number" &&
          isPidAlive(config.background_pid)
        ) {
          count++;
        }
      } catch (error) {
        if (process.env["VERBOSE"]) {
          console.warn(`Skipping team ${entry.name}: ${error instanceof Error ? error.message : String(error)}`);
        }
        continue;
      }
    }

    return count;
  } catch {
    // Missing teams directory or other filesystem error
    return 0;
  }
}

/**
 * Hook to poll and track running background teams
 *
 * @param pollIntervalMs - Polling interval in milliseconds (default 30000)
 * @returns Current count of running background teams
 */
export function useTeamCount(pollIntervalMs = 30000): number {
  const { backgroundTeamCount, setBackgroundTeamCount } = useStore();

  useEffect(() => {
    let isMounted = true;

    const poll = async (): Promise<void> => {
      const sessionDir = process.env["GOGENT_SESSION_DIR"];

      if (!sessionDir) {
        if (isMounted) setBackgroundTeamCount(0);
        return;
      }

      const count = await countRunningTeams(sessionDir);
      if (isMounted) setBackgroundTeamCount(count);
    };

    // Poll immediately on mount
    void poll();

    // Then poll on interval
    const interval = setInterval(() => {
      void poll();
    }, pollIntervalMs);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [pollIntervalMs, setBackgroundTeamCount]);

  return backgroundTeamCount;
}
