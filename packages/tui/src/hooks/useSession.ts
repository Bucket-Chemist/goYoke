/**
 * Session file operations hook
 * Manages session persistence with Go-compatible format
 *
 * SESSION FILE FORMAT (CRITICAL - Must Match Go):
 * {
 *   "id": "uuid-here",
 *   "name": "optional-name",
 *   "created_at": "2026-02-01T10:00:00Z",
 *   "last_used": "2026-02-01T11:30:00Z",
 *   "cost": 0.42,
 *   "tool_calls": 127
 * }
 */

import { promises as fs } from "fs";
import { join } from "path";
import { homedir } from "os";
import type { SessionData } from "../store/types.js";

/**
 * Get session directory path
 * Allows override via environment variable for testing
 */
function getSessionDir(): string {
  const home = process.env["HOME"] || homedir();
  return join(home, ".claude", "sessions");
}

/**
 * Ensure session directory exists
 */
async function ensureSessionDir(): Promise<void> {
  const sessionDir = getSessionDir();
  try {
    await fs.mkdir(sessionDir, { recursive: true });
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "EEXIST") {
      throw error;
    }
  }
}

/**
 * List all sessions from ~/.claude/sessions/
 * Sorted by last_used descending (most recent first)
 * Skips invalid JSON files gracefully
 */
export async function listSessions(): Promise<SessionData[]> {
  await ensureSessionDir();
  const sessionDir = getSessionDir();

  try {
    const files = await fs.readdir(sessionDir);
    const sessions: SessionData[] = [];

    for (const file of files) {
      if (!file.endsWith(".json")) continue;

      try {
        const filePath = join(sessionDir, file);
        const content = await fs.readFile(filePath, "utf-8");
        const session = JSON.parse(content) as SessionData;

        // Validate required fields
        if (
          session.id &&
          session.created_at &&
          session.last_used &&
          typeof session.cost === "number" &&
          typeof session.tool_calls === "number"
        ) {
          sessions.push(session);
        }
      } catch (error) {
        // Skip invalid files - don't crash on corrupted JSON
        if (process.env["VERBOSE"]) {
          console.error(`Skipping invalid session file: ${file}`, error);
        }
      }
    }

    // Sort by last_used descending (most recent first)
    sessions.sort((a, b) => {
      return new Date(b.last_used).getTime() - new Date(a.last_used).getTime();
    });

    return sessions;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return []; // Directory doesn't exist yet
    }
    throw error;
  }
}

/**
 * Load specific session by ID
 * Throws if session doesn't exist or is invalid
 */
export async function loadSession(id: string): Promise<SessionData> {
  const sessionDir = getSessionDir();
  const filePath = join(sessionDir, `${id}.json`);

  try {
    const content = await fs.readFile(filePath, "utf-8");
    const session = JSON.parse(content) as SessionData;

    // Validate required fields
    if (
      !session.id ||
      !session.created_at ||
      !session.last_used ||
      typeof session.cost !== "number" ||
      typeof session.tool_calls !== "number"
    ) {
      throw new Error(`Invalid session format: ${id}`);
    }

    return session;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      throw new Error(`Session not found: ${id}`);
    }
    throw error;
  }
}

/**
 * Save session to file
 * Creates/updates session file with Go-compatible format
 * Uses snake_case field names for Go compatibility
 */
export async function saveSession(session: SessionData): Promise<void> {
  await ensureSessionDir();
  const sessionDir = getSessionDir();
  const filePath = join(sessionDir, `${session.id}.json`);

  // Ensure ISO 8601 format with Z suffix
  const data: SessionData = {
    id: session.id,
    name: session.name,
    created_at: session.created_at,
    last_used: new Date().toISOString(), // Update last_used on save
    cost: session.cost,
    tool_calls: session.tool_calls,
  };

  await fs.writeFile(filePath, JSON.stringify(data, null, 2), "utf-8");
}

/**
 * Delete session file
 */
export async function deleteSession(id: string): Promise<void> {
  const sessionDir = getSessionDir();
  const filePath = join(sessionDir, `${id}.json`);

  try {
    await fs.unlink(filePath);
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      throw new Error(`Session not found: ${id}`);
    }
    throw error;
  }
}

/**
 * React hook for session operations
 * Provides session CRUD operations with error handling
 */
export function useSession() {
  return {
    listSessions,
    loadSession,
    saveSession,
    deleteSession,
  };
}
