/**
 * Internal debug logger for TUI errors and events.
 * See: Gemini audit GAP-3 (debug view)
 *
 * Behavior:
 * - When DEBUG=true: writes to ~/.cache/gofortress-tui/debug.log
 * - Always: captures last N errors in memory for error boundary display
 * - Does NOT display in TUI (would clutter conversation)
 */

import { appendFile, mkdir } from "fs/promises";
import { join } from "path";

const DEBUG = process.env["DEBUG"] === "true";
const LOG_DIR = join(process.env["HOME"]!, ".cache", "gofortress-tui");
const LOG_FILE = join(LOG_DIR, "debug.log");
const MAX_MEMORY_LOGS = 50;

export interface LogEntry {
  timestamp: string;
  level: "debug" | "info" | "warn" | "error";
  message: string;
  context?: Record<string, unknown>;
}

// In-memory ring buffer for error boundary display
const memoryLogs: LogEntry[] = [];

/**
 * Log a message with the specified level
 * Always stores in memory, writes to file only if DEBUG=true
 */
export async function log(
  level: LogEntry["level"],
  message: string,
  context?: Record<string, unknown>
): Promise<void> {
  const entry: LogEntry = {
    timestamp: new Date().toISOString(),
    level,
    message,
    context,
  };

  // Always keep in memory (for error boundary)
  memoryLogs.push(entry);
  if (memoryLogs.length > MAX_MEMORY_LOGS) {
    memoryLogs.shift();
  }

  // Write to file only if DEBUG=true
  if (DEBUG) {
    await mkdir(LOG_DIR, { recursive: true });
    await appendFile(LOG_FILE, JSON.stringify(entry) + "\n");
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
 * Convenience logger object with level methods
 */
export const logger = {
  debug: (msg: string, ctx?: Record<string, unknown>) => log("debug", msg, ctx),
  info: (msg: string, ctx?: Record<string, unknown>) => log("info", msg, ctx),
  warn: (msg: string, ctx?: Record<string, unknown>) => log("warn", msg, ctx),
  error: (msg: string, ctx?: Record<string, unknown>) => log("error", msg, ctx),
};
