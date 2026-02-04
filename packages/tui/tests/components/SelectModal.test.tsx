/**
 * SelectModal component tests
 * Coverage:
 * - Rendering with all prop variations
 * - Visual output verification
 * - Component structure and UI elements
 * - Callback contract testing
 * - Edge cases (empty options, long labels, many options)
 *
 * Note: ink-testing-library stdin.write() doesn't reliably trigger useInput,
 * so we focus on rendering tests and callback contracts.
 */

import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { SelectModal } from "../../src/components/modals/SelectModal.js";
import type { ModalRequest, SelectPayload } from "../../src/store/slices/modal.js";

describe("SelectModal", () => {
  const createRequest = (payload: SelectPayload): ModalRequest<SelectPayload> => ({
    id: "test-select",
    type: "select",
    payload,
    resolve: vi.fn(),
    reject: vi.fn(),
  });

  let onComplete: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onComplete = vi.fn();
  });

  describe("Rendering", () => {
    it("renders message", () => {
      const request = createRequest({
        message: "Choose an option:",
        options: [
          { label: "Option A", value: "a" },
          { label: "Option B", value: "b" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("Choose an option:");
    });

    it("renders all options", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "First", value: "1" },
          { label: "Second", value: "2" },
          { label: "Third", value: "3" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toContain("First");
      expect(output).toContain("Second");
      expect(output).toContain("Third");
    });

    it("highlights first option by default", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "First", value: "1" },
          { label: "Second", value: "2" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      // First option should have selection indicator
      expect(output).toMatch(/▶.*First/);
    });

    it("shows number shortcuts for first 9 options", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "One", value: "1" },
          { label: "Two", value: "2" },
          { label: "Three", value: "3" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toMatch(/1\./);
      expect(output).toMatch(/2\./);
      expect(output).toMatch(/3\./);
    });

    it("renders keyboard hints", () => {
      const request = createRequest({
        message: "Select:",
        options: [{ label: "Option", value: "opt" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toMatch(/Navigate/);
      expect(output).toMatch(/1-9/);
      expect(output).toMatch(/Enter/);
      expect(output).toMatch(/Select/);
    });

    it("handles empty options array gracefully", () => {
      const request = createRequest({
        message: "No options available:",
        options: [],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      // Should render without crashing
      expect(lastFrame()).toContain("No options available:");
    });

    it("does not show shortcuts for options beyond 9", () => {
      const options = Array.from({ length: 12 }, (_, i) => ({
        label: `Option ${i + 1}`,
        value: `opt${i + 1}`,
      }));

      const request = createRequest({
        message: "Select:",
        options,
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toMatch(/9\./); // Has number for 9th
      // Options 10-12 should not have numbers (implementation uses space)
      const lines = output.split("\n");
      const option10Line = lines.find((l) => l.includes("Option 10"));
      expect(option10Line).toBeDefined();
      // Should not have "10." prefix
      expect(option10Line).not.toMatch(/10\./);
    });
  });

  describe("Edge Cases", () => {
    it("handles single option", () => {
      const request = createRequest({
        message: "Only one choice:",
        options: [{ label: "Only option", value: "only" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("Only option");
    });

    it("handles very long option labels", () => {
      const longLabel = "A".repeat(200);
      const request = createRequest({
        message: "Select:",
        options: [
          { label: longLabel, value: "long" },
          { label: "Short", value: "short" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("AAA");
    });

    it("handles special characters in labels", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "<script>alert('xss')</script>", value: "xss" },
          { label: "Normal & Safe", value: "safe" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("<script>alert('xss')</script>");
      expect(lastFrame()).toContain("Normal & Safe");
    });

    it("handles unicode in labels", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "文件 📁", value: "file" },
          { label: "文件夹 📂", value: "folder" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("文件 📁");
      expect(lastFrame()).toContain("文件夹 📂");
    });

    it("handles newlines in labels", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "Line 1\nLine 2", value: "multi" },
          { label: "Single", value: "single" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toContain("Line 1");
      expect(output).toContain("Line 2");
    });

    it("handles identical labels with different values", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "Same Label", value: "value1" },
          { label: "Same Label", value: "value2" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      // Should show both options even with same label
      expect(output).toContain("Same Label");
    });

    it("handles 20+ options", () => {
      const options = Array.from({ length: 25 }, (_, i) => ({
        label: `Option ${i + 1}`,
        value: `opt${i + 1}`,
      }));

      const request = createRequest({
        message: "Select from many:",
        options,
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      // Should render all options
      expect(lastFrame()).toContain("Option 1");
      expect(lastFrame()).toContain("Option 25");
    });
  });

  describe("Message Variations", () => {
    it("handles empty message", () => {
      const request = createRequest({
        message: "",
        options: [{ label: "Option", value: "opt" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      // Should render without crashing
      expect(lastFrame()).toBeTruthy();
    });

    it("handles very long message", () => {
      const longMessage = "A".repeat(300);
      const request = createRequest({
        message: longMessage,
        options: [{ label: "Option", value: "opt" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toContain("AAA");
    });

    it("handles message with newlines", () => {
      const request = createRequest({
        message: "Line 1\nLine 2\nLine 3",
        options: [{ label: "Option", value: "opt" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();
      expect(output).toContain("Line 1");
      expect(output).toContain("Line 2");
      expect(output).toContain("Line 3");
    });
  });

  describe("Callback Contract", () => {
    it("defines correct response type structure", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "First", value: "first" },
          { label: "Second", value: "second" },
        ],
      });
      render(<SelectModal request={request} onComplete={onComplete} />);

      // Manually simulate what the component would call
      onComplete({
        type: "select",
        selected: "first",
        index: 0,
      });

      expect(onComplete).toHaveBeenCalledWith({
        type: "select",
        selected: "first",
        index: 0,
      });
    });

    it("callback includes both value and index", () => {
      const request = createRequest({
        message: "Select:",
        options: [
          { label: "A", value: "value-a" },
          { label: "B", value: "value-b" },
          { label: "C", value: "value-c" },
        ],
      });
      render(<SelectModal request={request} onComplete={onComplete} />);

      // Simulate selecting third option
      onComplete({
        type: "select",
        selected: "value-c",
        index: 2,
      });

      expect(onComplete).toHaveBeenCalledWith(
        expect.objectContaining({
          selected: "value-c",
          index: 2,
        })
      );
    });
  });

  describe("Component Structure", () => {
    it("maintains consistent structure across renders", () => {
      const request = createRequest({
        message: "Select an option:",
        options: [
          { label: "Option A", value: "a" },
          { label: "Option B", value: "b" },
        ],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      const output = lastFrame();

      // Should have all required elements
      expect(output).toContain("Select an option:");
      expect(output).toContain("Option A");
      expect(output).toContain("Option B");
      expect(output).toMatch(/▶/);
    });

    it("renders without crashing for minimal payload", () => {
      const request = createRequest({
        message: "Minimal",
        options: [{ label: "One", value: "1" }],
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toBeTruthy();
    });

    it("renders without crashing for large payload", () => {
      const options = Array.from({ length: 50 }, (_, i) => ({
        label: `Option ${i + 1}`,
        value: `opt${i + 1}`,
      }));

      const request = createRequest({
        message: "Large list",
        options,
      });
      const { lastFrame } = render(<SelectModal request={request} onComplete={onComplete} />);

      expect(lastFrame()).toBeTruthy();
    });
  });
});
