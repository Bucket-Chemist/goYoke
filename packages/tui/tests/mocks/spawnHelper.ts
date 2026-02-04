import { spawn, ChildProcess } from "child_process";
import { createMockClaude, cleanupMockClaude, MockOptions } from "./mockClaude";

export interface SpawnResult {
  stdout: string;
  stderr: string;
  exitCode: number | null;
  killed: boolean;
  duration: number;
}

/**
 * Spawns the mock CLI and collects output.
 * Use for testing spawn_agent behavior.
 */
export async function spawnMockClaude(
  options: MockOptions,
  stdinContent?: string,
  timeout?: number
): Promise<SpawnResult> {
  const mockPath = await createMockClaude(options);
  const startTime = Date.now();

  return new Promise(async (resolve) => {
    let stdout = "";
    let stderr = "";
    let killed = false;
    let resolved = false;

    const proc = spawn(mockPath, [], {
      stdio: ["pipe", "pipe", "pipe"],
    });

    // Write stdin if provided, then close it
    if (stdinContent) {
      proc.stdin.write(stdinContent);
    }
    proc.stdin.end();

    proc.stdout.on("data", (data) => {
      stdout += data.toString();
    });

    proc.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    // Timeout handling
    let timer: NodeJS.Timeout | null = null;
    let killTimer: NodeJS.Timeout | null = null;
    if (timeout) {
      timer = setTimeout(() => {
        killed = true;
        proc.kill("SIGTERM");
        // Escalate to SIGKILL after 100ms (faster than 1s)
        killTimer = setTimeout(() => {
          if (!proc.killed) {
            proc.kill("SIGKILL");
          }
        }, 100);
      }, timeout);
    }

    const cleanup = async (code: number | null) => {
      if (resolved) return;
      resolved = true;

      if (timer) clearTimeout(timer);
      if (killTimer) clearTimeout(killTimer);
      await cleanupMockClaude(mockPath);

      resolve({
        stdout,
        stderr,
        exitCode: code,
        killed,
        duration: Date.now() - startTime,
      });
    };

    proc.on("close", cleanup);
    proc.on("exit", cleanup);
  });
}
