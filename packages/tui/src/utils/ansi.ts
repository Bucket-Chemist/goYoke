/**
 * ANSI escape code sanitization utilities
 * Strips ANSI codes from external content (tool results, agent output)
 * to prevent rendering conflicts in Ink components
 */

import stripAnsi from "strip-ansi";

/**
 * Strip ANSI escape codes from external content
 * Use on tool results, agent output, etc. before rendering in Ink
 *
 * @param text - Raw text that may contain ANSI codes
 * @returns Sanitized text without ANSI codes
 */
export function sanitizeAnsi(text: string): string {
  try {
    return stripAnsi(text);
  } catch {
    // Fallback to original text if stripping fails
    return text;
  }
}
