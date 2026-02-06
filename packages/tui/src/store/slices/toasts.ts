/**
 * Toast notification slice
 * Manages temporary notification messages with auto-dismiss
 */

import type { StateCreator } from "zustand";
import type { Store, ToastSlice } from "../types.js";
import { nanoid } from "nanoid";

export const createToastSlice: StateCreator<Store, [], [], ToastSlice> = (set) => ({
  toasts: [],

  addToast: (message, type = "info"): void => {
    const id = nanoid();
    set((state) => ({
      // Keep max 3 toasts
      toasts: [...state.toasts.slice(-2), { id, message, type, createdAt: Date.now() }],
    }));
    // Auto-remove after 3 seconds
    setTimeout(() => {
      set((state) => ({
        toasts: state.toasts.filter((t) => t.id !== id),
      }));
    }, 3000);
  },

  removeToast: (id): void => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },
});
