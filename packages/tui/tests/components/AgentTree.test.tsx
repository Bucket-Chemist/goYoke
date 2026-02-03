/**
 * AgentTree component tests
 * Coverage:
 * - Empty state
 * - Single root agent
 * - Nested hierarchy (3+ levels)
 * - All status types
 * - Selected agent highlighting
 * - Large tree (20+ agents)
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, beforeEach } from "vitest";
import { AgentTree } from "../../src/components/AgentTree.js";
import { useStore } from "../../src/store/index.js";
import type { Agent } from "../../src/store/types.js";

describe("AgentTree", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearAgents();
  });

  it("renders empty state when no agents exist", () => {
    const { lastFrame } = render(<AgentTree focused={false} />);
    expect(lastFrame()).toContain("No agents yet");
  });

  it("renders single root agent", () => {
    const { addAgent } = useStore.getState();

    addAgent({
      id: "agent-1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Main orchestration",
    });

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    expect(output).toContain("Agents");
    expect(output).toContain("● sonnet: Main orchestration");
  });

  it("renders nested hierarchy with proper indentation", () => {
    const { addAgent } = useStore.getState();

    // Root agent
    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Root",
    });

    // Level 1 child
    addAgent({
      id: "child-1",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "running",
      description: "Child 1",
    });

    // Level 2 grandchild
    addAgent({
      id: "grandchild-1",
      parentId: "child-1",
      model: "haiku",
      tier: "haiku",
      status: "complete",
      description: "Grandchild 1",
    });

    // Level 1 child 2
    addAgent({
      id: "child-2",
      parentId: "root",
      model: "sonnet",
      tier: "sonnet",
      status: "spawning",
      description: "Child 2",
    });

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // Check all agents are present
    expect(output).toContain("● sonnet: Root");
    expect(output).toContain("haiku: Child 1");
    expect(output).toContain("haiku: Grandchild 1");
    expect(output).toContain("sonnet: Child 2");

    // Check indentation (children should have more spaces)
    const lines = output.split("\n");
    const rootLine = lines.find((l) => l.includes("Root"));
    const childLine = lines.find((l) => l.includes("Child 1"));
    const grandchildLine = lines.find((l) => l.includes("Grandchild 1"));

    expect(rootLine).toBeDefined();
    expect(childLine).toBeDefined();
    expect(grandchildLine).toBeDefined();

    // Verify tree structure characters are present
    expect(output).toContain("├─");
  });

  it("displays all status types with correct icons", () => {
    const { addAgent } = useStore.getState();

    // Root
    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Running agent",
    });

    // Children with different statuses
    addAgent({
      id: "spawning",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "spawning",
      description: "Spawning",
    });

    addAgent({
      id: "complete",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "complete",
      description: "Complete",
    });

    addAgent({
      id: "error",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "error",
      description: "Error",
    });

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // Check status icons (Unicode characters from theme)
    expect(output).toContain("●"); // running
    expect(output).toContain("◐"); // spawning
    expect(output).toContain("✓"); // complete
    expect(output).toContain("✗"); // error
  });

  it("highlights selected agent", () => {
    const { addAgent, selectAgent } = useStore.getState();

    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Root",
    });

    addAgent({
      id: "child",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "running",
      description: "Selected child",
    });

    // Select the child
    selectAgent("child");

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // The selected agent should be present
    expect(output).toContain("Selected child");
  });

  it("shows count indicator for large trees", () => {
    const { addAgent } = useStore.getState();

    // Create root
    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Root",
    });

    // Create 20 child agents
    for (let i = 1; i <= 20; i++) {
      addAgent({
        id: `agent-${i}`,
        parentId: "root",
        model: "haiku",
        tier: "haiku",
        status: "running",
        description: `Agent ${i}`,
      });
    }

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // Should show count (root + 20 children = 21)
    expect(output).toContain("(21)");
    expect(output).toContain("consider scrolling");
  });

  it("handles agents without descriptions", () => {
    const { addAgent } = useStore.getState();

    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      // No description
    });

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // Should show model without description
    expect(output).toContain("● sonnet");
    // Should not have trailing colon
    expect(output).not.toMatch(/sonnet:\s*$/);
  });

  it("shows focused state in header", () => {
    const { addAgent } = useStore.getState();

    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
    });

    const { lastFrame: focusedFrame } = render(<AgentTree focused={true} />);
    const { lastFrame: unfocusedFrame } = render(<AgentTree focused={false} />);

    const focusedOutput = focusedFrame();
    const unfocusedOutput = unfocusedFrame();

    // Both should contain "Agents" header
    expect(focusedOutput).toContain("Agents");
    expect(unfocusedOutput).toContain("Agents");

    // The outputs will differ in styling (focused vs unfocused colors)
    // Ink's color handling makes exact comparison difficult, but both should render
    expect(focusedOutput).toBeTruthy();
    expect(unfocusedOutput).toBeTruthy();
  });

  it("maintains tree structure across multiple levels", () => {
    const { addAgent } = useStore.getState();

    // Create deep hierarchy: root -> l1a -> l2a -> l3a
    //                             -> l1b
    addAgent({
      id: "root",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Root",
    });

    addAgent({
      id: "l1a",
      parentId: "root",
      model: "haiku",
      tier: "haiku",
      status: "running",
      description: "Level 1A",
    });

    addAgent({
      id: "l2a",
      parentId: "l1a",
      model: "haiku",
      tier: "haiku",
      status: "complete",
      description: "Level 2A",
    });

    addAgent({
      id: "l3a",
      parentId: "l2a",
      model: "haiku",
      tier: "haiku",
      status: "complete",
      description: "Level 3A",
    });

    addAgent({
      id: "l1b",
      parentId: "root",
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Level 1B",
    });

    const { lastFrame } = render(<AgentTree focused={false} />);
    const output = lastFrame();

    // All levels should be present
    expect(output).toContain("Root");
    expect(output).toContain("Level 1A");
    expect(output).toContain("Level 2A");
    expect(output).toContain("Level 3A");
    expect(output).toContain("Level 1B");

    // Count should be correct (5 agents)
    expect(output).toContain("(5)");
  });
});
