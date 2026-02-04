/**
 * Tests for useKeymap hook
 * Verifies keybinding resolution, modifier handling, and conflict detection
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useKeymap } from "../../src/hooks/useKeymap.js";

// Mock Ink's useInput hook
const mockInputCallback = vi.fn();
let registeredHandler: ((input: string, key: any) => void) | null = null;

vi.mock("ink", () => ({
  useInput: (handler: (input: string, key: any) => void) => {
    registeredHandler = handler;
  },
}));

describe("useKeymap", () => {
  beforeEach(() => {
    mockInputCallback.mockClear();
    registeredHandler = null;
  });

  afterEach(() => {
    registeredHandler = null;
  });

  /**
   * Helper to simulate key press
   */
  function pressKey(
    input: string,
    keyState: {
      upArrow?: boolean;
      downArrow?: boolean;
      leftArrow?: boolean;
      rightArrow?: boolean;
      return?: boolean;
      escape?: boolean;
      ctrl?: boolean;
      shift?: boolean;
      meta?: boolean;
      tab?: boolean;
      backspace?: boolean;
      delete?: boolean;
      pageDown?: boolean;
      pageUp?: boolean;
    } = {}
  ) {
    if (!registeredHandler) {
      throw new Error("No input handler registered");
    }

    const fullKeyState = {
      upArrow: false,
      downArrow: false,
      leftArrow: false,
      rightArrow: false,
      return: false,
      escape: false,
      ctrl: false,
      shift: false,
      meta: false,
      tab: false,
      backspace: false,
      delete: false,
      pageDown: false,
      pageUp: false,
      ...keyState,
    };

    registeredHandler(input, fullKeyState);
  }

  describe("Basic Key Matching", () => {
    it("should match single character keys", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "a", action, description: "Test" }])
      );

      pressKey("a");
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should not match different character", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "a", action, description: "Test" }])
      );

      pressKey("b");
      expect(action).not.toHaveBeenCalled();
    });

    it("should match case-sensitive keys", () => {
      const actionLower = vi.fn();
      const actionUpper = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "a", action: actionLower, description: "Lower" },
          { key: "A", action: actionUpper, description: "Upper" },
        ])
      );

      pressKey("a");
      expect(actionLower).toHaveBeenCalledTimes(1);
      expect(actionUpper).not.toHaveBeenCalled();

      actionLower.mockClear();
      pressKey("A");
      expect(actionUpper).toHaveBeenCalledTimes(1);
      expect(actionLower).not.toHaveBeenCalled();
    });
  });

  describe("Special Key Matching", () => {
    it("should match arrow keys", () => {
      const upAction = vi.fn();
      const downAction = vi.fn();
      const leftAction = vi.fn();
      const rightAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "up", action: upAction, description: "Up" },
          { key: "down", action: downAction, description: "Down" },
          { key: "left", action: leftAction, description: "Left" },
          { key: "right", action: rightAction, description: "Right" },
        ])
      );

      pressKey("", { upArrow: true });
      expect(upAction).toHaveBeenCalledTimes(1);

      pressKey("", { downArrow: true });
      expect(downAction).toHaveBeenCalledTimes(1);

      pressKey("", { leftArrow: true });
      expect(leftAction).toHaveBeenCalledTimes(1);

      pressKey("", { rightArrow: true });
      expect(rightAction).toHaveBeenCalledTimes(1);
    });

    it("should match return/enter key", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "return", action, description: "Enter" }])
      );

      pressKey("", { return: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should match escape key", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "escape", action, description: "Escape" }])
      );

      pressKey("", { escape: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should match tab key", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "tab", action, description: "Tab" }])
      );

      pressKey("", { tab: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should match backspace and delete", () => {
      const backspaceAction = vi.fn();
      const deleteAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "backspace", action: backspaceAction, description: "Backspace" },
          { key: "delete", action: deleteAction, description: "Delete" },
        ])
      );

      pressKey("", { backspace: true });
      expect(backspaceAction).toHaveBeenCalledTimes(1);

      pressKey("", { delete: true });
      expect(deleteAction).toHaveBeenCalledTimes(1);
    });

    it("should match page up/down keys", () => {
      const pageUpAction = vi.fn();
      const pageDownAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "pageUp", action: pageUpAction, description: "Page Up" },
          { key: "pageDown", action: pageDownAction, description: "Page Down" },
        ])
      );

      pressKey("", { pageUp: true });
      expect(pageUpAction).toHaveBeenCalledTimes(1);

      pressKey("", { pageDown: true });
      expect(pageDownAction).toHaveBeenCalledTimes(1);
    });
  });

  describe("Modifier Key Combinations", () => {
    it("should match ctrl+key combinations", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "c", ctrl: true, action, description: "Ctrl+C" }])
      );

      pressKey("c", { ctrl: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should not match ctrl+key without ctrl pressed", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "c", ctrl: true, action, description: "Ctrl+C" }])
      );

      pressKey("c");
      expect(action).not.toHaveBeenCalled();
    });

    it("should match meta+key combinations", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "k", meta: true, action, description: "Meta+K" }])
      );

      pressKey("k", { meta: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should match shift+key combinations", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "A", shift: true, action, description: "Shift+A" }])
      );

      pressKey("A", { shift: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should match multiple modifiers", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([
          {
            key: "s",
            ctrl: true,
            shift: true,
            action,
            description: "Ctrl+Shift+S",
          },
        ])
      );

      pressKey("s", { ctrl: true, shift: true });
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should not match when required modifier is missing", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([
          {
            key: "s",
            ctrl: true,
            shift: true,
            action,
            description: "Ctrl+Shift+S",
          },
        ])
      );

      // Only ctrl, missing shift
      pressKey("s", { ctrl: true });
      expect(action).not.toHaveBeenCalled();

      // Only shift, missing ctrl
      pressKey("s", { shift: true });
      expect(action).not.toHaveBeenCalled();
    });

    it("should match ctrl+special key combinations", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([
          { key: "left", ctrl: true, action, description: "Ctrl+Left" },
        ])
      );

      pressKey("", { leftArrow: true, ctrl: true });
      expect(action).toHaveBeenCalledTimes(1);
    });
  });

  describe("Binding Priority", () => {
    it("should execute first matching binding only", () => {
      const action1 = vi.fn();
      const action2 = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "a", action: action1, description: "First" },
          { key: "a", action: action2, description: "Second" },
        ])
      );

      pressKey("a");
      expect(action1).toHaveBeenCalledTimes(1);
      expect(action2).not.toHaveBeenCalled();
    });

    it("should respect binding order for overlapping patterns", () => {
      const genericAction = vi.fn();
      const specificAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "c", ctrl: true, action: specificAction, description: "Specific" },
          { key: "c", action: genericAction, description: "Generic" },
        ])
      );

      // Ctrl+C should match first (specific)
      pressKey("c", { ctrl: true });
      expect(specificAction).toHaveBeenCalledTimes(1);
      expect(genericAction).not.toHaveBeenCalled();

      specificAction.mockClear();
      // Plain "c" should match second (generic)
      pressKey("c");
      expect(genericAction).toHaveBeenCalledTimes(1);
      expect(specificAction).not.toHaveBeenCalled();
    });
  });

  describe("Enabled State", () => {
    it("should not trigger actions when disabled", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "a", action, description: "Test" }], false)
      );

      pressKey("a");
      expect(action).not.toHaveBeenCalled();
    });

    it("should trigger actions when enabled (default)", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "a", action, description: "Test" }])
      );

      pressKey("a");
      expect(action).toHaveBeenCalledTimes(1);
    });

    it("should update behavior when enabled state changes", () => {
      const action = vi.fn();
      const { rerender } = renderHook(
        ({ enabled }) =>
          useKeymap([{ key: "a", action, description: "Test" }], enabled),
        { initialProps: { enabled: true } }
      );

      pressKey("a");
      expect(action).toHaveBeenCalledTimes(1);

      action.mockClear();
      rerender({ enabled: false });
      pressKey("a");
      expect(action).not.toHaveBeenCalled();

      rerender({ enabled: true });
      pressKey("a");
      expect(action).toHaveBeenCalledTimes(1);
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty bindings array", () => {
      renderHook(() => useKeymap([]));

      // Should not throw
      expect(() => pressKey("a")).not.toThrow();
    });

    it("should handle unknown special keys gracefully", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([
          // @ts-expect-error Testing invalid key
          { key: "nonexistent", action, description: "Invalid" },
        ])
      );

      pressKey("a");
      expect(action).not.toHaveBeenCalled();
    });

    it("should handle rapid key presses", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "a", action, description: "Test" }])
      );

      pressKey("a");
      pressKey("a");
      pressKey("a");

      expect(action).toHaveBeenCalledTimes(3);
    });

    it("should handle modifier-only presses (no matching binding)", () => {
      const action = vi.fn();
      renderHook(() =>
        useKeymap([{ key: "c", ctrl: true, action, description: "Ctrl+C" }])
      );

      // Just ctrl pressed, but no "c"
      pressKey("", { ctrl: true });
      expect(action).not.toHaveBeenCalled();
    });

    it("should handle action throwing error", () => {
      const errorAction = vi.fn(() => {
        throw new Error("Action error");
      });
      const fallbackAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "a", action: errorAction, description: "Error" },
          { key: "b", action: fallbackAction, description: "Fallback" },
        ])
      );

      // First binding throws error (should propagate)
      expect(() => pressKey("a")).toThrow("Action error");

      // Second binding should still work
      pressKey("b");
      expect(fallbackAction).toHaveBeenCalledTimes(1);
    });
  });

  describe("Real-World Scenarios", () => {
    it("should handle typical TUI navigation bindings", () => {
      const actions = {
        switchPanel: vi.fn(),
        quit: vi.fn(),
        scrollUp: vi.fn(),
        scrollDown: vi.fn(),
        submit: vi.fn(),
      };

      renderHook(() =>
        useKeymap([
          { key: "tab", action: actions.switchPanel, description: "Switch panel" },
          { key: "q", action: actions.quit, description: "Quit" },
          { key: "c", ctrl: true, action: actions.quit, description: "Quit (Ctrl+C)" },
          { key: "up", action: actions.scrollUp, description: "Scroll up" },
          { key: "down", action: actions.scrollDown, description: "Scroll down" },
          { key: "return", action: actions.submit, description: "Submit" },
        ])
      );

      pressKey("", { tab: true });
      expect(actions.switchPanel).toHaveBeenCalledTimes(1);

      pressKey("q");
      expect(actions.quit).toHaveBeenCalledTimes(1);

      pressKey("", { upArrow: true });
      expect(actions.scrollUp).toHaveBeenCalledTimes(1);

      pressKey("", { downArrow: true });
      expect(actions.scrollDown).toHaveBeenCalledTimes(1);

      pressKey("", { return: true });
      expect(actions.submit).toHaveBeenCalledTimes(1);
    });

    it("should handle vim-style navigation", () => {
      const actions = {
        up: vi.fn(),
        down: vi.fn(),
        left: vi.fn(),
        right: vi.fn(),
      };

      renderHook(() =>
        useKeymap([
          { key: "k", action: actions.up, description: "Up" },
          { key: "j", action: actions.down, description: "Down" },
          { key: "h", action: actions.left, description: "Left" },
          { key: "l", action: actions.right, description: "Right" },
        ])
      );

      pressKey("k");
      expect(actions.up).toHaveBeenCalledTimes(1);

      pressKey("j");
      expect(actions.down).toHaveBeenCalledTimes(1);

      pressKey("h");
      expect(actions.left).toHaveBeenCalledTimes(1);

      pressKey("l");
      expect(actions.right).toHaveBeenCalledTimes(1);
    });

    it("should handle modal-specific bindings that override global", () => {
      const globalQuit = vi.fn();
      const modalCancel = vi.fn();

      // Global bindings
      const { rerender } = renderHook(
        ({ isModal }) =>
          useKeymap(
            isModal
              ? [{ key: "escape", action: modalCancel, description: "Cancel" }]
              : [{ key: "q", action: globalQuit, description: "Quit" }],
            true
          ),
        { initialProps: { isModal: false } }
      );

      // Global mode
      pressKey("q");
      expect(globalQuit).toHaveBeenCalledTimes(1);
      expect(modalCancel).not.toHaveBeenCalled();

      globalQuit.mockClear();
      // Modal mode
      rerender({ isModal: true });
      pressKey("", { escape: true });
      expect(modalCancel).toHaveBeenCalledTimes(1);

      pressKey("q");
      expect(globalQuit).not.toHaveBeenCalled();
    });
  });

  describe("Conflict Detection", () => {
    it("should detect duplicate bindings at same priority", () => {
      const action1 = vi.fn();
      const action2 = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "a", action: action1, description: "First" },
          { key: "a", action: action2, description: "Duplicate" },
        ])
      );

      // First wins, but both are defined (potential conflict)
      pressKey("a");
      expect(action1).toHaveBeenCalledTimes(1);
      expect(action2).not.toHaveBeenCalled();
    });

    it("should differentiate bindings by modifiers", () => {
      const plainAction = vi.fn();
      const ctrlAction = vi.fn();
      const shiftAction = vi.fn();

      renderHook(() =>
        useKeymap([
          { key: "a", action: plainAction, description: "Plain" },
          { key: "a", ctrl: true, action: ctrlAction, description: "Ctrl" },
          { key: "A", shift: true, action: shiftAction, description: "Shift" },
        ])
      );

      pressKey("a");
      expect(plainAction).toHaveBeenCalledTimes(1);
      expect(ctrlAction).not.toHaveBeenCalled();
      expect(shiftAction).not.toHaveBeenCalled();

      plainAction.mockClear();
      pressKey("a", { ctrl: true });
      expect(ctrlAction).toHaveBeenCalledTimes(1);
      expect(plainAction).not.toHaveBeenCalled();

      ctrlAction.mockClear();
      pressKey("A", { shift: true });
      expect(shiftAction).toHaveBeenCalledTimes(1);
      expect(plainAction).not.toHaveBeenCalled();
    });
  });
});
