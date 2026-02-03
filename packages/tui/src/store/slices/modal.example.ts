/**
 * Example usage of modal queue with Promise-based resolution
 * This demonstrates how MCP tools will interact with the modal system
 */

import { useStore } from "../index.js";
import type {
  AskPayload,
  ConfirmPayload,
  InputPayload,
  SelectPayload,
} from "./modal.js";

/**
 * Example 1: Ask modal (choose from options)
 */
export async function exampleAskModal(): Promise<void> {
  const { enqueue } = useStore.getState();

  try {
    const response = await enqueue<AskPayload>({
      type: "ask",
      payload: {
        message: "What color scheme would you like?",
        options: ["Dark", "Light", "Auto"],
        defaultValue: "Auto",
      },
      timeout: 30000, // 30 second timeout
    });

    if (response.type === "ask") {
      console.log(`User selected: ${response.value}`);
    }
  } catch (error) {
    console.error("Modal was cancelled or timed out:", error);
  }
}

/**
 * Example 2: Confirm modal (destructive action)
 */
export async function exampleConfirmModal(): Promise<void> {
  const { enqueue } = useStore.getState();

  try {
    const response = await enqueue<ConfirmPayload>({
      type: "confirm",
      payload: {
        action: "Delete all conversation history",
        destructive: true,
      },
    });

    if (response.type === "confirm" && response.confirmed && !response.cancelled) {
      console.log("User confirmed deletion");
      // Proceed with destructive action
    } else {
      console.log("User cancelled deletion");
    }
  } catch (error) {
    console.error("Modal was cancelled:", error);
  }
}

/**
 * Example 3: Input modal (free text entry)
 */
export async function exampleInputModal(): Promise<void> {
  const { enqueue } = useStore.getState();

  try {
    const response = await enqueue<InputPayload>({
      type: "input",
      payload: {
        prompt: "Enter your API key",
        placeholder: "sk-...",
      },
      timeout: 60000, // 1 minute timeout
    });

    if (response.type === "input") {
      console.log(`User entered: ${response.value}`);
    }
  } catch (error) {
    console.error("Modal was cancelled or timed out:", error);
  }
}

/**
 * Example 4: Select modal (structured options)
 */
export async function exampleSelectModal(): Promise<void> {
  const { enqueue } = useStore.getState();

  try {
    const response = await enqueue<SelectPayload>({
      type: "select",
      payload: {
        message: "Choose a model",
        options: [
          { label: "Claude Opus", value: "claude-opus-4" },
          { label: "Claude Sonnet", value: "claude-sonnet-4" },
          { label: "Claude Haiku", value: "claude-haiku-4" },
        ],
      },
    });

    if (response.type === "select") {
      console.log(`User selected: ${response.selected} (index ${response.index})`);
    }
  } catch (error) {
    console.error("Modal was cancelled or timed out:", error);
  }
}

/**
 * Example 5: Multiple modals queued (FIFO)
 */
export async function exampleQueuedModals(): Promise<void> {
  const { enqueue } = useStore.getState();

  // All three will queue and display one at a time
  const [color, confirm, name] = await Promise.all([
    enqueue<AskPayload>({
      type: "ask",
      payload: { message: "Pick a color", options: ["Red", "Blue", "Green"] },
    }),
    enqueue<ConfirmPayload>({
      type: "confirm",
      payload: { action: "Save settings" },
    }),
    enqueue<InputPayload>({
      type: "input",
      payload: { prompt: "Enter your name" },
    }),
  ]);

  console.log("All modals completed:", { color, confirm, name });
}

/**
 * Example 6: Cancellable modal
 */
export async function exampleCancellableModal(): Promise<void> {
  const { enqueue, cancel, modalQueue } = useStore.getState();

  const promise = enqueue<AskPayload>({
    type: "ask",
    payload: { message: "This can be cancelled", options: ["Option 1", "Option 2"] },
  });

  // Get the modal ID for cancellation
  const lastModal = modalQueue[modalQueue.length - 1];
  if (!lastModal) {
    throw new Error("Modal not found in queue");
  }
  const modalId = lastModal.id;

  // Cancel after 5 seconds
  setTimeout(() => {
    cancel(modalId);
  }, 5000);

  try {
    const response = await promise;
    console.log("User responded before cancellation:", response);
  } catch (error) {
    console.error("Modal was cancelled:", error);
  }
}

/**
 * Example 7: MCP tool integration pattern
 * This is how MCP tools will use the modal queue
 */
export async function mcpToolExample(): Promise<string> {
  const { enqueue } = useStore.getState();

  // MCP tool asks user for confirmation
  const response = await enqueue<ConfirmPayload>({
    type: "confirm",
    payload: {
      action: "Execute git push --force",
      destructive: true,
    },
    timeout: 30000,
  });

  if (response.type === "confirm" && response.confirmed && !response.cancelled) {
    // User confirmed - proceed with action
    return "Action confirmed by user";
  }

  throw new Error("Action cancelled by user");
}
