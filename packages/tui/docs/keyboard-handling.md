# Keyboard Handling System (TUI-010)

## Overview

Comprehensive keyboard handling system for the TUI application with global and context-specific bindings, input history navigation, and modal input capture.

## Architecture

### Components

1. **`useKeymap` Hook** (`src/hooks/useKeymap.ts`)
   - Declarative keyboard binding system
   - Pattern matching for keys and modifiers
   - Processes bindings in order, executes first match
   - Can be enabled/disabled dynamically

2. **Key Binding Definitions** (`src/config/keybindings.ts`)
   - Factory functions for different contexts
   - Organized by scope: global, ClaudePanel, AgentsPanel
   - Help text generation for documentation

3. **Modal Timeout Logic** (`src/store/slices/modal.ts`)
   - Promise-based timeout handling
   - Automatic cleanup on resolve/reject
   - Default values for confirm modals on timeout
   - Reject for input/select modals on timeout

## Key Bindings

### Global Bindings

Active when no modal is present:

| Key      | Action               | Description                |
|----------|---------------------|----------------------------|
| Tab      | Toggle Focus        | Switch between panels      |
| Escape   | Handle Escape       | Cancel modal or exit       |
| Ctrl+C   | Force Quit          | Terminate application      |
| Ctrl+L   | Clear Screen        | Clear message history      |

### Claude Panel Bindings

Active when Claude panel is focused and no modal is present:

| Key      | Action               | Description                      |
|----------|---------------------|----------------------------------|
| Enter    | Submit Message      | Send current input               |
| Up       | History Previous    | Navigate to previous input       |
| Down     | History Next        | Navigate to next input (or back) |

### Agents Panel Bindings

Active when Agents panel is focused and no modal is present (placeholder for future implementation):

| Key      | Action               | Description                |
|----------|---------------------|----------------------------|
| Up       | Select Previous     | Select previous agent      |
| Down     | Select Next         | Select next agent          |
| Enter    | Expand Agent        | Show agent details         |

### Modal Bindings

Active when modal is visible (overrides all other bindings):

| Key      | Action               | Description                |
|----------|---------------------|----------------------------|
| Escape   | Cancel Modal        | Close modal without action |
| Enter    | Confirm/Submit      | Handled by specific modal  |

## Input Priority

The keyboard input priority follows Ink's component mounting order:

1. **Modal** (highest priority when active)
   - Captures all input via `useInput`
   - Only Escape cancels, other keys handled by modal type

2. **Panel-specific bindings** (when focused and no modal)
   - Claude panel: Enter, Up, Down
   - Agents panel: Enter, Up, Down

3. **Global bindings** (when no modal)
   - Tab, Escape, Ctrl+C, Ctrl+L
   - Always active in background

This ensures:
- Modal captures all input when visible (task #29 ✓)
- Panel bindings only fire when focused (task #28 ✓)
- No key binding conflicts (task #32 ✓)

## Input History Integration

The Claude panel integrates with the input history slice (TUI-005):

### Navigation Behavior

1. **Up Arrow** - Navigate backward in time (newer → older)
   - First press: Save current input, show most recent
   - Subsequent presses: Move through history
   - Stops at oldest entry

2. **Down Arrow** - Navigate forward in time (older → newer)
   - Move toward present
   - When reaching end: Restore current input, reset index

3. **Submit** - Add to history and reset
   - Calls `addToHistory(input)`
   - Resets navigation index to -1
   - Deduplicates automatically

### Implementation

```typescript
// Save current input when starting navigation
const currentInputRef = useRef("");

const handleHistoryPrev = (): void => {
  const historyIndex = useStore.getState().inputHistoryIndex;
  if (historyIndex === -1) {
    currentInputRef.current = input; // Save current
  }

  const historyItem = navigateHistory("up");
  if (historyItem !== null) {
    setInput(historyItem);
  }
};

const handleHistoryNext = (): void => {
  const historyItem = navigateHistory("down");
  if (historyItem !== null) {
    setInput(historyItem);
  } else {
    setInput(currentInputRef.current); // Restore current
    resetHistoryIndex();
  }
};
```

## Modal Timeout Logic

### Timeout Behavior

When a modal has a `timeout` specified:

1. **Confirm Modal**
   - Resolves with `{ type: "confirm", confirmed: false, cancelled: true }`
   - Safe default: assumes user declined

2. **Ask Modal**
   - Resolves with default value if `defaultValue` provided
   - Rejects otherwise

3. **Input/Select Modal**
   - Rejects with error: `"Modal timeout after ${ms}ms"`

### Cleanup

All timeouts are properly cleaned up:
- On resolve: `clearTimeout` before resolving
- On reject: `clearTimeout` before rejecting
- On manual cancel: `clearTimeout` in cancel action

### Implementation

```typescript
const cleanup = (): void => {
  if (timeoutId) clearTimeout(timeoutId);
};

const wrappedResolve = (response: ModalResponse): void => {
  cleanup();
  resolve(response);
};

const wrappedReject = (error: Error): void => {
  cleanup();
  reject(error);
};
```

## Usage Examples

### Adding Global Bindings

```typescript
import { useKeymap } from "../hooks/useKeymap.js";
import { createGlobalBindings } from "../config/keybindings.js";

const globalBindings = createGlobalBindings({
  toggleFocus: () => setFocusedPanel(/* ... */),
  handleEscape: () => process.exit(0),
  forceQuit: () => process.exit(0),
  clearScreen: () => clearMessages(),
});

// Only active when no modal
useKeymap(globalBindings, modalQueue.length === 0);
```

### Adding Panel-Specific Bindings

```typescript
import { createClaudePanelBindings } from "../config/keybindings.js";

const panelBindings = createClaudePanelBindings({
  submitMessage: handleSubmit,
  historyPrev: handleHistoryPrev,
  historyNext: handleHistoryNext,
});

// Only active when focused and no modal
useKeymap(panelBindings, focused && modalQueue.length === 0);
```

### Modal with Timeout

```typescript
// Confirm modal with 5-second timeout
const result = await enqueue({
  type: "confirm",
  payload: { action: "Delete file", destructive: true },
  timeout: 5000, // 5 seconds
});

// Result will be { confirmed: false, cancelled: true } if timeout
```

## Testing

### Key Binding Conflicts

No conflicts exist because:

1. **Modal vs Global**: Modal active = global disabled
2. **Modal vs Panel**: Modal active = panel disabled
3. **Claude vs Agents**: Different panels, same keys OK
4. **Global**: No overlap with panel-specific keys

### Verification Matrix

| Context      | Tab | Escape | Ctrl+C | Ctrl+L | Enter | Up/Down |
|--------------|-----|--------|--------|--------|-------|---------|
| Modal        | ❌  | ✓ (cancel) | ❌ | ❌ | ✓ (type-specific) | ❌ |
| Claude Panel | ❌  | ❌     | ❌     | ❌     | ✓     | ✓ (history) |
| Agents Panel | ❌  | ❌     | ❌     | ❌     | ✓     | ✓ (select) |
| Global       | ✓   | ✓      | ✓      | ✓      | ❌    | ❌      |

## Acceptance Criteria Status

- ✅ Task #27: Global key bindings work (Tab, Escape, Ctrl+C, Ctrl+L)
- ✅ Task #28: Panel-specific bindings only fire when focused
- ✅ Task #29: Modal captures all input when active
- ✅ Task #30: Modal timeout returns default/cancels appropriately
- ✅ Task #31: Key bindings documented in help text
- ✅ Task #32: No key binding conflicts

## Files Created/Modified

### Created
- `packages/tui/src/hooks/useKeymap.ts` - Keyboard binding hook (113 lines)
- `packages/tui/src/config/keybindings.ts` - Key binding definitions (138 lines)
- `packages/tui/docs/keyboard-handling.md` - This documentation

### Modified
- `packages/tui/src/store/slices/modal.ts` - Added timeout logic with cleanup
- `packages/tui/src/components/Layout.tsx` - Integrated global key bindings
- `packages/tui/src/components/ClaudePanel.tsx` - Added input history navigation

## Dependencies

- TUI-005: Input history slice (`navigateHistory`, `addToHistory`, `resetHistoryIndex`)
- TUI-006: Modal queue system
- TUI-008: ClaudePanel component
- TUI-009: Modal components

## Future Enhancements

1. **Help Modal** - Display key bindings on Ctrl+H
2. **Custom Bindings** - User-configurable key mappings
3. **Vim Mode** - Alternative hjkl navigation
4. **Command Palette** - Ctrl+P fuzzy command search
