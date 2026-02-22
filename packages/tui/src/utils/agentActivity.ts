/**
 * Pure utility functions for extracting AgentActivity from SDK events and ndjson streams.
 * No store dependencies - safe to import anywhere.
 */

import type { AgentActivity } from "../store/types.js";

const MAX_TARGET_LENGTH = 60;
const MAX_ERROR_LENGTH = 100;

/**
 * Priority-ordered list of input keys to use as the tool target display string.
 * The first key that resolves to a non-empty string wins.
 */
const TARGET_KEY_PRIORITY = [
  "file_path",
  "path",
  "command",
  "pattern",
  "query",
  "url",
  "description",
] as const;

/**
 * Truncate a string to `max` characters, appending "..." if truncated.
 */
function truncate(value: string, max: number): string {
  if (value.length <= max) {
    return value;
  }
  return value.slice(0, max) + "...";
}

/**
 * Extract a human-readable target string from a tool input record.
 *
 * Priority lookup: file_path > path > command > pattern > query > url > description
 * Fallback: first non-empty string value found in the record.
 * Result is truncated to 60 characters.
 */
export function extractToolTarget(
  _toolName: string,
  input: Record<string, unknown>,
): string {
  // Priority-ordered lookup
  for (const key of TARGET_KEY_PRIORITY) {
    const value = input[key];
    if (typeof value === "string" && value.trim().length > 0) {
      return truncate(value.trim(), MAX_TARGET_LENGTH);
    }
  }

  // Fallback: first non-empty string value in the record
  for (const value of Object.values(input)) {
    if (typeof value === "string" && value.trim().length > 0) {
      return truncate(value.trim(), MAX_TARGET_LENGTH);
    }
  }

  return "";
}

/**
 * Build AgentActivity from a Task() SDK block (assistant tool_use invocation).
 *
 * For SDK Task() blocks we cannot observe intermediate tools inside the spawned
 * agent, so currentTool is always null.  The activity reflects the outer task
 * lifecycle only.
 *
 * @param taskInput  - The raw input record from the tool_use block (e.g. { description, prompt })
 * @param taskResult - The tool_result content string, or null if still running
 * @param isError    - Whether the tool_result carries an error
 */
export function activityFromTaskBlocks(
  taskInput: Record<string, unknown>,
  taskResult: string | null,
  isError: boolean,
): AgentActivity {
  const lastText =
    typeof taskInput["description"] === "string"
      ? taskInput["description"]
      : null;

  let toolResult: AgentActivity["toolResult"];
  if (taskResult === null) {
    toolResult = { status: "pending" };
  } else if (isError) {
    toolResult = {
      status: "failed",
      error: taskResult.slice(0, MAX_ERROR_LENGTH),
    };
  } else {
    toolResult = { status: "success" };
  }

  return {
    lastText,
    currentTool: null,
    toolResult,
  };
}

// ---------------------------------------------------------------------------
// Internal types for ndjson parsing
// ---------------------------------------------------------------------------

interface NdjsonTextBlock {
  type: "text";
  text: string;
}

interface NdjsonToolUseBlock {
  type: "tool_use";
  id: string;
  name: string;
  input: Record<string, unknown>;
}

interface NdjsonToolResultBlock {
  type: "tool_result";
  tool_use_id: string;
  content: string;
  is_error?: boolean;
}

type NdjsonContentBlock =
  | NdjsonTextBlock
  | NdjsonToolUseBlock
  | NdjsonToolResultBlock;

interface NdjsonAssistantMessage {
  type: "assistant";
  message: {
    content: NdjsonContentBlock[];
  };
}

interface NdjsonUserMessage {
  type: "user";
  message: {
    content: NdjsonContentBlock[];
  };
}

type NdjsonLine = NdjsonAssistantMessage | NdjsonUserMessage;

// ---------------------------------------------------------------------------
// Type guards for defensive parsing
// ---------------------------------------------------------------------------

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isArray(value: unknown): value is unknown[] {
  return Array.isArray(value);
}

function asNdjsonLine(value: unknown): NdjsonLine | null {
  if (!isRecord(value)) return null;

  const type = value["type"];
  if (type !== "assistant" && type !== "user") return null;

  const message = value["message"];
  if (!isRecord(message)) return null;

  const content = message["content"];
  if (!isArray(content)) return null;

  return value as unknown as NdjsonLine;
}

function asContentBlocks(content: unknown[]): NdjsonContentBlock[] {
  const result: NdjsonContentBlock[] = [];

  for (const item of content) {
    if (!isRecord(item)) continue;

    const type = item["type"];

    if (type === "text" && typeof item["text"] === "string") {
      result.push({ type: "text", text: item["text"] });
      continue;
    }

    if (
      type === "tool_use" &&
      typeof item["id"] === "string" &&
      typeof item["name"] === "string" &&
      isRecord(item["input"])
    ) {
      result.push({
        type: "tool_use",
        id: item["id"],
        name: item["name"],
        input: item["input"] as Record<string, unknown>,
      });
      continue;
    }

    if (
      type === "tool_result" &&
      typeof item["tool_use_id"] === "string"
    ) {
      const content =
        typeof item["content"] === "string" ? item["content"] : "";
      const is_error =
        typeof item["is_error"] === "boolean" ? item["is_error"] : false;
      result.push({
        type: "tool_result",
        tool_use_id: item["tool_use_id"],
        content,
        is_error,
      });
      continue;
    }
  }

  return result;
}

/**
 * Parse an array of ndjson text lines (one JSON object per line) and extract
 * the latest AgentActivity state.
 *
 * Lines are processed in forward order so that later events overwrite earlier
 * ones — the last tool_use seen becomes currentTool, the last tool_result seen
 * is reflected in toolResult, and whether the final tool_use appeared *after*
 * the last tool_result determines pending status.
 *
 * Returns null if no useful information could be extracted.
 */
export function activityFromNdjsonChunk(lines: string[]): AgentActivity | null {
  let lastText: string | null = null;
  let lastToolUse: NdjsonToolUseBlock | null = null;
  let lastToolResult: NdjsonToolResultBlock | null = null;

  // Track ordering by line index so we can tell whether tool_use came after
  // tool_result or vice-versa.
  let lastToolUseLineIndex = -1;
  let lastToolResultLineIndex = -1;

  for (let lineIdx = 0; lineIdx < lines.length; lineIdx++) {
    const rawLine = lines[lineIdx];
    if (!rawLine || rawLine.trim().length === 0) continue;

    let parsed: unknown;
    try {
      parsed = JSON.parse(rawLine);
    } catch {
      // Non-JSON line (e.g. startup noise) — skip gracefully
      continue;
    }

    const line = asNdjsonLine(parsed);
    if (line === null) continue;

    const blocks = asContentBlocks(line.message.content);

    if (line.type === "assistant") {
      for (const block of blocks) {
        if (block.type === "text" && block.text.trim().length > 0) {
          lastText = block.text;
        }
        if (block.type === "tool_use") {
          lastToolUse = block;
          lastToolUseLineIndex = lineIdx;
        }
      }
    } else {
      // type === "user"
      for (const block of blocks) {
        if (block.type === "tool_result") {
          lastToolResult = block;
          lastToolResultLineIndex = lineIdx;
        }
      }
    }
  }

  // Nothing useful found
  if (lastText === null && lastToolUse === null && lastToolResult === null) {
    return null;
  }

  // Build currentTool from the last observed tool_use
  let currentTool: AgentActivity["currentTool"] = null;
  if (lastToolUse !== null) {
    currentTool = {
      name: lastToolUse.name,
      target: extractToolTarget(lastToolUse.name, lastToolUse.input),
      toolUseId: lastToolUse.id,
    };
  }

  // Determine toolResult status:
  // - If the last tool_use appeared AFTER the last tool_result → still pending
  // - If there is a tool_result that corresponds to the last tool_use (or any
  //   result when there is no tool_use) → reflect its outcome
  // - If there is a tool_use but no tool_result at all → pending
  let toolResult: AgentActivity["toolResult"] = null;

  if (lastToolUse !== null && lastToolUseLineIndex > lastToolResultLineIndex) {
    // Tool was invoked after the latest result — waiting for the response
    toolResult = { status: "pending" };
  } else if (lastToolResult !== null) {
    if (lastToolResult.is_error === true) {
      toolResult = {
        status: "failed",
        error: lastToolResult.content.slice(0, MAX_ERROR_LENGTH),
      };
    } else {
      toolResult = { status: "success" };
    }
  } else if (lastToolUse !== null) {
    // tool_use exists but no tool_result seen yet
    toolResult = { status: "pending" };
  }

  return {
    lastText,
    currentTool,
    toolResult,
  };
}
