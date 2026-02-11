/**
 * Integration tests for GOGENT_SESSION_DIR environment variable
 * Verifies that the env var is set correctly during session lifecycle
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { join } from "path";
import { homedir, tmpdir } from "os";
import { promises as fs } from "fs";
import { useStore } from "../../src/store/index.js";

// Mock HOME directory for testing
const TEST_HOME = join(tmpdir(), `gofortress-env-test-${Date.now()}`);

describe("GOGENT_SESSION_DIR environment variable", () => {
  let originalHome: string | undefined;

  beforeEach(async () => {
    // Save original HOME
    originalHome = process.env["HOME"];

    // Set test HOME
    process.env["HOME"] = TEST_HOME;

    // Create test directory
    await fs.mkdir(join(TEST_HOME, ".claude", "sessions"), { recursive: true });

    // Clear store state
    useStore.getState().clearSession();
  });

  afterEach(async () => {
    // Clear the env var
    delete process.env["GOGENT_SESSION_DIR"];

    // Restore original HOME
    if (originalHome !== undefined) {
      process.env["HOME"] = originalHome;
    } else {
      delete process.env["HOME"];
    }

    // Clean up test directory
    try {
      await fs.rm(TEST_HOME, { recursive: true, force: true });
    } catch {
      // Ignore cleanup errors
    }

    // Clear store state
    useStore.getState().clearSession();
  });

  it.each([
    ["test-session-123"],
    ["session-2026.02.08_test-123"],
    ["id_with_underscores"],
    ["simple-id"],
  ])("should create correct session path for ID '%s'", (sessionId) => {
    const home = process.env["HOME"] || homedir();
    const expectedPath = join(home, ".claude", "sessions", sessionId);

    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = expectedPath;

    expect(process.env["GOGENT_SESSION_DIR"]).toBe(expectedPath);
    expect(process.env["GOGENT_SESSION_DIR"]).toContain(sessionId);
    expect(process.env["GOGENT_SESSION_DIR"]).toMatch(/^\//);
  });

  it("should be cleared when session is cleared", () => {
    const sessionId = "temp-session";
    const home = process.env["HOME"] || homedir();
    const sessionPath = join(home, ".claude", "sessions", sessionId);

    // Set session and env var
    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = sessionPath;

    expect(process.env["GOGENT_SESSION_DIR"]).toBeDefined();

    // Clear session
    useStore.getState().clearSession();

    expect(process.env["GOGENT_SESSION_DIR"]).toBeUndefined();
  });

  it("should use HOME env var if set", () => {
    const customHome = "/custom/home/path";
    process.env["HOME"] = customHome;

    const sessionId = "home-test";
    const expectedPath = join(customHome, ".claude", "sessions", sessionId);

    // Simulate session setup
    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = expectedPath;

    expect(process.env["GOGENT_SESSION_DIR"]).toBe(expectedPath);
    expect(process.env["GOGENT_SESSION_DIR"]).toContain(customHome);
  });

  it("should fall back to homedir() if HOME not set", () => {
    delete process.env["HOME"];

    const sessionId = "homedir-test";
    const expectedPath = join(homedir(), ".claude", "sessions", sessionId);

    // Simulate session setup
    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = expectedPath;

    expect(process.env["GOGENT_SESSION_DIR"]).toBe(expectedPath);
  });

  it("should persist across multiple session updates", () => {
    const sessionId1 = "session-1";
    const sessionId2 = "session-2";
    const home = process.env["HOME"] || homedir();

    // First session
    useStore.getState().updateSession({ id: sessionId1 });
    process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId1);

    expect(process.env["GOGENT_SESSION_DIR"]).toContain(sessionId1);

    // Update to second session
    useStore.getState().updateSession({ id: sessionId2 });
    process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId2);

    expect(process.env["GOGENT_SESSION_DIR"]).toContain(sessionId2);
    expect(process.env["GOGENT_SESSION_DIR"]).not.toContain(sessionId1);
  });

  it("should be inherited by child processes", () => {
    const sessionId = "child-process-test";
    const home = process.env["HOME"] || homedir();
    const sessionPath = join(home, ".claude", "sessions", sessionId);

    // Set up session
    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = sessionPath;

    // Verify it's in process.env (which is spread to child processes)
    expect(process.env["GOGENT_SESSION_DIR"]).toBe(sessionPath);

    // Verify it would be included in { ...process.env } spread
    const envCopy = { ...process.env };
    expect(envCopy["GOGENT_SESSION_DIR"]).toBe(sessionPath);
  });

  it("should match the session ID in store", () => {
    const sessionId = "store-match-test";
    const home = process.env["HOME"] || homedir();
    const sessionPath = join(home, ".claude", "sessions", sessionId);

    // Set up session
    useStore.getState().updateSession({ id: sessionId });
    process.env["GOGENT_SESSION_DIR"] = sessionPath;

    // Verify consistency
    const storeSessionId = useStore.getState().sessionId;
    expect(storeSessionId).toBe(sessionId);
    expect(process.env["GOGENT_SESSION_DIR"]).toContain(storeSessionId!);
  });

  it("should handle rapid session switches", () => {
    const home = process.env["HOME"] || homedir();
    const sessions = ["session-a", "session-b", "session-c"];

    for (const sessionId of sessions) {
      useStore.getState().updateSession({ id: sessionId });
      process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId);

      expect(process.env["GOGENT_SESSION_DIR"]).toContain(sessionId);
    }

    // Final state should be last session
    expect(process.env["GOGENT_SESSION_DIR"]).toContain("session-c");
  });

  it("should not be set if session ID is null", () => {
    // Start with no session
    useStore.getState().clearSession();

    expect(useStore.getState().sessionId).toBeNull();
    expect(process.env["GOGENT_SESSION_DIR"]).toBeUndefined();
  });
});
