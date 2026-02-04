---
type: sharp-edge
title: Async Re-entry in useClaudeQuery Spawns Multiple Sessions
date: 2026-02-04
file: packages/tui/src/hooks/useClaudeQuery.ts
error_type: RaceCondition
occurrences: 2
status: resolved
tags: [debugging, react-hooks, async, race-conditions, concurrency]
---

# Sharp Edge: Async Re-entry Without Guard Causes Duplicate Execution

## Problem

The `useClaudeQuery` hook is called on:
1. Every keystroke (onChange)
2. Query submission (onSubmit)
3. Command execution (/model, /clear)
4. State changes (Redux updates)

Without a re-entry guard, async flows fire multiple times:

```typescript
const handleQuery = async (query) => {
  // User types in input → handleQuery called
  // Before async completes, onChange fires again
  // Before previous async completes, onSubmit fires

  const session = await createSession()  // Concurrent calls
  const response = await query(session)  // Multiple sessions created!
}
```

Manifested as:
- Multiple session spawning simultaneously
- API calls to wrong sessions
- Session IDs mismatched in UI and backend
- Memory leak: orphaned sessions not cleaned up

## Root Cause

React's `useCallback` doesn't prevent concurrent execution of the same callback. If the callback contains async work:

```typescript
// No protection
const handleQuery = useCallback(async (q) => {
  await someAsyncWork()  // Can run concurrently
}, [])

// Called multiple times before first completes
handleQuery('q1')
handleQuery('q2')  // Overlaps with q1
```

## Resolution

Add a re-entry guard using `useRef`:

```typescript
const isSubmittingRef = useRef(false)

const handleQuery = async (query) => {
  // Guard: exit if already executing
  if (isSubmittingRef.current) return

  isSubmittingRef.current = true
  try {
    const session = await createSession()
    const response = await query(session)
  } finally {
    isSubmittingRef.current = false  // Reset after completion
  }
}
```

Why this works:
- `isSubmittingRef` is mutable and persists across renders
- Checked synchronously before async work starts
- Early return prevents concurrent execution
- `finally` ensures flag reset even on error

## Comparison: Other Patterns

| Pattern | Pros | Cons |
|---------|------|------|
| **Re-entry guard (useRef)** | Simple, synchronous, zero cost | Manual flag management |
| **AbortController** | Built-in cancellation | More complex, may not apply |
| **Promise debounce** | Automatic deduplication | Complexity, may drop requests |
| **Dependency array** | Built-in mechanism | Doesn't prevent concurrent calls |

## Prevention

Apply this pattern to ANY hook with async work:

```typescript
// Template
const myAsyncHook = (deps) => {
  const isExecutingRef = useRef(false)

  const executeAsync = useCallback(async () => {
    if (isExecutingRef.current) return  // Guard

    isExecutingRef.current = true
    try {
      // Async work here
      await someAsyncCall()
    } finally {
      isExecutingRef.current = false
    }
  }, [deps])

  return executeAsync
}
```

## Variants

### When You WANT Overlapping Calls

If you intentionally need multiple concurrent requests:

```typescript
const executeAsync = async () => {
  // No guard - each call proceeds
  const result = await someAsyncCall()
  return result
}

// Caller decides: wait for first, or fire both
Promise.all([
  executeAsync(),
  executeAsync()
])
```

### When You WANT Latest Result Only

Use AbortController:

```typescript
const abortRef = useRef<AbortController | null>(null)

const executeAsync = async () => {
  // Cancel previous request
  if (abortRef.current) abortRef.current.abort()
  abortRef.current = new AbortController()

  const result = await someAsyncCall({
    signal: abortRef.current.signal
  })
  return result
}
```

## Impact

This pattern prevents concurrent session creation and API call storms. Cost: one-line guard per async function.

## Code Location

- **File**: `/packages/tui/src/hooks/useClaudeQuery.ts` line 35-40
- **Pattern**: Applied to handleQuery, setSession, executeQuery

---

_Archived by memory-archivist on 2026-02-04_
