# TUI Revision Plan: GOfortress Conversation Panel Overhaul

## 1. Root Cause Analysis (Current Bugs)

### Bug 1: Item-Count Viewport vs Line-Count Reality
**File:** `Viewport.tsx`
**Problem:** The Viewport treats `height` as "number of visible items" (messages), but each message renders as **variable** number of terminal lines (role header + multi-line text + tool blocks + padding). A 5-item viewport window may actually consume 80+ terminal lines, causing massive overflow.

```tsx
// Current: slices by item count
const visibleItems = items.slice(scrollOffset, scrollOffset + height);
```

**Impact:** Content overflows the allocated Box, overlaps with the input area, and bleeds into the right panel.

### Bug 2: `console.log()` Writes to Ink-Managed stdout
**Files:** `useClaudeQuery.ts` (lines 227, 521-544, 699), `App.tsx` (lines 53-54)
**Problem:** Ink takes ownership of stdout for its rendering loop. Any `console.log()` or `fs.appendFileSync` to stdout intermixes raw text with Ink's ANSI cursor-controlled output, producing garbled lines.

**Impact:** Lines concatenate with Ink's render output, creating unreadable merged text.

### Bug 3: Raw Escape Code in Resize Handler
**File:** `useTerminalDimensions.ts` (line 37)
**Problem:** `process.stdout.write('\x1b[2J\x1b[H')` (clear screen + cursor home) directly fights Ink's rendering loop. Ink doesn't know the screen was cleared and continues rendering at its remembered cursor positions.

```tsx
// DESTRUCTIVE: bypasses Ink's render state
process.stdout.write('\x1b[2J\x1b[H');
```

**Impact:** After any terminal resize, existing content ghost-renders at wrong positions.

### Bug 4: Stale Closure in Streaming Detection
**File:** `useClaudeQuery.ts` (line 440)
**Problem:** `handleAssistantEvent` captures `isStreaming` from its closure, but React state updates are asynchronous. When the first assistant event arrives, `isStreaming` may still be `false` (set on line 537), causing a duplicate `addMessage` instead of `updateLastMessage`.

```tsx
// isStreaming may be stale when this runs
if (isStreaming && currentMessageRef.current.length > 0) {
  updateLastMessage(contentBlocks);  // Should happen
} else {
  addMessage(...);  // Happens instead → duplicate message
}
```

**Impact:** Duplicate assistant messages appear and the "update" path is missed, leaving orphaned partial messages.

### Bug 5: No Width Constraints on Text Content
**File:** `ClaudePanel.tsx` (lines 78-84)
**Problem:** Text lines render without any width constraint. Long lines (tool output, file paths, code) extend beyond the panel boundary.

**Impact:** Horizontal overflow wraps unpredictably in the terminal, creating apparent line concatenation.

### Bug 6: Missing `overflow: "hidden"` on Content Containers
**File:** `Layout.tsx` (lines 83-86), `ClaudePanel.tsx` (line 347)
**Problem:** Only ClaudePanel's message area has `overflow="hidden"`. The outer Layout Box and the ClaudePanel itself don't clip overflow. Ink's `overflow="hidden"` must be on the **immediate parent** of overflowing content.

**Impact:** Even with Viewport slicing, rendered content that exceeds height bleeds through.

---

## 2. What Ralph-TUI Does Well (Applicable Patterns)

### 2.1 Native `<scrollbox>` with Sticky Scroll
Ralph uses OpenTUI's built-in `<scrollbox stickyScroll={true} stickyStart="bottom">` which handles:
- Line-aware scrolling (not item-count)
- Auto-anchor to bottom during streaming
- User scroll-up pauses auto-scroll
- Proper clipping of overflowed content

**GOfortress equivalent:** We need a line-aware scrollable container. Since Ink doesn't have `<scrollbox>`, we must implement one using `ink`'s `measureElement` or a string-buffer approach that pre-renders content to lines, then slices by terminal row count.

### 2.2 Streaming Output Parser with FormattedSegments
Ralph separates parsing from rendering:
- `StreamingOutputParser` class accumulates chunks with a 100KB buffer limit
- Output is parsed into `FormattedSegment[]` (text + semantic color)
- `FormattedText` component renders segments without ANSI codes
- ANSI codes are stripped universally before display

**GOfortress equivalent:** Create a `MessageRenderer` that converts `ContentBlock[]` into pre-formatted line arrays with semantic styling, separate from the scrolling container.

### 2.3 Three-Section Layout (Header / Content / Footer)
```
┌─────────────── Header (fixed) ──────────────┐
│ Status bar, session info, model, cost        │
├─────────────── Content (flex) ───────────────┤
│ ┌──────────────┐ ┌─────────────────────────┐ │
│ │  Left Panel  │ │     Right Panel         │ │
│ │  (chat)      │ │  (agents/detail/dash)   │ │
│ └──────────────┘ └─────────────────────────┘ │
├─────────────── Footer (fixed) ───────────────┤
│ Keybindings, scroll position, streaming      │
└──────────────────────────────────────────────┘
```

**GOfortress equivalent:** Extract Banner into a proper Header, add a Footer with contextual keybind hints and scroll position.

### 2.4 Right Panel View Cycling
Ralph cycles the right panel between views with a single keypress:
- Task details / Agent tree / Subagent trace / Dashboard / Settings

**GOfortress equivalent:** Add view mode cycling (Tab or `o` key) to the right panel: Agents → Dashboard → Settings.

### 2.5 Text Width Management
Ralph calculates available character width and truncates with ellipsis:
```tsx
const availableWidth = totalWidth - indent - icon - label - duration;
const truncated = text.length > availableWidth
  ? text.slice(0, availableWidth - 1) + '…'
  : text;
```

### 2.6 Responsive Layout
Ralph detects terminal width and switches between side-by-side (wide) and stacked (narrow) panel layout at 80 columns.

---

## 3. Comprehensive Revision List

### Phase 1: Critical Bug Fixes (Rendering Corruption)

#### 1.1 Replace Item-Count Viewport with Line-Aware ScrollView
**Priority:** P0 (blocks all other UX work)
**Files:** New `ScrollView.tsx`, modify `ClaudePanel.tsx`
**Approach:**
- Create `ScrollView` component that accepts React children (not items array)
- Pre-measure content height using Ink's `measureElement()` or string rendering
- Maintain a `scrollOffset` in terminal rows (not item count)
- Support keyboard scrolling: Up/Down (1 line), PgUp/PgDn (1 page), Home/End
- Auto-scroll to bottom when new content arrives AND user is at bottom
- Pause auto-scroll when user scrolls up (with "[New messages below]" indicator)
- Clip content using `overflow="hidden"` on the measured container

```tsx
interface ScrollViewProps {
  height: number;           // Available terminal rows
  autoScroll?: boolean;     // Anchor to bottom
  focused?: boolean;        // Enable keyboard scrolling
  children: React.ReactNode;
}
```

#### 1.2 Eliminate All console.log/console.warn/console.error
**Priority:** P0
**Files:** `useClaudeQuery.ts`, `App.tsx`, `ClaudePanel.tsx`
**Approach:**
- Replace all `console.*` with a file-based logger (`utils/logger.ts` already exists)
- Logger should write to `~/.local/share/gofortress/tui.log` or `/tmp/gofortress-tui.log`
- Ensure logger never touches stdout/stderr (Ink owns those)
- Remove all `fs.appendFileSync("/tmp/tui-*.log")` debug writes from production code

#### 1.3 Remove Raw Escape Code from Resize Handler
**Priority:** P0
**File:** `useTerminalDimensions.ts`
**Approach:**
- Remove `process.stdout.write('\x1b[2J\x1b[H')` entirely
- Ink handles re-rendering on dimension changes automatically
- Keep the debounce (it's good) but just update state, let Ink re-render

#### 1.4 Fix Stale Closure in Streaming Detection
**Priority:** P0
**File:** `useClaudeQuery.ts`
**Approach:**
- Use a `useRef` for streaming state instead of depending on `isStreaming` in the callback closure
- Or use Zustand's `getState()` to read current streaming value at call time

```tsx
const streamingRef = useRef(false);
// Keep in sync
useEffect(() => { streamingRef.current = isStreaming; }, [isStreaming]);

const handleAssistantEvent = useCallback(async (event) => {
  // Use ref instead of stale closure
  if (streamingRef.current && currentMessageRef.current.length > 0) {
    updateLastMessage(contentBlocks);
  } else {
    addMessage({ ... });
  }
}, [addMessage, updateLastMessage]); // No isStreaming dependency
```

#### 1.5 Add Width Constraints to Message Content
**Priority:** P0
**File:** `ClaudePanel.tsx`
**Approach:**
- Pass terminal width to ClaudePanel (from Layout)
- Calculate content width: `panelWidth - borderChars - paddingChars`
- Apply `<Text wrap="wrap">` on all text content
- Truncate tool input values that exceed width

### Phase 2: Architecture Improvements

#### 2.1 Message Renderer Separation
**Priority:** P1
**New File:** `components/MessageRenderer.tsx`
**Approach:**
- Extract `MessageItem` from `ClaudePanel.tsx` into dedicated component
- Accept `maxWidth` prop for text wrapping
- Render tool_use blocks as collapsible summaries (name + abbreviated input)
- Render tool_result blocks as expandable previews
- Support markdown rendering via `marked-terminal` (currently disabled due to ANSI conflicts - fix by sanitizing ANSI output before passing to Ink `<Text>`)

#### 2.2 Markdown Rendering Recovery
**Priority:** P1
**File:** `utils/markdown.ts`, `MessageRenderer.tsx`
**Approach:**
- The current `renderMarkdown()` is disabled (line 21 of ClaudePanel: "causes ANSI conflicts with Ink")
- Fix: render markdown to ANSI string, then parse ANSI into Ink `<Text>` elements using `ink`'s built-in ANSI support or a library like `ansi-to-react`
- Alternative: use `marked` with a custom renderer that outputs Ink-compatible JSX directly (no ANSI intermediate)

#### 2.3 Three-Section Layout (Header/Content/Footer)
**Priority:** P1
**Files:** Modify `Layout.tsx`, new `Footer.tsx`
**Approach:**
- Header (existing Banner, 3 rows): Session ID, model, cost, streaming status
- Content (flex): Left/Right panel split (current)
- Footer (2 rows): Keybind hints (context-aware), scroll position indicator

```tsx
<Box flexDirection="column" height={terminalHeight}>
  <Header height={3} />
  <Box flexDirection="row" flexGrow={1}>
    <LeftPanel width="70%" />
    <RightPanel width="30%" />
  </Box>
  <Footer height={2} />
</Box>
```

#### 2.4 Proper Height Distribution
**Priority:** P1
**File:** `Layout.tsx`, `ClaudePanel.tsx`
**Approach:**
- Remove `maxHeight` prop from ClaudePanel - use flexbox `flexGrow={1}` instead
- Let Ink's yoga layout engine calculate available space
- Use `measureElement` ref to get actual pixel/row height for ScrollView
- This eliminates all magic-number height math

### Phase 3: Feature Parity with Ralph-TUI

#### 3.0 Status Line (Footer Bar)
**Priority:** P1 (user requested)
**New Files:** `components/StatusLine.tsx`, `scripts/statusline.sh`
**Approach:**
Implement a customizable, informative status bar at the bottom of the TUI, inspired by Claude Code's statusline feature. The status line is a fixed-height (1-2 row) bar anchored at the bottom that displays at-a-glance session metadata.

**Default display (2 lines):**
```
Line 1: [Model] 📁 project-name | 🌿 branch +staged ~modified
Line 2: ▓▓▓░░░░░░░ 30% context | $0.45 | ⏱️ 12m 34s | Agents: 3 running
```

**Data sources (from Zustand store):**
- `activeModel` / `preferredModel` → Model name
- `totalCost` → Session cost (formatted as $X.XX)
- `tokenCount` → Context percentage (input+output / 200K window)
- `streaming` → Streaming indicator
- `agents` → Active agent count by status
- Git branch/status → via cached shell exec (5s cache, like ralph-tui)
- Session duration → computed from session start time

**Component design:**
```tsx
interface StatusLineProps {
  width: number;          // Terminal columns
  height?: 1 | 2;        // 1 or 2 row mode
}
```

**Features:**
- Color-coded context bar: green (<70%), yellow (70-89%), red (90%+)
- Streaming pulse indicator (braille animation) when active
- Agent count with status breakdown (running/queued/complete)
- Responsive: collapses to 1-line on narrow terminals (<100 cols)
- Updates reactively from Zustand store (no polling)
- Git info cached to avoid perf impact (5s TTL like statusline docs recommend)

**Integration with Layout:**
```tsx
<Box flexDirection="column" height={terminalHeight}>
  <Header height={3} />
  <Box flexDirection="row" flexGrow={1}>
    <LeftPanel width="70%" />
    <RightPanel width="30%" />
  </Box>
  <StatusLine width={terminalColumns} height={2} />
</Box>
```

#### 3.1 Right Panel View Cycling
**Priority:** P2
**File:** Modify `Layout.tsx`, new `DashboardView.tsx`, new `SettingsView.tsx`
**Approach:**
- Add `rightPanelMode` to UI store: `"agents" | "dashboard" | "settings"`
- `o` key cycles through modes
- **Agents view:** Current AgentTree + AgentDetail (default)
- **Dashboard view:** Session stats, model info, cost breakdown, git status
- **Settings view:** Read-only config display (model, permission mode, hooks status)

#### 3.2 Streaming Progress Indicator
**Priority:** P2
**File:** New `components/StreamingIndicator.tsx`
**Approach:**
- Braille spinner animation (like ralph-tui's 10-character cycle at 80ms)
- Show token count accumulating in real-time
- Show elapsed time since query start
- Position in footer or below last message

#### 3.3 Toast Notifications
**Priority:** P2
**File:** New `components/Toast.tsx`
**Approach:**
- Floating notification for non-blocking events (model changed, session saved, agent spawned)
- Auto-dismiss after 3 seconds
- Stack up to 3 toasts
- Position: bottom-right overlay

#### 3.4 Tool Block Collapsing
**Priority:** P2
**File:** `MessageRenderer.tsx`
**Approach:**
- Tool blocks default to collapsed: `[Tool: Read] /path/to/file ▸`
- Expandable on selection/keypress: shows full input + result preview
- Reduces visual noise dramatically for long conversations
- Track expanded state per-message in a local Map

#### 3.5 Responsive Layout
**Priority:** P3
**File:** `Layout.tsx`
**Approach:**
- Below 100 columns: hide right panel, show as overlay on Tab
- Below 80 columns: single-column layout, cycle between panels
- Above 120 columns: consider 60/40 split with more detail

#### 3.6 Session Lookback / History Navigation
**Priority:** P2
**File:** Modify `ScrollView.tsx`, `ClaudePanel.tsx`
**Approach:**
- Full conversation history preserved in Zustand store (already done)
- ScrollView enables full scroll-back to beginning of session
- Scroll position indicator: `[msg 5-12 of 47] 26%`
- Home key jumps to top, End key jumps to bottom
- Search within conversation (Ctrl+F opens search overlay)

#### 3.7 ANSI Sanitization Layer
**Priority:** P2
**File:** New `utils/ansi.ts`
**Approach:**
- Strip ANSI escape codes from all external content (tool results, agent output)
- Convert semantic colors via theme (like ralph-tui's `FormattedSegment` pattern)
- Prevents external ANSI from corrupting Ink's rendering state

---

## 4. Implementation Order

```
Phase 1 (Critical - fixes garbled rendering):
  1.1 ScrollView          ← Biggest impact, most complex
  1.2 Logger cleanup      ← Quick win, high impact
  1.3 Resize fix          ← Quick win
  1.4 Streaming closure   ← Medium complexity
  1.5 Width constraints   ← Medium complexity

Phase 2 (Architecture - enables features):
  2.1 MessageRenderer     ← Depends on 1.1, 1.5
  2.2 Markdown recovery   ← Depends on 2.1
  2.3 Header/Footer       ← Independent
  2.4 Height distribution ← Depends on 1.1

Phase 3 (Features - visual upgrade):
  3.1 Panel cycling       ← Depends on 2.3
  3.2 Streaming indicator ← Independent
  3.3 Toast notifications ← Independent
  3.4 Tool collapsing     ← Depends on 2.1
  3.5 Responsive layout   ← Depends on 2.3
  3.6 Session lookback    ← Depends on 1.1
  3.7 ANSI sanitization   ← Depends on 2.1
```

---

## 5. Technology Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scrolling | Custom `ScrollView` with `measureElement` | Ink has no native scrollbox; `ink-scroll-area` is unmaintained |
| Markdown | Custom marked renderer → Ink JSX | Avoids ANSI-in-Ink conflict entirely |
| State management | Zustand (existing) | Already in place, works well |
| ANSI stripping | `strip-ansi` package | Standard, well-maintained |
| Text measurement | Ink's `measureElement` + `string-width` | Handles unicode/emoji widths correctly |
| Logger | File-based, never stdout | Critical for Ink compatibility |

---

## 6. Files Affected Summary

### New Files
- `src/components/ScrollView.tsx` - Line-aware scrollable container
- `src/components/MessageRenderer.tsx` - Message formatting + tool blocks
- `src/components/Footer.tsx` - Keybinds + scroll position
- `src/components/DashboardView.tsx` - Session stats panel
- `src/components/SettingsView.tsx` - Config display panel
- `src/components/StreamingIndicator.tsx` - Token/time progress
- `src/components/Toast.tsx` - Floating notifications
- `src/utils/ansi.ts` - ANSI sanitization

### Modified Files
- `src/components/ClaudePanel.tsx` - Major rewrite (use ScrollView, MessageRenderer)
- `src/components/Layout.tsx` - Add Footer, responsive breakpoints, view cycling
- `src/components/primitives/Viewport.tsx` - Deprecate (replaced by ScrollView)
- `src/hooks/useClaudeQuery.ts` - Fix streaming closure, remove console.log
- `src/hooks/useTerminalDimensions.ts` - Remove escape code write
- `src/utils/markdown.ts` - New renderer producing Ink JSX
- `src/store/slices/ui.ts` - Add rightPanelMode, toasts
- `src/store/types.ts` - Add new UI state types
- `src/config/theme.ts` - Expand with semantic color mappings
- `src/App.tsx` - Remove console.log, clean up demo modes

### Deprecated
- `src/components/primitives/Viewport.tsx` - Replaced by ScrollView
- `src/components/LayoutSpike.tsx` - Spike testing no longer needed
- `src/components/BorderStyleTest.tsx` - Spike testing no longer needed
- `src/components/ResponsiveLayout.tsx` - Merged into Layout.tsx
