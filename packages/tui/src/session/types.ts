/**
 * SessionManager type definitions
 *
 * Defines types for the SessionManager class that coordinates async message flow
 * between user input and Claude Agent SDK query() AsyncIterable.
 */

import type { ClassifiedError } from "../types/events.js";

/**
 * SessionState enum - lifecycle states for session connection
 *
 * State transitions:
 * UNINITIALIZED -> CONNECTING -> READY -> STREAMING <-> READY -> ERROR/DEAD
 *
 * @example
 * ```ts
 * const state = SessionState.READY;
 * if (state === SessionState.STREAMING) {
 *   // Handle streaming response
 * }
 * ```
 */
export enum SessionState {
  /** Initial state - no connection attempt made */
  UNINITIALIZED = "UNINITIALIZED",
  /** Connection attempt in progress */
  CONNECTING = "CONNECTING",
  /** Connected and idle, ready to accept messages */
  READY = "READY",
  /** Currently streaming a response */
  STREAMING = "STREAMING",
  /** Error state - may be recoverable with reconnect */
  ERROR = "ERROR",
  /** Terminal state - session cannot be recovered */
  DEAD = "DEAD",
}

/**
 * Configuration options for SessionManager
 *
 * @example
 * ```ts
 * const config: SessionManagerConfig = {
 *   maxQueueSize: 10,
 *   reconnectDelayMs: 1000,
 *   maxReconnectAttempts: 3
 * };
 * ```
 */
export interface SessionManagerConfig {
  /** Maximum number of messages to queue before rejecting new messages */
  maxQueueSize: number;

  /** Delay in milliseconds before attempting reconnection after failure */
  reconnectDelayMs: number;

  /** Maximum number of reconnection attempts before marking session DEAD */
  maxReconnectAttempts: number;

  /**
   * Backend to use for the session.
   * - 'claude': Standard Claude Code CLI (default)
   * - 'gemini': Local Gemini Adapter
   */
  backend?: 'claude' | 'gemini';
}

/**
 * QueuedMessage - message awaiting delivery to Claude
 *
 * Each queued message has a unique ID and Promise-based resolution mechanism
 * to coordinate async message flow.
 *
 * @example
 * ```ts
 * const message: QueuedMessage = {
 *   id: nanoid(),
 *   text: "What is 2+2?",
 *   enqueuedAt: Date.now(),
 *   resolve: (success: boolean) => console.log("Delivered:", success),
 *   reject: (error: Error) => console.error("Failed:", error)
 * };
 * ```
 */
export interface QueuedMessage {
  /** Unique message identifier (nanoid) */
  id: string;

  /** Message text to send to Claude */
  text: string;

  /** Unix timestamp (ms) when message was enqueued */
  enqueuedAt: number;

  /** Resolve callback - called when message delivery succeeds or fails */
  resolve: (success: boolean) => void;

  /** Reject callback - called when message delivery encounters error */
  reject: (error: Error) => void;
}

/**
 * SessionManagerEvents - callbacks for state synchronization with Zustand store
 *
 * @example
 * ```ts
 * const events: SessionManagerEvents = {
 *   onStateChange: (state) => store.getState().setSessionState(state),
 *   onError: (error) => store.getState().addToast(error.message, "error"),
 *   onSessionId: (id) => store.getState().updateSession({ sessionId: id })
 * };
 * ```
 */
export interface SessionManagerEvents {
  /** Called when session state changes */
  onStateChange: (state: SessionState) => void;

  /** Called when an error occurs (including classified error details) */
  onError: (error: ClassifiedError) => void;

  /** Called when session ID is received from SDK */
  onSessionId: (id: string) => void;

  /** Called when streaming completes (success or error) - for queue drain signaling */
  onStreamingComplete: () => void;
}

/**
 * MessageCoordinator - coordination mechanism for async message/response flow
 *
 * Based on test-async-iterable.ts pattern. Coordinates timing between:
 * 1. Generator yielding user message
 * 2. Event consumer receiving assistant response
 * 3. Caller awaiting delivery confirmation
 *
 * @example
 * ```ts
 * const coordinator: MessageCoordinator = {
 *   text: "Hello Claude",
 *   yieldedAt: 0,
 *   responsePromise: new Promise((resolve, reject) => {
 *     coordinator.resolveResponse = resolve;
 *     coordinator.rejectResponse = reject;
 *   }),
 *   resolveResponse: () => {},
 *   rejectResponse: (error) => console.error(error)
 * };
 *
 * // Generator yields message
 * coordinator.yieldedAt = performance.now();
 * yield createUserMessage(coordinator.text);
 *
 * // Wait for response
 * await coordinator.responsePromise;
 * ```
 */
export interface MessageCoordinator {
  /** Message text to send */
  text: string;

  /** Timestamp (performance.now()) when generator yielded this message */
  yieldedAt: number;

  /** Promise that resolves when response is complete */
  responsePromise: Promise<void>;

  /** Resolve callback - called by event consumer when response complete */
  resolveResponse: () => void;

  /** Reject callback - called by event consumer on error */
  rejectResponse: (error: Error) => void;

  /** Reference to the queued message to resolve its promise on completion */
  queuedMessage?: QueuedMessage;
}
