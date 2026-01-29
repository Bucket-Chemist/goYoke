---
id: GOgent-MCP-007
title: "Modal Input Handling"
time: "3 hours"
priority: MEDIUM
dependencies: "GOgent-MCP-005, GOgent-MCP-006"
status: pending
---

# GOgent-MCP-007: Modal Input Handling


**Time:** 3 hours
**Dependencies:** GOgent-MCP-005, GOgent-MCP-006
**Priority:** MEDIUM

**Task:**
Handle keyboard input when a modal is active, routing to appropriate response actions.

**File:** Update `internal/tui/claude/input.go`

**Implementation (additions):**
```go
// HandleModalInput processes key presses when modal is active
// Returns true if the key was consumed by the modal
func (m *PanelModel) HandleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    if !m.modal.Active {
        return m, nil
    }

    switch msg.String() {
    case "enter":
        return m.submitModalResponse()

    case "esc":
        m.modal.SendResponse("", true)
        return m, nil

    case "y", "Y":
        if m.modal.Type == ConfirmModal {
            m.modal.SendResponse("yes", false)
            return m, nil
        }

    case "n", "N":
        if m.modal.Type == ConfirmModal {
            m.modal.SendResponse("no", false)
            return m, nil
        }
    }

    // Delegate to component
    var cmd tea.Cmd
    switch m.modal.Type {
    case TextInputModal:
        m.modal.TextInput, cmd = m.modal.TextInput.Update(msg)
    case SelectionModal:
        m.modal.SelectList, cmd = m.modal.SelectList.Update(msg)
    }

    return m, cmd
}

func (m *PanelModel) submitModalResponse() (tea.Model, tea.Cmd) {
    var value string

    switch m.modal.Type {
    case ConfirmModal:
        value = "yes"
    case TextInputModal:
        value = m.modal.TextInput.Value()
    case SelectionModal:
        if item, ok := m.modal.SelectList.SelectedItem().(listItem); ok {
            value = item.title
        }
    }

    m.modal.SendResponse(value, false)
    return m, nil
}
```

**Acceptance Criteria:**
- [ ] Enter submits current value
- [ ] Esc cancels with cancelled=true
- [ ] Y/N work for confirm modals
- [ ] Arrow keys navigate selections
- [ ] Text input captures typing


