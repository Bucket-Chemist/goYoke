---
type: sharp-edge
title: TypeScript Optional Doesn't Match SDK Runtime Validation
date: 2026-02-04
file: packages/tui/src/hooks/useClaudeQuery.ts
error_type: TypeMismatch
occurrences: 2
status: resolved
tags: [debugging, typescript, sdk, type-safety, validation]
---

# Sharp Edge: TypeScript Optional Types vs SDK Zod Runtime Validation

## Problem

Anthropic SDK types mark fields as optional with TypeScript's `?` operator:

```typescript
interface ToolUseResponse {
  canUse: boolean
  updatedInput?: Record<string, unknown>  // TypeScript says optional
}
```

But SDK's runtime validation (Zod) enforces them as required:

```typescript
// SDK's actual Zod schema
const schema = z.object({
  canUse: z.boolean(),
  updatedInput: z.record(z.unknown()).required()  // Zod says required!
})
```

Result: Code passes TypeScript but fails at runtime with ZodError.

```
Symptom: Error thrown at runtime even though TypeScript compilation succeeds
Cause: Type definitions don't match validation schema
Cost: 2 hours debugging "optional" field that's actually required
```

## Root Cause

SDK library has:
1. **TypeScript definitions** (index.d.ts) - marks field optional
2. **Zod validation schema** (runtime) - marks field required

These are out of sync. The TypeScript maintainers didn't realize Zod was stricter than the types implied.

## Resolution

**Never trust TypeScript optional markers for SDK types.** Always:

1. Check SDK source code for Zod schemas
2. Test with actual SDK (not just types)
3. Provide all response fields even if marked optional
4. Add comments in code noting the discrepancy

```typescript
// Correct implementation
canUseTool: (request) => {
  // NOTE: SDK marks updatedInput as optional but Zod schema requires it
  // Always provide even if empty
  return {
    canUse: true,
    updatedInput: request.tool_input  // Always include
  }
}
```

## Detection Strategy

When integrating third-party SDK:

| Check | How | Result |
|-------|-----|--------|
| **TypeScript compilation** | Run `tsc` | ✅ Pass (but incomplete) |
| **Runtime test** | Actually call SDK | ❌ Fail with validation error |
| **Check Zod schemas** | Look at SDK source | Shows real requirements |

**Lesson**: TypeScript is NOT a runtime contract. Always test with actual SDK.

## Prevention

1. **Read SDK source code** for validation schemas (not just types)
2. **Test integration end-to-end** before shipping
3. **Document SDK quirks** in comments for future developers
4. **Check GitHub issues** for known TypeScript/runtime mismatches

## Related Issues

This same pattern appears with:
- Anthropic SDK response types (sometimes optional in types but required in validation)
- Other JavaScript SDKs using Zod
- Any library that auto-generates types from Zod schemas

## Code Impact

Change from:
```typescript
return { canUse: true }  // TypeScript OK, SDK fails
```

To:
```typescript
return { canUse: true, updatedInput: {} }  // Both pass
```

## Debugging Tips

If you see ZodError from SDK:

1. Find the Zod schema in SDK source
2. Compare against TypeScript types
3. Provide ALL required fields (ignore TypeScript optional)
4. Test again

---

_Archived by memory-archivist on 2026-02-04_
