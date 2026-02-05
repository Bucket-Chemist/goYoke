import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as child_process from "child_process";
import * as fs from "fs/promises";
import {
  validateSpawnEnvironment,
  formatValidationResult,
  assertValidSpawnEnvironment,
} from "./validation";

// Mock child_process.execSync
vi.mock("child_process", () => ({
  execSync: vi.fn(),
}));

// Mock fs/promises
vi.mock("fs/promises", () => ({
  writeFile: vi.fn(),
  unlink: vi.fn(),
}));

describe("validateSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Default: all checks pass
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should pass when all checks succeed", async () => {
    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("should fail when claude CLI not found", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_CLAUDE_NOT_FOUND" })
    );
  });

  it("should fail when /tmp not writable", async () => {
    vi.mocked(fs.writeFile).mockRejectedValue(new Error("EACCES"));

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_TMP_NOT_WRITABLE" })
    );
  });

  it("should warn when GOGENT_MCP_SPAWN_ENABLED=false", async () => {
    const originalEnv = process.env.GOGENT_MCP_SPAWN_ENABLED;
    process.env.GOGENT_MCP_SPAWN_ENABLED = "false";

    try {
      const result = await validateSpawnEnvironment();

      expect(result.ok).toBe(true); // Warnings don't fail
      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_SPAWN_DISABLED" })
      );
    } finally {
      process.env.GOGENT_MCP_SPAWN_ENABLED = originalEnv;
    }
  });
});

describe("formatValidationResult", () => {
  it("should format success result", () => {
    const result = { ok: true, errors: [], warnings: [] };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("✅ Environment validation passed");
  });

  it("should format errors with resolution", () => {
    const result = {
      ok: false,
      errors: [
        {
          code: "E_TEST",
          message: "Test error",
          resolution: "Fix the test",
        },
      ],
      warnings: [],
    };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("❌ Environment validation failed");
    expect(formatted).toContain("[E_TEST]");
    expect(formatted).toContain("Test error");
    expect(formatted).toContain("Fix the test");
  });
});

describe("assertValidSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should not throw when validation passes", async () => {
    await expect(assertValidSpawnEnvironment()).resolves.not.toThrow();
  });

  it("should throw when validation fails", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    await expect(assertValidSpawnEnvironment()).rejects.toThrow(
      /Spawn environment validation failed/
    );
  });
});
