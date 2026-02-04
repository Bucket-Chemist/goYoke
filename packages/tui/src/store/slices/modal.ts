/**
 * Modal slice for Zustand store
 * Implements Promise-based modal queue with FIFO ordering
 */

import type { StateCreator } from "zustand";
import type { Store } from "../types.js";
import { nanoid } from "nanoid";

// Payload types for different modal variants
export interface AskPayload {
  message: string;
  options?: Array<{ label: string; value: string }>;
  defaultValue?: string;
}

export interface ConfirmPayload {
  action: string;
  destructive?: boolean;
}

export interface InputPayload {
  prompt: string;
  placeholder?: string;
}

export interface SelectPayload {
  message: string;
  options: Array<{ label: string; value: string }>;
}

// Discriminated union for modal responses
export type ModalResponse =
  | { type: "ask"; value: string }
  | { type: "confirm"; confirmed: boolean; cancelled: boolean }
  | { type: "input"; value: string }
  | { type: "select"; selected: string; index: number };

// Internal request structure with Promise handlers
export interface ModalRequest<T = unknown> {
  id: string;
  type: "ask" | "confirm" | "input" | "select";
  payload: T;
  resolve: (response: ModalResponse) => void;
  reject: (error: Error) => void;
  timeout?: number;
  timeoutId?: NodeJS.Timeout;
}

// Public API for enqueuing modals (omits internal fields)
export type EnqueueRequest<T = unknown> = Omit<
  ModalRequest<T>,
  "id" | "resolve" | "reject" | "timeoutId"
>;

// Modal slice interface
export interface ModalSlice {
  modalQueue: ModalRequest[];
  enqueue: <T>(request: EnqueueRequest<T>) => Promise<ModalResponse>;
  dequeue: (id: string, response: ModalResponse) => void;
  cancel: (id: string) => void;
}

export const createModalSlice: StateCreator<Store, [], [], ModalSlice> = (
  set,
  get
) => ({
  modalQueue: [],

  enqueue: <T>(request: EnqueueRequest<T>): Promise<ModalResponse> => {
    return new Promise<ModalResponse>((resolve, reject) => {
      const id = nanoid();
      let timeoutId: NodeJS.Timeout | undefined;
      let completed = false;

      const cleanup = (): void => {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
      };

      // Wrapped resolve/reject that prevent double-resolution
      const wrappedResolve = (response: ModalResponse): void => {
        if (completed) return;
        completed = true;
        cleanup();
        resolve(response);
      };

      const wrappedReject = (error: Error): void => {
        if (completed) return;
        completed = true;
        cleanup();
        reject(error);
      };

      // Set up optional timeout
      if (request.timeout && request.timeout > 0) {
        timeoutId = setTimeout(() => {
          // Remove from queue
          set((state) => ({
            modalQueue: state.modalQueue.filter((req) => req.id !== id),
          }));

          // Return default or cancel based on modal type
          if (request.type === "confirm") {
            // Confirm modals default to "not confirmed" on timeout
            wrappedResolve({ type: "confirm", confirmed: false, cancelled: true });
          } else if (request.type === "ask" && (request.payload as AskPayload).defaultValue) {
            // Ask modals return default value if provided
            const payload = request.payload as AskPayload;
            wrappedResolve({ type: "ask", value: payload.defaultValue! });
          } else {
            // Other modals reject on timeout
            wrappedReject(new Error(`Modal timeout after ${request.timeout}ms`));
          }
        }, request.timeout);
      }

      const fullRequest: ModalRequest<T> = {
        ...request,
        id,
        resolve: wrappedResolve,
        reject: wrappedReject,
        timeoutId,
      };

      // Add to queue (FIFO)
      set((state) => ({
        modalQueue: [...state.modalQueue, fullRequest],
      }));
    });
  },

  dequeue: (id: string, response: ModalResponse): void => {
    const queue = get().modalQueue;
    const request = queue.find((req) => req.id === id);

    if (!request) {
      return;
    }

    // Clear timeout if exists
    if (request.timeoutId) {
      clearTimeout(request.timeoutId);
    }

    // Remove from queue
    set((state) => ({
      modalQueue: state.modalQueue.filter((req) => req.id !== id),
    }));

    // Resolve the Promise
    request.resolve(response);
  },

  cancel: (id: string): void => {
    const queue = get().modalQueue;
    const request = queue.find((req) => req.id === id);

    if (!request) {
      return;
    }

    // Clear timeout if exists
    if (request.timeoutId) {
      clearTimeout(request.timeoutId);
    }

    // Remove from queue
    set((state) => ({
      modalQueue: state.modalQueue.filter((req) => req.id !== id),
    }));

    // Reject the Promise
    request.reject(new Error("Modal cancelled by user"));
  },
});
