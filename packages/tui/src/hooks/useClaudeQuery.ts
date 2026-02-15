/**
 * useClaudeQuery hook - Thin React adapter over SessionManager
 *
 * Delegates all session/process lifecycle to SessionManager singleton.
 * This hook manages only React-specific concerns: local state, store subscriptions,
 * and callback registration.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useStore } from "../store/index.js";
import { getSessionManager } from "../session/index.js";
import { SessionState } from "../session/types.js";
import type { ClassifiedError } from "../types/events.js";
import { logger } from "../utils/logger.js";

/**
 * Hook options
 */
interface UseClaudeQueryOptions {
  onStreamingComplete?: () => void;
}

/**
 * Hook return type
 */
interface UseClaudeQueryReturn {
  sendMessage: (message: string) => Promise<void>;
  setModel: (modelId: string) => Promise<boolean>;
  isStreaming: boolean;
  error: ClassifiedError | null;
}

/**
 * Main hook for Claude query integration
 *
 * Thin adapter over SessionManager singleton. SessionManager owns:
 * - query() call and persistent AsyncIterable process
 * - All SDK event handlers (system, assistant, user, result, status)
 * - streamInput for built-in tools (AskUserQuestion, RequestInput, ConfirmAction)
 * - canUseTool permission flow (including plan mode)
 * - State machine (UNINITIALIZED → CONNECTING → READY ⇄ STREAMING → ERROR → DEAD)
 * - Error classification and reconnection
 *
 * This hook owns:
 * - Local React state (isStreaming, error) for hook return API
 * - User message display (addMessage before enqueue)
 * - SessionManager event registration and cleanup
 * - Interrupt function registration in store
 */
export function useClaudeQuery(
  options?: UseClaudeQueryOptions
): UseClaudeQueryReturn {
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<ClassifiedError | null>(null);

  // Store the callback ref to avoid stale closures
  const onStreamingCompleteRef = useRef(options?.onStreamingComplete);

  // Stable reference to SessionManager singleton
  const managerRef = useRef(getSessionManager());

  // Track whether we've already registered events to avoid re-registration
  const eventsRegistered = useRef(false);

  // Keep callback ref in sync with latest version
  useEffect(() => {
    onStreamingCompleteRef.current = options?.onStreamingComplete;
  }, [options?.onStreamingComplete]);

  // Register SessionManager events on mount
  useEffect(() => {
    if (eventsRegistered.current) return;
    eventsRegistered.current = true;

    const manager = managerRef.current;
    manager.setEvents({
      onStateChange: (state: SessionState) => {
        // Register interrupt function when entering STREAMING
        if (state === SessionState.STREAMING) {
          useStore.getState().setInterruptQuery(() => manager.interrupt());
        }
      },
      onError: (classifiedError: ClassifiedError) => {
        setError(classifiedError);

        // Add error message to chat
        useStore.getState().addMessage({
          role: "assistant",
          content: [
            {
              type: "text",
              text: `Error: ${classifiedError.message}`,
            },
          ],
          partial: false,
        });

        // Stop streaming on error
        setIsStreaming(false);
        useStore.getState().setStreaming(false);
        useStore.getState().setInterruptQuery(null);

        // Notify streaming complete even on error (for queue drain)
        onStreamingCompleteRef.current?.();
      },
      onSessionId: (_id: string) => {
        // SessionManager already updates store via useStore.getState().updateSession()
        // No additional action needed in hook
      },
      onStreamingComplete: () => {
        setIsStreaming(false);
        useStore.getState().setStreaming(false);
        useStore.getState().setInterruptQuery(null);

        // Notify caller that streaming is complete (for queue drain)
        onStreamingCompleteRef.current?.();
      },
    });
  }, []);

  // Store actions for sendMessage
  const addMessage = useStore((state) => state.addMessage);
  const setStreamingState = useStore((state) => state.setStreaming);

  /**
   * Send message to Claude via SessionManager
   *
   * Hook responsibilities:
   * 1. Guard against concurrent queries
   * 2. Add user message to store (immediate display)
   * 3. Set streaming state
   * 4. Delegate to SessionManager.enqueue()
   */
  const sendMessage = useCallback(
    async (message: string): Promise<void> => {
      const manager = managerRef.current;

      // GUARD: Prevent concurrent queries
      if (isStreaming) {
        void logger.warn("Query already in progress, ignoring duplicate call");
        return;
      }

      try {
        // Reset error state
        setError(null);

        // Add user message to store (user sees it immediately)
        addMessage({
          role: "user",
          content: [
            {
              type: "text",
              text: message,
            },
          ],
          partial: false,
        });

        // Start streaming - update both local state and store
        setIsStreaming(true);
        setStreamingState(true);

        // Set interruptQuery immediately so Esc works from the start.
        // For the first message, SessionManager skips the STREAMING state
        // transition (stays CONNECTING→READY), so the onStateChange handler
        // never fires — without this, interruptQuery stays null.
        useStore.getState().setInterruptQuery(() => managerRef.current.interrupt());

        // Delegate to SessionManager (auto-connects if needed)
        const success = await manager.enqueue(message);

        if (!success) {
          // Enqueue failed (session dead or queue full)
          setIsStreaming(false);
          setStreamingState(false);
          setError({
            type: "unknown",
            message:
              "Failed to send message (session unavailable or queue full)",
            retryable: false,
          });
        }
      } catch (err) {
        // Unexpected error during enqueue
        setIsStreaming(false);
        setStreamingState(false);

        const errorMessage = err instanceof Error ? err.message : String(err);
        setError({
          type: "unknown",
          message: errorMessage || "An unexpected error occurred",
          retryable: false,
        });

        void logger.error("sendMessage error", {
          error: errorMessage,
        });
      }
    },
    [isStreaming, addMessage, setStreamingState]
  );

  /**
   * Switch the model for the active query
   * Returns true on success, false if no active query or on error
   */
  const setModel = useCallback(async (modelId: string): Promise<boolean> => {
    return managerRef.current.setModel(modelId);
  }, []);

  return {
    sendMessage,
    setModel,
    isStreaming,
    error,
  };
}
