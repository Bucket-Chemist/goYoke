/**
 * Markdown rendering utilities for terminal output
 * Uses marked with marked-terminal for ANSI-styled markdown
 */

import { marked } from "marked";
import { markedTerminal } from "marked-terminal";

// Configure marked to use terminal renderer
// markedTerminal returns a configuration object but types don't align perfectly
marked.use(markedTerminal() as Parameters<typeof marked.use>[0]);

/**
 * Render markdown content to ANSI-styled terminal output
 * @param content - Raw markdown string
 * @returns ANSI-styled string ready for terminal display
 */
export function renderMarkdown(content: string): string {
  try {
    return marked(content) as string;
  } catch (_error) {
    // Fallback to raw content if parsing fails
    return content;
  }
}
