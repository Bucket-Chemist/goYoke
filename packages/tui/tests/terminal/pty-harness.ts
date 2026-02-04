/**
 * PTY Harness for Terminal Compatibility Testing
 *
 * Provides utilities for spawning the TUI in a pseudo-terminal,
 * capturing ANSI output, sending input, and testing resize behavior.
 *
 * Built on node-pty for realistic terminal emulation.
 */

import { spawn as ptySpawn, IPty } from "node-pty";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = resolve(__dirname, "../..");
// Use tsx to run source directly (avoids dist/index.js double-shebang issue)
const srcPath = resolve(projectRoot, "src/index.tsx");
const tsxPath = resolve(projectRoot, "node_modules/.bin/tsx");

export interface PtyHarnessOptions {
  cols?: number;
  rows?: number;
  env?: Record<string, string>;
  timeout?: number; // milliseconds before force kill
}

export interface AnsiAnalysis {
  hasColorCodes: boolean;
  hasUnicode: boolean;
  colors: string[];
  unicodeChars: string[];
  rawOutput: string;
}

/**
 * Spawns the TUI in a pseudo-terminal
 *
 * @param options - Terminal configuration
 * @returns PTY instance and utilities
 */
export function spawnTUIInPty(options: PtyHarnessOptions = {}) {
  const {
    cols = 80,
    rows = 24,
    env = {},
    timeout = 10000,
  } = options;

  // Run source via tsx (avoids build artifacts with double shebang)
  const ptyProcess = ptySpawn(tsxPath, [srcPath], {
    name: "xterm-256color",
    cols,
    rows,
    cwd: projectRoot,
    env: {
      ...process.env,
      TERM: "xterm-256color",
      FORCE_COLOR: "3", // Force truecolor support
      COLORTERM: "truecolor",
      ...env,
    },
  });

  let output = "";
  let killed = false;
  let timeoutHandle: NodeJS.Timeout | null = null;

  // Accumulate all output
  ptyProcess.onData((data) => {
    output += data;
  });

  // Setup timeout if specified
  if (timeout > 0) {
    timeoutHandle = setTimeout(() => {
      if (!killed) {
        ptyProcess.kill("SIGKILL");
        killed = true;
      }
    }, timeout);
  }

  return {
    pty: ptyProcess,
    getOutput: () => output,
    clearOutput: () => {
      output = "";
    },
    cleanup: () => {
      if (timeoutHandle) {
        clearTimeout(timeoutHandle);
      }
      if (!killed) {
        ptyProcess.kill();
        killed = true;
      }
    },
  };
}

/**
 * Sends keyboard input sequences to the PTY
 *
 * @param pty - The PTY instance
 * @param input - Input string or control sequence
 */
export function sendInput(pty: IPty, input: string): void {
  pty.write(input);
}

/**
 * Sends common control sequences
 */
export const ControlSequences = {
  CTRL_C: "\x03",
  CTRL_D: "\x04",
  ENTER: "\r",
  ESCAPE: "\x1b",
  ARROW_UP: "\x1b[A",
  ARROW_DOWN: "\x1b[B",
  ARROW_LEFT: "\x1b[D",
  ARROW_RIGHT: "\x1b[C",
  TAB: "\t",
} as const;

/**
 * Resizes the terminal during operation
 *
 * @param pty - The PTY instance
 * @param cols - New column count
 * @param rows - New row count
 */
export function resizeTerminal(pty: IPty, cols: number, rows: number): void {
  pty.resize(cols, rows);
}

/**
 * Analyzes captured output for ANSI sequences and Unicode
 *
 * @param output - Raw terminal output with ANSI sequences
 * @returns Analysis of color codes and Unicode characters
 */
export function analyzeOutput(output: string): AnsiAnalysis {
  const analysis: AnsiAnalysis = {
    hasColorCodes: false,
    hasUnicode: false,
    colors: [],
    unicodeChars: [],
    rawOutput: output,
  };

  // Detect ANSI color codes
  const ansiColorPattern = /\x1b\[([0-9;]+)m/g;
  const colorMatches = [...output.matchAll(ansiColorPattern)];

  if (colorMatches.length > 0) {
    analysis.hasColorCodes = true;
    analysis.colors = colorMatches.map((m) => m[1] || "");
  }

  // Detect Unicode box-drawing characters
  // Common box-drawing range: U+2500 to U+257F
  const unicodeBoxPattern = /[\u2500-\u257F]/g;
  const unicodeMatches = [...output.matchAll(unicodeBoxPattern)];

  if (unicodeMatches.length > 0) {
    analysis.hasUnicode = true;
    analysis.unicodeChars = [...new Set(unicodeMatches.map((m) => m[0]))];
  }

  return analysis;
}

/**
 * Waits for specific output to appear in the PTY stream
 *
 * @param getOutput - Function to get current output
 * @param pattern - String or RegExp to match
 * @param timeout - Maximum wait time in milliseconds
 * @returns Promise that resolves when pattern is found
 */
export function waitForOutput(
  getOutput: () => string,
  pattern: string | RegExp,
  timeout = 5000
): Promise<void> {
  return new Promise((resolve, reject) => {
    const startTime = Date.now();
    const checkInterval = 100;

    const checker = setInterval(() => {
      const output = getOutput();
      const matches =
        typeof pattern === "string"
          ? output.includes(pattern)
          : pattern.test(output);

      if (matches) {
        clearInterval(checker);
        resolve();
      } else if (Date.now() - startTime > timeout) {
        clearInterval(checker);
        reject(
          new Error(
            `Timeout waiting for pattern: ${pattern}. Output: ${output.slice(0, 200)}...`
          )
        );
      }
    }, checkInterval);
  });
}

/**
 * Extracts visible text from ANSI output (strips control sequences)
 *
 * @param output - Raw terminal output with ANSI sequences
 * @returns Plain text without ANSI codes
 */
export function stripAnsi(output: string): string {
  // Remove all ANSI escape sequences
  return output.replace(
    /[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g,
    ""
  );
}

/**
 * Helper to verify terminal output contains expected elements
 */
export interface OutputExpectations {
  hasColor?: boolean;
  hasUnicode?: boolean;
  containsText?: string[];
  minColorCodes?: number;
  minUnicodeChars?: number;
}

export function verifyOutput(
  output: string,
  expectations: OutputExpectations
): { passed: boolean; failures: string[] } {
  const analysis = analyzeOutput(output);
  const plainText = stripAnsi(output);
  const failures: string[] = [];

  if (expectations.hasColor !== undefined) {
    if (expectations.hasColor && !analysis.hasColorCodes) {
      failures.push("Expected color codes but none found");
    } else if (!expectations.hasColor && analysis.hasColorCodes) {
      failures.push("Found color codes when none expected");
    }
  }

  if (expectations.hasUnicode !== undefined) {
    if (expectations.hasUnicode && !analysis.hasUnicode) {
      failures.push("Expected Unicode characters but none found");
    } else if (!expectations.hasUnicode && analysis.hasUnicode) {
      failures.push("Found Unicode characters when none expected");
    }
  }

  if (expectations.containsText) {
    for (const text of expectations.containsText) {
      if (!plainText.includes(text)) {
        failures.push(`Expected text not found: "${text}"`);
      }
    }
  }

  if (expectations.minColorCodes !== undefined) {
    if (analysis.colors.length < expectations.minColorCodes) {
      failures.push(
        `Expected at least ${expectations.minColorCodes} color codes, found ${analysis.colors.length}`
      );
    }
  }

  if (expectations.minUnicodeChars !== undefined) {
    if (analysis.unicodeChars.length < expectations.minUnicodeChars) {
      failures.push(
        `Expected at least ${expectations.minUnicodeChars} Unicode chars, found ${analysis.unicodeChars.length}`
      );
    }
  }

  return {
    passed: failures.length === 0,
    failures,
  };
}
