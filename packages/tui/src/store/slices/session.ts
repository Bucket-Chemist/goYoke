/**
 * Session slice for Zustand store
 * Matches Go CLI session format for rollback compatibility
 * Supports per-provider state management
 */

import type { StateCreator } from "zustand";
import type { Store, SessionSlice, ProviderId, Message, ContentBlock } from "../types.js";
import { nanoid } from "nanoid";

/**
 * Initialize per-provider state records
 */
const initializeProviderRecords = <T>(defaultValue: T): Record<ProviderId, T> => ({
  anthropic: defaultValue,
  google: defaultValue,
  openai: defaultValue,
  local: defaultValue,
});

export const createSessionSlice: StateCreator<Store, [], [], SessionSlice> = (
  set,
  get
) => ({
  // Global session state
  totalCost: 0,
  tokenCount: {
    input: 0,
    output: 0,
  },
  contextWindow: {
    usedTokens: 0,
    totalCapacity: 200000,
  },
  permissionMode: "default",
  isCompacting: false,

  // Legacy fields (computed properties for backward compatibility)
  // Note: optional chaining guards against get() being undefined during Zustand init
  get sessionId(): string | null {
    return get()?.getActiveSessionId?.() ?? null;
  },
  get preferredModel(): string | null {
    return get()?.getActiveModel?.() ?? null;
  },
  get activeModel(): string | null {
    return get()?.getActiveModel?.() ?? null;
  },

  // Per-provider state
  providerMessages: initializeProviderRecords([]),
  providerSessionIds: initializeProviderRecords(null),
  providerModels: initializeProviderRecords(null),
  providerProjectDirs: initializeProviderRecords(null),

  // Global actions
  updateSession: (data): void => {
    set((state) => {
      const updates: Partial<typeof state> = {
        totalCost: data.cost ?? state.totalCost,
      };

      // If an id is provided, also store it in providerSessionIds for the
      // active provider so SessionManager.connect() can pass it as `resume:`
      // to query(). Previously the id field was silently discarded.
      if (data.id) {
        const activeProvider = get().activeProvider;
        updates.providerSessionIds = {
          ...state.providerSessionIds,
          [activeProvider]: data.id,
        };
      }

      return updates;
    });
  },

  incrementCost: (cost): void => {
    set((state) => ({
      totalCost: state.totalCost + cost,
    }));
  },

  addTokens: (tokens): void => {
    set((state) => ({
      tokenCount: {
        input: state.tokenCount.input + (tokens.input ?? 0),
        output: state.tokenCount.output + (tokens.output ?? 0),
      },
    }));
  },

  updateContextWindow: (usedTokens, totalCapacity): void => {
    set({
      contextWindow: {
        usedTokens,
        totalCapacity,
      },
    });
  },

  setPermissionMode: (mode): void => {
    set({ permissionMode: mode });
  },

  setCompacting: (compacting): void => {
    set({ isCompacting: compacting });
  },

  isPlanMode: (): boolean => {
    return get().permissionMode === "plan";
  },

  clearSession: (): void => {
    set({
      totalCost: 0,
      tokenCount: {
        input: 0,
        output: 0,
      },
      contextWindow: {
        usedTokens: 0,
        totalCapacity: 200000,
      },
      permissionMode: "default",
      isCompacting: false,
      providerMessages: initializeProviderRecords([]),
      providerSessionIds: initializeProviderRecords(null),
      providerModels: initializeProviderRecords(null),
      providerProjectDirs: initializeProviderRecords(null),
    });

    // Clear GOGENT_SESSION_DIR environment variable
    delete process.env["GOGENT_SESSION_DIR"];
  },

  // Legacy actions (backward compatibility shims)
  setPreferredModel: (model): void => {
    const activeProvider = get().activeProvider;
    get().setProviderModel(activeProvider, model);
  },

  setActiveModel: (model): void => {
    const activeProvider = get().activeProvider;
    get().setProviderModel(activeProvider, model);
  },

  // Per-provider actions
  addProviderMessage: (provider, msg): void => {
    set((state) => ({
      providerMessages: {
        ...state.providerMessages,
        [provider]: [
          ...state.providerMessages[provider]!,
          {
            ...msg,
            id: nanoid(),
            timestamp: Date.now(),
          },
        ],
      },
    }));
  },

  updateLastProviderMessage: (provider, content): void => {
    set((state) => {
      const messages = state.providerMessages[provider] ?? [];
      if (messages.length === 0) {
        return state;
      }

      const updatedMessages = [...messages];
      const lastIndex = updatedMessages.length - 1;
      const lastMessage = updatedMessages[lastIndex];

      if (!lastMessage) {
        return state;
      }

      // Preserve existing text blocks if new content has none
      const newHasText = content.some((b) => b.type === "text");
      const existingTextBlocks = lastMessage.content.filter((b) => b.type === "text");
      const mergedContent =
        !newHasText && existingTextBlocks.length > 0
          ? [...existingTextBlocks, ...content]
          : content;

      updatedMessages[lastIndex] = {
        ...lastMessage,
        content: mergedContent,
        partial: false,
      };

      return {
        providerMessages: {
          ...state.providerMessages,
          [provider]: updatedMessages,
        },
      };
    });
  },

  clearProviderMessages: (provider): void => {
    set((state) => ({
      providerMessages: {
        ...state.providerMessages,
        [provider]: [],
      },
    }));
  },

  setProviderMessages: (provider, messages): void => {
    set((state) => ({
      providerMessages: {
        ...state.providerMessages,
        [provider]: messages,
      },
    }));
  },

  setProviderSessionId: (provider, sessionId): void => {
    set((state) => ({
      providerSessionIds: {
        ...state.providerSessionIds,
        [provider]: sessionId,
      },
    }));
  },

  setProviderModel: (provider, model): void => {
    set((state) => ({
      providerModels: {
        ...state.providerModels,
        [provider]: model,
      },
    }));
  },

  setProviderProjectDir: (provider, dir): void => {
    set((state) => ({
      providerProjectDirs: {
        ...state.providerProjectDirs,
        [provider]: dir,
      },
    }));
  },

  // Convenience getters (use activeProvider from UISlice)
  getActiveMessages: (): Message[] => {
    const state = get();
    const activeProvider = state.activeProvider;
    const messages = state.providerMessages[activeProvider] ?? [];
    return messages.filter((msg) => !msg.subagentToolUseId);
  },

  getActiveSessionId: (): string | null => {
    const state = get();
    const activeProvider = state.activeProvider;
    return state.providerSessionIds[activeProvider] ?? null;
  },

  getActiveModel: (): string | null => {
    const state = get();
    const activeProvider = state.activeProvider;
    return state.providerModels[activeProvider] ?? null;
  },

  // Handoff injection (for provider switching)
  injectHandoffMessage: (provider, handoffContent, fromProvider): void => {
    set((state) => ({
      providerMessages: {
        ...state.providerMessages,
        [provider]: [
          ...state.providerMessages[provider]!,
          {
            id: nanoid(),
            role: "system" as const,
            content: [
              {
                type: "text" as const,
                text: `# Context Handoff from ${fromProvider}\n\n${handoffContent}`,
              },
            ],
            partial: false,
            timestamp: Date.now(),
          },
        ],
      },
    }));
  },
});
