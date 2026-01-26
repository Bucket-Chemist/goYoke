package claude

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	"github.com/charmbracelet/lipgloss"
)

// handleEvent processes events from the Claude CLI process.
// Updates state based on event type (assistant, result, system).
func (m PanelModel) handleEvent(event cli.Event) PanelModel {
	switch event.Type {
	case "assistant":
		// Handle assistant message (streaming text)
		if ae, err := event.AsAssistant(); err == nil {
			// Extract text from content blocks
			for _, block := range ae.Message.Content {
				if block.Type == "text" && block.Text != "" {
					m.appendStreamingText(block.Text)
				}
			}
		}

	case "result":
		// Handle result event (cost update, streaming complete)
		if re, err := event.AsResult(); err == nil {
			m.cost = re.TotalCostUSD
			m.streaming = false
			m.state = StateReady // Back to ready after result
			m.updateViewport()   // Remove streaming indicator
		}

	case "system":
		// Handle system event (hook responses)
		if se, err := event.AsSystem(); err == nil {
			if se.Subtype == "hook_response" {
				m.addHookEvent(se.HookName, se.ExitCode == 0)
			}
		}

	case "error":
		// Handle error event
		m.streaming = false
		if ee, err := event.AsError(); err == nil {
			// Add error as assistant message
			errorText := fmt.Sprintf("[Error: %s]", ee.Error)
			m.appendStreamingText(errorText)
		}
	}

	return m
}

// renderHookSidebar renders the hook event sidebar.
// Shows the last 5 hooks with success/failure indicators.
// maxWidth is the maximum width available for the sidebar.
func (m PanelModel) renderHookSidebar(maxWidth int) string {
	var b strings.Builder

	// Header
	header := "Recent Hooks"
	if len(header) > maxWidth {
		header = header[:maxWidth]
	}
	b.WriteString(hookHeaderStyle.Render(header))
	b.WriteString("\n")

	// Show last 5 hooks
	start := 0
	if len(m.hooks) > 5 {
		start = len(m.hooks) - 5
	}

	if start >= len(m.hooks) {
		// No hooks to display
		b.WriteString(hookEmptyStyle.Render("No hooks yet"))
		return b.String()
	}

	for _, hook := range m.hooks[start:] {
		var indicator string
		var style lipgloss.Style

		if hook.Success {
			indicator = "✓" // Unicode checkmark
			style = hookSuccessStyle
		} else {
			indicator = "✗" // Unicode cross
			style = hookFailStyle
		}

		// Truncate hook name to fit available width
		// Reserve 4 chars for: indicator (1) + spaces (2) + padding (1)
		name := hook.Name
		maxNameLen := maxWidth - 4
		if maxNameLen < 3 {
			maxNameLen = 3 // Minimum for "..."
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		line := fmt.Sprintf("%s %s", indicator, name)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// addHookEvent adds a hook event to the history.
func (m *PanelModel) addHookEvent(name string, success bool) {
	m.hooks = append(m.hooks, HookEvent{
		Name:    name,
		Success: success,
	})
}

// Styles for hook sidebar
var (
	hookHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Padding(0, 1)

	hookSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("green")).
				Padding(0, 1)

	hookFailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Padding(0, 1)

	hookEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 1)
)
