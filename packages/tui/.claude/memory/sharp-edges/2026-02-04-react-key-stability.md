---
type: sharp-edge
title: React Viewport Key Instability Causes Reconciliation Issues
date: 2026-02-04
file: packages/tui/src/components/primitives/Viewport.tsx
error_type: RenderingBug
occurrences: 1
status: resolved
tags: [debugging, react, rendering, keys, reconciliation]
---

# Sharp Edge: Unstable Keys Cause React Reconciliation Issues

## Problem

Viewport component was using composite keys with dynamic values:

```typescript
// WRONG: scrollOffset changes on every scroll
items.map(item => (
  <Message
    key={`${scrollOffset}-${item.id}`}  // Key changes → unmount/remount
    item={item}
  />
))
```

When user scrolled, `scrollOffset` changed, which changed the key, which caused React to:
1. Unmount the DOM node
2. Remount a new one
3. Lose scroll position
4. Visual flicker

Manifested as:
- Viewport jumping to top on scroll
- Messages disappearing and reappearing
- Scroll position not preserved

## Root Cause

React uses keys to identify which items have changed, been added, or been removed. When a key changes, React treats it as a new item and unmounts/remounts.

Using derived state (`scrollOffset`) as part of the key is inherently unstable:

```
Render 1: scrollOffset=0  → key="0-item123"
Render 2: scrollOffset=50 → key="50-item123"  // Different key!
// React sees "old key deleted, new key added" → unmount + remount
```

## Resolution

Use a **stable identifier** that never changes for that item:

```typescript
// CORRECT: item.id is stable
items.map(item => (
  <Message
    key={item.id}  // Always the same for this item
    item={item}
  />
))
```

Why this works:
- `item.id` is immutable (assigned at creation)
- React recognizes same item across renders
- Component state and DOM preserved
- No unnecessary unmount/remount cycles

## Prevention

**Key Rules for React Lists:**

1. ✅ **DO** use stable, unique identifiers (database IDs, UUIDs)
2. ✅ **DO** use index IF and only if list never reorders/filters
3. ❌ **DON'T** use derived values (scroll position, computed hashes)
4. ❌ **DON'T** use timestamps (technically stable but semantically wrong)
5. ❌ **DON'T** use array index for mutable lists

**Quick Check:**
- If the value could change between renders → NOT a valid key
- If removing an item would change the key of another item → NOT a valid key
- If you move items around → array index is NOT valid

## Impact

This bug affected every viewport scroll. Was masked by other issues until viewport refactor. Fix was one-line change to key prop.

## Code Changes

```typescript
// Before
{messages.map((msg, idx) => (
  <div key={`${scrollOffset}-${idx}`}>
    {msg.content}
  </div>
))}

// After
{messages.map(msg => (
  <div key={msg.id}>
    {msg.content}
  </div>
))}
```

## Related

- **File**: `/packages/tui/src/components/primitives/Viewport.tsx` line 45-50
- **Component**: Message, ChatItem
- **Lesson**: Debugging list rendering? Check keys first.

---

_Archived by memory-archivist on 2026-02-04_
