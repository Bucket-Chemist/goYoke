/**
 * Session management module for persistent CLI process reuse.
 *
 * @module session
 *
 * @example
 * ```ts
 * import { getSessionManager, SessionState } from "./session/index.js";
 *
 * const manager = getSessionManager();
 * await manager.connect();
 * const success = await manager.enqueue("Hello!");
 * ```
 */

export { getSessionManager, resetSessionManager } from "./SessionManager.js";
export {
  SessionState,
  type SessionManagerConfig,
  type QueuedMessage,
  type MessageCoordinator,
  type SessionManagerEvents,
} from "./types.js";
