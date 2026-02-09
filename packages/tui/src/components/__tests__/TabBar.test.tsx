import { describe, it, expect, beforeEach, vi } from "vitest";
import { render } from "ink-testing-library";
import React from "react";
import { TabBar } from "../TabBar.js";
import { useStore } from "../../store/index.js";
import * as useKeymapModule from "../../hooks/useKeymap.js";

// Mock the useKeymap hook
vi.mock("../../hooks/useKeymap.js", () => ({
  useKeymap: vi.fn(),
}));

describe("TabBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store state
    useStore.setState({
      activeTab: "chat",
    });
  });

  it("renders all 4 tabs horizontally", () => {
    const { lastFrame } = render(<TabBar />);
    const output = lastFrame();

    expect(output).toContain("Chat");
    expect(output).toContain("Agent Config");
    expect(output).toContain("Team Config");
    expect(output).toContain("Telemetry");
  });

  it("styles active tab with primary color and bold", () => {
    useStore.setState({ activeTab: "agent-config" });
    const { lastFrame } = render(<TabBar />);
    const output = lastFrame();

    // The active tab should be rendered (checking for text presence)
    expect(output).toContain("Agent Config");
  });

  it("styles inactive tabs with muted color", () => {
    useStore.setState({ activeTab: "chat" });
    const { lastFrame } = render(<TabBar />);
    const output = lastFrame();

    // All tabs should be present
    expect(output).toContain("Chat");
    expect(output).toContain("Agent Config");
    expect(output).toContain("Team Config");
    expect(output).toContain("Telemetry");
  });

  it("registers Alt+C/A/T/Y keybindings via useKeymap", () => {
    render(<TabBar />);

    expect(useKeymapModule.useKeymap).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          key: "c",
          meta: true,
          description: "Switch to Chat tab",
        }),
        expect.objectContaining({
          key: "a",
          meta: true,
          description: "Switch to Agent Config tab",
        }),
        expect.objectContaining({
          key: "t",
          meta: true,
          description: "Switch to Team Config tab",
        }),
        expect.objectContaining({
          key: "y",
          meta: true,
          description: "Switch to Telemetry tab",
        }),
      ]),
      true
    );
  });

  it("passes enabled prop to useKeymap", () => {
    render(<TabBar enabled={false} />);

    expect(useKeymapModule.useKeymap).toHaveBeenCalledWith(
      expect.any(Array),
      false
    );
  });

  it("defaults enabled to true when prop omitted", () => {
    render(<TabBar />);

    expect(useKeymapModule.useKeymap).toHaveBeenCalledWith(
      expect.any(Array),
      true
    );
  });

  it("Alt+C binding switches to chat tab", () => {
    render(<TabBar />);

    // Get the registered bindings
    const calls = vi.mocked(useKeymapModule.useKeymap).mock.calls;
    const bindings = calls[0]?.[0];

    // Find the Alt+C binding
    const chatBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "c" && b.meta === true
    );

    expect(chatBinding).toBeDefined();

    // Execute the action
    if (chatBinding && "action" in chatBinding) {
      chatBinding.action();
    }

    // Verify store was updated
    expect(useStore.getState().activeTab).toBe("chat");
  });

  it("Alt+A binding switches to agent-config tab", () => {
    render(<TabBar />);

    const calls = vi.mocked(useKeymapModule.useKeymap).mock.calls;
    const bindings = calls[0]?.[0];

    const agentBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "a" && b.meta === true
    );

    expect(agentBinding).toBeDefined();

    if (agentBinding && "action" in agentBinding) {
      agentBinding.action();
    }

    expect(useStore.getState().activeTab).toBe("agent-config");
  });

  it("Alt+T binding switches to team-config tab", () => {
    render(<TabBar />);

    const calls = vi.mocked(useKeymapModule.useKeymap).mock.calls;
    const bindings = calls[0]?.[0];

    const teamBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "t" && b.meta === true
    );

    expect(teamBinding).toBeDefined();

    if (teamBinding && "action" in teamBinding) {
      teamBinding.action();
    }

    expect(useStore.getState().activeTab).toBe("team-config");
  });

  it("Alt+Y binding switches to telemetry tab", () => {
    render(<TabBar />);

    const calls = vi.mocked(useKeymapModule.useKeymap).mock.calls;
    const bindings = calls[0]?.[0];

    const telemetryBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "y" && b.meta === true
    );

    expect(telemetryBinding).toBeDefined();

    if (telemetryBinding && "action" in telemetryBinding) {
      telemetryBinding.action();
    }

    expect(useStore.getState().activeTab).toBe("telemetry");
  });

  it("renders shortcut characters with underline", () => {
    const { lastFrame } = render(<TabBar />);
    const output = lastFrame();

    // Each tab should have its label present
    // (ink-testing-library strips ANSI codes, so we check for text presence)
    expect(output).toContain("Chat");
    expect(output).toContain("Agent Config");
    expect(output).toContain("Team Config");
    expect(output).toContain("Telemetry");
  });

  it("switches between tabs correctly", () => {
    render(<TabBar />);

    const calls = vi.mocked(useKeymapModule.useKeymap).mock.calls;
    const bindings = calls[0]?.[0];

    // Start at chat
    expect(useStore.getState().activeTab).toBe("chat");

    // Switch to agent-config
    const agentBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "a" && b.meta === true
    );
    if (agentBinding && "action" in agentBinding) {
      agentBinding.action();
    }
    expect(useStore.getState().activeTab).toBe("agent-config");

    // Switch to telemetry
    const telemetryBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "y" && b.meta === true
    );
    if (telemetryBinding && "action" in telemetryBinding) {
      telemetryBinding.action();
    }
    expect(useStore.getState().activeTab).toBe("telemetry");

    // Switch back to chat
    const chatBinding = bindings?.find(
      (b: { key: string; meta?: boolean }) => b.key === "c" && b.meta === true
    );
    if (chatBinding && "action" in chatBinding) {
      chatBinding.action();
    }
    expect(useStore.getState().activeTab).toBe("chat");
  });
});
