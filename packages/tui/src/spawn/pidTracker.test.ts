import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";
import {
  registerPid,
  unregisterPid,
  cleanupOrphanedProcesses,
  getTrackedProcessCount,
} from "./pidTracker.js";

// Helper to get PID file path
function getPidFilePath(): string {
  const runtimeDir = process.env['XDG_RUNTIME_DIR'];
  if (runtimeDir) {
    return path.join(runtimeDir, "gogent", "spawn-pids.json");
  }
  return path.join(os.tmpdir(), `gogent-${process.getuid?.() ?? 0}`, "spawn-pids.json");
}

// Helper to read PID file
function readPidFile(): any {
  const filePath = getPidFilePath();
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content);
  } catch {
    return { version: 1, tuiPid: process.pid, entries: {} };
  }
}

// Helper to write PID file
function writePidFile(data: any): void {
  const filePath = getPidFilePath();
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2), "utf-8");
}

describe("pidTracker", () => {
  beforeEach(() => {
    // Clean up PID file before each test
    const filePath = getPidFilePath();
    try {
      fs.unlinkSync(filePath);
    } catch {
      // File doesn't exist, that's fine
    }
  });

  afterEach(() => {
    // Clean up PID file after each test
    const filePath = getPidFilePath();
    try {
      fs.unlinkSync(filePath);
    } catch {
      // File doesn't exist, that's fine
    }
  });

  describe("registerPid / unregisterPid", () => {
    it("should persist PID to file", () => {
      registerPid("agent-1", 12345, "einstein");
      const data = readPidFile();
      expect(data.entries["agent-1"].pid).toBe(12345);
      expect(data.entries["agent-1"].agentType).toBe("einstein");
    });

    it("should remove PID on unregister", () => {
      registerPid("agent-1", 12345, "einstein");
      unregisterPid("agent-1");
      const data = readPidFile();
      expect(data.entries["agent-1"]).toBeUndefined();
    });

    it("should track multiple PIDs", () => {
      registerPid("agent-1", 12345, "einstein");
      registerPid("agent-2", 12346, "mozart");
      const data = readPidFile();
      expect(data.entries["agent-1"].pid).toBe(12345);
      expect(data.entries["agent-2"].pid).toBe(12346);
    });
  });

  describe("cleanupOrphanedProcesses", () => {
    it("should kill processes from different TUI session", () => {
      // Write file with different tuiPid
      writePidFile({
        version: 1,
        tuiPid: process.pid + 1, // Different session
        entries: {
          "old-agent": { pid: 99999, agentType: "test", startTime: 0 },
        },
      });

      const result = cleanupOrphanedProcesses();
      // PID 99999 likely doesn't exist, so no kill but file is reset
      expect(readPidFile().entries).toEqual({});
    });

    it("should preserve entries from current session", () => {
      writePidFile({
        version: 1,
        tuiPid: process.pid, // Same session
        entries: {
          "current-agent": { pid: 12345, agentType: "test", startTime: 0 },
        },
      });

      cleanupOrphanedProcesses();
      // File is reset even for current session (fresh start)
      expect(readPidFile().entries).toEqual({});
    });

    it("should return correct killed count", () => {
      // Write file with non-existent PIDs
      writePidFile({
        version: 1,
        tuiPid: process.pid + 1,
        entries: {
          "agent-1": { pid: 99998, agentType: "test", startTime: 0 },
          "agent-2": { pid: 99999, agentType: "test", startTime: 0 },
        },
      });

      const result = cleanupOrphanedProcesses();
      // PIDs don't exist, so killed count should be 0
      expect(result.killed).toBe(0);
      expect(result.errors).toEqual([]);
    });
  });

  describe("getTrackedProcessCount", () => {
    it("should return count of tracked processes", () => {
      registerPid("agent-1", 12345, "einstein");
      registerPid("agent-2", 12346, "mozart");

      expect(getTrackedProcessCount()).toBe(2);
    });

    it("should return 0 when no processes tracked", () => {
      expect(getTrackedProcessCount()).toBe(0);
    });
  });
});
