#!/usr/bin/env node
/**
 * CLI argument parsing for GOfortress TUI
 * Handles session management flags and legacy fallback
 */

import { Command } from "commander";

export interface CLIOptions {
  list: boolean;
  session?: string;
  verbose: boolean;
  legacy: boolean;
}

/**
 * Parse CLI arguments
 * Returns parsed options for use in App initialization
 */
export function parseCLI(): CLIOptions {
  const program = new Command();

  program
    .name("gofortress-tui")
    .description("GOfortress Terminal User Interface")
    .version("1.0.0")
    .option("-l, --list", "List available sessions", false)
    .option("-s, --session <id>", "Resume session by ID")
    .option("-v, --verbose", "Enable verbose logging", false)
    .option("--legacy", "Use Go TUI (fallback)", false)
    .parse(process.argv);

  const options = program.opts<CLIOptions>();

  return {
    list: options.list ?? false,
    session: options.session,
    verbose: options.verbose ?? false,
    legacy: options.legacy ?? false,
  };
}
