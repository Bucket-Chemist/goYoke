package claude

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// handleInput processes keyboard input from the user.
// Returns updated model and any commands to execute.
func (m PanelModel) handleInput(msg tea.KeyMsg) (PanelModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Send message if not currently streaming and textarea has content
		if !m.streaming && m.textarea.Value() != "" {
			content := m.textarea.Value()

			// Clear textarea
			m.textarea.Reset()

			// Check for native command before sending to Claude
			if strings.HasPrefix(content, "/") && IsNativeCommand(content) {
				return m.executeNativeCommand(content)
			}

			// Add user message to history
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: content,
			})

			// Mark as streaming
			m.streaming = true
			m.state = StateStreaming

			// Update viewport to show user message
			m.updateViewport()

			// Send to Claude process
			return m, m.sendMessage(content)
		}
		// Don't pass empty enter to textarea
		return m, nil

	case "esc":
		// Clear current input
		m.textarea.Reset()
		return m, nil

	case "ctrl+l":
		// Clear conversation history (visual only)
		m.ClearConversation()
		return m, nil
	}

	// Pass through to textarea for normal editing
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// sendMessage sends a message to the Claude process.
// Returns a command that performs the send operation.
func (m PanelModel) sendMessage(content string) tea.Cmd {
	return func() tea.Msg {
		err := m.process.Send(content)
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// executeNativeCommand processes a native TUI command (e.g., /model, /context).
// Returns updated model and any commands to execute.
func (m PanelModel) executeNativeCommand(input string) (PanelModel, tea.Cmd) {
	ctx := CommandContext{
		SessionID:    m.sessionID,
		CurrentModel: m.currentModel,
		MessageCount: len(m.messages),
		TotalCost:    m.cost,
	}

	result := ExecuteCommand(input, ctx)

	// Show result message in chat
	if result.Message != "" {
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: result.Message,
		})
		m.updateViewport()
	}

	// Handle errors
	if result.Error != nil {
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: "Error: " + result.Error.Error(),
		})
		m.updateViewport()
		return m, nil
	}

	// Handle model change restart
	if result.RequiresRestart {
		return m, m.requestModelChange(result.NewModel)
	}

	return m, nil
}
