# TUI Interactive Features - Remaining Tasks

**Date:** 2026-02-04
**Context:** Session ran out of context (6%) after completing core implementation

---

## Summary

**Completed:** 22 tasks (SDK verification, Phase 1-2 implementation, 6 bug fixes)
**Remaining:** Testing verification, Phase 3 enhancements, cleanup

---

## HIGH PRIORITY - Testing Verification

### T1: Verify All Fixes Work Together
**Status:** Not tested after final bug fixes
**What:** Run comprehensive test of all implemented features

```bash
cd ~/Documents/GOgent-Fortress/packages/tui
npm run build && node dist/index.js
```

**Test Matrix:**
| Test | Command/Action | Expected Result |
|------|----------------|-----------------|
| Permission prompt | `run echo hello` | Modal appears, approve/deny works |
| Model switch modal | `/model` | Modal shows, selection switches model |
| Model switch direct | `/model haiku` | Switches to haiku, verified by asking |
| Clear history | `/clear` | Messages cleared |
| Help | `/help` | Shows command list |
| Unknown passthrough | `/foobar` | Sent to Claude, gets response |
| AskUserQuestion | `plan a new feature` | Modal with options, selection captured |
| No duplicate sessions | Any message | Single session_id in logs |
| No rendering overlap | Multiple messages | Clean display, no garbling |

### T2: Verify AskUserQuestion Response Captured
**Status:** Fix applied but not verified
**What:** Confirm user selection reaches Claude correctly

**Test:**
1. Send: `plan how to add dark mode`
2. Claude asks clarifying question via AskUserQuestion
3. Select an option from modal
4. Verify Claude acknowledges YOUR selection (not empty/generic)

**Debug:** Check `/tmp/tui-events.jsonl` for:
```json
"content": "User selected: [your actual selection]"
```

### T3: Verify Model Actually Switches
**Status:** setModel() added but not verified
**What:** Confirm model change persists

**Test:**
1. Send any message (starts session)
2. `/model haiku`
3. Ask: `what model are you`
4. Should say Haiku, NOT Sonnet

---

## MEDIUM PRIORITY - Cleanup

### T4: Remove Debug Logging
**Status:** Still enabled
**Files:**
- `packages/tui/src/hooks/useClaudeQuery.ts` lines 18-47

**Changes:**
```typescript
// Set to false or remove entirely
const DEBUG_EVENTS = false;  // Was true
```

Also remove or disable file writes to `/tmp/tui-events.jsonl`

### T5: Remove Test Files
**Files to delete:**
- `packages/tui/test-permission.md`
- `packages/tui/test-askuserquestion.md`
- Any other test-*.md files created during development

---

## LOW PRIORITY - Phase 3 Enhancements

### T6: Dedicated PermissionModal Component
**Status:** Using generic ConfirmModal
**Goal:** Better UX for permission prompts

**Features:**
- Display tool name prominently
- Show input preview (what will be executed)
- Keyboard shortcuts: Y=approve, N=deny
- "Always allow this tool" checkbox
- Color coding for destructive vs safe tools

**File:** Create `packages/tui/src/components/modals/PermissionModal.tsx`

### T7: Dedicated ModelSelectModal Component
**Status:** Using generic SelectModal
**Goal:** Better model selection UX

**Features:**
- Show model descriptions
- Highlight current model
- Show cost/speed indicators
- Remember last selection

**File:** Create `packages/tui/src/components/modals/ModelSelectModal.tsx`

### T8: Permission Caching
**Status:** Not implemented
**Goal:** "Remember" permission decisions for session

**Implementation:**
1. Add to store: `allowedTools: Set<string>`, `deniedTools: Set<string>`
2. In canUseTool callback, check cache before showing modal
3. Add "Always allow" checkbox to PermissionModal
4. Cache persists for session only (cleared on restart)

**Files:**
- Create `packages/tui/src/store/slices/permissions.ts`
- Modify `packages/tui/src/hooks/useClaudeQuery.ts`

### T9: Final Polish
**Status:** Not started
**Tasks:**
- Review all console.log/warn statements
- Ensure proper error messages for edge cases
- Add loading states where needed
- Verify accessibility (ARIA labels)

---

## KNOWN BUGS (may still exist)

### B1: Rendering Overlap
**Status:** Fix applied (Viewport keys) but user reported still seeing issues
**Investigate:** May be multiple causes beyond key collision

### B2: Multiple "/model failed" Messages
**Status:** User saw 3x "Failed to switch model" messages
**Cause:** Possibly related to multiple session spawning (should be fixed by #20)

### B3: Duplicate User Messages
**Status:** User messages appearing twice
**Cause:** Possibly React StrictMode or event handler issue

---

## Architecture Notes for Next Session

### SDK Two-Tool-System
```
MCP Tools (ask_user, confirm_action)
  → Registered with createSdkMcpServer()
  → Handler returns CallToolResult
  → SDK handles automatically ✅

SDK Built-in Tools (AskUserQuestion, EnterPlanMode)
  → Appear as tool_use in assistant events
  → Must respond via query.streamInput()
  → Implemented in handleBuiltinToolUse() ✅
```

### Key Files Modified This Session
| File | What Changed |
|------|--------------|
| `useClaudeQuery.ts` | canUseTool, streamInput, setModel, re-entry guard |
| `ClaudePanel.tsx` | Slash commands, plan mode indicator, setModel call |
| `Viewport.tsx` | Stable keys using item.id |
| `AskModal.tsx` | Label/value structure preservation |
| `session.ts` | isPlanMode computed property |
| `modal.ts` | AskPayload with label/value objects |

### Sharp Edges Discovered
1. **updatedInput required** - SDK Zod schema stricter than TypeScript types
2. **Viewport keys** - Must use stable ID, not scrollOffset+index
3. **Re-entry guard** - Async functions need explicit guard against concurrent calls
4. **Label vs Value** - SDK options have both, must preserve structure

---

## Quick Start for Next Session

```bash
# 1. Read handoff
cat packages/tui/.claude/memory/last-handoff.md

# 2. Build and test
cd packages/tui
npm run build
node dist/index.js

# 3. Run test matrix from T1 above

# 4. Check debug logs if issues
tail -f /tmp/tui-events.jsonl | jq '.type + " " + (.subtype // "")'
```

---

## Files Reference

**Handoff:** `packages/tui/.claude/memory/last-handoff.md`
**Sharp Edges:** `packages/tui/.claude/memory/sharp-edges/*.md`
**Decisions:** `packages/tui/.claude/memory/decisions/2026-02-04-tui-interactive-features.md`
**Implementation Specs:** `.claude/tmp/implementation-specs.md`
**SDK Validation:** `.claude/tmp/sdk-validation-summary.md`
