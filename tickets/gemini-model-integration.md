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
