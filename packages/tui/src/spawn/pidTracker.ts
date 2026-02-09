import * as fs from "fs";
import * as path from "path";
import * as os from "os";

interface PidEntry {
  pid: number;
  agentType: string;
  startTime: number;
}

interface PidFile {
  version: 1;
  tuiPid: number;
  entries: Record<string, PidEntry>;
}

const PID_FILE_NAME = "spawn-pids.json";

function getPidFilePath(): string {
  const runtimeDir = process.env['XDG_RUNTIME_DIR'];
  if (runtimeDir) {
    const dir = path.join(runtimeDir, "gogent");
    fs.mkdirSync(dir, { recursive: true });
    return path.join(dir, PID_FILE_NAME);
  }
  // Fallback for systems without XDG_RUNTIME_DIR
  const fallbackDir = path.join(os.tmpdir(), `gogent-${process.getuid?.() ?? 0}`);
  fs.mkdirSync(fallbackDir, { recursive: true });
  return path.join(fallbackDir, PID_FILE_NAME);
}

function readPidFile(): PidFile {
  const filePath = getPidFilePath();
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content) as PidFile;
  } catch {
    return { version: 1, tuiPid: process.pid, entries: {} };
  }
}

function writePidFile(data: PidFile): void {
  const filePath = getPidFilePath();
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2), "utf-8");
}

/**
 * Register a spawned process PID for orphan tracking.
 * Call immediately after spawn() succeeds.
 */
export function registerPid(agentId: string, pid: number, agentType: string): void {
  const data = readPidFile();
  data.tuiPid = process.pid;
  data.entries[agentId] = {
    pid,
    agentType,
    startTime: Date.now(),
  };
  writePidFile(data);
}

/**
 * Unregister a PID when process exits normally.
 */
export function unregisterPid(agentId: string): void {
  const data = readPidFile();
  delete data.entries[agentId];
  writePidFile(data);
}

/**
 * Check if a process is still running.
 */
function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0); // Signal 0 = check existence
    return true;
  } catch {
    return false;
  }
}

/**
 * Kill orphaned processes from previous TUI sessions.
 * Call at TUI startup BEFORE registering MCP server.
 */
export function cleanupOrphanedProcesses(): { killed: number; errors: string[] } {
  const data = readPidFile();
  const errors: string[] = [];
  let killed = 0;

  // If TUI PID matches, this is a restart - file is stale
  // If TUI PID doesn't match, previous TUI crashed
  if (data.tuiPid !== process.pid) {
    for (const [agentId, entry] of Object.entries(data.entries)) {
      if (isProcessRunning(entry.pid)) {
        try {
          // SIGTERM first
          process.kill(entry.pid, "SIGTERM");
          console.log(`[pidTracker] Killed orphaned process ${entry.pid} (${entry.agentType})`);
          killed++;

          // Schedule SIGKILL fallback
          setTimeout(() => {
            if (isProcessRunning(entry.pid)) {
              try {
                process.kill(entry.pid, "SIGKILL");
                console.log(`[pidTracker] Force-killed ${entry.pid}`);
              } catch {
                // Already dead
              }
            }
          }, 5000);
        } catch (err) {
          errors.push(`Failed to kill PID ${entry.pid}: ${err}`);
        }
      }
    }
  }

  // Reset file for this session
  writePidFile({ version: 1, tuiPid: process.pid, entries: {} });

  return { killed, errors };
}

/**
 * Get current orphan-trackable process count.
 */
export function getTrackedProcessCount(): number {
  return Object.keys(readPidFile().entries).length;
}
