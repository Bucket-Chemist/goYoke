/**
 * Unit tests for session slice
 * Tests: updateSession stores session ID, setProviderSessionId, resume flow
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useStore } from "../../src/store";

describe("Session Slice", () => {
  beforeEach(() => {
    // Clear store before each test
    useStore.getState().clearSession();
  });

  describe("updateSession", () => {
    it("should update totalCost when cost is provided", () => {
      useStore.getState().updateSession({ cost: 1.5 });

      expect(useStore.getState().totalCost).toBe(1.5);
    });

    it("should store session ID in providerSessionIds when id is provided", () => {
      const sessionId = "abc-123-def-456";

      useStore.getState().updateSession({ id: sessionId, cost: 0.5 });

      const activeProvider = useStore.getState().activeProvider;
      expect(useStore.getState().providerSessionIds[activeProvider]).toBe(sessionId);
    });

    it("should not overwrite providerSessionIds when id is not provided", () => {
      const sessionId = "existing-session-id";

      // Set existing session ID
      const activeProvider = useStore.getState().activeProvider;
      useStore.getState().setProviderSessionId(activeProvider, sessionId);

      // Update cost only — should not clear session ID
      useStore.getState().updateSession({ cost: 2.0 });

      expect(useStore.getState().providerSessionIds[activeProvider]).toBe(sessionId);
    });

    it("should make session ID available via getActiveSessionId", () => {
      const sessionId = "resume-session-id";

      useStore.getState().updateSession({ id: sessionId });

      expect(useStore.getState().getActiveSessionId()).toBe(sessionId);
    });
  });

  describe("setProviderSessionId", () => {
    it("should store session ID for the specified provider", () => {
      useStore.getState().setProviderSessionId("anthropic", "session-1");

      expect(useStore.getState().providerSessionIds.anthropic).toBe("session-1");
    });

    it("should not affect other providers", () => {
      useStore.getState().setProviderSessionId("anthropic", "session-a");
      useStore.getState().setProviderSessionId("google", "session-g");

      expect(useStore.getState().providerSessionIds.anthropic).toBe("session-a");
      expect(useStore.getState().providerSessionIds.google).toBe("session-g");
    });

    it("should allow clearing session ID with null", () => {
      useStore.getState().setProviderSessionId("anthropic", "session-1");
      useStore.getState().setProviderSessionId("anthropic", null);

      expect(useStore.getState().providerSessionIds.anthropic).toBeNull();
    });
  });

  describe("session resume flow", () => {
    it("should make session ID retrievable by SessionManager.connect() pattern", () => {
      const resumeSessionId = "0322851c-21b2-4125-a558-961747ea025f";

      // Simulate what App.tsx resumeSession() does after the fix:
      // 1. setProviderSessionId (explicit)
      const activeProvider = useStore.getState().activeProvider;
      useStore.getState().setProviderSessionId(activeProvider, resumeSessionId);

      // 2. updateSession (also stores ID now)
      useStore.getState().updateSession({ id: resumeSessionId, cost: 1.23 });

      // Simulate what SessionManager.connect() does:
      const store = useStore.getState();
      const existingSessionId = store.providerSessionIds[store.activeProvider];

      expect(existingSessionId).toBe(resumeSessionId);
      // This would be passed as: resume: existingSessionId || undefined
      expect(existingSessionId || undefined).toBe(resumeSessionId);
    });

    it("should return null for new sessions (no resume)", () => {
      // Fresh store — no session loaded
      const store = useStore.getState();
      const existingSessionId = store.providerSessionIds[store.activeProvider];

      expect(existingSessionId).toBeNull();
      // This would be passed as: resume: null || undefined → undefined (new session)
      expect(existingSessionId || undefined).toBeUndefined();
    });
  });

  describe("clearSession", () => {
    it("should reset all session state including providerSessionIds", () => {
      // Set up state
      useStore.getState().updateSession({ id: "session-to-clear", cost: 5.0 });
      useStore.getState().setProviderSessionId("anthropic", "session-to-clear");

      // Clear
      useStore.getState().clearSession();

      expect(useStore.getState().totalCost).toBe(0);
      expect(useStore.getState().providerSessionIds.anthropic).toBeNull();
      expect(useStore.getState().getActiveSessionId()).toBeNull();
    });
  });
});
