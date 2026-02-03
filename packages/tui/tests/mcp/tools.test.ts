/**
 * Unit tests for MCP tools
 * Tests integration with modal queue system
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { useStore } from "../../src/store/index.js";
import { askUserTool, askUserSchema } from "../../src/mcp/tools/askUser.js";
import { confirmActionTool, confirmActionSchema } from "../../src/mcp/tools/confirmAction.js";
import { requestInputTool, requestInputSchema } from "../../src/mcp/tools/requestInput.js";
import { selectOptionTool, selectOptionSchema } from "../../src/mcp/tools/selectOption.js";
import type { ModalResponse } from "../../src/store/slices/modal.js";

describe("MCP Tools", () => {
  beforeEach(() => {
    // Reset store before each test
    useStore.setState({
      modalQueue: [],
    });
  });

  describe("askUserTool", () => {
    it("should enqueue ask modal and return response value", async () => {
      // Simulate user response
      const promise = askUserTool.handler({
        message: "What is your name?",
      });

      // Wait for modal to be enqueued
      await new Promise((resolve) => setTimeout(resolve, 0));

      // Get the enqueued modal
      const queue = useStore.getState().modalQueue;
      expect(queue).toHaveLength(1);
      expect(queue[0]?.type).toBe("ask");
      expect(queue[0]?.payload).toEqual({
        message: "What is your name?",
        options: undefined,
        defaultValue: undefined,
      });

      // Simulate user response
      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "ask",
        value: "Alice",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("Alice");
    });

    it("should pass options to modal", async () => {
      const promise = askUserTool.handler({
        message: "Choose a color",
        options: ["red", "green", "blue"],
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue[0]?.payload).toEqual({
        message: "Choose a color",
        options: ["red", "green", "blue"],
        defaultValue: undefined,
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "ask",
        value: "red",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("red");
    });

    it("should pass default value to modal", async () => {
      const promise = askUserTool.handler({
        message: "Enter your age",
        default: "25",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue[0]?.payload).toEqual({
        message: "Enter your age",
        options: undefined,
        defaultValue: "25",
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "ask",
        value: "30",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("30");
    });
  });

  describe("confirmActionTool", () => {
    it("should enqueue confirm modal and return response", async () => {
      const promise = confirmActionTool.handler({
        action: "Delete all files",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue).toHaveLength(1);
      expect(queue[0]?.type).toBe("confirm");
      expect(queue[0]?.payload).toEqual({
        action: "Delete all files",
        destructive: false,
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "confirm",
        confirmed: true,
        cancelled: false,
      });

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        confirmed: true,
        cancelled: false,
      });
    });

    it("should pass destructive flag to modal", async () => {
      const promise = confirmActionTool.handler({
        action: "Drop database",
        destructive: true,
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue[0]?.payload).toEqual({
        action: "Drop database",
        destructive: true,
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "confirm",
        confirmed: false,
        cancelled: true,
      });

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        confirmed: false,
        cancelled: true,
      });
    });
  });

  describe("requestInputTool", () => {
    it("should enqueue input modal and return response value", async () => {
      const promise = requestInputTool.handler({
        prompt: "Enter your email",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue).toHaveLength(1);
      expect(queue[0]?.type).toBe("input");
      expect(queue[0]?.payload).toEqual({
        prompt: "Enter your email",
        placeholder: undefined,
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "input",
        value: "user@example.com",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("user@example.com");
    });

    it("should pass placeholder to modal", async () => {
      const promise = requestInputTool.handler({
        prompt: "Enter API key",
        placeholder: "sk-...",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue[0]?.payload).toEqual({
        prompt: "Enter API key",
        placeholder: "sk-...",
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "input",
        value: "sk-test123",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("sk-test123");
    });
  });

  describe("selectOptionTool", () => {
    it("should enqueue select modal and return response", async () => {
      const promise = selectOptionTool.handler({
        message: "Select your framework",
        options: [
          { label: "React", value: "react" },
          { label: "Vue", value: "vue" },
          { label: "Angular", value: "angular" },
        ],
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const queue = useStore.getState().modalQueue;
      expect(queue).toHaveLength(1);
      expect(queue[0]?.type).toBe("select");
      expect(queue[0]?.payload).toEqual({
        message: "Select your framework",
        options: [
          { label: "React", value: "react" },
          { label: "Vue", value: "vue" },
          { label: "Angular", value: "angular" },
        ],
      });

      const modalId = queue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "select",
        selected: "react",
        index: 0,
      });

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        selected: "react",
        index: 0,
      });
    });

    it("should handle selection of different indices", async () => {
      const promise = selectOptionTool.handler({
        message: "Pick a number",
        options: [
          { label: "One", value: "1" },
          { label: "Two", value: "2" },
          { label: "Three", value: "3" },
        ],
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const modalId = useStore.getState().modalQueue[0]?.id ?? "";
      useStore.getState().dequeue(modalId, {
        type: "select",
        selected: "3",
        index: 2,
      });

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        selected: "3",
        index: 2,
      });
    });
  });

  describe("Argument Validation", () => {
    describe("selectOptionTool", () => {
      it("should reject empty options array", () => {
        expect(() =>
          selectOptionSchema.parse({
            message: "Select something",
            options: [],
          })
        ).toThrow("Options array must contain at least one option");
      });

      it("should reject empty message", () => {
        expect(() =>
          selectOptionSchema.parse({
            message: "",
            options: [{ label: "Option 1", value: "1" }],
          })
        ).toThrow("Message must not be empty");
      });

      it("should reject empty option label", () => {
        expect(() =>
          selectOptionSchema.parse({
            message: "Select something",
            options: [{ label: "", value: "1" }],
          })
        ).toThrow("Option label must not be empty");
      });
    });

    describe("askUserTool", () => {
      it("should reject empty message", () => {
        expect(() =>
          askUserSchema.parse({
            message: "",
          })
        ).toThrow("Message must not be empty");
      });

      it("should reject empty option in options array", () => {
        expect(() =>
          askUserSchema.parse({
            message: "Choose",
            options: ["valid", ""],
          })
        ).toThrow("Option must not be empty");
      });
    });

    describe("requestInputTool", () => {
      it("should reject empty prompt", () => {
        expect(() =>
          requestInputSchema.parse({
            prompt: "",
          })
        ).toThrow("Prompt must not be empty");
      });
    });

    describe("confirmActionTool", () => {
      it("should reject empty action", () => {
        expect(() =>
          confirmActionSchema.parse({
            action: "",
          })
        ).toThrow("Action description must not be empty");
      });
    });
  });

  describe("MCP Response Format", () => {
    it("all tools should return content array with text type", async () => {
      const promises = [
        askUserTool.handler({ message: "test" }),
        confirmActionTool.handler({ action: "test" }),
        requestInputTool.handler({ prompt: "test" }),
        selectOptionTool.handler({
          message: "test",
          options: [{ label: "A", value: "a" }],
        }),
      ];

      // Trigger all promises
      await new Promise((resolve) => setTimeout(resolve, 0));

      // Respond to all modals
      const queue = useStore.getState().modalQueue;
      queue.forEach((modal) => {
        let response: ModalResponse;
        switch (modal.type) {
          case "ask":
            response = { type: "ask", value: "test" };
            break;
          case "confirm":
            response = { type: "confirm", confirmed: true, cancelled: false };
            break;
          case "input":
            response = { type: "input", value: "test" };
            break;
          case "select":
            response = { type: "select", selected: "a", index: 0 };
            break;
        }
        useStore.getState().dequeue(modal.id, response);
      });

      const results = await Promise.all(promises);

      results.forEach((result) => {
        expect(result).toHaveProperty("content");
        expect(Array.isArray(result.content)).toBe(true);
        expect(result.content).toHaveLength(1);
        expect(result.content[0]).toHaveProperty("type", "text");
        expect(result.content[0]).toHaveProperty("text");
        expect(typeof result.content[0]?.text).toBe("string");
      });
    });
  });
});
