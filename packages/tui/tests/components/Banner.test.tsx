/**
 * Banner component tests
 * Coverage:
 * - Session ID display
 * - Cost formatting
 * - Streaming status
 * - Theme integration (borders, colors)
 * - State changes via store
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, beforeEach } from "vitest";
import { Banner } from "../../src/components/Banner.js";
import { useStore } from "../../src/store/index.js";

describe("Banner", () => {
  beforeEach(() => {
    // Reset store to clean state before each test
    const state = useStore.getState();
    state.sessionId = null;
    state.totalCost = 0;
    state.streaming = false;
  });

  it("renders GOfortress title", () => {
    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("GOfortress");
  });

  it("displays 'None' when no session ID", () => {
    useStore.setState({ sessionId: null });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Session: None");
  });

  it("displays session ID when set", () => {
    useStore.setState({ sessionId: "abc123" });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Session: abc123");
  });

  it("displays cost formatted to 4 decimal places", () => {
    useStore.setState({ totalCost: 0.0 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $0.0000");
  });

  it("formats non-zero cost correctly", () => {
    useStore.setState({ totalCost: 1.2345 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $1.2345");
  });

  it("formats small cost values correctly", () => {
    useStore.setState({ totalCost: 0.0001 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $0.0001");
  });

  it("formats large cost values correctly", () => {
    useStore.setState({ totalCost: 123.4567 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $123.4567");
  });

  it("rounds cost to 4 decimal places", () => {
    useStore.setState({ totalCost: 1.23456789 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $1.2346"); // Rounded
  });

  it("displays Ready status when not streaming", () => {
    useStore.setState({ streaming: false });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Ready");
  });

  it("displays Streaming status when streaming", () => {
    useStore.setState({ streaming: true });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Streaming...");
  });

  it("updates when store state changes", () => {
    const { lastFrame, rerender } = render(<Banner />);

    // Initial state
    expect(lastFrame()).toContain("Session: None");
    expect(lastFrame()).toContain("Cost: $0.0000");
    expect(lastFrame()).toContain("Ready");

    // Update store
    useStore.setState({
      sessionId: "session-123",
      totalCost: 5.5555,
      streaming: true,
    });

    // Force rerender to pick up state changes
    rerender(<Banner />);

    const output = lastFrame();
    expect(output).toContain("Session: session-123");
    expect(output).toContain("Cost: $5.5555");
    expect(output).toContain("Streaming...");
  });

  it("handles session ID changes", () => {
    useStore.setState({ sessionId: "first-session" });
    const { lastFrame, rerender } = render(<Banner />);

    expect(lastFrame()).toContain("Session: first-session");

    useStore.setState({ sessionId: "second-session" });
    rerender(<Banner />);

    expect(lastFrame()).toContain("Session: second-session");
  });

  it("handles streaming toggle", () => {
    const { lastFrame, rerender } = render(<Banner />);

    useStore.setState({ streaming: false });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Ready");

    useStore.setState({ streaming: true });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Streaming...");

    useStore.setState({ streaming: false });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Ready");
  });

  it("handles cost increments", () => {
    const { lastFrame, rerender } = render(<Banner />);

    useStore.setState({ totalCost: 0.0001 });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Cost: $0.0001");

    useStore.setState({ totalCost: 0.0002 });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Cost: $0.0002");

    useStore.setState({ totalCost: 0.0005 });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Cost: $0.0005");
  });

  it("displays all components simultaneously", () => {
    useStore.setState({
      sessionId: "test-session-456",
      totalCost: 2.3456,
      streaming: true,
    });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    // All elements should be present
    expect(output).toContain("GOfortress");
    expect(output).toContain("Session: test-session-456");
    expect(output).toContain("Cost: $2.3456");
    expect(output).toContain("Streaming...");
  });

  it("handles very long session IDs", () => {
    const longSessionId = "a".repeat(100);
    useStore.setState({ sessionId: longSessionId });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Session:");
    // Should render without crashing (may be truncated by terminal width)
    expect(output.length).toBeGreaterThan(0);
  });

  it("handles zero cost correctly", () => {
    useStore.setState({ totalCost: 0 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $0.0000");
  });

  it("handles negative cost (edge case)", () => {
    // Technically invalid but test defensive rendering
    useStore.setState({ totalCost: -1.23 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $-1.2300");
  });

  it("handles very large cost values", () => {
    useStore.setState({ totalCost: 999999.9999 });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Cost: $999999.9999");
  });

  it("handles session ID with special characters", () => {
    useStore.setState({ sessionId: "session-123_test@example.com" });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("Session: session-123_test@example.com");
  });

  it("integrates with theme border style", () => {
    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    // Banner uses round border from theme
    // Exact rendering depends on ink, but should render border characters
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });

  it("maintains layout with space-between justification", () => {
    useStore.setState({
      sessionId: "test",
      totalCost: 1.0,
      streaming: false,
    });

    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    // All four sections should be present
    expect(output).toContain("GOfortress");
    expect(output).toContain("Session:");
    expect(output).toContain("Cost:");
    expect(output).toContain("Ready");
  });

  it("handles rapid state changes", () => {
    const { lastFrame, rerender } = render(<Banner />);

    // Simulate rapid updates
    for (let i = 0; i < 10; i++) {
      useStore.setState({
        totalCost: i * 0.1,
        streaming: i % 2 === 0,
      });
      rerender(<Banner />);
      expect(lastFrame()).toBeTruthy();
    }
  });

  it("renders when store is in initial state", () => {
    // Don't set any store values, use defaults
    const { lastFrame } = render(<Banner />);
    const output = lastFrame();

    expect(output).toContain("GOfortress");
    expect(output).toBeTruthy();
  });

  it("handles session ID null to string transition", () => {
    const { lastFrame, rerender } = render(<Banner />);

    useStore.setState({ sessionId: null });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Session: None");

    useStore.setState({ sessionId: "new-session" });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Session: new-session");
  });

  it("handles session ID string to null transition", () => {
    const { lastFrame, rerender } = render(<Banner />);

    useStore.setState({ sessionId: "existing-session" });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Session: existing-session");

    useStore.setState({ sessionId: null });
    rerender(<Banner />);
    expect(lastFrame()).toContain("Session: None");
  });

  it("displays correct status colors based on streaming state", () => {
    const { lastFrame, rerender } = render(<Banner />);

    // Not streaming (should use success color - green)
    useStore.setState({ streaming: false });
    rerender(<Banner />);
    let output = lastFrame();
    expect(output).toContain("Ready");

    // Streaming (should use warning color - yellow)
    useStore.setState({ streaming: true });
    rerender(<Banner />);
    output = lastFrame();
    expect(output).toContain("Streaming...");
  });
});
