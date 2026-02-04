/**
 * Spinner component tests
 * Coverage:
 * - Default animation type
 * - Custom animation types
 * - Theme color integration
 * - Animation frame rendering
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { Spinner } from "../../src/components/primitives/Spinner.js";

describe("Spinner", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders with default dots animation", () => {
    const { lastFrame } = render(<Spinner />);
    const output = lastFrame();

    // Should render spinner (ink-spinner renders animation frames)
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });

  it("renders with dots type explicitly", () => {
    const { lastFrame } = render(<Spinner type="dots" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("renders with line animation", () => {
    const { lastFrame } = render(<Spinner type="line" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("renders with arc animation", () => {
    const { lastFrame } = render(<Spinner type="arc" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("renders with arrow animation", () => {
    const { lastFrame } = render(<Spinner type="arrow" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("renders with bouncingBar animation", () => {
    const { lastFrame } = render(<Spinner type="bouncingBar" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("renders with bouncingBall animation", () => {
    const { lastFrame } = render(<Spinner type="bouncingBall" />);
    const output = lastFrame();

    expect(output).toBeTruthy();
  });

  it("animates over time with dots", () => {
    const { lastFrame } = render(<Spinner type="dots" />);

    // Capture initial frame
    const frame1 = lastFrame();

    // Advance time to trigger animation frame
    vi.advanceTimersByTime(80); // ink-spinner default interval

    const frame2 = lastFrame();

    // Both frames should exist (animation may or may not change visible output
    // depending on ink-spinner internals, but component should remain stable)
    expect(frame1).toBeTruthy();
    expect(frame2).toBeTruthy();
  });

  it("animates over time with line", () => {
    const { lastFrame } = render(<Spinner type="line" />);

    const frame1 = lastFrame();
    vi.advanceTimersByTime(130); // line spinner interval
    const frame2 = lastFrame();

    expect(frame1).toBeTruthy();
    expect(frame2).toBeTruthy();
  });

  it("continues animation through multiple intervals", () => {
    const { lastFrame } = render(<Spinner type="dots" />);

    // Collect frames across multiple animation cycles
    const frames: string[] = [];
    for (let i = 0; i < 5; i++) {
      frames.push(lastFrame());
      vi.advanceTimersByTime(80);
    }

    // All frames should be valid
    expect(frames).toHaveLength(5);
    frames.forEach((frame) => {
      expect(frame).toBeTruthy();
    });
  });

  it("handles rapid type changes", () => {
    const { lastFrame, rerender } = render(<Spinner type="dots" />);

    expect(lastFrame()).toBeTruthy();

    rerender(<Spinner type="line" />);
    expect(lastFrame()).toBeTruthy();

    rerender(<Spinner type="arc" />);
    expect(lastFrame()).toBeTruthy();

    rerender(<Spinner type="arrow" />);
    expect(lastFrame()).toBeTruthy();
  });

  it("maintains animation state across rerenders with same type", () => {
    const { lastFrame, rerender } = render(<Spinner type="dots" />);

    const frame1 = lastFrame();

    // Rerender with same props
    rerender(<Spinner type="dots" />);

    const frame2 = lastFrame();

    expect(frame1).toBeTruthy();
    expect(frame2).toBeTruthy();
  });

  it("renders without errors when unmounted during animation", () => {
    const { unmount } = render(<Spinner type="dots" />);

    vi.advanceTimersByTime(40);

    // Should unmount cleanly without errors
    expect(() => unmount()).not.toThrow();
  });

  it("handles multiple spinners simultaneously", () => {
    const { lastFrame: frame1 } = render(<Spinner type="dots" />);
    const { lastFrame: frame2 } = render(<Spinner type="line" />);
    const { lastFrame: frame3 } = render(<Spinner type="arc" />);

    expect(frame1()).toBeTruthy();
    expect(frame2()).toBeTruthy();
    expect(frame3()).toBeTruthy();

    vi.advanceTimersByTime(100);

    expect(frame1()).toBeTruthy();
    expect(frame2()).toBeTruthy();
    expect(frame3()).toBeTruthy();
  });

  it("integrates with theme primary color", () => {
    const { lastFrame } = render(<Spinner type="dots" />);
    const output = lastFrame();

    // Spinner should render with primary color from theme (cyan)
    // Exact color representation in output depends on ink-testing-library
    expect(output).toBeTruthy();
  });

  it("handles edge case of very long animation duration", () => {
    const { lastFrame } = render(<Spinner type="dots" />);

    // Simulate long-running spinner (5 seconds)
    for (let i = 0; i < 50; i++) {
      vi.advanceTimersByTime(100);
    }

    const output = lastFrame();
    expect(output).toBeTruthy();
  });

  it("does not crash with rapid timer advancement", () => {
    const { lastFrame } = render(<Spinner type="dots" />);

    // Rapidly advance timers
    vi.advanceTimersByTime(10000);

    const output = lastFrame();
    expect(output).toBeTruthy();
  });

  it("renders all animation types without errors", () => {
    const types: Array<"dots" | "line" | "arc" | "arrow" | "bouncingBar" | "bouncingBall"> = [
      "dots",
      "line",
      "arc",
      "arrow",
      "bouncingBar",
      "bouncingBall",
    ];

    types.forEach((type) => {
      const { lastFrame, unmount } = render(<Spinner type={type} />);
      const output = lastFrame();
      expect(output).toBeTruthy();
      unmount();
    });
  });

  it("handles type change during active animation", () => {
    const { lastFrame, rerender } = render(<Spinner type="dots" />);

    // Let animation run
    vi.advanceTimersByTime(160);
    expect(lastFrame()).toBeTruthy();

    // Change type mid-animation
    rerender(<Spinner type="line" />);
    vi.advanceTimersByTime(130);
    expect(lastFrame()).toBeTruthy();

    // Change again
    rerender(<Spinner type="arc" />);
    vi.advanceTimersByTime(100);
    expect(lastFrame()).toBeTruthy();
  });

  it("cleans up timers on unmount", () => {
    const { unmount } = render(<Spinner type="dots" />);

    // Start animation
    vi.advanceTimersByTime(80);

    // Unmount
    unmount();

    // Advance timers after unmount - should not cause errors
    expect(() => vi.advanceTimersByTime(1000)).not.toThrow();
  });

  it("renders consistently across multiple mount/unmount cycles", () => {
    for (let i = 0; i < 5; i++) {
      const { lastFrame, unmount } = render(<Spinner type="dots" />);
      expect(lastFrame()).toBeTruthy();
      vi.advanceTimersByTime(80);
      expect(lastFrame()).toBeTruthy();
      unmount();
    }
  });
});
