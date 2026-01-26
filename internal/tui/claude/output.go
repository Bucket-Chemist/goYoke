package claude

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// updateViewport regenerates the viewport content from message history.
// This is called whenever messages are added or modified.
func (m *PanelModel) updateViewport() {
	var b strings.Builder

	// Render each message
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render("You: "))
			b.WriteString(msg.Content)
		case "assistant":
			b.WriteString(assistantStyle.Render("Claude: "))
			b.WriteString(msg.Content)
		}
		b.WriteString("\n\n")
	}

	// Add streaming indicator if currently streaming
	if m.streaming {
		b.WriteString(streamingStyle.Render("[streaming...]"))
	}

	// Update viewport content and scroll to bottom
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

// appendStreamingText appends text to the last assistant message.
// If the last message is not from the assistant, creates a new assistant message.
// This is used for streaming text display with typewriter effect.
func (m *PanelModel) appendStreamingText(text string) {
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
		// Append to existing assistant message
		m.messages[len(m.messages)-1].Content += text
	} else {
		// Create new assistant message
		m.messages = append(m.messages, Message{
			Role:    "assistant",
			Content: text,
		})
	}

	// Update viewport to show new content
	m.updateViewport()
}

// Styles for different message types
var (
	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan"))

	assistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("green"))

	streamingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)
