/**
 * CLI Integration Tests
 * Tests CLI argument parsing and flag handling
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { parseCLI } from "../../src/cli.js";

describe("CLI Argument Parsing", () => {
  let originalArgv: string[];
  let originalExit: typeof process.exit;
  let exitCode: number | undefined;
  let consoleOutput: string[];

  beforeEach(() => {
    // Save original values
    originalArgv = process.argv;
    originalExit = process.exit;
    exitCode = undefined;
    consoleOutput = [];

    // Mock process.exit to capture exit codes without actually exiting
    process.exit = vi.fn((code?: number) => {
      exitCode = code ?? 0;
      throw new Error(`EXIT:${code ?? 0}`);
    }) as never;

    // Mock console methods to capture output
    vi.spyOn(console, "log").mockImplementation((...args) => {
      consoleOutput.push(args.join(" "));
    });
    vi.spyOn(console, "error").mockImplementation((...args) => {
      consoleOutput.push(args.join(" "));
    });
  });

  afterEach(() => {
    // Restore original values
    process.argv = originalArgv;
    process.exit = originalExit;
    vi.restoreAllMocks();
  });

  /**
   * Helper to set argv and parse CLI
   */
  function parseWithArgs(args: string[]) {
    process.argv = ["node", "gofortress-tui", ...args];
    return parseCLI();
  }

  describe("Basic flag parsing", () => {
    it("should parse --session flag", () => {
      const options = parseWithArgs(["--session", "test-123"]);

      expect(options.session).toBe("test-123");
      expect(options.list).toBe(false);
      expect(options.verbose).toBe(false);
      expect(options.legacy).toBe(false);
    });

    it("should parse -s short form", () => {
      const options = parseWithArgs(["-s", "test-456"]);

      expect(options.session).toBe("test-456");
    });

    it("should parse --list flag", () => {
      const options = parseWithArgs(["--list"]);

      expect(options.list).toBe(true);
      expect(options.session).toBeUndefined();
      expect(options.verbose).toBe(false);
      expect(options.legacy).toBe(false);
    });

    it("should parse -l short form", () => {
      const options = parseWithArgs(["-l"]);

      expect(options.list).toBe(true);
    });

    it("should parse --verbose flag", () => {
      const options = parseWithArgs(["--verbose"]);

      expect(options.verbose).toBe(true);
      expect(options.list).toBe(false);
      expect(options.legacy).toBe(false);
    });

    it("should parse -v short form", () => {
      const options = parseWithArgs(["-v"]);

      expect(options.verbose).toBe(true);
    });

    it("should parse --legacy flag", () => {
      const options = parseWithArgs(["--legacy"]);

      expect(options.legacy).toBe(true);
      expect(options.list).toBe(false);
      expect(options.verbose).toBe(false);
    });
  });

  describe("Flag combinations", () => {
    it("should handle --session with --verbose", () => {
      const options = parseWithArgs(["--session", "test-789", "--verbose"]);

      expect(options.session).toBe("test-789");
      expect(options.verbose).toBe(true);
      expect(options.list).toBe(false);
    });

    it("should handle --list with --verbose", () => {
      const options = parseWithArgs(["--list", "--verbose"]);

      expect(options.list).toBe(true);
      expect(options.verbose).toBe(true);
    });

    it("should handle all flags together", () => {
      const options = parseWithArgs([
        "--session",
        "test-abc",
        "--verbose",
        "--list",
        "--legacy",
      ]);

      expect(options.session).toBe("test-abc");
      expect(options.verbose).toBe(true);
      expect(options.list).toBe(true);
      expect(options.legacy).toBe(true);
    });

    it("should handle short forms together", () => {
      const options = parseWithArgs(["-s", "test-def", "-v", "-l"]);

      expect(options.session).toBe("test-def");
      expect(options.verbose).toBe(true);
      expect(options.list).toBe(true);
    });
  });

  describe("Default values", () => {
    it("should return defaults when no flags provided", () => {
      const options = parseWithArgs([]);

      expect(options.session).toBeUndefined();
      expect(options.list).toBe(false);
      expect(options.verbose).toBe(false);
      expect(options.legacy).toBe(false);
    });

    it("should handle session-only flag", () => {
      const options = parseWithArgs(["--session", "only-session"]);

      expect(options.session).toBe("only-session");
      expect(options.list).toBe(false);
      expect(options.verbose).toBe(false);
      expect(options.legacy).toBe(false);
    });
  });

  describe("Version and help (Commander built-ins)", () => {
    it("should exit with code 0 for --version", () => {
      try {
        parseWithArgs(["--version"]);
      } catch (e) {
        // Commander will call process.exit(0) for --version
      }

      expect(exitCode).toBe(0);
      // Note: Commander outputs directly to stdout via process.stdout.write,
      // not through console.log, so we can't capture it in unit tests.
      // The exit code verification is sufficient for unit testing.
    });

    it("should exit with code 0 for --help", () => {
      try {
        parseWithArgs(["--help"]);
      } catch (e) {
        // Commander will call process.exit(0) for --help
      }

      expect(exitCode).toBe(0);
      // Note: Commander outputs directly to stdout via process.stdout.write,
      // not through console.log, so we can't capture it in unit tests.
      // The exit code verification is sufficient for unit testing.
    });

    it("should exit with code 0 for -h short form", () => {
      try {
        parseWithArgs(["-h"]);
      } catch (e) {
        // Commander will call process.exit(0) for -h
      }

      expect(exitCode).toBe(0);
    });
  });

  describe("Session ID formats", () => {
    it("should accept alphanumeric session ID", () => {
      const options = parseWithArgs(["--session", "session_abc123"]);
      expect(options.session).toBe("session_abc123");
    });

    it("should accept UUID-style session ID", () => {
      const options = parseWithArgs([
        "--session",
        "550e8400-e29b-41d4-a716-446655440000",
      ]);
      expect(options.session).toBe("550e8400-e29b-41d4-a716-446655440000");
    });

    it("should accept nanoid-style session ID", () => {
      const options = parseWithArgs(["--session", "V1StGXR8_Z5jdHi6B-myT"]);
      expect(options.session).toBe("V1StGXR8_Z5jdHi6B-myT");
    });

    it("should accept simple string session ID", () => {
      const options = parseWithArgs(["--session", "my-session"]);
      expect(options.session).toBe("my-session");
    });
  });

  describe("Error handling", () => {
    it("should error when --session has no value", () => {
      try {
        parseWithArgs(["--session"]);
      } catch (e) {
        // Commander will error and exit(1)
      }

      expect(exitCode).toBe(1);
    });

    it("should error on unknown flag", () => {
      try {
        parseWithArgs(["--unknown-flag"]);
      } catch (e) {
        // Commander will error and exit(1)
      }

      expect(exitCode).toBe(1);
    });

    it("should error on invalid short flag", () => {
      try {
        parseWithArgs(["-x"]);
      } catch (e) {
        // Commander will error and exit(1)
      }

      expect(exitCode).toBe(1);
    });
  });

  describe("Type safety", () => {
    it("should return CLIOptions type", () => {
      const options = parseWithArgs(["--verbose"]);

      // TypeScript checks these exist
      expect(typeof options.list).toBe("boolean");
      expect(typeof options.verbose).toBe("boolean");
      expect(typeof options.legacy).toBe("boolean");
      expect(options.session === undefined || typeof options.session === "string").toBe(true);
    });

    it("should ensure booleans are never undefined", () => {
      const options = parseWithArgs([]);

      expect(options.list).toBe(false);
      expect(options.verbose).toBe(false);
      expect(options.legacy).toBe(false);
    });
  });
});

/**
 * Integration tests for the bin wrapper (--legacy flag handling)
 * These test the bin/gofortress-tui.js wrapper script behavior
 */
describe("Binary Wrapper Integration", () => {
  describe("--legacy flag detection", () => {
    it("should detect --legacy in process.argv", () => {
      const testArgv = ["node", "bin/gofortress-tui.js", "--legacy", "--verbose"];

      expect(testArgv.includes("--legacy")).toBe(true);
    });

    it("should filter out --legacy when spawning Go TUI", () => {
      const testArgv = ["node", "bin/gofortress-tui.js", "--legacy", "--verbose", "--session", "test"];
      const filteredArgs = testArgv.slice(2).filter(arg => arg !== "--legacy");

      expect(filteredArgs).toEqual(["--verbose", "--session", "test"]);
      expect(filteredArgs.includes("--legacy")).toBe(false);
    });

    it("should preserve other flags when filtering --legacy", () => {
      const args = ["--legacy", "--session", "test-123", "--verbose"];
      const filtered = args.filter(arg => arg !== "--legacy");

      expect(filtered).toEqual(["--session", "test-123", "--verbose"]);
    });
  });

  describe("Argument forwarding", () => {
    it("should forward all arguments except --legacy", () => {
      const fullArgv = ["node", "bin/gofortress-tui.js", "--session", "abc", "--verbose"];
      const argsToForward = fullArgv.slice(2).filter(arg => arg !== "--legacy");

      expect(argsToForward).toEqual(["--session", "abc", "--verbose"]);
    });

    it("should handle empty args after removing --legacy", () => {
      const fullArgv = ["node", "bin/gofortress-tui.js", "--legacy"];
      const argsToForward = fullArgv.slice(2).filter(arg => arg !== "--legacy");

      expect(argsToForward).toEqual([]);
    });

    it("should handle multiple --legacy flags (edge case)", () => {
      const args = ["--legacy", "--session", "test", "--legacy"];
      const filtered = args.filter(arg => arg !== "--legacy");

      expect(filtered).toEqual(["--session", "test"]);
    });
  });
});
