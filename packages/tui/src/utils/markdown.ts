/**
 * Markdown rendering utilities for terminal output
 * Uses marked with marked-terminal for ANSI-styled markdown
 */

import { marked } from "marked";
import { markedTerminal } from "marked-terminal";

// Configure marked to use terminal renderer
marked.use(markedTerminal() as any);

/**
 * Render markdown content to ANSI-styled terminal output
 * @param content - Raw markdown string
 * @returns ANSI-styled string ready for terminal display
 */
export function renderMarkdown(content: string): string {
  try {
    return marked(content) as string;
  } catch (error) {
    // Fallback to raw content if parsing fails
    return content;
  }
}
