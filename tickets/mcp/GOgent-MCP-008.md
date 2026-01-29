---
id: GOgent-MCP-008
title: "External Event Integration"
time: "4 hours"
priority: HIGH
dependencies: "GOgent-MCP-001, GOgent-MCP-007"
status: pending
---

# GOgent-MCP-008: External Event Integration


**Time:** 4 hours
**Dependencies:** GOgent-MCP-001, GOgent-MCP-007
**Priority:** HIGH

**Task:**
Integrate the callback server's prompt channel with Bubbletea's event loop using tea.Cmd. **CRITICAL:** Fix channel blocking issue where bare channel read blocks forever if TUI quits (staff-architect issue #4).

**File:** Update `internal/tui/claude/panel.go`

**Problem (From Staff Architect Review):**
Bare channel read `req := <-m.callbackServer.PromptChan` blocks forever if the TUI quits while waiting. This causes goroutine leaks and prevents clean shutdown.

**Implementation (additions):**
```go
// Add to PanelModel struct
type PanelModel struct {
    // ... existing fields ...

    callbackServer *callback.Server
    modal          ModalState
    ctx            context.Context // Added: cancellation context
}

// NewPanelModelWithCallback creates a panel with callback server
func NewPanelModelWithCallback(ctx context.Context, process ClaudeProcessInterface, cfg cli.Config, server *callback.Server) PanelModel {
    m := NewPanelModel(process, cfg)
    m.callbackServer = server
    m.modal = NewModalState()
    m.ctx = ctx // Store context for cancellation
    return m
}

// ListenForPrompts creates a command that waits for the next prompt
// CRITICAL: Uses select with context to prevent goroutine leak on shutdown
func (m PanelModel) ListenForPrompts() tea.Cmd {
    if m.callbackServer == nil {
        return nil
    }

    return func() tea.Msg {
        // FIXED: Use select with context.Done() to avoid blocking forever
        select {
        case req := <-m.callbackServer.PromptChan:
            // Create response channel
            respChan := make(chan callback.PromptResponse, 1)
            // Register with server
            m.callbackServer.RegisterPending(req.ID, respChan)
            return MCPPromptMsg{
                Request:      req,
                ResponseChan: respChan,
            }
        case <-m.ctx.Done():
            // TUI is shutting down, return nil to stop listening
            return nil
        }
    }
}

// Update in Update method
func (m PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case MCPPromptMsg:
        cmd := m.modal.HandlePrompt(msg.Request, msg.ResponseChan)
        return m, tea.Batch(cmd, m.ListenForPrompts())

    case tea.KeyMsg:
        if m.modal.Active {
            return m.HandleModalInput(msg)
        }
        // ... existing key handling ...
    }

    // ... existing update logic ...
}

// Update in View method
func (m PanelModel) View() string {
    main := m.renderMainContent()

    if m.modal.Active {
        return OverlayModal(main, m.modal.RenderModal(), m.width, m.height)
    }

    return main
}
```

**Acceptance Criteria:**
- [ ] Panel listens for prompts via tea.Cmd
- [ ] MCPPromptMsg triggers modal display
- [ ] Response delivered back to callback server
- [ ] Listens for next prompt after response
- [ ] Modal overlays conversation view
- [ ] **CRITICAL:** Goroutine exits cleanly on context cancellation
- [ ] **CRITICAL:** No goroutine leaks on TUI shutdown

**Shutdown Test:**
```go
func TestListenForPrompts_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    server := callback.NewServer(os.Getpid())
    _ = server.Start(ctx)
    defer server.Cleanup()

    panel := NewPanelModelWithCallback(ctx, nil, cli.Config{}, server)

    // Start listening command
    cmd := panel.ListenForPrompts()

    // Cancel context immediately
    cancel()

    // Command should return nil quickly, not block
    done := make(chan struct{})
    go func() {
        result := cmd()
        if result != nil {
            t.Errorf("Expected nil on cancellation, got %v", result)
        }
        close(done)
    }()

    select {
    case <-done:
        // Good - command exited
    case <-time.After(time.Second):
        t.Error("ListenForPrompts blocked after context cancellation")
    }
}
```


