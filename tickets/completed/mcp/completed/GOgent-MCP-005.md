---
id: GOgent-MCP-005
title: "Modal State Management"
description: "Add modal state management to the Claude panel for displaying prompts over the conversation view"
time_estimate: "4h"
priority: HIGH
dependencies: ["GOgent-MCP-001"]
status: pending
---

# GOgent-MCP-005: Modal State Management


**Time:** 4 hours
**Dependencies:** GOgent-MCP-001
**Priority:** HIGH

**Task:**
Add modal state management to the Claude panel for displaying prompts over the conversation view.

**File:** `internal/tui/claude/modal.go`

**Imports:**
```go
package claude

import (
    "strings"

    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)
```

**Implementation:**
```go
// ModalType identifies the kind of modal displayed
type ModalType int

const (
    NoModal ModalType = iota
    ConfirmModal
    TextInputModal
    SelectionModal
)

// ModalState holds the current modal's state
type ModalState struct {
    Active       bool
    Type         ModalType
    Prompt       callback.PromptRequest
    TextInput    textinput.Model
    SelectList   list.Model
    ResponseChan chan<- callback.PromptResponse
}

// NewModalState creates an empty modal state
func NewModalState() ModalState {
    ti := textinput.New()
    ti.Placeholder = "Type your response..."
    ti.CharLimit = 500
    ti.Width = 40

    return ModalState{
        TextInput: ti,
    }
}

// MCPPromptMsg is sent when a prompt request arrives
type MCPPromptMsg struct {
    Request      callback.PromptRequest
    ResponseChan chan<- callback.PromptResponse
}

// MCPResponseSentMsg is sent after response is delivered
type MCPResponseSentMsg struct {
    PromptID string
}

// HandlePrompt activates a modal for the given prompt
func (m *ModalState) HandlePrompt(prompt callback.PromptRequest, respChan chan<- callback.PromptResponse) tea.Cmd {
    m.Active = true
    m.Prompt = prompt
    m.ResponseChan = respChan

    switch prompt.Type {
    case "confirm":
        m.Type = ConfirmModal
        return nil

    case "input", "ask":
        if len(prompt.Options) > 0 {
            m.Type = SelectionModal
            m.SelectList = createSelectList(prompt.Options)
            return nil
        }
        m.Type = TextInputModal
        m.TextInput.Reset()
        if prompt.Default != "" {
            m.TextInput.SetValue(prompt.Default)
        }
        return m.TextInput.Focus()

    case "select":
        m.Type = SelectionModal
        m.SelectList = createSelectList(prompt.Options)
        return nil

    default:
        // Fallback to text input
        m.Type = TextInputModal
        m.TextInput.Reset()
        return m.TextInput.Focus()
    }
}

// SendResponse sends the response and closes the modal
func (m *ModalState) SendResponse(value string, cancelled bool) {
    if m.ResponseChan == nil {
        return
    }

    m.ResponseChan <- callback.PromptResponse{
        ID:        m.Prompt.ID,
        Value:     value,
        Cancelled: cancelled,
    }

    m.Active = false
    m.ResponseChan = nil
}

// listItem implements list.Item for selection
type listItem struct {
    title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }

func createSelectList(options []string) list.Model {
    items := make([]list.Item, len(options))
    for i, opt := range options {
        items[i] = listItem{title: opt}
    }

    delegate := list.NewDefaultDelegate()
    delegate.SetHeight(1)

    l := list.New(items, delegate, 40, min(len(options)+4, 12))
    l.SetShowTitle(false)
    l.SetShowStatusBar(false)
    l.SetShowHelp(false)
    l.SetFilteringEnabled(false)

    return l
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Acceptance Criteria:**
- [x] ModalState tracks active prompt
- [x] Handles all prompt types: confirm, input, select
- [x] SendResponse delivers to channel
- [x] State cleared after response


