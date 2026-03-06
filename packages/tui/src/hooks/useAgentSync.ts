/**
 * useAgentSync hook
 * Syncs Agent tool_use blocks from providerMessages into the agents Zustand store.
 * Also extracts live intermediate activity from subagent-tagged messages.
 * Called unconditionally in Layout (pure side-effects, no render output).
 */

import { useEffect, useRef } from "react";
import { useStore } from "../store/index.js";
import { activityFromTaskBlocks, extractToolTarget } from "../utils/agentActivity.js";
import type { Message, AgentActivity, ToolUseContent, ToolResultContent } from "../store/types.js";

const ROUTER_ROOT_ID = "router-root";

/**
 * Tier mapping from model string to tier bucket.
 */
const TIER_MAP: Record<string, "haiku" | "sonnet" | "opus"> = {
  haiku: "haiku",
  sonnet: "sonnet",
  opus: "opus",
};

/**
 * Scan messages for Task() tool_use and tool_result blocks and sync them
 * into the agents store. Idempotent — already-registered IDs are skipped.
 */
function syncTaskAgents(
  messages: Message[],
  registeredIds: Set<string>,
  completedIds: Set<string>,
): void {
  // Build a map of tool_use_id -> input for correlating tool_results
  const taskInputs = new Map<string, Record<string, unknown>>();

  for (const msg of messages) {
    for (const block of msg.content) {
      if (block.type === "tool_use" && (block.name === "Agent" || block.name === "Task")) {
        const toolUseId = block.id;
        const input = block.input as Record<string, unknown>;
        taskInputs.set(toolUseId, input);

        if (registeredIds.has(toolUseId)) continue;
        registeredIds.add(toolUseId);

        // Read state fresh before each mutation
        const stateBeforeRoot = useStore.getState();

        // Ensure router-root exists — defensive fallback if eager SessionManager
        // creation didn't fire before the first Task() block arrived.
        if (!stateBeforeRoot.rootAgentId) {
          console.warn(
            "[useAgentSync] Root agent missing at first Task() — creating fallback",
          );
          const activeModel =
            stateBeforeRoot.getActiveModel() ?? "opus";
          const activeTier: "haiku" | "sonnet" | "opus" =
            TIER_MAP[
              Object.keys(TIER_MAP).find((k) => activeModel.includes(k)) ?? ""
            ] ?? "opus";
          stateBeforeRoot.addAgent({
            id: ROUTER_ROOT_ID,
            parentId: null,
            model: activeModel,
            tier: activeTier,
            status: "running",
            description: "Router",
            agentType: "router",
            spawnMethod: "task",
          });
        }

        // Extract metadata from input
        const description = (
          typeof input["description"] === "string"
            ? input["description"]
            : "Task agent"
        ) as string;

        const model = (
          typeof input["model"] === "string" ? input["model"] : "sonnet"
        ) as string;

        const prompt =
          typeof input["prompt"] === "string" ? input["prompt"] : "";
        const agentMatch = prompt.match(/^AGENT:\s*(\S+)/m);
        const agentType =
          agentMatch?.[1] ??
          description.split(" ")[0]?.toLowerCase() ??
          "unknown";

        // Re-read state after potential root creation
        const stateAfterRoot = useStore.getState();

        stateAfterRoot.addAgent({
          id: toolUseId,
          parentId: stateAfterRoot.rootAgentId ?? ROUTER_ROOT_ID,
          model,
          tier: TIER_MAP[model] ?? "sonnet",
          status: "running",
          description,
          agentType,
          spawnMethod: "task",
          activity: activityFromTaskBlocks(input, null, false),
        });
      }

      if (block.type === "tool_result") {
        const toolUseId = block.tool_use_id;

        // Only process tool_results for Task() blocks we know about
        if (!taskInputs.has(toolUseId)) continue;
        if (completedIds.has(toolUseId)) continue;
        if (!registeredIds.has(toolUseId)) continue;

        // Check agent exists and is still running
        const currentState = useStore.getState();
        const agent = currentState.agents[toolUseId];
        if (!agent || agent.status !== "running") continue;

        completedIds.add(toolUseId);

        const isError = block.is_error === true;
        const resultContent =
          typeof block.content === "string" ? block.content : "";
        const input = taskInputs.get(toolUseId) ?? {};

        currentState.updateAgent(toolUseId, {
          status: isError ? "error" : "complete",
          endTime: Date.now(),
          error: isError ? resultContent.slice(0, 200) : undefined,
        });

        currentState.updateAgentActivity(
          toolUseId,
          activityFromTaskBlocks(input, resultContent, isError),
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Live subagent activity extraction
// ---------------------------------------------------------------------------

/**
 * Extract live AgentActivity from the subset of provider messages that carry
 * `subagentToolUseId === agentId`.  These are intermediate messages produced
 * by the SDK as the spawned agent executes — they contain the same
 * ContentBlock types (text, tool_use, tool_result) as root messages.
 *
 * Processing order: messages are scanned in insertion order so later events
 * overwrite earlier ones.  The very last tool_use seen becomes currentTool;
 * the very last tool_result determines toolResult status; whether the last
 * tool_use appeared after the last tool_result decides pending vs settled.
 *
 * Returns null when no subagent messages exist for this id.
 */
export function extractSubagentActivity(
  messages: Message[],
  agentId: string,
): AgentActivity | null {
  let lastText: string | null = null;
  let lastToolUse: ToolUseContent | null = null;
  let lastToolResult: ToolResultContent | null = null;
  let lastToolUseMsgIndex = -1;
  let lastToolResultMsgIndex = -1;
  let msgIndex = 0;

  for (const msg of messages) {
    if (msg.subagentToolUseId !== agentId) {
      msgIndex++;
      continue;
    }

    for (const block of msg.content) {
      if (block.type === "text" && block.text.trim().length > 0) {
        lastText = block.text;
      } else if (block.type === "tool_use") {
        lastToolUse = block;
        lastToolUseMsgIndex = msgIndex;
      } else if (block.type === "tool_result") {
        lastToolResult = block;
        lastToolResultMsgIndex = msgIndex;
      }
    }

    msgIndex++;
  }

  // Nothing useful found
  if (lastText === null && lastToolUse === null && lastToolResult === null) {
    return null;
  }

  // Build currentTool from the last observed tool_use
  const currentTool: AgentActivity["currentTool"] =
    lastToolUse !== null
      ? {
          name: lastToolUse.name,
          target: extractToolTarget(lastToolUse.name, lastToolUse.input),
          toolUseId: lastToolUse.id,
        }
      : null;

  // Determine toolResult status:
  // - Last tool_use came AFTER the last tool_result → still pending
  // - tool_result present and it is the most recent → reflect its outcome
  // - tool_use present but no tool_result at all → pending
  let toolResult: AgentActivity["toolResult"] = null;

  if (lastToolUse !== null && lastToolUseMsgIndex > lastToolResultMsgIndex) {
    toolResult = { status: "pending" };
  } else if (lastToolResult !== null) {
    if (lastToolResult.is_error === true) {
      toolResult = {
        status: "failed",
        error: (typeof lastToolResult.content === "string"
          ? lastToolResult.content
          : ""
        ).slice(0, 100),
      };
    } else {
      toolResult = { status: "success" };
    }
  } else if (lastToolUse !== null) {
    toolResult = { status: "pending" };
  }

  return { lastText, currentTool, toolResult };
}

/**
 * For every running agent that has subagent-tagged messages, extract the
 * latest AgentActivity and call updateAgentActivity when it has changed.
 *
 * Idempotency: a JSON snapshot of the last emitted activity per agentId is
 * kept in `lastActivitySnapshots`. updateAgentActivity is only called when
 * the snapshot differs from the new value.
 */
export function syncSubagentActivity(
  messages: Message[],
  lastActivitySnapshots: Map<string, string>,
): void {
  const state = useStore.getState();

  for (const [agentId, agent] of Object.entries(state.agents)) {
    // Only process running agents — completed/error agents don't need updates
    if (agent.status !== "running") continue;

    const activity = extractSubagentActivity(messages, agentId);
    if (activity === null) continue;

    const snapshot = JSON.stringify(activity);
    if (lastActivitySnapshots.get(agentId) === snapshot) continue;

    lastActivitySnapshots.set(agentId, snapshot);
    state.updateAgentActivity(agentId, activity);
  }
}

/**
 * Syncs Task() invocations from the active provider's messages into the
 * agents Zustand store. Call unconditionally in Layout alongside useTeamsPoller.
 */
export function useAgentSync(): void {
  const activeProvider = useStore((s) => s.activeProvider);
  const registeredIds = useRef(new Set<string>());
  const completedIds = useRef(new Set<string>());
  const lastProvider = useRef(activeProvider);
  // Idempotency cache: agentId → JSON snapshot of last emitted AgentActivity
  const lastActivitySnapshots = useRef(new Map<string, string>());

  useEffect(() => {
    // Reset tracking on provider switch
    if (lastProvider.current !== activeProvider) {
      registeredIds.current.clear();
      completedIds.current.clear();
      lastActivitySnapshots.current.clear();
      lastProvider.current = activeProvider;
    }

    // Run initial sync for current messages
    const messages =
      useStore.getState().providerMessages[activeProvider] ?? [];
    syncTaskAgents(messages, registeredIds.current, completedIds.current);
    syncSubagentActivity(messages, lastActivitySnapshots.current);

    // Subscribe to any store change and re-sync if messages changed.
    // useStore does NOT use subscribeWithSelector, so we use the 1-arg form
    // and check the messages reference manually.
    let lastMessages = messages;

    const unsub = useStore.subscribe((state) => {
      const msgs = state.providerMessages[state.activeProvider] ?? [];
      if (msgs !== lastMessages) {
        lastMessages = msgs;
        syncTaskAgents(
          msgs,
          registeredIds.current,
          completedIds.current,
        );
        syncSubagentActivity(msgs, lastActivitySnapshots.current);
      }
    });

    return unsub;
  }, [activeProvider]);
}
