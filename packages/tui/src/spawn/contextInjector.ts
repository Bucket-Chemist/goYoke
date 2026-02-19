/**
 * contextInjector.ts — TypeScript port of pkg/routing/identity_loader.go
 *
 * Injects agent identity + rules + conventions into prompts for spawn_agent
 * spawned agents, mirroring what gogent-validate does for Task() calls and
 * what gogent-team-run does for team-run spawns.
 *
 * Injection order (matches BuildFullAgentContext in Go):
 *   1. Session directory context  (if GOGENT_SESSION_DIR is set)
 *   2. Agent identity             (~/.claude/agents/{id}/{id}.md body)
 *   3. Rules + conventions        (from context_requirements in agents-index.json)
 *   4. Original prompt
 *
 * Marker strings must stay in sync with pkg/routing/identity_loader.go constants.
 */

import { readFile } from "fs/promises";
import { join } from "path";
import { homedir } from "os";
import { logger } from "../utils/logger.js";
import type { AgentConfig } from "./agentConfig.js";

// ── Markers (keep in sync with pkg/routing/identity_loader.go) ──────────────

export const IDENTITY_MARKER = "[AGENT IDENTITY - AUTO-INJECTED]";
export const IDENTITY_END_MARKER = "[END AGENT IDENTITY]";
export const SESSION_MARKER = "[SESSION CONTEXT]";
export const SESSION_END_MARKER = "[END SESSION CONTEXT]";
export const CONVENTIONS_MARKER = "[CONVENTIONS]";
export const CONVENTIONS_END_MARKER = "[END CONVENTIONS]";

// ── Helpers ──────────────────────────────────────────────────────────────────

function getClaudeConfigDir(): string {
  return join(process.env["HOME"] || homedir(), ".claude");
}

/** Read a file, returning empty string on any error (ENOENT, EACCES, etc). */
async function readFileSafe(filePath: string): Promise<string> {
  try {
    return await readFile(filePath, "utf-8");
  } catch {
    return "";
  }
}

/**
 * Strip YAML frontmatter delimited by --- from markdown.
 * Pure string processing — no YAML parser. Matches Go StripYAMLFrontmatter.
 */
export function stripYAMLFrontmatter(content: string): string {
  const trimmed = content.trimStart();
  if (!trimmed.startsWith("---")) {
    return content;
  }

  const openIdx = content.indexOf("---");
  const rest = content.slice(openIdx + 3);

  const closeIdx = rest.indexOf("\n---");
  if (closeIdx === -1) {
    return content; // Malformed frontmatter — return as-is
  }

  let afterClose = rest.slice(closeIdx + 4);
  const nlIdx = afterClose.indexOf("\n");
  if (nlIdx >= 0) {
    afterClose = afterClose.slice(nlIdx + 1);
  }

  return afterClose.replace(/^\n+/, "");
}

// ── Loaders ──────────────────────────────────────────────────────────────────

/**
 * Load agent identity body from ~/.claude/agents/{agentId}/{agentId}.md.
 * Returns empty string if the file doesn't exist.
 */
export async function loadAgentIdentity(agentId: string): Promise<string> {
  if (!agentId) return "";
  const path = join(getClaudeConfigDir(), "agents", agentId, `${agentId}.md`);
  const content = await readFileSafe(path);
  return content ? stripYAMLFrontmatter(content) : "";
}

/** Load a rules file from ~/.claude/rules/{filename}. */
async function loadRulesContent(filename: string): Promise<string> {
  return readFileSafe(join(getClaudeConfigDir(), "rules", filename));
}

/** Load a convention file from ~/.claude/conventions/{filename}. */
async function loadConventionContent(filename: string): Promise<string> {
  return readFileSafe(join(getClaudeConfigDir(), "conventions", filename));
}

/** Read GOGENT_SESSION_DIR from env (matches Go's GetSessionDir). */
function getSessionDir(): string {
  return process.env["GOGENT_SESSION_DIR"] ?? "";
}

// ── Main entry point ─────────────────────────────────────────────────────────

/**
 * Build complete agent context: session + identity + rules + conventions + prompt.
 *
 * Equivalent to Go's routing.BuildFullAgentContext. Used by spawn_agent MCP tool
 * to inject context into the initial stdin prompt before spawning a claude CLI process.
 *
 * Returns the augmented prompt on success, or the original prompt if nothing
 * was available to inject (so the spawn is never blocked by injection failures).
 */
export async function buildFullAgentContext(
  agentId: string,
  agentConfig: AgentConfig | null,
  originalPrompt: string
): Promise<string> {
  // Double-injection prevention
  if (originalPrompt.includes(IDENTITY_MARKER)) {
    return originalPrompt;
  }

  const sections: string[] = [];
  let injected = false;

  try {
    // 0. Session directory context
    if (!originalPrompt.includes(SESSION_MARKER)) {
      const sessionDir = getSessionDir();
      if (sessionDir) {
        sections.push(SESSION_MARKER);
        sections.push(`SESSION_DIR: ${sessionDir}`);
        sections.push("Write output artifacts (plans, reviews, analysis) to SESSION_DIR/.");
        sections.push(SESSION_END_MARKER);
        sections.push("");
        injected = true;
      }
    }

    // 1. Agent identity
    const identity = await loadAgentIdentity(agentId);
    if (identity) {
      sections.push(IDENTITY_MARKER);
      sections.push(`--- ${agentId} identity ---`);
      sections.push(identity);
      sections.push(IDENTITY_END_MARKER);
      sections.push("");
      injected = true;
    }

    // 2. Rules and conventions
    const requirements = agentConfig?.context_requirements;
    if (requirements && !originalPrompt.includes(CONVENTIONS_MARKER)) {
      const convSections: string[] = [CONVENTIONS_MARKER, ""];

      // Rules
      for (const rulesFile of requirements.rules ?? []) {
        const content = await loadRulesContent(rulesFile);
        if (content) {
          convSections.push(`--- ${rulesFile} ---`);
          convSections.push(content);
          convSections.push("");
        }
      }

      // Base conventions
      const baseConventions = requirements.conventions?.base ?? [];
      for (const convFile of baseConventions) {
        const content = await loadConventionContent(convFile);
        if (content) {
          convSections.push(`--- ${convFile} ---`);
          convSections.push(content);
          convSections.push("");
        }
      }

      convSections.push(CONVENTIONS_END_MARKER);
      convSections.push("");

      // Only include if something was actually loaded (> just the 4 structural lines)
      if (convSections.length > 4) {
        sections.push(convSections.join("\n"));
        injected = true;
      }
    }
  } catch (err) {
    // Never block a spawn due to injection failure — fall back to original prompt
    void logger.warn("[contextInjector] Failed to build agent context, using raw prompt", {
      agentId,
      error: err instanceof Error ? err.message : String(err),
    });
    return originalPrompt;
  }

  if (!injected) {
    return originalPrompt;
  }

  sections.push("---");
  sections.push("");
  sections.push(originalPrompt);

  return sections.join("\n");
}
