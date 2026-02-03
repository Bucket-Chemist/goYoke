/**
 * Integration tests for session persistence
 * Tests session CRUD operations and file format validation
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { promises as fs } from "fs";
import { join } from "path";
import { tmpdir } from "os";
import {
  listSessions,
  loadSession,
  saveSession,
  deleteSession,
} from "../../src/hooks/useSession.js";
import type { SessionData } from "../../src/store/types.js";

// Mock homedir to use test directory
const originalHomedir = process.env["HOME"];
const TEST_HOME = join(tmpdir(), `test-home-${Date.now()}`);
const TEST_SESSION_DIR = join(TEST_HOME, ".claude", "sessions");

beforeEach(async () => {
  // Set test home directory
  process.env["HOME"] = TEST_HOME;

  // Clean up test directory
  try {
    await fs.rm(TEST_HOME, { recursive: true, force: true });
  } catch (error) {
    // Ignore if doesn't exist
  }

  // Create fresh test directory structure
  await fs.mkdir(TEST_SESSION_DIR, { recursive: true });
});

afterEach(async () => {
  // Restore original homedir
  process.env["HOME"] = originalHomedir;

  // Clean up test directory
  try {
    await fs.rm(TEST_HOME, { recursive: true, force: true });
  } catch (error) {
    // Ignore cleanup errors
  }
});

describe("Session CRUD Operations", () => {
  it("should create and save a new session", async () => {
    const session: SessionData = {
      id: "test-session-1",
      name: "Test Session",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0.42,
      tool_calls: 127,
    };

    await saveSession(session);

    // Verify file exists
    const filePath = join(TEST_SESSION_DIR, "test-session-1.json");
    const exists = await fs
      .access(filePath)
      .then(() => true)
      .catch(() => false);
    expect(exists).toBe(true);

    // Verify file content
    const content = await fs.readFile(filePath, "utf-8");
    const saved = JSON.parse(content);
    expect(saved.id).toBe("test-session-1");
    expect(saved.name).toBe("Test Session");
    expect(saved.cost).toBe(0.42);
    expect(saved.tool_calls).toBe(127);
  });

  it("should load an existing session", async () => {
    const session: SessionData = {
      id: "test-session-2",
      name: "Load Test",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T11:30:00Z",
      cost: 1.23,
      tool_calls: 45,
    };

    await saveSession(session);

    const loaded = await loadSession("test-session-2");
    expect(loaded.id).toBe("test-session-2");
    expect(loaded.name).toBe("Load Test");
    expect(loaded.cost).toBe(1.23);
    expect(loaded.tool_calls).toBe(45);
  });

  it("should throw error when loading non-existent session", async () => {
    await expect(loadSession("non-existent")).rejects.toThrow(
      "Session not found: non-existent"
    );
  });

  it("should delete a session", async () => {
    const session: SessionData = {
      id: "test-session-3",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0,
      tool_calls: 0,
    };

    await saveSession(session);
    await deleteSession("test-session-3");

    // Verify file is gone
    const filePath = join(TEST_SESSION_DIR, "test-session-3.json");
    const exists = await fs
      .access(filePath)
      .then(() => true)
      .catch(() => false);
    expect(exists).toBe(false);
  });

  it("should throw error when deleting non-existent session", async () => {
    await expect(deleteSession("non-existent")).rejects.toThrow(
      "Session not found: non-existent"
    );
  });
});

describe("Session Listing", () => {
  it("should list all sessions sorted by last_used", async () => {
    // Create sessions with delays to ensure different last_used timestamps
    // Note: saveSession updates last_used to current time
    await saveSession({
      id: "session-1",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0.1,
      tool_calls: 10,
    });

    // Small delay to ensure different timestamps
    await new Promise((resolve) => setTimeout(resolve, 10));

    await saveSession({
      id: "session-2",
      created_at: "2026-02-01T11:00:00Z",
      last_used: "2026-02-01T11:00:00Z",
      cost: 0.2,
      tool_calls: 20,
    });

    await new Promise((resolve) => setTimeout(resolve, 10));

    await saveSession({
      id: "session-3",
      created_at: "2026-02-01T09:00:00Z",
      last_used: "2026-02-01T09:00:00Z",
      cost: 0.3,
      tool_calls: 30,
    });

    const listed = await listSessions();
    expect(listed).toHaveLength(3);

    // Verify sorted by last_used descending (most recent first)
    // Since saveSession updates last_used, session-3 was saved last
    expect(listed[0].id).toBe("session-3"); // Most recently saved
    expect(listed[1].id).toBe("session-2");
    expect(listed[2].id).toBe("session-1"); // Least recently saved
  });

  it("should return empty array when no sessions exist", async () => {
    const listed = await listSessions();
    expect(listed).toHaveLength(0);
  });

  it("should skip invalid JSON files gracefully", async () => {
    // Create valid session
    const validSession: SessionData = {
      id: "valid-session",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0.5,
      tool_calls: 50,
    };
    await saveSession(validSession);

    // Create invalid JSON file
    const invalidPath = join(TEST_SESSION_DIR, "invalid.json");
    await fs.writeFile(invalidPath, "{invalid json}", "utf-8");

    // Create JSON with missing fields
    const incompletePath = join(TEST_SESSION_DIR, "incomplete.json");
    await fs.writeFile(incompletePath, JSON.stringify({ id: "test" }), "utf-8");

    const listed = await listSessions();

    // Should only return valid session, skipping invalid files
    expect(listed).toHaveLength(1);
    expect(listed[0].id).toBe("valid-session");
  });

  it("should ignore non-JSON files", async () => {
    // Create valid session
    const session: SessionData = {
      id: "test-session",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0.1,
      tool_calls: 10,
    };
    await saveSession(session);

    // Create non-JSON file
    const txtPath = join(TEST_SESSION_DIR, "readme.txt");
    await fs.writeFile(txtPath, "This is not a session file", "utf-8");

    const listed = await listSessions();
    expect(listed).toHaveLength(1);
    expect(listed[0].id).toBe("test-session");
  });
});

describe("File Format Validation", () => {
  it("should use snake_case field names for Go compatibility", async () => {
    const session: SessionData = {
      id: "format-test",
      name: "Format Test",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0.42,
      tool_calls: 127,
    };

    await saveSession(session);

    const filePath = join(TEST_SESSION_DIR, "format-test.json");
    const content = await fs.readFile(filePath, "utf-8");
    const saved = JSON.parse(content);

    // Verify snake_case field names (not camelCase)
    expect(saved).toHaveProperty("created_at");
    expect(saved).toHaveProperty("last_used");
    expect(saved).toHaveProperty("tool_calls");

    // Verify no camelCase variants exist
    expect(saved).not.toHaveProperty("createdAt");
    expect(saved).not.toHaveProperty("lastUsed");
    expect(saved).not.toHaveProperty("toolCalls");
  });

  it("should use ISO 8601 dates with Z suffix", async () => {
    const session: SessionData = {
      id: "date-test",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T11:30:00Z",
      cost: 0,
      tool_calls: 0,
    };

    await saveSession(session);

    const filePath = join(TEST_SESSION_DIR, "date-test.json");
    const content = await fs.readFile(filePath, "utf-8");
    const saved = JSON.parse(content);

    // Verify ISO 8601 format with Z suffix
    expect(saved.created_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$/);
    expect(saved.last_used).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$/);
  });

  it("should update last_used timestamp on save", async () => {
    const session: SessionData = {
      id: "timestamp-test",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0,
      tool_calls: 0,
    };

    const beforeSave = new Date();
    await saveSession(session);
    const afterSave = new Date();

    const loaded = await loadSession("timestamp-test");
    const lastUsed = new Date(loaded.last_used);

    // Verify last_used was updated to current time
    expect(lastUsed.getTime()).toBeGreaterThanOrEqual(beforeSave.getTime());
    expect(lastUsed.getTime()).toBeLessThanOrEqual(afterSave.getTime());
  });

  it("should handle optional name field", async () => {
    const sessionWithName: SessionData = {
      id: "named-session",
      name: "Test Name",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0,
      tool_calls: 0,
    };

    const sessionWithoutName: SessionData = {
      id: "unnamed-session",
      created_at: "2026-02-01T10:00:00Z",
      last_used: "2026-02-01T10:00:00Z",
      cost: 0,
      tool_calls: 0,
    };

    await saveSession(sessionWithName);
    await saveSession(sessionWithoutName);

    const loaded1 = await loadSession("named-session");
    const loaded2 = await loadSession("unnamed-session");

    expect(loaded1.name).toBe("Test Name");
    expect(loaded2.name).toBeUndefined();
  });
});

describe("Error Handling", () => {
  it("should reject invalid session format on load", async () => {
    // Create session with missing required fields
    const invalidPath = join(TEST_SESSION_DIR, "invalid-format.json");
    await fs.writeFile(
      invalidPath,
      JSON.stringify({
        id: "invalid-format",
        // missing created_at, last_used, cost, tool_calls
      }),
      "utf-8"
    );

    await expect(loadSession("invalid-format")).rejects.toThrow(
      "Invalid session format: invalid-format"
    );
  });

  it("should handle missing session directory gracefully", async () => {
    // Remove test directory
    await fs.rm(TEST_SESSION_DIR, { recursive: true, force: true });

    const listed = await listSessions();
    expect(listed).toHaveLength(0);
  });
});
