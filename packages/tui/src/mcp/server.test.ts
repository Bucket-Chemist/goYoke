import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("MCP Server Registration", () => {
  // Store original env
  let originalEnv: string | undefined;

  beforeEach(() => {
    originalEnv = process.env['GOGENT_MCP_SPAWN_ENABLED'];
    // Clear module cache to ensure fresh imports
    vi.resetModules();
  });

  afterEach(() => {
    // Restore original env
    if (originalEnv !== undefined) {
      process.env['GOGENT_MCP_SPAWN_ENABLED'] = originalEnv;
    } else {
      delete process.env['GOGENT_MCP_SPAWN_ENABLED'];
    }
  });

  describe("isSpawnEnabled", () => {
    it("should return true when GOGENT_MCP_SPAWN_ENABLED is not set", async () => {
      delete process.env['GOGENT_MCP_SPAWN_ENABLED'];
      const { isSpawnEnabled } = await import("./server.js");
      expect(isSpawnEnabled()).toBe(true);
    });

    it("should return true when GOGENT_MCP_SPAWN_ENABLED is 'true'", async () => {
      process.env['GOGENT_MCP_SPAWN_ENABLED'] = "true";
      const { isSpawnEnabled } = await import("./server.js");
      expect(isSpawnEnabled()).toBe(true);
    });

    it("should return false when GOGENT_MCP_SPAWN_ENABLED is 'false'", async () => {
      process.env['GOGENT_MCP_SPAWN_ENABLED'] = "false";
      const { isSpawnEnabled } = await import("./server.js");
      expect(isSpawnEnabled()).toBe(false);
    });
  });

  describe("createMcpServer", () => {
    it("should include spawn_agent when spawn is enabled", async () => {
      delete process.env['GOGENT_MCP_SPAWN_ENABLED'];
      const { getServerTools } = await import("./server.js");
      const tools = getServerTools();
      const toolNames = tools.map((t) => t.name);
      expect(toolNames).toContain("spawn_agent");
    });

    it("should exclude spawn_agent when spawn is disabled", async () => {
      process.env['GOGENT_MCP_SPAWN_ENABLED'] = "false";
      const { getServerTools } = await import("./server.js");
      const tools = getServerTools();
      const toolNames = tools.map((t) => t.name);
      expect(toolNames).not.toContain("spawn_agent");
    });

    it("should always include test_mcp_ping tool", async () => {
      const { getServerTools } = await import("./server.js");
      const tools = getServerTools();
      const toolNames = tools.map((t) => t.name);
      expect(toolNames).toContain("test_mcp_ping");
    });
  });
});
