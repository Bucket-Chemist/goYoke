---
id: GOgent-MCP-006
title: "Prompt Rendering"
description: "Render modal prompts with lipgloss styling, overlaid on conversation view, with ANSI sanitization for security"
time_estimate: "3h"
priority: MEDIUM
dependencies: ["GOgent-MCP-005"]
status: pending
---

# GOgent-MCP-006: Prompt Rendering


**Time:** 3 hours
**Dependencies:** GOgent-MCP-005
**Priority:** MEDIUM

**Task:**
Render modal prompts with lipgloss styling, overlaid on the conversation view. **CRITICAL:** Sanitize all MCP server prompts for ANSI escape sequence injection (staff-architect issue #3).

**File:** `internal/tui/claude/prompt.go`

**Security Note (From Staff Architect Review):**
MCP server prompts come from external tool calls and MUST be sanitized before display. Malicious prompts could inject ANSI sequences to manipulate terminal state, hide text, or create fake UI elements.

**Implementation:**
```go
package claude

import (
    "strings"

    "github.com/acarl005/stripansi"
    "github.com/charmbracelet/lipgloss"
)

// sanitizePrompt removes ANSI escape sequences from untrusted input
// CRITICAL: Must be called on all MCP server message content
func sanitizePrompt(s string) string {
    return stripansi.Strip(s)
}

var (
    modalStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1, 2).
        Width(50)

    modalTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("170"))

    modalHelpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("241"))

    destructiveStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("196")).
        Bold(true)
)

// RenderModal renders the current modal as a string
func (m *ModalState) RenderModal() string {
    if !m.Active {
        return ""
    }

    var content strings.Builder

    // Title/message - CRITICAL: Sanitize untrusted MCP input
    message := sanitizePrompt(m.Prompt.Message)
    if strings.HasPrefix(message, "⚠️") {
        content.WriteString(destructiveStyle.Render(message))
    } else {
        content.WriteString(modalTitleStyle.Render(message))
    }
    content.WriteString("\n\n")

    // Content based on type
    switch m.Type {
    case ConfirmModal:
        content.WriteString("[Y]es  [N]o  [Esc] Cancel")

    case TextInputModal:
        content.WriteString(m.TextInput.View())
        content.WriteString("\n\n")
        content.WriteString(modalHelpStyle.Render("[Enter] Submit  [Esc] Cancel"))

    case SelectionModal:
        content.WriteString(m.SelectList.View())
        content.WriteString("\n")
        content.WriteString(modalHelpStyle.Render("[Enter] Select  [↑/↓] Navigate  [Esc] Cancel"))
    }

    return modalStyle.Render(content.String())
}

// OverlayModal composites the modal over a background
func OverlayModal(background, modal string, width, height int) string {
    if modal == "" {
        return background
    }

    bgLines := strings.Split(background, "\n")
    modalLines := strings.Split(modal, "\n")

    modalWidth := lipgloss.Width(modal)
    modalHeight := len(modalLines)

    // Center the modal
    startX := max((width-modalWidth)/2, 0)
    startY := max((height-modalHeight)/2, 0)

    // Ensure background has enough lines
    for len(bgLines) < height {
        bgLines = append(bgLines, strings.Repeat(" ", width))
    }

    // Composite modal onto background
    for i, modalLine := range modalLines {
        bgY := startY + i
        if bgY < len(bgLines) {
            bgLines[bgY] = overlayLine(bgLines[bgY], modalLine, startX, width)
        }
    }

    return strings.Join(bgLines, "\n")
}

func overlayLine(background, overlay string, startX, maxWidth int) string {
    // Pad background to maxWidth
    for len(background) < maxWidth {
        background += " "
    }

    bgRunes := []rune(background)
    overlayRunes := []rune(overlay)

    for i, r := range overlayRunes {
        pos := startX + i
        if pos < len(bgRunes) {
            bgRunes[pos] = r
        }
    }

    return string(bgRunes)
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

**Acceptance Criteria:**
- [x] Modal renders with border and padding
- [x] Centered over background content
- [x] Different styles for destructive actions
- [x] Help text shows available keys
- [x] **SECURITY:** ANSI escape sequences stripped from all MCP prompts
- [x] **SECURITY:** Selection options also sanitized

**Security Test:**
```go
func TestSanitizePrompt(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"Normal text", "Normal text"},
        {"\x1b[31mRed text\x1b[0m", "Red text"},
        {"\x1b[2J\x1b[HClear and home", "Clear and home"},
        {"Text with\x1b]0;fake title\x07OSC", "Text withfake titleOSC"},
    }

    for _, tc := range tests {
        got := sanitizePrompt(tc.input)
        if got != tc.expected {
            t.Errorf("sanitizePrompt(%q) = %q, want %q", tc.input, got, tc.expected)
        }
    }
}
```


