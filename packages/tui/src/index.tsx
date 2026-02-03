#!/usr/bin/env node
import React from "react";
import { render } from "ink";
import { App } from "./App.js";
import { parseCLI } from "./cli.js";
import { listSessions } from "./hooks/useSession.js";
import { ListSessions } from "./commands/list.jsx";

/**
 * GOfortress TUI Entry Point
 * Renders the main App component in the terminal
 */

async function main() {
  const options = parseCLI();

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

  // Normal mode: render main app
  render(<App sessionId={options.session} verbose={options.verbose} />);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
