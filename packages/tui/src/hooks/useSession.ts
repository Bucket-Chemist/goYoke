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

import { promises as fs, createReadStream } from "fs";
import { createInterface } from "readline";
import { join } from "path";
import { homedir } from "os";
import { nanoid } from "nanoid";
import type { SessionData, Message, ContentBlock } from "../store/types.js";

/**
 * Get session directory path
 * Allows override via environment variable for testing
 */
function getConfigDir(): string {
  return process.env["CLAUDE_CONFIG_DIR"] || join(process.env["HOME"] || homedir(), ".claude");
}

function getSessionDir(): string {
  return join(getConfigDir(), "sessions");
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
 * List all sessions, merging two sources:
 *   1. ~/.claude/sessions/*.json (TUI-created, has accurate cost/tool_calls)
 *   2. ~/.claude/projects/{project}/*.jsonl (CLI conversation logs)
 * Deduplicates by ID — .json data wins when both exist.
 * Sorted by last_used descending (most recent first)
 * Skips invalid files gracefully
 */
export async function listSessions(): Promise<SessionData[]> {
  await ensureSessionDir();
  const sessionDir = getSessionDir();
  const sessionsById = new Map<string, SessionData>();

  // Source 1: TUI .json metadata files (authoritative when present)
  try {
    const files = await fs.readdir(sessionDir);

    for (const file of files) {
      if (!file.endsWith(".json")) continue;

      try {
        const filePath = join(sessionDir, file);
        const content = await fs.readFile(filePath, "utf-8");
        const session = JSON.parse(content) as SessionData;

        if (
          session.id &&
          session.created_at &&
          session.last_used &&
          typeof session.cost === "number" &&
          typeof session.tool_calls === "number"
        ) {
          sessionsById.set(session.id, session);
        }
      } catch (error) {
        if (process.env["VERBOSE"]) {
          console.error(`Skipping invalid session file: ${file}`, error);
        }
      }
    }
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") {
      throw error;
    }
  }

  // Source 2: CLI conversation logs across all project directories
  const projectsDir = join(getConfigDir(), "projects");
  try {
    const projectDirs = await fs.readdir(projectsDir);

    for (const projectDir of projectDirs) {
      const dirPath = join(projectsDir, projectDir);
      let entries: string[];
      try {
        entries = await fs.readdir(dirPath);
      } catch {
        continue;
      }

      for (const entry of entries) {
        if (!entry.endsWith(".jsonl")) continue;
        const id = entry.slice(0, -6); // strip .jsonl
        if (sessionsById.has(id)) continue; // .json data wins

        try {
          const jsonlPath = join(dirPath, entry);
          const stat = await fs.stat(jsonlPath);
          if (!stat.isFile()) continue;

          sessionsById.set(id, {
            id,
            created_at: new Date(stat.birthtime ?? stat.mtime).toISOString(),
            last_used: new Date(stat.mtime).toISOString(),
            cost: 0,
            tool_calls: 0,
          });
        } catch {
          // Skip unreadable files
        }
      }
    }
  } catch {
    // ~/.claude/projects doesn't exist — skip
  }

  // Sort by last_used descending (most recent first)
  const sessions = Array.from(sessionsById.values());
  sessions.sort((a, b) => {
    return new Date(b.last_used).getTime() - new Date(a.last_used).getTime();
  });

  return sessions;
}

/**
 * Extract the original CWD from a JSONL conversation log.
 *
 * Claude CLI writes session metadata (including `cwd`) in the very first JSONL
 * entry (the system init message). In practice the field appears within the
 * first 1–3 lines, but we scan up to 20 lines to tolerate any preamble lines
 * that may be added by future CLI versions.
 *
 * Uses a readline stream over a small read window instead of loading the whole
 * file — JSONL conversation logs can exceed 100 MB for long sessions, so
 * reading the full file just to find the first few lines is wasteful.
 *
 * Returns the CWD string, or null if not found or on any I/O error.
 */
async function extractCwdFromJsonl(jsonlPath: string): Promise<string | null> {
  const MAX_LINES = 20;

  return new Promise<string | null>((resolve) => {
    let stream: ReturnType<typeof createReadStream> | undefined;
    let rl: ReturnType<typeof createInterface> | undefined;
    let settled = false;

    const finish = (result: string | null): void => {
      if (settled) return;
      settled = true;
      try {
        rl?.close();
        stream?.destroy();
      } catch {
        // ignore cleanup errors
      }
      resolve(result);
    };

    try {
      // Read only the first 4 KB — more than enough for the system init entry.
      stream = createReadStream(jsonlPath, { encoding: "utf-8", end: 4095 });
      rl = createInterface({ input: stream, crlfDelay: Infinity });
    } catch {
      resolve(null);
      return;
    }

    stream.on("error", () => finish(null));

    let lineCount = 0;
    rl.on("line", (line: string) => {
      if (settled) return;

      lineCount++;
      const trimmed = line.trim();

      if (trimmed) {
        try {
          const entry = JSON.parse(trimmed) as Record<string, unknown>;
          if (typeof entry["cwd"] === "string") {
            finish(entry["cwd"]);
            return;
          }
        } catch {
          // not valid JSON — skip line
        }
      }

      if (lineCount >= MAX_LINES) {
        finish(null);
      }
    });

    rl.on("close", () => finish(null));
  });
}

/**
 * Search for a session across all Claude CLI project directories.
 * Returns a minimal SessionData constructed from JSONL file metadata,
 * or null if not found in any project directory.
 *
 * This covers CLI-created sessions that have no .json metadata file.
 * The TUI only needs the `id` field for query({resume: id}).
 * Cost and tool_calls are zeroed since accurate data is unavailable
 * without reading the full JSONL — these are display-only fields.
 *
 * Also extracts the original CWD from the JSONL so that session resume
 * can pass the correct working directory to query().
 *
 * NOTE: Does NOT persist the result (no saveSession call).
 * Persisting would create ghost .json files with zeroed metrics and
 * corrupted last_used timestamps (Staff Architect review C2).
 */
async function findSessionInConversationLogs(
  id: string,
): Promise<SessionData | null> {
  const projectsDir = join(getConfigDir(), "projects");

  let projectDirs: string[];
  try {
    projectDirs = await fs.readdir(projectsDir);
  } catch {
    return null; // ~/.claude/projects doesn't exist
  }

  for (const projectDir of projectDirs) {
    const jsonlPath = join(projectsDir, projectDir, `${id}.jsonl`);
    try {
      const stat = await fs.stat(jsonlPath);
      if (!stat.isFile()) continue;

      // File exists — construct minimal SessionData from file metadata
      const lastUsed = new Date(stat.mtime).toISOString();
      const createdAt = new Date(stat.birthtime ?? stat.mtime).toISOString();

      // Extract CWD from the JSONL for cross-project resume
      const projectDir2 = await extractCwdFromJsonl(jsonlPath);

      return {
        id,
        created_at: createdAt,
        last_used: lastUsed,
        cost: 0,
        tool_calls: 0,
        projectDir: projectDir2 ?? undefined,
      };
    } catch {
      // File doesn't exist in this project dir — try next
    }
  }

  return null;
}

/**
 * Load specific session by ID
 * Primary: reads from ~/.claude/sessions/{id}.json (TUI-created sessions)
 * Fallback: searches conversation logs in ~/.claude/projects/{project}/{id}.jsonl
 * Throws if session not found in either source
 */
export async function loadSession(id: string): Promise<SessionData> {
  const sessionDir = getSessionDir();
  const filePath = join(sessionDir, `${id}.json`);

  // Primary path: load from .json metadata file (TUI-created sessions)
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
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") {
      throw error; // Unexpected error (parse error, etc.) — propagate
    }
    // .json file not found — fall through to JSONL search
  }

  // Fallback: search conversation logs across all project directories
  const sessionFromLog = await findSessionInConversationLogs(id);
  if (sessionFromLog) {
    if (process.env["VERBOSE"]) {
      console.error(`Session ${id} found in conversation logs (no .json metadata)`);
    }
    return sessionFromLog;
  }

  throw new Error(`Session not found: ${id}`);
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
 * Parse a JSONL conversation log and return TUI Message objects.
 * Searches across all ~/.claude/projects/{project}/{sessionId}.jsonl directories.
 * Deduplicates assistant messages by message ID (last occurrence wins).
 * Single-pass for performance on large files.
 */
export async function loadConversationHistory(sessionId: string): Promise<Message[]> {
  const projectsDir = join(getConfigDir(), "projects");

  let projectDirs: string[];
  try {
    projectDirs = await fs.readdir(projectsDir);
  } catch {
    return [];
  }

  let jsonlPath: string | null = null;
  for (const projectDir of projectDirs) {
    const candidate = join(projectsDir, projectDir, `${sessionId}.jsonl`);
    try {
      const stat = await fs.stat(candidate);
      if (stat.isFile()) {
        jsonlPath = candidate;
        break;
      }
    } catch {
      // not in this project dir
    }
  }

  if (!jsonlPath) {
    return [];
  }

  let raw: string;
  try {
    raw = await fs.readFile(jsonlPath, "utf-8");
  } catch {
    return [];
  }

  const lines = raw.split("\n");

  // Track user messages in order, and assistant messages by ID (last wins)
  const userMessages: Message[] = [];
  const assistantById = new Map<string, { msg: Message; index: number }>();
  let orderIndex = 0;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    let entry: unknown;
    try {
      entry = JSON.parse(trimmed) as unknown;
    } catch {
      continue;
    }

    if (
      typeof entry !== "object" ||
      entry === null ||
      !("type" in entry) ||
      !("message" in entry)
    ) {
      continue;
    }

    const record = entry as { type: string; message: unknown };

    if (record.type === "user") {
      const msg = record.message as { role?: string; content?: unknown };
      if (msg.role !== "user") continue;

      // Skip tool_result lines
      if (Array.isArray(msg.content)) {
        const hasToolResult = (msg.content as Array<{ type?: string }>).some(
          (b) => b.type === "tool_result"
        );
        if (hasToolResult) continue;
      }

      let contentBlocks: ContentBlock[];
      if (typeof msg.content === "string") {
        if (!msg.content) continue;
        contentBlocks = [{ type: "text", text: msg.content }];
      } else if (Array.isArray(msg.content)) {
        contentBlocks = (msg.content as Array<{ type?: string; text?: string }>)
          .filter((b) => b.type === "text" && typeof b.text === "string" && b.text.length > 0)
          .map((b) => ({ type: "text" as const, text: b.text as string }));
      } else {
        continue;
      }

      if (contentBlocks.length === 0) continue;

      userMessages.push({
        id: nanoid(),
        role: "user",
        content: contentBlocks,
        partial: false,
        timestamp: orderIndex++,
      });
    } else if (record.type === "assistant") {
      const msg = record.message as { role?: string; id?: string; content?: unknown };
      if (msg.role !== "assistant" || typeof msg.id !== "string") continue;

      const rawBlocks = Array.isArray(msg.content)
        ? (msg.content as Array<{ type?: string; text?: string }>)
        : [];

      const contentBlocks: ContentBlock[] = rawBlocks
        .filter((b) => b.type === "text" && typeof b.text === "string" && b.text.length > 0)
        .map((b) => ({ type: "text" as const, text: b.text as string }));

      if (contentBlocks.length === 0) continue;

      const existing = assistantById.get(msg.id);
      assistantById.set(msg.id, {
        msg: {
          id: existing?.msg.id ?? nanoid(),
          role: "assistant",
          content: contentBlocks,
          partial: false,
          timestamp: existing?.msg.timestamp ?? orderIndex++,
        },
        index: existing?.index ?? orderIndex - 1,
      });
    }
  }

  // Merge user messages and deduplicated assistant messages, sorted by timestamp
  const assistantMessages = Array.from(assistantById.values()).map((e) => e.msg);
  const all = [...userMessages, ...assistantMessages];
  all.sort((a, b) => a.timestamp - b.timestamp);

  // Replace synthetic ordering timestamps with real wall-clock time
  const now = Date.now();
  for (let i = 0; i < all.length; i++) {
    all[i]!.timestamp = now - (all.length - i) * 1000;
  }

  return all;
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
