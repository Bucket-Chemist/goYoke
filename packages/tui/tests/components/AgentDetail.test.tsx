/**
 * Tests for AgentDetail component
 */

import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render } from "ink-testing-library";
import { AgentDetail } from "../../src/components/AgentDetail.js";
import { useStore } from "../../src/store/index.js";
import type { Agent } from "../../src/store/types.js";

describe("AgentDetail", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearAgents();
  });

  it("renders empty state when no agent selected", () => {
    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("Select an agent to view details");
  });

  it("displays agent model and tier", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-sonnet-4",
      tier: "sonnet",
      status: "running",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    const output = lastFrame();

    expect(output).toContain("claude-sonnet-4");
    expect(output).toContain("sonnet");
  });

  it("displays agent status with correct formatting", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "complete",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("complete");
  });

  it("formats duration in milliseconds for <1s", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "complete",
    };

    useStore.getState().addAgent(agent);

    // Manually set startTime and endTime for testing
    const now = Date.now();
    useStore.getState().updateAgent("agent-1", {
      endTime: now + 500,
    });

    // Get actual startTime that was set by addAgent
    const actualStartTime = useStore.getState().agents["agent-1"].startTime;
    useStore.getState().updateAgent("agent-1", {
      endTime: actualStartTime + 500,
    });

    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("500ms");
  });

  it("formats duration in seconds for <1m", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-sonnet-4",
      tier: "sonnet",
      status: "complete",
    };

    useStore.getState().addAgent(agent);

    // Get actual startTime that was set by addAgent, then set endTime 5s later
    const actualStartTime = useStore.getState().agents["agent-1"].startTime;
    useStore.getState().updateAgent("agent-1", {
      endTime: actualStartTime + 5000,
    });

    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("5.0s");
  });

  it("formats duration in minutes and seconds for >=1m", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-opus-4",
      tier: "opus",
      status: "complete",
    };

    useStore.getState().addAgent(agent);

    // Get actual startTime that was set by addAgent, then set endTime 1m 5s later
    const actualStartTime = useStore.getState().agents["agent-1"].startTime;
    useStore.getState().updateAgent("agent-1", {
      endTime: actualStartTime + 65000,
    });

    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("1m 5s");
  });

  it("displays token usage when available", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-sonnet-4",
      tier: "sonnet",
      status: "complete",
      tokenUsage: {
        input: 1250,
        output: 3420,
      },
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    const output = lastFrame();

    expect(output).toContain("1,250"); // Input tokens
    expect(output).toContain("3,420"); // Output tokens
    expect(output).toContain("4,670"); // Total tokens
  });

  it("displays description when available", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "running",
      description: "Searching codebase for patterns",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("Searching codebase for patterns");
  });

  it("uses focused color when focused prop is true", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "running",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={true} />);
    const output = lastFrame();

    // When focused, the header should have cyan color
    expect(output).toContain("Agent Detail");
  });

  it("handles agent with error status", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-sonnet-4",
      tier: "sonnet",
      status: "error",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("error");
  });

  it("updates when different agent is selected", () => {
    const agent1: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "complete",
    };

    const agent2: Omit<Agent, "startTime"> = {
      id: "agent-2",
      parentId: null,
      model: "claude-sonnet-4",
      tier: "sonnet",
      status: "running",
    };

    useStore.getState().addAgent(agent1);
    useStore.getState().addAgent(agent2);

    // Select first agent
    useStore.getState().selectAgent("agent-1");
    const { lastFrame, rerender } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("claude-haiku-4");

    // Select second agent
    useStore.getState().selectAgent("agent-2");
    rerender(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("claude-sonnet-4");
  });

  it("returns to empty state when selection is cleared", () => {
    const agent: Omit<Agent, "startTime"> = {
      id: "agent-1",
      parentId: null,
      model: "claude-haiku-4",
      tier: "haiku",
      status: "running",
    };

    useStore.getState().addAgent(agent);
    useStore.getState().selectAgent("agent-1");

    const { lastFrame, rerender } = render(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("claude-haiku-4");

    // Clear selection
    useStore.getState().selectAgent(null);
    rerender(<AgentDetail focused={false} />);
    expect(lastFrame()).toContain("Select an agent to view details");
  });
});
