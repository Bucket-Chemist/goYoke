import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import { join } from "path";
import { homedir } from "os";
import { writeFile, mkdir, unlink, symlink, lstat } from "fs/promises";
import { colors } from "./config/theme.js";
import { Layout } from "./components/Layout.js";
import { LayoutSpike } from "./components/LayoutSpike.js";
import { ResponsiveLayout } from "./components/ResponsiveLayout.js";
import { BorderStyleTest } from "./components/BorderStyleTest.js";
import { loadSession, saveSession } from "./hooks/useSession.js";
import { useStore } from "./store/index.js";

import { logger } from "./utils/logger.js";

/** Set GOGENT_SESSION_DIR for child processes and team polling.
 *  Also writes .claude/current-session marker and .claude/tmp symlink. */
function setSessionDir(sessionId: string): void {
  const home = process.env["HOME"] || homedir();
  const sessionDirPath = join(home, ".claude", "sessions", sessionId);
  process.env["GOGENT_SESSION_DIR"] = sessionDirPath;

  // Write current-session marker + setup tmp symlink (best-effort, non-blocking)
  const projectRoot = process.env["GOGENT_PROJECT_DIR"] || process.cwd();
  void setupSessionFiles(projectRoot, sessionDirPath);
}

/** Write .claude/current-session and symlink .claude/tmp → session dir */
async function setupSessionFiles(projectRoot: string, sessionDirPath: string): Promise<void> {
  try {
    await mkdir(sessionDirPath, { recursive: true });
    await writeFile(join(projectRoot, ".claude", "current-session"), sessionDirPath + "\n");

    // Setup .claude/tmp symlink
    const tmpPath = join(projectRoot, ".claude", "tmp");
    try {
      const stat = await lstat(tmpPath);
      if (stat.isSymbolicLink()) {
        await unlink(tmpPath);
      } else {
        // Real directory — skip (migration handled by gogent-load-context on CLI start)
        return;
      }
    } catch {
      // Doesn't exist — proceed to create symlink
    }
    await symlink(sessionDirPath, tmpPath);
  } catch {
    // Best-effort — don't crash TUI if session file ops fail
  }
}

type DemoMode = "main" | "hello" | "layout" | "responsive" | "borders";

interface AppProps {
  sessionId?: string;
  verbose?: boolean;
}

/**
 * Root component for GOfortress TUI
 * Entry point for the application UI structure
 * Includes spike testing modes for layout validation
 * Handles session persistence and resumption
 */
export function App({ sessionId, verbose }: AppProps): JSX.Element {
  const [mode, setMode] = useState<DemoMode>("main");
  const [loading, setLoading] = useState(true);
  const updateSession = useStore((state) => state.updateSession);
  const totalCost = useStore((state) => state.totalCost);
  const currentSessionId = useStore((state) => state.sessionId);

  // Set verbose mode
  useEffect(() => {
    if (verbose) {
      process.env["VERBOSE"] = "1";
    }
  }, [verbose]);

  // Load session on mount if sessionId provided
  useEffect(() => {
    async function resumeSession() {
      if (!sessionId) {
        // No session ID provided — don't pre-populate sessionId in store.
        // Leave it null so query() starts a new SDK session (resume: undefined).
        // The real session ID arrives via system.init event in handleSystemEvent.
        setLoading(false);
        return;
      }

      try {
        const session = await loadSession(sessionId);
        updateSession({
          id: session.id,
          cost: session.cost,
        });
        setSessionDir(session.id);

        if (verbose) {
          void logger.info("Session resumed", {
            sessionId: session.id,
            cost: session.cost,
            toolCalls: session.tool_calls,
          });
        }
      } catch (error) {
        void logger.error("Failed to load session", {
          sessionId,
          error: error instanceof Error ? error.message : String(error),
        });
        // Continue with new session on error — leave sessionId null
        // so query() starts a fresh SDK session
      } finally {
        setLoading(false);
      }
    }

    resumeSession();
  }, [sessionId, updateSession, verbose]);

  // Auto-save session on cost changes (debounced to prevent overlapping writes)
  useEffect(() => {
    if (!currentSessionId) return;
    if (totalCost === 0) return;

    const timeout = setTimeout(async () => {
      try {
        const tokenCount = useStore.getState().tokenCount;
        const toolCalls = tokenCount.input + tokenCount.output;

        await saveSession({
          id: currentSessionId,
          created_at: new Date().toISOString(),
          last_used: new Date().toISOString(),
          cost: totalCost,
          tool_calls: toolCalls,
        });

        if (verbose) {
          void logger.info("Session saved", { sessionId: currentSessionId });
        }
      } catch (error) {
        void logger.error("Failed to save session", {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }, 2000);

    return () => clearTimeout(timeout);
  }, [totalCost, currentSessionId, verbose]);

  // Demo mode switching - only active when NOT in main mode
  // (prevents number keys from intercepting text input)
  useInput((input, _key) => {
    if (mode === "main") return; // Don't intercept typing in main app
    if (input === "0") setMode("main");
    if (input === "1") setMode("hello");
    if (input === "2") setMode("layout");
    if (input === "3") setMode("responsive");
    if (input === "4") setMode("borders");
  });

  // Show loading state while resuming session
  if (loading) {
    return (
      <Box padding={1}>
        <Text color={colors.muted}>Loading session...</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" width="100%" height="100%">
      {mode === "main" ? (
        // Main application layout (TUI-007)
        <Layout />
      ) : (
        // Spike testing modes
        <>
          {/* Header */}
          <Box borderStyle="round" borderColor={colors.primary} paddingX={2}>
            <Text bold color={colors.primary}>
              GOfortress TUI - Ink Layout Spike
            </Text>
          </Box>

          {/* Mode selector */}
          <Box paddingX={2} paddingY={1}>
            <Text dimColor color={colors.muted}>
              Press: [0] Main | [1] Hello | [2] Layout | [3] Responsive | [4] Borders | [Ctrl+C] Exit
            </Text>
          </Box>

          {/* Content area */}
          <Box flexGrow={1}>
            {mode === "hello" && (
              <Box flexDirection="column" padding={1}>
                <Text bold color={colors.primary}>
                  GOfortress TUI
                </Text>
                <Text color={colors.muted}>Hello from Ink!</Text>
                <Box marginTop={1}>
                  <Text color={colors.secondary}>
                    Use number keys to test different spike components
                  </Text>
                </Box>
              </Box>
            )}

            {mode === "layout" && <LayoutSpike />}
            {mode === "responsive" && <ResponsiveLayout />}
            {mode === "borders" && <BorderStyleTest />}
          </Box>
        </>
      )}
    </Box>
  );
}
