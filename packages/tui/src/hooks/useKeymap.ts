/**
 * useKeymap hook - Keyboard binding matcher and handler
 * Provides declarative key binding system for Ink applications
 */

import { useInput } from "ink";

/**
 * Key binding definition with action and metadata
 */
export interface KeyBinding {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  action: () => void;
  description: string;
}

/**
 * Match input against a key binding specification
 */
function matchesBinding(
  input: string,
  key: {
    upArrow: boolean;
    downArrow: boolean;
    leftArrow: boolean;
    rightArrow: boolean;
    return: boolean;
    escape: boolean;
    ctrl: boolean;
    shift: boolean;
    tab: boolean;
    backspace: boolean;
    delete: boolean;
    pageDown: boolean;
    pageUp: boolean;
    meta: boolean;
  },
  binding: KeyBinding
): boolean {
  // Handle special keys
  const specialKeyMap: Record<string, boolean> = {
    up: key.upArrow,
    down: key.downArrow,
    left: key.leftArrow,
    right: key.rightArrow,
    return: key.return,
    escape: key.escape,
    tab: key.tab,
    backspace: key.backspace,
    delete: key.delete,
    pageDown: key.pageDown,
    pageUp: key.pageUp,
  };

  // Check if binding is for a special key
  if (binding.key in specialKeyMap) {
    if (!specialKeyMap[binding.key]) {
      return false;
    }
  } else {
    // Regular character key
    if (input !== binding.key) {
      return false;
    }
  }

  // Check modifiers
  if (binding.ctrl && !key.ctrl) {
    return false;
  }
  if (binding.meta && !key.meta) {
    return false;
  }
  if (binding.shift && !key.shift) {
    return false;
  }

  return true;
}

/**
 * Hook to register keyboard bindings
 * Processes bindings in order and executes the first match
 *
 * @param bindings - Array of key bindings to register
 * @param enabled - Whether bindings are active (default: true)
 *
 * @example
 * ```tsx
 * useKeymap([
 *   { key: "tab", action: () => switchPanel(), description: "Switch panel" },
 *   { key: "c", ctrl: true, action: () => quit(), description: "Quit" },
 * ]);
 * ```
 */
export function useKeymap(bindings: KeyBinding[], enabled = true): void {
  useInput((input, key) => {
    if (!enabled) {
      return;
    }

    for (const binding of bindings) {
      if (matchesBinding(input, key, binding)) {
        binding.action();
        return; // Stop after first match
      }
    }
  });
}
