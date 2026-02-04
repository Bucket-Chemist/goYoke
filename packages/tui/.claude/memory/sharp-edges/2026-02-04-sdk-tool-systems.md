---
type: sharp-edge
title: SDK Has Two Tool Systems - MCP Tools vs Built-in Tools
date: 2026-02-04
file: packages/tui/src/hooks/useClaudeQuery.ts
error_type: ArchitectureConfusion
occurrences: 1
status: resolved
tags: [architecture, sdk, tools, mcp, streaming]
---

# Sharp Edge: SDK Has Two Tool Systems (Not One)

## Problem

Anthropic SDK provides tools in two ways, but documentation conflates them:

1. **MCP Tools** (e.g., browser-use, fetch)
   - Defined in server config (Remote Procedure Call protocol)
   - SDK auto-handles execution
   - User just defines tool name/schema
   - `on_result` callback gives result

2. **SDK Built-in Tools** (e.g., AskUserQuestion, computer_20241022)
   - Defined in SDK itself
   - Require manual handling via `streamInput`
   - User must handle request, generate response
   - Special tool that needs interactive input

Confusion occurs because both look like "tools" but have different architecture:

```typescript
// MCP Tool (auto-handled)
const result = await client.messages.create({
  tools: [{ type: 'computer_20241022', ... }]
})
// SDK handles execution automatically
// Developer just receives result

// Built-in Tool (manual streaming)
const stream = await client.messages.stream({
  tools: [{ type: 'computer_20241022', ... }]
})
// Developer must handle via streamInput callback
// AskUserQuestion requires interactive input from user
```

## Root Cause

Anthropic documentation lists all tools together without distinguishing:
- Which are auto-handled (MCP)
- Which require manual streaming (built-in)
- Which need user interaction (AskUserQuestion)

This causes developers to assume all tools work the same way, then discover at runtime that AskUserQuestion needs special handling.

## Resolution

When using SDK tools, check the **metadata.type**:

```typescript
// In message stream callback
for (let event of stream) {
  if (event.type === 'content_block_delta') {
    const delta = event.delta

    if (delta.type === 'input_json_delta') {
      // Tool is being called - what kind?
      const toolMeta = findToolMetadata(...)

      if (toolMeta.type === 'builtin') {
        // Use streamInput - user interaction required
        await stream.streamInput({
          content: userResponse
        })
      } else {
        // MCP Tool - SDK handles automatically
        // Just let stream continue
      }
    }
  }
}
```

## Tool Classification

| Tool | Type | Handling | Example |
|------|------|----------|---------|
| AskUserQuestion | Built-in | `streamInput` | Prompt user for input |
| computer_20241022 | Built-in | `streamInput` | Accept user screenshot/action |
| browser-use | MCP | Auto | SDK makes HTTP calls |
| fetch | MCP | Auto | SDK makes HTTP calls |
| custom_tool | Depends | Check metadata | Depends on definition |

## Prevention

1. **Check tool documentation** for "streamInput" mention
2. **Test with actual SDK** to see if auto-handled
3. **Read stream events carefully** - tool execution shows up in events
4. **Look for "User Interaction" in tool docs** - that's the clue it needs streamInput

## Key Insight

```
If tool needs user input → streamInput
If tool is API call → auto-handled (MCP)
```

## Code Location

- **File**: `/packages/tui/src/hooks/useClaudeQuery.ts` line 145-160
- **Pattern**: Check metadata.type, route to streamInput for built-in tools

## Documentation Recommendation

For future developers reading SDK docs:
- **MCP Tools** (auto-handled by SDK) - browser-use, fetch, etc.
- **Built-in Tools** (require streamInput) - AskUserQuestion, computer_20241022
- **Check tool source** to see if it needs streamInput parameter

---

_Archived by memory-archivist on 2026-02-04_
