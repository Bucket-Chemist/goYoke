/**
 * Centralized theme constants for GOfortress TUI.
 * All color/style values should be imported from here.
 * See: Gemini audit GAP-1 (theming)
 */

export const colors = {
  // Primary palette
  primary: "cyan",
  secondary: "blue",
  accent: "magenta",

  // Semantic colors
  success: "green",
  warning: "yellow",
  error: "red",
  muted: "gray",

  // Panel states
  focused: "cyan",
  unfocused: "gray",

  // Agent status colors
  agentSpawning: "yellow",
  agentRunning: "blue",
  agentComplete: "green",
  agentError: "red",

  // Message roles
  userMessage: "cyan",
  assistantMessage: "white",
  systemMessage: "gray",
} as const;

export const borders = {
  panel: "single",
  modal: "double",
  banner: "round",
  input: "single",
} as const;

export const icons = {
  agentSpawning: "◐",
  agentRunning: "●",
  agentComplete: "✓",
  agentError: "✗",
  treeIndent: "│",
  treeBranch: "├─",
  treeLeaf: "└─",
  teamRoot: "▶",
  sectionHeader: "─",
} as const;

export type ThemeColor = typeof colors[keyof typeof colors];
export type BorderStyle = typeof borders[keyof typeof borders];
