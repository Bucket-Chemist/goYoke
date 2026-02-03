/**
 * Modal slice for Zustand store
 * Implements Promise-based modal queue with FIFO ordering
 */

import type { StateCreator } from "zustand";
import type { Store } from "../types.js";
import { randomUUID } from "crypto";

// Payload types for different modal variants
export interface AskPayload {
  message: string;
  options?: string[];
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
      const id = randomUUID();
      let timeoutId: NodeJS.Timeout | undefined;

      // Set up optional timeout
      if (request.timeout !== undefined && request.timeout > 0) {
        timeoutId = setTimeout(() => {
          // Remove from queue and reject
          const queue = get().modalQueue;
          const index = queue.findIndex((req) => req.id === id);
          if (index !== -1) {
            set((state) => ({
              modalQueue: state.modalQueue.filter((req) => req.id !== id),
            }));
          }
          reject(new Error(`Modal timeout after ${request.timeout}ms`));
        }, request.timeout);
      }

      const fullRequest: ModalRequest<T> = {
        ...request,
        id,
        resolve,
        reject,
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
