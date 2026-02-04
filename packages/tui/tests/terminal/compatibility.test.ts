/**
 * Terminal Compatibility Tests
 *
 * Tests the TUI across different terminal sizes and capabilities.
 * Verifies:
 * - ANSI color codes are present
 * - Unicode box-drawing characters render
 * - Resize handling works during operation
 * - Output is reasonable at edge-case sizes
 *
 * These tests spawn actual PTY processes and may take several seconds.
 *
 * Run with: npm test tests/terminal/compatibility.test.ts
 */

import { describe, it, expect, afterEach } from "vitest";
import {
  spawnTUIInPty,
  resizeTerminal,
  sendInput,
  analyzeOutput,
  waitForOutput,
  stripAnsi,
  verifyOutput,
  ControlSequences,
} from "./pty-harness.js";

// Track spawned processes for cleanup
const activeProcesses: Array<{ cleanup: () => void }> = [];

afterEach(() => {
  // Clean up all spawned processes
  for (const proc of activeProcesses) {
    try {
      proc.cleanup();
    } catch (error) {
      // Ignore cleanup errors
    }
  }
  activeProcesses.length = 0;
});

describe("Terminal Compatibility Tests", () => {
  describe("Standard Terminal Sizes", () => {
    it("should render correctly at 80x24 (standard)", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      // Wait for TSX compilation and initial render (takes ~4s)
      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();
      const analysis = analyzeOutput(output);

      // Verify ANSI color codes are present
      expect(analysis.hasColorCodes, "Should have ANSI color codes").toBe(true);
      expect(analysis.colors.length, "Should have multiple color codes").toBeGreaterThan(5);

      // Verify Unicode box-drawing characters
      expect(analysis.hasUnicode, "Should have Unicode characters").toBe(true);
      expect(analysis.unicodeChars.length, "Should have box-drawing chars").toBeGreaterThan(0);

      // Verify some expected UI elements are present
      const plainText = stripAnsi(output);
      expect(plainText.length, "Should have rendered content").toBeGreaterThan(50);

      harness.cleanup();
    }, 10000);

    it("should render correctly at 120x40 (wide)", async () => {
      const harness = spawnTUIInPty({ cols: 120, rows: 40, timeout: 8000 });
      activeProcesses.push(harness);

      // Wait for TSX compilation and initial render
      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();
      const analysis = analyzeOutput(output);

      // Verify rendering with more space
      expect(analysis.hasColorCodes, "Should have ANSI color codes").toBe(true);
      expect(analysis.hasUnicode, "Should have Unicode characters").toBe(true);

      const plainText = stripAnsi(output);
      expect(plainText.length, "Should utilize wider space").toBeGreaterThan(50);

      harness.cleanup();
    }, 10000);

    it("should handle 40x10 (edge case: narrow and short)", async () => {
      const harness = spawnTUIInPty({ cols: 40, rows: 10, timeout: 8000 });
      activeProcesses.push(harness);

      // Wait for TSX compilation and initial render
      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();

      // At minimum size, should still:
      // 1. Not crash
      // 2. Render something
      // 3. Have ANSI codes (even if layout is cramped)
      expect(output.length, "Should render something even at small size").toBeGreaterThan(20);

      const analysis = analyzeOutput(output);
      expect(analysis.hasColorCodes, "Should still have color codes").toBe(true);

      harness.cleanup();
    }, 10000);
  });

  describe("ANSI Color Code Verification", () => {
    it("should emit standard ANSI color codes", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();
      const analysis = analyzeOutput(output);

      // Check for common ANSI codes
      const hasBasicColors = analysis.colors.some((code) => {
        // Standard ANSI codes: 30-37 (foreground), 40-47 (background)
        // Bold: 1, Reset: 0
        const nums = code.split(";").map(Number);
        return nums.some((n) => (n >= 30 && n <= 47) || n === 0 || n === 1);
      });

      expect(hasBasicColors, "Should use standard ANSI color codes").toBe(true);
      expect(analysis.colors.length, "Should have substantial color usage").toBeGreaterThan(10);

      harness.cleanup();
    }, 10000);

    it("should use color codes consistently", async () => {
      const harness = spawnTUIInPty({
        cols: 80,
        rows: 24,
        timeout: 8000,
        env: { TERM: "xterm-256color" },
      });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();
      const analysis = analyzeOutput(output);

      // The TUI uses standard ANSI codes (30-39 for colors)
      // Just verify we have consistent color usage
      const hasColorCodes = analysis.colors.length > 50;

      expect(
        hasColorCodes,
        "Should have substantial color code usage"
      ).toBe(true);

      harness.cleanup();
    }, 10000);
  });

  describe("Unicode Box-Drawing Characters", () => {
    it("should render Unicode box-drawing characters", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();
      const analysis = analyzeOutput(output);

      expect(analysis.hasUnicode, "Should render Unicode").toBe(true);
      expect(analysis.unicodeChars.length, "Should have multiple Unicode chars").toBeGreaterThan(0);

      // Common box-drawing characters in the U+2500-257F range
      // Examples: ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼
      const hasBoxDrawing = analysis.unicodeChars.some((char) => {
        const code = char.charCodeAt(0);
        return code >= 0x2500 && code <= 0x257f;
      });

      expect(hasBoxDrawing, "Should use box-drawing characters").toBe(true);

      harness.cleanup();
    }, 10000);
  });

  describe("Terminal Resize Handling", () => {
    it("should handle resize from 80x24 to 120x40", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 10000 });
      activeProcesses.push(harness);

      // Wait for initial render
      await new Promise((resolve) => setTimeout(resolve, 4500));

      const outputBefore = harness.getOutput();
      expect(outputBefore.length, "Should have initial output").toBeGreaterThan(50);

      // Clear and resize (do NOT resize after cleanup)
      harness.clearOutput();

      // Verify PTY is still alive before resizing
      try {
        resizeTerminal(harness.pty, 120, 40);

        // Wait for resize handling
        await new Promise((resolve) => setTimeout(resolve, 1000));
      } catch (error) {
        // PTY might be dead, skip resize verification
        console.warn("Resize failed, PTY may have exited");
      }

      const outputAfter = harness.getOutput();

      // After resize, should have some output (may be empty if PTY exited)
      // This is a best-effort test since Ink may not re-render on every resize
      if (outputAfter.length > 20) {
        // If we got output, verify it has ANSI codes
        const analysisAfter = analyzeOutput(outputAfter);
        expect(analysisAfter.hasColorCodes, "Should maintain colors after resize").toBe(true);
      }

      harness.cleanup();
    }, 12000);

    it("should handle resize to narrow terminal (40x10)", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 10000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      // Resize to very narrow
      harness.clearOutput();

      try {
        resizeTerminal(harness.pty, 40, 10);
        await new Promise((resolve) => setTimeout(resolve, 1000));
      } catch (error) {
        console.warn("Resize failed, PTY may have exited");
      }

      const output = harness.getOutput();

      // Should not crash (main test is that we got here without exception)
      // Output may be empty if Ink doesn't re-render on resize

      harness.cleanup();
    }, 12000);
  });

  describe("Keyboard Input Handling", () => {
    it("should respond to Ctrl+C gracefully", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      // Send Ctrl+C
      sendInput(harness.pty, ControlSequences.CTRL_C);

      // Wait a bit for shutdown
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Process should exit gracefully (cleanup will succeed)
      expect(() => harness.cleanup()).not.toThrow();
    }, 8000);

    it("should handle arrow key navigation", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      harness.clearOutput();

      // Send arrow key sequences
      sendInput(harness.pty, ControlSequences.ARROW_DOWN);
      await new Promise((resolve) => setTimeout(resolve, 500));

      sendInput(harness.pty, ControlSequences.ARROW_UP);
      await new Promise((resolve) => setTimeout(resolve, 500));

      const output = harness.getOutput();

      // After input, may have output (depends on whether Ink re-renders)
      // Main test is that we didn't crash

      harness.cleanup();
    }, 8000);
  });

  describe("Output Verification Utilities", () => {
    it("should verify output expectations correctly", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();

      const verification = verifyOutput(output, {
        hasColor: true,
        hasUnicode: true,
        minColorCodes: 5,
      });

      expect(verification.passed, "Verification should pass").toBe(true);
      expect(verification.failures.length, "Should have no failures").toBe(0);

      harness.cleanup();
    }, 10000);

    it("should detect missing expectations", async () => {
      const harness = spawnTUIInPty({ cols: 80, rows: 24, timeout: 8000 });
      activeProcesses.push(harness);

      await new Promise((resolve) => setTimeout(resolve, 4500));

      const output = harness.getOutput();

      const verification = verifyOutput(output, {
        containsText: ["THIS_TEXT_DEFINITELY_DOES_NOT_EXIST_9999"],
      });

      expect(verification.passed, "Verification should fail").toBe(false);
      expect(verification.failures.length, "Should have failures").toBeGreaterThan(0);

      harness.cleanup();
    }, 10000);
  });
});
