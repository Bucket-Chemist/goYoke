/**
 * Messages slice for Zustand store
 * Handles conversation message state with partial update support
 */

import type { StateCreator } from "zustand";
import type { Store, MessagesSlice } from "../types.js";
import { nanoid } from "nanoid";

export const createMessagesSlice: StateCreator<
  Store,
  [],
  [],
  MessagesSlice
> = (set) => ({
  messages: [],

  addMessage: (msg): void => {
    set((state) => ({
      messages: [
        ...state.messages,
        {
          ...msg,
          id: nanoid(),
          timestamp: Date.now(),
        },
      ],
    }));
  },

  updateLastMessage: (content): void => {
    set((state) => {
      if (state.messages.length === 0) {
        return state;
      }

      // Update in-place to prevent flicker
      const messages = [...state.messages];
      const lastIndex = messages.length - 1;
      const lastMessage = messages[lastIndex];

      if (!lastMessage) {
        return state;
      }

      // Preserve existing text blocks if new content has none.
      // When the SDK fires an event with only tool_use blocks (between tool calls),
      // the text block from Claude's prior streaming would otherwise be lost.
      const newHasText = content.some((b) => b.type === "text");
      const existingTextBlocks = lastMessage.content.filter((b) => b.type === "text");
      const mergedContent =
        !newHasText && existingTextBlocks.length > 0
          ? [...existingTextBlocks, ...content]
          : content;

      messages[lastIndex] = {
        ...lastMessage,
        content: mergedContent,
        partial: false, // Mark as complete when updating
      };

      return { messages };
    });
  },

  clearMessages: (): void => {
    set({ messages: [] });
  },
});
