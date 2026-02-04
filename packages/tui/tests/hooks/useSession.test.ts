/**
 * Tests for useSession hook
 * Verifies session persistence, ID generation, and error recovery
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { promises as fs } from "fs";
import { join } from "path";
import { tmpdir } from "os";
import {
  useSession,
  listSessions,
  loadSession,
  saveSession,
  deleteSession,
} from "../../src/hooks/useSession.js";
import type { SessionData } from "../../src/store/types.js";

// Mock HOME directory for testing
const TEST_HOME = join(tmpdir(), `gofortress-session-test-${Date.now()}`);
const TEST_SESSION_DIR = join(TEST_HOME, ".claude", "sessions");

describe("useSession", () => {
  beforeEach(async () => {
    // Set test HOME
    process.env["HOME"] = TEST_HOME;

    // Create session directory
    await fs.mkdir(TEST_SESSION_DIR, { recursive: true });
  });

  afterEach(async () => {
    // Clean up test directory
    try {
      await fs.rm(TEST_HOME, { recursive: true, force: true });
    } catch (error) {
      // Ignore cleanup errors
    }

    // Restore original HOME
    delete process.env["HOME"];
  });

  /**
   * Helper to create a valid session object
   */
  function createSession(overrides: Partial<SessionData> = {}): SessionData {
    return {
      id: `test-session-${Date.now()}-${Math.random()}`,
      name: "Test Session",
      created_at: new Date().toISOString(),
      last_used: new Date().toISOString(),
      cost: 0.42,
      tool_calls: 127,
      ...overrides,
    };
  }

  describe("Session Hook Interface", () => {
    it("should return all session operations", () => {
      const session = useSession();

      expect(session).toHaveProperty("listSessions");
      expect(session).toHaveProperty("loadSession");
      expect(session).toHaveProperty("saveSession");
      expect(session).toHaveProperty("deleteSession");
      expect(typeof session.listSessions).toBe("function");
      expect(typeof session.loadSession).toBe("function");
      expect(typeof session.saveSession).toBe("function");
      expect(typeof session.deleteSession).toBe("function");
    });
  });

  describe("saveSession", () => {
    it("should save session to file with Go-compatible format", async () => {
      const session = createSession({
        id: "test-123",
        name: "My Session",
        cost: 1.23,
        tool_calls: 42,
      });

      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "test-123.json");
      const content = await fs.readFile(filePath, "utf-8");
      const saved = JSON.parse(content) as SessionData;

      expect(saved.id).toBe("test-123");
      expect(saved.name).toBe("My Session");
      expect(saved.cost).toBe(1.23);
      expect(saved.tool_calls).toBe(42);
      expect(saved.created_at).toBe(session.created_at);
      // last_used should be updated on save
      expect(new Date(saved.last_used).getTime()).toBeGreaterThanOrEqual(
        new Date(session.last_used).getTime()
      );
    });

    it("should use snake_case field names for Go compatibility", async () => {
      const session = createSession({ id: "test-snake" });
      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "test-snake.json");
      const content = await fs.readFile(filePath, "utf-8");

      expect(content).toContain('"created_at"');
      expect(content).toContain('"last_used"');
      expect(content).toContain('"tool_calls"');
      expect(content).not.toContain('"createdAt"');
      expect(content).not.toContain('"lastUsed"');
      expect(content).not.toContain('"toolCalls"');
    });

    it("should update last_used timestamp on save", async () => {
      const session = createSession({
        id: "test-timestamp",
        last_used: "2020-01-01T00:00:00Z",
      });

      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "test-timestamp.json");
      const content = await fs.readFile(filePath, "utf-8");
      const saved = JSON.parse(content) as SessionData;

      expect(new Date(saved.last_used).getTime()).toBeGreaterThan(
        new Date("2020-01-01T00:00:00Z").getTime()
      );
    });

    it("should create session directory if it doesn't exist", async () => {
      await fs.rm(TEST_SESSION_DIR, { recursive: true });

      const session = createSession({ id: "test-mkdir" });
      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "test-mkdir.json");
      const exists = await fs
        .access(filePath)
        .then(() => true)
        .catch(() => false);

      expect(exists).toBe(true);
    });

    it("should overwrite existing session file", async () => {
      const session = createSession({ id: "test-overwrite", cost: 1.0 });
      await saveSession(session);

      session.cost = 2.5;
      await saveSession(session);

      const loaded = await loadSession("test-overwrite");
      expect(loaded.cost).toBe(2.5);
    });

    it("should handle sessions without optional name field", async () => {
      const session = createSession({ id: "test-no-name", name: undefined });
      await saveSession(session);

      const loaded = await loadSession("test-no-name");
      expect(loaded.name).toBeUndefined();
    });
  });

  describe("loadSession", () => {
    it("should load existing session", async () => {
      const session = createSession({
        id: "test-load",
        name: "Load Test",
        cost: 3.14,
        tool_calls: 256,
      });
      await saveSession(session);

      const loaded = await loadSession("test-load");

      expect(loaded.id).toBe("test-load");
      expect(loaded.name).toBe("Load Test");
      expect(loaded.cost).toBe(3.14);
      expect(loaded.tool_calls).toBe(256);
    });

    it("should throw error for non-existent session", async () => {
      await expect(loadSession("non-existent")).rejects.toThrow(
        "Session not found: non-existent"
      );
    });

    it("should throw error for invalid session format", async () => {
      const filePath = join(TEST_SESSION_DIR, "invalid.json");
      await fs.writeFile(
        filePath,
        JSON.stringify({ id: "invalid" }) // Missing required fields
      );

      await expect(loadSession("invalid")).rejects.toThrow(
        "Invalid session format: invalid"
      );
    });

    it("should validate required fields", async () => {
      const invalidSessions = [
        { created_at: "2020-01-01T00:00:00Z", last_used: "2020-01-01T00:00:00Z", cost: 0, tool_calls: 0 }, // Missing id
        { id: "test", last_used: "2020-01-01T00:00:00Z", cost: 0, tool_calls: 0 }, // Missing created_at
        { id: "test", created_at: "2020-01-01T00:00:00Z", cost: 0, tool_calls: 0 }, // Missing last_used
        { id: "test", created_at: "2020-01-01T00:00:00Z", last_used: "2020-01-01T00:00:00Z", tool_calls: 0 }, // Missing cost
        { id: "test", created_at: "2020-01-01T00:00:00Z", last_used: "2020-01-01T00:00:00Z", cost: 0 }, // Missing tool_calls
      ];

      for (let i = 0; i < invalidSessions.length; i++) {
        const id = `invalid-${i}`;
        const filePath = join(TEST_SESSION_DIR, `${id}.json`);
        await fs.writeFile(filePath, JSON.stringify(invalidSessions[i]));

        await expect(loadSession(id)).rejects.toThrow(
          `Invalid session format: ${id}`
        );
      }
    });

    it("should handle corrupted JSON gracefully", async () => {
      const filePath = join(TEST_SESSION_DIR, "corrupted.json");
      await fs.writeFile(filePath, "{invalid json}");

      await expect(loadSession("corrupted")).rejects.toThrow();
    });
  });

  describe("listSessions", () => {
    it("should return empty array when no sessions exist", async () => {
      const sessions = await listSessions();
      expect(sessions).toEqual([]);
    });

    it("should return all valid sessions", async () => {
      const session1 = createSession({ id: "session-1", name: "First" });
      const session2 = createSession({ id: "session-2", name: "Second" });
      const session3 = createSession({ id: "session-3", name: "Third" });

      await saveSession(session1);
      await saveSession(session2);
      await saveSession(session3);

      const sessions = await listSessions();

      expect(sessions.length).toBe(3);
      expect(sessions.map((s) => s.id)).toContain("session-1");
      expect(sessions.map((s) => s.id)).toContain("session-2");
      expect(sessions.map((s) => s.id)).toContain("session-3");
    });

    it("should sort by last_used descending (most recent first)", async () => {
      const now = Date.now();
      const session1 = createSession({
        id: "oldest",
        last_used: new Date(now - 3000).toISOString(),
      });
      const session2 = createSession({
        id: "newest",
        last_used: new Date(now).toISOString(),
      });
      const session3 = createSession({
        id: "middle",
        last_used: new Date(now - 1000).toISOString(),
      });

      await saveSession(session1);
      await saveSession(session2);
      await saveSession(session3);

      const sessions = await listSessions();

      expect(sessions[0].id).toBe("newest");
      expect(sessions[1].id).toBe("middle");
      expect(sessions[2].id).toBe("oldest");
    });

    it("should skip invalid JSON files gracefully", async () => {
      const validSession = createSession({ id: "valid" });
      await saveSession(validSession);

      // Create invalid files
      await fs.writeFile(join(TEST_SESSION_DIR, "invalid.json"), "{bad json}");
      await fs.writeFile(join(TEST_SESSION_DIR, "missing-fields.json"), "{}");

      const sessions = await listSessions();

      // Should only return valid session
      expect(sessions.length).toBe(1);
      expect(sessions[0].id).toBe("valid");
    });

    it("should skip non-JSON files", async () => {
      const session = createSession({ id: "test" });
      await saveSession(session);

      await fs.writeFile(join(TEST_SESSION_DIR, "readme.txt"), "Not JSON");
      await fs.writeFile(join(TEST_SESSION_DIR, "backup.bak"), "{}");

      const sessions = await listSessions();

      expect(sessions.length).toBe(1);
      expect(sessions[0].id).toBe("test");
    });

    it("should handle empty session directory", async () => {
      await fs.rm(TEST_SESSION_DIR, { recursive: true });

      const sessions = await listSessions();
      expect(sessions).toEqual([]);
    });

    it("should log skipped files in verbose mode", async () => {
      process.env["VERBOSE"] = "true";
      const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation();

      await fs.writeFile(join(TEST_SESSION_DIR, "bad.json"), "{invalid}");
      await listSessions();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining("Skipping invalid session file: bad.json"),
        expect.anything()
      );

      consoleErrorSpy.mockRestore();
      delete process.env["VERBOSE"];
    });
  });

  describe("deleteSession", () => {
    it("should delete existing session", async () => {
      const session = createSession({ id: "to-delete" });
      await saveSession(session);

      await deleteSession("to-delete");

      await expect(loadSession("to-delete")).rejects.toThrow(
        "Session not found: to-delete"
      );
    });

    it("should throw error for non-existent session", async () => {
      await expect(deleteSession("non-existent")).rejects.toThrow(
        "Session not found: non-existent"
      );
    });

    it("should remove session from list", async () => {
      const session1 = createSession({ id: "keep" });
      const session2 = createSession({ id: "delete" });

      await saveSession(session1);
      await saveSession(session2);

      await deleteSession("delete");

      const sessions = await listSessions();
      expect(sessions.length).toBe(1);
      expect(sessions[0].id).toBe("keep");
    });
  });

  describe("Session ID Generation", () => {
    it("should handle UUIDs correctly", async () => {
      const session = createSession({
        id: "550e8400-e29b-41d4-a716-446655440000",
      });

      await saveSession(session);
      const loaded = await loadSession("550e8400-e29b-41d4-a716-446655440000");

      expect(loaded.id).toBe("550e8400-e29b-41d4-a716-446655440000");
    });

    it("should handle custom ID formats", async () => {
      const session = createSession({ id: "session-2026-02-04-001" });

      await saveSession(session);
      const loaded = await loadSession("session-2026-02-04-001");

      expect(loaded.id).toBe("session-2026-02-04-001");
    });

    it("should handle IDs with special characters", async () => {
      const session = createSession({ id: "test_session-123.v2" });

      await saveSession(session);
      const loaded = await loadSession("test_session-123.v2");

      expect(loaded.id).toBe("test_session-123.v2");
    });
  });

  describe("Error Recovery", () => {
    it("should handle filesystem errors during save", async () => {
      const session = createSession({ id: "test" });

      // Make directory read-only to trigger EACCES
      await fs.chmod(TEST_SESSION_DIR, 0o444);

      await expect(saveSession(session)).rejects.toThrow();

      // Restore permissions for cleanup
      await fs.chmod(TEST_SESSION_DIR, 0o755);
    });

    it("should handle concurrent saves to same session", async () => {
      const session1 = createSession({ id: "concurrent", cost: 1.0 });
      const session2 = { ...session1, cost: 2.0 };

      // Save both concurrently
      await Promise.all([saveSession(session1), saveSession(session2)]);

      const loaded = await loadSession("concurrent");
      // One of them should win
      expect([1.0, 2.0]).toContain(loaded.cost);
    });

    it("should handle malformed timestamps gracefully", async () => {
      const filePath = join(TEST_SESSION_DIR, "bad-timestamp.json");
      await fs.writeFile(
        filePath,
        JSON.stringify({
          id: "bad-timestamp",
          created_at: "invalid-date",
          last_used: "2020-01-01T00:00:00Z",
          cost: 0,
          tool_calls: 0,
        })
      );

      // Should load without throwing (even if timestamps are invalid)
      const loaded = await loadSession("bad-timestamp");
      expect(loaded.created_at).toBe("invalid-date");
    });

    it("should handle negative cost and tool_calls", async () => {
      const session = createSession({
        id: "negative",
        cost: -1.0,
        tool_calls: -5,
      });

      await saveSession(session);
      const loaded = await loadSession("negative");

      // Should preserve values even if they're negative
      expect(loaded.cost).toBe(-1.0);
      expect(loaded.tool_calls).toBe(-5);
    });

    it("should handle very large cost and tool_calls", async () => {
      const session = createSession({
        id: "large",
        cost: 999999.99,
        tool_calls: 1000000,
      });

      await saveSession(session);
      const loaded = await loadSession("large");

      expect(loaded.cost).toBe(999999.99);
      expect(loaded.tool_calls).toBe(1000000);
    });
  });

  describe("Go Format Compatibility", () => {
    it("should match Go struct field names exactly", async () => {
      const session = createSession({ id: "go-compat" });
      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "go-compat.json");
      const content = await fs.readFile(filePath, "utf-8");
      const parsed = JSON.parse(content);

      // Verify Go struct field names
      expect(parsed).toHaveProperty("id");
      expect(parsed).toHaveProperty("created_at");
      expect(parsed).toHaveProperty("last_used");
      expect(parsed).toHaveProperty("cost");
      expect(parsed).toHaveProperty("tool_calls");
    });

    it("should use ISO8601 format with Z suffix", async () => {
      const session = createSession({ id: "iso8601" });
      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "iso8601.json");
      const content = await fs.readFile(filePath, "utf-8");
      const parsed = JSON.parse(content);

      expect(parsed.created_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
      expect(parsed.last_used).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
    });

    it("should use 2-space indentation for readability", async () => {
      const session = createSession({ id: "format" });
      await saveSession(session);

      const filePath = join(TEST_SESSION_DIR, "format.json");
      const content = await fs.readFile(filePath, "utf-8");

      // Verify 2-space indentation
      expect(content).toContain('{\n  "id"');
      expect(content).toContain('\n  "created_at"');
    });
  });

  describe("State Persistence", () => {
    it("should persist all session data across save/load cycle", async () => {
      const original = createSession({
        id: "persist-test",
        name: "Persistence Test",
        cost: 12.34,
        tool_calls: 567,
        created_at: "2026-01-01T10:00:00.000Z",
        last_used: "2026-02-01T15:30:00.000Z",
      });

      await saveSession(original);
      const loaded = await loadSession("persist-test");

      expect(loaded.id).toBe(original.id);
      expect(loaded.name).toBe(original.name);
      expect(loaded.cost).toBe(original.cost);
      expect(loaded.tool_calls).toBe(original.tool_calls);
      expect(loaded.created_at).toBe(original.created_at);
      // last_used is updated on save, so just check it exists
      expect(loaded.last_used).toBeTruthy();
    });

    it("should maintain session order after multiple saves", async () => {
      const sessions = [
        createSession({ id: "s1", name: "First" }),
        createSession({ id: "s2", name: "Second" }),
        createSession({ id: "s3", name: "Third" }),
      ];

      for (const session of sessions) {
        await saveSession(session);
        // Small delay to ensure different last_used timestamps
        await new Promise((resolve) => setTimeout(resolve, 10));
      }

      const listed = await listSessions();

      // Most recently saved (s3) should be first
      expect(listed[0].id).toBe("s3");
      expect(listed[1].id).toBe("s2");
      expect(listed[2].id).toBe("s1");
    });
  });
});
