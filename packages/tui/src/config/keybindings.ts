/**
 * Key binding definitions for TUI application
 * Organized by context: global, ClaudePanel, AgentsPanel
 */

import type { KeyBinding } from "../hooks/useKeymap.js";

/**
 * Global key bindings (always active when no modal is present)
 */
export function createGlobalBindings(actions: {
  toggleFocus: () => void;
  interruptQuery: () => void;
  forceQuit: () => void;
  clearScreen: () => void;
}): KeyBinding[] {
  return [
    {
      key: "tab",
      action: actions.toggleFocus,
      description: "Switch panel focus",
    },
    {
      key: "escape",
      action: actions.interruptQuery,
      description: "Interrupt / Cancel",
    },
    {
      key: "c",
      ctrl: true,
      action: actions.forceQuit,
      description: "Force quit",
    },
    {
      key: "l",
      ctrl: true,
      action: actions.clearScreen,
      description: "Clear screen",
    },
  ];
}

/**
 * ClaudePanel key bindings (active when ClaudePanel is focused)
 */
export function createClaudePanelBindings(actions: {
  submitMessage: () => void;
  historyPrev: () => void;
  historyNext: () => void;
}): KeyBinding[] {
  return [
    {
      key: "return",
      action: actions.submitMessage,
      description: "Send message",
    },
    {
      key: "up",
      action: actions.historyPrev,
      description: "Previous input (history)",
    },
    {
      key: "down",
      action: actions.historyNext,
      description: "Next input (history)",
    },
  ];
}

/**
 * AgentsPanel key bindings (active when AgentsPanel is focused)
 */
export function createAgentsPanelBindings(actions: {
  selectPrev: () => void;
  selectNext: () => void;
  expandAgent: () => void;
}): KeyBinding[] {
  return [
    {
      key: "up",
      action: actions.selectPrev,
      description: "Select previous agent",
    },
    {
      key: "down",
      action: actions.selectNext,
      description: "Select next agent",
    },
    {
      key: "return",
      action: actions.expandAgent,
      description: "Expand agent details",
    },
  ];
}

/**
 * Key bindings reference for help display
 */
export interface KeyBindingReference {
  context: string;
  bindings: Array<{
    keys: string;
    action: string;
  }>;
}

/**
 * Generate help text for all key bindings
 */
export function getKeyBindingsHelp(): KeyBindingReference[] {
  return [
    {
      context: "Global",
      bindings: [
        { keys: "Tab", action: "Switch panel focus" },
        { keys: "Escape", action: "Interrupt query / Cancel modal" },
        { keys: "Ctrl+C", action: "Force quit" },
        { keys: "Ctrl+L", action: "Clear screen" },
      ],
    },
    {
      context: "Tabs",
      bindings: [
        { keys: "Alt+C", action: "Switch to Chat" },
        { keys: "Alt+A", action: "Switch to Agent Config" },
        { keys: "Alt+T", action: "Switch to Team Config" },
        { keys: "Alt+Y", action: "Switch to Telemetry" },
      ],
    },
    {
      context: "Claude Panel",
      bindings: [
        { keys: "Enter", action: "Submit input" },
        { keys: "Up/Down", action: "Input history" },
      ],
    },
    {
      context: "Agents Panel",
      bindings: [
        { keys: "Up/Down", action: "Navigate tree" },
        { keys: "Enter", action: "Expand agent" },
      ],
    },
  ];
}
