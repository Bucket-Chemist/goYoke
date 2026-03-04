/**
 * useAgentSync — unit tests for live subagent activity extraction.
 *
 * Covers the two newly-exported pure functions:
 *   - extractSubagentActivity(messages, agentId)
 *   - syncSubagentActivity(messages, snapshots)
 *
 * Both functions are tested without React rendering. extractSubagentActivity
 * is a pure transform over Message[]; syncSubagentActivity calls into the
 * Zustand store (mocked below) but has no React dependency.
 *
 * The agentActivity.js module is NOT mocked here so that extractToolTarget
 * (used inside extractSubagentActivity) exercises its real priority-key logic.
 */

import { vi, describe, it, expect, beforeEach } from "vitest";

// ── Store mock ────────────────────────────────────────────────────────────────

const mutableAgents: Record<string, { status: string; activity?: unknown }> =
  {};

const mockUpdateAgentActivity = vi.fn();

const mockStore = {
  get agents(): Record<string, { status: string; activity?: unknown }> {
    return mutableAgents;
  },
  updateAgentActivity: mockUpdateAgentActivity,
};

vi.mock("../store/index.js", () => ({
  useStore: Object.assign(vi.fn(), {
    getState: vi.fn(() => mockStore),
    subscribe: vi.fn(() => () => {}),
  }),
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

import type { Message } from "../store/types.js";

/**
 * Build a minimal Message with the given content blocks and optional
 * subagentToolUseId.
 */
function msg(
  content: Message["content"],
  subagentToolUseId?: string,
  role: "user" | "assistant" = "assistant",
): Message {
  return {
    id: Math.random().toString(36).slice(2),
    role,
    content,
    partial: false,
    timestamp: Date.now(),
    subagentToolUseId,
  };
}

/** A text block. */
function textBlock(text: string): Message["content"][number] {
  return { type: "text", text };
}

/** A tool_use block. */
function toolUseBlock(
  id: string,
  name: string,
  input: Record<string, unknown>,
): Message["content"][number] {
  return { type: "tool_use", id, name, input };
}

/** A tool_result block. */
function toolResultBlock(
  tool_use_id: string,
  content: string,
  is_error?: boolean,
): Message["content"][number] {
  return { type: "tool_result", tool_use_id, content, is_error };
}

// ── Import the functions under test ──────────────────────────────────────────

import {
  extractSubagentActivity,
  syncSubagentActivity,
} from "./useAgentSync.js";

// ─────────────────────────────────────────────────────────────────────────────
// extractSubagentActivity
// ─────────────────────────────────────────────────────────────────────────────

describe("extractSubagentActivity", () => {
  it("returns null when no messages match the agentId", () => {
    const messages: Message[] = [
      msg([textBlock("Hello")], "other-agent-id"),
      msg([textBlock("World")]),
    ];

    expect(extractSubagentActivity(messages, "agent-123")).toBeNull();
  });

  it("returns null when matching messages have empty content", () => {
    const messages: Message[] = [
      msg([], "agent-123"),
    ];

    expect(extractSubagentActivity(messages, "agent-123")).toBeNull();
  });

  it("returns null when matching messages have only whitespace text", () => {
    const messages: Message[] = [
      msg([textBlock("   \n  ")], "agent-123"),
    ];

    expect(extractSubagentActivity(messages, "agent-123")).toBeNull();
  });

  it("extracts lastText from a text-only message", () => {
    const messages: Message[] = [
      msg([textBlock("Thinking about the problem...")], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity).not.toBeNull();
    expect(activity!.lastText).toBe("Thinking about the problem...");
    expect(activity!.currentTool).toBeNull();
    expect(activity!.toolResult).toBeNull();
  });

  it("uses LAST text block across multiple messages", () => {
    const messages: Message[] = [
      msg([textBlock("First thought")], "agent-123"),
      msg([textBlock("Second thought")], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.lastText).toBe("Second thought");
  });

  it("extracts currentTool from a tool_use block with file_path", () => {
    const messages: Message[] = [
      msg(
        [toolUseBlock("tu-1", "Read", { file_path: "/src/foo.ts" })],
        "agent-123",
      ),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.currentTool).toEqual({
      name: "Read",
      target: "/src/foo.ts",
      toolUseId: "tu-1",
    });
    expect(activity!.toolResult).toEqual({ status: "pending" });
  });

  it("marks toolResult pending when tool_use appears after tool_result", () => {
    const messages: Message[] = [
      msg([toolUseBlock("tu-1", "Bash", { command: "ls" })], "agent-123"),
      msg([toolResultBlock("tu-1", "file.ts")], "agent-123", "user"),
      msg([toolUseBlock("tu-2", "Read", { file_path: "/new.ts" })], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.currentTool!.toolUseId).toBe("tu-2");
    expect(activity!.toolResult).toEqual({ status: "pending" });
  });

  it("marks toolResult success when tool_result is most recent", () => {
    const messages: Message[] = [
      msg([toolUseBlock("tu-1", "Read", { file_path: "/foo.ts" })], "agent-123"),
      msg([toolResultBlock("tu-1", "content here")], "agent-123", "user"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.toolResult).toEqual({ status: "success" });
  });

  it("marks toolResult failed with truncated error on is_error tool_result", () => {
    const longError = "E".repeat(200);
    const messages: Message[] = [
      msg([toolUseBlock("tu-1", "Bash", { command: "fail" })], "agent-123"),
      msg([toolResultBlock("tu-1", longError, true)], "agent-123", "user"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.toolResult!.status).toBe("failed");
    // Error is capped at 100 characters
    expect(activity!.toolResult!.error).toHaveLength(100);
  });

  it("ignores messages belonging to other agents", () => {
    const messages: Message[] = [
      msg([textBlock("Other agent output")], "other-agent"),
      msg([toolUseBlock("tu-1", "Read", { file_path: "/mine.ts" })], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.currentTool!.name).toBe("Read");
    // lastText should be null since the text block belonged to a different agent
    expect(activity!.lastText).toBeNull();
  });

  it("ignores messages with no subagentToolUseId (root messages)", () => {
    const messages: Message[] = [
      msg([textBlock("Root output")], undefined),
      msg([textBlock("Subagent output")], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.lastText).toBe("Subagent output");
  });

  it("uses extractToolTarget priority: file_path > command", () => {
    const messages: Message[] = [
      msg(
        [toolUseBlock("tu-1", "Bash", { file_path: "/should-win.ts", command: "ls" })],
        "agent-123",
      ),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.currentTool!.target).toBe("/should-win.ts");
  });

  it("uses extractToolTarget fallback to first string value when no priority key", () => {
    const messages: Message[] = [
      msg(
        [toolUseBlock("tu-1", "CustomTool", { custom_field: "custom-value" })],
        "agent-123",
      ),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.currentTool!.target).toBe("custom-value");
  });

  it("truncates tool target at 60 characters", () => {
    const longPath = "/".repeat(80);
    const messages: Message[] = [
      msg(
        [toolUseBlock("tu-1", "Read", { file_path: longPath })],
        "agent-123",
      ),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    // extractToolTarget truncates to 60 chars with "..."
    expect(activity!.currentTool!.target.length).toBeLessThanOrEqual(63); // 60 + "..."
    expect(activity!.currentTool!.target.endsWith("...")).toBe(true);
  });

  it("handles message with multiple blocks: text + tool_use + tool_result", () => {
    const messages: Message[] = [
      msg(
        [
          textBlock("Starting Read"),
          toolUseBlock("tu-1", "Read", { file_path: "/foo.ts" }),
        ],
        "agent-123",
      ),
      msg(
        [toolResultBlock("tu-1", "file content")],
        "agent-123",
        "user",
      ),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    expect(activity!.lastText).toBe("Starting Read");
    expect(activity!.currentTool!.name).toBe("Read");
    expect(activity!.toolResult).toEqual({ status: "success" });
  });

  it("correctly handles tool_result appearing before tool_use in same message list order", () => {
    // Edge case: interleaved messages — result then new tool_use
    const messages: Message[] = [
      msg([toolUseBlock("tu-1", "Read", { file_path: "/a.ts" })], "agent-123"),
      msg([toolResultBlock("tu-1", "ok")], "agent-123", "user"),
      msg([toolUseBlock("tu-2", "Grep", { pattern: "foo" })], "agent-123"),
    ];

    const activity = extractSubagentActivity(messages, "agent-123");

    // tu-2 came after the tool_result, so pending
    expect(activity!.currentTool!.toolUseId).toBe("tu-2");
    expect(activity!.currentTool!.name).toBe("Grep");
    expect(activity!.toolResult).toEqual({ status: "pending" });
  });

  it("returns null when there are no messages at all", () => {
    expect(extractSubagentActivity([], "agent-123")).toBeNull();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// syncSubagentActivity
// ─────────────────────────────────────────────────────────────────────────────

describe("syncSubagentActivity", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Clear the agents map
    Object.keys(mutableAgents).forEach((k) => {
      delete mutableAgents[k];
    });
  });

  it("does nothing when there are no running agents", () => {
    mutableAgents["agent-1"] = { status: "complete" };
    mutableAgents["agent-2"] = { status: "error" };

    const snapshots = new Map<string, string>();
    syncSubagentActivity([], snapshots);

    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });

  it("does nothing when running agents have no tagged messages", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const messages: Message[] = [
      msg([textBlock("Root message")], undefined),
    ];

    const snapshots = new Map<string, string>();
    syncSubagentActivity(messages, snapshots);

    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });

  it("calls updateAgentActivity for a running agent with tagged messages", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const messages: Message[] = [
      msg([textBlock("Thinking...")], "agent-1"),
    ];

    const snapshots = new Map<string, string>();
    syncSubagentActivity(messages, snapshots);

    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();
    expect(mockUpdateAgentActivity).toHaveBeenCalledWith("agent-1", {
      lastText: "Thinking...",
      currentTool: null,
      toolResult: null,
    });
  });

  it("is idempotent: does not call updateAgentActivity again when activity unchanged", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const messages: Message[] = [
      msg([textBlock("No change")], "agent-1"),
    ];

    const snapshots = new Map<string, string>();

    // First call
    syncSubagentActivity(messages, snapshots);
    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();

    vi.clearAllMocks();

    // Second call with same messages — snapshot matches, no update
    syncSubagentActivity(messages, snapshots);
    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });

  it("calls updateAgentActivity again when activity changes", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const initialMessages: Message[] = [
      msg([textBlock("Step 1")], "agent-1"),
    ];
    const snapshots = new Map<string, string>();

    syncSubagentActivity(initialMessages, snapshots);
    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();

    vi.clearAllMocks();

    // New message added — activity changes
    const updatedMessages: Message[] = [
      ...initialMessages,
      msg([toolUseBlock("tu-1", "Read", { file_path: "/new.ts" })], "agent-1"),
    ];

    syncSubagentActivity(updatedMessages, snapshots);
    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();
    expect(mockUpdateAgentActivity).toHaveBeenCalledWith(
      "agent-1",
      expect.objectContaining({
        currentTool: expect.objectContaining({ name: "Read" }),
      }),
    );
  });

  it("skips completed agents even if they have tagged messages", () => {
    mutableAgents["agent-complete"] = { status: "complete" };
    mutableAgents["agent-error"] = { status: "error" };
    mutableAgents["agent-timeout"] = { status: "timeout" };

    const messages: Message[] = [
      msg([textBlock("output")], "agent-complete"),
      msg([textBlock("output")], "agent-error"),
      msg([textBlock("output")], "agent-timeout"),
    ];

    const snapshots = new Map<string, string>();
    syncSubagentActivity(messages, snapshots);

    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });

  it("processes multiple running agents independently", () => {
    mutableAgents["agent-a"] = { status: "running" };
    mutableAgents["agent-b"] = { status: "running" };

    const messages: Message[] = [
      msg([textBlock("A output")], "agent-a"),
      msg([toolUseBlock("tu-b1", "Grep", { pattern: "foo" })], "agent-b"),
    ];

    const snapshots = new Map<string, string>();
    syncSubagentActivity(messages, snapshots);

    expect(mockUpdateAgentActivity).toHaveBeenCalledTimes(2);

    const callArgs = mockUpdateAgentActivity.mock.calls.map(
      (c) => [c[0], c[1]] as [string, unknown],
    );
    const agentACall = callArgs.find(([id]) => id === "agent-a");
    const agentBCall = callArgs.find(([id]) => id === "agent-b");

    expect(agentACall).toBeDefined();
    expect(agentBCall).toBeDefined();

    expect((agentACall![1] as { lastText: string }).lastText).toBe("A output");
    expect(
      (agentBCall![1] as { currentTool: { name: string } }).currentTool.name,
    ).toBe("Grep");
  });

  it("resets snapshot cache works correctly per-agent", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const messages: Message[] = [
      msg([textBlock("Hello")], "agent-1"),
    ];
    const snapshots = new Map<string, string>();

    // Populate snapshot
    syncSubagentActivity(messages, snapshots);
    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();

    vi.clearAllMocks();

    // Simulate provider switch: clear snapshots
    snapshots.clear();

    // Same messages but cleared snapshot → should re-emit
    syncSubagentActivity(messages, snapshots);
    expect(mockUpdateAgentActivity).toHaveBeenCalledOnce();
  });

  it("handles an agent with queued status (non-running) gracefully", () => {
    mutableAgents["agent-queued"] = { status: "queued" };

    const messages: Message[] = [
      msg([textBlock("queued output")], "agent-queued"),
    ];

    const snapshots = new Map<string, string>();
    syncSubagentActivity(messages, snapshots);

    // "queued" is not "running", so no update
    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });

  it("handles empty messages array with running agents gracefully", () => {
    mutableAgents["agent-1"] = { status: "running" };

    const snapshots = new Map<string, string>();
    syncSubagentActivity([], snapshots);

    expect(mockUpdateAgentActivity).not.toHaveBeenCalled();
  });
});
