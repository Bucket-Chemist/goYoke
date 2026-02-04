/**
 * Graceful shutdown handling
 * Coordinates with session persistence and Go hooks on app exit
 */

import { useStore } from "../store/index.js";
import { saveSession } from "../hooks/useSession.js";

type ShutdownHandler = () => Promise<void>;

/**
 * Registered shutdown handlers
 * Executed in order during graceful shutdown
 */
const handlers: ShutdownHandler[] = [];

/**
 * Track if shutdown is in progress to prevent duplicate execution
 */
let isShuttingDown = false;

/**
 * Register a shutdown handler
 * Handlers execute in registration order during shutdown
 */
export function onShutdown(handler: ShutdownHandler): void {
  handlers.push(handler);
}

/**
 * Initiate graceful shutdown sequence
 * 1. Save session state
 * 2. Execute registered handlers
 * 3. Allow time for Go hooks (gogent-archive)
 * 4. Exit process
 */
export async function initiateShutdown(signal: string): Promise<void> {
  // Prevent duplicate shutdown execution
  if (isShuttingDown) {
    return;
  }
  isShuttingDown = true;

  console.log(`\n[Shutdown] Received ${signal}, shutting down gracefully...`);

  try {
    // Step 1: Save session state
    const state = useStore.getState();
    if (state.sessionId) {
      console.log(`[Shutdown] Saving session ${state.sessionId}...`);
      await saveSession({
        id: state.sessionId,
        created_at: new Date().toISOString(), // Will be preserved if exists
        last_used: new Date().toISOString(),
        cost: state.totalCost,
        tool_calls: 0, // Not tracking tool calls in TUI yet
      });
      console.log("[Shutdown] Session saved");
    }

    // Step 2: Run all registered handlers
    console.log(`[Shutdown] Running ${handlers.length} shutdown handlers...`);
    for (let i = 0; i < handlers.length; i++) {
      try {
        await handlers[i]();
      } catch (error) {
        console.error(`[Shutdown] Handler ${i} error:`, error);
        // Continue with other handlers even if one fails
      }
    }
    console.log("[Shutdown] Handlers complete");

    // Step 3: Allow time for Go hooks (gogent-archive runs on SessionEnd)
    // The hook needs time to write handoff and metrics
    console.log("[Shutdown] Waiting for Go hooks...");
    await new Promise((resolve) => setTimeout(resolve, 500));

    console.log("[Shutdown] Graceful shutdown complete");
  } catch (error) {
    console.error("[Shutdown] Error during shutdown:", error);
    // Continue to exit even if shutdown has errors
  } finally {
    process.exit(0);
  }
}

/**
 * Setup signal handlers for graceful shutdown
 * Handles SIGINT (Ctrl+C), SIGTERM, and uncaught exceptions
 */
export function setupSignalHandlers(): void {
  // Handle SIGINT (Ctrl+C)
  process.on("SIGINT", () => {
    void initiateShutdown("SIGINT");
  });

  // Handle SIGTERM (kill command)
  process.on("SIGTERM", () => {
    void initiateShutdown("SIGTERM");
  });

  // Handle uncaught exceptions - log and shutdown
  process.on("uncaughtException", (error) => {
    console.error("[Fatal] Uncaught exception:", error);
    void initiateShutdown("uncaughtException");
  });

  // Handle unhandled promise rejections - log but don't exit
  // This follows Node.js best practices - log and continue
  process.on("unhandledRejection", (reason, promise) => {
    console.error("[Warning] Unhandled promise rejection:", reason);
    console.error("Promise:", promise);
    // Don't exit - just log for debugging
  });

  // Handle process warnings (e.g., memory leaks)
  process.on("warning", (warning) => {
    console.warn("[Warning]", warning.name, warning.message);
  });
}

/**
 * Register cleanup for child processes
 * Ensures no orphaned processes remain after shutdown
 */
export function registerChildProcessCleanup(
  cleanup: () => Promise<void>
): void {
  onShutdown(cleanup);
}

/**
 * Get shutdown state for testing
 */
export function isShutdownInProgress(): boolean {
  return isShuttingDown;
}

/**
 * Reset shutdown state
 * Primarily for testing - allows multiple test runs
 */
export function resetShutdownState(): void {
  isShuttingDown = false;
  handlers.length = 0; // Clear handlers array
}
