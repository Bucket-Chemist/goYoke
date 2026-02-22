/**
 * useAgentSync hook
 * Syncs Task() tool_use blocks from providerMessages into the agents Zustand store.
 * Called unconditionally in Layout (pure side-effects, no render output).
 */

import { useEffect, useRef } from "react";
import { useStore } from "../store/index.js";
import { activityFromTaskBlocks } from "../utils/agentActivity.js";
import type { Message } from "../store/types.js";

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
      if (block.type === "tool_use" && block.name === "Task") {
        const toolUseId = block.id;
        const input = block.input as Record<string, unknown>;
        taskInputs.set(toolUseId, input);

        if (registeredIds.has(toolUseId)) continue;
        registeredIds.add(toolUseId);

        // Read state fresh before each mutation
        const stateBeforeRoot = useStore.getState();

        // Ensure router-root exists (first Task() block seen)
        if (!stateBeforeRoot.rootAgentId) {
          stateBeforeRoot.addAgent({
            id: ROUTER_ROOT_ID,
            parentId: null,
            model: "opus",
            tier: "opus",
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

/**
 * Syncs Task() invocations from the active provider's messages into the
 * agents Zustand store. Call unconditionally in Layout alongside useTeamsPoller.
 */
export function useAgentSync(): void {
  const activeProvider = useStore((s) => s.activeProvider);
  const registeredIds = useRef(new Set<string>());
  const completedIds = useRef(new Set<string>());
  const lastProvider = useRef(activeProvider);

  useEffect(() => {
    // Reset tracking on provider switch
    if (lastProvider.current !== activeProvider) {
      registeredIds.current.clear();
      completedIds.current.clear();
      lastProvider.current = activeProvider;
    }

    // Run initial sync for current messages
    const messages =
      useStore.getState().providerMessages[activeProvider] ?? [];
    syncTaskAgents(messages, registeredIds.current, completedIds.current);

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
      }
    });

    return unsub;
  }, [activeProvider]);
}
