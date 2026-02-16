# Gemini Model Integration Status

**Date**: 2026-02-16
**Status**: Integrated (Alpha) - Parsing Issues

## Goal

Enable the TUI to use the local `gemini` CLI as a backend replacement for `claude code`, while preserving all TUI features (personas, hooks, subagents).

## Implementation Details

The implementation follows an Adapter pattern to bridge the `claude-agent-sdk` protocol with the `gemini` CLI.

### 1. Session Manager (`packages/tui/src/session/SessionManager.ts`)

- **Logic**: In `connect()`, if `preferredModel` starts with `gemini`, we swap the `pathToClaudeCodeExecutable` to our custom adapter script.
- **Environment**: We pass the selected model (e.g., `gemini-pro`) to the adapter via the `GEMINI_MODEL` environment variable.
- **Path Resolution**: The adapter path is resolved relative to the built `dist/index.js` file using `import.meta.url`, ensuring it works from any working directory.

### 2. The Adapter (`packages/tui/scripts/gemini-adapter.ts`)

- **Role**: Acts as a "fake" `claude code` process.
- **Input**: Reads JSON lines from `stdin` (SDK protocol).
- **Process**:
  - Spawns `/usr/bin/gemini` with arguments: `--output-format stream-json`, `--prompt "..."`, `-m <mapped_model>`.
  - Example Model Mapping: `gemini-pro` -> `gemini-3-pro-preview`.
- **Output**:
  - Reads `gemini` JSONL output.
  - Maps `gemini` events to SDK events (`system` init, `user` message, `assistant` message/tool use).
  - Emits mapped events to `stdout`.

### 3. Frontend (`packages/tui/src/components/ClaudePanel.tsx`)

- **Model Selector**: Added "Gemini 3 Pro" and "Gemini 3 Flash" to the `/model` dropdown.
- **Aliases**: Added `gemini` -> `gemini-pro` alias.

## Current Status & Issues

**What Works**:

- The TUI successfully launches the adapter.
- The adapter successfully spawns `gemini`.
- `gemini` receives the prompt and generates a response (visible in debug logs if enabled).

**What's Broken**:

- **Response Rendering**: The TUI chat interface does not display the Gemini response.
- **Diagnosis**: The adapter is likely emitting events in a slightly incorrect format or sequence that the SDK's `query` function (or `SessionManager` event handlers) doesn't approve of. It expects a strict sequence (e.g., `assistant` message start -> content block -> delta -> stop). The current adapter might be emitting full text blocks or missing a specific "start" event.

## Next Steps needed

1.  **Debug Adapter Output**:
    - Enable debug logging in `gemini-adapter.ts` (`DEBUG_ADAPTER=1`).
    - Compare the JSON output of the adapter against a real `claude code` trace.
2.  **Fix Event Mapping**:
    - Ensure `assistant` messages are properly chunked if the SDK expects streams.
    - Verify `message_start` vs `content_block_start` events.
3.  **Tool Use**:
    - Verify that `gemini` tool calls (if supported by the CLI) are translated to the SDK's `tool_use` structure correctly.

## Architectural Critique & Future Direction (Ultrathink)

The current approach of "hot-swapping" the backend executable via `SessionManager` has significant limitations:

1.  **Context Fragmentation**: Swapping the CLI process likely loses conversation context/history unless explicitly serialized and re-injected. Claude Code and Gemini CLI utilize different local state/caching mechanisms.
2.  **Capability Mismatch**: Different models/providers have different capabilities (MCP support, tool definitions, output formats). Forcing them all through a single "Claude-like" pipe is leaky.

**Proposed Architecture: Provider-Based Subtabs**

Instead of a single linear chat that swaps backends, the TUI should evolve into an **AI Command Center** with separate spaces for different agents.

#### 1. True Isolation

- **UI**: Separate "Subtabs" or views for different Providers (e.g., `[ Claude ]`, `[ Gemini ]`, `[ DeepSeek ]`).
- **State**: Each Provider maintains its own:
  - Active Session/Process.
  - Context/History.
  - Keybindings/Mode (e.g., Claude might be in "Plan Mode", Gemini in "Research Mode").

#### 2. Skill Specialization

Providers should play to their strengths rather than being forced into a generic mold:

- **Claude Tab**: Optimized for Coding, "Computer Use", "Bash", "Project Context".
- **Gemini Tab**: Optimized for "Google Search", "1M+ Context Window", "Workspace Integration".
- **DeepSeek/Other**: Specialized reasoning or niche tasks.

#### 3. Context Handoff (Explicit > Implicit)

Instead of assuming implicit shared state (which fails across different CLI tools), interactions should be explicit:

- **"Summarize to Gemini"**: A user action that takes the current Claude context, summarizes it, and injects it into the Gemini tab's prompt. "Consulting a colleague".
- **Copy/Paste**: First-class support for moving code blocks or context between tabs.

#### 4. File System & Config Isolation

To ensure full compatibility, each provider should reference its own configuration directory within the project root:

- **Claude**: Defaults to `.claude/` or `.gogent/`.
- **Gemini**: Uses `.gemini/` for its specific hooks, memory, and `config.json`.
- **Legacy/Other**: Folders like `.max2.5/` for specific model versions.
  This prevents "Hook Hell" where a tool designed for Claude 3.5 breaks Gemini 1.5.

### Implementation Roadmap

1.  **Refactor SessionManager**: Stop it from being a Singleton.
    - Create `SessionRegistry` that holds a map of `providerId -> SessionInstance`.
    - `SessionInstance` interface becomes the contract, with different implementations for `ClaudeSession` (SDK) and `GeminiSession` (Adapter/Native).
2.  **UI Upgrade**: Add a tab bar above `ClaudePanel`.
    - `[ Claude ]` `[ Gemini ]` `[ + Add Agent ]`
    - State `activeTab` determines which `SessionInstance` receives input/renders output.
3.  **State Slice Refactor**:
    - Redesign the Zustand store to be keyed by `sessionId` or `providerId`.
    - `messages` -> `sessions: { [id]: { messages: [], historyIndex: 0 } }`.

### Package Architecture Decision (Monolith vs Split)

**Question**: Should we split into `packages/tui-claude`, `packages/tui-gemini`, etc?

**Decision: Keep within `packages/tui` (Modular Monolith)**.

**Rationale**:

1.  **Unified "Command Center" UI**: The goal is a single application with tabs. Splitting packages implies separate binaries/processes, which makes "Context Handoff" and shared UI state (Zustand) extremely difficult without a complex IPC layer.
2.  **Shared Components**: All providers utilize the same Ink/React components (`Input`, `MarkdownRenderer`, `Spinner`).
3.  **Simplicity**: Managing one `package.json` with multiple SDK dependencies is manageable.
4.  **Refactoring Pattern**:
    - `src/providers/claude/` (Encapsulate Claude-specific SDK logic)
    - `src/providers/gemini/` (Encapsulate Gemini-specific SDK logic/Adapters)
    - `src/core/` (Shared interfaces)

We can extract `packages/provider-claude` later if the codebase becomes unmanageable, but for now, **folder-level isolation** is sufficient and superior for DX.
