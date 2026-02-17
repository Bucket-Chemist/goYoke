/**
 * Slash command registry — TUI built-in + native CC CLI + skills from SKILL.md
 * Loaded once at import time, cached for the session.
 */

import { readdirSync, readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

export interface SlashCommand {
  name: string;
  description: string;
  source: "builtin" | "native" | "skill";
}

/** TUI-specific commands (handled in ClaudePanel before reaching Claude) */
const BUILTIN_COMMANDS: SlashCommand[] = [
  { name: "model", description: "Switch model (haiku/sonnet/opus)", source: "builtin" },
  { name: "clear", description: "Clear message history", source: "builtin" },
  { name: "help", description: "Show available commands", source: "builtin" },
];

/** Native Claude Code CLI commands (passed through to the SDK/CLI process) */
const NATIVE_COMMANDS: SlashCommand[] = [
  { name: "compact", description: "Compact conversation with optional focus instructions", source: "native" },
  { name: "config", description: "Open the Settings interface (Config tab)", source: "native" },
  { name: "context", description: "Visualize current context usage as a colored grid", source: "native" },
  { name: "copy", description: "Copy the last assistant response to clipboard", source: "native" },
  { name: "cost", description: "Show token usage statistics for this session", source: "native" },
  { name: "debug", description: "Troubleshoot the current session by reading the debug log", source: "native" },
  { name: "desktop", description: "Hand off the current CLI session to Claude Code Desktop app", source: "native" },
  { name: "doctor", description: "Check the health of your Claude Code installation", source: "native" },
  { name: "exit", description: "Exit the REPL", source: "native" },
  { name: "export", description: "Export the current conversation to a file or clipboard", source: "native" },
  { name: "init", description: "Initialize project with CLAUDE.md guide", source: "native" },
  { name: "login", description: "Log in to your Anthropic account", source: "native" },
  { name: "logout", description: "Log out of your Anthropic account", source: "native" },
  { name: "mcp", description: "Manage MCP server connections and OAuth authentication", source: "native" },
  { name: "memory", description: "Edit CLAUDE.md memory files", source: "native" },
  { name: "permissions", description: "View or update permission rules", source: "native" },
  { name: "plan", description: "Enter plan mode directly from the prompt", source: "native" },
  { name: "rename", description: "Rename the current session for easier identification", source: "native" },
  { name: "resume", description: "Resume a conversation by ID or name, or open session picker", source: "native" },
  { name: "rewind", description: "Rewind the conversation and/or code to a previous point", source: "native" },
  { name: "stats", description: "Visualize daily usage, session history, streaks, and model preferences", source: "native" },
  { name: "status", description: "Show version, model, account, and connectivity info", source: "native" },
  { name: "statusline", description: "Set up Claude Code's status line UI", source: "native" },
  { name: "tasks", description: "List and manage background tasks", source: "native" },
  { name: "teleport", description: "Resume a remote session from claude.ai", source: "native" },
  { name: "theme", description: "Change the color theme", source: "native" },
  { name: "todos", description: "List current TODO items", source: "native" },
  { name: "usage", description: "Show plan usage limits and rate limit status", source: "native" },
  { name: "vim", description: "Toggle Vim keybinding mode for input", source: "native" },
];

/** Extract name + description from SKILL.md YAML frontmatter */
function parseSkillFrontmatter(content: string): { name: string; description: string } | null {
  const match = content.match(/^---\n([\s\S]*?)\n---/);
  if (!match?.[1]) return null;

  const frontmatter = match[1];
  const nameMatch = frontmatter.match(/^name:\s*(.+)$/m);
  const descMatch = frontmatter.match(/^description:\s*(.+)$/m);

  if (!nameMatch?.[1]) return null;

  return {
    name: nameMatch[1].trim(),
    description: descMatch?.[1]?.trim() || "",
  };
}

/** Scan skill directories for available skills */
function loadSkills(): SlashCommand[] {
  const skillsDir = join(homedir(), ".claude", "skills");
  const skills: SlashCommand[] = [];

  try {
    const dirs = readdirSync(skillsDir, { withFileTypes: true });
    for (const dir of dirs) {
      if (!dir.isDirectory()) continue;
      const skillFile = join(skillsDir, dir.name, "SKILL.md");
      try {
        const content = readFileSync(skillFile, "utf-8");
        const parsed = parseSkillFrontmatter(content);
        if (parsed) {
          skills.push({ ...parsed, source: "skill" });
        }
      } catch {
        // SKILL.md doesn't exist or can't be read — skip
      }
    }
  } catch {
    // Skills directory doesn't exist — return empty
  }

  return skills;
}

/** All available slash commands, sorted alphabetically */
export const ALL_COMMANDS: SlashCommand[] = [
  ...BUILTIN_COMMANDS,
  ...NATIVE_COMMANDS,
  ...loadSkills(),
].sort((a, b) => a.name.localeCompare(b.name));

/** Filter commands by prefix (case-insensitive) */
export function filterCommands(query: string): SlashCommand[] {
  if (!query) return ALL_COMMANDS;
  const lower = query.toLowerCase();
  return ALL_COMMANDS.filter((cmd) => cmd.name.toLowerCase().startsWith(lower));
}
