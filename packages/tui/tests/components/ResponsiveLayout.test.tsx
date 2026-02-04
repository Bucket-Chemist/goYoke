/**
 * ResponsiveLayout component tests
 * Coverage:
 * - Terminal size detection via useStdout
 * - Responsive breakpoints (narrow/short/standard)
 * - Layout mode switching
 * - Terminal dimension display
 * - Component rendering
 *
 * Note: ink-testing-library uses a virtual stdout that doesn't respect
 * process.stdout modifications, so we test with default terminal size
 * and verify component structure/logic rather than specific dimensions.
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { ResponsiveLayout } from "../../src/components/ResponsiveLayout.js";

describe("ResponsiveLayout", () => {
  it("renders responsive layout test component", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should show title
    expect(output).toContain("Responsive Layout Test");
  });

  it("displays terminal size section", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should display terminal size label
    expect(output).toContain("Terminal Size:");
    // Should show some dimension (actual values depend on test environment)
    expect(output).toMatch(/\d+x\d+/);
  });

  it("displays layout mode section", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should show layout mode heading
    expect(output).toContain("Layout Mode:");
    // Should show one of the mode indicators
    const hasMode =
      output.includes("Narrow mode") ||
      output.includes("Short mode") ||
      output.includes("Standard mode");
    expect(hasMode).toBe(true);
  });

  it("renders help text for user", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should show help text
    expect(output).toContain("Resize your terminal to see this update");
    expect(output).toContain("Press Ctrl+C to exit");
  });

  it("renders without errors", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should render successfully
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });

  it("displays all required sections", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // All sections should be present
    expect(output).toContain("Responsive Layout Test");
    expect(output).toContain("Terminal Size:");
    expect(output).toContain("Layout Mode:");
    expect(output).toContain("Resize your terminal");
  });

  it("shows dimension format correctly", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should show dimensions in NxM format
    expect(output).toMatch(/\d+x\d+/);
  });

  it("component structure renders correctly", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Verify the component renders with expected structure
    // (we can't control exact terminal size in tests, but we can verify output exists)
    expect(output).toContain("Terminal Size:");
    expect(output).toContain("Layout Mode:");

    // Should contain at least one layout mode indicator
    const modes = ["Narrow mode", "Short mode", "Standard mode"];
    const hasAtLeastOneMode = modes.some(mode => output.includes(mode));
    expect(hasAtLeastOneMode).toBe(true);
  });

  it("renders title with proper formatting", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Title should be present and properly formatted
    expect(output).toContain("Responsive Layout Test");
  });

  it("includes exit instructions", () => {
    const { lastFrame } = render(<ResponsiveLayout />);
    const output = lastFrame();

    // Should include Ctrl+C instruction
    expect(output).toContain("Ctrl+C");
    expect(output).toContain("exit");
  });
});
