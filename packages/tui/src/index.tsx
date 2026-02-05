import React from "react";
import { render } from "ink";
import { App } from "./App.js";
import { parseCLI } from "./cli.js";
import { listSessions } from "./hooks/useSession.js";
import { ListSessions } from "./commands/list.jsx";
import { setupSignalHandlers } from "./lifecycle/shutdown.js";
import { getRestartManager } from "./lifecycle/restart.js";
import { validateSpawnEnvironment, formatValidationResult } from "./spawn/validation.js";

/**
 * GOfortress TUI Entry Point
 * Renders the main App component in the terminal with lifecycle management
 */

async function main() {
  // Setup graceful shutdown handlers first
  setupSignalHandlers();

  // Validate spawn environment before starting
  // Errors are logged but don't block startup - allows running with spawn disabled
  try {
    const validationResult = await validateSpawnEnvironment();
    if (!validationResult.ok || validationResult.warnings.length > 0) {
      console.error(formatValidationResult(validationResult));
      if (!validationResult.ok) {
        console.error("\n⚠️  Some spawn features may be unavailable\n");
      }
    }
  } catch (err) {
    console.error("Environment validation error:", err instanceof Error ? err.message : err);
    console.error("⚠️  Continuing with warnings - some features may be unavailable\n");
  }

  const options = parseCLI();
  const restartManager = getRestartManager();

  // Handle --list flag
  if (options.list) {
    const sessions = await listSessions();
    const { waitUntilExit } = render(<ListSessions sessions={sessions} />);
    await waitUntilExit();
    process.exit(0);
  }

  // Handle --legacy flag (fallback to Go TUI)
  if (options.legacy) {
    console.error("Legacy mode: Falling back to Go TUI");
    console.error("Run: claude-tui (Go binary)");
    process.exit(1);
  }

  // Record successful start for backoff reset detection
  restartManager.recordSuccessfulStart();

  // Normal mode: render main app
  render(<App sessionId={options.session} verbose={options.verbose} />);
}

/**
 * Restart wrapper with exponential backoff
 * Catches crashes and retries with increasing delays
 */
async function runWithRestart() {
  const restartManager = getRestartManager();

  while (true) {
    try {
      await main();
      // Normal exit - check if we should reset backoff
      restartManager.checkAndResetIfSuccessful();
      break;
    } catch (error) {
      console.error("\n[Restart] App crashed:", error);

      // Check if we should restart
      if (!restartManager.shouldRestart()) {
        console.error("[Restart] Max restart attempts reached");
        console.error("[Restart] Giving up");
        process.exit(1);
      }

      // Calculate delay and log restart info
      const delay = restartManager.getDelay();
      const state = restartManager.getState();
      console.error(
        `[Restart] Attempt ${state.attempts + 1}/${state.maxAttempts} - waiting ${delay}ms...`
      );

      // Record this attempt
      restartManager.recordAttempt();

      // Wait before restarting
      await new Promise((resolve) => setTimeout(resolve, delay));

      console.error("[Restart] Restarting app...\n");
    }
  }
}

runWithRestart().catch((error) => {
  console.error("Fatal error in restart manager:", error);
  process.exit(1);
});
