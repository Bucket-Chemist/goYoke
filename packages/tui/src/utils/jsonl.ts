/**
 * Offset-tracking JSONL reader for telemetry files
 * Handles file rotation and incremental reading
 */

import { open, stat } from "fs/promises";

export interface JsonlReader {
  path: string;
  offset: number;
  inode: number;
}

/**
 * Create a new JSONL reader for the given file path
 */
export function createReader(path: string): JsonlReader {
  return { path, offset: 0, inode: 0 };
}

/**
 * Read new lines from JSONL file since last read
 * Handles file rotation by detecting inode changes
 *
 * @returns Array of parsed JSON objects and new file offset
 */
export async function readNewLines(reader: JsonlReader): Promise<{
  lines: unknown[];
  newOffset: number;
}> {
  // Check if file exists
  const stats = await stat(reader.path).catch(() => null);
  if (!stats) {
    return { lines: [], newOffset: reader.offset };
  }

  // File was rotated/replaced - reset offset
  if (stats.ino !== reader.inode) {
    reader.offset = 0;
    reader.inode = stats.ino;
  }

  // No new content
  if (stats.size <= reader.offset) {
    return { lines: [], newOffset: reader.offset };
  }

  // Read only new bytes
  const handle = await open(reader.path, "r");
  try {
    const buffer = Buffer.alloc(stats.size - reader.offset);
    await handle.read(buffer, 0, buffer.length, reader.offset);

    const content = buffer.toString("utf-8");
    const lines = content
      .trim()
      .split("\n")
      .filter(Boolean)
      .map(line => JSON.parse(line));

    return { lines, newOffset: stats.size };
  } finally {
    await handle.close();
  }
}
