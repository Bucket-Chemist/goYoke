package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
	"github.com/charmbracelet/lipgloss"
)

// handleEvent processes events from the Claude CLI process.
// Updates state based on event type (assistant, result, system).
func (m PanelModel) handleEvent(event cli.Event) PanelModel {
	// Log all events to debug file
	if m.logFile != nil {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Fprintf(m.logFile, "[%s] EVENT: type=%s subtype=%s\n", timestamp, event.Type, event.Subtype)
		m.logFile.Sync()
	}

	switch event.Type {
	case "assistant":
		// Handle assistant message (streaming text and tool use)
		if ae, err := event.AsAssistant(); err == nil {
			// Extract content from all block types
			for _, block := range ae.Message.Content {
				switch block.Type {
				case "text":
					if block.Text != "" {
						m.appendStreamingText(block.Text)
					}
				case "tool_use":
					// Show tool invocation with details
					toolInfo := formatToolUse(block.Name, block.Input)
					m.appendStreamingText(toolInfo)
				case "thinking":
					// Show thinking content if present
					if block.Text != "" {
						thinkInfo := fmt.Sprintf("\n💭 %s\n", truncateText(block.Text, 200))
						m.appendStreamingText(thinkInfo)
					}
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

			// Log complete assistant response to debug file
			if m.logFile != nil && len(m.messages) > 0 {
				lastMsg := m.messages[len(m.messages)-1]
				if lastMsg.Role == "assistant" {
					timestamp := time.Now().Format("15:04:05")
					fmt.Fprintf(m.logFile, "[%s] assistant:\n%s\n\n", timestamp, lastMsg.Content)
					m.logFile.Sync()
				}
			}
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

	case "user":
		// TEMPORARY: Capture raw event for schema discovery
		// This will be replaced with proper permission handling once we understand the schema
		debugPath := fmt.Sprintf("/tmp/user-event-%d.json", time.Now().Unix())
		if err := os.WriteFile(debugPath, event.Raw, 0644); err != nil {
			// Log error but don't block - this is just instrumentation
			m.appendStreamingText(fmt.Sprintf("\n[Warning: Failed to save debug event: %v]\n", err))
		} else {
			m.addMessage("system", fmt.Sprintf("🔍 Permission event detected. Raw data saved to: %s", debugPath))
			m.updateViewport()
		}

	default:
		// Log unknown event types for debugging
		// This helps identify what else Claude CLI sends
		if event.Type != "" {
			debugInfo := fmt.Sprintf("\n[Event: %s", event.Type)
			if event.Subtype != "" {
				debugInfo += "/" + event.Subtype
			}
			debugInfo += "]\n"
			m.appendStreamingText(debugInfo)
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

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("magenta")).
			Bold(true)

	toolDetailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// formatToolUse formats a tool invocation for display.
// Shows tool name and a summary of the input.
func formatToolUse(name string, input map[string]interface{}) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(toolStyle.Render(fmt.Sprintf("🔧 [%s]", name)))

	// Format key details based on tool type
	if input != nil {
		switch name {
		case "Bash":
			if cmd, ok := input["command"].(string); ok {
				b.WriteString("\n")
				b.WriteString(toolDetailStyle.Render("  $ " + truncateText(cmd, 100)))
			}
		case "Read":
			if path, ok := input["file_path"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s", path)))
			}
		case "Write", "Edit":
			if path, ok := input["file_path"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s", path)))
			}
		case "Glob":
			if pattern, ok := input["pattern"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s", pattern)))
			}
		case "Grep":
			if pattern, ok := input["pattern"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" /%s/", pattern)))
			}
		case "Task":
			if desc, ok := input["description"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s", desc)))
			}
		case "WebFetch", "WebSearch":
			if url, ok := input["url"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s", truncateText(url, 60))))
			} else if query, ok := input["query"].(string); ok {
				b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" \"%s\"", query)))
			}
		default:
			// For unknown tools, show first key-value pair
			for k, v := range input {
				if str, ok := v.(string); ok {
					b.WriteString(toolDetailStyle.Render(fmt.Sprintf(" %s=%s", k, truncateText(str, 50))))
					break
				}
			}
		}
	}

	b.WriteString("\n")
	return b.String()
}

// truncateText truncates text to maxLen, adding ellipsis if needed.
func truncateText(text string, maxLen int) string {
	// Remove newlines for single-line display
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")

	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return "..."
	}
	return text[:maxLen-3] + "..."
}

// formatEventDetails formats raw event JSON for debugging.
func formatEventDetails(event cli.Event) string {
	if event.Raw == nil {
		return ""
	}

	// Pretty print with indentation, but limit size
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(event.Raw, &prettyJSON); err != nil {
		return string(event.Raw)
	}

	formatted, err := json.MarshalIndent(prettyJSON, "  ", "  ")
	if err != nil {
		return string(event.Raw)
	}

	result := string(formatted)
	if len(result) > 500 {
		result = result[:500] + "\n  ..."
	}
	return result
}
