---
type: decision
title: TUI Interactive Features Architecture
date: 2026-02-04
source: session
status: active
tags: [tui, interactive, permissions, plan-mode, slash-commands, react, typescript]
related: [2026-02-04-sdk-tool-systems.md, 2026-02-04-react-key-stability.md]
---

# TUI Interactive Features Architecture

## Context

GOgent-Fortress TUI must support Claude interactive features:
- Permission prompts for tool use
- Slash commands for local actions
- Plan mode UI indicators
- AskUserQuestion integration

These features require coordination between React components, Redux state management, and Anthropic SDK callbacks.

## Decision: Callback-Driven Architecture

### Chosen Approach

```
SDK Event (canUseTool, AskUserQuestion)
    ↓
useClaudeQuery Hook (handles + parses)
    ↓
Redux State (modal queued, isPlanMode set)
    ↓
React Components Render (Modal, Badge)
    ↓
User Input (approve/deny) or (type response)
    ↓
Callback Response (updatedInput, streamInput)
    ↓
SDK Continues Execution
```

### Why This Design

1. **Separation of Concerns**: Hooks handle SDK integration, store manages state, components render UI
2. **Testability**: Each layer independently testable
3. **Maintainability**: Clear data flow prevents spaghetti logic
4. **Scalability**: Easy to add new interactive features (new reducer actions)

## Implementation Details

### Permission Modal Flow

```typescript
canUseTool: (request) => {
  // 1. Queue modal in Redux
  dispatch(queueModal({
    type: 'AskToolPermission',
    toolName: request.tool_name,
    input: request.tool_input,
    onApprove: () => response.updatedInput = {...}
  }))

  // 2. Wait for user (re-render cycle)
  // 3. On approve/deny, callback triggered
  // 4. Return updatedInput to SDK
}
```

**Key constraint**: SDK callbacks must return **synchronously** but UI is async. Solution: Queue modal, store approval handler, execute when ready.

### Slash Command Parsing

```typescript
// Pre-execution in useClaudeQuery
if (query.startsWith('/')) {
  const [cmd, arg] = parseCommand(query)
  switch (cmd) {
    case 'model': setModel(arg); return
    case 'clear': clearHistory(); return
    case 'help': showHelp(); return
  }
  // Never reach SDK
}
```

**Decision**: Commands are local-only, never sent to Claude. Keeps logic simple and model-agnostic.

### Plan Mode Detection

```typescript
// SDK event listener
onSystemStatus: (status) => {
  if (status.status === 'planning') {
    dispatch(setPlanMode(true))
    // Render yellow badge on input
  }
}
```

**Alternative considered**: Parsing tool names from SDK events. Rejected: Plan mode is explicit signal, no ambiguity.

## Rationale for Key Decisions

### Why Not Promise-Based Modal?

Option: Make modal async so callback can await user input.

**Rejected** because:
- SDK expects synchronous callbacks (timed out if delayed)
- Promise-based would require event listeners or custom control flow
- Current queue-based approach is simpler and guaranteed synchronous

### Why Queue Instead of Global?

Option: Set global modal ref directly.

**Rejected** because:
- Multiple tools could request permissions simultaneously
- Queue preserves order, prevents race conditions
- Redux gives us deterministic state updates

### Why Stable Keys for Viewport?

Option: Use array index or derived values like `scrollOffset-${index}` as React key.

**Rejected** because:
- React reconciliation unmounts/remounts when key changes
- Causes visual flicker and scroll position loss
- Stable `item.id` prevents unnecessary DOM updates

## Trade-offs

### Synchronous Callbacks vs. Async UX

**Trade-off**: SDK callbacks must return immediately, but modal rendering is async.

**Solution**:
- Queue modal
- Store approval handler
- Call handler on user input
- Pass result to SDK via closure

**Cost**: One extra re-render cycle, negligible for user interaction speed.

### Local Commands vs. Server-Side

**Trade-off**: `/model` and `/clear` are local. Should they sync to server?

**Decision**: No. These are user preferences, not conversation state. Store in localStorage if persistence needed.

**Rationale**: Simplicity + model-agnostic behavior.

## Constraints & Assumptions

- ✅ SDK callbacks are **synchronous** (fail if delayed)
- ✅ Modal queue is **FIFO** (user processes one at a time)
- ✅ Plan mode is **read-only** UI indicator (no user action required)
- ✅ Slash commands are **synchronous** and **local-only**
- ✅ AskUserQuestion requires **streamInput** method on SDK instance

## Failure Modes & Mitigations

| Failure Mode | Symptom | Mitigation |
|--------------|---------|-----------|
| Missing `updatedInput` | ZodError in canUseTool | Always include field (even empty object) |
| Re-entry in async flow | Multiple sessions spawned | Re-entry guard: `isSubmittingRef.current` check |
| Changing viewport keys | React flicker, scroll loss | Use stable `item.id` as key |
| Command sent to Claude | Model confusion, errors | Check for `/` prefix BEFORE query execution |
| Permission timeout | Modal never resolves | Add timeout to modal queue, auto-deny |

## Verified Behaviors

- ✅ Permission modal appears on tool request
- ✅ User approval updates tool input and continues execution
- ✅ User denial prevents tool execution
- ✅ `/model` switches model without clearing chat
- ✅ `/clear` resets history and system prompt
- ✅ `/help` displays command reference
- ✅ Plan mode badge appears when SDK emits planning status
- ✅ AskUserQuestion modal appears and accepts text input
- ✅ Response sent back to SDK via streamInput

## Related Decisions

- **SDK Tool Systems** (2026-02-04-sdk-tool-systems.md): MCP tools auto-handled vs built-in tools require streamInput
- **React Key Stability** (2026-02-04-react-key-stability.md): Why viewport uses item.id instead of derived keys
- **Permission Caching** (deferred): Should permissions persist across sessions?

## Future Enhancements

1. **Permission Caching**: Store approved tools, auto-approve on repeat
2. **Tool-Specific Modals**: Different UI for different tool types (confirmation vs. input)
3. **User Preferences**: localStorage for model choice, command aliases
4. **Analytics**: Track permission grants per tool

---

_Archived by memory-archivist on 2026-02-04_
