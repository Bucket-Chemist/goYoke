#!/usr/bin/env node
/**
 * CLI argument parsing for GOfortress TUI
 * Handles session management flags and legacy fallback
 */

import { Command } from "commander";
import { readFileSync, unlinkSync } from "fs";

export interface CLIOptions {
  list: boolean;
  session?: string;
  resume: boolean;
  verbose: boolean;
  legacy: boolean;
}

/**
 * Read launch args from temp file written by the gofortress bash function.
 * Zellij doesn't propagate env vars to layout pane commands, so the bash
 * function writes args to a temp file instead.
 */
function readLaunchArgs(): { session?: string; resume?: boolean } {
  const argsFile = "/tmp/.gofortress-launch-args";
  try {
    const content = readFileSync(argsFile, "utf-8");
    unlinkSync(argsFile);
    const result: { session?: string; resume?: boolean } = {};
    for (const line of content.split("\n")) {
      const [key, val] = line.split("=", 2);
      if (key === "session" && val) result.session = val.trim();
      if (key === "resume" && val?.trim() === "1") result.resume = true;
    }
    return result;
  } catch {
    return {};
  }
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
    .option("-r, --resume", "Resume most recent session", false)
    .option("-v, --verbose", "Enable verbose logging", false)
    .option("--legacy", "Use Go TUI (fallback)", false)
    .parse(process.argv);

  const options = program.opts<CLIOptions>();
  const launchArgs = readLaunchArgs();

  return {
    list: options.list ?? false,
    session: options.session ?? launchArgs.session,
    resume: options.resume ?? launchArgs.resume ?? false,
    verbose: options.verbose ?? false,
    legacy: options.legacy ?? false,
  };
}
