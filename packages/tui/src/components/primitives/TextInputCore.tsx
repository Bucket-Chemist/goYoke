/**
 * TextInputCore - Custom text input replacing ink-text-input
 *
 * Fixes the delete key border tearing bug caused by ink-text-input's:
 * - Stale closure state (useState cursor in useInput callback)
 * - chalk.inverse() ANSI bleed into Ink's box-drawing borders
 * - Unconditional setState on every keypress
 *
 * Design choices:
 * - useRef for cursor position (no stale closures during rapid key repeat)
 * - Ink's native <Text inverse> for cursor (Yoga layout, no raw ANSI injection)
 * - onChange only fires when value actually changes
 * - forceRender via useReducer for cursor-only moves (left/right arrow)
 */

import React, { useRef, useReducer, useEffect } from "react";
import { Text, useInput } from "ink";

export interface TextInputCoreProps {
  /** Current input value (controlled) */
  value: string;
  /** Called when the input value changes */
  onChange: (value: string) => void;
  /** Called when Enter is pressed */
  onSubmit?: () => void;
  /** Placeholder text when value is empty */
  placeholder?: string;
  /** Whether this input is focused and captures keystrokes */
  focused?: boolean;
}

/**
 * Custom text input with ref-based cursor for stable rapid key handling.
 * Drop-in replacement for ink-text-input with identical visual behavior.
 */
export function TextInputCore({
  value,
  onChange,
  onSubmit,
  placeholder = "",
  focused = true,
}: TextInputCoreProps): React.ReactElement {
  // Cursor position stored in ref — always current in useInput callback
  const cursorRef = useRef(value.length);

  // Force re-render trigger for cursor-only moves (no value change)
  const [, forceRender] = useReducer((n: number) => n + 1, 0);

  // Sync cursor to value length when value changes externally (e.g., parent resets to "")
  useEffect(() => {
    if (cursorRef.current > value.length) {
      cursorRef.current = value.length;
    }
  }, [value]);

  // Input handler — reads cursorRef.current (never stale)
  useInput(
    (input, key) => {
      // Pass through keys we don't handle
      if (
        key.upArrow ||
        key.downArrow ||
        (key.ctrl && input === "c") ||
        key.tab ||
        (key.shift && key.tab)
      ) {
        return;
      }

      if (key.return) {
        onSubmit?.();
        return;
      }

      const cursor = cursorRef.current;
      let nextValue = value;
      let nextCursor = cursor;

      if (key.leftArrow) {
        nextCursor = Math.max(0, cursor - 1);
      } else if (key.rightArrow) {
        nextCursor = Math.min(value.length, cursor + 1);
      } else if (key.backspace || key.delete) {
        // Ink 5 quirk: on Linux terminals, Backspace sends \x7f which Ink
        // maps to key.delete (not key.backspace). key.backspace only fires
        // for \x08 (Ctrl+H). Since we have no real use for forward-delete
        // in a single-line input, treat both as backward-delete.
        if (cursor > 0) {
          nextValue =
            value.slice(0, cursor - 1) + value.slice(cursor);
          nextCursor = cursor - 1;
        }
      } else if (input && input.length > 0 && !key.ctrl && !key.meta) {
        // Regular character input (including pasted text)
        nextValue =
          value.slice(0, cursor) + input + value.slice(cursor);
        nextCursor = cursor + input.length;
      }

      // Clamp cursor to valid range
      nextCursor = Math.max(0, Math.min(nextValue.length, nextCursor));

      // Update cursor ref
      cursorRef.current = nextCursor;

      if (nextValue !== value) {
        // Value changed — onChange triggers parent re-render
        onChange(nextValue);
      } else if (nextCursor !== cursor) {
        // Cursor moved without value change — force re-render for visual update
        forceRender();
      }
      // If neither changed (e.g., delete at position 0), do nothing — no re-render
    },
    { isActive: focused },
  );

  // Render: cursor as Ink <Text inverse> (no chalk ANSI injection)
  if (!focused) {
    // Unfocused: plain text, no cursor
    if (value.length === 0 && placeholder) {
      return <Text dimColor>{placeholder}</Text>;
    }
    return <Text>{value}</Text>;
  }

  // Focused with empty value: show inverse cursor on placeholder or blank
  if (value.length === 0) {
    if (placeholder.length > 0) {
      return (
        <Text>
          <Text inverse>{placeholder[0]}</Text>
          <Text dimColor>{placeholder.slice(1)}</Text>
        </Text>
      );
    }
    return <Text inverse> </Text>;
  }

  // Focused with value: render with cursor indicator
  const cursor = cursorRef.current;
  const before = value.slice(0, cursor);
  const cursorChar = cursor < value.length ? value[cursor] : " ";
  const after = cursor < value.length ? value.slice(cursor + 1) : "";

  return (
    <Text>
      {before}
      <Text inverse>{cursorChar}</Text>
      {after}
    </Text>
  );
}
