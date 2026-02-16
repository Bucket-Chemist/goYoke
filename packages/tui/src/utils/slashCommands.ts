/**
 * Slash command registry — built-in commands + skills from SKILL.md files
 * Loaded once at import time, cached for the session.
 */

import { readdirSync, readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

export interface SlashCommand {
  name: string;
  description: string;
  source: "builtin" | "skill";
}

/** Built-in TUI commands (handled in ClaudePanel before reaching Claude) */
const BUILTIN_COMMANDS: SlashCommand[] = [
  { name: "model", description: "Switch model (haiku/sonnet/opus)", source: "builtin" },
  { name: "clear", description: "Clear message history", source: "builtin" },
  { name: "help", description: "Show available commands", source: "builtin" },
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
  ...loadSkills(),
].sort((a, b) => a.name.localeCompare(b.name));

/** Filter commands by prefix (case-insensitive) */
export function filterCommands(query: string): SlashCommand[] {
  if (!query) return ALL_COMMANDS;
  const lower = query.toLowerCase();
  return ALL_COMMANDS.filter((cmd) => cmd.name.toLowerCase().startsWith(lower));
}
