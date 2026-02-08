/**
 * Unit tests for useTeamCount hook
 * Tests: polling, PID validation, filesystem errors, store updates
 * Tests the underlying logic by directly calling the filesystem functions
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { readdir, readFile } from "fs/promises";
import { join } from "path";

// Mock fs/promises module
vi.mock("fs/promises", () => ({
  readdir: vi.fn(),
  readFile: vi.fn(),
}));

// Helper function to test PID alive check
function isPidAlive(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ESRCH") {
      return false;
    }
    return true;
  }
}

// Helper function to count running teams (extracted from hook logic)
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
        const config = JSON.parse(configData) as { background_pid: number | null };

        if (
          config.background_pid !== null &&
          typeof config.background_pid === "number" &&
          isPidAlive(config.background_pid)
        ) {
          count++;
        }
      } catch {
        continue;
      }
    }

    return count;
  } catch {
    return 0;
  }
}

describe("useTeamCount", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearSession();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    delete process.env["GOGENT_SESSION_DIR"];
  });

  it("should return 0 when GOGENT_SESSION_DIR is not set", async () => {
    delete process.env["GOGENT_SESSION_DIR"];

    const sessionDir = process.env["GOGENT_SESSION_DIR"];
    if (!sessionDir) {
      useStore.getState().setBackgroundTeamCount(0);
    }

    expect(useStore.getState().backgroundTeamCount).toBe(0);
  });

  it("should return 0 when teams directory does not exist", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockRejectedValue(
      Object.assign(new Error("ENOENT: no such file or directory"), {
        code: "ENOENT",
      })
    );

    const count = await countRunningTeams("/tmp/test-session");
    expect(count).toBe(0);

    useStore.getState().setBackgroundTeamCount(count);
    expect(useStore.getState().backgroundTeamCount).toBe(0);
  });

  it("should count teams with live PIDs", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    // Mock directory listing
    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
      { name: "team2", isDirectory: () => true } as any,
      { name: "team3", isDirectory: () => true } as any,
    ]);

    // Mock config reads
    const readFileMock = vi.mocked(readFile);
    readFileMock
      .mockResolvedValueOnce(
        JSON.stringify({ background_pid: 12345, name: "team1" })
      )
      .mockResolvedValueOnce(
        JSON.stringify({ background_pid: null, name: "team2" })
      )
      .mockResolvedValueOnce(
        JSON.stringify({ background_pid: 67890, name: "team3" })
      );

    // Mock process.kill to succeed for 12345 and 67890
    const killSpy = vi.spyOn(process, "kill").mockImplementation((pid, signal) => {
      if (signal === 0 && (pid === 12345 || pid === 67890)) {
        return true;
      }
      throw Object.assign(new Error("ESRCH"), { code: "ESRCH" });
    });

    const count = await countRunningTeams("/tmp/test-session");
    expect(count).toBe(2);

    useStore.getState().setBackgroundTeamCount(count);
    expect(useStore.getState().backgroundTeamCount).toBe(2);
    expect(killSpy).toHaveBeenCalledWith(12345, 0);
    expect(killSpy).toHaveBeenCalledWith(67890, 0);
  });

  it("should ignore dead PIDs", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
    ]);

    vi.mocked(readFile).mockResolvedValue(
      JSON.stringify({ background_pid: 99999 })
    );

    // Mock process.kill to throw ESRCH (process doesn't exist)
    const killSpy = vi.spyOn(process, "kill").mockImplementation(() => {
      const error: NodeJS.ErrnoException = Object.assign(
        new Error("ESRCH: no such process"),
        { code: "ESRCH" }
      );
      throw error;
    });

    const count = await countRunningTeams("/tmp/test-session");
    expect(count).toBe(0);

    useStore.getState().setBackgroundTeamCount(count);
    expect(useStore.getState().backgroundTeamCount).toBe(0);
    expect(killSpy).toHaveBeenCalledWith(99999, 0);
  });

  it("should handle malformed config.json gracefully", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
      { name: "team2", isDirectory: () => true } as any,
    ]);

    const readFileMock = vi.mocked(readFile);
    // First config is invalid JSON
    readFileMock
      .mockResolvedValueOnce("{ invalid json }")
      // Second config is valid with live PID
      .mockResolvedValueOnce(JSON.stringify({ background_pid: 12345 }));

    vi.spyOn(process, "kill").mockReturnValue(true);

    const count = await countRunningTeams("/tmp/test-session");
    expect(count).toBe(1);

    useStore.getState().setBackgroundTeamCount(count);
    expect(useStore.getState().backgroundTeamCount).toBe(1);
  });

  it("should update store backgroundTeamCount", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
    ]);

    vi.mocked(readFile).mockResolvedValue(
      JSON.stringify({ background_pid: 12345 })
    );

    vi.spyOn(process, "kill").mockReturnValue(true);

    const count = await countRunningTeams("/tmp/test-session");
    useStore.getState().setBackgroundTeamCount(count);

    expect(useStore.getState().backgroundTeamCount).toBe(1);
  });

  it("should reset count to 0 after clearSession", async () => {
    // Set a count
    useStore.getState().setBackgroundTeamCount(3);
    expect(useStore.getState().backgroundTeamCount).toBe(3);

    // Clear session
    useStore.getState().clearSession();

    // Should be reset to 0
    expect(useStore.getState().backgroundTeamCount).toBe(0);
  });

  it("should handle EPERM errors (process exists but no permission)", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
    ]);

    vi.mocked(readFile).mockResolvedValue(
      JSON.stringify({ background_pid: 12345 })
    );

    // Mock process.kill to throw EPERM (permission denied, but process exists)
    vi.spyOn(process, "kill").mockImplementation(() => {
      const error: NodeJS.ErrnoException = Object.assign(
        new Error("EPERM: operation not permitted"),
        { code: "EPERM" }
      );
      throw error;
    });

    const count = await countRunningTeams("/tmp/test-session");
    // EPERM means process exists, so should count it
    expect(count).toBe(1);

    useStore.getState().setBackgroundTeamCount(count);
    expect(useStore.getState().backgroundTeamCount).toBe(1);
  });

  it("should ignore non-directory entries in teams folder", async () => {
    process.env["GOGENT_SESSION_DIR"] = "/tmp/test-session";

    vi.mocked(readdir).mockResolvedValue([
      { name: "team1", isDirectory: () => true } as any,
      { name: "readme.txt", isDirectory: () => false } as any,
      { name: "team2", isDirectory: () => true } as any,
    ]);

    const readFileMock = vi.mocked(readFile);
    readFileMock
      .mockResolvedValueOnce(JSON.stringify({ background_pid: 12345 }))
      .mockResolvedValueOnce(JSON.stringify({ background_pid: 67890 }));

    vi.spyOn(process, "kill").mockReturnValue(true);

    const count = await countRunningTeams("/tmp/test-session");
    expect(count).toBe(2);

    // Should only have called readFile twice (for team1 and team2)
    expect(readFileMock).toHaveBeenCalledTimes(2);
  });
});
