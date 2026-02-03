/**
 * MCP Integration Tests
 * Tests end-to-end flow: tool handler → modal queue → user response → tool return
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store/index.js";
import { askUserTool } from "../../src/mcp/tools/askUser.js";
import { confirmActionTool } from "../../src/mcp/tools/confirmAction.js";
import { requestInputTool } from "../../src/mcp/tools/requestInput.js";
import { selectOptionTool } from "../../src/mcp/tools/selectOption.js";
import type { ModalResponse } from "../../src/store/slices/modal.js";

/**
 * Helper to simulate user response after modal is enqueued
 */
async function simulateUserResponse(
  response: ModalResponse,
  delay = 0
): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, delay));
  const queue = useStore.getState().modalQueue;
  const modal = queue[queue.length - 1];
  if (modal) {
    useStore.getState().dequeue(modal.id, response);
  }
}

describe("MCP Integration - Basic Tool Flow", () => {
  beforeEach(() => {
    useStore.setState({
      modalQueue: [],
    });
  });

  describe("ask_user tool", () => {
    it("should complete full flow: tool call → modal → user response → return value", async () => {
      // Start tool call
      const promise = askUserTool.handler({
        message: "What is your favorite color?",
      });

      // Simulate user responding after modal is enqueued
      setTimeout(() => {
        simulateUserResponse({
          type: "ask",
          value: "Blue",
        });
      }, 10);

      // Tool should return user's response
      const result = await promise;
      expect(result.content[0]?.text).toBe("Blue");

      // Queue should be empty after completion
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should handle options flow correctly", async () => {
      const promise = askUserTool.handler({
        message: "Choose a size",
        options: ["Small", "Medium", "Large"],
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      // Verify modal has options
      const modal = useStore.getState().modalQueue[0];
      expect(modal?.payload).toEqual({
        message: "Choose a size",
        options: ["Small", "Medium", "Large"],
        defaultValue: undefined,
      });

      // User selects option
      await simulateUserResponse({
        type: "ask",
        value: "Large",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("Large");
    });

    it("should handle default value correctly", async () => {
      const promise = askUserTool.handler({
        message: "Enter your username",
        default: "guest",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const modal = useStore.getState().modalQueue[0];
      expect(modal?.payload).toHaveProperty("defaultValue", "guest");

      await simulateUserResponse({
        type: "ask",
        value: "alice",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("alice");
    });
  });

  describe("confirm_action tool", () => {
    it("should complete confirmation flow with YES", async () => {
      const promise = confirmActionTool.handler({
        action: "Delete file config.yaml",
      });

      setTimeout(() => {
        simulateUserResponse({
          type: "confirm",
          confirmed: true,
          cancelled: false,
        });
      }, 10);

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        confirmed: true,
        cancelled: false,
      });

      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should complete confirmation flow with NO", async () => {
      const promise = confirmActionTool.handler({
        action: "Reset database",
      });

      setTimeout(() => {
        simulateUserResponse({
          type: "confirm",
          confirmed: false,
          cancelled: false,
        });
      }, 10);

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        confirmed: false,
        cancelled: false,
      });
    });

    it("should handle user cancellation (Escape)", async () => {
      const promise = confirmActionTool.handler({
        action: "Drop table users",
        destructive: true,
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      // Verify destructive flag passed
      const modal = useStore.getState().modalQueue[0];
      expect(modal?.payload).toEqual({
        action: "Drop table users",
        destructive: true,
      });

      // User cancels
      await simulateUserResponse({
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

  describe("request_input tool", () => {
    it("should complete input flow correctly", async () => {
      const promise = requestInputTool.handler({
        prompt: "Enter your API key",
      });

      setTimeout(() => {
        simulateUserResponse({
          type: "input",
          value: "sk-test-123456",
        });
      }, 10);

      const result = await promise;
      expect(result.content[0]?.text).toBe("sk-test-123456");
      expect(useStore.getState().modalQueue).toHaveLength(0);
    });

    it("should pass placeholder to modal", async () => {
      const promise = requestInputTool.handler({
        prompt: "Enter email",
        placeholder: "user@example.com",
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      const modal = useStore.getState().modalQueue[0];
      expect(modal?.payload).toEqual({
        prompt: "Enter email",
        placeholder: "user@example.com",
      });

      await simulateUserResponse({
        type: "input",
        value: "alice@test.com",
      });

      const result = await promise;
      expect(result.content[0]?.text).toBe("alice@test.com");
    });

    it("should handle empty input", async () => {
      const promise = requestInputTool.handler({
        prompt: "Optional comment",
      });

      setTimeout(() => {
        simulateUserResponse({
          type: "input",
          value: "",
        });
      }, 10);

      const result = await promise;
      expect(result.content[0]?.text).toBe("");
    });
  });

  describe("select_option tool", () => {
    it("should complete selection flow correctly", async () => {
      const promise = selectOptionTool.handler({
        message: "Select your framework",
        options: [
          { label: "React", value: "react" },
          { label: "Vue", value: "vue" },
          { label: "Svelte", value: "svelte" },
        ],
      });

      await new Promise((resolve) => setTimeout(resolve, 0));

      // Verify modal payload
      const modal = useStore.getState().modalQueue[0];
      expect(modal?.payload).toEqual({
        message: "Select your framework",
        options: [
          { label: "React", value: "react" },
          { label: "Vue", value: "vue" },
          { label: "Svelte", value: "svelte" },
        ],
      });

      // User selects first option
      await simulateUserResponse({
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

      expect(useStore.getState().modalQueue).toHaveLength(0);
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

      setTimeout(() => {
        simulateUserResponse({
          type: "select",
          selected: "3",
          index: 2,
        });
      }, 10);

      const result = await promise;
      const response = JSON.parse(result.content[0]?.text ?? "{}");
      expect(response).toEqual({
        selected: "3",
        index: 2,
      });
    });
  });

  describe("MCP response format validation", () => {
    it("all tools should return MCP-compliant response format", async () => {
      const promises = [
        askUserTool.handler({ message: "Test ask" }),
        confirmActionTool.handler({ action: "Test confirm" }),
        requestInputTool.handler({ prompt: "Test input" }),
        selectOptionTool.handler({
          message: "Test select",
          options: [{ label: "A", value: "a" }],
        }),
      ];

      // Respond to all
      await new Promise((resolve) => setTimeout(resolve, 0));
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

      // All should have MCP-compliant format
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
