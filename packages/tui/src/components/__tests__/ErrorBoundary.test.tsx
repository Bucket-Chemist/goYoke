import { describe, it, expect, beforeEach, vi } from "vitest";
import { render } from "ink-testing-library";
import React from "react";
import { Text } from "ink";
import { ErrorBoundary } from "../ErrorBoundary.js";
import * as loggerModule from "../../utils/logger.js";

// Mock the logger module
vi.mock("../../utils/logger.js", () => ({
  logger: {
    error: vi.fn(),
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
  },
  getRecentLogs: vi.fn(() => []),
  getRecentErrors: vi.fn(() => []),
  clearLogs: vi.fn(),
}));

// Component that throws an error
const ThrowError = ({ message }: { message: string }) => {
  throw new Error(message);
};

// Component that works normally
const WorkingComponent = () => <Text>All good!</Text>;

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Suppress console.error for these tests (React error boundary logs to console)
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when no error occurs", () => {
    const { lastFrame } = render(
      <ErrorBoundary>
        <WorkingComponent />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain("All good!");
  });

  it("renders error message when child throws", () => {
    const { lastFrame } = render(
      <ErrorBoundary>
        <ThrowError message="Test error message" />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain("Component error: Test error message");
  });

  it("logs error to logger when component throws", () => {
    render(
      <ErrorBoundary>
        <ThrowError message="Logger test error" />
      </ErrorBoundary>
    );

    expect(loggerModule.logger.error).toHaveBeenCalledWith(
      "Component error caught by ErrorBoundary",
      expect.objectContaining({
        message: "Logger test error",
        errorName: "Error",
      })
    );
  });

  it("captures error stack trace in logger", () => {
    render(
      <ErrorBoundary>
        <ThrowError message="Stack trace test" />
      </ErrorBoundary>
    );

    expect(loggerModule.logger.error).toHaveBeenCalledWith(
      "Component error caught by ErrorBoundary",
      expect.objectContaining({
        stack: expect.stringContaining("Error: Stack trace test"),
      })
    );
  });

  it("renders custom fallback when provided", () => {
    const customFallback = <Text color="yellow">Custom error UI</Text>;

    const { lastFrame } = render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowError message="Custom fallback test" />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain("Custom error UI");
    expect(lastFrame()).not.toContain("Component error:");
  });

  it("includes component stack in error context", () => {
    render(
      <ErrorBoundary>
        <ThrowError message="Component stack test" />
      </ErrorBoundary>
    );

    expect(loggerModule.logger.error).toHaveBeenCalledWith(
      "Component error caught by ErrorBoundary",
      expect.objectContaining({
        componentStack: expect.any(String),
      })
    );
  });
});
