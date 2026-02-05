/**
 * Internal debug logger for TUI errors and events.
 * See: Gemini audit GAP-3 (debug view)
 *
 * Behavior:
 * - Always: writes to ~/.cache/gofortress-tui/debug.log with session tracking
 * - Always: captures last N errors in memory for error boundary display
 * - Always: outputs to console for TUI visibility
 */

import { appendFile, mkdir } from "fs/promises";
import { join } from "path";
import { randomUUID } from "crypto";

const LOG_DIR = join(process.env["HOME"]!, ".cache", "gofortress-tui");
const LOG_FILE = join(LOG_DIR, "debug.log");
const MAX_MEMORY_LOGS = 50;

// Session ID for this TUI instance
const SESSION_ID = randomUUID();

export interface LogEntry {
  timestamp: string;
  sessionId: string;
  level: "debug" | "info" | "warn" | "error";
  message: string;
  context?: Record<string, unknown>;
}

// In-memory ring buffer for error boundary display
const memoryLogs: LogEntry[] = [];

/**
 * Log a message with the specified level
 * Always stores in memory, writes to file, and outputs to console
 */
export async function log(
  level: LogEntry["level"],
  message: string,
  context?: Record<string, unknown>,
  sessionId?: string
): Promise<void> {
  const entry: LogEntry = {
    timestamp: new Date().toISOString(),
    sessionId: sessionId ?? SESSION_ID,
    level,
    message,
    context,
  };

  // Always keep in memory (for error boundary)
  memoryLogs.push(entry);
  if (memoryLogs.length > MAX_MEMORY_LOGS) {
    memoryLogs.shift();
  }

  // Always write to console (for TUI visibility)
  console.log(`[${level.toUpperCase()}] ${message}`, context ? JSON.stringify(context, null, 2) : "");

  // Always write to file (no DEBUG check)
  try {
    await mkdir(LOG_DIR, { recursive: true });
    await appendFile(LOG_FILE, JSON.stringify(entry) + "\n");
  } catch (error) {
    // Don't crash if logging fails, but log to stderr
    console.error("Failed to write to debug.log:", error);
  }
}

/**
 * Get all recent logs from memory buffer
 */
export function getRecentLogs(): LogEntry[] {
  return [...memoryLogs];
}

/**
 * Get only error-level logs from memory buffer
 */
export function getRecentErrors(): LogEntry[] {
  return memoryLogs.filter(e => e.level === "error");
}

/**
 * Clear all logs from memory buffer (for testing)
 */
export function clearLogs(): void {
  memoryLogs.length = 0;
}

/**
 * Get the current session ID
 */
export function getSessionId(): string {
  return SESSION_ID;
}

/**
 * Convenience logger object with level methods
 */
export const logger = {
  debug: (msg: string, ctx?: Record<string, unknown>, sessionId?: string) =>
    log("debug", msg, ctx, sessionId),
  info: (msg: string, ctx?: Record<string, unknown>, sessionId?: string) =>
    log("info", msg, ctx, sessionId),
  warn: (msg: string, ctx?: Record<string, unknown>, sessionId?: string) =>
    log("warn", msg, ctx, sessionId),
  error: (msg: string, ctx?: Record<string, unknown>, sessionId?: string) =>
    log("error", msg, ctx, sessionId),
};
