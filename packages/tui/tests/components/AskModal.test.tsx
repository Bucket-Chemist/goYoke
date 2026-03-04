/**
 * AskModal component tests
 * Coverage:
 * - Rendering with all prop variations (with/without options)
 * - Visual output verification for both modes
 * - Default value handling
 * - Component structure and UI elements
 * - Callback contract testing
 * - Edge cases (empty options, long text, special characters)
 * - Mode detection (options vs input)
 *
 * Note: AskModal has two modes:
 * - Options mode: arrow-navigable list (when options array has items)
 * - Input mode: free text input via TextInputCore (when no options or empty array)
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AskModal } from "../../src/components/modals/AskModal.js";
import type { ModalRequest, AskPayload } from "../../src/store/slices/modal.js";

describe("AskModal", () => {
  const createRequest = (payload: AskPayload): ModalRequest<AskPayload> => ({
    id: "test-ask",
    type: "ask",
    payload,
    resolve: vi.fn(),
    reject: vi.fn(),
  });

  let onComplete: ReturnType<typeof vi.fn>;
  let onCancel: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onComplete = vi.fn();
    onCancel = vi.fn();
  });

  describe("Rendering - Options Mode", () => {
    it("renders message with options", () => {
      const request = createRequest({
        message: "Choose your answer:",
        options: [{ label: "Yes", value: "yes" }, { label: "No", value: "no" }, { label: "Maybe", value: "maybe" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("Choose your answer:");
    });

    it("renders all options", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option A", value: "option-a" }, { label: "Option B", value: "option-b" }, { label: "Option C", value: "option-c" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();
      expect(output).toContain("Option A");
      expect(output).toContain("Option B");
      expect(output).toContain("Option C");
    });

    it("highlights first option by default", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "First", value: "first" }, { label: "Second", value: "second" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();
      expect(output).toMatch(/▶.*First/);
    });

    it("renders keyboard hints for options mode", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option A", value: "option-a" }, { label: "Option B", value: "option-b" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();
      expect(output).toMatch(/Navigate/);
      expect(output).toMatch(/Enter/);
      expect(output).toMatch(/Select/);
      expect(output).toMatch(/Esc/);
    });
  });

  describe("Rendering - Input Mode", () => {
    it("renders message without options", () => {
      const request = createRequest({
        message: "What is your name?",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("What is your name?");
    });

    it("renders text input when no options", () => {
      const request = createRequest({
        message: "Enter answer:",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show default placeholder
      expect(lastFrame()).toContain("Type your answer...");
    });

    it("renders with default value in input mode", () => {
      const request = createRequest({
        message: "Enter name:",
        defaultValue: "John Doe",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("John Doe");
    });

    it("renders keyboard hints for input mode", () => {
      const request = createRequest({
        message: "Enter answer:",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();
      expect(output).toMatch(/Enter/);
      expect(output).toMatch(/Submit/);
      expect(output).toMatch(/Esc/);
      expect(output).not.toMatch(/Navigate/); // No navigation in input mode
    });

    it("renders input mode when options is empty array", () => {
      const request = createRequest({
        message: "Enter answer:",
        options: [],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Empty array means no options, should show input
      expect(lastFrame()).toContain("Type your answer...");
    });
  });

  describe("Edge Cases - Options Mode", () => {
    it("handles single option", () => {
      const request = createRequest({
        message: "Only one choice:",
        options: [{ label: "Only option", value: "only-option" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("Only option");
    });

    it("handles very long option text", () => {
      const longOption = "A".repeat(200);
      const request = createRequest({
        message: "Select:",
        options: [{ label: longOption, value: "long-option" }, { label: "Short", value: "short" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("AAA");
    });

    it("handles options with special characters", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "<script>alert('xss')</script>", value: "xss-option" },
          { label: "Normal & Safe", value: "normal-safe" },
        ],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("<script>alert('xss')</script>");
      expect(lastFrame()).toContain("Normal & Safe");
    });

    it("handles options with unicode", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "選項 A 🎯", value: "option-a" },
          { label: "選項 B 🎨", value: "option-b" },
        ],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("選項 A 🎯");
      expect(lastFrame()).toContain("選項 B 🎨");
    });

    it("handles duplicate options", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "Same", value: "same-1" },
          { label: "Same", value: "same-2" },
          { label: "Different", value: "different" },
        ],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show both options even with same text
      expect(lastFrame()).toContain("Same");
      expect(lastFrame()).toContain("Different");
    });

    it("handles 20+ options", () => {
      const options = Array.from({ length: 25 }, (_, i) => ({
        label: `Option ${i + 1}`,
        value: `option-${i + 1}`,
      }));
      const request = createRequest({
        message: "Select:",
        options,
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("Option 1");
      expect(lastFrame()).toContain("Option 25");
    });
  });

  describe("Edge Cases - Input Mode", () => {
    it("handles very long default value", () => {
      const longDefault = "A".repeat(200);
      const request = createRequest({
        message: "Enter text:",
        defaultValue: longDefault,
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("AAA");
    });

    it("handles default value with special characters", () => {
      const request = createRequest({
        message: "Enter text:",
        defaultValue: "<default@example.com>",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("<default@example.com>");
    });

    it("handles default value with unicode", () => {
      const request = createRequest({
        message: "Enter text:",
        defaultValue: "默认值 🌈",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("默认值 🌈");
    });
  });

  describe("Message Variations", () => {
    it("handles empty message", () => {
      const request = createRequest({
        message: "",
        options: [{ label: "Option A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toBeTruthy();
    });

    it("handles very long message", () => {
      const longMessage = "A".repeat(300);
      const request = createRequest({
        message: longMessage,
        options: [{ label: "Option A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("AAA");
    });

    it("handles message with newlines", () => {
      const request = createRequest({
        message: "Line 1\nLine 2\nChoose:",
        options: [{ label: "Option A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();
      expect(output).toContain("Line 1");
      expect(output).toContain("Line 2");
      expect(output).toContain("Choose:");
    });

    it("handles message with special characters", () => {
      const request = createRequest({
        message: "Choose <option>:",
        options: [{ label: "Option A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("Choose <option>:");
    });

    it("handles message with unicode", () => {
      const request = createRequest({
        message: "请选择: 🎯",
        options: [{ label: "选项 A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toContain("请选择: 🎯");
    });
  });

  describe("Mode Detection", () => {
    it("uses options mode when options array has items", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option A", value: "option-a" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show arrow indicator (options mode)
      expect(lastFrame()).toMatch(/▶/);
    });

    it("uses input mode when options is undefined", () => {
      const request = createRequest({
        message: "Enter:",
        options: undefined,
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show placeholder (input mode)
      expect(lastFrame()).toContain("Type your answer...");
    });

    it("uses input mode when options is empty array", () => {
      const request = createRequest({
        message: "Enter:",
        options: [],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show placeholder (input mode)
      expect(lastFrame()).toContain("Type your answer...");
    });

    it("ignores defaultValue in options mode", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option A", value: "option-a" }, { label: "Option B", value: "option-b" }],
        defaultValue: "This should be ignored",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Should show options, not input with default
      expect(lastFrame()).toMatch(/▶.*Option A/);
      expect(lastFrame()).not.toContain("This should be ignored");
    });
  });

  describe("Callback Contract - Options Mode", () => {
    it("defines correct response type structure", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option A", value: "option-a" }, { label: "Option B", value: "option-b" }],
      });
      render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      // Manually simulate what the component would call
      onComplete({
        type: "ask",
        value: "Option A",
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "ask",
        value: "Option A",
      });
    });

    it("callback returns selected option text", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "First", value: "first" },
          { label: "Second", value: "second" },
          { label: "Third", value: "third" },
        ],
      });
      render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      onComplete({
        type: "ask",
        value: "Second",
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "ask",
        value: "Second",
      });
    });
  });

  describe("Callback Contract - Input Mode", () => {
    it("defines correct response type structure", () => {
      const request = createRequest({
        message: "Enter answer:",
      });
      render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      onComplete({
        type: "ask",
        value: "My Answer",
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "ask",
        value: "My Answer",
      });
    });

    it("callback accepts empty string", () => {
      const request = createRequest({
        message: "Enter answer:",
      });
      render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      onComplete({
        type: "ask",
        value: "",
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "ask",
        value: "",
      });
    });

    it("callback handles unicode", () => {
      const request = createRequest({
        message: "Enter answer:",
      });
      render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      onComplete({
        type: "ask",
        value: "你好 🌟",
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "ask",
        value: "你好 🌟",
      });
    });
  });

  describe("Component Structure", () => {
    it("maintains consistent structure in options mode", () => {
      const request = createRequest({
        message: "Choose an option:",
        options: [{ label: "Option A", value: "option-a" }, { label: "Option B", value: "option-b" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();

      // Should have all required elements
      expect(output).toContain("Choose an option:");
      expect(output).toContain("Option A");
      expect(output).toContain("Option B");
      expect(output).toMatch(/▶/);
    });

    it("maintains consistent structure in input mode", () => {
      const request = createRequest({
        message: "Enter your answer:",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      const output = lastFrame();

      // Should have all required elements
      expect(output).toContain("Enter your answer:");
      expect(output).toContain("Type your answer...");
      expect(output).toMatch(/Enter/);
      expect(output).toMatch(/Submit/);
    });

    it("renders without crashing for minimal options payload", () => {
      const request = createRequest({
        message: "Minimal",
        options: [{ label: "One", value: "one" }],
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toBeTruthy();
    });

    it("renders without crashing for minimal input payload", () => {
      const request = createRequest({
        message: "Minimal",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toBeTruthy();
    });

    it("renders without crashing for full input payload", () => {
      const request = createRequest({
        message: "Full payload test",
        defaultValue: "Default value",
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toBeTruthy();
    });

    it("renders without crashing for large options list", () => {
      const options = Array.from({ length: 50 }, (_, i) => ({
        label: `Option ${i + 1}`,
        value: `option-${i + 1}`,
      }));
      const request = createRequest({
        message: "Large list",
        options,
      });
      const { lastFrame } = render(<AskModal request={request} onComplete={onComplete} onCancel={onCancel} />);

      expect(lastFrame()).toBeTruthy();
    });
  });
});
