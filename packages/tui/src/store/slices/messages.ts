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

      messages[lastIndex] = {
        ...lastMessage,
        content,
        partial: false, // Mark as complete when updating
      };

      return { messages };
    });
  },

  clearMessages: (): void => {
    set({ messages: [] });
  },
});
