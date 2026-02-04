---
type: sharp-edge
title: SDK canUseTool Requires updatedInput (Despite Optional TypeScript Type)
date: 2026-02-04
file: packages/tui/src/hooks/useClaudeQuery.ts
error_type: ZodError
occurrences: 3
status: resolved
tags: [debugging, typescript, sdk, zod, type-safety]
---

# Sharp Edge: SDK canUseTool Requires updatedInput Field

## Problem

When implementing `canUseTool` callback in Anthropic SDK, TypeScript marks the response field as optional:

```typescript
interface CanUseToolResponse {
  canUse: boolean
  updatedInput?: Record<string, unknown>  // TypeScript: optional
}
```

However, the SDK's Zod validator **requires** `updatedInput` to be present in all cases. Omitting it causes:

```
ZodError: Required field missing: updatedInput
  at function_schema.parse(response)
```

This happens even when field is marked optional in TypeScript, and even when approving a tool (where updatedInput might be empty).

## Root Cause

SDK implementation uses strict Zod validation before type-checking. The schema requires the field regardless of TypeScript types. This is a mismatch between type definitions and runtime validation.

```typescript
// SDK's Zod schema (not our code)
const schema = z.object({
  canUse: z.boolean(),
  updatedInput: z.record(z.unknown())  // REQUIRED by Zod
})

// Our code (wrong)
return { canUse: true }  // Missing field → ZodError

// Fix required
return { canUse: true, updatedInput: {} }  // Always include
```

## Resolution

Always include `updatedInput` field in `canUseTool` response, even if empty:

```typescript
canUseTool: (request) => {
  // Queue modal...
  dispatch(queueModal({
    // ... modal config
    onApprove: () => {
      // ALWAYS include updatedInput
      return { updatedInput: request.tool_input }  // ✅ Correct
    },
    onDeny: () => {
      return { updatedInput: {} }  // ✅ Even for denial
    }
  }))

  return { updatedInput: {} }  // ✅ Initial response
}
```

## Prevention

When working with SDK callbacks:

1. **Never trust TypeScript optional markers** for SDK methods - check Zod schemas
2. **Always provide all response fields** even if empty
3. **Test with actual SDK validation** before relying on TypeScript
4. **Check Anthropic SDK source** for runtime validation rules

## Impact

This cost ~2 hours of debugging. The fix was one-line: always include the field.

## Related

- **File**: `/packages/tui/src/hooks/useClaudeQuery.ts` line 89-95
- **Lesson**: SDK runtime != TypeScript types. Verify with actual SDK code.

---

_Archived by memory-archivist on 2026-02-04_
