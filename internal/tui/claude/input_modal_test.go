package claude

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

// TestHandleInput_ModalRouting verifies that handleInput delegates to HandleModalInput when modal is active
func TestHandleInput_ModalRouting(t *testing.T) {
	t.Helper()

	// Create a panel with active modal
	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate a confirm modal
	prompt := callback.PromptRequest{
		ID:      "test-confirm",
		Type:    "confirm",
		Message: "Are you sure?",
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Send 'y' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	m, _ = m.handleInput(keyMsg)

	// Verify response was sent
	select {
	case resp := <-respChan:
		if resp.Value != "yes" {
			t.Errorf("Expected value 'yes', got %q", resp.Value)
		}
		if resp.Cancelled {
			t.Error("Expected Cancelled to be false")
		}
	default:
		t.Error("No response received")
	}

	// Modal should be deactivated
	if m.modal.Active {
		t.Error("Modal should be inactive after response")
	}
}

// TestHandleInput_NoModalRouting verifies that handleInput proceeds normally when modal is inactive
func TestHandleInput_NoModalRouting(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})

	// Ensure modal is not active
	if m.modal.Active {
		t.Fatal("Modal should not be active")
	}

	// Set some text in textarea
	m.textarea.SetValue("test message")

	// Press Enter - should send message to Claude, not trigger modal
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, _ = m.handleInput(keyMsg)

	// Verify message was added to history
	if len(m.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(m.messages))
	}
	if len(m.messages) > 0 && m.messages[0].Content != "test message" {
		t.Errorf("Expected message content 'test message', got %q", m.messages[0].Content)
	}

	// Verify textarea was cleared
	if m.textarea.Value() != "" {
		t.Errorf("Textarea should be cleared, got %q", m.textarea.Value())
	}
}

// TestHandleModalInput_EnterKey verifies Enter key submits modal response
func TestHandleModalInput_EnterKey(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		modalType     ModalType
		setupModal    func(m *PanelModel, respChan chan callback.PromptResponse)
		expectedValue string
	}{
		{
			name:      "ConfirmModal Enter submits yes",
			modalType: ConfirmModal,
			setupModal: func(m *PanelModel, respChan chan callback.PromptResponse) {
				prompt := callback.PromptRequest{
					ID:      "confirm-enter",
					Type:    "confirm",
					Message: "Confirm?",
				}
				m.modal.HandlePrompt(prompt, respChan)
			},
			expectedValue: "yes",
		},
		{
			name:      "TextInputModal Enter submits text value",
			modalType: TextInputModal,
			setupModal: func(m *PanelModel, respChan chan callback.PromptResponse) {
				prompt := callback.PromptRequest{
					ID:      "input-enter",
					Type:    "input",
					Message: "Enter name:",
				}
				m.modal.HandlePrompt(prompt, respChan)
				m.modal.TextInput.SetValue("Alice")
			},
			expectedValue: "Alice",
		},
		{
			name:      "SelectionModal Enter submits selected item",
			modalType: SelectionModal,
			setupModal: func(m *PanelModel, respChan chan callback.PromptResponse) {
				prompt := callback.PromptRequest{
					ID:      "select-enter",
					Type:    "select",
					Message: "Pick option:",
					Options: []string{"option1", "option2", "option3"},
				}
				m.modal.HandlePrompt(prompt, respChan)
				// Default is first item
			},
			expectedValue: "option1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			process := NewMockClaudeProcess("test-session")
			m := NewPanelModel(process, cli.Config{Model: "sonnet"})
			respChan := make(chan callback.PromptResponse, 1)

			// Setup modal
			tt.setupModal(&m, respChan)

			// Press Enter
			keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
			m, _ = m.HandleModalInput(keyMsg)

			// Verify response
			select {
			case resp := <-respChan:
				if resp.Value != tt.expectedValue {
					t.Errorf("Expected value %q, got %q", tt.expectedValue, resp.Value)
				}
				if resp.Cancelled {
					t.Error("Expected Cancelled to be false")
				}
			default:
				t.Error("No response received")
			}

			// Modal should be inactive
			if m.modal.Active {
				t.Error("Modal should be inactive after Enter")
			}
		})
	}
}

// TestHandleModalInput_EscKey verifies Esc key cancels modal
func TestHandleModalInput_EscKey(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate a text input modal
	prompt := callback.PromptRequest{
		ID:      "input-esc",
		Type:    "input",
		Message: "Enter something:",
	}
	m.modal.HandlePrompt(prompt, respChan)
	m.modal.TextInput.SetValue("partial input")

	// Press Esc
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m, _ = m.HandleModalInput(keyMsg)

	// Verify response
	select {
	case resp := <-respChan:
		if resp.Value != "" {
			t.Errorf("Expected empty value, got %q", resp.Value)
		}
		if !resp.Cancelled {
			t.Error("Expected Cancelled to be true")
		}
	default:
		t.Error("No response received")
	}

	// Modal should be inactive
	if m.modal.Active {
		t.Error("Modal should be inactive after Esc")
	}
}

// TestHandleModalInput_YNKeys verifies Y/N keys work for confirm modals
func TestHandleModalInput_YNKeys(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		key           string
		expectedValue string
	}{
		{name: "lowercase y", key: "y", expectedValue: "yes"},
		{name: "uppercase Y", key: "Y", expectedValue: "yes"},
		{name: "lowercase n", key: "n", expectedValue: "no"},
		{name: "uppercase N", key: "N", expectedValue: "no"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			process := NewMockClaudeProcess("test-session")
			m := NewPanelModel(process, cli.Config{Model: "sonnet"})
			respChan := make(chan callback.PromptResponse, 1)

			// Activate confirm modal
			prompt := callback.PromptRequest{
				ID:      "confirm-yn",
				Type:    "confirm",
				Message: "Proceed?",
			}
			m.modal.HandlePrompt(prompt, respChan)

			// Press key
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			m, _ = m.HandleModalInput(keyMsg)

			// Verify response
			select {
			case resp := <-respChan:
				if resp.Value != tt.expectedValue {
					t.Errorf("Expected value %q, got %q", tt.expectedValue, resp.Value)
				}
				if resp.Cancelled {
					t.Error("Expected Cancelled to be false")
				}
			default:
				t.Error("No response received")
			}

			// Modal should be inactive
			if m.modal.Active {
				t.Error("Modal should be inactive after Y/N key")
			}
		})
	}
}

// TestHandleModalInput_YNKeysIgnoredForNonConfirm verifies Y/N keys are ignored for non-confirm modals
func TestHandleModalInput_YNKeysIgnoredForNonConfirm(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate text input modal
	prompt := callback.PromptRequest{
		ID:      "input-yn",
		Type:    "input",
		Message: "Enter text:",
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Press 'y' - should be treated as regular text input
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	m, _ = m.HandleModalInput(keyMsg)

	// Modal should still be active
	if !m.modal.Active {
		t.Error("Modal should still be active")
	}

	// No response should be sent yet
	select {
	case <-respChan:
		t.Error("Response should not be sent for Y/N keys in text input modal")
	default:
		// Expected - no response yet
	}

	// The 'y' should be processed as text input
	if m.modal.TextInput.Value() != "y" {
		t.Errorf("Expected TextInput value 'y', got %q", m.modal.TextInput.Value())
	}
}

// TestHandleModalInput_ArrowKeys verifies arrow keys navigate selection modal
func TestHandleModalInput_ArrowKeys(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate selection modal
	prompt := callback.PromptRequest{
		ID:      "select-arrows",
		Type:    "select",
		Message: "Choose:",
		Options: []string{"first", "second", "third"},
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Initially first item is selected
	if item, ok := m.modal.SelectList.SelectedItem().(listItem); ok {
		if item.title != "first" {
			t.Errorf("Expected initial selection 'first', got %q", item.title)
		}
	} else {
		t.Fatal("Failed to get selected item")
	}

	// Press down arrow
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.HandleModalInput(keyMsg)

	// Should move to second item
	if item, ok := m.modal.SelectList.SelectedItem().(listItem); ok {
		if item.title != "second" {
			t.Errorf("Expected selection after down arrow 'second', got %q", item.title)
		}
	} else {
		t.Fatal("Failed to get selected item after down arrow")
	}

	// Modal should still be active
	if !m.modal.Active {
		t.Error("Modal should still be active after arrow navigation")
	}

	// No response sent yet
	select {
	case <-respChan:
		t.Error("Response should not be sent for arrow keys")
	default:
		// Expected
	}
}

// TestHandleModalInput_TextInputTyping verifies typing works in text input modal
func TestHandleModalInput_TextInputTyping(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate text input modal
	prompt := callback.PromptRequest{
		ID:      "input-typing",
		Type:    "input",
		Message: "Enter text:",
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Type some characters
	chars := []rune{'h', 'e', 'l', 'l', 'o'}
	for _, ch := range chars {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		m, _ = m.HandleModalInput(keyMsg)
	}

	// Verify text was captured
	if m.modal.TextInput.Value() != "hello" {
		t.Errorf("Expected TextInput value 'hello', got %q", m.modal.TextInput.Value())
	}

	// Modal should still be active
	if !m.modal.Active {
		t.Error("Modal should still be active during typing")
	}

	// No response sent yet
	select {
	case <-respChan:
		t.Error("Response should not be sent during typing")
	default:
		// Expected
	}
}

// TestHandleModalInput_InactiveModal verifies HandleModalInput returns immediately if modal is not active
func TestHandleModalInput_InactiveModal(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})

	// Ensure modal is not active
	if m.modal.Active {
		t.Fatal("Modal should not be active")
	}

	// Try to handle input
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd := m.HandleModalInput(keyMsg)

	// Should return nil command
	if cmd != nil {
		t.Error("Expected nil command when modal is inactive")
	}

	// Modal should still be inactive
	if m.modal.Active {
		t.Error("Modal should remain inactive")
	}
}

// TestSubmitModalResponse_ConfirmModal verifies submitModalResponse for confirm modal
func TestSubmitModalResponse_ConfirmModal(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate confirm modal
	prompt := callback.PromptRequest{
		ID:      "confirm-submit",
		Type:    "confirm",
		Message: "Confirm?",
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Submit response
	m, cmd := m.submitModalResponse()

	// Should return nil command
	if cmd != nil {
		t.Error("Expected nil command from submitModalResponse")
	}

	// Verify response
	select {
	case resp := <-respChan:
		if resp.Value != "yes" {
			t.Errorf("Expected value 'yes', got %q", resp.Value)
		}
		if resp.Cancelled {
			t.Error("Expected Cancelled to be false")
		}
	default:
		t.Error("No response received")
	}

	// Modal should be inactive
	if m.modal.Active {
		t.Error("Modal should be inactive after submit")
	}
}

// TestSubmitModalResponse_TextInputModal verifies submitModalResponse for text input modal
func TestSubmitModalResponse_TextInputModal(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate text input modal
	prompt := callback.PromptRequest{
		ID:      "input-submit",
		Type:    "input",
		Message: "Enter value:",
	}
	m.modal.HandlePrompt(prompt, respChan)
	m.modal.TextInput.SetValue("test value")

	// Submit response
	m, _ = m.submitModalResponse()

	// Verify response
	select {
	case resp := <-respChan:
		if resp.Value != "test value" {
			t.Errorf("Expected value 'test value', got %q", resp.Value)
		}
		if resp.Cancelled {
			t.Error("Expected Cancelled to be false")
		}
	default:
		t.Error("No response received")
	}

	// Modal should be inactive
	if m.modal.Active {
		t.Error("Modal should be inactive after submit")
	}
}

// TestSubmitModalResponse_SelectionModal verifies submitModalResponse for selection modal
func TestSubmitModalResponse_SelectionModal(t *testing.T) {
	t.Helper()

	process := NewMockClaudeProcess("test-session")
	m := NewPanelModel(process, cli.Config{Model: "sonnet"})
	respChan := make(chan callback.PromptResponse, 1)

	// Activate selection modal
	prompt := callback.PromptRequest{
		ID:      "select-submit",
		Type:    "select",
		Message: "Choose option:",
		Options: []string{"alpha", "beta", "gamma"},
	}
	m.modal.HandlePrompt(prompt, respChan)

	// Default selection is first item
	// Submit response
	m, _ = m.submitModalResponse()

	// Verify response
	select {
	case resp := <-respChan:
		if resp.Value != "alpha" {
			t.Errorf("Expected value 'alpha', got %q", resp.Value)
		}
		if resp.Cancelled {
			t.Error("Expected Cancelled to be false")
		}
	default:
		t.Error("No response received")
	}

	// Modal should be inactive
	if m.modal.Active {
		t.Error("Modal should be inactive after submit")
	}
}
