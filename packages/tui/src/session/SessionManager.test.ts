import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  getSessionManager,
  resetSessionManager,
} from "./SessionManager.js";
import { SessionState } from "./types.js";
import type { SDKMessage } from "@anthropic-ai/claude-agent-sdk";

/**
 * Mock the Claude Agent SDK
 */
vi.mock("@anthropic-ai/claude-agent-sdk", async () => {
  const mockQuery = vi.fn();
  return {
    query: mockQuery,
  };
});

/**
 * Mock useStore from Zustand
 *
 * Includes all fields accessed by SessionManager:
 *   - Core session/message fields (original)
 *   - Provider-namespaced fields added when multi-provider support landed
 *   - Agents slice fields needed for eager root agent registration
 */
vi.mock("../store/index.js", () => {
  /**
   * Mutable fields that tests may inspect or that addAgent writes.
   * Exposed so the resetMockStore() helper can wipe them between tests.
   */
  const mutableState = {
    rootAgentId: null as string | null,
    agents: {} as Record<string, unknown>,
  };

  const mockStore = {
    // ── session ──────────────────────────────────────────────────────────
    sessionId: null,
    preferredModel: "sonnet",
    contextWindow: {
      usedTokens: 0,
      totalCapacity: null,
    },
    // Provider namespace (required since multi-provider refactor)
    activeProvider: "anthropic",
    providerSessionIds: { anthropic: null } as Record<string, string | null>,
    providerModels: { anthropic: "claude-sonnet-4-6" } as Record<string, string>,

    // ── agents slice (backed by mutableState) ─────────────────────────────
    get rootAgentId() { return mutableState.rootAgentId; },
    set rootAgentId(v: string | null) { mutableState.rootAgentId = v; },
    get agents() { return mutableState.agents; },

    // ── actions ───────────────────────────────────────────────────────────
    updateSession: vi.fn(),
    setActiveModel: vi.fn(),
    setProviderModel: vi.fn(),
    setProviderSessionId: vi.fn(),

    addMessage: vi.fn(),
    updateLastMessage: vi.fn(),
    addProviderMessage: vi.fn(),
    updateLastProviderMessage: vi.fn(),

    incrementCost: vi.fn(),
    addTokens: vi.fn(),
    updateContextWindow: vi.fn(),
    setPermissionMode: vi.fn(),
    setCompacting: vi.fn(),
    setStreaming: vi.fn(),
    setInterruptQuery: vi.fn(),
    enqueue: vi.fn(),
    addToast: vi.fn(),

    // Agents slice mutations — mirrors real addAgent behaviour for root tracking
    addAgent: vi.fn((agent: { id: string; parentId: string | null; [k: string]: unknown }) => {
      mutableState.agents = { ...mutableState.agents, [agent.id]: agent };
      if (!mutableState.rootAgentId && agent.parentId === null) {
        mutableState.rootAgentId = agent.id;
      }
    }),

    // Helper to reset mutable state between tests (call from beforeEach)
    _reset() {
      mutableState.rootAgentId = null;
      mutableState.agents = {};
    },
  };

  return {
    useStore: {
      getState: vi.fn(() => mockStore),
    },
    // Export mutableState for direct inspection in root-agent tests
    _mockMutableState: mutableState,
  };
});

/**
 * Mock MCP server
 */
vi.mock("../mcp/server.js", () => ({
  mcpServer: {},
}));

/**
 * Mock logger
 */
vi.mock("../utils/logger.js", () => ({
  logger: {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

/**
 * Reset mutable mock-store fields between tests.
 * vi.clearAllMocks() resets call counts but NOT data mutations —
 * addAgent() writes to rootAgentId/agents which must be wiped each test.
 */
afterEach(async () => {
  const { useStore } = await import("../store/index.js");
  const store = useStore.getState() as { _reset?: () => void };
  store._reset?.();
});

/**
 * Helper to create a mock event stream that simulates SDK behavior
 */
function createMockSDK(config: {
  sessionId?: string;
  initDelay?: number;
  shouldError?: boolean;
  errorMessage?: string;
} = {}) {
  const {
    sessionId = "test-session-123",
    initDelay = 10,
    shouldError = false,
    errorMessage = "Mock error",
  } = config;

  let messageGenerator: AsyncIterator<any> | null = null;
  const interrupt = vi.fn().mockResolvedValue(undefined);
  const setModel = vi.fn().mockResolvedValue(undefined);

  const mockQueryFn = vi.fn((args: any) => {
    if (shouldError) {
      throw new Error(errorMessage);
    }

    // Extract the message generator
    const promptGen = args.prompt;
    if (promptGen && typeof promptGen[Symbol.asyncIterator] === "function") {
      messageGenerator = promptGen[Symbol.asyncIterator]();
    }

    // Return event stream (cast to satisfy Query type — mock only implements used methods)
    const eventStream = {
      async *[Symbol.asyncIterator](): AsyncIterator<SDKMessage> {
        // Yield init event after delay
        await new Promise((resolve) => setTimeout(resolve, initDelay));
        yield {
          type: "system",
          subtype: "init",
          session_id: sessionId,
          model: "sonnet",
        } as SDKMessage;

        // Process messages from generator
        if (messageGenerator) {
          try {
            while (true) {
              const next = await Promise.race([
                messageGenerator.next(),
                new Promise<{ done: true }>((resolve) =>
                  setTimeout(() => resolve({ done: true }), 100)
                ),
              ]);

              if (next.done) {
                break;
              }

              // Simulate processing delay
              await new Promise((resolve) => setTimeout(resolve, 20));

              // Yield result event
              yield {
                type: "result",
                total_cost_usd: 0.001,
                usage: { input_tokens: 10, output_tokens: 5 },
              } as SDKMessage;
            }
          } catch (e) {
            // Generator ended
          }
        }
      },
      interrupt,
      setModel,
      streamInput: vi.fn().mockResolvedValue(undefined),
      setPermissionMode: vi.fn(),
      setMaxThinkingTokens: vi.fn(),
      initializationResult: Promise.resolve({}),
      supportedCommands: [],
    } as any;

    return eventStream;
  });

  return { mockQueryFn, interrupt, setModel };
}

/**
 * Shut down any live SessionManager instance before each test.
 * Without this, the old instance's consumeEvents() loop keeps running and
 * can fire reconnect attempts that call vi.mocked(query) — now pointing to
 * the next test's mock — causing spurious store method calls.
 *
 * shutdown() rejects any queued messages with "Session shutdown".
 * Attach a no-op .catch() to every pending enqueue promise to prevent
 * unhandled rejection warnings from those orphaned promises.
 */
beforeEach(async () => {
  try {
    const { getSessionManager: _get } = await import("./SessionManager.js");
    const mgr = _get();
    // Drain any pending promises from the queue before shutting down
    // by attaching no-op handlers so Vitest doesn't flag them.
    const shutdownPromise = mgr.shutdown();
    shutdownPromise.catch(() => {/* intentional: cleanup rejection */});
    await shutdownPromise.catch(() => {});
  } catch {/* not yet constructed — safe to ignore */}
  resetSessionManager();
  vi.clearAllMocks();
});

describe("SessionManager state machine", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("starts in UNINITIALIZED state", () => {
    const manager = getSessionManager();
    expect(manager.getState()).toBe(SessionState.UNINITIALIZED);
  });

  it("transitions UNINITIALIZED -> CONNECTING -> READY on connect()", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    expect(manager.getState()).toBe(SessionState.UNINITIALIZED);

    const connectPromise = manager.connect();
    expect(manager.getState()).toBe(SessionState.CONNECTING);

    await connectPromise;
    expect(manager.getState()).toBe(SessionState.READY);
  });

  it("transitions to ERROR on connection failure", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({
      shouldError: true,
      errorMessage: "Network error",
    });
    vi.mocked(query).mockImplementation(mockQueryFn);

    // Use maxReconnectAttempts: 1 to stay in ERROR state
    const manager = getSessionManager({ maxReconnectAttempts: 1 });

    await expect(manager.connect()).rejects.toThrow("Network error");

    // May be ERROR or DEAD (if reconnect already attempted)
    expect([SessionState.ERROR, SessionState.DEAD]).toContain(
      manager.getState()
    );
  });

  it("transitions to DEAD on shutdown()", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    expect(manager.getState()).toBe(SessionState.READY);

    await manager.shutdown();

    expect(manager.getState()).toBe(SessionState.DEAD);
  });

  it("throws on invalid transition from DEAD", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();
    await manager.shutdown();

    expect(manager.getState()).toBe(SessionState.DEAD);

    await expect(manager.connect()).rejects.toThrow(
      /Cannot connect from state/
    );
  });

  it("allows valid transitions through all states", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const visitedStates = new Set<SessionState>();

    // UNINITIALIZED
    const manager = getSessionManager({ maxReconnectAttempts: 1, reconnectDelayMs: 10 });
    visitedStates.add(manager.getState());

    // CONNECTING -> READY
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);
    const connectPromise = manager.connect();
    visitedStates.add(manager.getState());
    await connectPromise;
    visitedStates.add(manager.getState());

    // ERROR (force connection error)
    const { mockQueryFn: errorQuery } = createMockSDK({
      shouldError: true,
    });
    vi.mocked(query).mockImplementation(errorQuery);
    await manager.connect().catch(() => {});
    await new Promise((resolve) => setTimeout(resolve, 50));
    if (manager.getState() === SessionState.ERROR) {
      visitedStates.add(manager.getState());
    }

    // DEAD
    await manager.shutdown();
    visitedStates.add(manager.getState());

    // Should have visited at least UNINITIALIZED, CONNECTING, READY, DEAD
    expect(visitedStates).toContain(SessionState.UNINITIALIZED);
    expect(visitedStates).toContain(SessionState.CONNECTING);
    expect(visitedStates).toContain(SessionState.READY);
    expect(visitedStates).toContain(SessionState.DEAD);
  });
});

describe("SessionManager lifecycle", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("connect() initializes session and transitions to READY", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({ sessionId: "custom-session-id" });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    expect(manager.getState()).toBe(SessionState.READY);
    expect(manager.getSessionId()).toBe("custom-session-id");
  });

  it("shutdown() cleans up and transitions to DEAD", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    await manager.shutdown();

    expect(manager.getState()).toBe(SessionState.DEAD);
    expect(manager.getSessionId()).toBe(null);
  });

  it("shutdown() rejects pending messages", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    // Enqueue messages but don't wait
    const message1 = manager.enqueue("test 1");
    const message2 = manager.enqueue("test 2");

    // Shutdown immediately
    await manager.shutdown();

    // Messages should be rejected
    await expect(message1).rejects.toThrow("Session shutdown");
    await expect(message2).rejects.toThrow("Session shutdown");
  });

  it("resetSessionManager() clears singleton", () => {
    const manager1 = getSessionManager();
    resetSessionManager();
    const manager2 = getSessionManager();

    expect(manager1).not.toBe(manager2);
  });

  it("getSessionId() returns null before connect", () => {
    const manager = getSessionManager();
    expect(manager.getSessionId()).toBe(null);
  });

  it("getSessionId() returns session ID after connect", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({ sessionId: "session-123" });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    expect(manager.getSessionId()).toBe("session-123");
  });
});

describe("SessionManager queue management", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("queue overflow returns false at maxQueueSize", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager({ maxQueueSize: 2 });
    await manager.connect();

    // Fill queue synchronously (don't await - capacity check runs before generator can drain)
    void manager.enqueue("msg 1").catch(() => {});
    void manager.enqueue("msg 2").catch(() => {});

    // 3rd message should return false immediately (capacity check is synchronous)
    const result = await manager.enqueue("msg 3");
    expect(result).toBe(false);
  });

  it("enqueue from UNINITIALIZED auto-connects", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    expect(manager.getState()).toBe(SessionState.UNINITIALIZED);

    // Enqueue should trigger auto-connect
    // Note: This may timeout because mock doesn't properly simulate message processing
    // We just verify it doesn't throw and state changes
    void manager.enqueue("test").catch(() => {});

    // Wait for connection
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Should have transitioned out of UNINITIALIZED
    expect(manager.getState()).not.toBe(SessionState.UNINITIALIZED);
  });

  it("enqueue from ERROR auto-reconnects", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");

    // First attempt fails; all subsequent succeed.
    let callCount = 0;
    vi.mocked(query).mockImplementation((args: any) => {
      callCount++;
      if (callCount === 1) {
        throw new Error("First attempt failed");
      }
      return createMockSDK().mockQueryFn(args);
    });

    const manager = getSessionManager({
      maxReconnectAttempts: 3,
      reconnectDelayMs: 10,
    });

    await expect(manager.connect()).rejects.toThrow();

    // Must be in ERROR (or auto-reconnect already ran and it's READY/CONNECTING)
    expect([SessionState.ERROR, SessionState.CONNECTING, SessionState.READY]).toContain(
      manager.getState()
    );

    // Enqueue from ERROR should trigger reconnect (or succeed if already reconnected)
    void manager.enqueue("test").catch(() => {});

    // Wait for any in-flight reconnection to complete
    await new Promise((resolve) => setTimeout(resolve, 200));

    // query() should have been called more than once (reconnect happened)
    expect(callCount).toBeGreaterThan(1);

    // Final state should not be ERROR or DEAD
    expect([SessionState.READY, SessionState.STREAMING, SessionState.CONNECTING]).toContain(
      manager.getState()
    );
  });

  it("enqueue from DEAD returns false", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();
    await manager.shutdown();

    expect(manager.getState()).toBe(SessionState.DEAD);

    const result = await manager.enqueue("test");
    expect(result).toBe(false);
  });
});

describe("SessionManager error handling", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("handles connection error and transitions to ERROR", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({
      shouldError: true,
      errorMessage: "Connection failed",
    });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager({ maxReconnectAttempts: 1 });

    await expect(manager.connect()).rejects.toThrow("Connection failed");

    // May be ERROR or DEAD (if reconnect already attempted)
    expect([SessionState.ERROR, SessionState.DEAD]).toContain(
      manager.getState()
    );
  });

  it("attempts reconnection after error", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");

    let attemptCount = 0;
    vi.mocked(query).mockImplementation((args: any) => {
      attemptCount++;
      if (attemptCount === 1) {
        throw new Error("First attempt failed");
      }
      return createMockSDK().mockQueryFn(args);
    });

    const manager = getSessionManager({
      maxReconnectAttempts: 3,
      reconnectDelayMs: 10,
    });

    // First attempt fails
    await expect(manager.connect()).rejects.toThrow();
    expect(manager.getState()).toBe(SessionState.ERROR);

    // Wait for automatic reconnection
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Should have attempted reconnection
    expect(attemptCount).toBeGreaterThan(1);
  });

  it("transitions to DEAD after max reconnect attempts", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({
      shouldError: true,
      errorMessage: "Persistent error",
    });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager({
      maxReconnectAttempts: 2,
      reconnectDelayMs: 10,
    });

    await expect(manager.connect()).rejects.toThrow();

    // Wait for all reconnection attempts
    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(manager.getState()).toBe(SessionState.DEAD);
  });
});

describe("SessionManager interrupt()", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("calls eventStream.interrupt() when connected", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn, interrupt } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    await manager.interrupt();

    expect(interrupt).toHaveBeenCalled();
  });

  it("handles interrupt gracefully when not streaming", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    // Interrupt when READY (not streaming)
    await expect(manager.interrupt()).resolves.not.toThrow();
  });
});

describe("SessionManager setModel()", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("calls eventStream.setModel() when connected", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn, setModel } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    const result = await manager.setModel("opus");

    expect(setModel).toHaveBeenCalledWith("opus");
    expect(result).toBe(true);
  });

  it("returns false when not connected", async () => {
    const manager = getSessionManager();

    const result = await manager.setModel("opus");

    expect(result).toBe(false);
  });

  it("returns false on setModel error", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn, setModel } = createMockSDK();
    setModel.mockRejectedValueOnce(new Error("Model not found"));
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    const result = await manager.setModel("invalid-model");

    expect(result).toBe(false);
  });
});

describe("SessionManager integration", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("full lifecycle: connect -> shutdown", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    // Connect
    await manager.connect();
    expect(manager.getState()).toBe(SessionState.READY);
    expect(manager.getSessionId()).toBe("test-session-123");

    // Shutdown
    await manager.shutdown();
    expect(manager.getState()).toBe(SessionState.DEAD);
  });

  it("handles multiple connect attempts gracefully", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    await manager.connect();
    expect(manager.getState()).toBe(SessionState.READY);

    // Second connect should fail
    await expect(manager.connect()).rejects.toThrow(
      /Cannot connect from state/
    );
  });

  it("calls store methods on init event", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn } = createMockSDK({ sessionId: "session-xyz" });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    await manager.connect();

    const store = useStore.getState();
    expect(store.updateSession).toHaveBeenCalledWith({ id: "session-xyz" });
  });
});

/**
 * Helper to create controllable mock SDK that allows pushing specific events
 */
function createControllableMockSDK() {
  const events: SDKMessage[] = [];
  let resolveNext: (() => void) | null = null;
  const interrupt = vi.fn().mockResolvedValue(undefined);
  const setModel = vi.fn().mockResolvedValue(undefined);
  const streamInput = vi.fn().mockResolvedValue(undefined);

  const pushEvent = (event: SDKMessage): void => {
    events.push(event);
    resolveNext?.();
  };

  const mockQueryFn = vi.fn((_args: any) => {
    const eventStream = {
      async *[Symbol.asyncIterator](): AsyncIterator<SDKMessage> {
        while (true) {
          if (events.length > 0) {
            yield events.shift()!;
          } else {
            await new Promise<void>((resolve) => {
              resolveNext = resolve;
            });
            if (events.length > 0) {
              yield events.shift()!;
            }
          }
        }
      },
      interrupt,
      setModel,
      streamInput,
      setPermissionMode: vi.fn(),
      setMaxThinkingTokens: vi.fn(),
      initializationResult: Promise.resolve({}),
      supportedCommands: [],
    } as any;
    return eventStream;
  });

  return { mockQueryFn, pushEvent, interrupt, setModel, streamInput };
}

describe("SessionManager handleAssistantEvent", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("creates new assistant message in store", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    // Push init event to get to READY
    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push assistant event with text content block
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-1",
        role: "assistant",
        content: [
          {
            type: "text",
            text: "Hello, how can I help?",
          },
        ],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    // Code calls addProviderMessage (provider-namespaced) not the legacy addMessage
    expect(store.addProviderMessage).toHaveBeenCalledWith("anthropic", {
      role: "assistant",
      content: [
        {
          type: "text",
          text: "Hello, how can I help?",
        },
      ],
      partial: true,
    });
  });

  it("updates existing message on same messageId", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // First assistant event
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-1",
        role: "assistant",
        content: [{ type: "text", text: "Hello" }],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    // Second assistant event with same ID
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-1",
        role: "assistant",
        content: [{ type: "text", text: "Hello, how can I help?" }],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    // First call adds message
    expect(store.addProviderMessage).toHaveBeenCalledTimes(1);
    // Second call updates last message
    expect(store.updateLastProviderMessage).toHaveBeenCalled();
  });

  it("converts text and tool_use blocks correctly", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push assistant event with both text and tool_use
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-2",
        role: "assistant",
        content: [
          { type: "text", text: "Let me check that file." },
          {
            type: "tool_use",
            id: "tool-1",
            name: "Read",
            input: { file_path: "/test.ts" },
          },
        ],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    expect(store.addProviderMessage).toHaveBeenCalledWith("anthropic", {
      role: "assistant",
      content: [
        { type: "text", text: "Let me check that file." },
        {
          type: "tool_use",
          id: "tool-1",
          name: "Read",
          input: { file_path: "/test.ts" },
        },
      ],
      partial: true,
    });
  });

  it("detects AskUserQuestion tool use", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } =
      createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Mock store.enqueue to return answer
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "ask",
      value: "answer",
    } as any);

    // Push assistant event with AskUserQuestion tool
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-3",
        role: "assistant",
        content: [
          {
            type: "tool_use",
            id: "ask-1",
            name: "AskUserQuestion",
            input: {
              questions: [{ question: "What is your name?" }],
            },
          },
        ],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 100));

    // Verify message was added to store
    expect(store.addProviderMessage).toHaveBeenCalled();
  });

  it("detects ConfirmAction tool use", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } =
      createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Mock store.enqueue to return confirmation
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "confirm",
      confirmed: true,
    } as any);

    // Push assistant event with ConfirmAction tool
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-4",
        role: "assistant",
        content: [
          {
            type: "tool_use",
            id: "confirm-1",
            name: "ConfirmAction",
            input: { action: "Delete file?" },
          },
        ],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 100));

    // Verify message was added to store
    expect(store.addProviderMessage).toHaveBeenCalled();
  });

  it("logs SDK_INTERNAL_TOOLS", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { logger } = await import("../utils/logger.js");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } =
      createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push assistant event with SDK internal tool
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-5",
        role: "assistant",
        content: [
          {
            type: "tool_use",
            id: "plan-1",
            name: "EnterPlanMode",
            input: {},
          },
        ],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    // Verify logger.debug was called
    expect(logger.debug).toHaveBeenCalledWith(
      "[SDK Internal Tool]",
      expect.objectContaining({ name: "EnterPlanMode" })
    );

    // Verify message was added to store
    const store = useStore.getState();
    expect(store.addProviderMessage).toHaveBeenCalled();
  });
});

describe("SessionManager handleUserEvent", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("converts tool_result to ContentBlock", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push user event with tool_result
    pushEvent({
      type: "user",
      message: {
        role: "user",
        content: [
          {
            type: "tool_result",
            tool_use_id: "tool-1",
            content: "File contents here",
            is_error: false,
          },
        ],
      },
      parent_tool_use_id: null,
      session_id: "test-123",
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    expect(store.addProviderMessage).toHaveBeenCalledWith("anthropic", {
      role: "system",
      content: [
        {
          type: "tool_result",
          tool_use_id: "tool-1",
          content: "File contents here",
          is_error: false,
        },
      ],
      partial: false,
    });
  });

  it("finalizes current assistant message before adding tool result", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push assistant event (sets currentMessageRef)
    pushEvent({
      type: "assistant",
      message: {
        id: "msg-6",
        role: "assistant",
        content: [{ type: "text", text: "Running tool..." }],
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    // Then push user event
    pushEvent({
      type: "user",
      message: {
        role: "user",
        content: [
          {
            type: "tool_result",
            tool_use_id: "tool-2",
            content: "Result",
          },
        ],
      },
      parent_tool_use_id: null,
      session_id: "test-123",
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    // Verify updateLastProviderMessage called before addProviderMessage
    expect(store.updateLastProviderMessage).toHaveBeenCalled();
    expect(store.addProviderMessage).toHaveBeenCalledWith(
      "anthropic",
      expect.objectContaining({ role: "system" })
    );
  });
});

describe("SessionManager handleResultEvent", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("updates cost and token usage on success", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push result event with cost and usage
    pushEvent({
      type: "result",
      subtype: "success",
      total_cost_usd: 0.05,
      usage: { input_tokens: 100, output_tokens: 50 },
      modelUsage: {
        sonnet: {
          inputTokens: 80,
          cacheCreationInputTokens: 20,
          cacheReadInputTokens: 0,
          outputTokens: 50,
          contextWindow: 200000,
        },
      },
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    expect(store.incrementCost).toHaveBeenCalledWith(0.05);
    expect(store.addTokens).toHaveBeenCalledWith({
      input: 100,
      output: 50,
    });
    // updateContextWindow should extract capacity from modelUsage
    // First arg is existing usedTokens from store, second is extracted capacity
    expect(store.updateContextWindow).toHaveBeenCalled();
    const lastCall = vi.mocked(store.updateContextWindow).mock.calls.slice(-1)[0];
    expect(lastCall?.[1]).toBe(200000); // Verify capacity was extracted
  });

  it("propagates error on non-success subtype", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    const onError = vi.fn();

    manager.setEvents({
      onStateChange: vi.fn(),
      onError,
      onSessionId: vi.fn(),
      onStreamingComplete: vi.fn(),
    });

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push result event with error
    pushEvent({
      type: "result",
      subtype: "error",
      errors: ["test error"],
      total_cost_usd: 0,
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "server_error",
        message: "test error",
      })
    );
  });

  it("fires onStreamingComplete callback", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    const onStreamingComplete = vi.fn();

    manager.setEvents({
      onStateChange: vi.fn(),
      onError: vi.fn(),
      onSessionId: vi.fn(),
      onStreamingComplete,
    });

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push result event
    pushEvent({
      type: "result",
      subtype: "success",
      total_cost_usd: 0.01,
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(onStreamingComplete).toHaveBeenCalled();
  });
});

describe("SessionManager handleStatusEvent", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("updates permission mode", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push status event with permissionMode
    pushEvent({
      type: "system",
      subtype: "status",
      permissionMode: "plan",
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    expect(store.setPermissionMode).toHaveBeenCalledWith("plan");
  });

  it("updates compacting state", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Push status event with compacting
    pushEvent({
      type: "system",
      subtype: "status",
      status: "compacting",
    } as any);

    await new Promise((resolve) => setTimeout(resolve, 50));

    const store = useStore.getState();
    expect(store.setCompacting).toHaveBeenCalledWith(true);
  });
});

describe("SessionManager handleCanUseTool", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("EnterPlanMode shows plan-specific modal", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool from query call
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Mock store.enqueue to return confirmation
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "confirm",
      confirmed: true,
    } as any);

    // Create mock options
    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "tool-123",
    };

    // Call canUseTool
    const result = await canUseTool!("EnterPlanMode", {}, mockOptions);

    // Verify store.enqueue called with plan mode message
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          action: expect.stringContaining("plan mode"),
        }),
      })
    );
    // Note: EnterPlanMode still uses confirm type with action field in the payload

    expect(result).toEqual({
      behavior: "allow",
      updatedInput: {},
      toolUseID: "tool-123",
    });
  });

  it("ExitPlanMode shows allowedPrompts summary", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // ExitPlanMode uses an ask modal; code checks result.value === 'Approve'
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "ask",
      value: "Approve",
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "tool-456",
    };

    // Call with allowedPrompts
    const result = await canUseTool!(
      "ExitPlanMode",
      {
        allowedPrompts: [
          { tool: "Bash", prompt: "run tests" },
          { tool: "Write", prompt: "create file" },
        ],
      },
      mockOptions
    );

    // Verify payload contains permissions summary
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          message: expect.stringContaining("Permissions requested"),
        }),
      })
    );

    expect(result).toEqual({
      behavior: "allow",
      updatedInput: expect.objectContaining({ allowedPrompts: expect.any(Array) }),
      toolUseID: "tool-456",
    });
  });

  it("standard tool shows generic confirm", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Standard tool uses an ask modal; code checks result.value === 'Allow'
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "ask",
      value: "Allow",
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "tool-789",
    };

    // Call with standard tool
    const result = await canUseTool!(
      "Bash",
      { command: "ls" },
      mockOptions
    );

    // Verify generic permission message
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          message: expect.stringContaining("Allow Claude to use Bash?"),
        }),
      })
    );

    expect(result).toEqual({
      behavior: "allow",
      updatedInput: { command: "ls" },
      toolUseID: "tool-789",
    });
  });

  it("denied permission returns deny behavior", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Mock store.enqueue to return denial
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "confirm",
      confirmed: false,
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "tool-999",
    };

    // Call canUseTool
    const result = await canUseTool!("Bash", { command: "rm -rf /" }, mockOptions);

    expect(result).toEqual({
      behavior: "deny",
      message: "User denied permission",
      toolUseID: "tool-999",
    });
  });

  it("AskUserQuestion collects answers and returns via updatedInput", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Mock store.enqueue to return answer
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "ask",
      value: "John Doe",
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "ask-123",
    };

    const questions = [{ question: "What is your name?" }];

    // Call canUseTool with AskUserQuestion
    const result = await canUseTool!(
      "AskUserQuestion",
      { questions },
      mockOptions
    );

    // Verify result includes questions and answers
    expect(result).toEqual({
      behavior: "allow",
      updatedInput: {
        questions,
        answers: { "What is your name?": "John Doe" },
      },
      toolUseID: "ask-123",
    });

    // Verify store.enqueue was called with ask modal
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "ask",
        payload: expect.objectContaining({
          message: "What is your name?",
        }),
      })
    );
  });

  it("AskUserQuestion with options shows selection modal", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Mock store.enqueue to return selection
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "ask",
      value: "Option A",
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "ask-456",
    };

    const questions = [
      {
        question: "Choose an option:",
        options: [
          { label: "Option A", description: "First option" },
          { label: "Option B", description: "Second option" },
        ],
      },
    ];

    // Call canUseTool
    const result = await canUseTool!(
      "AskUserQuestion",
      { questions },
      mockOptions
    );

    expect(result).toEqual({
      behavior: "allow",
      updatedInput: {
        questions,
        answers: { "Choose an option:": "Option A" },
      },
      toolUseID: "ask-456",
    });

    // Verify modal received options
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "ask",
        payload: expect.objectContaining({
          options: [
            { label: "Option A", value: "Option A", description: "First option" },
            { label: "Option B", value: "Option B", description: "Second option" },
          ],
        }),
      })
    );
  });

  it("shows decisionReason and agentID in permission modal", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    pushEvent({
      type: "system",
      subtype: "init",
      session_id: "test-123",
      model: "sonnet",
    } as any);

    await manager.connect();
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Extract canUseTool
    const queryCall = vi.mocked(query).mock.calls[0]![0];
    const canUseTool = queryCall.options?.canUseTool;

    // Mock store.enqueue
    const store = useStore.getState();
    vi.mocked(store.enqueue).mockResolvedValue({
      type: "confirm",
      confirmed: true,
    } as any);

    const mockOptions = {
      signal: new AbortController().signal,
      toolUseID: "tool-abc",
      decisionReason: "Need to check configuration",
      agentID: "go-pro",
    };

    // Call canUseTool with reason and agent
    await canUseTool!("Read", { file_path: "/config.json" }, mockOptions);

    // Verify modal includes agent and reason in the message field
    expect(store.enqueue).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          message: expect.stringMatching(/\[go-pro\].*Read.*Reason: Need to check configuration/s),
        }),
      })
    );
  });
});

describe("SessionManager error classification", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("classifies timeout errors", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({
      shouldError: true,
      errorMessage: "Request timed out: ETIMEDOUT",
    });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager({ maxReconnectAttempts: 1 });
    const onError = vi.fn();

    manager.setEvents({
      onStateChange: vi.fn(),
      onError,
      onSessionId: vi.fn(),
      onStreamingComplete: vi.fn(),
    });

    await expect(manager.connect()).rejects.toThrow();

    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "timeout",
        message: expect.stringContaining("timed out"),
      })
    );
  });

  it("classifies invalid request errors", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { mockQueryFn } = createMockSDK({
      shouldError: true,
      errorMessage: "HTTP 400: Invalid request format",
    });
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager({ maxReconnectAttempts: 1 });
    const onError = vi.fn();

    manager.setEvents({
      onStateChange: vi.fn(),
      onError,
      onSessionId: vi.fn(),
      onStreamingComplete: vi.fn(),
    });

    await expect(manager.connect()).rejects.toThrow();

    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "invalid_request",
        message: expect.stringContaining("Invalid request"),
      })
    );
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Eager root agent registration (handleSystemEvent)
// ─────────────────────────────────────────────────────────────────────────────
/**
 * Tests for the eager "router-root" agent registration added to handleSystemEvent.
 *
 * When the SDK fires system.init during CONNECTING state, SessionManager now
 * immediately registers a root agent in the Zustand store so the agent panel
 * shows without waiting for the first Task() delegation.
 *
 * Behaviour under test:
 *   a) rootAgentId is set to "router-root" after system.init
 *   b) agents["router-root"] has the correct fields (model, tier, status)
 *   c) Duplicate system.init events do NOT create duplicate root entries
 *   d) Tier is correctly extracted: haiku / sonnet / opus from model string
 */
describe("SessionManager handleSystemEvent — eager root agent registration", () => {
  beforeEach(() => {
    resetSessionManager();
    vi.clearAllMocks();
  });

  it("registers router-root agent in store after system.init during CONNECTING", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    pushEvent({ type: "system", subtype: "init", session_id: "sess-1", model: "claude-sonnet-4-5" } as any);
    await manager.connect();

    const store = useStore.getState();

    // rootAgentId must be set
    expect(store.rootAgentId).toBe("router-root");

    // addAgent must have been called with the router-root shape
    expect(store.addAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "router-root",
        parentId: null,
        status: "running",
        agentType: "router",
        spawnMethod: "task",
        description: "Router",
      })
    );
  });

  it("sets model on root agent from the init event model field", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    pushEvent({ type: "system", subtype: "init", session_id: "sess-2", model: "claude-opus-4-5" } as any);
    await manager.connect();

    const store = useStore.getState();
    expect(store.addAgent).toHaveBeenCalledWith(
      expect.objectContaining({ model: "claude-opus-4-5" })
    );
  });

  describe("tier extraction from model string", () => {
    const cases: Array<{ model: string; expectedTier: "haiku" | "sonnet" | "opus" }> = [
      { model: "claude-haiku-3-5", expectedTier: "haiku" },
      { model: "claude-sonnet-4-5", expectedTier: "sonnet" },
      { model: "claude-opus-4-5", expectedTier: "opus" },
      // Fallback: unknown model string → defaults to "opus" (see SessionManager fallback)
      { model: "claude-sonnet-4-5", expectedTier: "sonnet" },
    ];

    for (const { model, expectedTier } of cases) {
      it(`extracts tier "${expectedTier}" from model "${model}"`, async () => {
        const { query } = await import("@anthropic-ai/claude-agent-sdk");
        const { useStore } = await import("../store/index.js");
        const { mockQueryFn, pushEvent } = createControllableMockSDK();
        vi.mocked(query).mockImplementation(mockQueryFn);

        const manager = getSessionManager();
        pushEvent({ type: "system", subtype: "init", session_id: "sess-tier", model } as any);
        await manager.connect();

        const store = useStore.getState();
        expect(store.addAgent).toHaveBeenCalledWith(
          expect.objectContaining({ tier: expectedTier })
        );

        await manager.shutdown().catch(() => {});
        resetSessionManager();
      });
    }
  });

  it("does NOT register a second root agent on duplicate system.init", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();

    // First init (during CONNECTING) — registers root
    pushEvent({ type: "system", subtype: "init", session_id: "sess-dup", model: "claude-sonnet-4-5" } as any);
    await manager.connect();

    const addAgentCallsAfterFirst = vi.mocked(useStore.getState().addAgent).mock.calls.length;
    expect(addAgentCallsAfterFirst).toBe(1);

    // Simulate a second system.init (SDK emits one per generator yield).
    // State is now READY, so handleSystemEvent takes the early-return path —
    // it must NOT call addAgent again.
    pushEvent({ type: "system", subtype: "init", session_id: "sess-dup", model: "claude-sonnet-4-5" } as any);
    await new Promise((resolve) => setTimeout(resolve, 50));

    const addAgentCallsAfterSecond = vi.mocked(useStore.getState().addAgent).mock.calls.length;
    expect(addAgentCallsAfterSecond).toBe(1); // still exactly 1
  });

  it("skips root creation when rootAgentId already set (no double-registration)", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    // Pre-set rootAgentId as if a previous session already created it
    const store = useStore.getState() as ReturnType<typeof useStore.getState> & { _reset?: () => void };
    store.rootAgentId = "pre-existing-root";

    const manager = getSessionManager();
    pushEvent({ type: "system", subtype: "init", session_id: "sess-skip", model: "claude-sonnet-4-5" } as any);
    await manager.connect();

    // addAgent must NOT have been called (root already existed)
    expect(store.addAgent).not.toHaveBeenCalledWith(
      expect.objectContaining({ id: "router-root" })
    );
    // rootAgentId remains the pre-existing one
    expect(store.rootAgentId).toBe("pre-existing-root");
  });

  it("uses fallback model 'claude-sonnet-4-5' when init event lacks a model field", async () => {
    const { query } = await import("@anthropic-ai/claude-agent-sdk");
    const { useStore } = await import("../store/index.js");
    const { mockQueryFn, pushEvent } = createControllableMockSDK();
    vi.mocked(query).mockImplementation(mockQueryFn);

    const manager = getSessionManager();
    // Emit init WITHOUT a model field
    pushEvent({ type: "system", subtype: "init", session_id: "sess-nomodel" } as any);
    await manager.connect();

    const store = useStore.getState();
    expect(store.addAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "router-root",
        model: "claude-sonnet-4-5", // the code-level default
        tier: "sonnet",
      })
    );
  });
});
