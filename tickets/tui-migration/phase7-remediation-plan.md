# Phase 7 Remediation Plan: Architectural Gaps & Lost Features

> **Generated:** 2026-03-23
> **Context:** Post-Phase 7 architectural analysis identified 5 gaps between the Go TUI and the TypeScript source implementation. This plan addresses all 5 with concrete, delegation-ready tasks.

---

## Problem Statement

Phase 7 delivered a structurally sound multi-provider system (TUI-028–032, 1069 tests, 21 packages). However, comparison with the TypeScript TUI reveals:

1. **7 Phase 7 widgets not wired in main.go** — components won't render in the running TUI
2. **No handoff generation on provider switch** — UX regression (context loss when switching)
3. **No debounce on provider switch** — rapid Shift+Tab creates/destroys CLI drivers
4. **ToolBlocks lost on message save/restore** — conversation fidelity degraded
5. **No in-session model switching** — requires CLI restart (TS also lacks this — parity, not regression)

Items 1–4 are implementable now. Item 5 is blocked by CLI protocol limitations and deferred.

---

## Task Breakdown

### Task R-1: Wire Phase 7 Widgets in main.go
**Priority:** CRITICAL — without this, Phase 7 components don't render
**Agent:** go-tui
**Effort:** Small (30–50 lines)
**Files:** `cmd/gofortress/main.go`

**What to do:**
Insert after the existing `app.SetTeamList(&teamList)` call (line 98), before CLI driver setup (line 100):

```go
// Phase 7: Provider components
ps := app.ProviderState() // need getter, or read from sharedState
ptb := providers.NewProviderTabBarModel(ps, 0)
app.SetProviderTabBar(&ptb)

// Phase 7: Right-panel components
dashModel := dashboard.NewDashboardModel()
app.SetDashboard(&dashModel)

settingsModel := settings.NewSettingsModel()
app.SetSettings(&settingsModel)

telemetryModel := telemetry.NewTelemetryModel()
app.SetTelemetry(&telemetryModel)

ppModel := planpreview.NewPlanPreviewModel()
app.SetPlanPreview(&ppModel)

tbModel := taskboard.NewTaskBoardModel()
app.SetTaskBoard(&tbModel)
```

**Note:** `NewProviderTabBarModel` requires `*state.ProviderState`. Currently `providerState` is created inside `NewAppModel()` — either:
- (a) Add a `ProviderState() *state.ProviderState` getter on AppModel, or
- (b) Create ProviderState in main.go and pass it in via `SetProviderState()`

Option (b) is more consistent with the existing pattern (SetCLIDriver, SetBridge, etc.).

**Also wire initial data:**
```go
// After CLI driver setup — initial settings display
settingsModel.SetConfig(
    *modelOverride,
    "Anthropic", // default provider
    *permMode,
    ".", // project dir
    []string{"gofortress"}, // MCP servers
)
```

**Acceptance criteria:**
- [ ] All 7 widgets instantiated in main.go
- [ ] Provider tab bar visible between top tab bar and main area
- [ ] Alt+R cycles through all 5 right-panel modes with real content
- [ ] Alt+B toggles task board overlay
- [ ] Existing tests still pass

---

### Task R-2: Add Provider Switch Debounce
**Priority:** HIGH — prevents real bug (rapid driver churn)
**Agent:** go-tui
**Effort:** Small (40–60 lines)
**Files:** `internal/tui/model/app.go`, `internal/tui/model/messages.go`, `internal/tui/model/app_test.go`

**What to do:**

1. Add a debounce message type in `messages.go`:
```go
// ProviderSwitchExecuteMsg is the debounced execution of a provider switch.
// It fires 300ms after the last Shift+Tab press.
type ProviderSwitchExecuteMsg struct{}
```

2. Add a debounce field to `AppModel`:
```go
providerSwitchPending bool // true while debounce timer is active
```

3. Modify `handleKey` CycleProvider case to schedule instead of execute:
```go
case key.Matches(msg, m.keys.Global.CycleProvider):
    if m.shared != nil && m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming() {
        return m, nil
    }
    // Debounce: schedule execution after 300ms. Rapid presses reset the timer
    // by discarding the pending message (ProviderSwitchMsg is idempotent).
    m.providerSwitchPending = true
    return m, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
        return ProviderSwitchExecuteMsg{}
    })
```

4. Handle `ProviderSwitchExecuteMsg` in Update:
```go
case ProviderSwitchExecuteMsg:
    if !m.providerSwitchPending {
        return m, nil // debounce cancelled by newer keypress
    }
    m.providerSwitchPending = false
    return m.handleProviderSwitch()
```

5. In the CycleProvider key handler, when a new press arrives while pending, the old timer's message will find `providerSwitchPending` already reset. Only the latest timer fires.

**Acceptance criteria:**
- [ ] Rapid Shift+Tab (3x within 100ms) results in only 1 provider switch
- [ ] Single Shift+Tab still works (switches after 300ms)
- [ ] Streaming still blocks switch
- [ ] Tests cover debounce behavior

---

### Task R-3: Implement Handoff Generation on Provider Switch
**Priority:** HIGH — major UX feature
**Agent:** go-pro
**Effort:** Medium (80–120 lines)
**Files:** `internal/tui/model/app.go`, `internal/tui/model/handoff.go` (new), `internal/tui/model/handoff_test.go` (new)

**What to do:**

Create `handoff.go` with a static handoff generator (no AI call needed for v1):

```go
// buildHandoffSummary creates a human-readable context summary from the last N
// messages of a conversation. This is injected as a system message when switching
// providers so the new provider has context about what was being discussed.
//
// Returns "" if there are fewer than 2 messages (nothing meaningful to summarize).
func buildHandoffSummary(msgs []state.DisplayMessage, fromProvider, toProvider state.ProviderID) string
```

**Logic:**
1. Take last 10 messages (or fewer if conversation is short)
2. Extract key content:
   - Last user message (the most recent request)
   - Last assistant message summary (first 200 chars)
   - Count of tool calls mentioned
3. Format as a compact summary:
   ```
   [Context from Anthropic → Google]
   Last request: "Help me implement the authentication module"
   Assistant was: Working on auth middleware implementation...
   Conversation: 12 messages, 3 tool calls
   ```
4. Return "" if < 2 messages (no useful context)

**Wire into `handleProviderSwitch()`:**

After step 4 (restore messages) and before step 5 (shutdown driver):
```go
// 4.5: Generate and inject handoff context
if m.shared.claudePanel != nil {
    oldMsgs := ps.GetActiveMessages() // already saved in step 1
    // Read from the OLD provider's messages (before switch)
    handoff := buildHandoffSummary(oldMsgs, oldProvider, ps.GetActiveProvider())
    if handoff != "" {
        ps.AppendMessage(state.DisplayMessage{
            Role:      "system",
            Content:   handoff,
            Timestamp: time.Now(),
        })
    }
}
```

**Note:** Need to capture `oldProvider` before step 3 (the cycle):
```go
oldProvider := ps.GetActiveProvider() // capture BEFORE cycling
```

**Acceptance criteria:**
- [ ] Switching from Anthropic (with 5+ messages) to Google shows handoff system message
- [ ] Switching to a provider with empty history shows just the handoff (no old messages)
- [ ] Fewer than 2 messages → no handoff injected
- [ ] Handoff content is human-readable and concise
- [ ] Table-driven tests for buildHandoffSummary

---

### Task R-4: Preserve ToolBlocks Across Provider Switch
**Priority:** MEDIUM — conversation fidelity
**Agent:** go-pro
**Effort:** Small (30–50 lines)
**Files:** `internal/tui/state/provider.go`, `internal/tui/components/claude/panel.go`, `internal/tui/components/claude/panel_test.go`, `internal/tui/state/provider_test.go`

**What to do:**

1. Add a `ToolBlock` type to `state/provider.go` (mirror of `claude.ToolBlock`):
```go
// ToolBlock represents a tool invocation stored within a DisplayMessage.
// This is a cross-package type; the claude package converts between its
// local DisplayMessage.ToolBlocks and these for persistence.
type ToolBlock struct {
    Name     string
    Input    string
    Output   string
}
```
Note: `Expanded` field is NOT included — that's transient UI state.

2. Add `ToolBlocks []ToolBlock` field to `state.DisplayMessage`:
```go
type DisplayMessage struct {
    Role       string
    Content    string
    Timestamp  time.Time
    ToolBlocks []ToolBlock  // preserved across provider switch
}
```

3. Update `claude/panel.go` `SaveMessages()` to include ToolBlocks:
```go
for i, msg := range m.messages {
    var blocks []state.ToolBlock
    for _, tb := range msg.ToolBlocks {
        blocks = append(blocks, state.ToolBlock{
            Name:   tb.Name,
            Input:  tb.Input,
            Output: tb.Output,
        })
    }
    result[i] = state.DisplayMessage{
        Role:       msg.Role,
        Content:    msg.Content,
        Timestamp:  msg.Timestamp,
        ToolBlocks: blocks,
    }
}
```

4. Update `claude/panel.go` `RestoreMessages()` to restore ToolBlocks:
```go
for i, msg := range msgs {
    var blocks []ToolBlock
    for _, tb := range msg.ToolBlocks {
        blocks = append(blocks, ToolBlock{
            Name:   tb.Name,
            Input:  tb.Input,
            Output: tb.Output,
        })
    }
    m.messages[i] = DisplayMessage{
        Role:       msg.Role,
        Content:    msg.Content,
        Timestamp:  msg.Timestamp,
        ToolBlocks: blocks,
    }
}
```

**Acceptance criteria:**
- [ ] ToolBlocks preserved across provider switch roundtrip
- [ ] Expanded state NOT preserved (always starts collapsed on restore)
- [ ] state.ToolBlock has no dependency on claude package
- [ ] Tests for SaveMessages/RestoreMessages with ToolBlocks

---

### Task R-5: In-Session Model Switching (DEFERRED — Design Only)
**Priority:** LOW — TS also doesn't have this, not a regression
**Agent:** N/A (documentation only)
**Effort:** N/A

**Status:** DEFERRED to post-migration. The Claude CLI subprocess doesn't support hot model changes — the `--model` flag is set at launch time. Changing models requires:
1. Full CLI restart (loses streaming state)
2. Session resume with `--resume` + new `--model` (preserves conversation but restarts subprocess)

Option 2 is viable post-migration via the existing `handleProviderSwitch` pattern (shutdown→create→start). The model change would follow the same flow but skip the provider cycle. This could be triggered by a `/model` command in the input.

**No code changes needed now.** Document as a known limitation.

---

## Dependency Order

```
R-1 (main.go wiring) ─────── No dependencies, enables visual testing
R-2 (debounce) ──────────── No dependencies, standalone
R-3 (handoff) ───────────── No dependencies, standalone
R-4 (ToolBlock persist) ──── No dependencies, standalone
R-5 (model switch) ────────── DEFERRED
```

All tasks R-1 through R-4 are independent and can be executed **in parallel**.

---

## Verification

After all tasks complete:

1. **Build and run:** `go build ./cmd/gofortress && ./gofortress`
   - Provider tab bar visible below top tab bar
   - Alt+R cycles through 5 right-panel modes
   - All panels show real content (not placeholder text)
   - Alt+B toggles task board

2. **Provider switching:** Shift+Tab
   - Debounce prevents rapid switching (visual: tab bar doesn't flicker)
   - Handoff message appears in new provider's conversation
   - ToolBlocks preserved when switching back to original provider

3. **Tests:** `go test -race ./internal/tui/...`
   - All packages pass
   - Race detector clean
   - Coverage: model ≥80%, new files ≥85%

4. **Total test count:** Should increase from 1069 to ~1100+

---

## Effort Summary

| Task | Agent | Effort | Lines | Priority |
|------|-------|--------|-------|----------|
| R-1: main.go wiring | go-tui | Small | 30–50 | CRITICAL |
| R-2: Debounce | go-tui | Small | 40–60 | HIGH |
| R-3: Handoff | go-pro | Medium | 80–120 | HIGH |
| R-4: ToolBlock persist | go-pro | Small | 30–50 | MEDIUM |
| R-5: Model switch | N/A | Deferred | 0 | LOW |
| **Total** | | | **~180–280** | |

**Estimated session time:** 30–45 minutes (tasks R-1 through R-4 can run in parallel)
