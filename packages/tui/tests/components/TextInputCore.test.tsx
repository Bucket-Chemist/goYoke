/**
 * TextInputCore component tests
 *
 * Coverage:
 * - Rendering: placeholder, value, cursor visibility by focus state
 * - Input handling: typing, backspace, forward-delete, arrow keys, enter
 * - Edge cases: boundary clamping, external value reset, paste
 * - Props contract: focused, onChange fire semantics, onSubmit
 *
 * Design note:
 * TextInputCore is a controlled component — the parent owns value state.
 * Tests use a stateful <Wrapper> component so ink re-renders correctly when
 * onChange fires. Without this, onChange fires but lastFrame() never updates.
 *
 * Cursor rendering:
 * Ink's <Text inverse> emits ANSI escape sequences (\x1b[7m ... \x1b[27m).
 * lastFrame() includes these raw sequences, so cursor presence is detected
 * via regex matching the inverse-on/inverse-off pair around a character.
 */

import React, { useState } from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { TextInputCore } from "../../src/components/primitives/TextInputCore.js";

// ANSI escape sequence for Ink's <Text inverse> (SGR 7 = reverse video)
const INVERSE_ON = "\x1b[7m";
const INVERSE_OFF = "\x1b[27m";

/** Returns true if frame contains an inverse-highlighted character (cursor visible) */
function hasCursor(frame: string | undefined): boolean {
  return typeof frame === "string" && frame.includes(INVERSE_ON);
}

/**
 * Returns the character(s) rendered inside the inverse highlight.
 * Useful for asserting which character is at the cursor position.
 */
function cursorChar(frame: string | undefined): string {
  if (!frame) return "";
  const match = frame.match(new RegExp(`${INVERSE_ON}([^${INVERSE_OFF}]*)${INVERSE_OFF}`));
  return match ? match[1] ?? "" : "";
}

// ---------------------------------------------------------------------------
// Stateful wrapper — lets tests exercise the full controlled-component loop
// ---------------------------------------------------------------------------

interface WrapperProps {
  initialValue?: string;
  placeholder?: string;
  focused?: boolean;
  onSubmit?: () => void;
  onChangeSpy?: (v: string) => void;
}

/**
 * Controlled wrapper that owns the value state so ink-testing-library
 * re-renders after each onChange call, keeping lastFrame() current.
 */
function Wrapper({
  initialValue = "",
  placeholder = "",
  focused = true,
  onSubmit,
  onChangeSpy,
}: WrapperProps) {
  const [value, setValue] = useState(initialValue);

  const handleChange = (v: string) => {
    setValue(v);
    onChangeSpy?.(v);
  };

  return (
    <TextInputCore
      value={value}
      onChange={handleChange}
      onSubmit={onSubmit}
      placeholder={placeholder}
      focused={focused}
    />
  );
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

describe("TextInputCore", () => {
  describe("Rendering", () => {
    it("renders placeholder text when value is empty and focused", () => {
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} placeholder="Type here" focused />);
      const frame = lastFrame() ?? "";
      expect(frame).toContain("Type here");
    });

    it("shows first placeholder character as cursor when focused and empty", () => {
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} placeholder="Type here" focused />);
      // First char of placeholder should be under the inverse cursor
      expect(cursorChar(lastFrame())).toBe("T");
    });

    it("shows remaining placeholder chars dimmed after cursor when focused and empty", () => {
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} placeholder="Type here" focused />);
      // The rest of the placeholder is rendered after the cursor highlight
      expect(lastFrame()).toContain("ype here");
    });

    it("renders an inverse space as cursor when focused, empty, and no placeholder", () => {
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} placeholder="" focused />);
      // Cursor on blank space — inverse highlight present
      expect(hasCursor(lastFrame())).toBe(true);
      expect(cursorChar(lastFrame())).toBe(" ");
    });

    it("renders value with inverse cursor at end when focused", () => {
      const { lastFrame } = render(<TextInputCore value="hello" onChange={vi.fn()} focused />);
      const frame = lastFrame() ?? "";
      // Text before cursor
      expect(frame).toContain("hell");
      // Cursor sits on a trailing space (end of string)
      expect(hasCursor(frame)).toBe(true);
      expect(cursorChar(frame)).toBe(" ");
    });

    it("renders value without cursor when unfocused", () => {
      const { lastFrame } = render(<TextInputCore value="hello" onChange={vi.fn()} focused={false} />);
      expect(lastFrame()).toContain("hello");
      expect(hasCursor(lastFrame())).toBe(false);
    });

    it("renders placeholder dimmed (no cursor) when unfocused and value is empty", () => {
      const { lastFrame } = render(
        <TextInputCore value="" onChange={vi.fn()} placeholder="Type here" focused={false} />,
      );
      const frame = lastFrame() ?? "";
      expect(frame).toContain("Type here");
      // No inverse highlight — placeholder is purely dimmed
      expect(hasCursor(frame)).toBe(false);
    });

    it("renders empty string (no placeholder, no cursor) when unfocused and value is empty", () => {
      const { lastFrame } = render(
        <TextInputCore value="" onChange={vi.fn()} placeholder="" focused={false} />,
      );
      // Nothing to show — frame should be empty or whitespace
      expect(hasCursor(lastFrame())).toBe(false);
    });
  });

  // ---------------------------------------------------------------------------
  // Input handling
  // ---------------------------------------------------------------------------

  describe("Input handling", () => {
    it("typing a character appends to empty value", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("a");
      expect(onChange).toHaveBeenCalledWith("a");
    });

    it("typing multiple characters builds value sequentially", () => {
      const { lastFrame, stdin } = render(<Wrapper />);
      stdin.write("h");
      stdin.write("i");
      const frame = lastFrame() ?? "";
      expect(frame).toContain("hi");
    });

    it("backspace deletes the character before the cursor", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper initialValue="hello" onChangeSpy={onChange} />);
      stdin.write("\x7f"); // DEL / backspace
      expect(onChange).toHaveBeenCalledWith("hell");
    });

    it("backspace removes from middle of string at cursor position", () => {
      // Type "abc", move left twice so cursor is after "a", then backspace
      const { lastFrame, stdin } = render(<Wrapper />);
      stdin.write("abc");
      stdin.write("\x1b[D"); // left
      stdin.write("\x1b[D"); // left — cursor now between "a" and "b"
      stdin.write("\x7f");   // backspace deletes "a"
      const frame = lastFrame() ?? "";
      expect(frame).toContain("bc");
      expect(frame).not.toContain("abc");
    });

    it("forward-delete removes character at cursor position", () => {
      const onChange = vi.fn();
      // Start with "hello", cursor at end. Move left once to sit on "o", then delete.
      const { stdin } = render(<Wrapper initialValue="hell" onChangeSpy={onChange} />);
      // Type one more char to get "hello" in state
      // Reset: use initialValue="hello" directly
      const onChange2 = vi.fn();
      const { stdin: stdin2 } = render(<Wrapper initialValue="hello" onChangeSpy={onChange2} />);
      stdin2.write("\x1b[D");   // move left — cursor on trailing space? No: cursor is at index 5 (end), move to 4 (on "o")
      stdin2.write("\x1b[3~"); // forward-delete at "o"
      expect(onChange2).toHaveBeenCalledWith("hell");
    });

    it("left arrow moves cursor left — character under cursor changes", () => {
      const { lastFrame, stdin } = render(<Wrapper initialValue="ab" />);
      // Initially cursor at end (index 2) — cursorChar is " " (trailing space)
      expect(cursorChar(lastFrame())).toBe(" ");
      stdin.write("\x1b[D"); // move left — cursor on "b" (index 1)
      expect(cursorChar(lastFrame())).toBe("b");
    });

    it("right arrow moves cursor right after left", () => {
      const { lastFrame, stdin } = render(<Wrapper initialValue="ab" />);
      stdin.write("\x1b[D"); // left — cursor on "b"
      stdin.write("\x1b[C"); // right — cursor back at end (" ")
      expect(cursorChar(lastFrame())).toBe(" ");
    });

    it("enter calls onSubmit", () => {
      const onSubmit = vi.fn();
      const { stdin } = render(<Wrapper onSubmit={onSubmit} />);
      stdin.write("\r");
      expect(onSubmit).toHaveBeenCalledTimes(1);
    });

    it("enter does not call onChange", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\r");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("up arrow is ignored — no onChange, no crash", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\x1b[A");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("down arrow is ignored — no onChange, no crash", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\x1b[B");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("ctrl+c is ignored — no onChange, no crash", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\x03"); // ctrl+c
      expect(onChange).not.toHaveBeenCalled();
    });

    it("tab is ignored — no onChange, no crash", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\t");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("paste (multi-char string) inserts all characters at cursor", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("paste");
      expect(onChange).toHaveBeenLastCalledWith("paste");
    });

    it("paste inserts in the middle of existing value at cursor position", () => {
      // Build "ac" then move left to sit between a and c, paste "b"
      const { lastFrame, stdin } = render(<Wrapper />);
      stdin.write("ac");
      stdin.write("\x1b[D"); // move left — cursor on "c"
      stdin.write("b");      // insert "b" before "c"
      const frame = lastFrame() ?? "";
      expect(frame).toContain("abc");
    });
  });

  // ---------------------------------------------------------------------------
  // Edge cases
  // ---------------------------------------------------------------------------

  describe("Edge cases", () => {
    it("backspace at position 0 does nothing", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper initialValue="hi" onChangeSpy={onChange} />);
      // Move cursor all the way to position 0
      stdin.write("\x1b[D");
      stdin.write("\x1b[D");
      onChange.mockClear();
      stdin.write("\x7f"); // backspace at position 0
      expect(onChange).not.toHaveBeenCalled();
    });

    it("forward-delete at end of string does nothing", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper initialValue="hi" onChangeSpy={onChange} />);
      // Cursor is already at the end
      stdin.write("\x1b[3~"); // forward-delete beyond end
      expect(onChange).not.toHaveBeenCalled();
    });

    it("left arrow at position 0 does nothing — no crash", () => {
      const onChange = vi.fn();
      const { lastFrame, stdin } = render(<Wrapper initialValue="a" onChangeSpy={onChange} />);
      stdin.write("\x1b[D"); // move to index 0
      onChange.mockClear();
      stdin.write("\x1b[D"); // cannot go further left
      expect(onChange).not.toHaveBeenCalled();
      // Frame should still show "a" with cursor on it
      expect(cursorChar(lastFrame())).toBe("a");
    });

    it("right arrow at end of string does nothing — no crash", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper initialValue="a" onChangeSpy={onChange} />);
      // Cursor is already at end
      stdin.write("\x1b[C"); // cannot go further right
      expect(onChange).not.toHaveBeenCalled();
    });

    it("cursor clamps when value is externally reset to empty string", () => {
      // Simulate parent resetting value="" after submit
      // We render with controlled value and replace the whole tree
      const onChange1 = vi.fn();
      // Build up "hello" in a wrapper first, then re-render with value=""
      const { lastFrame, rerender } = render(
        <TextInputCore value="hello" onChange={onChange1} focused />,
      );
      // Cursor is at index 5 (end of "hello")
      expect(hasCursor(lastFrame())).toBe(true);

      // Parent resets value to ""
      rerender(<TextInputCore value="" onChange={onChange1} focused />);
      // After reset, cursor should clamp to 0 — component shows blank cursor
      expect(hasCursor(lastFrame())).toBe(true);
      expect(cursorChar(lastFrame())).toBe(" ");
    });

    it("empty value with no placeholder shows single inverse space", () => {
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} focused />);
      expect(hasCursor(lastFrame())).toBe(true);
      expect(cursorChar(lastFrame())).toBe(" ");
    });

    it("onChange is not called when neither value nor cursor changes", () => {
      // Pressing a pass-through key like up arrow should not fire onChange
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      stdin.write("\x1b[A"); // up arrow — ignored
      expect(onChange).not.toHaveBeenCalled();
    });
  });

  // ---------------------------------------------------------------------------
  // Props
  // ---------------------------------------------------------------------------

  describe("Props", () => {
    it("focused=false prevents keystroke capture", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper focused={false} onChangeSpy={onChange} />);
      stdin.write("z");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("focused=false means no cursor in frame", () => {
      const { lastFrame } = render(
        <TextInputCore value="test" onChange={vi.fn()} focused={false} />,
      );
      expect(hasCursor(lastFrame())).toBe(false);
    });

    it("onChange only called when value actually changes", () => {
      const onChange = vi.fn();
      const { stdin } = render(<Wrapper onChangeSpy={onChange} />);
      // Arrow keys move cursor but do not change value — onChange should not fire
      stdin.write("hi");
      onChange.mockClear();
      stdin.write("\x1b[D"); // left — cursor move only
      expect(onChange).not.toHaveBeenCalled();
    });

    it("onSubmit is called on Enter key", () => {
      const onSubmit = vi.fn();
      const { stdin } = render(<Wrapper onSubmit={onSubmit} />);
      stdin.write("\r");
      expect(onSubmit).toHaveBeenCalledTimes(1);
    });

    it("onSubmit is not called on regular character input", () => {
      const onSubmit = vi.fn();
      const { stdin } = render(<Wrapper onSubmit={onSubmit} />);
      stdin.write("abc");
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it("onSubmit is optional — enter with no handler does not throw", () => {
      const { stdin } = render(
        <TextInputCore value="" onChange={vi.fn()} focused />,
      );
      expect(() => stdin.write("\r")).not.toThrow();
    });

    it("placeholder defaults to empty string when not provided", () => {
      // Focused + empty value + no placeholder → renders inverse space cursor
      const { lastFrame } = render(<TextInputCore value="" onChange={vi.fn()} focused />);
      expect(hasCursor(lastFrame())).toBe(true);
    });

    it("focused defaults to true when not provided", () => {
      const { lastFrame } = render(<TextInputCore value="hi" onChange={vi.fn()} />);
      // Default focused=true means cursor should be visible
      expect(hasCursor(lastFrame())).toBe(true);
    });
  });
});
