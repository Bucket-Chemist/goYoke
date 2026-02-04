/**
 * Tests for JSONL reader with offset tracking
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtemp, writeFile, unlink, rm } from "fs/promises";
import { join } from "path";
import { tmpdir } from "os";
import { createReader, readNewLines } from "../../src/utils/jsonl.js";

describe("jsonl reader", () => {
  let testDir: string;
  let testFile: string;

  beforeEach(async () => {
    testDir = await mkdtemp(join(tmpdir(), "jsonl-test-"));
    testFile = join(testDir, "test.jsonl");
  });

  afterEach(async () => {
    await rm(testDir, { recursive: true, force: true });
  });

  describe("createReader", () => {
    it("should create a reader with initial state", () => {
      const reader = createReader(testFile);
      expect(reader.path).toBe(testFile);
      expect(reader.offset).toBe(0);
      expect(reader.inode).toBe(0);
    });
  });

  describe("readNewLines", () => {
    it("should return empty array for non-existent file", async () => {
      const reader = createReader(testFile);
      const result = await readNewLines(reader);

      expect(result.lines).toEqual([]);
      expect(result.newOffset).toBe(0);
    });

    it("should read all lines from new file", async () => {
      const reader = createReader(testFile);
      const data = [
        { id: 1, type: "routing" },
        { id: 2, type: "handoff" },
      ];

      await writeFile(testFile, data.map(d => JSON.stringify(d)).join("\n") + "\n");

      const result = await readNewLines(reader);

      expect(result.lines).toHaveLength(2);
      expect(result.lines[0]).toEqual(data[0]);
      expect(result.lines[1]).toEqual(data[1]);
      expect(result.newOffset).toBeGreaterThan(0);
    });

    it("should track offset and only read new lines", async () => {
      const reader = createReader(testFile);

      // Initial write
      const line1 = { id: 1, message: "first" };
      await writeFile(testFile, JSON.stringify(line1) + "\n");

      const result1 = await readNewLines(reader);
      expect(result1.lines).toHaveLength(1);
      expect(result1.lines[0]).toEqual(line1);

      // Update reader offset
      reader.offset = result1.newOffset;

      // No change - should return empty
      const result2 = await readNewLines(reader);
      expect(result2.lines).toEqual([]);
      expect(result2.newOffset).toBe(result1.newOffset);

      // Append new line
      const line2 = { id: 2, message: "second" };
      await writeFile(testFile, JSON.stringify(line1) + "\n" + JSON.stringify(line2) + "\n");

      const result3 = await readNewLines(reader);
      expect(result3.lines).toHaveLength(1);
      expect(result3.lines[0]).toEqual(line2);
      expect(result3.newOffset).toBeGreaterThan(result1.newOffset);
    });

    it("should handle file rotation (inode change)", async () => {
      const reader = createReader(testFile);

      // Initial file
      const line1 = { id: 1, message: "original" };
      await writeFile(testFile, JSON.stringify(line1) + "\n");

      const result1 = await readNewLines(reader);
      expect(result1.lines).toHaveLength(1);

      // Update offset and inode
      reader.offset = result1.newOffset;

      // Simulate rotation: delete and recreate (new inode)
      await unlink(testFile);
      const line2 = { id: 2, message: "rotated" };
      await writeFile(testFile, JSON.stringify(line2) + "\n");

      const result2 = await readNewLines(reader);

      // Should read from beginning (offset reset)
      expect(result2.lines).toHaveLength(1);
      expect(result2.lines[0]).toEqual(line2);
      expect(reader.offset).toBe(0); // Offset was reset
      expect(reader.inode).not.toBe(0); // Inode updated
    });

    it("should handle empty lines in file", async () => {
      const reader = createReader(testFile);
      const data = [
        { id: 1 },
        { id: 2 },
      ];

      // Write with extra newlines
      await writeFile(testFile, `${JSON.stringify(data[0])}\n\n\n${JSON.stringify(data[1])}\n\n`);

      const result = await readNewLines(reader);

      // Should filter out empty lines
      expect(result.lines).toHaveLength(2);
      expect(result.lines[0]).toEqual(data[0]);
      expect(result.lines[1]).toEqual(data[1]);
    });

    it("should parse complex JSON objects", async () => {
      const reader = createReader(testFile);
      const complexData = {
        timestamp: "2026-02-04T10:00:00Z",
        agent: "go-pro",
        metadata: {
          nested: {
            value: 42,
            array: [1, 2, 3],
          },
        },
      };

      await writeFile(testFile, JSON.stringify(complexData) + "\n");

      const result = await readNewLines(reader);

      expect(result.lines).toHaveLength(1);
      expect(result.lines[0]).toEqual(complexData);
    });

    it("should handle file size unchanged scenario", async () => {
      const reader = createReader(testFile);

      // Write initial data
      const line1 = { id: 1 };
      await writeFile(testFile, JSON.stringify(line1) + "\n");

      const result1 = await readNewLines(reader);
      reader.offset = result1.newOffset;

      // File size hasn't changed
      const result2 = await readNewLines(reader);
      expect(result2.lines).toEqual([]);
      expect(result2.newOffset).toBe(result1.newOffset);
    });

    it("should update reader state correctly after rotation", async () => {
      const reader = createReader(testFile);

      // First file
      await writeFile(testFile, JSON.stringify({ id: 1 }) + "\n");
      const result1 = await readNewLines(reader);
      const firstInode = reader.inode;

      reader.offset = result1.newOffset;

      // Rotate file
      await unlink(testFile);
      await writeFile(testFile, JSON.stringify({ id: 2 }) + "\n" + JSON.stringify({ id: 3 }) + "\n");

      const result2 = await readNewLines(reader);

      // Verify state after rotation
      expect(reader.offset).toBe(0); // Reset
      expect(reader.inode).not.toBe(firstInode); // Changed
      expect(reader.inode).not.toBe(0); // Set to new inode
      expect(result2.lines).toHaveLength(2);
    });
  });
});
